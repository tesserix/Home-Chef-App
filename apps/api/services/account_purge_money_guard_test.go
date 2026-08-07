package services

// #948: erasing a user does not erase their financial position. The guard has to
// see all three ways money can be left behind, and must not fire on an account
// that genuinely owes nothing.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func purgeGuardDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupLoyaltyDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE ledger_entries (
		id text PRIMARY KEY, transaction_id text, account_kind text, user_id text,
		direction text, amount_minor integer, currency text, created_at datetime)`).Error)
	// #948: the guard reads gateway_payment_id / wallet_applied / loyalty_applied to
	// tell a real capture from a per-day order that only INHERITED payment_status.
	// Rebuild orders from the model so this fixture cannot drift from what it reads.
	require.NoError(t, db.Exec(`DROP TABLE IF EXISTS orders`).Error)
	createTableFor(t, db, &models.Order{})
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
// insertGuardOrder writes a terminated order with explicit capture evidence.
func insertGuardOrder(t *testing.T, db *gorm.DB, u uuid.UUID,
	status string, total, refund float64, payID string, walletApplied, loyaltyApplied float64) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, status, payment_status, subtotal, total, refund_amount,
		   gateway_payment_id, wallet_applied, loyalty_applied)
		 VALUES (?,?,?,?,0,?,?,?,?,?)`,
		uuid.NewString(), u.String(), status, "completed", total, refund,
		payID, walletApplied, loyaltyApplied).Error)
}

func TestAccountUnsettledMoney_CapturedNeverRefunded(t *testing.T) {
	db := purgeGuardDB(t)
	u := uuid.New()

	insertGuardOrder(t, db, u, "cancelled", 154.19, 0, "pay_A", 0, 0)      // gateway capture, unrefunded
	insertGuardOrder(t, db, u, "cancelled", 143.39, 0, "", 143.39, 0)      // wallet-funded, unrefunded
	insertGuardOrder(t, db, u, "cancelled", 297.65, 297.65, "pay_B", 0, 0) // properly refunded
	insertGuardOrder(t, db, u, "delivered", 500.00, 0, "pay_C", 0, 0)      // live order, not terminated

	owed, err := AccountUnsettledMoney(db, u)
	require.NoError(t, err)
	require.True(t, owed.Any())
	require.InDelta(t, 297.58, owed.CapturedNotRefund, 0.001)
	require.Equal(t, 2, owed.OrderCount)
}

// #948: `payment_status = completed` does NOT mean money was taken. A meal-plan
// per-day fulfillment order inherits that status from its parent plan while
// carrying no payment of its own — the plan captures once on its escrow payment,
// never per day.
//
// Counting those invented ₹297.58 of "stranded customer money" on the very
// account this guard was written for, and would have blocked that erasure
// forever on a balance nobody was ever charged. Erasure is a legal commitment,
// so a phantom blocker is not a harmless false positive.
func TestAccountUnsettledMoney_IgnoresOrdersThatNeverCaptured(t *testing.T) {
	db := purgeGuardDB(t)
	u := uuid.New()

	// The two real production rows: cancelled, payment_status completed, no payment
	// id, no credit applied — nothing was ever charged for them.
	insertGuardOrder(t, db, u, "cancelled", 154.19, 0, "", 0, 0)
	insertGuardOrder(t, db, u, "cancelled", 143.39, 0, "", 0, 0)

	owed, err := AccountUnsettledMoney(db, u)
	require.NoError(t, err)
	require.Zero(t, owed.CapturedNotRefund, "an order that never captured owes nothing")
	require.Zero(t, owed.OrderCount)
	require.False(t, owed.Any(), "the account is erasable: %s", owed)

	// A loyalty-funded order DID take the customer's money and must still count,
	// even though it has no gateway payment id.
	insertGuardOrder(t, db, u, "cancelled", 50.00, 0, "", 0, 2.50)
	owed, err = AccountUnsettledMoney(db, u)
	require.NoError(t, err)
	require.InDelta(t, 50.00, owed.CapturedNotRefund, 0.001)
	require.True(t, owed.Any())
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
