package model

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const (
	FinancePeriodTypeMonth = "month"
	FinancePeriodTypeYear  = "year"

	FinanceRankingViewIncome    = "income"
	FinanceRankingViewUsage     = "usage"
	FinanceRankingViewPaidUsage = "paid_usage"
	FinanceRankingViewChannel   = "channel_cost"
)

type FinanceAmount struct {
	USD      float64 `json:"usd"`
	COS      float64 `json:"cos"`
	Platform int     `json:"-"`
}

type FinanceDashboardKPI struct {
	SalesIncome         FinanceAmount `json:"sales_income"`
	GiftGranted         FinanceAmount `json:"gift_granted"`
	ChannelCost         FinanceAmount `json:"channel_cost"`
	CurrentPaidBalance  FinanceAmount `json:"current_paid_balance"`
	CurrentGiftBalance  FinanceAmount `json:"current_gift_balance"`
	ActiveCustomerCount int           `json:"active_customer_count"`
	PaidCustomerCount   int           `json:"paid_customer_count"`
}

type FinanceAlertItem struct {
	Type            string  `json:"type"`
	Title           string  `json:"title"`
	AbnormalValue   float64 `json:"abnormal_value"`
	BaselineValue   float64 `json:"baseline_value"`
	SuggestedAction string  `json:"suggested_action"`
	LastOccurredAt  int64   `json:"last_occurred_at"`
}

type FinanceDashboardSummary struct {
	PeriodType string              `json:"period_type"`
	Period     string              `json:"period"`
	KPIs       FinanceDashboardKPI `json:"kpis"`
	Alerts     []FinanceAlertItem  `json:"alerts"`
}

type FinanceTodoItem struct {
	Type            string `json:"type"`
	TargetType      string `json:"target_type"`
	TargetID        int    `json:"target_id"`
	TargetName      string `json:"target_name"`
	AbnormalValue   string `json:"abnormal_value"`
	SuggestedAction string `json:"suggested_action"`
	LastOccurredAt  int64  `json:"last_occurred_at"`
}

type FinanceRankingItem struct {
	Rank          int     `json:"rank"`
	TargetType    string  `json:"target_type"`
	TargetID      int     `json:"target_id"`
	TargetName    string  `json:"target_name"`
	ValueUSD      float64 `json:"value_usd"`
	ValueCOS      float64 `json:"value_cos"`
	ValuePlatform int     `json:"-"`
	ValueTokens   int     `json:"value_tokens"`
	Extra         string  `json:"extra"`
}

type FinanceRevenueSummary struct {
	PeriodType            string        `json:"period_type"`
	Period                string        `json:"period"`
	PaidRechargeTotal     FinanceAmount `json:"paid_recharge_total"`
	GiftGrantedTotal      FinanceAmount `json:"gift_granted_total"`
	UnissuedPaidVoucher   FinanceAmount `json:"unissued_paid_voucher_total"`
	UnredeemedGiftVoucher FinanceAmount `json:"unredeemed_gift_voucher_total"`
}

type FinancePaidSourceSummaryItem struct {
	SourceType              string  `json:"source_type"`
	SourceLabel             string  `json:"source_label"`
	RechargeCount           int     `json:"recharge_count"`
	UserCount               int     `json:"user_count"`
	RechargeAmountUSD       float64 `json:"recharge_amount_usd"`
	RechargeAmountCOS       float64 `json:"recharge_amount_cos"`
	RechargeAmountPlatform  int     `json:"-"`
	RemainingAmountUSD      float64 `json:"remaining_amount_usd"`
	RemainingAmountCOS      float64 `json:"remaining_amount_cos"`
	RemainingAmountPlatform int     `json:"-"`
}

type FinancePaidSourceDetailItem struct {
	CreatedAt               int64   `json:"created_at"`
	SourceType              string  `json:"source_type"`
	SourceLabel             string  `json:"source_label"`
	UserID                  int     `json:"user_id"`
	Username                string  `json:"username"`
	RechargeAmountUSD       float64 `json:"recharge_amount_usd"`
	RechargeAmountCOS       float64 `json:"recharge_amount_cos"`
	RechargeAmountPlatform  int     `json:"-"`
	RemainingAmountUSD      float64 `json:"remaining_amount_usd"`
	RemainingAmountCOS      float64 `json:"remaining_amount_cos"`
	RemainingAmountPlatform int     `json:"-"`
	SourceRefID             int     `json:"source_ref_id"`
	SourceName              string  `json:"source_name"`
	Remark                  string  `json:"remark"`
}

type FinanceGiftAuditItem struct {
	CreatedAt             int64   `json:"created_at"`
	UserID                int     `json:"user_id"`
	Username              string  `json:"username"`
	SourceType            string  `json:"source_type"`
	SourceLabel           string  `json:"source_label"`
	GrantedAmountUSD      float64 `json:"granted_amount_usd"`
	GrantedAmountCOS      float64 `json:"granted_amount_cos"`
	GrantedAmountPlatform int     `json:"-"`
	OperatorUserID        int     `json:"operator_user_id"`
	OperatorUsername      string  `json:"operator_username"`
	VoucherCode           string  `json:"voucher_code"`
	Remark                string  `json:"remark"`
}

type FinanceGiftAuditSummaryItem struct {
	SourceType            string  `json:"source_type"`
	SourceLabel           string  `json:"source_label"`
	ActivityDescription   string  `json:"activity_description"`
	ExchangeCount         int     `json:"exchange_count"`
	GrantedAmountUSD      float64 `json:"granted_amount_usd"`
	GrantedAmountCOS      float64 `json:"granted_amount_cos"`
	GrantedAmountPlatform int     `json:"-"`
}

type FinanceChannelCostChartPoint struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type FinanceChannelCostSummary struct {
	PeriodType string `json:"period_type"`
	Period     string `json:"period"`
	KPIs       struct {
		TotalCostUSD float64 `json:"total_cost_usd"`
		TotalTokens  int     `json:"total_tokens"`
	} `json:"kpis"`
	Charts struct {
		ByChannelUSD    []FinanceChannelCostChartPoint `json:"by_channel_usd"`
		ByChannelTokens []FinanceChannelCostChartPoint `json:"by_channel_tokens"`
		ByModelUSD      []FinanceChannelCostChartPoint `json:"by_model_usd"`
		ByModelTokens   []FinanceChannelCostChartPoint `json:"by_model_tokens"`
	} `json:"charts"`
}

type FinanceChannelCostChannelItem struct {
	ChannelID        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	ProviderSnapshot string  `json:"provider_snapshot"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	CacheTokens      int     `json:"cache_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	RevenueUSD       float64 `json:"revenue_usd"`
	CostUSD          float64 `json:"cost_usd"`
}

type FinanceChannelCostModelItem struct {
	ChannelID        int     `json:"channel_id"`
	ChannelName      string  `json:"channel_name"`
	ProviderSnapshot string  `json:"provider_snapshot"`
	ModelName        string  `json:"model_name"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	CacheTokens      int     `json:"cache_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	RevenueUSD       float64 `json:"revenue_usd"`
	CostUSD          float64 `json:"cost_usd"`
}

type FinanceCustomerBillSummaryItem struct {
	UserID                 int     `json:"user_id"`
	Username               string  `json:"username"`
	CurrentBalanceUSD      float64 `json:"current_balance_usd"`
	CurrentBalanceCOS      float64 `json:"current_balance_cos"`
	CurrentBalancePlatform int     `json:"-"`
	TotalConsumeUSD        float64 `json:"total_consume_usd"`
	TotalConsumeCOS        float64 `json:"total_consume_cos"`
	TotalConsumePlatform   int     `json:"-"`
	PaidConsumeUSD         float64 `json:"paid_consume_usd"`
	PaidConsumeCOS         float64 `json:"paid_consume_cos"`
	PaidConsumePlatform    int     `json:"-"`
	GiftConsumeUSD         float64 `json:"gift_consume_usd"`
	GiftConsumeCOS         float64 `json:"gift_consume_cos"`
	GiftConsumePlatform    int     `json:"-"`
}

