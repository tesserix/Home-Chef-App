package services

import (
	"testing"
	"time"

	"github.com/homechef/api/models"
)

func TestPreviousISTDay(t *testing.T) {
	// 2026-06-10 06:00 UTC = 11:30 IST on Jun 10 → previous IST day is Jun 9.
	now := time.Date(2026, 6, 10, 6, 0, 0, 0, time.UTC)
	start, end := previousISTDay(now)
	wantStart := time.Date(2026, 6, 9, 0, 0, 0, 0, istLoc)
	wantEnd := time.Date(2026, 6, 10, 0, 0, 0, 0, istLoc)
	if !start.Equal(wantStart) {
		t.Errorf("start = %s, want %s", start, wantStart.UTC())
	}
	if !end.Equal(wantEnd) {
		t.Errorf("end = %s, want %s", end, wantEnd.UTC())
	}
	if d := end.Sub(start); d != 24*time.Hour {
		t.Errorf("window = %s, want 24h", d)
	}
}

func TestPreviousISTDay_JustAfterMidnightIST(t *testing.T) {
	// 18:35 UTC = 00:05 IST next day → previous IST day is the UTC day.
	now := time.Date(2026, 6, 9, 18, 35, 0, 0, time.UTC) // 2026-06-10 00:05 IST
	start, _ := previousISTDay(now)
	wantStart := time.Date(2026, 6, 9, 0, 0, 0, 0, istLoc)
	if !start.Equal(wantStart) {
		t.Errorf("start = %s, want %s", start, wantStart.UTC())
	}
}

// A row stamped with the retired INR gateway has no gateway left to reconcile
// against (#1086), so it must be SKIPPED rather than reported as drift — a
// DriftGatewayUnreachable on every historical order would drown the real
// findings the sweep exists to surface. It normalizes to Cashfree and carries no
// Cashfree order id, which is exactly what makes it a skip (#1132).
func TestReconcileOne_LegacyGatewayOrder_Skips(t *testing.T) {
	o := &models.Order{
		OrderNumber:      "ORD-LEGACY",
		PaymentProvider:  retiredGateway,
		GatewayPaymentID: "pay_legacy",
	}
	drifts, ok := reconcileOne(o)
	if ok {
		t.Errorf("reconcileOne ok = true, want false for a retired-gateway order")
	}
	if drifts != nil {
		t.Errorf("drifts = %v, want nil", drifts)
	}
}

// A test-partition order was paid with the TEST slot's credentials, so it can
// only be reconciled against the test merchant account. Asking the live account
// about it returns no such order, and the sweep reports DriftPaymentNotCaptured
// on money that was in fact captured — a false finding on every sandbox order,
// which is what the drift report exists to not do. models.ModePartition states
// the rule: a money operation reads the row's own Mode, never the chef's
// current one.
func TestReconcileCashfree_AsksTheSlotTheOrderWasPaidOn(t *testing.T) {
	for _, mode := range []string{models.ChefModeTest, models.ChefModeLive} {
		t.Run(mode, func(t *testing.T) {
			orig := cashfreeClientFor
			t.Cleanup(func() { cashfreeClientFor = orig })

			asked := ""
			cashfreeClientFor = func(m string) *CashfreeClient {
				asked = m
				return nil // unconfigured slot — reconcileCashfree skips, no network
			}

			o := &models.Order{OrderNumber: "ORD-MODE", GatewayOrderID: "cf_order_1"}
			o.Mode = mode
			if drifts := reconcileCashfree(o); drifts != nil {
				t.Errorf("drifts = %v, want nil when the slot is unconfigured", drifts)
			}
			if asked != mode {
				t.Errorf("reconciled against the %q slot, want %q", asked, mode)
			}
		})
	}
}

func TestReconcileOne_NoGatewayRef_Skips(t *testing.T) {
	// An order with no gateway reference can't be reconciled — the bool must be
	// false (a skip, not a drift) without touching any gateway.
	o := &models.Order{OrderNumber: "ORD-NO-REF"}
	drifts, ok := reconcileOne(o)
	if ok {
		t.Errorf("reconcileOne ok = true, want false for order with no gateway ref")
	}
	if drifts != nil {
		t.Errorf("drifts = %v, want nil", drifts)
	}
}
