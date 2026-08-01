package services

// chef_penalty_test.go — the chef-cancellation levy (#834 item 6).
//
// Every branch here moves money out of a chef's settlement, so the arithmetic, both guards
// (grace allowance and admin waiver), the lead-time boundary, and idempotency on a
// double-submitted cancel are all pinned.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func setupPenaltyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE chef_penalties (id TEXT PRIMARY KEY, chef_id TEXT, user_id TEXT,
		kind TEXT, status TEXT, source_key TEXT UNIQUE, order_id TEXT, reference TEXT, currency TEXT,
		basis_amount REAL, rate_percent REAL, amount REAL, lead_hours REAL, reason TEXT,
		deducted_statement_id TEXT, deducted_at DATETIME, waived_by TEXT, waived_at DATETIME,
		waive_reason TEXT, occurred_at DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE weekly_statements (id TEXT PRIMARY KEY, chef_id TEXT, user_id TEXT,
		week_start DATETIME, week_end DATETIME, currency TEXT, orders_count INTEGER, gross_revenue REAL,
		platform_commission REAL, cgst REAL, sgst REAL, igst REAL, tds REAL, penalty_deductions REAL,
		bonus_additions REAL DEFAULT 0, net_payout REAL, status TEXT, paid_at DATETIME, payout_ref TEXT, created_at DATETIME)`).Error)
	return db
}

// penaltyPolicy installs the levy config: 6%, under 4h notice, `grace` free cancellations per 30d.
func penaltyPolicy(t *testing.T, enabled bool, grace int) {
	t.Helper()
	p := DefaultPlatformPolicy()
	p.ChefCancelPenaltyEnabled = enabled
	p.ChefCancelPenaltyPercent = 6
	p.ChefCancelPenaltyLeadHours = 4
	p.ChefCancelPenaltyGraceCount = grace
	p.ChefCancelPenaltyGraceDays = 30
	withPlatformPolicy(t, p)
}

// levy is the common call: a ₹500 order cancelled with `lead` hours of notice.
func levy(t *testing.T, db *gorm.DB, chefID, userID uuid.UUID, lead float64) *models.ChefPenalty {
	t.Helper()
	p, err := LevyChefCancelPenalty(db, chefID, userID, uuid.New(), "HC-TEST", 500, lead)
	require.NoError(t, err)
	return p
}

// 6% of the order value, on a cancellation inside the notice window.
func TestLevyChefCancelPenalty_Arithmetic(t *testing.T) {
	penaltyPolicy(t, true, 0)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()

	p := levy(t, db, chefID, userID, 1)
	require.NotNil(t, p)
	require.Equal(t, 30.0, p.Amount, "6% of ₹500")
	require.Equal(t, 500.0, p.BasisAmount)
	require.Equal(t, 6.0, p.RatePercent)
	require.Equal(t, models.ChefPenaltyPending, p.Status)
	require.Equal(t, models.ChefPenaltyCancelLate, p.Kind)
}

// The lead boundary, from both sides. 4h notice is enough; a minute less is not.
func TestLevyChefCancelPenalty_LeadBoundary(t *testing.T) {
	cases := []struct {
		name      string
		lead      float64
		wantLevy  bool
		rationale string
	}{
		{"a day's notice", 24, false, "ample warning — no levy"},
		{"just over 4h", 4.01, false, ""},
		{"exactly 4h", 4, false, "the boundary is enough notice"},
		{"just under 4h", 3.99, true, "inside the window"},
		{"an hour out", 1, true, ""},
		{"already past service", -2, true, "the worst case of all"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			penaltyPolicy(t, true, 0)
			db := setupPenaltyDB(t)
			p := levy(t, db, uuid.New(), uuid.New(), tc.lead)
			if tc.wantLevy {
				require.NotNil(t, p, tc.rationale)
			} else {
				require.Nil(t, p, tc.rationale)
			}
		})
	}
}

// GUARD 1 — the first cancellation in the window is exempt, and is RECORDED as a waived zero
// row so the allowance is actually consumed rather than granted forever.
func TestLevyChefCancelPenalty_GraceAllowance(t *testing.T) {
	penaltyPolicy(t, true, 1)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()

	first := levy(t, db, chefID, userID, 1)
	require.Nil(t, first, "the first cancellation is forgiven")

	var rows []models.ChefPenalty
	require.NoError(t, db.Where("chef_id = ?", chefID).Find(&rows).Error)
	require.Len(t, rows, 1, "the exemption is still recorded")
	require.Equal(t, models.ChefPenaltyWaived, rows[0].Status)
	require.Equal(t, 0.0, rows[0].Amount)

	second := levy(t, db, chefID, userID, 1)
	require.NotNil(t, second, "the allowance is spent — the second cancellation is charged")
	require.Equal(t, 30.0, second.Amount)
}

// The allowance is per chef: one chef's cancellations never spend another's.
func TestLevyChefCancelPenalty_GraceIsPerChef(t *testing.T) {
	penaltyPolicy(t, true, 1)
	db := setupPenaltyDB(t)
	chefA, chefB := uuid.New(), uuid.New()

	require.Nil(t, levy(t, db, chefA, uuid.New(), 1))
	require.NotNil(t, levy(t, db, chefA, uuid.New(), 1))
	require.Nil(t, levy(t, db, chefB, uuid.New(), 1), "chef B still has their own allowance")
}

// A cancellation older than the window no longer counts against the allowance.
func TestLevyChefCancelPenalty_GraceWindowExpires(t *testing.T) {
	penaltyPolicy(t, true, 1)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()

	// An exemption used 60 days ago — outside the 30-day window.
	require.NoError(t, db.Exec(`INSERT INTO chef_penalties (id, chef_id, user_id, kind, status, source_key, amount, occurred_at)
		VALUES (?,?,?,?,?,?,?,?)`, uuid.NewString(), chefID.String(), userID.String(),
		string(models.ChefPenaltyCancelLate), string(models.ChefPenaltyWaived), "chefcancel:old", 0.0,
		time.Now().UTC().Add(-60*24*time.Hour)).Error)

	require.Nil(t, levy(t, db, chefID, userID, 1), "the stale exemption has aged out; this one is free again")
}

// Idempotency: a retried or double-submitted cancel levies once — and crucially does not burn
// a second grace slot on the way.
func TestLevyChefCancelPenalty_Idempotent(t *testing.T) {
	penaltyPolicy(t, true, 0)
	db := setupPenaltyDB(t)
	chefID, userID, orderID := uuid.New(), uuid.New(), uuid.New()

	first, err := LevyChefCancelPenalty(db, chefID, userID, orderID, "HC-1", 500, 1)
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := LevyChefCancelPenalty(db, chefID, userID, orderID, "HC-1", 500, 1)
	require.NoError(t, err)
	require.Nil(t, second, "the retry raises nothing new")

	var count int64
	require.NoError(t, db.Model(&models.ChefPenalty{}).Where("chef_id = ?", chefID).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

// Disabled, or a zero rate, means no levy at all — the runtime kill switch.
func TestLevyChefCancelPenalty_Disabled(t *testing.T) {
	penaltyPolicy(t, false, 0)
	db := setupPenaltyDB(t)
	require.Nil(t, levy(t, db, uuid.New(), uuid.New(), 0))
}

// GUARD 2 — an admin waiver zeroes a pending levy and records who and why.
func TestWaiveChefPenalty(t *testing.T) {
	penaltyPolicy(t, true, 0)
	db := setupPenaltyDB(t)
	chefID, userID, adminID := uuid.New(), uuid.New(), uuid.New()
	p := levy(t, db, chefID, userID, 1)
	require.NotNil(t, p)

	require.NoError(t, WaiveChefPenalty(db, p.ID, adminID, "kitchen flood, photos supplied"))

	var got models.ChefPenalty
	require.NoError(t, db.First(&got, "id = ?", p.ID).Error)
	require.Equal(t, models.ChefPenaltyWaived, got.Status)
	require.Equal(t, 0.0, got.Amount)
	require.Equal(t, "kitchen flood, photos supplied", got.WaiveReason)
	require.NotNil(t, got.WaivedAt)

	total, err := PendingChefPenaltyTotal(db, chefID)
	require.NoError(t, err)
	require.Equal(t, 0.0, total)

	// A second waiver is rejected rather than silently succeeding.
	require.ErrorIs(t, WaiveChefPenalty(db, p.ID, adminID, "again"), ErrPenaltyNotPending)
}

// A levy already netted off a settlement cannot be waived — that would need a credit, not a
// waiver, and silently zeroing it would leave the statement's arithmetic wrong.
func TestWaiveChefPenalty_AlreadyDeducted(t *testing.T) {
	penaltyPolicy(t, true, 0)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()
	p := levy(t, db, chefID, userID, 1)
	require.NotNil(t, p)

	stmt := &models.WeeklyStatement{ID: uuid.New(), ChefID: chefID, UserID: userID, NetPayout: 1000}
	require.NoError(t, db.Create(stmt).Error)
	deducted, err := ApplyChefPenaltiesToStatement(db, stmt)
	require.NoError(t, err)
	require.Equal(t, 30.0, deducted)

	require.ErrorIs(t, WaiveChefPenalty(db, p.ID, uuid.New(), "too late"), ErrPenaltyNotPending)
}

// The settlement deduction: pending levies come off the payout and land as the statement's
// penalty line.
func TestApplyChefPenaltiesToStatement(t *testing.T) {
	penaltyPolicy(t, true, 0)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()
	require.NotNil(t, levy(t, db, chefID, userID, 1))
	require.NotNil(t, levy(t, db, chefID, userID, 1))

	stmt := &models.WeeklyStatement{ID: uuid.New(), ChefID: chefID, UserID: userID, NetPayout: 1000}
	require.NoError(t, db.Create(stmt).Error)

	deducted, err := ApplyChefPenaltiesToStatement(db, stmt)
	require.NoError(t, err)
	require.Equal(t, 60.0, deducted, "two ₹30 levies")

	var saved models.WeeklyStatement
	require.NoError(t, db.First(&saved, "id = ?", stmt.ID).Error)
	require.Equal(t, 60.0, saved.PenaltyDeductions, "the statement's invoice line")
	require.Equal(t, 940.0, saved.NetPayout, "the payout is what the chef actually receives")

	var claimed []models.ChefPenalty
	require.NoError(t, db.Where("chef_id = ?", chefID).Find(&claimed).Error)
	for _, p := range claimed {
		require.Equal(t, models.ChefPenaltyDeducted, p.Status)
		require.NotNil(t, p.DeductedStatementID)
		require.Equal(t, stmt.ID, *p.DeductedStatementID)
	}
}

// Re-running statement generation must not deduct the same levies twice.
func TestApplyChefPenaltiesToStatement_Idempotent(t *testing.T) {
	penaltyPolicy(t, true, 0)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()
	require.NotNil(t, levy(t, db, chefID, userID, 1))

	stmt := &models.WeeklyStatement{ID: uuid.New(), ChefID: chefID, UserID: userID, NetPayout: 1000}
	require.NoError(t, db.Create(stmt).Error)

	first, err := ApplyChefPenaltiesToStatement(db, stmt)
	require.NoError(t, err)
	require.Equal(t, 30.0, first)

	second, err := ApplyChefPenaltiesToStatement(db, stmt)
	require.NoError(t, err)
	require.Equal(t, 0.0, second, "nothing left pending")

	var saved models.WeeklyStatement
	require.NoError(t, db.First(&saved, "id = ?", stmt.ID).Error)
	require.Equal(t, 970.0, saved.NetPayout, "deducted once, not twice")
}

// A levy larger than the week's payout floors it at zero rather than going negative.
func TestApplyChefPenaltiesToStatement_FloorsAtZero(t *testing.T) {
	penaltyPolicy(t, true, 0)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()
	_, err := LevyChefCancelPenalty(db, chefID, userID, uuid.New(), "HC-BIG", 100000, 1)
	require.NoError(t, err)

	stmt := &models.WeeklyStatement{ID: uuid.New(), ChefID: chefID, UserID: userID, NetPayout: 200}
	require.NoError(t, db.Create(stmt).Error)
	_, err = ApplyChefPenaltiesToStatement(db, stmt)
	require.NoError(t, err)

	var saved models.WeeklyStatement
	require.NoError(t, db.First(&saved, "id = ?", stmt.ID).Error)
	require.Equal(t, 0.0, saved.NetPayout, "a settlement never goes negative")
}

// A chef with no levies is untouched.
func TestApplyChefPenaltiesToStatement_NoPenalties(t *testing.T) {
	penaltyPolicy(t, true, 0)
	db := setupPenaltyDB(t)
	stmt := &models.WeeklyStatement{ID: uuid.New(), ChefID: uuid.New(), UserID: uuid.New(), NetPayout: 1000}
	require.NoError(t, db.Create(stmt).Error)

	deducted, err := ApplyChefPenaltiesToStatement(db, stmt)
	require.NoError(t, err)
	require.Equal(t, 0.0, deducted)
	require.Equal(t, 1000.0, stmt.NetPayout)
}

// ── Gateway-fee levy (#885) ─────────────────────────────────────────────────────────────────

// gatewayFeePolicy installs the gateway-fee levy config: `percent`%, grace/stack as given.
func gatewayFeePolicy(t *testing.T, enabled bool, percent float64, stack bool) {
	t.Helper()
	p := DefaultPlatformPolicy()
	p.GatewayFeeLevyEnabled = enabled
	p.GatewayFeeLevyPercent = percent
	p.GatewayFeeLevyGraceEnabled = false
	p.GatewayFeeLevyGraceCount = 0
	p.GatewayFeeLevyGraceDays = 0
	p.GatewayFeeLevyStackWithCancelLevy = stack
	withPlatformPolicy(t, p)
}

// levyFee is the common gateway-fee call.
func levyFee(t *testing.T, db *gorm.DB, chefID, userID, orderID uuid.UUID, provider string, refundedAmount float64, key string) *models.ChefPenalty {
	t.Helper()
	p, err := LevyGatewayFeePenalty(db, chefID, userID, orderID, "HC-TEST", provider, refundedAmount, key)
	require.NoError(t, err)
	return p
}

// 2% of the REFUNDED amount (not a hypothetical order total).
func TestLevyGatewayFeePenalty_Arithmetic(t *testing.T) {
	gatewayFeePolicy(t, true, 2, false)
	db := setupPenaltyDB(t)
	chefID, userID, orderID := uuid.New(), uuid.New(), uuid.New()

	p := levyFee(t, db, chefID, userID, orderID, models.PaymentProviderCashfree, 250, "refund:order:full")
	require.NotNil(t, p)
	require.Equal(t, 5.0, p.Amount, "2% of ₹250")
	require.Equal(t, 250.0, p.BasisAmount)
	require.Equal(t, 2.0, p.RatePercent)
	require.Equal(t, models.ChefPenaltyPending, p.Status)
	require.Equal(t, models.ChefPenaltyGatewayFee, p.Kind)
}

// Cashfree only — a razorpay refund never levies, and creates zero rows.
func TestLevyGatewayFeePenalty_CashfreeOnly(t *testing.T) {
	gatewayFeePolicy(t, true, 2, false)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()

	require.Nil(t, levyFee(t, db, chefID, userID, uuid.New(), "razorpay", 250, "refund:order-a:full"))
	var count int64
	require.NoError(t, db.Model(&models.ChefPenalty{}).Where("chef_id = ?", chefID).Count(&count).Error)
	require.Equal(t, int64(0), count, "razorpay must not create a row")

	require.NotNil(t, levyFee(t, db, chefID, userID, uuid.New(), models.PaymentProviderCashfree, 250, "refund:order-b:full"))
}

// Disabled means no levy at all, regardless of provider/amount.
func TestLevyGatewayFeePenalty_Disabled(t *testing.T) {
	gatewayFeePolicy(t, false, 2, false)
	db := setupPenaltyDB(t)
	require.Nil(t, levyFee(t, db, uuid.New(), uuid.New(), uuid.New(), models.PaymentProviderCashfree, 250, "refund:order:full"))
}

// A wallet/loyalty-only "refund" that never touched Cashfree must not levy.
func TestLevyGatewayFeePenalty_ZeroRefundedAmount(t *testing.T) {
	gatewayFeePolicy(t, true, 2, false)
	db := setupPenaltyDB(t)
	require.Nil(t, levyFee(t, db, uuid.New(), uuid.New(), uuid.New(), models.PaymentProviderCashfree, 0, "refund:order:full"))
	require.Nil(t, levyFee(t, db, uuid.New(), uuid.New(), uuid.New(), models.PaymentProviderCashfree, -10, "refund:order:full"))
}

// One cancellation is one penalty event — an order that already carries a cancel_late levy does
// not also raise a gateway_fee levy, unless the stack flag is explicitly on.
func TestLevyGatewayFeePenalty_SkipsWhenCancelLevyExists(t *testing.T) {
	gatewayFeePolicy(t, true, 2, false)
	db := setupPenaltyDB(t)
	chefID, userID, orderID := uuid.New(), uuid.New(), uuid.New()

	require.NoError(t, db.Create(&models.ChefPenalty{
		ChefID: chefID, UserID: userID, Kind: models.ChefPenaltyCancelLate,
		Status: models.ChefPenaltyPending, SourceKey: "chefcancel:" + orderID.String(),
		OrderID: &orderID, Amount: 30, OccurredAt: time.Now().UTC(),
	}).Error)

	require.Nil(t, levyFee(t, db, chefID, userID, orderID, models.PaymentProviderCashfree, 250, "refund:"+orderID.String()+":full"),
		"a cancel_late levy already covers this cancellation")

	gatewayFeePolicy(t, true, 2, true)
	require.NotNil(t, levyFee(t, db, chefID, userID, orderID, models.PaymentProviderCashfree, 250, "refund:"+orderID.String()+":full"),
		"stacking explicitly enabled")
}

// A retry with the identical logical key levies at most once.
func TestLevyGatewayFeePenalty_Idempotent(t *testing.T) {
	gatewayFeePolicy(t, true, 2, false)
	db := setupPenaltyDB(t)
	chefID, userID, orderID := uuid.New(), uuid.New(), uuid.New()
	key := "refund:" + orderID.String() + ":full"

	first := levyFee(t, db, chefID, userID, orderID, models.PaymentProviderCashfree, 250, key)
	require.NotNil(t, first)

	second := levyFee(t, db, chefID, userID, orderID, models.PaymentProviderCashfree, 250, key)
	require.Nil(t, second, "the retry raises nothing new")

	var count int64
	require.NoError(t, db.Model(&models.ChefPenalty{}).Where("chef_id = ?", chefID).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

// Two genuinely different partial refunds on the same order each levy separately.
func TestLevyGatewayFeePenalty_DistinctKeysLevySeparately(t *testing.T) {
	gatewayFeePolicy(t, true, 2, false)
	db := setupPenaltyDB(t)
	chefID, userID, orderID := uuid.New(), uuid.New(), uuid.New()

	first := levyFee(t, db, chefID, userID, orderID, models.PaymentProviderCashfree, 100, "refund:"+orderID.String()+":0")
	require.NotNil(t, first)
	require.Equal(t, 2.0, first.Amount)

	second := levyFee(t, db, chefID, userID, orderID, models.PaymentProviderCashfree, 50, "refund:"+orderID.String()+":10000")
	require.NotNil(t, second)
	require.Equal(t, 1.0, second.Amount)

	var count int64
	require.NoError(t, db.Model(&models.ChefPenalty{}).Where("chef_id = ?", chefID).Count(&count).Error)
	require.Equal(t, int64(2), count)
}

// GraceEnabled=false, even with a nonzero GraceCount configured, gives no exemption — the very
// first gateway-fee event still levies in full (decision 4's "no grace by default").
func TestLevyGatewayFeePenalty_GraceDefaultOff(t *testing.T) {
	p := DefaultPlatformPolicy()
	p.GatewayFeeLevyEnabled = true
	p.GatewayFeeLevyPercent = 2
	p.GatewayFeeLevyGraceEnabled = false
	p.GatewayFeeLevyGraceCount = 1 // configured but inert while GraceEnabled is false
	p.GatewayFeeLevyGraceDays = 30
	withPlatformPolicy(t, p)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()

	first := levyFee(t, db, chefID, userID, uuid.New(), models.PaymentProviderCashfree, 250, "refund:a:full")
	require.NotNil(t, first, "no exemption even though a grace count happens to be configured")
	require.Equal(t, models.ChefPenaltyPending, first.Status)
	require.Equal(t, 5.0, first.Amount)
}

// With grace explicitly enabled, the first event is waived (still consuming the SourceKey) and
// the second (distinct key) levies in full — mirrors TestLevyChefCancelPenalty_GraceAllowance.
func TestLevyGatewayFeePenalty_GraceEnabled(t *testing.T) {
	p := DefaultPlatformPolicy()
	p.GatewayFeeLevyEnabled = true
	p.GatewayFeeLevyPercent = 2
	p.GatewayFeeLevyGraceEnabled = true
	p.GatewayFeeLevyGraceCount = 1
	p.GatewayFeeLevyGraceDays = 30
	withPlatformPolicy(t, p)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()

	first := levyFee(t, db, chefID, userID, uuid.New(), models.PaymentProviderCashfree, 250, "refund:a:full")
	require.Nil(t, first, "the first gateway-fee event is forgiven")

	var rows []models.ChefPenalty
	require.NoError(t, db.Where("chef_id = ?", chefID).Find(&rows).Error)
	require.Len(t, rows, 1, "the exemption is still recorded")
	require.Equal(t, models.ChefPenaltyWaived, rows[0].Status)
	require.Equal(t, 0.0, rows[0].Amount)

	second := levyFee(t, db, chefID, userID, uuid.New(), models.PaymentProviderCashfree, 250, "refund:b:full")
	require.NotNil(t, second, "the allowance is spent — the second event is charged")
	require.Equal(t, 5.0, second.Amount)
}

// statement.go needs zero changes for the new kind: a mixed chef with one cancel_late levy (on
// one order) and one gateway_fee levy (on a different order) has both summed into one deduction.
func TestApplyChefPenaltiesToStatement_MixedKinds(t *testing.T) {
	p := DefaultPlatformPolicy()
	p.ChefCancelPenaltyEnabled = true
	p.ChefCancelPenaltyPercent = 6
	p.ChefCancelPenaltyLeadHours = 4
	p.ChefCancelPenaltyGraceCount = 0
	p.ChefCancelPenaltyGraceDays = 30
	p.GatewayFeeLevyEnabled = true
	p.GatewayFeeLevyPercent = 2
	p.GatewayFeeLevyGraceEnabled = false
	withPlatformPolicy(t, p)
	db := setupPenaltyDB(t)
	chefID, userID := uuid.New(), uuid.New()
	orderA, orderB := uuid.New(), uuid.New()

	cancelLevy, err := LevyChefCancelPenalty(db, chefID, userID, orderA, "HC-A", 500, 1)
	require.NoError(t, err)
	require.NotNil(t, cancelLevy)

	feeLevy := levyFee(t, db, chefID, userID, orderB, models.PaymentProviderCashfree, 250, "refund:"+orderB.String()+":full")
	require.NotNil(t, feeLevy)

	stmt := &models.WeeklyStatement{ID: uuid.New(), ChefID: chefID, UserID: userID, NetPayout: 1000}
	require.NoError(t, db.Create(stmt).Error)

	deducted, err := ApplyChefPenaltiesToStatement(db, stmt)
	require.NoError(t, err)
	require.Equal(t, cancelLevy.Amount+feeLevy.Amount, deducted, "the sum of both kinds")
}
