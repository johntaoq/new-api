package model

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type FinanceAuditLogItem struct {
	Id                       int    `json:"id"`
	Module                   string `json:"module"`
	Action                   string `json:"action"`
	OperatorUserID           int    `json:"operator_user_id"`
	OperatorUsernameSnapshot string `json:"operator_username_snapshot"`
	OperatorUsername         string `json:"operator_username"`
	TargetType               string `json:"target_type"`
	TargetID                 int    `json:"target_id"`
	TargetUserID             int    `json:"target_user_id"`
	TargetUsername           string `json:"target_username"`
	BeforeJSON               string `json:"before_json"`
	AfterJSON                string `json:"after_json"`
	Remark                   string `json:"remark"`
	CreatedAt                int64  `json:"created_at"`
}

func ListFinanceAuditLogs(periodType string, period string, module string, action string, operatorKeyword string, targetKeyword string, pageInfo *common.PageInfo) (*common.PageInfo, error) {
	periodRange, err := parseFinancePeriod(periodType, period)
	if err != nil {
		return nil, err
	}

	query := DB.Model(&FinancialAuditLog{}).
		Where("created_at >= ? AND created_at <= ?", periodRange.Start, periodRange.End)

	module = strings.TrimSpace(module)
	if module != "" {
		query = query.Where("module = ?", module)
	}

	action = strings.TrimSpace(action)
	if action != "" {
		query = query.Where("action = ?", action)
	}

	operatorKeyword = strings.TrimSpace(operatorKeyword)
	if operatorKeyword != "" {
		operatorMatchedIDs, err := financeMatchUserIDs(operatorKeyword)
		if err != nil {
			return nil, err
		}
		lowerKeyword := strings.ToLower(operatorKeyword)
		if len(operatorMatchedIDs) > 0 {
			query = query.Where("operator_user_id IN ? OR LOWER(operator_username_snapshot) LIKE ?", operatorMatchedIDs, "%"+lowerKeyword+"%")
		} else {
			query = query.Where("LOWER(operator_username_snapshot) LIKE ?", "%"+lowerKeyword+"%")
		}
	}

	targetKeyword = strings.TrimSpace(targetKeyword)
	if targetKeyword != "" {
		targetMatchedIDs, err := financeMatchUserIDs(targetKeyword)
		if err != nil {
			return nil, err
		}
		if targetUserID, parseErr := strconv.Atoi(targetKeyword); parseErr == nil && targetUserID > 0 {
			found := false
			for _, item := range targetMatchedIDs {
				if item == targetUserID {
					found = true
					break
				}
			}
			if !found {
				targetMatchedIDs = append(targetMatchedIDs, targetUserID)
			}
		}
		if len(targetMatchedIDs) == 0 {
			pageInfo.SetTotal(0)
			pageInfo.SetItems([]FinanceAuditLogItem{})
			return pageInfo, nil
		}
		query = query.Where("target_user_id IN ?", targetMatchedIDs)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var logs []FinancialAuditLog
	if err := query.Order("id DESC").Offset(pageInfo.GetStartIdx()).Limit(pageInfo.GetPageSize()).Find(&logs).Error; err != nil {
		return nil, err
	}

	operatorUserIDs := make([]int, 0, len(logs))
	targetUserIDs := make([]int, 0, len(logs))
	for _, item := range logs {
		if item.OperatorUserId > 0 {
			operatorUserIDs = append(operatorUserIDs, item.OperatorUserId)
		}
		if item.TargetUserId > 0 {
			targetUserIDs = append(targetUserIDs, item.TargetUserId)
		}
	}

	operatorMap, err := financeUserMap(operatorUserIDs)
	if err != nil {
		return nil, err
	}
	targetMap, err := financeUserMap(targetUserIDs)
	if err != nil {
		return nil, err
	}

	items := make([]FinanceAuditLogItem, 0, len(logs))
	for _, item := range logs {
		operatorName := firstNonEmpty(operatorMap[item.OperatorUserId].Username, item.OperatorUsernameSnapshot)
		targetName := firstNonEmpty(targetMap[item.TargetUserId].Username, "")
		items = append(items, FinanceAuditLogItem{
			Id:                       item.Id,
			Module:                   item.Module,
			Action:                   item.Action,
			OperatorUserID:           item.OperatorUserId,
			OperatorUsernameSnapshot: item.OperatorUsernameSnapshot,
			OperatorUsername:         operatorName,
			TargetType:               item.TargetType,
			TargetID:                 item.TargetId,
			TargetUserID:             item.TargetUserId,
			TargetUsername:           targetName,
			BeforeJSON:               item.BeforeJSON,
			AfterJSON:                item.AfterJSON,
			Remark:                   item.Remark,
			CreatedAt:                item.CreatedAt,
		})
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	return pageInfo, nil
}
