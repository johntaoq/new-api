package controller

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

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
