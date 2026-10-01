package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/homechef/api/config"
	"github.com/stretchr/testify/require"
)

func TestStripeSlotsIsolateCredentialsAndNeverPromoteTestKeys(t *testing.T) {
	oldConfig, oldBao, oldSM := config.AppConfig, baoClient, secretClient
	config.AppConfig = &config.Config{StripeSecretKey: "sk_test_same", StripePublishableKey: "pk_test_same", StripeKeyID: "mk_same", StripeTestSecretKey: "sk_test_same", StripeTestPublishableKey: "pk_test_same", StripeTestKeyID: "mk_same"}
	baoClient, secretClient = nil, nil
	InvalidateStripe()
	t.Cleanup(func() { config.AppConfig, baoClient, secretClient = oldConfig, oldBao, oldSM; InvalidateStripe() })
	live, test := GetStripeFor("live"), GetStripeFor("test")
	require.NotNil(t, live)
	require.NotNil(t, test)
	require.NotSame(t, live, test)
	require.True(t, live.IsTestMode())
	require.True(t, test.IsTestMode())
	require.Equal(t, "mk_same", live.GetSecretKeyID())
	require.Equal(t, "mk_same", test.GetSecretKeyID())
	InvalidateStripeFor("test")
	require.Same(t, live, GetStripeFor("live"))
	config.AppConfig.StripeTestSecretKey = ""
	require.Nil(t, GetStripeFor("test"), "test slot must not fall back to live slot")
	config.AppConfig.StripeTestSecretKey = "sk_live_forbidden"
	require.Nil(t, GetStripeFor("test"), "test slot must never authenticate live payments")
	config.AppConfig.StripeTestSecretKey = "sk_test_valid"
	config.AppConfig.StripeTestPublishableKey = "pk_live_wrong"
	require.Nil(t, GetStripeFor("test"), "publishable key must match credential environment")
	live.fetchedAt = time.Now().Add(-2 * stripeCacheTTL)
	config.AppConfig.StripeSecretKey = "sk_live_rotated"
	require.Nil(t, GetStripeFor("live"), "invalid rotated credentials must not keep an old client active")
}

func TestStripeSecretNamesAreDistinctAndMatchLegacyLiveSlot(t *testing.T) {
	sk, pk, wh, id := StripeSecretNames("live")
	require.Equal(t, "prod-homechef-stripe-secret-key", sk)
	require.Equal(t, "prod-homechef-stripe-publishable-key", pk)
	require.Equal(t, "prod-homechef-stripe-webhook-secret", wh)
	require.Equal(t, "prod-homechef-stripe-key-id", id)
	sk, pk, wh, id = StripeSecretNames("test")
	require.Equal(t, "prod-homechef-stripe-test-secret-key", sk)
	require.Equal(t, "prod-homechef-stripe-test-publishable-key", pk)
	require.Equal(t, "prod-homechef-stripe-test-webhook-secret", wh)
	require.Equal(t, "prod-homechef-stripe-test-key-id", id)
}

func TestStripeWebhookUsesSelectedSlotSigningSecret(t *testing.T) {
	InvalidateStripe()
	t.Cleanup(InvalidateStripe)
	stripeClients["test"] = &StripeClient{secretKey: "sk_test_fixture", webhookSecret: "whsec_test", fetchedAt: time.Now()}
	stripeClients["live"] = &StripeClient{secretKey: "sk_test_fixture", webhookSecret: "whsec_live_slot", fetchedAt: time.Now()}
	payload := []byte(`{"type":"payment_intent.succeeded"}`)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte("whsec_test"))
	mac.Write([]byte(ts + "." + string(payload)))
	sig := "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	require.True(t, VerifyStripeWebhookSignatureFor("test", payload, sig))
	require.False(t, VerifyStripeWebhookSignatureFor("live", payload, sig))
	require.False(t, VerifyStripeWebhookSignatureFor("test", []byte(`{}`), sig))
}
