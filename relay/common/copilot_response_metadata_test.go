package common_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/claude"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestFilterCopilotResponseMetadata(t *testing.T) {
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{
		ChannelOtherSettings: dto.ChannelOtherSettings{StripCopilotMetadata: true},
	}}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"response metadata", `{"copilot_usage":{"cost":999},"copilot":{"trace":"x"},"Copilot_trace":false,"id":"Copilot-original","model":"gpt-test","output_text":null,"created_at":9007199254740993,"usage":{"input_tokens":9,"copilot_cost":999},"output":[{"type":"reasoning","encrypted_content":"original"},{"type":"function_call","arguments":"{\"copilot_usage\":\"keep\"}"}],"metadata":{"copilot_usage":"keep"}}`, `{"id":"Copilot-original","model":"gpt-test","output_text":null,"created_at":9007199254740993,"usage":{"input_tokens":9},"output":[{"type":"reasoning","encrypted_content":"original"},{"type":"function_call","arguments":"{\"copilot_usage\":\"keep\"}"}],"metadata":{"copilot_usage":"keep"}}`},
		{"SSE response envelope", `{"type":"response.completed","response":{"copilot_usage":{},"id":"resp_1","usage":{"input_tokens":9,"copilot_trace":"x"},"output":[]}}`, `{"type":"response.completed","response":{"id":"resp_1","usage":{"input_tokens":9},"output":[]}}`},
		{"Claude message envelope", `{"type":"message_start","message":{"copilot_usage":{},"usage":{"input_tokens":9,"copilot_trace":"x"},"content":[{"type":"tool_use","input":{"copilot_usage":"keep"}}]}}`, `{"type":"message_start","message":{"usage":{"input_tokens":9},"content":[{"type":"tool_use","input":{"copilot_usage":"keep"}}]}}`},
		{"literal text delta", `{"type":"response.output_text.delta","delta":"Copilot copilot_usage","copilot_usage":{}}`, `{"type":"response.output_text.delta","delta":"Copilot copilot_usage"}`},
		{"escaped property names", `{"copilot_usage.trace":1,"copilot_usage\\trace":2,"copilot_usage:trace":3,"usage":{"input_tokens":9}}`, `{"usage":{"input_tokens":9}}`},
		{"no vendor fields", "{\n  \"id\": \"resp_1\", \"output_text\": null\n}", "{\n  \"id\": \"resp_1\", \"output_text\": null\n}"},
		{"malformed", `{"copilot_usage":`, `{"copilot_usage":`},
		{"array", `[{"copilot_usage":"user data"}]`, `[{"copilot_usage":"user data"}]`},
		{"done sentinel", `[DONE]`, `[DONE]`},
		{"empty", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := []byte(tc.body)
			original := bytes.Clone(payload)
			filtered := relaycommon.FilterCopilotResponseMetadata(info, payload)
			assert.Equal(t, string(filtered), relaycommon.FilterCopilotResponseMetadataString(info, tc.body))
			assert.Equal(t, original, payload, "the original wire bytes belong to billing")
			if gjson.Valid(tc.want) && tc.body != tc.want {
				assert.JSONEq(t, tc.want, string(filtered))
			} else {
				assert.Equal(t, tc.want, string(filtered))
			}
			if strings.Contains(tc.body, "9007199254740993") {
				assert.Contains(t, string(filtered), "9007199254740993", "no number re-encoding")
			}
		})
	}
	for _, info := range []*relaycommon.RelayInfo{nil, {}, {ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 86}}} {
		payload := []byte(`{"copilot_usage":{},"usage":{"prompt_tokens":9}}`)
		assert.Equal(t, payload, relaycommon.FilterCopilotResponseMetadata(info, payload), "unconfigured channels must remain unchanged")
		assert.Equal(t, string(payload), relaycommon.FilterCopilotResponseMetadataString(info, string(payload)))
	}
}

