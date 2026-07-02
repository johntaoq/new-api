package model

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"gorm.io/gorm"
)

const (
	CustomerMonthlyStatementStatusFinalized = "finalized"

	CustomerMonthlyStatementEntryTypeConsume    = "consume"
	CustomerMonthlyStatementEntryTypeRefund     = "refund"
	CustomerMonthlyStatementEntryTypeTopup      = "topup"
	CustomerMonthlyStatementEntryTypeGift       = "gift"
	CustomerMonthlyStatementEntryTypeAdjustment = "adjustment"

	CustomerMonthlyStatementSourceTableLogs           = "logs"
	CustomerMonthlyStatementSourceTableFundings       = "user_quota_fundings"
	CustomerMonthlyStatementSourceTableChannelLedgers = "channel_cost_ledgers"
	CustomerMonthlyStatementSourceTableBalanceLedgers = "user_balance_ledgers"
)

type CustomerMonthlyStatement struct {
	Id                           int     `json:"id"`
	StatementNo                  string  `json:"statement_no" gorm:"type:varchar(64);uniqueIndex;default:''"`
	BillMonth                    string  `json:"bill_month" gorm:"type:varchar(7);uniqueIndex:idx_statement_user_month;index;default:''"`
	UserId                       int     `json:"user_id" gorm:"uniqueIndex:idx_statement_user_month;index"`
	UsernameSnapshot             string  `json:"username_snapshot" gorm:"type:varchar(64);index;default:''"`
	PeriodStart                  int64   `json:"period_start" gorm:"bigint"`
	PeriodEnd                    int64   `json:"period_end" gorm:"bigint"`
	Status                       string  `json:"status" gorm:"type:varchar(32);index;default:''"`
	ItemCount                    int     `json:"item_count" gorm:"default:0"`
	TotalConsumeDisplayAmount    float64 `json:"total_consume_display_amount" gorm:"default:0"`
	TotalConsumeUSD              float64 `json:"total_consume_usd" gorm:"default:0"`
	TotalRefundDisplayAmount     float64 `json:"total_refund_display_amount" gorm:"default:0"`
	TotalRefundUSD               float64 `json:"total_refund_usd" gorm:"default:0"`
	TotalTopupDisplayAmount      float64 `json:"total_topup_display_amount" gorm:"default:0"`
	TotalTopupUSD                float64 `json:"total_topup_usd" gorm:"default:0"`
	TotalGiftDisplayAmount       float64 `json:"total_gift_display_amount" gorm:"default:0"`
	TotalGiftUSD                 float64 `json:"total_gift_usd" gorm:"default:0"`
	TotalAdjustmentDisplayAmount float64 `json:"total_adjustment_display_amount" gorm:"default:0"`
	TotalAdjustmentUSD           float64 `json:"total_adjustment_usd" gorm:"default:0"`
	TotalNetDisplayAmount        float64 `json:"total_net_display_amount" gorm:"default:0"`
	TotalNetUSD                  float64 `json:"total_net_usd" gorm:"default:0"`
	QuotaPerUnitSnapshot         float64 `json:"quota_per_unit_snapshot" gorm:"default:0"`
	CurrencyDisplayTypeSnapshot  string  `json:"currency_display_type_snapshot" gorm:"type:varchar(16);default:''"`
	CurrencySymbolSnapshot       string  `json:"currency_symbol_snapshot" gorm:"type:varchar(16);default:''"`
	USDToCurrencyRateSnapshot    float64 `json:"usd_to_currency_rate_snapshot" gorm:"default:0"`
	GeneratedAt                  int64   `json:"generated_at" gorm:"bigint"`
	CreatedAt                    int64   `json:"created_at" gorm:"bigint;index"`
	UpdatedAt                    int64   `json:"updated_at" gorm:"bigint"`
}

type CustomerMonthlyStatementItem struct {
	Id                          int     `json:"id"`
	StatementId                 int     `json:"statement_id" gorm:"index:idx_statement_item_statement_time,priority:1"`
	BillMonth                   string  `json:"bill_month" gorm:"type:varchar(7);index;default:''"`
	UserId                      int     `json:"user_id" gorm:"index"`
	UsernameSnapshot            string  `json:"username_snapshot" gorm:"type:varchar(64);index;default:''"`
	OccurredAt                  int64   `json:"occurred_at" gorm:"bigint;index:idx_statement_item_statement_time,priority:2;index"`
	EntryType                   string  `json:"entry_type" gorm:"type:varchar(32);index;default:''"`
	OperationType               string  `json:"operation_type" gorm:"type:varchar(32);default:''"`
	TokenId                     int     `json:"token_id" gorm:"index"`
	TokenNameSnapshot           string  `json:"token_name_snapshot" gorm:"type:varchar(255);default:''"`
	TokenMasked                 string  `json:"token_masked" gorm:"type:varchar(128);default:''"`
	ModelName                   string  `json:"model_name" gorm:"type:varchar(255);index;default:''"`
	RequestId                   string  `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	ChannelId                   int     `json:"channel_id" gorm:"index"`
	ChannelNameSnapshot         string  `json:"channel_name_snapshot" gorm:"type:varchar(255);default:''"`
	GroupName                   string  `json:"group_name" gorm:"type:varchar(64);index;default:''"`
	PromptTokens                int     `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens            int     `json:"completion_tokens" gorm:"default:0"`
	TotalTokens                 int     `json:"total_tokens" gorm:"default:0"`
	QuotaRaw                    int     `json:"quota_raw" gorm:"default:0"`
	PaidQuotaRaw                int     `json:"paid_quota_raw" gorm:"default:0"`
	GiftQuotaRaw                int     `json:"gift_quota_raw" gorm:"default:0"`
	DisplayCurrencyAmount       float64 `json:"display_currency_amount" gorm:"default:0"`
	USDAmount                   float64 `json:"usd_amount" gorm:"default:0"`
	QuotaPerUnitSnapshot        float64 `json:"quota_per_unit_snapshot" gorm:"default:0"`
	CurrencyDisplayTypeSnapshot string  `json:"currency_display_type_snapshot" gorm:"type:varchar(16);default:''"`
	CurrencySymbolSnapshot      string  `json:"currency_symbol_snapshot" gorm:"type:varchar(16);default:''"`
	USDToCurrencyRateSnapshot   float64 `json:"usd_to_currency_rate_snapshot" gorm:"default:0"`
	ContentSummary              string  `json:"content_summary" gorm:"type:text"`
	SourceTable                 string  `json:"source_table" gorm:"type:varchar(32);index;default:''"`
	SourceId                    int     `json:"source_id" gorm:"index"`
	Status                      string  `json:"status" gorm:"type:varchar(16);index;default:'valid'"`
	CreatedAt                   int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt                   int64   `json:"updated_at" gorm:"bigint"`
}

