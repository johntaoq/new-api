package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type FinanceStatementAmount struct {
	USD float64 `json:"usd"`
	CNY float64 `json:"cny"`
}

type FinanceBillWalletSummary struct {
	Version           int                    `json:"version"`
	QuotaPerUSD       float64                `json:"quota_per_usd"`
	Opening           FinanceStatementAmount `json:"opening"`
	Closing           FinanceStatementAmount `json:"closing"`
	Topup             FinanceStatementAmount `json:"topup"`
	Gift              FinanceStatementAmount `json:"gift"`
	Adjustment        FinanceStatementAmount `json:"adjustment"`
	Refund            FinanceStatementAmount `json:"refund"`
	Consumption       FinanceStatementAmount `json:"consumption"`
	FundsIn           FinanceStatementAmount `json:"funds_in"`
	FundsOut          FinanceStatementAmount `json:"funds_out"`
	ExchangeChangeCNY float64                `json:"exchange_change_cny"`
	Reconciliation    FinanceStatementAmount `json:"reconciliation"`
	OpeningBasis      string                 `json:"opening_basis"`
	ClosingBasis      string                 `json:"closing_basis"`
}

func financeQuotaAmount(quota int64, quotaPerUSD float64) decimal.Decimal {
	if quotaPerUSD <= 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(quota).Div(decimal.NewFromFloat(quotaPerUSD))
}

func financeStatementAmount(usd decimal.Decimal, rate float64) FinanceStatementAmount {
	return FinanceStatementAmount{
		USD: usd.Round(8).InexactFloat64(),
		CNY: usd.Mul(decimal.NewFromFloat(rate)).Round(8).InexactFloat64(),
	}
}

func financeBillItemSection(item FinanceMonthlyBillItem) int {
	if item.EntryType == CustomerMonthlyStatementEntryTypeConsume || item.EntryType == CustomerMonthlyStatementEntryTypeRefund {
		return 1
	}
	return 0
}

func decodeFinanceWalletSummary(bill *FinanceMonthlyBill) error {
	if bill.WalletSummaryJSON == "" {
		return nil
	}
	return common.UnmarshalJsonStr(bill.WalletSummaryJSON, &bill.WalletSummary)
}

// Audit balances are independent checkpoints. A mismatch remains a discrepancy,
// never an invented adjustment or a modification to the customer's wallet.
func financeWalletBalanceAt(userID int, at int64, quotaPerUSD float64, now time.Time) (decimal.Decimal, string, error) {
	query := DB.Where("target_user_id = ? AND module = ? AND target_type = ?", userID, FinancialAuditModuleUserQuota, FinancialAuditTargetUserWallet)
	var following, preceding []FinancialAuditLog
	if err := query.Session(&gorm.Session{}).Where("created_at >= ? AND before_json <> ''", at).Order("created_at, id").Limit(1).Find(&following).Error; err != nil {
		return decimal.Zero, "", err
	}
	if err := query.Session(&gorm.Session{}).Where("created_at < ? AND after_json <> ''", at).Order("created_at DESC, id DESC").Limit(1).Find(&preceding).Error; err != nil {
		return decimal.Zero, "", err
	}
	anchorTime := now.Unix() + 1
	if anchorTime < at {
		return decimal.Zero, "", errors.New("Wallet checkpoint precedes the requested period")
	}
	distance := anchorTime - at
	var audit *FinancialAuditLog
	forward := false
	if len(following) > 0 && following[0].CreatedAt-at < distance {
		audit = &following[0]
		distance = audit.CreatedAt - at
	}
	if len(preceding) > 0 && at-preceding[0].CreatedAt < distance {
		audit, forward = &preceding[0], true
	}
	basis := "current_wallet_reconstruction"
	var quota int64
	var auditedDelta int64
	if audit != nil {
		var before, after struct {
			Quota *int64 `json:"quota"`
		}
		if err := common.UnmarshalJsonStr(audit.BeforeJSON, &before); err != nil {
			return decimal.Zero, "", err
		}
		if err := common.UnmarshalJsonStr(audit.AfterJSON, &after); err != nil {
			return decimal.Zero, "", err
		}
		if before.Quota == nil || after.Quota == nil {
			return decimal.Zero, "", errors.New("Wallet audit is missing a balance checkpoint")
		}
		quota, anchorTime, basis = *before.Quota, audit.CreatedAt, "wallet_audit_reconstruction"
		if forward {
			quota = *after.Quota
			auditedDelta = *after.Quota - *before.Quota
		}
	} else {
		var user User
		if err := DB.Unscoped().Select("id, quota").Where("id = ?", userID).First(&user).Error; err != nil {
			return decimal.Zero, "", err
		}
		quota = int64(user.Quota)
		if anchorTime == at {
			basis = "current_wallet_snapshot"
		}
	}
	start, end := at, anchorTime
	if forward {
		start, end = anchorTime, at
	}
	if end < start {
		return decimal.Zero, "", errors.New("Wallet checkpoint precedes the requested period")
	}
	movements, _, err := financeMonthlySourceItems(FinanceBillingMonth{StartsAt: start, EndsAt: end, QuotaPerUSD: quotaPerUSD}, userID)
	if err != nil {
		return decimal.Zero, "", err
	}
	net := decimal.Zero
	for _, row := range movements {
		if row.OperationType != "subscription" {
			net = net.Add(financeQuotaAmount(int64(row.QuotaRaw), quotaPerUSD))
		}
	}
	anchor := financeQuotaAmount(quota, quotaPerUSD)
	if forward {
		return anchor.Add(net).Sub(financeQuotaAmount(auditedDelta, quotaPerUSD)), basis, nil
	}
	return anchor.Sub(net), basis, nil
}

