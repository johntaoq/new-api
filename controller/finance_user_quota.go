package controller

import (
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// AdjustUserWallet preserves the production paid/gift wallet adjustment contract.
func AdjustUserWallet(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	var req struct {
		FundingType string   `json:"funding_type"`
		DeltaQuota  int      `json:"delta_quota"`
		SourceType  string   `json:"source_type"`
		Remark      string   `json:"remark"`
		RevenueUSD  *float64 `json:"revenue_usd"`
	}
	if err != nil || id <= 0 || common.DecodeJson(c.Request.Body, &req) != nil ||
		req.DeltaQuota == 0 || req.DeltaQuota > common.MaxWalletQuota || req.DeltaQuota < -common.MaxWalletQuota {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	user, err := model.GetUserById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	root := c.GetInt("role") == common.RoleRootUser
	if !canManageTargetRole(c.GetInt("role"), user.Role) || (!root && req.FundingType == model.QuotaFundingTypePaid) {
		common.ApiErrorT(c, "Insufficient permissions")
		return
	}
	if req.FundingType != model.QuotaFundingTypePaid && req.FundingType != model.QuotaFundingTypeGift {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	source := strings.TrimSpace(req.SourceType)
	entry := model.CustomerMonthlyStatementEntryTypeAdjustment
	if req.FundingType == model.QuotaFundingTypePaid {
		switch source {
		case model.QuotaFundingSourceCompensation, model.QuotaFundingSourceRefund,
			model.QuotaFundingSourceManualTopUp, model.QuotaFundingSourceAccountClosure:
		default:
			common.ApiErrorT(c, "Select an adjustment reason")
			return
		}
		if source == model.QuotaFundingSourceManualTopUp && req.DeltaQuota > 0 {
			entry = model.CustomerMonthlyStatementEntryTypeTopup
		}
	} else {
		switch source {
		case model.QuotaFundingSourceAdminGrant, model.QuotaFundingSourcePromoCampaign,
			model.QuotaFundingSourceCompensation, model.QuotaFundingSourceGiftVoucher:
		case model.QuotaFundingSourceSystemAdjust:
			if !root {
				common.ApiErrorT(c, "Insufficient permissions")
				return
			}
		default:
			common.ApiErrorT(c, "Select an adjustment reason")
			return
		}
		if req.DeltaQuota > 0 {
			entry = model.CustomerMonthlyStatementEntryTypeGift
		}
	}
	revenue := 0.0
	if req.FundingType == model.QuotaFundingTypePaid && common.QuotaPerUnit > 0 {
		revenue = float64(req.DeltaQuota) / common.QuotaPerUnit
	}
	if req.RevenueUSD != nil {
		revenue = *req.RevenueUSD
	}
	if math.IsNaN(revenue) || math.IsInf(revenue, 0) ||
		(req.FundingType == model.QuotaFundingTypeGift && revenue != 0) {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	if err := model.AdjustUserQuotaWithAudit(model.UserQuotaAdjustmentParams{
		UserId: id, FundingType: req.FundingType, DeltaQuota: req.DeltaQuota,
		SourceType: source, RevenueUSD: revenue, EntryType: entry,
		OperatorUserId: c.GetInt("id"), OperatorUsernameSnapshot: c.GetString("username"),
		Remark: strings.TrimSpace(req.Remark),
	}); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccessT(c, "Quota adjusted successfully", nil)
}
