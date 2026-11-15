package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPic365ClientAndPKCE(t *testing.T) {
	secret := "local-pic365-client-secret-at-least-32-characters"
	t.Setenv("PIC365_CLIENT_SECRET", secret)
	t.Setenv("PIC365_ORIGIN", "https://www.pic365.org")
	t.Setenv("PIC365_ISSUER", "https://api.99off.net")
	t.Setenv("PIC365_IMAGE_MODELS", "gpt-image-2,gpt-image-2")
	config, err := GetPic365Config()
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-image-2"}, config.Models)
	assert.Equal(t, "newapi16", config.Source)
	for _, name := range []string{"gpt-image-2.5-flare", "gemini-3.1-flash-image-preview", "gemini-nano-banana-2.1", "grok-imagine-image-2.0"} {
		assert.True(t, IsPic365ImageModel(name))
	}
	for _, name := range []string{"gpt-6.1-sol", "gemini-3-flash", "grok-4.7", "gpt-image-2/../chat"} {
		assert.False(t, IsPic365ImageModel(name))
	}
	assert.True(t, Pic365SecretMatches(secret, secret))
	assert.False(t, Pic365SecretMatches(secret, "wrong-secret"))
	assert.Equal(t, "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", Pic365PKCEChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"))
	t.Setenv("PIC365_ORIGIN", "https://www.pic365.org/evil?redirect=https://evil.test")
	_, err = GetPic365Config()
	assert.Error(t, err)
}

func TestPic365DoesNotRestrictOrdinaryTokens(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	require.NoError(t, ValidatePic365Request(c, &model.Token{}))
	assert.False(t, c.GetBool("pic365_wallet_only"))
	assert.Error(t, ValidatePic365Request(c, &model.Token{DelegatedClient: model.Pic365Client}))
}

func TestPic365UsesOwnerWalletWithoutChangingOrdinaryBilling(t *testing.T) {
	truncate(t)
	const userID = 76516
	seedUser(t, userID, 500000)
	const tokenID = 765160
	seedToken(t, tokenID, userID, "local-pic365-wallet-test", 0)
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", tokenID).Update("unlimited_quota", true).Error)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
	info := &relaycommon.RelayInfo{UserId: userID, TokenId: tokenID, TokenKey: "local-pic365-wallet-test", TokenUnlimited: true, ForcePreConsume: true,
		OriginModelName: "gpt-image-2", UserSetting: dto.UserSetting{BillingPreference: "subscription_only"}}
	ordinary, apiErr := NewBillingSession(context, info, 100000)
	assert.Nil(t, ordinary)
	require.NotNil(t, apiErr, "ordinary subscription-only requests must not silently use the wallet")
	quota, err := model.GetUserQuota(userID, false)
	require.NoError(t, err)
	assert.Equal(t, 500000, quota)
	context.Set("pic365_wallet_only", true)
	session, apiErr := NewBillingSession(context, info, 100000)
	require.Nil(t, apiErr)
	require.NotNil(t, session)
	assert.Equal(t, BillingSourceWallet, session.funding.Source())
	quota, err = model.GetUserQuota(userID, false)
	require.NoError(t, err)
	assert.Equal(t, 400000, quota)
	require.NoError(t, session.Settle(75000))
	require.NoError(t, session.Settle(75000))
	quota, err = model.GetUserQuota(userID, false)
	require.NoError(t, err)
	assert.Equal(t, 425000, quota, "actual charge is settled once by the existing wallet path")
	var token model.Token
	require.NoError(t, model.DB.First(&token, tokenID).Error)
	assert.Equal(t, 75000, token.UsedQuota)
	_, apiErr = NewBillingSession(context, info, 500000)
	require.NotNil(t, apiErr, "insufficient source wallet must reject before upstream")
}
