package controller

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type studioSSOExchangeRequest struct {
	Ticket   string `json:"ticket"`
	State    string `json:"state"`
	Audience string `json:"audience"`
	Nonce    string `json:"nonce"`
}

type studioUserScopedRequest struct {
	UserId int `json:"user_id"`
}

type studioImageReservationRequest struct {
	UserId     int    `json:"user_id"`
	RequestId  string `json:"request_id"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Size       string `json:"size"`
	Quality    string `json:"quality"`
	N          int    `json:"n"`
	PromptHash string `json:"prompt_hash"`
}

type studioImageReservationActionRequest struct {
	UserId      int        `json:"user_id"`
	JobId       string     `json:"job_id"`
	FinalCost   int        `json:"final_cost"`
	OutputCount int        `json:"output_count"`
	Usage       *dto.Usage `json:"usage"`
}

type studioImageAuditRequest struct {
	UserId        int                    `json:"user_id"`
	ReservationId string                 `json:"reservation_id"`
	JobId         string                 `json:"job_id"`
	EventType     string                 `json:"event_type"`
	Provider      string                 `json:"provider"`
	Model         string                 `json:"model"`
	Amount        int                    `json:"amount"`
	Status        string                 `json:"status"`
	ErrorCode     string                 `json:"error_code"`
	LatencyMs     int64                  `json:"latency_ms"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type studioImageUpstreamSelection struct {
	Provider      string
	BaseURL       string
	APIKey        string
	APIVersion    string
	Deployment    string
	UpstreamModel string
	UpstreamRef   string
	ChannelId     int
	ChannelName   string
	ChannelType   int
}

func getStudioOpenWebUIURL() string {
	return strings.TrimRight(common.GetEnvOrDefaultString("STUDIO_OPEN_WEBUI_URL", "https://studio.unikeyx.com"), "/")
}

func getStudioInternalSecret() string {
	secret := os.Getenv("STUDIO_INTERNAL_SECRET")
	if secret == "" {
		secret = common.SessionSecret
	}
	return secret
}

func getStudioTicketTTLSeconds() int64 {
	ttl, err := strconv.Atoi(os.Getenv("STUDIO_SSO_TICKET_TTL_SECONDS"))
	if err != nil || ttl <= 0 {
		return 120
	}
	return int64(ttl)
}

func getStudioChatKeyTTLSeconds() int64 {
	ttl, err := strconv.Atoi(os.Getenv("STUDIO_CHAT_KEY_TTL_SECONDS"))
	if err != nil || ttl <= 0 {
		return 24 * 60 * 60
	}
	return int64(ttl)
}

func getStudioImageReservationTTLSeconds() int64 {
	ttl, err := strconv.Atoi(os.Getenv("STUDIO_IMAGE_RESERVATION_TTL_SECONDS"))
	if err != nil || ttl <= 0 {
		return 20 * 60
	}
	return int64(ttl)
}

func getStudioImageCredentialTTLSeconds() int64 {
	ttl, err := strconv.Atoi(os.Getenv("STUDIO_IMAGE_CREDENTIAL_TTL_SECONDS"))
	if err != nil || ttl <= 0 {
		return 15 * 60
	}
	return int64(ttl)
}

func getStudioImagePreconsumeQuota() int {
	quota, err := strconv.Atoi(os.Getenv("STUDIO_IMAGE_PRECONSUME_QUOTA"))
	if err != nil || quota <= 0 {
		return 250000
	}
	return quota
}

func studioImageUserGroup(user *model.User) string {
	if user == nil || user.Group == "" {
		return "default"
	}
	return user.Group
}

func studioImageRequestForPricing(modelName string, size string, quality string, n int) *dto.ImageRequest {
	if n <= 0 {
		n = 1
	}
	nValue := uint(n)
	return &dto.ImageRequest{
		Model:   modelName,
		Size:    size,
		Quality: quality,
		N:       &nValue,
	}
}

func buildStudioImageRelayInfo(
	c *gin.Context,
	user *model.User,
	requestId string,
	imageRequest *dto.ImageRequest,
	upstream *studioImageUpstreamSelection,
) *relaycommon.RelayInfo {
	group := studioImageUserGroup(user)
	now := time.Now()
	relayInfo := &relaycommon.RelayInfo{
		Request:           imageRequest,
		RequestId:         requestId,
		UserId:            user.Id,
		UsingGroup:        group,
		UserGroup:         group,
		UserQuota:         user.Quota,
		UserEmail:         user.Email,
		OriginModelName:   imageRequest.Model,
		TokenGroup:        group,
		StartTime:         now,
		FirstResponseTime: now,
		RequestURLPath:    "/api/studio/images/generations",
		UserSetting:       user.GetSetting(),
		ForcePreConsume:   true,
		RelayFormat:       types.RelayFormatOpenAIImage,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:            upstream.ChannelId,
			ChannelType:          upstream.ChannelType,
			ChannelBaseUrl:       upstream.BaseURL,
			UpstreamModelName:    upstream.UpstreamModel,
			SupportStreamOptions: false,
		},
	}

	if c != nil {
		c.Set("token_name", "AI Studio Image generation")
		c.Set(string(constant.ContextKeyUserId), user.Id)
		c.Set(string(constant.ContextKeyUserGroup), group)
		c.Set(string(constant.ContextKeyUsingGroup), group)
		c.Set(string(constant.ContextKeyOriginalModel), imageRequest.Model)
		c.Set(string(constant.ContextKeyUserQuota), user.Quota)
		c.Set(string(constant.ContextKeyUserEmail), user.Email)
		c.Set(string(constant.ContextKeyChannelId), upstream.ChannelId)
		c.Set(string(constant.ContextKeyChannelName), upstream.ChannelName)
		c.Set(string(constant.ContextKeyChannelType), upstream.ChannelType)
		c.Set(string(constant.ContextKeyChannelBaseUrl), upstream.BaseURL)
		c.Set(common.RequestIdKey, requestId)
	}
	return relayInfo
}

