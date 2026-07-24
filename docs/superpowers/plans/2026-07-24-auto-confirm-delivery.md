# Auto-confirm Delivery Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When an order is delivered but the customer never confirms receipt, send up to 3 reminder pushes (every 10 min) then auto-confirm on their behalf — durably, via a Temporal workflow + NATS — marking the order fulfilled (`release_eligible`) so the existing payout sweep pays the chef on its own schedule.

**Architecture:** A per-order Temporal workflow (`homechef:confirm:<orderID>`) started at the two delivered-transition sites, gated behind a config flag. It loops a 10-min timer up to 3×, firing a reminder activity each time; a `order.confirmed` or `order.disputed` signal ends it early; on exhaustion an auto-confirm activity calls the *existing* idempotent `ConfirmOrderHold`. Money release stays decoupled (existing governor). The existing 24h `payout-auto-confirm` cron remains the fallback.

**Tech Stack:** Go 1.26, Temporal (`go.temporal.io/sdk`), GORM, NATS JetStream (via transactional outbox), FCM push.

## Global Constraints

- **Feature flag, default OFF:** everything gated behind `config.AppConfig.ConfirmReceiptFlowEnabled` (env `CONFIRM_RECEIPT_FLOW_ENABLED`, default `false`) AND `temporalRT != nil`. Mirror `sagaActive()` in `services/temporal_order.go`.
- **Confirm ≠ release:** the workflow only advances the hold to `release_eligible` (or `disputed`) via `services.ConfirmOrderHold`. It MUST NOT call any release/`ReleaseHold`/payout-move code. Money is released by the existing `payout-release` sweep.
- **Idempotent everywhere:** workflow ID keyed per order; `ConfirmOrderHold` is guarded on `customer_confirmed_at IS NULL`; the auto-confirm activity re-reads order state before acting.
- **Reuse, don't duplicate:** `services.ConfirmOrderHold(db, order)` (payout_hold.go:280), `services.SendPushNotification(userID, title, body, data)` (push.go:255), `services.EnqueueEvent(tx, subject, eventType, userID, data)` (outbox.go:68). Follow `OrderSagaWorkflow` (temporal/workflows/order.go) for timer+signal, and the `StartOrderSaga`/`signalOrderSaga` trio (temporal_order.go:34-74) for the producer.
- **Do not change** the 24h `payout-auto-confirm` cron (`services/payout_auto_confirm_cron.go`); it is the belt-and-suspenders fallback.
- **Reminder cadence source of truth:** interval/count are read once from PlatformSettings at workflow *start* and passed as workflow input (a running workflow stays deterministic). Keys `payout.confirm_reminder_interval_minutes` (default 10), `payout.confirm_reminder_max_count` (default 3), read via the `GetCustomerConfirmWindowHours` fold pattern (payout_hold.go:388).

---

### Task 1: `AutoConfirmOrderReceipt` service + settings accessors

**Files:**
- Create: `apps/api/services/confirm_receipt.go`
- Test: `apps/api/services/confirm_receipt_test.go`

**Interfaces:**
- Consumes: `ConfirmOrderHold(db *gorm.DB, order *models.Order) (models.PayoutHoldStatus, error)`; `models.Order` (fields `PayoutHoldStatus`, `CustomerConfirmedAt`, `Status`, `RefundedAt`); `models.PayoutHoldAwaitingConfirmation`, `models.OrderStatusDelivered`.
- Produces:
  - `func AutoConfirmOrderReceipt(db *gorm.DB, orderID uuid.UUID) (models.PayoutHoldStatus, bool, error)` — returns `(status, acted, err)`. `acted=false` (no error) when the order is already confirmed / not awaiting / terminal / refunded.
  - `func ConfirmReminderIntervalMinutes(db *gorm.DB) int` (default 10, key `payout.confirm_reminder_interval_minutes`).
  - `func ConfirmReminderMaxCount(db *gorm.DB) int` (default 3, key `payout.confirm_reminder_max_count`).

- [ ] **Step 1: Write the failing test**

