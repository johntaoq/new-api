package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	UserBalanceDirectionCredit = "credit"
	UserBalanceDirectionDebit  = "debit"
)

type UserBalanceLedger struct {
	Id                          int     `json:"id"`
	UserId                      int     `json:"user_id" gorm:"index:idx_user_balance_ledger_user_month_time,priority:1;index"`
	BillMonth                   string  `json:"bill_month" gorm:"type:varchar(7);index:idx_user_balance_ledger_user_month_time,priority:2;index"`
	OccurredAt                  int64   `json:"occurred_at" gorm:"bigint;index:idx_user_balance_ledger_user_month_time,priority:3;index"`
	BucketType                  string  `json:"bucket_type" gorm:"type:varchar(16);index"`
	EntryType                   string  `json:"entry_type" gorm:"type:varchar(16);index"`
	Direction                   string  `json:"direction" gorm:"type:varchar(16);index"`
	AmountQuota                 int     `json:"amount_quota"`
	AmountUSD                   float64 `json:"amount_usd" gorm:"default:0"`
	QuotaPerUnitSnapshot        float64 `json:"quota_per_unit_snapshot" gorm:"default:0"`
	CurrencyDisplayTypeSnapshot string  `json:"currency_display_type_snapshot" gorm:"type:varchar(16);default:''"`
	CurrencySymbolSnapshot      string  `json:"currency_symbol_snapshot" gorm:"type:varchar(16);default:''"`
	USDToCurrencyRateSnapshot   float64 `json:"usd_to_currency_rate_snapshot" gorm:"default:0"`
	SourceType                  string  `json:"source_type" gorm:"type:varchar(32);index"`
	SourceRefId                 int     `json:"source_ref_id" gorm:"index"`
	SourceName                  string  `json:"source_name" gorm:"type:varchar(255);default:''"`
	ExternalRef                 string  `json:"external_ref" gorm:"type:varchar(128);default:''"`
	OperatorUserId              int     `json:"operator_user_id" gorm:"index;default:0"`
	OperatorUsernameSnapshot    string  `json:"operator_username_snapshot" gorm:"type:varchar(64);default:''"`
	Remark                      string  `json:"remark" gorm:"type:varchar(255);default:''"`
	CreatedAt                   int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt                   int64   `json:"updated_at" gorm:"bigint"`
}

type RecordUserBalanceLedgerParams struct {
	UserId                   int
	OccurredAt               int64
	BucketType               string
	EntryType                string
	AmountQuota              int
	AmountUSD                float64
	QuotaPerUnitSnapshot     float64
	SourceType               string
	SourceRefId              int
	SourceName               string
	ExternalRef              string
	OperatorUserId           int
	OperatorUsernameSnapshot string
	Remark                   string
}

type UserQuotaAdjustmentParams struct {
	UserId                   int
	FundingType              string
	DeltaQuota               int
	SourceType               string
	RevenueUSD               float64
	ExternalRef              string
	OperatorUserId           int
	OperatorUsernameSnapshot string
	Remark                   string
	EntryType                string
	OccurredAt               int64
}

func normalizeUserBalanceBucketType(bucketType string) string {
	if strings.TrimSpace(bucketType) == QuotaFundingTypePaid {
		return QuotaFundingTypePaid
	}
	return QuotaFundingTypeGift
}

func normalizeUserBalanceEntryType(entryType string, bucketType string) string {
	switch strings.TrimSpace(entryType) {
	case CustomerMonthlyStatementEntryTypeTopup:
		return CustomerMonthlyStatementEntryTypeTopup
	case CustomerMonthlyStatementEntryTypeGift:
		return CustomerMonthlyStatementEntryTypeGift
	case CustomerMonthlyStatementEntryTypeAdjustment:
		return CustomerMonthlyStatementEntryTypeAdjustment
	}
	if normalizeUserBalanceBucketType(bucketType) == QuotaFundingTypePaid {
		return CustomerMonthlyStatementEntryTypeTopup
	}
	return CustomerMonthlyStatementEntryTypeGift
}

func normalizeUserBalanceDirection(amountQuota int) string {
	if amountQuota < 0 {
		return UserBalanceDirectionDebit
	}
	return UserBalanceDirectionCredit
}

