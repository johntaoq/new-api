package middleware

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const utf8BOMLength = 3

func StripJSONBOM() gin.HandlerFunc {
	return func(c *gin.Context) {
		if shouldStripJSONBOM(c.Request) {
			body, stripped := stripUTF8BOMBody(c.Request.Body)
			c.Request.Body = body
			if stripped && c.Request.ContentLength >= utf8BOMLength {
				c.Request.ContentLength -= utf8BOMLength
				c.Request.Header.Set("Content-Length", strconv.FormatInt(c.Request.ContentLength, 10))
			}
		}
		c.Next()
	}
}

func shouldStripJSONBOM(r *http.Request) bool {
	if r == nil || r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 {
		return false
	}

	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	return isJSONContentType(r.Header.Get("Content-Type"))
}

func isJSONContentType(contentType string) bool {
	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.Split(contentType, ";")[0]
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))

	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func stripUTF8BOMBody(body io.ReadCloser) (io.ReadCloser, bool) {
	prefix := make([]byte, utf8BOMLength)
	n, _ := io.ReadFull(body, prefix)
	prefix = prefix[:n]

	trimmed := common.TrimUTF8BOM(prefix)
	if len(trimmed) != len(prefix) {
		return body, true
	}

	return bodyReadCloser{
		Reader: io.MultiReader(bytes.NewReader(prefix), body),
		closer: body,
	}, false
}

type bodyReadCloser struct {
	io.Reader
	closer io.Closer
}

func (b bodyReadCloser) Close() error {
	return b.closer.Close()
}
