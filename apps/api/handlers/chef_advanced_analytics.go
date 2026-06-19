package handlers

import (
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// chef_advanced_analytics.go — the premium-only "advanced analytics" perk (#44).
// Standard chefs get the existing analytics; premium unlocks deeper insights:
// repeat-customer cohort, revenue per customer, and the best ordering day. The
// gate is the only #44-specific bit — the math is plain aggregation over the
// chef's delivered orders.

var weekdayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// GetAdvancedAnalytics returns premium-gated deep insights for the authed chef.
// Non-premium chefs get a 402 (payment required) nudge to upgrade rather than a
// 403, so the client can route them straight to the upgrade screen.
func (h *ChefHandler) GetAdvancedAnalytics(c *gin.Context) {
	chef, ok := authedChef(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	if !services.IsChefPremium(chef.ID) {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":   "Advanced analytics is a Premium feature",
			"upgrade": true,
		})
		return
	}

	since := time.Now().AddDate(0, 0, -90)

	// Per-customer order counts + spend over the window.
	type custAgg struct {
		Orders int64
		Spend  float64
	}
	var aggs []custAgg
	database.DB.Model(&models.Order{}).
		Select("COUNT(*) AS orders, COALESCE(SUM(total),0) AS spend").
		Where("chef_id = ? AND status = ? AND created_at >= ?", chef.ID, "delivered", since).
		Group("customer_id").Scan(&aggs)

	totalCustomers := len(aggs)
	repeatCustomers := 0
	var totalOrders int64
	var totalSpend float64
	for _, a := range aggs {
		totalOrders += a.Orders
		totalSpend += a.Spend
		if a.Orders >= 2 {
			repeatCustomers++
		}
	}

	round1 := func(v float64) float64 { return math.Round(v*10) / 10 }
	round2 := func(v float64) float64 { return math.Round(v*100) / 100 }

	repeatRate, avgRevPerCust, avgOrdersPerCust := 0.0, 0.0, 0.0
	if totalCustomers > 0 {
		repeatRate = round1(float64(repeatCustomers) / float64(totalCustomers) * 100)
		avgRevPerCust = round2(totalSpend / float64(totalCustomers))
		avgOrdersPerCust = round2(float64(totalOrders) / float64(totalCustomers))
	}

	// Best ordering day-of-week.
	type dayRow struct {
		Dow    int
		Orders int64
	}
	var days []dayRow
	database.DB.Model(&models.Order{}).
		Select("EXTRACT(DOW FROM created_at)::int AS dow, COUNT(*) AS orders").
		Where("chef_id = ? AND status = ? AND created_at >= ?", chef.ID, "delivered", since).
		Group("dow").Order("orders DESC").Scan(&days)
	bestDay := ""
	var bestDayOrders int64
	if len(days) > 0 && days[0].Dow >= 0 && days[0].Dow < 7 {
		bestDay = weekdayNames[days[0].Dow]
		bestDayOrders = days[0].Orders
	}

	c.JSON(http.StatusOK, gin.H{
		"windowDays":            90,
		"customers":             totalCustomers,
		"repeatCustomers":       repeatCustomers,
		"repeatRate":            repeatRate,
		"avgRevenuePerCustomer": avgRevPerCust,
		"avgOrdersPerCustomer":  avgOrdersPerCust,
		"bestDay":               gin.H{"day": bestDay, "orders": bestDayOrders},
		"currency":              services.EarningsCurrency,
	})
}
