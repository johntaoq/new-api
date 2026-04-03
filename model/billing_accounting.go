package model

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"gorm.io/gorm"
)

const (
	QuotaFundingTypePaid = "paid"
	QuotaFundingTypeGift = "gift"

	QuotaFundingSourceOnlineTopUp   = "online_topup"
	QuotaFundingSourceStripe        = "stripe"
	QuotaFundingSourceCreem         = "creem"
	QuotaFundingSourceWaffo         = "waffo"
	QuotaFundingSourcePaidVoucher   = "paid_voucher"
	QuotaFundingSourceGiftVoucher   = "gift_voucher"
	QuotaFundingSourceInviteReward  = "invite_reward"
	QuotaFundingSourceSignupBonus   = "signup_bonus"
	QuotaFundingSourceAdminGrant    = "admin_grant"
	QuotaFundingSourcePromoCampaign = "promo_campaign"
	QuotaFundingSourceCompensation  = "compensation"
	QuotaFundingSourceSystemAdjust  = "system_adjustment"
	QuotaFundingSourceCheckin       = "checkin_bonus"
	QuotaFundingSourceLegacyBalance = "legacy_balance"
	QuotaFundingSourceLegacyGift    = "legacy_gift_balance"

	ChannelCostEntryTypeConsume = "consume"
	ChannelCostEntryTypeRefund  = "refund"

	ChannelMonthlyStatementStatusFinalized = "finalized"
)

type UserQuotaFunding struct {
	Id                        int     `json:"id"`
	UserId                    int     `json:"user_id" gorm:"index:idx_quota_funding_user_type,priority:1;index"`
	FundingType               string  `json:"funding_type" gorm:"type:varchar(16);index:idx_quota_funding_user_type,priority:2;index"`
	SourceType                string  `json:"source_type" gorm:"type:varchar(32);index"`
	BusinessType              string  `json:"business_type" gorm:"type:varchar(32);index;default:''"`
	SourceRefId               int     `json:"source_ref_id" gorm:"index"`
	SourceName                string  `json:"source_name" gorm:"type:varchar(255);default:''"`
	ExternalRef               string  `json:"external_ref" gorm:"type:varchar(128);default:''"`
	GrantedQuota              int     `json:"granted_quota"`
	RemainingQuota            int     `json:"remaining_quota" gorm:"index"`
	RecognizedRevenueUSDTotal float64 `json:"recognized_revenue_usd_total" gorm:"default:0"`
	QuotaPerUnitSnapshot      float64 `json:"quota_per_unit_snapshot" gorm:"default:0"`
	ExpiredAt                 int64   `json:"expired_at" gorm:"bigint;index"`
	OperatorUserId            int     `json:"operator_user_id" gorm:"index;default:0"`
	OperatorUsernameSnapshot  string  `json:"operator_username_snapshot" gorm:"type:varchar(64);default:''"`
	Remark                    string  `json:"remark" gorm:"type:varchar(255);default:''"`
	CreatedAt                 int64   `json:"created_at" gorm:"bigint;index"`
	UpdatedAt                 int64   `json:"updated_at" gorm:"bigint"`
}

