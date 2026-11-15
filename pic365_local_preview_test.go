package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/router"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Opt-in preview never uses production databases, credentials or background jobs.
func TestPic365LocalPreview(t *testing.T) {
	if os.Getenv("PIC365_LOCAL_PREVIEW") != "true" {
		t.Skip("local preview is opt-in")
	}
	directory := filepath.Dir(os.Getenv("SQLITE_PATH"))
	require.True(t, filepath.IsAbs(directory) && strings.HasPrefix(filepath.Base(directory), "pic365-local-preview-"))
	require.Empty(t, os.Getenv("SQL_DSN"))
	require.Empty(t, os.Getenv("LOG_SQL_DSN"))
	require.Empty(t, os.Getenv("REDIS_CONN_STRING"))
	require.Empty(t, os.Getenv("FRONTEND_BASE_URL"))
	require.Equal(t, "http://127.0.0.1:3016", os.Getenv("PIC365_ISSUER"))
	require.Equal(t, "http://127.0.0.1:3018", os.Getenv("PIC365_ORIGIN"))
	common.InitEnv()
	common.RedisEnabled, common.MemoryCacheEnabled, common.BatchUpdateEnabled = false, false, false
	logger.SetupLogger()
	ratio_setting.InitRatioSettings()
	require.NoError(t, model.InitDB())
	defer model.CloseDB()
	require.NoError(t, model.InitLogDB())
	require.NoError(t, authz.Init(model.DB))
	model.InitOptionMap()
	model.CheckSetup()
	require.NoError(t, model.EnsureFinanceBillingMonths(time.Now()))
	require.NoError(t, i18n.Init())
	i18n.SetUserLangLoader(model.GetUserLanguage)
	gin.SetMode(gin.ReleaseMode)
	server := gin.New()
	require.NoError(t, server.SetTrustedProxies(nil))
	server.Use(gin.Recovery(), middleware.RequestId(), middleware.Version(), middleware.I18n())
	router.SetRouter(server, router.WebAssets{BuildFS: buildFS, IndexPage: indexPage})
	t.Log("local preview: http://127.0.0.1:3016")
	require.NoError(t, http.ListenAndServe("127.0.0.1:3016", server))
}