func TestCopilotMetadataChannelSettingsRoundTrip(t *testing.T) {
	for _, id := range []int{11, 12, 86} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			channel := &model.Channel{Id: id, OtherSettings: `{"allow_service_tier":true,"strip_copilot_metadata":true}`}
			settings := channel.GetOtherSettings()
			require.True(t, settings.StripCopilotMetadata)
			require.True(t, settings.AllowServiceTier)
			channel.SetOtherSettings(settings)
			assert.True(t, channel.GetOtherSettings().StripCopilotMetadata)
			assert.True(t, channel.GetOtherSettings().AllowServiceTier)
		})
	}
	assert.False(t, (&model.Channel{OtherSettings: `{}`}).GetOtherSettings().StripCopilotMetadata)
}

const copilotResponsesFixture = `{"id":"resp_original","model":"gpt-test","status":"completed","output_text":null,"copilot_usage":{"token_count":999999},"output":[{"type":"reasoning","id":"rs_original","encrypted_content":"original-signature"},{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Copilot stays in the answer"}]},{"type":"web_search_call","id":"search_1","status":"completed"},{"type":"image_generation_call","id":"image_1","status":"completed","result":"original-image"},{"type":"function_call","name":"plain_tool","arguments":"{\"copilot_usage\":\"user data\"}"}],"usage":{"input_tokens":9,"output_tokens":7,"total_tokens":16,"input_tokens_details":{"cached_tokens":5},"output_tokens_details":{"reasoning_tokens":4}}}`

const copilotChatFixture = `{"id":"chatcmpl_original","model":"gpt-test","object":"chat.completion","created":1234,"copilot_usage":{"token_count":999999},"choices":[{"index":0,"message":{"role":"assistant","content":"Copilot stays in the answer","reasoning_content":"original reasoning"},"finish_reason":"stop"}],"usage":{"prompt_tokens":9,"completion_tokens":7,"total_tokens":16,"prompt_tokens_details":{"cached_tokens":5},"completion_tokens_details":{"reasoning_tokens":4}}}`

const copilotClaudeFixture = `{"id":"msg_original","model":"claude-test","type":"message","role":"assistant","copilot_usage":{"token_count":999999},"content":[{"type":"thinking","thinking":"original thinking","signature":"original-signature"},{"type":"text","text":"Copilot stays in the answer"},{"type":"tool_use","id":"tool_1","name":"plain_tool","input":{"copilot_usage":"user data"}}],"stop_reason":"end_turn","usage":{"input_tokens":9,"output_tokens":7,"cache_read_input_tokens":5,"cache_creation_input_tokens":3,"cache_creation":{"ephemeral_5m_input_tokens":2,"ephemeral_1h_input_tokens":1},"server_tool_use":{"web_search_requests":2}}}`

func TestCopilotMetadataRelayPreservesBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = oldTimeout })
	chatStream := "data: " + strings.Replace(copilotChatFixture, `"message":`, `"delta":`, 1) + "\n\ndata: [DONE]\n\n"
	responsesStream := `data: {"type":"response.output_text.delta","delta":"Copilot stays in the answer","copilot_trace":"x"}` + "\n\n" +
		`data: {"type":"response.completed","copilot_trace":"x","response":` + copilotResponsesFixture + "}\n\ndata: [DONE]\n\n"
	claudeStream := strings.Join([]string{
		`data: {"type":"message_start","copilot_trace":"x","message":{"id":"msg_original","model":"claude-test","role":"assistant","content":[],"copilot_usage":{"token_count":999999},"usage":{"input_tokens":9,"output_tokens":0,"cache_read_input_tokens":5,"cache_creation_input_tokens":3,"cache_creation":{"ephemeral_5m_input_tokens":2,"ephemeral_1h_input_tokens":1}}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""},"copilot_trace":"x"}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Copilot stays in the answer"},"copilot_trace":"x"}`,
		`data: {"type":"content_block_stop","index":0,"copilot_trace":"x"}`,
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"thinking","thinking":""},"copilot_trace":"x"}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"thinking_delta","thinking":"original thinking"},"copilot_trace":"x"}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"signature_delta","signature":"original-signature"},"copilot_trace":"x"}`,
		`data: {"type":"content_block_stop","index":1,"copilot_trace":"x"}`,
		`data: {"type":"content_block_start","index":2,"content_block":{"type":"tool_use","id":"tool_1","name":"plain_tool","input":{}},"copilot_trace":"x"}`,
		`data: {"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"{\"copilot_usage\":\"user data\"}"},"copilot_trace":"x"}`,
		`data: {"type":"content_block_stop","index":2,"copilot_trace":"x"}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":7,"copilot_usage":{"token_count":999999},"server_tool_use":{"web_search_requests":2}},"copilot_trace":"x"}`,
		`data: {"type":"message_stop","copilot_trace":"x"}`,
	}, "\n\n") + "\n\n"
	for _, tc := range []struct {
		name   string
		format types.RelayFormat
		stream bool
		body   string
	}{
		{"responses JSON", types.RelayFormatOpenAIResponses, false, copilotResponsesFixture},
		{"chat JSON", types.RelayFormatOpenAI, false, copilotChatFixture},
		{"Claude JSON", types.RelayFormatClaude, false, copilotClaudeFixture},
		{"responses SSE", types.RelayFormatOpenAIResponses, true, responsesStream},
		{"chat SSE", types.RelayFormatOpenAI, true, chatStream},
		{"Claude SSE", types.RelayFormatClaude, true, claudeStream},
		{"interrupted responses SSE", types.RelayFormatOpenAIResponses, true, `data: {"type":"response.output_text.delta","delta":"Copilot text","copilot_usage":{}}` + "\n\n"},
	} {
		for _, passthrough := range []bool{false, true} {
			t.Run(tc.name+"/passthrough="+strconv.FormatBool(passthrough), func(t *testing.T) {
				before := runCopilotRelayFixture(t, tc.format, tc.stream, tc.body, false, passthrough)
				after := runCopilotRelayFixture(t, tc.format, tc.stream, tc.body, true, passthrough)
				assert.Equal(t, before.usage, after.usage, "tokens, caches, and BillingUsage must not change")
				if !strings.HasPrefix(tc.name, "interrupted") {
					assert.Equal(t, 9, after.usage.PromptTokens)
					assert.Equal(t, 7, after.usage.CompletionTokens)
					assert.Equal(t, 5, after.usage.PromptTokensDetails.CachedTokens)
					if tc.format == types.RelayFormatClaude {
						assert.Equal(t, 3, after.usage.PromptTokensDetails.CachedCreationTokens)
						assert.Equal(t, 2, after.usage.ClaudeCacheCreation5mTokens)
						assert.Equal(t, 1, after.usage.ClaudeCacheCreation1hTokens)
					}
				}
				assert.Equal(t, before.tools, after.tools, "tool and image accounting must not change")
				assert.Equal(t, before.vendorBytes, after.vendorBytes, "vendor accounting must receive original bytes")
				if len(after.vendorBytes) > 0 {
					assert.Contains(t, string(after.vendorBytes), `"copilot_usage"`)
				}
				filterInfo := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelOtherSettings: dto.ChannelOtherSettings{StripCopilotMetadata: true}}}
				if tc.stream {
					beforeFrames := copilotSSEFrames(before.body)
					afterFrames := copilotSSEFrames(after.body)
					require.Equal(t, len(beforeFrames), len(afterFrames), "event count and ordering must stay intact")
					for i := range beforeFrames {
						assert.JSONEq(t, string(relaycommon.FilterCopilotResponseMetadata(filterInfo, []byte(beforeFrames[i]))), afterFrames[i])
						assert.False(t, gjson.Get(afterFrames[i], "copilot_usage").Exists())
						assert.False(t, gjson.Get(afterFrames[i], "copilot_trace").Exists())
						assert.False(t, gjson.Get(afterFrames[i], "response.copilot_usage").Exists())
						assert.False(t, gjson.Get(afterFrames[i], "message.copilot_usage").Exists())
						assert.False(t, gjson.Get(afterFrames[i], "usage.copilot_usage").Exists())
					}
				} else {
					assert.Equal(t, strconv.Itoa(len(after.body)), after.contentLength)
					assert.JSONEq(t, string(relaycommon.FilterCopilotResponseMetadata(filterInfo, []byte(before.body))), after.body)
					assert.False(t, gjson.Get(after.body, "copilot_usage").Exists())
				}
				assert.Contains(t, after.body, "Copilot", "literal model output must not be rewritten")
			})
		}
	}
}