func attachFinanceWalletBalances(bill *FinanceMonthlyBill, month FinanceBillingMonth, source []CustomerMonthlyStatementItem, now time.Time) error {
	summary := bill.WalletSummary
	if summary == nil {
		return errors.New("Wallet statement summary is missing")
	}
	opening, openingBasis, err := financeWalletBalanceAt(bill.UserId, month.StartsAt, month.QuotaPerUSD, now)
	if err != nil {
		return err
	}
	closing, closingBasis, err := financeWalletBalanceAt(bill.UserId, month.EndsAt, month.QuotaPerUSD, now)
	if err != nil {
		return err
	}
	summary.Opening = financeStatementAmount(opening, month.CNYRate)
	summary.Closing = financeStatementAmount(closing, month.CNYRate)
	summary.OpeningBasis, summary.ClosingBasis = openingBasis, closingBasis
	previousMonth := time.Unix(month.StartsAt, 0).In(financeBillingLocation).AddDate(0, -1, 0).Format("2006-01")
	var previous FinanceMonthlyBill
	err = finalizedFinanceBills(DB).Where("user_id = ? AND bill_month = ?", bill.UserId, previousMonth).First(&previous).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil {
		if err := decodeFinanceWalletSummary(&previous); err != nil {
			return err
		}
		if previous.WalletSummary != nil {
			opening = decimal.NewFromFloat(previous.WalletSummary.Closing.USD)
			summary.Opening = previous.WalletSummary.Closing
			summary.OpeningBasis = "previous_fixed_statement"
			summary.ExchangeChangeCNY = opening.Mul(decimal.NewFromFloat(month.CNYRate)).Sub(decimal.NewFromFloat(summary.Opening.CNY)).Round(8).InexactFloat64()
		}
	}
	net := decimal.Zero
	for _, row := range source {
		if row.OperationType != "subscription" {
			net = net.Add(financeQuotaAmount(int64(row.QuotaRaw), row.QuotaPerUnitSnapshot))
		}
	}
	summary.Reconciliation = financeStatementAmount(closing.Sub(opening.Add(net)), month.CNYRate)
	content, err := common.Marshal(summary)
	if err != nil {
		return err
	}
	bill.WalletSummaryJSON = string(content)
	return nil
}

func financeStatementBucketLabel(bucket string) string {
	switch bucket {
	case QuotaFundingTypePaid:
		return "付费钱包"
	case QuotaFundingTypeGift:
		return "赠送钱包"
	case "mixed":
		return "付费及赠送钱包"
	default:
		return ""
	}
}

func financeStatementBalanceLabel(basis string) string {
	switch basis {
	case "previous_fixed_statement":
		return "上期固定账单"
	case "wallet_audit_reconstruction":
		return "就近钱包审计余额回溯，需核对历史完整性"
	case "current_wallet_reconstruction":
		return "当前钱包余额回溯，需核对历史完整性"
	case "current_wallet_snapshot":
		return "当前钱包余额快照"
	default:
		return basis
	}
}
