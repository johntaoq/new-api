package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"gorm.io/gorm"
)

type AdminBillingFundingSummaryItem struct {
	FundingType            string  `json:"funding_type"`
	SourceType             string  `json:"source_type"`
	EntryCount             int     `json:"entry_count"`
	UserCount              int     `json:"user_count"`
	GrantedQuota           int     `json:"granted_quota"`
	RemainingQuota         int     `json:"remaining_quota"`
	RecognizedRevenueUSD   float64 `json:"recognized_revenue_usd"`
	GrantedEquivalentUSD   float64 `json:"granted_equivalent_usd"`
	RemainingEquivalentUSD float64 `json:"remaining_equivalent_usd"`
}

type AdminBillingCustomerSummaryItem struct {
	UserId                int     `json:"user_id"`
	Username              string  `json:"username"`
	ConsumeCount          int     `json:"consume_count"`
	RefundCount           int     `json:"refund_count"`
	ActualQuota           int     `json:"actual_quota"`
	PaidQuotaUsed         int     `json:"paid_quota_used"`
	GiftQuotaUsed         int     `json:"gift_quota_used"`
	RecognizedRevenueUSD  float64 `json:"recognized_revenue_usd"`
	EstimatedCostUSD      float64 `json:"estimated_cost_usd"`
	GrossProfitUSD        float64 `json:"gross_profit_usd"`
	InternalEquivalentUSD float64 `json:"internal_equivalent_usd"`
}

type AdminBillingOverview struct {
	BillMonth                       string                            `json:"bill_month"`
	PeriodStart                     int64                             `json:"period_start"`
	PeriodEnd                       int64                             `json:"period_end"`
	ActiveUserCount                 int                               `json:"active_user_count"`
	FundingUserCount                int                               `json:"funding_user_count"`
	GeneratedStatementCount         int64                             `json:"generated_statement_count"`
	TotalPaidGrantedQuota           int                               `json:"total_paid_granted_quota"`
	TotalGiftGrantedQuota           int                               `json:"total_gift_granted_quota"`
	TotalGiftRemainingQuota         int                               `json:"total_gift_remaining_quota"`
	TotalSalesIncomeUSD             float64                           `json:"total_sales_income_usd"`
	TotalGiftGrantedEquivalentUSD   float64                           `json:"total_gift_granted_equivalent_usd"`
	TotalGiftRemainingEquivalentUSD float64                           `json:"total_gift_remaining_equivalent_usd"`
	UsesLegacyLogsFallback          bool                              `json:"uses_legacy_logs_fallback"`
	FallbackNote                    string                            `json:"fallback_note"`
	ChannelStatement                *ChannelMonthlyStatement          `json:"channel_statement"`
	FundingSummary                  []AdminBillingFundingSummaryItem  `json:"funding_summary"`
	CustomerSummary                 []AdminBillingCustomerSummaryItem `json:"customer_summary"`
}