```go
package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
)

func TestAutoConfirmOrderReceipt_AwaitingOrder_Confirms(t *testing.T) {
	db := newTestDB(t) // existing helper used by other services tests
	order := models.Order{
		ID:               uuid.New(),
		Status:           models.OrderStatusDelivered,
		PayoutHoldStatus: models.PayoutHoldAwaitingConfirmation,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}

	status, acted, err := AutoConfirmOrderReceipt(db, order.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !acted {
		t.Fatalf("expected acted=true")
	}
	if status != models.PayoutHoldReleaseEligible {
		t.Fatalf("got %q want release_eligible", status)
	}

	var reloaded models.Order
	db.First(&reloaded, "id = ?", order.ID)
	if reloaded.CustomerConfirmedAt == nil {
		t.Fatal("expected customer_confirmed_at stamped")
	}
}

func TestAutoConfirmOrderReceipt_AlreadyConfirmed_NoOp(t *testing.T) {
	db := newTestDB(t)
	now := timeNow()
	order := models.Order{
		ID:                  uuid.New(),
		Status:              models.OrderStatusDelivered,
		PayoutHoldStatus:    models.PayoutHoldReleaseEligible,
		CustomerConfirmedAt: &now,
	}
	db.Create(&order)

	_, acted, err := AutoConfirmOrderReceipt(db, order.ID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if acted {
		t.Fatal("expected acted=false for already-confirmed order")
	}
}
```

> **Implementer note:** find the existing services test DB helper (grep `func newTestDB` / how `payout_hold_test.go` sets up gorm+sqlite). Reuse it verbatim; do not invent a new harness. `timeNow()` — use whatever "now" helper the package already uses, or `time.Now()`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && GOFLAGS=-mod=mod go test ./services/ -run TestAutoConfirmOrderReceipt -count=1`
Expected: FAIL — `AutoConfirmOrderReceipt` undefined.

- [ ] **Step 3: Write minimal implementation**

```go
package services

import (
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// AutoConfirmOrderReceipt confirms receipt on the customer's behalf after the
// reminder window elapses. It re-reads the order and only acts on an order that
// is still awaiting confirmation and not terminal/refunded, then delegates to
// the SAME ConfirmOrderHold transition the customer's confirm button uses
// (→ release_eligible, or → disputed if an issue is open). It moves no money;
// the existing payout-release sweep does that on its own schedule.
//
// Returns (status, acted, err). acted=false with a nil error means "nothing to
// do" (already confirmed / not awaiting / terminal / refunded).
func AutoConfirmOrderReceipt(db *gorm.DB, orderID uuid.UUID) (models.PayoutHoldStatus, bool, error) {
	var order models.Order
	if err := db.First(&order, "id = ?", orderID).Error; err != nil {
		return "", false, err
	}
	if order.CustomerConfirmedAt != nil ||
		order.PayoutHoldStatus != models.PayoutHoldAwaitingConfirmation ||
		order.RefundedAt != nil ||
		order.Status == models.OrderStatusCancelled ||
		order.Status == models.OrderStatusRefunded {
		return order.PayoutHoldStatus, false, nil
	}
	status, err := ConfirmOrderHold(db, &order)
	if err != nil {
		return "", false, err
	}
	return status, true, nil
}

func confirmReminderSetting(db *gorm.DB, key string, def int) int {
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "payout.%").Find(&settings)
	for _, s := range settings {
		if s.Key == key {
			if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
				return v
			}
		}
	}
	return def
}

// ConfirmReminderIntervalMinutes is the gap between confirm-receipt reminders.
func ConfirmReminderIntervalMinutes(db *gorm.DB) int {
	return confirmReminderSetting(db, "payout.confirm_reminder_interval_minutes", 10)
}

// ConfirmReminderMaxCount is how many reminders fire before auto-confirm.
func ConfirmReminderMaxCount(db *gorm.DB) int {
	return confirmReminderSetting(db, "payout.confirm_reminder_max_count", 3)
}
```

> **Implementer note:** confirm the exact `models.Order` field names (`CustomerConfirmedAt`, `RefundedAt`, `PayoutHoldStatus`) and `PayoutHoldStatus` constants in `models/order.go` + `models/payout_hold.go` before writing; adjust if they differ. If `newTestDB` doesn't auto-migrate `PlatformSettings`, the setting tests aren't in this task — only the auto-confirm tests above are required here.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/api && GOFLAGS=-mod=mod go test ./services/ -run TestAutoConfirmOrderReceipt -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/confirm_receipt.go apps/api/services/confirm_receipt_test.go
git commit -m "feat(api): AutoConfirmOrderReceipt + reminder settings accessors"
```

