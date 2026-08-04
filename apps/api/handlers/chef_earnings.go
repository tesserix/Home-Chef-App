package handlers

// chef_earnings.go — GET /chef/earnings/breakdown
//
// Computes a per-order earnings breakdown applying India-specific tax rules:
//   - Platform commission on item revenue (subtotal only)
//   - GST 18% on platform commission: CGST 9% + SGST 9% (intra-state)
//     or IGST 18% (inter-state). GST is levied on the platform's revenue,
//     NOT deducted from the chef's payout.
//   - TDS 1% under Section 194-O on gross order value. Per the spec, computed
//     always for display; the ₹2.5L annual threshold is enforced by the platform
//     finance team at settlement time, not here.
//   - netPayout = gross − platformCommission − tds
//
// Period: week (current Mon–Sun settlement week, IST), month (current calendar
//         month, IST), cycle (chef's active subscription billing cycle — falls
//         back to the calendar month when no subscription).

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// ChefEarningsHandler handles earnings-breakdown requests.
type ChefEarningsHandler struct{}

// NewChefEarningsHandler constructs the handler.
func NewChefEarningsHandler() *ChefEarningsHandler {
	return &ChefEarningsHandler{}
}

// earningsOrderRow is an intermediate scan target used by the DB query.
type earningsOrderRow struct {
	OrderID            uuid.UUID `gorm:"column:id"`
	OrderNumber        string    `gorm:"column:order_number"`
	CompletedAt        time.Time `gorm:"column:delivered_at"`
	ItemRevenue        float64   `gorm:"column:subtotal"`
	Tax                float64   `gorm:"column:tax"`
	TaxFood            float64   `gorm:"column:tax_food"`
	TaxService         float64   `gorm:"column:tax_service"`
	ChefFundedDiscount float64   `gorm:"column:chef_funded_discount"`
	DeliveryFee        float64   `gorm:"column:delivery_fee"`
	ChefTip            float64   `gorm:"column:chef_tip"`
	DeliveryState      string    `gorm:"column:delivery_address_state"`
	// CommissionRate is the rate FROZEN on the order at checkout (#390); 0 for
	// legacy rows → falls back to the live/default rate via rowRate.
	CommissionRate float64 `gorm:"column:commission_rate"`
	// PayoutHoldStatus is the escrow hold lifecycle for this order (#617). Empty
	// for every order while the escrow flags are off, so the chef-facing held/
	// released split degrades to zero and the per-order pill hides.
	PayoutHoldStatus string `gorm:"column:payout_hold_status"`
}

// earningsOrderResponse is the per-order breakdown shape on the wire.
type earningsOrderResponse struct {
	OrderID            uuid.UUID `json:"orderId"`
	OrderNumber        string    `json:"orderNumber"`
	CompletedAt        time.Time `json:"completedAt"`
	ItemRevenue        float64   `json:"itemRevenue"`
	DeliveryFee        float64   `json:"deliveryFee"`
	Tip                float64   `json:"tip"`
	Gross              float64   `json:"gross"`
	PlatformCommission float64   `json:"platformCommission"`
	CGST               float64   `json:"cgst"`
	SGST               float64   `json:"sgst"`
	IGST               float64   `json:"igst"`
	TDS                float64   `json:"tds"`
	NetPayout          float64   `json:"netPayout"`
	// PayoutHoldStatus surfaces the escrow hold lifecycle so the vendor app can
	// pill the row (awaiting/confirmed/released/disputed). Omitted when empty
	// (no hold — escrow flags off), so pre-launch rows render exactly as before.
	PayoutHoldStatus models.PayoutHoldStatus `json:"payoutHoldStatus,omitempty"`
}

// earningsTotals is the aggregate totals shape.
type earningsTotals struct {
	GrossRevenue       float64 `json:"grossRevenue"`
	PlatformCommission float64 `json:"platformCommission"`
	CGST               float64 `json:"cgst"`
	SGST               float64 `json:"sgst"`
	IGST               float64 `json:"igst"`
	TDS                float64 `json:"tds"`
	NetPayout          float64 `json:"netPayout"`
	OrdersCount        int     `json:"ordersCount"`
	// Held is the net payout still in escrow (awaiting confirmation / eligible /
	// disputed); Released is the net payout the platform has released to the chef.
	// Both are 0 while the escrow flags are off (every hold is empty), so the
	// vendor screen shows a Held/Released split only once escrow is live (#617).
	Held     float64 `json:"held"`
	Released float64 `json:"released"`
}

