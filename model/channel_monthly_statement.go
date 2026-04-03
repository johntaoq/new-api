package model

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type ChannelMonthlyStatementBundle struct {
	Statement *ChannelMonthlyStatement `json:"statement"`
	Items     *common.PageInfo         `json:"items"`
}

type channelMonthlyStatementAggregate struct {
	ProviderSnapshot     string
	ChannelId            int
	ChannelNameSnapshot  string
	OriginModelName      string
	BillingSource        string
	ConsumeCount         int
	RefundCount          int
	PromptTokens         int
	CompletionTokens     int
	CacheTokens          int
	TotalTokens          int
	PaidQuotaUsed        int
	GiftQuotaUsed        int
	RecognizedRevenueUSD float64
	EstimatedCostUSD     float64
	GiftCostUSD          float64
	GrossProfitUSD       float64
}

func GenerateChannelMonthlyStatement(billMonth string, force bool) (*ChannelMonthlyStatement, error) {
	normalizedBillMonth, periodStart, periodEnd, err := ParseCustomerStatementBillMonth(billMonth)
	if err != nil {
		return nil, err
	}

	if !force {
		var existing ChannelMonthlyStatement
		if err := DB.Where("bill_month = ?", normalizedBillMonth).First(&existing).Error; err == nil {
			if existing.TotalLedgerCount > 0 ||
				existing.TotalConsumeCount > 0 ||
				existing.TotalRefundCount > 0 ||
				existing.TotalEstimatedCostUSD > 0 ||
				existing.TotalPaidQuotaUsed > 0 ||
				existing.TotalGiftQuotaUsed > 0 {
				return &existing, nil
			}
		} else if err != nil && !errorsIsRecordNotFound(err) {
			return nil, err
		}
	}

	var ledgers []ChannelCostLedger
	if err := DB.Where("bill_month = ? AND occurred_at >= ? AND occurred_at <= ?", normalizedBillMonth, periodStart, periodEnd).
		Order("occurred_at asc, id asc").
		Find(&ledgers).Error; err != nil {
		return nil, err
	}

	legacyFallbackLogs := make([]*Log, 0)
	useLegacyLogsFallback := len(ledgers) == 0
	if useLegacyLogsFallback {
		legacyFallbackLogs, err = loadLegacyBillingLogs(periodStart, periodEnd)
		if err != nil {
			return nil, err
		}
	}

	var fundings []UserQuotaFunding
	if err := DB.Where("created_at >= ? AND created_at <= ?", periodStart, periodEnd).Find(&fundings).Error; err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	statementNo := fmt.Sprintf("CHSTM-%s", normalizedBillMonth)
	statement := &ChannelMonthlyStatement{}

	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("bill_month = ?", normalizedBillMonth).First(statement).Error
		if err != nil && !errorsIsRecordNotFound(err) {
			return err
		}

		exists := err == nil
		if !exists {
			statement.StatementNo = statementNo
			statement.BillMonth = normalizedBillMonth
			statement.CreatedAt = now
		}

		statement.PeriodStart = periodStart
		statement.PeriodEnd = periodEnd
		statement.Status = ChannelMonthlyStatementStatusFinalized
		statement.GeneratedAt = now
		statement.UpdatedAt = now

		if exists {
			if err := tx.Model(statement).Updates(map[string]interface{}{
				"period_start": periodStart,
				"period_end":   periodEnd,
				"status":       statement.Status,
				"generated_at": now,
				"updated_at":   now,
			}).Error; err != nil {
				return err
			}
			if err := tx.Where("statement_id = ?", statement.Id).Delete(&ChannelMonthlyStatementItem{}).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Create(statement).Error; err != nil {
				return err
			}
		}

		var items []ChannelMonthlyStatementItem
		var summary channelMonthlyStatementSummary
		if useLegacyLogsFallback && len(legacyFallbackLogs) > 0 {
			items, summary, err = buildChannelMonthlyStatementItemsFromLegacyLogs(statement.Id, normalizedBillMonth, legacyFallbackLogs, now)
			if err != nil {
				return err
			}
		} else {
			items, summary = buildChannelMonthlyStatementItems(statement.Id, normalizedBillMonth, ledgers, now)
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}

		if useLegacyLogsFallback && len(legacyFallbackLogs) > 0 {
			statement.TotalLedgerCount = len(legacyFallbackLogs)
		} else {
			statement.TotalLedgerCount = len(ledgers)
		}
		statement.TotalConsumeCount = summary.TotalConsumeCount
		statement.TotalRefundCount = summary.TotalRefundCount
		statement.TotalPaidGrantedQuota, statement.TotalGiftGrantedQuota, statement.TotalGiftGrantedEquivalentUSD, statement.TotalSalesIncomeUSD = summarizeFundingGrants(fundings)
		statement.TotalPaidQuotaUsed = summary.TotalPaidQuotaUsed
		statement.TotalGiftQuotaUsed = summary.TotalGiftQuotaUsed
		statement.TotalPaidConsumptionRevenueUSD = summary.TotalPaidConsumptionRevenueUSD
		statement.TotalEstimatedCostUSD = summary.TotalEstimatedCostUSD
		statement.TotalGiftCostUSD = summary.TotalGiftCostUSD
		statement.TotalGrossProfitUSD = summary.TotalGrossProfitUSD

		return tx.Model(statement).Updates(map[string]interface{}{
			"total_ledger_count":                 statement.TotalLedgerCount,
			"total_consume_count":                statement.TotalConsumeCount,
			"total_refund_count":                 statement.TotalRefundCount,
			"total_paid_granted_quota":           statement.TotalPaidGrantedQuota,
			"total_gift_granted_quota":           statement.TotalGiftGrantedQuota,
			"total_gift_granted_equivalent_usd":  statement.TotalGiftGrantedEquivalentUSD,
			"total_sales_income_usd":             statement.TotalSalesIncomeUSD,
			"total_paid_quota_used":              statement.TotalPaidQuotaUsed,
			"total_gift_quota_used":              statement.TotalGiftQuotaUsed,
			"total_paid_consumption_revenue_usd": statement.TotalPaidConsumptionRevenueUSD,
			"total_estimated_cost_usd":           statement.TotalEstimatedCostUSD,
			"total_gift_cost_usd":                statement.TotalGiftCostUSD,
			"total_gross_profit_usd":             statement.TotalGrossProfitUSD,
			"generated_at":                       statement.GeneratedAt,
			"updated_at":                         statement.UpdatedAt,
		}).Error
	})
	if err != nil {
		return nil, err
	}

	return statement, nil
}