func priceStudioImageRequest(
	c *gin.Context,
	user *model.User,
	requestId string,
	imageRequest *dto.ImageRequest,
	upstream *studioImageUpstreamSelection,
) (*relaycommon.RelayInfo, types.PriceData, error) {
	relayInfo := buildStudioImageRelayInfo(c, user, requestId, imageRequest, upstream)
	meta := imageRequest.GetTokenCountMeta()
	priceData, err := helper.ModelPriceHelper(c, relayInfo, 0, meta)
	if err != nil {
		return nil, types.PriceData{}, err
	}
	return relayInfo, priceData, nil
}

func studioImageChannelIdFromRef(upstreamRef string) int {
	if !strings.HasPrefix(upstreamRef, "channel:") {
		return 0
	}
	channelId, err := strconv.Atoi(strings.TrimPrefix(upstreamRef, "channel:"))
	if err != nil || channelId < 0 {
		return 0
	}
	return channelId
}

func studioImageReservationAllocations(reservation *model.StudioImageReservation) []types.QuotaFundingAllocation {
	if reservation == nil || reservation.AllocationsJSON == "" {
		return nil
	}
	var allocations []types.QuotaFundingAllocation
	_ = json.Unmarshal([]byte(reservation.AllocationsJSON), &allocations)
	return allocations
}

func studioImageUsageHasActualTokens(usage *dto.Usage) bool {
	return usage != nil &&
		(usage.PromptTokens > 0 ||
			usage.CompletionTokens > 0 ||
			usage.TotalTokens > 0 ||
			usage.InputTokens > 0 ||
			usage.OutputTokens > 0 ||
			usage.NumInputTextTokens > 0 ||
			usage.NumInputImageTokens > 0 ||
			usage.NumOutputTokens > 0 ||
			usage.PromptTokensDetails.TextTokens > 0 ||
			usage.PromptTokensDetails.ImageTokens > 0)
}

func normalizeStudioImageUsage(usage *dto.Usage, outputCount int) *dto.Usage {
	if usage == nil {
		usage = &dto.Usage{}
	}
	normalized := *usage
	maiInputTokens := normalized.NumInputTextTokens + normalized.NumInputImageTokens
	if normalized.NumInputTextTokens > 0 && normalized.PromptTokensDetails.TextTokens == 0 {
		normalized.PromptTokensDetails.TextTokens = normalized.NumInputTextTokens
	}
	if normalized.NumInputImageTokens > 0 && normalized.PromptTokensDetails.ImageTokens == 0 {
		normalized.PromptTokensDetails.ImageTokens = normalized.NumInputImageTokens
	}
	if normalized.InputTokens == 0 && maiInputTokens > 0 {
		normalized.InputTokens = maiInputTokens
	}
	if normalized.OutputTokens == 0 && normalized.NumOutputTokens > 0 {
		normalized.OutputTokens = normalized.NumOutputTokens
	}
	if normalized.PromptTokens == 0 && normalized.InputTokens > 0 {
		normalized.PromptTokens = normalized.InputTokens
	}
	if normalized.CompletionTokens == 0 && normalized.OutputTokens > 0 {
		normalized.CompletionTokens = normalized.OutputTokens
	}
	if normalized.InputTokensDetails != nil {
		if normalized.PromptTokensDetails.CachedTokens == 0 {
			normalized.PromptTokensDetails.CachedTokens = normalized.InputTokensDetails.CachedTokens
		}
		if normalized.PromptTokensDetails.ImageTokens == 0 {
			normalized.PromptTokensDetails.ImageTokens = normalized.InputTokensDetails.ImageTokens
		}
		if normalized.PromptTokensDetails.TextTokens == 0 {
			normalized.PromptTokensDetails.TextTokens = normalized.InputTokensDetails.TextTokens
		}
		if normalized.PromptTokensDetails.AudioTokens == 0 {
			normalized.PromptTokensDetails.AudioTokens = normalized.InputTokensDetails.AudioTokens
		}
	}
	inputDetailTokens := normalized.PromptTokensDetails.TextTokens +
		normalized.PromptTokensDetails.ImageTokens +
		normalized.PromptTokensDetails.AudioTokens
	if normalized.InputTokens == 0 && inputDetailTokens > 0 {
		normalized.InputTokens = inputDetailTokens
	}
	if normalized.PromptTokens == 0 && inputDetailTokens > 0 {
		normalized.PromptTokens = inputDetailTokens
	}
	if normalized.PromptTokens == 0 && normalized.CompletionTokens == 0 && normalized.TotalTokens > 0 {
		normalized.PromptTokens = normalized.TotalTokens
	}
	if normalized.TotalTokens == 0 {
		normalized.TotalTokens = normalized.PromptTokens + normalized.CompletionTokens
	}
	if normalized.TotalTokens == 0 && outputCount > 0 {
		normalized.PromptTokens = outputCount
		normalized.TotalTokens = outputCount
	}
	return &normalized
}

func calculateStudioImageFinalCost(
	c *gin.Context,
	reservation *model.StudioImageReservation,
	relayInfo *relaycommon.RelayInfo,
	priceData types.PriceData,
	usage *dto.Usage,
	outputCount int,
) (int, *dto.Usage) {
	normalizedUsage := normalizeStudioImageUsage(usage, outputCount)
	if !priceData.UsePrice && !studioImageUsageHasActualTokens(usage) {
		return reservation.EstimatedCost, normalizedUsage
	}
	relayInfo.PriceData = priceData
	finalCost := service.CalculateTextQuota(c, relayInfo, normalizedUsage)
	if finalCost <= 0 && !priceData.FreeModel && normalizedUsage.TotalTokens > 0 {
		return 1, normalizedUsage
	}
	return finalCost, normalizedUsage
}

