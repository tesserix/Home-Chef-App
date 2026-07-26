# Test Chef Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an admin approve a chef as Test or Live, hide test kitchens from everyone but an allowlist, route their payments through Razorpay test credentials, partition all their data by mode, and clone a live kitchen into a test session for debugging — all on production, in the same database.

**Architecture:** `mode` becomes a first-class dimension. `ChefProfile.Mode` decides visibility, credentials and which data partition a chef currently inhabits; every chef-scoped transactional row carries its own `mode` so money operations on existing records stay correct across flips. A two-entry Razorpay client cache serves live and test credentials from separate Secret Manager slots. Flipping an established live chef to test opens a numbered `ChefTestSession` and clones its config plus a 30-day order window into test rows; live rows are never written while in test, so flipping back needs no restore.

**Tech Stack:** Go 1.26 / Gin / GORM / PostgreSQL 16 (`apps/api`), Next.js 16 (`tesserix-home/apps/web`), Expo React Native (`apps/mobile-customer`, `apps/mobile-vendor`), Helm + ArgoCD schema bootstrap (`tesserix-k8s`).

**Spec:** `docs/superpowers/specs/2026-07-26-test-chef-mode-design.md` — read it before starting. Every ambiguity is resolved there.

## Global Constraints

- **Default live, always.** Every new column defaults to `'live'`. An empty or unrecognised mode value MUST read as live. No code path may infer test mode from absence of data.
- **Never write raw `.sql` files into this repo.** All DDL goes in `tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`. GORM tags in `apps/api/models/` drive local `AutoMigrate` only. This is a hard repo rule (`CLAUDE.md`).
- **Never run container builds, image pushes, or deploys.** Verify with `go build ./...`, `go vet ./...`, `go test ./...`, `pnpm build`, `npx tsc --noEmit`. The user runs deploys.
- **Git identity** must be `sam123ben` / `samyak.rout@gmail.com` before any commit. No AI/Claude/Co-Authored-By references anywhere in commits, PRs, code comments or file content.
- **Branch:** `feat/test-chef-mode` in `Home-Chef-App` (already created, spec committed). Create matching branches in `tesserix-home` and `tesserix-k8s`.
- **Money code is test-first.** Every task that touches a payment, refund, transfer, payout, wallet or loyalty path writes the failing test first. This is the established convention in this codebase — see `services/gateway_idempotency_test.go`.
- **`ListChefs` / `SearchDishes` are Postgres-only** (they use `ANY()`, `ILIKE`, `NOW()`) and cannot run under the sqlite test harness. Test their filtering by unit-testing the scope/predicate functions in isolation, exactly as `handlers/chefs_search_test.go` does for the bounding-box math.
- **sqlite test harness pattern:** `gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})` with hand-written `CREATE TABLE` fixtures, then assign `database.DB = db`. Copy the shape from `services/account_lifecycle_test.go:25-60`. Fixtures MUST include every column GORM writes on insert, or inserts fail with "no such column".
- **UUID scanning gotcha:** in sqlite fixtures, a bare `Scan(&uuid.UUID{})` fails. Scan into `string` and parse. Recorded in prior work on this repo.

---

## File Structure

**`apps/api` (Go) — created:**

| File | Responsibility |
|---|---|
| `models/mode.go` | Mode constants, `NormalizeMode`, `IsTest` |
| `models/chef_test_session.go` | `ChefTestSession`, `ChefModeStats` |
| `services/test_mode_policy.go` | Viewer-allowlist platform setting |
| `services/test_mode_visibility.go` | `TestChefVisibility` scope, `ExcludeTestOrders` scope |
| `services/payment_mode.go` | `PaymentModeForChef`, `IsTestMode` |
| `services/test_session.go` | Open/close/purge sessions, flip blockers |
| `services/test_session_clone.go` | The live→test clone |
| `services/chef_mode_stats.go` | Per-mode aggregate writer |
| `handlers/admin_test_mode.go` | Admin endpoints for mode, policy, sessions |

**`apps/api` (Go) — modified:** `models/chef.go`, `models/order.go` (+ the other transactional models), `database/database.go` (AutoMigrate), `services/razorpay.go`, `services/onboarding_activation.go`, `services/provider_dispatch.go`, `services/payout_release.go`, `services/notifications.go`, `handlers/chefs.go`, `handlers/orders.go`, `handlers/payment.go`, `handlers/approval.go`, `handlers/admin.go`, `routes/routes.go`.

**`tesserix-home/apps/web` — created:** `components/admin/homechef/mode-toggle.tsx`, `components/admin/homechef/test-badge.tsx`, `app/admin/apps/homechef/chefs/[id]/test-sessions/page.tsx`.
**Modified:** `app/admin/apps/homechef/{approvals/[id],chefs,platform-settings,payment-gateway}/page.tsx`, `lib/products/homechef/*`.

**`apps/mobile-customer` / `apps/mobile-vendor`:** a `TestBadge` component and a vendor `TestModeBanner`, wired into the chef card, chef detail header, checkout screen and vendor dashboard shell.

**`tesserix-k8s`:** `charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`.

---

## Task 1: Mode primitives and chef columns

**Files:**
- Create: `apps/api/models/mode.go`
- Create: `apps/api/models/mode_test.go`
- Modify: `apps/api/models/chef.go` (add fields after `IsVerified`/`VerifiedAt`, around line 50)

**Interfaces:**
- Produces: `models.ChefModeLive`, `models.ChefModeTest`, `models.NormalizeMode(string) string`, `models.IsTestMode(string) bool`, `ChefProfile.Mode`, `ChefProfile.FirstLiveAt`, `ChefProfile.ActiveTestSessionID`, `ChefProfile.IsTestMode() bool`, `ChefProfile.IsBornTest() bool`

- [ ] **Step 1: Write the failing test**

`apps/api/models/mode_test.go`:

```go
package models

import "testing"

func TestNormalizeMode(t *testing.T) {
	cases := map[string]string{
		"live": ChefModeLive, "test": ChefModeTest,
		"": ChefModeLive, "LIVE": ChefModeLive, "TEST": ChefModeTest,
		" test ": ChefModeTest, "garbage": ChefModeLive,
	}
	for in, want := range cases {
		if got := NormalizeMode(in); got != want {
			t.Fatalf("NormalizeMode(%q) = %q, want %q", in, got, want)
		}
	}
}

// The fail-safe direction: anything we don't recognise must read as live, so a
// column-default glitch can never hide a real kitchen or route a real payment
// through sandbox credentials.
func TestIsTestModeIsConservative(t *testing.T) {
	for _, in := range []string{"", "live", "garbage", "tes", "testing"} {
		if IsTestMode(in) {
			t.Fatalf("IsTestMode(%q) must be false", in)
		}
	}
	if !IsTestMode("test") || !IsTestMode("TEST") {
		t.Fatal(`IsTestMode must accept "test" case-insensitively`)
	}
}

func TestChefBornTestVsFlipped(t *testing.T) {
	born := ChefProfile{Mode: ChefModeTest}
	if !born.IsBornTest() {
		t.Fatal("a chef that has never been live is born-test")
	}
	now := timeNowForTest()
	flipped := ChefProfile{Mode: ChefModeTest, FirstLiveAt: &now}
	if flipped.IsBornTest() {
		t.Fatal("a chef that has been live is not born-test")
	}
	if (&ChefProfile{Mode: ChefModeLive}).IsTestMode() {
		t.Fatal("a live chef is not in test mode")
	}
}
```

Add at the bottom of the same file:

```go
func timeNowForTest() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
```

and import `"time"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./models/ -run 'TestNormalizeMode|TestIsTestModeIsConservative|TestChefBornTestVsFlipped' -v`
Expected: FAIL — `undefined: ChefModeLive`.

- [ ] **Step 3: Write the implementation**

`apps/api/models/mode.go`:

```go
package models

import "strings"

// Mode partitions the platform into two worlds that share one database.
//
// "live" is the real marketplace: real customers, real money, live Razorpay
// credentials. "test" is a sandbox kitchen — visible only to the test-mode
// viewer allowlist, paid for with Razorpay test credentials, and excluded from
// every real-money and reporting path.
//
// Mode appears in two places with two different meanings. On ChefProfile it is
// configuration: which world the kitchen currently inhabits, freely flippable
// by an admin. On a transactional row it is a snapshot taken at creation and
// never changed, so a refund on an order paid in test mode still routes to the
// test gateway years after the chef went live.
const (
	ChefModeLive = "live"
	ChefModeTest = "test"
)

// NormalizeMode coerces any stored or user-supplied value to a known mode.
//
// Anything that is not recognisably "test" becomes "live". This asymmetry is
// deliberate and load-bearing: a corrupt or missing value must never hide a
// real kitchen from customers, and must never send a real payment through
// sandbox credentials (which would silently capture no money). The failure
// direction is always toward live.
func NormalizeMode(m string) string {
	if strings.EqualFold(strings.TrimSpace(m), ChefModeTest) {
		return ChefModeTest
	}
	return ChefModeLive
}

// IsTestMode reports whether a stored mode value means test.
func IsTestMode(m string) bool { return NormalizeMode(m) == ChefModeTest }
```

In `apps/api/models/chef.go`, immediately after the `VerifiedAt` field:

```go
	// Mode selects which Razorpay credential set, which visibility rules and
	// which data partition apply to this kitchen. Defaults to live so every
	// existing chef and every new onboarding is a real kitchen unless an admin
	// explicitly says otherwise.
	Mode string `gorm:"type:varchar(4);not null;default:'live';index" json:"mode"`

	// FirstLiveAt is stamped the first time this chef becomes live and is never
	// cleared. It distinguishes a born-test kitchen (nil — hidden from customers
	// entirely, since nobody has heard of it) from an established kitchen
	// temporarily flipped to test for debugging (set — still listed, shown as
	// closed, so its regulars don't think it shut down).
	FirstLiveAt *time.Time `gorm:"" json:"firstLiveAt,omitempty"`

	// ActiveTestSessionID points at the open ChefTestSession while Mode is
	// "test", and is nil while live.
	ActiveTestSessionID *uuid.UUID `gorm:"type:uuid" json:"activeTestSessionId,omitempty"`
```

And the helpers, near the other `ChefProfile` methods:

```go
// IsTestMode reports whether this kitchen currently inhabits the test partition.
func (c *ChefProfile) IsTestMode() bool { return IsTestMode(c.Mode) }

// IsBornTest reports whether this kitchen has never been live. A born-test
// kitchen is hidden from customers outright; a kitchen that has been live and
// is currently in test is shown as closed instead. See the visibility matrix in
// the design doc, §4.
func (c *ChefProfile) IsBornTest() bool { return c.FirstLiveAt == nil }
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./models/ -run 'TestNormalizeMode|TestIsTestModeIsConservative|TestChefBornTestVsFlipped' -v && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 5: Commit**

```bash
git add apps/api/models/mode.go apps/api/models/mode_test.go apps/api/models/chef.go
git commit -m "feat(test-mode): add mode primitives and chef mode columns"
```

---

## Task 2: Test session and per-mode stats models

**Files:**
- Create: `apps/api/models/chef_test_session.go`
- Modify: `apps/api/database/database.go` (AutoMigrate list at line 146)

**Interfaces:**
- Consumes: `models.ChefModeLive` (Task 1)
- Produces: `models.ChefTestSession`, `models.ChefModeStats`, `models.TestSessionOpen/Closed/Purged`

- [ ] **Step 1: Write the model**

`apps/api/models/chef_test_session.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Test session lifecycle. A session is open while the chef is in test mode,
// closed when they flip back to live, and purged once an admin deletes its rows.
const (
	TestSessionOpen   = "open"
	TestSessionClosed = "closed"
	TestSessionPurged = "purged"
)

// ChefTestSession is one debugging episode for one kitchen.
//
// Every live→test flip opens a new session with a fresh clone, so the evidence
// from a previous investigation is never overwritten by the next one. Sessions
// are retained indefinitely and removed only by an explicit admin purge.
type ChefTestSession struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`

	// SessionNo is a per-chef counter starting at 1, for human reference in the
	// admin UI ("session 3") — the UUID is not memorable enough to talk about.
	SessionNo int `gorm:"not null" json:"sessionNo"`

	Status string `gorm:"type:varchar(10);not null;default:'open';index" json:"status"`

	// Reason is the admin's free-text note for why this kitchen was flipped.
	// Required at open time: six months later nobody remembers.
	Reason string `gorm:"type:text" json:"reason"`

	// OrderWindowDays records how far back the clone reached, so a session's
	// contents are self-describing even after the default changes.
	OrderWindowDays int `gorm:"default:30" json:"orderWindowDays"`

	ClonedAt *time.Time `gorm:"" json:"clonedAt,omitempty"`

	// CloneSummary is a per-table row count, e.g. {"menu_items":42,"orders":118}.
	// Surfaced in the admin UI so an admin can tell at a glance whether the
	// clone actually caught the data they needed.
	CloneSummary datatypes.JSON `gorm:"type:jsonb" json:"cloneSummary,omitempty"`

	OpenedByID uuid.UUID  `gorm:"type:uuid" json:"openedById"`
	OpenedAt   time.Time  `gorm:"autoCreateTime" json:"openedAt"`
	ClosedByID *uuid.UUID `gorm:"type:uuid" json:"closedById,omitempty"`
	ClosedAt   *time.Time `gorm:"" json:"closedAt,omitempty"`
	PurgedAt   *time.Time `gorm:"" json:"purgedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (ChefTestSession) TableName() string { return "chef_test_sessions" }