// breakdownRates is the rates object included in the response.
type breakdownRates struct {
	PlatformCommission float64 `json:"platformCommission"`
	GST                float64 `json:"gst"`
	TDS                float64 `json:"tds"`
}

// GetEarningsBreakdown returns a period-scoped earnings breakdown.
// GET /chef/earnings/breakdown?period=week|month|cycle
func (h *ChefEarningsHandler) GetEarningsBreakdown(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	period := c.DefaultQuery("period", "week")
	cycleStart, cycleEnd := resolvePeriod(period, userID)

	// Query delivered orders within the period
	var rows []earningsOrderRow
	if err := database.DB.Raw(`
		-- tax_food/tax_service are NOT optional: ChefTaxOf falls back to the whole
		-- order tax when both scan as zero, which shows the chef the platform's
		-- own GST on the fee and delivery as if it were theirs.
		SELECT id, order_number, delivered_at, subtotal, tax,
		       tax_food, tax_service, chef_funded_discount,
		       delivery_fee, chef_tip, delivery_address_state, commission_rate,
		       payout_hold_status
		FROM   orders
		WHERE  chef_id       = ?
		AND    status        = 'delivered'
		AND    delivered_at >= ?
		AND    delivered_at <= ?
		AND    deleted_at    IS NULL
		ORDER  BY delivered_at ASC
	`, chef.ID, cycleStart, cycleEnd).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	orderItems := make([]earningsOrderResponse, 0, len(rows))
	var totals earningsTotals

	// Flat platform commission (ADR-0001 / #390) — resolved once from
	// PlatformSettings so the breakdown matches the rate the chef is charged.
	commissionRate := services.GetCommissionRate(database.DB)

	for _, row := range rows {
		breakdown := computeOrderBreakdown(row, chef.State, commissionRate)

		orderItems = append(orderItems, breakdown)

		totals.GrossRevenue += breakdown.Gross
		totals.PlatformCommission += breakdown.PlatformCommission
		totals.CGST += breakdown.CGST
		totals.SGST += breakdown.SGST
		totals.IGST += breakdown.IGST
		totals.TDS += breakdown.TDS
		totals.NetPayout += breakdown.NetPayout
		totals.OrdersCount++

		// Escrow split (#617): bucket the net payout by hold lifecycle.
		if held, released := payoutBucket(breakdown.PayoutHoldStatus); held {
			totals.Held += breakdown.NetPayout
		} else if released {
			totals.Released += breakdown.NetPayout
		}
	}

	// Round totals to 2dp
	totals.GrossRevenue = round2(totals.GrossRevenue)
	totals.PlatformCommission = round2(totals.PlatformCommission)
	totals.CGST = round2(totals.CGST)
	totals.SGST = round2(totals.SGST)
	totals.IGST = round2(totals.IGST)
	totals.TDS = round2(totals.TDS)
	totals.NetPayout = round2(totals.NetPayout)
	totals.Held = round2(totals.Held)
	totals.Released = round2(totals.Released)

	// Surface the resolved runtime commission rate the chef is charged.
	effectiveCommission := commissionRate

	c.JSON(http.StatusOK, gin.H{
		"cycleStart": cycleStart,
		"cycleEnd":   cycleEnd,
		"currency":   services.EarningsCurrency,
		"rates": breakdownRates{
			PlatformCommission: effectiveCommission,
			GST:                services.RateGST,
			TDS:                services.RateTDS,
		},
		"totals": totals,
		"orders": orderItems,
	})
}

