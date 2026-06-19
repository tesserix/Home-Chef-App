package handlers

import (
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// chef_analytics.go — the analytics endpoints behind the chef dashboard (#49):
// subscription health (#229) and a simple demand forecast (#230). Headline sales
// metrics (AOV / repeat-rate) ride on GetChefAnalytics in chefs.go. Aggregations
// reuse the meal-plan prep rollup (buildPrepManifest) and the IST helpers.

// pct returns num/den as a percentage rounded to 1 decimal (0 when den == 0).
func pct(num, den int64) float64 {
	if den == 0 {
		return 0
	}
	return math.Round(float64(num)/float64(den)*1000) / 10
}

// forecastAvg averages a total count over a number of periods, rounded to the
// nearest whole unit. Pure + unit-tested — the heart of the demand forecast.
func forecastAvg(total, periods int) int {
	if periods <= 0 || total <= 0 {
		return 0
	}
	return int(math.Round(float64(total) / float64(periods)))
}

// GetSubscriptionMetrics — GET /chef/analytics/subscriptions (#229). Active plan
// count, churn %, and per-day adherence % for the authed chef.
func (h *ChefHandler) GetSubscriptionMetrics(c *gin.Context) {
	chef, ok := authedChef(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	plan := func(statuses []string) int64 {
		var n int64
		database.DB.Model(&models.MealPlan{}).
			Where("chef_id = ? AND status IN ?", chef.ID, statuses).Count(&n)
		return n
	}

	active := plan([]string{"confirmed", "active"})
	// Churn = cancelled/expired ÷ every plan that reached a decision (exclude the
	// in-flight negotiation states, which aren't churn yet).
	decided := plan([]string{"confirmed", "active", "completed", "cancelled", "expired"})
	churned := plan([]string{"cancelled", "expired"})

	var subscribers int64
	database.DB.Model(&models.MealPlan{}).
		Where("chef_id = ? AND status IN ?", chef.ID, []string{"confirmed", "active", "completed"}).
		Distinct("customer_id").Count(&subscribers)

	// Adherence = delivered days ÷ (delivered + skipped) days — how much of what
	// was committed actually got cooked + delivered.
	dayCount := func(status string) int64 {
		var n int64
		database.DB.Table("meal_plan_days").
			Joins("JOIN meal_plans ON meal_plans.id = meal_plan_days.meal_plan_id").
			Where("meal_plans.chef_id = ? AND meal_plan_days.status = ?", chef.ID, status).Count(&n)
		return n
	}
	delivered := dayCount("delivered")
	skipped := dayCount("skipped")

	c.JSON(http.StatusOK, gin.H{
		"activePlans":   active,
		"subscribers":   subscribers,
		"churnRate":     pct(churned, decided),
		"adherenceRate": pct(delivered, delivered+skipped),
		"deliveredDays": delivered,
		"skippedDays":   skipped,
	})
}

// GetDemandForecast — GET /chef/analytics/forecast?date=YYYY-MM-DD (default
// tomorrow IST, #230). Blends the confirmed subscription meals for the date with
// a 4-week weekday average of à-la-carte orders. Deliberately simple + auditable.
func (h *ChefHandler) GetDemandForecast(c *gin.Context) {
	chef, ok := authedChef(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	day := startOfDayIST(time.Now()).Add(24 * time.Hour) // tomorrow
	if ds := c.Query("date"); ds != "" {
		d, err := parsePlanDate(ds)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must be YYYY-MM-DD"})
			return
		}
		day = d
	}

	// Subscription meals for the date — exact, from the confirmed/prepared days
	// (reuses the #50 prep rollup).
	rows := queryPrepRows(chef.ID, day, day.Add(24*time.Hour))
	_, totals := buildPrepManifest(rows)

	// À-la-carte expectation — average orders on this weekday over the last 4
	// weeks. Compare the IST weekday (created_at shifted +5:30 to IST).
	weekday := int(day.In(istLoc).Weekday()) // 0=Sun..6=Sat
	const lookbackWeeks = 4
	var alaCarteCount int
	database.DB.Raw(`
		SELECT COUNT(*) FROM orders
		WHERE chef_id = ? AND deleted_at IS NULL
		  AND created_at >= ?
		  AND EXTRACT(DOW FROM created_at + interval '330 minutes') = ?
	`, chef.ID, time.Now().AddDate(0, 0, -7*lookbackWeeks), weekday).Scan(&alaCarteCount)
	alaCarteForecast := forecastAvg(alaCarteCount, lookbackWeeks)

	// Likely top à-la-carte dishes on this weekday (avg qty over the lookback).
	type dishAgg struct {
		Name string
		Qty  int
	}
	var topDishes []dishAgg
	database.DB.Raw(`
		SELECT oi.name, SUM(oi.quantity) AS qty
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE o.chef_id = ? AND o.deleted_at IS NULL
		  AND o.created_at >= ?
		  AND EXTRACT(DOW FROM o.created_at + interval '330 minutes') = ?
		GROUP BY oi.name
		ORDER BY qty DESC
		LIMIT 5
	`, chef.ID, time.Now().AddDate(0, 0, -7*lookbackWeeks), weekday).Scan(&topDishes)

	likely := make([]gin.H, 0, len(topDishes))
	for _, d := range topDishes {
		if exp := forecastAvg(d.Qty, lookbackWeeks); exp > 0 {
			likely = append(likely, gin.H{"name": d.Name, "expected": exp})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"date":               day.In(istLoc).Format("2006-01-02"),
		"subscriptionMeals":  totals.Total,
		"subscriptionLunch":  totals.Lunch,
		"subscriptionDinner": totals.Dinner,
		"alaCarteForecast":   alaCarteForecast,
		"totalExpected":      totals.Total + alaCarteForecast,
		"likelyDishes":       likely,
		"basis":              "Confirmed subscriptions + 4-week average for this weekday",
	})
}