type ChannelCostLedger struct {
	Id                    int     `json:"id"`
	BillMonth             string  `json:"bill_month" gorm:"type:varchar(7);index"`
	RequestId             string  `json:"request_id" gorm:"type:varchar(128);index"`
	LogId                 int     `json:"log_id" gorm:"index"`
	UserId                int     `json:"user_id" gorm:"index"`
	BillingSource         string  `json:"billing_source" gorm:"type:varchar(32);index"`
	ChannelId             int     `json:"channel_id" gorm:"index"`
	ChannelNameSnapshot   string  `json:"channel_name_snapshot" gorm:"type:varchar(255);default:''"`
	ProviderSnapshot      string  `json:"provider_snapshot" gorm:"type:varchar(64);index;default:''"`
	OriginModelName       string  `json:"origin_model_name" gorm:"type:varchar(255);index;default:''"`
	UpstreamModelName     string  `json:"upstream_model_name" gorm:"type:varchar(255);default:''"`
	EntryType             string  `json:"entry_type" gorm:"type:varchar(16);index"`
	PromptTokens          int     `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens      int     `json:"completion_tokens" gorm:"default:0"`
	CacheTokens           int     `json:"cache_tokens" gorm:"default:0"`
	TotalTokens           int     `json:"total_tokens" gorm:"default:0"`
	ActualQuota           int     `json:"actual_quota" gorm:"default:0"`
	PaidQuotaUsed         int     `json:"paid_quota_used" gorm:"default:0"`
	GiftQuotaUsed         int     `json:"gift_quota_used" gorm:"default:0"`
	InternalEquivalentUSD float64 `json:"internal_equivalent_usd" gorm:"default:0"`
	RecognizedRevenueUSD  float64 `json:"recognized_revenue_usd" gorm:"default:0"`
	EstimatedCostUSD      float64 `json:"estimated_cost_usd" gorm:"default:0"`
	CostBasis             string  `json:"cost_basis" gorm:"type:varchar(64);default:''"`
	OccurredAt            int64   `json:"occurred_at" gorm:"bigint;index"`
	CreatedAt             int64   `json:"created_at" gorm:"bigint"`
}

type ChannelCostAllocation struct {
	Id                  int     `json:"id"`
	LedgerId            int     `json:"ledger_id" gorm:"index:idx_channel_cost_alloc_ledger,priority:1;index"`
	FundingId           int     `json:"funding_id" gorm:"index"`
	FundingType         string  `json:"funding_type" gorm:"type:varchar(16);index"`
	SourceTypeSnapshot  string  `json:"source_type_snapshot" gorm:"type:varchar(32);index;default:''"`
	AllocatedQuota      int     `json:"allocated_quota" gorm:"default:0"`
	AllocatedRevenueUSD float64 `json:"allocated_revenue_usd" gorm:"default:0"`
	AllocatedCostUSD    float64 `json:"allocated_cost_usd" gorm:"default:0"`
	OccurredAt          int64   `json:"occurred_at" gorm:"bigint;index"`
	CreatedAt           int64   `json:"created_at" gorm:"bigint"`
}

type ChannelMonthlyStatement struct {
	Id                             int     `json:"id"`
	StatementNo                    string  `json:"statement_no" gorm:"type:varchar(64);uniqueIndex;default:''"`
	BillMonth                      string  `json:"bill_month" gorm:"type:varchar(7);uniqueIndex;index;default:''"`
	PeriodStart                    int64   `json:"period_start" gorm:"bigint"`
	PeriodEnd                      int64   `json:"period_end" gorm:"bigint"`
	Status                         string  `json:"status" gorm:"type:varchar(16);index;default:''"`
	TotalLedgerCount               int     `json:"total_ledger_count" gorm:"default:0"`
	TotalConsumeCount              int     `json:"total_consume_count" gorm:"default:0"`
	TotalRefundCount               int     `json:"total_refund_count" gorm:"default:0"`
	TotalPaidGrantedQuota          int     `json:"total_paid_granted_quota" gorm:"default:0"`
	TotalGiftGrantedQuota          int     `json:"total_gift_granted_quota" gorm:"default:0"`
	TotalGiftGrantedEquivalentUSD  float64 `json:"total_gift_granted_equivalent_usd" gorm:"default:0"`
	TotalPaidQuotaUsed             int     `json:"total_paid_quota_used" gorm:"default:0"`
	TotalGiftQuotaUsed             int     `json:"total_gift_quota_used" gorm:"default:0"`
	TotalSalesIncomeUSD            float64 `json:"total_sales_income_usd" gorm:"default:0"`
	TotalPaidConsumptionRevenueUSD float64 `json:"total_paid_consumption_revenue_usd" gorm:"default:0"`
	TotalEstimatedCostUSD          float64 `json:"total_estimated_cost_usd" gorm:"default:0"`
	TotalGiftCostUSD               float64 `json:"total_gift_cost_usd" gorm:"default:0"`
	TotalGrossProfitUSD            float64 `json:"total_gross_profit_usd" gorm:"default:0"`
	GeneratedAt                    int64   `json:"generated_at" gorm:"bigint"`
	CreatedAt                      int64   `json:"created_at" gorm:"bigint;index"`
	UpdatedAt                      int64   `json:"updated_at" gorm:"bigint"`
}

type ChannelMonthlyStatementItem struct {
	Id                   int     `json:"id"`
	StatementId          int     `json:"statement_id" gorm:"index:idx_channel_monthly_stmt_item,priority:1;index"`
	BillMonth            string  `json:"bill_month" gorm:"type:varchar(7);index;default:''"`
	ProviderSnapshot     string  `json:"provider_snapshot" gorm:"type:varchar(64);index;default:''"`
	ChannelId            int     `json:"channel_id" gorm:"index"`
	ChannelNameSnapshot  string  `json:"channel_name_snapshot" gorm:"type:varchar(255);default:''"`
	OriginModelName      string  `json:"origin_model_name" gorm:"type:varchar(255);index;default:''"`
	BillingSource        string  `json:"billing_source" gorm:"type:varchar(32);index;default:''"`
	ConsumeCount         int     `json:"consume_count" gorm:"default:0"`
	RefundCount          int     `json:"refund_count" gorm:"default:0"`
	PromptTokens         int     `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens     int     `json:"completion_tokens" gorm:"default:0"`
	CacheTokens          int     `json:"cache_tokens" gorm:"default:0"`
	TotalTokens          int     `json:"total_tokens" gorm:"default:0"`
	PaidQuotaUsed        int     `json:"paid_quota_used" gorm:"default:0"`
	GiftQuotaUsed        int     `json:"gift_quota_used" gorm:"default:0"`
	RecognizedRevenueUSD float64 `json:"recognized_revenue_usd" gorm:"default:0"`
	EstimatedCostUSD     float64 `json:"estimated_cost_usd" gorm:"default:0"`
	GiftCostUSD          float64 `json:"gift_cost_usd" gorm:"default:0"`
	GrossProfitUSD       float64 `json:"gross_profit_usd" gorm:"default:0"`
	CreatedAt            int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt            int64   `json:"updated_at" gorm:"bigint"`
}

