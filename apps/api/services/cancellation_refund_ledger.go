package services

// cancellation_refund_ledger.go — put cancellation refunds on the refund ledger
// (#940).
//
// ExecuteCancellationRefund moves money itself rather than going through
// OrderRefundCoordinator, so until now it wrote no `refund_transactions` row.
// The consequences were all invisible until someone went looking:
//
//   - `gateway_refund_reconcile` reads that table, so it could not see — and
//     therefore could not reconcile — a single cancellation refund. Roughly
//     ₹3,190 across three days had no provider refund id recorded anywhere.
//   - Support and finance had no way to tie a customer's refund to a gateway
//     transaction.
//   - `refund_transactions.idempotency_key` is UNIQUE and is the ledger's half of
//     double-refund protection. This path relied solely on the order-row claim
//     (`payment_status` + `refunded_at`), so the two guards protected disjoint
//     sets of refunds.
//
// Routing this path fully through the Coordinator is the #687/#690 refactor and
// is deliberately NOT attempted here — it would rewrite the most sensitive money
// path in the system for a reporting defect. Recording the row is what #940 asks
// for as the minimum, and it closes the reconciliation and idempotency gaps
// without touching how the money moves.
//
// KEY CHOICE: the row is keyed `refund:<orderID>:full` via ScopeFull — byte-identical
// to the RefundFullIdempotencyKey this path already sends the gateway, and to the key
// the Coordinator would use for the same logical refund. So the unique index now
// mutually excludes the two paths for one order instead of merely describing one of
// them. That is the part that turns this from bookkeeping into a real guard.

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services/orderrefund"
)

// cancellationRefundLedgerInput is what the ledger row records about a
// cancellation refund that has ALREADY moved.
type cancellationRefundLedgerInput struct {
	Order *models.Order
	// Amount is the total returned to the customer, in rupees.
	Amount float64
	// GatewayRefundID is the provider's refund id for the card slice. Empty when
	// the refund was funded entirely from wallet/loyalty credit — a legitimate
	// outcome, not a failure (see models.RefundTransaction.Provider).
	GatewayRefundID string
	// WalletRef is the wallet reference used when there is no gateway leg.
	WalletRef string
	Reason    string
}

// recordCancellationRefundLedger writes the succeeded `refund_transactions` row
// for a cancellation refund, inside the caller's transaction so the ledger row
// and the money move commit together.
//
// Status is `succeeded` rather than the Coordinator's reserve-then-confirm pair
// because this path records AFTER the gateway call has returned successfully —
// there is no window here to reserve for. A duplicate key is treated as success:
// the sweep re-drives ExecuteCancellationRefund, and a refund that already
// recorded must not fail on its second pass.
func recordCancellationRefundLedger(tx *gorm.DB, in cancellationRefundLedgerInput) error {
	if in.Order == nil || in.Amount <= 0 {
		return nil // nothing moved — nothing to record
	}
	provider := in.Order.PaymentProvider
	refundID := in.GatewayRefundID
	if refundID == "" {
		// Wallet/loyalty-funded: no gateway leg exists. Record it as a wallet refund
		// so the row still states truthfully where the money went, rather than
		// claiming a gateway refund with an empty id.
		provider = "wallet"
		refundID = in.WalletRef
	}
	now := time.Now()
	row := models.RefundTransaction{
		ID:                uuid.New(),
		OrderID:           in.Order.ID,
		Provider:          provider,
		ProviderPaymentID: in.Order.RazorpayPaymentID,
		ProviderRefundID:  refundID,
		Amount:            Round2(in.Amount),
		CurrencyCode:      "INR",
		Status:            models.RefundTxnSucceeded,
		Reason:            in.Reason,
		IdempotencyKey:    orderrefund.IdempotencyKeyFor(in.Order.ID, orderrefund.ScopeFull),
		ScopeID:           orderrefund.ScopeFull,
		Actor:             "customer",
		CompletedAt:       &now,
	}
	if err := tx.Create(&row).Error; err != nil {
		var existing models.RefundTransaction
		if e := tx.Where("idempotency_key = ?", row.IdempotencyKey).First(&existing).Error; e == nil {
			return nil // already on the ledger (sweep re-drive, or the coordinator got there)
		}
		return err
	}
	return nil
}
