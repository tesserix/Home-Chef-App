# Checkout Credits (Wallet + Loyalty) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a customer pay for an on-demand order with wallet credit and loyalty points, capped so the platform fee and GST are always settled in real money, with the server as sole authority on every figure.

**Architecture:** One pure allocation function (`PlanCheckoutCredit`) computes the split in paise. A DB-backed assembler loads the order, balances and config and calls it. A new read-only `/quote` endpoint serves the result to the checkout screen, and `create` re-runs the identical computation before charging — so the client never sends a payable. Refunds mirror the funding split pro-rata.

**Tech Stack:** Go 1.26 / Gin / GORM / PostgreSQL (tests on in-memory SQLite), React Native + Expo (mobile-customer), Razorpay Route.

## Global Constraints

- Money arithmetic in **paise (int)** inside the engine; rupees (`float64`) only at the API boundary. Follow `services/wallet_split.go`.
- `RedeemableCap = max(0, min(Subtotal − Discount + DeliveryFee, Total − ServiceFee − Tax))`. Both branches always computed; smaller wins.
- Invariant, never violated: `payable ≥ ServiceFee + Tax`, and `wallet + loyalty + payable == Total` exactly.
- Precedence is **wallet first, then loyalty**.
- `MinRedeem` (500 points) is **waived at checkout**, still enforced in `RedeemLoyalty` (redeem-to-wallet).
- Monthly cap counts **both** checkout and redeem-to-wallet redemptions against one pool.
- Credit is **INR/Razorpay only** — Stripe orders get `walletEnabled:false, loyaltyEnabled:false`.
- Refunds split **pro-rata**; the card slice absorbs the rounding remainder. Loyalty slice refunds to **wallet as rupees**. Monthly cap is **not** released on refund.
- **No SQL files in this repo.** All DDL goes to `tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/`. GORM models here must match.
- No AI/Claude/Co-Authored-By references in commits.
- Git identity: `sam123ben` / `samyak.rout@gmail.com`.
- Mobile styling per `.impeccable.md`: persimmon accent, hairlines, tabular numerals, 44px targets, `cubic-bezier(0.22, 1, 0.36, 1)`, no bounce.

---

### Task 1: Pure allocation engine

**Files:**
- Create: `apps/api/services/checkout_credit.go`
- Test: `apps/api/services/checkout_credit_test.go`

**Interfaces:**
- Consumes: `services.LoyaltyConfig` (`services/loyalty.go:32`), `services.ToPaise` / `services.FromPaise`.
- Produces: `CreditInputs`, `CreditQuote`, `func PlanCheckoutCredit(in CreditInputs) CreditQuote`. Tasks 4, 6 and 7 depend on these exact names.

- [ ] **Step 1: Write the failing tests**

```go
package services

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func cfgFor(t *testing.T) LoyaltyConfig {
	t.Helper()
	return LoyaltyConfig{Enabled: true, RedeemRate: 0.05, MinRedeem: 500,
		MaxRedeemPct: 0.10, MonthlyRedeemCap: 300, ExpiryDays: 365}
}

// The owner's rule: fees and GST are never funded by credit.
func TestPlanCheckoutCredit_NeverFundsFeesOrTax(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 258000, DeliveryFeePaise: 20000, ServiceFeePaise: 12874,
		TaxPaise: 14544, TotalPaise: 305418,
		WalletBalancePaise: 1000000, PointsBalance: 100000, Cfg: cfgFor(t),
	})
	require.Equal(t, 278000, q.RedeemableCapPaise, "food + delivery only")
	require.Equal(t, 27418, q.PayablePaise, "service fee + tax, in cash")
	require.GreaterOrEqual(t, q.PayablePaise, 12874+14544)
}

// Wallet is consumed before loyalty; on a small order the points survive.
func TestPlanCheckoutCredit_WalletFirstLeavesPointsUntouched(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 34000, DeliveryFeePaise: 0, ServiceFeePaise: 0,
		TaxPaise: 0, TotalPaise: 34000,
		WalletBalancePaise: 50000, PointsBalance: 100000, Cfg: cfgFor(t),
	})
	require.Equal(t, 34000, q.WalletAppliedPaise)
	require.Equal(t, 0, q.PointsAppliedPaise)
	require.Equal(t, float64(0), q.PointsAppliedPoints)
	require.Equal(t, 0, q.PayablePaise)
	require.Equal(t, "order_covered", q.LoyaltyLimitReason)
}

// A tip is a gratuity, not food — credit must never absorb it.
func TestPlanCheckoutCredit_TipIsNotRedeemable(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, DeliveryFeePaise: 0, ServiceFeePaise: 0,
		TaxPaise: 0, TotalPaise: 110000, // 10000 paise tip
		WalletBalancePaise: 500000, Cfg: cfgFor(t),
	})
	require.Equal(t, 100000, q.RedeemableCapPaise)
	require.Equal(t, 10000, q.PayablePaise, "the tip stays in cash")
}

// Inclusive-tax regime: Tax lives INSIDE Subtotal and is not added to Total.
// The Total−ServiceFee−Tax branch must win so GST is still paid in cash.
func TestPlanCheckoutCredit_InclusiveTaxStillPaidInCash(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, DeliveryFeePaise: 0, ServiceFeePaise: 5000,
		TaxPaise: 4762, TotalPaise: 105000, // tax embedded in subtotal
		WalletBalancePaise: 500000, Cfg: cfgFor(t),
	})
	require.Equal(t, 95238, q.RedeemableCapPaise)
	require.Equal(t, 9762, q.PayablePaise)
	require.GreaterOrEqual(t, q.PayablePaise, 5000+4762)
}

// The 10%-of-subtotal per-order cap finally binds (it never has before).
func TestPlanCheckoutCredit_PerOrderLoyaltyCapBinds(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 100000,
		WalletBalancePaise: 0, PointsBalance: 100000, Cfg: cfgFor(t),
	})
	require.Equal(t, 10000, q.PointsAppliedPaise, "10% of 1000.00")
	require.Equal(t, "per_order_cap", q.LoyaltyLimitReason)
}

func TestPlanCheckoutCredit_MonthlyCapBinds(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 1000000, TotalPaise: 1000000,
		WalletBalancePaise: 0, PointsBalance: 1000000,
		MonthlyRedeemedPaise: 28000, Cfg: cfgFor(t),
	})
	require.Equal(t, 2000, q.PointsAppliedPaise, "300.00 cap − 280.00 used")
	require.Equal(t, "monthly_cap", q.LoyaltyLimitReason)
}

// MinRedeem is a redeem-to-wallet rule; at checkout any balance is spendable.
func TestPlanCheckoutCredit_MinRedeemWaivedAtCheckout(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 100000,
		WalletBalancePaise: 0, PointsBalance: 180, Cfg: cfgFor(t),
	})
	require.Equal(t, 900, q.PointsAppliedPaise, "180 pts x 0.05")
	require.Equal(t, float64(180), q.PointsAppliedPoints)
	require.Equal(t, "balance", q.LoyaltyLimitReason)
}

// Explicit requests are honoured and clamped, never expanded.
func TestPlanCheckoutCredit_ExplicitRequestsAreClamped(t *testing.T) {
	w, p := 5000, float64(100)
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 100000,
		WalletBalancePaise: 90000, PointsBalance: 100000,
		RequestedWalletPaise: &w, RequestedPoints: &p, Cfg: cfgFor(t),
	})
	require.Equal(t, 5000, q.WalletAppliedPaise, "not expanded to the max")
	require.Equal(t, 500, q.PointsAppliedPaise, "100 pts honoured, not maxed")
}

func TestPlanCheckoutCredit_DisabledConfigSpendsNoPoints(t *testing.T) {
	c := cfgFor(t)
	c.Enabled = false
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 100000,
		WalletBalancePaise: 0, PointsBalance: 100000, Cfg: c,
	})
	require.Equal(t, 0, q.PointsAppliedPaise)
	require.False(t, q.LoyaltyEnabled)
}

// The invariants must hold for every input, including adversarial ones.
func TestPlanCheckoutCredit_InvariantsHoldUnderFuzz(t *testing.T) {
	r := rand.New(rand.NewSource(20260725))
	for i := 0; i < 20000; i++ {
		sub := r.Intn(500000)
		del := r.Intn(50000)
		svc := r.Intn(30000)
		tax := r.Intn(30000)
		tip := r.Intn(20000)
		disc := r.Intn(sub + 1000) // may exceed subtotal
		total := sub + del + svc + tax + tip - disc
		if total < 0 {
			total = 0
		}
		q := PlanCheckoutCredit(CreditInputs{
			SubtotalPaise: sub, DiscountPaise: disc, DeliveryFeePaise: del,
			ServiceFeePaise: svc, TaxPaise: tax, TotalPaise: total,
			WalletBalancePaise:   r.Intn(1000000),
			PointsBalance:        float64(r.Intn(200000)),
			MonthlyRedeemedPaise: r.Intn(40000),
			Cfg:                  cfgFor(t),
		})
		require.Equal(t, total, q.WalletAppliedPaise+q.PointsAppliedPaise+q.PayablePaise,
			"credit + payable must reconstitute the total exactly")
		require.LessOrEqual(t, q.WalletAppliedPaise+q.PointsAppliedPaise, q.RedeemableCapPaise)
		require.GreaterOrEqual(t, q.PayablePaise, 0)
		if svc+tax <= total {
			require.GreaterOrEqual(t, q.PayablePaise, svc+tax,
				"fees and tax are never funded by credit")
		}
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd apps/api && go test ./services/ -run TestPlanCheckoutCredit -v`
Expected: FAIL — `undefined: PlanCheckoutCredit`, `undefined: CreditInputs`.