type QuotaFundingGrantParams struct {
	UserId                   int
	FundingType              string
	SourceType               string
	BusinessType             string
	SourceRefId              int
	SourceName               string
	ExternalRef              string
	GrantedQuota             int
	RevenueUSD               float64
	ExpiredAt                int64
	OperatorUserId           int
	OperatorUsernameSnapshot string
	Remark                   string
	BalanceEntryType         string
	OccurredAt               int64
	SkipBalanceLedger        bool
}

type RecordChannelCostLedgerParams struct {
	RequestId         string
	LogId             int
	UserId            int
	BillingSource     string
	ChannelId         int
	ChannelType       int
	ChannelName       string
	OriginModelName   string
	UpstreamModelName string
	EntryType         string
	PromptTokens      int
	CompletionTokens  int
	CacheTokens       int
	ActualQuota       int
	EstimatedCostUSD  float64
	CostBasis         string
	OccurredAt        int64
	Allocations       []types.QuotaFundingAllocation
}

func quotaToUSDWithSnapshot(quota int, quotaPerUnit float64) float64 {
	if quota <= 0 || quotaPerUnit <= 0 {
		return 0
	}
	return roundAccountingAmount(float64(quota) / quotaPerUnit)
}

func roundAccountingAmount(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}

func normalizeQuotaFundingType(fundingType string) string {
	if fundingType == QuotaFundingTypePaid {
		return QuotaFundingTypePaid
	}
	return QuotaFundingTypeGift
}

func estimateCostUSDFromQuota(actualQuota int, effectiveGroupRatio float64) float64 {
	internalUSD := quotaToUSDWithSnapshot(actualQuota, common.QuotaPerUnit)
	if internalUSD <= 0 || effectiveGroupRatio <= 0 {
		return 0
	}
	return roundAccountingAmount(internalUSD / effectiveGroupRatio)
}

func buildRelayEffectiveGroupRatio(relayInfo *relaycommon.RelayInfo) float64 {
	if relayInfo == nil {
		return 1
	}
	if relayInfo.PriceData.GroupRatioInfo.HasSpecialRatio && relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio > 0 {
		return relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio
	}
	if relayInfo.PriceData.GroupRatioInfo.GroupRatio > 0 {
		return relayInfo.PriceData.GroupRatioInfo.GroupRatio
	}
	return 1
}

func BuildRelayEstimatedCostUSD(relayInfo *relaycommon.RelayInfo, actualQuota int) float64 {
	return estimateCostUSDFromQuota(actualQuota, buildRelayEffectiveGroupRatio(relayInfo))
}

