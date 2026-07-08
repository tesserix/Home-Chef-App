package services

// escrow_ledger_reconcile.go — #398 item 1. The escrow conservation ledger: a read-only
// daily audit that compares the platform's recorded payout-hold state against what the
// gateway reports for each hold's Route transfer, and writes a PaymentDrift row on any
// mismatch. The alarm it exists to raise: an aggregate that is REFUNDED (or withheld/reversed)
// while its chef/rider transfer is still settled at the gateway — i.e. the chef was paid AND the
// customer was refunded for the same order (audit #10 / the issue's acceptance criterion).
//
// Detection only — it NEVER moves money (the payout-reconcile cron is the self-heal). It scans
// only aggregates that carry an escrow hold (payout_hold_status <> ''), so with the escrow flags
// OFF (no hold is ever parked) the scan set is empty and the sweep is a pure no-op. Bounded per
// run; a gateway-fetch failure skips that aggregate (retried next run), never a false drift.
//
// SCOPE (v1): the ORDER aggregate (transfers are order-linked → FetchOrderTransfers). Meal-plan-day
// and group holds are DIRECT transfers (PayoutTransferID, no order link) reconciled via
// FetchTransfer(id) — a follow-up slice; the PaymentDrift table + detector generalise to them.

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// driftFinding is one detected discrepancy for an aggregate (pre-persistence).
type driftFinding struct {
	kind          models.PaymentDriftKind
	detail        string
	expectedPaise int64
	gatewayPaise  int64
}

// transferLedger summarises an aggregate's gateway transfers into the three conservation
// buckets: reversed (clawed back), settled (paid out, net of reversal), and held (still on hold).
// By construction reversed+settled+held == total transferred, so the buckets ARE the
// conservation of the captured advance across the payout leg.
type transferLedger struct {
	reversedPaise int64
	settledPaise  int64 // released to the recipient, not on hold, net of any partial reversal
	heldPaise     int64 // still on hold at the gateway
}

// summariseTransfers folds the gateway transfer list into the conservation buckets.
func summariseTransfers(transfers []TransferResponse) transferLedger {
	var l transferLedger
	for _, t := range transfers {
		if t.ID == "" {
			continue
		}
		reversed := int64(t.AmountReversed)
		remaining := int64(t.Amount) - reversed
		if remaining < 0 {
			remaining = 0
		}
		l.reversedPaise += reversed
		if t.OnHold {
			l.heldPaise += remaining
		} else {
			l.settledPaise += remaining
		}
	}
	return l
}

// orderLedgerState is the platform's recorded view of an order's payout, distilled to what the
// drift check needs (so the detector is a pure function, unit-testable without a DB or gateway).
type orderLedgerState struct {
	holdStatus models.PayoutHoldStatus
	// mustNotPay is true when the recorded state means the chef must NOT keep the money:
	// the order is refunded/cancelled, or the hold was withheld/reversed.
	mustNotPay bool
}

// detectOrderTransferDrift compares the recorded order state against the gateway transfer
// summary and returns every discrepancy. Pure — no I/O.
func detectOrderTransferDrift(st orderLedgerState, l transferLedger) []driftFinding {
	var out []driftFinding

	// The severe class: money must NOT be with the recipient, but it (still) is.
	if st.mustNotPay {
		if l.settledPaise > 0 {
			// Report the TOTAL not-yet-reversed amount (settled + still-held) as the money at
			// risk — a mixed chef-settled + rider-still-held set on a refunded order must not
			// understate the exposure to just the settled slice.
			out = append(out, driftFinding{
				kind:         models.DriftChefPaidOnRefund,
				detail:       "order is refunded/withheld but its transfer is settled at the gateway (chef paid AND customer refunded)",
				gatewayPaise: l.settledPaise + l.heldPaise,
			})
		} else if l.heldPaise > 0 {
			out = append(out, driftFinding{
				kind:         models.DriftReversedButNotAtGateway,
				detail:       "order is refunded/withheld but its transfer is still on hold at the gateway (not yet clawed back)",
				gatewayPaise: l.heldPaise,
			})
		}
		return out // a mustNotPay order's held/released expectations below don't apply
	}

	switch st.holdStatus {
	case models.PayoutHoldReleased:
		// We think the chef was paid; the gateway must show it off hold.
		if l.heldPaise > 0 {
			out = append(out, driftFinding{
				kind:         models.DriftReleasedButHeldAtGateway,
				detail:       "payout_hold_status=released but the transfer is still on hold at the gateway (chef not actually paid)",
				gatewayPaise: l.heldPaise,
			})
		}
	case models.PayoutHoldAwaitingConfirmation, models.PayoutHoldReleaseEligible, models.PayoutHoldDisputed:
		// We think the payout is still held; the gateway must not have settled it.
		if l.settledPaise > 0 {
			out = append(out, driftFinding{
				kind:         models.DriftHeldButReleasedAtGateway,
				detail:       "payout is still held on the platform but the transfer is already settled at the gateway (premature release)",
				gatewayPaise: l.settledPaise,
			})
		}
	}
	return out
}

