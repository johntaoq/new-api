package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareFinanceAggregationTest(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&UserQuotaFunding{}, &ChannelCostLedger{}, &CustomerMonthlyStatement{}, &CustomerMonthlyStatementItem{}))
	require.NoError(t, DB.Exec("DELETE FROM customer_monthly_statement_items").Error)
	require.NoError(t, DB.Exec("DELETE FROM customer_monthly_statements").Error)
	require.NoError(t, DB.Exec("DELETE FROM channel_cost_ledgers").Error)
	require.NoError(t, DB.Exec("DELETE FROM user_quota_fundings").Error)
	t.Cleanup(func() {
		DB.Exec("DELETE FROM customer_monthly_statement_items")
		DB.Exec("DELETE FROM customer_monthly_statements")
		DB.Exec("DELETE FROM channel_cost_ledgers")
		DB.Exec("DELETE FROM user_quota_fundings")
	})
}

func TestFinanceStatementUsageAggregationAndDetailPagination(t *testing.T) {
	prepareFinanceAggregationTest(t)
	statement := CustomerMonthlyStatement{
		StatementNo:      "finance-test-2026-07-42",
		BillMonth:        "2026-07",
		UserId:           42,
		UsernameSnapshot: "deleted-user",
		PeriodStart:      100,
		PeriodEnd:        400,
	}
	require.NoError(t, DB.Create(&statement).Error)
	statementItems := []CustomerMonthlyStatementItem{
		{StatementId: statement.Id, BillMonth: "2026-07", UserId: 42, EntryType: CustomerMonthlyStatementEntryTypeConsume, OccurredAt: 100, QuotaRaw: 1000, PaidQuotaRaw: 700, GiftQuotaRaw: 300, USDAmount: 1.25, QuotaPerUnitSnapshot: 500000},
		{StatementId: statement.Id, BillMonth: "2026-07", UserId: 42, EntryType: CustomerMonthlyStatementEntryTypeGift, OccurredAt: 200, QuotaRaw: 100, USDAmount: 0.2, QuotaPerUnitSnapshot: 500000},
		{StatementId: statement.Id, BillMonth: "2026-07", UserId: 42, EntryType: CustomerMonthlyStatementEntryTypeConsume, OccurredAt: 300, QuotaRaw: 500, PaidQuotaRaw: 500, USDAmount: 0.75, QuotaPerUnitSnapshot: 500000},
	}
	require.NoError(t, DB.Create(&statementItems).Error)

	expectedUsage := financeBuildUserUsageFromStatementItems(statementItems)
	actualUsage, err := loadFinanceUserUsageFromStatementItems([]int{statement.Id})
	require.NoError(t, err)
	require.Len(t, actualUsage, len(expectedUsage))
	expected := expectedUsage[42]
	actual := actualUsage[42]
	require.NotNil(t, actual)
	assert.Equal(t, expected.ConsumePlatform, actual.ConsumePlatform)
	assert.Equal(t, expected.PaidConsumePlatform, actual.PaidConsumePlatform)
	assert.Equal(t, expected.GiftConsumePlatform, actual.GiftConsumePlatform)
	assert.Equal(t, expected.ConsumeCount, actual.ConsumeCount)
	assert.Equal(t, expected.LastOccurredAt, actual.LastOccurredAt)
	assert.InDelta(t, expected.ConsumeUSD, actual.ConsumeUSD, 0.000001)
	assert.InDelta(t, expected.PaidConsumeUSD, actual.PaidConsumeUSD, 0.000001)
	assert.InDelta(t, expected.GiftConsumeUSD, actual.GiftConsumeUSD, 0.000001)

	pageInfo := &common.PageInfo{Page: 2, PageSize: 1}
	page, err := ListFinanceCustomerBillDetailsV2("2026-07", 42, pageInfo)
	require.NoError(t, err)
	assert.Equal(t, 3, page.Total)
	detailItems, ok := page.Items.([]FinanceCustomerBillDetailItem)
	require.True(t, ok)
	require.Len(t, detailItems, 1)
	assert.Equal(t, int64(200), detailItems[0].OccurredAt)
}

func financeChannelAggregateKey(item financeChannelAggregate) string {
	return fmt.Sprintf("%d|%s|%s|%s", item.ChannelID, item.ChannelName, item.ProviderSnapshot, item.ModelName)
}

