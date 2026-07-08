package services

// escrow_ledger_reconcile.go — #398. The escrow conservation ledger: a read-only
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
// SCOPE: all three payout-hold aggregates. The ORDER aggregate's transfers are order-linked
// (FetchOrderTransfers → a list). Meal-plan-day and group holds are DIRECT transfers
// (PayoutTransferID, no order link) read one-by-one via FetchTransfer(id) — the same PaymentDrift
// table + pure detector serve all three (a single direct transfer is just a one-element list).

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

// aggLedgerState is the platform's recorded view of an order's payout, distilled to what the
// drift check needs (so the detector is a pure function, unit-testable without a DB or gateway).
type aggLedgerState struct {
	holdStatus models.PayoutHoldStatus
	// mustNotPay is true when the recorded state means the chef must NOT keep the money:
	// the order is refunded/cancelled, or the hold was withheld/reversed.
	mustNotPay bool
}

// detectTransferDrift compares the recorded order state against the gateway transfer
// summary and returns every discrepancy. Pure — no I/O.
func detectTransferDrift(st aggLedgerState, l transferLedger) []driftFinding {
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

// RunEscrowLedgerReconcile scans every escrow-held payout aggregate — orders, meal-plan-days,
// and group-orders — fetches each hold's gateway transfer(s), detects drift, and reconciles the
// PaymentDrift ledger (open new drifts, resolve healed ones). Returns the total number of OPEN
// drift findings across the three scans' batches (each bounded by sweepBatchLimit, so not
// necessarily the table's total open count). Pure no-op while no hold is parked (flags OFF).
//
// SCOPE CAVEAT: the ORDER scan keys on a non-empty payout_hold_status (its transfers are order-linked;
// no single transfer-id column to key on). An order's payment-linked transfers are created on-hold
// at CHECKOUT but payout_hold_status is set at DELIVERY, so an order refunded BEFORE delivery (hold
// never parked) is out of that scan set — safe today because nothing can SETTLE that transfer
// pre-delivery (release is delivery-gated), but revisit if a pre-delivery settlement path is added.
// The DIRECT-transfer aggregates (meal-plan-day / group-order) instead key on payout_transfer_id,
// so they do NOT share that pre-delivery blind spot — see each scan's doc below.
func RunEscrowLedgerReconcile() int {
	rz := GetRazorpay()
	openCount := scanHeldOrders(rz) + scanHeldMealPlanDays(rz) + scanHeldGroupOrders(rz)
	if openCount > 0 {
		log.Printf("escrow-ledger: %d OPEN payment-drift row(s) after scan", openCount)
		CaptureBackgroundError(fmt.Errorf("escrow conservation reconcile found %d open payment drift(s)", openCount))
	}
	return openCount
}

// scanHeldOrders reconciles the ORDER aggregate. Its held chef/rider transfers are order-linked,
// so the whole set is fetched via FetchOrderTransfers and folded together (a mixed
// chef-settled + rider-still-held set on a refunded order reports the total exposure).
func scanHeldOrders(rz *RazorpayClient) int {
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
		st := aggLedgerState{
			holdStatus: o.PayoutHoldStatus,
			mustNotPay: o.RefundedAt != nil ||
				o.Status == models.OrderStatusRefunded || o.Status == models.OrderStatusCancelled ||
				o.PayoutHoldStatus == models.PayoutHoldWithheld || o.PayoutHoldStatus == models.PayoutHoldReversed,
		}
		findings := detectTransferDrift(st, summariseTransfers(transfers))
		openCount += reconcileDriftRows(aggTypeOrder, o.ID, findings)
	}
	return openCount
}

