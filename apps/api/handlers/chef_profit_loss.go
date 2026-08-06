package handlers

// chef_profit_loss.go — GET /chef/analytics/pl (#pl).
//
// Analytics showed what came IN and nothing about what went OUT, so a chef
// could see a good revenue week without knowing whether it paid for itself.
// The expense data has existed since chef_expenses; this joins the two sides.
//
// Earnings here are the chef's NET payout, not gross revenue, and they are
// computed with services.ComputeOrderEarnings — the same calculator behind the
// Earnings screen and the weekly settlement statement. That is the whole point:
// a P/L that derived its own commission or GST would drift from the figure the
// chef is actually paid, and two screens disagreeing about money is worse than
// no screen at all.
//
// Scoped to the calling chef throughout — the row loader is keyed on their own
// chef_id and expenses on their own chef_id, so nothing here can surface another
// kitchen's numbers.

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

type plCategoryLine struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

// GetChefProfitLoss returns earnings vs expenses for a period.
func (h *ChefHandler) GetChefProfitLoss(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// Same period vocabulary as the analytics screen's tabs (7d/30d/90d), so the
	// P/L card and the cards above it describe the same window.
	days := 7
	switch c.DefaultQuery("period", "7d") {
	case "30d":
		days = 30
	case "90d":
		days = 90
	}
	loc := services.BusinessLocation()
	now := time.Now().In(loc)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(days - 1))

	// ── Earnings side: delivered orders, netted exactly as the payout is ──
	var rows []earningsOrderRow
	if err := database.DB.Raw(`
		SELECT id, order_number, delivered_at, subtotal, tax,
		       tax_food, tax_service, chef_funded_discount,
		       delivery_fee, chef_tip, delivery_address_state, commission_rate,
		       payout_hold_status
		FROM   orders
		WHERE  chef_id       = ?
		AND    status        = 'delivered'
		AND    delivered_at >= ?
		AND    deleted_at    IS NULL
		ORDER  BY delivered_at ASC
	`, chef.ID, start.UTC()).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	chefState := normaliseState(chef.State)
	liveRate := services.GetCommissionRate(database.DB)

	// Same levy netting as the Earnings screen, so "what I was paid" is one number
	// across both surfaces.
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.OrderID
	}
	penalties := services.ChefOrderPenalties(database.DB, ids)

	var gross, commission, taxes, tds, net float64
	for _, r := range rows {
		e := computeOrderBreakdown(r, chefState, liveRate)
		e.applyPenalty(penalties[r.OrderID])
		gross += e.Gross
		commission += e.PlatformCommission
		taxes += e.CGST + e.SGST + e.IGST
		tds += e.TDS
		net += e.NetPayout
	}

	// ── Expense side ──
	// Bucketed on expense_date, not created_at: a chef entering yesterday's gas
	// bill this morning belongs in yesterday's week, which is what the FY
	// statement already does.
	var expenseRows []plCategoryLine
	database.DB.Raw(`
		SELECT category, COALESCE(SUM(amount), 0) AS amount
		FROM   chef_expenses
		WHERE  chef_id       = ?
		AND    expense_date >= ?
		AND    deleted_at    IS NULL
		GROUP  BY category
		ORDER  BY amount DESC
	`, chef.ID, start.UTC()).Scan(&expenseRows)

	var expenses float64
	for _, e := range expenseRows {
		expenses += e.Amount
	}
	if expenseRows == nil {
		expenseRows = []plCategoryLine{}
	}

	profit := services.Round2(net - expenses)
	// Margin against NET, not gross: it answers "of what I was actually paid,
	// how much did I keep", which is the question a chef is asking.
	margin := 0.0
	if net > 0 {
		margin = services.Round2(profit / net * 100)
	}

	c.JSON(http.StatusOK, gin.H{
		"period":     c.DefaultQuery("period", "7d"),
		"from":       start.Format("2006-01-02"),
		"to":         now.Format("2006-01-02"),
		"orders":     len(rows),
		"grossSales": services.Round2(gross),
		"deductions": gin.H{
			"platformCommission": services.Round2(commission),
			"gst":                services.Round2(taxes),
			"tds":                services.Round2(tds),
		},
		"netEarnings":        services.Round2(net),
		"expenses":           services.Round2(expenses),
		"expensesByCategory": expenseRows,
		"netProfit":          profit,
		"marginPercent":      margin,
	})
}
