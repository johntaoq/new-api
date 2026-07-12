package openai

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func applyUsagePostProcessing(info *relaycommon.RelayInfo, usage *dto.Usage, responseBody []byte) {
	if info == nil || usage == nil {
		return
	}

	if info.RelayMode == relayconstant.RelayModeImagesGenerations || info.RelayMode == relayconstant.RelayModeImagesEdits {
		applyOpenAIImageTopLevelUsage(usage, responseBody)
	}
	usage.NormalizeCacheWriteTokens()

	switch info.ChannelType {
	case constant.ChannelTypeDeepSeek:
		if usage.PromptTokensDetails.CachedTokens == 0 && usage.PromptCacheHitTokens != 0 {
			usage.PromptTokensDetails.CachedTokens = usage.PromptCacheHitTokens
		}
	case constant.ChannelTypeZhipu_v4:
		// 智普的cached_tokens在标准位置: usage.prompt_tokens_details.cached_tokens
		if usage.PromptTokensDetails.CachedTokens == 0 {
			if usage.InputTokensDetails != nil && usage.InputTokensDetails.CachedTokens > 0 {
				usage.PromptTokensDetails.CachedTokens = usage.InputTokensDetails.CachedTokens
			} else if cachedTokens, ok := extractCachedTokensFromBody(responseBody); ok {
				usage.PromptTokensDetails.CachedTokens = cachedTokens
			} else if usage.PromptCacheHitTokens > 0 {
				usage.PromptTokensDetails.CachedTokens = usage.PromptCacheHitTokens
			}
		}
	case constant.ChannelTypeMoonshot:
		// Moonshot的cached_tokens在非标准位置: choices[].usage.cached_tokens
		if usage.PromptTokensDetails.CachedTokens == 0 {
			if usage.InputTokensDetails != nil && usage.InputTokensDetails.CachedTokens > 0 {
				usage.PromptTokensDetails.CachedTokens = usage.InputTokensDetails.CachedTokens
			} else if cachedTokens, ok := extractMoonshotCachedTokensFromBody(responseBody); ok {
				usage.PromptTokensDetails.CachedTokens = cachedTokens
			} else if cachedTokens, ok := extractCachedTokensFromBody(responseBody); ok {
				usage.PromptTokensDetails.CachedTokens = cachedTokens
			} else if usage.PromptCacheHitTokens > 0 {
				usage.PromptTokensDetails.CachedTokens = usage.PromptCacheHitTokens
			}
		}
	case constant.ChannelTypeOpenAI:
		if usage.PromptTokensDetails.CachedTokens == 0 {
			if cachedTokens, ok := extractLlamaCachedTokensFromBody(responseBody); ok {
				usage.PromptTokensDetails.CachedTokens = cachedTokens
			}
		}
	}
}

