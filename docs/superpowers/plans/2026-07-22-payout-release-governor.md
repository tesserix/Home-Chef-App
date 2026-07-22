# Payout Release Governor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Release each order's already-held Razorpay Route transfers 2 hours after delivery, subject to a guardrail chain and a per-chef automation switch, so chefs are paid within two days.

**Architecture:** Transfers are already created held and correctly attributed at checkout — that path is untouched. A pure decision function (`payouts.DecideRelease`) evaluates one order against every guardrail and returns all block reasons at once. A Temporal-scheduled sweep runs every 15 minutes, applies that decision to matured delivered orders, and calls the existing `ReleaseOrderPayouts`. Blocked orders surface in the existing admin payout queue.

**Tech Stack:** Go 1.26, Gin, GORM, PostgreSQL 16, Temporal Schedules, NATS JetStream (via the existing transactional outbox), Razorpay Route.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-07-22-payout-release-governor-design.md`. Read it before starting.
- **Never create SQL/migration files in this repo.** All schema lives in `tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/`. This repo holds GORM models only.
- **Never run container builds, image pushes or deploys.** Verify with `go build ./...`, `go vet ./...`, `go test ./...` only.
- Git identity: `git config user.name "sam123ben"` and `git config user.email "samyak.rout@gmail.com"`.
- No AI/Claude/Copilot/`Co-Authored-By` references in commits, PRs, comments or any file content.
- `payouts` package must stay domain-free — no imports from `github.com/homechef/api/...`. `payouts/boundary_test.go` enforces this and will fail the build if violated.
- Money is integer minor units (paise) via `payouts.Money`. Never use float for money.
- Reason codes are persisted and grouped in analytics — treat their string values as stable API.
- All thresholds are `PlatformSettings` keys, read at runtime. No recompile to change a limit.
- Events go through `EnqueueOutbox` (`services/outbox.go:42`), never a direct NATS publish. The relay delivers to **JetStream** with PubAck and `Nats-Msg-Id` dedup. Both new subjects sit under the existing `PAYMENTS` stream (`payments.>`, `services/nats.go`), so no stream definition changes.

---

### Task 1: Release decision governor

Pure, domain-free decision function. No database, no gateway, no clock of its own — so a decision is reproducible from recorded inputs, which is what makes the admin queue's "why was this blocked" answerable months later.

**Files:**
- Create: `apps/api/payouts/governor.go`
- Test: `apps/api/payouts/governor_test.go`

**Interfaces:**
- Consumes: `Money`, `Zero`, `CurrencyINR` from `payouts/money.go`; `Balance` from `payouts/ledger.go`.
- Produces: `payouts.DecideRelease(ReleaseInput) ReleaseDecision`, `payouts.BlockReason` (string type) with constants `BlockNotMatured`, `BlockAutomationOff`, `BlockSettlementNotActivated`, `BlockRefundOpen`, `BlockRecoveryBalance`, `BlockNewChefRamp`, `BlockAboveReviewThreshold`; `BlockReason.Overridable() bool`; `ReleaseDecision.Blocked() bool`.

- [ ] **Step 1: Write the failing test**

Create `apps/api/payouts/governor_test.go`:

```go
package payouts

import (
	"testing"
	"time"
)

// #741 — the per-order release decision. Pure: every input is passed in so the
// decision recorded on a blocked order can be re-derived later.

var govNow = time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)

func clean() ReleaseInput {
	return ReleaseInput{
		Now:                 govNow,
		DeliveredAt:         govNow.Add(-3 * time.Hour),
		Maturation:          2 * time.Hour,
		AutomationEnabled:   true,
		SettlementActivated: true,
		RefundOpen:          false,
		RecoveryBalance:     Zero(CurrencyINR),
		DeliveredOrderCount: 10,
		RampOrders:          3,
		OrderTotal:          Money{Minor: 50_000, Currency: CurrencyINR},
		ReviewAbove:         Money{Minor: 500_000, Currency: CurrencyINR},
	}
}

func hasReason(d ReleaseDecision, r BlockReason) bool {
	for _, got := range d.Reasons {
		if got == r {
			return true
		}
	}
	return false
}

func TestDecideRelease_ReleasesACleanMaturedOrder(t *testing.T) {
	d := DecideRelease(clean())
	if !d.Release {
		t.Fatalf("clean order must release, blocked by %v", d.Reasons)
	}
	if len(d.Reasons) != 0 {
		t.Fatalf("reasons = %v, want none", d.Reasons)
	}
}

func TestDecideRelease_HoldsUntilMaturation(t *testing.T) {
	in := clean()
	in.DeliveredAt = govNow.Add(-time.Hour) // only 1h of a 2h window
	d := DecideRelease(in)
	if d.Release {
		t.Fatal("an order inside the maturation window must not release")
	}
	if !hasReason(d, BlockNotMatured) {
		t.Fatalf("reasons = %v, want %q", d.Reasons, BlockNotMatured)
	}
}

func TestDecideRelease_MaturationBoundaryIsInclusive(t *testing.T) {
	// Off-by-one here silently delays every payout by a full sweep cycle.
	in := clean()
	in.DeliveredAt = govNow.Add(-2 * time.Hour) // exactly matured
	if !DecideRelease(in).Release {
		t.Fatal("an order maturing exactly now is releasable")
	}
}

func TestDecideRelease_BlocksOnEachGuardrail(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ReleaseInput)
		want   BlockReason
	}{
		{"automation off", func(in *ReleaseInput) { in.AutomationEnabled = false }, BlockAutomationOff},
		{"settlement not activated", func(in *ReleaseInput) { in.SettlementActivated = false }, BlockSettlementNotActivated},
		{"refund open", func(in *ReleaseInput) { in.RefundOpen = true }, BlockRefundOpen},
		{"recovery balance", func(in *ReleaseInput) {
			in.RecoveryBalance = Money{Minor: 25_000, Currency: CurrencyINR}
		}, BlockRecoveryBalance},
		{"new chef ramp", func(in *ReleaseInput) { in.DeliveredOrderCount = 1 }, BlockNewChefRamp},
		{"above review threshold", func(in *ReleaseInput) {
			in.OrderTotal = Money{Minor: 900_000, Currency: CurrencyINR}
		}, BlockAboveReviewThreshold},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := clean()
			tc.mutate(&in)
			d := DecideRelease(in)
			if d.Release {
				t.Fatal("must not release")
			}
			if !hasReason(d, tc.want) {
				t.Fatalf("reasons = %v, want %q", d.Reasons, tc.want)
			}
		})
	}
}

func TestDecideRelease_ReportsEveryReasonAtOnce(t *testing.T) {
	// Short-circuiting makes an admin fix one problem, retry, and immediately
	// hit the next. They must see the whole picture in one pass.
	in := clean()
	in.SettlementActivated = false
	in.RefundOpen = true
	in.DeliveredOrderCount = 0

	d := DecideRelease(in)
	if len(d.Reasons) < 3 {
		t.Fatalf("reasons = %v, want all three reported together", d.Reasons)
	}
}