type FinanceCustomerBillDetailItem struct {
	OccurredAt     int64   `json:"occurred_at"`
	TokenDisplay   string  `json:"token_display"`
	EntryType      string  `json:"entry_type"`
	ModelName      string  `json:"model_name"`
	AmountCOS      float64 `json:"amount_cos"`
	AmountPlatform int     `json:"-"`
	AmountUSD      float64 `json:"amount_usd"`
	ChannelName    string  `json:"channel_name"`
	RequestID      string  `json:"request_id"`
	Remark         string  `json:"remark"`
}

type financePeriodRange struct {
	PeriodType    string
	Period        string
	Start         int64
	End           int64
	PreviousStart int64
	PreviousEnd   int64
}

type financeUserUsageAggregate struct {
	UserID              int
	ConsumePlatform     int
	PaidConsumePlatform int
	GiftConsumePlatform int
	ConsumeUSD          float64
	PaidConsumeUSD      float64
	GiftConsumeUSD      float64
	ConsumeCount        int
	LastOccurredAt      int64
}

type financeChannelAggregate struct {
	ChannelID        int
	ChannelName      string
	ProviderSnapshot string
	ModelName        string
	PromptTokens     int
	CompletionTokens int
	CacheTokens      int
	TotalTokens      int
	RevenueUSD       float64
	CostUSD          float64
}

func parseFinancePeriod(periodType string, period string) (financePeriodRange, error) {
	normalizedType := strings.TrimSpace(periodType)
	if normalizedType != FinancePeriodTypeYear {
		normalizedType = FinancePeriodTypeMonth
	}

	if normalizedType == FinancePeriodTypeMonth {
		normalizedPeriod, start, end, err := ParseCustomerStatementBillMonth(period)
		if err != nil {
			return financePeriodRange{}, err
		}
		currentStart := time.Unix(start, 0).In(time.Local)
		previousStart := currentStart.AddDate(0, -1, 0)
		return financePeriodRange{
			PeriodType:    normalizedType,
			Period:        normalizedPeriod,
			Start:         start,
			End:           end,
			PreviousStart: previousStart.Unix(),
			PreviousEnd:   currentStart.Add(-time.Second).Unix(),
		}, nil
	}

	normalizedPeriod := strings.TrimSpace(period)
	if normalizedPeriod == "" {
		normalizedPeriod = time.Now().In(time.Local).Format("2006")
	}
	yearStart, err := time.ParseInLocation("2006", normalizedPeriod, time.Local)
	if err != nil {
		return financePeriodRange{}, fmt.Errorf("invalid period, expected YYYY")
	}
	return financePeriodRange{
		PeriodType:    normalizedType,
		Period:        yearStart.Format("2006"),
		Start:         yearStart.Unix(),
		End:           yearStart.AddDate(1, 0, 0).Add(-time.Second).Unix(),
		PreviousStart: yearStart.AddDate(-1, 0, 0).Unix(),
		PreviousEnd:   yearStart.Add(-time.Second).Unix(),
	}, nil
}

func financeSourceLabel(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case QuotaFundingSourceOnlineTopUp:
		return "在线充值"
	case QuotaFundingSourceStripe:
		return "Stripe"
	case QuotaFundingSourceCreem:
		return "Creem"
	case QuotaFundingSourceWaffo:
		return "Waffo"
	case QuotaFundingSourcePaidVoucher:
		return "付费兑换码"
	case QuotaFundingSourceGiftVoucher:
		return "免费兑换码"
	case QuotaFundingSourceInviteReward:
		return "邀请奖励"
	case QuotaFundingSourceSignupBonus:
		return "新开户赠送"
	case QuotaFundingSourceAdminGrant:
		return "管理员赠送"
	case QuotaFundingSourcePromoCampaign:
		return "活动赠送"
	case QuotaFundingSourceCompensation:
		return "补偿"
	case QuotaFundingSourceSystemAdjust:
		return "系统修正"
	case QuotaFundingSourceCheckin:
		return "签到赠送"
	case QuotaFundingSourceLegacyBalance:
		return "历史付费余额"
	case QuotaFundingSourceLegacyGift:
		return "历史赠送余额"
	default:
		if sourceType == "" {
			return "-"
		}
		return sourceType
	}
}

func financeAmountFromQuota(quota int, snapshot float64) FinanceAmount {
	usd := roundAccountingAmount(quotaToUSDWithSnapshot(quota, snapshot))
	return FinanceAmount{
		USD:      usd,
		COS:      financeCOSAmountFromUSD(usd),
		Platform: quota,
	}
}

func financeAmountFromUSDPlatform(usd float64, platform int) FinanceAmount {
	normalizedUSD := roundAccountingAmount(usd)
	return FinanceAmount{
		USD:      normalizedUSD,
		COS:      financeCOSAmountFromUSD(normalizedUSD),
		Platform: platform,
	}
}

func financeCOSRate() float64 {
	rate := operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate
	if rate <= 0 {
		return 1
	}
	return rate
}

func financeCOSAmountFromUSD(usd float64) float64 {
	return roundAccountingAmount(usd * financeCOSRate())
}

func financeVoucherCodeMap(fundings []UserQuotaFunding) (map[int]string, error) {
	ids := make([]int, 0)
	seen := make(map[int]struct{})
	for _, funding := range fundings {
		if funding.SourceRefId <= 0 {
			continue
		}
		if funding.SourceType != QuotaFundingSourceGiftVoucher && funding.SourceType != QuotaFundingSourcePaidVoucher {
			continue
		}
		if _, ok := seen[funding.SourceRefId]; ok {
			continue
		}
		seen[funding.SourceRefId] = struct{}{}
		ids = append(ids, funding.SourceRefId)
	}
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	var redemptions []Redemption
	if err := DB.Unscoped().Where("id IN ?", ids).Find(&redemptions).Error; err != nil {
		return nil, err
	}
	result := make(map[int]string, len(redemptions))
	for _, redemption := range redemptions {
		result[redemption.Id] = redemption.Key
	}
	return result, nil
}

func financeUserMap(userIDs []int) (map[int]User, error) {
	if len(userIDs) == 0 {
		return map[int]User{}, nil
	}
	var users []User
	if err := DB.Unscoped().Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	result := make(map[int]User, len(users))
	for _, user := range users {
		result[user.Id] = user
	}
	return result, nil
}

func financePageItems[T any](items []T, pageInfo *common.PageInfo) []T {
	if pageInfo == nil {
		return items
	}
	start := pageInfo.GetStartIdx()
	if start >= len(items) {
		return []T{}
	}
	end := start + pageInfo.GetPageSize()
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func financeGiftAuditActivityDescription(funding UserQuotaFunding) string {
	return firstNonEmpty(funding.Remark, funding.SourceName, "-")
}

func financeMatchUserIDs(userKeyword string) ([]int, error) {
	keyword := strings.TrimSpace(userKeyword)
	if keyword == "" {
		return nil, nil
	}
	query := DB.Unscoped().Model(&User{})
	lowerKeyword := strings.ToLower(keyword)
	if userID, err := strconv.Atoi(keyword); err == nil && userID > 0 {
		query = query.Where("id = ? OR LOWER(username) LIKE ?", userID, "%"+lowerKeyword+"%")
	} else {
		query = query.Where("LOWER(username) LIKE ?", "%"+lowerKeyword+"%")
	}
	var users []User
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.Id)
	}
	return ids, nil
}