---

### Task 2: `SendConfirmReceiptReminder` service

**Files:**
- Modify: `apps/api/services/confirm_receipt.go`
- Test: `apps/api/services/confirm_receipt_test.go`

**Interfaces:**
- Consumes: `SendPushNotification(userID uuid.UUID, title, body string, data map[string]string) error` (push.go:255); `EnqueueEvent(tx *gorm.DB, subject, eventType string, userID uuid.UUID, data map[string]any) error` (outbox.go:68); `SubjectOrderConfirmReminder` (added Task 5 — for now use the literal `"orders.confirm_reminder"`, replaced by the const in Task 5); `models.Order` (`CustomerID`, `ChefID`, `PayoutHoldStatus`, `Status`), `models.ChefProfile` (`BusinessName`).
- Produces: `func SendConfirmReceiptReminder(db *gorm.DB, orderID uuid.UUID, attempt int) (bool, error)` — returns `(sent, err)`. `sent=false` when the order is no longer awaiting confirmation (already confirmed / disputed / terminal).

- [ ] **Step 1: Write the failing test**

```go
func TestSendConfirmReceiptReminder_SkipsConfirmed(t *testing.T) {
	db := newTestDB(t)
	now := timeNow()
	order := models.Order{
		ID:                  uuid.New(),
		Status:              models.OrderStatusDelivered,
		PayoutHoldStatus:    models.PayoutHoldReleaseEligible,
		CustomerConfirmedAt: &now,
		CustomerID:          uuid.New(),
	}
	db.Create(&order)

	sent, err := SendConfirmReceiptReminder(db, order.ID, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sent {
		t.Fatal("expected sent=false for a confirmed order")
	}
}

func TestSendConfirmReceiptReminder_AwaitingSends(t *testing.T) {
	db := newTestDB(t)
	order := models.Order{
		ID:               uuid.New(),
		Status:           models.OrderStatusDelivered,
		PayoutHoldStatus: models.PayoutHoldAwaitingConfirmation,
		CustomerID:       uuid.New(),
		ChefID:           uuid.New(),
	}
	db.Create(&order)

	// Customer has no FCM token in the test DB, so SendPushNotification no-ops
	// (returns nil). The reminder still counts as "sent" (attempted) and stages
	// the outbox event — assert no error + sent=true.
	sent, err := SendConfirmReceiptReminder(db, order.ID, 1)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !sent {
		t.Fatal("expected sent=true for an awaiting order")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && GOFLAGS=-mod=mod go test ./services/ -run TestSendConfirmReceiptReminder -count=1`
Expected: FAIL — undefined.

- [ ] **Step 3: Write minimal implementation** (append to `confirm_receipt.go`)

```go
import (
	"fmt"
	// ...existing imports
)

// SendConfirmReceiptReminder nudges the customer to confirm they received their
// order. Best-effort push (SendPushNotification no-ops on a missing token) plus
// a transactional-outbox event. Skips an order that is no longer awaiting
// confirmation. Returns (sent, err); sent=false means "nothing to remind".
func SendConfirmReceiptReminder(db *gorm.DB, orderID uuid.UUID, attempt int) (bool, error) {
	var order models.Order
	if err := db.First(&order, "id = ?", orderID).Error; err != nil {
		return false, err
	}
	if order.CustomerConfirmedAt != nil ||
		order.PayoutHoldStatus != models.PayoutHoldAwaitingConfirmation {
		return false, nil
	}

	chefName := "your chef"
	var chef models.ChefProfile
	if err := db.First(&chef, "id = ?", order.ChefID).Error; err == nil && chef.BusinessName != "" {
		chefName = chef.BusinessName
	}

	title := "Did your order arrive?"
	body := fmt.Sprintf("Tap to confirm you received your order from %s.", chefName)
	data := map[string]string{
		"order_id": order.ID.String(),
		"type":     "confirm_receipt",
	}
	// Best-effort: a missing/invalid token is not a failure of the reminder.
	_ = SendPushNotification(order.CustomerID, title, body, data)

	if err := EnqueueEvent(db, "orders.confirm_reminder", "order.confirm_reminder", order.CustomerID, map[string]any{
		"order_id": order.ID.String(),
		"attempt":  attempt,
	}); err != nil {
		return true, err
	}
	return true, nil
}
```

