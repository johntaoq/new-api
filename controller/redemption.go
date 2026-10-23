package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

func GetAllRedemptions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.GetAllRedemptions(pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func SearchRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	status := c.Query("status")
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.SearchRedemptions(keyword, status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetRedemption(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	redemption, err := model.GetRedemptionById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    redemption,
	})
	return
}

func AddRedemption(c *gin.Context) {
	if !operation_setting.IsPaymentComplianceConfirmed() {
		common.ApiErrorT(c, "Payment, redemption, subscription, and invitation reward features are disabled. The administrator must confirm compliance terms before enabling them.")
		return
	}

	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if redemption.Count <= 0 {
		common.ApiErrorT(c, "Redemption code count must be greater than 0")
		return
	}
	if redemption.Count > 100 {
		common.ApiErrorT(c, "Maximum 100 redemption codes can be generated at once")
		return
	}
	redemption.QuotaPerUnitSnapshot = common.QuotaPerUnit
	redemption.ExchangeRate = 0
	if redemption.Currency != "" {
		if err := redemption.SetFinanceFaceValue(redemption.Currency, redemption.InputAmount); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if err := redemption.ValidateFinance(); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := validateExpiredTime(redemption.ExpiredTime); err != nil {
		common.ApiError(c, err)
		return
	}
	keys, err := model.CreateRedemptionsWithAudit(redemption, redemption.Count,
		model.FinancialAuditOperator{UserId: c.GetInt("id"), UsernameSnapshot: c.GetString("username")})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "redemption.create", map[string]any{
		"name":  redemption.Name,
		"count": redemption.Count,
		"quota": logger.FormatQuota(redemption.Quota),
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
	return
}

func DeleteRedemption(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteRedemptionByIdWithAudit(id,
		model.FinancialAuditOperator{UserId: c.GetInt("id"), UsernameSnapshot: c.GetString("username")})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func UpdateRedemption(c *gin.Context) {
	statusOnly := c.Query("status_only")
	redemption := model.Redemption{}
	err := c.ShouldBindJSON(&redemption)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanRedemption, err := model.GetRedemptionById(redemption.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if statusOnly == "" {
		if err := validateExpiredTime(redemption.ExpiredTime); err != nil {
			common.ApiError(c, err)
			return
		}
		// If you add more fields, please also update redemption.Update()
		cleanRedemption.Name = redemption.Name
		if redemption.Currency != "" {
			if redemption.Currency != cleanRedemption.Currency || redemption.InputAmount != cleanRedemption.InputAmount {
				if err := cleanRedemption.SetFinanceFaceValue(redemption.Currency, redemption.InputAmount); err != nil {
					common.ApiError(c, err)
					return
				}
			}
		} else if redemption.Quota != cleanRedemption.Quota {
			cleanRedemption.Quota = redemption.Quota
			cleanRedemption.QuotaPerUnitSnapshot = common.QuotaPerUnit
			cleanRedemption.Currency, cleanRedemption.InputAmount, cleanRedemption.ExchangeRate = "", 0, 0
		}
		cleanRedemption.ExpiredTime = redemption.ExpiredTime
		// Older clients omit these fields; preserve the voucher's classification.
		if redemption.FundingType != "" {
			cleanRedemption.FundingType = redemption.FundingType
			cleanRedemption.RecognizedRevenueUSD = redemption.RecognizedRevenueUSD
			cleanRedemption.Remark = redemption.Remark
		}
		cleanRedemption.AmountUSD = redemption.AmountUSD
		if err := cleanRedemption.ValidateFinance(); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if statusOnly != "" {
		cleanRedemption.Status = redemption.Status
	}
	cleanRedemption, err = model.UpdateRedemptionWithAudit(*cleanRedemption, statusOnly != "",
		model.FinancialAuditOperator{UserId: c.GetInt("id"), UsernameSnapshot: c.GetString("username")})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanRedemption,
	})
	return
}

func DeleteInvalidRedemption(c *gin.Context) {
	rows, err := model.DeleteInvalidRedemptionsWithAudit(
		model.FinancialAuditOperator{UserId: c.GetInt("id"), UsernameSnapshot: c.GetString("username")})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
	return
}

func validateExpiredTime(expired int64) error {
	if expired != 0 && expired < common.GetTimestamp() {
		return common.NewMessage("Expiration time cannot be earlier than current time")
	}
	return nil
}

func DeleteRedemptionBatch(c *gin.Context) {
	var request struct {
		Ids []int `json:"ids" binding:"required,min=1,max=1000,dive,gt=0"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	count, err := model.BatchDeleteRedemptionsWithAudit(request.Ids,
		model.FinancialAuditOperator{UserId: c.GetInt("id"), UsernameSnapshot: c.GetString("username")})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "redemption.delete_batch", map[string]any{
		"count":                    count,
		"total":                    len(request.Ids),
		"requested_redemption_ids": request.Ids,
	})
	common.ApiSuccess(c, count)
}
