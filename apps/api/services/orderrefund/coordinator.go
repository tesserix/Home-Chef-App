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

	"github.com/homechef/api/models"
	"github.com/homechef/api/services/refundreserve"
)

var (
	// ErrRefundsDisabled — the kill switch is off. Deliberately distinguishable
	// so a caller can tell "refused by policy" from "the gateway broke".
	ErrRefundsDisabled = errors.New("orderrefund: refunds are disabled")
	// ErrNoCapturedPayment — nothing was captured, so there is nothing to refund.
	ErrNoCapturedPayment = errors.New("orderrefund: order has no captured payment")
	// ErrExceedsRemaining — the request is larger than what is left to refund.
	ErrExceedsRemaining = errors.New("orderrefund: amount exceeds the refundable remaining")
	// ErrRefundInFlight — a sibling refund path holds the order's claim right now.
	// Distinct from the errors above because it is TRANSIENT: the same command will
	// succeed once that sibling finishes, so a caller may retry it. The other two
	// are terminal for this order.
	ErrRefundInFlight = errors.New("orderrefund: another refund is in flight for this order")

	// errOverAsked is internal: it unwinds the reserve transaction (releasing the
	// claim via rollback) when an explicit amount exceeded the remaining.
	errOverAsked = errors.New("over-asked")
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

// hasPendingRefund reports whether an attempt for this order is mid-gateway. Only
// coordinator-driven refunds leave a ledger row, so a legacy path's in-flight
// refund is invisible here and reads as terminal — conservative in the safe
// direction (we refuse rather than race it) and self-correcting as #690 migrates
// the remaining paths onto the ledger.
func (c *Coordinator) hasPendingRefund(ctx context.Context, orderID uuid.UUID) (bool, error) {
	var n int64
	if err := c.db.WithContext(ctx).Model(&models.RefundTransaction{}).
		Where("order_id = ? AND status = ?", orderID, models.RefundTxnPending).
		Count(&n).Error; err != nil {
		return false, fmt.Errorf("orderrefund: check in-flight refunds for order %s: %w", orderID, err)
	}
	return n > 0, nil
}

// reserveLedgerRow writes the pending ledger row for this attempt, BEFORE the
// gateway call, and returns its id. `retryID` is set when a previous attempt for
// this exact scope failed and we are reusing its row.
func (c *Coordinator) reserveLedgerRow(tx *gorm.DB, cmd RefundCommand, key string, retryID uuid.UUID, amount float64, order models.Order) (uuid.UUID, error) {
	if retryID != uuid.Nil {
		// Reusing a FAILED row: reset it to pending for this attempt. Clearing
		// failure_reason/completed_at matters — a stale failure on a row that then
		// succeeds would read as "this refund failed" during reconciliation. We
		// reuse rather than insert because idempotency_key is UNIQUE: a second row
		// would hit the constraint and strand the refund forever.
		if err := tx.Model(&models.RefundTransaction{}).Where("id = ?", retryID).
			Updates(map[string]any{
				"status":         models.RefundTxnPending,
				"amount":         amount,
				"failure_reason": "",
				"completed_at":   nil,
			}).Error; err != nil {
			return uuid.Nil, err
		}
		return retryID, nil
	}
	row := models.RefundTransaction{
		ID: uuid.New(), OrderID: cmd.OrderID, Provider: order.PaymentProvider,
		ProviderPaymentID: order.RazorpayPaymentID, Amount: amount, CurrencyCode: "INR",
		Status: models.RefundTxnPending, Reason: cmd.Reason,
		IdempotencyKey: key, ScopeID: cmd.ScopeID, Actor: cmd.Actor,
	}
	if err := tx.Create(&row).Error; err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
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

	if cmd.Amount != nil && models.RoundAmount(*cmd.Amount) <= 0 {
		return RefundResult{}, fmt.Errorf("orderrefund: refund amount must be positive, got %v", *cmd.Amount)
	}

	// ── reserve — the SHARED claim, not a private one ─────────────────────────
	//
	// This MUST be refundreserve's reservation and not a mechanism of our own.
	// The legacy paths mutually exclude each other by claiming payment_status
	// completed→refunded on the order row; that shared claim IS the cross-path
	// mutex. A coordinator that reserved differently would be invisible to them:
	// a sibling would see payment_status='completed' and an un-incremented
	// refund_amount while we were mid-gateway, win its own claim, and refund the
	// same money twice. That is not hypothetical — it is exactly what
	// TestInterop_LegacyFullRefund_CannotRaceACoordinatorRefundInFlight proved
	// against the first draft of this file. For the whole of #690's migration a
	// migrated and an un-migrated path are concurrent, so sharing the claim is
	// the precondition for migrating anything at all.
	//
	// This also replaces the pending-ledger-sum cap the first draft used: the
	// reservation increments refund_amount UP FRONT, so an in-flight refund is
	// already subtracted from every other path's `remaining`. The ledger stays
	// the audit trail and the idempotency record — it is no longer the cap.
	requested := 0.0 // <=0 means "the full remaining" to ReserveRefund
	if cmd.Amount != nil {
		requested = models.RoundAmount(*cmd.Amount)
	}
	// The reservation and its ledger row commit TOGETHER. Separately, a crash in
	// between would leave refund_amount incremented with no ledger row: the order
	// reads as refunded, the customer never gets the money, and nothing records
	// that a refund was ever owed. One transaction makes that unreachable.
	var (
		res      refundreserve.Reservation
		ledgerID uuid.UUID
	)
	if err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var e error
		if res, e = refundreserve.ReserveRefundInTx(tx, cmd.OrderID, requested); e != nil {
			return e
		}
		if !res.Won {
			return nil // the caller maps the outcome to an error below
		}
		// An explicit amount that ReserveRefund silently capped is an over-ask, not
		// a smaller refund. Bail before writing a ledger row for it; the rollback
		// releases the reservation for us.
		if cmd.Amount != nil && res.Amount < requested {
			return errOverAsked
		}
		ledgerID, e = c.reserveLedgerRow(tx, cmd, key, retryLedgerID, res.Amount, res.Order)
		return e
	}); err != nil {
		if errors.Is(err, errOverAsked) {
			return RefundResult{}, fmt.Errorf("%w: asked %v, remaining %v", ErrExceedsRemaining, requested, res.Amount)
		}
		return RefundResult{}, fmt.Errorf("orderrefund: reserve order %s: %w", cmd.OrderID, err)
	}
	switch res.Outcome {
	case refundreserve.OutcomeNotPayable:
		return RefundResult{}, ErrNoCapturedPayment
	case refundreserve.OutcomeNothingLeft:
		return RefundResult{}, fmt.Errorf("%w: the order is already fully refunded", ErrExceedsRemaining)
	case refundreserve.OutcomeLostClaim:
		return RefundResult{}, ErrRefundInFlight
	case refundreserve.OutcomeAlreadyRefunded:
		// payment_status='refunded' is ambiguous: the terminal fully-refunded state,
		// OR the short-lived mutex a partial holds while it is at the gateway. The
		// ledger disambiguates — this is exactly what it is for. Getting it wrong
		// means telling a caller "already refunded" (terminal, don't retry) about a
		// refund that is merely a second away from releasing.
		inFlight, e := c.hasPendingRefund(ctx, cmd.OrderID)
		if e != nil {
			return RefundResult{}, e
		}
		if inFlight {
			return RefundResult{}, ErrRefundInFlight
		}
		return RefundResult{}, fmt.Errorf("%w: the order is already fully refunded", ErrExceedsRemaining)
	}

	amount := res.Amount
	release := func() { refundreserve.ReleaseRefundReservation(c.db, cmd.OrderID, amount) }

	// ── gateway — OUTSIDE any transaction ────────────────────────────────────
	refundID, gwErr := c.gateway.RefundPayment(ctx, res.Order.RazorpayPaymentID, toPaise(amount), key, map[string]string{
		"order_id": cmd.OrderID.String(),
		"scope":    cmd.ScopeID,
		"reason":   cmd.Reason,
	})

	// ── tx#2 — finalize ──────────────────────────────────────────────────────
	now := time.Now()
	if gwErr != nil {
		// No money moved, so RELEASE the reservation — payment_status back to
		// completed and refund_amount decremented — or the order is left looking
		// refunded forever and every later refund path reads a remaining of 0.
		release()
		// Best-effort: a failure to record must not mask the gateway error the
		// caller needs to see.
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
		// refund_amount is NOT incremented here — the reservation already did it,
		// up front, which is what made the in-flight refund visible to every other
		// path. Incrementing again would double-count and under-refund the next
		// caller.
		//
		// What IS left to us is the terminal marker, and it differs by outcome —
		// this is the convention every legacy path re-implements (payment.go
		// InitiateRefund, chef_order_cancel), and encapsulating it here is the
		// point of the coordinator:
		//
		//   FULL    — payment_status stays 'refunded' and refunded_at is stamped.
		//   PARTIAL — payment_status must go BACK to 'completed'. ReserveRefund
		//             flipped it as a short-lived MUTEX, not as a state change; a
		//             partial that leaves it 'refunded' freezes the order — the
		//             chef's payout hold never releases and no later refund can
		//             ever reserve again. refunded_at stays NULL for the same
		//             reason (every release-side payout guard blocks the whole
		//             hold on refunded_at IS NOT NULL).
		markers := map[string]any{"payment_status": models.PaymentCompleted}
		if res.FullRefund {
			markers["payment_status"] = models.PaymentRefunded
			markers["refunded_at"] = now
		}
		return tx.Model(&models.Order{}).Where("id = ?", cmd.OrderID).Updates(markers).Error
	}); err != nil {
		// Money MOVED but our books didn't record it. Never revert the ledger row
		// here — that would invite a re-refund of money already sent. Surface it
		// for reconciliation.
		return RefundResult{}, fmt.Errorf("orderrefund: refund %s succeeded at the gateway but bookkeeping failed: %w", refundID, err)
	}

	return RefundResult{Amount: amount, FullRefund: res.FullRefund, ProviderRefundID: refundID}, nil
}