// ChefModeStats holds a kitchen's aggregate counters for ONE mode.
//
// chef_profiles keeps the live numbers verbatim so every customer-facing
// surface reads them unchanged and a fake order can never move a real rating.
// This table is what the vendor and admin dashboards read for whichever mode is
// currently active.
type ChefModeStats struct {
	ChefID uuid.UUID `gorm:"type:uuid;primaryKey" json:"chefId"`
	Mode   string    `gorm:"type:varchar(4);primaryKey;default:'live'" json:"mode"`

	TotalOrders  int     `gorm:"default:0" json:"totalOrders"`
	Rating       float64 `gorm:"default:0" json:"rating"`
	TotalReviews int     `gorm:"default:0" json:"totalReviews"`
	IssueCount   int     `gorm:"default:0" json:"issueCount"`

	UpdatedAt time.Time `json:"updatedAt"`
}

func (ChefModeStats) TableName() string { return "chef_mode_stats" }
```

- [ ] **Step 2: Register both models in AutoMigrate**

In `apps/api/database/database.go`, add `&models.ChefTestSession{}, &models.ChefModeStats{},` to the `DB.AutoMigrate(` argument list at line 146, next to the other chef models.

- [ ] **Step 3: Verify it compiles and migrates**

Run: `cd apps/api && go build ./... && go vet ./...`
Expected: clean. If `gorm.io/datatypes` is not already a dependency, run `go get gorm.io/datatypes` and commit the `go.mod`/`go.sum` change with this task.

- [ ] **Step 4: Commit**

```bash
git add apps/api/models/chef_test_session.go apps/api/database/database.go apps/api/go.mod apps/api/go.sum
git commit -m "feat(test-mode): add test session and per-mode stats models"
```

---

## Task 3: Mode column on transactional tables

**Files:**
- Modify: `apps/api/models/order.go`, `group_order.go`, `meal_plan.go`, `meal_subscription.go`, `catering.go`, `tip.go`, `promotion.go`, `review.go`, `refund_transaction.go`, `order_issue.go`, `cancellation_request.go`, `statement.go`
- Create: `apps/api/models/mode_columns_test.go`

**Interfaces:**
- Consumes: `models.ChefModeLive` (Task 1)
- Produces: a `Mode string`, `TestSessionID *uuid.UUID`, `ClonedFromID *uuid.UUID` triple on each listed model

- [ ] **Step 1: Inventory the tables**

Run and record the output — this is the authoritative list, not the one above:

```bash
cd apps/api && grep -rln "ChefID" models/*.go | grep -v _test
grep -rn "RazorpayOrderID" models/*.go | grep -v _test
```

Any model that is (a) chef-scoped AND (b) transactional (represents an event, not configuration) gets the triple. Configuration models (`MenuItem`, `ChefSchedule`, `WeeklyMenu`, `DailyMenu`, `ChefSettings`, capacity) get **`Mode` and `TestSessionID` only** — they are cloned, so they need the partition key, but `ClonedFromID` is added to them too since the clone copies them. In practice: give all cloned or partitioned tables the full triple. Simpler and uniform.

- [ ] **Step 2: Add the field block to each model**

Paste this identical block into each model struct, adjusting nothing:

```go
	// Mode is the data partition this row belongs to, snapshotted at creation
	// from the chef's mode and never changed afterwards. Money operations read
	// THIS field, not the chef's current mode — a chef flipped test→live must
	// still be able to refund an order that was paid with test credentials.
	Mode string `gorm:"type:varchar(4);not null;default:'live';index" json:"mode"`

	// TestSessionID ties a test row to the debugging session it belongs to, so
	// sessions can be listed and purged independently.
	TestSessionID *uuid.UUID `gorm:"type:uuid;index" json:"testSessionId,omitempty"`

	// ClonedFromID is set on rows produced by the live→test clone. A cloned row
	// is a historical replica belonging to a real customer who never placed it
	// in the sandbox, so it must never surface to any customer.
	ClonedFromID *uuid.UUID `gorm:"type:uuid;index" json:"-"`
```

Add `"github.com/google/uuid"` to imports where missing.

- [ ] **Step 3: Write a test that pins the default**

`apps/api/models/mode_columns_test.go`:

```go
package models

import "testing"

// Every partitioned model must default to live when constructed zero-valued.
// NormalizeMode is what guarantees this at read time; this test guards against
// someone "helpfully" changing the default to test during development.
func TestZeroValuedModeReadsLive(t *testing.T) {
	if NormalizeMode((&Order{}).Mode) != ChefModeLive {
		t.Fatal("a zero-valued Order must read as live")
	}
	if NormalizeMode((&Tip{}).Mode) != ChefModeLive {
		t.Fatal("a zero-valued Tip must read as live")
	}
	if NormalizeMode((&Review{}).Mode) != ChefModeLive {
		t.Fatal("a zero-valued Review must read as live")
	}
}
```

- [ ] **Step 4: Run the tests and build**

Run: `cd apps/api && go test ./models/ -v && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 5: Commit**

```bash
git add apps/api/models/
git commit -m "feat(test-mode): partition transactional models by mode"
```

---

## Task 4: Test-mode viewer allowlist policy

**Files:**
- Create: `apps/api/services/test_mode_policy.go`
- Create: `apps/api/services/test_mode_policy_test.go`

**Interfaces:**
- Produces: `services.TestModePolicy{ViewerEmails []string}`, `DefaultTestModePolicy()`, `(TestModePolicy) MayViewTestChefs(string) bool`, `GetTestModePolicy()`, `SaveTestModePolicy(TestModePolicy, *uuid.UUID) error`, `InvalidateTestModePolicy()`

- [ ] **Step 1: Write the failing test**

`apps/api/services/test_mode_policy_test.go`:

```go
package services

import "testing"

func TestMayViewTestChefs(t *testing.T) {
	p := TestModePolicy{ViewerEmails: []string{"Samyak.Rout@Gmail.com", " unidevidp@gmail.com "}}

	for _, ok := range []string{
		"samyak.rout@gmail.com", "SAMYAK.ROUT@GMAIL.COM", "unidevidp@gmail.com", " unidevidp@gmail.com ",
	} {
		if !p.MayViewTestChefs(ok) {
			t.Fatalf("%q must be allowed (case-insensitive, whitespace-trimmed)", ok)
		}
	}
	for _, no := range []string{"", "   ", "someone@else.com", "samyak.rout@gmail.com.evil.com"} {
		if p.MayViewTestChefs(no) {
			t.Fatalf("%q must NOT be allowed", no)
		}
	}
}

// An anonymous caller has no email. It must never match, including against a
// policy that has accidentally been saved with an empty string in the list.
func TestAnonymousNeverMatches(t *testing.T) {
	p := TestModePolicy{ViewerEmails: []string{"", "  "}}
	if p.MayViewTestChefs("") {
		t.Fatal("anonymous callers must never see test chefs")
	}
}

func TestDefaultPolicySeedsTheThreeTesters(t *testing.T) {
	d := DefaultTestModePolicy()
	for _, want := range []string{
		"samyak.rout@gmail.com", "unidevidp@gmail.com", "mahesh.sangawar@gmail.com",
	} {
		if !d.MayViewTestChefs(want) {
			t.Fatalf("default policy must seed %s", want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./services/ -run 'TestMayViewTestChefs|TestAnonymousNeverMatches|TestDefaultPolicySeeds' -v`
Expected: FAIL — `undefined: TestModePolicy`.

- [ ] **Step 3: Write the implementation**

`apps/api/services/test_mode_policy.go` — mirror `services/platform_config.go` exactly (same cache/TTL/save shape, `platformConfigTTL` is already defined there and is reused):

```go
package services

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// testModePolicyKey is the PlatformSettings row holding the whole policy blob.
const testModePolicyKey = "test_mode_policy"

// TestModePolicy controls who can see test-mode kitchens.
//
// Test kitchens exist on production alongside real ones. The only thing
// standing between a real customer and a fake kitchen is this list, so it is
// deliberately an explicit allowlist with no wildcard, no domain matching and
// no "all admins" shortcut.
type TestModePolicy struct {
	// ViewerEmails may browse, open and order from test-mode kitchens.
	// Matched case-insensitively with surrounding whitespace trimmed.
	ViewerEmails []string `json:"viewerEmails"`
}

// DefaultTestModePolicy seeds the three platform testers so the feature works
// the moment it deploys, before anyone opens the admin UI. Unlike the security
// policy's exempt list — which is deliberately empty because it bypasses 2FA —
// this list grants no privilege over real data, only visibility of fake data.
func DefaultTestModePolicy() TestModePolicy {
	return TestModePolicy{ViewerEmails: []string{
		"samyak.rout@gmail.com",
		"unidevidp@gmail.com",
		"mahesh.sangawar@gmail.com",
	}}
}

// MayViewTestChefs reports whether this email may see test-mode kitchens.
// An empty email (an anonymous caller) is always false, and an empty entry in
// the list can never match it.
func (p TestModePolicy) MayViewTestChefs(email string) bool {
	e := strings.TrimSpace(email)
	if e == "" {
		return false
	}
	for _, v := range p.ViewerEmails {
		if v := strings.TrimSpace(v); v != "" && strings.EqualFold(v, e) {
			return true
		}
	}
	return false
}

var (
	testModeCache     *TestModePolicy
	testModeFetchedAt time.Time
	testModeMu        sync.RWMutex
)

// GetTestModePolicy returns the policy, cached for platformConfigTTL. A missing
// or malformed row falls back to defaults so the policy is always valid.
func GetTestModePolicy() TestModePolicy {
	testModeMu.RLock()
	if testModeCache != nil && time.Since(testModeFetchedAt) < platformConfigTTL {
		defer testModeMu.RUnlock()
		return *testModeCache
	}
	testModeMu.RUnlock()

	testModeMu.Lock()
	defer testModeMu.Unlock()
	if testModeCache != nil && time.Since(testModeFetchedAt) < platformConfigTTL {
		return *testModeCache
	}
	fresh := loadTestModePolicyFromDB()
	testModeCache = &fresh
	testModeFetchedAt = time.Now()
	return fresh
}

// InvalidateTestModePolicy drops the cache so the next read refetches.
func InvalidateTestModePolicy() {
	testModeMu.Lock()
	defer testModeMu.Unlock()
	testModeCache = nil
}

// SaveTestModePolicy persists the policy and invalidates the cache.
func SaveTestModePolicy(p TestModePolicy, updatedBy *uuid.UUID) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	var setting models.PlatformSettings
	err = database.DB.Where("key = ?", testModePolicyKey).First(&setting).Error
	if err != nil {
		setting = models.PlatformSettings{
			Key: testModePolicyKey, Value: string(raw), Type: "json", UpdatedBy: updatedBy,
		}
		if err := database.DB.Create(&setting).Error; err != nil {
			return err
		}
	} else {
		setting.Value, setting.Type, setting.UpdatedBy = string(raw), "json", updatedBy
		if err := database.DB.Save(&setting).Error; err != nil {
			return err
		}
	}
	InvalidateTestModePolicy()
	return nil
}

func loadTestModePolicyFromDB() TestModePolicy {
	def := DefaultTestModePolicy()
	var setting models.PlatformSettings
	if err := database.DB.Where("key = ?", testModePolicyKey).First(&setting).Error; err != nil {
		return def
	}
	if setting.Value == "" {
		return def
	}
	var parsed TestModePolicy
	if err := json.Unmarshal([]byte(setting.Value), &parsed); err != nil {
		log.Printf("test_mode_policy: parse failed, using defaults: %v", err)
		return def
	}
	// A saved policy with an empty list is a legitimate choice — it means "no
	// one but admins can see test kitchens" — so it is NOT backfilled with
	// defaults. Only an unparseable or absent row falls back.
	return parsed
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run 'TestMayViewTestChefs|TestAnonymousNeverMatches|TestDefaultPolicySeeds' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/test_mode_policy.go apps/api/services/test_mode_policy_test.go
git commit -m "feat(test-mode): add admin-editable viewer allowlist policy"
```

---

## Task 5: Mode-aware Razorpay client factory

**Files:**
- Modify: `apps/api/services/razorpay.go:36-170`
- Create: `apps/api/services/razorpay_mode_test.go`

**Interfaces:**
- Consumes: `models.ChefModeLive`, `models.NormalizeMode` (Task 1)
- Produces: `services.SecretRazorpayTestKeyID`, `SecretRazorpayTestKeySecret`, `SecretRazorpayTestWebhookSecret`, `GetRazorpayFor(mode string) *RazorpayClient`, `InvalidateRazorpayFor(mode string)`, `SetRazorpayClientFor(mode string, c *RazorpayClient)` (test seam)

- [ ] **Step 1: Write the failing test**

`apps/api/services/razorpay_mode_test.go`:

```go
package services

import (
	"testing"

	"github.com/homechef/api/models"
)

// The two slots must be genuinely independent. A shared cache would mean the
// first chef to transact decides which credentials everyone else uses — the
// exact failure this feature exists to prevent.
func TestRazorpaySlotsAreIndependent(t *testing.T) {
	t.Cleanup(func() {
		InvalidateRazorpayFor(models.ChefModeLive)
		InvalidateRazorpayFor(models.ChefModeTest)
	})

	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{keyID: "rzp_live_AAA", keySecret: "s1"})
	SetRazorpayClientFor(models.ChefModeTest, &RazorpayClient{keyID: "rzp_test_BBB", keySecret: "s2"})

	if got := GetRazorpayFor(models.ChefModeLive).GetKeyID(); got != "rzp_live_AAA" {
		t.Fatalf("live slot = %q, want rzp_live_AAA", got)
	}
	if got := GetRazorpayFor(models.ChefModeTest).GetKeyID(); got != "rzp_test_BBB" {
		t.Fatalf("test slot = %q, want rzp_test_BBB", got)
	}

	// Invalidating one slot must not evict the other.
	InvalidateRazorpayFor(models.ChefModeTest)
	if GetRazorpayFor(models.ChefModeLive).GetKeyID() != "rzp_live_AAA" {
		t.Fatal("invalidating test must not evict the live client")
	}
}

// An unrecognised mode must resolve to live, matching NormalizeMode. A typo in
// a caller must never silently fall through to sandbox credentials.
func TestUnknownModeUsesLiveSlot(t *testing.T) {
	t.Cleanup(func() { InvalidateRazorpayFor(models.ChefModeLive) })
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{keyID: "rzp_live_AAA"})
	if GetRazorpayFor("garbage").GetKeyID() != "rzp_live_AAA" {
		t.Fatal("an unknown mode must use the live slot")
	}
}

// GetRazorpay() is retained for the ~20 non-chef-scoped call sites and must
// remain exactly the live slot.
func TestLegacyGetRazorpayIsLiveSlot(t *testing.T) {
	t.Cleanup(func() { InvalidateRazorpayFor(models.ChefModeLive) })
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{keyID: "rzp_live_AAA"})
	if GetRazorpay().GetKeyID() != "rzp_live_AAA" {
		t.Fatal("GetRazorpay() must be the live slot")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./services/ -run 'TestRazorpaySlots|TestUnknownModeUsesLive|TestLegacyGetRazorpay' -v`
Expected: FAIL — `undefined: GetRazorpayFor`.

- [ ] **Step 3: Rework the cache in `services/razorpay.go`**

Add next to the existing secret-name constants (line 38):

```go
	// Test-mode credential slot. Separate secrets rather than a mode suffix on
	// one value, so rotating or clearing one slot cannot disturb the other.
	SecretRazorpayTestKeyID         = "prod-homechef-razorpay-test-key-id"
	SecretRazorpayTestKeySecret     = "prod-homechef-razorpay-test-key-secret"
	SecretRazorpayTestWebhookSecret = "prod-homechef-razorpay-test-webhook-secret"
```

Replace the `razorpayClient *RazorpayClient` singleton with a per-mode map:

```go
var (
	razorpayClients = map[string]*RazorpayClient{}
	razorpayMu      sync.Mutex
)

// razorpaySecretNames returns the Secret Manager keys for a mode's slot.
func razorpaySecretNames(mode string) (keyID, keySecret, webhookSecret string) {
	if models.IsTestMode(mode) {
		return SecretRazorpayTestKeyID, SecretRazorpayTestKeySecret, SecretRazorpayTestWebhookSecret
	}
	return SecretRazorpayKeyID, SecretRazorpayKeySecret, SecretRazorpayWebhookSecret
}

// GetRazorpayFor returns the cached client for a mode, fetching that mode's
// credentials from Secret Manager on a cache miss. Slots are cached and
// invalidated independently. Returns nil when the slot is not configured —
// every caller already handles nil.
func GetRazorpayFor(mode string) *RazorpayClient {
	mode = models.NormalizeMode(mode)

	razorpayMu.Lock()
	defer razorpayMu.Unlock()

	if c := razorpayClients[mode]; c != nil && time.Since(c.fetchedAt) < razorpayCacheTTL {
		return c
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fresh, err := fetchRazorpayFromSM(ctx, mode)
	if err != nil {
		// Keep serving a stale client through a transient SM outage rather than
		// going dark — only fetchedAt is out of date, the credentials are fine.
		if c := razorpayClients[mode]; c != nil {
			log.Printf("razorpay[%s]: SM fetch failed, using cached credentials: %v", mode, err)
			return c
		}
		log.Printf("razorpay[%s]: not configured (%v)", mode, err)
		return nil
	}
	razorpayClients[mode] = fresh
	return fresh
}

// GetRazorpay is the live slot. Retained so the non-chef-scoped call sites
// (admin gateway status, reconciliation, wallet top-ups, platform
// subscriptions) keep compiling and keep meaning exactly what they meant.
func GetRazorpay() *RazorpayClient { return GetRazorpayFor(models.ChefModeLive) }

// InvalidateRazorpayFor drops one slot's cached client. Saving test keys must
// not evict a healthy live client, and vice versa.
func InvalidateRazorpayFor(mode string) {
	mode = models.NormalizeMode(mode)
	razorpayMu.Lock()
	defer razorpayMu.Unlock()
	delete(razorpayClients, mode)
	log.Printf("razorpay[%s]: credential cache invalidated", mode)
}

// InvalidateRazorpay clears every slot.
func InvalidateRazorpay() {
	InvalidateRazorpayFor(models.ChefModeLive)
	InvalidateRazorpayFor(models.ChefModeTest)
}

// SetRazorpayClientFor installs a client directly. Test seam only.
func SetRazorpayClientFor(mode string, c *RazorpayClient) {
	mode = models.NormalizeMode(mode)
	razorpayMu.Lock()
	defer razorpayMu.Unlock()
	if c != nil && c.fetchedAt.IsZero() {
		c.fetchedAt = time.Now()
	}
	razorpayClients[mode] = c
}
```

Change `fetchRazorpayFromSM(ctx context.Context)` to `fetchRazorpayFromSM(ctx context.Context, mode string)`, resolve its three secret names via `razorpaySecretNames(mode)`, and gate the env-var dev fallback on `!models.IsTestMode(mode)` — the env fallback only ever described the live slot, and applying it to test would silently serve live credentials to a sandbox order.

Update `snapshotRazorpayClient()` to take a mode and read `razorpayClients[mode]`. Update `InitRazorpay()` to probe both slots and log each independently.

Add `"github.com/homechef/api/models"` to the imports.

- [ ] **Step 4: Fix the existing call sites of the removed singleton**

Run: `cd apps/api && go build ./... 2>&1 | head -40`
Fix each reported reference to the old `razorpayClient` variable. Do NOT change the semantics of any handler yet — every existing site keeps calling `GetRazorpay()` (live) in this task. Threading mode through handlers is Task 7.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run 'Razorpay' -v && go build ./... && go vet ./...`
Expected: PASS, clean.

- [ ] **Step 6: Commit**

```bash
git add apps/api/services/razorpay.go apps/api/services/razorpay_mode_test.go
git commit -m "feat(test-mode): serve razorpay credentials from independent live and test slots"
```

---

## Task 6: Mode-aware signature verification

**Files:**
- Modify: `apps/api/services/razorpay.go:574-605` (`VerifyWebhookSignature`, `VerifyPaymentSignature`)
- Create: `apps/api/services/razorpay_signature_mode_test.go`
- Modify: `apps/api/handlers/payment.go` (webhook handler + verify handler)

**Interfaces:**
- Consumes: `GetRazorpayFor`, `SetRazorpayClientFor` (Task 5)
- Produces: `services.VerifyWebhookSignatureMode(payload []byte, signature string) (ok bool, mode string)`, `services.VerifyPaymentSignatureFor(mode, razorpayOrderID, razorpayPaymentID, signature string) bool`

- [ ] **Step 1: Write the failing test**

`apps/api/services/razorpay_signature_mode_test.go`:

```go
package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/homechef/api/models"
)

func sign(secret string, payload []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(payload)
	return hex.EncodeToString(m.Sum(nil))
}

func TestWebhookSignatureIdentifiesSigningMode(t *testing.T) {
	t.Cleanup(InvalidateRazorpay)
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{webhookSecret: "live-secret"})
	SetRazorpayClientFor(models.ChefModeTest, &RazorpayClient{webhookSecret: "test-secret"})

	payload := []byte(`{"event":"payment.captured"}`)

	ok, mode := VerifyWebhookSignatureMode(payload, sign("live-secret", payload))
	if !ok || mode != models.ChefModeLive {
		t.Fatalf("live-signed webhook = (%v,%q), want (true,live)", ok, mode)
	}
	ok, mode = VerifyWebhookSignatureMode(payload, sign("test-secret", payload))
	if !ok || mode != models.ChefModeTest {
		t.Fatalf("test-signed webhook = (%v,%q), want (true,test)", ok, mode)
	}
	if ok, _ := VerifyWebhookSignatureMode(payload, sign("wrong", payload)); ok {
		t.Fatal("an unsigned/foreign webhook must be rejected")
	}
}

// The interim production state: both slots hold the SAME test key until a real
// live key is issued. The live attempt then always wins, so mode reported here
// is not trustworthy on its own — which is exactly why the handler must also
// compare against the record's own mode. This test documents the behaviour so
// nobody "fixes" it later without understanding the consequence.
func TestIdenticalSecretsResolveToLive(t *testing.T) {
	t.Cleanup(InvalidateRazorpay)
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{webhookSecret: "same"})
	SetRazorpayClientFor(models.ChefModeTest, &RazorpayClient{webhookSecret: "same"})

	payload := []byte(`{"event":"payment.captured"}`)
	ok, mode := VerifyWebhookSignatureMode(payload, sign("same", payload))
	if !ok || mode != models.ChefModeLive {
		t.Fatalf("identical secrets = (%v,%q), want (true,live)", ok, mode)
	}
}

func TestPaymentSignatureIsModeScoped(t *testing.T) {
	t.Cleanup(InvalidateRazorpay)
	SetRazorpayClientFor(models.ChefModeLive, &RazorpayClient{keySecret: "live-key-secret"})
	SetRazorpayClientFor(models.ChefModeTest, &RazorpayClient{keySecret: "test-key-secret"})

	oid, pid := "order_LIVE1", "pay_LIVE1"
	body := []byte(oid + "|" + pid)

	if !VerifyPaymentSignatureFor(models.ChefModeTest, oid, pid, sign("test-key-secret", body)) {
		t.Fatal("a test-mode payment must verify against the test key secret")
	}
	if VerifyPaymentSignatureFor(models.ChefModeLive, oid, pid, sign("test-key-secret", body)) {
		t.Fatal("a test-signed payment must NOT verify against live credentials")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./services/ -run 'TestWebhookSignatureIdentifies|TestIdenticalSecrets|TestPaymentSignatureIsModeScoped' -v`
Expected: FAIL — `undefined: VerifyWebhookSignatureMode`.

- [ ] **Step 3: Implement in `services/razorpay.go`**

```go
// VerifyWebhookSignatureMode verifies a webhook against both credential slots
// and reports which one signed it.
//
// Razorpay sends test-dashboard and live-dashboard webhooks to the same URL,
// each signed with its own secret, and the payload carries no mode marker. So
// we try live first, then test. The returned mode is a HINT, not an authority:
// when both slots hold the same key (the interim state while a real live key is
// pending) every event resolves to live. The caller MUST additionally compare
// this against the target record's own mode and reject a mismatch — that
// comparison, not this function, is what keeps the two worlds apart.
func VerifyWebhookSignatureMode(payload []byte, signature string) (bool, string) {
	for _, mode := range []string{models.ChefModeLive, models.ChefModeTest} {
		c := snapshotRazorpayClient(mode)
		if c == nil || c.webhookSecret == "" {
			continue
		}
		mac := hmac.New(sha256.New, []byte(c.webhookSecret))
		mac.Write(payload)
		if hmac.Equal([]byte(signature), []byte(hex.EncodeToString(mac.Sum(nil)))) {
			return true, mode
		}
	}
	return false, ""
}

// VerifyWebhookSignature is the live-only shim retained for callers that have
// no record context.
func VerifyWebhookSignature(payload []byte, signature string) bool {
	ok, _ := VerifyWebhookSignatureMode(payload, signature)
	return ok
}

// VerifyPaymentSignatureFor verifies the checkout callback signature against a
// specific mode's key secret. Always called with the ORDER's mode, never the
// chef's current mode.
func VerifyPaymentSignatureFor(mode, razorpayOrderID, razorpayPaymentID, signature string) bool {
	c := snapshotRazorpayClient(models.NormalizeMode(mode))
	if c == nil || c.keySecret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(c.keySecret))
	mac.Write([]byte(razorpayOrderID + "|" + razorpayPaymentID))
	return hmac.Equal([]byte(signature), []byte(hex.EncodeToString(mac.Sum(nil))))
}
```

Keep the existing `VerifyPaymentSignature` as `VerifyPaymentSignatureFor(models.ChefModeLive, …)`.

- [ ] **Step 4: Enforce mode agreement in the webhook handler**

In `apps/api/handlers/payment.go`, in the Razorpay webhook handler: call `VerifyWebhookSignatureMode`, resolve the local record from the event's order/payment id, and before applying any mutation:

```go
	// A live-signed webhook must never mutate a test-partition record, and a
	// test-signed one must never mutate a live record. With both slots holding
	// the same key during the interim period this check is the ONLY thing
	// keeping the two worlds apart, so it is not merely defensive.
	if models.NormalizeMode(order.Mode) != models.NormalizeMode(signedMode) {
		log.Printf("razorpay-webhook: mode mismatch — signed=%s record=%s order=%s; dropping",
			signedMode, order.Mode, order.ID)
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}
```

Respond 200 rather than an error so Razorpay does not retry an event that will never apply.

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ ./handlers/ -run 'Signature|Webhook' -v && go build ./...`
Expected: PASS. Existing `services/razorpay_webhook_test.go` and `handlers/payment_webhook_*_test.go` must still pass unchanged.

- [ ] **Step 6: Commit**

```bash
git add apps/api/services/razorpay.go apps/api/services/razorpay_signature_mode_test.go apps/api/handlers/payment.go
git commit -m "feat(test-mode): verify razorpay signatures per credential slot and reject mode mismatches"
```

---

## Task 7: Payment mode resolution and stamping

**Files:**
- Create: `apps/api/services/payment_mode.go`
- Create: `apps/api/services/payment_mode_test.go`
- Modify: `apps/api/handlers/orders.go` (CreateOrder), `handlers/payment.go`, `handlers/tips.go`, `handlers/group_order.go`, `handlers/meal_plan.go`, `handlers/catering.go`, `handlers/promotion.go`, `services/meal_plan_escrow.go`, `services/orderrefund_gateway.go`, `services/payout_release.go`, `handlers/chef_order_cancel.go`

**Interfaces:**
- Consumes: `models.NormalizeMode` (Task 1), `GetRazorpayFor` (Task 5)
- Produces: `services.PaymentModeForChef(chefID uuid.UUID) string`

- [ ] **Step 1: Write the failing test**

`apps/api/services/payment_mode_test.go` — use the sqlite harness shape from `services/account_lifecycle_test.go:25-60`:

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupPaymentModeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, user_id TEXT,
		mode TEXT DEFAULT 'live', first_live_at DATETIME, active_test_session_id TEXT,
		is_verified BOOLEAN DEFAULT 0, is_active BOOLEAN DEFAULT 1,
		created_at DATETIME, updated_at DATETIME)`).Error)
	old := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = old })
	return db
}

func TestPaymentModeForChef(t *testing.T) {
	db := setupPaymentModeDB(t)
	live, test := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?,?),(?,?)`,
		live.String(), "live", test.String(), "test").Error)

	if got := PaymentModeForChef(live); got != models.ChefModeLive {
		t.Fatalf("live chef = %q", got)
	}
	if got := PaymentModeForChef(test); got != models.ChefModeTest {
		t.Fatalf("test chef = %q", got)
	}
}

// A missing chef, a DB error, or a garbage stored value must all resolve to
// live. Resolving to test would route a real payment through sandbox
// credentials and silently capture nothing.
func TestPaymentModeFailsSafeToLive(t *testing.T) {
	db := setupPaymentModeDB(t)
	garbage := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?,?)`,
		garbage.String(), "sandbox").Error)

	if got := PaymentModeForChef(uuid.New()); got != models.ChefModeLive {
		t.Fatalf("missing chef = %q, want live", got)
	}
	if got := PaymentModeForChef(garbage); got != models.ChefModeLive {
		t.Fatalf("garbage mode = %q, want live", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./services/ -run 'TestPaymentMode' -v`
Expected: FAIL — `undefined: PaymentModeForChef`.

- [ ] **Step 3: Write the implementation**

`apps/api/services/payment_mode.go`:

```go
package services

import (
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// PaymentModeForChef resolves which world a NEW record for this chef belongs
// to. This is the single place chef.Mode is read for money purposes; every
// operation on an already-created record reads that record's own mode instead.
//
// Fails safe to live on any error. A wrong "live" answer produces a loud
// gateway failure; a wrong "test" answer produces a real order that quietly
// took no money, which is far worse.
func PaymentModeForChef(chefID uuid.UUID) string {
	var mode string
	err := database.DB.Model(&models.ChefProfile{}).
		Where("id = ?", chefID).Select("mode").Scan(&mode).Error
	if err != nil {
		log.Printf("payment-mode: chef %s lookup failed (%v) — defaulting to live", chefID, err)
		return models.ChefModeLive
	}
	return models.NormalizeMode(mode)
}
```

- [ ] **Step 4: Stamp mode at every record creation**

For each create path, resolve the mode once and set it on the record before insert. In `handlers/orders.go` `CreateOrder`, after the chef is loaded and validated:

```go
	// Snapshot the chef's mode onto the order. From here on, every gateway
	// operation for this order reads order.Mode — never the chef's current
	// mode, which an admin may flip at any time.
	order.Mode = services.PaymentModeForChef(chef.ID)
	if chef.ActiveTestSessionID != nil && models.IsTestMode(order.Mode) {
		order.TestSessionID = chef.ActiveTestSessionID
	}
```

Repeat the same two-line pattern for `GroupOrder`, `MealPlan`, `MealSubscription`, `MealTrial`, `CateringRequest`, `Tip`, `ChefPromotion` and `Review` at their creation sites.

- [ ] **Step 5: Route every chef-scoped gateway call through the record's mode**

Replace `services.GetRazorpay()` with `services.GetRazorpayFor(<record>.Mode)` at each of these sites, and `services.VerifyPaymentSignature(...)` with `services.VerifyPaymentSignatureFor(<record>.Mode, ...)`:

| File | What to thread |
|---|---|
| `handlers/payment.go:131,226,360,730,1301` | `order.Mode` |
| `handlers/chef_order_cancel.go:181,412,589` | `order.Mode` |
| `handlers/tips.go:86,153,192` | `tip.Mode` |
| `handlers/group_order.go:650,670,701` | `groupOrder.Mode` |
| `handlers/meal_plan.go:474,497` | `plan.Mode` |
| `handlers/catering.go:742,760,803` | `req.Mode` |
| `handlers/promotion.go:82,130,175` | `promotion.Mode` |
| `handlers/chefs.go:2285` | `chef.Mode` (linked-account creation) |
| `services/meal_plan_escrow.go:147,174,211,280,408` | the plan's `Mode` |
| `services/orderrefund_gateway.go` | the order's `Mode` |
| `services/payout_release.go` | the order's `Mode` |

Leave `handlers/admin.go:1007,1023,1117` and `services/reconciliation.go:130` explicitly on the live slot — they are platform-level, not chef-scoped, and Task 15 gives the admin ones a mode parameter.

Line numbers are from the pre-change file and will drift; re-grep with `grep -rn "GetRazorpay()" --include="*.go" apps/api` and work the list to zero for chef-scoped files.

- [ ] **Step 6: Run the full suite**

Run: `cd apps/api && go build ./... && go vet ./... && go test ./... 2>&1 | tail -30`
Expected: PASS. Any failing money test here is a real regression — do not adjust the test to fit, fix the threading.

- [ ] **Step 7: Commit**

```bash
git add apps/api/
git commit -m "feat(test-mode): snapshot payment mode on records and route gateway calls by it"
```

---

## Task 8: Chef visibility scope

**Files:**
- Create: `apps/api/services/test_mode_visibility.go`
- Create: `apps/api/services/test_mode_visibility_test.go`

**Interfaces:**
- Consumes: `GetTestModePolicy` (Task 4), `models.IsTestMode` (Task 1)
- Produces: `services.ChefVisibilityDecision` (`VisibilityFull`, `VisibilityClosed`, `VisibilityHidden`), `services.ChefVisibility(chef *models.ChefProfile, viewerEmail string) ChefVisibilityDecision`, `services.TestChefVisibility(viewerEmail string) func(*gorm.DB) *gorm.DB`, `services.ExcludeTestOrders(*gorm.DB) *gorm.DB`, `services.ModeScope(mode string) func(*gorm.DB) *gorm.DB`

- [ ] **Step 1: Write the failing test**

`apps/api/services/test_mode_visibility_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/homechef/api/models"
)

func TestChefVisibilityMatrix(t *testing.T) {
	now := time.Now()
	live := &models.ChefProfile{Mode: models.ChefModeLive}
	bornTest := &models.ChefProfile{Mode: models.ChefModeTest}
	flipped := &models.ChefProfile{Mode: models.ChefModeTest, FirstLiveAt: &now}

	cases := []struct {
		name  string
		chef  *models.ChefProfile
		email string
		want  ChefVisibilityDecision
	}{
		{"live chef, anonymous", live, "", VisibilityFull},
		{"live chef, tester", live, "samyak.rout@gmail.com", VisibilityFull},
		// A kitchen nobody has heard of should not exist for anyone but a tester.
		{"born-test, anonymous", bornTest, "", VisibilityHidden},
		{"born-test, stranger", bornTest, "someone@else.com", VisibilityHidden},
		{"born-test, tester", bornTest, "samyak.rout@gmail.com", VisibilityFull},
		// A kitchen with regulars should not vanish — it should look closed.
		{"flipped, anonymous", flipped, "", VisibilityClosed},
		{"flipped, stranger", flipped, "someone@else.com", VisibilityClosed},
		{"flipped, tester", flipped, "unidevidp@gmail.com", VisibilityFull},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ChefVisibility(tc.chef, tc.email); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
```

This test reads the policy through `GetTestModePolicy()`, which falls back to `DefaultTestModePolicy()` when `database.DB` has no row — so it needs no DB fixture. Confirm that holds; if `database.DB` is nil in this package's tests, add `t.Cleanup(InvalidateTestModePolicy)` and stub the policy via `SaveTestModePolicy` against a sqlite fixture instead.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./services/ -run TestChefVisibilityMatrix -v`
Expected: FAIL — `undefined: ChefVisibility`.

- [ ] **Step 3: Write the implementation**

`apps/api/services/test_mode_visibility.go`:

```go
package services

import (
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ChefVisibilityDecision is how much of a kitchen a given viewer may see.
type ChefVisibilityDecision int

const (
	// VisibilityFull — the kitchen renders normally: listed, browsable,
	// orderable (subject to the usual open/closed and FSSAI rules).
	VisibilityFull ChefVisibilityDecision = iota

	// VisibilityClosed — the kitchen is listed by name and shown as closed, with
	// no menu, no prices and no ordering. Used for an established kitchen that
	// has been flipped to test for debugging: its regulars would read a sudden
	// disappearance as "they shut down", which is worse than "closed today".
	VisibilityClosed

	// VisibilityHidden — the kitchen does not exist for this viewer. Used for
	// born-test kitchens, which no real customer has ever heard of.
	VisibilityHidden
)

// ChefVisibility decides how much of a kitchen a viewer may see. viewerEmail is
// empty for anonymous callers.
func ChefVisibility(chef *models.ChefProfile, viewerEmail string) ChefVisibilityDecision {
	if chef == nil || !chef.IsTestMode() {
		return VisibilityFull
	}
	if GetTestModePolicy().MayViewTestChefs(viewerEmail) {
		return VisibilityFull
	}
	if chef.IsBornTest() {
		return VisibilityHidden
	}
	return VisibilityClosed
}

// TestChefVisibility is the set-based mirror of ChefVisibility for
// chef_profiles list queries — the same rule, expressed in SQL so pagination
// and counts stay correct (no post-filtering a page down to three rows).
//
// Allowlisted viewers get no filter at all. Everyone else has born-test
// kitchens removed in SQL; flipped kitchens survive the query and are reduced
// to their closed presentation by the response mapper, which is where
// per-row policy belongs.
func TestChefVisibility(viewerEmail string) func(*gorm.DB) *gorm.DB {
	allowed := GetTestModePolicy().MayViewTestChefs(viewerEmail)
	return func(db *gorm.DB) *gorm.DB {
		if allowed {
			return db
		}
		return db.Where("NOT (mode = ? AND first_live_at IS NULL)", models.ChefModeTest)
	}
}

// ModeScope filters any partitioned table to one mode.
func ModeScope(mode string) func(*gorm.DB) *gorm.DB {
	m := models.NormalizeMode(mode)
	return func(db *gorm.DB) *gorm.DB { return db.Where("mode = ?", m) }
}

// ExcludeTestOrders removes test-partition rows. Applied to every real-money
// and reporting path: revenue analytics, chef earnings, GST/TDS, reconciliation
// and the ledger. A fake order must never move a real number.
func ExcludeTestOrders(db *gorm.DB) *gorm.DB {
	return db.Where("mode = ?", models.ChefModeLive)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run TestChefVisibilityMatrix -v`
Expected: PASS, all nine sub-cases.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/test_mode_visibility.go apps/api/services/test_mode_visibility_test.go
git commit -m "feat(test-mode): add chef visibility decision and mode scopes"
```

---

## Task 9: Apply visibility to customer-facing chef endpoints

**Files:**
- Modify: `apps/api/handlers/chefs.go` (`ListChefs:89-96`, `SearchDishes:321`, `GetChef:362`, `GetChefMenu:434`, `GetChefReviews:496`)
- Modify: `apps/api/handlers/weekly_menu.go`, `daily_menu.go`, `meal_subscription.go` (`GetChefOffer`), `handlers/orders.go` (`QuoteDeliveryFee`)
- Create: `apps/api/handlers/test_mode_visibility_test.go`

**Interfaces:**
- Consumes: `services.ChefVisibility`, `services.TestChefVisibility` (Task 8)
- Produces: `handlers.viewerEmail(c *gin.Context) string`, `handlers.guardChefVisibility(c *gin.Context, chef *models.ChefProfile) (proceed bool, reduced bool)`

- [ ] **Step 1: Write the failing test**

`apps/api/handlers/test_mode_visibility_test.go`:

```go
package handlers

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/models"
)

func ctxWithEmail(email string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if email != "" {
		c.Set("userEmail", email)
	}
	return c
}

func TestViewerEmailReadsBothContextKeys(t *testing.T) {
	if got := viewerEmail(ctxWithEmail("a@b.com")); got != "a@b.com" {
		t.Fatalf("got %q", got)
	}
	if got := viewerEmail(ctxWithEmail("")); got != "" {
		t.Fatalf("anonymous must yield empty, got %q", got)
	}
}

func TestGuardChefVisibility(t *testing.T) {
	now := time.Now()
	born := &models.ChefProfile{Mode: models.ChefModeTest}
	flipped := &models.ChefProfile{Mode: models.ChefModeTest, FirstLiveAt: &now}
	live := &models.ChefProfile{Mode: models.ChefModeLive}

	if proceed, _ := guardChefVisibility(ctxWithEmail("stranger@x.com"), born); proceed {
		t.Fatal("a born-test kitchen must 404 for a stranger")
	}
	proceed, reduced := guardChefVisibility(ctxWithEmail("stranger@x.com"), flipped)
	if !proceed || !reduced {
		t.Fatalf("a flipped kitchen must proceed reduced, got proceed=%v reduced=%v", proceed, reduced)
	}
	proceed, reduced = guardChefVisibility(ctxWithEmail("samyak.rout@gmail.com"), flipped)
	if !proceed || reduced {
		t.Fatal("a tester must get the full kitchen")
	}
	if proceed, reduced := guardChefVisibility(ctxWithEmail(""), live); !proceed || reduced {
		t.Fatal("a live kitchen is always full, even anonymously")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./handlers/ -run 'TestViewerEmail|TestGuardChefVisibility' -v`
Expected: FAIL — `undefined: viewerEmail`.

- [ ] **Step 3: Write the helpers**

Add to `apps/api/handlers/chefs.go` (near `resolveChefID`):

```go
// viewerEmail returns the calling customer's email, or "" when anonymous.
// The /chefs group runs bffAuthOptional, which populates this when a session is
// present without making the routes authenticated.
func viewerEmail(c *gin.Context) string {
	if v, ok := c.Get("userEmail"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// guardChefVisibility applies the test-mode visibility rule to a single chef.
//
// Returns proceed=false when the handler must 404 — deliberately 404 and not
// 403, so a shared link to a sandbox kitchen discloses nothing about whether it
// exists. Returns reduced=true when the handler must serve the closed
// presentation: name and photo only, no menu, no prices, not orderable.
func guardChefVisibility(c *gin.Context, chef *models.ChefProfile) (proceed bool, reduced bool) {
	switch services.ChefVisibility(chef, viewerEmail(c)) {
	case services.VisibilityHidden:
		return false, false
	case services.VisibilityClosed:
		return true, true
	default:
		return true, false
	}
}
```

- [ ] **Step 4: Apply to the list queries**

In `ListChefs`, extend the existing scope chain at line 96:

```go
		Scopes(
			services.ExcludeFSSAILocked,               // FSSAI lockout (#91)
			services.TestChefVisibility(viewerEmail(c)), // test-mode kitchens
		)
```

Do the same at `SearchDishes:321`. In the dietary sub-query at `ListChefs:130`, add `AND chef_id NOT IN (SELECT id FROM chef_profiles WHERE mode = 'test' AND first_live_at IS NULL)` for non-allowlisted viewers.

In the response mapper for each listed chef, when `services.ChefVisibility(&chef, viewerEmail(c)) == services.VisibilityClosed`, force `acceptingOrders: false`, clear `pausedUntil`, and omit menu/price/rating fields from the payload. Do this in the mapper — never by mutating the loaded model, which would risk a write-back.

- [ ] **Step 5: Apply to the single-chef routes**

In each of `GetChef`, `GetChefMenu`, `GetChefReviews`, `GetPublicWeeklyMenu`, `GetPublicDailyMenu`, `GetChefOffer`, `GetChefDeliverySlots`, `GetChefFulfillmentTimes`, `QuoteDeliveryFee` — immediately after the chef is loaded:

```go
	proceed, reduced := guardChefVisibility(c, &chef)
	if !proceed {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
```

`GetChef` serves the reduced payload when `reduced`. Every other endpoint in that list is menu/price/ordering data and returns **404 when `reduced`** — the closed presentation exposes the name only.

- [ ] **Step 6: Run tests and build**

Run: `cd apps/api && go test ./handlers/ -v 2>&1 | tail -20 && go build ./...`
Expected: PASS, clean.

- [ ] **Step 7: Commit**

```bash
git add apps/api/handlers/
git commit -m "feat(test-mode): hide test kitchens from customer discovery surfaces"
```

---

## Task 10: Ordering and order-history guards

**Files:**
- Modify: `apps/api/handlers/orders.go` (`CreateOrder`), `group_order.go`, `meal_plan.go`, `meal_subscription.go`, `catering.go` (create paths); customer order-list queries in `handlers/orders.go` and `handlers/customer.go`
- Create: `apps/api/handlers/test_mode_order_guard_test.go`

**Interfaces:**
- Consumes: `services.ChefVisibility` (Task 8), `viewerEmail` (Task 9)
- Produces: `handlers.assertMayOrderFromChef(c *gin.Context, chef *models.ChefProfile) error`

- [ ] **Step 1: Write the failing test**

```go
package handlers

import (
	"testing"
	"time"

	"github.com/homechef/api/models"
)

func TestAssertMayOrderFromChef(t *testing.T) {
	now := time.Now()
	born := &models.ChefProfile{Mode: models.ChefModeTest}
	flipped := &models.ChefProfile{Mode: models.ChefModeTest, FirstLiveAt: &now}
	live := &models.ChefProfile{Mode: models.ChefModeLive}

	if err := assertMayOrderFromChef(ctxWithEmail("stranger@x.com"), born); err == nil {
		t.Fatal("a stranger must not be able to order from a born-test kitchen")
	}
	if err := assertMayOrderFromChef(ctxWithEmail("stranger@x.com"), flipped); err == nil {
		t.Fatal("a stranger must not be able to order from a kitchen flipped to test")
	}
	if err := assertMayOrderFromChef(ctxWithEmail(""), born); err == nil {
		t.Fatal("an anonymous caller must never reach a test kitchen")
	}
	if err := assertMayOrderFromChef(ctxWithEmail("samyak.rout@gmail.com"), born); err != nil {
		t.Fatalf("a tester must be able to order from a test kitchen: %v", err)
	}
	// The allowlist grants visibility of fake kitchens, not discounts on real
	// ones. A tester ordering from a live chef is an ordinary paying customer.
	if err := assertMayOrderFromChef(ctxWithEmail("samyak.rout@gmail.com"), live); err != nil {
		t.Fatalf("a tester ordering from a live kitchen must be unaffected: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./handlers/ -run TestAssertMayOrderFromChef -v`
Expected: FAIL — `undefined: assertMayOrderFromChef`.

- [ ] **Step 3: Write the implementation**

In `apps/api/handlers/orders.go`:

```go
// errTestChefNotOrderable is returned when a customer who is not on the
// test-mode viewer allowlist tries to transact with a sandbox kitchen.
var errTestChefNotOrderable = errors.New("this kitchen is not accepting orders")

// assertMayOrderFromChef is the last line of defence for test-mode isolation.
//
// Discovery filtering (Task 9) is what normally keeps a sandbox kitchen out of
// sight, but discovery is spread across nine endpoints and will grow. This
// check sits on the four create paths instead, where the number of doors is
// small and fixed, so a missed discovery surface can never become a real
// customer transacting with a fake kitchen.
func assertMayOrderFromChef(c *gin.Context, chef *models.ChefProfile) error {
	if services.ChefVisibility(chef, viewerEmail(c)) != services.VisibilityFull {
		return errTestChefNotOrderable
	}
	return nil
}
```

Call it in `CreateOrder` right after the chef is loaded, and on the group-order, meal-plan, meal-subscription and catering create paths:

```go
	if err := assertMayOrderFromChef(c, &chef); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
```

- [ ] **Step 4: Filter customer order history**

Every customer-facing order/meal-plan/group-order list query gains:

```go
	// Customers see live orders, plus test orders they placed themselves while
	// on the allowlist. Cloned rows are NEVER shown to any customer: a clone
	// replicates a real customer's order into the sandbox, and that customer
	// never placed it there.
	q = q.Where("cloned_from_id IS NULL")
	if !services.GetTestModePolicy().MayViewTestChefs(viewerEmail(c)) {
		q = q.Where("mode = ?", models.ChefModeLive)
	}
```

Apply to the customer order list, order detail, reorder source query, and the meal-plan and group-order customer lists.

- [ ] **Step 5: Run tests and build**

Run: `cd apps/api && go test ./handlers/ -v 2>&1 | tail -20 && go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/handlers/
git commit -m "feat(test-mode): block non-allowlisted customers from test kitchens and hide cloned rows"
```

---

## Task 11: Blast-radius guards — dispatch, payouts, wallet, loyalty, reporting

**Files:**
- Modify: `apps/api/services/provider_dispatch.go`, `payout_release.go`, `payout_automation.go`, `payout_mealplan_release_cron.go`, `wallet.go`, `wallet_split.go`, `loyalty.go`, `loyalty_order_redeem.go`, `referral_reward.go`, `ledger.go`, `statement.go`, `reconciliation.go`, `gst.go`, `earnings.go`
- Modify: `apps/api/handlers/chef_analytics.go`, `admin.go` (revenue analytics)
- Create: `apps/api/services/test_mode_blast_radius_test.go`

**Interfaces:**
- Consumes: `models.IsTestMode` (Task 1), `services.ExcludeTestOrders` (Task 8)
- Produces: `services.IsTestOrder(o *models.Order) bool`, `services.ShouldDispatchToProvider(o *models.Order) bool`, `services.WalletAllowedForOrder(o *models.Order) bool`, `services.LoyaltyAllowedForOrder(o *models.Order) bool`, `services.PayoutAllowedForOrder(o *models.Order) bool`

- [ ] **Step 1: Write the failing test**

```go
package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestIsTestOrder(t *testing.T) {
	if !IsTestOrder(&models.Order{Mode: models.ChefModeTest}) {
		t.Fatal("a test-mode order is a test order")
	}
	for _, m := range []string{"", "live", "garbage"} {
		if IsTestOrder(&models.Order{Mode: m}) {
			t.Fatalf("mode %q must not read as a test order", m)
		}
	}
	if IsTestOrder(nil) {
		t.Fatal("a nil order must not read as a test order")
	}
}
```

Then one guard test per bypass, each asserting the *refusal*, not the happy path. For example:

```go
// A real courier must never be dispatched to a fake order — a rider would be
// sent to a real address for food nobody is cooking.
func TestTestOrderNeverDispatchesToProvider(t *testing.T) {
	o := &models.Order{Mode: models.ChefModeTest}
	if ShouldDispatchToProvider(o) {
		t.Fatal("a test order must never reach a 3PL provider")
	}
	if !ShouldDispatchToProvider(&models.Order{Mode: models.ChefModeLive}) {
		t.Fatal("a live order must still dispatch normally")
	}
}

// Sandbox money must not become spendable balance. A test-order refund
// crediting the real wallet would mint value out of nothing.
func TestTestOrderCannotUseWallet(t *testing.T) {
	if WalletAllowedForOrder(&models.Order{Mode: models.ChefModeTest}) {
		t.Fatal("a test order must not use wallet as a payment source or refund destination")
	}
}

// Points earned on a fake order would be redeemable against a real kitchen.
func TestTestOrderEarnsNoLoyalty(t *testing.T) {
	if LoyaltyAllowedForOrder(&models.Order{Mode: models.ChefModeTest}) {
		t.Fatal("a test order must neither earn nor redeem loyalty points")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/api && go test ./services/ -run 'TestIsTestOrder|TestTestOrder' -v`
Expected: FAIL — undefined predicates.

- [ ] **Step 3: Write the predicates**

Add to `apps/api/services/payment_mode.go`:

```go
// IsTestOrder reports whether an order belongs to the test partition.
func IsTestOrder(o *models.Order) bool {
	return o != nil && models.IsTestMode(o.Mode)
}

// ShouldDispatchToProvider reports whether an order may be handed to an
// external 3PL courier. Test orders never may: a real rider would be sent to a
// real address for an order nobody is cooking. Pickup and chef self-delivery
// still work, and our own platform drivers can still be assigned, so the driver
// flow stays fully testable.
func ShouldDispatchToProvider(o *models.Order) bool { return !IsTestOrder(o) }

// WalletAllowedForOrder reports whether wallet may be used as a payment source
// or refund destination. Never for a test order: a refund into the real wallet
// would mint spendable balance from sandbox money.
func WalletAllowedForOrder(o *models.Order) bool { return !IsTestOrder(o) }

// LoyaltyAllowedForOrder reports whether an order participates in loyalty.
// Never for a test order: points earned on a fake order would be redeemable
// against a real kitchen. Referral rewards and promo-usage counters follow the
// same rule.
func LoyaltyAllowedForOrder(o *models.Order) bool { return !IsTestOrder(o) }

// PayoutAllowedForOrder reports whether an order enters the real payout and
// settlement engine. Test orders do not — their Route transfers are created and
// released inside the Razorpay TEST account, so the split-payment path is
// genuinely exercised without anything reaching a real bank.
func PayoutAllowedForOrder(o *models.Order) bool { return !IsTestOrder(o) }
```

- [ ] **Step 4: Wire the guards in**

- `provider_dispatch.go` — early-return at the dispatch entry point when `!ShouldDispatchToProvider(order)`, logging the skip.
- `payout_release.go`, `payout_automation.go`, `payout_mealplan_release_cron.go`, `statement.go` — skip orders where `!PayoutAllowedForOrder(order)`; for the cron selection queries, add `.Scopes(services.ExcludeTestOrders)` so test rows are never even fetched.
- `wallet.go`, `wallet_split.go` — reject with a clear error when `!WalletAllowedForOrder(order)`.
- `loyalty.go`, `loyalty_order_redeem.go`, `referral_reward.go` — no-op when `!LoyaltyAllowedForOrder(order)`.
- `ledger.go`, `reconciliation.go`, `gst.go`, `earnings.go`, `handlers/chef_analytics.go`, `handlers/admin.go` revenue queries — add `.Scopes(services.ExcludeTestOrders)`.

- [ ] **Step 5: Run the full suite**

Run: `cd apps/api && go test ./... 2>&1 | tail -30 && go build ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/
git commit -m "feat(test-mode): keep test orders out of couriers, payouts, wallet, loyalty and reporting"
```

---

## Task 12: `[TEST]` notification tagging

**Files:**
- Modify: `apps/api/services/notifications.go`, `push.go`, `email_templates.go`
- Create: `apps/api/services/test_mode_notifications_test.go`

**Interfaces:**
- Consumes: `models.IsTestMode` (Task 1)
- Produces: `services.TagSubjectForMode(mode, subject string) string`

- [ ] **Step 1: Write the failing test**

```go
package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestTagSubjectForMode(t *testing.T) {
	if got := TagSubjectForMode(models.ChefModeTest, "Your order is on the way"); got != "[TEST] Your order is on the way" {
		t.Fatalf("got %q", got)
	}
	if got := TagSubjectForMode(models.ChefModeLive, "Your order is on the way"); got != "Your order is on the way" {
		t.Fatalf("a live notification must be untouched, got %q", got)
	}
	// Re-tagging must be idempotent — notification paths compose subjects from
	// several helpers and a doubled prefix looks broken.
	if got := TagSubjectForMode(models.ChefModeTest, "[TEST] Already tagged"); got != "[TEST] Already tagged" {
		t.Fatalf("tagging must be idempotent, got %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && go test ./services/ -run TestTagSubjectForMode -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

```go
// testNotificationPrefix marks a notification as belonging to a sandbox order.
const testNotificationPrefix = "[TEST] "

// TagSubjectForMode prefixes an email subject or push title for test-mode
// notifications. Test notifications are deliberately NOT suppressed: their
// recipients are structurally limited to the viewer allowlist plus the test
// chef and assigned driver, and firing them is the only way to verify the FCM
// and Temporal dispatch path on production. Idempotent.
func TagSubjectForMode(mode, subject string) string {
	if !models.IsTestMode(mode) || strings.HasPrefix(subject, testNotificationPrefix) {
		return subject
	}
	return testNotificationPrefix + subject
}
```

- [ ] **Step 4: Wire into every order-scoped notification**

At each order/meal-plan/group-order notification send site, wrap the subject and push title: `TagSubjectForMode(order.Mode, subject)`. Find them with `grep -rn "func Send.*Push\|func Send.*Email" services/notifications.go services/push.go`.

Cloned rows emit nothing at all — the clone inserts directly and never calls these paths (Task 15 asserts this).

- [ ] **Step 5: Run tests and build**

Run: `cd apps/api && go test ./services/ -run 'Notification|TagSubject' -v && go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/services/
git commit -m "feat(test-mode): prefix test-order notifications with [TEST]"
```

---

## Task 13: Per-mode chef stats

**Files:**
- Create: `apps/api/services/chef_mode_stats.go`, `apps/api/services/chef_mode_stats_test.go`
- Modify: `apps/api/handlers/reviews.go`, `orders.go` (wherever `chef_profiles.rating` / `total_orders` / `total_reviews` / `issue_count` is updated), `handlers/chefs.go` (`GetChefDashboard`)

**Interfaces:**
- Consumes: `models.ChefModeStats` (Task 2)
- Produces: `services.ApplyChefStatDelta(db *gorm.DB, chefID uuid.UUID, mode string, d ChefStatDelta) error`, `services.GetChefModeStats(db, chefID, mode) (models.ChefModeStats, error)`, `services.ChefStatDelta{OrdersDelta, ReviewsDelta, IssuesDelta int; NewRating *float64}`

- [ ] **Step 1: Write the failing test**

```go
// A fake order must never move a real rating. chef_profiles is the
// customer-facing source of truth and only the live partition may write it.
func TestTestStatsNeverTouchChefProfile(t *testing.T) {
	db := setupStatsDB(t)
	chefID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, rating, total_orders, total_reviews) VALUES (?,?,?,?)`,
		chefID.String(), 4.8, 100, 40).Error)

	four := 1.0
	require.NoError(t, ApplyChefStatDelta(db, chefID, models.ChefModeTest,
		ChefStatDelta{OrdersDelta: 5, ReviewsDelta: 3, NewRating: &four}))

	var rating float64
	var orders int
	require.NoError(t, db.Raw(`SELECT rating, total_orders FROM chef_profiles WHERE id = ?`,
		chefID.String()).Row().Scan(&rating, &orders))
	if rating != 4.8 || orders != 100 {
		t.Fatalf("chef_profiles was mutated by test activity: rating=%v orders=%v", rating, orders)
	}

	s, err := GetChefModeStats(db, chefID, models.ChefModeTest)
	require.NoError(t, err)
	if s.TotalOrders != 5 || s.TotalReviews != 3 || s.Rating != 1.0 {
		t.Fatalf("test stats not recorded: %+v", s)
	}
}

// The live partition must stay mirrored into chef_profiles so the two never
// drift and no customer-facing query has to change.
func TestLiveStatsMirrorIntoChefProfile(t *testing.T) {
	db := setupStatsDB(t)
	chefID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, rating, total_orders, total_reviews) VALUES (?,?,?,?)`,
		chefID.String(), 4.8, 100, 40).Error)

	r := 4.9
	require.NoError(t, ApplyChefStatDelta(db, chefID, models.ChefModeLive,
		ChefStatDelta{OrdersDelta: 1, NewRating: &r}))

	var rating float64
	var orders int
	require.NoError(t, db.Raw(`SELECT rating, total_orders FROM chef_profiles WHERE id = ?`,
		chefID.String()).Row().Scan(&rating, &orders))
	if rating != 4.9 || orders != 101 {
		t.Fatalf("live stats did not mirror: rating=%v orders=%v", rating, orders)
	}
}
```

Write `setupStatsDB` with `chef_profiles` and `chef_mode_stats` fixtures following the harness pattern in Task 7 Step 1.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/api && go test ./services/ -run 'TestTestStatsNever|TestLiveStatsMirror' -v`
Expected: FAIL.

- [ ] **Step 3: Implement `services/chef_mode_stats.go`**

`ApplyChefStatDelta` upserts the `(chef_id, mode)` row inside a transaction, applying the integer deltas and setting `Rating` when `NewRating != nil`. When `mode` normalises to live it additionally mirrors the same values into `chef_profiles`, so every existing customer-facing read is unchanged and the two can never drift. When mode is test it touches `chef_profiles` under no circumstances.

- [ ] **Step 4: Route existing aggregate writes through it**

Find every direct write with:

```bash
cd apps/api && grep -rn "total_orders\|total_reviews\|issue_count\|\"rating\"" handlers/ services/ | grep -v _test | grep -i "update\|increment"
```

Replace each with an `ApplyChefStatDelta` call carrying the record's mode. `GetChefDashboard` and the vendor stats endpoints read `GetChefModeStats(db, chefID, chef.Mode)`.

- [ ] **Step 5: Run tests and build**

Run: `cd apps/api && go test ./services/ ./handlers/ 2>&1 | tail -20 && go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/services/chef_mode_stats.go apps/api/services/chef_mode_stats_test.go apps/api/handlers/
git commit -m "feat(test-mode): keep chef aggregates per mode so fake orders never move a real rating"
```

---

## Task 14: Test sessions — open, close, flip blockers

**Files:**
- Create: `apps/api/services/test_session.go`, `apps/api/services/test_session_test.go`

**Interfaces:**
- Consumes: `models.ChefTestSession` (Task 2)
- Produces: `services.TestFlipBlockers(db *gorm.DB, chefID uuid.UUID) []string`, `services.OpenTestSession(db *gorm.DB, chefID, adminID uuid.UUID, reason string, windowDays int) (*models.ChefTestSession, error)`, `services.CloseTestSession(db *gorm.DB, chefID, adminID uuid.UUID) error`, `services.ErrFlipBlocked`

- [ ] **Step 1: Write the failing test**

```go
// Flipping a kitchen mid-dinner strands real customers holding an order they
// can no longer see. The blocker list is what the admin UI shows, so it must
// name each obstruction rather than returning a bare boolean.
func TestFlipBlockedWhileOrdersInFlight(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedLiveChef(t, db)
	seedActiveOrder(t, db, chefID)

	blockers := TestFlipBlockers(db, chefID)
	if len(blockers) == 0 {
		t.Fatal("an active order must block a live→test flip")
	}
	if _, err := OpenTestSession(db, chefID, uuid.New(), "debugging", 30); !errors.Is(err, ErrFlipBlocked) {
		t.Fatalf("open must refuse with ErrFlipBlocked, got %v", err)
	}
}

func TestFlipAllowedOnceSettled(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedLiveChef(t, db)

	s, err := OpenTestSession(db, chefID, uuid.New(), "reproducing #123", 30)
	require.NoError(t, err)
	if s.SessionNo != 1 || s.Status != models.TestSessionOpen {
		t.Fatalf("first session must be no=1 open, got %+v", s)
	}

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	if !chef.IsTestMode() || chef.ActiveTestSessionID == nil || *chef.ActiveTestSessionID != s.ID {
		t.Fatal("opening a session must put the chef in test mode and link the session")
	}
}

// Each flip is its own investigation. Overwriting the previous session would
// destroy the evidence from the last one.
func TestSecondFlipOpensSessionTwoAndRetainsTheFirst(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedLiveChef(t, db)
	adminID := uuid.New()

	s1, err := OpenTestSession(db, chefID, adminID, "first", 30)
	require.NoError(t, err)
	require.NoError(t, CloseTestSession(db, chefID, adminID))
	s2, err := OpenTestSession(db, chefID, adminID, "second", 30)
	require.NoError(t, err)

	if s2.SessionNo != 2 {
		t.Fatalf("second session must be no=2, got %d", s2.SessionNo)
	}
	var first models.ChefTestSession
	require.NoError(t, db.First(&first, "id = ?", s1.ID).Error)
	if first.Status != models.TestSessionClosed || first.ClosedAt == nil {
		t.Fatal("the first session must be retained and marked closed")
	}
}

// Going live for the first time stamps FirstLiveAt, which is what turns a
// born-test kitchen (hidden) into an established one (shown as closed).
func TestCloseStampsFirstLiveAtOnce(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedBornTestChef(t, db)
	adminID := uuid.New()

	require.NoError(t, CloseTestSession(db, chefID, adminID))
	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.NotNil(t, chef.FirstLiveAt)
	stamped := *chef.FirstLiveAt

	_, err := OpenTestSession(db, chefID, adminID, "again", 30)
	require.NoError(t, err)
	require.NoError(t, CloseTestSession(db, chefID, adminID))
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	if !chef.FirstLiveAt.Equal(stamped) {
		t.Fatal("FirstLiveAt must be stamped once and never moved")
	}
}
```

Write `setupSessionDB`, `seedLiveChef`, `seedBornTestChef`, `seedActiveOrder` as sqlite fixtures covering `chef_profiles`, `chef_test_sessions`, `orders`, `payouts`, `refund_transactions`, `cancellation_requests`, `meal_plans`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/api && go test ./services/ -run 'TestFlip|TestSecondFlip|TestCloseStamps' -v`
Expected: FAIL.

- [ ] **Step 3: Implement `services/test_session.go`**

`TestFlipBlockers` returns a human-readable string per obstruction — active (non-terminal) orders, pending/in-progress payouts, open refund or cancellation requests, active meal plans or subscriptions — counting each category and formatting as e.g. `"3 active orders"`. Empty slice means clear to flip.

`OpenTestSession` runs in a transaction: refuse with `ErrFlipBlocked` if blockers exist; compute `SessionNo` as `MAX(session_no)+1` for the chef; insert the session; set the chef's `Mode = test` and `ActiveTestSessionID`.

`CloseTestSession` runs in a transaction: mark the open session closed with `ClosedAt`/`ClosedByID`; set the chef's `Mode = live` and `ActiveTestSessionID = nil`; stamp `FirstLiveAt` only when it is currently nil. Closing when no session is open is a no-op that still sets the chef live, so a chef stuck in an inconsistent state can always be recovered.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run 'TestFlip|TestSecondFlip|TestCloseStamps' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/test_session.go apps/api/services/test_session_test.go
git commit -m "feat(test-mode): add test sessions with in-flight flip guard"
```

---

## Task 15: The live→test clone

**Files:**
- Create: `apps/api/services/test_session_clone.go`, `apps/api/services/test_session_clone_test.go`
- Modify: `apps/api/services/test_session.go` (call the clone from `OpenTestSession`)

**Interfaces:**
- Consumes: `models.ChefTestSession` (Task 2), `OpenTestSession` (Task 14)
- Produces: `services.CloneChefIntoSession(db *gorm.DB, chefID uuid.UUID, session *models.ChefTestSession, windowDays int) (map[string]int, error)`

- [ ] **Step 1: Write the failing test**

```go
// The clone must reproduce the kitchen's setup so the sandbox behaves like the
// real thing, and reach back far enough to include the order being debugged.
func TestCloneCopiesConfigAndWindowedOrders(t *testing.T) {
	db := setupCloneDB(t)
	chefID := seedChefWithData(t, db) // 3 menu items; 1 order 5d old, 1 order 200d old

	session := &models.ChefTestSession{ID: uuid.New(), ChefID: chefID, SessionNo: 1}
	summary, err := CloneChefIntoSession(db, chefID, session, 30)
	require.NoError(t, err)

	if summary["menu_items"] != 3 {
		t.Fatalf("all menu items must be cloned, got %d", summary["menu_items"])
	}
	if summary["orders"] != 1 {
		t.Fatalf("only orders inside the window may be cloned, got %d", summary["orders"])
	}
}

// Cloned rows are historical replicas. If they carried live gateway ids they
// could be charged or refunded against a real payment.
func TestClonedOrdersCarryNoGatewayIdentifiers(t *testing.T) {
	db := setupCloneDB(t)
	chefID := seedChefWithData(t, db)
	session := &models.ChefTestSession{ID: uuid.New(), ChefID: chefID, SessionNo: 1}
	_, err := CloneChefIntoSession(db, chefID, session, 30)
	require.NoError(t, err)

	var rows []models.Order
	require.NoError(t, db.Where("mode = ? AND cloned_from_id IS NOT NULL", models.ChefModeTest).Find(&rows).Error)
	require.NotEmpty(t, rows)
	for _, o := range rows {
		if o.RazorpayOrderID != "" {
			t.Fatalf("cloned order %s kept a gateway id", o.ID)
		}
		if o.TestSessionID == nil || *o.TestSessionID != session.ID {
			t.Fatalf("cloned order %s is not tied to the session", o.ID)
		}
	}
}

// The single biggest correctness risk in this feature. Cloning 118 orders must
// not fire 118 order-created pushes at a real customer, or enqueue 118 NATS
// events, or start 118 Temporal workflows.
func TestCloneEmitsNoSideEffects(t *testing.T) {
	db := setupCloneDB(t)
	chefID := seedChefWithData(t, db)
	session := &models.ChefTestSession{ID: uuid.New(), ChefID: chefID, SessionNo: 1}

	_, err := CloneChefIntoSession(db, chefID, session, 30)
	require.NoError(t, err)

	var outbox int64
	require.NoError(t, db.Table("outbox_events").Count(&outbox).Error)
	if outbox != 0 {
		t.Fatalf("the clone enqueued %d events; it must emit none", outbox)
	}
	var notifications int64
	require.NoError(t, db.Table("notifications").Count(&notifications).Error)
	if notifications != 0 {
		t.Fatalf("the clone created %d notifications; it must create none", notifications)
	}
}

// A partial clone would leave a kitchen half-copied and in test mode with no
// way to tell what is missing.
func TestCloneRollsBackWholly(t *testing.T) {
	db := setupCloneDB(t)
	chefID := seedChefWithData(t, db)
	require.NoError(t, db.Exec(`DROP TABLE orders`).Error) // force a mid-clone failure

	session := &models.ChefTestSession{ID: uuid.New(), ChefID: chefID, SessionNo: 1}
	_, err := CloneChefIntoSession(db, chefID, session, 30)
	require.Error(t, err)

	var cloned int64
	require.NoError(t, db.Table("menu_items").Where("mode = ?", models.ChefModeTest).Count(&cloned).Error)
	if cloned != 0 {
		t.Fatalf("a failed clone left %d rows behind", cloned)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/api && go test ./services/ -run TestClone -v`
Expected: FAIL — `undefined: CloneChefIntoSession`.

- [ ] **Step 3: Implement `services/test_session_clone.go`**

Structure it as a table-driven list of clone steps so adding a table later is one entry, not a new code path:

```go
// cloneStep copies one table's rows for a chef into the test partition.
type cloneStep struct {
	// Table is the physical table name, used for the summary key.
	Table string
	// Copy performs the copy inside the caller's transaction and returns how
	// many rows it wrote.
	Copy func(tx *gorm.DB, chefID uuid.UUID, session *models.ChefTestSession, since time.Time) (int, error)
}
```

Rules every step must follow, stated once at the top of the file:

```go
// CloneChefIntoSession copies a live kitchen into a fresh test session.
//
// Depth is configuration in full plus a bounded window of order history. That
// is enough to reproduce essentially any production issue while completing in
// seconds — a recursive copy of a year of ledger entries would be slower,
// far more fragile, and no more useful.
//
// Four invariants hold for every step, and the tests enforce all four:
//
//  1. INSERT ONLY, NO HOOKS. Rows are written with Session(&gorm.Session{
//     SkipHooks: true}) so no BeforeSave/AfterCreate fires. Cloning 118 orders
//     must not push 118 notifications at a real customer, enqueue 118 NATS
//     events, or start 118 Temporal workflows.
//  2. NO GATEWAY IDENTIFIERS. Razorpay order/payment/transfer ids are cleared,
//     so a cloned order can be inspected and driven through its status machine
//     but can never be charged or refunded against a real payment.
//  3. PROVENANCE. Every row carries mode=test, the session id, and
//     cloned_from_id pointing at its original — so a clone is always
//     distinguishable from something actually done in the sandbox, and a purge
//     can find every row it created.
//  4. ALL OR NOTHING. The whole clone runs in one transaction. A partial clone
//     would leave a kitchen half-copied and in test mode with no way to tell
//     what is missing.
//
// PII columns (including the #710 encrypted companions) are copied verbatim
// rather than re-encrypted, so no key material is touched.
//
// Not cloned: ledger entries, payout records, statements, invoices, wallet
// balances and loyalty lots. Those are real-money artefacts — a copy is
// meaningless in the sandbox and dangerous if it ever leaked into reporting.
func CloneChefIntoSession(db *gorm.DB, chefID uuid.UUID, session *models.ChefTestSession, windowDays int) (map[string]int, error)
```

Steps to implement: `menu_items` (and modifiers/options), `menu_categories`, `weekly_menus`, `daily_menus`, `chef_schedules`, `chef_settings`, capacity settings, delivery slots, `chef_subscription_configs`; then `orders` and `order_items` filtered to `created_at >= now - windowDays`. Confirm each table name against `apps/api/models/` before writing the step.

Then call it from `OpenTestSession` inside the same transaction, writing `ClonedAt` and `CloneSummary` onto the session before commit.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run 'TestClone|TestFlip|TestSecondFlip' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/test_session_clone.go apps/api/services/test_session_clone_test.go apps/api/services/test_session.go
git commit -m "feat(test-mode): clone a live kitchen into a test session"
```

---

## Task 16: Session purge and the round-trip guarantee

**Files:**
- Modify: `apps/api/services/test_session.go`
- Create: `apps/api/services/test_session_roundtrip_test.go`

**Interfaces:**
- Produces: `services.PurgeTestSession(db *gorm.DB, sessionID uuid.UUID) (map[string]int, error)`

- [ ] **Step 1: Write the headline failing test**

```go
// The behaviour the whole feature is judged on: flip a live kitchen to test,
// wreck it in the sandbox, flip back — and every live row is exactly as it was.
// This works because live rows are never written while in test mode, so there
// is no restore step to get wrong.
func TestLiveDataSurvivesARoundTrip(t *testing.T) {
	db := setupCloneDB(t)
	chefID := seedChefWithData(t, db)
	adminID := uuid.New()

	before := snapshotLiveRows(t, db, chefID)

	_, err := OpenTestSession(db, chefID, adminID, "reproducing a prod issue", 30)
	require.NoError(t, err)

	// Wreck the sandbox thoroughly.
	require.NoError(t, db.Exec(`UPDATE orders SET status='cancelled' WHERE mode='test'`).Error)
	require.NoError(t, db.Exec(`DELETE FROM menu_items WHERE mode='test'`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, chef_id, mode, status, created_at) VALUES (?,?,?,?,?)`,
		uuid.New().String(), chefID.String(), "test", "delivered", time.Now()).Error)

	require.NoError(t, CloseTestSession(db, chefID, adminID))

	after := snapshotLiveRows(t, db, chefID)
	require.Equal(t, before, after, "live data must be byte-identical after a test round trip")

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	if chef.IsTestMode() || chef.ActiveTestSessionID != nil {
		t.Fatal("closing must return the chef to live with no active session")
	}
}

// Purge must remove exactly the session's rows and nothing else — above all,
// not a single live row.
func TestPurgeRemovesOnlyItsOwnSession(t *testing.T) {
	db := setupCloneDB(t)
	chefID := seedChefWithData(t, db)
	adminID := uuid.New()

	s1, err := OpenTestSession(db, chefID, adminID, "first", 30)
	require.NoError(t, err)
	require.NoError(t, CloseTestSession(db, chefID, adminID))
	s2, err := OpenTestSession(db, chefID, adminID, "second", 30)
	require.NoError(t, err)
	require.NoError(t, CloseTestSession(db, chefID, adminID))

	liveBefore := snapshotLiveRows(t, db, chefID)

	_, err = PurgeTestSession(db, s1.ID)
	require.NoError(t, err)

	var remaining int64
	require.NoError(t, db.Table("orders").Where("test_session_id = ?", s1.ID).Count(&remaining).Error)
	require.Zero(t, remaining, "session 1 rows must be gone")

	require.NoError(t, db.Table("orders").Where("test_session_id = ?", s2.ID).Count(&remaining).Error)
	require.NotZero(t, remaining, "session 2 must be untouched")

	require.Equal(t, liveBefore, snapshotLiveRows(t, db, chefID), "purge must never touch live data")

	var s models.ChefTestSession
	require.NoError(t, db.First(&s, "id = ?", s1.ID).Error)
	require.Equal(t, models.TestSessionPurged, s.Status)
}
```

Write `snapshotLiveRows` to select every live-partition row for the chef across `orders`, `order_items`, `menu_items` and `reviews`, ordered deterministically, into a comparable struct slice.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/api && go test ./services/ -run 'TestLiveDataSurvives|TestPurgeRemoves' -v`
Expected: FAIL — `undefined: PurgeTestSession`.

- [ ] **Step 3: Implement `PurgeTestSession`**

Delete rows `WHERE test_session_id = ?` across every partitioned table in one transaction, mark the session `purged` with `PurgedAt`, and return per-table deletion counts. Refuse to purge an **open** session — an admin must close it first, so a purge can never race live traffic returning.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd apps/api && go test ./services/ -run 'TestLiveDataSurvives|TestPurgeRemoves|TestClone|TestFlip' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/test_session.go apps/api/services/test_session_roundtrip_test.go
git commit -m "feat(test-mode): purge test sessions and guarantee live data survives a round trip"
```

---

## Task 17: Admin API

**Files:**
- Create: `apps/api/handlers/admin_test_mode.go`, `apps/api/handlers/admin_test_mode_test.go`
- Modify: `apps/api/handlers/approval.go` (`ApproveRequest`, `approveOneRequest`), `services/onboarding_activation.go`, `handlers/admin.go` (`GetChefs`, `GetPaymentGatewayStatus`, `UpdatePaymentGatewayKeys`), `routes/routes.go`

**Interfaces:**
- Consumes: everything from Tasks 4, 8, 14, 15, 16
- Produces: routes listed below; `services.ActivateChefOnboarding(db *gorm.DB, approvalID uuid.UUID, mode string) error`

- [ ] **Step 1: Write the failing test for approve-with-mode**

```go
// Approving as Test must produce a kitchen that is in test mode and has NEVER
// been live — the born-test state, which is hidden from customers outright.
func TestApproveAsTestCreatesBornTestChef(t *testing.T) {
	db := setupApprovalDB(t)
	chefID, approvalID := seedPendingOnboarding(t, db)

	require.NoError(t, ActivateChefOnboarding(db, approvalID, models.ChefModeTest))

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.True(t, chef.IsVerified, "approval must still verify the kitchen")
	require.Equal(t, models.ChefModeTest, chef.Mode)
	require.Nil(t, chef.FirstLiveAt, "a born-test kitchen has never been live")
}

// The default must be live. An admin who does not choose gets a real kitchen.
func TestApproveDefaultsToLive(t *testing.T) {
	db := setupApprovalDB(t)
	chefID, approvalID := seedPendingOnboarding(t, db)

	require.NoError(t, ActivateChefOnboarding(db, approvalID, ""))

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.Equal(t, models.ChefModeLive, chef.Mode)
	require.NotNil(t, chef.FirstLiveAt, "going live stamps FirstLiveAt")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd apps/api && go test ./services/ -run 'TestApproveAsTest|TestApproveDefaults' -v`
Expected: FAIL — wrong arity on `ActivateChefOnboarding`.

- [ ] **Step 3: Thread mode through approval**

Change `ActivateChefOnboarding(db, approvalID)` to `ActivateChefOnboarding(db, approvalID, mode string)`. In its `Updates` map add `"mode": models.NormalizeMode(mode)`, and set `first_live_at` to now **only** when the mode is live and the column is currently null. Keep it idempotent — a retried Temporal activity must converge.

In `approveOneRequest(id, adminUserID, notes)` add a `mode string` parameter and pass it through. `ApproveRequest` binds `{"notes":"…","mode":"live|test"}`. `BulkApproveRequests` always passes `models.ChefModeLive` with a comment explaining that bulk is never the right place to mint a sandbox kitchen. Update `services.StartOnboardingActivation` and its Temporal activity signature to carry mode.

- [ ] **Step 4: Add the admin endpoints**

`apps/api/handlers/admin_test_mode.go`:

| Method | Path | Behaviour |
|---|---|---|
| `PATCH` | `/admin/chefs/:id/mode` | body `{"mode":"live\|test","reason":"…","orderWindowDays":30}`. test → `OpenTestSession`; live → `CloseTestSession`. Returns **409** with `{"blockers":[…]}` on `ErrFlipBlocked`. Audited via `services.LogAudit`. |
| `GET` | `/admin/chefs/:id/test-sessions` | session list, newest first, with clone summaries |
| `DELETE` | `/admin/test-sessions/:id` | `PurgeTestSession`; 409 if the session is open |
| `GET` | `/admin/test-mode-policy` | current allowlist |
| `PUT` | `/admin/test-mode-policy` | `SaveTestModePolicy`; audited |

Register all five in `routes/routes.go` inside the existing `admin` group next to `admin.GET("/chefs", …)` at line 976.

- [ ] **Step 5: Add the admin mode scope**

Read `X-HomeChef-Mode` (values `live`/`test`, default live) in the admin group and expose it as `handlers.adminMode(c) string`. Apply `services.ModeScope(adminMode(c))` to admin list and aggregate queries for orders, payouts, refunds, cancellations, wallets, reviews and analytics. `GetChefs` returns `mode` on each row and accepts `?mode=` as a filter.

- [ ] **Step 6: Make the payment gateway endpoints mode-scoped**

`GetPaymentGatewayStatus` reads `?mode=`, uses `services.GetRazorpayFor(mode)`, and adds to its response:

```go
		// The live slot legitimately holds a test key until a real one is
		// issued, so a mismatch is surfaced as a warning rather than refused —
		// blocking the save would make the interim state unreachable.
		"slotWarning": slotWarning, // "" when the key prefix matches the slot
```

`UpdatePaymentGatewayKeys` reads `mode` from the body, writes via `razorpaySecretNames(mode)`, calls `services.InvalidateRazorpayFor(mode)`, and **saves regardless of prefix mismatch**, returning the warning in the response. Both default to `live` when mode is absent so the current admin UI keeps working through the deploy window.

- [ ] **Step 7: Run everything**

Run: `cd apps/api && go build ./... && go vet ./... && go test ./... 2>&1 | tail -30`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/api/
git commit -m "feat(test-mode): add admin endpoints for mode, sessions, policy and dual gateway slots"
```

---

## Task 18: Schema DDL in tesserix-k8s

**Files:**
- Modify: `tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`

- [ ] **Step 1: Read the file's existing conventions**

Run: `tail -60 tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`
Match its existing style for `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` and section comments.

- [ ] **Step 2: Append the DDL**

```sql
-- ---------------------------------------------------------------------------
-- Test chef mode: per-chef live/test partitioning with dual Razorpay slots.
-- Everything defaults to 'live' so the deploy is a no-op until an admin
-- explicitly marks a kitchen as test.
-- ---------------------------------------------------------------------------

ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS first_live_at timestamptz NULL;
ALTER TABLE chef_profiles ADD COLUMN IF NOT EXISTS active_test_session_id uuid NULL;
CREATE INDEX IF NOT EXISTS idx_chef_profiles_mode ON chef_profiles(mode);

-- Existing kitchens are live and have been live since they were verified, so
-- they read as established (shown as closed if ever flipped) rather than
-- born-test (hidden). Without this backfill an existing kitchen flipped to
-- test would vanish from its regulars' app instead of showing as closed.
UPDATE chef_profiles
   SET first_live_at = COALESCE(verified_at, created_at)
 WHERE first_live_at IS NULL AND mode = 'live';

CREATE TABLE IF NOT EXISTS chef_test_sessions (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chef_id           uuid NOT NULL,
    session_no        integer NOT NULL,
    status            varchar(10) NOT NULL DEFAULT 'open',
    reason            text DEFAULT '',
    order_window_days integer DEFAULT 30,
    cloned_at         timestamptz NULL,
    clone_summary     jsonb NULL,
    opened_by_id      uuid NULL,
    opened_at         timestamptz NOT NULL DEFAULT now(),
    closed_by_id      uuid NULL,
    closed_at         timestamptz NULL,
    purged_at         timestamptz NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_chef_test_sessions_chef ON chef_test_sessions(chef_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_chef_test_sessions_chef_no ON chef_test_sessions(chef_id, session_no);

CREATE TABLE IF NOT EXISTS chef_mode_stats (
    chef_id       uuid NOT NULL,
    mode          varchar(4) NOT NULL DEFAULT 'live',
    total_orders  integer NOT NULL DEFAULT 0,
    rating        double precision NOT NULL DEFAULT 0,
    total_reviews integer NOT NULL DEFAULT 0,
    issue_count   integer NOT NULL DEFAULT 0,
    updated_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (chef_id, mode)
);
```

- [ ] **Step 3: Generate the per-table columns**

For every table in the Task 3 inventory, append:

```sql
ALTER TABLE <table> ADD COLUMN IF NOT EXISTS mode varchar(4) NOT NULL DEFAULT 'live';
ALTER TABLE <table> ADD COLUMN IF NOT EXISTS test_session_id uuid NULL;
ALTER TABLE <table> ADD COLUMN IF NOT EXISTS cloned_from_id uuid NULL;
-- Partial index: the overwhelming majority of rows are live and are not
-- indexed at all, so the live hot path is unaffected.
CREATE INDEX IF NOT EXISTS idx_<table>_mode ON <table>(mode) WHERE mode <> 'live';
```

- [ ] **Step 4: Verify the SQL parses**

Run against a scratch Postgres 16, not production:

```bash
docker run --rm -d --name schema-check -e POSTGRES_PASSWORD=x -p 55432:5432 postgres:16-alpine
sleep 5
psql "postgres://postgres:x@localhost:55432/postgres" -v ON_ERROR_STOP=1 \
  -f tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql
docker rm -f schema-check
```

Expected: no errors. Run it **twice** against the same container to prove idempotency — the bootstrap CronJob runs every 30 minutes.

- [ ] **Step 5: Commit in tesserix-k8s**

```bash
cd ../tesserix-k8s
git config user.name "sam123ben" && git config user.email "samyak.rout@gmail.com"
git checkout -b feat/homechef-test-chef-mode
git add charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql
git commit -m "feat(homechef): add test chef mode schema"
```

---

## Task 19: tesserix-home — global mode toggle and chefs page

**Files:**
- Create: `tesserix-home/apps/web/components/admin/homechef/mode-toggle.tsx`, `test-badge.tsx`
- Modify: `apps/web/app/admin/apps/homechef/page.tsx` (shell), `chefs/page.tsx`, `app/api/admin/apps/homechef/gw/[...path]/route.ts` (forward `X-HomeChef-Mode`)

- [ ] **Step 1: Read the existing patterns**

```bash
cd tesserix-home/apps/web
cat app/admin/apps/homechef/chefs/page.tsx
cat components/admin/homechef/status-badge.tsx
cat app/api/admin/apps/homechef/gw/\[...path\]/route.ts
```

Match the existing data-fetching, styling and component conventions exactly. Do not introduce a new state library or styling approach.

- [ ] **Step 2: Build the toggle**

`mode-toggle.tsx` — a two-state Live/Test switch persisted in `localStorage` under `homechef:adminMode`, defaulting to `live`, exposed via a React context so every page can read it. When Test is active, render a persistent full-width high-contrast bar reading **"TEST MODE — you are viewing sandbox data"** above the page content. This is not decoration: an admin acting on a fake refund believing it is real is the failure this prevents.

- [ ] **Step 3: Forward the header**

In the gw proxy route, read the mode from the incoming request and forward it as `X-HomeChef-Mode` on the upstream call. Confirm the HMAC signing covers only the path/body as before — adding a header must not break the signature.

- [ ] **Step 4: Update the chefs page**

Add a `TEST` badge on rows where `chef.mode === 'test'`, a mode filter, and a "Switch mode" action. The action opens a dialog asking for a reason (required) and, for live→test, an order window in days (default 30). On a `409` response, render the returned `blockers` array as a list — "3 active orders", "1 pending payout" — rather than a generic error.

- [ ] **Step 5: Verify**

Run: `cd tesserix-home && pnpm build && npx tsc --noEmit`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git checkout -b feat/homechef-test-chef-mode
git add apps/web/
git commit -m "feat(homechef-admin): add global test/live toggle and chef mode switching"
```

---

## Task 20: tesserix-home — approvals, sessions, settings, gateway

**Files:**
- Modify: `apps/web/app/admin/apps/homechef/approvals/[id]/page.tsx`, `platform-settings/page.tsx`, `payment-gateway/page.tsx`
- Create: `apps/web/app/admin/apps/homechef/chefs/[id]/test-sessions/page.tsx`

- [ ] **Step 1: Approve dialog**

Replace the bare Approve button on the approval detail page with a dialog offering **Live** (preselected) and **Test**, plus one line of plain-language explanation: *"Test kitchens are visible only to the test-mode allowlist and take payments through Razorpay test credentials. No real money moves."* Submit posts `{"notes": "...", "mode": "live" | "test"}`.

- [ ] **Step 2: Test sessions page**

A table of sessions for one chef: number, status, reason, opened/closed timestamps, order window, and the clone summary rendered as per-table counts. Each **closed** session gets a Purge action behind a confirmation naming the chef and session number. Open sessions show no Purge action.

- [ ] **Step 3: Platform settings**

Add a "Test-mode viewers" card: an editable email list wired to `GET`/`PUT /admin/test-mode-policy`, with add/remove and basic email-shape validation. Include a one-line note that these are the only accounts that can see or order from test kitchens.

- [ ] **Step 4: Payment gateway — two cards**

Split the page into **Live credentials** and **Test credentials**, each with key ID, key secret, webhook secret, a Save action and a status readout (configured, key prefix, webhook secret set, health-check result). Render `slotWarning` from the API as a prominent amber banner on the affected card. Show the webhook URL on both cards with a note that **both** Razorpay dashboards must point at it.

- [ ] **Step 5: Verify**

Run: `cd tesserix-home && pnpm build && npx tsc --noEmit`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add apps/web/
git commit -m "feat(homechef-admin): approve-as-test, test sessions, viewer allowlist and dual gateway cards"
```

---

## Task 21: Mobile surfaces

**Files:**
- Create: `apps/mobile-customer/src/components/TestBadge.tsx`, `apps/mobile-vendor/src/components/TestModeBanner.tsx`
- Modify: the customer chef card, chef detail header and checkout screen; the vendor dashboard shell

- [ ] **Step 1: Locate the components**

```bash
cd apps/mobile-customer && grep -rln "ChefCard\|chef-card" src/ | head
cd ../mobile-vendor && grep -rln "Dashboard" src/screens src/app 2>/dev/null | head
```

- [ ] **Step 2: Customer badge**

A small pill reading `TEST`, using the existing design tokens (persimmon accent, 8px radius, Inter 500 — see `.impeccable.md`). Render it on the chef card and chef detail header when `chef.mode === 'test'`, and show a one-line notice above the checkout pay button: *"Test kitchen — this payment uses Razorpay test mode and no real money will be charged."* Non-allowlisted users never receive a test chef from the API, so these render for nobody else.

- [ ] **Step 3: Vendor banner**

A persistent banner at the top of the vendor dashboard shell when the signed-in chef's `mode === 'test'`, reading **"TEST MODE — session {sessionNo}. Earnings shown here are not real."** A chef mistaking sandbox money for real earnings is the failure this prevents.

- [ ] **Step 4: Verify**

Run: `cd apps/mobile-customer && npx tsc --noEmit` and the same in `apps/mobile-vendor`.
Expected: clean. Do not run EAS builds.

- [ ] **Step 5: Commit**

```bash
git add apps/mobile-customer/ apps/mobile-vendor/
git commit -m "feat(test-mode): surface test badges in customer app and a test banner in vendor app"
```

---

## Task 22: Full verification pass

- [ ] **Step 1: Backend**

```bash
cd apps/api && go build ./... && go vet ./... && go test ./... 2>&1 | tail -40
```
Expected: all green. Record the pass/fail counts.

- [ ] **Step 2: Regression evidence**

Confirm the pre-existing money and discovery suites pass **unchanged**: `payment_*_test.go`, `meal_plan_*_test.go`, `payout_*_test.go`, `wallet_*_test.go`, `loyalty_*_test.go`, `chefs_search_test.go`. This is the primary safety argument for shipping live — the feature is inert until a chef is marked test.

- [ ] **Step 3: Frontends**

```bash
cd tesserix-home && pnpm build && npx tsc --noEmit
cd ../Home-Chef-App/apps/mobile-customer && npx tsc --noEmit
cd ../mobile-vendor && npx tsc --noEmit
```

- [ ] **Step 4: Grep for stragglers**

```bash
cd apps/api
# Every chef-scoped gateway call must now be mode-aware.
grep -rn "GetRazorpay()" --include="*.go" handlers/ services/ | grep -v _test
```
Each remaining hit must be a genuinely platform-level call (admin gateway status, reconciliation, wallet top-up, platform subscription). Anything chef-scoped is a bug — fix it.

- [ ] **Step 5: Report honestly**

State exactly what passed, what did not, and anything left undone. Do not claim completion for anything unverified.

- [ ] **Step 6: Final commit and push**

```bash
git add -A && git commit -m "chore(test-mode): final verification pass"
git push -u origin feat/test-chef-mode
```

---

## Post-merge runbook (user-executed)

Claude does not run these — hand them to the user.

1. Deploy the API and confirm the schema bootstrap CronJob has applied the new columns.
2. Copy the current live Razorpay secrets into the new test slot (confirm before running):
   ```bash
   for s in key-id key-secret webhook-secret; do
     gcloud secrets versions access latest --secret="prod-homechef-razorpay-$s" \
       --project=tesseracthub-480811 \
     | gcloud secrets create "prod-homechef-razorpay-test-$s" --data-file=- \
       --project=tesseracthub-480811 2>/dev/null \
     || gcloud secrets versions access latest --secret="prod-homechef-razorpay-$s" \
          --project=tesseracthub-480811 \
        | gcloud secrets versions add "prod-homechef-razorpay-test-$s" --data-file=- \
          --project=tesseracthub-480811
   done
   ```
3. Admin → Payment Gateway → both cards green, with the expected "live slot holds a test key" warning.
4. Point the Razorpay **test** dashboard webhook at `https://api.fe3dr.com/webhooks/razorpay`.
5. Admin → Platform Settings → confirm the three viewer emails.
6. Onboard a chef, approve as **Test**, verify invisible on a non-allowlisted account and visible on an allowlisted one.
7. Place a test order end to end: test checkout opens, no real money moves, no 3PL dispatch, `[TEST]` notifications arrive, order absent from revenue analytics.
8. On a settled live chef, flip to test, confirm the clone summary and that the kitchen shows as Closed to a real customer; flip back and confirm live data is exactly as it was.
