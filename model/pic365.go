package model

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const Pic365Client = "pic365"
const Pic365AuthorizationTTL = 5 * time.Minute

var ErrPic365GrantInvalid = errors.New("Pic365 authorization is unavailable; enter Pic365 from your dashboard again")

// A stable dedicated key is scoped to the originating dashboard session.
type Pic365Grant struct {
	TokenID        int    `gorm:"primaryKey;autoIncrement:false"`
	UserID         int    `gorm:"not null;index"`
	Subject        string `gorm:"type:varchar(64);not null"`
	SessionID      string `gorm:"type:varchar(64);not null"`
	AuthVersion    int64  `gorm:"not null"`
	SessionVersion int64  `gorm:"not null"`
	Models         string `gorm:"type:text;not null"`
	ExpiresAt      int64  `gorm:"not null"`
}

func CreatePic365GrantWithTx(tx *gorm.DB, identity AuthSessionIdentity, models string, expiresAt int64) (*Token, string, error) {
	if models == "" || expiresAt <= time.Now().Unix() {
		return nil, "", ErrPic365GrantInvalid
	}
	var user User
	if err := lockForUpdate(tx).First(&user, identity.UserID).Error; err != nil {
		return nil, "", err
	}
	if err := ValidateAuthSessionWithTx(tx, identity); err != nil {
		return nil, "", err
	}
	var session UserSession
	if err := tx.Where("sid = ? AND user_id = ?", identity.SessionID, user.Id).First(&session).Error; err != nil {
		return nil, "", err
	}
	expiresAt = min(expiresAt, session.ExpiresAt)
	if user.Pic365Subject == "" {
		user.Pic365Subject = uuid.NewString()
		if err := tx.Model(&user).Update("pic365_subject", user.Pic365Subject).Error; err != nil {
			return nil, "", err
		}
	}
	var existing []Token
	if err := tx.Where("user_id = ? AND delegated_client = ?", user.Id, Pic365Client).Order("id desc").Find(&existing).Error; err != nil {
		return nil, "", err
	}
	token := &Token{UserId: user.Id, Name: "Pic365", CreatedTime: time.Now().Unix(), DelegatedClient: Pic365Client}
	if len(existing) > 0 {
		token = &existing[0]
		var previous Pic365Grant
		if err := tx.First(&previous, "token_id = ?", token.Id).Error; err != nil || previous.Subject != user.Pic365Subject ||
			(token.Status != common.TokenStatusEnabled && token.Status != common.TokenStatusExpired) {
			return nil, "", ErrPic365GrantInvalid
		}
		if err := invalidateTokensCache(existing); err != nil {
			return nil, "", err
		}
		// Password changes rotate the credential; ordinary launches retain it.
		if previous.AuthVersion != identity.UserAuthVersion {
			token.Key = ""
		}
	} else {
		var count int64
		if err := tx.Model(&Token{}).Where("user_id = ?", user.Id).Count(&count).Error; err != nil {
			return nil, "", err
		}
		if count >= int64(operation_setting.GetMaxUserTokens()) {
			return nil, "", ErrPic365GrantInvalid
		}
	}
	if token.Key == "" {
		key, err := common.GenerateKey()
		if err != nil {
			return nil, "", err
		}
		token.Key = key
	}
	token.Name, token.Status, token.Group = "Pic365", common.TokenStatusEnabled, "default"
	token.ExpiredTime, token.UnlimitedQuota = expiresAt, true
	token.ModelLimitsEnabled, token.ModelLimits = true, models
	token.AllowUserQuotaQuery, token.CrossGroupRetry, token.AutoGroups = true, false, ""
	if token.Id == 0 {
		if err := tx.Create(token).Error; err != nil {
			return nil, "", err
		}
	} else {
		if err := tx.Model(token).Updates(map[string]any{
			"key": token.Key, "name": token.Name, "status": token.Status, "group": token.Group,
			"expired_time": token.ExpiredTime, "unlimited_quota": true, "model_limits_enabled": true,
			"model_limits": models, "allow_user_quota_query": true, "cross_group_retry": false, "auto_groups": "",
		}).Error; err != nil {
			return nil, "", err
		}
	}
	// Retire only old bridge-created duplicates, never ordinary user keys.
	if len(existing) > 1 {
		ids := make([]int, 0, len(existing)-1)
		for _, duplicate := range existing[1:] {
			ids = append(ids, duplicate.Id)
		}
		if err := tx.Where("token_id IN ? AND user_id = ?", ids, user.Id).Delete(&Pic365Grant{}).Error; err != nil {
			return nil, "", err
		}
		if err := tx.Where("id IN ? AND user_id = ? AND delegated_client = ?", ids, user.Id, Pic365Client).Delete(&Token{}).Error; err != nil {
			return nil, "", err
		}
	}
	grant := Pic365Grant{
		TokenID: token.Id, UserID: user.Id, Subject: user.Pic365Subject,
		SessionID: identity.SessionID, AuthVersion: identity.UserAuthVersion,
		SessionVersion: identity.SessionVersion, Models: models, ExpiresAt: expiresAt,
	}
	if err := tx.Save(&grant).Error; err != nil {
		return nil, "", err
	}
	return token, user.Pic365Subject, nil
}

func ValidatePic365Grant(token *Token) error {
	return validatePic365GrantWithTx(DB, token, false)
}

