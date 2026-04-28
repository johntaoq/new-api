package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CardIssueDefaultAppId = "default"
)

type CardIssue struct {
	Id                       int     `json:"id"`
	RedemptionId             int     `json:"redemption_id" gorm:"uniqueIndex;not null"`
	RequestId                string  `json:"request_id" gorm:"type:varchar(128);uniqueIndex;not null"`
	AppId                    string  `json:"app_id" gorm:"type:varchar(64);index;not null"`
	FundingType              string  `json:"funding_type" gorm:"type:varchar(16);index;not null"`
	AmountUSD                float64 `json:"amount_usd" gorm:"type:decimal(16,6);index;not null"`
	Quota                    int     `json:"quota" gorm:"index;not null"`
	IssuedTo                 string  `json:"issued_to" gorm:"type:varchar(128);default:''"`
	Remark                   string  `json:"remark" gorm:"type:varchar(255);default:''"`
	OperatorUserId           int     `json:"operator_user_id" gorm:"index;default:0"`
	OperatorUsernameSnapshot string  `json:"operator_username_snapshot" gorm:"type:varchar(64);default:''"`
	CreatedAt                int64   `json:"created_at" gorm:"bigint;index"`
}

type ClaimCardIssueParams struct {
	RequestId  string
	AppId      string
	IssuedTo   string
	Remark     string
	Template   Redemption
	AutoCreate bool
	Operator   FinancialAuditOperator
}

type ClaimCardIssueResult struct {
	CardIssueId   int     `json:"card_issue_id"`
	RedemptionId  int     `json:"redemption_id"`
	RequestId     string  `json:"request_id"`
	AppId         string  `json:"app_id"`
	Key           string  `json:"key"`
	FundingType   string  `json:"funding_type"`
	AmountUSD     float64 `json:"amount_usd"`
	Quota         int     `json:"quota"`
	CreatedNew    bool    `json:"created_new"`
	ReusedRequest bool    `json:"reused_request"`
}