func financeBuildUserUsageFromLedgers(ledgers []ChannelCostLedger) map[int]*financeUserUsageAggregate {
	result := make(map[int]*financeUserUsageAggregate)
	for _, ledger := range ledgers {
		sign := 1
		if ledger.EntryType == ChannelCostEntryTypeRefund {
			sign = -1
		}
		item := result[ledger.UserId]
		if item == nil {
			item = &financeUserUsageAggregate{UserID: ledger.UserId}
			result[ledger.UserId] = item
		}
		if ledger.EntryType == ChannelCostEntryTypeConsume {
			item.ConsumeCount++
		}
		item.ConsumePlatform += sign * ledger.ActualQuota
		item.PaidConsumePlatform += sign * ledger.PaidQuotaUsed
		item.GiftConsumePlatform += sign * ledger.GiftQuotaUsed
		item.ConsumeUSD = roundAccountingAmount(item.ConsumeUSD + float64(sign)*ledger.InternalEquivalentUSD)
		item.PaidConsumeUSD = roundAccountingAmount(item.PaidConsumeUSD + float64(sign)*quotaToUSDWithSnapshot(ledger.PaidQuotaUsed, common.QuotaPerUnit))
		item.GiftConsumeUSD = roundAccountingAmount(item.GiftConsumeUSD + float64(sign)*quotaToUSDWithSnapshot(ledger.GiftQuotaUsed, common.QuotaPerUnit))
		if ledger.OccurredAt > item.LastOccurredAt {
			item.LastOccurredAt = ledger.OccurredAt
		}
	}
	return result
}

func financeBuildUserUsageFromLogs(logs []*Log) map[int]*financeUserUsageAggregate {
	result := make(map[int]*financeUserUsageAggregate)
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
		item := result[logRecord.UserId]
		if item == nil {
			item = &financeUserUsageAggregate{UserID: logRecord.UserId}
			result[logRecord.UserId] = item
		}
		if entryType == CustomerMonthlyStatementEntryTypeConsume {
			item.ConsumeCount++
		}
		other := parseStatementOther(logRecord.Other)
		paidQuotaUsed, giftQuotaUsed := legacyBillingQuotaSplit(logRecord, other)
		item.ConsumePlatform += sign * logRecord.Quota
		item.PaidConsumePlatform += sign * paidQuotaUsed
		item.GiftConsumePlatform += sign * giftQuotaUsed
		item.ConsumeUSD = roundAccountingAmount(item.ConsumeUSD + float64(sign)*legacyBillingInternalEquivalentUSD(logRecord))
		item.PaidConsumeUSD = roundAccountingAmount(item.PaidConsumeUSD + float64(sign)*quotaToUSDWithSnapshot(paidQuotaUsed, common.QuotaPerUnit))
		item.GiftConsumeUSD = roundAccountingAmount(item.GiftConsumeUSD + float64(sign)*quotaToUSDWithSnapshot(giftQuotaUsed, common.QuotaPerUnit))
		if logRecord.CreatedAt > item.LastOccurredAt {
			item.LastOccurredAt = logRecord.CreatedAt
		}
	}
	return result
}

func financeBuildChannelAggregatesFromLedgers(ledgers []ChannelCostLedger) []financeChannelAggregate {
	aggregateMap := make(map[string]*financeChannelAggregate)
	for _, ledger := range ledgers {
		sign := 1
		if ledger.EntryType == ChannelCostEntryTypeRefund {
			sign = -1
		}
		key := fmt.Sprintf("%d|%s|%s|%s", ledger.ChannelId, ledger.ChannelNameSnapshot, ledger.ProviderSnapshot, ledger.OriginModelName)
		item := aggregateMap[key]
		if item == nil {
			item = &financeChannelAggregate{
				ChannelID:        ledger.ChannelId,
				ChannelName:      ledger.ChannelNameSnapshot,
				ProviderSnapshot: ledger.ProviderSnapshot,
				ModelName:        ledger.OriginModelName,
			}
			aggregateMap[key] = item
		}
		item.PromptTokens += sign * ledger.PromptTokens
		item.CompletionTokens += sign * ledger.CompletionTokens
		item.CacheTokens += sign * ledger.CacheTokens
		item.TotalTokens += sign * ledger.TotalTokens
		item.RevenueUSD = roundAccountingAmount(item.RevenueUSD + float64(sign)*ledger.RecognizedRevenueUSD)
		item.CostUSD = roundAccountingAmount(item.CostUSD + float64(sign)*ledger.EstimatedCostUSD)
	}
	items := make([]financeChannelAggregate, 0, len(aggregateMap))
	for _, item := range aggregateMap {
		items = append(items, *item)
	}
	return items
}

func financeBuildChannelAggregatesFromLogs(logs []*Log) ([]financeChannelAggregate, error) {
	channelSnapshots, err := loadLegacyBillingChannelSnapshots(logs)
	if err != nil {
		return nil, err
	}
	aggregateMap := make(map[string]*financeChannelAggregate)
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
		provider := legacyBillingProviderSnapshot(logRecord.ChannelId, channelSnapshots)
		channelName := firstNonEmpty(channelSnapshots[logRecord.ChannelId].Name, logRecord.ChannelName)
		key := fmt.Sprintf("%d|%s|%s|%s", logRecord.ChannelId, channelName, provider, logRecord.ModelName)
		item := aggregateMap[key]
		if item == nil {
			item = &financeChannelAggregate{
				ChannelID:        logRecord.ChannelId,
				ChannelName:      channelName,
				ProviderSnapshot: provider,
				ModelName:        logRecord.ModelName,
			}
			aggregateMap[key] = item
		}
		other := parseStatementOther(logRecord.Other)
		item.PromptTokens += sign * logRecord.PromptTokens
		item.CompletionTokens += sign * logRecord.CompletionTokens
		item.TotalTokens += sign * (logRecord.PromptTokens + logRecord.CompletionTokens)
		item.RevenueUSD = roundAccountingAmount(item.RevenueUSD + float64(sign)*legacyBillingRecognizedRevenueUSD(logRecord, other))
		item.CostUSD = roundAccountingAmount(item.CostUSD + float64(sign)*legacyBillingEstimatedCostUSD(logRecord, other))
	}
	items := make([]financeChannelAggregate, 0, len(aggregateMap))
	for _, item := range aggregateMap {
		items = append(items, *item)
	}
	return items, nil
}

func financeChartPoints(values map[string]float64) []FinanceChannelCostChartPoint {
	points := make([]FinanceChannelCostChartPoint, 0, len(values))
	for name, value := range values {
		points = append(points, FinanceChannelCostChartPoint{
			Name:  firstNonEmpty(name, "-"),
			Value: roundAccountingAmount(value),
		})
	}
	sort.Slice(points, func(i, j int) bool {
		return points[i].Value > points[j].Value
	})
	if len(points) > 10 {
		points = points[:10]
	}
	return points
}

func loadFinanceChannelItems(periodRange financePeriodRange) ([]financeChannelAggregate, error) {
	var ledgers []ChannelCostLedger
	if err := DB.Where("occurred_at >= ? AND occurred_at <= ?", periodRange.Start, periodRange.End).Find(&ledgers).Error; err != nil {
		return nil, err
	}
	if len(ledgers) > 0 {
		return financeBuildChannelAggregatesFromLedgers(ledgers), nil
	}
	logs, err := loadLegacyBillingLogs(periodRange.Start, periodRange.End)
	if err != nil {
		return nil, err
	}
	return financeBuildChannelAggregatesFromLogs(logs)
}

