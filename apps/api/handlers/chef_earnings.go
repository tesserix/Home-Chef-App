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
	"strings"
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
	Currency           string    `gorm:"column:currency"`
	TaxInclusive       bool      `gorm:"column:tax_inclusive"`
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
	// FulfillmentType decides who the delivery fee belongs to; DeliveryFeeFinal
	// is the #703 lowered-at-accept figure the customer was actually charged.
	FulfillmentType  string   `gorm:"column:fulfillment_type"`
	DeliveryFeeFinal *float64 `gorm:"column:delivery_fee_final"`
	// GatewaySplitPaise is what Easy Split actually settled to this chef's vendor
	// account at capture (#1087). Non-zero means the money has already moved and
	// this is the figure, not the recomputation of it.
	GatewaySplitPaise int64 `gorm:"column:gateway_split_paise"`
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
	// Penalty is the cancellation levy attributed to this order, already
	// subtracted from NetPayout so the row matches the order's payout card.
	Penalty   float64 `json:"penalty"`
	NetPayout float64 `json:"netPayout"`
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
	// Penalties is the cancellation levies attributed to these orders. Already
	// subtracted from NetPayout — the settlement nets the same money off the
	// weekly statement, so showing it before the levy contradicted the payout.
	Penalties   float64 `json:"penalties"`
	NetPayout   float64 `json:"netPayout"`
	OrdersCount int     `json:"ordersCount"`
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

	commissionRate := services.GetCommissionRate(database.DB)
	totals, orderItems, err := chefSettledEarnings(chef, cycleStart, cycleEnd, commissionRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	// Surface the resolved runtime commission rate the chef is charged.
	effectiveCommission := commissionRate

	c.JSON(http.StatusOK, gin.H{
		"cycleStart": cycleStart,
		"cycleEnd":   cycleEnd,
		"currency":   strings.ToUpper(services.CurrencyForCountry(chef.PayoutCountry)),
		"rates": breakdownRates{
			PlatformCommission: effectiveCommission,
			GST:                services.RateGST,
			TDS:                services.RateTDS,
		},
		"totals": totals,
		"orders": orderItems,
	})
}

// chefSettledEarnings is THE definition of a chef's earnings in a window, and the
// only place the query lives.
//
// Settled means: delivered, not refunded, not soft-deleted, in the chef's current
// mode — anchored on delivered_at, because money is earned when the food arrives,
// not when the order was placed. The Earnings screen, the Analytics money figures
// and the weekly statement all call this, so a chef cannot open two screens and be
// told two different numbers for the same week (#1030).
//
// The vendor Analytics screen used to count anything PAID (any status, any age,
// bucketed on created_at) as revenue, which is a demand metric — legitimate for
// the orders chart, wrong under a heading the chef reads as "what I made".
func chefSettledEarnings(
	chef models.ChefProfile,
	from, to time.Time,
	commissionRate float64,
) (earningsTotals, []earningsOrderResponse, error) {
	var rows []earningsOrderRow
	// tax_food/tax_service are NOT optional in the projection: ChefTaxOf falls back
	// to the whole order tax when both scan as zero, which would show the chef the
	// platform's own GST on the fee and delivery as if it were theirs.
	err := database.DB.Raw(`
		SELECT id, order_number, delivered_at, subtotal, tax,
		       currency, tax_inclusive, tax_food, tax_service, chef_funded_discount,
		       delivery_fee, delivery_fee_final, fulfillment_type,
		       chef_tip, delivery_address_state, commission_rate,
		       payout_hold_status, COALESCE(gateway_split_paise, 0) AS gateway_split_paise
		FROM   orders
		WHERE  chef_id       = ?
		AND    status        = 'delivered'
		AND    delivered_at >= ?
		AND    delivered_at <= ?
		AND    deleted_at    IS NULL
		-- Same two guards the weekly statement applies (services/statement.go). Without
		-- them a sandbox order counts toward a live chef's earnings, and a refunded
		-- order — which the statement and the payout-release path both exclude — is
		-- billed as if it had settled. Status stays 'delivered' on the order-issue
		-- refund path, so filtering on status alone does not catch it.
		AND    refunded_at   IS NULL
		AND    mode = COALESCE((SELECT mode FROM chef_profiles WHERE id = ?), 'live')
		ORDER  BY delivered_at ASC
	`, chef.ID, from, to, chef.ID).Scan(&rows).Error
	if err != nil {
		return earningsTotals{}, nil, err
	}

	// Levies are read once for the whole window: the settlement nets them off the
	// weekly statement, so an Earnings screen that ignored them showed the chef a
	// payout they were never going to receive.
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.OrderID
	}
	penalties := services.ChefOrderPenalties(database.DB, ids)

	orderItems := make([]earningsOrderResponse, 0, len(rows))
	var totals earningsTotals
	for _, row := range rows {
		breakdown := computeOrderBreakdown(row, chef.State, commissionRate)
		if !breakdown.applySettledSplit(row.GatewaySplitPaise) {
			breakdown.applyPenalty(penalties[row.OrderID])
		}
		orderItems = append(orderItems, breakdown)

		totals.GrossRevenue += breakdown.Gross
		totals.PlatformCommission += breakdown.PlatformCommission
		totals.CGST += breakdown.CGST
		totals.SGST += breakdown.SGST
		totals.IGST += breakdown.IGST
		totals.TDS += breakdown.TDS
		totals.Penalties += breakdown.Penalty
		totals.NetPayout += breakdown.NetPayout
		totals.OrdersCount++

		// Escrow split (#617): bucket the net payout by hold lifecycle.
		if held, released := payoutBucket(breakdown.PayoutHoldStatus); held {
			totals.Held += breakdown.NetPayout
		} else if released {
			totals.Released += breakdown.NetPayout
		}
	}

	totals.GrossRevenue = round2(totals.GrossRevenue)
	totals.PlatformCommission = round2(totals.PlatformCommission)
	totals.CGST = round2(totals.CGST)
	totals.SGST = round2(totals.SGST)
	totals.IGST = round2(totals.IGST)
	totals.TDS = round2(totals.TDS)
	totals.Penalties = round2(totals.Penalties)
	totals.NetPayout = round2(totals.NetPayout)
	totals.Held = round2(totals.Held)
	totals.Released = round2(totals.Released)
	return totals, orderItems, nil
}

