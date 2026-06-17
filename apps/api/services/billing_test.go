package services

// billing_test.go — deterministic unit tests for subscription proration
// (partial coverage of #13: the "credit/proration correct" money math on
// cancel and plan-change). calculateProratedRefundAt takes an explicit clock
// so every branch is testable without a DB, gateway, or wall-clock flakiness.
//
// A day-15 reference date is used so the yearly month-stepping loop never hits
// end-of-month normalization (e.g. Jan-31 + 1mo).

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/homechef/api/models"
)

func tptr(t time.Time) *time.Time { return &t }

var refNow = time.Date(2026, time.June, 15, 12, 0, 0, 0, time.UTC)

func TestCalculateProratedRefundAt_NilPeriodEnd_Zero(t *testing.T) {
	sub := &models.Subscription{
		BillingInterval:    models.BillingMonthly,
		PlanAmount:         1000,
		CurrentPeriodStart: tptr(refNow.AddDate(0, 0, -10)),
		CurrentPeriodEnd:   nil,
	}
	assert.Equal(t, 0.0, calculateProratedRefundAt(sub, refNow))
}

func TestCalculateProratedRefundAt_PeriodAlreadyEnded_Zero(t *testing.T) {
	sub := &models.Subscription{
		BillingInterval:    models.BillingMonthly,
		PlanAmount:         1000,
		CurrentPeriodStart: tptr(refNow.AddDate(0, 0, -40)),
		CurrentPeriodEnd:   tptr(refNow.AddDate(0, 0, -1)), // ended yesterday
	}
	assert.Equal(t, 0.0, calculateProratedRefundAt(sub, refNow))
}

func TestCalculateProratedRefundAt_Monthly_HalfRemaining(t *testing.T) {
	sub := &models.Subscription{
		BillingInterval:    models.BillingMonthly,
		PlanAmount:         1000,
		CurrentPeriodStart: tptr(refNow.AddDate(0, 0, -15)),
		CurrentPeriodEnd:   tptr(refNow.AddDate(0, 0, 15)),
	}
	// 15 of 30 days remain → half the plan amount.
	assert.InDelta(t, 500.0, calculateProratedRefundAt(sub, refNow), 1e-6)
}

func TestCalculateProratedRefundAt_Monthly_FullRemaining(t *testing.T) {
	sub := &models.Subscription{
		BillingInterval:    models.BillingMonthly,
		PlanAmount:         1000,
		CurrentPeriodStart: tptr(refNow),
		CurrentPeriodEnd:   tptr(refNow.AddDate(0, 0, 30)),
	}
	// Cancel immediately after starting → full refund.
	assert.InDelta(t, 1000.0, calculateProratedRefundAt(sub, refNow), 1e-6)
}

func TestCalculateProratedRefundAt_Monthly_QuarterRemaining(t *testing.T) {
	sub := &models.Subscription{
		BillingInterval:    models.BillingQuarterly,
		PlanAmount:         1200,
		CurrentPeriodStart: tptr(refNow.AddDate(0, 0, -75)), // 75 elapsed
		CurrentPeriodEnd:   tptr(refNow.AddDate(0, 0, 25)),  // 25 remain of 100
	}
	// 25/100 * 1200 = 300.
	assert.InDelta(t, 300.0, calculateProratedRefundAt(sub, refNow), 1e-6)
}

func TestCalculateProratedRefundAt_ZeroTotalDays_Zero(t *testing.T) {
	// Degenerate period (start == end, in the future) → guard returns 0.
	sameDay := refNow.AddDate(0, 0, 10)
	sub := &models.Subscription{
		BillingInterval:    models.BillingMonthly,
		PlanAmount:         1000,
		CurrentPeriodStart: tptr(sameDay),
		CurrentPeriodEnd:   tptr(sameDay),
	}
	assert.Equal(t, 0.0, calculateProratedRefundAt(sub, refNow))
}

func TestCalculateProratedRefundAt_Yearly_SixMonthsRemaining(t *testing.T) {
	sub := &models.Subscription{
		BillingInterval:    models.BillingYearly,
		PlanAmount:         1200,
		CurrentPeriodStart: tptr(refNow.AddDate(0, -6, 0)),
		CurrentPeriodEnd:   tptr(refNow.AddDate(0, 6, 0)), // exactly 6 months out
	}
	// 6 remaining full months / 12 * 1200 = 600.
	assert.InDelta(t, 600.0, calculateProratedRefundAt(sub, refNow), 1e-6)
}

func TestCalculateProratedRefundAt_Yearly_ThreeMonthsRemaining(t *testing.T) {
	sub := &models.Subscription{
		BillingInterval:    models.BillingYearly,
		PlanAmount:         1200,
		CurrentPeriodStart: tptr(refNow.AddDate(0, -9, 0)),
		CurrentPeriodEnd:   tptr(refNow.AddDate(0, 3, 0)),
	}
	// 3 remaining full months / 12 * 1200 = 300.
	assert.InDelta(t, 300.0, calculateProratedRefundAt(sub, refNow), 1e-6)
}

// CalculateProratedRefund (the exported wrapper) should agree with the
// fixed-clock variant evaluated at ~now — sanity that the wrapper wires through.
func TestCalculateProratedRefund_WrapperMatchesNow(t *testing.T) {
	now := time.Now().UTC()
	sub := &models.Subscription{
		BillingInterval:    models.BillingMonthly,
		PlanAmount:         1000,
		CurrentPeriodStart: tptr(now.AddDate(0, 0, -10)),
		CurrentPeriodEnd:   tptr(now.AddDate(0, 0, 20)),
	}
	// Both evaluate at essentially the same instant; allow a small delta for the
	// microseconds between the two time.Now() reads.
	assert.InDelta(t, calculateProratedRefundAt(sub, now), CalculateProratedRefund(sub), 0.5)
}
