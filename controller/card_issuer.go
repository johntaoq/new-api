package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type claimCardIssueRequest struct {
	RequestId              string  `json:"request_id"`
	AppId                  string  `json:"app_id"`
	FundingType            string  `json:"funding_type"`
	AmountUSD              float64 `json:"amount_usd"`
	Quota                  int     `json:"quota"`
	Name                   string  `json:"name"`
	Remark                 string  `json:"remark"`
	IssuedTo               string  `json:"issued_to"`
	ExpiredTime            int64   `json:"expired_time"`
	RecognizedRevenueUSD   float64 `json:"recognized_revenue_usd"`
	QuotaPerUnitSnapshot   float64 `json:"quota_per_unit_snapshot"`
	AutoCreate             *bool   `json:"auto_create"`
	RedemptionRemark       string  `json:"redemption_remark"`
	CardIssueRemark        string  `json:"card_issue_remark"`
	ExternalRecipient      string  `json:"external_recipient"`
	ExternalRequestIdAlias string  `json:"external_request_id"`
}

func ClaimCardIssue(c *gin.Context) {
	request := claimCardIssueRequest{}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiError(c, err)
		return
	}

	redemptionRemark := strings.TrimSpace(request.RedemptionRemark)
	if redemptionRemark == "" {
		redemptionRemark = request.Remark
	}
	issuedTo := strings.TrimSpace(request.IssuedTo)
	if issuedTo == "" {
		issuedTo = request.ExternalRecipient
	}
	requestId := strings.TrimSpace(request.RequestId)
	if requestId == "" {
		requestId = request.ExternalRequestIdAlias
	}

	redemption := model.Redemption{
		Name:                 request.Name,
		FundingType:          request.FundingType,
		Quota:                request.Quota,
		AmountUSD:            request.AmountUSD,
		RecognizedRevenueUSD: request.RecognizedRevenueUSD,
		Remark:               redemptionRemark,
		QuotaPerUnitSnapshot: request.QuotaPerUnitSnapshot,
		ExpiredTime:          request.ExpiredTime,
	}
	if err := normalizeRedemptionPayload(&redemption); err != nil {
		common.ApiError(c, err)
		return
	}
	if valid, msg := validateExpiredTime(c, redemption.ExpiredTime); !valid {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}

	autoCreate := true
	if request.AutoCreate != nil {
		autoCreate = *request.AutoCreate
	}
	issueRemark := strings.TrimSpace(request.CardIssueRemark)
	if issueRemark == "" {
		issueRemark = request.Remark
	}

	result, err := model.ClaimCardIssue(model.ClaimCardIssueParams{
		RequestId:  requestId,
		AppId:      request.AppId,
		IssuedTo:   issuedTo,
		Remark:     issueRemark,
		Template:   redemption,
		AutoCreate: autoCreate,
		Operator: model.FinancialAuditOperator{
			UserId:           c.GetInt("id"),
			UsernameSnapshot: c.GetString("username"),
		},
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}