func BuildAdminBillingOverview(billMonth string) (*AdminBillingOverview, error) {
	normalizedBillMonth, periodStart, periodEnd, err := ParseCustomerStatementBillMonth(billMonth)
	if err != nil {
		return nil, err
	}

	channelStatement, err := GenerateChannelMonthlyStatement(normalizedBillMonth, false)
	if err != nil {
		return nil, err
	}

	var fundings []UserQuotaFunding
	if err := DB.Where("created_at >= ? AND created_at <= ?", periodStart, periodEnd).
		Order("created_at ASC").
		Order("id ASC").
		Find(&fundings).Error; err != nil {
		return nil, err
	}

	var ledgers []ChannelCostLedger
	if err := DB.Where("bill_month = ? AND occurred_at >= ? AND occurred_at <= ?", normalizedBillMonth, periodStart, periodEnd).
		Order("occurred_at ASC").
		Order("id ASC").
		Find(&ledgers).Error; err != nil {
		return nil, err
	}

	useLegacyLogsFallback := len(ledgers) == 0
	legacyLogs := make([]*Log, 0)
	if useLegacyLogsFallback {
		legacyLogs, err = loadLegacyBillingLogs(periodStart, periodEnd)
		if err != nil {
			return nil, err
		}
	}

	var generatedStatementCount int64
	if err := DB.Model(&CustomerMonthlyStatement{}).Where("bill_month = ?", normalizedBillMonth).Count(&generatedStatementCount).Error; err != nil {
		return nil, err
	}

	fundingSummary, fundingTotals, fundingUserCount := buildAdminFundingSummary(fundings)
	var customerSummary []AdminBillingCustomerSummaryItem
	activeUserCount := 0
	if useLegacyLogsFallback && len(legacyLogs) > 0 {
		customerSummary, activeUserCount, err = buildAdminCustomerSummaryFromLegacyLogs(legacyLogs)
		if err != nil {
			return nil, err
		}
	} else {
		customerSummary, activeUserCount, err = buildAdminCustomerSummary(ledgers)
		if err != nil {
			return nil, err
		}
	}

	return &AdminBillingOverview{
		BillMonth:                       normalizedBillMonth,
		PeriodStart:                     periodStart,
		PeriodEnd:                       periodEnd,
		ActiveUserCount:                 activeUserCount,
		FundingUserCount:                fundingUserCount,
		GeneratedStatementCount:         generatedStatementCount,
		TotalPaidGrantedQuota:           fundingTotals.TotalPaidGrantedQuota,
		TotalGiftGrantedQuota:           fundingTotals.TotalGiftGrantedQuota,
		TotalGiftRemainingQuota:         fundingTotals.TotalGiftRemainingQuota,
		TotalSalesIncomeUSD:             fundingTotals.TotalSalesIncomeUSD,
		TotalGiftGrantedEquivalentUSD:   fundingTotals.TotalGiftGrantedEquivalentUSD,
		TotalGiftRemainingEquivalentUSD: fundingTotals.TotalGiftRemainingEquivalentUSD,
		UsesLegacyLogsFallback:          useLegacyLogsFallback && len(legacyLogs) > 0,
		FallbackNote:                    buildAdminBillingFallbackNote(useLegacyLogsFallback && len(legacyLogs) > 0),
		ChannelStatement:                channelStatement,
		FundingSummary:                  fundingSummary,
		CustomerSummary:                 customerSummary,
	}, nil
}

type adminFundingSummaryTotals struct {
	TotalPaidGrantedQuota           int
	TotalGiftGrantedQuota           int
	TotalGiftRemainingQuota         int
	TotalSalesIncomeUSD             float64
	TotalGiftGrantedEquivalentUSD   float64
	TotalGiftRemainingEquivalentUSD float64
}

