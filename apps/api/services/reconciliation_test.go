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

// A legacy razorpay-provider row has no gateway left to reconcile against (#1086), so it
// must be SKIPPED rather than reported as drift — a DriftGatewayUnreachable on every
// historical order would drown the real findings the sweep exists to surface.
func TestReconcileOne_LegacyRazorpayOrder_Skips(t *testing.T) {
	o := &models.Order{
		OrderNumber:       "ORD-LEGACY-RZP",
		PaymentProvider:   models.PaymentProviderRazorpay,
		GatewayPaymentID: "pay_legacy",
	}
	drifts, ok := reconcileOne(o)
	if ok {
		t.Errorf("reconcileOne ok = true, want false for a legacy razorpay order")
	}
	if drifts != nil {
		t.Errorf("drifts = %v, want nil", drifts)
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