func buildChannelMonthlyStatementItemsFromLegacyLogs(statementId int, billMonth string, logs []*Log, now int64) ([]ChannelMonthlyStatementItem, channelMonthlyStatementSummary, error) {
	channelSnapshots, err := loadLegacyBillingChannelSnapshots(logs)
	if err != nil {
		return nil, channelMonthlyStatementSummary{}, err
	}

	itemsMap := make(map[string]*channelMonthlyStatementAggregate)
	summary := channelMonthlyStatementSummary{}

	for _, logRecord := range logs {
		if logRecord == nil {
			continue
		}

		entryType := convertLogTypeToStatementEntryType(logRecord.Type)
		if entryType == "" {
			continue
		}

		sign := 1
		if entryType == CustomerMonthlyStatementEntryTypeRefund {
			sign = -1
			summary.TotalRefundCount++
		} else {
			summary.TotalConsumeCount++
		}

		other := parseStatementOther(logRecord.Other)
		paidQuotaUsed, giftQuotaUsed := legacyBillingQuotaSplit(logRecord, other)
		recognizedRevenueUSD := legacyBillingRecognizedRevenueUSD(logRecord, other)
		estimatedCostUSD := legacyBillingEstimatedCostUSD(logRecord, other)
		giftCostUSD := 0.0
		if logRecord.Quota > 0 && giftQuotaUsed > 0 && estimatedCostUSD > 0 {
			giftCostUSD = estimatedCostUSD * float64(giftQuotaUsed) / float64(logRecord.Quota)
		}

		summary.TotalPaidQuotaUsed += sign * paidQuotaUsed
		summary.TotalGiftQuotaUsed += sign * giftQuotaUsed
		summary.TotalPaidConsumptionRevenueUSD = roundAccountingAmount(summary.TotalPaidConsumptionRevenueUSD + float64(sign)*recognizedRevenueUSD)
		summary.TotalEstimatedCostUSD = roundAccountingAmount(summary.TotalEstimatedCostUSD + float64(sign)*estimatedCostUSD)
		summary.TotalGiftCostUSD = roundAccountingAmount(summary.TotalGiftCostUSD + float64(sign)*giftCostUSD)
		summary.TotalGrossProfitUSD = roundAccountingAmount(summary.TotalGrossProfitUSD + float64(sign)*(recognizedRevenueUSD-estimatedCostUSD))

		billingSource := legacyBillingSource(other)
		key := fmt.Sprintf("%s|%d|%s|%s", legacyBillingProviderSnapshot(logRecord.ChannelId, channelSnapshots), logRecord.ChannelId, logRecord.ModelName, billingSource)
		aggregate, ok := itemsMap[key]
		if !ok {
			channelSnapshot := channelSnapshots[logRecord.ChannelId]
			aggregate = &channelMonthlyStatementAggregate{
				ProviderSnapshot:    legacyBillingProviderSnapshot(logRecord.ChannelId, channelSnapshots),
				ChannelId:           logRecord.ChannelId,
				ChannelNameSnapshot: firstNonEmpty(channelSnapshot.Name, logRecord.ChannelName),
				OriginModelName:     logRecord.ModelName,
				BillingSource:       billingSource,
			}
			itemsMap[key] = aggregate
		}
		if entryType == CustomerMonthlyStatementEntryTypeRefund {
			aggregate.RefundCount++
		} else {
			aggregate.ConsumeCount++
		}
		aggregate.PromptTokens += sign * logRecord.PromptTokens
		aggregate.CompletionTokens += sign * logRecord.CompletionTokens
		aggregate.CacheTokens += 0
		aggregate.TotalTokens += sign * (logRecord.PromptTokens + logRecord.CompletionTokens)
		aggregate.PaidQuotaUsed += sign * paidQuotaUsed
		aggregate.GiftQuotaUsed += sign * giftQuotaUsed
		aggregate.RecognizedRevenueUSD = roundAccountingAmount(aggregate.RecognizedRevenueUSD + float64(sign)*recognizedRevenueUSD)
		aggregate.EstimatedCostUSD = roundAccountingAmount(aggregate.EstimatedCostUSD + float64(sign)*estimatedCostUSD)
		aggregate.GiftCostUSD = roundAccountingAmount(aggregate.GiftCostUSD + float64(sign)*giftCostUSD)
		aggregate.GrossProfitUSD = roundAccountingAmount(aggregate.GrossProfitUSD + float64(sign)*(recognizedRevenueUSD-estimatedCostUSD))
	}

	keys := make([]string, 0, len(itemsMap))
	for key := range itemsMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]ChannelMonthlyStatementItem, 0, len(keys))
	for _, key := range keys {
		aggregate := itemsMap[key]
		items = append(items, ChannelMonthlyStatementItem{
			StatementId:          statementId,
			BillMonth:            billMonth,
			ProviderSnapshot:     aggregate.ProviderSnapshot,
			ChannelId:            aggregate.ChannelId,
			ChannelNameSnapshot:  aggregate.ChannelNameSnapshot,
			OriginModelName:      aggregate.OriginModelName,
			BillingSource:        aggregate.BillingSource,
			ConsumeCount:         aggregate.ConsumeCount,
			RefundCount:          aggregate.RefundCount,
			PromptTokens:         aggregate.PromptTokens,
			CompletionTokens:     aggregate.CompletionTokens,
			CacheTokens:          aggregate.CacheTokens,
			TotalTokens:          aggregate.TotalTokens,
			PaidQuotaUsed:        aggregate.PaidQuotaUsed,
			GiftQuotaUsed:        aggregate.GiftQuotaUsed,
			RecognizedRevenueUSD: aggregate.RecognizedRevenueUSD,
			EstimatedCostUSD:     aggregate.EstimatedCostUSD,
			GiftCostUSD:          aggregate.GiftCostUSD,
			GrossProfitUSD:       aggregate.GrossProfitUSD,
			CreatedAt:            now,
			UpdatedAt:            now,
		})
	}
	return items, summary, nil
}