func BuildRelayChannelCostLedgerParams(relayInfo *relaycommon.RelayInfo, entryType string, promptTokens int, completionTokens int, actualQuota int, occurredAt int64) RecordChannelCostLedgerParams {
	params := RecordChannelCostLedgerParams{
		UserId:            relayInfo.UserId,
		BillingSource:     relayInfo.BillingSource,
		ChannelId:         relayInfo.ChannelId,
		ChannelType:       relayInfo.ChannelType,
		OriginModelName:   relayInfo.OriginModelName,
		UpstreamModelName: relayInfo.UpstreamModelName,
		EntryType:         entryType,
		PromptTokens:      promptTokens,
		CompletionTokens:  completionTokens,
		ActualQuota:       actualQuota,
		EstimatedCostUSD:  BuildRelayEstimatedCostUSD(relayInfo, actualQuota),
		CostBasis:         "quota_div_group_ratio",
		OccurredAt:        occurredAt,
	}
	if relayInfo != nil {
		params.RequestId = relayInfo.RequestId
		params.Allocations = cloneQuotaFundingAllocations(relayInfo.QuotaFundingAllocations)
	}
	return params
}

func GrantQuotaFunding(params QuotaFundingGrantParams) error {
	return grantUserQuota(params)
}

func cloneQuotaFundingAllocations(allocations []types.QuotaFundingAllocation) []types.QuotaFundingAllocation {
	if len(allocations) == 0 {
		return nil
	}
	cloned := make([]types.QuotaFundingAllocation, len(allocations))
	copy(cloned, allocations)
	return cloned
}

func SumQuotaFundingAllocations(allocations []types.QuotaFundingAllocation) (paidQuota int, giftQuota int, revenueUSD float64) {
	for _, allocation := range allocations {
		if allocation.FundingType == QuotaFundingTypePaid {
			paidQuota += allocation.AllocatedQuota
		} else {
			giftQuota += allocation.AllocatedQuota
		}
		revenueUSD += allocation.AllocatedRevenueUSD
	}
	return paidQuota, giftQuota, roundAccountingAmount(revenueUSD)
}

func GrantPaidQuota(userId int, grantedQuota int, sourceType string, sourceRefId int, sourceName string, revenueUSD float64, remark string) error {
	return grantUserQuota(QuotaFundingGrantParams{
		UserId:       userId,
		FundingType:  QuotaFundingTypePaid,
		SourceType:   sourceType,
		SourceRefId:  sourceRefId,
		SourceName:   sourceName,
		GrantedQuota: grantedQuota,
		RevenueUSD:   revenueUSD,
		Remark:       remark,
	})
}

func GrantGiftQuota(userId int, grantedQuota int, sourceType string, sourceRefId int, sourceName string, remark string) error {
	return grantUserQuota(QuotaFundingGrantParams{
		UserId:       userId,
		FundingType:  QuotaFundingTypeGift,
		SourceType:   sourceType,
		SourceRefId:  sourceRefId,
		SourceName:   sourceName,
		GrantedQuota: grantedQuota,
		Remark:       remark,
	})
}

func grantUserQuota(params QuotaFundingGrantParams) error {
	if params.GrantedQuota < 0 {
		return errors.New("granted quota must not be negative")
	}
	if params.GrantedQuota == 0 {
		return nil
	}

	var user User
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		user, err = grantUserQuotaTx(tx, params)
		return err
	})
	if err != nil {
		return err
	}
	return updateUserCache(user)
}

