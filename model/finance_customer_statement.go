package model

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

type FinanceCustomerBillGenerateResult struct {
	BillMonth          string `json:"bill_month"`
	GeneratedUserCount int    `json:"generated_user_count"`
}

func financeCollectCustomerStatementUserIDs(normalizedMonth string, start int64, end int64, matchedUserIDs []int, restrictToMatchedUsers bool) ([]int, error) {
	seen := make(map[int]struct{})
	userIDs := make([]int, 0)
	appendUserID := func(userID int) {
		if userID <= 0 {
			return
		}
		if _, ok := seen[userID]; ok {
			return
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}

	for _, userID := range matchedUserIDs {
		appendUserID(userID)
	}

	if restrictToMatchedUsers {
		return userIDs, nil
	}

	var statements []CustomerMonthlyStatement
	if err := DB.Select("user_id").Where("bill_month = ?", normalizedMonth).Find(&statements).Error; err != nil {
		return nil, err
	}
	for _, statement := range statements {
		appendUserID(statement.UserId)
	}

	var channelLedgers []ChannelCostLedger
	if err := DB.Select("user_id").Where("bill_month = ? AND occurred_at >= ? AND occurred_at <= ?", normalizedMonth, start, end).Find(&channelLedgers).Error; err != nil {
		return nil, err
	}
	for _, ledger := range channelLedgers {
		appendUserID(ledger.UserId)
	}

	var balanceLedgers []UserBalanceLedger
	if err := DB.Select("user_id").Where("bill_month = ? AND occurred_at >= ? AND occurred_at <= ?", normalizedMonth, start, end).Find(&balanceLedgers).Error; err != nil {
		return nil, err
	}
	for _, ledger := range balanceLedgers {
		appendUserID(ledger.UserId)
	}

	if len(userIDs) == 0 {
		legacyLogs, err := loadLegacyBillingLogs(start, end)
		if err != nil {
			return nil, err
		}
		for _, logRecord := range legacyLogs {
			if logRecord != nil {
				appendUserID(logRecord.UserId)
			}
		}
		var fundings []UserQuotaFunding
		if err := DB.Select("user_id").Where("created_at >= ? AND created_at <= ?", start, end).Find(&fundings).Error; err != nil {
			return nil, err
		}
		for _, funding := range fundings {
			appendUserID(funding.UserId)
		}
	}
	return userIDs, nil
}

func GenerateFinanceCustomerBills(billMonth string, userKeyword string) (*FinanceCustomerBillGenerateResult, error) {
	normalizedMonth, start, end, err := ParseCustomerStatementBillMonth(billMonth)
	if err != nil {
		return nil, err
	}
	restrictToMatchedUsers := strings.TrimSpace(userKeyword) != ""
	matchedUserIDs, err := financeMatchUserIDs(userKeyword)
	if err != nil {
		return nil, err
	}
	userIDs, err := financeCollectCustomerStatementUserIDs(normalizedMonth, start, end, matchedUserIDs, restrictToMatchedUsers)
	if err != nil {
		return nil, err
	}
	for _, userID := range userIDs {
		if _, err := GenerateCustomerMonthlyStatement(userID, normalizedMonth, true); err != nil {
			return nil, err
		}
	}
	return &FinanceCustomerBillGenerateResult{
		BillMonth:          normalizedMonth,
		GeneratedUserCount: len(userIDs),
	}, nil
}

func financeEnsureCustomerStatements(normalizedMonth string, start int64, end int64, matchedUserIDs []int, restrictToMatchedUsers bool) ([]int, error) {
	userIDs, err := financeCollectCustomerStatementUserIDs(normalizedMonth, start, end, matchedUserIDs, restrictToMatchedUsers)
	if err != nil {
		return nil, err
	}
	if len(userIDs) == 0 {
		return userIDs, nil
	}
	var existingStatements []CustomerMonthlyStatement
	if err := DB.Select("user_id").Where("bill_month = ? AND user_id IN ?", normalizedMonth, userIDs).Find(&existingStatements).Error; err != nil {
		return nil, err
	}
	existingUserIDs := make(map[int]struct{}, len(existingStatements))
	for _, statement := range existingStatements {
		existingUserIDs[statement.UserId] = struct{}{}
	}
	for _, userID := range userIDs {
		if _, ok := existingUserIDs[userID]; ok {
			continue
		}
		if _, err := GenerateCustomerMonthlyStatement(userID, normalizedMonth, false); err != nil {
			return nil, err
		}
	}
	return userIDs, nil
}

func financeBuildUserUsageFromStatementItems(items []CustomerMonthlyStatementItem) map[int]*financeUserUsageAggregate {
	result := make(map[int]*financeUserUsageAggregate)
	for _, item := range items {
		if item.EntryType != CustomerMonthlyStatementEntryTypeConsume {
			continue
		}
		row := result[item.UserId]
		if row == nil {
			row = &financeUserUsageAggregate{UserID: item.UserId}
			result[item.UserId] = row
		}
		row.ConsumeCount++
		row.ConsumePlatform += absInt(item.QuotaRaw)
		row.PaidConsumePlatform += absInt(item.PaidQuotaRaw)
		row.GiftConsumePlatform += absInt(item.GiftQuotaRaw)
		row.ConsumeUSD = roundAccountingAmount(row.ConsumeUSD + mathAbs(item.USDAmount))
		row.PaidConsumeUSD = roundAccountingAmount(row.PaidConsumeUSD + mathAbs(quotaToUSD(item.PaidQuotaRaw, item.QuotaPerUnitSnapshot)))
		row.GiftConsumeUSD = roundAccountingAmount(row.GiftConsumeUSD + mathAbs(quotaToUSD(item.GiftQuotaRaw, item.QuotaPerUnitSnapshot)))
		if item.OccurredAt > row.LastOccurredAt {
			row.LastOccurredAt = item.OccurredAt
		}
	}
	return result
}

func loadFinanceUserUsageFromStatementItems(statementIDs []int) (map[int]*financeUserUsageAggregate, error) {
	if len(statementIDs) == 0 {
		return map[int]*financeUserUsageAggregate{}, nil
	}
	var rows []financeUserUsageAggregate
	err := DB.Model(&CustomerMonthlyStatementItem{}).
		Select(`
			user_id,
			SUM(CASE WHEN entry_type = ? THEN 1 ELSE 0 END) AS consume_count,
			SUM(CASE WHEN entry_type = ? THEN ABS(quota_raw) ELSE 0 END) AS consume_platform,
			SUM(CASE WHEN entry_type = ? THEN ABS(paid_quota_raw) ELSE 0 END) AS paid_consume_platform,
			SUM(CASE WHEN entry_type = ? THEN ABS(gift_quota_raw) ELSE 0 END) AS gift_consume_platform,
			SUM(CASE WHEN entry_type = ? THEN ABS(usd_amount) ELSE 0 END) AS consume_usd,
			SUM(CASE WHEN entry_type = ? AND quota_per_unit_snapshot > 0 THEN ROUND(ABS(paid_quota_raw) * 1.0 / quota_per_unit_snapshot, 6) ELSE 0 END) AS paid_consume_usd,
			SUM(CASE WHEN entry_type = ? AND quota_per_unit_snapshot > 0 THEN ROUND(ABS(gift_quota_raw) * 1.0 / quota_per_unit_snapshot, 6) ELSE 0 END) AS gift_consume_usd,
			MAX(CASE WHEN entry_type = ? THEN occurred_at ELSE 0 END) AS last_occurred_at`,
			CustomerMonthlyStatementEntryTypeConsume,
			CustomerMonthlyStatementEntryTypeConsume,
			CustomerMonthlyStatementEntryTypeConsume,
			CustomerMonthlyStatementEntryTypeConsume,
			CustomerMonthlyStatementEntryTypeConsume,
			CustomerMonthlyStatementEntryTypeConsume,
			CustomerMonthlyStatementEntryTypeConsume,
			CustomerMonthlyStatementEntryTypeConsume,
		).
		Where("statement_id IN ?", statementIDs).
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int]*financeUserUsageAggregate, len(rows))
	for i := range rows {
		row := rows[i]
		if row.ConsumeCount == 0 {
			continue
		}
		row.ConsumeUSD = roundAccountingAmount(row.ConsumeUSD)
		row.PaidConsumeUSD = roundAccountingAmount(row.PaidConsumeUSD)
		row.GiftConsumeUSD = roundAccountingAmount(row.GiftConsumeUSD)
		result[row.UserID] = &row
	}
	return result, nil
}

