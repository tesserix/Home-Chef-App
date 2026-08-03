package services

// #948: erasing a user does not erase their financial position. The guard has to
// see all three ways money can be left behind, and must not fire on an account
// that genuinely owes nothing.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func purgeGuardDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupLoyaltyDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE ledger_entries (
		id text PRIMARY KEY, transaction_id text, account_kind text, user_id text,
		direction text, amount_minor integer, currency text, created_at datetime)`).Error)
	return db
}

func addLedger(t *testing.T, db *gorm.DB, user uuid.UUID, direction string, minor int) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO ledger_entries (id, transaction_id, account_kind, user_id, direction, amount_minor, currency)
		 VALUES (?,?,?,?,?,?,'INR')`,
		uuid.NewString(), uuid.NewString(), "user_wallet_refund", user.String(), direction, minor).Error)
}

func TestAccountUnsettledMoney_Clean(t *testing.T) {
	db := purgeGuardDB(t)
	owed, err := AccountUnsettledMoney(db, uuid.New())
	require.NoError(t, err)
	require.False(t, owed.Any(), "an account with nothing outstanding is erasable: %s", owed)
}

func TestAccountUnsettledMoney_WalletBalance(t *testing.T) {
	db := purgeGuardDB(t)
	u := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO wallets (id, user_id, balance) VALUES (?,?,?)`,
		uuid.NewString(), u.String(), 255.30).Error)

	owed, err := AccountUnsettledMoney(db, u)
	require.NoError(t, err)
	require.True(t, owed.Any())
	require.InDelta(t, 255.30, owed.WalletBalance, 0.001)
}

// The reported DRIFT: a net ledger credit with NO wallet row to hold it.
func TestAccountUnsettledMoney_LedgerCreditWithoutWallet(t *testing.T) {
	db := purgeGuardDB(t)
	u := uuid.New()
	addLedger(t, db, u, "credit", 14587) // ₹145.87, the live figure

	owed, err := AccountUnsettledMoney(db, u)
	require.NoError(t, err)
	require.True(t, owed.Any(), "a ledger credit with no wallet is exactly the stranded case")
	require.InDelta(t, 145.87, owed.LedgerPositionRupe, 0.001)
	require.Zero(t, owed.WalletBalance)
}

func TestAccountUnsettledMoney_LedgerNetsToZero(t *testing.T) {
	db := purgeGuardDB(t)
	u := uuid.New()
	addLedger(t, db, u, "credit", 14587)
	addLedger(t, db, u, "debit", 14587)

	owed, err := AccountUnsettledMoney(db, u)
	require.NoError(t, err)
	require.False(t, owed.Any(), "a credit already drawn down is not outstanding: %s", owed)
}

// The #872 shape: captured, then cancelled, never refunded.
func TestAccountUnsettledMoney_CapturedNeverRefunded(t *testing.T) {
	db := purgeGuardDB(t)
	u := uuid.New()
	insert := func(status string, total, refund float64) {
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, customer_id, status, payment_status, subtotal, total, refund_amount)
			 VALUES (?,?,?,?,0,?,?)`,
			uuid.NewString(), u.String(), status, "completed", total, refund).Error)
	}
	insert("cancelled", 154.19, 0) // both live rows
	insert("cancelled", 143.39, 0)
	insert("cancelled", 297.65, 297.65) // properly refunded — not outstanding
	insert("delivered", 500.00, 0)      // still a live order, not a terminated one

	owed, err := AccountUnsettledMoney(db, u)
	require.NoError(t, err)
	require.True(t, owed.Any())
	require.InDelta(t, 297.58, owed.CapturedNotRefund, 0.001)
	require.Equal(t, 2, owed.OrderCount)
}

func TestAccountUnsettledMoney_NilInputs(t *testing.T) {
	db := purgeGuardDB(t)
	owed, err := AccountUnsettledMoney(db, uuid.Nil)
	require.NoError(t, err)
	require.False(t, owed.Any())

	owed, err = AccountUnsettledMoney(nil, uuid.New())
	require.NoError(t, err)
	require.False(t, owed.Any())
}