func normalizeUserBalanceLedgerParams(params RecordUserBalanceLedgerParams) (RecordUserBalanceLedgerParams, customerStatementCurrencySnapshot) {
	snapshot := getCurrentStatementCurrencySnapshot()
	if params.OccurredAt == 0 {
		params.OccurredAt = common.GetTimestamp()
	}
	params.BucketType = normalizeUserBalanceBucketType(params.BucketType)
	params.EntryType = normalizeUserBalanceEntryType(params.EntryType, params.BucketType)
	if params.QuotaPerUnitSnapshot <= 0 {
		params.QuotaPerUnitSnapshot = common.QuotaPerUnit
	}
	if params.AmountUSD == 0 && params.AmountQuota != 0 {
		params.AmountUSD = signedQuotaToUSDWithSnapshot(params.AmountQuota, params.QuotaPerUnitSnapshot)
	}
	params.AmountUSD = roundAccountingAmount(params.AmountUSD)
	params.SourceType = strings.TrimSpace(params.SourceType)
	params.SourceName = strings.TrimSpace(params.SourceName)
	params.ExternalRef = strings.TrimSpace(params.ExternalRef)
	params.OperatorUsernameSnapshot = strings.TrimSpace(params.OperatorUsernameSnapshot)
	params.Remark = strings.TrimSpace(params.Remark)
	return params, snapshot
}

func createUserBalanceLedgerTx(tx *gorm.DB, params RecordUserBalanceLedgerParams) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}
	if params.UserId <= 0 {
		return errors.New("user id is invalid")
	}
	if params.AmountQuota == 0 {
		return nil
	}

	params, snapshot := normalizeUserBalanceLedgerParams(params)
	if params.AmountQuota > 0 && params.AmountUSD < 0 {
		return errors.New("credit ledger amount_usd must not be negative")
	}
	if params.AmountQuota < 0 && params.AmountUSD > 0 {
		return errors.New("debit ledger amount_usd must not be positive")
	}

	now := common.GetTimestamp()
	ledger := &UserBalanceLedger{
		UserId:                      params.UserId,
		BillMonth:                   timestampToBillMonth(params.OccurredAt),
		OccurredAt:                  params.OccurredAt,
		BucketType:                  params.BucketType,
		EntryType:                   params.EntryType,
		Direction:                   normalizeUserBalanceDirection(params.AmountQuota),
		AmountQuota:                 params.AmountQuota,
		AmountUSD:                   params.AmountUSD,
		QuotaPerUnitSnapshot:        params.QuotaPerUnitSnapshot,
		CurrencyDisplayTypeSnapshot: snapshot.DisplayType,
		CurrencySymbolSnapshot:      snapshot.CurrencySymbol,
		USDToCurrencyRateSnapshot:   snapshot.USDToCurrencyRate,
		SourceType:                  params.SourceType,
		SourceRefId:                 params.SourceRefId,
		SourceName:                  params.SourceName,
		ExternalRef:                 params.ExternalRef,
		OperatorUserId:              params.OperatorUserId,
		OperatorUsernameSnapshot:    params.OperatorUsernameSnapshot,
		Remark:                      params.Remark,
		CreatedAt:                   now,
		UpdatedAt:                   now,
	}
	return tx.Create(ledger).Error
}

func buildGrantUserBalanceLedgerParams(params QuotaFundingGrantParams) RecordUserBalanceLedgerParams {
	amountUSD := 0.0
	if params.RevenueUSD > 0 {
		amountUSD = params.RevenueUSD
	}
	return RecordUserBalanceLedgerParams{
		UserId:                   params.UserId,
		OccurredAt:               params.OccurredAt,
		BucketType:               normalizeQuotaFundingType(params.FundingType),
		EntryType:                params.BalanceEntryType,
		AmountQuota:              params.GrantedQuota,
		AmountUSD:                amountUSD,
		QuotaPerUnitSnapshot:     common.QuotaPerUnit,
		SourceType:               params.SourceType,
		SourceRefId:              params.SourceRefId,
		SourceName:               params.SourceName,
		ExternalRef:              params.ExternalRef,
		OperatorUserId:           params.OperatorUserId,
		OperatorUsernameSnapshot: params.OperatorUsernameSnapshot,
		Remark:                   params.Remark,
	}
}

func AdjustUserQuotaWithLedger(params UserQuotaAdjustmentParams) error {
	var user User
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		user, err = adjustUserQuotaWithLedgerTx(tx, params)
		return err
	})
	if err != nil {
		return err
	}
	return updateUserCache(user)
}