func GetChannelMonthlyStatementByMonth(billMonth string) (*ChannelMonthlyStatement, error) {
	normalizedBillMonth, _, _, err := ParseCustomerStatementBillMonth(billMonth)
	if err != nil {
		return nil, err
	}
	var statement ChannelMonthlyStatement
	if err := DB.Where("bill_month = ?", normalizedBillMonth).First(&statement).Error; err != nil {
		return nil, err
	}
	return &statement, nil
}

func ListChannelMonthlyStatements(billMonth string, pageInfo *common.PageInfo) ([]*ChannelMonthlyStatement, int64, error) {
	var statements []*ChannelMonthlyStatement
	var total int64

	tx := DB.Model(&ChannelMonthlyStatement{})
	if billMonth != "" {
		tx = tx.Where("bill_month = ?", billMonth)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Order("bill_month desc, id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&statements).Error; err != nil {
		return nil, 0, err
	}
	return statements, total, nil
}

func GetChannelMonthlyStatementItems(statementId int, pageInfo *common.PageInfo) ([]*ChannelMonthlyStatementItem, int64, error) {
	var items []*ChannelMonthlyStatementItem
	var total int64
	tx := DB.Model(&ChannelMonthlyStatementItem{}).Where("statement_id = ?", statementId)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Order("estimated_cost_usd desc, id asc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func GetAllChannelMonthlyStatementItems(statementId int) ([]*ChannelMonthlyStatementItem, error) {
	var items []*ChannelMonthlyStatementItem
	if err := DB.Where("statement_id = ?", statementId).Order("estimated_cost_usd desc, id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func BuildChannelMonthlyStatementBundle(statement *ChannelMonthlyStatement, pageInfo *common.PageInfo) (*ChannelMonthlyStatementBundle, error) {
	items, total, err := GetChannelMonthlyStatementItems(statement.Id, pageInfo)
	if err != nil {
		return nil, err
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	return &ChannelMonthlyStatementBundle{
		Statement: statement,
		Items:     pageInfo,
	}, nil
}

func ExportChannelMonthlyStatementCSV(statement *ChannelMonthlyStatement) ([]byte, string, error) {
	items, err := GetAllChannelMonthlyStatementItems(statement.Id)
	if err != nil {
		return nil, "", err
	}

	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)
	header := []string{
		"bill_month", "provider", "channel_id", "channel_name", "model", "billing_source",
		"consume_count", "refund_count", "prompt_tokens", "completion_tokens", "cache_tokens", "total_tokens",
		"paid_quota_used", "gift_quota_used", "recognized_revenue_usd", "estimated_cost_usd",
		"gift_cost_usd", "gross_profit_usd",
	}
	if err := writer.Write(header); err != nil {
		return nil, "", err
	}
	for _, item := range items {
		record := []string{
			statement.BillMonth,
			item.ProviderSnapshot,
			fmt.Sprintf("%d", item.ChannelId),
			item.ChannelNameSnapshot,
			item.OriginModelName,
			item.BillingSource,
			fmt.Sprintf("%d", item.ConsumeCount),
			fmt.Sprintf("%d", item.RefundCount),
			fmt.Sprintf("%d", item.PromptTokens),
			fmt.Sprintf("%d", item.CompletionTokens),
			fmt.Sprintf("%d", item.CacheTokens),
			fmt.Sprintf("%d", item.TotalTokens),
			fmt.Sprintf("%d", item.PaidQuotaUsed),
			fmt.Sprintf("%d", item.GiftQuotaUsed),
			fmt.Sprintf("%.6f", item.RecognizedRevenueUSD),
			fmt.Sprintf("%.6f", item.EstimatedCostUSD),
			fmt.Sprintf("%.6f", item.GiftCostUSD),
			fmt.Sprintf("%.6f", item.GrossProfitUSD),
		}
		if err := writer.Write(record); err != nil {
			return nil, "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("channel-billing-%s.csv", statement.BillMonth)
	return buffer.Bytes(), filename, nil
}

type channelMonthlyStatementSummary struct {
	TotalConsumeCount              int
	TotalRefundCount               int
	TotalPaidQuotaUsed             int
	TotalGiftQuotaUsed             int
	TotalPaidConsumptionRevenueUSD float64
	TotalEstimatedCostUSD          float64
	TotalGiftCostUSD               float64
	TotalGrossProfitUSD            float64
}

func buildChannelMonthlyStatementItems(statementId int, billMonth string, ledgers []ChannelCostLedger, now int64) ([]ChannelMonthlyStatementItem, channelMonthlyStatementSummary) {
	itemsMap := make(map[string]*channelMonthlyStatementAggregate)
	summary := channelMonthlyStatementSummary{}

	for _, ledger := range ledgers {
		sign := 1
		if ledger.EntryType == ChannelCostEntryTypeRefund {
			sign = -1
			summary.TotalRefundCount++
		} else {
			summary.TotalConsumeCount++
		}
		giftCostUSD := 0.0
		if ledger.ActualQuota > 0 && ledger.GiftQuotaUsed > 0 && ledger.EstimatedCostUSD > 0 {
			giftCostUSD = ledger.EstimatedCostUSD * float64(ledger.GiftQuotaUsed) / float64(ledger.ActualQuota)
		}

		summary.TotalPaidQuotaUsed += sign * ledger.PaidQuotaUsed
		summary.TotalGiftQuotaUsed += sign * ledger.GiftQuotaUsed
		summary.TotalPaidConsumptionRevenueUSD = roundAccountingAmount(summary.TotalPaidConsumptionRevenueUSD + float64(sign)*ledger.RecognizedRevenueUSD)
		summary.TotalEstimatedCostUSD = roundAccountingAmount(summary.TotalEstimatedCostUSD + float64(sign)*ledger.EstimatedCostUSD)
		summary.TotalGiftCostUSD = roundAccountingAmount(summary.TotalGiftCostUSD + float64(sign)*giftCostUSD)
		summary.TotalGrossProfitUSD = roundAccountingAmount(summary.TotalGrossProfitUSD + float64(sign)*(ledger.RecognizedRevenueUSD-ledger.EstimatedCostUSD))

		key := fmt.Sprintf("%s|%d|%s|%s", ledger.ProviderSnapshot, ledger.ChannelId, ledger.OriginModelName, ledger.BillingSource)
		aggregate, ok := itemsMap[key]
		if !ok {
			aggregate = &channelMonthlyStatementAggregate{
				ProviderSnapshot:    ledger.ProviderSnapshot,
				ChannelId:           ledger.ChannelId,
				ChannelNameSnapshot: ledger.ChannelNameSnapshot,
				OriginModelName:     ledger.OriginModelName,
				BillingSource:       ledger.BillingSource,
			}
			itemsMap[key] = aggregate
		}
		if ledger.EntryType == ChannelCostEntryTypeRefund {
			aggregate.RefundCount++
		} else {
			aggregate.ConsumeCount++
		}
		aggregate.PromptTokens += sign * ledger.PromptTokens
		aggregate.CompletionTokens += sign * ledger.CompletionTokens
		aggregate.CacheTokens += sign * ledger.CacheTokens
		aggregate.TotalTokens += sign * ledger.TotalTokens
		aggregate.PaidQuotaUsed += sign * ledger.PaidQuotaUsed
		aggregate.GiftQuotaUsed += sign * ledger.GiftQuotaUsed
		aggregate.RecognizedRevenueUSD = roundAccountingAmount(aggregate.RecognizedRevenueUSD + float64(sign)*ledger.RecognizedRevenueUSD)
		aggregate.EstimatedCostUSD = roundAccountingAmount(aggregate.EstimatedCostUSD + float64(sign)*ledger.EstimatedCostUSD)
		aggregate.GiftCostUSD = roundAccountingAmount(aggregate.GiftCostUSD + float64(sign)*giftCostUSD)
		aggregate.GrossProfitUSD = roundAccountingAmount(aggregate.GrossProfitUSD + float64(sign)*(ledger.RecognizedRevenueUSD-ledger.EstimatedCostUSD))
	}

	keys := make([]string, 0, len(itemsMap))
	for key := range itemsMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	items := make([]ChannelMonthlyStatementItem, 0, len(keys))
	for _, key := range keys {
		aggregate := itemsMap[key]
		items = append(items, ChannelMonthlyStatementItem{
			StatementId:          statementId,
			BillMonth:            billMonth,
			ProviderSnapshot:     aggregate.ProviderSnapshot,
			ChannelId:            aggregate.ChannelId,
			ChannelNameSnapshot:  aggregate.ChannelNameSnapshot,
			OriginModelName:      aggregate.OriginModelName,
			BillingSource:        aggregate.BillingSource,
			ConsumeCount:         aggregate.ConsumeCount,
			RefundCount:          aggregate.RefundCount,
			PromptTokens:         aggregate.PromptTokens,
			CompletionTokens:     aggregate.CompletionTokens,
			CacheTokens:          aggregate.CacheTokens,
			TotalTokens:          aggregate.TotalTokens,
			PaidQuotaUsed:        aggregate.PaidQuotaUsed,
			GiftQuotaUsed:        aggregate.GiftQuotaUsed,
			RecognizedRevenueUSD: aggregate.RecognizedRevenueUSD,
			EstimatedCostUSD:     aggregate.EstimatedCostUSD,
			GiftCostUSD:          aggregate.GiftCostUSD,
			GrossProfitUSD:       aggregate.GrossProfitUSD,
			CreatedAt:            now,
			UpdatedAt:            now,
		})
	}
	return items, summary
}

func summarizeFundingGrants(fundings []UserQuotaFunding) (int, int, float64, float64) {
	paidGrantedQuota := 0
	giftGrantedQuota := 0
	giftGrantedUSD := 0.0
	salesIncomeUSD := 0.0
	for _, funding := range fundings {
		if funding.GrantedQuota <= 0 {
			continue
		}
		if funding.FundingType == QuotaFundingTypePaid {
			paidGrantedQuota += funding.GrantedQuota
			salesIncomeUSD += funding.RecognizedRevenueUSDTotal
			continue
		}
		giftGrantedQuota += funding.GrantedQuota
		if funding.QuotaPerUnitSnapshot > 0 {
			giftGrantedUSD += float64(funding.GrantedQuota) / funding.QuotaPerUnitSnapshot
		} else {
			giftGrantedUSD += quotaToUSDWithSnapshot(funding.GrantedQuota, common.QuotaPerUnit)
		}
	}
	return paidGrantedQuota, giftGrantedQuota, roundAccountingAmount(giftGrantedUSD), roundAccountingAmount(salesIncomeUSD)
}

func errorsIsRecordNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound || (err != nil && err.Error() == gorm.ErrRecordNotFound.Error())
}