func grantUserQuotaTx(tx *gorm.DB, params QuotaFundingGrantParams) (User, error) {
	var user User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", params.UserId).First(&user).Error; err != nil {
		return user, err
	}
	if err := ensureLegacyQuotaFundingCoverageTx(tx, &user); err != nil {
		return user, err
	}

	switch normalizeQuotaFundingType(params.FundingType) {
	case QuotaFundingTypePaid:
		user.PaidQuota += params.GrantedQuota
	default:
		user.GiftQuota += params.GrantedQuota
	}
	user.Quota = user.PaidQuota + user.GiftQuota

	now := common.GetTimestamp()
	if err := tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"paid_quota": user.PaidQuota,
		"gift_quota": user.GiftQuota,
		"quota":      user.Quota,
	}).Error; err != nil {
		return user, err
	}

	funding := &UserQuotaFunding{
		UserId:                    user.Id,
		FundingType:               normalizeQuotaFundingType(params.FundingType),
		SourceType:                params.SourceType,
		BusinessType:              firstNonEmpty(strings.TrimSpace(params.BusinessType), strings.TrimSpace(params.SourceType)),
		SourceRefId:               params.SourceRefId,
		SourceName:                params.SourceName,
		ExternalRef:               strings.TrimSpace(params.ExternalRef),
		GrantedQuota:              params.GrantedQuota,
		RemainingQuota:            params.GrantedQuota,
		RecognizedRevenueUSDTotal: roundAccountingAmount(params.RevenueUSD),
		QuotaPerUnitSnapshot:      common.QuotaPerUnit,
		ExpiredAt:                 params.ExpiredAt,
		OperatorUserId:            params.OperatorUserId,
		OperatorUsernameSnapshot:  strings.TrimSpace(params.OperatorUsernameSnapshot),
		Remark:                    params.Remark,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	if err := tx.Create(funding).Error; err != nil {
		return user, err
	}
	if !params.SkipBalanceLedger {
		if err := createUserBalanceLedgerTx(tx, buildGrantUserBalanceLedgerParams(params)); err != nil {
			return user, err
		}
	}
	return user, nil
}

func ConsumeUserQuotaWithAllocation(userId int, quota int) ([]types.QuotaFundingAllocation, error) {
	var allocations []types.QuotaFundingAllocation
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		allocations, err = consumeUserQuotaTx(tx, userId, quota)
		return err
	})
	return allocations, err
}

func RefundUserQuotaAllocations(userId int, allocations []types.QuotaFundingAllocation, quota int) ([]types.QuotaFundingAllocation, []types.QuotaFundingAllocation, error) {
	var remaining []types.QuotaFundingAllocation
	var refunded []types.QuotaFundingAllocation
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		remaining, refunded, err = refundUserQuotaAllocationsTx(tx, userId, allocations, quota)
		return err
	})
	return remaining, refunded, err
}

// RefundUserQuotaLegacy is an inventory-only fallback for legacy refund paths
// where detailed quota allocations are unavailable. It restores wallet balance
// into the legacy paid funding pool without creating a user-visible balance
// ledger event.
func RefundUserQuotaLegacy(userId int, quota int) ([]types.QuotaFundingAllocation, error) {
	if quota < 0 {
		return nil, errors.New("quota must not be negative")
	}
	if quota == 0 {
		return nil, nil
	}

	var user User
	var refunded []types.QuotaFundingAllocation
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", userId).First(&user).Error; err != nil {
			return err
		}
		if err := ensureLegacyQuotaFundingCoverageTx(tx, &user); err != nil {
			return err
		}

		now := common.GetTimestamp()
		var funding UserQuotaFunding
		err := tx.Where("user_id = ? AND funding_type = ? AND source_type = ?", userId, QuotaFundingTypePaid, QuotaFundingSourceLegacyBalance).
			Order("id asc").
			First(&funding).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			funding = UserQuotaFunding{
				UserId:                    userId,
				FundingType:               QuotaFundingTypePaid,
				SourceType:                QuotaFundingSourceLegacyBalance,
				SourceName:                "legacy refund balance",
				GrantedQuota:              0,
				RemainingQuota:            0,
				RecognizedRevenueUSDTotal: 0,
				QuotaPerUnitSnapshot:      common.QuotaPerUnit,
				CreatedAt:                 now,
				UpdatedAt:                 now,
			}
			if err := tx.Create(&funding).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&UserQuotaFunding{}).Where("id = ?", funding.Id).Updates(map[string]interface{}{
			"granted_quota":   gorm.Expr("granted_quota + ?", quota),
			"remaining_quota": gorm.Expr("remaining_quota + ?", quota),
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}

		user.PaidQuota += quota
		user.Quota = user.PaidQuota + user.GiftQuota
		if err := tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
			"paid_quota": user.PaidQuota,
			"gift_quota": user.GiftQuota,
			"quota":      user.Quota,
		}).Error; err != nil {
			return err
		}

		refunded = []types.QuotaFundingAllocation{
			{
				FundingId:           funding.Id,
				FundingType:         QuotaFundingTypePaid,
				SourceType:          QuotaFundingSourceLegacyBalance,
				AllocatedQuota:      quota,
				AllocatedRevenueUSD: 0,
			},
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := updateUserCache(user); err != nil {
		common.SysLog("failed to update user cache after legacy refund: " + err.Error())
	}
	return refunded, nil
}

