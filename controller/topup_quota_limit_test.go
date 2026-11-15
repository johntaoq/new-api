package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTopUpQuotaValidation(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	common.QuotaPerUnit = 500000
	t.Cleanup(func() {
		common.QuotaPerUnit = oldQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
	})

	testCases := []struct {
		name        string
		displayType string
		amount      int64
		wantQuota   int
		wantErr     bool
	}{
		{
			name:        "currency amount below limit",
			displayType: operation_setting.QuotaDisplayTypeUSD,
			amount:      4294,
			wantQuota:   2_147_000_000,
		},
		{
			name:        "currency amount above limit",
			displayType: operation_setting.QuotaDisplayTypeUSD,
			amount:      4295,
			wantQuota:   2_147_500_000,
		},
		{
			name:        "token amount preserves settlement truncation",
			displayType: operation_setting.QuotaDisplayTypeTokens,
			amount:      2_147_500_000,
			wantQuota:   2_147_500_000,
		},
		{
			name:        "token amount above legacy int32 range",
			displayType: operation_setting.QuotaDisplayTypeTokens,
			amount:      4_294_500_000,
			wantQuota:   4_294_500_000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			operation_setting.GetGeneralSetting().QuotaDisplayType = tc.displayType
			quota, err := getTopUpQuota(tc.amount)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantQuota, quota)
		})
	}
}

func TestFinanceTopupCurrencySnapshot(t *testing.T) {
	oldRate, oldQuota, oldMin := operation_setting.USDExchangeRate, common.QuotaPerUnit, operation_setting.MinTopUp
	oldDiscount := operation_setting.GetPaymentSetting().AmountDiscount
	t.Cleanup(func() {
		operation_setting.USDExchangeRate, common.QuotaPerUnit, operation_setting.MinTopUp = oldRate, oldQuota, oldMin
		operation_setting.GetPaymentSetting().AmountDiscount = oldDiscount
	})
	operation_setting.USDExchangeRate, common.QuotaPerUnit, operation_setting.MinTopUp = 7, 500000, 10
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{70: 0.9}
	order, err := calculateFinanceTopup(70, "CNY", "default")
	require.NoError(t, err)
	assert.EqualValues(t, 5000000, order.Quota)
	assert.Equal(t, 63.0, order.Money)
	assert.Equal(t, 9.0, order.RevenueUSD)
	assert.Equal(t, 7.0, order.ExchangeRate)
	order, err = calculateFinanceTopup(10, "USD", "default")
	require.NoError(t, err)
	assert.EqualValues(t, 5000000, order.Quota)
	assert.Equal(t, 70.0, order.Money)
	_, err = calculateFinanceTopup(9, "CNY", "default")
	require.Error(t, err)
	_, err = calculateFinanceTopup(100, "COS", "default")
	require.Error(t, err)
	_, err = calculateFinanceTopup(common.MaxWalletQuota, "USD", "default")
	require.Error(t, err)
}

func TestFinanceTopupOneYuanMinimum(t *testing.T) {
	oldRate, oldQuota, oldMin := operation_setting.USDExchangeRate, common.QuotaPerUnit, operation_setting.MinTopUp
	oldDiscount := operation_setting.GetPaymentSetting().AmountDiscount
	t.Cleanup(func() {
		operation_setting.USDExchangeRate, common.QuotaPerUnit, operation_setting.MinTopUp = oldRate, oldQuota, oldMin
		operation_setting.GetPaymentSetting().AmountDiscount = oldDiscount
	})
	common.QuotaPerUnit = 500000
	operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}
	for _, tc := range []struct {
		name     string
		minimum  int
		amount   int64
		currency string
		rate     float64
		money    float64
		quota    int64
		wantErr  bool
	}{
		{name: "one yuan is accepted", minimum: 1, amount: 1, currency: "CNY", rate: 7, money: 1, quota: 71428},
		{name: "one dollar retains exchange conversion", minimum: 1, amount: 1, currency: "USD", rate: 7, money: 7, quota: 500000},
		{name: "unset minimum falls back to one yuan", minimum: 0, amount: 1, currency: "CNY", rate: 7, money: 1, quota: 71428},
		{name: "below one yuan after conversion is rejected", minimum: 1, amount: 1, currency: "USD", rate: 0.5, wantErr: true},
		{name: "higher configured minimum is respected", minimum: 5, amount: 1, currency: "CNY", rate: 7, wantErr: true},
		{name: "zero amount is rejected", minimum: 1, amount: 0, currency: "CNY", rate: 7, wantErr: true},
		{name: "negative amount is rejected", minimum: 1, amount: -1, currency: "CNY", rate: 7, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operation_setting.MinTopUp, operation_setting.USDExchangeRate = tc.minimum, tc.rate
			order, err := calculateFinanceTopup(tc.amount, tc.currency, "default")
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.money, order.Money)
			assert.Equal(t, tc.quota, order.Quota)
		})
	}
}