func buildAdminFundingSummary(fundings []UserQuotaFunding) ([]AdminBillingFundingSummaryItem, adminFundingSummaryTotals, int) {
	summaryMap := make(map[string]*AdminBillingFundingSummaryItem)
	userSets := make(map[string]map[int]struct{})
	fundingUserSet := make(map[int]struct{})
	totals := adminFundingSummaryTotals{}

	for _, funding := range fundings {
		key := fmt.Sprintf("%s|%s", funding.FundingType, funding.SourceType)
		item, ok := summaryMap[key]
		if !ok {
			item = &AdminBillingFundingSummaryItem{
				FundingType: funding.FundingType,
				SourceType:  funding.SourceType,
			}
			summaryMap[key] = item
			userSets[key] = make(map[int]struct{})
		}

		item.EntryCount++
		item.GrantedQuota += funding.GrantedQuota
		item.RemainingQuota += funding.RemainingQuota
		item.RecognizedRevenueUSD = roundAccountingAmount(item.RecognizedRevenueUSD + funding.RecognizedRevenueUSDTotal)
		item.GrantedEquivalentUSD = roundAccountingAmount(item.GrantedEquivalentUSD + quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding)))
		item.RemainingEquivalentUSD = roundAccountingAmount(item.RemainingEquivalentUSD + quotaToUSDWithSnapshot(funding.RemainingQuota, fundingQuotaSnapshot(funding)))
		userSets[key][funding.UserId] = struct{}{}
		fundingUserSet[funding.UserId] = struct{}{}

		if funding.FundingType == QuotaFundingTypePaid {
			totals.TotalPaidGrantedQuota += funding.GrantedQuota
			totals.TotalSalesIncomeUSD = roundAccountingAmount(totals.TotalSalesIncomeUSD + funding.RecognizedRevenueUSDTotal)
			continue
		}
		totals.TotalGiftGrantedQuota += funding.GrantedQuota
		totals.TotalGiftRemainingQuota += funding.RemainingQuota
		totals.TotalGiftGrantedEquivalentUSD = roundAccountingAmount(totals.TotalGiftGrantedEquivalentUSD + quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding)))
		totals.TotalGiftRemainingEquivalentUSD = roundAccountingAmount(totals.TotalGiftRemainingEquivalentUSD + quotaToUSDWithSnapshot(funding.RemainingQuota, fundingQuotaSnapshot(funding)))
	}

	items := make([]AdminBillingFundingSummaryItem, 0, len(summaryMap))
	for key, item := range summaryMap {
		item.UserCount = len(userSets[key])
		items = append(items, *item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].FundingType != items[j].FundingType {
			return items[i].FundingType < items[j].FundingType
		}
		if items[i].GrantedEquivalentUSD != items[j].GrantedEquivalentUSD {
			return items[i].GrantedEquivalentUSD > items[j].GrantedEquivalentUSD
		}
		return items[i].SourceType < items[j].SourceType
	})

	return items, totals, len(fundingUserSet)
}

func buildAdminCustomerSummary(ledgers []ChannelCostLedger) ([]AdminBillingCustomerSummaryItem, int, error) {
	customerMap := make(map[int]*AdminBillingCustomerSummaryItem)

	for _, ledger := range ledgers {
		sign := 1
		if ledger.EntryType == ChannelCostEntryTypeRefund {
			sign = -1
		}

		item, ok := customerMap[ledger.UserId]
		if !ok {
			item = &AdminBillingCustomerSummaryItem{
				UserId: ledger.UserId,
			}
			customerMap[ledger.UserId] = item
		}

		if ledger.EntryType == ChannelCostEntryTypeRefund {
			item.RefundCount++
		} else {
			item.ConsumeCount++
		}
		item.ActualQuota += sign * ledger.ActualQuota
		item.PaidQuotaUsed += sign * ledger.PaidQuotaUsed
		item.GiftQuotaUsed += sign * ledger.GiftQuotaUsed
		item.InternalEquivalentUSD = roundAccountingAmount(item.InternalEquivalentUSD + float64(sign)*ledger.InternalEquivalentUSD)
		item.RecognizedRevenueUSD = roundAccountingAmount(item.RecognizedRevenueUSD + float64(sign)*ledger.RecognizedRevenueUSD)
		item.EstimatedCostUSD = roundAccountingAmount(item.EstimatedCostUSD + float64(sign)*ledger.EstimatedCostUSD)
		item.GrossProfitUSD = roundAccountingAmount(item.GrossProfitUSD + float64(sign)*(ledger.RecognizedRevenueUSD-ledger.EstimatedCostUSD))
	}

	userIds := make([]int, 0, len(customerMap))
	for userId := range customerMap {
		userIds = append(userIds, userId)
	}

	if len(userIds) > 0 {
		var users []User
		if err := DB.Unscoped().Where("id IN ?", userIds).Find(&users).Error; err != nil {
			return nil, 0, err
		}
		for _, user := range users {
			if item, ok := customerMap[user.Id]; ok {
				item.Username = user.Username
			}
		}
	}

	items := make([]AdminBillingCustomerSummaryItem, 0, len(customerMap))
	for _, item := range customerMap {
		if strings.TrimSpace(item.Username) == "" {
			item.Username = fmt.Sprintf("user-%d", item.UserId)
		}
		items = append(items, *item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].RecognizedRevenueUSD != items[j].RecognizedRevenueUSD {
			return items[i].RecognizedRevenueUSD > items[j].RecognizedRevenueUSD
		}
		if items[i].ActualQuota != items[j].ActualQuota {
			return items[i].ActualQuota > items[j].ActualQuota
		}
		return items[i].UserId < items[j].UserId
	})

	if len(items) > 200 {
		items = items[:200]
	}

	return items, len(customerMap), nil
}

