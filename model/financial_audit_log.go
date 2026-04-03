package model

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	FinancialAuditModuleRedemption = "redemption"
	FinancialAuditModuleUserQuota  = "user_quota"

	FinancialAuditActionCreate         = "create"
	FinancialAuditActionUpdate         = "update"
	FinancialAuditActionDelete         = "delete"
	FinancialAuditActionCleanupInvalid = "cleanup_invalid"
	FinancialAuditActionAdjust         = "adjust"

	FinancialAuditTargetRedemption      = "redemption"
	FinancialAuditTargetRedemptionBatch = "redemption_batch"
	FinancialAuditTargetUserWallet      = "user_wallet"
)

type FinancialAuditLog struct {
	Id                       int    `json:"id"`
	Module                   string `json:"module" gorm:"type:varchar(32);index"`
	Action                   string `json:"action" gorm:"type:varchar(32);index"`
	OperatorUserId           int    `json:"operator_user_id" gorm:"index;default:0"`
	OperatorUsernameSnapshot string `json:"operator_username_snapshot" gorm:"type:varchar(64);default:''"`
	TargetType               string `json:"target_type" gorm:"type:varchar(32);index"`
	TargetId                 int    `json:"target_id" gorm:"index;default:0"`
	TargetUserId             int    `json:"target_user_id" gorm:"index;default:0"`
	BeforeJSON               string `json:"before_json" gorm:"type:text"`
	AfterJSON                string `json:"after_json" gorm:"type:text"`
	Remark                   string `json:"remark" gorm:"type:varchar(255);default:''"`
	CreatedAt                int64  `json:"created_at" gorm:"bigint;index"`
}

type FinancialAuditOperator struct {
	UserId           int
	UsernameSnapshot string
}

type RecordFinancialAuditParams struct {
	Module                   string
	Action                   string
	OperatorUserId           int
	OperatorUsernameSnapshot string
	TargetType               string
	TargetId                 int
	TargetUserId             int
	Before                   any
	After                    any
	Remark                   string
	CreatedAt                int64
}

type financialAuditRedemptionSnapshot struct {
	Id                   int     `json:"id"`
	CreatorUserId        int     `json:"creator_user_id"`
	Status               int     `json:"status"`
	Name                 string  `json:"name"`
	FundingType          string  `json:"funding_type"`
	Quota                int     `json:"quota"`
	AmountUSD            float64 `json:"amount_usd"`
	RecognizedRevenueUSD float64 `json:"recognized_revenue_usd"`
	Remark               string  `json:"remark"`
	QuotaPerUnitSnapshot float64 `json:"quota_per_unit_snapshot"`
	CreatedTime          int64   `json:"created_time"`
	RedeemedTime         int64   `json:"redeemed_time"`
	UsedUserId           int     `json:"used_user_id"`
	ExpiredTime          int64   `json:"expired_time"`
}

type financialAuditUserQuotaSnapshot struct {
	Quota     int `json:"quota"`
	PaidQuota int `json:"paid_quota"`
	GiftQuota int `json:"gift_quota"`
}

func RecordFinancialAudit(params RecordFinancialAuditParams) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		return recordFinancialAuditTx(tx, params)
	})
}

func recordFinancialAuditTx(tx *gorm.DB, params RecordFinancialAuditParams) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}
	beforeJSON, err := marshalFinancialAuditPayload(params.Before)
	if err != nil {
		return err
	}
	afterJSON, err := marshalFinancialAuditPayload(params.After)
	if err != nil {
		return err
	}
	createdAt := params.CreatedAt
	if createdAt == 0 {
		createdAt = common.GetTimestamp()
	}
	log := &FinancialAuditLog{
		Module:                   strings.TrimSpace(params.Module),
		Action:                   strings.TrimSpace(params.Action),
		OperatorUserId:           params.OperatorUserId,
		OperatorUsernameSnapshot: strings.TrimSpace(params.OperatorUsernameSnapshot),
		TargetType:               strings.TrimSpace(params.TargetType),
		TargetId:                 params.TargetId,
		TargetUserId:             params.TargetUserId,
		BeforeJSON:               beforeJSON,
		AfterJSON:                afterJSON,
		Remark:                   strings.TrimSpace(params.Remark),
		CreatedAt:                createdAt,
	}
	return tx.Create(log).Error
}