- [ ] **Step 3: Write the implementation**

```go
package services

import "math"

// checkout_credit.go — the single allocation authority for wallet + loyalty
// credit applied at checkout.
//
// The owner's rule: credit may fund FOOD and DELIVERY only. The platform service
// fee and GST are always settled in real money — the platform remits GST to the
// government and the service fee is its revenue, so neither may be funded by a
// liability the platform itself issued.
//
// Pure: no DB, no gateway. Paise throughout to stay exact. The DB-backed
// assembler in checkout_credit_quote.go feeds it; handlers never reimplement any
// part of this arithmetic.

// CreditInputs is everything the allocation needs, in paise.
type CreditInputs struct {
	SubtotalPaise    int
	DiscountPaise    int
	DeliveryFeePaise int
	ServiceFeePaise  int
	TaxPaise         int
	TotalPaise       int

	WalletBalancePaise int
	PointsBalance      float64

	// nil means "auto" — apply as much as the ceilings allow. A non-nil value is
	// the customer's explicit slider/typed choice and is clamped, never expanded.
	RequestedWalletPaise *int
	RequestedPoints      *float64

	MonthlyRedeemedPaise int
	Cfg                  LoyaltyConfig

	// WalletDisabled / LoyaltyDisabled let the caller veto a rail regardless of
	// balance — feature flags, or a non-INR (Stripe) order.
	WalletDisabled  bool
	LoyaltyDisabled bool
}

// CreditQuote is the authoritative breakdown. The client renders it verbatim and
// never recomputes any field.
type CreditQuote struct {
	RedeemableCapPaise int `json:"redeemableCapPaise"`
	NonRedeemablePaise int `json:"nonRedeemablePaise"`

	WalletBalancePaise int `json:"walletBalancePaise"`
	WalletAppliedPaise int `json:"walletAppliedPaise"`
	WalletMaxPaise     int `json:"walletMaxPaise"`

	PointsBalance       float64 `json:"pointsBalance"`
	PointsAppliedPoints float64 `json:"pointsAppliedPoints"`
	PointsAppliedPaise  int     `json:"pointsAppliedPaise"`
	PointsMaxPoints     float64 `json:"pointsMaxPoints"`

	PayablePaise int `json:"payablePaise"`

	// LoyaltyLimitReason tells the UI WHY the points row is capped, so it can
	// explain the limit without reimplementing the cap logic to guess.
	// "balance" | "per_order_cap" | "monthly_cap" | "order_covered" | "disabled"
	LoyaltyLimitReason string `json:"loyaltyLimitReason"`

	WalletEnabled  bool `json:"walletEnabled"`
	LoyaltyEnabled bool `json:"loyaltyEnabled"`
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// PlanCheckoutCredit allocates wallet credit and loyalty points against an order.
//
// The ceiling takes the MINIMUM of two expressions:
//
//	food + delivery      — encodes the product rule directly
//	total − fees − tax   — a structural guarantee that payable >= fees + tax
//	                       holds however Total is composed, now or later
//
// Two correct behaviours fall out of the minimum: a tip is excluded (it is in
// Total but not in the food branch), and inclusive-tax orders — where Tax is
// embedded in Subtotal rather than added to Total — still pay GST in cash,
// because the second branch is the smaller one there.
func PlanCheckoutCredit(in CreditInputs) CreditQuote {
	foodBranch := in.SubtotalPaise - in.DiscountPaise + in.DeliveryFeePaise
	feeBranch := in.TotalPaise - in.ServiceFeePaise - in.TaxPaise
	cap := foodBranch
	if feeBranch < cap {
		cap = feeBranch
	}
	if cap > in.TotalPaise {
		cap = in.TotalPaise
	}
	if cap < 0 {
		cap = 0
	}

	walletEnabled := !in.WalletDisabled
	loyaltyEnabled := !in.LoyaltyDisabled && in.Cfg.Enabled && in.Cfg.RedeemRate > 0

	q := CreditQuote{
		RedeemableCapPaise: cap,
		NonRedeemablePaise: in.TotalPaise - cap,
		WalletBalancePaise: in.WalletBalancePaise,
		PointsBalance:      in.PointsBalance,
		WalletEnabled:      walletEnabled,
		LoyaltyEnabled:     loyaltyEnabled,
	}

	// --- Wallet first. It is a booked liability the platform has already issued;
	// loyalty points are a capped, expiring promotional currency. Burning the
	// liability first is both the owner's preference and the better balance sheet.
	if walletEnabled {
		q.WalletMaxPaise = clampInt(in.WalletBalancePaise, 0, cap)
	}
	q.WalletAppliedPaise = q.WalletMaxPaise
	if in.RequestedWalletPaise != nil {
		q.WalletAppliedPaise = clampInt(*in.RequestedWalletPaise, 0, q.WalletMaxPaise)
	}

	// --- Loyalty second, against whatever the wallet left behind.
	room := cap - q.WalletAppliedPaise
	if !loyaltyEnabled {
		q.LoyaltyLimitReason = "disabled"
	} else {
		// Four ceilings; the binding one is reported so the UI can explain itself.
		balancePaise := pointsToPaise(in.PointsBalance, in.Cfg.RedeemRate)
		perOrderPaise := int(math.Floor(float64(in.SubtotalPaise) * in.Cfg.MaxRedeemPct))
		monthlyPaise := ToPaise(in.Cfg.MonthlyRedeemCap) - in.MonthlyRedeemedPaise
		if monthlyPaise < 0 {
			monthlyPaise = 0
		}

		maxPaise, reason := balancePaise, "balance"
		if perOrderPaise < maxPaise {
			maxPaise, reason = perOrderPaise, "per_order_cap"
		}
		if monthlyPaise < maxPaise {
			maxPaise, reason = monthlyPaise, "monthly_cap"
		}
		if room < maxPaise {
			maxPaise, reason = room, "order_covered"
		}
		if maxPaise < 0 {
			maxPaise = 0
		}
		q.LoyaltyLimitReason = reason

		// Floor to whole points so the rupee figure is an exact multiple of the
		// redeem rate and the points ledger never carries a fraction.
		q.PointsMaxPoints = paiseToWholePoints(maxPaise, in.Cfg.RedeemRate)
		pts := q.PointsMaxPoints
		if in.RequestedPoints != nil {
			pts = math.Floor(*in.RequestedPoints)
			if pts < 0 {
				pts = 0
			}
			if pts > q.PointsMaxPoints {
				pts = q.PointsMaxPoints
			}
		}
		q.PointsAppliedPoints = pts
		q.PointsAppliedPaise = pointsToPaise(pts, in.Cfg.RedeemRate)
	}

	q.PayablePaise = in.TotalPaise - q.WalletAppliedPaise - q.PointsAppliedPaise
	if q.PayablePaise < 0 { // defence in depth; the clamps above already prevent it
		q.PayablePaise = 0
	}
	return q
}

// pointsToPaise converts points to paise, rounding DOWN so a redemption can never
// hand out more value than the points are worth.
func pointsToPaise(points, rate float64) int {
	if points <= 0 || rate <= 0 {
		return 0
	}
	return int(math.Floor(points * rate * 100))
}

// paiseToWholePoints is the inverse, rounding DOWN to a whole point.
func paiseToWholePoints(paise int, rate float64) float64 {
	if paise <= 0 || rate <= 0 {
		return 0
	}
	return math.Floor(float64(paise) / (rate * 100))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run TestPlanCheckoutCredit -v`