func studioImageEstimatedCostUSD(quota int, groupRatio float64) float64 {
	if quota <= 0 || common.QuotaPerUnit <= 0 {
		return 0
	}
	if groupRatio <= 0 {
		groupRatio = 1
	}
	return float64(quota) / common.QuotaPerUnit / groupRatio
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func recordStudioImageConsume(
	c *gin.Context,
	reservation *model.StudioImageReservation,
	jobId string,
	quota int,
	usage *dto.Usage,
	relayInfo *relaycommon.RelayInfo,
	priceData types.PriceData,
	outputCount int,
) {
	if reservation == nil || quota <= 0 {
		return
	}
	channelId := studioImageChannelIdFromRef(reservation.UpstreamRef)
	channelType := 0
	channelName := ""
	if channelId > 0 {
		if channel, err := model.GetChannelById(channelId, true); err == nil && channel != nil {
			channelType = channel.Type
			channelName = channel.Name
		}
	}
	group, err := model.GetUserGroup(reservation.UserId, false)
	if err != nil || group == "" {
		group = "default"
	}

	model.UpdateUserUsedQuotaAndRequestCount(reservation.UserId, quota)
	if channelId > 0 {
		model.UpdateChannelUsedQuota(channelId, quota)
	}

	paidQuotaUsed, giftQuotaUsed, _ := model.SumQuotaFundingAllocations(studioImageReservationAllocations(reservation))
	other := service.GenerateTextOtherInfo(
		c,
		relayInfo,
		priceData.ModelRatio,
		priceData.GroupRatioInfo.GroupRatio,
		priceData.CompletionRatio,
		usage.PromptTokensDetails.CachedTokens,
		priceData.CacheRatio,
		priceData.ModelPrice,
		priceData.GroupRatioInfo.GroupSpecialRatio,
	)
	other["source"] = "ai_studio"
	other["reservation_id"] = reservation.Id
	other["job_id"] = jobId
	other["provider"] = reservation.Provider
	other["size"] = reservation.Size
	other["quality"] = reservation.Quality
	other["n"] = reservation.N
	other["upstream_ref"] = reservation.UpstreamRef
	other["estimated_cost"] = reservation.EstimatedCost
	other["final_cost"] = quota
	other["quota_type"] = func() int {
		if priceData.UsePrice {
			return 1
		}
		return 0
	}()
	other["billing_source"] = "wallet"
	other["prompt_tokens"] = usage.PromptTokens
	other["completion_tokens"] = usage.CompletionTokens
	other["total_tokens"] = usage.TotalTokens
	other["input_tokens"] = usage.InputTokens
	other["output_tokens"] = usage.OutputTokens
	if usage.PromptTokensDetails.TextTokens > 0 {
		other["input_text_tokens"] = usage.PromptTokensDetails.TextTokens
	}
	if usage.PromptTokensDetails.ImageTokens > 0 {
		other["input_image_tokens"] = usage.PromptTokensDetails.ImageTokens
		other["image_ratio"] = priceData.ImageRatio
	}
	other["output_count"] = outputCount
	other["paid_quota_used"] = paidQuotaUsed
	other["gift_quota_used"] = giftQuotaUsed
	other["request_path"] = "/api/studio/images/generations"

	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:           reservation.UserId,
		LogType:          model.LogTypeConsume,
		Content:          fmt.Sprintf("AI Studio image generation: size %s, quality %s, n %d", reservation.Size, reservation.Quality, reservation.N),
		ChannelId:        channelId,
		ModelName:        reservation.Model,
		Quota:            quota,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TokenName:        "AI Studio Image generation",
		UseTimeSeconds:   int(time.Now().Unix() - reservation.CreatedAt),
		Group:            group,
		RequestId:        firstNonEmpty(jobId, reservation.RequestId, reservation.Id),
		Other:            other,
	})

	_ = model.RecordChannelCostLedger(model.RecordChannelCostLedgerParams{
		RequestId:         firstNonEmpty(jobId, reservation.RequestId, reservation.Id),
		UserId:            reservation.UserId,
		BillingSource:     "wallet",
		ChannelId:         channelId,
		ChannelType:       channelType,
		ChannelName:       channelName,
		OriginModelName:   reservation.Model,
		UpstreamModelName: reservation.Model,
		EntryType:         model.ChannelCostEntryTypeConsume,
		PromptTokens:      usage.PromptTokens,
		CompletionTokens:  usage.CompletionTokens,
		ActualQuota:       quota,
		EstimatedCostUSD:  studioImageEstimatedCostUSD(quota, priceData.GroupRatioInfo.GroupRatio),
		CostBasis:         "studio_image_new_api_pricing",
		OccurredAt:        common.GetTimestamp(),
		Allocations:       studioImageReservationAllocations(reservation),
	})
}

func studioImageAuditAmount(req studioImageAuditRequest, reservation *model.StudioImageReservation) int {
	if req.Amount > 0 || reservation == nil {
		return req.Amount
	}
	switch req.EventType {
	case "succeeded":
		if reservation.Status == model.StudioImageReservationStatusCommitted {
			return reservation.FinalCost
		}
		return reservation.EstimatedCost
	default:
		return 0
	}
}

func currentSessionUserId(c *gin.Context) (int, error) {
	session := sessions.Default(c)
	id := session.Get("id")
	status := session.Get("status")
	if id == nil {
		return 0, errors.New("unauthorized")
	}
	if statusInt, ok := status.(int); ok && statusInt == common.UserStatusDisabled {
		return 0, errors.New("user is disabled")
	}
	switch value := id.(type) {
	case int:
		return value, nil
	case int64:
		return int(value), nil
	case float64:
		return int(value), nil
	case string:
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0, errors.New("invalid session user id")
		}
		return parsed, nil
	default:
		return 0, errors.New("invalid session user id")
	}
}

func currentStudioUserId(c *gin.Context) (int, error) {
	userId, err := currentSessionUserId(c)
	if err == nil {
		return userId, nil
	}

	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader == "" {
		return 0, err
	}

	user := model.ValidateAccessToken(authHeader)
	if user == nil || user.Id == 0 {
		return 0, errors.New("unauthorized")
	}
	if user.Status == common.UserStatusDisabled {
		return 0, errors.New("user is disabled")
	}
	return user.Id, nil
}

func shouldReturnStudioOpenJSON(c *gin.Context) bool {
	if strings.EqualFold(strings.TrimSpace(c.Query("format")), "json") {
		return true
	}
	return strings.Contains(strings.ToLower(c.GetHeader("Accept")), "application/json")
}

