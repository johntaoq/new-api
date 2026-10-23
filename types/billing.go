package types

type QuotaFundingAllocation struct {
	FundingId           int     `json:"funding_id"`
	FundingType         string  `json:"funding_type"`
	SourceType          string  `json:"source_type"`
	AllocatedQuota      int     `json:"allocated_quota"`
	AllocatedRevenueUSD float64 `json:"allocated_revenue_usd"`
}
