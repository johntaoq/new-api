package common

import (
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// FilterCopilotResponseMetadata returns a client-only copy. Callers must keep
// the original payload for usage, tool accounting, and protocol processing.
func FilterCopilotResponseMetadata(info *RelayInfo, payload []byte) []byte {
	if info == nil || info.ChannelMeta == nil || !info.ChannelOtherSettings.StripCopilotMetadata || !gjson.ValidBytes(payload) {
		return payload
	}

	root := gjson.ParseBytes(payload)
	var paths []string
	escape := strings.NewReplacer("\\", "\\\\", ".", "\\.", ":", "\\:")
	// Do not descend into content, output items, tool inputs, or user metadata.
	for _, prefix := range []string{"", "response", "message", "usage", "response.usage", "message.usage"} {
		node := root
		if prefix != "" {
			node = root.Get(prefix)
		}
		if !node.IsObject() {
			continue
		}
		node.ForEach(func(key, _ gjson.Result) bool {
			name := strings.ToLower(key.String())
			if name == "copilot" || strings.HasPrefix(name, "copilot_") {
				path := escape.Replace(key.String())
				if prefix != "" {
					path = prefix + "." + path
				}
				paths = append(paths, path)
			}
			return true
		})
	}

	filtered := payload
	for _, path := range paths {
		var err error
		filtered, err = sjson.DeleteBytes(filtered, path)
		if err != nil {
			return payload
		}
	}
	return filtered
}

// The disabled SSE path keeps the original string without a byte conversion.
func FilterCopilotResponseMetadataString(info *RelayInfo, payload string) string {
	if info == nil || info.ChannelMeta == nil || !info.ChannelOtherSettings.StripCopilotMetadata {
		return payload
	}
	return string(FilterCopilotResponseMetadata(info, []byte(payload)))
}
