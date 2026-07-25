package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func setupMFASettingsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupMFADB(t)
	require.NoError(t, db.Exec(`CREATE TABLE user_mfa_settings (user_id TEXT PRIMARY KEY,
		enabled INTEGER DEFAULT 0, email_enrolled INTEGER DEFAULT 0, phone_enrolled INTEGER DEFAULT 0,
		phone_e164_enc TEXT DEFAULT '', phone_e164_bidx TEXT DEFAULT '',
		enrolled_at DATETIME, disabled_at DATETIME, updated_at DATETIME)`).Error)
	return db
}

func seedMFA(t *testing.T, db *gorm.DB, userID uuid.UUID, enabled, email, phone bool) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Create(&models.UserMFASettings{
		UserID: userID, Enabled: enabled,
		EmailEnrolled: email, PhoneEnrolled: phone,
		PhoneE164Enc: "+919876543210",
		EnrolledAt:   &now, UpdatedAt: now,
	}).Error)
}

// The masked hint is served to whoever holds the password, so it must be
// recognisable to the owner and useless to anyone probing the account.
func TestMaskEmail(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"samyak@gmail.com", "s•••k@gmail.com"},
		{"ab@x.com", "••@x.com"},
		{"a@x.com", "•@x.com"},
		{"no-at-sign", "•••"},
		{"", "•••"},
	} {
		require.Equal(t, tc.want, MaskEmail(tc.in), "MaskEmail(%q)", tc.in)
	}
}

func TestMaskEmail_NeverLeaksWholeLocalPart(t *testing.T) {
	const email = "samyakrout@gmail.com"
	masked := MaskEmail(email)
	require.NotContains(t, masked, "samyakrout")
	require.Contains(t, masked, "@gmail.com")
}

func TestMaskPhone(t *testing.T) {
	masked := MaskPhone("+919876543210")
	require.NotContains(t, masked, "98765432")
	require.True(t, strings.HasSuffix(masked, "10"))
	require.Equal(t, "•••", MaskPhone("123"))
}

func TestEvaluateMFA_FlagOffIsAlwaysAllowed(t *testing.T) {
	db := setupMFASettingsDB(t)
	withMiniredis(t)
	uid := uuid.New()
	seedMFA(t, db, uid, true, true, true)

	req, err := EvaluateMFA(context.Background(), db, false, uid, "vendor", "")
	require.NoError(t, err)
	require.False(t, req.Required)
}

func TestEvaluateMFA_NoSettingsRowIsAllowed(t *testing.T) {
	db := setupMFASettingsDB(t)
	withMiniredis(t)

	req, err := EvaluateMFA(context.Background(), db, true, uuid.New(), "vendor", "")
	require.NoError(t, err)
	require.False(t, req.Required)
}

func TestEvaluateMFA_EnabledUntrustedDeviceIsChallenged(t *testing.T) {
	db := setupMFASettingsDB(t)
	withMiniredis(t)
	uid := uuid.New()
	seedMFA(t, db, uid, true, true, true)

	req, err := EvaluateMFA(context.Background(), db, true, uid, "vendor", "")
	require.NoError(t, err)
	require.True(t, req.Required)
	require.ElementsMatch(t, []MFAChannel{MFAChannelEmail, MFAChannelPhone}, req.Channels)
}

func TestEvaluateMFA_TrustedDeviceSkipsChallenge(t *testing.T) {
	db := setupMFASettingsDB(t)
	withMiniredis(t)
	uid := uuid.New()
	seedMFA(t, db, uid, true, true, false)

	token, err := IssueTrustedDevice(db, uid, "vendor", "Pixel", "android")
	require.NoError(t, err)

	req, err := EvaluateMFA(context.Background(), db, true, uid, "vendor", token)
	require.NoError(t, err)
	require.False(t, req.Required)
}

// A device trusted for one app must still be challenged on another.
func TestEvaluateMFA_TrustIsPerApp(t *testing.T) {
	db := setupMFASettingsDB(t)
	withMiniredis(t)
	uid := uuid.New()
	seedMFA(t, db, uid, true, true, false)

	token, err := IssueTrustedDevice(db, uid, "vendor", "Pixel", "android")
	require.NoError(t, err)

	req, err := EvaluateMFA(context.Background(), db, true, uid, "admin", token)
	require.NoError(t, err)
	require.True(t, req.Required)
}