func validateStudioInternalRequest(c *gin.Context) bool {
	secret := getStudioInternalSecret()
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		authHeader = strings.TrimSpace(authHeader[7:])
	}
	if secret == "" || authHeader == "" || authHeader != secret {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "unauthorized",
		})
		return false
	}
	return true
}

func roleNameForStudio(role int) string {
	if role >= common.RoleAdminUser {
		return "admin"
	}
	return "user"
}

func endpointTypesContain(endpointTypes []constant.EndpointType, target constant.EndpointType) bool {
	for _, endpointType := range endpointTypes {
		if endpointType == target {
			return true
		}
	}
	return false
}

type studioImageSizeConstraints struct {
	MinWidth  int `json:"min_width"`
	MinHeight int `json:"min_height"`
	MaxPixels int `json:"max_pixels"`
}

type studioImageModelConfig struct {
	Sizes              []string
	Quality            []string
	SupportsQuality    bool
	SupportsN          bool
	MaxN               int
	RequestShape       string
	SizeMode           string
	SizeConstraints    *studioImageSizeConstraints
	SupportsImageEdit  bool
	MaxReferenceImages int
	ReferenceMimeTypes []string
	ReferenceFieldName string
	EditPayloadShape   string
}

func compactStudioModelName(modelName string) string {
	return strings.NewReplacer("-", "", "_", "", ".", "").Replace(strings.ToLower(modelName))
}

func isMaiImage25Model(modelName string) bool {
	return strings.Contains(compactStudioModelName(modelName), "maiimage25")
}

func defaultMaiImageSizes() []string {
	return []string{
		"1024x1024",
		"1152x896",
		"896x1152",
		"1216x832",
		"832x1216",
		"1344x768",
		"768x1344",
		"1280x800",
		"800x1280",
	}
}

func defaultMaiImageSizeConstraints() *studioImageSizeConstraints {
	return &studioImageSizeConstraints{
		MinWidth:  768,
		MinHeight: 768,
		MaxPixels: 1048576,
	}
}

func studioImageConfigForModel(modelName string) studioImageModelConfig {
	normalized := strings.ToLower(modelName)
	if strings.Contains(normalized, "mai-image") {
		supportsEdit := isMaiImage25Model(normalized)
		maxReferenceImages := 0
		referenceMimeTypes := []string{}
		editPayloadShape := ""
		if supportsEdit {
			maxReferenceImages = 1
			referenceMimeTypes = []string{"image/png", "image/jpeg"}
			editPayloadShape = "mai-multipart"
		}
		return studioImageModelConfig{
			Sizes:              defaultMaiImageSizes(),
			SupportsN:          false,
			MaxN:               1,
			RequestShape:       "width-height",
			SizeMode:           "preset",
			SizeConstraints:    defaultMaiImageSizeConstraints(),
			SupportsImageEdit:  supportsEdit,
			MaxReferenceImages: maxReferenceImages,
			ReferenceMimeTypes: referenceMimeTypes,
			ReferenceFieldName: "image",
			EditPayloadShape:   editPayloadShape,
		}
	}
	if strings.Contains(normalized, "gpt-image-2") {
		return studioImageModelConfig{
			Sizes:              []string{"1024x1024", "1024x1536", "1536x1024", "2048x2048", "3840x2160", "2160x3840"},
			Quality:            []string{"auto", "low", "medium", "high"},
			SupportsQuality:    true,
			SupportsN:          true,
			MaxN:               3,
			RequestShape:       "size",
			SizeMode:           "preset",
			SupportsImageEdit:  true,
			MaxReferenceImages: 16,
			ReferenceMimeTypes: []string{"image/png", "image/jpeg", "image/webp"},
			EditPayloadShape:   "openai-multipart",
		}
	}
	if strings.Contains(normalized, "gpt-image") {
		return studioImageModelConfig{
			Sizes:              []string{"1024x1024", "1024x1536", "1536x1024"},
			Quality:            []string{"auto", "low", "medium", "high"},
			SupportsQuality:    true,
			SupportsN:          true,
			MaxN:               3,
			RequestShape:       "size",
			SizeMode:           "preset",
			SupportsImageEdit:  true,
			MaxReferenceImages: 16,
			ReferenceMimeTypes: []string{"image/png", "image/jpeg", "image/webp"},
			EditPayloadShape:   "openai-multipart",
		}
	}
	if normalized == "dall-e" || normalized == "dall-e-2" {
		return studioImageModelConfig{
			Sizes:        []string{"256x256", "512x512", "1024x1024"},
			SupportsN:    true,
			MaxN:         3,
			RequestShape: "size",
			SizeMode:     "preset",
		}
	}
	if normalized == "dall-e-3" {
		return studioImageModelConfig{
			Sizes:           []string{"1024x1024", "1024x1792", "1792x1024"},
			Quality:         []string{"standard", "hd"},
			SupportsQuality: true,
			SupportsN:       false,
			MaxN:            1,
			RequestShape:    "size",
			SizeMode:        "preset",
		}
	}
	return studioImageModelConfig{
		Sizes:        []string{"1024x1024"},
		SupportsN:    true,
		MaxN:         3,
		RequestShape: "size",
		SizeMode:     "preset",
	}
}

func (config studioImageModelConfig) toPayload() gin.H {
	quality := config.Quality
	if quality == nil {
		quality = []string{}
	}

	defaultSize := "1024x1024"
	if len(config.Sizes) > 0 {
		defaultSize = config.Sizes[0]
	}

	defaultQuality := "auto"
	if config.SupportsQuality {
		if stringSliceContains(quality, "auto") {
			defaultQuality = "auto"
		} else if len(quality) > 0 {
			defaultQuality = quality[0]
		}
	}

	sizeMode := config.SizeMode
	if sizeMode == "" {
		sizeMode = "preset"
	}
	editPayloadShape := config.EditPayloadShape
	if editPayloadShape == "" && config.SupportsImageEdit {
		editPayloadShape = "openai-multipart"
	}
	referenceMimeTypes := config.ReferenceMimeTypes
	if referenceMimeTypes == nil {
		referenceMimeTypes = []string{}
	}

	return gin.H{
		"sizes":                config.Sizes,
		"quality":              quality,
		"supports_quality":     config.SupportsQuality,
		"supports_n":           config.SupportsN,
		"max_n":                config.MaxN,
		"request_shape":        config.RequestShape,
		"size_mode":            sizeMode,
		"size_constraints":     config.SizeConstraints,
		"supports_image_edit":  config.SupportsImageEdit,
		"max_reference_images": config.MaxReferenceImages,
		"reference_mime_types": referenceMimeTypes,
		"reference_field_name": config.ReferenceFieldName,
		"edit_payload_shape":   editPayloadShape,
		"default_size":         defaultSize,
		"default_quality":      defaultQuality,
		"default_n":            1,
	}
}

