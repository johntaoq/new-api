package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
)

func DoubaoNativeRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyDoubaoNativeAPI, true)

		switch c.Request.Method {
		case http.MethodPost:
			convertDoubaoNativeSubmit(c)
			if c.IsAborted() {
				return
			}
		case http.MethodGet:
			taskID := strings.TrimSpace(c.Param("task_id"))
			if taskID == "" {
				abortWithOpenAiMessage(c, http.StatusBadRequest, "task_id is required")
				return
			}
			c.Request.URL.Path = "/v1/video/generations/" + taskID
			c.Set("task_id", taskID)
			c.Set("relay_mode", relayconstant.RelayModeVideoFetchByID)
		}

		c.Next()
	}
}

func convertDoubaoNativeSubmit(c *gin.Context) {
	var originalReq map[string]interface{}
	if err := common.UnmarshalBodyReusable(c, &originalReq); err != nil {
		abortWithOpenAiMessage(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	model, _ := originalReq["model"].(string)
	if strings.TrimSpace(model) == "" {
		abortWithOpenAiMessage(c, http.StatusBadRequest, "model field is required")
		return
	}

	prompt, _ := originalReq["prompt"].(string)
	if strings.TrimSpace(prompt) == "" {
		prompt = extractDoubaoNativePrompt(originalReq["content"])
	}
	if strings.TrimSpace(prompt) == "" {
		abortWithOpenAiMessage(c, http.StatusBadRequest, "content text is required")
		return
	}

	unifiedReq := map[string]interface{}{
		"model":    model,
		"prompt":   prompt,
		"metadata": originalReq,
	}

	if duration, ok := normalizeDoubaoNativeDuration(originalReq["duration"]); ok {
		unifiedReq["duration"] = duration
		unifiedReq["seconds"] = strconv.Itoa(duration)
	}
	if size, _ := originalReq["size"].(string); strings.TrimSpace(size) != "" {
		unifiedReq["size"] = size
	}

	jsonData, err := json.Marshal(unifiedReq)
	if err != nil {
		abortWithOpenAiMessage(c, http.StatusInternalServerError, "Failed to marshal request body")
		return
	}

	common.CleanupBodyStorage(c)
	c.Set(common.KeyRequestBody, jsonData)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(jsonData))
	c.Request.ContentLength = int64(len(jsonData))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.URL.Path = "/v1/video/generations"
	c.Set("relay_mode", relayconstant.RelayModeVideoSubmit)
}

func extractDoubaoNativePrompt(content interface{}) string {
	items, ok := content.([]interface{})
	if !ok {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if typ, _ := m["type"].(string); typ != "" && typ != "text" {
			continue
		}
		text, _ := m["text"].(string)
		if text = strings.TrimSpace(text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func normalizeDoubaoNativeDuration(v interface{}) (int, bool) {
	switch val := v.(type) {
	case float64:
		if val > 0 {
			return int(val), true
		}
	case int:
		if val > 0 {
			return val, true
		}
	case json.Number:
		i, err := val.Int64()
		if err == nil && i > 0 {
			return int(i), true
		}
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(val))
		if err == nil && i > 0 {
			return i, true
		}
	}
	return 0, false
}
