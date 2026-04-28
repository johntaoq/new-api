package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestClaimCardIssueCreatesAndReusesRequest(t *testing.T) {
	truncateTables(t)
	setTestQuotaPerUnit(t, 1000)

	params := ClaimCardIssueParams{
		RequestId:  "order-1001",
		AppId:      "unit-test",
		IssuedTo:   "buyer@example.com",
		Remark:     "paid order",
		AutoCreate: true,
		Template: Redemption{
			Name:                 "Paid $10.00",
			FundingType:          QuotaFundingTypePaid,
			AmountUSD:            10,
			RecognizedRevenueUSD: 10,
			Quota:                10000,
			Remark:               "paid order",
		},
		Operator: FinancialAuditOperator{UserId: 1, UsernameSnapshot: "issuer"},
	}

	first, err := ClaimCardIssue(params)
	require.NoError(t, err)
	require.True(t, first.CreatedNew)
	require.False(t, first.ReusedRequest)
	require.NotEmpty(t, first.Key)
	require.Equal(t, 10000, first.Quota)

	second, err := ClaimCardIssue(params)
	require.NoError(t, err)
	require.False(t, second.CreatedNew)
	require.True(t, second.ReusedRequest)
	require.Equal(t, first.Key, second.Key)
	require.Equal(t, first.RedemptionId, second.RedemptionId)

	var issueCount int64
	require.NoError(t, DB.Model(&CardIssue{}).Count(&issueCount).Error)
	require.EqualValues(t, 1, issueCount)

	var redemptionCount int64
	require.NoError(t, DB.Model(&Redemption{}).Count(&redemptionCount).Error)
	require.EqualValues(t, 1, redemptionCount)
}

func TestClaimCardIssueUsesExistingUnissuedRedemption(t *testing.T) {
	truncateTables(t)
	setTestQuotaPerUnit(t, 1000)

	redemption := &Redemption{
		UserId:      1,
		Key:         "existing-code",
		Status:      common.RedemptionCodeStatusEnabled,
		Name:        "Gift $5.00",
		FundingType: QuotaFundingTypeGift,
		AmountUSD:   5,
		Quota:       5000,
		Remark:      "promo",
		CreatedTime: common.GetTimestamp(),
	}
	redemption.normalizeForPersistence()
	require.NoError(t, DB.Create(redemption).Error)

	result, err := ClaimCardIssue(ClaimCardIssueParams{
		RequestId:  "promo-5001",
		AppId:      "unit-test",
		Remark:     "promo",
		AutoCreate: true,
		Template: Redemption{
			Name:        "Gift $5.00",
			FundingType: QuotaFundingTypeGift,
			AmountUSD:   5,
			Quota:       5000,
			Remark:      "promo",
		},
		Operator: FinancialAuditOperator{UserId: 1, UsernameSnapshot: "issuer"},
	})
	require.NoError(t, err)
	require.False(t, result.CreatedNew)
	require.Equal(t, "existing-code", result.Key)
	require.Equal(t, redemption.Id, result.RedemptionId)
}

func setTestQuotaPerUnit(t *testing.T, value float64) {
	t.Helper()
	previous := common.QuotaPerUnit
	common.QuotaPerUnit = value
	t.Cleanup(func() {
		common.QuotaPerUnit = previous
	})
}