> **Implementer note:** verify `ChefProfile.BusinessName` is the right field for the chef's display name (grep how other pushes name the chef, e.g. `pushChefNewOrder`). Verify `EnqueueEvent`'s exact signature/arg order in `outbox.go`; adapt the call. Replace the literal `"orders.confirm_reminder"` with `SubjectOrderConfirmReminder` once Task 5 defines it.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/api && GOFLAGS=-mod=mod go test ./services/ -run TestSendConfirmReceiptReminder -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/confirm_receipt.go apps/api/services/confirm_receipt_test.go
git commit -m "feat(api): SendConfirmReceiptReminder push + outbox event"
```

---

### Task 3: `ConfirmReceiptWorkflow` + activities

**Files:**
- Create: `apps/api/temporal/workflows/confirm_receipt.go`
- Test: `apps/api/temporal/workflows/confirm_receipt_test.go`

**Interfaces:**
- Consumes: `apitemporal.Activities(ctx workflow.Context, d time.Duration) workflow.Context` (temporal/retry.go:25); the timer+signal pattern from `OrderSagaWorkflow` (workflows/order.go).
- Produces (all in package `workflows`):
  - Signal name consts: `SignalOrderConfirmed = "order.confirmed"`, `SignalOrderDisputed = "order.disputed"`.
  - `type ConfirmReceiptInput struct { OrderID uuid.UUID; ReminderIntervalSeconds int; MaxReminders int }`.
  - `var ConfirmReminderFunc func(ctx context.Context, orderID uuid.UUID, attempt int) error` and `var AutoConfirmFunc func(ctx context.Context, orderID uuid.UUID) error` (transport seams, wired by the worker in Task 5).
  - `func ReminderActivity(ctx context.Context, in ReminderActivityInput) error` where `ReminderActivityInput struct { OrderID uuid.UUID; Attempt int }`.
  - `func AutoConfirmActivity(ctx context.Context, orderID uuid.UUID) error`.
  - `func ConfirmReceiptWorkflow(ctx workflow.Context, in ConfirmReceiptInput) error`.

- [ ] **Step 1: Write the failing test** (Temporal test env — mirror any existing `*_workflow_test.go` in this package for the harness)

```go
package workflows

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestConfirmReceiptWorkflow_NoAction_AutoConfirms(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	reminders := 0
	autoConfirmed := 0
	env.RegisterActivity(ReminderActivity)
	env.RegisterActivity(AutoConfirmActivity)
	ConfirmReminderFunc = func(_ context.Context, _ uuid.UUID, _ int) error { reminders++; return nil }
	AutoConfirmFunc = func(_ context.Context, _ uuid.UUID) error { autoConfirmed++; return nil }

	env.ExecuteWorkflow(ConfirmReceiptWorkflow, ConfirmReceiptInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 600, MaxReminders: 3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 3, reminders)
	require.Equal(t, 1, autoConfirmed)
}

