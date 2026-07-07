package services

// refund_request_dedup_test.go — #600. ClaimRefundRequest dedups a client-visible refund
// submission so a retry AFTER a successful round-trip doesn't issue a second real refund. A
// client Idempotency-Key dedups exactly + permanently; the (order,amount,reason) fallback dedups
// only within refundDedupWindow so a genuine identical repeat later is still allowed.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupRefundDedupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE processed_events (
		consumer TEXT NOT NULL, msg_id TEXT NOT NULL, subject TEXT DEFAULT '',
		processed_at DATETIME, PRIMARY KEY (consumer, msg_id)
	)`).Error)
	return db
}

// backdateRefundClaim ages a claim past the window so a fallback repeat is treated as a genuine
// new request (there's no time travel in sqlite; back-dating processed_at simulates it).
func backdateRefundClaim(t *testing.T, db *gorm.DB, key string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`UPDATE processed_events SET processed_at = ? WHERE consumer = 'refund-request' AND msg_id = ?`,
		time.Now().Add(-2*refundDedupWindow), key).Error)
}

// ── client Idempotency-Key: exact + permanent ──

func TestClaimRefundRequest_ClientKey_DedupsRetry(t *testing.T) {
	db := setupRefundDedupDB(t)
	order := uuid.New()

	ok1, k1, err := ClaimRefundRequest(db, order, "cli-key-1", 10000, "goodwill")
	require.NoError(t, err)
	require.True(t, ok1, "first submission proceeds")

	ok2, k2, err := ClaimRefundRequest(db, order, "cli-key-1", 10000, "goodwill")
	require.NoError(t, err)
	require.False(t, ok2, "a retry with the SAME client key is deduped — no second refund")
	require.Equal(t, k1, k2)
}

func TestClaimRefundRequest_ClientKey_DistinctKeysBothProceed(t *testing.T) {
	db := setupRefundDedupDB(t)
	order := uuid.New()
	ok1, _, _ := ClaimRefundRequest(db, order, "cli-key-1", 10000, "goodwill")
	ok2, _, _ := ClaimRefundRequest(db, order, "cli-key-2", 10000, "goodwill")
	require.True(t, ok1)
	require.True(t, ok2, "two DISTINCT client keys are two distinct intentional requests → both refund")
}

func TestClaimRefundRequest_ClientKey_ReleaseAllowsRetry(t *testing.T) {
	db := setupRefundDedupDB(t)
	order := uuid.New()
	ok1, key, _ := ClaimRefundRequest(db, order, "cli-key-1", 10000, "goodwill")
	require.True(t, ok1)

	ReleaseRefundRequestClaim(db, key) // the refund failed pre-commit

	ok2, _, err := ClaimRefundRequest(db, order, "cli-key-1", 10000, "goodwill")
	require.NoError(t, err)
	require.True(t, ok2, "after a release (pre-commit failure) the same key re-attempts")
}

// ── fallback (order+amount+reason): windowed ──

func TestClaimRefundRequest_Fallback_DedupsWithinWindow(t *testing.T) {
	db := setupRefundDedupDB(t)
	order := uuid.New()

	ok1, _, err := ClaimRefundRequest(db, order, "", 10000, "goodwill")
	require.NoError(t, err)
	require.True(t, ok1)

	ok2, _, err := ClaimRefundRequest(db, order, "", 10000, "goodwill")
	require.NoError(t, err)
	require.False(t, ok2, "an identical resubmit WITHIN the window is a client retry → deduped")
}

func TestClaimRefundRequest_Fallback_AllowsRepeatAfterWindow(t *testing.T) {
	db := setupRefundDedupDB(t)
	order := uuid.New()

	ok1, key, _ := ClaimRefundRequest(db, order, "", 10000, "goodwill")
	require.True(t, ok1)
	backdateRefundClaim(t, db, key)

	ok2, _, err := ClaimRefundRequest(db, order, "", 10000, "goodwill")
	require.NoError(t, err)
	require.True(t, ok2, "a genuinely-intended identical repeat AFTER the window is allowed")
}

func TestClaimRefundRequest_Fallback_DistinctAmountOrReasonProceed(t *testing.T) {
	db := setupRefundDedupDB(t)
	order := uuid.New()

	ok1, _, _ := ClaimRefundRequest(db, order, "", 10000, "goodwill")
	okAmt, _, _ := ClaimRefundRequest(db, order, "", 20000, "goodwill")   // different amount
	okReason, _, _ := ClaimRefundRequest(db, order, "", 10000, "damaged") // different reason
	require.True(t, ok1)
	require.True(t, okAmt, "a different amount is a different refund")
	require.True(t, okReason, "a different reason is a different refund")
}

func TestClaimRefundRequest_Fallback_ReasonWhitespaceNormalized(t *testing.T) {
	db := setupRefundDedupDB(t)
	order := uuid.New()
	ok1, _, _ := ClaimRefundRequest(db, order, "", 10000, "goodwill")
	ok2, _, _ := ClaimRefundRequest(db, order, "", 10000, "  goodwill  ") // padded retry
	require.True(t, ok1)
	require.False(t, ok2, "surrounding whitespace is normalized so a padded retry still dedups")
}

// Two back-to-back identical submissions (the double-submit the bug is about): exactly one wins.
func TestClaimRefundRequest_ConcurrentDoubleSubmit_OneWins(t *testing.T) {
	db := setupRefundDedupDB(t)
	order := uuid.New()
	a, _, _ := ClaimRefundRequest(db, order, "", 10000, "goodwill")
	b, _, _ := ClaimRefundRequest(db, order, "", 10000, "goodwill")
	require.True(t, a != b, "exactly one of two identical submissions proceeds")
	require.True(t, a, "the first proceeds, the second dedups")
}

func TestNormalizeRefundClientKey(t *testing.T) {
	require.Equal(t, "", NormalizeRefundClientKey("   "))
	require.Equal(t, "abc", NormalizeRefundClientKey("  abc  "))
	require.Len(t, NormalizeRefundClientKey(string(make([]byte, 500))), maxRefundClientKey)
}