func buildAdminCustomerSummaryFromLegacyLogs(logs []*Log) ([]AdminBillingCustomerSummaryItem, int, error) {
	customerMap := make(map[int]*AdminBillingCustomerSummaryItem)

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
		}

		item, ok := customerMap[logRecord.UserId]
		if !ok {
			item = &AdminBillingCustomerSummaryItem{
				UserId:   logRecord.UserId,
				Username: logRecord.Username,
			}
			customerMap[logRecord.UserId] = item
		}

		if entryType == CustomerMonthlyStatementEntryTypeRefund {
			item.RefundCount++
		} else {
			item.ConsumeCount++
		}

		other := parseStatementOther(logRecord.Other)
		paidQuotaUsed, giftQuotaUsed := legacyBillingQuotaSplit(logRecord, other)
		recognizedRevenueUSD := legacyBillingRecognizedRevenueUSD(logRecord, other)
		estimatedCostUSD := legacyBillingEstimatedCostUSD(logRecord, other)
		internalEquivalentUSD := legacyBillingInternalEquivalentUSD(logRecord)

		item.ActualQuota += sign * logRecord.Quota
		item.PaidQuotaUsed += sign * paidQuotaUsed
		item.GiftQuotaUsed += sign * giftQuotaUsed
		item.InternalEquivalentUSD = roundAccountingAmount(item.InternalEquivalentUSD + float64(sign)*internalEquivalentUSD)
		item.RecognizedRevenueUSD = roundAccountingAmount(item.RecognizedRevenueUSD + float64(sign)*recognizedRevenueUSD)
		item.EstimatedCostUSD = roundAccountingAmount(item.EstimatedCostUSD + float64(sign)*estimatedCostUSD)
		item.GrossProfitUSD = roundAccountingAmount(item.GrossProfitUSD + float64(sign)*(recognizedRevenueUSD-estimatedCostUSD))
	}

	userIds := make([]int, 0, len(customerMap))
	for userId := range customerMap {
		userIds = append(userIds, userId)
	}
	userSnapshots, err := loadLegacyBillingUserSnapshots(userIds)
	if err != nil {
		return nil, 0, err
	}
	for userId, snapshot := range userSnapshots {
		if item, ok := customerMap[userId]; ok && strings.TrimSpace(snapshot.Username) != "" {
			item.Username = snapshot.Username
		}
	}

	items := make([]AdminBillingCustomerSummaryItem, 0, len(customerMap))
	for _, item := range customerMap {
		if strings.TrimSpace(item.Username) == "" {
			item.Username = fmt.Sprintf("user-%d", item.UserId)
		}
		items = append(items, *item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].RecognizedRevenueUSD != items[j].RecognizedRevenueUSD {
			return items[i].RecognizedRevenueUSD > items[j].RecognizedRevenueUSD
		}
		if items[i].ActualQuota != items[j].ActualQuota {
			return items[i].ActualQuota > items[j].ActualQuota
		}
		return items[i].UserId < items[j].UserId
	})

	if len(items) > 200 {
		items = items[:200]
	}

	return items, len(customerMap), nil
}

func buildAdminBillingFallbackNote(usingLegacyLogs bool) string {
	if !usingLegacyLogs {
		return ""
	}
	return "该月份缺少新版账务台账，当前页面已回退到历史 logs 重建；销售收入、赠送来源和付费/赠送拆分可能只有部分数据。"
}