func stringSliceContains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func parseStudioImageSize(size string) (int, int, bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return width, height, width > 0 && height > 0
}

func (config studioImageModelConfig) supportsSize(size string) bool {
	if stringSliceContains(config.Sizes, size) {
		return true
	}
	if config.SizeConstraints == nil {
		return false
	}
	width, height, ok := parseStudioImageSize(size)
	if !ok {
		return false
	}
	if config.SizeConstraints.MinWidth > 0 && width < config.SizeConstraints.MinWidth {
		return false
	}
	if config.SizeConstraints.MinHeight > 0 && height < config.SizeConstraints.MinHeight {
		return false
	}
	if config.SizeConstraints.MaxPixels > 0 && width*height > config.SizeConstraints.MaxPixels {
		return false
	}
	return true
}

func getStudioUserFromInternalRequest(c *gin.Context) (*model.User, bool) {
	if !validateStudioInternalRequest(c) {
		return nil, false
	}
	var req studioUserScopedRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	if req.UserId <= 0 {
		common.ApiErrorMsg(c, "missing user_id")
		return nil, false
	}
	user, err := model.GetUserById(req.UserId, false)
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	if user.Status == common.UserStatusDisabled {
		common.ApiErrorMsg(c, "user is disabled")
		return nil, false
	}
	return user, true
}

func buildStudioModelPayload(modelName string, imageOnly bool) gin.H {
	endpointTypes := model.GetModelSupportEndpointTypes(modelName)
	capabilities := []string{}
	if common.IsImageGenerationModel(modelName) || endpointTypesContain(endpointTypes, constant.EndpointTypeImageGeneration) {
		capabilities = append(capabilities, "image_generation")
	} else {
		capabilities = append(capabilities, "chat")
	}
	if !imageOnly && len(capabilities) == 0 {
		capabilities = append(capabilities, "chat")
	}

	payload := gin.H{
		"id":           modelName,
		"display_name": modelName,
		"enabled":      true,
		"capabilities": capabilities,
	}

	if endpointTypesContain(endpointTypes, constant.EndpointTypeImageGeneration) || common.IsImageGenerationModel(modelName) {
		for key, value := range studioImageConfigForModel(modelName).toPayload() {
			payload[key] = value
		}
	}
	return payload
}

