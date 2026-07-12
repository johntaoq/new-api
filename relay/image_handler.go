package relay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

var (
	imageRelaySemaphoreMu       sync.Mutex
	imageRelaySemaphore         chan struct{}
	imageRelaySemaphoreCapacity int
)

func acquireImageRelaySlot() (func(), bool) {
	limit := common.ImageRelayConcurrency
	if limit <= 0 {
		return func() {}, true
	}

	imageRelaySemaphoreMu.Lock()
	if imageRelaySemaphore == nil || imageRelaySemaphoreCapacity != limit {
		imageRelaySemaphore = make(chan struct{}, limit)
		imageRelaySemaphoreCapacity = limit
	}
	sem := imageRelaySemaphore
	imageRelaySemaphoreMu.Unlock()

	select {
	case sem <- struct{}{}:
		return func() { <-sem }, true
	default:
		return nil, false
	}
}

func normalizedImageRequestN(request *dto.ImageRequest) uint {
	if request == nil || request.N == nil || *request.N == 0 {
		return 1
	}
	return *request.N
}

func applyImageUsageFallback(usage *dto.Usage, request *dto.ImageRequest) {
	if usage == nil {
		return
	}

	if usage.PromptTokens <= 0 {
		if usage.InputTokens > 0 {
			usage.PromptTokens = usage.InputTokens
		} else if usage.NumInputTextTokens > 0 || usage.NumInputImageTokens > 0 {
			usage.PromptTokens = usage.NumInputTextTokens + usage.NumInputImageTokens
		} else {
			usage.PromptTokens = 1
		}
	}

	if usage.CompletionTokens <= 0 {
		if usage.OutputTokens > 0 {
			usage.CompletionTokens = usage.OutputTokens
		} else if usage.NumOutputTokens > 0 {
			usage.CompletionTokens = usage.NumOutputTokens
		} else if usage.TotalTokens > usage.PromptTokens {
			usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
		} else if request != nil {
			meta := request.GetTokenCountMeta()
			if meta != nil && meta.MaxTokens > 0 {
				usage.CompletionTokens = meta.MaxTokens * int(normalizedImageRequestN(request))
			}
		}
	}

	if usage.InputTokens <= 0 {
		usage.InputTokens = usage.PromptTokens
	}
	if usage.OutputTokens <= 0 {
		usage.OutputTokens = usage.CompletionTokens
	}
	if usage.NumOutputTokens <= 0 {
		usage.NumOutputTokens = usage.CompletionTokens
	}
	if usage.CompletionTokenDetails.ImageTokens <= 0 {
		usage.CompletionTokenDetails.ImageTokens = usage.CompletionTokens
	}

	totalTokens := usage.PromptTokens + usage.CompletionTokens
	if usage.TotalTokens <= 0 || usage.TotalTokens < totalTokens {
		usage.TotalTokens = totalTokens
	}
}

func ImageHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	imageReq, ok := info.Request.(*dto.ImageRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.ImageRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	releaseImageRelaySlot, ok := acquireImageRelaySlot()
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("too many image requests, please retry later"), types.ErrorCodeChannelImageRelayBusy, http.StatusTooManyRequests, types.ErrOptionWithSkipRetry())
	}
	defer releaseImageRelaySlot()

	request, err := common.DeepCopy(imageReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to ImageRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	var requestBody io.Reader

	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		requestBody = common.ReaderOnly(storage)
	} else {
		convertedRequest, err := adaptor.ConvertImageRequest(c, info, *request)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed)
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)

		switch convertedRequest.(type) {
		case *bytes.Buffer:
			requestBody = convertedRequest.(io.Reader)
		default:
			jsonData, err := common.Marshal(convertedRequest)
			if err != nil {
				return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}

			// apply param override
			if len(info.ParamOverride) > 0 {
				jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
				if err != nil {
					return newAPIErrorFromParamOverride(err)
				}
			}

			logger.LogDebug(c, "image request body: %s", jsonData)
			body, size, closer, err := relaycommon.NewOutboundJSONBody(jsonData)
			if err != nil {
				return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
			defer closer.Close()
			jsonData = nil
			info.UpstreamRequestBodySize = size
			requestBody = body
		}
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return types.NewErrorWithStatusCode(fmt.Errorf("image relay timeout after %d seconds", common.ImageRelayTimeout), types.ErrorCodeChannelResponseTimeExceeded, http.StatusGatewayTimeout, types.ErrOptionWithSkipRetry())
		}
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
		if httpResp.StatusCode != http.StatusOK {
			if httpResp.StatusCode == http.StatusCreated && info.ApiType == constant.APITypeReplicate {
				// replicate channel returns 201 Created when using Prefer: wait, treat it as success.
				httpResp.StatusCode = http.StatusOK
			} else {
				newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
				// reset status code 重置状态码
				service.ResetStatusCode(newAPIError, statusCodeMappingStr)
				return newAPIError
			}
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	imageN := normalizedImageRequestN(request)

	// n is handled via OtherRatio so it is applied exactly once in quota
	// calculation (both price-based and ratio-based paths).
	// Adaptors may have already set a more accurate count from the
	// upstream response; only set the default when they haven't.
	if info.PriceData.UsePrice { // only price model use N ratio
		if _, hasN := info.PriceData.OtherRatios["n"]; !hasN {
			info.PriceData.AddOtherRatio("n", float64(imageN))
		}
	}

	applyImageUsageFallback(usage.(*dto.Usage), request)

	quality := request.Quality
	if quality == "" {
		quality = "standard"
	}

	var logContent []string

	if len(request.Size) > 0 {
		logContent = append(logContent, fmt.Sprintf("大小 %s", request.Size))
	}
	if len(quality) > 0 {
		logContent = append(logContent, fmt.Sprintf("品质 %s", quality))
	}
	if imageN > 0 {
		logContent = append(logContent, fmt.Sprintf("生成数量 %d", imageN))
	}

	service.PostTextConsumeQuota(c, info, usage.(*dto.Usage), logContent)
	return nil
}