func TestDecideRelease_MarksHardBlocksAsNotOverridable(t *testing.T) {
	// An admin forcing a release to an unactivated account strands the money
	// rather than delivering it; paying against an open refund may settle
	// before the claw-back lands.
	if BlockSettlementNotActivated.Overridable() {
		t.Fatal("settlement activation is not an admin decision")
	}
	if BlockRefundOpen.Overridable() {
		t.Fatal("an open refund is not an admin decision")
	}
	if !BlockRecoveryBalance.Overridable() {
		t.Fatal("a recovery balance is an admin judgement call")
	}
	if !BlockNewChefRamp.Overridable() || !BlockAboveReviewThreshold.Overridable() {
		t.Fatal("ramp and threshold holds exist to be reviewed and released")
	}
}

func TestDecideRelease_ZeroRampOrdersDisablesTheRampCheck(t *testing.T) {
	// The ramp is runtime-tunable; setting it to 0 must turn it off rather
	// than block every chef forever.
	in := clean()
	in.RampOrders = 0
	in.DeliveredOrderCount = 0
	if hasReason(DecideRelease(in), BlockNewChefRamp) {
		t.Fatal("a zero ramp must disable the check")
	}
}

func TestDecideRelease_ZeroThresholdDisablesTheReviewCheck(t *testing.T) {
	in := clean()
	in.ReviewAbove = Zero(CurrencyINR)
	in.OrderTotal = Money{Minor: 10_000_000, Currency: CurrencyINR}
	if hasReason(DecideRelease(in), BlockAboveReviewThreshold) {
		t.Fatal("a zero threshold must disable the check")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/api && go test ./payouts/ -run TestDecideRelease`
Expected: FAIL — `undefined: ReleaseInput`, `undefined: DecideRelease`, `undefined: BlockNotMatured`.

- [ ] **Step 3: Write the minimal implementation**

Create `apps/api/payouts/governor.go`:

```go
package payouts

import "time"

// governor.go — the per-order release decision (#741).
//
// Route creates each order's payee transfers held at checkout, so a payout is
// not a batch to build but a hold to clear. This decides whether one order's
// transfers may be released.
//
// Pure by construction: no database, no gateway, no clock of its own. A block
// recorded against an order can therefore be re-derived from the same inputs
// months later, which is what makes the admin queue auditable.

// BlockReason is why a release was withheld. Persisted and grouped in
// analytics, so these string values are stable API.
type BlockReason string

const (
	// BlockNotMatured — still inside the maturation window.
	BlockNotMatured BlockReason = "not_matured"
	// BlockAutomationOff — automation is off globally or for this chef.
	BlockAutomationOff BlockReason = "automation_off"
	// BlockSettlementNotActivated — the chef's linked account has no working
	// settlement destination, so a release would strand the money.
	BlockSettlementNotActivated BlockReason = "settlement_not_activated"
	// BlockRefundOpen — a refund, chargeback or cancellation is in flight.
	BlockRefundOpen BlockReason = "refund_open"
	// BlockRecoveryBalance — the chef owes the platform.
	BlockRecoveryBalance BlockReason = "recovery_balance"
	// BlockNewChefRamp — inside the new-chef review period.
	BlockNewChefRamp BlockReason = "new_chef_ramp"
	// BlockAboveReviewThreshold — order value warrants human eyes.
	BlockAboveReviewThreshold BlockReason = "above_review_threshold"
)

// Overridable reports whether an admin may release despite this reason.
//
// The two hard blocks are not judgement calls: releasing to an unactivated
// account strands the money in a balance with no destination, and releasing
// against an open refund can settle to the chef's bank before the claw-back
// lands.
func (r BlockReason) Overridable() bool {
	switch r {
	case BlockSettlementNotActivated, BlockRefundOpen:
		return false
	default:
		return true
	}
}

// ReleaseInput is everything the decision may consider.
type ReleaseInput struct {
	Now         time.Time
	DeliveredAt time.Time
	Maturation  time.Duration

	AutomationEnabled   bool
	SettlementActivated bool
	RefundOpen          bool

	// RecoveryBalance is what the chef owes the platform, zero when clear.
	RecoveryBalance Money

	// DeliveredOrderCount is the chef's lifetime delivered orders; RampOrders
	// is the review period, and zero disables the check.
	DeliveredOrderCount int
	RampOrders          int

	// OrderTotal is this order's value; ReviewAbove is the threshold above
	// which a human looks, and zero disables the check.
	OrderTotal  Money
	ReviewAbove Money
}

// ReleaseDecision is the outcome for one order.
type ReleaseDecision struct {
	Release bool
	Reasons []BlockReason
}

// Blocked reports whether anything held this order back.
func (d ReleaseDecision) Blocked() bool { return len(d.Reasons) > 0 }

// DecideRelease evaluates every guardrail without short-circuiting, so an
// admin sees the whole picture rather than fixing one problem and immediately
// hitting the next.
func DecideRelease(in ReleaseInput) ReleaseDecision {
	var reasons []BlockReason

	if in.Now.Before(in.DeliveredAt.Add(in.Maturation)) {
		reasons = append(reasons, BlockNotMatured)
	}
	if !in.AutomationEnabled {
		reasons = append(reasons, BlockAutomationOff)
	}
	if !in.SettlementActivated {
		reasons = append(reasons, BlockSettlementNotActivated)
	}
	if in.RefundOpen {
		reasons = append(reasons, BlockRefundOpen)
	}
	if in.RecoveryBalance.IsPositive() {
		reasons = append(reasons, BlockRecoveryBalance)
	}
	if in.RampOrders > 0 && in.DeliveredOrderCount < in.RampOrders {
		reasons = append(reasons, BlockNewChefRamp)
	}
	if !in.ReviewAbove.IsZero() {
		// A comparison we cannot trust must block, never pass.
		if cmp, err := in.OrderTotal.Cmp(in.ReviewAbove); err != nil || cmp > 0 {
			reasons = append(reasons, BlockAboveReviewThreshold)
		}
	}

	return ReleaseDecision{Release: len(reasons) == 0, Reasons: reasons}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd apps/api && go test ./payouts/ -v -run TestDecideRelease`
Expected: all PASS. Then `go test ./payouts/` — the whole package including `boundary_test.go` must stay green, proving `governor.go` introduced no domain import.

- [ ] **Step 5: Commit**

```bash
cd /Users/samyakrout/Desktop/samyak-work/projects/new-repos/Home-Chef-App
git add apps/api/payouts/governor.go apps/api/payouts/governor_test.go
git commit -m "feat(api): per-order payout release decision (#741)

Route holds each order's payee transfers from checkout, so a payout is a hold
to clear rather than a batch to build. Evaluates every guardrail without
short-circuiting: an admin fixing one reason and retrying would otherwise walk
into the next one blind.

Settlement activation and open refunds are marked non-overridable. Neither is
a judgement call — releasing to an unactivated account strands the money, and
releasing against an open refund can reach the chef's bank before the
claw-back does."
```

---

### Task 2: Per-chef automation switch

Resolves the master kill switch, the per-chef tri-state and the configurable default into one boolean the governor consumes.

**Files:**
- Modify: `apps/api/models/chef.go` (add `PayoutAutoRelease`)
- Create: `apps/api/services/payout_automation.go`
- Test: `apps/api/services/payout_automation_test.go`

**Interfaces:**
- Consumes: `models.PlatformSettings`, `models.ChefProfile`.
- Produces: `services.PayoutAutomationEnabled(db *gorm.DB, chef *models.ChefProfile) bool`; constants `services.PayoutAutoOn = "on"`, `services.PayoutAutoOff = "off"`; setting keys `payout.sweep_enabled`, `payout.auto_release_default`.

- [ ] **Step 1: Add the model field**

In `apps/api/models/chef.go`, immediately after the `RazorpaySettlementRequirements` field added in #740:

```go
	// PayoutAutoRelease is the admin's per-chef automation switch: "on",
	// "off", or "" to follow payout.auto_release_default. A tri-state rather
	// than a boolean because rollout wants opt-in and steady state wants
	// opt-out, and a boolean would need live money config migrated to swap.
	PayoutAutoRelease string `gorm:"type:varchar(8);default:''" json:"payoutAutoRelease"`
```

- [ ] **Step 2: Write the failing test**

Create `apps/api/services/payout_automation_test.go`:

```go
package services

import (
	"testing"

	"github.com/homechef/api/models"
)

// #741 — resolving the master switch, the per-chef tri-state and the default.
// The precedence is the safety property: a kill switch that individual records
// can opt out of is not a kill switch.

func TestPayoutAutomation_MasterSwitchOffBeatsEverything(t *testing.T) {
	db := newTestDB(t)
	setSetting(t, db, "payout.sweep_enabled", "false")
	setSetting(t, db, "payout.auto_release_default", "on")

	chef := &models.ChefProfile{PayoutAutoRelease: PayoutAutoOn}
	if PayoutAutomationEnabled(db, chef) {
		t.Fatal("a per-chef 'on' must not override the master kill switch")
	}
}

func TestPayoutAutomation_ChefOnWithMasterOn(t *testing.T) {
	db := newTestDB(t)
	setSetting(t, db, "payout.sweep_enabled", "true")
	setSetting(t, db, "payout.auto_release_default", "off")

	chef := &models.ChefProfile{PayoutAutoRelease: PayoutAutoOn}
	if !PayoutAutomationEnabled(db, chef) {
		t.Fatal("an explicitly enabled chef releases even when the default is off")
	}
}

func TestPayoutAutomation_ChefOffBeatsDefaultOn(t *testing.T) {
	db := newTestDB(t)
	setSetting(t, db, "payout.sweep_enabled", "true")
	setSetting(t, db, "payout.auto_release_default", "on")

	chef := &models.ChefProfile{PayoutAutoRelease: PayoutAutoOff}
	if PayoutAutomationEnabled(db, chef) {
		t.Fatal("a suspended chef must not auto-release")
	}
}

func TestPayoutAutomation_UnsetFollowsTheDefault(t *testing.T) {
	db := newTestDB(t)
	setSetting(t, db, "payout.sweep_enabled", "true")

	chef := &models.ChefProfile{PayoutAutoRelease: ""}

	setSetting(t, db, "payout.auto_release_default", "off")
	if PayoutAutomationEnabled(db, chef) {
		t.Fatal("unset must follow the default (off)")
	}

	setSetting(t, db, "payout.auto_release_default", "on")
	if !PayoutAutomationEnabled(db, chef) {
		t.Fatal("unset must follow the default (on)")
	}
}

func TestPayoutAutomation_DefaultsClosedWhenUnconfigured(t *testing.T) {
	// An empty settings table must not start moving money.
	db := newTestDB(t)
	chef := &models.ChefProfile{}
	if PayoutAutomationEnabled(db, chef) {
		t.Fatal("missing settings must default to no automation")
	}
}

func TestPayoutAutomation_GarbageValuesFailClosed(t *testing.T) {
	db := newTestDB(t)
	setSetting(t, db, "payout.sweep_enabled", "yes-please")
	chef := &models.ChefProfile{PayoutAutoRelease: PayoutAutoOn}
	if PayoutAutomationEnabled(db, chef) {
		t.Fatal("an unparseable master switch must fail closed")
	}
}
```

Note: `newTestDB` and `setSetting` are the existing helpers used by `payout_readiness_test.go`. Read that file first and reuse them verbatim; if the names differ, use the ones it actually defines rather than adding duplicates.

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd apps/api && go test ./services/ -run TestPayoutAutomation`
Expected: FAIL — `undefined: PayoutAutomationEnabled`, `undefined: PayoutAutoOn`.

- [ ] **Step 4: Write the minimal implementation**

Create `apps/api/services/payout_automation.go`:

```go
package services

import (
	"strconv"

	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// payout_automation.go — resolving whether a chef's payouts auto-release
// (#741).
//
// Three controls with distinct roles, and the precedence between them is the
// safety property:
//
//  1. payout.sweep_enabled is an absolute kill switch. Off means nothing
//     releases for anyone; a per-chef "on" does not override it. This is the
//     control reached for when something is going wrong with live money, so
//     it cannot have exceptions.
//  2. The chef's tri-state decides next.
//  3. payout.auto_release_default fills in for an unset chef.
//
// Everything fails closed: a missing or unparseable setting means no
// automation, because the failure mode of over-releasing is money that has
// left the platform.

// PlatformSettings keys.
const (
	payoutSweepEnabledKey      = "payout.sweep_enabled"
	payoutAutoReleaseDefaultKey = "payout.auto_release_default"
)

// Per-chef switch values. Empty string means follow the default.
const (
	PayoutAutoOn  = "on"
	PayoutAutoOff = "off"
)

// settingValue reads a raw platform setting, empty when absent.
func settingValue(db *gorm.DB, key string) string {
	var setting models.PlatformSettings
	if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
		return ""
	}
	return setting.Value
}

// settingBool parses a boolean setting, defaulting to false on anything
// unparseable so a typo cannot switch money movement on.
func settingBool(db *gorm.DB, key string) bool {
	v, err := strconv.ParseBool(settingValue(db, key))
	return err == nil && v
}

// PayoutAutomationEnabled reports whether this chef's matured payouts may
// release without a human.
//
// It grants candidacy only — the guardrail chain still runs. A chef switched
// on with an open refund is still blocked.
func PayoutAutomationEnabled(db *gorm.DB, chef *models.ChefProfile) bool {
	if !settingBool(db, payoutSweepEnabledKey) {
		return false
	}
	if chef == nil {
		return false
	}
	switch chef.PayoutAutoRelease {
	case PayoutAutoOn:
		return true
	case PayoutAutoOff:
		return false
	default:
		return settingValue(db, payoutAutoReleaseDefaultKey) == PayoutAutoOn
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd apps/api && go test ./services/ -v -run TestPayoutAutomation`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/samyakrout/Desktop/samyak-work/projects/new-repos/Home-Chef-App
git add apps/api/models/chef.go apps/api/services/payout_automation.go apps/api/services/payout_automation_test.go
git commit -m "feat(api): per-chef payout automation switch (#741)

An admin-controlled tri-state, resolved under an absolute master kill switch:
a per-chef 'on' cannot override payout.sweep_enabled, because a kill switch
individual records can opt out of is not a kill switch.

Tri-state rather than a boolean because rollout wants opt-in and steady state
wants opt-out, and a boolean would need live money configuration migrated to
swap between them. Everything fails closed — a missing or unparseable setting
means no automation."
```

- [ ] **Step 7: Add the schema columns in tesserix-k8s**

The GORM fields added here and in #740 need real columns. In the **tesserix-k8s** repo, edit
`charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql` and append:

```sql
ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS razorpay_product_id VARCHAR(255) DEFAULT '';
ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS razorpay_settlement_status VARCHAR(64) DEFAULT '';
ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS razorpay_settlement_requirements TEXT DEFAULT '';
ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS payout_auto_release VARCHAR(8) DEFAULT '';
```

Commit and push there; the bootstrap CronJob applies it idempotently within 30 minutes. **Do not** add this SQL to the Home-Chef-App repo.

---

### Task 3: Recovery deduction at transfer creation

Route transfers are per-payment, so a debt cannot be netted across orders after the fact. It has to come off the next order's transfer before that transfer is created.

**Files:**
- Create: `apps/api/services/payout_recovery.go`
- Test: `apps/api/services/payout_recovery_test.go`
- Modify: `apps/api/handlers/payment.go:204-215` (`chefNetPayout`)

**Interfaces:**
- Consumes: `payouts.DeriveBalance`, `payouts.Balance.Recovery()`, `payouts.Money` from the merged core.
- Produces: `services.ApplyRecoveryDeduction(db *gorm.DB, chefID uuid.UUID, gross payouts.Money, now time.Time) (net payouts.Money, deducted payouts.Money, err error)`.

- [ ] **Step 1: Write the failing test**

Create `apps/api/services/payout_recovery_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/payouts"
)

// #741 — recovering a chef's debt from their next payout.
//
// Conservation is the property that matters: what is paid plus what is
// recovered plus what is carried forward must equal what was owed. Anything
// else quietly creates or destroys money.

func inr(minor int64) payouts.Money {
	return payouts.Money{Minor: minor, Currency: payouts.CurrencyINR}
}

func TestApplyRecoveryDeduction_NoDebtPaysInFull(t *testing.T) {
	db := newTestDB(t)
	net, deducted, err := ApplyRecoveryDeduction(db, uuid.New(), inr(50_000), time.Now())
	if err != nil {
		t.Fatalf("ApplyRecoveryDeduction: %v", err)
	}
	if net.Minor != 50_000 {
		t.Fatalf("net = %d, want 50000", net.Minor)
	}
	if !deducted.IsZero() {
		t.Fatalf("deducted = %v, want zero", deducted)
	}
}

func TestApplyRecoveryDeduction_PartialDebtReducesThePayout(t *testing.T) {
	db := newTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 15_000)

	net, deducted, err := ApplyRecoveryDeduction(db, chefID, inr(50_000), time.Now())
	if err != nil {
		t.Fatalf("ApplyRecoveryDeduction: %v", err)
	}
	if net.Minor != 35_000 {
		t.Fatalf("net = %d, want 35000", net.Minor)
	}
	if deducted.Minor != 15_000 {
		t.Fatalf("deducted = %d, want 15000", deducted.Minor)
	}
}

func TestApplyRecoveryDeduction_DebtLargerThanPayoutFloorsAtZero(t *testing.T) {
	// Never emit a negative transfer. The remainder stays owed.
	db := newTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 80_000)

	net, deducted, err := ApplyRecoveryDeduction(db, chefID, inr(50_000), time.Now())
	if err != nil {
		t.Fatalf("ApplyRecoveryDeduction: %v", err)
	}
	if net.Minor != 0 {
		t.Fatalf("net = %d, want 0 — never a negative transfer", net.Minor)
	}
	if deducted.Minor != 50_000 {
		t.Fatalf("deducted = %d, want the whole payout", deducted.Minor)
	}
}