// applySettledSplit replaces the computed net with what Easy Split actually paid
// this chef at capture, reporting whether it did (#1087).
//
// The two differ by the flat platform fee, and by the capture cap when the share
// exceeds what was collected — so the recomputation is not the money that
// arrived. A levy is deliberately NOT netted off a settled row: nothing was
// withheld from a transfer that already completed, and the levy is collected
// from a later payout as a recovery deduction (#1092).
func (r *earningsOrderResponse) applySettledSplit(splitPaise int64) bool {
	if splitPaise <= 0 {
		return false
	}
	r.NetPayout = round2(float64(splitPaise) / 100)
	// Escrow buckets describe money the platform is still holding. A split order
	// never entered escrow, so it belongs in neither.
	r.PayoutHoldStatus = ""
	return true
}

// applyPenalty nets a cancellation levy off this row. Never negative: a levy
// larger than the order cannot make the chef owe money on it, and the remainder
// stays outstanding in the penalty ledger (services.ComputeChefPayout).
func (r *earningsOrderResponse) applyPenalty(penalty float64) {
	if penalty <= 0 {
		return
	}
	r.Penalty = round2(penalty)
	r.NetPayout = round2(r.NetPayout - r.Penalty)
	if r.NetPayout < 0 {
		r.NetPayout = 0
	}
}

// computeOrderBreakdown applies the earnings rules to a single order row and
// shapes the result into the wire response. The math itself lives in
// services.ComputeOrderEarnings so the live endpoint, the weekly statement
// generator, and the TDS certificate all settle identically.
func computeOrderBreakdown(row earningsOrderRow, chefState string, commissionRate float64) earningsOrderResponse {
	fee := row.DeliveryFee
	if row.DeliveryFeeFinal != nil {
		fee = *row.DeliveryFeeFinal
	}
	e := services.ComputeOrderEarnings(services.EarningsInput{
		Currency:             row.Currency,
		TaxInclusive:         row.TaxInclusive,
		OrderID:              row.OrderID,
		OrderNumber:          row.OrderNumber,
		CompletedAt:          row.CompletedAt,
		ItemRevenue:          row.ItemRevenue,
		Tax:                  services.ChefTaxOf(row.Tax, row.TaxFood, row.TaxService),
		ChefFundedDiscount:   row.ChefFundedDiscount,
		DeliveryFee:          fee,
		ChefEarnsDeliveryFee: services.SettledChefEarnsDeliveryFee(row.FulfillmentType),
		ChefTip:              row.ChefTip,
		DeliveryState:        row.DeliveryState,
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