func encryptStudioCredential(payload gin.H) (string, error) {
	secret := os.Getenv("STUDIO_CREDENTIAL_SECRET")
	if secret == "" {
		secret = getStudioInternalSecret()
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func resolveStudioMappedModel(modelName string, modelMapping string) string {
	if modelMapping == "" || modelMapping == "{}" {
		return modelName
	}

	var modelMap map[string]string
	if err := json.Unmarshal([]byte(modelMapping), &modelMap); err != nil {
		return modelName
	}

	current := modelName
	visited := map[string]bool{current: true}
	for i := 0; i < 8; i++ {
		next := modelMap[current]
		if next == "" || visited[next] {
			break
		}
		current = next
		visited[current] = true
	}
	return current
}

func studioImageProviderForChannel(channel *model.Channel, baseURL string) string {
	if channel.Type == constant.ChannelTypeAzure || strings.Contains(baseURL, "openai.azure.com") {
		return "azure_openai"
	}
	if channel.Type == constant.ChannelTypeOpenAI {
		return "openai"
	}
	return "openai_compatible"
}

func fallbackStudioImageUpstream(reservation *model.StudioImageReservation) (*studioImageUpstreamSelection, error) {
	baseURL := os.Getenv("STUDIO_IMAGE_UPSTREAM_BASE_URL")
	apiKey := os.Getenv("STUDIO_IMAGE_UPSTREAM_API_KEY")
	if baseURL == "" || apiKey == "" {
		return nil, errors.New("studio image upstream is not configured")
	}
	upstreamModel := common.GetEnvOrDefaultString("STUDIO_IMAGE_UPSTREAM_MODEL", reservation.Model)
	return &studioImageUpstreamSelection{
		Provider:      common.GetEnvOrDefaultString("STUDIO_IMAGE_UPSTREAM_PROVIDER", reservation.Provider),
		BaseURL:       strings.TrimRight(baseURL, "/"),
		APIKey:        apiKey,
		APIVersion:    common.GetEnvOrDefaultString("STUDIO_IMAGE_UPSTREAM_API_VERSION", "2025-04-01-preview"),
		Deployment:    common.GetEnvOrDefaultString("STUDIO_IMAGE_UPSTREAM_DEPLOYMENT", upstreamModel),
		UpstreamModel: upstreamModel,
		UpstreamRef:   common.GetEnvOrDefaultString("STUDIO_IMAGE_UPSTREAM_REF", "env"),
	}, nil
}

func selectStudioImageUpstream(user *model.User, reservation *model.StudioImageReservation, upstreamRef string) (*studioImageUpstreamSelection, error) {
	group := user.Group
	if group == "" {
		group = "default"
	}

	var channel *model.Channel
	var err error
	if strings.HasPrefix(upstreamRef, "channel:") {
		channelId, parseErr := strconv.Atoi(strings.TrimPrefix(upstreamRef, "channel:"))
		if parseErr == nil && channelId > 0 {
			channel, err = model.GetChannelById(channelId, true)
		}
	} else {
		channel, err = model.GetChannel(group, reservation.Model, 0, "/api/studio/images/generations")
	}

	if err != nil {
		return nil, err
	}
	if channel == nil {
		return fallbackStudioImageUpstream(reservation)
	}

	key, _, apiErr := channel.GetNextEnabledKey()
	if apiErr != nil {
		return nil, apiErr
	}

	baseURL := strings.TrimRight(channel.GetBaseURL(), "/")
	if baseURL == "" || key == "" {
		return nil, fmt.Errorf("studio image channel %d is missing base_url or key", channel.Id)
	}

	upstreamModel := resolveStudioMappedModel(reservation.Model, channel.GetModelMapping())
	apiVersion := channel.Other
	if apiVersion == "" && channel.Type == constant.ChannelTypeAzure {
		apiVersion = common.GetEnvOrDefaultString("STUDIO_IMAGE_UPSTREAM_API_VERSION", "2025-04-01-preview")
	}

	return &studioImageUpstreamSelection{
		Provider:      studioImageProviderForChannel(channel, baseURL),
		BaseURL:       baseURL,
		APIKey:        key,
		APIVersion:    apiVersion,
		Deployment:    upstreamModel,
		UpstreamModel: upstreamModel,
		UpstreamRef:   fmt.Sprintf("channel:%d", channel.Id),
		ChannelId:     channel.Id,
		ChannelName:   channel.Name,
		ChannelType:   channel.Type,
	}, nil
}

func studioImageBillingUpstreamFromReservation(reservation *model.StudioImageReservation) (*studioImageUpstreamSelection, error) {
	channelId := studioImageChannelIdFromRef(reservation.UpstreamRef)
	if channelId <= 0 {
		return &studioImageUpstreamSelection{
			Provider:      reservation.Provider,
			UpstreamModel: reservation.Model,
			UpstreamRef:   reservation.UpstreamRef,
		}, nil
	}

	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimRight(channel.GetBaseURL(), "/")
	upstreamModel := resolveStudioMappedModel(reservation.Model, channel.GetModelMapping())
	return &studioImageUpstreamSelection{
		Provider:      studioImageProviderForChannel(channel, baseURL),
		BaseURL:       baseURL,
		APIVersion:    channel.Other,
		Deployment:    upstreamModel,
		UpstreamModel: upstreamModel,
		UpstreamRef:   fmt.Sprintf("channel:%d", channel.Id),
		ChannelId:     channel.Id,
		ChannelName:   channel.Name,
		ChannelType:   channel.Type,
	}, nil
}

func buildStudioImageCredentialEnvelope(reservation *model.StudioImageReservation, upstream *studioImageUpstreamSelection) (gin.H, error) {
	now := time.Now().Unix()
	expiresAt := now + getStudioImageCredentialTTLSeconds()
	payload := gin.H{
		"reservation_id":  reservation.Id,
		"user_id":         reservation.UserId,
		"provider":        upstream.Provider,
		"model":           upstream.UpstreamModel,
		"requested_model": reservation.Model,
		"base_url":        upstream.BaseURL,
		"api_key":         upstream.APIKey,
		"api_version":     upstream.APIVersion,
		"deployment":      upstream.Deployment,
		"upstream_ref":    upstream.UpstreamRef,
		"channel_id":      upstream.ChannelId,
		"channel_name":    upstream.ChannelName,
		"channel_type":    upstream.ChannelType,
		"expires_at":      expiresAt,
	}
	envelope, err := encryptStudioCredential(payload)
	if err != nil {
		return nil, err
	}
	return gin.H{
		"provider":              upstream.Provider,
		"base_url":              payload["base_url"],
		"api_version":           payload["api_version"],
		"deployment":            payload["deployment"],
		"model":                 payload["model"],
		"upstream_ref":          upstream.UpstreamRef,
		"channel_id":            upstream.ChannelId,
		"credential_envelope":   envelope,
		"credential_expires_at": expiresAt,
	}, nil
}

func listStudioModelsForUser(user *model.User, imageOnly bool) []gin.H {
	group := user.Group
	if group == "" {
		group = "default"
	}
	modelNames := model.GetGroupEnabledModels(group)
	models := make([]gin.H, 0, len(modelNames))
	seen := map[string]bool{}
	for _, modelName := range modelNames {
		if modelName == "" || seen[modelName] {
			continue
		}
		seen[modelName] = true
		isImageModel := common.IsImageGenerationModel(modelName) ||
			endpointTypesContain(model.GetModelSupportEndpointTypes(modelName), constant.EndpointTypeImageGeneration)
		if imageOnly && !isImageModel {
			continue
		}
		if !imageOnly && isImageModel {
			continue
		}
		models = append(models, buildStudioModelPayload(modelName, imageOnly))
	}
	return models
}

func studioUserCanUseModel(user *model.User, modelName string, imageOnly bool) bool {
	for _, item := range listStudioModelsForUser(user, imageOnly) {
		if id, ok := item["id"].(string); ok && id == modelName {
			return true
		}
	}
	return false
}

func normalizeStudioModelLimits(modelLimits []string) []string {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(modelLimits))
	for _, modelName := range modelLimits {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" || seen[modelName] {
			continue
		}
		seen[modelName] = true
		normalized = append(normalized, modelName)
	}
	sort.Strings(normalized)
	return normalized
}

