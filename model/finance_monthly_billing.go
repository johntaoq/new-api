package model

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Finance owns this calendar, independently of the host timezone and display options.
var financeBillingLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

var ErrFinanceMonthNotClosed = errors.New("Monthly statement is not finalized")
var ErrFinanceRateLocked = errors.New("Only next month's exchange rate can be changed")

type FinanceBillingMonth struct {
	BillMonth   string  `json:"bill_month" gorm:"primaryKey;type:varchar(7)"`
	CNYRate     float64 `json:"cny_rate" gorm:"type:decimal(20,8);not null"`
	COSRate     float64 `json:"cos_rate" gorm:"type:decimal(20,8);not null"`
	QuotaPerUSD float64 `json:"quota_per_usd" gorm:"not null"`
	StartsAt    int64   `json:"starts_at"`
	EndsAt      int64   `json:"ends_at"`
	ClosesAt    int64   `json:"closes_at" gorm:"index"`
	FinalizedAt int64   `json:"finalized_at" gorm:"index"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
	OperatorID  int     `json:"operator_id"`
	LeaseOwner  string  `json:"-" gorm:"type:varchar(64)"`
	LeaseUntil  int64   `json:"-"`
	LastError   string  `json:"last_error" gorm:"type:varchar(128)"`
}

// Separate tables deliberately preserve every legacy customer statement.
type FinanceMonthlyBill struct {
	Id                int                       `json:"id"`
	BillMonth         string                    `json:"bill_month" gorm:"type:varchar(7);uniqueIndex:idx_finance_monthly_bill_user"`
	UserId            int                       `json:"user_id" gorm:"uniqueIndex:idx_finance_monthly_bill_user"`
	Username          string                    `json:"username" gorm:"type:varchar(64)"`
	CNYRate           float64                   `json:"cny_rate" gorm:"type:decimal(20,8)"`
	COSRate           float64                   `json:"cos_rate" gorm:"type:decimal(20,8)"`
	ConsumeUSD        float64                   `json:"total_consume_usd" gorm:"type:decimal(28,8)"`
	ConsumeCNY        float64                   `json:"total_consume_cny" gorm:"type:decimal(28,8)"`
	ConsumeCOS        float64                   `json:"total_consume_cos" gorm:"type:decimal(28,8)"`
	PaidConsumeUSD    float64                   `json:"paid_consume_usd" gorm:"type:decimal(28,8)"`
	PaidConsumeCNY    float64                   `json:"paid_consume_cny" gorm:"type:decimal(28,8)"`
	PaidConsumeCOS    float64                   `json:"paid_consume_cos" gorm:"type:decimal(28,8)"`
	GiftConsumeUSD    float64                   `json:"gift_consume_usd" gorm:"type:decimal(28,8)"`
	GiftConsumeCNY    float64                   `json:"gift_consume_cny" gorm:"type:decimal(28,8)"`
	GiftConsumeCOS    float64                   `json:"gift_consume_cos" gorm:"type:decimal(28,8)"`
	ItemCount         int                       `json:"item_count"`
	GeneratedAt       int64                     `json:"generated_at"`
	WalletSummaryJSON string                    `json:"-" gorm:"type:text"`
	WalletSummary     *FinanceBillWalletSummary `json:"wallet_summary,omitempty" gorm:"-"`
}

type FinanceMonthlyBillItem struct {
	Id            int     `json:"id"`
	BillId        int     `json:"bill_id" gorm:"index:idx_finance_bill_item_time,priority:1"`
	OccurredAt    int64   `json:"occurred_at" gorm:"index:idx_finance_bill_item_time,priority:2"`
	EntryType     string  `json:"entry_type" gorm:"type:varchar(32)"`
	TokenDisplay  string  `json:"token_display" gorm:"type:varchar(255)"`
	ModelName     string  `json:"model_name" gorm:"type:varchar(255)"`
	RequestId     string  `json:"request_id" gorm:"type:varchar(128)"`
	QuotaRaw      int     `json:"quota_raw"`
	AmountUSD     float64 `json:"amount_usd" gorm:"type:decimal(28,8)"`
	AmountCNY     float64 `json:"amount_cny" gorm:"type:decimal(28,8)"`
	AmountCOS     float64 `json:"amount_cos" gorm:"type:decimal(28,8)"`
	SourceTable   string  `json:"-" gorm:"type:varchar(40)"`
	SourceId      int     `json:"-"`
	BucketType    string  `json:"bucket_type" gorm:"type:varchar(16)"`
	BillingSource string  `json:"billing_source" gorm:"type:varchar(32)"`
	Remark        string  `json:"remark" gorm:"type:text"`
}

// Store the actual export bytes alongside the frozen rows, not a live CSV query.
type FinanceMonthlyBillFile struct {
	BillId  int    `gorm:"primaryKey;autoIncrement:false"`
	Content []byte `json:"-"`
	SHA256  string `gorm:"type:varchar(64);not null"`
}

func FinanceMonthlyModels() []any {
	return []any{&FinanceBillingMonth{}, &FinanceMonthlyBill{}, &FinanceMonthlyBillItem{}, &FinanceMonthlyBillFile{}}
}

func financeMonthStart(now time.Time) time.Time {
	local := now.In(financeBillingLocation)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, financeBillingLocation)
}

func newFinanceBillingMonth(start time.Time, rate, cos, quota float64, now time.Time) FinanceBillingMonth {
	end := start.AddDate(0, 1, 0)
	return FinanceBillingMonth{
		BillMonth: start.Format("2006-01"), CNYRate: rate, COSRate: cos, QuotaPerUSD: quota,
		StartsAt: start.Unix(), EndsAt: end.Unix(), ClosesAt: end.Add(30 * time.Minute).Unix(),
		CreatedAt: now.Unix(), UpdatedAt: now.Unix(),
	}
}

func validateFinanceMonthlyRate(rate float64) error {
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate <= 0 || rate > 1000000 {
		return errors.New("Invalid monthly exchange rate")
	}
	if decimal.NewFromFloat(rate).Round(8).InexactFloat64() != rate {
		return errors.New("Monthly exchange rate supports at most 8 decimal places")
	}
	return nil
}

// Bootstrap only the activation month. Missing earlier history is never invented.
// After activation, missed months inherit their preceding rate, including across downtime.
func EnsureFinanceBillingMonths(now time.Time) error {
	current := financeMonthStart(now)
	var previous FinanceBillingMonth
	err := DB.Where("bill_month <= ?", current.Format("2006-01")).Order("bill_month DESC").First(&previous).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		rate := operation_setting.USDExchangeRate
		if err := validateFinanceMonthlyRate(rate); err != nil {
			return err
		}
		if err := validateFinanceMonthlyRate(financeCOSRate()); err != nil {
			return err
		}
		if common.QuotaPerUnit <= 0 || math.IsInf(common.QuotaPerUnit, 0) || math.IsNaN(common.QuotaPerUnit) {
			return errors.New("Invalid quota conversion")
		}
		month := newFinanceBillingMonth(current, rate, financeCOSRate(), common.QuotaPerUnit, now)
		return DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&month).Error
	}
	if err != nil {
		return err
	}
	for start := time.Unix(previous.StartsAt, 0).In(financeBillingLocation).AddDate(0, 1, 0); !start.After(current); start = start.AddDate(0, 1, 0) {
		next := newFinanceBillingMonth(start, previous.CNYRate, previous.COSRate, previous.QuotaPerUSD, now)
		if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&next).Error; err != nil {
			return err
		}
		previous = FinanceBillingMonth{}
		if err := DB.Where("bill_month = ?", next.BillMonth).First(&previous).Error; err != nil {
			return err
		}
	}
	return nil
}

func SetFinanceNextMonthRate(billMonth string, rate float64, operator int, now time.Time) error {
	if err := validateFinanceMonthlyRate(rate); err != nil {
		return err
	}
	current := financeMonthStart(now)
	nextStart := current.AddDate(0, 1, 0)
	if billMonth != nextStart.Format("2006-01") {
		return ErrFinanceRateLocked
	}
	if err := EnsureFinanceBillingMonths(now); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var active FinanceBillingMonth
		if err := lockForUpdate(tx).Where("bill_month = ?", current.Format("2006-01")).First(&active).Error; err != nil {
			return err
		}
		next := newFinanceBillingMonth(nextStart, rate, active.COSRate, active.QuotaPerUSD, now)
		next.OperatorID = operator
		var before FinanceBillingMonth
		err := tx.Where("bill_month = ?", next.BillMonth).First(&before).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "bill_month"}},
			DoUpdates: clause.AssignmentColumns([]string{"cny_rate", "operator_id", "updated_at"}),
		}).Create(&next).Error; err != nil {
			return err
		}
		return recordFinancialAuditTx(tx, RecordFinancialAuditParams{
			Module: "monthly_billing", Action: FinancialAuditActionUpdate, OperatorUserId: operator,
			TargetType: "monthly_exchange_rate", Before: before, After: next,
			Remark: next.BillMonth, CreatedAt: now.Unix(),
		})
	})
}

func ListFinanceBillingMonths() ([]FinanceBillingMonth, error) {
	var months []FinanceBillingMonth
	err := DB.Order("bill_month DESC").Limit(120).Find(&months).Error
	return months, err
}

// Late financial events are carried into the next open billing period, without
// changing their original occurrence time or mutating the upstream ledger.
func financeMonthlyLedgerQuery(db *gorm.DB, month FinanceBillingMonth) *gorm.DB {
	if month.ClosesAt == 0 {
		// Balance reconstruction follows posting time, not a backdated occurrence.
		return db.Where("(CASE WHEN created_at > 0 THEN created_at ELSE occurred_at END) >= ? AND (CASE WHEN created_at > 0 THEN created_at ELSE occurred_at END) < ?", month.StartsAt, month.EndsAt)
	}
	return db.Where(
		"((occurred_at >= ? AND occurred_at < ? AND (created_at = 0 OR created_at < ?)) OR (occurred_at < ? AND created_at >= ? AND created_at < ?))",
		month.StartsAt, month.EndsAt, month.ClosesAt,
		month.StartsAt, month.StartsAt+30*60, month.ClosesAt,
	)
}

func financeMonthlyUserIDs(month FinanceBillingMonth) ([]int, error) {
	seen := map[int]bool{}
	previousMonth := time.Unix(month.StartsAt, 0).In(financeBillingLocation).AddDate(0, -1, 0).Format("2006-01")
	var carriedUsers []int
	if err := finalizedFinanceBills(DB).Where("bill_month = ? AND wallet_summary_json <> ''", previousMonth).Pluck("user_id", &carriedUsers).Error; err != nil {
		return nil, err
	}
	for _, id := range carriedUsers {
		seen[id] = true
	}
	for _, table := range []any{&ChannelCostLedger{}, &UserBalanceLedger{}} {
		var ids []int
		if err := financeMonthlyLedgerQuery(DB.Model(table), month).Distinct("user_id").Pluck("user_id", &ids).Error; err != nil {
			return nil, err
		}
		for _, id := range ids {
			seen[id] = true
		}
	}
	var ids []int
	if err := LOG_DB.Model(&Log{}).Where("created_at >= ? AND created_at < ? AND type IN ?", month.StartsAt, month.EndsAt, []int{LogTypeConsume, LogTypeRefund}).Distinct("user_id").Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		seen[id] = true
	}
	if err := DB.Model(&UserQuotaFunding{}).Where("created_at >= ? AND created_at < ?", month.StartsAt, month.EndsAt).Distinct("user_id").Pluck("user_id", &ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		seen[id] = true
	}
	result := make([]int, 0, len(seen))
	for id := range seen {
		if id > 0 {
			result = append(result, id)
		}
	}
	sort.Ints(result)
	return result, nil
}

func financeMonthlySourceItems(month FinanceBillingMonth, userID int) ([]CustomerMonthlyStatementItem, string, error) {
	if month.QuotaPerUSD <= 0 || math.IsNaN(month.QuotaPerUSD) || math.IsInf(month.QuotaPerUSD, 0) {
		return nil, "", errors.New("Invalid quota conversion")
	}
	var ledgers []ChannelCostLedger
	if err := financeMonthlyLedgerQuery(DB.Where("user_id = ?", userID), month).Order("occurred_at, id").Find(&ledgers).Error; err != nil {
		return nil, "", err
	}
	var balances []UserBalanceLedger
	if err := financeMonthlyLedgerQuery(DB.Where("user_id = ?", userID), month).Order("occurred_at, id").Find(&balances).Error; err != nil {
		return nil, "", err
	}
	var logs []*Log
	if err := LOG_DB.Where("user_id = ? AND created_at >= ? AND created_at < ? AND type IN ?", userID, month.StartsAt, month.EndsAt, []int{LogTypeConsume, LogTypeRefund}).Order("created_at, id").Find(&logs).Error; err != nil {
		return nil, "", err
	}
	// Query coverage independently of the billing window: a late ledger's log must
	// not be counted now as legacy usage and again in its carry-forward statement.
	var coverage []ChannelCostLedger
	if err := DB.Select("log_id, request_id, entry_type").Where("user_id = ?", userID).Find(&coverage).Error; err != nil {
		return nil, "", err
	}
	logIDs, requests := map[string]bool{}, map[string]bool{}
	for _, ledger := range coverage {
		if ledger.LogId > 0 {
			logIDs[fmt.Sprintf("%d|%s", ledger.LogId, ledger.EntryType)] = true
		}
		if ledger.RequestId != "" {
			requests[ledger.RequestId+"|"+ledger.EntryType] = true
		}
	}
	legacyLogs := make([]*Log, 0)
	for _, log := range logs {
		kind := ChannelCostEntryTypeConsume
		if log.Type == LogTypeRefund {
			kind = ChannelCostEntryTypeRefund
		}
		if !logIDs[fmt.Sprintf("%d|%s", log.Id, kind)] && (log.RequestId == "" || !requests[log.RequestId+"|"+kind]) {
			legacyLogs = append(legacyLogs, log)
		}
	}
	var fundings []UserQuotaFunding
	if err := DB.Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, month.StartsAt, month.EndsAt).Find(&fundings).Error; err != nil {
		return nil, "", err
	}
	legacyFundings := make([]UserQuotaFunding, 0)
	var balanceCoverage []UserBalanceLedger
	if err := DB.Where("user_id = ?", userID).Find(&balanceCoverage).Error; err != nil {
		return nil, "", err
	}
	matchedBalances := map[int]bool{}
	for _, funding := range fundings {
		if funding.BusinessType == "revenue_adjustment" || funding.GrantedQuota == 0 ||
			funding.SourceType == QuotaFundingSourceLegacyBalance || funding.SourceType == QuotaFundingSourceLegacyGift {
			continue
		}
		covered := false
		for _, balance := range balanceCoverage {
			if !matchedBalances[balance.Id] && balance.SourceType == funding.SourceType && balance.SourceRefId == funding.SourceRefId &&
				balance.AmountQuota == funding.GrantedQuota &&
				(balance.CreatedAt == funding.CreatedAt || balance.OccurredAt == funding.CreatedAt ||
					(funding.ExternalRef != "" && balance.ExternalRef == funding.ExternalRef)) {
				covered = true
				matchedBalances[balance.Id] = true
				break
			}
		}
		if !covered {
			legacyFundings = append(legacyFundings, funding)
		}
	}
	logMap, tokens, err := loadStatementLogContextByLedgers(ledgers)
	if err != nil {
		return nil, "", err
	}
	legacyTokens, err := getStatementTokenSnapshots(legacyLogs)
	if err != nil {
		return nil, "", err
	}
	channels, err := getStatementChannelSnapshots(legacyLogs)
	if err != nil {
		return nil, "", err
	}
	var user User
	if err := DB.Unscoped().Select("id, username").Where("id = ?", userID).First(&user).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}
	username := firstNonEmpty(user.Username, fmt.Sprintf("user-%d", userID))
	snapshot := customerStatementCurrencySnapshot{QuotaPerUnit: month.QuotaPerUSD, DisplayType: operation_setting.QuotaDisplayTypeUSD, CurrencySymbol: "$", USDToCurrencyRate: 1}
	header := &CustomerMonthlyStatement{BillMonth: month.BillMonth, UserId: userID, UsernameSnapshot: username}
	items, _ := buildCustomerMonthlyStatementItems(header, ledgers, balances, legacyLogs, legacyFundings, logMap, tokens, legacyTokens, channels, snapshot, month.ClosesAt)
	ledgerSources := make(map[int]string, len(ledgers))
	legacySources := make(map[int]string, len(legacyLogs))
	logsByRequest := make(map[string]*Log, len(logs))
	for _, log := range logs {
		if log.RequestId != "" && log.Type == LogTypeConsume {
			logsByRequest[log.RequestId] = log
		}
	}
	for _, ledger := range ledgers {
		ledgerSources[ledger.Id] = firstNonEmpty(ledger.BillingSource, "wallet")
	}
	for _, log := range legacyLogs {
		legacySources[log.Id] = legacyBillingSource(parseStatementOther(log.Other))
	}
	for i := range items {
		row := &items[i]
		if row.SourceTable == CustomerMonthlyStatementSourceTableChannelLedgers {
			row.OperationType = ledgerSources[row.SourceId]
			if row.TokenNameSnapshot == "" && row.RequestId != "" {
				if log := logsByRequest[row.RequestId]; log != nil {
					row.TokenNameSnapshot = log.TokenName
				}
			}
		} else if row.SourceTable == CustomerMonthlyStatementSourceTableLogs {
			row.OperationType = legacySources[row.SourceId]
		}
		// A wallet statement reports spendable quota, not recognized cash revenue.
		row.USDAmount = financeQuotaAmount(int64(row.QuotaRaw), row.QuotaPerUnitSnapshot).InexactFloat64()
	}
	return items, username, nil
}

func financeFixedAmount(usd, rate float64) float64 {
	return decimal.NewFromFloat(usd).Mul(decimal.NewFromFloat(rate)).Round(8).InexactFloat64()
}

func buildFinanceMonthlyBill(month FinanceBillingMonth, userID int, username string, source []CustomerMonthlyStatementItem, now time.Time) (FinanceMonthlyBill, []FinanceMonthlyBillItem) {
	bill := FinanceMonthlyBill{BillMonth: month.BillMonth, UserId: userID, Username: username, CNYRate: month.CNYRate, COSRate: month.COSRate, GeneratedAt: now.Unix(), ItemCount: len(source)}
	items := make([]FinanceMonthlyBillItem, 0, len(source))
	usd, cny, cos, paid, gift := decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	summary := &FinanceBillWalletSummary{Version: 2, QuotaPerUSD: month.QuotaPerUSD}
	topup, granted, adjustment, refund, consumption, credits, debits := decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	for _, row := range source {
		amount := financeQuotaAmount(int64(row.QuotaRaw), row.QuotaPerUnitSnapshot)
		bucket := ""
		if row.PaidQuotaRaw != 0 && row.GiftQuotaRaw != 0 {
			bucket = "mixed"
		} else if row.PaidQuotaRaw != 0 {
			bucket = QuotaFundingTypePaid
		} else if row.GiftQuotaRaw != 0 {
			bucket = QuotaFundingTypeGift
		}
		item := FinanceMonthlyBillItem{
			OccurredAt: row.OccurredAt, EntryType: row.EntryType, TokenDisplay: row.TokenNameSnapshot,
			ModelName: row.ModelName, RequestId: row.RequestId, QuotaRaw: row.QuotaRaw,
			AmountUSD: amount.Round(8).InexactFloat64(), AmountCNY: amount.Mul(decimal.NewFromFloat(month.CNYRate)).Round(8).InexactFloat64(),
			AmountCOS: amount.Mul(decimal.NewFromFloat(month.COSRate)).Round(8).InexactFloat64(), SourceTable: row.SourceTable, SourceId: row.SourceId,
			BucketType: bucket, BillingSource: row.OperationType, Remark: row.ContentSummary,
		}
		items = append(items, item)
		if row.OperationType == "subscription" {
			continue
		}
		if row.EntryType == CustomerMonthlyStatementEntryTypeConsume || row.EntryType == CustomerMonthlyStatementEntryTypeRefund {
			usd = usd.Sub(amount)
			cny = cny.Sub(amount.Mul(decimal.NewFromFloat(month.CNYRate)))
			cos = cos.Sub(amount.Mul(decimal.NewFromFloat(month.COSRate)))
			paid = paid.Sub(financeQuotaAmount(int64(row.PaidQuotaRaw), row.QuotaPerUnitSnapshot))
			gift = gift.Sub(financeQuotaAmount(int64(row.GiftQuotaRaw), row.QuotaPerUnitSnapshot))
			if row.EntryType == CustomerMonthlyStatementEntryTypeConsume {
				consumption = consumption.Sub(amount)
				debits = debits.Sub(amount)
			} else {
				refund = refund.Add(amount)
				credits = credits.Add(amount)
			}
			continue
		}
		if amount.IsPositive() {
			credits = credits.Add(amount)
		} else {
			debits = debits.Sub(amount)
		}
		switch row.EntryType {
		case CustomerMonthlyStatementEntryTypeTopup:
			topup = topup.Add(amount)
		case CustomerMonthlyStatementEntryTypeGift:
			granted = granted.Add(amount)
		case CustomerMonthlyStatementEntryTypeAdjustment:
			adjustment = adjustment.Add(amount)
		}
	}
	bill.ConsumeUSD, bill.ConsumeCNY, bill.ConsumeCOS = usd.InexactFloat64(), cny.InexactFloat64(), cos.InexactFloat64()
	bill.PaidConsumeUSD, bill.GiftConsumeUSD = paid.InexactFloat64(), gift.InexactFloat64()
	bill.PaidConsumeCNY, bill.PaidConsumeCOS = financeFixedAmount(bill.PaidConsumeUSD, month.CNYRate), financeFixedAmount(bill.PaidConsumeUSD, month.COSRate)
	bill.GiftConsumeCNY, bill.GiftConsumeCOS = financeFixedAmount(bill.GiftConsumeUSD, month.CNYRate), financeFixedAmount(bill.GiftConsumeUSD, month.COSRate)
	summary.Topup = financeStatementAmount(topup, month.CNYRate)
	summary.Gift = financeStatementAmount(granted, month.CNYRate)
	summary.Adjustment = financeStatementAmount(adjustment, month.CNYRate)
	summary.Refund = financeStatementAmount(refund, month.CNYRate)
	summary.Consumption = financeStatementAmount(consumption, month.CNYRate)
	summary.FundsIn = financeStatementAmount(credits, month.CNYRate)
	summary.FundsOut = financeStatementAmount(debits, month.CNYRate)
	bill.WalletSummary = summary
	sort.SliceStable(items, func(i, j int) bool {
		left, right := financeBillItemSection(items[i]), financeBillItemSection(items[j])
		if left != right {
			return left < right
		}
		if items[i].OccurredAt != items[j].OccurredAt {
			return items[i].OccurredAt < items[j].OccurredAt
		}
		if items[i].SourceTable != items[j].SourceTable {
			return items[i].SourceTable < items[j].SourceTable
		}
		return items[i].SourceId < items[j].SourceId
	})
	return bill, items
}

func closeFinanceBillingMonth(month FinanceBillingMonth, now time.Time) error {
	owner := common.GetRandomString(32)
	claim := DB.Model(&FinanceBillingMonth{}).Where("bill_month = ? AND finalized_at = 0 AND lease_until <= ?", month.BillMonth, now.Unix()).
		Updates(map[string]any{"lease_owner": owner, "lease_until": now.Add(5 * time.Minute).Unix(), "last_error": ""})
	if claim.Error != nil || claim.RowsAffected == 0 {
		return claim.Error
	}
	// Heartbeats keep long customers from losing their lease. Every write checks
	// ownership under a database lock; an expired worker cannot publish.
	ctx, cancel := context.WithCancel(context.Background())
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				DB.Model(&FinanceBillingMonth{}).Where("bill_month = ? AND lease_owner = ?", month.BillMonth, owner).
					Update("lease_until", time.Now().Add(5*time.Minute).Unix())
			}
		}
	}()
	defer func() {
		cancel()
		<-heartbeatDone
		DB.Model(&FinanceBillingMonth{}).Where("bill_month = ? AND lease_owner = ?", month.BillMonth, owner).
			Updates(map[string]any{"lease_until": 0, "lease_owner": ""})
	}()
	ids, err := financeMonthlyUserIDs(month)
	if err != nil {
		return err
	}
	for _, userID := range ids {
		var count int64
		if err := DB.Model(&FinanceMonthlyBill{}).Where("bill_month = ? AND user_id = ?", month.BillMonth, userID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		source, username, err := financeMonthlySourceItems(month, userID)
		if err != nil {
			return err
		}
		bill, items := buildFinanceMonthlyBill(month, userID, username, source, now)
		if err := attachFinanceWalletBalances(&bill, month, source, now); err != nil {
			return err
		}
		err = DB.Transaction(func(tx *gorm.DB) error {
			var locked FinanceBillingMonth
			if err := lockForUpdate(tx).Where("bill_month = ?", month.BillMonth).First(&locked).Error; err != nil {
				return err
			}
			if locked.LeaseOwner != owner || locked.FinalizedAt != 0 {
				return errors.New("Monthly billing lease was lost")
			}
			var existing int64
			if err := tx.Model(&FinanceMonthlyBill{}).Where("bill_month = ? AND user_id = ?", month.BillMonth, userID).Count(&existing).Error; err != nil || existing > 0 {
				return err
			}
			if err := tx.Create(&bill).Error; err != nil {
				return err
			}
			for i := range items {
				items[i].BillId = bill.Id
			}
			if len(items) > 0 {
				if err := tx.CreateInBatches(&items, 200).Error; err != nil {
					return err
				}
			}
			content, err := financeBillCSV(bill, items, false)
			if err != nil {
				return err
			}
			return tx.Create(&FinanceMonthlyBillFile{BillId: bill.Id, Content: content, SHA256: fmt.Sprintf("%x", sha256.Sum256(content))}).Error
		})
		if err != nil {
			return err
		}
	}
	result := DB.Model(&FinanceBillingMonth{}).Where("bill_month = ? AND lease_owner = ? AND finalized_at = 0", month.BillMonth, owner).
		Updates(map[string]any{"finalized_at": now.Unix(), "updated_at": now.Unix(), "last_error": ""})
	if result.Error == nil && result.RowsAffected != 1 {
		return errors.New("Monthly billing lease was lost")
	}
	return result.Error
}

func RunFinanceMonthlyBilling(now time.Time) error {
	if err := EnsureFinanceBillingMonths(now); err != nil {
		return err
	}
	var due []FinanceBillingMonth
	if err := DB.Where("closes_at <= ? AND finalized_at = 0", now.Unix()).Order("bill_month").Find(&due).Error; err != nil {
		return err
	}
	for _, month := range due {
		if err := closeFinanceBillingMonth(month, now); err != nil {
			DB.Model(&FinanceBillingMonth{}).Where("bill_month = ? AND finalized_at = 0", month.BillMonth).
				Update("last_error", "Monthly billing failed; automatic retry pending")
			return err
		}
	}
	// Upgrade already-fixed bills from their frozen rows only. Never consult live
	// logs or current exchange rates, and never replace an existing file.
	var missing []FinanceMonthlyBill
	if err := DB.Where("id NOT IN (?)", DB.Model(&FinanceMonthlyBillFile{}).Select("bill_id")).Find(&missing).Error; err != nil {
		return err
	}
	for _, bill := range missing {
		var items []FinanceMonthlyBillItem
		if err := DB.Where("bill_id = ?", bill.Id).Order("id").Find(&items).Error; err != nil {
			return err
		}
		content, err := financeBillCSV(bill, items, false)
		if err != nil {
			return err
		}
		file := FinanceMonthlyBillFile{BillId: bill.Id, Content: content, SHA256: fmt.Sprintf("%x", sha256.Sum256(content))}
		if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&file).Error; err != nil {
			return err
		}
	}
	return nil
}

// Called by the finance module only; all query handlers below remain read-only.
func StartFinanceMonthlyBilling(ctx context.Context) {
	go func() {
		for {
			if err := RunFinanceMonthlyBilling(time.Now()); err != nil {
				common.SysError("Finance monthly billing failed: " + err.Error())
			}
			timer := time.NewTimer(time.Until(time.Now().Truncate(time.Minute).Add(time.Minute)))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
}

func finalizedFinanceBills(db *gorm.DB) *gorm.DB {
	return db.Model(&FinanceMonthlyBill{}).Where("bill_month IN (?)",
		DB.Model(&FinanceBillingMonth{}).Select("bill_month").Where("finalized_at > 0"))
}

func ListFixedFinanceBills(month, keyword string, selfID int, page *common.PageInfo) (*common.PageInfo, error) {
	query := finalizedFinanceBills(DB)
	if month != "" {
		if _, err := time.Parse("2006-01", month); err != nil {
			return nil, errors.New("Invalid bill month")
		}
		query = query.Where("bill_month = ?", month)
	}
	if selfID > 0 {
		query = query.Where("user_id = ?", selfID)
	} else if keyword = strings.TrimSpace(keyword); keyword != "" {
		if id, err := strconv.Atoi(keyword); err == nil {
			query = query.Where("user_id = ? OR username = ?", id, keyword)
		} else {
			query = query.Where("username = ?", keyword)
		}
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var bills []FinanceMonthlyBill
	if err := query.Order("bill_month DESC, user_id").Offset(page.GetStartIdx()).Limit(page.GetPageSize()).Find(&bills).Error; err != nil {
		return nil, err
	}
	page.SetTotal(int(total))
	for i := range bills {
		if err := decodeFinanceWalletSummary(&bills[i]); err != nil {
			return nil, err
		}
	}
	page.SetItems(bills)
	return page, nil
}

func GetFixedFinanceBill(month string, userID int) (*FinanceMonthlyBill, error) {
	if _, err := time.Parse("2006-01", month); err != nil || userID <= 0 {
		return nil, errors.New("Invalid bill month or user")
	}
	var bill FinanceMonthlyBill
	err := finalizedFinanceBills(DB).Where("bill_month = ? AND user_id = ?", month, userID).First(&bill).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFinanceMonthNotClosed
	}
	if err == nil {
		err = decodeFinanceWalletSummary(&bill)
	}
	return &bill, err
}

func ListFixedFinanceBillItems(month string, userID int, page *common.PageInfo) (*common.PageInfo, error) {
	bill, err := GetFixedFinanceBill(month, userID)
	if err != nil {
		return nil, err
	}
	var items []FinanceMonthlyBillItem
	if err := DB.Where("bill_id = ?", bill.Id).Order("CASE WHEN entry_type IN ('consume','refund') THEN 1 ELSE 0 END, occurred_at, id").
		Offset(page.GetStartIdx()).Limit(page.GetPageSize()).Find(&items).Error; err != nil {
		return nil, err
	}
	page.SetTotal(bill.ItemCount)
	page.SetItems(items)
	return page, nil
}

func ExportFixedFinanceBill(month string, userID int) ([]byte, error) {
	bill, err := GetFixedFinanceBill(month, userID)
	if err != nil {
		return nil, err
	}
	var file FinanceMonthlyBillFile
	if err := DB.Where("bill_id = ?", bill.Id).First(&file).Error; err != nil {
		return nil, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(file.Content)) != file.SHA256 {
		return nil, errors.New("Statement file integrity check failed")
	}
	return file.Content, nil
}

func CurrentFinanceBill(userID int, now time.Time) (FinanceMonthlyBill, []FinanceMonthlyBillItem, error) {
	if userID <= 0 {
		return FinanceMonthlyBill{}, nil, errors.New("Invalid user")
	}
	var month FinanceBillingMonth
	err := DB.Where("bill_month = ? AND finalized_at = 0", financeMonthStart(now).Format("2006-01")).First(&month).Error
	if err != nil {
		return FinanceMonthlyBill{}, nil, err
	}
	// An interim report ends now; future-dated events must not leak into it.
	month.EndsAt = min(month.EndsAt, now.Unix()+1)
	month.ClosesAt = min(month.ClosesAt, now.Unix()+1)
	source, username, err := financeMonthlySourceItems(month, userID)
	if err != nil {
		return FinanceMonthlyBill{}, nil, err
	}
	bill, items := buildFinanceMonthlyBill(month, userID, username, source, now)
	err = attachFinanceWalletBalances(&bill, month, source, now)
	return bill, items, err
}

func ExportCurrentFinanceBill(userID int, now time.Time) ([]byte, error) {
	bill, items, err := CurrentFinanceBill(userID, now)
	if err != nil {
		return nil, err
	}
	return financeBillCSV(bill, items, true)
}

func financeBillCSV(bill FinanceMonthlyBill, items []FinanceMonthlyBillItem, interim bool) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buffer)
	records := [][]string{
		{"账单月份", "用户", "USD/CNY汇率", "消费金额(CNY)", "状态", "生成时间(Asia/Shanghai)"},
		{bill.BillMonth, financeCSVText(bill.Username), fmt.Sprintf("%.8f", bill.CNYRate),
			fmt.Sprintf("%.8f", bill.ConsumeCNY), "已出账", time.Unix(bill.GeneratedAt, 0).In(financeBillingLocation).Format("2006-01-02 15:04:05")},
		{"时间(Asia/Shanghai)", "类型", "令牌", "模型", "余额变动(CNY)", "请求ID"},
	}
	if interim {
		records[1][4] = "本月明细，未出账"
	}
	if summary := bill.WalletSummary; summary != nil {
		status := "已出账"
		if summary.Reconciliation.USD != 0 {
			status = "已出账，存在待核对差额"
		}
		if interim {
			status = "本月明细，未出账"
		}
		records = [][]string{
			{"账单月份", "用户", "USD/CNY汇率", "上期余额(CNY)", "本月充值(CNY)", "本月赠送(CNY)", "调账净额(CNY)", "消费退回(CNY)", "资金流入合计(CNY)", "资金流出合计(CNY)", "总消费(CNY)", "汇率折算差额(CNY)", "对账差额(CNY)", "本期余额(CNY)", "上期余额依据", "本期余额依据", "状态", "生成时间(Asia/Shanghai)"},
			{bill.BillMonth, financeCSVText(bill.Username), fmt.Sprintf("%.8f", bill.CNYRate),
				fmt.Sprintf("%.2f", summary.Opening.CNY), fmt.Sprintf("%.2f", summary.Topup.CNY),
				fmt.Sprintf("%.2f", summary.Gift.CNY), fmt.Sprintf("%.2f", summary.Adjustment.CNY),
				fmt.Sprintf("%.2f", summary.Refund.CNY), fmt.Sprintf("%.2f", summary.FundsIn.CNY),
				fmt.Sprintf("%.2f", summary.FundsOut.CNY), fmt.Sprintf("%.2f", summary.Consumption.CNY),
				fmt.Sprintf("%.2f", summary.ExchangeChangeCNY), fmt.Sprintf("%.2f", summary.Reconciliation.CNY),
				fmt.Sprintf("%.2f", summary.Closing.CNY), financeStatementBalanceLabel(summary.OpeningBasis),
				financeStatementBalanceLabel(summary.ClosingBasis), status,
				time.Unix(bill.GeneratedAt, 0).In(financeBillingLocation).Format("2006-01-02 15:04:05")},
			{"明细分区", "时间(Asia/Shanghai)", "类型", "钱包", "增加金额(CNY)", "减少金额(CNY)", "余额变动(CNY)", "令牌", "模型", "订单号/请求ID", "备注"},
		}
	}
	if err := writer.WriteAll(records); err != nil {
		return nil, err
	}
	for _, item := range items {
		record := []string{
			time.Unix(item.OccurredAt, 0).In(financeBillingLocation).Format("2006-01-02 15:04:05"),
			financeCSVText(item.EntryType), financeCSVText(item.TokenDisplay), financeCSVText(item.ModelName),
			fmt.Sprintf("%.8f", item.AmountCNY), financeCSVText(item.RequestId),
		}
		if bill.WalletSummary != nil {
			section := "资金流水"
			if financeBillItemSection(item) == 1 {
				section = "API消费明细"
			}
			bucket := financeStatementBucketLabel(item.BucketType)
			amount := item.AmountCNY
			if item.BillingSource == "subscription" {
				bucket = "订阅额度（不扣钱包）"
				amount = 0
			}
			record = []string{section,
				time.Unix(item.OccurredAt, 0).In(financeBillingLocation).Format("2006-01-02 15:04:05"),
				financeStatementEntryTypeLabel(item.EntryType), bucket,
				fmt.Sprintf("%.2f", max(0, amount)), fmt.Sprintf("%.2f", max(0, -amount)), fmt.Sprintf("%.2f", amount),
				financeCSVText(item.TokenDisplay), financeCSVText(item.ModelName), financeCSVText(item.RequestId), financeCSVText(item.Remark)}
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}
