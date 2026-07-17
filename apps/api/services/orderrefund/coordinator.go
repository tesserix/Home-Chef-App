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
//	tx#1  take the SHARED reservation + write a pending ledger row, together
//	--    deliver the money OUTSIDE any transaction
//	tx#2  finalize the ledger + stamp the order's terminal markers
//
// The gateway call is outside a transaction deliberately. order_issue.go credits
// the wallet INSIDE its tx; putting a Razorpay round-trip there would hold a row
// lock across a network call. That is why #691 needs this coordinator rather
// than a one-line swap.
//
// WHY THE RESERVATION IS SHARED (#690)
//
// The reservation MUST be refundreserve's — the same claim on payment_status that
// every legacy path takes — and not a mechanism of our own. That claim IS the
// cross-path mutex. A coordinator that reserved differently is invisible to the
// paths it has not replaced yet: a sibling would see payment_status='completed'
// and an un-incremented refund_amount while we were mid-gateway, win its own
// claim, and refund the same money twice. The first draft of this file did
// exactly that, and services/refund_coordinator_interop_test.go proves it. Since
// #690 migrates the call sites one at a time, a migrated and an un-migrated path
// are concurrent for the whole migration — sharing the claim is the precondition
// for migrating anything at all.
//
// The reservation increments refund_amount UP FRONT, which is what makes an
// in-flight refund visible to every other path. So the ledger is NOT the cap: it
// is the audit trail and the idempotency record.
//
// NOT A PORT of mark8ly's coordinator — same shape, different money model (Route
// escrow + chef payouts vs store returns), and mark8ly's pending-sum cap does not
// apply here because HomeChef already had a shared reservation to join.

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

// GatewayRequest is one delivery of refund money to a customer.
type GatewayRequest struct {
	OrderID uuid.UUID
	// Amount is the FULL amount owed back, in major units — not a gateway share.
	// How it is delivered is the Gateway's business: a wallet-funded order only
	// captured (Total − WalletApplied) at the provider, so the provider cannot
	// refund more than that and the rest goes back as store credit (#141). That
	// split must NOT leak into the coordinator — it would silently under-reserve.
	Amount float64
	// IdempotencyKey is a LOGICAL operation id; the provider client normalizes it
	// into the provider's dedup header (#574).
	IdempotencyKey string
	Reason         string
	Actor          string
	ScopeID        string
}

// Gateway delivers refund money and returns a reference for it.
//
// It is deliberately ORDER-shaped rather than payment-shaped. The first draft took
// (providerPaymentID, amountPaise), which silently assumed Razorpay: it cannot
// express a wallet refund (no provider payment at all) or a Stripe one (a payment
// INTENT, different id, different minor-unit rules), and #691 is precisely the bug
// of a path that assumed one provider. Routing lives behind this interface, in
// services, where the provider clients already are — which also keeps orderrefund
// free of the services import that would be a cycle.
type Gateway interface {
	RefundPayment(ctx context.Context, req GatewayRequest) (string, error)
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

// Scope constants for refunds whose key must match a legacy path's byte-for-byte
// while #690 migrates the call sites one at a time.
const (
	// ScopeFull is the once-per-order full cancellation refund. The scope is "full"
	// and NOT something tidier like "cancel" for one specific reason: it makes
	// idempotencyKey(order, ScopeFull) identical to services.RefundFullIdempotencyKey
	// — "refund:<order>:full". That shared key is deliberate (see
	// gateway_idempotency.go): if two cancellation paths ever both fire for one
	// order, the GATEWAY dedups them. It is the second line of defence behind the
	// reservation, and it only works while every path derives the same key. A
	// migrated path with a prettier scope would quietly drop out of it, exactly
	// while migration makes a collision most likely.
	ScopeFull = "full"
)

// idempotencyKey builds the gateway-facing key for a logical refund. Keyed on
// (order, scope) so a retry of the SAME logical refund reuses it and the gateway
// dedups, while two different scopes stay distinct.
func idempotencyKey(orderID uuid.UUID, scopeID string) string {
	return fmt.Sprintf("refund:%s:%s", orderID, scopeID)
}

// IdempotencyKeyFor exposes the key a command WOULD use. Exported so the legacy
// paths' shared-key invariant can be pinned by a test rather than by a comment.
func IdempotencyKeyFor(orderID uuid.UUID, scopeID string) string {
	return idempotencyKey(orderID, scopeID)
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
	// A nil Amount is a FULL-refund intent, and full refunds take a STRICTER claim: they must
	// also see refunded_at IS NULL. That half of the mutex excludes the sibling paths that
	// stamp refunded_at without flipping payment_status (cancellation_execute.go). Reserving a
	// full refund on payment_status alone sails past an order a sibling already refunded and
	// refunds it again — caught by TestRefundOrderForCancellation_AlreadyRefundedNoOp when
	// this first routed everything through the partial claim.
	requested := 0.0 // <=0 means "the full remaining"
	intent := refundreserve.IntentFull
	if cmd.Amount != nil {
		requested = models.RoundAmount(*cmd.Amount)
		intent = refundreserve.IntentPartial
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
		if res, e = refundreserve.ReserveInTx(tx, cmd.OrderID, requested, intent); e != nil {
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
		// The provider columns are ours to read, not the reservation's to hand over: they
		// are for the LEDGER, and making the shared reservation fetch them for us would
		// couple every other caller to columns it never needed. Same transaction, same
		// lock, so it is the same consistent snapshot.
		var provider models.Order
		if e = tx.Select("payment_provider", "razorpay_payment_id").
			First(&provider, "id = ?", cmd.OrderID).Error; e != nil {
			return e
		}
		ledgerID, e = c.reserveLedgerRow(tx, cmd, key, retryLedgerID, res.Amount, provider)
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
	// Release the claim we actually took: the full one also stamped refunded_at, and leaving
	// that behind on a failed refund would freeze the order — every payout guard blocks the
	// whole chef hold on refunded_at IS NOT NULL, and the next full refund would lose the
	// claim forever.
	release := func() {
		if intent == refundreserve.IntentFull {
			refundreserve.ReleaseFullRefundReservation(c.db, cmd.OrderID, amount)
			return
		}
		refundreserve.ReleaseRefundReservation(c.db, cmd.OrderID, amount)
	}

	// ── gateway — OUTSIDE any transaction ────────────────────────────────────
	refundID, gwErr := c.gateway.RefundPayment(ctx, GatewayRequest{
		OrderID: cmd.OrderID, Amount: amount, IdempotencyKey: key,
		Reason: cmd.Reason, Actor: cmd.Actor, ScopeID: cmd.ScopeID,
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
			// IntentFull stamped refunded_at as part of its claim, so this only matters for a
			// PARTIAL-intent request that turned out to exhaust the order (a concurrent refund
			// shrank the remainder below what was asked). It still ends the order, so it still
			// gets the terminal marker.
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
