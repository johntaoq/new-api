package model

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
)

func setLogDetailLocators(logs []*Log) {
	for _, log := range logs {
		log.DetailId = log.Id
	}
}

func FormatUserLogDetail(log *Log) {
	formatUserLogs([]*Log{log}, 0)
}

// Projection follows role filtering; full audit data stays in storage.
func SummarizeLogs(logs []*Log) {
	for _, log := range logs {
		log.Summary = true
		log.Content = logPreview(log.Content, 256)
		var other map[string]json.RawMessage
		if common.UnmarshalJsonStr(log.Other, &other) != nil {
			log.Other = "{}"
			continue
		}
		result := make(map[string]json.RawMessage)
		for _, key := range strings.Fields(`model_ratio completion_ratio model_price group_ratio user_group_ratio
			cache_ratio cache_creation_ratio cache_creation_ratio_5m cache_creation_ratio_1h
			cache_tokens image_cache_tokens cache_creation_tokens cache_creation_tokens_5m cache_creation_tokens_1h
			audio audio_input audio_output text_input text_output audio_ratio audio_completion_ratio
			billing_mode billing_unit fixed_price matched_tier reasoning_effort frt
			is_model_mapped upstream_model_name response_model is_system_prompt_overwritten
			billing_source group subscription_plan_id subscription_plan_title subscription_id subscription_consumed
			image image_ratio image_output image_count web_search web_search_call_count web_search_price
			file_search file_search_call_count file_search_price tool_surcharges audio_input_seperate_price
			audio_input_token_count audio_input_price image_generation_call image_generation_call_price
			image_generation_call_count violation_fee violation_fee_code violation_fee_marker fee_quota
			is_task task_sync result_discarded task_id`) {
			if value := other[key]; len(value) > 0 && len(value) <= 1024 {
				result[key] = value
			}
		}
		for _, scoped := range []struct{ key, fields string }{
			{"admin_info", "use_channel is_multi_key multi_key_index quota_saturation channel_affinity task_plugin"},
			{"stream_status", "status end_reason error_count"},
			{"op", "action params"},
			{"audit_info", "success route"},
		} {
			var nested map[string]json.RawMessage
			if common.Unmarshal(other[scoped.key], &nested) != nil {
				continue
			}
			selected := make(map[string]json.RawMessage)
			for _, field := range strings.Fields(scoped.fields) {
				if value := nested[field]; len(value) > 0 && len(value) <= 1024 {
					selected[field] = value
				}
			}
			if len(selected) > 0 {
				if value, err := common.Marshal(selected); err == nil {
					result[scoped.key] = value
				}
			}
		}
		if parts := other["content_parts"]; len(parts) > 0 && len(parts) <= 1024 {
			result["content_parts"] = parts
		}
		if value, err := common.Marshal(result); err == nil {
			log.Other = string(value)
		} else {
			log.Other = "{}"
		}
	}
}

func logPreview(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit]) + "..."
}

// Ownership and time-range restrictions are applied before fetching the row.
// ClickHouse locates records by request_id, never by synthetic display ids.
func GetLogDetail(ctx context.Context, id int, filter LogFilter, userId int) (*Log, error) {
	tx := LOG_DB.WithContext(ctx).Model(&Log{})
	if userId > 0 {
		tx = tx.Where("logs.user_id = ?", userId)
	}
	if id > 0 && !common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		tx = tx.Where("logs.id = ?", id)
	}
	tx, err := filter.apply(tx)
	if err != nil {
		return nil, err
	}
	var log Log
	if err := tx.Take(&log).Error; err != nil {
		return nil, err
	}
	if userId == 0 && log.ChannelId != 0 {
		if common.MemoryCacheEnabled {
			if channel, err := CacheGetChannel(log.ChannelId); err == nil {
				log.ChannelName = channel.Name
			}
		} else {
			var channel Channel
			if err := DB.Select("name").Where("id = ?", log.ChannelId).Find(&channel).Error; err != nil {
				return nil, err
			}
			log.ChannelName = channel.Name
		}
	}
	return &log, nil
}
