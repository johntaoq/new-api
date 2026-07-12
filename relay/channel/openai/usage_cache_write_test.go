package openai

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestApplyUsagePostProcessingMapsCacheWriteTokens(t *testing.T) {
	usage := &dto.Usage{
		PromptTokens:     100000,
		CompletionTokens: 1000,
		TotalTokens:      101000,
		PromptTokensDetails: dto.InputTokenDetails{
			CachedTokens:     60000,
			CacheWriteTokens: 30000,
		},
	}

	applyUsagePostProcessing(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI},
	}, usage, nil)

	require.Equal(t, 60000, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 30000, usage.PromptTokensDetails.CacheWriteTokens)
	require.Equal(t, 30000, usage.PromptTokensDetails.CachedCreationTokens)
}

func TestCopyInputTokenDetailsToPromptDetailsMapsCacheWriteTokens(t *testing.T) {
	usage := &dto.Usage{}
	copyInputTokenDetailsToPromptDetails(usage, &dto.InputTokenDetails{
		CachedTokens:     60,
		CacheWriteTokens: 30,
		TextTokens:       10,
	})

	require.Equal(t, 60, usage.PromptTokensDetails.CachedTokens)
	require.Equal(t, 30, usage.PromptTokensDetails.CacheWriteTokens)
	require.Equal(t, 30, usage.PromptTokensDetails.CachedCreationTokens)
	require.Equal(t, 10, usage.PromptTokensDetails.TextTokens)
}