type customerStatementCurrencySnapshot struct {
	QuotaPerUnit      float64
	DisplayType       string
	CurrencySymbol    string
	USDToCurrencyRate float64
}

type customerStatementTokenSnapshot struct {
	Id   int    `gorm:"column:id"`
	Name string `gorm:"column:name"`
	Key  string `gorm:"column:key"`
}

type customerStatementChannelSnapshot struct {
	Id   int    `gorm:"column:id"`
	Name string `gorm:"column:name"`
}

type CustomerMonthlyStatementBundle struct {
	Statement *CustomerMonthlyStatement `json:"statement"`
	Items     *common.PageInfo          `json:"items"`
}

func ParseCustomerStatementBillMonth(billMonth string) (string, int64, int64, error) {
	normalized := strings.TrimSpace(billMonth)
	if normalized == "" {
		normalized = time.Now().In(time.Local).Format("2006-01")
	}
	monthStart, err := time.ParseInLocation("2006-01", normalized, time.Local)
	if err != nil {
		return "", 0, 0, errors.New("invalid bill_month, expected YYYY-MM")
	}
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Second)
	return monthStart.Format("2006-01"), monthStart.Unix(), monthEnd.Unix(), nil
}

func GenerateCustomerMonthlyStatement(userId int, billMonth string, force bool) (*CustomerMonthlyStatement, error) {
	normalizedBillMonth, periodStart, periodEnd, err := ParseCustomerStatementBillMonth(billMonth)
	if err != nil {
		return nil, err
	}

	if !force {
		var existing CustomerMonthlyStatement
		err = DB.Where("user_id = ? AND bill_month = ?", userId, normalizedBillMonth).First(&existing).Error
		if err == nil {
			return &existing, nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	var user User
	if err := DB.Unscoped().Where("id = ?", userId).First(&user).Error; err != nil {
		return nil, err
	}

	var channelLedgers []ChannelCostLedger
	if err := DB.Where("user_id = ? AND bill_month = ? AND occurred_at >= ? AND occurred_at <= ?", userId, normalizedBillMonth, periodStart, periodEnd).
		Order("occurred_at asc, id asc").
		Find(&channelLedgers).Error; err != nil {
		return nil, err
	}
	var balanceLedgers []UserBalanceLedger
	if err := DB.Where("user_id = ? AND bill_month = ? AND occurred_at >= ? AND occurred_at <= ?", userId, normalizedBillMonth, periodStart, periodEnd).
		Order("occurred_at asc, id asc").
		Find(&balanceLedgers).Error; err != nil {
		return nil, err
	}

	var legacyLogs []*Log
	var legacyFundings []UserQuotaFunding
	if len(channelLedgers) == 0 {
		if err := LOG_DB.Where("user_id = ? AND created_at >= ? AND created_at <= ? AND type IN ?", userId, periodStart, periodEnd, []int{LogTypeConsume, LogTypeRefund}).
			Order("created_at asc, id asc").
			Find(&legacyLogs).Error; err != nil {
			return nil, err
		}
	}
	if len(balanceLedgers) == 0 {
		if err := DB.Where("user_id = ? AND created_at >= ? AND created_at <= ?", userId, periodStart, periodEnd).
			Order("created_at asc, id asc").
			Find(&legacyFundings).Error; err != nil {
			return nil, err
		}
	}

	ledgerLogMap, ledgerTokenSnapshots, err := loadStatementLogContextByLedgers(channelLedgers)
	if err != nil {
		return nil, err
	}
	legacyTokenSnapshots, err := getStatementTokenSnapshots(legacyLogs)
	if err != nil {
		return nil, err
	}
	legacyChannelSnapshots, err := getStatementChannelSnapshots(legacyLogs)
	if err != nil {
		return nil, err
	}

	currencySnapshot := getCurrentStatementCurrencySnapshot()
	now := common.GetTimestamp()
	statementNo := fmt.Sprintf("STM-%s-U%d", strings.ReplaceAll(normalizedBillMonth, "-", ""), userId)

	statement := &CustomerMonthlyStatement{}
	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("user_id = ? AND bill_month = ?", userId, normalizedBillMonth).First(statement).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		statementExists := err == nil
		if !statementExists {
			statement.StatementNo = statementNo
			statement.BillMonth = normalizedBillMonth
			statement.UserId = userId
			statement.CreatedAt = now
		}

		statement.UsernameSnapshot = user.Username
		statement.PeriodStart = periodStart
		statement.PeriodEnd = periodEnd
		statement.Status = CustomerMonthlyStatementStatusFinalized
		statement.QuotaPerUnitSnapshot = currencySnapshot.QuotaPerUnit
		statement.CurrencyDisplayTypeSnapshot = currencySnapshot.DisplayType
		statement.CurrencySymbolSnapshot = currencySnapshot.CurrencySymbol
		statement.USDToCurrencyRateSnapshot = currencySnapshot.USDToCurrencyRate
		statement.GeneratedAt = now
		statement.UpdatedAt = now

		if statementExists {
			if err := tx.Model(statement).Updates(map[string]interface{}{
				"statement_no":                   statement.StatementNo,
				"username_snapshot":              statement.UsernameSnapshot,
				"period_start":                   statement.PeriodStart,
				"period_end":                     statement.PeriodEnd,
				"status":                         statement.Status,
				"quota_per_unit_snapshot":        statement.QuotaPerUnitSnapshot,
				"currency_display_type_snapshot": statement.CurrencyDisplayTypeSnapshot,
				"currency_symbol_snapshot":       statement.CurrencySymbolSnapshot,
				"usd_to_currency_rate_snapshot":  statement.USDToCurrencyRateSnapshot,
				"generated_at":                   statement.GeneratedAt,
				"updated_at":                     statement.UpdatedAt,
			}).Error; err != nil {
				return err
			}
			if err := tx.Where("statement_id = ?", statement.Id).Delete(&CustomerMonthlyStatementItem{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Create(statement).Error; err != nil {
				return err
			}
		}

		items, summary := buildCustomerMonthlyStatementItems(
			statement,
			channelLedgers,
			balanceLedgers,
			legacyLogs,
			legacyFundings,
			ledgerLogMap,
			ledgerTokenSnapshots,
			legacyTokenSnapshots,
			legacyChannelSnapshots,
			currencySnapshot,
			now,
		)
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}

		statement.ItemCount = len(items)
		statement.TotalConsumeDisplayAmount = summary.totalConsumeDisplayAmount
		statement.TotalConsumeUSD = summary.totalConsumeUSD
		statement.TotalRefundDisplayAmount = summary.totalRefundDisplayAmount
		statement.TotalRefundUSD = summary.totalRefundUSD
		statement.TotalTopupDisplayAmount = summary.totalTopupDisplayAmount
		statement.TotalTopupUSD = summary.totalTopupUSD
		statement.TotalGiftDisplayAmount = summary.totalGiftDisplayAmount
		statement.TotalGiftUSD = summary.totalGiftUSD
		statement.TotalAdjustmentDisplayAmount = summary.totalAdjustmentDisplayAmount
		statement.TotalAdjustmentUSD = summary.totalAdjustmentUSD
		statement.TotalNetDisplayAmount = summary.totalNetDisplayAmount
		statement.TotalNetUSD = summary.totalNetUSD

		return tx.Model(statement).Updates(map[string]interface{}{
			"item_count":                      statement.ItemCount,
			"total_consume_display_amount":    statement.TotalConsumeDisplayAmount,
			"total_consume_usd":               statement.TotalConsumeUSD,
			"total_refund_display_amount":     statement.TotalRefundDisplayAmount,
			"total_refund_usd":                statement.TotalRefundUSD,
			"total_topup_display_amount":      statement.TotalTopupDisplayAmount,
			"total_topup_usd":                 statement.TotalTopupUSD,
			"total_gift_display_amount":       statement.TotalGiftDisplayAmount,
			"total_gift_usd":                  statement.TotalGiftUSD,
			"total_adjustment_display_amount": statement.TotalAdjustmentDisplayAmount,
			"total_adjustment_usd":            statement.TotalAdjustmentUSD,
			"total_net_display_amount":        statement.TotalNetDisplayAmount,
			"total_net_usd":                   statement.TotalNetUSD,
			"updated_at":                      statement.UpdatedAt,
			"generated_at":                    statement.GeneratedAt,
		}).Error
	})
	if err != nil {
		return nil, err
	}

	return statement, nil
}

func GetCustomerMonthlyStatementByUserAndMonth(userId int, billMonth string) (*CustomerMonthlyStatement, error) {
	normalizedBillMonth, _, _, err := ParseCustomerStatementBillMonth(billMonth)
	if err != nil {
		return nil, err
	}
	var statement CustomerMonthlyStatement
	if err := DB.Where("user_id = ? AND bill_month = ?", userId, normalizedBillMonth).First(&statement).Error; err != nil {
		return nil, err
	}
	return &statement, nil
}

func GetCustomerMonthlyStatementById(id int, userId int) (*CustomerMonthlyStatement, error) {
	var statement CustomerMonthlyStatement
	tx := DB.Where("id = ?", id)
	if userId > 0 {
		tx = tx.Where("user_id = ?", userId)
	}
	if err := tx.First(&statement).Error; err != nil {
		return nil, err
	}
	return &statement, nil
}

func GetCustomerMonthlyStatementItems(statementId int, userId int, pageInfo *common.PageInfo) ([]*CustomerMonthlyStatementItem, int64, error) {
	var items []*CustomerMonthlyStatementItem
	var total int64

	tx := DB.Model(&CustomerMonthlyStatementItem{}).Where("statement_id = ?", statementId)
	if userId > 0 {
		tx = tx.Where("user_id = ?", userId)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Order("occurred_at desc, id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func GetAllCustomerMonthlyStatementItems(statementId int, userId int) ([]*CustomerMonthlyStatementItem, error) {
	var items []*CustomerMonthlyStatementItem
	tx := DB.Model(&CustomerMonthlyStatementItem{}).Where("statement_id = ?", statementId)
	if userId > 0 {
		tx = tx.Where("user_id = ?", userId)
	}
	if err := tx.Order("occurred_at asc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func ListCustomerMonthlyStatements(userId int, billMonth string, pageInfo *common.PageInfo) ([]*CustomerMonthlyStatement, int64, error) {
	var statements []*CustomerMonthlyStatement
	var total int64

	tx := DB.Model(&CustomerMonthlyStatement{})
	if userId > 0 {
		tx = tx.Where("user_id = ?", userId)
	}
	normalizedBillMonth := strings.TrimSpace(billMonth)
	if normalizedBillMonth != "" {
		tx = tx.Where("bill_month = ?", normalizedBillMonth)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Order("bill_month desc, id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&statements).Error; err != nil {
		return nil, 0, err
	}
	return statements, total, nil
}

func BuildCustomerMonthlyStatementBundle(statement *CustomerMonthlyStatement, userId int, pageInfo *common.PageInfo) (*CustomerMonthlyStatementBundle, error) {
	items, total, err := GetCustomerMonthlyStatementItems(statement.Id, userId, pageInfo)
	if err != nil {
		return nil, err
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	return &CustomerMonthlyStatementBundle{
		Statement: statement,
		Items:     pageInfo,
	}, nil
}

func ExportCustomerMonthlyStatementCSV(statement *CustomerMonthlyStatement, userId int) ([]byte, string, error) {
	items, err := GetAllCustomerMonthlyStatementItems(statement.Id, userId)
	if err != nil {
		return nil, "", err
	}

	buffer := &bytes.Buffer{}
	buffer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(buffer)

	header := []string{
		"账单月份",
		"时间",
		"令牌名称",
		"令牌脱敏Key",
		"消费类型",
		"操作类型",
		"模型",
		"花费",
		"等价USD",
		"请求ID",
		"渠道",
		"分组",
		"输入Tokens",
		"输出Tokens",
		"总Tokens",
		"说明",
	}
	if err := writer.Write(header); err != nil {
		return nil, "", err
	}

	for _, item := range items {
		displayAmount := formatStatementCSVDisplayAmount(item.DisplayCurrencyAmount, item.CurrencyDisplayTypeSnapshot, item.CurrencySymbolSnapshot)
		record := []string{
			item.BillMonth,
			time.Unix(item.OccurredAt, 0).In(time.Local).Format("2006-01-02 15:04:05"),
			item.TokenNameSnapshot,
			item.TokenMasked,
			item.EntryType,
			item.OperationType,
			item.ModelName,
			displayAmount,
			fmt.Sprintf("%.6f", item.USDAmount),
			item.RequestId,
			item.ChannelNameSnapshot,
			item.GroupName,
			fmt.Sprintf("%d", item.PromptTokens),
			fmt.Sprintf("%d", item.CompletionTokens),
			fmt.Sprintf("%d", item.TotalTokens),
			item.ContentSummary,
		}
		if err := writer.Write(record); err != nil {
			return nil, "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}
	filename := fmt.Sprintf("statement-%s-%s.csv", sanitizeStatementFilenamePart(statement.UsernameSnapshot, fmt.Sprintf("user-%d", statement.UserId)), statement.BillMonth)
	return buffer.Bytes(), filename, nil
}

type customerStatementSummary struct {
	totalConsumeDisplayAmount    float64
	totalConsumeUSD              float64
	totalRefundDisplayAmount     float64
	totalRefundUSD               float64
	totalTopupDisplayAmount      float64
	totalTopupUSD                float64
	totalGiftDisplayAmount       float64
	totalGiftUSD                 float64
	totalAdjustmentDisplayAmount float64
	totalAdjustmentUSD           float64
	totalNetDisplayAmount        float64
	totalNetUSD                  float64
}

func (summary *customerStatementSummary) addEntry(entryType string, displayAmount float64, usdAmount float64) {
	switch entryType {
	case CustomerMonthlyStatementEntryTypeConsume:
		summary.totalConsumeDisplayAmount += math.Abs(displayAmount)
		summary.totalConsumeUSD += math.Abs(usdAmount)
	case CustomerMonthlyStatementEntryTypeRefund:
		summary.totalRefundDisplayAmount += math.Abs(displayAmount)
		summary.totalRefundUSD += math.Abs(usdAmount)
	case CustomerMonthlyStatementEntryTypeTopup:
		summary.totalTopupDisplayAmount += math.Abs(displayAmount)
		summary.totalTopupUSD += math.Abs(usdAmount)
	case CustomerMonthlyStatementEntryTypeGift:
		summary.totalGiftDisplayAmount += math.Abs(displayAmount)
		summary.totalGiftUSD += math.Abs(usdAmount)
	case CustomerMonthlyStatementEntryTypeAdjustment:
		summary.totalAdjustmentDisplayAmount += displayAmount
		summary.totalAdjustmentUSD += usdAmount
	}
	summary.totalNetDisplayAmount += displayAmount
	summary.totalNetUSD += usdAmount
}

func buildCustomerMonthlyStatementItems(
	statement *CustomerMonthlyStatement,
	channelLedgers []ChannelCostLedger,
	balanceLedgers []UserBalanceLedger,
	legacyLogs []*Log,
	legacyFundings []UserQuotaFunding,
	ledgerLogMap map[int]Log,
	ledgerTokenSnapshots map[int]customerStatementTokenSnapshot,
	legacyTokenSnapshots map[int]customerStatementTokenSnapshot,
	legacyChannelSnapshots map[int]customerStatementChannelSnapshot,
	currencySnapshot customerStatementCurrencySnapshot,
	now int64,
) ([]CustomerMonthlyStatementItem, customerStatementSummary) {
	items := make([]CustomerMonthlyStatementItem, 0, len(channelLedgers)+len(balanceLedgers)+len(legacyLogs)+len(legacyFundings))
	summary := customerStatementSummary{}

	for _, ledger := range channelLedgers {
		entryType := strings.TrimSpace(ledger.EntryType)
		if entryType != ChannelCostEntryTypeConsume && entryType != ChannelCostEntryTypeRefund {
			continue
		}
		statementEntryType := CustomerMonthlyStatementEntryTypeConsume
		sign := -1
		if entryType == ChannelCostEntryTypeRefund {
			statementEntryType = CustomerMonthlyStatementEntryTypeRefund
			sign = 1
		}
		signedQuota := sign * ledger.ActualQuota
		signedPaidQuota := sign * ledger.PaidQuotaUsed
		signedGiftQuota := sign * ledger.GiftQuotaUsed
		usdAmount := roundCurrencyAmount(float64(sign) * ledger.InternalEquivalentUSD)
		if usdAmount == 0 {
			usdAmount = quotaToUSD(signedQuota, currencySnapshot.QuotaPerUnit)
		}
		displayAmount := roundCurrencyAmount(financeCOSAmountFromUSD(usdAmount))
		if currencySnapshot.DisplayType == operation_setting.QuotaDisplayTypeTokens {
			displayAmount = quotaToDisplayAmount(signedQuota, currencySnapshot)
		}

		logRecord, hasLog := ledgerLogMap[ledger.LogId]
		tokenNameSnapshot := ""
		tokenMasked := ""
		operationType := ""
		contentSummary := ""
		groupName := ""
		if hasLog {
			other := parseStatementOther(logRecord.Other)
			operationType = deriveStatementOperationType(&logRecord, other)
			contentSummary = deriveStatementContentSummary(statementEntryType, operationType, &logRecord, other)
			groupName = logRecord.Group
			tokenSnapshot := ledgerTokenSnapshots[logRecord.TokenId]
			tokenNameSnapshot = firstNonEmpty(logRecord.TokenName, tokenSnapshot.Name)
			tokenMasked = MaskTokenKey(tokenSnapshot.Key)
		}
		if operationType == "" {
			operationType = firstNonEmpty(strings.TrimSpace(ledger.BillingSource), "general")
		}
		if contentSummary == "" {
			contentSummary = buildChannelLedgerStatementContentSummary(statementEntryType, ledger)
		}

		items = append(items, CustomerMonthlyStatementItem{
			StatementId:                 statement.Id,
			BillMonth:                   statement.BillMonth,
			UserId:                      statement.UserId,
			UsernameSnapshot:            statement.UsernameSnapshot,
			OccurredAt:                  ledger.OccurredAt,
			EntryType:                   statementEntryType,
			OperationType:               operationType,
			TokenId:                     logRecord.TokenId,
			TokenNameSnapshot:           tokenNameSnapshot,
			TokenMasked:                 tokenMasked,
			ModelName:                   firstNonEmpty(logRecord.ModelName, ledger.OriginModelName),
			RequestId:                   firstNonEmpty(logRecord.RequestId, ledger.RequestId),
			ChannelId:                   ledger.ChannelId,
			ChannelNameSnapshot:         ledger.ChannelNameSnapshot,
			GroupName:                   groupName,
			PromptTokens:                ledger.PromptTokens,
			CompletionTokens:            ledger.CompletionTokens,
			TotalTokens:                 ledger.TotalTokens,
			QuotaRaw:                    signedQuota,
			PaidQuotaRaw:                signedPaidQuota,
			GiftQuotaRaw:                signedGiftQuota,
			DisplayCurrencyAmount:       displayAmount,
			USDAmount:                   usdAmount,
			QuotaPerUnitSnapshot:        currencySnapshot.QuotaPerUnit,
			CurrencyDisplayTypeSnapshot: currencySnapshot.DisplayType,
			CurrencySymbolSnapshot:      currencySnapshot.CurrencySymbol,
			USDToCurrencyRateSnapshot:   currencySnapshot.USDToCurrencyRate,
			ContentSummary:              contentSummary,
			SourceTable:                 CustomerMonthlyStatementSourceTableChannelLedgers,
			SourceId:                    ledger.Id,
			Status:                      "valid",
			CreatedAt:                   now,
			UpdatedAt:                   now,
		})
		summary.addEntry(statementEntryType, displayAmount, usdAmount)
	}

	for _, ledger := range balanceLedgers {
		entryType := normalizeUserBalanceEntryType(ledger.EntryType, ledger.BucketType)
		quotaPerUnit := ledger.QuotaPerUnitSnapshot
		if quotaPerUnit <= 0 {
			quotaPerUnit = currencySnapshot.QuotaPerUnit
		}
		usdAmount := roundCurrencyAmount(ledger.AmountUSD)
		if usdAmount == 0 && ledger.AmountQuota != 0 {
			usdAmount = quotaToUSD(ledger.AmountQuota, quotaPerUnit)
		}
		displayAmount := roundCurrencyAmount(financeCOSAmountFromUSD(usdAmount))
		if currencySnapshot.DisplayType == operation_setting.QuotaDisplayTypeTokens {
			displayAmount = quotaToDisplayAmount(ledger.AmountQuota, currencySnapshot)
		}
		paidQuotaRaw := 0
		giftQuotaRaw := 0
		if ledger.BucketType == QuotaFundingTypePaid {
			paidQuotaRaw = ledger.AmountQuota
		} else {
			giftQuotaRaw = ledger.AmountQuota
		}
		items = append(items, CustomerMonthlyStatementItem{
			StatementId:                 statement.Id,
			BillMonth:                   statement.BillMonth,
			UserId:                      statement.UserId,
			UsernameSnapshot:            statement.UsernameSnapshot,
			OccurredAt:                  ledger.OccurredAt,
			EntryType:                   entryType,
			OperationType:               financeSourceLabel(ledger.SourceType),
			TokenId:                     0,
			TokenNameSnapshot:           "",
			TokenMasked:                 "",
			ModelName:                   "",
			RequestId:                   firstNonEmpty(strings.TrimSpace(ledger.ExternalRef), strings.TrimSpace(ledger.SourceName)),
			ChannelId:                   0,
			ChannelNameSnapshot:         "",
			GroupName:                   "",
			PromptTokens:                0,
			CompletionTokens:            0,
			TotalTokens:                 0,
			QuotaRaw:                    ledger.AmountQuota,
			PaidQuotaRaw:                paidQuotaRaw,
			GiftQuotaRaw:                giftQuotaRaw,
			DisplayCurrencyAmount:       displayAmount,
			USDAmount:                   usdAmount,
			QuotaPerUnitSnapshot:        quotaPerUnit,
			CurrencyDisplayTypeSnapshot: currencySnapshot.DisplayType,
			CurrencySymbolSnapshot:      currencySnapshot.CurrencySymbol,
			USDToCurrencyRateSnapshot:   currencySnapshot.USDToCurrencyRate,
			ContentSummary:              buildBalanceLedgerStatementContentSummary(ledger),
			SourceTable:                 CustomerMonthlyStatementSourceTableBalanceLedgers,
			SourceId:                    ledger.Id,
			Status:                      "valid",
			CreatedAt:                   now,
			UpdatedAt:                   now,
		})
		summary.addEntry(entryType, displayAmount, usdAmount)
	}

	for _, logRecord := range legacyLogs {
		entryType := convertLogTypeToStatementEntryType(logRecord.Type)
		if entryType == "" {
			continue
		}
		signedQuota := -logRecord.Quota
		if entryType == CustomerMonthlyStatementEntryTypeRefund {
			signedQuota = logRecord.Quota
		}
		usdAmount := quotaToUSD(signedQuota, currencySnapshot.QuotaPerUnit)
		displayAmount := quotaToDisplayAmount(signedQuota, currencySnapshot)
		other := parseStatementOther(logRecord.Other)
		operationType := deriveStatementOperationType(logRecord, other)

		tokenSnapshot := legacyTokenSnapshots[logRecord.TokenId]
		tokenNameSnapshot := logRecord.TokenName
		if tokenNameSnapshot == "" {
			tokenNameSnapshot = tokenSnapshot.Name
		}

		items = append(items, CustomerMonthlyStatementItem{
			StatementId:                 statement.Id,
			BillMonth:                   statement.BillMonth,
			UserId:                      statement.UserId,
			UsernameSnapshot:            firstNonEmpty(logRecord.Username, statement.UsernameSnapshot),
			OccurredAt:                  logRecord.CreatedAt,
			EntryType:                   entryType,
			OperationType:               operationType,
			TokenId:                     logRecord.TokenId,
			TokenNameSnapshot:           tokenNameSnapshot,
			TokenMasked:                 MaskTokenKey(tokenSnapshot.Key),
			ModelName:                   logRecord.ModelName,
			RequestId:                   logRecord.RequestId,
			ChannelId:                   logRecord.ChannelId,
			ChannelNameSnapshot:         legacyChannelSnapshots[logRecord.ChannelId].Name,
			GroupName:                   logRecord.Group,
			PromptTokens:                logRecord.PromptTokens,
			CompletionTokens:            logRecord.CompletionTokens,
			TotalTokens:                 logRecord.PromptTokens + logRecord.CompletionTokens,
			QuotaRaw:                    signedQuota,
			DisplayCurrencyAmount:       displayAmount,
			USDAmount:                   usdAmount,
			QuotaPerUnitSnapshot:        currencySnapshot.QuotaPerUnit,
			CurrencyDisplayTypeSnapshot: currencySnapshot.DisplayType,
			CurrencySymbolSnapshot:      currencySnapshot.CurrencySymbol,
			USDToCurrencyRateSnapshot:   currencySnapshot.USDToCurrencyRate,
			ContentSummary:              deriveStatementContentSummary(entryType, operationType, logRecord, other),
			SourceTable:                 CustomerMonthlyStatementSourceTableLogs,
			SourceId:                    logRecord.Id,
			Status:                      "valid",
			CreatedAt:                   now,
			UpdatedAt:                   now,
		})
		summary.addEntry(entryType, displayAmount, usdAmount)
	}

	for _, funding := range legacyFundings {
		entryType := customerFundingToStatementEntryType(funding)
		if entryType == "" {
			continue
		}

		signedQuota := funding.GrantedQuota
		usdAmount := quotaToUSD(signedQuota, fundingQuotaSnapshot(funding))
		displayAmount := fundingToStatementDisplayAmount(signedQuota, usdAmount, currencySnapshot)
		paidQuotaRaw := 0
		giftQuotaRaw := 0
		if funding.FundingType == QuotaFundingTypePaid {
			paidQuotaRaw = signedQuota
		} else {
			giftQuotaRaw = signedQuota
		}

		items = append(items, CustomerMonthlyStatementItem{
			StatementId:                 statement.Id,
			BillMonth:                   statement.BillMonth,
			UserId:                      statement.UserId,
			UsernameSnapshot:            statement.UsernameSnapshot,
			OccurredAt:                  funding.CreatedAt,
			EntryType:                   entryType,
			OperationType:               financeSourceLabel(funding.SourceType),
			TokenId:                     0,
			TokenNameSnapshot:           "",
			TokenMasked:                 "",
			ModelName:                   "",
			RequestId:                   firstNonEmpty(strings.TrimSpace(funding.ExternalRef), strings.TrimSpace(funding.SourceName)),
			ChannelId:                   0,
			ChannelNameSnapshot:         "",
			GroupName:                   "",
			PromptTokens:                0,
			CompletionTokens:            0,
			TotalTokens:                 0,
			QuotaRaw:                    signedQuota,
			PaidQuotaRaw:                paidQuotaRaw,
			GiftQuotaRaw:                giftQuotaRaw,
			DisplayCurrencyAmount:       displayAmount,
			USDAmount:                   usdAmount,
			QuotaPerUnitSnapshot:        fundingQuotaSnapshot(funding),
			CurrencyDisplayTypeSnapshot: currencySnapshot.DisplayType,
			CurrencySymbolSnapshot:      currencySnapshot.CurrencySymbol,
			USDToCurrencyRateSnapshot:   currencySnapshot.USDToCurrencyRate,
			ContentSummary:              buildFundingStatementContentSummary(funding),
			SourceTable:                 CustomerMonthlyStatementSourceTableFundings,
			SourceId:                    funding.Id,
			Status:                      "valid",
			CreatedAt:                   now,
			UpdatedAt:                   now,
		})
		summary.addEntry(entryType, displayAmount, usdAmount)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].OccurredAt != items[j].OccurredAt {
			return items[i].OccurredAt < items[j].OccurredAt
		}
		if items[i].SourceTable != items[j].SourceTable {
			return items[i].SourceTable < items[j].SourceTable
		}
		return items[i].SourceId < items[j].SourceId
	})

	summary.totalConsumeDisplayAmount = roundCurrencyAmount(summary.totalConsumeDisplayAmount)
	summary.totalConsumeUSD = roundCurrencyAmount(summary.totalConsumeUSD)
	summary.totalRefundDisplayAmount = roundCurrencyAmount(summary.totalRefundDisplayAmount)
	summary.totalRefundUSD = roundCurrencyAmount(summary.totalRefundUSD)
	summary.totalTopupDisplayAmount = roundCurrencyAmount(summary.totalTopupDisplayAmount)
	summary.totalTopupUSD = roundCurrencyAmount(summary.totalTopupUSD)
	summary.totalGiftDisplayAmount = roundCurrencyAmount(summary.totalGiftDisplayAmount)
	summary.totalGiftUSD = roundCurrencyAmount(summary.totalGiftUSD)
	summary.totalAdjustmentDisplayAmount = roundCurrencyAmount(summary.totalAdjustmentDisplayAmount)
	summary.totalAdjustmentUSD = roundCurrencyAmount(summary.totalAdjustmentUSD)
	summary.totalNetDisplayAmount = roundCurrencyAmount(summary.totalNetDisplayAmount)
	summary.totalNetUSD = roundCurrencyAmount(summary.totalNetUSD)
	return items, summary
}

func loadStatementLogContextByLedgers(ledgers []ChannelCostLedger) (map[int]Log, map[int]customerStatementTokenSnapshot, error) {
	logMap := make(map[int]Log)
	logIDs := make([]int, 0)
	seenLogIDs := make(map[int]struct{})
	for _, ledger := range ledgers {
		if ledger.LogId <= 0 {
			continue
		}
		if _, ok := seenLogIDs[ledger.LogId]; ok {
			continue
		}
		seenLogIDs[ledger.LogId] = struct{}{}
		logIDs = append(logIDs, ledger.LogId)
	}
	if len(logIDs) == 0 {
		return logMap, map[int]customerStatementTokenSnapshot{}, nil
	}

	var logs []Log
	if err := LOG_DB.Where("id IN ?", logIDs).Find(&logs).Error; err != nil {
		return nil, nil, err
	}
	logPointers := make([]*Log, 0, len(logs))
	for _, logRecord := range logs {
		logMap[logRecord.Id] = logRecord
		logRecordCopy := logRecord
		logPointers = append(logPointers, &logRecordCopy)
	}
	tokenSnapshots, err := getStatementTokenSnapshots(logPointers)
	if err != nil {
		return nil, nil, err
	}
	return logMap, tokenSnapshots, nil
}

func getStatementTokenSnapshots(logs []*Log) (map[int]customerStatementTokenSnapshot, error) {
	tokenIds := make([]int, 0)
	seen := make(map[int]struct{})
	for _, logRecord := range logs {
		if logRecord.TokenId <= 0 {
			continue
		}
		if _, ok := seen[logRecord.TokenId]; ok {
			continue
		}
		seen[logRecord.TokenId] = struct{}{}
		tokenIds = append(tokenIds, logRecord.TokenId)
	}
	if len(tokenIds) == 0 {
		return map[int]customerStatementTokenSnapshot{}, nil
	}

	var tokens []customerStatementTokenSnapshot
	if err := DB.Unscoped().Table("tokens").Select("id, name, key").Where("id IN ?", tokenIds).Find(&tokens).Error; err != nil {
		return nil, err
	}

	result := make(map[int]customerStatementTokenSnapshot, len(tokens))
	for _, tokenSnapshot := range tokens {
		result[tokenSnapshot.Id] = tokenSnapshot
	}
	return result, nil
}

func getStatementChannelSnapshots(logs []*Log) (map[int]customerStatementChannelSnapshot, error) {
	channelIds := make([]int, 0)
	seen := make(map[int]struct{})
	for _, logRecord := range logs {
		if logRecord.ChannelId <= 0 {
			continue
		}
		if _, ok := seen[logRecord.ChannelId]; ok {
			continue
		}
		seen[logRecord.ChannelId] = struct{}{}
		channelIds = append(channelIds, logRecord.ChannelId)
	}
	if len(channelIds) == 0 {
		return map[int]customerStatementChannelSnapshot{}, nil
	}

	var channels []customerStatementChannelSnapshot
	if err := DB.Table("channels").Select("id, name").Where("id IN ?", channelIds).Find(&channels).Error; err != nil {
		return nil, err
	}

	result := make(map[int]customerStatementChannelSnapshot, len(channels))
	for _, channelSnapshot := range channels {
		result[channelSnapshot.Id] = channelSnapshot
	}
	return result, nil
}

func getCurrentStatementCurrencySnapshot() customerStatementCurrencySnapshot {
	displayType := operation_setting.GetQuotaDisplayType()
	usdToCurrencyRate := 0.0
	if displayType != operation_setting.QuotaDisplayTypeTokens {
		usdToCurrencyRate = operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	}
	return customerStatementCurrencySnapshot{
		QuotaPerUnit:      common.QuotaPerUnit,
		DisplayType:       displayType,
		CurrencySymbol:    operation_setting.GetCurrencySymbol(),
		USDToCurrencyRate: usdToCurrencyRate,
	}
}

func buildChannelLedgerStatementContentSummary(entryType string, ledger ChannelCostLedger) string {
	prefix := "消费"
	if entryType == CustomerMonthlyStatementEntryTypeRefund {
		prefix = "退款"
	}
	if modelName := strings.TrimSpace(ledger.OriginModelName); modelName != "" {
		return fmt.Sprintf("%s - %s", prefix, modelName)
	}
	if billingSource := strings.TrimSpace(ledger.BillingSource); billingSource != "" {
		return fmt.Sprintf("%s - %s", prefix, billingSource)
	}
	return prefix
}

func quotaToUSD(quota int, quotaPerUnit float64) float64 {
	if quotaPerUnit <= 0 {
		return 0
	}
	return roundCurrencyAmount(float64(quota) / quotaPerUnit)
}

func signedQuotaToUSDWithSnapshot(quota int, quotaPerUnit float64) float64 {
	return quotaToUSD(quota, quotaPerUnit)
}

func quotaToDisplayAmount(quota int, snapshot customerStatementCurrencySnapshot) float64 {
	if snapshot.DisplayType == operation_setting.QuotaDisplayTypeTokens {
		return roundCurrencyAmount(float64(quota))
	}
	return roundCurrencyAmount(quotaToUSD(quota, snapshot.QuotaPerUnit) * snapshot.USDToCurrencyRate)
}

func fundingToStatementDisplayAmount(quota int, usdAmount float64, snapshot customerStatementCurrencySnapshot) float64 {
	if snapshot.DisplayType == operation_setting.QuotaDisplayTypeTokens {
		return roundCurrencyAmount(float64(quota))
	}
	return roundCurrencyAmount(usdAmount * snapshot.USDToCurrencyRate)
}

func roundCurrencyAmount(value float64) float64 {
	return math.Round(value*1e6) / 1e6
}

func convertLogTypeToStatementEntryType(logType int) string {
	switch logType {
	case LogTypeConsume:
		return CustomerMonthlyStatementEntryTypeConsume
	case LogTypeRefund:
		return CustomerMonthlyStatementEntryTypeRefund
	default:
		return ""
	}
}

func parseStatementOther(other string) map[string]interface{} {
	if strings.TrimSpace(other) == "" {
		return map[string]interface{}{}
	}
	otherMap, err := common.StrToMap(other)
	if err != nil || otherMap == nil {
		return map[string]interface{}{}
	}
	return otherMap
}

func deriveStatementOperationType(logRecord *Log, other map[string]interface{}) string {
	switch {
	case getBoolFromMap(other, "image_generation_call"):
		return "image_generation"
	case getBoolFromMap(other, "image"):
		return "image"
	case getBoolFromMap(other, "audio_input_seperate_price"):
		return "audio"
	case getBoolFromMap(other, "file_search"):
		return "file_search"
	case getBoolFromMap(other, "web_search"):
		return "web_search"
	case getStringFromMap(other, "task_id") != "":
		return "task"
	case logRecord.ModelName != "":
		return "chat"
	default:
		return "general"
	}
}

func customerFundingToStatementEntryType(funding UserQuotaFunding) string {
	if funding.FundingType == QuotaFundingTypePaid {
		return CustomerMonthlyStatementEntryTypeTopup
	}
	return CustomerMonthlyStatementEntryTypeGift
}

func buildBalanceLedgerStatementContentSummary(ledger UserBalanceLedger) string {
	parts := make([]string, 0, 4)
	parts = append(parts, financeSourceLabel(ledger.SourceType))
	if sourceName := strings.TrimSpace(ledger.SourceName); sourceName != "" {
		parts = append(parts, sourceName)
	}
	if remark := strings.TrimSpace(ledger.Remark); remark != "" {
		parts = append(parts, remark)
	}
	if len(parts) == 0 {
		return ledger.EntryType
	}
	return strings.Join(parts, " | ")
}

func buildFundingStatementContentSummary(funding UserQuotaFunding) string {
	parts := make([]string, 0, 3)
	parts = append(parts, financeSourceLabel(funding.SourceType))
	if sourceName := strings.TrimSpace(funding.SourceName); sourceName != "" {
		parts = append(parts, sourceName)
	}
	if remark := strings.TrimSpace(funding.Remark); remark != "" {
		parts = append(parts, remark)
	}
	return strings.Join(parts, " | ")
}

func deriveStatementContentSummary(entryType string, operationType string, logRecord *Log, other map[string]interface{}) string {
	prefix := "消费"
	if entryType == CustomerMonthlyStatementEntryTypeRefund {
		prefix = "退款"
	}

	switch operationType {
	case "image_generation":
		return fmt.Sprintf("%s - 图像生成", prefix)
	case "image":
		return fmt.Sprintf("%s - 图像相关请求", prefix)
	case "audio":
		return fmt.Sprintf("%s - 音频相关请求", prefix)
	case "file_search":
		return fmt.Sprintf("%s - 文件检索", prefix)
	case "web_search":
		return fmt.Sprintf("%s - 联网搜索", prefix)
	case "task":
		return fmt.Sprintf("%s - 异步任务", prefix)
	}

	if logRecord.ModelName != "" {
		return fmt.Sprintf("%s - %s", prefix, logRecord.ModelName)
	}
	if requestPath := getStringFromMap(other, "request_path"); requestPath != "" {
		return fmt.Sprintf("%s - %s", prefix, requestPath)
	}
	return prefix
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	value, ok := m[key]
	if !ok {
		return false
	}
	boolValue, ok := value.(bool)
	return ok && boolValue
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(common.Interface2String(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func formatStatementCSVDisplayAmount(amount float64, displayType string, symbol string) string {
	formatted := fmt.Sprintf("%.6f", amount)
	formatted = strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
	if formatted == "" || formatted == "-0" {
		formatted = "0"
	}
	if displayType == operation_setting.QuotaDisplayTypeTokens || symbol == "" {
		return formatted
	}
	return symbol + formatted
}

func sanitizeStatementFilenamePart(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	safeValue := strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		}
		if r < 32 {
			return -1
		}
		return r
	}, value)
	safeValue = strings.TrimSpace(safeValue)
	if safeValue == "" {
		return fallback
	}
	return safeValue
}
