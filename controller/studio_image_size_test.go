package controller

import "testing"

func TestStudioImageSizeConstraintsForMaiImage(t *testing.T) {
	config := studioImageConfigForModel("MAI-Image-2.5")

	validSizes := []string{
		"1024x1024",
		"768x1365",
		"1365x768",
	}
	for _, size := range validSizes {
		if !config.supportsSize(size) {
			t.Fatalf("expected MAI image size %s to be supported", size)
		}
	}

	invalidSizes := []string{
		"767x1024",
		"1024x767",
		"768x1366",
	}
	for _, size := range invalidSizes {
		if config.supportsSize(size) {
			t.Fatalf("expected MAI image size %s to be rejected", size)
		}
	}
}

func TestStudioImageSizeConstraintsForGPTImage2(t *testing.T) {
	config := studioImageConfigForModel("gpt-image-2")

	validSizes := []string{
		"1024x1024",
		"1280x512",
		"3840x2160",
		"2160x3840",
	}
	for _, size := range validSizes {
		if !config.supportsSize(size) {
			t.Fatalf("expected GPT Image 2 size %s to be supported", size)
		}
	}

	invalidSizes := []string{
		"1008x640",
		"1025x1024",
		"3856x2160",
		"3840x2176",
		"3840x1264",
		"1264x3840",
	}
	for _, size := range invalidSizes {
		if config.supportsSize(size) {
			t.Fatalf("expected GPT Image 2 size %s to be rejected", size)
		}
	}
}