func TestApplyRecoveryDeduction_Conserves(t *testing.T) {
	db := newTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 20_000)

	gross := inr(50_000)
	net, deducted, err := ApplyRecoveryDeduction(db, chefID, gross, time.Now())
	if err != nil {
		t.Fatalf("ApplyRecoveryDeduction: %v", err)
	}
	if net.Minor+deducted.Minor != gross.Minor {
		t.Fatalf("conservation broken: net %d + deducted %d != gross %d",
			net.Minor, deducted.Minor, gross.Minor)
	}
}
```

Add `seedPenalty` to the same file, inserting one `payouts.LedgerEntry` with `Kind: payouts.EntryDebitPenalty` for that chef:

```go
func seedPenalty(t *testing.T, db *gorm.DB, chefID uuid.UUID, minor int64) {
	t.Helper()
	entry := payouts.LedgerEntry{
		ID: uuid.New(), TenantID: "t1",
		PayeeType: payouts.PayeeChef, PayeeID: chefID,
		Kind: payouts.EntryDebitPenalty, AmountMinor: minor,
		Currency: payouts.CurrencyINR,
		SourceType: "order_issue", SourceID: uuid.NewString(),
	}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seed penalty: %v", err)
	}
}
```

The test DB must have `payout_ledger_entries`. Read how `payout_readiness_test.go` builds its schema and follow the same approach — a missing table makes these tests silently pass by returning a zero balance, which is exactly the failure this feature must not have.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/api && go test ./services/ -run TestApplyRecoveryDeduction`
Expected: FAIL — `undefined: ApplyRecoveryDeduction`.