func AdjustUserQuotaWithAudit(params UserQuotaAdjustmentParams) error {
	var user User
	err := DB.Transaction(func(tx *gorm.DB) error {
		if params.UserId <= 0 {
			return errors.New("user id is invalid")
		}
		if params.DeltaQuota == 0 {
			return errors.New("delta quota must not be zero")
		}

		var beforeUser User
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", params.UserId).First(&beforeUser).Error; err != nil {
			return err
		}
		beforePayload := buildFinancialAuditUserQuotaSnapshot(beforeUser)

		var err error
		user, err = adjustUserQuotaWithLedgerTx(tx, params)
		if err != nil {
			return err
		}

		fundingType := normalizeQuotaFundingType(params.FundingType)
		sourceType := strings.TrimSpace(params.SourceType)
		if sourceType == "" {
			if params.DeltaQuota > 0 && fundingType == QuotaFundingTypeGift {
				sourceType = QuotaFundingSourceAdminGrant
			} else {
				sourceType = QuotaFundingSourceSystemAdjust
			}
		}

		afterPayload := map[string]any{
			"quota":        user.Quota,
			"paid_quota":   user.PaidQuota,
			"gift_quota":   user.GiftQuota,
			"bucket_type":  fundingType,
			"funding_type": fundingType,
			"delta_quota":  params.DeltaQuota,
			"source_type":  sourceType,
			"remark":       strings.TrimSpace(params.Remark),
		}
		return recordFinancialAuditTx(tx, RecordFinancialAuditParams{
			Module:                   FinancialAuditModuleUserQuota,
			Action:                   FinancialAuditActionAdjust,
			OperatorUserId:           params.OperatorUserId,
			OperatorUsernameSnapshot: params.OperatorUsernameSnapshot,
			TargetType:               FinancialAuditTargetUserWallet,
			TargetId:                 params.UserId,
			TargetUserId:             params.UserId,
			Before:                   beforePayload,
			After:                    afterPayload,
			Remark:                   strings.TrimSpace(params.Remark),
		})
	})
	if err != nil {
		return err
	}
	return updateUserCache(user)
}

func adjustUserQuotaWithLedgerTx(tx *gorm.DB, params UserQuotaAdjustmentParams) (User, error) {
	var user User
	if params.UserId <= 0 {
		return user, errors.New("user id is invalid")
	}
	if params.DeltaQuota == 0 {
		return user, errors.New("delta quota must not be zero")
	}

	fundingType := normalizeQuotaFundingType(params.FundingType)
	sourceType := strings.TrimSpace(params.SourceType)
	if sourceType == "" {
		if params.DeltaQuota > 0 && fundingType == QuotaFundingTypeGift {
			sourceType = QuotaFundingSourceAdminGrant
		} else {
			sourceType = QuotaFundingSourceSystemAdjust
		}
	}

	if params.DeltaQuota > 0 {
		return grantUserQuotaTx(tx, QuotaFundingGrantParams{
			UserId:                   params.UserId,
			FundingType:              fundingType,
			SourceType:               sourceType,
			SourceName:               "",
			ExternalRef:              params.ExternalRef,
			GrantedQuota:             params.DeltaQuota,
			RevenueUSD:               params.RevenueUSD,
			OperatorUserId:           params.OperatorUserId,
			OperatorUsernameSnapshot: params.OperatorUsernameSnapshot,
			Remark:                   params.Remark,
			BalanceEntryType:         normalizeUserBalanceEntryType(params.EntryType, fundingType),
			OccurredAt:               params.OccurredAt,
		})
	}

	if _, err := consumeUserQuotaByFundingTypeTx(tx, params.UserId, -params.DeltaQuota, fundingType); err != nil {
		return user, err
	}
	if err := createUserBalanceLedgerTx(tx, RecordUserBalanceLedgerParams{
		UserId:                   params.UserId,
		OccurredAt:               params.OccurredAt,
		BucketType:               fundingType,
		EntryType:                CustomerMonthlyStatementEntryTypeAdjustment,
		AmountQuota:              params.DeltaQuota,
		AmountUSD:                -signedQuotaToUSDWithSnapshot(-params.DeltaQuota, common.QuotaPerUnit),
		QuotaPerUnitSnapshot:     common.QuotaPerUnit,
		SourceType:               sourceType,
		ExternalRef:              params.ExternalRef,
		OperatorUserId:           params.OperatorUserId,
		OperatorUsernameSnapshot: params.OperatorUsernameSnapshot,
		Remark:                   params.Remark,
	}); err != nil {
		return user, err
	}
	if err := tx.Where("id = ?", params.UserId).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}