func TestEvaluateMFA_ElevatedSessionSkipsChallenge(t *testing.T) {
	db := setupMFASettingsDB(t)
	withMiniredis(t)
	ctx := context.Background()
	uid := uuid.New()
	seedMFA(t, db, uid, true, true, false)

	token, err := ElevateSession(ctx, uid, "vendor")
	require.NoError(t, err)

	req, err := EvaluateMFA(ctx, db, true, uid, "vendor", token)
	require.NoError(t, err)
	require.False(t, req.Required)
}

func TestElevation_ExpiresAndIsScoped(t *testing.T) {
	mr := withMiniredis(t)
	ctx := context.Background()
	uid := uuid.New()

	token, err := ElevateSession(ctx, uid, "vendor")
	require.NoError(t, err)
	require.True(t, IsSessionElevated(ctx, uid, "vendor", token))

	// Wrong app and wrong user must not match.
	require.False(t, IsSessionElevated(ctx, uid, "admin", token))
	require.False(t, IsSessionElevated(ctx, uuid.New(), "vendor", token))

	mr.FastForward(elevationTTL + time.Minute)
	require.False(t, IsSessionElevated(ctx, uid, "vendor", token))
}

func TestElevation_FailsClosedWhenRedisDown(t *testing.T) {
	prev := SetRedisClientForTest(nil)
	t.Cleanup(func() { SetRedisClientForTest(prev) })
	require.False(t, IsSessionElevated(context.Background(), uuid.New(), "vendor", "any-token"))
}

// Enabled with nothing enrolled would be an unrecoverable lockout, so the gate
// lets it through and leaves the settings screen to repair the state.
func TestEvaluateMFA_EnabledWithNoChannelsIsNotALockout(t *testing.T) {
	db := setupMFASettingsDB(t)
	withMiniredis(t)
	uid := uuid.New()
	seedMFA(t, db, uid, true, false, false)

	req, err := EvaluateMFA(context.Background(), db, true, uid, "vendor", "")
	require.NoError(t, err)
	require.False(t, req.Required)
}

func TestDisableMFA_RevokesDevicesAndBurnsBackupCodes(t *testing.T) {
	db := setupMFASettingsDB(t)
	withMiniredis(t)
	uid := uuid.New()
	seedMFA(t, db, uid, true, true, true)

	token, err := IssueTrustedDevice(db, uid, "vendor", "Pixel", "android")
	require.NoError(t, err)
	codes, err := GenerateBackupCodes(db, uid, testBackupKey)
	require.NoError(t, err)

	require.NoError(t, DisableMFA(db, uid))

	// Trust granted under the old setup must not survive.
	require.False(t, IsDeviceTrusted(db, uid, "vendor", token))
	// Nor may a leaked backup sheet.
	_, err = RedeemBackupCode(db, uid, codes[0], testBackupKey)
	require.ErrorIs(t, err, ErrBackupCodesMissing)

	s, err := GetMFASettings(db, uid)
	require.NoError(t, err)
	require.False(t, s.Enabled)
	require.False(t, s.EmailEnrolled)
	require.False(t, s.PhoneEnrolled)
}

func TestChallengeSubject_RejectsUnenrolledChannel(t *testing.T) {
	s := models.UserMFASettings{EmailEnrolled: true}
	_, _, err := ChallengeSubject(s, "chef@fe3dr.com", MFAChannelPhone)
	require.ErrorIs(t, err, ErrMFAChannelNotEnrolled)

	_, masked, err := ChallengeSubject(s, "chef@fe3dr.com", MFAChannelEmail)
	require.NoError(t, err)
	require.Equal(t, "c•••f@fe3dr.com", masked)

	_, _, err = ChallengeSubject(s, "chef@fe3dr.com", MFAChannel("carrier-pigeon"))
	require.ErrorIs(t, err, ErrMFAChannelUnknown)
}