- [ ] **Step 3: Write the minimal implementation**

Create `apps/api/services/payout_recovery.go`:

```go
package services

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/payouts"
)

// payout_recovery.go — collecting a chef's debt from their next payout (#741).
//
// Route moves money as per-payment transfers, so a penalty cannot be netted
// across orders after the fact the way a daily batch would. Recovery therefore
// happens before the next transfer is created: the outstanding balance comes
// off the gross, floored at zero, and whatever is left stays owed and is
// collected from the order after that.

// ApplyRecoveryDeduction reduces a gross payout by the chef's outstanding
// recovery balance.
//
// Returns the net to transfer and the amount recovered; the two always sum to
// the gross, so no money is created or destroyed by the deduction.
func ApplyRecoveryDeduction(db *gorm.DB, chefID uuid.UUID, gross payouts.Money, now time.Time) (payouts.Money, payouts.Money, error) {
	zero := payouts.Zero(gross.Currency)

	var entries []payouts.LedgerEntry
	if err := db.Where("payee_type = ? AND payee_id = ?", payouts.PayeeChef, chefID).
		Find(&entries).Error; err != nil {
		return zero, zero, err
	}

	balance, err := payouts.DeriveBalance(entries, now)
	if err != nil {
		return zero, zero, err
	}
	owed := balance.Recovery()
	if owed.IsZero() {
		return gross, zero, nil
	}

	// Deduct at most the whole payout — never emit a negative transfer.
	deducted := owed
	if cmp, err := owed.Cmp(gross); err == nil && cmp > 0 {
		deducted = gross
	}
	net, err := gross.Sub(deducted)
	if err != nil {
		return zero, zero, err
	}
	return net, deducted, nil
}
```