func getOrCreateStudioChatToken(user *model.User, modelLimits []string) (*model.Token, error) {
	now := time.Now().Unix()
	modelLimits = normalizeStudioModelLimits(modelLimits)
	modelLimitsValue := strings.Join(modelLimits, ",")
	ttl := getStudioChatKeyTTLSeconds()

	var token model.Token
	err := model.DB.Where(
		"user_id = ? AND name = ? AND status = ?",
		user.Id,
		"AI Studio Chat Session",
		common.TokenStatusEnabled,
	).Order("id desc").First(&token).Error
	if err == nil {
		token.AccessedTime = now
		token.ExpiredTime = now + ttl
		token.UnlimitedQuota = true
		token.ModelLimitsEnabled = len(modelLimits) > 0
		token.ModelLimits = modelLimitsValue
		if updateErr := token.Update(); updateErr != nil {
			return nil, updateErr
		}
		if updateErr := token.SelectUpdate(); updateErr != nil {
			return nil, updateErr
		}
		return &token, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	key, err := common.GenerateKey()
	if err != nil {
		return nil, err
	}
	token = model.Token{
		UserId:             user.Id,
		Name:               "AI Studio Chat Session",
		Key:                key,
		CreatedTime:        now,
		AccessedTime:       now,
		ExpiredTime:        now + ttl,
		UnlimitedQuota:     true,
		ModelLimitsEnabled: len(modelLimits) > 0,
		ModelLimits:        modelLimitsValue,
	}
	if err := token.Insert(); err != nil {
		return nil, err
	}
	return &token, nil
}

func OpenStudio(c *gin.Context) {
	userId, err := currentStudioUserId(c)
	if err != nil {
		if shouldReturnStudioOpenJSON(c) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "unauthorized",
			})
			return
		}
		redirect := "/login?redirect=/console"
		c.Redirect(http.StatusFound, redirect)
		return
	}

	ticket, err := common.GenerateRandomKey(48)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	state, err := common.GenerateRandomKey(24)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	nonce, err := common.GenerateRandomKey(24)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	_, err = model.CreateStudioSSOTicket(
		ticket,
		userId,
		state,
		nonce,
		getStudioTicketTTLSeconds(),
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	callbackURL := fmt.Sprintf(
		"%s/auth/new-api/callback?ticket=%s&state=%s",
		getStudioOpenWebUIURL(),
		url.QueryEscape(ticket),
		url.QueryEscape(state),
	)

	if shouldReturnStudioOpenJSON(c) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"url": callbackURL,
			},
		})
		return
	}
	c.Redirect(http.StatusFound, callbackURL)
}

func GetStudioChatConfig(c *gin.Context) {
	user, ok := getStudioUserFromInternalRequest(c)
	if !ok {
		return
	}
	chatModels := listStudioModelsForUser(user, false)
	modelLimits := make([]string, 0, len(chatModels))
	for _, item := range chatModels {
		if id, ok := item["id"].(string); ok && id != "" {
			modelLimits = append(modelLimits, id)
		}
	}

	token, err := getOrCreateStudioChatToken(user, modelLimits)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"base_url":   common.GetEnvOrDefaultString("STUDIO_CHAT_BASE_URL", "https://www.unikeyx.com/v1"),
		"api_key":    "sk-" + token.Key,
		"models":     chatModels,
		"expires_at": token.ExpiredTime,
	})
}

func GetStudioImageModels(c *gin.Context) {
	user, ok := getStudioUserFromInternalRequest(c)
	if !ok {
		return
	}
	common.ApiSuccess(c, gin.H{
		"models": listStudioModelsForUser(user, true),
	})
}