// RunEscrowLedgerReconcile scans escrow-held orders, fetches each order's gateway transfers,
// detects drift, and reconciles the PaymentDrift ledger (open new drifts, resolve healed ones).
// Returns the number of drift FINDINGS detected in this scan's batch (bounded by sweepBatchLimit,
// so not necessarily the table's total open count). Pure no-op while no hold is parked.
//
// SCOPE CAVEAT: only orders that carry an escrow hold (payout_hold_status <> '') are scanned.
// payment-linked transfers are created on-hold at CHECKOUT but payout_hold_status is set at
// DELIVERY, so an order refunded BEFORE delivery (hold never parked) is out of scan set — safe
// today because nothing can settle that transfer pre-delivery (release/reverse are delivery-gated),
// but revisit if a pre-delivery settlement path is ever added. Meal-plan-day / group direct
// transfers are the separate v1 exclusion noted in the file header.
func RunEscrowLedgerReconcile() int {
	rz := GetRazorpay()

	var orders []models.Order
	// Only aggregates that carry an escrow hold AND a gateway order to fetch transfers from.
	// payout_hold_status <> '' is empty for every order while the escrow flags are OFF.
	if err := database.DB.
		Select("id", "order_number", "razorpay_order_id", "payout_hold_status", "refunded_at", "status").
		Where("payout_hold_status <> '' AND razorpay_order_id <> ''").
		Order("updated_at ASC").
		Limit(sweepBatchLimit).
		Find(&orders).Error; err != nil {
		log.Printf("escrow-ledger: load held orders failed: %v", err)
		return 0
	}

	openCount := 0
	for i := range orders {
		o := &orders[i]
		if rz == nil {
			continue // gateway not configured — can't reconcile, skip silently (logged at startup)
		}
		transfers, err := rz.FetchOrderTransfers(o.RazorpayOrderID)
		if err != nil {
			// Our own fetch failure is not the aggregate's drift — skip, retry next run.
			log.Printf("escrow-ledger: fetch transfers for order %s failed: %v", o.ID, err)
			continue
		}
		st := orderLedgerState{
			holdStatus: o.PayoutHoldStatus,
			mustNotPay: o.RefundedAt != nil ||
				o.Status == models.OrderStatusRefunded || o.Status == models.OrderStatusCancelled ||
				o.PayoutHoldStatus == models.PayoutHoldWithheld || o.PayoutHoldStatus == models.PayoutHoldReversed,
		}
		findings := detectOrderTransferDrift(st, summariseTransfers(transfers))
		openCount += reconcileDriftRows(aggTypeOrder, o.ID, findings)
	}

	if openCount > 0 {
		log.Printf("escrow-ledger: %d OPEN payment-drift row(s) after scan of %d held order(s)", openCount, len(orders))
		CaptureBackgroundError(fmt.Errorf("escrow conservation reconcile found %d open payment drift(s)", openCount))
	}
	return openCount
}

// reconcileDriftRows makes the PaymentDrift ledger match `findings` for one aggregate: it opens
// (or refreshes) a row per detected kind and RESOLVES any previously-open row whose kind is no
// longer present (the aggregate self-healed). Idempotent — a re-scan of a still-drifting
// aggregate updates the existing open row rather than duplicating it. Returns the open count.
func reconcileDriftRows(aggType string, aggID uuid.UUID, findings []driftFinding) int {
	present := make(map[string]bool, len(findings))
	for _, f := range findings {
		present[f.kind] = true
		log.Printf("escrow-ledger DRIFT: agg=%s/%s kind=%s expected=%d gateway=%d detail=%q",
			aggType, aggID, f.kind, f.expectedPaise, f.gatewayPaise, f.detail)
		if err := upsertOpenDrift(aggType, aggID, f); err != nil {
			log.Printf("escrow-ledger: upsert drift %s/%s/%s failed: %v", aggType, aggID, f.kind, err)
		}
	}
	// Resolve any open row whose kind is no longer present (healed).
	if err := database.DB.Model(&models.PaymentDrift{}).
		Where("agg_type = ? AND agg_id = ? AND resolved_at IS NULL", aggType, aggID).
		Where("kind NOT IN ?", keysOf(present)).
		Update("resolved_at", time.Now()).Error; err != nil {
		log.Printf("escrow-ledger: resolve healed drifts for %s/%s failed: %v", aggType, aggID, err)
	}
	return len(findings)
}

// upsertOpenDrift ensures exactly one OPEN row for (aggType, aggID, kind): refresh the existing
// open row's amounts/detail, or insert a new one. Find-then-write (not a DB unique upsert) so it
// stays portable across sqlite (tests) and Postgres.
func upsertOpenDrift(aggType string, aggID uuid.UUID, f driftFinding) error {
	var existing models.PaymentDrift
	err := database.DB.
		Where("agg_type = ? AND agg_id = ? AND kind = ? AND resolved_at IS NULL", aggType, aggID, f.kind).
		First(&existing).Error
	if err == nil {
		return database.DB.Model(&models.PaymentDrift{}).Where("id = ?", existing.ID).
			Updates(map[string]any{"detail": f.detail, "expected_paise": f.expectedPaise, "gateway_paise": f.gatewayPaise}).Error
	}
	return database.DB.Create(&models.PaymentDrift{
		ID:      uuid.New(), // set explicitly — the gen_random_uuid() default is Postgres-only
		AggType: aggType, AggID: aggID, Kind: f.kind,
		Detail: f.detail, ExpectedPaise: f.expectedPaise, GatewayPaise: f.gatewayPaise,
	}).Error
}

// keysOf returns the keys of a presence set (or a single "" so a NOT IN never matches the empty
// set — resolving ALL open rows for the aggregate when no drift is present).
func keysOf(m map[string]bool) []string {
	if len(m) == 0 {
		return []string{""}
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