func fundingQuotaSnapshot(funding UserQuotaFunding) float64 {
	if funding.QuotaPerUnitSnapshot > 0 {
		return funding.QuotaPerUnitSnapshot
	}
	return common.QuotaPerUnit
}

func ConsumeUserQuotaByFundingType(userId int, quota int, fundingType string) ([]types.QuotaFundingAllocation, error) {
	var allocations []types.QuotaFundingAllocation
	var err error
	fundingType = normalizeQuotaFundingType(fundingType)

	err = DB.Transaction(func(tx *gorm.DB) error {
		allocations, err = consumeUserQuotaByFundingTypeTx(tx, userId, quota, fundingType)
		return err
	})

	return allocations, err
}

func consumeUserQuotaByFundingTypeTx(tx *gorm.DB, userId int, quota int, fundingType string) ([]types.QuotaFundingAllocation, error) {
	if quota < 0 {
		return nil, fmt.Errorf("quota must not be negative")
	}
	if quota == 0 {
		return []types.QuotaFundingAllocation{}, nil
	}

	var user User
	if err := tx.Where("id = ?", userId).First(&user).Error; err != nil {
		return nil, err
	}
	if err := ensureLegacyQuotaFundingCoverageTx(tx, &user); err != nil {
		return nil, err
	}

	availableQuota := user.GiftQuota
	if fundingType == QuotaFundingTypePaid {
		availableQuota = user.PaidQuota
	}
	if availableQuota < quota {
		return nil, fmt.Errorf("insufficient %s quota: remain=%d need=%d", fundingType, availableQuota, quota)
	}

	var fundings []UserQuotaFunding
	if err := tx.Where("user_id = ? AND funding_type = ? AND remaining_quota > 0", userId, fundingType).
		Order("created_at ASC").
		Order("id ASC").
		Find(&fundings).Error; err != nil {
		return nil, err
	}

	remainingQuota := quota
	allocations := make([]types.QuotaFundingAllocation, 0, len(fundings))
	for _, funding := range fundings {
		if remainingQuota <= 0 {
			break
		}
		if funding.RemainingQuota <= 0 {
			continue
		}

		used := funding.RemainingQuota
		if used > remainingQuota {
			used = remainingQuota
		}
		if used <= 0 {
			continue
		}

		nextRemaining := funding.RemainingQuota - used
		if err := tx.Model(&UserQuotaFunding{}).Where("id = ?", funding.Id).Updates(map[string]interface{}{
			"remaining_quota": nextRemaining,
			"updated_at":      common.GetTimestamp(),
		}).Error; err != nil {
			return nil, err
		}

		revenueUSD := 0.0
		if funding.GrantedQuota > 0 && funding.RecognizedRevenueUSDTotal > 0 {
			revenueUSD = funding.RecognizedRevenueUSDTotal * float64(used) / float64(funding.GrantedQuota)
		}
		allocations = append(allocations, types.QuotaFundingAllocation{
			FundingId:           funding.Id,
			FundingType:         funding.FundingType,
			SourceType:          funding.SourceType,
			AllocatedQuota:      used,
			AllocatedRevenueUSD: roundAccountingAmount(revenueUSD),
		})
		remainingQuota -= used
	}

	if remainingQuota > 0 {
		return nil, fmt.Errorf("failed to allocate %s quota, remaining=%d", fundingType, remainingQuota)
	}

	if fundingType == QuotaFundingTypePaid {
		user.PaidQuota -= quota
	} else {
		user.GiftQuota -= quota
	}
	user.Quota = user.PaidQuota + user.GiftQuota
	if user.PaidQuota < 0 || user.GiftQuota < 0 || user.Quota < 0 {
		return nil, fmt.Errorf("quota pool became negative")
	}

	if err := tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"paid_quota": user.PaidQuota,
		"gift_quota": user.GiftQuota,
		"quota":      user.Quota,
	}).Error; err != nil {
		return nil, err
	}

	if err := updateUserCache(user); err != nil {
		common.SysLog("failed to update user cache after specific consume: " + err.Error())
	}

	return allocations, nil
}