func CreateStudioImageReservation(c *gin.Context) {
	if !validateStudioInternalRequest(c) {
		return
	}
	var req studioImageReservationRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.UserId <= 0 || req.RequestId == "" || req.Model == "" {
		common.ApiErrorMsg(c, "invalid image reservation request")
		return
	}
	user, err := model.GetUserById(req.UserId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user.Status == common.UserStatusDisabled {
		common.ApiErrorMsg(c, "user is disabled")
		return
	}
	if common.IsImageGenerationModel(req.Model) == false &&
		!endpointTypesContain(model.GetModelSupportEndpointTypes(req.Model), constant.EndpointTypeImageGeneration) {
		common.ApiErrorMsg(c, "model is not an image generation model")
		return
	}
	if !studioUserCanUseModel(user, req.Model, true) {
		common.ApiErrorMsg(c, "model is not enabled for user")
		return
	}
	if req.N <= 0 {
		req.N = 1
	}
	if req.Provider == "" {
		req.Provider = "azure_openai"
	}

	modelConfig := studioImageConfigForModel(req.Model)
	if req.Size == "" {
		req.Size = modelConfig.toPayload()["default_size"].(string)
	}
	if req.Quality == "" {
		req.Quality = modelConfig.toPayload()["default_quality"].(string)
	}
	if !modelConfig.supportsSize(req.Size) {
		common.ApiErrorMsg(c, "size is not supported by image model")
		return
	}
	if modelConfig.SupportsQuality {
		if req.Quality == "" {
			req.Quality = modelConfig.Quality[0]
		}
		if req.Quality != "auto" && !stringSliceContains(modelConfig.Quality, req.Quality) {
			common.ApiErrorMsg(c, "quality is not supported by image model")
			return
		}
	} else {
		req.Quality = "auto"
	}
	if !modelConfig.SupportsN && req.N > 1 {
		common.ApiErrorMsg(c, "n is not supported by image model")
		return
	}
	if req.N > modelConfig.MaxN {
		common.ApiErrorMsg(c, "n must be less than 4")
		return
	}

	reservation := &model.StudioImageReservation{
		Id:        "resv_" + common.GetUUID(),
		UserId:    req.UserId,
		RequestId: req.RequestId,
		Provider:  req.Provider,
		Model:     req.Model,
		Size:      req.Size,
		Quality:   req.Quality,
		N:         req.N,
	}
	selectedUpstream, err := selectStudioImageUpstream(user, reservation, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	reservation.Provider = selectedUpstream.Provider
	reservation.UpstreamRef = selectedUpstream.UpstreamRef

	imageRequest := studioImageRequestForPricing(req.Model, req.Size, req.Quality, req.N)
	if _, _, err := priceStudioImageRequest(c, user, req.RequestId, imageRequest, selectedUpstream); err != nil {
		common.ApiError(c, err)
		return
	}

	cost := getStudioImagePreconsumeQuota()
	allocations, err := model.ConsumeUserQuotaWithAllocation(req.UserId, cost)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	now := time.Now().Unix()
	reservation.EstimatedCost = cost
	reservation.ExpiresAt = now + getStudioImageReservationTTLSeconds()
	if err := model.CreateStudioImageReservation(reservation, allocations); err != nil {
		_, _, _ = model.RefundUserQuotaAllocations(req.UserId, allocations, cost)
		common.ApiError(c, err)
		return
	}

	upstream, err := buildStudioImageCredentialEnvelope(reservation, selectedUpstream)
	if err != nil {
		_, _, _ = model.ReleaseStudioImageReservation(reservation.Id)
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"reservation_id": reservation.Id,
		"status":         reservation.Status,
		"estimated_cost": reservation.EstimatedCost,
		"currency":       "credits",
		"expires_at":     reservation.ExpiresAt,
		"upstream":       upstream,
		"model_config": func() gin.H {
			payload := modelConfig.toPayload()
			payload["model"] = reservation.Model
			return payload
		}(),
	})
}

func RefreshStudioImageCredential(c *gin.Context) {
	if !validateStudioInternalRequest(c) {
		return
	}
	id := c.Param("id")
	var req studioImageReservationActionRequest
	_ = common.DecodeJson(c.Request.Body, &req)
	reservation, err := model.GetStudioImageReservation(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if req.UserId > 0 && req.UserId != reservation.UserId {
		common.ApiErrorMsg(c, "reservation does not belong to user")
		return
	}
	if reservation.Status != model.StudioImageReservationStatusReserved || reservation.ExpiresAt < time.Now().Unix() {
		common.ApiErrorMsg(c, "image reservation is not refreshable")
		return
	}
	user, err := model.GetUserById(reservation.UserId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	selectedUpstream, err := selectStudioImageUpstream(user, reservation, reservation.UpstreamRef)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	upstream, err := buildStudioImageCredentialEnvelope(reservation, selectedUpstream)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, upstream)
}

func CommitStudioImageReservation(c *gin.Context) {
	if !validateStudioInternalRequest(c) {
		return
	}
	id := c.Param("id")
	var req studioImageReservationActionRequest
	_ = common.DecodeJson(c.Request.Body, &req)
	existing, err := model.GetStudioImageReservation(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if req.UserId > 0 && req.UserId != existing.UserId {
		common.ApiErrorMsg(c, "reservation does not belong to user")
		return
	}
	user, err := model.GetUserById(existing.UserId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	outputCount := req.OutputCount
	if outputCount <= 0 {
		outputCount = existing.N
	}
	if outputCount > existing.N && existing.N > 0 {
		outputCount = existing.N
	}
	selectedUpstream, err := studioImageBillingUpstreamFromReservation(existing)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	imageRequest := studioImageRequestForPricing(existing.Model, existing.Size, existing.Quality, outputCount)
	relayInfo, priceData, err := priceStudioImageRequest(c, user, existing.RequestId, imageRequest, selectedUpstream)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	finalCost, usage := calculateStudioImageFinalCost(c, existing, relayInfo, priceData, req.Usage, outputCount)

	shouldRecordConsume := existing.Status == model.StudioImageReservationStatusReserved
	reservation, err := model.CommitStudioImageReservation(id, finalCost)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if shouldRecordConsume {
		relayInfo.FinalPreConsumedQuota = finalCost
		relayInfo.BillingSource = service.BillingSourceWallet
		relayInfo.QuotaFundingAllocations = studioImageReservationAllocations(reservation)
		relayInfo.PaidQuotaConsumed, relayInfo.GiftQuotaConsumed, _ = model.SumQuotaFundingAllocations(relayInfo.QuotaFundingAllocations)
		recordStudioImageConsume(c, reservation, req.JobId, finalCost, usage, relayInfo, priceData, outputCount)
	}
	common.ApiSuccess(c, reservation)
}

func ReleaseStudioImageReservation(c *gin.Context) {
	if !validateStudioInternalRequest(c) {
		return
	}
	id := c.Param("id")
	var req studioImageReservationActionRequest
	_ = common.DecodeJson(c.Request.Body, &req)
	existing, err := model.GetStudioImageReservation(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if req.UserId > 0 && req.UserId != existing.UserId {
		common.ApiErrorMsg(c, "reservation does not belong to user")
		return
	}
	reservation, _, err := model.ReleaseStudioImageReservation(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, reservation)
}

func CreateStudioImageAudit(c *gin.Context) {
	if !validateStudioInternalRequest(c) {
		return
	}
	var req studioImageAuditRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	var reservation *model.StudioImageReservation
	if req.ReservationId != "" {
		var err error
		reservation, err = model.GetStudioImageReservation(req.ReservationId)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if req.UserId > 0 && reservation.UserId != req.UserId {
			common.ApiErrorMsg(c, "reservation does not belong to user")
			return
		}
	}
	metadata, _ := json.Marshal(req.Metadata)
	row := &model.StudioImageAuditLog{
		ReservationId: req.ReservationId,
		UserId:        req.UserId,
		JobId:         req.JobId,
		EventType:     req.EventType,
		Provider:      req.Provider,
		Model:         req.Model,
		Amount:        studioImageAuditAmount(req, reservation),
		Status:        req.Status,
		ErrorCode:     req.ErrorCode,
		LatencyMs:     req.LatencyMs,
		MetadataJSON:  string(metadata),
		CreatedAt:     time.Now().Unix(),
	}
	if err := model.DB.Create(row).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, row)
}

func ExchangeStudioSSO(c *gin.Context) {
	if !validateStudioInternalRequest(c) {
		return
	}

	var req studioSSOExchangeRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Ticket == "" || req.Audience != "open-webui" {
		common.ApiErrorMsg(c, "invalid studio sso request")
		return
	}

	ticket, err := model.ConsumeStudioSSOTicket(req.Ticket)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if req.State != "" && ticket.State != "" && req.State != ticket.State {
		common.ApiErrorMsg(c, "invalid studio sso state")
		return
	}

	user, err := model.GetUserById(ticket.UserId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user.Status == common.UserStatusDisabled {
		common.ApiErrorMsg(c, "user is disabled")
		return
	}

	displayName := user.DisplayName
	if displayName == "" {
		displayName = user.Username
	}

	common.ApiSuccess(c, gin.H{
		"user": gin.H{
			"id":           user.Id,
			"username":     user.Username,
			"email":        user.Email,
			"display_name": displayName,
			"role":         roleNameForStudio(user.Role),
			"status":       "active",
		},
		"entitlements": gin.H{
			"ai_studio":        true,
			"chat":             true,
			"image_generation": true,
		},
		"expires_at": ticket.ExpiresAt,
	})
}
