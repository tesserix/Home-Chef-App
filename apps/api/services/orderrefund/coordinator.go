package orderrefund

// coordinator.go — the single door for refund money (#689, part of #687).
//
// WHY THIS EXISTS
//
// HomeChef refunds work, but money leaves through five independently-written
// call sites (cancellation_order_refund, cancellation_execute, payment.go,
// chef_order_cancel, order_issue). Correctness is held together by CONVENTION:
// every path must remember to reserve under a row lock (#609) and to use the
// right shared idempotency key (gateway_idempotency.go). #549/#582/#586/#609/
// #618/#624 are all races between those paths — and order_issue.go forgot the
// provider routing entirely, which is the #691 bug: it credits an unspendable
// wallet whatever the customer paid with.
//
// Duplication is the vulnerability. This is the one place gateway money moves.
//
// THE SAGA
//
//	tx#1  lock the order, validate against the remaining INCLUDING in-flight
//	      pending ledger rows, reserve a pending row
//	--    call the gateway OUTSIDE any transaction
//	tx#2  finalize the ledger + bookkeep the order
//
// The gateway call is outside a transaction deliberately. order_issue.go credits
// the wallet INSIDE its tx; putting a Razorpay round-trip there would hold a row
// lock across a network call. That is why #691 needs this coordinator rather
// than a one-line swap.
//
// WHY THE PENDING SUM MATTERS
//
// The shared idempotency keys make the GATEWAY dedup a same-key retry. They do
// nothing for two DIFFERENT logical refunds (say a cancellation and an order
// issue) racing: each could read the same stale refund_amount, each clear the
// cap, and both move real money. Counting in-flight pending rows under the row
// lock is what closes that.
//
// NOT A PORT of mark8ly's coordinator — same shape, different money model (Route
// escrow + chef payouts vs store returns).

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

var (
	// ErrRefundsDisabled — the kill switch is off. Deliberately distinguishable
	// so a caller can tell "refused by policy" from "the gateway broke".
	ErrRefundsDisabled = errors.New("orderrefund: refunds are disabled")
	// ErrNoCapturedPayment — nothing was captured, so there is nothing to refund.
	ErrNoCapturedPayment = errors.New("orderrefund: order has no captured payment")
	// ErrExceedsRemaining — the request is larger than what is left to refund.
	ErrExceedsRemaining = errors.New("orderrefund: amount exceeds the refundable remaining")
)

// scopeIDPattern guards the LOGICAL scope, not the gateway charset.
//
// Unlike mark8ly — which passes the raw ScopeID into the provider's dedup header
// and therefore must restrict it to [A-Za-z0-9_-] — HomeChef HASHES the logical
// key (gateway_idempotency.go normalizeIdempotencyKey → 32-char hex), so any
// charset is already safe by the time it reaches Razorpay. That is why the
// existing RefundFullIdempotencyKey ("refund:<uuid>:full") works with colons.
//
// So we only reject what would break OUR identity: an empty or whitespace-only
// scope would collapse distinct refunds onto one key and silently dedup a
// legitimate second refund away.
var scopeIDPattern = regexp.MustCompile(`^\S+$`)

// Gateway is the narrow surface the coordinator needs. An interface (not the
// concrete Razorpay client) so tests can fake it while still exercising the real
// ledger and cap logic against a DB.
type Gateway interface {
	// RefundPayment refunds amountPaise against providerPaymentID and returns the
	// provider's refund id. idempotencyKey is a LOGICAL operation id — the client
	// normalizes it into the provider's dedup header (#574).
	RefundPayment(ctx context.Context, providerPaymentID string, amountPaise int, idempotencyKey string, notes map[string]string) (string, error)
}

// RefundCommand is one logical refund. Amount==nil means "the full remaining".
type RefundCommand struct {
	OrderID uuid.UUID
	Amount  *float64 // nil ⇒ full remaining
	Reason  string
	Actor   string // "customer" | "chef" | "admin" | "system" | "user:<id>"
	// ScopeID is the logical refund's identity — "cancel", "issue:<id>",
	// "line:<id>", "admin:<id>". Two commands with the same (OrderID, ScopeID)
	// are the SAME refund and must move money once.
	ScopeID string
}

