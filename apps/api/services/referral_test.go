package services

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// referral_test.go — #38. The referral core: code minting (format + idempotent
// per user), and the accept-eligibility guards (valid code, no self-referral,
// one redemption per referee, new-customers-only). These guards are the first
// line of fraud defence, so they're pinned with tests.

func setupReferralDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE referral_codes (
			id TEXT PRIMARY KEY, user_id TEXT NOT NULL UNIQUE, code TEXT NOT NULL UNIQUE, created_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE referrals (
			id TEXT PRIMARY KEY,
			referrer_user_id TEXT NOT NULL,
			referee_user_id TEXT NOT NULL UNIQUE,
			code TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			order_id TEXT,
			referrer_reward REAL DEFAULT 0,
			referee_reward REAL DEFAULT 0,
			rewarded_at DATETIME,
			referee_device TEXT,
			referee_ip TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, customer_id TEXT, payment_status TEXT, deleted_at DATETIME)`).Error)
	return db
}

func TestNormalizeReferralCode(t *testing.T) {
	assert.Equal(t, "ABC123", NormalizeReferralCode("  abc123 "))
	assert.Equal(t, "FE3DR", NormalizeReferralCode("fe3dr"))
	assert.Equal(t, "", NormalizeReferralCode("   "))
}

func TestRandomReferralCodeFormat(t *testing.T) {
	for i := 0; i < 200; i++ {
		code := randomReferralCode()
		require.Len(t, code, referralCodeLen)
		for _, ch := range code {
			require.True(t, strings.ContainsRune(referralAlphabet, ch), "code %q has out-of-alphabet char %q", code, ch)
		}
		// No ambiguous characters.
		for _, bad := range []string{"0", "O", "1", "I", "L"} {
			require.False(t, strings.Contains(code, bad), "code %q contains ambiguous %q", code, bad)
		}
	}
}

func TestGetOrCreateReferralCode(t *testing.T) {
	db := setupReferralDB(t)
	uid := uuid.New()

	code, err := GetOrCreateReferralCode(db, uid)
	require.NoError(t, err)
	require.Len(t, code, referralCodeLen)

	// Second call returns the SAME code (idempotent per user).
	again, err := GetOrCreateReferralCode(db, uid)
	require.NoError(t, err)
	assert.Equal(t, code, again)

	var n int64
	db.Model(&models.ReferralCode{}).Where("user_id = ?", uid).Count(&n)
	assert.Equal(t, int64(1), n)
}

func seedCode(t *testing.T, db *gorm.DB, userID uuid.UUID) string {
	t.Helper()
	code, err := GetOrCreateReferralCode(db, userID)
	require.NoError(t, err)
	return code
}

func TestAcceptReferral(t *testing.T) {
	referrer := uuid.New()

	t.Run("happy path creates a pending referral", func(t *testing.T) {
		db := setupReferralDB(t)
		code := seedCode(t, db, referrer)
		referee := uuid.New()
		ref, err := AcceptReferral(db, AcceptReferralInput{RefereeUserID: referee, Code: code, Device: "dev-1", IP: "1.2.3.4"})
		require.NoError(t, err)
		assert.Equal(t, models.ReferralStatePending, ref.Status)
		assert.Equal(t, referrer, ref.ReferrerUserID)
		assert.Equal(t, "dev-1", ref.RefereeDevice)
	})

	t.Run("normalises a lowercase code", func(t *testing.T) {
		db := setupReferralDB(t)
		code := seedCode(t, db, referrer)
		ref, err := AcceptReferral(db, AcceptReferralInput{RefereeUserID: uuid.New(), Code: strings.ToLower(code)})
		require.NoError(t, err)
		assert.Equal(t, code, ref.Code)
	})

	t.Run("rejects an unknown code", func(t *testing.T) {
		db := setupReferralDB(t)
		_, err := AcceptReferral(db, AcceptReferralInput{RefereeUserID: uuid.New(), Code: "NOPE2345"})
		assert.ErrorIs(t, err, ErrReferralCodeInvalid)
	})

	t.Run("rejects self-referral", func(t *testing.T) {
		db := setupReferralDB(t)
		code := seedCode(t, db, referrer)
		_, err := AcceptReferral(db, AcceptReferralInput{RefereeUserID: referrer, Code: code})
		assert.ErrorIs(t, err, ErrReferralSelf)
	})

	t.Run("same code re-accept is idempotent; a different code is rejected", func(t *testing.T) {
		db := setupReferralDB(t)
		code := seedCode(t, db, referrer)
		other := uuid.New()
		otherCode := seedCode(t, db, other)
		referee := uuid.New()

		first, err := AcceptReferral(db, AcceptReferralInput{RefereeUserID: referee, Code: code})
		require.NoError(t, err)
		again, err := AcceptReferral(db, AcceptReferralInput{RefereeUserID: referee, Code: code})
		require.NoError(t, err)
		assert.Equal(t, first.ID, again.ID)

		_, err = AcceptReferral(db, AcceptReferralInput{RefereeUserID: referee, Code: otherCode})
		assert.ErrorIs(t, err, ErrReferralAlreadyUsed)
	})

	t.Run("rejects a referee who already has a paid order", func(t *testing.T) {
		db := setupReferralDB(t)
		code := seedCode(t, db, referrer)
		referee := uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, payment_status) VALUES (?, ?, 'completed')`,
			uuid.New().String(), referee.String()).Error)
		_, err := AcceptReferral(db, AcceptReferralInput{RefereeUserID: referee, Code: code})
		assert.ErrorIs(t, err, ErrReferralNotNewUser)
	})
}