func BuildFinanceDashboardSummary(periodType string, period string) (*FinanceDashboardSummary, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}

	var fundings []UserQuotaFunding
	if err := DB.Where("created_at >= ? AND created_at <= ?", periodRange.Start, periodRange.End).
		Order("created_at asc, id asc").
		Find(&fundings).Error; err != nil {
		return nil, err
	}
	var previousFundings []UserQuotaFunding
	if err := DB.Where("created_at >= ? AND created_at <= ?", periodRange.PreviousStart, periodRange.PreviousEnd).
		Find(&previousFundings).Error; err != nil {
		return nil, err
	}

	var ledgers []ChannelCostLedger
	if err := DB.Where("occurred_at >= ? AND occurred_at <= ?", periodRange.Start, periodRange.End).
		Order("occurred_at asc, id asc").
		Find(&ledgers).Error; err != nil {
		return nil, err
	}
	var previousLedgers []ChannelCostLedger
	if err := DB.Where("occurred_at >= ? AND occurred_at <= ?", periodRange.PreviousStart, periodRange.PreviousEnd).
		Find(&previousLedgers).Error; err != nil {
		return nil, err
	}

	activeUsage := financeBuildUserUsageFromLedgers(ledgers)
	if len(ledgers) == 0 {
		legacyLogs, err := loadLegacyBillingLogs(periodRange.Start, periodRange.End)
		if err != nil {
			return nil, err
		}
		activeUsage = financeBuildUserUsageFromLogs(legacyLogs)
	}
	previousUsage := financeBuildUserUsageFromLedgers(previousLedgers)
	if len(previousLedgers) == 0 {
		legacyLogs, err := loadLegacyBillingLogs(periodRange.PreviousStart, periodRange.PreviousEnd)
		if err != nil {
			return nil, err
		}
		previousUsage = financeBuildUserUsageFromLogs(legacyLogs)
	}

	var allUsers []User
	if err := DB.Unscoped().Where("deleted_at IS NULL").Find(&allUsers).Error; err != nil {
		return nil, err
	}

	summary := &FinanceDashboardSummary{
		PeriodType: periodRange.PeriodType,
		Period:     periodRange.Period,
	}

	giftGrantedQuota := 0
	giftGrantedUSD := 0.0
	paidIncomeUSD := 0.0
	paidIncomePlatform := 0
	paidUserSet := make(map[int]struct{})
	for _, funding := range fundings {
		if funding.FundingType == QuotaFundingTypePaid {
			paidIncomeUSD = roundAccountingAmount(paidIncomeUSD + funding.RecognizedRevenueUSDTotal)
			paidIncomePlatform += funding.GrantedQuota
			paidUserSet[funding.UserId] = struct{}{}
			continue
		}
		giftGrantedQuota += funding.GrantedQuota
		giftGrantedUSD = roundAccountingAmount(giftGrantedUSD + quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding)))
	}

	channelCostUSD := 0.0
	channelCostPlatform := 0
	for _, usage := range activeUsage {
		channelCostUSD = roundAccountingAmount(channelCostUSD + usage.ConsumeUSD)
		channelCostPlatform += usage.ConsumePlatform
	}

	currentPaid := 0
	currentGift := 0
	lowBalanceUsers := 0
	inactiveUsers := 0
	activeUserSet := make(map[int]struct{}, len(activeUsage))
	for userID := range activeUsage {
		activeUserSet[userID] = struct{}{}
	}
	lowBalanceThreshold := int(common.QuotaPerUnit * 5)
	for _, user := range allUsers {
		currentPaid += user.PaidQuota
		currentGift += user.GiftQuota
		if user.Quota > 0 && user.Quota <= lowBalanceThreshold {
			lowBalanceUsers++
		}
		if user.Quota > 0 {
			if _, ok := activeUserSet[user.Id]; !ok {
				inactiveUsers++
			}
		}
	}

	summary.KPIs.SalesIncome = financeAmountFromUSDPlatform(paidIncomeUSD, paidIncomePlatform)
	summary.KPIs.GiftGranted = financeAmountFromUSDPlatform(giftGrantedUSD, giftGrantedQuota)
	summary.KPIs.ChannelCost = financeAmountFromUSDPlatform(channelCostUSD, channelCostPlatform)
	summary.KPIs.CurrentPaidBalance = financeAmountFromQuota(currentPaid, common.QuotaPerUnit)
	summary.KPIs.CurrentGiftBalance = financeAmountFromQuota(currentGift, common.QuotaPerUnit)
	summary.KPIs.ActiveCustomerCount = len(activeUsage)
	summary.KPIs.PaidCustomerCount = len(paidUserSet)

	prevGiftUSD := 0.0
	refundCompensationUSD := 0.0
	prevRefundCompensationUSD := 0.0
	for _, funding := range previousFundings {
		if funding.FundingType == QuotaFundingTypeGift {
			prevGiftUSD = roundAccountingAmount(prevGiftUSD + quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding)))
		}
		if funding.SourceType == QuotaFundingSourceCompensation {
			prevRefundCompensationUSD = roundAccountingAmount(prevRefundCompensationUSD + quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding)))
		}
	}
	for _, funding := range fundings {
		if funding.SourceType == QuotaFundingSourceCompensation {
			refundCompensationUSD = roundAccountingAmount(refundCompensationUSD + quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding)))
		}
	}
	summary.Alerts = []FinanceAlertItem{
		{Type: "gift_spike", Title: "赠送异常上升", AbnormalValue: giftGrantedUSD, BaselineValue: prevGiftUSD, SuggestedAction: "检查活动赠送、管理员调账与补偿发放", LastOccurredAt: periodRange.End},
		{Type: "zero_usage_users", Title: "零调用用户数", AbnormalValue: float64(inactiveUsers), BaselineValue: float64(len(previousUsage)), SuggestedAction: "筛查长期未调用但仍持有余额的客户", LastOccurredAt: periodRange.End},
		{Type: "refund_compensation", Title: "退款/补偿异常", AbnormalValue: refundCompensationUSD, BaselineValue: prevRefundCompensationUSD, SuggestedAction: "核对补偿原因与审批记录", LastOccurredAt: periodRange.End},
		{Type: "balance_alert_users", Title: "余额告警客户数", AbnormalValue: float64(lowBalanceUsers), BaselineValue: 0, SuggestedAction: "关注低余额客户，及时提醒充值或续费", LastOccurredAt: periodRange.End},
	}

	return summary, nil
}

