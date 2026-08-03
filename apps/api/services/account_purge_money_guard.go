package services

// account_purge_money_guard.go — refuse to erase an account that still holds
// money (#948).
//
// Deleting the user row does not delete their financial position. Observed in
// production: user 91768f92 has no `users` row, no `wallets` row and no
// `wallet_txns`, but still owns three orders, a ₹145.87 `user_wallet_refund`
// credit in the double-entry ledger, and two cancelled orders holding a
// COMPLETED payment with `refund_amount = 0` — ₹297.58 taken and never returned.
//
// The ledger-reconcile cron reports that position as DRIFT on every run and
// explicitly refuses to auto-correct, so it is permanent noise that will mask
// the next real drift.
//
// Erasure is a legal commitment, so this does not block forever silently: it
// skips the purge and logs a distinct, greppable line naming the amounts, so the
// account surfaces for a deliberate settle-or-write-off rather than being erased
// on top of money that belongs to someone.

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// UnsettledMoney describes why an account cannot be erased yet.
type UnsettledMoney struct {
	WalletBalance      float64
	LedgerPositionRupe float64
	CapturedNotRefund  float64
	OrderCount         int
}

func (u UnsettledMoney) String() string {
	return fmt.Sprintf("wallet=%.2f ledger=%.2f captured_not_refunded=%.2f orders=%d",
		u.WalletBalance, u.LedgerPositionRupe, u.CapturedNotRefund, u.OrderCount)
}

// Any reports whether anything is outstanding.
func (u UnsettledMoney) Any() bool {
	return u.WalletBalance > 0.005 ||
		u.LedgerPositionRupe > 0.005 ||
		u.CapturedNotRefund > 0.005
}

// AccountUnsettledMoney totals what an account still holds, so the purge can
// refuse rather than erase on top of it.
//
// Three independent positions, because they fail independently:
//
//   - a wallet balance the customer can still spend;
//   - a net-credit position in the double-entry ledger, which can exist with no
//     wallet row at all (that is exactly the reported DRIFT);
//   - money captured on an order that was then cancelled or refunded in status
//     but never actually refunded — the #872 shape, which no forward-looking
//     cron heals because the row has already left the pending window.
func AccountUnsettledMoney(db *gorm.DB, userID uuid.UUID) (UnsettledMoney, error) {
	var out UnsettledMoney
	if db == nil || userID == uuid.Nil {
		return out, nil
	}

	var wallet models.Wallet
	if err := db.Unscoped().Where("user_id = ?", userID).First(&wallet).Error; err == nil {
		out.WalletBalance = wallet.Balance
	} else if err != gorm.ErrRecordNotFound {
		return out, fmt.Errorf("purge-guard: wallet for %s: %w", userID, err)
	}

	// Net ledger position: credits owed to the user, less debits already taken.
	var creditMinor, debitMinor int64
	if err := db.Model(&models.LedgerEntry{}).
		Where("user_id = ? AND direction = ?", userID, "credit").
		Select("COALESCE(SUM(amount_minor), 0)").Scan(&creditMinor).Error; err != nil {
		return out, fmt.Errorf("purge-guard: ledger credits for %s: %w", userID, err)
	}
	if err := db.Model(&models.LedgerEntry{}).
		Where("user_id = ? AND direction = ?", userID, "debit").
		Select("COALESCE(SUM(amount_minor), 0)").Scan(&debitMinor).Error; err != nil {
		return out, fmt.Errorf("purge-guard: ledger debits for %s: %w", userID, err)
	}
	if net := creditMinor - debitMinor; net > 0 {
		out.LedgerPositionRupe = float64(net) / 100.0
	}

	// Captured but never refunded on a terminated order.
	type row struct {
		Total  float64
		Refund float64
	}
	var rows []row
	if err := db.Model(&models.Order{}).Unscoped().
		Where("customer_id = ?", userID).
		Where("payment_status = ?", models.PaymentCompleted).
		Where("status IN ?", []models.OrderStatus{
			models.OrderStatusCancelled,
			models.OrderStatusRejected,
			models.OrderStatusRefunded,
		}).
		Select("total, COALESCE(refund_amount, 0) as refund").Scan(&rows).Error; err != nil {
		return out, fmt.Errorf("purge-guard: terminated orders for %s: %w", userID, err)
	}
	for _, r := range rows {
		if owed := r.Total - r.Refund; owed > 0.005 {
			out.CapturedNotRefund += owed
			out.OrderCount++
		}
	}

	return out, nil
}
