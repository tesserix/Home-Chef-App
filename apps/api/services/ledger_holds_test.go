package services

// ledger_holds_test.go — Phase 3 (docs/wallet-ledger-plan.md). A generic wallet hold reserves
// available balance, then captures (spends) or releases (restores) it. Invariants: a hold moves
// value available → held without changing the total; capture spends it; release restores it to
// the exact buckets it drained; available (never total) is what a new hold is checked against;
// a hold is captured XOR released, once.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedBucket posts an opening credit straight into one user bucket, so hold tests can set up a
// known available balance without going through the legacy wallet.
func seedBucket(t *testing.T, db *gorm.DB, userID uuid.UUID, kind models.LedgerAccountKind, rupees float64) {
	t.Helper()
	uid := userID
	amt := models.RupeesToMoney(rupees)
	err := db.Transaction(func(tx *gorm.DB) error {
		_, e := PostLedgerTransaction(tx, "seed:"+uid.String()+":"+string(kind), "seed", "seed", uid.String(),
			[]LedgerLeg{
				{AccountKind: kind, UserID: &uid, Direction: models.LedgerCredit, Amount: amt},
				{AccountKind: models.LedgerAcctSystemOpening, Direction: models.LedgerDebit, Amount: amt},
			})
		return e
	})
	require.NoError(t, err)
}

// Placing a hold moves value from the spendable buckets (in priority) into held, leaving the
// total unchanged and the double-entry invariant intact.
func TestWalletHold_PlaceReservesAvailable(t *testing.T) {
	db := setupShadowDB(t)
	u := uuid.New()
	seedBucket(t, db, u, models.LedgerAcctUserPromo, 100)
	seedBucket(t, db, u, models.LedgerAcctUserRefund, 200)

	_, err := PlaceWalletHold(db, u, models.RupeesToMoney(150), "order", "o1")
	require.NoError(t, err)

	avail, err := LedgerUserAvailableBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(15000), avail, "300 − 150 reserved")

	held, err := LedgerUserHeldBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(15000), held)

	total, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(30000), total, "total unchanged: available + held")

	buckets, err := LedgerUserBucketBalances(db, u)
	require.NoError(t, err)
	require.Zero(t, buckets[models.LedgerAcctUserPromo], "promo drained first")
	require.Equal(t, models.Money(15000), buckets[models.LedgerAcctUserRefund], "refund keeps 200−50")

	d, c, err := LedgerTotals(db)
	require.NoError(t, err)
	require.Equal(t, d, c)
}

// A hold is checked against available, not total: once money is held, it cannot be reserved
// again, and an over-reserve is rejected.
func TestWalletHold_InsufficientAvailableRejected(t *testing.T) {
	db := setupShadowDB(t)
	u := uuid.New()
	seedBucket(t, db, u, models.LedgerAcctUserRefund, 100)

	_, err := PlaceWalletHold(db, u, models.RupeesToMoney(150), "order", "big")
	require.ErrorIs(t, err, ErrInsufficientWalletBalance)

	// Reserve all of it, then a second hold has nothing left (held is excluded from available).
	_, err = PlaceWalletHold(db, u, models.RupeesToMoney(100), "order", "a")
	require.NoError(t, err)
	_, err = PlaceWalletHold(db, u, models.RupeesToMoney(1), "order", "b")
	require.ErrorIs(t, err, ErrInsufficientWalletBalance)
}

// Capturing spends the reserved value: held → system_spend, total drops, available unchanged.
func TestWalletHold_CaptureSpends(t *testing.T) {
	db := setupShadowDB(t)
	u := uuid.New()
	seedBucket(t, db, u, models.LedgerAcctUserPromo, 100)
	seedBucket(t, db, u, models.LedgerAcctUserRefund, 200)
	_, err := PlaceWalletHold(db, u, models.RupeesToMoney(150), "order", "o1")
	require.NoError(t, err)

	_, err = CaptureWalletHold(db, "order", "o1")
	require.NoError(t, err)

	held, err := LedgerUserHeldBalance(db, u)
	require.NoError(t, err)
	require.Zero(t, held, "hold consumed")
	total, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(15000), total, "300 − 150 spent")
	avail, err := LedgerUserAvailableBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(15000), avail)

	d, c, err := LedgerTotals(db)
	require.NoError(t, err)
	require.Equal(t, d, c)

	// Idempotent: capturing again is a no-op.
	_, err = CaptureWalletHold(db, "order", "o1")
	require.NoError(t, err)
	total2, _ := LedgerUserBalance(db, u)
	require.Equal(t, total, total2)
}