// scanHeldMealPlanDays reconciles the MEAL-PLAN-DAY aggregate. A day's held chef payout is a
// single DIRECT Route transfer (PayoutTransferID, not order-linked) read via FetchTransfer(id).
//
// The scan keys on payout_transfer_id — NOT payout_hold_status like the order scan. For the
// direct-transfer aggregates PayoutTransferID is the precise "escrow money exists at the gateway"
// signal, and it is only ever set behind the escrow flag (HoldChefPayouts is gated on
// MealPlanEscrowActive), so flags-OFF remains a pure no-op. This is stricter than the order scan
// on purpose: a day's transfer is created at plan-approval but its hold only advances off none on
// delivery, so a day refunded BEFORE it starts (hold still none) whose gateway ReverseTransfer
// failed would otherwise strand a live transfer invisibly. Keying on the transfer id keeps that
// stranded-money row in scope (mustNotPay via refund_txn_id → not-yet-clawed-back alarm).
//
// mustNotPay mirrors the order: the day is refunded (refund_txn_id set — the per-day refund marker
// regardless of status label), cancelled/refunded by status, or its hold was withheld/reversed.
// A `failed` day (frozen disputed, admin-pending) is deliberately NOT mustNotPay — its held
// transfer is still expected on hold, so the detector's held-vs-gateway branch applies.
func scanHeldMealPlanDays(rz *RazorpayClient) int {
	var days []models.MealPlanDay
	if err := database.DB.
		Select("id", "status", "payout_hold_status", "payout_transfer_id", "refund_txn_id").
		Where("payout_transfer_id <> ''").
		Order("updated_at ASC").
		Limit(sweepBatchLimit).
		Find(&days).Error; err != nil {
		log.Printf("escrow-ledger: load held meal-plan days failed: %v", err)
		return 0
	}

	openCount := 0
	for i := range days {
		d := &days[i]
		if rz == nil {
			continue
		}
		t, err := rz.FetchTransfer(d.PayoutTransferID)
		if err != nil {
			log.Printf("escrow-ledger: fetch transfer for meal-plan-day %s failed: %v", d.ID, err)
			continue
		}
		st := aggLedgerState{
			holdStatus: d.PayoutHoldStatus,
			mustNotPay: d.RefundTxnID != nil ||
				d.Status == models.MealPlanDayRefunded || d.Status == models.MealPlanDayCancelled ||
				d.PayoutHoldStatus == models.PayoutHoldWithheld || d.PayoutHoldStatus == models.PayoutHoldReversed,
		}
		findings := detectTransferDrift(st, summariseTransfers(oneTransfer(t)))
		openCount += reconcileDriftRows(aggTypeMealPlanDay, d.ID, findings)
	}
	return openCount
}

// scanHeldGroupOrders reconciles the GROUP-ORDER aggregate. Like the meal-plan-day it holds a
// single DIRECT transfer read via FetchTransfer(id), and keys on payout_transfer_id (the precise
// escrow-money signal, only set behind the escrow flag → flags-OFF no-op preserved).
//
// mustNotPay: groups have NO per-group refund column (refunds are per-participant), and both refund
// paths — host cancel and delivery-failure resolve — set status to `cancelled` (committed before
// the post-commit hold reverse). So `cancelled` is the group's refund proxy; withheld/reversed is
// the direct hold signal. `failed`/`expired` are excluded for the same reason as the day.
func scanHeldGroupOrders(rz *RazorpayClient) int {
	var groups []models.GroupOrder
	if err := database.DB.
		Select("id", "status", "payout_hold_status", "payout_transfer_id").
		Where("payout_transfer_id <> ''").
		Order("updated_at ASC").
		Limit(sweepBatchLimit).
		Find(&groups).Error; err != nil {
		log.Printf("escrow-ledger: load held group orders failed: %v", err)
		return 0
	}

	openCount := 0
	for i := range groups {
		g := &groups[i]
		if rz == nil {
			continue
		}
		t, err := rz.FetchTransfer(g.PayoutTransferID)
		if err != nil {
			log.Printf("escrow-ledger: fetch transfer for group-order %s failed: %v", g.ID, err)
			continue
		}
		st := aggLedgerState{
			holdStatus: g.PayoutHoldStatus,
			mustNotPay: g.Status == models.GroupOrderCancelled ||
				g.PayoutHoldStatus == models.PayoutHoldWithheld || g.PayoutHoldStatus == models.PayoutHoldReversed,
		}
		findings := detectTransferDrift(st, summariseTransfers(oneTransfer(t)))
		openCount += reconcileDriftRows(aggTypeGroupOrder, g.ID, findings)
	}
	return openCount
}

// oneTransfer wraps a single DIRECT transfer as a one-element list so it feeds the same
// summariseTransfers fold as the order's transfer list. A nil transfer folds to an empty ledger.
func oneTransfer(t *TransferResponse) []TransferResponse {
	if t == nil {
		return nil
	}
	return []TransferResponse{*t}
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
