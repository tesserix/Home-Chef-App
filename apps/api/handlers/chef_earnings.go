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
// Period: week (last 7d), month (last 30d), cycle (chef's active subscription
//         billing cycle — falls back to calendar month when no subscription).

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

// Earnings rate constants. Defined as named consts so they can be made
// configurable (env / DB-backed config row) in a future pass without
// touching the math functions.
const (
	// rateCommission is the platform's share of each order's item revenue.
	rateCommission = 0.15

	// rateGST is the total GST rate applied to the commission amount.
	// Splits into CGST + SGST (intra-state) or IGST (inter-state).
	rateGST = 0.18

	// rateTDS is the TDS rate deducted from gross payout under Section 194-O.
	// Applied whenever the chef's cumulative annual gross meets the threshold —
	// for now, computed always; enforcement is at settlement time.
	rateTDS = 0.01

	// earningsCurrency is the reporting currency for all India-domiciled chefs.
	// International chefs are out of scope for this breakdown endpoint (v1).
	earningsCurrency = "INR"
)

// ChefEarningsHandler handles earnings-breakdown requests.
type ChefEarningsHandler struct{}

// NewChefEarningsHandler constructs the handler.
func NewChefEarningsHandler() *ChefEarningsHandler {
	return &ChefEarningsHandler{}
}

// earningsOrderRow is an intermediate scan target used by the DB query.
type earningsOrderRow struct {
	OrderID       uuid.UUID `gorm:"column:id"`
	OrderNumber   string    `gorm:"column:order_number"`
	CompletedAt   time.Time `gorm:"column:delivered_at"`
	ItemRevenue   float64   `gorm:"column:subtotal"`
	DeliveryFee   float64   `gorm:"column:delivery_fee"`
	ChefTip       float64   `gorm:"column:chef_tip"`
	DeliveryState string    `gorm:"column:delivery_address_state"`
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
		SELECT id, order_number, delivered_at, subtotal, delivery_fee,
		       chef_tip, delivery_address_state
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

	for _, row := range rows {
		breakdown := computeOrderBreakdown(row, chef.State)

		orderItems = append(orderItems, breakdown)

		totals.GrossRevenue += breakdown.Gross
		totals.PlatformCommission += breakdown.PlatformCommission
		totals.CGST += breakdown.CGST
		totals.SGST += breakdown.SGST
		totals.IGST += breakdown.IGST
		totals.TDS += breakdown.TDS
		totals.NetPayout += breakdown.NetPayout
		totals.OrdersCount++
	}

	// Round totals to 2dp
	totals.GrossRevenue = round2(totals.GrossRevenue)
	totals.PlatformCommission = round2(totals.PlatformCommission)
	totals.CGST = round2(totals.CGST)
	totals.SGST = round2(totals.SGST)
	totals.IGST = round2(totals.IGST)
	totals.TDS = round2(totals.TDS)
	totals.NetPayout = round2(totals.NetPayout)

	c.JSON(http.StatusOK, gin.H{
		"cycleStart": cycleStart,
		"cycleEnd":   cycleEnd,
		"currency":   earningsCurrency,
		"rates": breakdownRates{
			PlatformCommission: rateCommission,
			GST:                rateGST,
			TDS:                rateTDS,
		},
		"totals": totals,
		"orders": orderItems,
	})
}

// computeOrderBreakdown applies the earnings rules to a single order row.
//
// Rules (from spec):
//   - commission = rateCommission × itemRevenue
//   - gross       = itemRevenue + deliveryFee + chefTip
//   - intra-state (order.state == chef.state): CGST 9% + SGST 9% on commission
//   - inter-state (order.state != chef.state): IGST 18% on commission
//   - tds        = rateTDS × gross
//   - netPayout  = gross − commission − tds   (GST is not deducted from chef)
func computeOrderBreakdown(row earningsOrderRow, chefState string) earningsOrderResponse {
	commission := round2(rateCommission * row.ItemRevenue)
	gross := round2(row.ItemRevenue + row.DeliveryFee + row.ChefTip)

	var cgst, sgst, igst float64
	halfGST := round2(rateGST / 2 * commission)
	fullGST := round2(rateGST * commission)

	intraState := normaliseState(row.DeliveryState) == normaliseState(chefState)
	if intraState {
		cgst = halfGST
		sgst = halfGST
	} else {
		igst = fullGST
	}

	tds := round2(rateTDS * gross)
	netPayout := round2(gross - commission - tds)

	return earningsOrderResponse{
		OrderID:            row.OrderID,
		OrderNumber:        row.OrderNumber,
		CompletedAt:        row.CompletedAt,
		ItemRevenue:        round2(row.ItemRevenue),
		DeliveryFee:        round2(row.DeliveryFee),
		Tip:                round2(row.ChefTip),
		Gross:              gross,
		PlatformCommission: commission,
		CGST:               cgst,
		SGST:               sgst,
		IGST:               igst,
		TDS:                tds,
		NetPayout:          netPayout,
	}
}

// resolvePeriod returns the start/end timestamps for the requested period.
// "cycle" resolves from the chef's active subscription billing window;
// falls back to the current calendar month when no subscription row exists.
func resolvePeriod(period string, userID uuid.UUID) (time.Time, time.Time) {
	now := time.Now().UTC()

	switch period {
	case "week":
		start := now.AddDate(0, 0, -7).Truncate(24 * time.Hour)
		end := endOfDay(now)
		return start, end

	case "month":
		start := now.AddDate(0, 0, -30).Truncate(24 * time.Hour)
		end := endOfDay(now)
		return start, end

	case "cycle":
		var sub models.Subscription
		err := database.DB.Where(
			"user_id = ? AND subscriber_type = ? AND status IN ('trial','active','past_due')",
			userID, models.SubscriberChef,
		).Order("created_at DESC").First(&sub).Error
		if err == nil && sub.CurrentPeriodStart != nil && sub.CurrentPeriodEnd != nil {
			return sub.CurrentPeriodStart.UTC(), sub.CurrentPeriodEnd.UTC()
		}
		// Fallback: current calendar month
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0).Add(-time.Second) // last second of month
		return start, end

	default:
		// Treat unknown period as week
		start := now.AddDate(0, 0, -7).Truncate(24 * time.Hour)
		end := endOfDay(now)
		return start, end
	}
}

// endOfDay returns 23:59:59.999999999 of the given day in UTC.
func endOfDay(t time.Time) time.Time {
	d := t.UTC().Truncate(24 * time.Hour)
	return d.Add(24*time.Hour - time.Nanosecond)
}

// normaliseState lowercases and trims state names for comparison so that
// "Maharashtra" == "maharashtra" and " MAHARASHTRA " == "maharashtra".
func normaliseState(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == ' ' || r == '\t' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			out = append(out, r+32)
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

// round2 rounds a float64 to 2 decimal places using banker's-style rounding.
// All monetary values in the earnings breakdown are non-negative, so the
// simpler positive-only path is the hot path. The negative branch is retained
// for correctness.
func round2(v float64) float64 {
	const factor = 100.0
	if v >= 0 {
		return float64(int64(v*factor+0.5)) / factor
	}
	return -float64(int64(-v*factor+0.5)) / factor
}