func financeStatementEntryTypeLabel(entryType string) string {
	switch strings.TrimSpace(entryType) {
	case CustomerMonthlyStatementEntryTypeConsume:
		return "消费"
	case CustomerMonthlyStatementEntryTypeRefund:
		return "退款"
	case CustomerMonthlyStatementEntryTypeTopup:
		return "充值"
	case CustomerMonthlyStatementEntryTypeGift:
		return "赠送"
	case CustomerMonthlyStatementEntryTypeAdjustment:
		return "调整"
	default:
		return entryType
	}
}

func financeCustomerBillSummaryTotals(items []FinanceCustomerBillSummaryItem) FinanceCustomerBillSummaryTotals {
	totals := FinanceCustomerBillSummaryTotals{CustomerCount: len(items)}
	for _, item := range items {
		totals.CurrentBalanceUSD += item.CurrentBalanceUSD
		totals.CurrentBalanceCOS += item.CurrentBalanceCOS
		totals.TotalConsumeUSD += item.TotalConsumeUSD
		totals.TotalConsumeCOS += item.TotalConsumeCOS
		totals.PaidConsumeUSD += item.PaidConsumeUSD
		totals.PaidConsumeCOS += item.PaidConsumeCOS
		totals.GiftConsumeUSD += item.GiftConsumeUSD
		totals.GiftConsumeCOS += item.GiftConsumeCOS
	}
	totals.CurrentBalanceUSD = roundAccountingAmount(totals.CurrentBalanceUSD)
	totals.CurrentBalanceCOS = roundAccountingAmount(totals.CurrentBalanceCOS)
	totals.TotalConsumeUSD = roundAccountingAmount(totals.TotalConsumeUSD)
	totals.TotalConsumeCOS = roundAccountingAmount(totals.TotalConsumeCOS)
	totals.PaidConsumeUSD = roundAccountingAmount(totals.PaidConsumeUSD)
	totals.PaidConsumeCOS = roundAccountingAmount(totals.PaidConsumeCOS)
	totals.GiftConsumeUSD = roundAccountingAmount(totals.GiftConsumeUSD)
	totals.GiftConsumeCOS = roundAccountingAmount(totals.GiftConsumeCOS)
	return totals
}