Expected: PASS, all nine tests including 20,000 fuzz iterations.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/checkout_credit.go apps/api/services/checkout_credit_test.go
git commit -m "feat(checkout): allocation engine capping credit at food + delivery

Platform fee and GST are never funded by wallet or loyalty credit. Wallet
is consumed before points. Enforces the per-order and monthly redemption
caps, which were configurable but inert."
```

---

### Task 2: Rolling monthly redemption window

**Files:**
- Create: `apps/api/services/loyalty_monthly.go`
- Test: `apps/api/services/loyalty_monthly_test.go`

**Interfaces:**
- Produces: `func MonthlyRedeemedPaise(db *gorm.DB, userID uuid.UUID) (int, error)`. Task 4 consumes it.

Counts **both** redemption routes against one pool: `redeem` (redeem-to-wallet) and the new `order_redemption` (checkout). Without this a customer draws ₹300 through each route against a ₹300 limit.

- [ ] **Step 1: Write the failing test**

```go
package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestMonthlyRedeemedPaise_CountsBothRoutesInsideWindow(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	now := time.Now()

	insert := func(src models.LoyaltyTxnSource, pts float64, at time.Time) {
		require.NoError(t, db.Create(&models.LoyaltyTransaction{
			ID: uuid.New(), UserID: uid, Type: models.LoyaltyDebit, Source: src,
			Points: pts, IdempotencyKey: uuid.New().String(), CreatedAt: at,
		}).Error)
	}
	insert(models.LoyaltySourceRedeem, 1000, now.Add(-2*24*time.Hour))          // ₹50
	insert(models.LoyaltySourceOrderRedemption, 2000, now.Add(-10*24*time.Hour)) // ₹100
	insert(models.LoyaltySourceOrderRedemption, 4000, now.Add(-40*24*time.Hour)) // outside window
	insert(models.LoyaltySourceOrder, 5000, now)                                 // an EARN, not a redemption

	got, err := MonthlyRedeemedPaise(db, uid)
	require.NoError(t, err)
	require.Equal(t, 15000, got, "50.00 + 100.00, window and source filtered")
}

