package controller

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPic365AuthorizationCodeExchange(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	connection.SetMaxOpenConns(1)
	oldDB, oldRedis := model.DB, common.RedisEnabled
	model.DB, common.RedisEnabled = db, false
	t.Cleanup(func() { model.DB, common.RedisEnabled = oldDB, oldRedis; require.NoError(t, connection.Close()) })
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.AuthFlow{}, &model.Token{}, &model.Pic365Grant{}, &model.Ability{}, &model.Pic365StorageMandate{}, &model.Pic365StorageInvoice{}))
	secret := "pic365-local-client-secret-for-security-tests"
	t.Setenv("PIC365_CLIENT_SECRET", secret)
	t.Setenv("PIC365_ORIGIN", "https://www.pic365.org")
	t.Setenv("PIC365_ISSUER", "https://api.99off.net")
	t.Setenv("PIC365_IMAGE_MODELS", "gpt-image-2,not-in-user-group")
	user := model.User{Username: "pic365-code-test", AuthVersion: 1, Role: 1, Status: common.UserStatusEnabled, Group: "default", AffCode: uuid.NewString()}
	require.NoError(t, db.Create(&user).Error)
	session := model.UserSession{SID: uuid.NewString(), UserID: user.Id, Version: 1, UserAuthVersion: 1, Status: model.UserSessionStatusActive, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "gpt-image-2", ChannelId: 10, Enabled: true}).Error)
	verifier := strings.Repeat("v", 43)
	payload, err := common.Marshal(pic365Authorization{State: strings.Repeat("s", 43), Challenge: service.Pic365PKCEChallenge(verifier), Identity: model.AuthSessionIdentity{UserID: user.Id, SessionID: session.SID, UserAuthVersion: 1, SessionVersion: 1}})
	require.NoError(t, err)
	code, _, err := model.CreateAuthFlow(model.AuthFlowCreate{Purpose: pic365FlowPurpose, Provider: model.Pic365Client, Intent: model.AuthFlowIntentLogin, UserId: user.Id, SessionId: session.SID, Payload: string(payload), ExpiresAt: time.Now().Add(time.Minute)})
	require.NoError(t, err)
	router := gin.New()
	router.POST("/exchange", ExchangePic365Code)
	call := func(clientSecret, proof, redirect string, textModels ...string) *httptest.ResponseRecorder {
		body, marshalErr := common.Marshal(map[string]any{"code": code, "code_verifier": proof, "redirect_uri": redirect, "text_models": textModels})
		require.NoError(t, marshalErr)
		request := httptest.NewRequest("POST", "/exchange", bytes.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+clientSecret)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	callback := "https://www.pic365.org/api/auth/newapi16/callback"
	assert.Equal(t, 401, call("incorrect", verifier, callback).Code)
	assert.Equal(t, 400, call(secret, verifier, "https://evil.test/callback").Code)
	assert.Equal(t, 400, call(secret, strings.Repeat("x", 43), callback).Code)
	assert.Equal(t, 400, call(secret, verifier, callback, "gpt-image-2").Code)
	assert.Equal(t, 400, call(secret, verifier, callback, "gpt-6-luna/../other").Code)
	assert.Equal(t, 400, call(secret, verifier, callback, make([]string, 33)...).Code)
	response := call(secret, verifier, callback)
	require.Equal(t, 200, response.Code)
	var result struct {
		Data struct {
			AccessToken string   `json:"access_token"`
			TokenID     int      `json:"token_id"`
			Models      []string `json:"models"`
			TextModels  []string `json:"text_models"`
			Binding     string   `json:"grant_binding"`
			Group       string   `json:"group"`
			Username    string   `json:"username"`
			ExpiresAt   int64    `json:"expires_at"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
	assert.Equal(t, []string{"gpt-image-2"}, result.Data.Models)
	assert.Empty(t, result.Data.TextModels)
	assert.Equal(t, "default", result.Data.Group)
	assert.Equal(t, user.Username, result.Data.Username)
	assert.Len(t, result.Data.Binding, 43)
	assert.NotContains(t, response.Body.String(), secret)
	assert.InDelta(t, time.Now().Add(5*time.Minute).Unix(), result.Data.ExpiresAt, 2)
	assert.Equal(t, 400, call(secret, verifier, callback).Code, "a successful exchange cannot be replayed")
	key, err := model.GetTokenById(result.Data.TokenID)
	require.NoError(t, err)
	require.NoError(t, model.ValidatePic365Grant(key))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
	c.Request.Header.Set("X-Pic365-Client", secret)
	c.Request.Header.Set("X-Pic365-Grant", result.Data.Binding)
	require.NoError(t, service.ValidatePic365Request(c, key))
	assert.True(t, c.GetBool("pic365_wallet_only"))
	assert.Empty(t, c.Request.Header.Get("X-Pic365-Client"), "the bridge secret is removed before relay/header passthrough")
	assert.Empty(t, c.Request.Header.Get("X-Pic365-Grant"))
	c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
	c.Request.Header.Set("X-Pic365-Client", secret)
	c.Request.Header.Set("X-Pic365-Grant", strings.Repeat("x", 43))
	assert.Error(t, service.ValidatePic365Request(c, key), "a stable key alone cannot revive an old session")
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	c.Request.Header.Set("X-Pic365-Client", secret)
	assert.Error(t, service.ValidatePic365Request(c, key))

	t.Setenv("PIC365_IMAGE_MODELS", "")
	for _, name := range []string{"gemini-nano-banana-2.1", "grok-imagine-image-2.0", "gpt-6.1-sol"} {
		require.NoError(t, db.Create(&model.Ability{Group: "default", Model: name, ChannelId: 10, Enabled: true}).Error)
	}
	require.NoError(t, db.Create(&model.Ability{Group: "other", Model: "gpt-image-private", ChannelId: 10, Enabled: true}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "other", Model: "private-text", ChannelId: 10, Enabled: true}).Error)
	require.NoError(t, db.Create(&model.Ability{Group: "default", Model: "disabled-text", ChannelId: 10, Enabled: false}).Error)
	code, _, err = model.CreateAuthFlow(model.AuthFlowCreate{Purpose: pic365FlowPurpose, Provider: model.Pic365Client, Intent: model.AuthFlowIntentLogin,
		UserId: user.Id, SessionId: session.SID, Payload: string(payload), ExpiresAt: time.Now().Add(time.Minute)})
	require.NoError(t, err)
	response = call(secret, verifier, callback, "gpt-6.1-sol", "gpt-6.1-sol", "private-text", "disabled-text", "missing-text")
	require.Equal(t, 200, response.Code)
	var updated = result
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &updated))
	assert.Equal(t, result.Data.TokenID, updated.Data.TokenID)
	assert.Equal(t, result.Data.AccessToken, updated.Data.AccessToken)
	assert.Equal(t, []string{"gemini-nano-banana-2.1", "gpt-image-2", "grok-imagine-image-2.0"}, updated.Data.Models)
	assert.Equal(t, []string{"gpt-6.1-sol"}, updated.Data.TextModels)
	key, err = model.GetTokenById(updated.Data.TokenID)
	require.NoError(t, err)
	assert.Contains(t, key.GetModelLimits(), "gpt-6.1-sol")
	assert.NotContains(t, key.GetModelLimits(), "private-text")
	assert.NotContains(t, key.GetModelLimits(), "disabled-text")
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("X-Pic365-Client", secret)
	c.Request.Header.Set("X-Pic365-Grant", updated.Data.Binding)
	require.NoError(t, service.ValidatePic365Request(c, key))
	assert.True(t, c.GetBool("pic365_wallet_only"))
	tampered := *key
	tampered.ModelLimits += ",private-text"
	c.Request.Header.Set("X-Pic365-Client", secret)
	c.Request.Header.Set("X-Pic365-Grant", updated.Data.Binding)
	assert.Error(t, service.ValidatePic365Request(c, &tampered))
	t.Run("storage_authorization_and_service_boundary", func(t *testing.T) {
		router.POST("/api/pic365/storage/authorize", func(ctx *gin.Context) {
			if service.ValidatePic365Request(ctx, key) != nil {
				ctx.Status(401)
				return
			}
			ctx.Set("id", user.Id)
			ctx.Set("token_id", key.Id)
			assert.Empty(t, ctx.GetHeader("X-Pic365-Grant"))
			AuthorizePic365Storage(ctx)
		})
		router.GET("/storage-capabilities", GetPic365StorageCapabilities)
		request := func(enabled, consent bool, clientSecret string) *httptest.ResponseRecorder {
			t.Setenv("PIC365_STORAGE_ENABLED", fmt.Sprint(enabled))
			data, err := common.Marshal(map[string]any{"consent": consent, "unit_price_cents_per_gib": 300, "monthly_limit_cents": 1000, "days": 30})
			require.NoError(t, err)
			req := httptest.NewRequest("POST", "/api/pic365/storage/authorize", bytes.NewReader(data))
			req.Header.Set("X-Pic365-Client", clientSecret)
			req.Header.Set("X-Pic365-Grant", updated.Data.Binding)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			return response
		}
		assert.Equal(t, 401, request(false, true, secret).Code)
		assert.Equal(t, 401, request(true, true, "wrong").Code)
		assert.Equal(t, 401, request(true, false, secret).Code)
		require.Equal(t, 200, request(true, true, secret).Code)
		for _, credential := range []string{"wrong", secret} {
			req := httptest.NewRequest("GET", "/storage-capabilities", nil)
			req.Header.Set("Authorization", "Bearer "+credential)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if credential == secret {
				assert.Equal(t, 200, response.Code)
			} else {
				assert.Equal(t, 401, response.Code)
			}
		}
	})

	t.Setenv("PIC365_SOURCE", "newapi40")
	t.Setenv("PIC365_ISSUER", "https://www.unikeyx.com")
	code, _, err = model.CreateAuthFlow(model.AuthFlowCreate{Purpose: pic365FlowPurpose, Provider: model.Pic365Client, Intent: model.AuthFlowIntentLogin,
		UserId: user.Id, SessionId: session.SID, Payload: string(payload), ExpiresAt: time.Now().Add(time.Minute)})
	require.NoError(t, err)
	assert.Equal(t, 400, call(secret, verifier, callback).Code, "40 must not accept the 16 callback URI")
	response = call(secret, verifier, "https://www.pic365.org/api/auth/newapi40/callback")
	require.Equal(t, 200, response.Code)
	assert.Contains(t, response.Body.String(), "https://www.unikeyx.com")
	t.Run("renewal requires the source client and the current session-bound key", func(t *testing.T) {
		router.POST("/renew", RenewPic365Authorization)
		var current = result
		require.NoError(t, common.Unmarshal(response.Body.Bytes(), &current))
		callRenew := func(clientSecret, key, binding string) *httptest.ResponseRecorder {
			request := httptest.NewRequest("POST", "/renew", nil)
			request.Header.Set("Authorization", "Bearer "+key)
			request.Header.Set("X-Pic365-Client", clientSecret)
			request.Header.Set("X-Pic365-Grant", binding)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			return recorder
		}
		assert.Equal(t, 401, callRenew("wrong", current.Data.AccessToken, current.Data.Binding).Code)
		assert.Equal(t, 401, callRenew(secret, current.Data.AccessToken, strings.Repeat("x", 43)).Code)
		assert.Equal(t, 401, callRenew(secret, "sk-"+strings.Repeat("x", 48), current.Data.Binding).Code)
		before := time.Now().Add(-time.Minute).Unix()
		require.NoError(t, db.Model(&model.Token{}).Where("id = ?", current.Data.TokenID).Update("expired_time", before).Error)
		require.NoError(t, db.Model(&model.Pic365Grant{}).Where("token_id = ?", current.Data.TokenID).Update("expires_at", before).Error)
		renewed := callRenew(secret, current.Data.AccessToken, current.Data.Binding)
		require.Equal(t, 200, renewed.Code)
		var renewedData = current
		require.NoError(t, common.Unmarshal(renewed.Body.Bytes(), &renewedData))
		assert.Equal(t, current.Data.Binding, renewedData.Data.Binding)
		assert.InDelta(t, time.Now().Add(5*time.Minute).Unix(), renewedData.Data.ExpiresAt, 2)
		assert.NotContains(t, renewed.Body.String(), current.Data.AccessToken)
		assert.NotContains(t, renewed.Body.String(), secret)
		require.NoError(t, db.Model(&model.UserSession{}).Where("sid = ?", session.SID).Update("revoked_at", time.Now().Unix()).Error)
		assert.Equal(t, 401, callRenew(secret, current.Data.AccessToken, current.Data.Binding).Code)
	})
}