func financeCustomerBillSummaryPage(pageInfo *common.PageInfo, items []FinanceCustomerBillSummaryItem) *FinanceCustomerBillSummaryPage {
	return &FinanceCustomerBillSummaryPage{
		Page:     pageInfo.GetPage(),
		PageSize: pageInfo.GetPageSize(),
		Total:    len(items),
		Items:    financePageItems(items, pageInfo),
		Summary:  financeCustomerBillSummaryTotals(items),
	}
}

func ListFinanceCustomerBillSummaryV2(billMonth string, userKeyword string, pageInfo *common.PageInfo) (*FinanceCustomerBillSummaryPage, error) {
	normalizedMonth, start, end, err := ParseCustomerStatementBillMonth(billMonth)
	if err != nil {
		return nil, err
	}
	restrictToMatchedUsers := strings.TrimSpace(userKeyword) != ""
	matchedUserIDs, err := financeMatchUserIDs(userKeyword)
	if err != nil {
		return nil, err
	}
	userIDs, err := financeEnsureCustomerStatements(normalizedMonth, start, end, matchedUserIDs, restrictToMatchedUsers)
	if err != nil {
		return nil, err
	}
	if restrictToMatchedUsers && len(userIDs) == 0 {
		return financeCustomerBillSummaryPage(pageInfo, []FinanceCustomerBillSummaryItem{}), nil
	}

	var statements []CustomerMonthlyStatement
	query := DB.Where("bill_month = ?", normalizedMonth)
	if len(userIDs) > 0 {
		query = query.Where("user_id IN ?", userIDs)
	}
	if err := query.Find(&statements).Error; err != nil {
		return nil, err
	}

	statementByUser := make(map[int]CustomerMonthlyStatement, len(statements))
	statementIDs := make([]int, 0, len(statements))
	for _, statement := range statements {
		statementByUser[statement.UserId] = statement
		statementIDs = append(statementIDs, statement.Id)
	}

	usageMap := map[int]*financeUserUsageAggregate{}
	if len(statementIDs) > 0 {
		usageMap, err = loadFinanceUserUsageFromStatementItems(statementIDs)
		if err != nil {
			return nil, err
		}
	}

	userMap, err := financeUserMap(userIDs)
	if err != nil {
		return nil, err
	}
	items := make([]FinanceCustomerBillSummaryItem, 0, len(userIDs))
	for _, userID := range userIDs {
		user := userMap[userID]
		item := FinanceCustomerBillSummaryItem{
			UserID:                 userID,
			Username:               firstNonEmpty(user.Username, statementByUser[userID].UsernameSnapshot, fmt.Sprintf("user-%d", userID)),
			CurrentBalancePlatform: user.Quota,
			CurrentBalanceUSD:      roundAccountingAmount(quotaToUSDWithSnapshot(user.Quota, common.QuotaPerUnit)),
			CurrentBalanceCOS:      financeCOSAmountFromUSD(quotaToUSDWithSnapshot(user.Quota, common.QuotaPerUnit)),
		}
		if statement, ok := statementByUser[userID]; ok {
			item.TotalConsumePlatform = 0
			item.TotalConsumeUSD = statement.TotalConsumeUSD
			item.TotalConsumeCOS = financeCOSAmountFromUSD(statement.TotalConsumeUSD)
		}
		if usage := usageMap[userID]; usage != nil {
			item.TotalConsumePlatform = usage.ConsumePlatform
			item.PaidConsumePlatform = usage.PaidConsumePlatform
			item.PaidConsumeUSD = usage.PaidConsumeUSD
			item.PaidConsumeCOS = financeCOSAmountFromUSD(usage.PaidConsumeUSD)
			item.GiftConsumePlatform = usage.GiftConsumePlatform
			item.GiftConsumeUSD = usage.GiftConsumeUSD
			item.GiftConsumeCOS = financeCOSAmountFromUSD(usage.GiftConsumeUSD)
			if item.TotalConsumeUSD == 0 {
				item.TotalConsumeUSD = usage.ConsumeUSD
				item.TotalConsumeCOS = financeCOSAmountFromUSD(usage.ConsumeUSD)
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].TotalConsumeUSD > items[j].TotalConsumeUSD })
	return financeCustomerBillSummaryPage(pageInfo, items), nil
}

