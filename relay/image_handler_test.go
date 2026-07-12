package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestApplyImageUsageFallbackUsesEstimatedOutputTokens(t *testing.T) {
	imageN := uint(2)
	request := &dto.ImageRequest{
		Model:  "MAI-Image-2",
		Prompt: "draw a small red house",
		N:      &imageN,
	}
	usage := &dto.Usage{}

	applyImageUsageFallback(usage, request)

	if usage.PromptTokens != 1 {
		t.Fatalf("PromptTokens = %d, want 1", usage.PromptTokens)
	}
	if usage.CompletionTokens != 3168 {
		t.Fatalf("CompletionTokens = %d, want 3168", usage.CompletionTokens)
	}
	if usage.TotalTokens != 3169 {
		t.Fatalf("TotalTokens = %d, want 3169", usage.TotalTokens)
	}
	if usage.OutputTokens != 3168 {
		t.Fatalf("OutputTokens = %d, want 3168", usage.OutputTokens)
	}
	if usage.CompletionTokenDetails.ImageTokens != 3168 {
		t.Fatalf("CompletionTokenDetails.ImageTokens = %d, want 3168", usage.CompletionTokenDetails.ImageTokens)
	}
}

func TestApplyImageUsageFallbackPrefersUpstreamNumOutputTokens(t *testing.T) {
	imageN := uint(2)
	request := &dto.ImageRequest{
		Model:  "MAI-Image-2",
		Prompt: "draw a small red house",
		N:      &imageN,
	}
	usage := &dto.Usage{
		NumOutputTokens: 1024,
	}

	applyImageUsageFallback(usage, request)

	if usage.PromptTokens != 1 {
		t.Fatalf("PromptTokens = %d, want 1", usage.PromptTokens)
	}
	if usage.CompletionTokens != 1024 {
		t.Fatalf("CompletionTokens = %d, want 1024", usage.CompletionTokens)
	}
	if usage.TotalTokens != 1025 {
		t.Fatalf("TotalTokens = %d, want 1025", usage.TotalTokens)
	}
	if usage.OutputTokens != 1024 {
		t.Fatalf("OutputTokens = %d, want 1024", usage.OutputTokens)
	}
	if usage.CompletionTokenDetails.ImageTokens != 1024 {
		t.Fatalf("CompletionTokenDetails.ImageTokens = %d, want 1024", usage.CompletionTokenDetails.ImageTokens)
	}
}

func TestApplyImageUsageFallbackKeepsUpstreamUsage(t *testing.T) {
	imageN := uint(3)
	request := &dto.ImageRequest{
		Model:  "MAI-Image-2",
		Prompt: "draw a small red house",
		N:      &imageN,
	}
	usage := &dto.Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		OutputTokens:     20,
		CompletionTokenDetails: dto.OutputTokenDetails{
			ImageTokens: 20,
		},
	}

	applyImageUsageFallback(usage, request)

	if usage.PromptTokens != 10 {
		t.Fatalf("PromptTokens = %d, want 10", usage.PromptTokens)
	}
	if usage.CompletionTokens != 20 {
		t.Fatalf("CompletionTokens = %d, want 20", usage.CompletionTokens)
	}
	if usage.TotalTokens != 30 {
		t.Fatalf("TotalTokens = %d, want 30", usage.TotalTokens)
	}
	if usage.OutputTokens != 20 {
		t.Fatalf("OutputTokens = %d, want 20", usage.OutputTokens)
	}
	if usage.CompletionTokenDetails.ImageTokens != 20 {
		t.Fatalf("CompletionTokenDetails.ImageTokens = %d, want 20", usage.CompletionTokenDetails.ImageTokens)
	}
}