func ListFinanceDashboardTodos(periodType string, period string, limit int) ([]FinanceTodoItem, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}
	var users []User
	if err := DB.Unscoped().Where("deleted_at IS NULL").Find(&users).Error; err != nil {
		return nil, err
	}
	var ledgers []ChannelCostLedger
	if err := DB.Where("occurred_at >= ? AND occurred_at <= ?", periodRange.Start, periodRange.End).Find(&ledgers).Error; err != nil {
		return nil, err
	}
	usageMap := financeBuildUserUsageFromLedgers(ledgers)
	activeSet := make(map[int]struct{}, len(usageMap))
	for userID := range usageMap {
		activeSet[userID] = struct{}{}
	}
	items := make([]FinanceTodoItem, 0, limit*2)
	lowBalanceThreshold := int(common.QuotaPerUnit * 5)
	for _, user := range users {
		if user.Quota > 0 && user.Quota <= lowBalanceThreshold {
			items = append(items, FinanceTodoItem{
				Type:            "balance_alert",
				TargetType:      "user",
				TargetID:        user.Id,
				TargetName:      user.Username,
				AbnormalValue:   fmt.Sprintf("$%.2f", quotaToUSDWithSnapshot(user.Quota, common.QuotaPerUnit)),
				SuggestedAction: "提醒客户充值或补充付费额度",
				LastOccurredAt:  periodRange.End,
			})
		}
		if user.Quota > 0 {
			if _, ok := activeSet[user.Id]; !ok {
				items = append(items, FinanceTodoItem{
					Type:            "inactive_balance",
					TargetType:      "user",
					TargetID:        user.Id,
					TargetName:      user.Username,
					AbnormalValue:   fmt.Sprintf("COS币余额 %.2f", financeCOSAmountFromUSD(quotaToUSDWithSnapshot(user.Quota, common.QuotaPerUnit))),
					SuggestedAction: "检查是否需要唤醒或回访",
					LastOccurredAt:  periodRange.End,
				})
			}
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Type != items[j].Type {
			return items[i].Type < items[j].Type
		}
		return items[i].TargetID < items[j].TargetID
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func ListFinanceDashboardRankings(periodType string, period string, view string, limit int) ([]FinanceRankingItem, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}

	switch strings.TrimSpace(view) {
	case FinanceRankingViewIncome:
		var fundings []UserQuotaFunding
		if err := DB.Where("funding_type = ? AND created_at >= ? AND created_at <= ?", QuotaFundingTypePaid, periodRange.Start, periodRange.End).Find(&fundings).Error; err != nil {
			return nil, err
		}
		type incomeAgg struct {
			UserID   int
			USD      float64
			Platform int
		}
		aggMap := make(map[int]*incomeAgg)
		for _, funding := range fundings {
			item := aggMap[funding.UserId]
			if item == nil {
				item = &incomeAgg{UserID: funding.UserId}
				aggMap[funding.UserId] = item
			}
			item.USD = roundAccountingAmount(item.USD + funding.RecognizedRevenueUSDTotal)
			item.Platform += funding.GrantedQuota
		}
		userIDs := make([]int, 0, len(aggMap))
		for userID := range aggMap {
			userIDs = append(userIDs, userID)
		}
		userMap, err := financeUserMap(userIDs)
		if err != nil {
			return nil, err
		}
		items := make([]FinanceRankingItem, 0, len(aggMap))
		for _, agg := range aggMap {
			items = append(items, FinanceRankingItem{
				TargetType:    "user",
				TargetID:      agg.UserID,
				TargetName:    firstNonEmpty(userMap[agg.UserID].Username, fmt.Sprintf("user-%d", agg.UserID)),
				ValueUSD:      agg.USD,
				ValueCOS:      financeCOSAmountFromUSD(agg.USD),
				ValuePlatform: agg.Platform,
			})
		}
		sort.Slice(items, func(i, j int) bool { return items[i].ValueUSD > items[j].ValueUSD })
		if len(items) > limit {
			items = items[:limit]
		}
		for i := range items {
			items[i].Rank = i + 1
		}
		return items, nil
	case FinanceRankingViewChannel:
		channelItems, err := loadFinanceChannelItems(periodRange)
		if err != nil {
			return nil, err
		}
		channelMap := make(map[int]*FinanceRankingItem)
		for _, item := range channelItems {
			rankItem := channelMap[item.ChannelID]
			if rankItem == nil {
				rankItem = &FinanceRankingItem{
					TargetType: "channel",
					TargetID:   item.ChannelID,
					TargetName: item.ChannelName,
					Extra:      item.ProviderSnapshot,
				}
				channelMap[item.ChannelID] = rankItem
			}
			rankItem.ValueUSD = roundAccountingAmount(rankItem.ValueUSD + item.CostUSD)
			rankItem.ValueTokens += item.TotalTokens
		}
		items := make([]FinanceRankingItem, 0, len(channelMap))
		for _, item := range channelMap {
			items = append(items, *item)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].ValueUSD > items[j].ValueUSD })
		if len(items) > limit {
			items = items[:limit]
		}
		for i := range items {
			items[i].Rank = i + 1
		}
		return items, nil
	default:
		var ledgers []ChannelCostLedger
		if err := DB.Where("occurred_at >= ? AND occurred_at <= ?", periodRange.Start, periodRange.End).Find(&ledgers).Error; err != nil {
			return nil, err
		}
		usageMap := financeBuildUserUsageFromLedgers(ledgers)
		if len(ledgers) == 0 {
			logs, err := loadLegacyBillingLogs(periodRange.Start, periodRange.End)
			if err != nil {
				return nil, err
			}
			usageMap = financeBuildUserUsageFromLogs(logs)
		}
		userIDs := make([]int, 0, len(usageMap))
		for userID := range usageMap {
			userIDs = append(userIDs, userID)
		}
		userMap, err := financeUserMap(userIDs)
		if err != nil {
			return nil, err
		}
		items := make([]FinanceRankingItem, 0, len(usageMap))
		for _, usage := range usageMap {
			item := FinanceRankingItem{
				TargetType:    "user",
				TargetID:      usage.UserID,
				TargetName:    firstNonEmpty(userMap[usage.UserID].Username, fmt.Sprintf("user-%d", usage.UserID)),
				ValueUSD:      usage.ConsumeUSD,
				ValueCOS:      financeCOSAmountFromUSD(usage.ConsumeUSD),
				ValuePlatform: usage.ConsumePlatform,
				ValueTokens:   usage.ConsumePlatform,
			}
			if strings.TrimSpace(view) == FinanceRankingViewPaidUsage {
				item.ValueUSD = usage.PaidConsumeUSD
				item.ValueCOS = financeCOSAmountFromUSD(usage.PaidConsumeUSD)
				item.ValuePlatform = usage.PaidConsumePlatform
				item.ValueTokens = usage.PaidConsumePlatform
			}
			items = append(items, item)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].ValueUSD > items[j].ValueUSD })
		if len(items) > limit {
			items = items[:limit]
		}
		for i := range items {
			items[i].Rank = i + 1
		}
		return items, nil
	}
}

func BuildFinanceRevenueSummary(periodType string, period string) (*FinanceRevenueSummary, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}
	var fundings []UserQuotaFunding
	if err := DB.Where("created_at >= ? AND created_at <= ?", periodRange.Start, periodRange.End).Find(&fundings).Error; err != nil {
		return nil, err
	}
	summary := &FinanceRevenueSummary{
		PeriodType: periodRange.PeriodType,
		Period:     periodRange.Period,
	}
	paidQuota := 0
	paidUSD := 0.0
	giftQuota := 0
	giftUSD := 0.0
	for _, funding := range fundings {
		if funding.FundingType == QuotaFundingTypePaid {
			paidQuota += funding.GrantedQuota
			paidUSD = roundAccountingAmount(paidUSD + funding.RecognizedRevenueUSDTotal)
			continue
		}
		giftQuota += funding.GrantedQuota
		giftUSD = roundAccountingAmount(giftUSD + quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding)))
	}
	summary.PaidRechargeTotal = financeAmountFromUSDPlatform(paidUSD, paidQuota)
	summary.GiftGrantedTotal = financeAmountFromUSDPlatform(giftUSD, giftQuota)

	var redemptions []Redemption
	now := common.GetTimestamp()
	if err := DB.Where("status = ? AND (expired_time = 0 OR expired_time >= ?)", common.RedemptionCodeStatusEnabled, now).Find(&redemptions).Error; err != nil {
		return nil, err
	}
	for _, redemption := range redemptions {
		if redemption.FundingType == QuotaFundingTypePaid {
			summary.UnissuedPaidVoucher.COS = roundAccountingAmount(summary.UnissuedPaidVoucher.COS + financeCOSAmountFromUSD(redemption.AmountUSD))
			summary.UnissuedPaidVoucher.Platform += redemption.Quota
			summary.UnissuedPaidVoucher.USD = roundAccountingAmount(summary.UnissuedPaidVoucher.USD + redemption.AmountUSD)
			continue
		}
		summary.UnredeemedGiftVoucher.COS = roundAccountingAmount(summary.UnredeemedGiftVoucher.COS + financeCOSAmountFromUSD(redemption.AmountUSD))
		summary.UnredeemedGiftVoucher.Platform += redemption.Quota
		summary.UnredeemedGiftVoucher.USD = roundAccountingAmount(summary.UnredeemedGiftVoucher.USD + redemption.AmountUSD)
	}
	return summary, nil
}