func GetFinanceCustomerBillDetailsV2(billMonth string, userID int) ([]FinanceCustomerBillDetailItem, error) {
	statement, err := GenerateCustomerMonthlyStatement(userID, billMonth, false)
	if err != nil {
		return nil, err
	}
	return buildFinanceCustomerBillDetails(statement)
}

func buildFinanceCustomerBillDetails(statement *CustomerMonthlyStatement) ([]FinanceCustomerBillDetailItem, error) {
	statementItems, err := GetAllCustomerMonthlyStatementItems(statement.Id, 0)
	if err != nil {
		return nil, err
	}

	items := make([]FinanceCustomerBillDetailItem, 0, len(statementItems))
	for _, item := range statementItems {
		if item == nil {
			continue
		}
		items = append(items, financeCustomerBillDetailFromStatementItem(*item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt > items[j].OccurredAt })
	return items, nil
}

func financeCustomerBillDetailFromStatementItem(item CustomerMonthlyStatementItem) FinanceCustomerBillDetailItem {
	tokenDisplay := "-"
	if item.TokenNameSnapshot != "" || item.TokenMasked != "" {
		tokenDisplay = fmt.Sprintf("%s / %s", firstNonEmpty(item.TokenNameSnapshot, "-"), firstNonEmpty(item.TokenMasked, "-"))
	}
	return FinanceCustomerBillDetailItem{
		OccurredAt:     item.OccurredAt,
		TokenDisplay:   tokenDisplay,
		EntryType:      item.EntryType,
		ModelName:      item.ModelName,
		AmountCOS:      financeCOSAmountFromUSD(item.USDAmount),
		AmountPlatform: item.QuotaRaw,
		AmountUSD:      item.USDAmount,
		ChannelName:    item.ChannelNameSnapshot,
		RequestID:      item.RequestId,
		Remark:         item.ContentSummary,
	}
}

func ListFinanceCustomerBillDetailsV2(billMonth string, userID int, pageInfo *common.PageInfo) (*common.PageInfo, error) {
	statement, err := GenerateCustomerMonthlyStatement(userID, billMonth, false)
	if err != nil {
		return nil, err
	}
	var total int64
	if err := DB.Model(&CustomerMonthlyStatementItem{}).Where("statement_id = ?", statement.Id).Count(&total).Error; err != nil {
		return nil, err
	}
	var statementItems []CustomerMonthlyStatementItem
	if err := DB.Where("statement_id = ?", statement.Id).
		Order("occurred_at desc, id desc").
		Offset(pageInfo.GetStartIdx()).
		Limit(pageInfo.GetPageSize()).
		Find(&statementItems).Error; err != nil {
		return nil, err
	}
	items := make([]FinanceCustomerBillDetailItem, 0, len(statementItems))
	for _, item := range statementItems {
		items = append(items, financeCustomerBillDetailFromStatementItem(item))
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	return pageInfo, nil
}

func ExportFinanceCustomerBillCSVV2(billMonth string, userID int) ([]byte, string, error) {
	statement, err := GenerateCustomerMonthlyStatement(userID, billMonth, false)
	if err != nil {
		return nil, "", err
	}
	items, err := buildFinanceCustomerBillDetails(statement)
	if err != nil {
		return nil, "", err
	}
	buffer := &bytes.Buffer{}
	buffer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(buffer)
	summaryHeader := []string{
		"客户名字",
		"账单时间周期",
		"本期消费",
		"赠送",
		"调整",
		"充值",
		"其他项目汇总",
	}
	if err := writer.Write(summaryHeader); err != nil {
		return nil, "", err
	}
	summaryRecord := []string{
		firstNonEmpty(statement.UsernameSnapshot, fmt.Sprintf("user-%d", userID)),
		fmt.Sprintf(
			"%s 至 %s",
			time.Unix(statement.PeriodStart, 0).In(time.Local).Format("2006-01-02"),
			time.Unix(statement.PeriodEnd, 0).In(time.Local).Format("2006-01-02"),
		),
		formatFinanceBillSummaryAmount(statement.TotalConsumeUSD),
		formatFinanceBillSummaryAmount(statement.TotalGiftUSD),
		formatFinanceBillSummaryAmount(statement.TotalAdjustmentUSD),
		formatFinanceBillSummaryAmount(statement.TotalTopupUSD),
		formatFinanceBillSummaryAmount(statement.TotalRefundUSD),
	}
	if err := writer.Write(summaryRecord); err != nil {
		return nil, "", err
	}
	if err := writer.Write([]string{}); err != nil {
		return nil, "", err
	}
	header := []string{"时间", "令牌", "消费类型", "模型", "COS币变动", "等价USD", "渠道", "请求ID", "说明"}
	if err := writer.Write(header); err != nil {
		return nil, "", err
	}
	for _, item := range items {
		record := []string{
			time.Unix(item.OccurredAt, 0).In(time.Local).Format("2006-01-02 15:04:05"),
			item.TokenDisplay,
			financeStatementEntryTypeLabel(item.EntryType),
			item.ModelName,
			fmt.Sprintf("%.6f", item.AmountCOS),
			fmt.Sprintf("%.6f", item.AmountUSD),
			item.ChannelName,
			item.RequestID,
			item.Remark,
		}
		if err := writer.Write(record); err != nil {
			return nil, "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}
	return buffer.Bytes(), fmt.Sprintf("customer-bill-%d-%s.csv", userID, billMonth), nil
}

func formatFinanceBillSummaryAmount(usdAmount float64) string {
	return fmt.Sprintf("%.6f COS / $%.6f", financeCOSAmountFromUSD(usdAmount), usdAmount)
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
