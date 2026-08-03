package services

// #936: wallet credit is stamped on an order at checkout but only debited at
// settlement, so between the two the balance still reads full and the same
// rupees are offered to a second order.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWalletClaimedByUnpaidOrders(t *testing.T) {
	db := setupLoyaltyDB(t)
	cust := uuid.New()

	insert := func(status, payStatus string, walletApplied float64) uuid.UUID {
		id := uuid.New()
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, customer_id, status, payment_status, subtotal, total, wallet_applied)
			 VALUES (?,?,?,?,500,630,?)`,
			id.String(), cust.String(), status, payStatus, walletApplied).Error)
		return id
	}

	// The live shape: two abandoned checkouts each holding ₹254.55 of a ₹255.30
	// wallet, while a third checkout was still offered the full balance.
	insert("pending", "pending", 254.55)
	insert("pending", "pending", 254.55)

	claimed, err := WalletClaimedByUnpaidOrdersPaise(db, cust, uuid.Nil)
	require.NoError(t, err)
	require.Equal(t, 50910, claimed, "both unpaid orders hold their credit")
}

func TestWalletClaimedByUnpaidOrders_IgnoresSettledAndTerminal(t *testing.T) {
	db := setupLoyaltyDB(t)
	cust := uuid.New()

	insert := func(status, payStatus string, walletApplied float64) {
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, customer_id, status, payment_status, subtotal, total, wallet_applied)
			 VALUES (?,?,?,?,500,630,?)`,
			uuid.NewString(), cust.String(), status, payStatus, walletApplied).Error)
	}

	// Paid: the credit was actually debited, so it is not a claim.
	insert("preparing", "completed", 100)
	// Terminal: the claim is released. The stale-order cron produces exactly the
	// first of these ~30 minutes after an abandoned checkout.
	insert("cancelled", "pending", 100)
	insert("rejected", "pending", 100)
	insert("refunded", "pending", 100)
	// No credit applied at all.
	insert("pending", "pending", 0)

	claimed, err := WalletClaimedByUnpaidOrdersPaise(db, cust, uuid.Nil)
	require.NoError(t, err)
	require.Equal(t, 0, claimed)
}

func TestWalletClaimedByUnpaidOrders_ExcludesTheOrderBeingQuoted(t *testing.T) {
	db := setupLoyaltyDB(t)
	cust := uuid.New()

	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, status, payment_status, subtotal, total, wallet_applied)
		 VALUES (?,?,?,?,500,630,?)`,
		id.String(), cust.String(), "pending", "pending", 200.0).Error)

	// Re-quoting that same order must not count its own claim against itself,
	// or the credit it already holds would look like someone else's.
	claimed, err := WalletClaimedByUnpaidOrdersPaise(db, cust, id)
	require.NoError(t, err)
	require.Equal(t, 0, claimed)

	// A checkout preview has no order id yet and sees the claim.
	claimed, err = WalletClaimedByUnpaidOrdersPaise(db, cust, uuid.Nil)
	require.NoError(t, err)
	require.Equal(t, 20000, claimed)
}

func TestWalletClaimedByUnpaidOrders_OtherCustomersDoNotCount(t *testing.T) {
	db := setupLoyaltyDB(t)
	cust, other := uuid.New(), uuid.New()

	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, status, payment_status, subtotal, total, wallet_applied)
		 VALUES (?,?,?,?,500,630,?)`,
		uuid.NewString(), other.String(), "pending", "pending", 300.0).Error)

	claimed, err := WalletClaimedByUnpaidOrdersPaise(db, cust, uuid.Nil)
	require.NoError(t, err)
	require.Equal(t, 0, claimed)
}

func TestSpendableWalletPaise(t *testing.T) {
	cases := []struct{ balance, claimed, want int }{
		{25530, 0, 25530},  // nothing held back
		{25530, 25455, 75}, // the live case: ₹255.30 balance, ₹254.55 claimed
		{25530, 50910, 0},  // two claims exceed the balance — offer nothing, never negative
		{25530, 25530, 0},  // exactly consumed
		{0, 5000, 0},       // empty wallet
	}
	for _, tc := range cases {
		if got := spendableWalletPaise(tc.balance, tc.claimed); got != tc.want {
			t.Errorf("spendableWalletPaise(%d, %d) = %d, want %d", tc.balance, tc.claimed, got, tc.want)
		}
	}
}