type copilotRelayResult struct {
	body          string
	contentLength string
	usage         *dto.Usage
	tools         map[string]*relaycommon.BuildInToolInfo
	vendorBytes   []byte
}

func runCopilotRelayFixture(t *testing.T, format types.RelayFormat, stream bool, body string, enabled, passthrough bool) copilotRelayResult {
	t.Helper()
	path, channelType, modelName := "/v1/chat/completions", constant.ChannelTypeOpenAI, "gpt-test"
	if format == types.RelayFormatOpenAIResponses {
		path = "/v1/responses"
	} else if format == types.RelayFormatClaude {
		path, channelType, modelName = "/v1/messages", constant.ChannelTypeAnthropic, "claude-test"
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, path, nil)
	c.Set(common.RequestIdKey, "copilot-metadata-test")
	info := &relaycommon.RelayInfo{
		RelayFormat: format, OriginModelName: modelName, DisablePing: true, ShouldIncludeUsage: true,
		StreamStatus: relaycommon.NewStreamStatus(),
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: 86, ChannelType: channelType, UpstreamModelName: modelName,
			ChannelSetting:       dto.ChannelSettings{PassThroughBodyEnabled: passthrough},
			ChannelOtherSettings: dto.ChannelOtherSettings{StripCopilotMetadata: enabled},
		},
	}
	var vendorBytes []byte
	info.VendorToolUsage = func(raw []byte) map[string]int {
		vendorBytes = bytes.Clone(raw)
		return map[string]int{"vendor-search": 2}
	}
	contentType := "application/json"
	if stream {
		contentType = "text/event-stream"
	}
	resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {contentType}, "Content-Length": {strconv.Itoa(len(body))}}}
	var usage *dto.Usage
	var apiErr *types.NewAPIError
	switch format {
	case types.RelayFormatOpenAIResponses:
		if stream {
			usage, apiErr = openai.OaiResponsesStreamHandler(c, info, resp)
		} else {
			usage, apiErr = openai.OaiResponsesHandler(c, info, resp)
		}
	case types.RelayFormatOpenAI:
		if stream {
			usage, apiErr = openai.OaiStreamHandler(c, info, resp)
		} else {
			usage, apiErr = openai.OpenaiHandler(c, info, resp)
		}
	case types.RelayFormatClaude:
		if stream {
			usage, apiErr = claude.ClaudeStreamHandler(c, resp, info)
		} else {
			usage, apiErr = claude.ClaudeHandler(c, resp, info)
		}
	}
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	require.Equal(t, http.StatusOK, w.Code)
	var tools map[string]*relaycommon.BuildInToolInfo
	if info.ResponsesUsageInfo != nil {
		tools = info.ResponsesUsageInfo.BuiltInTools
	}
	return copilotRelayResult{w.Body.String(), w.Header().Get("Content-Length"), usage, tools, vendorBytes}
}

func copilotSSEFrames(body string) []string {
	var frames []string
	for line := range strings.SplitSeq(body, "\n") {
		if data, ok := strings.CutPrefix(line, "data: "); ok && data != "[DONE]" {
			frames = append(frames, data)
		}
	}
	return frames
}
