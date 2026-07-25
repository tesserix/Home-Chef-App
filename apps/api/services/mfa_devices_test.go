package services

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// Tables are hand-DDL'd because the models' gen_random_uuid() default cannot
// run on sqlite — same approach as payout_release_test.go.
func setupMFADB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE trusted_devices (id TEXT PRIMARY KEY, user_id TEXT, app TEXT,
			token_hash TEXT, label TEXT, platform TEXT,
			created_at DATETIME, last_seen_at DATETIME, revoked_at DATETIME)`,
		`CREATE TABLE mfa_backup_codes (id TEXT PRIMARY KEY, user_id TEXT,
			code_hmac TEXT, used_at DATETIME, created_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

const testBackupKey = "test-hmac-key-not-a-real-secret"

func TestTrustedDevice_IssueAndVerify(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	token, err := IssueTrustedDevice(db, uid, "vendor", "Pixel 9", "android")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	require.True(t, IsDeviceTrusted(db, uid, "vendor", token))
}

// The token is a bearer credential, so the plaintext must never reach the
// database — a dump of trusted_devices has to be useless.
func TestTrustedDevice_PlaintextNeverStored(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	token, err := IssueTrustedDevice(db, uid, "customer", "iPhone", "ios")
	require.NoError(t, err)

	var rec models.TrustedDevice
	require.NoError(t, db.First(&rec).Error)
	require.NotEqual(t, token, rec.TokenHash)
	require.NotContains(t, rec.TokenHash, token)
}

func TestTrustedDevice_ScopedToApp(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	token, err := IssueTrustedDevice(db, uid, "vendor", "Pixel 9", "android")
	require.NoError(t, err)

	require.True(t, IsDeviceTrusted(db, uid, "vendor", token))
	// A vendor token must not vouch for the admin app.
	require.False(t, IsDeviceTrusted(db, uid, "admin", token))
	require.False(t, IsDeviceTrusted(db, uid, "customer", token))
}

func TestTrustedDevice_ScopedToUser(t *testing.T) {
	db := setupMFADB(t)
	owner, attacker := uuid.New(), uuid.New()

	token, err := IssueTrustedDevice(db, owner, "vendor", "Pixel 9", "android")
	require.NoError(t, err)

	require.False(t, IsDeviceTrusted(db, attacker, "vendor", token))
}

func TestTrustedDevice_UnknownAndEmptyTokens(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	require.False(t, IsDeviceTrusted(db, uid, "vendor", ""))
	require.False(t, IsDeviceTrusted(db, uid, "vendor", "not-a-real-token"))
}

func TestTrustedDevice_RejectsUnknownAppScope(t *testing.T) {
	db := setupMFADB(t)
	_, err := IssueTrustedDevice(db, uuid.New(), "not-an-app", "x", "y")
	require.Error(t, err)
}

func TestTrustedDevice_Revoke(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	token, err := IssueTrustedDevice(db, uid, "vendor", "Pixel 9", "android")
	require.NoError(t, err)
	devices, err := ListTrustedDevices(db, uid)
	require.NoError(t, err)
	require.Len(t, devices, 1)

	require.NoError(t, RevokeTrustedDevice(db, uid, devices[0].ID))
	require.False(t, IsDeviceTrusted(db, uid, "vendor", token))

	remaining, err := ListTrustedDevices(db, uid)
	require.NoError(t, err)
	require.Empty(t, remaining)
}

// Revoking by a guessed ID from another account must do nothing.
func TestTrustedDevice_RevokeIsOwnershipScoped(t *testing.T) {
	db := setupMFADB(t)
	owner, attacker := uuid.New(), uuid.New()

	token, err := IssueTrustedDevice(db, owner, "vendor", "Pixel 9", "android")
	require.NoError(t, err)
	devices, _ := ListTrustedDevices(db, owner)

	require.ErrorIs(t, RevokeTrustedDevice(db, attacker, devices[0].ID), ErrDeviceNotFound)
	require.True(t, IsDeviceTrusted(db, owner, "vendor", token))
}

func TestTrustedDevice_RevokeAll(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	t1, _ := IssueTrustedDevice(db, uid, "vendor", "Pixel", "android")
	t2, _ := IssueTrustedDevice(db, uid, "customer", "iPhone", "ios")

	n, err := RevokeAllTrustedDevices(db, uid)
	require.NoError(t, err)
	require.EqualValues(t, 2, n)
	require.False(t, IsDeviceTrusted(db, uid, "vendor", t1))
	require.False(t, IsDeviceTrusted(db, uid, "customer", t2))
}

func TestBackupCodes_GenerateAndRedeem(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	codes, err := GenerateBackupCodes(db, uid, testBackupKey)
	require.NoError(t, err)
	require.Len(t, codes, BackupCodeCount)

	remaining, err := RedeemBackupCode(db, uid, codes[0], testBackupKey)
	require.NoError(t, err)
	require.Equal(t, BackupCodeCount-1, remaining)
}

func TestBackupCodes_SingleUse(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	codes, err := GenerateBackupCodes(db, uid, testBackupKey)
	require.NoError(t, err)

	_, err = RedeemBackupCode(db, uid, codes[0], testBackupKey)
	require.NoError(t, err)

	// Second use of the same code is indistinguishable from a wrong code.
	_, err = RedeemBackupCode(db, uid, codes[0], testBackupKey)
	require.ErrorIs(t, err, ErrBackupCodeInvalid)
}

func TestBackupCodes_PlaintextNeverStored(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	codes, err := GenerateBackupCodes(db, uid, testBackupKey)
	require.NoError(t, err)

	var rows []models.MFABackupCode
	require.NoError(t, db.Find(&rows).Error)
	for _, r := range rows {
		for _, c := range codes {
			require.NotEqual(t, NormalizeBackupCode(c), r.CodeHMAC)
			require.NotContains(t, r.CodeHMAC, NormalizeBackupCode(c))
		}
	}
}

// Codes are read off paper, so case and dashes must not matter.
func TestBackupCodes_InputNormalization(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	codes, err := GenerateBackupCodes(db, uid, testBackupKey)
	require.NoError(t, err)

	messy := " " + strings.ToLower(strings.ReplaceAll(codes[0], "-", " ")) + " "
	_, err = RedeemBackupCode(db, uid, messy, testBackupKey)
	require.NoError(t, err)
}

func TestBackupCodes_WrongCodeRejected(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	_, err := GenerateBackupCodes(db, uid, testBackupKey)
	require.NoError(t, err)

	_, err = RedeemBackupCode(db, uid, "AAAA-BBBB-CCCC-DDDD", testBackupKey)
	require.ErrorIs(t, err, ErrBackupCodeInvalid)
}

// A different server key must not validate codes minted under the original.
func TestBackupCodes_KeyBound(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	codes, err := GenerateBackupCodes(db, uid, testBackupKey)
	require.NoError(t, err)

	_, err = RedeemBackupCode(db, uid, codes[0], "a-different-key")
	require.ErrorIs(t, err, ErrBackupCodeInvalid)
}

// Regenerating must invalidate the old sheet — users regenerate precisely
// because they think the old one leaked.
func TestBackupCodes_RegenerateInvalidatesOldSet(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	old, err := GenerateBackupCodes(db, uid, testBackupKey)
	require.NoError(t, err)
	fresh, err := GenerateBackupCodes(db, uid, testBackupKey)
	require.NoError(t, err)

	_, err = RedeemBackupCode(db, uid, old[0], testBackupKey)
	require.ErrorIs(t, err, ErrBackupCodeInvalid)

	remaining, err := RedeemBackupCode(db, uid, fresh[0], testBackupKey)
	require.NoError(t, err)
	require.Equal(t, BackupCodeCount-1, remaining)
}

func TestBackupCodes_NoCodesIssued(t *testing.T) {
	db := setupMFADB(t)
	_, err := RedeemBackupCode(db, uuid.New(), "AAAA-BBBB", testBackupKey)
	require.ErrorIs(t, err, ErrBackupCodesMissing)
}

func TestBackupCodes_MissingKeyIsAnError(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	_, err := GenerateBackupCodes(db, uid, "")
	require.ErrorIs(t, err, ErrBackupKeyMissing)
	_, err = RedeemBackupCode(db, uid, "AAAA", "")
	require.ErrorIs(t, err, ErrBackupKeyMissing)
}

func TestBackupCodes_LowBalanceCount(t *testing.T) {
	db := setupMFADB(t)
	uid := uuid.New()

	codes, err := GenerateBackupCodes(db, uid, testBackupKey)
	require.NoError(t, err)

	var remaining int
	for i := range BackupCodeCount - LowBackupCodeThreshold {
		remaining, err = RedeemBackupCode(db, uid, codes[i], testBackupKey)
		require.NoError(t, err)
	}
	require.Equal(t, LowBackupCodeThreshold, remaining)
}
