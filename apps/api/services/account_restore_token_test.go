package services

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
)

// withRestoreKey installs a signing key. config.AppConfig is a *Config that is
// nil until Load() runs, so tests must supply the whole struct.
func withRestoreKey(t *testing.T) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig = &config.Config{
		BFFInternalHMACKey: []byte("test-key-at-least-16-bytes-long!"),
	}
	t.Cleanup(func() { config.AppConfig = prev })
}

func TestRestoreToken_RoundTrips(t *testing.T) {
	withRestoreKey(t)
	userID := uuid.New()

	gotID, gotUID, err := ParseRestoreToken(MintRestoreToken(userID, "gip-abc"))
	require.NoError(t, err)
	require.Equal(t, userID, gotID)
	require.Equal(t, "gip-abc", gotUID, "token must carry its identity binding")
}

func TestRestoreToken_RejectsTamperedSubject(t *testing.T) {
	withRestoreKey(t)
	token := MintRestoreToken(uuid.New(), "gip-abc")

	// Swap in a different account id, keeping the original signature — the
	// attack this token exists to stop.
	parts := strings.Split(token, ".")
	parts[0] = uuid.New().String()
	_, _, err := ParseRestoreToken(strings.Join(parts, "."))
	require.Error(t, err, "a re-pointed token must not verify")
}

func TestRestoreToken_RejectsForeignKey(t *testing.T) {
	withRestoreKey(t)
	token := MintRestoreToken(uuid.New(), "gip-abc")

	config.AppConfig.BFFInternalHMACKey = []byte("a-completely-different-key-here!")
	_, _, err := ParseRestoreToken(token)
	require.Error(t, err, "a token signed with another key must not verify")
}

func TestRestoreToken_RejectsExpired(t *testing.T) {
	withRestoreKey(t)
	userID := uuid.New()

	// Hand-build an already-expired token with a genuine signature, so this
	// tests the expiry check rather than the signature check.
	past := time.Now().UTC().Add(-time.Minute).Unix()
	payload := restorePayload(userID.String(), "gip-abc", past)
	_, _, err := ParseRestoreToken(payload + "." + signRestorePayload(payload))
	require.Error(t, err)
	require.Contains(t, err.Error(), "expired")
}

func TestRestoreToken_RejectsMalformed(t *testing.T) {
	withRestoreKey(t)
	for _, bad := range []string{"", "a.b", "a.b.c.d.e", "not-a-token"} {
		_, _, err := ParseRestoreToken(bad)
		require.Error(t, err, "input %q must be rejected", bad)
	}
}