func ListFinancePaidSourceSummary(periodType string, period string, pageInfo *common.PageInfo) (*common.PageInfo, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}
	var fundings []UserQuotaFunding
	if err := DB.Where("funding_type = ? AND created_at >= ? AND created_at <= ?", QuotaFundingTypePaid, periodRange.Start, periodRange.End).Find(&fundings).Error; err != nil {
		return nil, err
	}
	type agg struct {
		Users map[int]struct{}
		FinancePaidSourceSummaryItem
	}
	aggMap := make(map[string]*agg)
	for _, funding := range fundings {
		item := aggMap[funding.SourceType]
		if item == nil {
			item = &agg{
				Users: make(map[int]struct{}),
				FinancePaidSourceSummaryItem: FinancePaidSourceSummaryItem{
					SourceType:  funding.SourceType,
					SourceLabel: financeSourceLabel(funding.SourceType),
				},
			}
			aggMap[funding.SourceType] = item
		}
		item.Users[funding.UserId] = struct{}{}
		item.RechargeCount++
		item.RechargeAmountPlatform += funding.GrantedQuota
		item.RechargeAmountUSD = roundAccountingAmount(item.RechargeAmountUSD + funding.RecognizedRevenueUSDTotal)
		item.RechargeAmountCOS = roundAccountingAmount(item.RechargeAmountCOS + financeCOSAmountFromUSD(funding.RecognizedRevenueUSDTotal))
		item.RemainingAmountPlatform += funding.RemainingQuota
		remainingUSD := roundAccountingAmount(quotaToUSDWithSnapshot(funding.RemainingQuota, fundingQuotaSnapshot(funding)))
		item.RemainingAmountUSD = roundAccountingAmount(item.RemainingAmountUSD + remainingUSD)
		item.RemainingAmountCOS = roundAccountingAmount(item.RemainingAmountCOS + financeCOSAmountFromUSD(remainingUSD))
	}
	items := make([]FinancePaidSourceSummaryItem, 0, len(aggMap))
	for _, item := range aggMap {
		item.UserCount = len(item.Users)
		items = append(items, item.FinancePaidSourceSummaryItem)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RechargeAmountUSD > items[j].RechargeAmountUSD })
	pageInfo.SetTotal(len(items))
	pageInfo.SetItems(financePageItems(items, pageInfo))
	return pageInfo, nil
}

func ListFinancePaidSourceDetails(periodType string, period string, pageInfo *common.PageInfo) (*common.PageInfo, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}
	var fundings []UserQuotaFunding
	if err := DB.Where("funding_type = ? AND created_at >= ? AND created_at <= ?", QuotaFundingTypePaid, periodRange.Start, periodRange.End).
		Order("created_at desc, id desc").
		Find(&fundings).Error; err != nil {
		return nil, err
	}
	userIDs := make([]int, 0, len(fundings))
	userSeen := make(map[int]struct{})
	for _, funding := range fundings {
		if _, ok := userSeen[funding.UserId]; ok {
			continue
		}
		userSeen[funding.UserId] = struct{}{}
		userIDs = append(userIDs, funding.UserId)
	}
	userMap, err := financeUserMap(userIDs)
	if err != nil {
		return nil, err
	}
	items := make([]FinancePaidSourceDetailItem, 0, len(fundings))
	for _, funding := range fundings {
		items = append(items, FinancePaidSourceDetailItem{
			CreatedAt:               funding.CreatedAt,
			SourceType:              funding.SourceType,
			SourceLabel:             financeSourceLabel(funding.SourceType),
			UserID:                  funding.UserId,
			Username:                firstNonEmpty(userMap[funding.UserId].Username, fmt.Sprintf("user-%d", funding.UserId)),
			RechargeAmountUSD:       roundAccountingAmount(funding.RecognizedRevenueUSDTotal),
			RechargeAmountCOS:       financeCOSAmountFromUSD(funding.RecognizedRevenueUSDTotal),
			RechargeAmountPlatform:  funding.GrantedQuota,
			RemainingAmountUSD:      roundAccountingAmount(quotaToUSDWithSnapshot(funding.RemainingQuota, fundingQuotaSnapshot(funding))),
			RemainingAmountCOS:      financeCOSAmountFromUSD(quotaToUSDWithSnapshot(funding.RemainingQuota, fundingQuotaSnapshot(funding))),
			RemainingAmountPlatform: funding.RemainingQuota,
			SourceRefID:             funding.SourceRefId,
			SourceName:              funding.SourceName,
			Remark:                  funding.Remark,
		})
	}
	pageInfo.SetTotal(len(items))
	pageInfo.SetItems(financePageItems(items, pageInfo))
	return pageInfo, nil
}

func ListFinanceGiftAuditSummary(periodType string, period string, sourceType string, userID int, pageInfo *common.PageInfo) (*common.PageInfo, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}
	query := DB.Where("funding_type = ? AND created_at >= ? AND created_at <= ?", QuotaFundingTypeGift, periodRange.Start, periodRange.End)
	if sourceType = strings.TrimSpace(sourceType); sourceType != "" {
		query = query.Where("source_type = ?", sourceType)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	var fundings []UserQuotaFunding
	if err := query.Order("created_at desc, id desc").Find(&fundings).Error; err != nil {
		return nil, err
	}

	type agg struct {
		FinanceGiftAuditSummaryItem
	}
	aggMap := make(map[string]*agg)
	for _, funding := range fundings {
		activityDescription := financeGiftAuditActivityDescription(funding)
		key := fmt.Sprintf("%s|%s", funding.SourceType, activityDescription)
		item := aggMap[key]
		if item == nil {
			item = &agg{
				FinanceGiftAuditSummaryItem: FinanceGiftAuditSummaryItem{
					SourceType:          funding.SourceType,
					SourceLabel:         financeSourceLabel(funding.SourceType),
					ActivityDescription: activityDescription,
				},
			}
			aggMap[key] = item
		}
		item.ExchangeCount++
		item.GrantedAmountPlatform += funding.GrantedQuota
		grantedUSD := roundAccountingAmount(quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding)))
		item.GrantedAmountUSD = roundAccountingAmount(item.GrantedAmountUSD + grantedUSD)
		item.GrantedAmountCOS = roundAccountingAmount(item.GrantedAmountCOS + financeCOSAmountFromUSD(grantedUSD))
	}

	items := make([]FinanceGiftAuditSummaryItem, 0, len(aggMap))
	for _, item := range aggMap {
		items = append(items, item.FinanceGiftAuditSummaryItem)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].GrantedAmountUSD != items[j].GrantedAmountUSD {
			return items[i].GrantedAmountUSD > items[j].GrantedAmountUSD
		}
		if items[i].ExchangeCount != items[j].ExchangeCount {
			return items[i].ExchangeCount > items[j].ExchangeCount
		}
		if items[i].SourceLabel != items[j].SourceLabel {
			return items[i].SourceLabel < items[j].SourceLabel
		}
		return items[i].ActivityDescription < items[j].ActivityDescription
	})
	pageInfo.SetTotal(len(items))
	pageInfo.SetItems(financePageItems(items, pageInfo))
	return pageInfo, nil
}

