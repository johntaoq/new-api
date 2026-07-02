package model

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
)

type legacyBillingChannelSnapshot struct {
	Id   int    `gorm:"column:id"`
	Name string `gorm:"column:name"`
	Type int    `gorm:"column:type"`
}

type legacyBillingUserSnapshot struct {
	Id       int    `gorm:"column:id"`
	Username string `gorm:"column:username"`
}

func loadLegacyBillingLogs(periodStart int64, periodEnd int64) ([]*Log, error) {
	var logs []*Log
	if err := LOG_DB.Where("created_at >= ? AND created_at <= ? AND type IN ?", periodStart, periodEnd, []int{LogTypeConsume, LogTypeRefund}).
		Order("created_at asc, id asc").
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func loadLegacyBillingChannelSnapshots(logs []*Log) (map[int]legacyBillingChannelSnapshot, error) {
	channelIds := make([]int, 0)
	seen := make(map[int]struct{})
	for _, logRecord := range logs {
		if logRecord == nil || logRecord.ChannelId <= 0 {
			continue
		}
		if _, ok := seen[logRecord.ChannelId]; ok {
			continue
		}
		seen[logRecord.ChannelId] = struct{}{}
		channelIds = append(channelIds, logRecord.ChannelId)
	}
	if len(channelIds) == 0 {
		return map[int]legacyBillingChannelSnapshot{}, nil
	}

	var channels []legacyBillingChannelSnapshot
	if err := DB.Table("channels").Select("id, name, type").Where("id IN ?", channelIds).Find(&channels).Error; err != nil {
		return nil, err
	}

	result := make(map[int]legacyBillingChannelSnapshot, len(channels))
	for _, channel := range channels {
		result[channel.Id] = channel
	}
	return result, nil
}

func loadLegacyBillingUserSnapshots(userIds []int) (map[int]legacyBillingUserSnapshot, error) {
	if len(userIds) == 0 {
		return map[int]legacyBillingUserSnapshot{}, nil
	}

	var users []legacyBillingUserSnapshot
	if err := DB.Unscoped().Table("users").Select("id, username").Where("id IN ?", userIds).Find(&users).Error; err != nil {
		return nil, err
	}

	result := make(map[int]legacyBillingUserSnapshot, len(users))
	for _, user := range users {
		result[user.Id] = user
	}
	return result, nil
}

func legacyBillingSource(other map[string]interface{}) string {
	source := strings.TrimSpace(getStringFromMap(other, "billing_source"))
	if source == "" {
		return "wallet"
	}
	return source
}

func legacyBillingEffectiveGroupRatio(other map[string]interface{}) float64 {
	if ratio := getPositiveFloatFromMap(other, "user_group_ratio"); ratio > 0 {
		return ratio
	}
	if ratio := getPositiveFloatFromMap(other, "group_ratio"); ratio > 0 {
		return ratio
	}
	return 1
}

func legacyBillingQuotaSplit(logRecord *Log, other map[string]interface{}) (int, int) {
	paidQuota := getNonNegativeIntFromMap(other, "paid_quota_used")
	giftQuota := getNonNegativeIntFromMap(other, "gift_quota_used")
	if logRecord == nil || logRecord.Quota <= 0 {
		return paidQuota, giftQuota
	}

	totalQuota := paidQuota + giftQuota
	if totalQuota == 0 {
		return 0, 0
	}
	if totalQuota >= logRecord.Quota {
		if giftQuota > logRecord.Quota {
			return 0, logRecord.Quota
		}
		return logRecord.Quota - giftQuota, giftQuota
	}
	return paidQuota + (logRecord.Quota - totalQuota), giftQuota
}

func legacyBillingRecognizedRevenueUSD(logRecord *Log, other map[string]interface{}) float64 {
	if logRecord == nil || logRecord.Quota <= 0 {
		return 0
	}

	paidQuota, giftQuota := legacyBillingQuotaSplit(logRecord, other)
	if paidQuota > 0 || giftQuota > 0 {
		return quotaToUSDWithSnapshot(paidQuota, common.QuotaPerUnit)
	}
	if legacyBillingSource(other) == "subscription" {
		return 0
	}
	return quotaToUSDWithSnapshot(logRecord.Quota, common.QuotaPerUnit)
}

func legacyBillingInternalEquivalentUSD(logRecord *Log) float64 {
	if logRecord == nil || logRecord.Quota <= 0 {
		return 0
	}
	return quotaToUSDWithSnapshot(logRecord.Quota, common.QuotaPerUnit)
}

func legacyBillingEstimatedCostUSD(logRecord *Log, other map[string]interface{}) float64 {
	if logRecord == nil || logRecord.Quota <= 0 {
		return 0
	}
	return estimateCostUSDFromQuota(logRecord.Quota, legacyBillingEffectiveGroupRatio(other))
}

func legacyBillingProviderSnapshot(channelId int, snapshots map[int]legacyBillingChannelSnapshot) string {
	if snapshot, ok := snapshots[channelId]; ok && snapshot.Type >= 0 {
		return constant.GetChannelTypeName(snapshot.Type)
	}
	return ""
}

func getPositiveFloatFromMap(m map[string]interface{}, key string) float64 {
	value := getFloat64FromMap(m, key)
	if value > 0 {
		return value
	}
	return 0
}

func getNonNegativeIntFromMap(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	value, ok := m[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		if typed >= 0 {
			return typed
		}
	case int32:
		if typed >= 0 {
			return int(typed)
		}
	case int64:
		if typed >= 0 {
			return int(typed)
		}
	case float32:
		if typed >= 0 {
			return int(typed)
		}
	case float64:
		if typed >= 0 {
			return int(typed)
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil && parsed >= 0 {
			return parsed
		}
	default:
		parsed, err := strconv.Atoi(strings.TrimSpace(common.Interface2String(value)))
		if err == nil && parsed >= 0 {
			return parsed
		}
	}
	return 0
}

func getFloat64FromMap(m map[string]interface{}, key string) float64 {
	if m == nil {
		return 0
	}
	value, ok := m[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed
		}
	default:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(common.Interface2String(value)), 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}
