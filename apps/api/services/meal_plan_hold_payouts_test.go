package services

// meal_plan_hold_payouts_test.go — #200 (tiffin E2E). Covers HoldChefPayouts, which creates the
// on-hold Route transfer per accepted day at confirm time. Two invariants matter for the money:
//   - idempotency: a re-run (retry, double-confirm) must NOT double-create transfers — the
//     PayoutTransferID guard skips days already held, and Razorpay dedups on the per-day key.
//   - the security gate: never hold against a plan whose advance was not captured (the on-hold
//     transfers draw from the platform balance, so that would pay the chef with platform money).
// Drives CreateTransfer (POST /transfers) against the shared httptest seam and counts the calls.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupHoldPayoutDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plans (id TEXT PRIMARY KEY, currency TEXT, total REAL,
		subtotal REAL, tax REAL, escrow_payment_id TEXT, created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, status TEXT,
		payout_transfer_id TEXT, commission_rate REAL, price REAL, created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE platform_settings (key TEXT PRIMARY KEY, value TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE audit_logs (id TEXT PRIMARY KEY, user_id TEXT, action TEXT,
		entity_type TEXT, entity_id TEXT, old_value TEXT, new_value TEXT, ip_address TEXT, user_agent TEXT,
		correlation_id TEXT, created_at DATETIME)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// holdTransferServer stubs POST /transfers (CreateTransfer), returning a fresh on-hold transfer id
// per call and incrementing creates so a test can assert how many transfers were actually created.
func holdTransferServer(t *testing.T, creates *int32) {
	t.Helper()
	withRazorpayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transfers") {
			n := atomic.AddInt32(creates, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": fmt.Sprintf("trf_hold_%d", n), "on_hold": true})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
}

func seedHoldPlan(t *testing.T, db *gorm.DB, escrowPaymentID string) uuid.UUID {
	t.Helper()
	planID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO meal_plans (id, currency, total, subtotal, tax, escrow_payment_id) VALUES (?,?,?,?,?,?)`,
		planID.String(), "INR", 480.0, 400.0, 40.0, escrowPaymentID).Error)
	return planID
}

func seedHoldDay(t *testing.T, db *gorm.DB, planID uuid.UUID, status models.MealPlanDayStatus, price float64) uuid.UUID {
	t.Helper()
	dayID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO meal_plan_days (id, meal_plan_id, status, payout_transfer_id, price) VALUES (?,?,?,?,?)`,
		dayID.String(), planID.String(), string(status), "", price).Error)
	return dayID
}

func loadHoldPlan(t *testing.T, db *gorm.DB, planID uuid.UUID) models.MealPlan {
	t.Helper()
	var plan models.MealPlan
	require.NoError(t, db.Preload("Days").First(&plan, "id = ?", planID).Error)
	return plan
}

func transferIDOf(t *testing.T, db *gorm.DB, dayID uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payout_transfer_id FROM meal_plan_days WHERE id = ?`, dayID.String()).Scan(&s).Error)
	return s
}

// One hold transfer per PAYABLE day; a re-run creates no new transfers (PayoutTransferID guard).
func TestHoldChefPayouts_Idempotent_SecondCallNoNewTransfers(t *testing.T) {
	escrowFlag(t, true)
	db := setupHoldPayoutDB(t)
	var creates int32
	holdTransferServer(t, &creates)

	planID := seedHoldPlan(t, db, "pay_adv") // advance captured
	dayA := seedHoldDay(t, db, planID, models.MealPlanDayAccepted, 200.0)
	dayB := seedHoldDay(t, db, planID, models.MealPlanDayConfirmed, 200.0)
	dayReq := seedHoldDay(t, db, planID, models.MealPlanDayRequested, 200.0) // not payable → skipped

	plan := loadHoldPlan(t, db, planID)
	require.NoError(t, HoldChefPayouts(db, &plan, "acc_chef_1"))
	require.Equal(t, int32(2), atomic.LoadInt32(&creates), "one hold per payable day; requested day skipped")
	idA, idB := transferIDOf(t, db, dayA), transferIDOf(t, db, dayB)
	require.NotEmpty(t, idA)
	require.NotEmpty(t, idB)
	require.Empty(t, transferIDOf(t, db, dayReq), "non-payable day is never held")

	// Re-run against a freshly loaded plan: every payable day already carries a transfer id.
	plan2 := loadHoldPlan(t, db, planID)
	require.NoError(t, HoldChefPayouts(db, &plan2, "acc_chef_1"))
	require.Equal(t, int32(2), atomic.LoadInt32(&creates), "re-run creates no new transfers")
	require.Equal(t, idA, transferIDOf(t, db, dayA), "held transfer id is unchanged")
	require.Equal(t, idB, transferIDOf(t, db, dayB))
}

// SECURITY: a plan whose advance was not captured (escrow_payment_id blank) must never hold a
// payout — the on-hold transfer would draw from the platform balance.
func TestHoldChefPayouts_UnpaidPlan_ErrorsAndHoldsNothing(t *testing.T) {
	escrowFlag(t, true)
	db := setupHoldPayoutDB(t)
	var creates int32
	holdTransferServer(t, &creates)

	planID := seedHoldPlan(t, db, "") // NOT captured
	day := seedHoldDay(t, db, planID, models.MealPlanDayAccepted, 200.0)

	plan := loadHoldPlan(t, db, planID)
	require.ErrorContains(t, HoldChefPayouts(db, &plan, "acc_chef_1"), "advance payment not captured")
	require.Equal(t, int32(0), atomic.LoadInt32(&creates), "no transfer created against an unpaid plan")
	require.Empty(t, transferIDOf(t, db, day))
}

// Escrow OFF → pure no-op.
func TestHoldChefPayouts_EscrowOff_NoOp(t *testing.T) {
	escrowFlag(t, false)
	db := setupHoldPayoutDB(t)
	var creates int32
	holdTransferServer(t, &creates)

	planID := seedHoldPlan(t, db, "pay_adv")
	seedHoldDay(t, db, planID, models.MealPlanDayAccepted, 200.0)

	plan := loadHoldPlan(t, db, planID)
	require.NoError(t, HoldChefPayouts(db, &plan, "acc_chef_1"))
	require.Equal(t, int32(0), atomic.LoadInt32(&creates))
}

// A missing chef linked account is rejected (cannot route a hold).
func TestHoldChefPayouts_NoChefAccount_Errors(t *testing.T) {
	escrowFlag(t, true)
	db := setupHoldPayoutDB(t)
	var creates int32
	holdTransferServer(t, &creates)

	planID := seedHoldPlan(t, db, "pay_adv")
	seedHoldDay(t, db, planID, models.MealPlanDayAccepted, 200.0)

	plan := loadHoldPlan(t, db, planID)
	require.ErrorContains(t, HoldChefPayouts(db, &plan, ""), "no Razorpay linked account")
	require.Equal(t, int32(0), atomic.LoadInt32(&creates))
}
