package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
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
	"gorm.io/gorm"
)

// This opt-in runner uses the local database without starting production background jobs.
func runFinanceLocalTest() error {
	dsn, err := url.Parse(os.Getenv("SQL_DSN"))
	if err != nil || dsn.Scheme != "postgres" || dsn.Host != "127.0.0.1:55447" ||
		dsn.Path != "/newapi_finance_local" || dsn.Query().Encode() != "sslmode=disable" ||
		dsn.User == nil || dsn.User.Username() != "finance_local" {
		return errors.New("finance local test requires the isolated loopback database")
	}
	if os.Getenv("LOG_SQL_DSN") != "" || os.Getenv("REDIS_CONN_STRING") != "" ||
		os.Getenv("FRONTEND_BASE_URL") != "" || os.Getenv("NODE_TYPE") == "slave" {
		return errors.New("external databases, Redis, and frontend redirects are not allowed in finance local test")
	}
	password := os.Getenv("FINANCE_LOCAL_ADMIN_PASSWORD")
	if len(password) < 24 || len(os.Getenv("SESSION_SECRET")) < 32 {
		return errors.New("finance local test requires independent local credentials")
	}
	if _, err := os.Stat("local-sanitized.ok"); err != nil {
		return errors.New("sanitize the restored local working database before starting")
	}
	common.InitEnv()
	common.RedisEnabled = false
	common.MemoryCacheEnabled = false
	common.BatchUpdateEnabled = false
	logger.SetupLogger()
	ratio_setting.InitRatioSettings()
	if err := model.InitDB(); err != nil {
		return err
	}
	defer model.CloseDB()
	if err := model.InitLogDB(); err != nil {
		return err
	}
	if err := authz.Init(model.DB); err != nil {
		return err
	}
	model.InitOptionMap()
	if err := model.EnsureFinanceBillingMonths(time.Now()); err != nil {
		return err
	}
	financeContext, stopFinance := context.WithCancel(context.Background())
	defer stopFinance()
	model.StartFinanceMonthlyBilling(financeContext)
	model.CheckSetup()
	if err := i18n.Init(); err != nil {
		return err
	}
	i18n.SetUserLangLoader(model.GetUserLanguage)
	var admin model.User
	err = model.DB.Where("username = ?", "finance-local-admin").First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, err := common.HashAccountPassword(password)
		if err != nil {
			return err
		}
		admin = model.User{
			Username: "finance-local-admin", DisplayName: "Local Finance",
			Password: hash, Role: common.RoleRootUser, Status: common.UserStatusEnabled,
			Group: "default", AffCode: common.GetRandomString(16),
		}
		if err := model.DB.Create(&admin).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	os.Unsetenv("FINANCE_LOCAL_ADMIN_PASSWORD")

	gin.SetMode(gin.ReleaseMode)
	server := gin.New()
	if err := server.SetTrustedProxies(nil); err != nil {
		return err
	}
	server.Use(gin.Recovery(), middleware.RequestId(), middleware.Version(), middleware.I18n())
	router.SetRouter(server, router.WebAssets{BuildFS: buildFS, IndexPage: indexPage})
	common.SysLog("Isolated local finance test listening on http://127.0.0.1:3017")
	return http.ListenAndServe("127.0.0.1:3017", server)
}