func ListFinanceGiftAudit(periodType string, period string, sourceType string, userID int, pageInfo *common.PageInfo) (*common.PageInfo, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}
	query := DB.Where("funding_type = ? AND created_at >= ? AND created_at <= ?", QuotaFundingTypeGift, periodRange.Start, periodRange.End)
	if sourceType = strings.TrimSpace(sourceType); sourceType != "" {
		query = query.Where("source_type = ?", sourceType)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	var fundings []UserQuotaFunding
	if err := query.Order("created_at desc, id desc").Find(&fundings).Error; err != nil {
		return nil, err
	}
	userIDs := make([]int, 0, len(fundings))
	operatorIDs := make([]int, 0)
	userSeen := make(map[int]struct{})
	operatorSeen := make(map[int]struct{})
	for _, funding := range fundings {
		if _, ok := userSeen[funding.UserId]; !ok {
			userSeen[funding.UserId] = struct{}{}
			userIDs = append(userIDs, funding.UserId)
		}
		if funding.OperatorUserId > 0 {
			if _, ok := operatorSeen[funding.OperatorUserId]; !ok {
				operatorSeen[funding.OperatorUserId] = struct{}{}
				operatorIDs = append(operatorIDs, funding.OperatorUserId)
			}
		}
	}
	userMap, err := financeUserMap(userIDs)
	if err != nil {
		return nil, err
	}
	operatorMap, err := financeUserMap(operatorIDs)
	if err != nil {
		return nil, err
	}
	voucherMap, err := financeVoucherCodeMap(fundings)
	if err != nil {
		return nil, err
	}
	items := make([]FinanceGiftAuditItem, 0, len(fundings))
	for _, funding := range fundings {
		operatorName := strings.TrimSpace(funding.OperatorUsernameSnapshot)
		if operatorName == "" && funding.OperatorUserId > 0 {
			operatorName = operatorMap[funding.OperatorUserId].Username
		}
		items = append(items, FinanceGiftAuditItem{
			CreatedAt:             funding.CreatedAt,
			UserID:                funding.UserId,
			Username:              firstNonEmpty(userMap[funding.UserId].Username, fmt.Sprintf("user-%d", funding.UserId)),
			SourceType:            funding.SourceType,
			SourceLabel:           financeSourceLabel(funding.SourceType),
			GrantedAmountUSD:      roundAccountingAmount(quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding))),
			GrantedAmountCOS:      financeCOSAmountFromUSD(quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding))),
			GrantedAmountPlatform: funding.GrantedQuota,
			OperatorUserID:        funding.OperatorUserId,
			OperatorUsername:      operatorName,
			VoucherCode:           voucherMap[funding.SourceRefId],
			Remark:                funding.Remark,
		})
	}
	pageInfo.SetTotal(len(items))
	pageInfo.SetItems(financePageItems(items, pageInfo))
	return pageInfo, nil
}

func BuildFinanceChannelCostSummary(periodType string, period string) (*FinanceChannelCostSummary, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}
	channelItems, err := loadFinanceChannelItems(periodRange)
	if err != nil {
		return nil, err
	}
	summary := &FinanceChannelCostSummary{
		PeriodType: periodRange.PeriodType,
		Period:     periodRange.Period,
	}
	byChannelUSD := make(map[string]float64)
	byChannelTokens := make(map[string]float64)
	byModelUSD := make(map[string]float64)
	byModelTokens := make(map[string]float64)
	for _, item := range channelItems {
		summary.KPIs.TotalTokens += item.TotalTokens
		summary.KPIs.TotalCostUSD = roundAccountingAmount(summary.KPIs.TotalCostUSD + item.CostUSD)
		byChannelUSD[item.ChannelName] += item.CostUSD
		byChannelTokens[item.ChannelName] += float64(item.TotalTokens)
		byModelUSD[item.ModelName] += item.CostUSD
		byModelTokens[item.ModelName] += float64(item.TotalTokens)
	}
	summary.Charts.ByChannelUSD = financeChartPoints(byChannelUSD)
	summary.Charts.ByChannelTokens = financeChartPoints(byChannelTokens)
	summary.Charts.ByModelUSD = financeChartPoints(byModelUSD)
	summary.Charts.ByModelTokens = financeChartPoints(byModelTokens)
	return summary, nil
}

func ListFinanceChannelCostChannels(periodType string, period string, pageInfo *common.PageInfo) (*common.PageInfo, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}
	channelItems, err := loadFinanceChannelItems(periodRange)
	if err != nil {
		return nil, err
	}
	aggMap := make(map[int]*FinanceChannelCostChannelItem)
	for _, item := range channelItems {
		row := aggMap[item.ChannelID]
		if row == nil {
			row = &FinanceChannelCostChannelItem{
				ChannelID:        item.ChannelID,
				ChannelName:      item.ChannelName,
				ProviderSnapshot: item.ProviderSnapshot,
			}
			aggMap[item.ChannelID] = row
		}
		row.PromptTokens += item.PromptTokens
		row.CompletionTokens += item.CompletionTokens
		row.CacheTokens += item.CacheTokens
		row.TotalTokens += item.TotalTokens
		row.RevenueUSD = roundAccountingAmount(row.RevenueUSD + item.RevenueUSD)
		row.CostUSD = roundAccountingAmount(row.CostUSD + item.CostUSD)
	}
	items := make([]FinanceChannelCostChannelItem, 0, len(aggMap))
	for _, item := range aggMap {
		items = append(items, *item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CostUSD > items[j].CostUSD })
	pageInfo.SetTotal(len(items))
	pageInfo.SetItems(financePageItems(items, pageInfo))
	return pageInfo, nil
}

func ListFinanceChannelCostModels(periodType string, period string, pageInfo *common.PageInfo) (*common.PageInfo, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}
	channelItems, err := loadFinanceChannelItems(periodRange)
	if err != nil {
		return nil, err
	}
	items := make([]FinanceChannelCostModelItem, 0, len(channelItems))
	for _, item := range channelItems {
		items = append(items, FinanceChannelCostModelItem{
			ChannelID:        item.ChannelID,
			ChannelName:      item.ChannelName,
			ProviderSnapshot: item.ProviderSnapshot,
			ModelName:        item.ModelName,
			PromptTokens:     item.PromptTokens,
			CompletionTokens: item.CompletionTokens,
			CacheTokens:      item.CacheTokens,
			TotalTokens:      item.TotalTokens,
			RevenueUSD:       item.RevenueUSD,
			CostUSD:          item.CostUSD,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CostUSD > items[j].CostUSD })
	pageInfo.SetTotal(len(items))
	pageInfo.SetItems(financePageItems(items, pageInfo))
	return pageInfo, nil
}

func ListFinanceCustomerBillSummary(billMonth string, userKeyword string, pageInfo *common.PageInfo) (*common.PageInfo, error) {
	normalizedMonth, start, end, err := ParseCustomerStatementBillMonth(billMonth)
	if err != nil {
		return nil, err
	}
	matchedUserIDs, err := financeMatchUserIDs(userKeyword)
	if err != nil {
		return nil, err
	}
	filterSet := make(map[int]struct{})
	for _, userID := range matchedUserIDs {
		filterSet[userID] = struct{}{}
	}

	var ledgers []ChannelCostLedger
	if err := DB.Where("bill_month = ? AND occurred_at >= ? AND occurred_at <= ?", normalizedMonth, start, end).Find(&ledgers).Error; err != nil {
		return nil, err
	}
	usageMap := financeBuildUserUsageFromLedgers(ledgers)
	if len(ledgers) == 0 {
		logs, err := loadLegacyBillingLogs(start, end)
		if err != nil {
			return nil, err
		}
		usageMap = financeBuildUserUsageFromLogs(logs)
	}

	userIDs := make([]int, 0, len(usageMap)+len(matchedUserIDs))
	seen := make(map[int]struct{})
	for userID := range usageMap {
		if len(filterSet) > 0 {
			if _, ok := filterSet[userID]; !ok {
				continue
			}
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}
	for _, userID := range matchedUserIDs {
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}

	userMap, err := financeUserMap(userIDs)
	if err != nil {
		return nil, err
	}
	items := make([]FinanceCustomerBillSummaryItem, 0, len(userIDs))
	for _, userID := range userIDs {
		user := userMap[userID]
		usage := usageMap[userID]
		item := FinanceCustomerBillSummaryItem{
			UserID:                 userID,
			Username:               firstNonEmpty(user.Username, fmt.Sprintf("user-%d", userID)),
			CurrentBalancePlatform: user.Quota,
			CurrentBalanceUSD:      roundAccountingAmount(quotaToUSDWithSnapshot(user.Quota, common.QuotaPerUnit)),
			CurrentBalanceCOS:      financeCOSAmountFromUSD(quotaToUSDWithSnapshot(user.Quota, common.QuotaPerUnit)),
		}
		if usage != nil {
			item.TotalConsumePlatform = usage.ConsumePlatform
			item.TotalConsumeUSD = usage.ConsumeUSD
			item.TotalConsumeCOS = financeCOSAmountFromUSD(usage.ConsumeUSD)
			item.PaidConsumePlatform = usage.PaidConsumePlatform
			item.PaidConsumeUSD = usage.PaidConsumeUSD
			item.PaidConsumeCOS = financeCOSAmountFromUSD(usage.PaidConsumeUSD)
			item.GiftConsumePlatform = usage.GiftConsumePlatform
			item.GiftConsumeUSD = usage.GiftConsumeUSD
			item.GiftConsumeCOS = financeCOSAmountFromUSD(usage.GiftConsumeUSD)
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].TotalConsumeUSD > items[j].TotalConsumeUSD })
	pageInfo.SetTotal(len(items))
	pageInfo.SetItems(financePageItems(items, pageInfo))
	return pageInfo, nil
}

