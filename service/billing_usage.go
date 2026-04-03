package service

import "github.com/QuantumNous/new-api/dto"

func usageCacheTokenTotal(usage interface{}) int {
	switch typed := usage.(type) {
	case *dto.Usage:
		if typed == nil {
			return 0
		}
		return typed.PromptTokensDetails.CachedTokens +
			typed.PromptTokensDetails.CachedCreationTokens +
			typed.ClaudeCacheCreation5mTokens +
			typed.ClaudeCacheCreation1hTokens
	case *dto.RealtimeUsage:
		if typed == nil {
			return 0
		}
		return typed.InputTokenDetails.CachedTokens + typed.InputTokenDetails.CachedCreationTokens
	default:
		return 0
	}
}
