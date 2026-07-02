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
	for _, userID := range userIDs {
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

func ListFinanceCustomerBillSummaryV2(billMonth string, userKeyword string, pageInfo *common.PageInfo) (*common.PageInfo, error) {
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
		pageInfo.SetTotal(0)
		pageInfo.SetItems([]FinanceCustomerBillSummaryItem{})
		return pageInfo, nil
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
		var statementItems []CustomerMonthlyStatementItem
		if err := DB.Where("statement_id IN ?", statementIDs).Find(&statementItems).Error; err != nil {
			return nil, err
		}
		usageMap = financeBuildUserUsageFromStatementItems(statementItems)
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
	pageInfo.SetTotal(len(items))
	pageInfo.SetItems(financePageItems(items, pageInfo))
	return pageInfo, nil
}

func GetFinanceCustomerBillDetailsV2(billMonth string, userID int) ([]FinanceCustomerBillDetailItem, error) {
	statement, err := GenerateCustomerMonthlyStatement(userID, billMonth, false)
	if err != nil {
		return nil, err
	}
	statementItems, err := GetAllCustomerMonthlyStatementItems(statement.Id, 0)
	if err != nil {
		return nil, err
	}

	items := make([]FinanceCustomerBillDetailItem, 0, len(statementItems))
	for _, item := range statementItems {
		tokenDisplay := "-"
		if item.TokenNameSnapshot != "" || item.TokenMasked != "" {
			tokenDisplay = fmt.Sprintf("%s / %s", firstNonEmpty(item.TokenNameSnapshot, "-"), firstNonEmpty(item.TokenMasked, "-"))
		}
		items = append(items, FinanceCustomerBillDetailItem{
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
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt > items[j].OccurredAt })
	return items, nil
}

func ExportFinanceCustomerBillCSVV2(billMonth string, userID int) ([]byte, string, error) {
	items, err := GetFinanceCustomerBillDetailsV2(billMonth, userID)
	if err != nil {
		return nil, "", err
	}
	buffer := &bytes.Buffer{}
	buffer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(buffer)
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
