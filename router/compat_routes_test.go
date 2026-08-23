package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func TestCompatibilityRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(sessions.Sessions("session", cookie.NewStore([]byte("compat-routes-test"))))
	SetApiRouter(r)
	SetVideoRouter(r)

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/channel?tag_mode=false&id_sort=false&p=1&page_size=20"},
		{method: http.MethodPost, path: "/api/channel", body: "{}"},
		{method: http.MethodPut, path: "/api/channel", body: "{}"},
		{method: http.MethodPost, path: "/api/v3/contents/generations/tasks", body: `{"model":"doubao-seedance-2-0-260128","content":[{"type":"text","text":"test"}]}`},
		{method: http.MethodGet, path: "/api/v3/contents/generations/tasks/task_test"},
		{method: http.MethodGet, path: "/v1/videos"},
		{method: http.MethodPost, path: "/v1/videos", body: "{}"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Fatalf("%s %s returned 404; route is not registered", tt.method, tt.path)
		}
	}
}