// Releasing restores the reserved value to the exact buckets it was drawn from (provenance
// preserved), and the total returns to what it was.
func TestWalletHold_ReleaseRestoresSourceBuckets(t *testing.T) {
	db := setupShadowDB(t)
	u := uuid.New()
	seedBucket(t, db, u, models.LedgerAcctUserPromo, 100)
	seedBucket(t, db, u, models.LedgerAcctUserRefund, 200)
	_, err := PlaceWalletHold(db, u, models.RupeesToMoney(150), "order", "o1")
	require.NoError(t, err)

	_, err = ReleaseWalletHold(db, "order", "o1")
	require.NoError(t, err)

	buckets, err := LedgerUserBucketBalances(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(10000), buckets[models.LedgerAcctUserPromo], "promo restored")
	require.Equal(t, models.Money(20000), buckets[models.LedgerAcctUserRefund], "refund restored")
	require.Zero(t, buckets[models.LedgerAcctUserHeld], "nothing held")

	avail, err := LedgerUserAvailableBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(30000), avail, "fully available again")

	// Idempotent: releasing again is a no-op.
	_, err = ReleaseWalletHold(db, "order", "o1")
	require.NoError(t, err)
	avail2, _ := LedgerUserAvailableBalance(db, u)
	require.Equal(t, avail, avail2)
}

// A hold is captured XOR released — never both.
func TestWalletHold_CaptureReleaseAreMutuallyExclusive(t *testing.T) {
	db := setupShadowDB(t)
	u := uuid.New()
	seedBucket(t, db, u, models.LedgerAcctUserRefund, 200)

	// Released then capture → rejected.
	_, err := PlaceWalletHold(db, u, models.RupeesToMoney(50), "order", "rel")
	require.NoError(t, err)
	_, err = ReleaseWalletHold(db, "order", "rel")
	require.NoError(t, err)
	_, err = CaptureWalletHold(db, "order", "rel")
	require.ErrorIs(t, err, ErrHoldAlreadyReleased)

	// Captured then release → rejected.
	_, err = PlaceWalletHold(db, u, models.RupeesToMoney(50), "order", "cap")
	require.NoError(t, err)
	_, err = CaptureWalletHold(db, "order", "cap")
	require.NoError(t, err)
	_, err = ReleaseWalletHold(db, "order", "cap")
	require.ErrorIs(t, err, ErrHoldAlreadyCaptured)
}

// Placing the same (refType, refID) twice reserves once.
func TestWalletHold_PlaceIdempotent(t *testing.T) {
	db := setupShadowDB(t)
	u := uuid.New()
	seedBucket(t, db, u, models.LedgerAcctUserRefund, 200)

	t1, err := PlaceWalletHold(db, u, models.RupeesToMoney(150), "order", "o1")
	require.NoError(t, err)
	t2, err := PlaceWalletHold(db, u, models.RupeesToMoney(150), "order", "o1")
	require.NoError(t, err)
	require.Equal(t, t1.ID, t2.ID, "same placement returned")

	held, err := LedgerUserHeldBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(15000), held, "reserved once, not twice")
}

// Capturing/releasing a hold that was never placed errors clearly.
func TestWalletHold_NotFound(t *testing.T) {
	db := setupShadowDB(t)
	_, err := CaptureWalletHold(db, "order", "ghost")
	require.ErrorIs(t, err, ErrHoldNotFound)
	_, err = ReleaseWalletHold(db, "order", "ghost")
	require.ErrorIs(t, err, ErrHoldNotFound)
}
