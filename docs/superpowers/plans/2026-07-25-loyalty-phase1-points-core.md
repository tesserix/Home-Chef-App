# Loyalty Phase 1 — Points Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden the existing points engine into the locked economics — earn on food subtotal, per-batch 1-year expiry with FIFO redemption + a daily sweep, refund reversal, a monthly redeem cap, an admin grant/adjust endpoint, an env kill-switch — and enable the mobile loyalty screen.

**Architecture:** Reuse the existing `LoyaltyAccount` / `LoyaltyTransaction` / `applyLoyaltyTxnInTx` / `RedeemLoyalty` / `AwardOrderLoyalty` / `CreditWallet`. Add a new dated-lot table `LoyaltyEarnBatch`: every point CREDIT writes a batch (with `expires_at`); every DEBIT (redeem, expiry, refund-reversal) FIFO-consumes `points_remaining` from the soonest-expiring batches. `LoyaltyAccount.Balance` stays the fast running total. All money paths stay idempotent and single-transaction.

**Tech Stack:** Go 1.26, Gin, GORM, PostgreSQL 16 (prod), in-memory sqlite for unit tests. Cron via the existing `cronJobs()` registry.

## Global Constraints

- **DDL lives in tesserix-k8s only** — `charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`. The app repo carries ONLY the GORM model. NEVER put `CREATE TABLE`/`ALTER TABLE` in the app repo.
- **Git identity:** `git config user.name "sam123ben"` / `git config user.email "samyak.rout@gmail.com"`. NEVER mention AI/Claude/Co-Authored-By in commits.
- **Economics (locked, all admin-tunable via `platform_settings`):** earn 1 pt / ₹10 **food subtotal** (`points_per_rupee`=0.1); 1 pt = ₹0.05 (`redeem_rate`=0.05); min redeem 500 pts (`min_redeem`=500); monthly redeem cap ₹300 (`monthly_redeem_cap`=300); expiry 365 days (`expiry_days`=365); max redeem 10%/order (`max_redeem_pct`=0.10) — config only in Phase 1, enforced at checkout in Phase 3.
- **Idempotency:** every credit/debit passes a stable `idempotency_key`; `applyLoyaltyTxnInTx` returns `(entry, created bool, err)` — `created==false` on a dedup hit.
- **Verify with `go build ./... && go vet ./... && go test ./...`** (from `apps/api`). Never run container builds/deploys.
- **Points redeem to WALLET (never cash).** "Restore points on refunded points-paid order" is already handled by the existing wallet refund (points → wallet → wallet applied → wallet refunded); Phase 1 only adds EARN reversal.

---

### Task 1: Loyalty config — new economics + three new keys

**Files:**
- Modify: `apps/api/services/loyalty.go` (`LoyaltyConfig` struct ~l.32, `defaultLoyaltyConfig` ~l.47, `GetLoyaltyConfig` switch ~l.62)
- Modify: `apps/api/handlers/admin_loyalty.go` (`UpdateLoyaltyConfig` ~l.25)
- Test: `apps/api/services/loyalty_config_test.go` (new)

**Interfaces:**
- Produces: `LoyaltyConfig{ …existing…, MaxRedeemPct, MonthlyRedeemCap, ExpiryDays float64 }`; defaults `RedeemRate=0.05, MinRedeem=500, MaxRedeemPct=0.10, MonthlyRedeemCap=300, ExpiryDays=365`.

- [ ] **Step 1: Write the failing test** — `apps/api/services/loyalty_config_test.go`:

```go
package services

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/homechef/api/models"
)

func TestGetLoyaltyConfig_NewDefaults(t *testing.T) {
	db := setupLoyaltyDB(t)
	cfg := GetLoyaltyConfig(db)
	require.Equal(t, 0.1, cfg.PointsPerRupee)
	require.Equal(t, 0.05, cfg.RedeemRate)
	require.Equal(t, 500.0, cfg.MinRedeem)
	require.Equal(t, 0.10, cfg.MaxRedeemPct)
	require.Equal(t, 300.0, cfg.MonthlyRedeemCap)
	require.Equal(t, 365.0, cfg.ExpiryDays)
}

func TestGetLoyaltyConfig_ReadsNewKeys(t *testing.T) {
	db := setupLoyaltyDB(t)
	for k, v := range map[string]string{
		"loyalty.redeem_rate":        "0.1",
		"loyalty.max_redeem_pct":     "0.2",
		"loyalty.monthly_redeem_cap": "500",
		"loyalty.expiry_days":        "180",
	} {
		require.NoError(t, db.Create(&models.PlatformSettings{Key: k, Value: v}).Error)
	}
	cfg := GetLoyaltyConfig(db)
	require.Equal(t, 0.1, cfg.RedeemRate)
	require.Equal(t, 0.2, cfg.MaxRedeemPct)
	require.Equal(t, 500.0, cfg.MonthlyRedeemCap)
	require.Equal(t, 180.0, cfg.ExpiryDays)
}
```

- [ ] **Step 2: Run — expect FAIL** (`cfg.MaxRedeemPct` undefined):

```
cd apps/api && go test ./services/ -run TestGetLoyaltyConfig -count=1
```

- [ ] **Step 3: Add the fields to `LoyaltyConfig`** (after `TierGoldAt`):

