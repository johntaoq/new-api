package controller

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const playgroundReferenceImageMaxBytes = 25 * 1024 * 1024

type playgroundImageReferenceRequest struct {
	URL string `json:"url"`
}

func Playground(c *gin.Context) {
	playgroundRelay(c, types.RelayFormatOpenAI)
}

func PlaygroundImage(c *gin.Context) {
	originalWriter := c.Writer
	captureWriter := &playgroundImageResponseWriter{
		ResponseWriter: originalWriter,
		statusCode:     http.StatusOK,
	}
	c.Writer = captureWriter
	defer func() {
		c.Writer = originalWriter
	}()

	playgroundRelay(c, types.RelayFormatOpenAIImage)

	responseBody := captureWriter.body.Bytes()
	if captureWriter.statusCode >= http.StatusOK && captureWriter.statusCode < http.StatusMultipleChoices {
		responseBody = service.ArchivePlaygroundImageResponse(c, responseBody)
	}

	originalWriter.Header().Set("Content-Length", fmt.Sprintf("%d", len(responseBody)))
	originalWriter.WriteHeader(captureWriter.statusCode)
	_, _ = originalWriter.Write(responseBody)
}

func PlaygroundImageReference(c *gin.Context) {
	var request playgroundImageReferenceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	imageURL := strings.TrimSpace(request.URL)
	if imageURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "image url is required"})
		return
	}

	response, err := service.DoDownloadRequest(imageURL, "image playground reference")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": "failed to load reference image"})
		return
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		c.JSON(http.StatusBadGateway, gin.H{"message": "reference image source returned an error"})
		return
	}

	imageBytes, err := io.ReadAll(io.LimitReader(response.Body, playgroundReferenceImageMaxBytes+1))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": "failed to read reference image"})
		return
	}
	contentType, err := validatePlaygroundReferenceImage(imageBytes, response.Header.Get("Content-Type"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, contentType, imageBytes)
}

func validatePlaygroundReferenceImage(imageBytes []byte, upstreamContentType string) (string, error) {
	if len(imageBytes) == 0 {
		return "", errors.New("reference image is empty")
	}
	if len(imageBytes) > playgroundReferenceImageMaxBytes {
		return "", errors.New("reference image exceeds 25 MB")
	}

	detectedContentType := strings.ToLower(strings.TrimSpace(strings.Split(http.DetectContentType(imageBytes), ";")[0]))
	switch detectedContentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return detectedContentType, nil
	}

	upstreamContentType = strings.ToLower(strings.TrimSpace(strings.Split(upstreamContentType, ";")[0]))
	if strings.HasPrefix(upstreamContentType, "image/") {
		return "", errors.New("unsupported reference image format")
	}
	return "", errors.New("reference source is not an image")
}

type playgroundImageResponseWriter struct {
	gin.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

func (w *playgroundImageResponseWriter) WriteHeader(code int) {
	w.statusCode = code
}

func (w *playgroundImageResponseWriter) WriteHeaderNow() {
}

func (w *playgroundImageResponseWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *playgroundImageResponseWriter) WriteString(data string) (int, error) {
	return w.body.WriteString(data)
}

func (w *playgroundImageResponseWriter) Status() int {
	return w.statusCode
}

func (w *playgroundImageResponseWriter) Size() int {
	return w.body.Len()
}

func (w *playgroundImageResponseWriter) Written() bool {
	return w.body.Len() > 0
}

func (w *playgroundImageResponseWriter) Flush() {
}

func playgroundRelay(c *gin.Context, relayFormat types.RelayFormat) {
	var newAPIError *types.NewAPIError

	defer func() {
		if newAPIError != nil {
			c.JSON(newAPIError.StatusCode, gin.H{
				"error": newAPIError.ToOpenAIError(),
			})
		}
	}()

	useAccessToken := c.GetBool("use_access_token")
	if useAccessToken {
		newAPIError = types.NewError(errors.New("暂不支持使用 access token"), types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry())
		return
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAI, nil, nil)
	if err != nil {
		newAPIError = types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
		return
	}

	userId := c.GetInt("id")

	// Write user context to ensure acceptUnsetRatio is available
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		newAPIError = types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry())
		return
	}
	userCache.WriteContext(c)

	tempToken := &model.Token{
		UserId: userId,
		Name:   fmt.Sprintf("playground-%s", relayInfo.UsingGroup),
		Group:  relayInfo.UsingGroup,
	}
	_ = middleware.SetupContextForToken(c, tempToken)

	Relay(c, relayFormat)
}
