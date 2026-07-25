package services

// loyalty_expiry_test.go — pins the daily expiry sweep's money-safety
// invariant: after a sweep, the account's cached balance must always equal
// Σ(points_remaining) across a user's lots. TestExpireLoyaltyBatches_
// MultipleLotsBalanceConsistent is the regression for a real bug found in
// the design brief, where force-zeroing an expired lot AFTER its debit
// (which already FIFO-drains the soonest-expiring lot internally) double-
// removes points whenever ≥2 lots are due in the same sweep.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestExpireLoyaltyBatches_ZeroesPastDueAndDebits(t *testing.T) {
	db := setupLoyaltyDB(t)
	u := uuid.New()
	cfg := GetLoyaltyConfig(db)
	// Credit 100 pts via a lot that already expired.
	_, _, err := applyLoyaltyTxnInTx(db, u, 100, models.LoyaltyCredit, models.LoyaltySourceOrder, nil, "e", "c1", nil, cfg)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`UPDATE loyalty_earn_batches SET expires_at = ? WHERE user_id = ?`,
		time.Now().Add(-time.Hour), u.String()).Error)

	n, err := ExpireLoyaltyBatches(db, time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, 0.0, batchRemaining(t, db, u)) // lot drained

	acct, err := LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 0.0, acct.Balance) // account balance decremented

	// Idempotent: a second sweep expires nothing.
	n2, err := ExpireLoyaltyBatches(db, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, n2)
}

// TestExpireLoyaltyBatches_MultipleLotsBalanceConsistent pins the
// soonest-expiry-first, no-force-zero fix (Correction A). One user earns
// three lots; two are past due. The buggy "force zero after debit" design
// would leave the account balance at 130 (only the FIFO-drained lot's points
// subtracted correctly, the other force-zeroed lot's points never actually
// debited from the balance) while the lots show 0 remaining — balance and
// lots disagree. The correct sweep keeps them in lockstep: balance drops by
// exactly the sum of the two due lots (80), landing at 100.
func TestExpireLoyaltyBatches_MultipleLotsBalanceConsistent(t *testing.T) {
	db := setupLoyaltyDB(t)
	u := uuid.New()
	cfg := GetLoyaltyConfig(db)

	_, _, err := applyLoyaltyTxnInTx(db, u, 30, models.LoyaltyCredit, models.LoyaltySourceOrder, nil, "e1", "c1", nil, cfg)
	require.NoError(t, err)
	_, _, err = applyLoyaltyTxnInTx(db, u, 50, models.LoyaltyCredit, models.LoyaltySourceOrder, nil, "e2", "c2", nil, cfg)
	require.NoError(t, err)
	_, _, err = applyLoyaltyTxnInTx(db, u, 100, models.LoyaltyCredit, models.LoyaltySourceOrder, nil, "e3", "c3", nil, cfg)
	require.NoError(t, err)

	// Backdate the two lots we want due, leaving the third (c3, 100 pts) in
	// the future. createEarnBatch keys batches as "batch:<idempotencyKey>".
	require.NoError(t, db.Exec(`UPDATE loyalty_earn_batches SET expires_at = ? WHERE idempotency_key = ?`,
		time.Now().Add(-1*time.Hour), "batch:c1").Error)
	require.NoError(t, db.Exec(`UPDATE loyalty_earn_batches SET expires_at = ? WHERE idempotency_key = ?`,
		time.Now().Add(-2*time.Hour), "batch:c2").Error)

	require.Equal(t, 180.0, batchRemaining(t, db, u))

	n, err := ExpireLoyaltyBatches(db, time.Now())
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, 100.0, batchRemaining(t, db, u)) // only the non-due lot (c3) survives

	acct, err := LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 100.0, acct.Balance) // balance stayed consistent with lots
}