// computeOrderBreakdown applies the earnings rules to a single order row and
// shapes the result into the wire response. The math itself lives in
// services.ComputeOrderEarnings so the live endpoint, the weekly statement
// generator, and the TDS certificate all settle identically.
func computeOrderBreakdown(row earningsOrderRow, chefState string, commissionRate float64) earningsOrderResponse {
	e := services.ComputeOrderEarnings(services.EarningsInput{
		OrderID:            row.OrderID,
		OrderNumber:        row.OrderNumber,
		CompletedAt:        row.CompletedAt,
		ItemRevenue:        row.ItemRevenue,
		Tax:                services.ChefTaxOf(row.Tax, row.TaxFood, row.TaxService),
		ChefFundedDiscount: row.ChefFundedDiscount,
		DeliveryFee:        row.DeliveryFee,
		ChefTip:            row.ChefTip,
		DeliveryState:      row.DeliveryState,
		// Per-row frozen rate (#390), falling back to the once-resolved live rate
		// for legacy orders so the breakdown matches the settlement statement.
		CommissionRate: rowRate(row.CommissionRate, commissionRate),
	}, chefState)

	return earningsOrderResponse{
		OrderID:            e.OrderID,
		OrderNumber:        e.OrderNumber,
		CompletedAt:        e.CompletedAt,
		ItemRevenue:        e.ItemRevenue,
		DeliveryFee:        e.DeliveryFee,
		Tip:                e.Tip,
		Gross:              e.Gross,
		PlatformCommission: e.PlatformCommission,
		CGST:               e.CGST,
		SGST:               e.SGST,
		IGST:               e.IGST,
		TDS:                e.TDS,
		NetPayout:          e.NetPayout,
		PayoutHoldStatus:   models.PayoutHoldStatus(row.PayoutHoldStatus),
	}
}

// resolvePeriod returns the start/end timestamps for the requested period.
//
// Every boundary is drawn in the business zone (IST), not UTC, and "week" is
// the Mon–Sun week the platform settles on — the same window the dashboard
// snapshot and the weekly statement use. Rolling 7/30-day windows truncated at
// UTC midnight put these three surfaces on three different weeks, so the same
// money read differently depending on which screen the chef opened (#937).
//
// "cycle" resolves from the chef's active subscription billing window; falls
// back to the current calendar month when no subscription row exists.
func resolvePeriod(period string, userID uuid.UUID) (time.Time, time.Time) {
	now := time.Now()

	switch period {
	case "month":
		return services.BusinessMonthStart(now), services.BusinessDayEnd(now)

	case "cycle":
		var sub models.Subscription
		err := database.DB.Where(
			"user_id = ? AND subscriber_type = ? AND status IN ('trial','active','past_due')",
			userID, models.SubscriberChef,
		).Order("created_at DESC").First(&sub).Error
		if err == nil && sub.CurrentPeriodStart != nil && sub.CurrentPeriodEnd != nil {
			return sub.CurrentPeriodStart.UTC(), sub.CurrentPeriodEnd.UTC()
		}
		start := services.BusinessMonthStart(now)
		return start, start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	default:
		// "week", and any unknown period, resolve to the current settlement week.
		return services.BusinessWeekStart(now), services.BusinessDayEnd(now)
	}
}

// normaliseState delegates to services.NormaliseState (kept as a local alias
// for the totals loop and the package's unit tests).
func normaliseState(s string) string { return services.NormaliseState(s) }

// round2 delegates to services.Round2 (kept as a local alias for the totals
// loop and the package's unit tests).
func round2(v float64) float64 { return services.Round2(v) }

// payoutBucket classifies an order's escrow hold into the chef-facing Held /
// Released buckets for the earnings summary (#617). Held = net payout still in
// escrow (awaiting customer confirmation / release-eligible / disputed); Released
// = the platform has released it to the chef. withheld/reversed (the chef won't
// receive them) and empty (no hold — escrow flags off) fall into NEITHER, so the
// split is 0/0 pre-launch. A payout is never in both buckets.
func payoutBucket(status models.PayoutHoldStatus) (held, released bool) {
	switch status {
	case models.PayoutHoldAwaitingConfirmation, models.PayoutHoldReleaseEligible, models.PayoutHoldDisputed:
		return true, false
	case models.PayoutHoldReleased:
		return false, true
	default:
		return false, false
	}
}

// rowRate returns the per-order FROZEN commission rate when set (>0), else the
// resolved live/default fallback (#390). Legacy orders placed before the
// commission_rate column existed carry 0 and settle on the live rate.
func rowRate(frozen, fallback float64) float64 {
	if frozen > 0 {
		return frozen
	}
	return fallback
}