// RefundResult is what a successful (or already-completed) Refund reports back.
type RefundResult struct {
	Amount           float64
	FullRefund       bool
	ProviderRefundID string
	// Replayed is true when this call matched an existing succeeded ledger row
	// and moved no new money.
	Replayed bool
}

// Coordinator runs the refund saga.
type Coordinator struct {
	db      *gorm.DB
	gateway Gateway
	enabled bool
}

// NewCoordinator builds a Coordinator. `enabled` is the REFUND_GATEWAY_ENABLED
// kill switch and should default ON — a refund switch that silently defaults off
// is how the unspendable-wallet trap (#691) shipped.
func NewCoordinator(db *gorm.DB, gateway Gateway, enabled bool) *Coordinator {
	return &Coordinator{db: db, gateway: gateway, enabled: enabled}
}

// idempotencyKey builds the gateway-facing key for a logical refund. Keyed on
// (order, scope) so a retry of the SAME logical refund reuses it and the gateway
// dedups, while two different scopes stay distinct.
func idempotencyKey(orderID uuid.UUID, scopeID string) string {
	return fmt.Sprintf("refund:%s:%s", orderID, scopeID)
}

// toPaise converts major units to integer minor units. Rounds rather than
// truncates: 0.1+0.2 style float error must not silently shave a paisa off a
// customer's refund.
func toPaise(amount float64) int {
	return int(amount*100 + 0.5)
}