func consumeUserQuotaTx(tx *gorm.DB, userId int, quota int) ([]types.QuotaFundingAllocation, error) {
	if quota < 0 {
		return nil, errors.New("quota must not be negative")
	}
	if quota == 0 {
		return nil, nil
	}

	var user User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", userId).First(&user).Error; err != nil {
		return nil, err
	}
	if err := ensureLegacyQuotaFundingCoverageTx(tx, &user); err != nil {
		return nil, err
	}
	if user.Quota < quota {
		return nil, fmt.Errorf("insufficient quota: remain=%d need=%d", user.Quota, quota)
	}

	var fundings []UserQuotaFunding
	query := tx.Where("user_id = ? AND remaining_quota > 0", userId).
		Order("CASE WHEN funding_type = 'gift' THEN 0 ELSE 1 END ASC").
		Order("created_at ASC").
		Order("id ASC")
	if err := query.Find(&fundings).Error; err != nil {
		return nil, err
	}

	remainingQuota := quota
	allocations := make([]types.QuotaFundingAllocation, 0, len(fundings))
	paidUsed := 0
	giftUsed := 0

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
		allocation := types.QuotaFundingAllocation{
			FundingId:           funding.Id,
			FundingType:         funding.FundingType,
			SourceType:          funding.SourceType,
			AllocatedQuota:      used,
			AllocatedRevenueUSD: roundAccountingAmount(revenueUSD),
		}
		allocations = append(allocations, allocation)
		if funding.FundingType == QuotaFundingTypePaid {
			paidUsed += used
		} else {
			giftUsed += used
		}
		remainingQuota -= used
	}
	if remainingQuota > 0 {
		return nil, fmt.Errorf("failed to allocate quota, remaining=%d", remainingQuota)
	}

	user.PaidQuota -= paidUsed
	user.GiftQuota -= giftUsed
	user.Quota = user.PaidQuota + user.GiftQuota
	if user.PaidQuota < 0 || user.GiftQuota < 0 || user.Quota < 0 {
		return nil, errors.New("quota pool became negative")
	}
	if err := tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"paid_quota": user.PaidQuota,
		"gift_quota": user.GiftQuota,
		"quota":      user.Quota,
	}).Error; err != nil {
		return nil, err
	}
	if err := updateUserCache(user); err != nil {
		common.SysLog("failed to update user cache after consume: " + err.Error())
	}
	return allocations, nil
}

func refundUserQuotaAllocationsTx(tx *gorm.DB, userId int, allocations []types.QuotaFundingAllocation, quota int) ([]types.QuotaFundingAllocation, []types.QuotaFundingAllocation, error) {
	if quota < 0 {
		return nil, nil, errors.New("quota must not be negative")
	}
	if quota == 0 || len(allocations) == 0 {
		return cloneQuotaFundingAllocations(allocations), nil, nil
	}

	var user User
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", userId).First(&user).Error; err != nil {
		return nil, nil, err
	}
	if err := ensureLegacyQuotaFundingCoverageTx(tx, &user); err != nil {
		return nil, nil, err
	}

	updated := cloneQuotaFundingAllocations(allocations)
	refunded := make([]types.QuotaFundingAllocation, 0)
	remainingQuota := quota

	for i := len(updated) - 1; i >= 0 && remainingQuota > 0; i-- {
		current := updated[i]
		if current.AllocatedQuota <= 0 {
			continue
		}
		refundAmount := current.AllocatedQuota
		if refundAmount > remainingQuota {
			refundAmount = remainingQuota
		}
		if refundAmount <= 0 {
			continue
		}

		if err := tx.Model(&UserQuotaFunding{}).Where("id = ?", current.FundingId).Updates(map[string]interface{}{
			"remaining_quota": gorm.Expr("remaining_quota + ?", refundAmount),
			"updated_at":      common.GetTimestamp(),
		}).Error; err != nil {
			return nil, nil, err
		}

		refundedRevenue := 0.0
		if current.AllocatedQuota > 0 && current.AllocatedRevenueUSD > 0 {
			refundedRevenue = current.AllocatedRevenueUSD * float64(refundAmount) / float64(current.AllocatedQuota)
		}
		refunded = append(refunded, types.QuotaFundingAllocation{
			FundingId:           current.FundingId,
			FundingType:         current.FundingType,
			SourceType:          current.SourceType,
			AllocatedQuota:      refundAmount,
			AllocatedRevenueUSD: roundAccountingAmount(refundedRevenue),
		})

		current.AllocatedQuota -= refundAmount
		current.AllocatedRevenueUSD = roundAccountingAmount(current.AllocatedRevenueUSD - refundedRevenue)
		updated[i] = current

		if current.FundingType == QuotaFundingTypePaid {
			user.PaidQuota += refundAmount
		} else {
			user.GiftQuota += refundAmount
		}
		remainingQuota -= refundAmount
	}

	if remainingQuota > 0 {
		return nil, nil, fmt.Errorf("failed to refund quota allocation, remaining=%d", remainingQuota)
	}

	user.Quota = user.PaidQuota + user.GiftQuota
	if err := tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"paid_quota": user.PaidQuota,
		"gift_quota": user.GiftQuota,
		"quota":      user.Quota,
	}).Error; err != nil {
		return nil, nil, err
	}
	if err := updateUserCache(user); err != nil {
		common.SysLog("failed to update user cache after refund: " + err.Error())
	}

	compacted := make([]types.QuotaFundingAllocation, 0, len(updated))
	for _, allocation := range updated {
		if allocation.AllocatedQuota > 0 {
			compacted = append(compacted, allocation)
		}
	}
	return compacted, refunded, nil
}

