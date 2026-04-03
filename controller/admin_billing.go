package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type AdminAdjustUserQuotaRequest struct {
	BucketType   string   `json:"bucket_type"`
	FundingType  string   `json:"funding_type"`
	DeltaQuota   int      `json:"delta_quota"`
	SourceType   string   `json:"source_type"`
	BusinessType string   `json:"business_type"`
	ExternalRef  string   `json:"external_ref"`
	Remark       string   `json:"remark"`
	RevenueUSD   *float64 `json:"revenue_usd"`
}

func AdjustUserQuotaByAdmin(c *gin.Context) {
	targetUserId, err := strconv.Atoi(strings.TrimSpace(c.Param("id")))
	if err != nil || targetUserId <= 0 {
		common.ApiError(c, errors.New("invalid user id"))
		return
	}

	var req AdminAdjustUserQuotaRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}
	if req.DeltaQuota == 0 {
		common.ApiError(c, errors.New("delta_quota must not be zero"))
		return
	}

	targetUser, err := model.GetUserById(targetUserId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	operatorRole := c.GetInt("role")
	operatorStaffRole := c.GetString("staff_role")
	if !model.CanAdjustUserFinance(operatorRole, operatorStaffRole, targetUser) {
		common.ApiError(c, errors.New("insufficient permission to adjust this user"))
		return
	}

	fundingType := strings.TrimSpace(req.BucketType)
	if fundingType == "" {
		fundingType = strings.TrimSpace(req.FundingType)
	}
	if fundingType != model.QuotaFundingTypePaid {
		fundingType = model.QuotaFundingTypeGift
	}
	if fundingType == model.QuotaFundingTypePaid && !model.HasPermission(operatorRole, operatorStaffRole, common.PermissionSystemManage) {
		common.ApiError(c, errors.New("only root can adjust paid quota"))
		return
	}

	sourceType, err := resolveQuotaAdjustmentSourceType(
		fundingType,
		req.DeltaQuota,
		model.HasPermission(operatorRole, operatorStaffRole, common.PermissionSystemManage),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	remark := strings.TrimSpace(req.Remark)
	if remark == "" {
		if req.DeltaQuota > 0 {
			remark = "manual quota adjustment"
		} else {
			remark = "manual quota deduction"
		}
	}

	revenueUSD := 0.0
	if req.RevenueUSD != nil && *req.RevenueUSD >= 0 {
		revenueUSD = *req.RevenueUSD
	}

	entryType := model.CustomerMonthlyStatementEntryTypeAdjustment
	if req.DeltaQuota > 0 {
		if fundingType == model.QuotaFundingTypePaid {
			entryType = model.CustomerMonthlyStatementEntryTypeTopup
		} else {
			entryType = model.CustomerMonthlyStatementEntryTypeGift
		}
	}

	if err := model.AdjustUserQuotaWithAudit(model.UserQuotaAdjustmentParams{
		UserId:                   targetUserId,
		FundingType:              fundingType,
		DeltaQuota:               req.DeltaQuota,
		SourceType:               sourceType,
		RevenueUSD:               revenueUSD,
		ExternalRef:              strings.TrimSpace(req.ExternalRef),
		OperatorUserId:           c.GetInt("id"),
		OperatorUsernameSnapshot: c.GetString("username"),
		Remark:                   remark,
		EntryType:                entryType,
	}); err != nil {
		common.ApiError(c, err)
		return
	}

	operator := c.GetString("username")
	actionLabel := "gift quota"
	if fundingType == model.QuotaFundingTypePaid {
		actionLabel = "paid quota"
	}
	model.RecordLog(targetUserId, model.LogTypeManage, fmt.Sprintf("admin %s adjusted %s %+d, source=%s, remark=%s", operator, actionLabel, req.DeltaQuota, sourceType, remark))

	updatedUser, err := model.GetUserById(targetUserId, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"user": updatedUser,
	})
}

func resolveQuotaAdjustmentSourceType(fundingType string, deltaQuota int, isRoot bool) (string, error) {
	if strings.TrimSpace(fundingType) == model.QuotaFundingTypePaid {
		if !isRoot {
			return "", errors.New("only root can adjust paid quota")
		}
		return model.QuotaFundingSourceSystemAdjust, nil
	}
	if deltaQuota > 0 {
		return model.QuotaFundingSourceAdminGrant, nil
	}
	return model.QuotaFundingSourceSystemAdjust, nil
}