`payouts.Money.Sub` already exists (`payouts/money.go:90`) and returns a currency-mismatch error, so no new money primitive is needed.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd apps/api && go test ./services/ -v -run TestApplyRecoveryDeduction`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/samyakrout/Desktop/samyak-work/projects/new-repos/Home-Chef-App
git add apps/api/services/payout_recovery.go apps/api/services/payout_recovery_test.go
git commit -m "feat(api): recover chef debt from the next payout (#741)

Route moves money as per-payment transfers, so a penalty cannot be netted
across orders the way a daily batch would. The outstanding balance instead
comes off the next transfer before it is created, floored at zero so a debt
larger than the payout never produces a negative transfer — the remainder
stays owed and is collected from the following order."
```

---

### Task 4: The 15-minute release sweep

Wires the governor to real orders behind the existing cron machinery.

**Files:**
- Create: `apps/api/services/payout_release_cron.go`
- Test: `apps/api/services/payout_release_cron_test.go`
- Modify: `apps/api/services/cron_temporal.go:26` (register the job)

**Interfaces:**
- Consumes: `payouts.DecideRelease`, `services.PayoutAutomationEnabled`, `services.ReleaseOrderPayouts` (existing, in `order_payout.go`).
- Produces: `services.runPayoutReleaseSweep(ctx context.Context)`, `services.StartPayoutReleaseCron(ctx context.Context)`, `services.payoutReleaseInterval = 15 * time.Minute`, `services.BuildReleaseInput(db *gorm.DB, order *models.Order, now time.Time) (payouts.ReleaseInput, error)`.

- [ ] **Step 1: Write the failing test**

Create `apps/api/services/payout_release_cron_test.go`. Test `BuildReleaseInput` — the mapping from an order to a decision — rather than the ticker, because the ticker is the part with no logic in it:

```go
package services

import (
	"testing"
	"time"

	"github.com/homechef/api/models"
)

// #741 — mapping an order onto the release decision. The sweep itself is a
// loop; the interesting behaviour is what it feeds the governor.

func TestBuildReleaseInput_ReadsSettlementActivation(t *testing.T) {
	db := newTestDB(t)
	order := seedDeliveredOrder(t, db, func(c *models.ChefProfile) {
		c.RazorpaySettlementStatus = "activated"
	})

	in, err := BuildReleaseInput(db, order, time.Now())
	if err != nil {
		t.Fatalf("BuildReleaseInput: %v", err)
	}
	if !in.SettlementActivated {
		t.Fatal("an activated chef must be reported as activated")
	}
}

func TestBuildReleaseInput_TreatsNeedsClarificationAsNotActivated(t *testing.T) {
	// The single most dangerous mis-mapping: needs_clarification means
	// Razorpay has NOT accepted the bank account, so a release strands money.
	db := newTestDB(t)
	order := seedDeliveredOrder(t, db, func(c *models.ChefProfile) {
		c.RazorpaySettlementStatus = "needs_clarification"
	})

	in, err := BuildReleaseInput(db, order, time.Now())
	if err != nil {
		t.Fatalf("BuildReleaseInput: %v", err)
	}
	if in.SettlementActivated {
		t.Fatal("needs_clarification must not count as activated")
	}
}

func TestBuildReleaseInput_UsesConfiguredMaturation(t *testing.T) {
	db := newTestDB(t)
	setSetting(t, db, "payout.maturation_minutes", "120")
	order := seedDeliveredOrder(t, db, nil)

	in, err := BuildReleaseInput(db, order, time.Now())
	if err != nil {
		t.Fatalf("BuildReleaseInput: %v", err)
	}
	if in.Maturation != 2*time.Hour {
		t.Fatalf("maturation = %v, want 2h", in.Maturation)
	}
}

func TestBuildReleaseInput_FallsBackToTwoHours(t *testing.T) {
	// An unset or garbage setting must not mean "release immediately".
	db := newTestDB(t)
	setSetting(t, db, "payout.maturation_minutes", "not-a-number")
	order := seedDeliveredOrder(t, db, nil)

	in, err := BuildReleaseInput(db, order, time.Now())
	if err != nil {
		t.Fatalf("BuildReleaseInput: %v", err)
	}
	if in.Maturation != 2*time.Hour {
		t.Fatalf("maturation = %v, want the 2h fallback", in.Maturation)
	}
}

func TestBuildReleaseInput_FlagsAnOpenRefund(t *testing.T) {
	db := newTestDB(t)
	order := seedDeliveredOrder(t, db, nil)
	order.RefundedAt = ptrTime(time.Now())
	if err := db.Save(order).Error; err != nil {
		t.Fatalf("save: %v", err)
	}

	in, err := BuildReleaseInput(db, order, time.Now())
	if err != nil {
		t.Fatalf("BuildReleaseInput: %v", err)
	}
	if !in.RefundOpen {
		t.Fatal("a refunded order must block release")
	}
}
```

Write `seedDeliveredOrder(t, db, mutateChef)` to insert a chef profile and a delivered order referencing it, returning `*models.Order` with `Chef` preloaded. Add `ptrTime`. Inspect `models.Order` first for the actual delivered-timestamp and refund field names — use the real ones, and if `RefundedAt` is not the field, use whatever marks a refund and adjust the test name to match.

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/api && go test ./services/ -run TestBuildReleaseInput`
Expected: FAIL — `undefined: BuildReleaseInput`.

- [ ] **Step 3: Write the minimal implementation**

Create `apps/api/services/payout_release_cron.go`:

```go
package services