// Refund moves refund money for an order. Safe to retry with the same ScopeID.
func (c *Coordinator) Refund(ctx context.Context, cmd RefundCommand) (RefundResult, error) {
	if !c.enabled {
		return RefundResult{}, ErrRefundsDisabled
	}
	if !scopeIDPattern.MatchString(cmd.ScopeID) {
		return RefundResult{}, fmt.Errorf("orderrefund: ScopeID %q must be non-empty and contain no whitespace — an empty scope collapses distinct refunds onto one idempotency key", cmd.ScopeID)
	}

	key := idempotencyKey(cmd.OrderID, cmd.ScopeID)

	// Set when a previous attempt for this exact scope FAILED and we are reusing
	// its ledger row (idempotency_key is unique — see below).
	var retryLedgerID uuid.UUID

	// Replay check first: a caller retrying after a success must get a no-op, not
	// an error and not a second refund.
	var existing models.RefundTransaction
	if err := c.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&existing).Error; err == nil {
		if existing.Status == models.RefundTxnSucceeded {
			return RefundResult{
				Amount:           existing.Amount,
				ProviderRefundID: existing.ProviderRefundID,
				Replayed:         true,
			}, nil
		}
		// A pending row means an attempt is in flight (or was stranded by a
		// crash). Refuse rather than race it; a sweep reconciles stranded rows.
		if existing.Status == models.RefundTxnPending {
			return RefundResult{}, fmt.Errorf("orderrefund: a refund for scope %q is already in flight", cmd.ScopeID)
		}
		// Failed: retry is legitimate (the gateway rejected it, no money moved).
		// But idempotency_key is UNIQUE, so we must REUSE the row rather than
		// insert a second one — otherwise the retry hits the constraint and the
		// refund is stranded forever. Reset it to pending in the reservation
		// below.
		retryLedgerID = existing.ID
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return RefundResult{}, err
	}

	// ── tx#1 — lock, validate against the remaining INCLUDING in-flight, reserve
	var (
		ledgerID   uuid.UUID
		amount     float64
		fullRefund bool
		paymentID  string
		provider   string
	)
	if err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockTx := tx
		// Postgres-only; a behaviour-preserving no-op on sqlite. Same approach as
		// services/refund_reserve.go — the deterministic tests pin the contract.
		if tx.Dialector.Name() == "postgres" {
			lockTx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}

		var o models.Order
		if e := lockTx.Select("id", "total", "refund_amount", "payment_status", "payment_provider", "razorpay_payment_id").
			First(&o, "id = ?", cmd.OrderID).Error; e != nil {
			return e
		}
		if o.PaymentStatus != models.PaymentCompleted {
			return ErrNoCapturedPayment
		}
		provider = o.PaymentProvider
		paymentID = o.RazorpayPaymentID

		// In-flight pending refunds count against the remaining. Without this,
		// two different-scope refunds each clear the cap on a stale balance.
		var pending float64
		if e := tx.Model(&models.RefundTransaction{}).
			Where("order_id = ? AND status = ?", cmd.OrderID, models.RefundTxnPending).
			Select("COALESCE(SUM(amount), 0)").Scan(&pending).Error; e != nil {
			return e
		}

		remaining := models.RoundAmount(o.Total - o.RefundAmount - pending)
		if remaining <= 0 {
			return ErrExceedsRemaining
		}

		amount = remaining
		if cmd.Amount != nil {
			amount = models.RoundAmount(*cmd.Amount)
			if amount <= 0 {
				return fmt.Errorf("orderrefund: refund amount must be positive, got %v", amount)
			}
			if amount > remaining {
				return fmt.Errorf("%w: asked %v, remaining %v", ErrExceedsRemaining, amount, remaining)
			}
		}
		fullRefund = models.RoundAmount(o.RefundAmount+pending+amount) >= o.Total

		if retryLedgerID != uuid.Nil {
			// Reusing a FAILED row: reset it to pending for this attempt. Clearing
			// failure_reason/completed_at matters — a stale failure on a row that
			// then succeeds would read as "this refund failed" during reconciliation.
			if e := tx.Model(&models.RefundTransaction{}).Where("id = ?", retryLedgerID).
				Updates(map[string]any{
					"status":         models.RefundTxnPending,
					"amount":         amount,
					"failure_reason": "",
					"completed_at":   nil,
				}).Error; e != nil {
				return e
			}
			ledgerID = retryLedgerID
			return nil
		}

		row := models.RefundTransaction{
			ID: uuid.New(), OrderID: cmd.OrderID, Provider: provider,
			ProviderPaymentID: paymentID, Amount: amount, CurrencyCode: "INR",
			Status: models.RefundTxnPending, Reason: cmd.Reason,
			IdempotencyKey: key, ScopeID: cmd.ScopeID, Actor: cmd.Actor,
		}
		if e := tx.Create(&row).Error; e != nil {
			return e
		}
		ledgerID = row.ID
		return nil
	}); err != nil {
		return RefundResult{}, err
	}

	// ── gateway — OUTSIDE any transaction ────────────────────────────────────
	refundID, gwErr := c.gateway.RefundPayment(ctx, paymentID, toPaise(amount), key, map[string]string{
		"order_id": cmd.OrderID.String(),
		"scope":    cmd.ScopeID,
		"reason":   cmd.Reason,
	})

	// ── tx#2 — finalize ──────────────────────────────────────────────────────
	now := time.Now()
	if gwErr != nil {
		// Mark failed and leave the order's balance untouched so a retry can
		// re-refund. Best-effort: a failure to record must not mask the gateway
		// error the caller needs to see.
		if e := c.db.WithContext(ctx).Model(&models.RefundTransaction{}).
			Where("id = ?", ledgerID).Updates(map[string]any{
			"status":         models.RefundTxnFailed,
			"failure_reason": gwErr.Error(),
			"completed_at":   &now,
		}).Error; e != nil {
			return RefundResult{}, fmt.Errorf("orderrefund: gateway failed (%v) AND ledger finalize failed: %w", gwErr, e)
		}
		return RefundResult{}, fmt.Errorf("orderrefund: gateway refund failed: %w", gwErr)
	}

	if err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Model(&models.RefundTransaction{}).Where("id = ?", ledgerID).Updates(map[string]any{
			"status":             models.RefundTxnSucceeded,
			"provider_refund_id": refundID,
			"completed_at":       &now,
		}).Error; e != nil {
			return e
		}
		// COALESCE so a NULL refund_amount increments from 0 rather than
		// collapsing to NULL (same reasoning as refund_reserve.go).
		return tx.Model(&models.Order{}).Where("id = ?", cmd.OrderID).
			Update("refund_amount", gorm.Expr("COALESCE(refund_amount, 0) + ?", amount)).Error
	}); err != nil {
		// Money MOVED but our books didn't record it. Never revert the ledger row
		// here — that would invite a re-refund of money already sent. Surface it
		// for reconciliation.
		return RefundResult{}, fmt.Errorf("orderrefund: refund %s succeeded at the gateway but bookkeeping failed: %w", refundID, err)
	}

	return RefundResult{Amount: amount, FullRefund: fullRefund, ProviderRefundID: refundID}, nil
}