func ensureLegacyQuotaFundingCoverageTx(tx *gorm.DB, user *User) error {
	if user == nil {
		return errors.New("user is nil")
	}

	if user.PaidQuota == 0 && user.GiftQuota == 0 && user.Quota > 0 {
		user.PaidQuota = user.Quota
		if err := tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
			"paid_quota": user.PaidQuota,
			"gift_quota": 0,
			"quota":      user.Quota,
		}).Error; err != nil {
			return err
		}
	}

	var paidRemaining int64
	var giftRemaining int64
	if err := tx.Model(&UserQuotaFunding{}).
		Where("user_id = ? AND funding_type = ?", user.Id, QuotaFundingTypePaid).
		Select("COALESCE(SUM(remaining_quota), 0)").
		Scan(&paidRemaining).Error; err != nil {
		return err
	}
	if err := tx.Model(&UserQuotaFunding{}).
		Where("user_id = ? AND funding_type = ?", user.Id, QuotaFundingTypeGift).
		Select("COALESCE(SUM(remaining_quota), 0)").
		Scan(&giftRemaining).Error; err != nil {
		return err
	}

	now := common.GetTimestamp()
	if user.PaidQuota > int(paidRemaining) {
		missingPaid := user.PaidQuota - int(paidRemaining)
		if err := tx.Create(&UserQuotaFunding{
			UserId:                    user.Id,
			FundingType:               QuotaFundingTypePaid,
			SourceType:                QuotaFundingSourceLegacyBalance,
			SourceName:                "legacy balance",
			GrantedQuota:              missingPaid,
			RemainingQuota:            missingPaid,
			RecognizedRevenueUSDTotal: 0,
			QuotaPerUnitSnapshot:      common.QuotaPerUnit,
			CreatedAt:                 now,
			UpdatedAt:                 now,
		}).Error; err != nil {
			return err
		}
	}
	if user.GiftQuota > int(giftRemaining) {
		missingGift := user.GiftQuota - int(giftRemaining)
		if err := tx.Create(&UserQuotaFunding{
			UserId:                    user.Id,
			FundingType:               QuotaFundingTypeGift,
			SourceType:                QuotaFundingSourceLegacyGift,
			SourceName:                "legacy gift balance",
			GrantedQuota:              missingGift,
			RemainingQuota:            missingGift,
			RecognizedRevenueUSDTotal: 0,
			QuotaPerUnitSnapshot:      common.QuotaPerUnit,
			CreatedAt:                 now,
			UpdatedAt:                 now,
		}).Error; err != nil {
			return err
		}
	}

	user.Quota = user.PaidQuota + user.GiftQuota
	return nil
}