func ClaimCardIssue(params ClaimCardIssueParams) (*ClaimCardIssueResult, error) {
	if err := normalizeClaimCardIssueParams(&params); err != nil {
		return nil, err
	}

	var result *ClaimCardIssueResult
	err := DB.Transaction(func(tx *gorm.DB) error {
		if existing, redemption, err := getCardIssueByRequestIdTx(tx, params.RequestId, true); err == nil {
			result = buildClaimCardIssueResult(existing, redemption, false, true)
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		redemption, createdNew, err := findOrCreateClaimableRedemptionTx(tx, params)
		if err != nil {
			return err
		}

		now := common.GetTimestamp()
		issue := &CardIssue{
			RedemptionId:             redemption.Id,
			RequestId:                params.RequestId,
			AppId:                    params.AppId,
			FundingType:              redemption.FundingType,
			AmountUSD:                redemption.AmountUSD,
			Quota:                    redemption.Quota,
			IssuedTo:                 params.IssuedTo,
			Remark:                   params.Remark,
			OperatorUserId:           params.Operator.UserId,
			OperatorUsernameSnapshot: params.Operator.UsernameSnapshot,
			CreatedAt:                now,
		}
		if err := tx.Create(issue).Error; err != nil {
			return err
		}

		if err := recordFinancialAuditTx(tx, RecordFinancialAuditParams{
			Module:                   FinancialAuditModuleRedemption,
			Action:                   FinancialAuditActionCreate,
			OperatorUserId:           params.Operator.UserId,
			OperatorUsernameSnapshot: params.Operator.UsernameSnapshot,
			TargetType:               FinancialAuditTargetCardIssue,
			TargetId:                 issue.Id,
			Before: map[string]any{
				"request_id":  params.RequestId,
				"app_id":      params.AppId,
				"issued_to":   params.IssuedTo,
				"auto_create": params.AutoCreate,
			},
			After: map[string]any{
				"card_issue_id": issue.Id,
				"redemption_id": redemption.Id,
				"funding_type":  redemption.FundingType,
				"amount_usd":    redemption.AmountUSD,
				"quota":         redemption.Quota,
				"created_new":   createdNew,
			},
			Remark:    "issued redemption code to external card system",
			CreatedAt: now,
		}); err != nil {
			return err
		}

		result = buildClaimCardIssueResult(issue, redemption, createdNew, false)
		return nil
	})
	if err != nil {
		if existing, redemption, existingErr := GetCardIssueByRequestId(params.RequestId); existingErr == nil {
			return buildClaimCardIssueResult(existing, redemption, false, true), nil
		}
		return nil, err
	}
	return result, nil
}

func GetCardIssueByRequestId(requestId string) (*CardIssue, *Redemption, error) {
	requestId = strings.TrimSpace(requestId)
	if requestId == "" {
		return nil, nil, errors.New("request_id is required")
	}
	return getCardIssueByRequestIdTx(DB, requestId, false)
}

func normalizeClaimCardIssueParams(params *ClaimCardIssueParams) error {
	if params == nil {
		return errors.New("invalid card issue params")
	}
	params.RequestId = strings.TrimSpace(params.RequestId)
	if params.RequestId == "" {
		return errors.New("request_id is required")
	}
	if len(params.RequestId) > 128 {
		return errors.New("request_id must be 128 characters or fewer")
	}

	params.AppId = strings.TrimSpace(params.AppId)
	if params.AppId == "" {
		params.AppId = CardIssueDefaultAppId
	}
	if len(params.AppId) > 64 {
		return errors.New("app_id must be 64 characters or fewer")
	}

	params.IssuedTo = strings.TrimSpace(params.IssuedTo)
	if len(params.IssuedTo) > 128 {
		return errors.New("issued_to must be 128 characters or fewer")
	}

	params.Remark = strings.TrimSpace(params.Remark)
	if len(params.Remark) > 255 {
		return errors.New("remark must be 255 characters or fewer")
	}

	params.Template.normalizeForPersistence()
	if params.Template.FundingType != QuotaFundingTypePaid {
		params.Template.FundingType = QuotaFundingTypeGift
	}
	if params.Template.AmountUSD <= 0 {
		return errors.New("amount_usd must be greater than 0")
	}
	if params.Template.Quota <= 0 {
		return errors.New("quota must be greater than 0")
	}
	if params.Template.Name == "" {
		params.Template.Name = fmt.Sprintf("%s $%.2f", params.Template.FundingType, params.Template.AmountUSD)
	}
	return nil
}

func getCardIssueByRequestIdTx(tx *gorm.DB, requestId string, lock bool) (*CardIssue, *Redemption, error) {
	query := tx.Where("request_id = ?", requestId)
	if lock && !common.UsingSQLite {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	issue := &CardIssue{}
	if err := query.First(issue).Error; err != nil {
		return nil, nil, err
	}

	redemption := &Redemption{}
	if err := tx.First(redemption, "id = ?", issue.RedemptionId).Error; err != nil {
		return nil, nil, err
	}
	redemption.normalizeForResponse()
	return issue, redemption, nil
}

func findOrCreateClaimableRedemptionTx(tx *gorm.DB, params ClaimCardIssueParams) (*Redemption, bool, error) {
	redemption := &Redemption{}
	query := tx.Model(&Redemption{}).
		Where("status = ?", common.RedemptionCodeStatusEnabled).
		Where("used_user_id = 0").
		Where("funding_type = ?", params.Template.FundingType).
		Where("amount_usd = ?", params.Template.AmountUSD).
		Where("quota = ?", params.Template.Quota).
		Where("(expired_time = 0 OR expired_time > ?)", common.GetTimestamp()).
		Where("NOT EXISTS (SELECT 1 FROM card_issues WHERE card_issues.redemption_id = redemptions.id)").
		Order("id ASC")
	if !common.UsingSQLite {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.First(redemption).Error
	if err == nil {
		redemption.normalizeForResponse()
		return redemption, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	if !params.AutoCreate {
		return nil, false, errors.New("no available redemption code")
	}

	now := common.GetTimestamp()
	redemption = &Redemption{
		UserId:               params.Operator.UserId,
		Key:                  common.GetUUID(),
		Status:               common.RedemptionCodeStatusEnabled,
		Name:                 params.Template.Name,
		FundingType:          params.Template.FundingType,
		Quota:                params.Template.Quota,
		AmountUSD:            params.Template.AmountUSD,
		RecognizedRevenueUSD: params.Template.RecognizedRevenueUSD,
		Remark:               params.Template.Remark,
		QuotaPerUnitSnapshot: params.Template.QuotaPerUnitSnapshot,
		CreatedTime:          now,
		ExpiredTime:          params.Template.ExpiredTime,
	}
	redemption.normalizeForPersistence()
	if err := tx.Create(redemption).Error; err != nil {
		return nil, false, err
	}
	return redemption, true, nil
}

func buildClaimCardIssueResult(issue *CardIssue, redemption *Redemption, createdNew bool, reusedRequest bool) *ClaimCardIssueResult {
	return &ClaimCardIssueResult{
		CardIssueId:   issue.Id,
		RedemptionId:  issue.RedemptionId,
		RequestId:     issue.RequestId,
		AppId:         issue.AppId,
		Key:           redemption.Key,
		FundingType:   redemption.FundingType,
		AmountUSD:     redemption.AmountUSD,
		Quota:         redemption.Quota,
		CreatedNew:    createdNew,
		ReusedRequest: reusedRequest,
	}
}
