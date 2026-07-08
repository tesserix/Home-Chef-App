package services

// escrow_ledger_daygroup_test.go — #398 day/group slice. The escrow conservation ledger extends
// from the ORDER aggregate to meal-plan-day and group-order DIRECT chef transfers (PayoutTransferID,
// fetched by id — not order-linked). Same detector/PaymentDrift machinery; these runner tests drive
// the day/group scans against canned single-transfer gateway responses via the shared razorpay seam.

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// singleTransferHandler serves FetchTransfer(id) — GET /transfers/{id} — from a map keyed by
// transfer id. An unknown id 404s (a fetch failure the scan skips, not a drift).
func singleTransferHandler(byID map[string]map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/transfers/") {
			if obj, ok := byID[path.Base(r.URL.Path)]; ok {
				_ = json.NewEncoder(w).Encode(obj)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

func seedHeldDay(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, status models.MealPlanDayStatus, transferID string, refunded bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var refTxn any // nil → NULL refund_txn_id
	if refunded {
		refTxn = uuid.New().String()
	}
	require.NoError(t, db.Exec(
		`INSERT INTO meal_plan_days (id, status, payout_hold_status, payout_transfer_id, refund_txn_id, updated_at)
		 VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)`,
		id.String(), string(status), string(hold), transferID, refTxn).Error)
	return id
}

func seedHeldGroup(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, status models.GroupOrderStatus, transferID string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO group_orders (id, status, payout_hold_status, payout_transfer_id, updated_at)
		 VALUES (?,?,?,?,CURRENT_TIMESTAMP)`,
		id.String(), string(status), string(hold), transferID).Error)
	return id
}

// --- meal-plan-day ---

// A refunded day (refund_txn_id set) whose chef transfer is still settled at the gateway opens the
// chef-paid-on-refund alarm keyed to the day aggregate; a re-scan showing it reversed resolves it.
func TestRunEscrowLedgerReconcile_Day_OpensThenResolves(t *testing.T) {
	db := setupLedgerDB(t)
	// A failed day is frozen `disputed`; the refund_txn_id (not the status) marks it refunded.
	id := seedHeldDay(t, db, models.PayoutHoldDisputed, models.MealPlanDayFailed, "trfDay1", true)

	withRazorpayTestServer(t, singleTransferHandler(map[string]map[string]any{
		"trfDay1": {"id": "trfDay1", "amount": 8000, "on_hold": false},
	}))
	require.Equal(t, 1, RunEscrowLedgerReconcile())
	open := openDrifts(t, db, id)
	require.Len(t, open, 1)
	require.Equal(t, models.DriftChefPaidOnRefund, open[0].Kind)
	require.Equal(t, aggTypeMealPlanDay, open[0].AggType)
	require.Equal(t, int64(8000), open[0].GatewayPaise)

	withRazorpayTestServer(t, singleTransferHandler(map[string]map[string]any{
		"trfDay1": {"id": "trfDay1", "amount": 8000, "amount_reversed": 8000, "on_hold": false},
	}))
	require.Equal(t, 0, RunEscrowLedgerReconcile())
	require.Empty(t, openDrifts(t, db, id), "healed day drift is resolved, not left open")
}

// A day we still think is held (awaiting) whose transfer is genuinely on hold at the gateway is
// consistent — no drift.
func TestRunEscrowLedgerReconcile_Day_ConsistentHeld_NoDrift(t *testing.T) {
	db := setupLedgerDB(t)
	id := seedHeldDay(t, db, models.PayoutHoldAwaitingConfirmation, models.MealPlanDayDelivered, "trfDay3", false)
	withRazorpayTestServer(t, singleTransferHandler(map[string]map[string]any{
		"trfDay3": {"id": "trfDay3", "amount": 8000, "on_hold": true},
	}))
	require.Equal(t, 0, RunEscrowLedgerReconcile())
	require.Empty(t, openDrifts(t, db, id))
}

// A held day with NO gateway transfer (payout_transfer_id empty — flags-OFF steady state) is out
// of the scan set: nothing to fetch, no drift.
func TestRunEscrowLedgerReconcile_Day_SkipsNoTransfer(t *testing.T) {
	db := setupLedgerDB(t)
	id := seedHeldDay(t, db, models.PayoutHoldWithheld, models.MealPlanDayRefunded, "", true)
	withRazorpayTestServer(t, singleTransferHandler(map[string]map[string]any{}))
	require.Equal(t, 0, RunEscrowLedgerReconcile(), "held day with no gateway transfer is not scanned")
	require.Empty(t, openDrifts(t, db, id))
}

// GAP (#398 review): a day's PayoutTransferID is created at plan approval (HoldChefPayouts), but
// payout_hold_status only advances off none on delivery. So a day refunded BEFORE it starts (hold
// still none) whose gateway ReverseTransfer failed leaves the chef transfer stranded on hold. The
// scan keys on payout_transfer_id (the precise "escrow money at the gateway" signal), NOT
// payout_hold_status, so this stranded-money row stays visible and the not-yet-clawed-back alarm
// fires. (payout_transfer_id is only ever set behind the escrow flag, so flags-OFF is still a no-op.)
func TestRunEscrowLedgerReconcile_Day_RefundedHoldNone_StrandedTransfer(t *testing.T) {
	db := setupLedgerDB(t)
	id := seedHeldDay(t, db, models.PayoutHoldNone, models.MealPlanDayRefunded, "trfDay9", true)
	withRazorpayTestServer(t, singleTransferHandler(map[string]map[string]any{
		"trfDay9": {"id": "trfDay9", "amount": 8000, "on_hold": true}, // reverse failed → still held
	}))
	require.Equal(t, 1, RunEscrowLedgerReconcile())
	open := openDrifts(t, db, id)
	require.Len(t, open, 1)
	require.Equal(t, models.DriftReversedButNotAtGateway, open[0].Kind)
	require.Equal(t, aggTypeMealPlanDay, open[0].AggType)
}

// A day still mid-flight (hold none, transfer set, NOT refunded) whose transfer is legitimately on
// hold is consistent — scanning it via the payout_transfer_id key must not open a false drift.
func TestRunEscrowLedgerReconcile_Day_HoldNoneNotRefunded_NoDrift(t *testing.T) {
	db := setupLedgerDB(t)
	id := seedHeldDay(t, db, models.PayoutHoldNone, models.MealPlanDayConfirmed, "trfDay10", false)
	withRazorpayTestServer(t, singleTransferHandler(map[string]map[string]any{
		"trfDay10": {"id": "trfDay10", "amount": 8000, "on_hold": true},
	}))
	require.Equal(t, 0, RunEscrowLedgerReconcile())
	require.Empty(t, openDrifts(t, db, id))
}

// --- group-order ---

// A cancelled group (its refund proxy — groups have no per-group refund column) whose chef transfer
// is still settled at the gateway opens the chef-paid-on-refund alarm keyed to the group aggregate.
func TestRunEscrowLedgerReconcile_Group_ChefPaidOnCancel(t *testing.T) {
	db := setupLedgerDB(t)
	id := seedHeldGroup(t, db, models.PayoutHoldReleased, models.GroupOrderCancelled, "trfGrp1")
	withRazorpayTestServer(t, singleTransferHandler(map[string]map[string]any{
		"trfGrp1": {"id": "trfGrp1", "amount": 12000, "on_hold": false},
	}))
	require.Equal(t, 1, RunEscrowLedgerReconcile())
	open := openDrifts(t, db, id)
	require.Len(t, open, 1)
	require.Equal(t, models.DriftChefPaidOnRefund, open[0].Kind)
	require.Equal(t, aggTypeGroupOrder, open[0].AggType)
	require.Equal(t, int64(12000), open[0].GatewayPaise)
}

// Re-scanning a still-drifting group updates the existing open row, never duplicates.
func TestRunEscrowLedgerReconcile_Group_Idempotent(t *testing.T) {
	db := setupLedgerDB(t)
	id := seedHeldGroup(t, db, models.PayoutHoldWithheld, models.GroupOrderDelivered, "trfGrp2")
	withRazorpayTestServer(t, singleTransferHandler(map[string]map[string]any{
		"trfGrp2": {"id": "trfGrp2", "amount": 5000, "on_hold": false},
	}))
	require.Equal(t, 1, RunEscrowLedgerReconcile())
	require.Equal(t, 1, RunEscrowLedgerReconcile())
	require.Len(t, openDrifts(t, db, id), 1, "a re-scan updates the existing open row, never duplicates")
}