func GetFinanceCustomerBillDetails(billMonth string, userID int) ([]FinanceCustomerBillDetailItem, error) {
	normalizedMonth, start, end, err := ParseCustomerStatementBillMonth(billMonth)
	if err != nil {
		return nil, err
	}
	var fundings []UserQuotaFunding
	if err := DB.Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, start, end).
		Order("created_at desc, id desc").
		Find(&fundings).Error; err != nil {
		return nil, err
	}

	items := make([]FinanceCustomerBillDetailItem, 0, len(fundings))
	for _, funding := range fundings {
		entryType := "赠送"
		if funding.FundingType == QuotaFundingTypePaid {
			entryType = "充值"
		}
		remarkParts := []string{financeSourceLabel(funding.SourceType)}
		if strings.TrimSpace(funding.SourceName) != "" {
			remarkParts = append(remarkParts, funding.SourceName)
		}
		if strings.TrimSpace(funding.Remark) != "" {
			remarkParts = append(remarkParts, funding.Remark)
		}
		items = append(items, FinanceCustomerBillDetailItem{
			OccurredAt:     funding.CreatedAt,
			TokenDisplay:   "-",
			EntryType:      entryType,
			ModelName:      "",
			AmountCOS:      financeCOSAmountFromUSD(quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding))),
			AmountPlatform: funding.GrantedQuota,
			AmountUSD:      roundAccountingAmount(quotaToUSDWithSnapshot(funding.GrantedQuota, fundingQuotaSnapshot(funding))),
			ChannelName:    "",
			RequestID:      "",
			Remark:         strings.Join(remarkParts, " | "),
		})
	}

	var ledgers []ChannelCostLedger
	if err := DB.Where("user_id = ? AND bill_month = ? AND occurred_at >= ? AND occurred_at <= ?", userID, normalizedMonth, start, end).
		Order("occurred_at desc, id desc").
		Find(&ledgers).Error; err != nil {
		return nil, err
	}
	if len(ledgers) > 0 {
		logIDs := make([]int, 0)
		logIDSeen := make(map[int]struct{})
		for _, ledger := range ledgers {
			if ledger.LogId <= 0 {
				continue
			}
			if _, ok := logIDSeen[ledger.LogId]; ok {
				continue
			}
			logIDSeen[ledger.LogId] = struct{}{}
			logIDs = append(logIDs, ledger.LogId)
		}
		logMap := make(map[int]Log)
		tokenIDs := make([]int, 0)
		tokenSeen := make(map[int]struct{})
		if len(logIDs) > 0 {
			var logs []Log
			if err := LOG_DB.Where("id IN ?", logIDs).Find(&logs).Error; err != nil {
				return nil, err
			}
			for _, logRecord := range logs {
				logMap[logRecord.Id] = logRecord
				if logRecord.TokenId > 0 {
					if _, ok := tokenSeen[logRecord.TokenId]; !ok {
						tokenSeen[logRecord.TokenId] = struct{}{}
						tokenIDs = append(tokenIDs, logRecord.TokenId)
					}
				}
			}
		}
		tokenMap := make(map[int]customerStatementTokenSnapshot)
		if len(tokenIDs) > 0 {
			var tokens []customerStatementTokenSnapshot
			if err := DB.Unscoped().Table("tokens").Select("id, name, key").Where("id IN ?", tokenIDs).Find(&tokens).Error; err != nil {
				return nil, err
			}
			for _, token := range tokens {
				tokenMap[token.Id] = token
			}
		}
		for _, ledger := range ledgers {
			sign := -1
			entryType := "消费"
			if ledger.EntryType == ChannelCostEntryTypeRefund {
				sign = 1
				entryType = "退款"
			}
			logRecord, hasLog := logMap[ledger.LogId]
			tokenDisplay := "-"
			if hasLog {
				tokenSnapshot := tokenMap[logRecord.TokenId]
				tokenDisplay = fmt.Sprintf("%s / %s",
					firstNonEmpty(logRecord.TokenName, tokenSnapshot.Name, "-"),
					MaskTokenKey(tokenSnapshot.Key),
				)
			}
			items = append(items, FinanceCustomerBillDetailItem{
				OccurredAt:     ledger.OccurredAt,
				TokenDisplay:   tokenDisplay,
				EntryType:      entryType,
				ModelName:      ledger.OriginModelName,
				AmountCOS:      financeCOSAmountFromUSD(float64(sign) * ledger.InternalEquivalentUSD),
				AmountPlatform: sign * ledger.ActualQuota,
				AmountUSD:      roundAccountingAmount(float64(sign) * ledger.InternalEquivalentUSD),
				ChannelName:    ledger.ChannelNameSnapshot,
				RequestID:      ledger.RequestId,
				Remark:         "",
			})
		}
	} else {
		logs, err := loadLegacyBillingLogs(start, end)
		if err != nil {
			return nil, err
		}
		userLogs := make([]*Log, 0)
		for _, logRecord := range logs {
			if logRecord != nil && logRecord.UserId == userID {
				userLogs = append(userLogs, logRecord)
			}
		}
		tokenSnapshots, err := getStatementTokenSnapshots(userLogs)
		if err != nil {
			return nil, err
		}
		channelSnapshots, err := getStatementChannelSnapshots(userLogs)
		if err != nil {
			return nil, err
		}
		for _, logRecord := range userLogs {
			entryType := convertLogTypeToStatementEntryType(logRecord.Type)
			if entryType == "" {
				continue
			}
			sign := -1
			label := "消费"
			if entryType == CustomerMonthlyStatementEntryTypeRefund {
				sign = 1
				label = "退款"
			}
			tokenSnapshot := tokenSnapshots[logRecord.TokenId]
			items = append(items, FinanceCustomerBillDetailItem{
				OccurredAt:     logRecord.CreatedAt,
				TokenDisplay:   fmt.Sprintf("%s / %s", firstNonEmpty(logRecord.TokenName, tokenSnapshot.Name, "-"), MaskTokenKey(tokenSnapshot.Key)),
				EntryType:      label,
				ModelName:      logRecord.ModelName,
				AmountCOS:      financeCOSAmountFromUSD(float64(sign) * quotaToUSDWithSnapshot(logRecord.Quota, common.QuotaPerUnit)),
				AmountPlatform: sign * logRecord.Quota,
				AmountUSD:      roundAccountingAmount(float64(sign) * quotaToUSDWithSnapshot(logRecord.Quota, common.QuotaPerUnit)),
				ChannelName:    channelSnapshots[logRecord.ChannelId].Name,
				RequestID:      logRecord.RequestId,
				Remark:         "",
			})
		}
	}

	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt > items[j].OccurredAt })
	return items, nil
}

func ExportFinanceCustomerBillCSV(billMonth string, userID int) ([]byte, string, error) {
	items, err := GetFinanceCustomerBillDetails(billMonth, userID)
	if err != nil {
		return nil, "", err
	}
	buffer := &bytes.Buffer{}
	buffer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(buffer)
	header := []string{"时间", "令牌", "消费类型", "模型", "COS币花费", "等价USD", "渠道", "请求ID", "说明"}
	if err := writer.Write(header); err != nil {
		return nil, "", err
	}
	for _, item := range items {
		record := []string{
			time.Unix(item.OccurredAt, 0).In(time.Local).Format("2006-01-02 15:04:05"),
			item.TokenDisplay,
			item.EntryType,
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