func TestLoadFinanceAggregatesFromLedgers(t *testing.T) {
	prepareFinanceAggregationTest(t)
	ledgers := []ChannelCostLedger{
		{UserId: 1, ChannelId: 10, ChannelNameSnapshot: "channel-a", ProviderSnapshot: "provider-a", OriginModelName: "model-a", EntryType: ChannelCostEntryTypeConsume, PromptTokens: 100, CompletionTokens: 40, CacheTokens: 10, TotalTokens: 150, ActualQuota: 1000, PaidQuotaUsed: 700, GiftQuotaUsed: 300, InternalEquivalentUSD: 1.25, RecognizedRevenueUSD: 1.25, EstimatedCostUSD: 0.8, OccurredAt: 110},
		{UserId: 1, ChannelId: 10, ChannelNameSnapshot: "channel-a", ProviderSnapshot: "provider-a", OriginModelName: "model-a", EntryType: ChannelCostEntryTypeRefund, PromptTokens: 10, CompletionTokens: 4, CacheTokens: 1, TotalTokens: 15, ActualQuota: 100, PaidQuotaUsed: 100, InternalEquivalentUSD: 0.2, RecognizedRevenueUSD: 0.2, EstimatedCostUSD: 0.1, OccurredAt: 120},
		{UserId: 2, ChannelId: 20, ChannelNameSnapshot: "channel-b", ProviderSnapshot: "provider-b", OriginModelName: "model-b", EntryType: ChannelCostEntryTypeConsume, PromptTokens: 50, CompletionTokens: 25, TotalTokens: 75, ActualQuota: 500, GiftQuotaUsed: 500, InternalEquivalentUSD: 0.75, RecognizedRevenueUSD: 0.75, EstimatedCostUSD: 0.4, OccurredAt: 130},
	}
	require.NoError(t, DB.Create(&ledgers).Error)

	expectedUsage := financeBuildUserUsageFromLedgers(ledgers)
	actualUsage, err := loadFinanceUserUsage(100, 200)
	require.NoError(t, err)
	require.Len(t, actualUsage, len(expectedUsage))
	for userID, expected := range expectedUsage {
		actual := actualUsage[userID]
		require.NotNil(t, actual)
		assert.Equal(t, expected.ConsumePlatform, actual.ConsumePlatform)
		assert.Equal(t, expected.PaidConsumePlatform, actual.PaidConsumePlatform)
		assert.Equal(t, expected.GiftConsumePlatform, actual.GiftConsumePlatform)
		assert.Equal(t, expected.ConsumeCount, actual.ConsumeCount)
		assert.Equal(t, expected.LastOccurredAt, actual.LastOccurredAt)
		assert.InDelta(t, expected.ConsumeUSD, actual.ConsumeUSD, 0.000001)
		assert.InDelta(t, expected.PaidConsumeUSD, actual.PaidConsumeUSD, 0.000001)
		assert.InDelta(t, expected.GiftConsumeUSD, actual.GiftConsumeUSD, 0.000001)
	}

	expectedChannels := financeBuildChannelAggregatesFromLedgers(ledgers)
	actualChannels, err := loadFinanceChannelItems(financePeriodRange{Start: 100, End: 200})
	require.NoError(t, err)
	require.Len(t, actualChannels, len(expectedChannels))
	actualByKey := make(map[string]financeChannelAggregate, len(actualChannels))
	for _, item := range actualChannels {
		actualByKey[financeChannelAggregateKey(item)] = item
	}
	for _, expected := range expectedChannels {
		actual, ok := actualByKey[financeChannelAggregateKey(expected)]
		require.True(t, ok)
		assert.Equal(t, expected.PromptTokens, actual.PromptTokens)
		assert.Equal(t, expected.CompletionTokens, actual.CompletionTokens)
		assert.Equal(t, expected.CacheTokens, actual.CacheTokens)
		assert.Equal(t, expected.TotalTokens, actual.TotalTokens)
		assert.InDelta(t, expected.RevenueUSD, actual.RevenueUSD, 0.000001)
		assert.InDelta(t, expected.CostUSD, actual.CostUSD, 0.000001)
	}
}

func TestFinanceCustomerBillSummaryTotals(t *testing.T) {
	items := []FinanceCustomerBillSummaryItem{
		{CurrentBalanceUSD: 1.25, CurrentBalanceCOS: 8.75, TotalConsumeUSD: 2.5, TotalConsumeCOS: 17.5, PaidConsumeUSD: 2, PaidConsumeCOS: 14, GiftConsumeUSD: 0.5, GiftConsumeCOS: 3.5},
		{CurrentBalanceUSD: 3.75, CurrentBalanceCOS: 26.25, TotalConsumeUSD: 4.5, TotalConsumeCOS: 31.5, PaidConsumeUSD: 3, PaidConsumeCOS: 21, GiftConsumeUSD: 1.5, GiftConsumeCOS: 10.5},
	}

	totals := financeCustomerBillSummaryTotals(items)
	assert.Equal(t, 2, totals.CustomerCount)
	assert.InDelta(t, 5, totals.CurrentBalanceUSD, 0.000001)
	assert.InDelta(t, 35, totals.CurrentBalanceCOS, 0.000001)
	assert.InDelta(t, 7, totals.TotalConsumeUSD, 0.000001)
	assert.InDelta(t, 49, totals.TotalConsumeCOS, 0.000001)
	assert.InDelta(t, 5, totals.PaidConsumeUSD, 0.000001)
	assert.InDelta(t, 35, totals.PaidConsumeCOS, 0.000001)
	assert.InDelta(t, 2, totals.GiftConsumeUSD, 0.000001)
	assert.InDelta(t, 14, totals.GiftConsumeCOS, 0.000001)
}