func TestMonthlyRedeemedPaise_NoHistoryIsZero(t *testing.T) {
	db := setupLoyaltyDB(t)
	got, err := MonthlyRedeemedPaise(db, uuid.New())
	require.NoError(t, err)
	require.Equal(t, 0, got)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd apps/api && go test ./services/ -run TestMonthlyRedeemedPaise -v`
Expected: FAIL — `undefined: MonthlyRedeemedPaise`, `undefined: LoyaltySourceOrderRedemption`.

- [ ] **Step 3: Add the source constant**

In `apps/api/models/loyalty.go`, beside the existing sources:

```go
	// LoyaltySourceOrderRedemption is a checkout redemption — points spent
	// directly against an order rather than converted to wallet credit.
	LoyaltySourceOrderRedemption LoyaltyTxnSource = "order_redemption"
```

- [ ] **Step 4: Write the implementation**

```go
package services

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// loyalty_monthly.go — the rolling redemption window that makes
// LoyaltyConfig.MonthlyRedeemCap bind. The cap was configurable and
// admin-editable from the start but was never enforced anywhere.

// monthlyWindow is the rolling period the cap is measured over.
const monthlyWindow = 30 * 24 * time.Hour

// MonthlyRedeemedPaise sums the rupee value of points a customer has redeemed in
// the last 30 days across BOTH routes — checkout redemptions and redeem-to-wallet.
// One cap, one pool: counting them separately would let a customer take the full
// cap through each route and draw double the limit.
func MonthlyRedeemedPaise(db *gorm.DB, userID uuid.UUID) (int, error) {
	cfg := GetLoyaltyConfig(db)
	var points float64
	err := db.Model(&models.LoyaltyTransaction{}).
		Where("user_id = ? AND type = ? AND source IN ? AND created_at >= ?",
			userID, models.LoyaltyDebit,
			[]models.LoyaltyTxnSource{models.LoyaltySourceRedeem, models.LoyaltySourceOrderRedemption},
			time.Now().Add(-monthlyWindow)).
		Select("COALESCE(SUM(points), 0)").Scan(&points).Error
	if err != nil {
		return 0, err
	}
	return pointsToPaise(points, cfg.RedeemRate), nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run "TestMonthlyRedeemedPaise|TestPlanCheckoutCredit" -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/services/loyalty_monthly.go apps/api/services/loyalty_monthly_test.go apps/api/models/loyalty.go
git commit -m "feat(loyalty): rolling 30-day redemption window across both routes

Checkout redemptions and redeem-to-wallet share one monthly cap pool, so
the configured limit can't be drawn twice."
```

---

### Task 3: Order funding columns and the checkout points debit

**Files:**
- Modify: `apps/api/models/order.go` (beside `WalletApplied`, line ~155)
- Create: `apps/api/services/loyalty_order_redeem.go`
- Test: `apps/api/services/loyalty_order_redeem_test.go`
- Create (in **tesserix-k8s**): DDL appended to `charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`

**Interfaces:**
- Consumes: `consumeBatchesFIFO`, `applyLoyaltyTxnInTx` (`services/loyalty_batch.go`, `services/loyalty.go:277`).
- Produces: `func RedeemLoyaltyToOrder(tx *gorm.DB, userID, orderID uuid.UUID, points float64) error`. Task 6 consumes it.

- [ ] **Step 1: Add the model fields**

In `apps/api/models/order.go`, directly after `WalletApplied`:

```go
	// LoyaltyApplied is the rupee value of loyalty points spent on this order at
	// checkout, and LoyaltyPointsSpent the points debited to fund it. Like
	// WalletApplied these reduce the gateway capture but never the chef or driver
	// payout — the platform funds the difference.
	LoyaltyApplied     float64 `gorm:"default:0" json:"loyaltyApplied"`
	LoyaltyPointsSpent float64 `gorm:"default:0" json:"loyaltyPointsSpent"`
	// WalletRefunded / LoyaltyRefunded track how much of each funding source has
	// ALREADY been returned. Without them two successive partial refunds would each
	// compute their share of the ORIGINAL slice and together return more than was
	// ever funded.
	WalletRefunded  float64 `gorm:"default:0" json:"walletRefunded"`
	LoyaltyRefunded float64 `gorm:"default:0" json:"loyaltyRefunded"`
```

- [ ] **Step 2: Add the DDL in tesserix-k8s**

Append to `charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql` — idempotent, matching the file's existing style:

```sql
-- Checkout credits (#141 follow-up): per-order funding split so refunds can be
-- returned to wallet / loyalty / card pro-rata.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS loyalty_applied      numeric(10,2) NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS loyalty_points_spent numeric(12,2) NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS wallet_refunded      numeric(10,2) NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS loyalty_refunded     numeric(10,2) NOT NULL DEFAULT 0;
```

- [ ] **Step 3: Write the failing test**

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestRedeemLoyaltyToOrder_DebitsFIFOAndIsIdempotent(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid, oid := uuid.New(), uuid.New()
	_, err := EarnLoyalty(db, uid, 2000, models.LoyaltySourceOrder, nil, "seed", "seed-1")
	require.NoError(t, err)

	require.NoError(t, RedeemLoyaltyToOrder(db, uid, oid, 500))

	acct, err := LoyaltyBalance(db, uid)
	require.NoError(t, err)
	require.Equal(t, float64(1500), acct.Balance)

	// A retried settle must not debit twice.
	require.NoError(t, RedeemLoyaltyToOrder(db, uid, oid, 500))
	acct, err = LoyaltyBalance(db, uid)
	require.NoError(t, err)
	require.Equal(t, float64(1500), acct.Balance, "idempotent per order")
}

func TestRedeemLoyaltyToOrder_RejectsOverdraw(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := EarnLoyalty(db, uid, 100, models.LoyaltySourceOrder, nil, "seed", "seed-1")
	require.NoError(t, err)
	require.Error(t, RedeemLoyaltyToOrder(db, uid, uuid.New(), 500))
}

func TestRedeemLoyaltyToOrder_ZeroIsNoOp(t *testing.T) {
	db := setupLoyaltyDB(t)
	require.NoError(t, RedeemLoyaltyToOrder(db, uuid.New(), uuid.New(), 0))
}
```

- [ ] **Step 4: Run to verify it fails**

Run: `cd apps/api && go test ./services/ -run TestRedeemLoyaltyToOrder -v`
Expected: FAIL — `undefined: RedeemLoyaltyToOrder`.

- [ ] **Step 5: Write the implementation**

```go
package services

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// loyalty_order_redeem.go — spending points DIRECTLY on an order at checkout.
//
// Distinct from RedeemLoyalty (points → wallet credit), which keeps its 500-point
// MinRedeem floor to stop dust conversions. At checkout any balance is spendable:
// the floor would leave a customer holding 180 unusable points while we are trying
// to encourage them to burn the balance.

// RedeemLoyaltyToOrder debits `points` from the customer's FIFO earn lots to fund
// an order, writing one ledger entry keyed to the order. Idempotent per order — a
// retried payment settle debits nothing further. Returns
// ErrInsufficientLoyaltyPoints if the lots cannot cover the request.
func RedeemLoyaltyToOrder(tx *gorm.DB, userID, orderID uuid.UUID, points float64) error {
	if points <= 0 {
		return nil
	}
	cfg := GetLoyaltyConfig(tx)
	oid := orderID
	return tx.Transaction(func(t *gorm.DB) error {
		_, created, err := applyLoyaltyTxnInTx(t, userID, points, models.LoyaltyDebit,
			models.LoyaltySourceOrderRedemption, &oid, "Points applied at checkout",
			"loyalty:order-redeem:"+orderID.String(), nil, cfg)
		if err != nil {
			return err
		}
		if !created {
			return nil // already redeemed for this order
		}
		return consumeBatchesFIFO(t, userID, points)
	})
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run TestRedeemLoyaltyToOrder -v && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 7: Commit both repos**

```bash
git add apps/api/models/order.go apps/api/services/loyalty_order_redeem.go apps/api/services/loyalty_order_redeem_test.go
git commit -m "feat(loyalty): spend points directly against an order at checkout

FIFO lot debit, idempotent per order. Adds the per-order funding columns
refunds need to split wallet/loyalty/card pro-rata."
# then, in the tesserix-k8s checkout:
git add charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql
git commit -m "feat(homechef): order funding-split columns for checkout credits"
```

---

### Task 4: DB-backed quote assembler

**Files:**
- Create: `apps/api/services/checkout_credit_quote.go`
- Test: `apps/api/services/checkout_credit_quote_test.go`

**Interfaces:**
- Consumes: `PlanCheckoutCredit` (Task 1), `MonthlyRedeemedPaise` (Task 2), `WalletBalance`, `LoyaltyBalance`, `GetLoyaltyConfig`.
- Produces: `type CreditRequest`, `type CreditFlags`, and
  `func BuildCreditQuote(db *gorm.DB, order *models.Order, userID uuid.UUID, req CreditRequest, flags CreditFlags) (CreditQuote, error)`.
  Tasks 6 and 7 both call it — this is what guarantees `/quote` and `create` can never disagree.

- [ ] **Step 1: Write the failing test**

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// bothRails is the normal server state: wallet and loyalty checkout both enabled.
var bothRails = CreditFlags{WalletCheckoutEnabled: true, LoyaltyCheckoutEnabled: true}

func TestBuildCreditQuote_StripeOrdersTakeNoCredit(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := CreditWallet(db, uid, 500, models.WalletSourcePromo, nil, "seed", "w-1", nil)
	require.NoError(t, err)

	order := &models.Order{ID: uuid.New(), CustomerID: uid, Currency: "USD",
		PaymentProvider: "stripe", Subtotal: 100, Total: 118}
	q, err := BuildCreditQuote(db, order, uid, CreditRequest{UseWallet: true, UseLoyalty: true}, bothRails)
	require.NoError(t, err)
	require.False(t, q.WalletEnabled)
	require.False(t, q.LoyaltyEnabled)
	require.Equal(t, 0, q.WalletAppliedPaise)
	require.Equal(t, ToPaise(118), q.PayablePaise)
}

func TestBuildCreditQuote_OptedOutRailContributesNothing(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := CreditWallet(db, uid, 500, models.WalletSourcePromo, nil, "seed", "w-1", nil)
	require.NoError(t, err)

	order := &models.Order{ID: uuid.New(), CustomerID: uid, Currency: "INR",
		PaymentProvider: "razorpay", Subtotal: 1000, DeliveryFee: 0,
		ServiceFee: 50, Tax: 50, Total: 1100}
	q, err := BuildCreditQuote(db, order, uid, CreditRequest{UseWallet: false, UseLoyalty: false}, bothRails)
	require.NoError(t, err)
	require.Equal(t, 0, q.WalletAppliedPaise)
	require.Equal(t, ToPaise(1100), q.PayablePaise)
	require.True(t, q.WalletEnabled, "still offered — the customer simply declined it")
}

func TestBuildCreditQuote_AppliesLiveWalletBalanceCappedAtFood(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := CreditWallet(db, uid, 5000, models.WalletSourcePromo, nil, "seed", "w-1", nil)
	require.NoError(t, err)

	order := &models.Order{ID: uuid.New(), CustomerID: uid, Currency: "INR",
		PaymentProvider: "razorpay", Subtotal: 1000, DeliveryFee: 100,
		ServiceFee: 50, Tax: 50, Total: 1200}
	q, err := BuildCreditQuote(db, order, uid, CreditRequest{UseWallet: true}, bothRails)
	require.NoError(t, err)
	require.Equal(t, ToPaise(1100), q.WalletAppliedPaise, "food + delivery only")
	require.Equal(t, ToPaise(100), q.PayablePaise, "service fee + tax in cash")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd apps/api && go test ./services/ -run TestBuildCreditQuote -v`
Expected: FAIL — `undefined: BuildCreditQuote`.

- [ ] **Step 3: Write the implementation**

```go
package services

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// checkout_credit_quote.go — loads the live balances and config for an order and
// runs the pure allocator. BOTH the /quote endpoint and payment creation call
// this, which is what makes it impossible for the figure the customer sees and
// the figure they are charged to disagree.

// CreditRequest is the customer's intent, in rupees/points. Amount fields are
// optional: a nil amount with UseX true means "apply as much as allowed".
type CreditRequest struct {
	UseWallet     bool     `json:"useWallet"`
	WalletAmount  *float64 `json:"walletAmount"`
	UseLoyalty    bool     `json:"useLoyalty"`
	LoyaltyPoints *float64 `json:"loyaltyPoints"`
}

// CreditFlags carries the server feature gates into the allocation.
type CreditFlags struct {
	WalletCheckoutEnabled  bool
	LoyaltyCheckoutEnabled bool
}

// BuildCreditQuote assembles CreditInputs from live state and allocates.
//
// Wallet and loyalty are rupee instruments settled through Razorpay Route, so a
// Stripe order — a chef settling in their own currency — takes no credit at all
// and both rails report disabled rather than silently applying zero.
func BuildCreditQuote(db *gorm.DB, order *models.Order, userID uuid.UUID, req CreditRequest, flags CreditFlags) (CreditQuote, error) {
	inr := strings.EqualFold(order.Currency, "INR") || order.Currency == ""
	razorpay := !strings.EqualFold(order.PaymentProvider, "stripe")
	rupeeRail := inr && razorpay

	in := CreditInputs{
		SubtotalPaise:    ToPaise(order.Subtotal),
		DiscountPaise:    ToPaise(order.Discount),
		DeliveryFeePaise: ToPaise(order.EffectiveDeliveryFee()),
		ServiceFeePaise:  ToPaise(order.ServiceFee),
		TaxPaise:         ToPaise(order.Tax),
		TotalPaise:       ToPaise(order.Total),
		Cfg:              GetLoyaltyConfig(db),
		WalletDisabled:   !rupeeRail || !flags.WalletCheckoutEnabled,
		LoyaltyDisabled:  !rupeeRail || !flags.LoyaltyCheckoutEnabled,
	}

	if !in.WalletDisabled {
		w, err := WalletBalance(db, userID)
		if err != nil {
			return CreditQuote{}, err
		}
		if w != nil {
			in.WalletBalancePaise = ToPaise(w.Balance)
		}
	}
	if !in.LoyaltyDisabled {
		acct, err := LoyaltyBalance(db, userID)
		if err != nil {
			return CreditQuote{}, err
		}
		if acct != nil {
			in.PointsBalance = acct.Balance
		}
		spent, err := MonthlyRedeemedPaise(db, userID)
		if err != nil {
			return CreditQuote{}, err
		}
		in.MonthlyRedeemedPaise = spent
	}

	// An opted-out rail is an explicit request for zero — distinct from "disabled",
	// which hides the row entirely.
	zeroP, zeroF := 0, float64(0)
	if !req.UseWallet {
		in.RequestedWalletPaise = &zeroP
	} else if req.WalletAmount != nil {
		p := ToPaise(*req.WalletAmount)
		in.RequestedWalletPaise = &p
	}
	if !req.UseLoyalty {
		in.RequestedPoints = &zeroF
	} else if req.LoyaltyPoints != nil {
		in.RequestedPoints = req.LoyaltyPoints
	}

	return PlanCheckoutCredit(in), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run TestBuildCreditQuote -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/checkout_credit_quote.go apps/api/services/checkout_credit_quote_test.go
git commit -m "feat(checkout): assemble credit quotes from live balances

Single entry point shared by the quote endpoint and payment creation, so
the amount shown and the amount charged cannot diverge. Stripe orders
take no credit."
```

---

### Task 5: Feature flags default on

**Files:**
- Modify: `apps/api/config/config.go:107-111, 235, 365`

**Interfaces:**
- Produces: `config.AppConfig.LoyaltyCheckoutEnabled` (bool). Tasks 6 and 7 read it via `creditFlags()`.

- [ ] **Step 1: Flip the wallet default and add the loyalty flag**

At line ~235, change the wallet default from `"false"` to `"true"` and add the new flag beside it:

```go
	// Wallet + loyalty credit at checkout default ON — the owner's rule is that
	// customers should always be able to spend the credit they hold. Both remain
	// overridable per environment for an emergency kill switch.
	walletCheckout, _ := strconv.ParseBool(getEnv("WALLET_CHECKOUT_ENABLED", "true"))
	loyaltyCheckout, _ := strconv.ParseBool(getEnv("LOYALTY_CHECKOUT_ENABLED", "true"))
```

Add the struct field beside `WalletCheckoutEnabled` (line ~111) and the assignment beside line ~365:

```go
	// LoyaltyCheckoutEnabled gates spending loyalty points directly at checkout.
	LoyaltyCheckoutEnabled bool
```
```go
		LoyaltyCheckoutEnabled: loyaltyCheckout,
```

- [ ] **Step 2: Verify the build**

Run: `cd apps/api && go build ./... && go vet ./config/`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add apps/api/config/config.go
git commit -m "feat(config): default wallet and loyalty checkout credit to enabled"
```

---

### Task 6: The quote endpoint

**Files:**
- Modify: `apps/api/handlers/payment.go` (add `QuoteOrderCredit`)
- Modify: `apps/api/routes/routes.go:897-906`
- Test: `apps/api/handlers/checkout_credit_http_test.go`

**Interfaces:**
- Consumes: `services.BuildCreditQuote`, `services.CreditRequest`, `services.CreditFlags`.
- Produces: `POST /v1/payments/order/:orderId/quote`.

- [ ] **Step 1: Write the failing test**

```go
package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// A customer may only quote their OWN order.
func TestQuoteOrderCredit_RejectsAnotherCustomersOrder(t *testing.T) {
	orderID := seedPaidOrderForQuote(t, uuid.New())
	w := callPay(uuid.New(), http.MethodPost, "/payments/order/"+orderID.String()+"/quote",
		regQuote, map[string]any{"useWallet": true})
	require.Equal(t, http.StatusNotFound, w.Code)
}

// The quote is read-only: calling it twice must not move the wallet balance.
func TestQuoteOrderCredit_DebitsNothing(t *testing.T) {
	cust := uuid.New()
	orderID := seedPaidOrderForQuote(t, cust)
	for i := 0; i < 2; i++ {
		w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/quote",
			regQuote, map[string]any{"useWallet": true, "useLoyalty": true})
		require.Equal(t, http.StatusOK, w.Code)
	}
	require.Equal(t, 500.0, walletBalanceOf(t, cust), "quote must not debit")
}
```

Write `seedPaidOrderForQuote`, `regQuote` and `walletBalanceOf` following the existing helper style in `apps/api/handlers/payment_test.go:184-197`.

- [ ] **Step 2: Run to verify it fails**

Run: `cd apps/api && go test ./handlers/ -run TestQuoteOrderCredit -v`
Expected: FAIL — route not registered.

- [ ] **Step 3: Add the handler**

```go
// QuoteOrderCredit returns the authoritative wallet + loyalty breakdown for an
// order without moving any money. The checkout screen renders this verbatim and
// performs no arithmetic of its own — CreateOrderPayment re-runs the identical
// computation, so what is shown and what is charged cannot diverge.
// POST /payments/order/:orderId/quote
func (h *PaymentHandler) QuoteOrderCredit(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}
	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	var req services.CreditRequest
	_ = c.ShouldBindJSON(&req) // absent body = no credit requested

	quote, err := services.BuildCreditQuote(database.DB, &order, userID, req, creditFlags())
	if err != nil {
		log.Printf("credit-quote failed order=%s: %v", order.OrderNumber, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not price this order"})
		return
	}
	c.JSON(http.StatusOK, creditQuoteResponse(quote))
}

// creditFlags snapshots the server feature gates for an allocation.
func creditFlags() services.CreditFlags {
	return services.CreditFlags{
		WalletCheckoutEnabled:  config.AppConfig.WalletCheckoutEnabled,
		LoyaltyCheckoutEnabled: config.AppConfig.LoyaltyCheckoutEnabled,
	}
}

// creditQuoteResponse renders a quote in rupees for the client. Paise stay
// server-side; the client only ever displays these figures.
func creditQuoteResponse(q services.CreditQuote) gin.H {
	return gin.H{
		"redeemableCap":  services.FromPaise(q.RedeemableCapPaise),
		"nonRedeemable":  services.FromPaise(q.NonRedeemablePaise),
		"walletBalance":  services.FromPaise(q.WalletBalancePaise),
		"walletApplied":  services.FromPaise(q.WalletAppliedPaise),
		"walletMax":      services.FromPaise(q.WalletMaxPaise),
		"pointsBalance":  q.PointsBalance,
		"pointsApplied":  q.PointsAppliedPoints,
		"pointsValue":    services.FromPaise(q.PointsAppliedPaise),
		"pointsMax":      q.PointsMaxPoints,
		"payable":        services.FromPaise(q.PayablePaise),
		"loyaltyLimit":   q.LoyaltyLimitReason,
		"walletEnabled":  q.WalletEnabled,
		"loyaltyEnabled": q.LoyaltyEnabled,
	}
}
```

- [ ] **Step 4: Register the route with its own rate limit**

In `routes/routes.go`, the `/payments` group carries `RateLimitByUser(2, 5)` — correct for mint/verify/refund but it would throttle a customer dragging a slider. Register `/quote` in a sibling group with a generous limit:

```go
		// Credit quoting is read-only and idempotent, and the checkout sliders call
		// it as the customer drags. It must NOT sit under the 2 rps payment limiter
		// that guards the money-moving endpoints.
		quoteLimit := middleware.RateLimitByUser(20, 40)
		orderQuotes := v1.Group("/payments")
		orderQuotes.Use(bffAuth(bffKey, bffWindow), quoteLimit)
		{
			orderQuotes.POST("/order/:orderId/quote", paymentHandler.QuoteOrderCredit)
		}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd apps/api && go test ./handlers/ -run TestQuoteOrderCredit -v && go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/handlers/payment.go apps/api/routes/routes.go apps/api/handlers/checkout_credit_http_test.go
git commit -m "feat(checkout): read-only credit quote endpoint

Serves the authoritative wallet/loyalty breakdown to the checkout screen.
Rate-limited separately from the money-moving payment routes so slider
drags aren't throttled."
```

---

### Task 7: Server-authoritative payment creation

**Files:**
- Modify: `apps/api/handlers/payment.go:64-105` (request parsing), `:107-186` (`createRazorpayPayment`), `:356-384` (`settleOrderWallet`)
- Test: `apps/api/handlers/checkout_credit_create_test.go`

**Interfaces:**
- Consumes: `services.BuildCreditQuote`, `services.RedeemLoyaltyToOrder`, `services.PlanWalletFunding`.

This is the task that fixes the reported "₹0 outstanding still opened Razorpay" bug. The regression test must fail against current `main` first.

- [ ] **Step 1: Write the failing regression test**

```go
package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// THE BUG: the client used to compute the payable and post a rupee amount. When
// its view of the balance or total drifted from the server's, the screen showed
// "To pay 0" while the server still minted a gateway charge. The server's figure
// is now the only one that exists.
func TestCreateOrderPayment_ClientAmountNeverOverridesServer(t *testing.T) {
	cust := uuid.New()
	// Wallet holds 100.00; the client wrongly believes it holds 1200.00.
	orderID := seedUnpaidOrder(t, cust, orderMoney{Subtotal: 1000, DeliveryFee: 100,
		ServiceFee: 50, Tax: 50, Total: 1200}, walletBalance(100))

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create",
		regCreate, map[string]any{"useWallet": true, "walletAmount": 1200.0})
	require.Equal(t, http.StatusOK, w.Code)

	body := decodeJSON(t, w)
	require.Equal(t, 100.0, body["walletApplied"], "clamped to the live balance")
	require.Equal(t, 110000.0, body["amount"], "1100.00 in paise still to capture")
}

// Credit can no longer reach the fees, so a fully-credit-covered order is not a
// thing on a fee-bearing order — the capture always covers fees + tax.
func TestCreateOrderPayment_CaptureAlwaysCoversFeesAndTax(t *testing.T) {
	cust := uuid.New()
	orderID := seedUnpaidOrder(t, cust, orderMoney{Subtotal: 1000, DeliveryFee: 100,
		ServiceFee: 50, Tax: 50, Total: 1200}, walletBalance(100000))

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create",
		regCreate, map[string]any{"useWallet": true})
	require.Equal(t, http.StatusOK, w.Code)

	body := decodeJSON(t, w)
	require.Equal(t, 1100.0, body["walletApplied"])
	require.Equal(t, 10000.0, body["amount"], "100.00 of fees + tax, in paise")
}

