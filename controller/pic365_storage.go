package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func pic365StorageClient(c *gin.Context) bool {
	config, err := service.GetPic365Config()
	if err != nil || !config.StorageEnabled || !service.Pic365SecretMatches(config.Secret, strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")) {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "PIC365_STORAGE_UNAVAILABLE"})
		return false
	}
	return true
}

func pic365StorageError(c *gin.Context, err error) {
	status, code := http.StatusServiceUnavailable, "PIC365_STORAGE_UNAVAILABLE"
	switch {
	case errors.Is(err, model.ErrPic365StorageAuthorization):
		status, code = http.StatusUnauthorized, "PIC365_STORAGE_AUTHORIZATION_REQUIRED"
	case errors.Is(err, model.ErrPic365StorageLimit):
		status, code = http.StatusPaymentRequired, "PIC365_STORAGE_MONTHLY_LIMIT"
	case errors.Is(err, model.ErrFinanceInsufficientQuota):
		status, code = http.StatusPaymentRequired, "PIC365_STORAGE_INSUFFICIENT_BALANCE"
	case errors.Is(err, model.ErrPic365StorageConflict):
		status, code = http.StatusConflict, "PIC365_STORAGE_INVOICE_CONFLICT"
	case errors.Is(err, gorm.ErrRecordNotFound):
		status, code = http.StatusNotFound, "PIC365_STORAGE_INVOICE_NOT_FOUND"
	}
	c.JSON(status, gin.H{"success": false, "error": code})
}

func GetPic365StorageCapabilities(c *gin.Context) {
	if !pic365StorageClient(c) {
		return
	}
	model.RetryPic365StorageLogs()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"version": model.Pic365StorageVersion, "billing_source": "wallet", "category": model.Pic365StorageCategory}})
}

func AuthorizePic365Storage(c *gin.Context) {
	config, err := service.GetPic365Config()
	token, tokenErr := model.GetTokenByIds(c.GetInt("token_id"), c.GetInt("id"))
	if err != nil || !config.StorageEnabled || tokenErr != nil || !c.GetBool("pic365_wallet_only") {
		pic365StorageError(c, model.ErrPic365StorageAuthorization)
		return
	}
	var req struct {
		Price   int64 `json:"unit_price_cents_per_gib"`
		Limit   int64 `json:"monthly_limit_cents"`
		Days    int   `json:"days"`
		Consent bool  `json:"consent"`
	}
	if c.ShouldBindJSON(&req) != nil || !req.Consent {
		pic365StorageError(c, model.ErrPic365StorageAuthorization)
		return
	}
	mandate, secret, err := model.CreatePic365StorageMandate(token, c.GetString("pic365_storage_binding"), req.Price, req.Limit, req.Days)
	if err != nil {
		pic365StorageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"mandate": mandate, "credential": secret}})
}

func ChargePic365Storage(c *gin.Context) {
	if !pic365StorageClient(c) {
		return
	}
	var req model.Pic365StorageChargeRequest
	if c.ShouldBindJSON(&req) != nil {
		pic365StorageError(c, model.ErrPic365StorageConflict)
		return
	}
	invoice, err := model.ChargePic365Storage(c.GetHeader("X-Pic365-Storage"), req)
	if err != nil {
		pic365StorageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": invoice})
}

func GetPic365StorageStatus(c *gin.Context) {
	if !pic365StorageClient(c) {
		return
	}
	var req struct {
		InvoiceID string `json:"invoice_id"`
	}
	if c.ShouldBindJSON(&req) != nil {
		pic365StorageError(c, model.ErrPic365StorageConflict)
		return
	}
	if req.InvoiceID != "" {
		invoice, err := model.GetPic365StorageReceipt(c.GetHeader("X-Pic365-Storage"), req.InvoiceID)
		if err != nil {
			pic365StorageError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": invoice})
		return
	}
	mandate, err := model.GetPic365StorageMandate(c.GetHeader("X-Pic365-Storage"))
	if err != nil {
		pic365StorageError(c, err)
		return
	}
	var user model.User
	if err := model.DB.First(&user, mandate.UserID).Error; err != nil {
		pic365StorageError(c, model.ErrPic365StorageAuthorization)
		return
	}
	active := model.ValidatePic365StorageMandate(mandate, &user) == nil
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"mandate": mandate, "active": active}})
}

func RevokePic365Storage(c *gin.Context) {
	if !pic365StorageClient(c) {
		return
	}
	if err := model.RevokePic365StorageMandate(c.GetHeader("X-Pic365-Storage")); err != nil {
		pic365StorageError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