func marshalFinancialAuditPayload(payload any) (string, error) {
	if payload == nil {
		return "", nil
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func buildFinancialAuditRedemptionSnapshot(redemption *Redemption) financialAuditRedemptionSnapshot {
	if redemption == nil {
		return financialAuditRedemptionSnapshot{}
	}
	return financialAuditRedemptionSnapshot{
		Id:                   redemption.Id,
		CreatorUserId:        redemption.UserId,
		Status:               redemption.Status,
		Name:                 strings.TrimSpace(redemption.Name),
		FundingType:          normalizeRedemptionFundingType(redemption.FundingType),
		Quota:                redemption.Quota,
		AmountUSD:            normalizeRedemptionMoneyValue(redemption.AmountUSD),
		RecognizedRevenueUSD: normalizeRedemptionMoneyValue(redemption.RecognizedRevenueUSD),
		Remark:               strings.TrimSpace(redemption.Remark),
		QuotaPerUnitSnapshot: redemption.QuotaPerUnitSnapshot,
		CreatedTime:          redemption.CreatedTime,
		RedeemedTime:         redemption.RedeemedTime,
		UsedUserId:           redemption.UsedUserId,
		ExpiredTime:          redemption.ExpiredTime,
	}
}

func buildFinancialAuditUserQuotaSnapshot(user User) financialAuditUserQuotaSnapshot {
	return financialAuditUserQuotaSnapshot{
		Quota:     user.Quota,
		PaidQuota: user.PaidQuota,
		GiftQuota: user.GiftQuota,
	}
}

func CreateRedemptionsWithAudit(template Redemption, count int, operator FinancialAuditOperator) ([]string, error) {
	keys := make([]string, 0, count)
	createdIDs := make([]int, 0, count)
	err := DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		for i := 0; i < count; i++ {
			key := common.GetUUID()
			redemption := &Redemption{
				UserId:               operator.UserId,
				Name:                 template.Name,
				Key:                  key,
				FundingType:          template.FundingType,
				Quota:                template.Quota,
				AmountUSD:            template.AmountUSD,
				RecognizedRevenueUSD: template.RecognizedRevenueUSD,
				Remark:               template.Remark,
				QuotaPerUnitSnapshot: template.QuotaPerUnitSnapshot,
				CreatedTime:          now,
				ExpiredTime:          template.ExpiredTime,
			}
			redemption.normalizeForPersistence()
			if err := tx.Create(redemption).Error; err != nil {
				return err
			}
			keys = append(keys, key)
			createdIDs = append(createdIDs, redemption.Id)
		}

		targetType := FinancialAuditTargetRedemptionBatch
		targetID := 0
		if len(createdIDs) == 1 {
			targetType = FinancialAuditTargetRedemption
			targetID = createdIDs[0]
		}
		beforePayload := map[string]any{
			"name":                    template.Name,
			"funding_type":            template.FundingType,
			"quota":                   template.Quota,
			"amount_usd":              template.AmountUSD,
			"recognized_revenue_usd":  template.RecognizedRevenueUSD,
			"count":                   count,
			"expired_time":            template.ExpiredTime,
			"remark":                  template.Remark,
			"quota_per_unit_snapshot": template.QuotaPerUnitSnapshot,
		}
		afterPayload := map[string]any{
			"created_count":  len(createdIDs),
			"redemption_ids": createdIDs,
		}
		return recordFinancialAuditTx(tx, RecordFinancialAuditParams{
			Module:                   FinancialAuditModuleRedemption,
			Action:                   FinancialAuditActionCreate,
			OperatorUserId:           operator.UserId,
			OperatorUsernameSnapshot: operator.UsernameSnapshot,
			TargetType:               targetType,
			TargetId:                 targetID,
			Before:                   beforePayload,
			After:                    afterPayload,
			Remark:                   fmt.Sprintf("created %d redemption codes", len(createdIDs)),
			CreatedAt:                now,
		})
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func UpdateRedemptionWithAudit(payload Redemption, statusOnly bool, operator FinancialAuditOperator) (*Redemption, error) {
	var updated Redemption
	err := DB.Transaction(func(tx *gorm.DB) error {
		current := &Redemption{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", payload.Id).First(current).Error; err != nil {
			return err
		}
		current.normalizeForResponse()
		beforeSnapshot := buildFinancialAuditRedemptionSnapshot(current)

		if statusOnly {
			current.Status = payload.Status
			if err := tx.Model(current).Select("status").Updates(current).Error; err != nil {
				return err
			}
		} else {
			current.Name = payload.Name
			current.FundingType = payload.FundingType
			current.Quota = payload.Quota
			current.AmountUSD = payload.AmountUSD
			current.RecognizedRevenueUSD = payload.RecognizedRevenueUSD
			current.Remark = payload.Remark
			current.QuotaPerUnitSnapshot = payload.QuotaPerUnitSnapshot
			current.ExpiredTime = payload.ExpiredTime
			current.normalizeForPersistence()
			if err := tx.Model(current).
				Select("name", "status", "funding_type", "quota", "amount_usd", "recognized_revenue_usd", "remark", "quota_per_unit_snapshot", "redeemed_time", "expired_time").
				Updates(current).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("id = ?", current.Id).First(&updated).Error; err != nil {
			return err
		}
		updated.normalizeForResponse()
		remark := "updated redemption"
		if statusOnly {
			remark = fmt.Sprintf("updated redemption status to %d", updated.Status)
		}
		return recordFinancialAuditTx(tx, RecordFinancialAuditParams{
			Module:                   FinancialAuditModuleRedemption,
			Action:                   FinancialAuditActionUpdate,
			OperatorUserId:           operator.UserId,
			OperatorUsernameSnapshot: operator.UsernameSnapshot,
			TargetType:               FinancialAuditTargetRedemption,
			TargetId:                 updated.Id,
			Before:                   beforeSnapshot,
			After:                    buildFinancialAuditRedemptionSnapshot(&updated),
			Remark:                   remark,
		})
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func DeleteRedemptionByIdWithAudit(id int, operator FinancialAuditOperator) error {
	if id == 0 {
		return fmt.Errorf("id is empty")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		redemption := &Redemption{}
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", id).First(redemption).Error; err != nil {
			return err
		}
		redemption.normalizeForResponse()
		beforeSnapshot := buildFinancialAuditRedemptionSnapshot(redemption)
		if err := tx.Delete(redemption).Error; err != nil {
			return err
		}
		return recordFinancialAuditTx(tx, RecordFinancialAuditParams{
			Module:                   FinancialAuditModuleRedemption,
			Action:                   FinancialAuditActionDelete,
			OperatorUserId:           operator.UserId,
			OperatorUsernameSnapshot: operator.UsernameSnapshot,
			TargetType:               FinancialAuditTargetRedemption,
			TargetId:                 redemption.Id,
			Before:                   beforeSnapshot,
			Remark:                   "deleted redemption",
		})
	})
}

func DeleteInvalidRedemptionsWithAudit(operator FinancialAuditOperator) (int64, error) {
	now := common.GetTimestamp()
	var deletedCount int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		var redemptions []Redemption
		query := tx.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)",
			[]int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled},
			common.RedemptionCodeStatusEnabled,
			now,
		)
		if err := query.Find(&redemptions).Error; err != nil {
			return err
		}
		ids := make([]int, 0, len(redemptions))
		for _, redemption := range redemptions {
			ids = append(ids, redemption.Id)
		}

		result := tx.Where("status IN ? OR (status = ? AND expired_time != 0 AND expired_time < ?)",
			[]int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled},
			common.RedemptionCodeStatusEnabled,
			now,
		).Delete(&Redemption{})
		if result.Error != nil {
			return result.Error
		}
		deletedCount = result.RowsAffected

		return recordFinancialAuditTx(tx, RecordFinancialAuditParams{
			Module:                   FinancialAuditModuleRedemption,
			Action:                   FinancialAuditActionCleanupInvalid,
			OperatorUserId:           operator.UserId,
			OperatorUsernameSnapshot: operator.UsernameSnapshot,
			TargetType:               FinancialAuditTargetRedemptionBatch,
			Before: map[string]any{
				"candidate_count": len(ids),
				"redemption_ids":  ids,
			},
			After: map[string]any{
				"deleted_count":  deletedCount,
				"redemption_ids": ids,
			},
			Remark:    fmt.Sprintf("cleaned %d invalid redemption codes", deletedCount),
			CreatedAt: now,
		})
	})
	if err != nil {
		return 0, err
	}
	return deletedCount, nil
}