func TestConfirmReceiptWorkflow_ConfirmedSignal_StopsEarly(t *testing.T) {
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()

	reminders := 0
	autoConfirmed := 0
	env.RegisterActivity(ReminderActivity)
	env.RegisterActivity(AutoConfirmActivity)
	ConfirmReminderFunc = func(_ context.Context, _ uuid.UUID, _ int) error { reminders++; return nil }
	AutoConfirmFunc = func(_ context.Context, _ uuid.UUID) error { autoConfirmed++; return nil }

	// Fire the confirmed signal ~15 min in (after the first reminder at +10m).
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SignalOrderConfirmed, nil)
	}, 15*time.Minute)

	env.ExecuteWorkflow(ConfirmReceiptWorkflow, ConfirmReceiptInput{
		OrderID: uuid.New(), ReminderIntervalSeconds: 600, MaxReminders: 3,
	})

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, 1, reminders)
	require.Equal(t, 0, autoConfirmed)
}
```

> **Implementer note:** check the module's testify availability (`grep stretchr/testify go.mod`); if absent, use plain `t.Fatalf` assertions. Verify the Temporal SDK test-suite import path matches what other workflow tests use.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && GOFLAGS=-mod=mod go test ./temporal/workflows/ -run TestConfirmReceiptWorkflow -count=1`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write minimal implementation**

```go
package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"

	apitemporal "github.com/homechef/api/temporal"
)

const (
	SignalOrderConfirmed = "order.confirmed"
	SignalOrderDisputed  = "order.disputed"
)

type ConfirmReceiptInput struct {
	OrderID                  uuid.UUID
	ReminderIntervalSeconds  int
	MaxReminders             int
}

type ReminderActivityInput struct {
	OrderID uuid.UUID
	Attempt int
}

// Transport seams — the worker wires these to services.* (Task 5). Nil in
// unit tests unless the test overrides them.
var (
	ConfirmReminderFunc func(ctx context.Context, orderID uuid.UUID, attempt int) error
	AutoConfirmFunc     func(ctx context.Context, orderID uuid.UUID) error
)

func ReminderActivity(ctx context.Context, in ReminderActivityInput) error {
	if ConfirmReminderFunc == nil {
		return nil
	}
	return ConfirmReminderFunc(ctx, in.OrderID, in.Attempt)
}

func AutoConfirmActivity(ctx context.Context, orderID uuid.UUID) error {
	if AutoConfirmFunc == nil {
		return nil
	}
	return AutoConfirmFunc(ctx, orderID)
}

// ConfirmReceiptWorkflow reminds the customer to confirm receipt up to
// MaxReminders times (one every ReminderIntervalSeconds), then auto-confirms if
// they never act. A confirmed/disputed signal ends it early. It advances the
// hold to release_eligible only — money release stays with the payout sweep.
func ConfirmReceiptWorkflow(ctx workflow.Context, in ConfirmReceiptInput) error {
	interval := time.Duration(in.ReminderIntervalSeconds) * time.Second
	confirmedCh := workflow.GetSignalChannel(ctx, SignalOrderConfirmed)
	disputedCh := workflow.GetSignalChannel(ctx, SignalOrderDisputed)

	actx := apitemporal.Activities(ctx, 30*time.Second)

	for attempt := 1; attempt <= in.MaxReminders; attempt++ {
		done := false
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(confirmedCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); done = true })
		sel.AddReceive(disputedCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); done = true })
		sel.AddFuture(workflow.NewTimer(ctx, interval), func(workflow.Future) {})
		sel.Select(ctx)
		if done {
			return nil // customer confirmed or a dispute opened
		}
		// Timer fired → send this attempt's reminder (best-effort).
		_ = workflow.ExecuteActivity(actx, ReminderActivity, ReminderActivityInput{
			OrderID: in.OrderID, Attempt: attempt,
		}).Get(ctx, nil)
	}

	// Reminders exhausted, no confirmation → auto-confirm.
	return workflow.ExecuteActivity(actx, AutoConfirmActivity, in.OrderID).Get(ctx, nil)
}
```

