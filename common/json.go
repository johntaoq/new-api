package common

import (
	"bytes"
	"encoding/json"
	"io"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func TrimUTF8BOM(data []byte) []byte {
	return bytes.TrimPrefix(data, utf8BOM)
}

func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(TrimUTF8BOM(data), v)
}

func UnmarshalJsonStr(data string, v any) error {
	return Unmarshal(StringToByteSlice(data), v)
}

func DecodeJson(reader io.Reader, v any) error {
	return json.NewDecoder(skipUTF8BOMReader(reader)).Decode(v)
}

func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func GetJsonType(data json.RawMessage) string {
	trimmed := bytes.TrimSpace(TrimUTF8BOM(data))
	if len(trimmed) == 0 {
		return "unknown"
	}
	firstChar := trimmed[0]
	switch firstChar {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}

// JsonRawMessageToString returns JSON strings as their decoded value and other JSON values as raw text.
func JsonRawMessageToString(data json.RawMessage) string {
	trimmed := bytes.TrimSpace(TrimUTF8BOM(data))
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return ""
	}
	if trimmed[0] != '"' {
		return string(trimmed)
	}
	var value string
	if err := Unmarshal(trimmed, &value); err != nil {
		return string(trimmed)
	}
	return value
}

func skipUTF8BOMReader(reader io.Reader) io.Reader {
	if reader == nil {
		return bytes.NewReader(nil)
	}

	prefix := make([]byte, len(utf8BOM))
	n, err := io.ReadFull(reader, prefix)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return io.MultiReader(bytes.NewReader(prefix[:n]), reader)
	}

	prefix = prefix[:n]
	if bytes.Equal(prefix, utf8BOM) {
		return reader
	}

	return io.MultiReader(bytes.NewReader(prefix), reader)
}
