package service

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type Pic365Config struct {
	Origin         string
	Issuer         string
	Secret         string
	Models         []string
	Source         string
	StorageEnabled bool
}

func GetPic365Config() (Pic365Config, error) {
	config := Pic365Config{
		Origin:         strings.TrimRight(os.Getenv("PIC365_ORIGIN"), "/"),
		Issuer:         strings.TrimRight(os.Getenv("PIC365_ISSUER"), "/"),
		Secret:         os.Getenv("PIC365_CLIENT_SECRET"),
		Source:         os.Getenv("PIC365_SOURCE"),
		StorageEnabled: os.Getenv("PIC365_STORAGE_ENABLED") == "true",
	}
	if config.Source == "" {
		config.Source = "newapi16"
	}
	if !slices.Contains([]string{"newapi16", "newapi40"}, config.Source) {
		return config, errors.New("Pic365 source is invalid")
	}
	if len(config.Secret) < 32 {
		return config, errors.New("Pic365 integration is not configured")
	}
	for _, value := range []string{config.Origin, config.Issuer} {
		u, err := url.Parse(value)
		if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" ||
			(u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost"))) {
			return config, errors.New("Pic365 integration requires fixed HTTPS origins")
		}
	}
	for item := range strings.SplitSeq(os.Getenv("PIC365_IMAGE_MODELS"), ",") {
		item = strings.TrimSpace(item)
		if item != "" && !slices.Contains(config.Models, item) {
			config.Models = append(config.Models, item)
		}
	}
	if len(config.Models) > 32 {
		return config, errors.New("Pic365 image model allowlist is too large")
	}
	return config, nil
}

func IsPic365ImageModel(name string) bool {
	if !IsPic365ModelName(name) {
		return false
	}
	name = strings.ToLower(name)
	return strings.HasPrefix(name, "gpt-image-") || strings.HasPrefix(name, "grok-imagine-image") ||
		(strings.HasPrefix(name, "gemini-") && (strings.Contains(name, "image") || strings.Contains(name, "nano-banana")))
}

func IsPic365ModelName(name string) bool {
	if len(name) == 0 || len(name) > 128 {
		return false
	}
	for _, char := range name {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_' || char == '.') {
			return false
		}
	}
	return true
}

func Pic365SecretMatches(expected, supplied string) bool {
	a, b := sha256.Sum256([]byte(expected)), sha256.Sum256([]byte(supplied))
	return len(expected) >= 32 && subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

func Pic365PKCEChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// Applied only to server-marked delegated keys, never to ordinary API keys.
func ValidatePic365Request(c *gin.Context, token *model.Token) error {
	if token.DelegatedClient == "" {
		return nil
	}
	config, err := GetPic365Config()
	if err != nil || token.DelegatedClient != model.Pic365Client ||
		!Pic365SecretMatches(config.Secret, c.GetHeader("X-Pic365-Client")) {
		return model.ErrPic365GrantInvalid
	}
	allowed := c.Request.Method == "POST" && slices.Contains([]string{
		"/v1/images/generations", "/v1/images/edits", "/v1/chat/completions",
	}, c.Request.URL.Path)
	allowed = allowed || (c.Request.Method == "GET" && c.Request.URL.Path == "/api/pic365/balance")
	allowed = allowed || (config.StorageEnabled && c.Request.Method == "POST" && c.Request.URL.Path == "/api/pic365/storage/authorize")
	if !allowed {
		return model.ErrPic365GrantInvalid
	}
	if err := model.ValidatePic365Grant(token); err != nil {
		return err
	}
	if !model.Pic365GrantBindingMatches(token, c.GetHeader("X-Pic365-Grant")) {
		return model.ErrPic365GrantInvalid
	}
	for _, name := range token.GetModelLimits() {
		// Text models are authorized by the authenticated Pic365 exchange and
		// checked against the stored grant above, not inferred from a model prefix.
		if !IsPic365ModelName(name) || (IsPic365ImageModel(name) && len(config.Models) > 0 && !slices.Contains(config.Models, name)) {
			return model.ErrPic365GrantInvalid
		}
	}
	c.Set("pic365_wallet_only", true)
	if c.Request.URL.Path == "/api/pic365/storage/authorize" {
		binding, err := model.GetPic365GrantBinding(token)
		if err != nil {
			return err
		}
		c.Set("pic365_storage_binding", binding)
	}
	c.Request.Header.Del("X-Pic365-Client")
	c.Request.Header.Del("X-Pic365-Grant")
	return nil
}
