package controller

import (
	"crypto/subtle"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const pic365FlowPurpose = "pic365_authorize"

var pic365OpaqueValue = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
var pic365Verifier = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

type pic365Authorization struct {
	State     string                    `json:"state"`
	Challenge string                    `json:"code_challenge"`
	Identity  model.AuthSessionIdentity `json:"identity"`
}

func GetPic365Entry(c *gin.Context) {
	config, err := service.GetPic365Config()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"enabled": false}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"enabled": true, "start_url": config.Origin + "/api/auth/" + config.Source + "/start",
	}})
}

func AuthorizePic365(c *gin.Context) {
	config, err := service.GetPic365Config()
	identity, ok := middleware.GetSessionAuthIdentity(c)
	if err != nil || !ok {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Pic365 authorization is unavailable"})
		return
	}
	var request pic365Authorization
	if err := c.ShouldBindJSON(&request); err != nil || !pic365OpaqueValue.MatchString(request.State) || !pic365OpaqueValue.MatchString(request.Challenge) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid Pic365 authorization request"})
		return
	}
	request.Identity = identity
	if _, _, err := service.ValidateLoginSession(identity); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Pic365 authorization is unavailable"})
		return
	}
	payload, err := common.Marshal(request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	code, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
		Purpose: pic365FlowPurpose, Provider: model.Pic365Client, Intent: model.AuthFlowIntentLogin,
		UserId: identity.UserID, SessionId: identity.SessionID, Payload: string(payload),
		ExpiresAt: time.Now().Add(model.Pic365AuthorizationTTL),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordUserSecurityAudit(c, identity.UserID, "user.pic365_authorize", map[string]any{"client": model.Pic365Client})
	callback := config.Origin + "/api/auth/" + config.Source + "/callback?" + url.Values{"code": {code}, "state": {request.State}}.Encode()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"redirect_url": callback}})
}

func ExchangePic365Code(c *gin.Context) {
	config, err := service.GetPic365Config()
	if err != nil || !service.Pic365SecretMatches(config.Secret, strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid_client"})
		return
	}
	var request struct {
		Code        string   `json:"code"`
		Verifier    string   `json:"code_verifier"`
		RedirectURI string   `json:"redirect_uri"`
		TextModels  []string `json:"text_models"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || !pic365OpaqueValue.MatchString(request.Code) ||
		!pic365Verifier.MatchString(request.Verifier) || request.RedirectURI != config.Origin+"/api/auth/"+config.Source+"/callback" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_grant"})
		return
	}
	if len(request.TextModels) > 32 || slices.ContainsFunc(request.TextModels, func(name string) bool {
		return !service.IsPic365ModelName(name) || service.IsPic365ImageModel(name)
	}) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_grant"})
		return
	}
	var token *model.Token
	var subject string
	var username string
	var models []string
	var textModels []string
	_, err = model.ConsumeAuthFlowWithAction(request.Code, model.AuthFlowMatch{
		Purpose: pic365FlowPurpose, Provider: model.Pic365Client, Intent: model.AuthFlowIntentLogin,
	}, func(tx *gorm.DB, flow *model.AuthFlow) error {
		var authorization pic365Authorization
		if err := common.UnmarshalJsonStr(flow.Payload, &authorization); err != nil {
			return model.ErrAuthFlowInvalid
		}
		challenge := service.Pic365PKCEChallenge(request.Verifier)
		if subtle.ConstantTimeCompare([]byte(challenge), []byte(authorization.Challenge)) != 1 ||
			authorization.Identity.UserID != flow.UserId || authorization.Identity.SessionID != flow.SessionId {
			return model.ErrAuthFlowInvalid
		}
		var user model.User
		if err := tx.First(&user, flow.UserId).Error; err != nil {
			return err
		}
		username = user.Username
		if _, permitted := service.GetUserUsableGroups(user.Group)["default"]; !permitted {
			return model.ErrPic365GrantInvalid
		}
		var available []string
		if err := tx.Model(&model.Ability{}).Where(map[string]any{"group": "default", "enabled": true}).Distinct("model").Order("model").Pluck("model", &available).Error; err != nil {
			return err
		}
		for _, name := range available {
			if service.IsPic365ImageModel(name) && (len(config.Models) == 0 || slices.Contains(config.Models, name)) {
				models = append(models, name)
			}
		}
		if len(models) == 0 || len(models) > 32 {
			return model.ErrPic365GrantInvalid
		}
		for _, name := range request.TextModels {
			if slices.Contains(available, name) && !slices.Contains(textModels, name) {
				textModels = append(textModels, name)
			}
		}
		var issueErr error
		token, subject, issueErr = model.CreatePic365GrantWithTx(tx, authorization.Identity, strings.Join(append(slices.Clone(models), textModels...), ","), time.Now().Add(model.Pic365AuthorizationTTL).Unix())
		return issueErr
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_grant"})
		return
	}
	binding, err := model.GetPic365GrantBinding(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_grant"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"issuer": config.Issuer, "subject": subject, "user_id": token.UserId, "username": username,
		"access_token": "sk-" + token.Key, "token_id": token.Id, "expires_at": token.ExpiredTime,
		"models": models, "text_models": textModels, "billing_source": "wallet", "group": "default", "grant_binding": binding,
	}})
}

func RenewPic365Authorization(c *gin.Context) {
	config, err := service.GetPic365Config()
	if err != nil || !service.Pic365SecretMatches(config.Secret, c.GetHeader("X-Pic365-Client")) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "invalid_client"})
		return
	}
	key, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer sk-")
	binding := c.GetHeader("X-Pic365-Grant")
	if !ok || len(key) < 32 || len(key) > 128 || !pic365OpaqueValue.MatchString(binding) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "authorization_required"})
		return
	}
	token, nextBinding, err := model.RenewPic365Grant(key, binding)
	if err != nil {
		if err == model.ErrPic365GrantInvalid {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "authorization_required"})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "authorization_unavailable"})
		}
		return
	}
	c.Request.Header.Del("X-Pic365-Client")
	c.Request.Header.Del("X-Pic365-Grant")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"issuer": config.Issuer, "user_id": token.UserId, "token_id": token.Id,
		"expires_at": token.ExpiredTime, "grant_binding": nextBinding,
	}})
}

func GetPic365Balance(c *gin.Context) {
	token, err := model.GetTokenByIds(c.GetInt("token_id"), c.GetInt("id"))
	if err != nil || token.DelegatedClient != model.Pic365Client || !c.GetBool("pic365_wallet_only") || model.ValidatePic365Grant(token) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "authorization_required"})
		return
	}
	quota, err := model.GetUserQuota(token.UserId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"quota": quota, "quota_per_unit": common.QuotaPerUnit, "billing_source": "wallet",
		"expires_at": token.ExpiredTime,
	}})
}