func applyOpenAIImageTopLevelUsage(usage *dto.Usage, body []byte) {
	if usage == nil || len(body) == 0 {
		return
	}

	var payload struct {
		InputTokens         *int `json:"input_tokens"`
		OutputTokens        *int `json:"output_tokens"`
		NumInputTextTokens  *int `json:"num_input_text_tokens"`
		NumInputImageTokens *int `json:"num_input_image_tokens"`
		NumOutputTokens     *int `json:"num_output_tokens"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return
	}

	if usage.InputTokens == 0 {
		if inputTokens := firstPositiveInt(payload.InputTokens); inputTokens > 0 {
			usage.InputTokens = inputTokens
		}
	}
	if usage.NumInputTextTokens == 0 {
		if textTokens := firstPositiveInt(payload.NumInputTextTokens); textTokens > 0 {
			usage.NumInputTextTokens = textTokens
		}
	}
	if usage.NumInputImageTokens == 0 {
		if imageTokens := firstPositiveInt(payload.NumInputImageTokens); imageTokens > 0 {
			usage.NumInputImageTokens = imageTokens
		}
	}
	if usage.PromptTokens == 0 {
		if usage.InputTokens > 0 {
			usage.PromptTokens = usage.InputTokens
		} else if usage.NumInputTextTokens > 0 || usage.NumInputImageTokens > 0 {
			usage.PromptTokens = usage.NumInputTextTokens + usage.NumInputImageTokens
		}
	}
	if usage.PromptTokensDetails.TextTokens == 0 && usage.NumInputTextTokens > 0 {
		usage.PromptTokensDetails.TextTokens = usage.NumInputTextTokens
	}
	if usage.PromptTokensDetails.ImageTokens == 0 && usage.NumInputImageTokens > 0 {
		usage.PromptTokensDetails.ImageTokens = usage.NumInputImageTokens
	}

	outputTokens := firstPositiveInt(payload.OutputTokens, payload.NumOutputTokens)
	if outputTokens > 0 {
		if usage.OutputTokens == 0 {
			usage.OutputTokens = outputTokens
		}
		if usage.NumOutputTokens == 0 {
			usage.NumOutputTokens = outputTokens
		}
		if usage.CompletionTokens == 0 {
			usage.CompletionTokens = outputTokens
		}
		if usage.CompletionTokenDetails.ImageTokens == 0 {
			usage.CompletionTokenDetails.ImageTokens = outputTokens
		}
	}

	totalTokens := usage.PromptTokens + usage.CompletionTokens
	if totalTokens > 0 && (usage.TotalTokens == 0 || usage.TotalTokens < totalTokens) {
		usage.TotalTokens = totalTokens
	}
}

func firstPositiveInt(values ...*int) int {
	for _, value := range values {
		if value != nil && *value > 0 {
			return *value
		}
	}
	return 0
}

func copyInputTokenDetailsToPromptDetails(usage *dto.Usage, details *dto.InputTokenDetails) {
	if usage == nil || details == nil {
		return
	}
	details.NormalizeCacheWriteTokens()
	usage.PromptTokensDetails.CachedTokens = details.CachedTokens
	usage.PromptTokensDetails.CachedCreationTokens = details.CachedCreationTokens
	usage.PromptTokensDetails.CacheWriteTokens = details.CacheWriteTokens
	usage.PromptTokensDetails.ImageTokens = details.ImageTokens
	usage.PromptTokensDetails.TextTokens = details.TextTokens
	usage.PromptTokensDetails.AudioTokens = details.AudioTokens
}

func extractCachedTokensFromBody(body []byte) (int, bool) {
	if len(body) == 0 {
		return 0, false
	}

	var payload struct {
		Usage struct {
			PromptTokensDetails struct {
				CachedTokens *int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
			CachedTokens         *int `json:"cached_tokens"`
			PromptCacheHitTokens *int `json:"prompt_cache_hit_tokens"`
		} `json:"usage"`
	}

	if err := common.Unmarshal(body, &payload); err != nil {
		return 0, false
	}

	if payload.Usage.PromptTokensDetails.CachedTokens != nil {
		return *payload.Usage.PromptTokensDetails.CachedTokens, true
	}
	if payload.Usage.CachedTokens != nil {
		return *payload.Usage.CachedTokens, true
	}
	if payload.Usage.PromptCacheHitTokens != nil {
		return *payload.Usage.PromptCacheHitTokens, true
	}
	return 0, false
}

// extractMoonshotCachedTokensFromBody 从Moonshot的非标准位置提取cached_tokens
// Moonshot的流式响应格式: {"choices":[{"usage":{"cached_tokens":111}}]}
func extractMoonshotCachedTokensFromBody(body []byte) (int, bool) {
	if len(body) == 0 {
		return 0, false
	}

	var payload struct {
		Choices []struct {
			Usage struct {
				CachedTokens *int `json:"cached_tokens"`
			} `json:"usage"`
		} `json:"choices"`
	}

	if err := common.Unmarshal(body, &payload); err != nil {
		return 0, false
	}

	// 遍历choices查找cached_tokens
	for _, choice := range payload.Choices {
		if choice.Usage.CachedTokens != nil && *choice.Usage.CachedTokens > 0 {
			return *choice.Usage.CachedTokens, true
		}
	}

	return 0, false
}

// extractLlamaCachedTokensFromBody 从llama.cpp的非标准位置提取cache_n
func extractLlamaCachedTokensFromBody(body []byte) (int, bool) {
	if len(body) == 0 {
		return 0, false
	}

	var payload struct {
		Timings struct {
			CachedTokens *int `json:"cache_n"`
		} `json:"timings"`
	}

	if err := common.Unmarshal(body, &payload); err != nil {
		return 0, false
	}

	if payload.Timings.CachedTokens == nil {
		return 0, false
	}
	return *payload.Timings.CachedTokens, true
}