func TestTopUpInfoReportsOneYuanMinimum(t *testing.T) {
	oldMin := operation_setting.MinTopUp
	t.Cleanup(func() { operation_setting.MinTopUp = oldMin })
	gin.SetMode(gin.TestMode)
	for _, minimum := range []int{0, 1, 5} {
		operation_setting.MinTopUp = minimum
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		GetTopUpInfo(ctx)
		var response struct {
			Success bool `json:"success"`
			Data    struct {
				MinimumCNY int `json:"min_topup_cny"`
			} `json:"data"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success)
		assert.Equal(t, max(1, minimum), response.Data.MinimumCNY)
	}
}

func TestValidateTopUpQuotaReturnsMaximumAmount(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	common.QuotaPerUnit = 500000
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	t.Cleanup(func() {
		common.QuotaPerUnit = oldQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
	})

	maxAmount := decimal.NewFromInt(common.MaxWalletQuota).
		Div(decimal.NewFromFloat(common.QuotaPerUnit)).
		Floor().IntPart()

	_, err := validateTopUpQuota(maxAmount)
	require.NoError(t, err)
	_, err = validateTopUpQuota(maxAmount + 1)
	require.EqualError(t, err, fmt.Sprintf("Top-up amount cannot exceed %d", maxAmount))
}

func TestRequestAmountRejectsTopUpThatCannotBeSettled(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	common.QuotaPerUnit = 500000
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	t.Cleanup(func() {
		common.QuotaPerUnit = oldQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
	})

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	maxAmount := decimal.NewFromInt(common.MaxWalletQuota).
		Div(decimal.NewFromFloat(common.QuotaPerUnit)).
		Floor().IntPart()
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/user/amount",
		strings.NewReader(fmt.Sprintf(`{"amount":%d}`, maxAmount+1)),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	RequestAmount(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, fmt.Sprintf(`{"message":"error","data":"Top-up amount cannot exceed %d","message_key":"Top-up amount cannot exceed {{max}}","message_params":{"max":%d}}`, maxAmount, maxAmount), recorder.Body.String())
}

func TestRequestAmountRejectsTopUpThatWouldOverflowWallet(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	oldDB := model.DB
	common.QuotaPerUnit = 500000
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	model.DB = db
	t.Cleanup(func() {
		common.QuotaPerUnit = oldQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
		model.DB = oldDB
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			require.NoError(t, sqlDB.Close())
		}
	})

	require.NoError(t, model.DB.Create(&model.User{
		Id:       42,
		Username: "topup_capacity_user",
		Quota:    common.MaxWalletQuota - 100_000,
		Status:   common.UserStatusEnabled,
	}).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("id", 42)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/user/amount",
		strings.NewReader(`{"amount":1}`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")

	RequestAmount(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"message":"error","data":"Top-up quota limit exceeded","message_key":"Top-up quota limit exceeded"}`, recorder.Body.String())
}

func TestValidateCreditedQuotaRejectsOverflow(t *testing.T) {
	_, err := validateCreditedQuota(decimal.NewFromInt(int64(common.MaxWalletQuota / 2)))
	require.NoError(t, err)
	_, err = validateCreditedQuota(decimal.Zero)
	require.EqualError(t, err, "Top-up quota must be greater than 0")
	_, err = validateCreditedQuota(decimal.NewFromInt(common.MaxWalletQuota + 1))
	require.EqualError(
		t,
		err,
		"Top-up quota exceeds the range the system can represent",
	)
}

func TestStripeCreditedQuotaIncludesGroupRatio(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	oldTopupGroupRatio := common.TopupGroupRatio2JSONString()
	common.QuotaPerUnit = 500000
	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"vip":2}`))
	t.Cleanup(func() {
		common.QuotaPerUnit = oldQuotaPerUnit
		require.NoError(t, common.UpdateTopupGroupRatioByJSONString(oldTopupGroupRatio))
	})

	_, err := validateCreditedQuota(getStripeCreditedQuota(2147, "vip"))
	require.NoError(t, err)
	_, err = validateCreditedQuota(getStripeCreditedQuota(2148, "vip"))
	require.NoError(t, err)
	_, err = validateCreditedQuota(getStripeCreditedQuota(int64(common.MaxWalletQuota), "vip"))
	require.Error(t, err)

	require.NoError(t, common.UpdateTopupGroupRatioByJSONString(`{"free":0}`))
	assert.True(t, decimal.NewFromInt(500000).Equal(getStripeCreditedQuota(1, "free")))
}