```go
	MaxRedeemPct     float64 `json:"maxRedeemPct"`     // max redemption per order as a fraction of food subtotal (checkout, Phase 3)
	MonthlyRedeemCap float64 `json:"monthlyRedeemCap"` // max ₹ of wallet credit redeemed per rolling 30 days
	ExpiryDays       float64 `json:"expiryDays"`       // points expire this many days after they are earned
```

- [ ] **Step 4: Change defaults in `defaultLoyaltyConfig()`** — set `RedeemRate: 0.05`, `MinRedeem: 500`, and add `MaxRedeemPct: 0.10, MonthlyRedeemCap: 300, ExpiryDays: 365`. Leave `PointsPerRupee: 0.1`.

- [ ] **Step 5: Extend the `GetLoyaltyConfig` switch** — add cases mirroring the existing ParseFloat cases:

```go
	case "loyalty.max_redeem_pct":
		if f, err := strconv.ParseFloat(s.Value, 64); err == nil {
			cfg.MaxRedeemPct = f
		}
	case "loyalty.monthly_redeem_cap":
		if f, err := strconv.ParseFloat(s.Value, 64); err == nil {
			cfg.MonthlyRedeemCap = f
		}
	case "loyalty.expiry_days":
		if f, err := strconv.ParseFloat(s.Value, 64); err == nil {
			cfg.ExpiryDays = f
		}
```

- [ ] **Step 6: Extend `UpdateLoyaltyConfig`** (`admin_loyalty.go`) — add `MaxRedeemPct/MonthlyRedeemCap/ExpiryDays *float64` to the req struct and three setter blocks using the existing `num()` helper, e.g.:

```go
	if req.MaxRedeemPct != nil {
		setPlatformSetting("loyalty.max_redeem_pct", num(*req.MaxRedeemPct), userID)
	}
	if req.MonthlyRedeemCap != nil {
		setPlatformSetting("loyalty.monthly_redeem_cap", num(*req.MonthlyRedeemCap), userID)
	}
	if req.ExpiryDays != nil {
		setPlatformSetting("loyalty.expiry_days", num(*req.ExpiryDays), userID)
	}
```

- [ ] **Step 7: Run — expect PASS** (`go test ./services/ -run TestGetLoyaltyConfig -count=1`) and `go build ./...`.

- [ ] **Step 8: Commit** — `git commit -m "feat(loyalty): points economics — 100pts=₹5, min 500, monthly cap, 1y expiry config"`

---

### Task 2: `LoyaltyEarnBatch` model + DDL + FIFO helpers

**Files:**
- Create: `apps/api/models/loyalty_batch.go`
- Create: `apps/api/services/loyalty_batch.go`
- Test: `apps/api/services/loyalty_batch_test.go`
- **DDL (separate repo):** `tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`

**Interfaces:**
- Produces:
  - `models.LoyaltyEarnBatch{ID, UserID, Source, Points, PointsRemaining, EarnedAt, ExpiresAt, OrderID *uuid.UUID, IdempotencyKey}`
  - `createEarnBatch(tx *gorm.DB, userID uuid.UUID, points float64, source models.LoyaltyTxnSource, orderID *uuid.UUID, expiryDays float64, idempotencyKey string) error` — idempotent on `idempotencyKey`.
  - `consumeBatchesFIFO(tx *gorm.DB, userID uuid.UUID, points float64) error` — decrements `points_remaining` from soonest-expiring non-empty batches until `points` consumed; returns `ErrInsufficientLoyaltyPoints` if batches can't cover it.

- [ ] **Step 1: Create the GORM model** — `apps/api/models/loyalty_batch.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoyaltyEarnBatch is a dated lot of earned points. Every point CREDIT writes one
// (with ExpiresAt = EarnedAt + expiry_days). Redeem/expiry/refund-reversal FIFO-consume
// PointsRemaining from the soonest-expiring lots. LoyaltyAccount.Balance stays the fast
// running total (= Σ PointsRemaining of the account's non-expired lots).
type LoyaltyEarnBatch struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID        `gorm:"type:uuid;not null;index" json:"userId"`
	Source          LoyaltyTxnSource `gorm:"type:varchar(20);not null" json:"source"`
	Points          float64          `gorm:"not null" json:"points"`          // originally earned
	PointsRemaining float64          `gorm:"not null" json:"pointsRemaining"` // unspent, un-expired
	EarnedAt        time.Time        `gorm:"not null;index" json:"earnedAt"`
	ExpiresAt       time.Time        `gorm:"not null;index" json:"expiresAt"`
	OrderID         *uuid.UUID       `gorm:"type:uuid;index" json:"orderId,omitempty"`
	IdempotencyKey  string           `gorm:"type:varchar(160);uniqueIndex;not null" json:"-"`
	CreatedAt       time.Time        `gorm:"autoCreateTime" json:"createdAt"`
}

func (b *LoyaltyEarnBatch) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
```

- [ ] **Step 2: Add the DDL to tesserix-k8s** (NOT the app repo). Append to `tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql` (idempotent):

