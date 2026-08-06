package services

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/piicrypto"
)

// #925 — the MFA enrollment path wrote a normalized plaintext phone into
// phone_e164_bidx, a column that must hold a keyed hash. These cover the
// remediation of rows written before the fix.

func setupBidxDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE user_mfa_settings (
		user_id text PRIMARY KEY,
		phone_enrolled integer DEFAULT 0,
		phone_e164_enc text DEFAULT '',
		phone_e164_bidx text DEFAULT ''
	)`).Error)
	return db
}

func insertMFARow(t *testing.T, db *gorm.DB, bidx string) string {
	t.Helper()
	id := uuid.New().String()
	require.NoError(t, db.Exec(
		`INSERT INTO user_mfa_settings (user_id, phone_e164_bidx) VALUES (?, ?)`, id, bidx).Error)
	return id
}

func readBidx(t *testing.T, db *gorm.DB, userID string) string {
	t.Helper()
	var got string
	require.NoError(t, db.Raw(
		`SELECT phone_e164_bidx FROM user_mfa_settings WHERE user_id = ?`, userID).Scan(&got).Error)
	return got
}

func TestLooksLikeBlindIndex(t *testing.T) {
	// BlindIndex emits hex-encoded HMAC-SHA256: exactly 64 lowercase hex chars.
	require.True(t, LooksLikeBlindIndex("a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"))

	// The shapes that must NOT pass — anything here would be left in place.
	require.False(t, LooksLikeBlindIndex("+919876543210"), "a plaintext phone is not a blind index")
	require.False(t, LooksLikeBlindIndex(""), "empty is not a hash")
	require.False(t, LooksLikeBlindIndex("A1B2C3D4E5F60718293A4B5C6D7E8F90A1B2C3D4E5F60718293A4B5C6D7E8F90"),
		"uppercase hex is not what BlindIndex emits")
	require.False(t, LooksLikeBlindIndex("a1b2c3"), "too short")
	require.False(t, LooksLikeBlindIndex("a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f9zz"),
		"non-hex characters")
}

func TestRemediate_RewritesPlaintextAndLeavesHashesAlone(t *testing.T) {
	db := setupBidxDB(t)

	leaked := insertMFARow(t, db, "+919876543210")
	alreadyHashed := "a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"
	untouched := insertMFARow(t, db, alreadyHashed)

	fixed := RemediatePlaintextPhoneBidx(db)
	require.Equal(t, 1, fixed, "only the plaintext row needs rewriting")

	after := readBidx(t, db, leaked)
	require.NotEqual(t, "+919876543210", after, "the phone number must no longer be stored in the clear")
	require.False(t, containsDigitsOfPhone(after), "no recoverable fragment of the number may remain")

	require.Equal(t, alreadyHashed, readBidx(t, db, untouched), "an existing hash must not be re-hashed")
}

func TestRemediate_MatchesWhatTheFixedWriteSiteProduces(t *testing.T) {
	// A row remediated now must equal a row written later by the handler, or the
	// two populations would never match each other once #710 reads this column.
	db := setupBidxDB(t)
	id := insertMFARow(t, db, "+919876543210")
	RemediatePlaintextPhoneBidx(db)

	want := piicrypto.BlindIndex(NormalizeE164("+919876543210"))
	require.Equal(t, want, readBidx(t, db, id))
}

func TestRemediate_IsIdempotent(t *testing.T) {
	db := setupBidxDB(t)
	insertMFARow(t, db, "+919876543210")

	require.Equal(t, 1, RemediatePlaintextPhoneBidx(db))
	require.Zero(t, RemediatePlaintextPhoneBidx(db), "a second pass has nothing left to do")
	require.Zero(t, RemediatePlaintextPhoneBidx(db))
}

func TestRemediate_SkipsEmptyRowsAndIsSafeWithoutTheTable(t *testing.T) {
	db := setupBidxDB(t)
	empty := insertMFARow(t, db, "")
	require.Zero(t, RemediatePlaintextPhoneBidx(db), "a never-enrolled row is not a leak")
	require.Equal(t, "", readBidx(t, db, empty))

	// The MFA tables ship separately, so a deployment without them must not fail
	// the boot this runs in.
	bare, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.Zero(t, RemediatePlaintextPhoneBidx(bare))
	require.Zero(t, RemediatePlaintextPhoneBidx(nil))
}

// containsDigitsOfPhone reports whether the national portion of the test number
// survives anywhere in the remediated value.
func containsDigitsOfPhone(v string) bool {
	for _, frag := range []string{"9876543210", "919876543210"} {
		if strings.Contains(v, frag) {
			return true
		}
	}
	return false
}