// A legacy build posts a bare walletAmount and no useWallet flag.
func TestCreateOrderPayment_LegacyWalletAmountBodyStillWorks(t *testing.T) {
	cust := uuid.New()
	orderID := seedUnpaidOrder(t, cust, orderMoney{Subtotal: 1000, DeliveryFee: 0,
		ServiceFee: 50, Tax: 50, Total: 1100}, walletBalance(500))

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create",
		regCreate, map[string]any{"walletAmount": 500.0})
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 500.0, decodeJSON(t, w)["walletApplied"])
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd apps/api && go test ./handlers/ -run TestCreateOrderPayment_ -v`
Expected: FAIL — the first test currently returns `walletApplied: 1200` or a wrong capture.

- [ ] **Step 3: Replace the request parsing (payment.go:64-76)**

```go
	// Wallet + loyalty credit to apply. The client sends INTENT (which rails, and
	// optionally how much); the server recomputes the allocation from live state and
	// its answer is the only one that counts. An older build posting a bare
	// {"walletAmount": N} is read as an explicit wallet request.
	var req services.CreditRequest
	_ = c.ShouldBindJSON(&req)
	if !req.UseWallet && req.WalletAmount != nil && *req.WalletAmount > 0 {
		req.UseWallet = true
	}
```

- [ ] **Step 4: Replace the allocation in `createRazorpayPayment`**

Change the signature to take `req services.CreditRequest`, and replace the balance/clamp block (lines ~133-143) with:

```go
	quote, err := services.BuildCreditQuote(database.DB, order, userID, req, creditFlags())
	if err != nil {
		log.Printf("credit-quote failed order=%s: %v", order.OrderNumber, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not price this order"})
		return
	}
	walletApplied := services.FromPaise(quote.WalletAppliedPaise)
	loyaltyApplied := services.FromPaise(quote.PointsAppliedPaise)

	// Loyalty behaves exactly like wallet at the gateway: a platform-funded discount
	// that shrinks the capture but never the chef's or driver's payout. Route tops up
	// the shortfall from the platform balance.
	creditPaise := quote.WalletAppliedPaise + quote.PointsAppliedPaise
	plan := services.PlanWalletFunding(totalPaise, creditPaise, creditPaise, settlements)
```

Stamp both figures in the existing `Updates` call and return them:

```go
	database.DB.Model(order).Updates(map[string]interface{}{
		"razorpay_order_id":    rzOrder.ID,
		"payment_provider":     "razorpay",
		"wallet_applied":       walletApplied,
		"loyalty_applied":      loyaltyApplied,
		"loyalty_points_spent": quote.PointsAppliedPoints,
	})
```

Add `"loyaltyApplied": loyaltyApplied` and `"payable": services.FromPaise(quote.PayablePaise)` to the JSON response beside `walletApplied`.

- [ ] **Step 5: Debit the points on settle (`settleOrderWallet`, ~line 365)**

The points debit must happen where the wallet debit happens — after capture, idempotently — so an abandoned checkout never burns points:

```go
	// Points are debited on the SAME seam as the wallet: after the capture, keyed to
	// the order, so an abandoned or failed checkout never burns a customer's points.
	if order.LoyaltyPointsSpent > 0 {
		if err := services.RedeemLoyaltyToOrder(database.DB, order.CustomerID, order.ID, order.LoyaltyPointsSpent); err != nil {
			log.Printf("loyalty-debit failed order=%s: %v", order.OrderNumber, err)
		}
	}
```

Also update the `plan` reconstruction a few lines below to use `wallet + loyalty`:

```go
	appliedPaise := services.ToPaise(order.WalletApplied) + services.ToPaise(order.LoyaltyApplied)
```

- [ ] **Step 6: Run the full suite**

Run: `cd apps/api && go test ./handlers/ ./services/ && go build ./... && go vet ./...`
Expected: PASS, clean.

- [ ] **Step 7: Commit**

```bash
git add apps/api/handlers/payment.go apps/api/handlers/checkout_credit_create_test.go
git commit -m "fix(checkout): server owns the payable, client sends intent only

The client used to post a computed rupee amount; when its view of the
balance or total drifted the screen showed a payable the server never
charged. It now sends which rails to use and the server recomputes.
Loyalty joins wallet as a platform-funded discount."
```

---

### Task 8: Pro-rata refund allocation

**Files:**
- Create: `apps/api/services/refund_funding_split.go`
- Test: `apps/api/services/refund_funding_split_test.go`

**Interfaces:**
- Produces: `type FundingSplit`, `func SplitRefundByFunding(order *models.Order, refundPaise int) FundingSplit`. Task 9 consumes it.

- [ ] **Step 1: Write the failing test**

```go
package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// The owner's example: wallet 400 + loyalty 100 + card 500, refund 50%.
func TestSplitRefundByFunding_ProRata(t *testing.T) {
	o := &models.Order{Total: 1000, WalletApplied: 400, LoyaltyApplied: 100}
	s := SplitRefundByFunding(o, ToPaise(500))
	require.Equal(t, ToPaise(200), s.WalletPaise)
	require.Equal(t, ToPaise(50), s.LoyaltyPaise)
	require.Equal(t, ToPaise(250), s.CardPaise)
}

// The parts must reconstitute the refund exactly — no paise created or lost.
func TestSplitRefundByFunding_CardAbsorbsRounding(t *testing.T) {
	o := &models.Order{Total: 300.03, WalletApplied: 100.01, LoyaltyApplied: 100.01}
	r := ToPaise(100.01)
	s := SplitRefundByFunding(o, r)
	require.Equal(t, r, s.WalletPaise+s.LoyaltyPaise+s.CardPaise)
}

// Two successive partials must never return more than a source ever funded.
func TestSplitRefundByFunding_RepeatedPartialsCannotOverReturn(t *testing.T) {
	o := &models.Order{Total: 1000, WalletApplied: 400, LoyaltyApplied: 100,
		WalletRefunded: 360, LoyaltyRefunded: 90}
	s := SplitRefundByFunding(o, ToPaise(500))
	require.Equal(t, ToPaise(40), s.WalletPaise, "only 40.00 of wallet remains")
	require.Equal(t, ToPaise(10), s.LoyaltyPaise)
	require.Equal(t, ToPaise(450), s.CardPaise, "the rest goes to the card")
}

func TestSplitRefundByFunding_NoCreditOrderIsAllCard(t *testing.T) {
	o := &models.Order{Total: 1000}
	s := SplitRefundByFunding(o, ToPaise(1000))
	require.Equal(t, 0, s.WalletPaise)
	require.Equal(t, 0, s.LoyaltyPaise)
	require.Equal(t, ToPaise(1000), s.CardPaise)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd apps/api && go test ./services/ -run TestSplitRefundByFunding -v`
Expected: FAIL — `undefined: SplitRefundByFunding`.

- [ ] **Step 3: Write the implementation**

```go
package services

import (
	"github.com/homechef/api/models"
)

// refund_funding_split.go — dividing a refund across the rails that funded the
// order. Every source is returned its exact proportion of whatever is refunded.
//
// This matters because the on-demand refundable base is the WHOLE total (see
// RemainingRefundable), and the cancellation policy then applies a percentage —
// so a refund is routinely smaller than the credit applied, and the split rule is
// load-bearing rather than cosmetic.

// FundingSplit is one refund divided across the rails that paid for the order.
// The three parts always sum to exactly the requested refund.
type FundingSplit struct {
	WalletPaise  int // → wallet, automatically, instant
	LoyaltyPaise int // → wallet as rupees (owner decision), NOT restored as points
	CardPaise    int // → the existing customer choice: wallet or original method
}

// SplitRefundByFunding divides refundPaise pro-rata across wallet, loyalty and
// card. Each credit rail is capped at what it has not already been refunded, so
// repeated partial refunds can never return more than a source ever funded.
//
// The CARD slice takes the remainder rather than being rounded independently,
// which guarantees the parts sum to exactly refundPaise — no half-paise is
// created or destroyed by three separate roundings.
func SplitRefundByFunding(order *models.Order, refundPaise int) FundingSplit {
	if refundPaise <= 0 {
		return FundingSplit{}
	}
	totalPaise := ToPaise(order.Total)
	if totalPaise <= 0 {
		return FundingSplit{CardPaise: refundPaise}
	}

	share := func(funded, alreadyReturned float64) int {
		v := refundPaise * ToPaise(funded) / totalPaise
		remaining := ToPaise(funded) - ToPaise(alreadyReturned)
		if remaining < 0 {
			remaining = 0
		}
		if v > remaining {
			v = remaining
		}
		if v < 0 {
			v = 0
		}
		return v
	}

	s := FundingSplit{
		WalletPaise:  share(order.WalletApplied, order.WalletRefunded),
		LoyaltyPaise: share(order.LoyaltyApplied, order.LoyaltyRefunded),
	}
	s.CardPaise = refundPaise - s.WalletPaise - s.LoyaltyPaise
	if s.CardPaise < 0 {
		s.CardPaise = 0
	}
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run TestSplitRefundByFunding -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/refund_funding_split.go apps/api/services/refund_funding_split_test.go
git commit -m "feat(refunds): pro-rata split across wallet, loyalty and card

Each rail is returned its exact share and capped at what it hasn't already
been refunded, so repeated partials can't over-return. The card slice
absorbs the rounding remainder so the parts sum exactly."
```

---

### Task 9: Wire the split into the refund money mover

**Files:**
- Modify: `apps/api/services/cancellation_order_refund.go` (`RefundOrderForCancellation`)
- Test: `apps/api/services/cancellation_order_refund_credit_test.go`

**Interfaces:**
- Consumes: `SplitRefundByFunding` (Task 8), `CreditWallet`.

- [ ] **Step 1: Write the failing test**

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// The credit slices return to the wallet automatically; only the card slice
// reaches the gateway.
func TestRefundOrderForCancellation_ReturnsCreditToWallet(t *testing.T) {
	db, order, cust := seedCreditFundedOrder(t, 1000, 400, 100) // total, wallet, loyalty

	require.NoError(t, RefundOrderForCancellation(db, order))

	w, err := WalletBalance(db, cust)
	require.NoError(t, err)
	require.Equal(t, 500.0, w.Balance, "wallet 400 + loyalty 100, both to wallet")

	var got models.Order
	require.NoError(t, db.First(&got, "id = ?", order.ID).Error)
	require.Equal(t, 400.0, got.WalletRefunded)
	require.Equal(t, 100.0, got.LoyaltyRefunded)
}

// A second call must be a no-op — the refund claim already guards this, and the
// wallet credit is keyed per order so it cannot double-credit either.
func TestRefundOrderForCancellation_CreditReturnIsIdempotent(t *testing.T) {
	db, order, cust := seedCreditFundedOrder(t, 1000, 400, 100)
	require.NoError(t, RefundOrderForCancellation(db, order))
	_ = RefundOrderForCancellation(db, order)

	w, err := WalletBalance(db, cust)
	require.NoError(t, err)
	require.Equal(t, 500.0, w.Balance, "no double credit")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd apps/api && go test ./services/ -run TestRefundOrderForCancellation_ -v`
Expected: FAIL — credit slices are not returned.

- [ ] **Step 3: Add the credit return inside the refund claim**

After the atomic claim succeeds and before the gateway call, return the credit rails and reduce the gateway refund to the card slice only:

```go
	// Return the credit-funded slices before touching the gateway. Both go to the
	// wallet: the wallet slice back where it came from, and — by owner decision —
	// the loyalty slice as rupees rather than restored points.
	//
	// The monthly redemption cap is deliberately NOT released. Holding it bounds a
	// redeem-then-cancel loop to MonthlyRedeemCap per customer per 30 days, so the
	// loop cannot pump capped, expiring points into uncapped permanent credit.
	split := SplitRefundByFunding(order, ToPaise(amount))
	if split.WalletPaise > 0 {
		if _, err := CreditWallet(db, order.CustomerID, FromPaise(split.WalletPaise),
			models.WalletSourceRefund, &order.ID, "Order refund — wallet portion",
			"refund-wallet:"+order.ID.String(), nil); err != nil {
			log.Printf("refund: wallet slice failed order=%s: %v", order.OrderNumber, err)
		}
	}
	if split.LoyaltyPaise > 0 {
		if _, err := CreditWallet(db, order.CustomerID, FromPaise(split.LoyaltyPaise),
			models.WalletSourceLoyalty, &order.ID, "Order refund — loyalty portion",
			"refund-loyalty:"+order.ID.String(), nil); err != nil {
			log.Printf("refund: loyalty slice failed order=%s: %v", order.OrderNumber, err)
		}
	}
	db.Model(order).Updates(map[string]interface{}{
		"wallet_refunded":  order.WalletRefunded + FromPaise(split.WalletPaise),
		"loyalty_refunded": order.LoyaltyRefunded + FromPaise(split.LoyaltyPaise),
	})
```

Then send only `split.CardPaise` to the gateway instead of the full amount, skipping the gateway call entirely when it is zero.

- [ ] **Step 4: Run the full suite**

Run: `cd apps/api && go test ./... && go build ./... && go vet ./...`
Expected: PASS. Existing refund tests must still pass — an order with no credit splits entirely to the card and behaves exactly as before.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/cancellation_order_refund.go apps/api/services/cancellation_order_refund_credit_test.go
git commit -m "feat(refunds): return credit slices to wallet, card slice to gateway

Wallet and loyalty portions credit instantly; only the card portion reaches
Razorpay. Monthly redemption cap is held on refund so a cancel loop can't
convert capped points into uncapped credit."
```

---

### Task 10: Checkout credits card (mobile)

**Files:**
- Create: `apps/mobile-customer/hooks/useCreditQuote.ts`
- Create: `apps/mobile-customer/components/checkout/CreditsCard.tsx`
- Modify: `apps/mobile-customer/app/checkout.tsx:90-92, 200, 556-561, 1329-1372`
- Modify: `apps/mobile-customer/lib/payment.ts:55-70`

**Interfaces:**
- Consumes: `POST /v1/payments/order/:id/quote`.
- Produces: `useCreditQuote(orderId, intent)`, `<CreditsCard />`.

- [ ] **Step 1: Add the quote hook**

```ts
// useCreditQuote.ts — the checkout screen's ONLY source of money figures.
// Every amount rendered in the credits card and the "To pay" row comes from
// here; the screen performs no arithmetic of its own, which is what stops the
// UI and the charge from ever disagreeing.
import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

export interface CreditIntent {
  useWallet: boolean;
  walletAmount?: number;
  useLoyalty: boolean;
  loyaltyPoints?: number;
}

export interface CreditQuote {
  redeemableCap: number;
  nonRedeemable: number;
  walletBalance: number;
  walletApplied: number;
  walletMax: number;
  pointsBalance: number;
  pointsApplied: number;
  pointsValue: number;
  pointsMax: number;
  payable: number;
  loyaltyLimit: 'balance' | 'per_order_cap' | 'monthly_cap' | 'order_covered' | 'disabled';
  walletEnabled: boolean;
  loyaltyEnabled: boolean;
}

export function useCreditQuote(orderId: string | null, intent: CreditIntent) {
  return useQuery({
    queryKey: ['credit-quote', orderId, intent],
    enabled: !!orderId,
    staleTime: 0,
    queryFn: async (): Promise<CreditQuote> => {
      const resp = await api.post<{ data?: CreditQuote }>(
        `/v1/payments/order/${orderId}/quote`,
        intent,
      );
      return (resp.data.data ?? resp.data) as CreditQuote;
    },
  });
}
```

- [ ] **Step 2: Build the credits card**

`CreditsCard.tsx` renders two rows — wallet then loyalty — each with a toggle, a slider (`@react-native-community/slider`, already a dependency of the delivery app; verify with `pnpm ls` and add via the hoisted root `node_modules` if absent, per the workspace lockfile constraint) and a tap-to-type numeric field. Requirements:

- Both toggles default **on**; the customer's choice persists via the existing Zustand cart store.
- Slider bounds are `0 → walletMax` and `0 → pointsMax` from the quote, never the raw balance.
- Until the customer touches either control both stay auto (no `walletAmount` / `loyaltyPoints` sent). On first touch, both become explicit and send their current values — so dragging wallet down does not silently expand loyalty into the gap.
- Slider input is debounced 250ms before re-quoting; the last good quote renders while a request is in flight.
- The loyalty row's caption is driven by `loyaltyLimit`: `per_order_cap` → "up to ₹X on this order", `monthly_cap` → "₹X left this month", `order_covered` → "your wallet covers this order", `balance` → "worth ₹X".
- Footer line: `Fees & taxes are always paid separately (₹{nonRedeemable})`.
- Per `.impeccable.md`: persimmon only on the active slider track and applied amounts, hairline separators, tabular numerals, 44px targets, no bounce.

- [ ] **Step 3: Rewire the checkout screen**

- Delete `WALLET_CHECKOUT_ENABLED` (line 92) and its use at line 559 — availability now comes from `quote.walletEnabled` / `quote.loyaltyEnabled`.
- Delete the local `walletApplied` / `payable` arithmetic (lines 556-561); read `quote.payable` instead.
- Replace the checkbox block (lines 1329-1372) with `<CreditsCard />`, moved **above** Price Details.
- The Place Order button label reads `quote.payable`. If the quote request fails, keep the last good quote and disable Place Order rather than guessing a payable.

- [ ] **Step 4: Send intent, not amounts, from `payment.ts`**

```ts
export async function startOrderPayment(
  orderId: string,
  intent: { useWallet: boolean; walletAmount?: number; useLoyalty: boolean; loyaltyPoints?: number },
): Promise<void> {
  const resp = await api.post<{ data: RazorpayPaymentData }>(
    `/v1/payments/order/${orderId}/create`,
    intent,
  );
```

- [ ] **Step 5: Typecheck and lint**

Run: `cd apps/mobile-customer && npx tsc --noEmit && npx eslint app/checkout.tsx components/checkout/CreditsCard.tsx hooks/useCreditQuote.ts`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add apps/mobile-customer/hooks/useCreditQuote.ts apps/mobile-customer/components/checkout/CreditsCard.tsx apps/mobile-customer/app/checkout.tsx apps/mobile-customer/lib/payment.ts
git commit -m "feat(checkout): prominent wallet + loyalty credits card

Replaces the checkbox buried under the promo field with a titled card
above Price Details, sliders and typed input on both rails, defaults on
and sized to the order. All figures come from the server quote."
```

---

### Task 11: Emulator validation against prod

**Files:** none — verification only.

- [ ] **Step 1: Deploy the API**

Report the built image tag and let the user run the deploy (never build/push/deploy directly). Confirm the tesserix-k8s DDL from Task 3 has been applied by the bootstrap CronJob **before** the API rolls, or `loyalty_applied` writes will fail.

- [ ] **Step 2: Verify the cap on a real order**

Place an on-demand order on the Android emulator with wallet and loyalty balances present. Confirm:
- the credits card is visible above Price Details, both rows on by default;
- wallet shows `min(balance, food + delivery)`, not the raw balance;
- the loyalty row is ₹0 when the wallet already covers the order;
- "To pay" equals platform fee + CGST + SGST exactly;
- the Razorpay sheet opens for that exact figure — never ₹0, never the full total.

- [ ] **Step 3: Verify the sliders**

Drag wallet down; confirm loyalty does **not** silently expand, "To pay" rises by the same amount, and the typed input agrees with the slider.

- [ ] **Step 4: Verify the pro-rata refund**

Cancel the order. Confirm the wallet and loyalty slices land in the wallet instantly, the card slice enters the existing wallet-or-original choice, and `wallet_refunded` / `loyalty_refunded` are stamped. Re-check the points balance: spent points are **not** restored — their value returns as wallet credit.

- [ ] **Step 5: Verify the caps bind**

With a large points balance, confirm the loyalty row caps at 10% of the food subtotal and reports the right reason, and that a second order in the same month reflects the reduced ₹300 monthly remainder.