```sql
CREATE TABLE IF NOT EXISTS loyalty_earn_batches (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL,
  source varchar(20) NOT NULL,
  points double precision NOT NULL,
  points_remaining double precision NOT NULL,
  earned_at timestamptz NOT NULL,
  expires_at timestamptz NOT NULL,
  order_id uuid,
  idempotency_key varchar(160) NOT NULL,
  created_at timestamptz DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS ux_loyalty_earn_batches_idem ON loyalty_earn_batches (idempotency_key);
CREATE INDEX IF NOT EXISTS ix_loyalty_earn_batches_user ON loyalty_earn_batches (user_id, expires_at);
CREATE INDEX IF NOT EXISTS ix_loyalty_earn_batches_expiry ON loyalty_earn_batches (expires_at) WHERE points_remaining > 0;
```

- [ ] **Step 3: Write the failing test** — `apps/api/services/loyalty_batch_test.go`. Extend the harness to add the `loyalty_earn_batches` table (create a local helper that calls `setupLoyaltyDB` then `db.Exec` the sqlite DDL):

```go
package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

func setupBatchDB(t *testing.T) *gorm.DB {
	db := setupLoyaltyDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE loyalty_earn_batches (
		id text PRIMARY KEY, user_id text, source text, points real, points_remaining real,
		earned_at datetime, expires_at datetime, order_id text, idempotency_key text UNIQUE, created_at datetime)`).Error)
	return db
}

func batchRemaining(t *testing.T, db *gorm.DB, u uuid.UUID) float64 {
	var sum float64
	db.Raw(`SELECT COALESCE(SUM(points_remaining),0) FROM loyalty_earn_batches WHERE user_id = ?`, u.String()).Scan(&sum)
	return sum
}

func TestCreateEarnBatch_Idempotent(t *testing.T) {
	db := setupBatchDB(t)
	u := uuid.New()
	require.NoError(t, createEarnBatch(db, u, 100, models.LoyaltySourceOrder, nil, 365, "k1"))
	require.NoError(t, createEarnBatch(db, u, 100, models.LoyaltySourceOrder, nil, 365, "k1")) // dup no-op
	require.Equal(t, 100.0, batchRemaining(t, db, u))
}

func TestConsumeBatchesFIFO_SoonestExpiryFirst(t *testing.T) {
	db := setupBatchDB(t)
	u := uuid.New()
	// Two lots: older-expiring 30 pts, later-expiring 100 pts.
	old := models.LoyaltyEarnBatch{ID: uuid.New(), UserID: u, Source: models.LoyaltySourceOrder, Points: 30, PointsRemaining: 30,
		EarnedAt: time.Now().Add(-48 * time.Hour), ExpiresAt: time.Now().Add(24 * time.Hour), IdempotencyKey: "b-old"}
	newer := models.LoyaltyEarnBatch{ID: uuid.New(), UserID: u, Source: models.LoyaltySourceOrder, Points: 100, PointsRemaining: 100,
		EarnedAt: time.Now(), ExpiresAt: time.Now().Add(240 * time.Hour), IdempotencyKey: "b-new"}
	require.NoError(t, db.Create(&old).Error)
	require.NoError(t, db.Create(&newer).Error)

	require.NoError(t, consumeBatchesFIFO(db, u, 50)) // drains the 30-lot, then 20 from the 100-lot
	require.Equal(t, 80.0, batchRemaining(t, db, u))

	var oldRow models.LoyaltyEarnBatch
	db.First(&oldRow, "idempotency_key = ?", "b-old")
	require.Equal(t, 0.0, oldRow.PointsRemaining)
}