> **Implementer note:** confirm `apitemporal.Activities` is the right helper + import alias (temporal/retry.go). If the reminder activity erroring should not abort the workflow, keep the `_ =`/best-effort on reminders but propagate the auto-confirm error (as written). Match the selector idiom to `OrderSagaWorkflow` exactly.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/api && GOFLAGS=-mod=mod go test ./temporal/workflows/ -run TestConfirmReceiptWorkflow -count=1`
Expected: PASS (both cases).

- [ ] **Step 5: Commit**

```bash
git add apps/api/temporal/workflows/confirm_receipt.go apps/api/temporal/workflows/confirm_receipt_test.go
git commit -m "feat(api): ConfirmReceiptWorkflow — reminder loop + auto-confirm"
```

---

### Task 4: Producer + gating (`temporal_confirm.go`)

**Files:**
- Create: `apps/api/services/temporal_confirm.go`
- Modify: `apps/api/config/config.go` (add `ConfirmReceiptFlowEnabled bool` from env `CONFIRM_RECEIPT_FLOW_ENABLED`, mirroring `OrderSagaEnabled`)
- Test: `apps/api/services/temporal_confirm_test.go`

**Interfaces:**
- Consumes: `temporalRT` (package-level in `services`, set by `SetTemporalRuntime`); `config.AppConfig.ConfirmReceiptFlowEnabled`; `apitemporal.TaskQueueOrders`; `workflows.ConfirmReceiptWorkflow`, `workflows.ConfirmReceiptInput`, `workflows.SignalOrderConfirmed`, `workflows.SignalOrderDisputed`; `ConfirmReminderIntervalMinutes`, `ConfirmReminderMaxCount` (Task 1); `database.DB`.
- Produces:
  - `func StartConfirmReceiptFlow(orderID uuid.UUID)` — gated, idempotent start.
  - `func SignalOrderConfirmedFlow(orderID uuid.UUID)` and `func SignalOrderDisputedFlow(orderID uuid.UUID)` — best-effort signals.

- [ ] **Step 1: Write the failing test**

```go
func TestStartConfirmReceiptFlow_NoOpWhenDisabled(t *testing.T) {
	// temporalRT is nil in unit tests and the flag defaults off → must not panic.
	StartConfirmReceiptFlow(uuid.New())
	SignalOrderConfirmedFlow(uuid.New())
	SignalOrderDisputedFlow(uuid.New())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/api && GOFLAGS=-mod=mod go test ./services/ -run TestStartConfirmReceiptFlow -count=1`
Expected: FAIL — undefined.

- [ ] **Step 3: Write minimal implementation**

First add the config flag (mirror `OrderSagaEnabled` in `config/config.go` — find its struct field + `getEnvBool`/equivalent load line and add alongside):

```go
// in the config struct:
ConfirmReceiptFlowEnabled bool
// in the loader (next to OrderSagaEnabled):
ConfirmReceiptFlowEnabled: getEnvBool("CONFIRM_RECEIPT_FLOW_ENABLED", false),
```

Then `services/temporal_confirm.go`:

```go
package services

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

func confirmFlowID(orderID uuid.UUID) string { return "homechef:confirm:" + orderID.String() }

func confirmFlowActive() bool {
	return temporalRT != nil && config.AppConfig != nil && config.AppConfig.ConfirmReceiptFlowEnabled
}

// StartConfirmReceiptFlow durably starts the reminder+auto-confirm workflow for
// a delivered order. Idempotent on the order-keyed workflow ID; no-op when the
// flow is disabled or Temporal is down (the 24h cron remains the fallback).
func StartConfirmReceiptFlow(orderID uuid.UUID) {
	if !confirmFlowActive() {
		return
	}
	in := workflows.ConfirmReceiptInput{
		OrderID:                 orderID,
		ReminderIntervalSeconds: ConfirmReminderIntervalMinutes(database.DB) * 60,
		MaxReminders:            ConfirmReminderMaxCount(database.DB),
	}
	if _, err := temporalRT.Start(context.Background(), apitemporal.TaskQueueOrders, confirmFlowID(orderID), workflows.ConfirmReceiptWorkflow, in); err != nil {
		log.Printf("confirm-receipt flow: start failed for %s: %v", orderID, err)
	}
}

func signalConfirmFlow(orderID uuid.UUID, signal string) {
	if !confirmFlowActive() {
		return
	}
	if err := temporalRT.Signal(context.Background(), confirmFlowID(orderID), signal, nil); err != nil {
		log.Printf("confirm-receipt flow: signal %s for %s dropped: %v", signal, orderID, err)
	}
}

// SignalOrderConfirmedFlow tells a running flow the customer confirmed.
func SignalOrderConfirmedFlow(orderID uuid.UUID) {
	signalConfirmFlow(orderID, workflows.SignalOrderConfirmed)
}

// SignalOrderDisputedFlow tells a running flow a dispute opened.
func SignalOrderDisputedFlow(orderID uuid.UUID) {
	signalConfirmFlow(orderID, workflows.SignalOrderDisputed)
}
```

> **Implementer note:** confirm the package-level Temporal runtime var name (`temporalRT`) and `SetTemporalRuntime` in `services/temporal_order.go`; reuse the same var. Confirm `config.getEnvBool` (or the actual helper name) and `AppConfig` field style by reading the `OrderSagaEnabled` definition.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/api && GOFLAGS=-mod=mod go test ./services/ -run TestStartConfirmReceiptFlow -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/services/temporal_confirm.go apps/api/services/temporal_confirm_test.go apps/api/config/config.go
git commit -m "feat(api): gated producer for the confirm-receipt flow"
```

---

### Task 5: Worker registration + transports + NATS subjects

**Files:**
- Modify: `apps/api/cmd/worker/main.go` (register workflow+activities on `TaskQueueOrders`; wire transports)
- Modify: `apps/api/services/nats.go` (add subject consts + replace the literal in Task 2)

**Interfaces:**
- Consumes: `temporal.Queue(...).Workflows(...).Activities(...)` builder (temporal/worker.go); `workflows.ConfirmReceiptWorkflow`, `workflows.ReminderActivity`, `workflows.AutoConfirmActivity`, `workflows.ConfirmReminderFunc`, `workflows.AutoConfirmFunc`; `services.SendConfirmReceiptReminder`, `services.AutoConfirmOrderReceipt`, `database.DB`.
- Produces: `SubjectOrderConfirmReminder = "orders.confirm_reminder"`, `SubjectOrderAutoConfirmed = "orders.auto_confirmed"`.

- [ ] **Step 1: Add NATS subject consts** in `services/nats.go` (in the `orders.*` const block), then replace the Task-2 literal `"orders.confirm_reminder"` with `SubjectOrderConfirmReminder`.

```go
SubjectOrderConfirmReminder = "orders.confirm_reminder"
SubjectOrderAutoConfirmed   = "orders.auto_confirmed"
```

- [ ] **Step 2: Wire transports** in `cmd/worker/main.go` (next to the existing `workflows.SendFunc = ...` block):

```go
workflows.ConfirmReminderFunc = func(_ context.Context, orderID uuid.UUID, attempt int) error {
	_, err := services.SendConfirmReceiptReminder(database.DB, orderID, attempt)
	return err
}
workflows.AutoConfirmFunc = func(_ context.Context, orderID uuid.UUID) error {
	_, _, err := services.AutoConfirmOrderReceipt(database.DB, orderID)
	return err
}
```

- [ ] **Step 3: Register on the orders queue.** Extend the existing `temporal.Queue(temporal.TaskQueueOrders)` registration to also list the new workflow + activities (a task queue may host multiple workflows/activities):

```go
temporal.Queue(temporal.TaskQueueOrders).
	Workflows(workflows.OrderSagaWorkflow, workflows.ConfirmReceiptWorkflow).
	Activities(workflows.NotifyChefActivity, workflows.DispatchDeliveryActivity,
		workflows.OrderSettleActivity, workflows.OrderRefundActivity,
		workflows.ReminderActivity, workflows.AutoConfirmActivity),
```

- [ ] **Step 4: Build the worker + API.**

Run: `cd apps/api && GOFLAGS=-mod=mod go build ./cmd/worker/ ./services/ ./temporal/... `
Expected: exit 0.

- [ ] **Step 5: Run the whole suite to confirm nothing regressed.**

Run: `cd apps/api && GOFLAGS=-mod=mod go test ./services/ ./temporal/... ./handlers/ -count=1`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/cmd/worker/main.go apps/api/services/nats.go apps/api/services/confirm_receipt.go
git commit -m "feat(api): register confirm-receipt workflow + wire transports + NATS subjects"
```

---

### Task 6: Trigger at delivered sites + signals from confirm/dispute

**Files:**
- Modify: `apps/api/handlers/chefs.go` (self-delivery delivered transition — where `services.SetOrderHoldAwaitingConfirmation(database.DB, order.ID)` is called, ~line 1457)
- Modify: `apps/api/handlers/delivery.go` (3PL `DeliveryDelivered` — where `SetOrderHoldAwaitingConfirmation(database.DB, delivery.OrderID)` is called, ~line 647)
- Modify: `apps/api/handlers/payout_hold.go` (`ConfirmOrderReceived` — after `ConfirmOrderHold` succeeds, signal confirmed)
- Modify: the dispute/issue-open path (grep for where an `OrderIssue` with `IssuePending` is created for an order — e.g. `services/…` report-issue / delivery-failure; signal disputed there)

**Interfaces:**
- Consumes: `services.StartConfirmReceiptFlow(orderID)`, `services.SignalOrderConfirmedFlow(orderID)`, `services.SignalOrderDisputedFlow(orderID)` (Task 4).
- Produces: none (wiring only).

- [ ] **Step 1: Start the flow at both delivered sites.** Immediately after each `SetOrderHoldAwaitingConfirmation(...)` call, add:

```go
services.StartConfirmReceiptFlow(order.ID) // chefs.go — use the local order var
// and in delivery.go:
services.StartConfirmReceiptFlow(delivery.OrderID)
```

- [ ] **Step 2: Signal confirmed from the manual confirm handler.** In `handlers/payout_hold.go` `ConfirmOrderReceived`, after `services.ConfirmOrderHold` returns without error, add:

```go
services.SignalOrderConfirmedFlow(order.ID)
```

- [ ] **Step 3: Signal disputed where a dispute opens.** Grep for the order-issue/dispute creation site (`models.IssuePending`, `IssueDeliveryFailed`, report-issue handler). After the issue is persisted, add:

```go
services.SignalOrderDisputedFlow(orderID) // use the site's order id var
```

> **Implementer note:** these are all best-effort no-ops when the flow is off, so they are safe to add unconditionally. If the dispute site is ambiguous or spans multiple handlers, signal at the single most authoritative "issue created" point; missing a dispute signal only means the workflow's AutoConfirmActivity re-reads state and routes to `disputed` via `ConfirmOrderHold` anyway (safe).

- [ ] **Step 4: Build + full suite.**

Run: `cd apps/api && GOFLAGS=-mod=mod go build ./... && GOFLAGS=-mod=mod go test ./handlers/ ./services/ ./temporal/... -count=1`
Expected: exit 0, PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/api/handlers/chefs.go apps/api/handlers/delivery.go apps/api/handlers/payout_hold.go
git commit -m "feat(api): trigger confirm-receipt flow on delivery + signal on confirm/dispute"
```

---

## Self-Review

**Spec coverage:** Trigger (Task 6) ✓; workflow reminder-loop + early-exit signals (Task 3) ✓; auto-confirm → release_eligible via ConfirmOrderHold (Task 1) ✓; reminders push+outbox (Task 2) ✓; producer gating (Task 4) ✓; worker registration + NATS subjects + config flag (Tasks 4-5) ✓; reconciliation with 24h cron (Global Constraints — cron untouched) ✓; config keys (Task 1) ✓; idempotency (Tasks 1,3,4) ✓; money-decoupling (Global Constraints + Task 1 note) ✓.

**Placeholder scan:** implementer notes flag the few spots requiring a codebase read (exact model field names, test-DB helper, config bool helper, dispute site) — these are verification directives, not placeholders; every code step contains real code.

**Type consistency:** `ConfirmReceiptInput{OrderID, ReminderIntervalSeconds, MaxReminders}` used identically in Tasks 3 & 4; `ConfirmReminderFunc`/`AutoConfirmFunc` signatures match between Task 3 (declare) and Task 5 (wire); `AutoConfirmOrderReceipt` returns `(status, acted, err)` in Task 1 and is called with `_, _, err` in Task 5; `SendConfirmReceiptReminder` returns `(sent, err)` in Task 2 and called `_, err` in Task 5; signal consts `SignalOrderConfirmed/Disputed` shared Tasks 3→4.