import (
	"context"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

// payout_release_cron.go — the 15-minute release sweep (#741).
//
// Route creates each order's payee transfers held at checkout. This finds
// delivered orders whose hold has matured, asks the governor whether they may
// be released, and clears the ones that may. Blocked orders keep their
// transfers held and surface in the admin queue with every reason.
//
// A sweep rather than a per-order timer: the worst-case lag is one interval,
// which is immaterial against Razorpay's fixed 2-working-day settlement, and a
// missed tick self-heals on the next one with no per-order state to reconcile.

const payoutReleaseInterval = 15 * time.Minute

// defaultMaturation is used when payout.maturation_minutes is unset or
// unparseable. Never zero — that would release the instant an order is
// delivered, removing the window a late refund needs.
const defaultMaturation = 2 * time.Hour

// maturationWindow reads the configured hold, falling back on anything invalid.
func maturationWindow(db *gorm.DB) time.Duration {
	mins, err := strconv.Atoi(settingValue(db, "payout.maturation_minutes"))
	if err != nil || mins <= 0 {
		return defaultMaturation
	}
	return time.Duration(mins) * time.Minute
}

// reviewThreshold reads the value above which a human looks; zero disables it.
func reviewThreshold(db *gorm.DB) payouts.Money {
	paise, err := strconv.ParseInt(settingValue(db, "payout.review_above_paise"), 10, 64)
	if err != nil || paise < 0 {
		return payouts.Zero(payouts.CurrencyINR)
	}
	return payouts.Money{Minor: paise, Currency: payouts.CurrencyINR}
}

// rampOrders reads the new-chef review period; zero disables it.
func rampOrders(db *gorm.DB) int {
	n, err := strconv.Atoi(settingValue(db, "payout.new_chef_ramp_orders"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// BuildReleaseInput maps one order onto the governor's inputs.
func BuildReleaseInput(db *gorm.DB, order *models.Order, now time.Time) (payouts.ReleaseInput, error) {
	var delivered int64
	if err := db.Model(&models.Order{}).
		Where("chef_id = ? AND status = ?", order.ChefID, models.OrderStatusDelivered).
		Count(&delivered).Error; err != nil {
		return payouts.ReleaseInput{}, err
	}

	gross := payouts.Money{Minor: int64(ToPaise(chefNetPayoutFor(order))), Currency: payouts.CurrencyINR}
	_, owed, err := ApplyRecoveryDeduction(db, order.ChefID, gross, now)
	if err != nil {
		return payouts.ReleaseInput{}, err
	}

	return payouts.ReleaseInput{
		Now:                 now,
		DeliveredAt:         deliveredAt(order),
		Maturation:          maturationWindow(db),
		AutomationEnabled:   PayoutAutomationEnabled(db, &order.Chef),
		SettlementActivated: order.Chef.RazorpaySettlementStatus == "activated",
		RefundOpen:          order.RefundedAt != nil,
		RecoveryBalance:     owed,
		DeliveredOrderCount: int(delivered),
		RampOrders:          rampOrders(db),
		OrderTotal:          payouts.Money{Minor: int64(ToPaise(order.Total)), Currency: payouts.CurrencyINR},
		ReviewAbove:         reviewThreshold(db),
	}, nil
}

// runPayoutReleaseSweep releases every matured, unblocked delivered order.
func runPayoutReleaseSweep(ctx context.Context) {
	now := time.Now()
	db := database.DB

	var orders []models.Order
	if err := db.Preload("Chef").
		Where("status = ? AND payout_settled_at IS NULL", models.OrderStatusDelivered).
		Limit(500).Find(&orders).Error; err != nil {
		log.Printf("payout-sweep: load orders: %v", err)
		return
	}

	for i := range orders {
		order := &orders[i]
		in, err := BuildReleaseInput(db, order, now)
		if err != nil {
			log.Printf("payout-sweep: build input order=%s: %v", order.OrderNumber, err)
			continue
		}
		decision := payouts.DecideRelease(in)
		if !decision.Release {
			recordPayoutBlock(db, order, decision)
			continue
		}
		if err := ReleaseOrderPayouts(order.ID); err != nil {
			// Returned so the Temporal activity retries; the next sweep
			// re-drives it either way.
			log.Printf("payout-sweep: release order=%s: %v", order.OrderNumber, err)
			continue
		}
	}
}

// StartPayoutReleaseCron is the legacy in-process fallback ticker.
func StartPayoutReleaseCron(ctx context.Context) {
	go func() {
		t := time.NewTicker(payoutReleaseInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runPayoutReleaseSweep(ctx)
			}
		}
	}()
}
```

Two helpers must be sourced from the existing codebase rather than reinvented:

- `chefNetPayoutFor(order)` — **move** the existing `chefNetPayout` from `handlers/payment.go:204` into `services` and have the handler call the moved version. Do not duplicate the calculation: two copies of a payout formula drifting apart is precisely how a marketplace pays the wrong amount.
- `deliveredAt(order)` — read the real delivered-timestamp field from `models.Order`. Confirm `payout_settled_at` is the correct settled marker while you are there; if it is not, use the real one.

`recordPayoutBlock` is new. Add it to the same file:

```go
// NATS subjects for payout movement. Product-scoped under payments.* to match
// the existing subject taxonomy.
const (
	SubjectPayoutReleased = "payments.payout_released"
	SubjectPayoutBlocked  = "payments.payout_blocked"
)

// PayoutBlockEvent is what the admin surface and analytics consume.
type PayoutBlockEvent struct {
	OrderID     uuid.UUID `json:"orderId"`
	OrderNumber string    `json:"orderNumber"`
	ChefID      uuid.UUID `json:"chefId"`
	Reasons     []string  `json:"reasons"`
	// Overridable is false when any reason is a hard block, so the admin UI can
	// disable the release action rather than rejecting it on click.
	Overridable bool `json:"overridable"`
}

// recordPayoutBlock publishes why an order was withheld.
//
// Through the transactional outbox rather than a direct NATS publish, so the
// event cannot be lost if the process dies between deciding and publishing —
// the same guarantee the rest of the money path relies on. OutboxRelay
// publishes to JetStream with PubAck and Nats-Msg-Id dedup, so a redelivery
// after a crash is at-least-once at the transport and effectively-once here.
func recordPayoutBlock(db *gorm.DB, order *models.Order, decision payouts.ReleaseDecision) {
	reasons := make([]string, 0, len(decision.Reasons))
	overridable := true
	for _, r := range decision.Reasons {
		reasons = append(reasons, string(r))
		if !r.Overridable() {
			overridable = false
		}
	}

	// A still-maturing order is the common case, not a problem — publishing it
	// every 15 minutes would drown the queue in noise.
	if len(reasons) == 1 && decision.Reasons[0] == payouts.BlockNotMatured {
		return
	}

	if err := EnqueueOutbox(db, SubjectPayoutBlocked, "order", order.ID.String(), PayoutBlockEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		ChefID:      order.ChefID,
		Reasons:     reasons,
		Overridable: overridable,
	}); err != nil {
		log.Printf("payout-sweep: enqueue block event order=%s: %v", order.OrderNumber, err)
	}
}
```

And publish the success side immediately after a successful `ReleaseOrderPayouts` in `runPayoutReleaseSweep`:

```go
		if err := EnqueueOutbox(db, SubjectPayoutReleased, "order", order.ID.String(), map[string]any{
			"orderId":     order.ID,
			"orderNumber": order.OrderNumber,
			"chefId":      order.ChefID,
		}); err != nil {
			log.Printf("payout-sweep: enqueue release event order=%s: %v", order.OrderNumber, err)
		}
```

Add a test asserting the noise guard, because without it the outbox fills with one row per unmatured order per 15 minutes:

```go
func TestRecordPayoutBlock_SkipsAStillMaturingOrder(t *testing.T) {
	db := newTestDB(t)
	order := seedDeliveredOrder(t, db, nil)
	recordPayoutBlock(db, order, payouts.ReleaseDecision{
		Reasons: []payouts.BlockReason{payouts.BlockNotMatured},
	})
	if outboxCount(t, db) != 0 {
		t.Fatal("a still-maturing order is the normal case and must not be published")
	}
}

func TestRecordPayoutBlock_PublishesARealBlock(t *testing.T) {
	db := newTestDB(t)
	order := seedDeliveredOrder(t, db, nil)
	recordPayoutBlock(db, order, payouts.ReleaseDecision{
		Reasons: []payouts.BlockReason{payouts.BlockSettlementNotActivated},
	})
	if outboxCount(t, db) != 1 {
		t.Fatal("a genuine block must reach the admin surface")
	}
}
```

Write `outboxCount` as a count over the outbox table used by `EnqueueOutbox` (`services/outbox.go:42`); read that file for the table name.

- [ ] **Step 4: Register the job**

In `apps/api/services/cron_temporal.go`, add to the `cronJobs()` slice:

```go
		// #741 — release matured, unblocked order payouts. Gated by
		// payout.sweep_enabled, which ships off.
		{"payout-release", payoutReleaseInterval, runPayoutReleaseSweep, StartPayoutReleaseCron},
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd apps/api && go build ./... && go test ./services/ ./payouts/`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/samyakrout/Desktop/samyak-work/projects/new-repos/Home-Chef-App
git add apps/api/services/payout_release_cron.go apps/api/services/payout_release_cron_test.go apps/api/services/cron_temporal.go
git commit -m "feat(api): 15-minute payout release sweep (#741)

Finds delivered orders whose hold has matured, asks the governor whether they
may be released, and clears the ones that may. Blocked orders keep their
transfers held and carry their reasons into the admin queue.

A sweep rather than a per-order timer: worst-case lag is one interval, which
is immaterial against Razorpay's fixed two-working-day settlement, and a
missed tick self-heals on the next one with no per-order state to reconcile.
An unset or unparseable maturation falls back to 2h rather than zero, which
would release the instant an order is delivered."
```

---

### Task 5: Admin API for blocked chefs and the automation switch

Extends the existing admin payout handler rather than starting a new surface.

**Files:**
- Modify: `apps/api/handlers/admin_payouts.go` (the file backing `adminPayoutHandler`)
- Modify: `apps/api/routes/routes.go:918-922`
- Test: `apps/api/handlers/admin_payouts_test.go`

**Interfaces:**
- Consumes: `services.PayoutAutomationEnabled`, `services.PayoutAutoOn`, `services.PayoutAutoOff`, `payouts.BlockReason`.
- Produces: `GET /admin/payouts/blocked-chefs`, `PUT /admin/chefs/:id/payout-automation`.

- [ ] **Step 1: Write the failing test**

Create `apps/api/handlers/admin_payouts_test.go`:

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/homechef/api/models"
)

// #747 — the admin controls over payout automation.

func TestSetPayoutAutomation_RejectsAnUnknownValue(t *testing.T) {
	// Only the three legal values may be stored. Anything else is read back as
	// "follow the default", which would silently re-enable a chef an admin
	// deliberately suspended.
	router, db := newAdminTestRouter(t)
	chef := seedChef(t, db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/admin/chefs/"+chef.ID.String()+"/payout-automation",
		strings.NewReader(`{"value":"enabled"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var reloaded models.ChefProfile
	if err := db.First(&reloaded, "id = ?", chef.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.PayoutAutoRelease != "" {
		t.Fatalf("stored %q — a rejected value must not be persisted", reloaded.PayoutAutoRelease)
	}
}

func TestSetPayoutAutomation_StoresALegalValue(t *testing.T) {
	router, db := newAdminTestRouter(t)
	chef := seedChef(t, db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/admin/chefs/"+chef.ID.String()+"/payout-automation",
		strings.NewReader(`{"value":"off"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var reloaded models.ChefProfile
	if err := db.First(&reloaded, "id = ?", chef.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.PayoutAutoRelease != "off" {
		t.Fatalf("stored %q, want off", reloaded.PayoutAutoRelease)
	}
}

func TestSetPayoutAutomation_WritesAnAuditEntry(t *testing.T) {
	// Suspending a chef acts on someone else's money; "who did this" must be
	// answerable months later.
	router, db := newAdminTestRouter(t)
	chef := seedChef(t, db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/admin/chefs/"+chef.ID.String()+"/payout-automation",
		strings.NewReader(`{"value":"off"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var count int64
	if err := db.Model(&models.AuditLog{}).
		Where("action = ? AND entity_id = ?", "chef.payout.automation", chef.ID.String()).
		Count(&count).Error; err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if count != 1 {
		t.Fatalf("audit rows = %d, want 1", count)
	}
}

func TestBlockedChefs_ListsNeedsClarificationWithRequirements(t *testing.T) {
	// Status alone tells an admin nothing about what to fix, so the
	// requirements have to travel with it.
	router, db := newAdminTestRouter(t)
	chef := seedChef(t, db)
	if err := db.Model(&chef).Updates(map[string]any{
		"razorpay_settlement_status":       "needs_clarification",
		"razorpay_settlement_requirements": `[{"field_reference":"settlements.ifsc_code"}]`,
	}).Error; err != nil {
		t.Fatalf("update: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/payouts/blocked-chefs", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body struct {
		Chefs []struct {
			SettlementStatus string `json:"settlementStatus"`
			Requirements     string `json:"requirements"`
		} `json:"chefs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Chefs) != 1 {
		t.Fatalf("chefs = %d, want 1", len(body.Chefs))
	}
	if !strings.Contains(body.Chefs[0].Requirements, "ifsc_code") {
		t.Fatalf("requirements = %q — must say what to fix", body.Chefs[0].Requirements)
	}
}

func TestBlockedChefs_ExcludesAnActivatedChef(t *testing.T) {
	router, db := newAdminTestRouter(t)
	chef := seedChef(t, db)
	if err := db.Model(&chef).Update("razorpay_settlement_status", "activated").Error; err != nil {
		t.Fatalf("update: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/payouts/blocked-chefs", nil)
	router.ServeHTTP(w, req)

	var body struct {
		Chefs []json.RawMessage `json:"chefs"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Chefs) != 0 {
		t.Fatalf("chefs = %d, want 0 — a healthy chef is not a blockage", len(body.Chefs))
	}
}
```

`newAdminTestRouter` and `seedChef` follow the existing handler-test setup in this package — read a neighbouring `*_test.go` in `apps/api/handlers/` and reuse its helpers verbatim rather than adding parallel ones. If the audit model is not `models.AuditLog`, use the real one.

- [ ] **Step 2: Run to verify it fails**

Run: `cd apps/api && go test ./handlers/ -run 'TestSetPayoutAutomation|TestBlockedChefs'`
Expected: FAIL — handler methods undefined.

- [ ] **Step 3: Implement the handlers**

Add to `apps/api/handlers/admin_payouts.go`:

```go
// SetPayoutAutomation switches a chef's payout automation on or off.
//
// Validated against the three legal values: an unrecognised string would be
// read back as "follow the default", silently re-enabling a chef an admin
// deliberately suspended.
func (h *AdminPayoutHandler) SetPayoutAutomation(c *gin.Context) {
	var req struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	switch req.Value {
	case services.PayoutAutoOn, services.PayoutAutoOff, "":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "value must be on, off or empty"})
		return
	}

	chefID := c.Param("id")
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "chef not found"})
		return
	}
	old := chef.PayoutAutoRelease
	if err := database.DB.Model(&chef).Update("payout_auto_release", req.Value).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
		return
	}

	services.LogAudit(c, "chef.payout.automation", "chef", chefID,
		gin.H{"payoutAutoRelease": old}, gin.H{"payoutAutoRelease": req.Value})
	c.JSON(http.StatusOK, gin.H{"payoutAutoRelease": req.Value})
}

// GetBlockedChefs lists chefs who cannot currently be paid, with Razorpay's
// own requirements so the blockage is actionable rather than merely visible.
func (h *AdminPayoutHandler) GetBlockedChefs(c *gin.Context) {
	var chefs []models.ChefProfile
	if err := database.DB.
		Where("razorpay_settlement_status IS NULL OR razorpay_settlement_status <> ?", "activated").
		Find(&chefs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load"})
		return
	}

	out := make([]gin.H, 0, len(chefs))
	for _, ch := range chefs {
		out = append(out, gin.H{
			"chefId":            ch.ID,
			"businessName":      ch.BusinessName,
			"settlementStatus":  ch.RazorpaySettlementStatus,
			"requirements":      ch.RazorpaySettlementRequirements,
			"payoutAutoRelease": ch.PayoutAutoRelease,
		})
	}
	c.JSON(http.StatusOK, gin.H{"chefs": out})
}
```

- [ ] **Step 4: Register the routes**

In `apps/api/routes/routes.go`, beside the existing payout routes:

```go
			admin.GET("/payouts/blocked-chefs", payoutPerm, adminPayoutHandler.GetBlockedChefs)
			admin.PUT("/chefs/:id/payout-automation", payoutPerm, adminPayoutHandler.SetPayoutAutomation)
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd apps/api && go build ./... && go test ./handlers/`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/samyakrout/Desktop/samyak-work/projects/new-repos/Home-Chef-App
git add apps/api/handlers/admin_payouts.go apps/api/handlers/admin_payouts_test.go apps/api/routes/routes.go
git commit -m "feat(api): admin blocked-chef list and payout automation switch (#747)

Surfaces chefs Razorpay has not accepted, carrying its own requirements so the
blockage is actionable rather than merely visible — an unactivated chef is
otherwise indistinguishable from a healthy one until they ask why they have
not been paid.

The switch validates against the three legal values: an unrecognised string
reads back as 'follow the default', which would silently re-enable a chef an
admin had suspended. Both actions are audited."
```

---

### Task 6: tesserix-home admin views

**Files (in the `tesserix-home` repo):**
- Create: `app/admin/apps/homechef/payouts/page.tsx`
- Create: `app/admin/apps/homechef/payouts/blocked-chefs.tsx`

**Interfaces:**
- Consumes: `GET /admin/payouts/blocked-chefs`, `PUT /admin/chefs/:id/payout-automation`, and the existing `GET /admin/payouts/pending`, `POST /admin/payouts/:aggType/:id/release`, `POST /admin/payouts/:aggType/:id/withhold` — all through the existing HMAC gateway proxy at `/api/admin/apps/homechef/gw`.
- Produces: no new backend surface.

- [ ] **Step 1: Confirm the proxy reaches the new routes**

The gateway proxies any `/admin/*` path, so no proxy change should be needed. Verify before building UI:

```bash
curl -s -X GET "$TESSERIX_HOME_URL/api/admin/apps/homechef/gw/admin/payouts/blocked-chefs" \
  -H "Cookie: $ADMIN_SESSION" | head -20
```

Expected: a JSON `{"chefs": [...]}` payload, not a 404. If it 404s, the proxy needs the route allow-listed — fix that before continuing.

- [ ] **Step 2: Build the blocked-chefs view**

A table of chefs who cannot be paid: business name, settlement status, Razorpay's requirements, and the automation tri-state as a three-way control (On / Off / Default). Follow the existing admin table components in this repo rather than introducing new ones.

Two things this view must get right:
- Render the `requirements` field, not just the status. "needs_clarification" alone tells an admin nothing about what to fix.
- Show what "Default" currently resolves to, so an admin setting a chef to Default knows whether that means on or off today.

- [ ] **Step 3: Extend the payout queue with block reasons**

The existing pending-payouts view gains a reasons column. Reasons marked non-overridable must render the release action as disabled with an explanation, not merely fail on click — an admin should not discover a rule by having a money action rejected.

- [ ] **Step 4: Verify**

Run: `pnpm build && pnpm tsc --noEmit` in the tesserix-home repo.
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add app/admin/apps/homechef/payouts/
git commit -m "feat(admin): HomeChef payout blocked-chef view and automation switch

Surfaces chefs Razorpay has not accepted along with its own requirements, and
exposes the per-chef automation tri-state. Non-overridable block reasons
disable the release action rather than rejecting it on click."
```

---

### Task 7: Retire the batch builder and reconcile ADR 0002

- [ ] **Step 1: Close PR #762**

```bash
gh pr close 762 --repo tesserix/Home-Chef-App \
  --comment "Superseded by the release governor. Route creates per-payment transfers held at checkout, so payouts are holds to clear rather than batches to build — see docs/superpowers/specs/2026-07-22-payout-release-governor-design.md. The payouts core from #754 is retained."
```

- [ ] **Step 2: Amend ADR 0002**

In `docs/adr/0002-automated-vendor-payouts.md`, add a `## Superseded` section at the top recording the three reversals — rail (RazorpayX → Route), cadence (daily batch, 24h → 15-minute sweep, 2h), recovery (batch netting → deduct at transfer creation) — each with a one-line reason and a pointer to the new spec. Leave the original decisions in place; an ADR records what was decided and why it changed, it is not edited into looking correct.

- [ ] **Step 3: Commit**

```bash
git add docs/adr/0002-automated-vendor-payouts.md
git commit -m "docs: record what supersedes ADR 0002

Rail, cadence and recovery all changed once Route was confirmed as the rail.
The original decisions stay as written — the ADR records what was decided and
why it changed rather than being edited into looking correct."
```

---

## Verification

After all tasks:

```bash
cd apps/api
go build ./... && go vet ./... && go test ./...
```

Expected: clean build, no vet findings, all tests pass — including `payouts/boundary_test.go`, which proves the payouts core took on no domain dependency.

**Do not enable `payout.sweep_enabled` as part of this work.** It ships off. Enabling it moves live settlement and belongs to the sandbox verification in #218, after a pilot chef has been switched on individually.