func TestConsumeBatchesFIFO_Insufficient(t *testing.T) {
	db := setupBatchDB(t)
	u := uuid.New()
	require.NoError(t, createEarnBatch(db, u, 10, models.LoyaltySourceOrder, nil, 365, "k"))
	require.ErrorIs(t, consumeBatchesFIFO(db, u, 50), ErrInsufficientLoyaltyPoints)
}
```

- [ ] **Step 4: Run — expect FAIL** (`createEarnBatch` undefined): `go test ./services/ -run 'TestCreateEarnBatch|TestConsumeBatchesFIFO' -count=1`

- [ ] **Step 5: Implement the helpers** — `apps/api/services/loyalty_batch.go`:

```go
package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// createEarnBatch records a dated lot for a point CREDIT. Idempotent on idempotencyKey
// (a redelivered event or retried grant writes no second lot).
func createEarnBatch(tx *gorm.DB, userID uuid.UUID, points float64, source models.LoyaltyTxnSource, orderID *uuid.UUID, expiryDays float64, idempotencyKey string) error {
	if points <= 0 {
		return nil
	}
	var existing models.LoyaltyEarnBatch
	err := tx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error
	if err == nil {
		return nil // already recorded
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	now := time.Now()
	if expiryDays <= 0 {
		expiryDays = 365
	}
	return tx.Create(&models.LoyaltyEarnBatch{
		ID: uuid.New(), UserID: userID, Source: source, Points: points, PointsRemaining: points,
		EarnedAt: now, ExpiresAt: now.Add(time.Duration(expiryDays) * 24 * time.Hour),
		OrderID: orderID, IdempotencyKey: idempotencyKey,
	}).Error
}

// consumeBatchesFIFO decrements points_remaining from the soonest-expiring non-empty lots
// until `points` is consumed. Returns ErrInsufficientLoyaltyPoints if the lots can't cover it
// (the caller's balance check should already prevent this — this is defense in depth).
func consumeBatchesFIFO(tx *gorm.DB, userID uuid.UUID, points float64) error {
	if points <= 0 {
		return nil
	}
	var batches []models.LoyaltyEarnBatch
	if err := tx.Where("user_id = ? AND points_remaining > 0", userID).
		Order("expires_at ASC").Find(&batches).Error; err != nil {
		return err
	}
	remaining := points
	for i := range batches {
		if remaining <= 0 {
			break
		}
		take := batches[i].PointsRemaining
		if take > remaining {
			take = remaining
		}
		if err := tx.Model(&batches[i]).Update("points_remaining", batches[i].PointsRemaining-take).Error; err != nil {
			return err
		}
		remaining -= take
	}
	if remaining > 1e-6 {
		return ErrInsufficientLoyaltyPoints
	}
	return nil
}
```

- [ ] **Step 6: Run — expect PASS** and `go build ./...`.
- [ ] **Step 7: Commit** — `git commit -m "feat(loyalty): dated earn batches + FIFO consumption helpers"` (app repo). Commit the tesserix-k8s DDL separately in its own repo with `git commit -m "feat(homechef-db): loyalty_earn_batches table"`.

---

### Task 3: Wire batches into `applyLoyaltyTxnInTx` (credit writes a lot, debit consumes FIFO)

**Files:**
- Modify: `apps/api/services/loyalty.go` (`applyLoyaltyTxnInTx` ~l.188)
- Test: `apps/api/services/loyalty_batch_wire_test.go`

**Interfaces:**
- Consumes: `createEarnBatch`, `consumeBatchesFIFO` (Task 2); `LoyaltyConfig.ExpiryDays` (Task 1).
- Produces: after this task, every credit through `applyLoyaltyTxnInTx` writes a matching earn batch keyed `batch:<idempotencyKey>`; every debit FIFO-consumes.

- [ ] **Step 1: Write the failing test** — `apps/api/services/loyalty_batch_wire_test.go`:

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/homechef/api/models"
)

func TestApplyTxn_CreditWritesBatch_DebitConsumes(t *testing.T) {
	db := setupBatchDB(t)
	u := uuid.New()
	cfg := GetLoyaltyConfig(db)

	_, created, err := applyLoyaltyTxnInTx(db, u, 300, models.LoyaltyCredit, models.LoyaltySourceOrder, nil, "earn", "credit-1", nil, cfg)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, 300.0, batchRemaining(t, db, u)) // one lot of 300

	_, _, err = applyLoyaltyTxnInTx(db, u, 120, models.LoyaltyDebit, models.LoyaltySourceRedeem, nil, "redeem", "debit-1", nil, cfg)
	require.NoError(t, err)
	require.Equal(t, 180.0, batchRemaining(t, db, u)) // 300 − 120
}
```

- [ ] **Step 2: Run — expect FAIL** (batch total wrong: no batch written yet): `go test ./services/ -run TestApplyTxn_CreditWritesBatch -count=1`

- [ ] **Step 3: Wire into `applyLoyaltyTxnInTx`.** Immediately AFTER the successful `tx.Model(acct).Updates(...)` call (just before `return entry, true, nil`), add:

```go
	// Dated-lot bookkeeping: a credit opens a lot; a debit FIFO-consumes lots.
	if txnType == models.LoyaltyCredit {
		if err := createEarnBatch(tx, userID, points, source, orderID, cfg.ExpiryDays, "batch:"+idempotencyKey); err != nil {
			return nil, false, err
		}
	} else {
		if err := consumeBatchesFIFO(tx, userID, points); err != nil {
			return nil, false, err
		}
	}
```

- [ ] **Step 4: Run — expect PASS**, then run the FULL existing loyalty suite to confirm no regression: `go test ./services/ -run Loyalty -count=1` and `go test ./handlers/ -run Loyalty -count=1`. (Existing tests use `setupLoyaltyDB` which has no `loyalty_earn_batches` table — the credit path would now error. **Fix:** add the `loyalty_earn_batches` sqlite DDL into `setupLoyaltyDB` itself in `loyalty_test.go` so all loyalty tests get the table; then `setupBatchDB` just delegates to it. Do this in Step 5.)

- [ ] **Step 5: Move the batch table into the shared harness** — cut the `CREATE TABLE loyalty_earn_batches …` sqlite statement into `setupLoyaltyDB` (`loyalty_test.go`) `stmts` slice, and simplify `setupBatchDB` to `return setupLoyaltyDB(t)`. Re-run `go test ./services/ ./handlers/ -run Loyalty -count=1` — expect PASS.

- [ ] **Step 6: Commit** — `git commit -m "feat(loyalty): credits open dated lots, debits FIFO-consume them"`

---

### Task 4: Earn on FOOD SUBTOTAL (not total)

**Files:**
- Modify: `apps/api/services/loyalty.go` (`AwardOrderLoyalty` ~l.283)
- Modify: `apps/api/services/notifications.go` (`handleOrderDelivered` ~l.506)
- Test: `apps/api/services/loyalty_earn_subtotal_test.go`

**Interfaces:**
- Produces: `AwardOrderLoyalty(db *gorm.DB, userID, orderID uuid.UUID) (float64, error)` — loads the order and earns on `order.Subtotal`. (Signature drops the `orderTotal` param.)

- [ ] **Step 1: Write the failing test** — needs an `orders` table in the harness. Add to `setupLoyaltyDB` `stmts`: `` `CREATE TABLE orders (id text PRIMARY KEY, customer_id text, subtotal real, total real, created_at datetime)` ``. Then:

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAwardOrderLoyalty_EarnsOnSubtotal(t *testing.T) {
	db := setupLoyaltyDB(t)
	u, oid := uuid.New(), uuid.New()
	// subtotal 500, total 620 (fees/GST) — must earn on 500 → floor(500 * 0.1) = 50 pts.
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, subtotal, total) VALUES (?,?,?,?)`,
		oid.String(), u.String(), 500.0, 620.0).Error)
	awarded, err := AwardOrderLoyalty(db, u, oid)
	require.NoError(t, err)
	require.Equal(t, 50.0, awarded)
}
```

- [ ] **Step 2: Run — expect FAIL** (signature mismatch): `go test ./services/ -run TestAwardOrderLoyalty_EarnsOnSubtotal -count=1`

- [ ] **Step 3: Change `AwardOrderLoyalty`** to load the order's subtotal:

```go
func AwardOrderLoyalty(db *gorm.DB, userID, orderID uuid.UUID) (float64, error) {
	cfg := GetLoyaltyConfig(db)
	if !cfg.Enabled || (config.AppConfig != nil && !config.AppConfig.LoyaltyEnabled) {
		return 0, nil
	}
	var order models.Order
	if err := db.Select("subtotal").First(&order, "id = ?", orderID).Error; err != nil {
		return 0, err
	}
	points := PointsForOrder(cfg, order.Subtotal) // earn on FOOD subtotal, not total
	if points <= 0 {
		return 0, nil
	}
	// ...unchanged transaction body, but pass &oid and the same idempotency key "loyalty:order:"+orderID.String()...
}
```

(Keep the rest of the tx body identical — the `applyLoyaltyTxnInTx` credit + `EnqueueEvent` — only the base amount source changed.) Note: `config` import already present; `config.AppConfig.LoyaltyEnabled` is added in Task 8 — if building this task before Task 8, temporarily gate only on `cfg.Enabled` and add the env check in Task 8.

- [ ] **Step 4: Update the caller** — `notifications.go:511` inside `handleOrderDelivered`:

```go
	if _, err := AwardOrderLoyalty(database.DB, event.CustomerID, event.OrderID); err != nil {
		log.Printf("loyalty award failed for order %s: %v", event.OrderID, err)
	}
```

- [ ] **Step 5: Run — expect PASS** and `go build ./...` (fix any other `AwardOrderLoyalty(` callers to drop the total arg — grep `AwardOrderLoyalty(`).
- [ ] **Step 6: Commit** — `git commit -m "feat(loyalty): earn points on food subtotal, not order total"`

---

### Task 5: Daily expiry sweep cron

**Files:**
- Create: `apps/api/services/loyalty_expiry_cron.go`
- Modify: `apps/api/services/cron_temporal.go` (`cronJobs()` ~l.25)
- Test: `apps/api/services/loyalty_expiry_test.go`

**Interfaces:**
- Produces: `ExpireLoyaltyBatches(db *gorm.DB, now time.Time) (int, error)` — expires all past-due non-empty lots (one debit txn per lot, `source=expiry`, key `loyalty:expiry:<batch_id>`), returns count expired. `StartLoyaltyExpiryCron(ctx)` + `runLoyaltyExpiryScan(ctx)` wrap it on a 24h ticker.

- [ ] **Step 1: Write the failing test** — `apps/api/services/loyalty_expiry_test.go`:

```go
package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/homechef/api/models"
)

func TestExpireLoyaltyBatches_ZeroesPastDueAndDebits(t *testing.T) {
	db := setupLoyaltyDB(t)
	u := uuid.New()
	cfg := GetLoyaltyConfig(db)
	// Credit 100 pts via a lot that already expired.
	_, _, err := applyLoyaltyTxnInTx(db, u, 100, models.LoyaltyCredit, models.LoyaltySourceOrder, nil, "e", "c1", nil, cfg)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`UPDATE loyalty_earn_batches SET expires_at = ? WHERE user_id = ?`,
		time.Now().Add(-time.Hour), u.String()).Error)

	n, err := ExpireLoyaltyBatches(db, time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, 0.0, batchRemaining(t, db, u)) // lot drained
	require.Equal(t, 0.0, LoyaltyBalance(db, u))    // account balance decremented

	// Idempotent: a second sweep expires nothing.
	n2, err := ExpireLoyaltyBatches(db, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, n2)
}
```

- [ ] **Step 2: Run — expect FAIL** (`ExpireLoyaltyBatches` undefined).

- [ ] **Step 3: Implement `ExpireLoyaltyBatches` + cron wrappers** — `apps/api/services/loyalty_expiry_cron.go`:

```go
package services

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

const loyaltyExpiryInterval = 24 * time.Hour

// ExpireLoyaltyBatches debits each past-due non-empty lot's remaining points (source=expiry,
// idempotent per lot) and zeroes the lot. Returns how many lots were expired.
func ExpireLoyaltyBatches(db *gorm.DB, now time.Time) (int, error) {
	var due []models.LoyaltyEarnBatch
	if err := db.Where("points_remaining > 0 AND expires_at <= ?", now).Find(&due).Error; err != nil {
		return 0, err
	}
	cfg := GetLoyaltyConfig(db)
	expired := 0
	for i := range due {
		b := due[i]
		err := db.Transaction(func(tx *gorm.DB) error {
			// Re-read the lot inside the tx; skip if already drained (idempotent).
			var cur models.LoyaltyEarnBatch
			if err := tx.First(&cur, "id = ?", b.ID).Error; err != nil {
				return err
			}
			if cur.PointsRemaining <= 0 {
				return nil
			}
			pts := cur.PointsRemaining
			_, created, err := applyLoyaltyTxnInTx(tx, cur.UserID, pts, models.LoyaltyDebit,
				models.LoyaltyTxnSource("expiry"), nil, "Points expired", "loyalty:expiry:"+cur.ID.String(), nil, cfg)
			if err != nil {
				return err
			}
			if !created {
				return nil // already expired in a prior run
			}
			// consumeBatchesFIFO inside applyLoyaltyTxnInTx already decremented lots FIFO;
			// force THIS lot to zero so a mixed FIFO order can't leave it non-zero.
			return tx.Model(&models.LoyaltyEarnBatch{}).Where("id = ?", cur.ID).Update("points_remaining", 0).Error
		})
		if err != nil {
			log.Printf("loyalty-expiry: lot %s failed: %v", b.ID, err)
			continue
		}
		expired++
	}
	return expired, nil
}

func runLoyaltyExpiryScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("loyalty-expiry: panic recovered: %v", r)
		}
	}()
	if _, err := ExpireLoyaltyBatches(database.DB, time.Now()); err != nil {
		log.Printf("loyalty-expiry: scan failed: %v", err)
	}
}

// StartLoyaltyExpiryCron launches the daily expiry loop. Returns immediately.
func StartLoyaltyExpiryCron(ctx context.Context) {
	go func() {
		runLoyaltyExpiryScan(ctx)
		ticker := time.NewTicker(loyaltyExpiryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runLoyaltyExpiryScan(ctx)
			}
		}
	}()
	log.Println("loyalty-expiry: cron started (interval=24h)")
}

var _ = uuid.Nil // keep uuid import if unused after edits
```

(Note: the `consumeBatchesFIFO` inside the debit may drain a *different* soonest-expiring lot; the explicit `Update points_remaining = 0` on THIS lot keeps the intent — expire exactly the past-due lot. Simpler alternative acceptable to the reviewer: expire per-user in one debit summing all that user's due lots. Implementer picks; the test pins the observable balance + idempotency.)

- [ ] **Step 4: Register the cron** — add one row to `cronJobs()` in `cron_temporal.go`:

```go
		{"loyalty-expiry", loyaltyExpiryInterval, runLoyaltyExpiryScan, StartLoyaltyExpiryCron},
```

- [ ] **Step 5: Run — expect PASS** and `go build ./...`.
- [ ] **Step 6: Commit** — `git commit -m "feat(loyalty): daily point-expiry sweep cron"`

---

### Task 6: Refund reversal of earned points

**Files:**
- Create: `apps/api/services/loyalty_refund.go`
- Modify: `apps/api/handlers/payment.go` (inside the refund persistence tx ~l.1274)
- Test: `apps/api/services/loyalty_refund_test.go`

**Interfaces:**
- Produces: `ReverseOrderLoyalty(tx *gorm.DB, orderID uuid.UUID) error` — debits the *still-unredeemed* points from that order's earn lot (idempotent `loyalty:order-refund:<orderID>`); a no-op if the order earned nothing or the points were already spent.

- [ ] **Step 1: Write the failing test** — `apps/api/services/loyalty_refund_test.go`:

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestReverseOrderLoyalty_DebitsUnredeemedEarn(t *testing.T) {
	db := setupLoyaltyDB(t)
	u, oid := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, subtotal, total) VALUES (?,?,?,?)`,
		oid.String(), u.String(), 500.0, 620.0).Error)
	_, err := AwardOrderLoyalty(db, u, oid) // +50 pts, lot tagged with order_id
	require.NoError(t, err)
	require.Equal(t, 50.0, LoyaltyBalance(db, u))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return ReverseOrderLoyalty(tx, oid) }))
	require.Equal(t, 0.0, LoyaltyBalance(db, u))

	// Idempotent — a second reversal is a no-op.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return ReverseOrderLoyalty(tx, oid) }))
	require.Equal(t, 0.0, LoyaltyBalance(db, u))
}
```

- [ ] **Step 2: Run — expect FAIL** (`ReverseOrderLoyalty` undefined).

- [ ] **Step 3: Implement** — `apps/api/services/loyalty_refund.go`:

```go
package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// ReverseOrderLoyalty claws back the points earned on an order when it is refunded — but only
// the portion still sitting unredeemed in that order's earn lot (if the customer already
// redeemed them to their wallet, the wallet refund handles that side, so we don't over-debit).
// Idempotent per order. Runs inside the caller's refund transaction.
func ReverseOrderLoyalty(tx *gorm.DB, orderID uuid.UUID) error {
	var lot models.LoyaltyEarnBatch
	err := tx.Where("order_id = ? AND source = ?", orderID, models.LoyaltySourceOrder).First(&lot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil // nothing earned on this order
	}
	if err != nil {
		return err
	}
	if lot.PointsRemaining <= 0 {
		return nil // already spent or expired
	}
	cfg := GetLoyaltyConfig(tx)
	oid := orderID
	_, _, err = applyLoyaltyTxnInTx(tx, lot.UserID, lot.PointsRemaining, models.LoyaltyDebit,
		models.LoyaltyTxnSource("refund_reversal"), &oid, "Order refunded — earned points reversed",
		"loyalty:order-refund:"+orderID.String(), nil, cfg)
	return err
}
```

- [ ] **Step 4: Run the service test — expect PASS**.

- [ ] **Step 5: Hook into the refund tx** — in `handlers/payment.go` `InitiateRefund`, inside the `database.DB.Transaction(func(tx *gorm.DB) error { … })` at ~l.1274, after the order `Updates` and before the `EnqueueEvent`, add:

```go
		if err := services.ReverseOrderLoyalty(tx, order.ID); err != nil {
			return err
		}
```

- [ ] **Step 6: Run — `go build ./... && go test ./services/ -run ReverseOrderLoyalty -count=1`** (both pass).
- [ ] **Step 7: Commit** — `git commit -m "feat(loyalty): reverse earned points when an order is refunded"`

---

### Task 7: `RedeemLoyalty` monthly redemption cap

**Files:**
- Modify: `apps/api/services/loyalty.go` (`RedeemLoyalty` ~l.320)
- Test: `apps/api/services/loyalty_redeem_cap_test.go`

**Interfaces:**
- Produces: new sentinel `ErrLoyaltyMonthlyCap = errors.New("monthly redemption cap reached")`; `RedeemLoyalty` rejects a redemption that would push the rolling 30-day redeemed ₹ past `cfg.MonthlyRedeemCap`.

- [ ] **Step 1: Write the failing test** — `apps/api/services/loyalty_redeem_cap_test.go`:

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/homechef/api/models"
)

func TestRedeemLoyalty_MonthlyCap(t *testing.T) {
	db := setupLoyaltyDB(t)
	u := uuid.New()
	cfg := GetLoyaltyConfig(db) // redeem_rate 0.05, monthly cap 300 → cap = 6000 pts of redemption/mo
	// Seed a big balance.
	_, _, err := applyLoyaltyTxnInTx(db, u, 20000, models.LoyaltyCredit, models.LoyaltySourceOrder, nil, "seed", "seed", nil, cfg)
	require.NoError(t, err)

	// Redeem 6000 pts = ₹300 (exactly the cap) — allowed.
	_, _, err = RedeemLoyalty(db, u, 6000)
	require.NoError(t, err)
	// Another 500 pts = ₹25 would exceed ₹300/mo — rejected.
	_, _, err = RedeemLoyalty(db, u, 500)
	require.ErrorIs(t, err, ErrLoyaltyMonthlyCap)
}
```

- [ ] **Step 2: Run — expect FAIL** (`ErrLoyaltyMonthlyCap` undefined).

- [ ] **Step 3: Add the sentinel + the check.** Add to the `var (…)` error block: `ErrLoyaltyMonthlyCap = errors.New("monthly redemption cap reached")`. In `RedeemLoyalty`, after computing `rupees` and before the transaction, add:

```go
	if cfg.MonthlyRedeemCap > 0 {
		var redeemedThisMonth float64
		db.Model(&models.WalletTxn{}).
			Where("user_id = ? AND source = ? AND created_at >= ?", userID, models.WalletSourceLoyalty, time.Now().AddDate(0, 0, -30)).
			Select("COALESCE(SUM(amount),0)").Scan(&redeemedThisMonth)
		if redeemedThisMonth+rupees > cfg.MonthlyRedeemCap+1e-6 {
			return nil, nil, ErrLoyaltyMonthlyCap
		}
	}
```

(Add `"time"` to imports if not present.)

- [ ] **Step 4: Map the error in the handler** — `handlers/loyalty.go` `RedeemLoyalty` switch, add a case:

```go
		case errors.Is(err, services.ErrLoyaltyMonthlyCap):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Monthly redemption limit reached — try again next month"})
```

- [ ] **Step 5: Run — expect PASS** and `go build ./...`.
- [ ] **Step 6: Commit** — `git commit -m "feat(loyalty): enforce monthly redemption cap"`

---

### Task 8: Admin grant/adjust points + `LOYALTY_ENABLED` flag + enable mobile

**Files:**
- Modify: `apps/api/config/config.go` (flag), `apps/api/services/loyalty.go` (`AdminAdjustLoyalty`), `apps/api/handlers/admin_loyalty.go` (`GrantLoyaltyPoints`), `apps/api/routes/routes.go` (route)
- Modify: `apps/mobile-customer/lib/features.ts` (`REWARDS_ENABLED`)
- Test: `apps/api/services/loyalty_admin_test.go`, `apps/api/handlers/loyalty_admin_test.go`

**Interfaces:**
- Produces: `AdminAdjustLoyalty(db, userID uuid.UUID, points float64, reason string, adminID uuid.UUID) (*models.LoyaltyTransaction, error)` — positive `points` = credit, negative = debit; source `admin_adjustment`; stamps `CreatedBy=adminID`. Route `POST /admin/loyalty/grant {userId, points, reason}`.

- [ ] **Step 1: Add the env flag** — `config/config.go`: struct field `LoyaltyEnabled bool` (in the Feature Flags block); in `Load()`: `loyaltyEnabled, _ := strconv.ParseBool(getEnv("LOYALTY_ENABLED", "true"))`; in the `AppConfig = &Config{…}` literal: `LoyaltyEnabled: loyaltyEnabled,`. (Also gate `RedeemLoyalty` with `if config.AppConfig != nil && !config.AppConfig.LoyaltyEnabled { return nil,nil,ErrLoyaltyDisabled }` at its top.)

- [ ] **Step 2: Write the failing service test** — `apps/api/services/loyalty_admin_test.go`:

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAdminAdjustLoyalty_CreditAndDebit(t *testing.T) {
	db := setupLoyaltyDB(t)
	u, admin := uuid.New(), uuid.New()

	_, err := AdminAdjustLoyalty(db, u, 500, "goodwill", admin)
	require.NoError(t, err)
	require.Equal(t, 500.0, LoyaltyBalance(db, u))

	_, err = AdminAdjustLoyalty(db, u, -200, "correction", admin)
	require.NoError(t, err)
	require.Equal(t, 300.0, LoyaltyBalance(db, u))
}
```

- [ ] **Step 3: Run — expect FAIL** (`AdminAdjustLoyalty` undefined).

- [ ] **Step 4: Implement `AdminAdjustLoyalty`** — in `loyalty.go`:

```go
// AdminAdjustLoyalty grants (points>0) or removes (points<0) loyalty points for a user, stamped
// with the acting admin. Non-idempotent by design (each call is a distinct manual adjustment).
func AdminAdjustLoyalty(db *gorm.DB, userID uuid.UUID, points float64, reason string, adminID uuid.UUID) (*models.LoyaltyTransaction, error) {
	if points == 0 {
		return nil, fmt.Errorf("adjustment must be non-zero")
	}
	cfg := GetLoyaltyConfig(db)
	txnType := models.LoyaltyCredit
	amt := points
	if points < 0 {
		txnType = models.LoyaltyDebit
		amt = -points
	}
	var out *models.LoyaltyTransaction
	err := db.Transaction(func(tx *gorm.DB) error {
		aid := adminID
		txn, _, err := applyLoyaltyTxnInTx(tx, userID, amt, txnType, models.LoyaltySourceAdminAdjust, nil, reason, "loyalty-admin:"+uuid.NewString(), &aid, cfg)
		out = txn
		return err
	})
	return out, err
}
```

- [ ] **Step 5: Run — expect PASS**.

- [ ] **Step 6: Add the admin handler** — `handlers/admin_loyalty.go`:

```go
type grantLoyaltyRequest struct {
	UserID string  `json:"userId" binding:"required"`
	Points float64 `json:"points" binding:"required"`
	Reason string  `json:"reason"`
}

// POST /admin/loyalty/grant  { userId, points, reason }  (points<0 removes)
func (h *AdminHandler) GrantLoyaltyPoints(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	var req grantLoyaltyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	uid, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userId"})
		return
	}
	txn, err := services.AdminAdjustLoyalty(database.DB, uid, req.Points, req.Reason, adminID)
	if err != nil {
		if errors.Is(err, services.ErrInsufficientLoyaltyPoints) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User has fewer points than the deduction"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to adjust points"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pointsBalance": txn.PointsAfter})
}
```

(Ensure `uuid`, `errors`, `middleware`, `services`, `database` imports exist in the file.)

- [ ] **Step 7: Register the route** — `routes.go`, right after the `admin.GET("/loyalty/analytics", …)` line: `admin.POST("/loyalty/grant", adminHandler.GrantLoyaltyPoints)`.

- [ ] **Step 8: Enable the mobile screen** — `apps/mobile-customer/lib/features.ts`: set `REWARDS_ENABLED = true` (keep `REFERRAL_ENABLED` off — that's Phase 2). Run `../../node_modules/.bin/tsc --noEmit -p tsconfig.json` (0 errors).

- [ ] **Step 9: Run everything** — `go build ./... && go vet ./services/ ./handlers/ && go test ./services/ ./handlers/ -run 'Loyalty|Redeem|Award|Batch|Expire|Reverse|Admin' -count=1`. All pass.
- [ ] **Step 10: Commit** — `git commit -m "feat(loyalty): admin grant/adjust points, LOYALTY_ENABLED flag, enable mobile rewards"`

---

## Self-review notes
- **Spec coverage:** Phase-1 items (1) economics ✓ T1, (2) earn-on-subtotal ✓ T4, (3) batches+FIFO+sweep ✓ T2/T3/T5, (4) refund reversal ✓ T6, (5) monthly cap ✓ T7 (max_redeem_pct is config-only per Global Constraints, enforced at checkout in Phase 3), (6) admin grant ✓ T8, (7) LOYALTY_ENABLED + mobile ✓ T8. DDL-in-tesserix-k8s enforced (T2 Step 2/7).
- **Idempotency keys used:** `batch:<idempotencyKey>` (lots), `loyalty:order:<id>` (earn), `loyalty:expiry:<batch_id>` (expiry), `loyalty:order-refund:<order_id>` (reversal), `loyalty-admin:<uuid>` (adjust), `loyalty-redeem:<txn_id>` (wallet credit, existing).
- **Type consistency:** `AwardOrderLoyalty(db, userID, orderID)` (T4) — grep all callers when landing T4. `applyLoyaltyTxnInTx` unchanged signature (batches wired inside its body, T3).
- **Harness evolution:** the `loyalty_earn_batches` + `orders` sqlite tables land in `setupLoyaltyDB` (T3 Step 5, T4 Step 1) so all loyalty tests share them.