func validatePic365GrantWithTx(tx *gorm.DB, token *Token, renewal bool) error {
	if token == nil || token.DelegatedClient != Pic365Client ||
		(token.Status != common.TokenStatusEnabled && !(renewal && token.Status == common.TokenStatusExpired)) {
		return ErrPic365GrantInvalid
	}
	var stored Token
	if err := tx.First(&stored, token.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPic365GrantInvalid
		}
		return err
	}
	if stored.UserId != token.UserId ||
		stored.Key != token.Key || stored.DelegatedClient != Pic365Client || stored.Status != token.Status ||
		stored.Group != "default" || token.Group != "default" || !stored.ModelLimitsEnabled || stored.ModelLimits != token.ModelLimits ||
		stored.CrossGroupRetry || (!renewal && stored.ExpiredTime <= time.Now().Unix()) {
		return ErrPic365GrantInvalid
	}
	var grant Pic365Grant
	if err := tx.First(&grant, "token_id = ?", token.Id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPic365GrantInvalid
		}
		return err
	}
	// Renewal can extend expiry after TokenAuth read its snapshot. Authoritative
	// grant/key expiry must agree, but an in-flight snapshot need not be current.
	if grant.UserID != token.UserId || grant.ExpiresAt != stored.ExpiredTime || grant.ExpiresAt <= 0 ||
		(!renewal && grant.ExpiresAt <= time.Now().Unix()) || token.Group != "default" ||
		!token.ModelLimitsEnabled || token.ModelLimits != grant.Models || token.CrossGroupRetry {
		return ErrPic365GrantInvalid
	}
	var user User
	if err := tx.First(&user, grant.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPic365GrantInvalid
		}
		return err
	}
	if user.Status != common.UserStatusEnabled || user.Pic365Subject != grant.Subject || user.AuthVersion != grant.AuthVersion {
		return ErrPic365GrantInvalid
	}
	var session UserSession
	if err := tx.Where("sid = ? AND user_id = ?", grant.SessionID, grant.UserID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPic365GrantInvalid
		}
		return err
	}
	if session.Status != UserSessionStatusActive || session.RevokedAt != 0 || session.ExpiresAt <= time.Now().Unix() ||
		session.Version != grant.SessionVersion || session.UserAuthVersion != grant.AuthVersion {
		return ErrPic365GrantInvalid
	}
	return nil
}

// Stable keys need a separate session binding so a later login cannot revive an old Pic365 authorization.
func GetPic365GrantBinding(token *Token) (string, error) {
	return getPic365GrantBindingWithTx(DB, token)
}

func getPic365GrantBindingWithTx(tx *gorm.DB, token *Token) (string, error) {
	if err := validatePic365GrantWithTx(tx, token, false); err != nil {
		return "", err
	}
	var grant Pic365Grant
	if err := tx.First(&grant, "token_id = ?", token.Id).Error; err != nil {
		return "", ErrPic365GrantInvalid
	}
	return pic365GrantBinding(token, &grant, false), nil
}

func pic365GrantBinding(token *Token, grant *Pic365Grant, legacy bool) string {
	payload := fmt.Sprintf("pic365-v2:%d:%s:%s:%d:%d:%s", grant.UserID, grant.Subject, grant.SessionID,
		grant.AuthVersion, grant.SessionVersion, grant.Models)
	if legacy {
		payload = fmt.Sprintf("pic365-v1:%d:%s:%s:%d:%d:%d:%s", grant.UserID, grant.Subject, grant.SessionID,
			grant.AuthVersion, grant.SessionVersion, grant.ExpiresAt, grant.Models)
	}
	mac := hmac.New(sha256.New, []byte(token.Key))
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func Pic365GrantBindingMatches(token *Token, binding string) bool {
	if validatePic365GrantWithTx(DB, token, false) != nil {
		return false
	}
	var grant Pic365Grant
	if DB.First(&grant, "token_id = ?", token.Id).Error != nil {
		return false
	}
	return hmac.Equal([]byte(binding), []byte(pic365GrantBinding(token, &grant, false))) ||
		hmac.Equal([]byte(binding), []byte(pic365GrantBinding(token, &grant, true)))
}

// Renewal never extends the source session or revives a disabled/replaced key.
func RenewPic365Grant(key, binding string) (*Token, string, error) {
	var token Token
	if err := DB.Where(map[string]any{"key": key}).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrPic365GrantInvalid
		}
		return nil, "", err
	}
	if token.DelegatedClient != Pic365Client {
		return nil, "", ErrPic365GrantInvalid
	}
	var nextBinding string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).First(&user, token.UserId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPic365GrantInvalid
			}
			return err
		}
		if err := tx.First(&token, token.Id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPic365GrantInvalid
			}
			return err
		}
		if token.Key != key {
			return ErrPic365GrantInvalid
		}
		if err := validatePic365GrantWithTx(tx, &token, true); err != nil {
			return err
		}
		var grant Pic365Grant
		if err := tx.First(&grant, "token_id = ?", token.Id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPic365GrantInvalid
			}
			return err
		}
		if !hmac.Equal([]byte(binding), []byte(pic365GrantBinding(&token, &grant, false))) &&
			!hmac.Equal([]byte(binding), []byte(pic365GrantBinding(&token, &grant, true))) {
			return ErrPic365GrantInvalid
		}
		var session UserSession
		if err := tx.First(&session, "sid = ? AND user_id = ?", grant.SessionID, token.UserId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPic365GrantInvalid
			}
			return err
		}
		if err := invalidateTokensCache([]Token{token}); err != nil {
			return err
		}
		grant.ExpiresAt = min(time.Now().Add(Pic365AuthorizationTTL).Unix(), session.ExpiresAt)
		token.ExpiredTime, token.Status = grant.ExpiresAt, common.TokenStatusEnabled
		if err := tx.Model(&token).Updates(map[string]any{"expired_time": token.ExpiredTime, "status": token.Status}).Error; err != nil {
			return err
		}
		if err := tx.Save(&grant).Error; err != nil {
			return err
		}
		nextBinding = pic365GrantBinding(&token, &grant, false)
		return nil
	})
	return &token, nextBinding, err
}