func RecordChannelCostLedger(params RecordChannelCostLedgerParams) error {
	if params.ActualQuota <= 0 {
		return nil
	}
	if params.OccurredAt == 0 {
		params.OccurredAt = common.GetTimestamp()
	}
	now := common.GetTimestamp()
	paidQuota, giftQuota, revenueUSD := SumQuotaFundingAllocations(params.Allocations)
	totalTokens := params.PromptTokens + params.CompletionTokens + params.CacheTokens

	return DB.Transaction(func(tx *gorm.DB) error {
		ledger := &ChannelCostLedger{
			BillMonth:             timestampToBillMonth(params.OccurredAt),
			RequestId:             params.RequestId,
			LogId:                 params.LogId,
			UserId:                params.UserId,
			BillingSource:         params.BillingSource,
			ChannelId:             params.ChannelId,
			ChannelNameSnapshot:   resolveChannelNameSnapshot(params.ChannelId, params.ChannelName),
			ProviderSnapshot:      constant.GetChannelTypeName(params.ChannelType),
			OriginModelName:       params.OriginModelName,
			UpstreamModelName:     params.UpstreamModelName,
			EntryType:             params.EntryType,
			PromptTokens:          params.PromptTokens,
			CompletionTokens:      params.CompletionTokens,
			CacheTokens:           params.CacheTokens,
			TotalTokens:           totalTokens,
			ActualQuota:           params.ActualQuota,
			PaidQuotaUsed:         paidQuota,
			GiftQuotaUsed:         giftQuota,
			InternalEquivalentUSD: quotaToUSDWithSnapshot(params.ActualQuota, common.QuotaPerUnit),
			RecognizedRevenueUSD:  revenueUSD,
			EstimatedCostUSD:      roundAccountingAmount(params.EstimatedCostUSD),
			CostBasis:             params.CostBasis,
			OccurredAt:            params.OccurredAt,
			CreatedAt:             now,
		}
		if err := tx.Create(ledger).Error; err != nil {
			return err
		}

		if len(params.Allocations) == 0 {
			return nil
		}

		allocations := make([]ChannelCostAllocation, 0, len(params.Allocations))
		for _, allocation := range params.Allocations {
			allocatedCostUSD := 0.0
			if params.ActualQuota > 0 && params.EstimatedCostUSD > 0 {
				allocatedCostUSD = params.EstimatedCostUSD * float64(allocation.AllocatedQuota) / float64(params.ActualQuota)
			}
			allocations = append(allocations, ChannelCostAllocation{
				LedgerId:            ledger.Id,
				FundingId:           allocation.FundingId,
				FundingType:         allocation.FundingType,
				SourceTypeSnapshot:  allocation.SourceType,
				AllocatedQuota:      allocation.AllocatedQuota,
				AllocatedRevenueUSD: allocation.AllocatedRevenueUSD,
				AllocatedCostUSD:    roundAccountingAmount(allocatedCostUSD),
				OccurredAt:          params.OccurredAt,
				CreatedAt:           now,
			})
		}
		return tx.Create(&allocations).Error
	})
}

func resolveChannelNameSnapshot(channelId int, fallback string) string {
	if fallback != "" {
		return fallback
	}
	if channelId <= 0 {
		return ""
	}
	if common.MemoryCacheEnabled {
		if channel, err := CacheGetChannel(channelId); err == nil && channel != nil {
			return channel.Name
		}
	}
	channel, err := GetChannelById(channelId, true)
	if err != nil || channel == nil {
		return ""
	}
	return channel.Name
}

func timestampToBillMonth(ts int64) string {
	return time.Unix(ts, 0).In(time.Local).Format("2006-01")
}

func UpdateTaskQuotaAllocationState(task *Task) error {
	if task == nil || task.ID == 0 {
		return nil
	}
	return DB.Model(&Task{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
		"quota":        task.Quota,
		"private_data": task.PrivateData,
	}).Error
}

func BackfillLegacyQuotaFunding() error {
	var users []User
	if err := DB.Where("(quota > 0) AND ((paid_quota = 0 AND gift_quota = 0) OR quota <> paid_quota + gift_quota)").Find(&users).Error; err != nil {
		return err
	}
	for _, user := range users {
		if err := DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", user.Id).First(&user).Error; err != nil {
				return err
			}
			if err := ensureLegacyQuotaFundingCoverageTx(tx, &user); err != nil {
				return err
			}
			return tx.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
				"paid_quota": user.PaidQuota,
				"gift_quota": user.GiftQuota,
				"quota":      user.PaidQuota + user.GiftQuota,
			}).Error
		}); err != nil {
			return err
		}
	}
	return nil
}
