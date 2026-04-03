package controller

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

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
}

func SearchRedemptions(c *gin.Context) {
	keyword := c.Query("keyword")
	pageInfo := common.GetPageQuery(c)
	redemptions, total, err := model.SearchRedemptions(keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(redemptions)
	common.ApiSuccess(c, pageInfo)
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
}

func AddRedemption(c *gin.Context) {
	redemption := model.Redemption{}
	if err := common.DecodeJson(c.Request.Body, &redemption); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := normalizeRedemptionPayload(&redemption); err != nil {
		common.ApiError(c, err)
		return
	}
	if redemption.Count <= 0 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountPositive)
		return
	}
	if redemption.Count > 100 {
		common.ApiErrorI18n(c, i18n.MsgRedemptionCountMax)
		return
	}
	if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}

	keys, err := model.CreateRedemptionsWithAudit(redemption, redemption.Count, model.FinancialAuditOperator{
		UserId:           c.GetInt("id"),
		UsernameSnapshot: c.GetString("username"),
	})
	if err != nil {
		common.SysError("failed to insert redemption: " + err.Error())
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.T(c, i18n.MsgRedemptionCreateFailed),
			"data":    []string{},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    keys,
	})
}

func DeleteRedemption(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteRedemptionByIdWithAudit(id, model.FinancialAuditOperator{
		UserId:           c.GetInt("id"),
		UsernameSnapshot: c.GetString("username"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func UpdateRedemption(c *gin.Context) {
	statusOnly := c.Query("status_only")
	redemption := model.Redemption{}
	if err := common.DecodeJson(c.Request.Body, &redemption); err != nil {
		common.ApiError(c, err)
		return
	}
	if statusOnly == "" {
		if err := normalizeRedemptionPayload(&redemption); err != nil {
			common.ApiError(c, err)
			return
		}
		if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
			return
		}
	}
	cleanRedemption, err := model.UpdateRedemptionWithAudit(redemption, statusOnly != "", model.FinancialAuditOperator{
		UserId:           c.GetInt("id"),
		UsernameSnapshot: c.GetString("username"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanRedemption,
	})
}

func DeleteInvalidRedemption(c *gin.Context) {
	rows, err := model.DeleteInvalidRedemptionsWithAudit(model.FinancialAuditOperator{
		UserId:           c.GetInt("id"),
		UsernameSnapshot: c.GetString("username"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
}

func validateExpiredTime(c *gin.Context, expired int64) (bool, string) {
	if expired != 0 && expired < common.GetTimestamp() {
		return false, i18n.T(c, i18n.MsgRedemptionExpireTimeInvalid)
	}
	return true, ""
}

func normalizeRedemptionPayload(redemption *model.Redemption) error {
	if redemption == nil {
		return errors.New("invalid redemption payload")
	}

	redemption.FundingType = strings.TrimSpace(redemption.FundingType)
	if redemption.FundingType != model.QuotaFundingTypePaid {
		redemption.FundingType = model.QuotaFundingTypeGift
	}
	redemption.Name = strings.TrimSpace(redemption.Name)
	redemption.Remark = strings.TrimSpace(redemption.Remark)

	if redemption.AmountUSD <= 0 && redemption.Quota > 0 {
		if common.QuotaPerUnit <= 0 {
			return errors.New("quota_per_unit is invalid")
		}
		redemption.AmountUSD = float64(redemption.Quota) / common.QuotaPerUnit
	}
	if redemption.AmountUSD <= 0 {
		return errors.New("amount_usd must be greater than 0")
	}
	if redemption.Quota <= 0 {
		if common.QuotaPerUnit <= 0 {
			return errors.New("quota_per_unit is invalid")
		}
		redemption.Quota = int(math.Round(redemption.AmountUSD * common.QuotaPerUnit))
	}
	if redemption.Quota <= 0 {
		return errors.New("quota must be greater than 0")
	}

	redemption.AmountUSD = modelAmountRound(redemption.AmountUSD)
	redemption.QuotaPerUnitSnapshot = common.QuotaPerUnit

	if redemption.FundingType == model.QuotaFundingTypePaid {
		if redemption.RecognizedRevenueUSD <= 0 {
			redemption.RecognizedRevenueUSD = redemption.AmountUSD
		}
		redemption.RecognizedRevenueUSD = modelAmountRound(redemption.RecognizedRevenueUSD)
	} else {
		redemption.RecognizedRevenueUSD = 0
		if redemption.Remark == "" {
			return errors.New("remark is required for free redemption codes")
		}
	}

	if utf8.RuneCountInString(redemption.Remark) > 255 {
		return errors.New("remark must be 255 characters or fewer")
	}
	if redemption.Name == "" {
		redemption.Name = buildDefaultRedemptionName(redemption.FundingType, redemption.AmountUSD)
	}
	if utf8.RuneCountInString(redemption.Name) == 0 || utf8.RuneCountInString(redemption.Name) > 20 {
		return errors.New("name must be between 1 and 20 characters")
	}
	return nil
}

func buildDefaultRedemptionName(fundingType string, amountUSD float64) string {
	label := "Free"
	if fundingType == model.QuotaFundingTypePaid {
		label = "Paid"
	}
	return fmt.Sprintf("%s $%s", label, strconv.FormatFloat(amountUSD, 'f', 2, 64))
}

func modelAmountRound(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}
