package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type SubscriptionHandler struct{}

func NewSubscriptionHandler() *SubscriptionHandler {
	return &SubscriptionHandler{}
}

// GetSubscription returns the current subscription for the authenticated user
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	subType := resolveSubscriberType(c)

	var sub models.Subscription
	if err := database.DB.Where("user_id = ? AND subscriber_type = ?", userID, subType).
		First(&sub).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"subscription": nil,
			"status":       "none",
		})
		return
	}

	// Get cycle earnings
	total, orderRev, deliveryFees, tips := services.GetCycleEarnings(sub.ID)

	// Get threshold from settings
	planCfg, _ := services.GetPlanSettings(sub.CountryCode, sub.SubscriberType)
	threshold := 0.0
	if planCfg != nil {
		threshold = planCfg.MinEarningsThreshold
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription": sub.ToResponse(),
		"status":       string(sub.Status),
		"earnings": gin.H{
			"total":        total,
			"orderRevenue": orderRev,
			"deliveryFees": deliveryFees,
			"tips":         tips,
			"threshold":    threshold,
			"thresholdMet": total >= threshold,
			"currency":     sub.Currency,
		},
	})
}

// GetAvailablePlans returns the available subscription plans
func (h *SubscriptionHandler) GetAvailablePlans(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	subType := resolveSubscriberType(c)
	countryCode := resolveCountryCode(userID, subType)

	planCfg, err := services.GetPlanSettings(countryCode, subType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load plan settings"})
		return
	}

	plans := []gin.H{
		{
			"interval": "monthly",
			"amount":   planCfg.MonthlyPrice,
			"currency": planCfg.Currency,
		},
		{
			"interval":        "quarterly",
			"amount":          planCfg.QuarterlyPrice,
			"currency":        planCfg.Currency,
			"savingsPercent":  models.RoundAmount((1 - planCfg.QuarterlyPrice/(planCfg.MonthlyPrice*3)) * 100),
		},
		{
			"interval":        "yearly",
			"amount":          planCfg.YearlyPrice,
			"currency":        planCfg.Currency,
			"savingsPercent":  models.RoundAmount((1 - planCfg.YearlyPrice/(planCfg.MonthlyPrice*12)) * 100),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"plans":                plans,
		"trialDays":            planCfg.TrialDays,
		"countryCode":          countryCode,
		"currency":             planCfg.Currency,
		"paymentGateway":       planCfg.PaymentGateway,
		"minEarningsThreshold": planCfg.MinEarningsThreshold,
	})
}

// ChoosePlan creates or updates a subscription with the chosen plan
func (h *SubscriptionHandler) ChoosePlan(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	subType := resolveSubscriberType(c)

	var req struct {
		Interval string `json:"interval" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	interval := models.BillingInterval(req.Interval)
	if interval != models.BillingMonthly && interval != models.BillingQuarterly && interval != models.BillingYearly {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval. Must be monthly, quarterly, or yearly"})
		return
	}

	countryCode := resolveCountryCode(userID, subType)
	planCfg, err := services.GetPlanSettings(countryCode, subType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load plan settings"})
		return
	}

	planAmount := services.GetPlanAmount(planCfg, interval)

	// Check if subscription exists
	var sub models.Subscription
	if err := database.DB.Where("user_id = ? AND subscriber_type = ?", userID, subType).
		First(&sub).Error; err != nil {
		// Create new trial subscription
		newSub, createErr := services.CreateTrialSubscription(userID, subType, countryCode)
		if createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
			return
		}
		// Update interval if not monthly
		if interval != models.BillingMonthly {
			database.DB.Model(newSub).Updates(map[string]interface{}{
				"billing_interval": interval,
				"plan_amount":      planAmount,
			})
			newSub.BillingInterval = interval
			newSub.PlanAmount = planAmount
		}

		// Update onboarding step for driver subscribers
		if subType == "driver" {
			database.DB.Model(&models.DeliveryPartner{}).
				Where("user_id = ? AND onboarding_step < ?", userID, 4).
				Update("onboarding_step", 4)
		}

		c.JSON(http.StatusCreated, gin.H{"subscription": newSub.ToResponse()})
		return
	}

	// Update existing subscription
	database.DB.Model(&sub).Updates(map[string]interface{}{
		"billing_interval": interval,
		"plan_amount":      planAmount,
	})
	sub.BillingInterval = interval
	sub.PlanAmount = planAmount

	// Update onboarding step for driver subscribers
	if subType == "driver" {
		database.DB.Model(&models.DeliveryPartner{}).
			Where("user_id = ? AND onboarding_step < ?", userID, 4).
			Update("onboarding_step", 4)
	}

	c.JSON(http.StatusOK, gin.H{"subscription": sub.ToResponse()})
}

// CancelSubscription cancels the user's subscription
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	subType := resolveSubscriberType(c)

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	var sub models.Subscription
	if err := database.DB.Where("user_id = ? AND subscriber_type = ? AND status NOT IN ?",
		userID, subType, []models.SubscriptionStatus{models.SubStatusCancelled, models.SubStatusExpired}).
		First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Active subscription not found"})
		return
	}

	refundAmount := services.CalculateProratedRefund(&sub)
	now := time.Now().UTC()

	database.DB.Model(&sub).Updates(map[string]interface{}{
		"status":        models.SubStatusCancelled,
		"cancelled_at":  &now,
		"cancel_reason": req.Reason,
		"refund_amount": refundAmount,
	})

	// Publish event
	go func() {
		if err := services.PublishEvent(services.SubjectSubscriptionCancelled, "subscription.cancelled", userID, map[string]interface{}{
			"subscription_id": sub.ID.String(),
			"reason":          req.Reason,
			"refund_amount":   refundAmount,
		}); err != nil {
			log.Printf("Failed to publish subscription cancelled event: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"subscription": sub.ToResponse(),
		"refundAmount": refundAmount,
	})
}

// ChangePlan changes the billing interval of an active subscription
func (h *SubscriptionHandler) ChangePlan(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	subType := resolveSubscriberType(c)

	var req struct {
		Interval string `json:"interval" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newInterval := models.BillingInterval(req.Interval)
	if newInterval != models.BillingMonthly && newInterval != models.BillingQuarterly && newInterval != models.BillingYearly {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval"})
		return
	}

	var sub models.Subscription
	if err := database.DB.Where("user_id = ? AND subscriber_type = ? AND status IN ?",
		userID, subType, []models.SubscriptionStatus{models.SubStatusTrial, models.SubStatusActive}).
		First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Active subscription not found"})
		return
	}

	if sub.BillingInterval == newInterval {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already on this plan"})
		return
	}

	planCfg, err := services.GetPlanSettings(sub.CountryCode, sub.SubscriberType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load plan settings"})
		return
	}

	newAmount := services.GetPlanAmount(planCfg, newInterval)
	oldAmount := services.GetPlanAmount(planCfg, sub.BillingInterval)

	updates := map[string]interface{}{
		"billing_interval": newInterval,
		"plan_amount":      newAmount,
	}

	// Upgrading (higher amount) takes effect immediately with credit
	if newAmount > oldAmount && sub.Status == models.SubStatusActive && sub.CurrentPeriodEnd != nil {
		creditAmount := services.CalculateProratedRefund(&sub)
		now := time.Now().UTC()
		newPeriodEnd := services.GetPlanAmount(planCfg, newInterval) // used just for interval calculation
		_ = newPeriodEnd
		periodEnd := now.AddDate(0, newInterval.IntervalMonths(), 0)
		updates["current_period_start"] = now
		updates["current_period_end"] = periodEnd

		database.DB.Model(&sub).Updates(updates)
		sub.BillingInterval = newInterval
		sub.PlanAmount = newAmount

		c.JSON(http.StatusOK, gin.H{
			"subscription": sub.ToResponse(),
			"credit":       creditAmount,
			"effectiveNow": true,
		})
		return
	}

	// Downgrading takes effect at next cycle
	database.DB.Model(&sub).Updates(updates)
	sub.BillingInterval = newInterval
	sub.PlanAmount = newAmount

	c.JSON(http.StatusOK, gin.H{
		"subscription": sub.ToResponse(),
		"effectiveNow": sub.Status == models.SubStatusTrial,
	})
}

// GetInvoices returns paginated invoices for the user's subscription
func (h *SubscriptionHandler) GetInvoices(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	subType := resolveSubscriberType(c)

	var sub models.Subscription
	if err := database.DB.Where("user_id = ? AND subscriber_type = ?", userID, subType).
		First(&sub).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"data":       []interface{}{},
			"pagination": gin.H{"page": 1, "limit": 20, "total": 0, "totalPages": 0, "hasNext": false, "hasPrev": false},
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	var total int64
	database.DB.Model(&models.SubscriptionInvoice{}).
		Where("subscription_id = ?", sub.ID).Count(&total)

	var invoices []models.SubscriptionInvoice
	if err := database.DB.Where("subscription_id = ?", sub.ID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&invoices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch invoices"})
		return
	}

	responses := make([]models.SubscriptionInvoiceResponse, len(invoices))
	for i, inv := range invoices {
		responses[i] = inv.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// GetEarningsSummary returns earnings summary for the current billing cycle
func (h *SubscriptionHandler) GetEarningsSummary(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	subType := resolveSubscriberType(c)

	var sub models.Subscription
	if err := database.DB.Where("user_id = ? AND subscriber_type = ?", userID, subType).
		First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	total, orderRev, deliveryFees, tips := services.GetCycleEarnings(sub.ID)

	planCfg, _ := services.GetPlanSettings(sub.CountryCode, sub.SubscriberType)
	threshold := 0.0
	if planCfg != nil {
		threshold = planCfg.MinEarningsThreshold
	}

	var cycleStart, cycleEnd *time.Time
	if sub.CurrentPeriodStart != nil {
		cycleStart = sub.CurrentPeriodStart
		cycleEnd = sub.CurrentPeriodEnd
	} else {
		cycleStart = &sub.TrialStartsAt
		cycleEnd = &sub.TrialEndsAt
	}

	c.JSON(http.StatusOK, models.EarningsSummary{
		CycleStart:    cycleStart,
		CycleEnd:      cycleEnd,
		TotalEarnings: total,
		OrderRevenue:  orderRev,
		DeliveryFees:  deliveryFees,
		Tips:          tips,
		Threshold:     threshold,
		ThresholdMet:  total >= threshold,
		Currency:      sub.Currency,
	})
}

// ---------- Admin Handlers ----------

// AdminGetSubscriptions returns a paginated list of all subscriptions
func (h *SubscriptionHandler) AdminGetSubscriptions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit
	status := c.Query("status")
	subType := c.Query("subscriberType")
	countryCode := c.Query("countryCode")

	query := database.DB.Model(&models.Subscription{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if subType != "" {
		query = query.Where("subscriber_type = ?", subType)
	}
	if countryCode != "" {
		query = query.Where("country_code = ?", strings.ToUpper(countryCode))
	}

	var total int64
	query.Count(&total)

	var subs []models.Subscription
	if err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&subs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscriptions"})
		return
	}

	responses := make([]gin.H, len(subs))
	for i, sub := range subs {
		resp := gin.H{
			"subscription": sub.ToResponse(),
		}
		if sub.User.ID != uuid.Nil {
			resp["user"] = gin.H{
				"id":        sub.User.ID,
				"email":     sub.User.Email,
				"firstName": sub.User.FirstName,
				"lastName":  sub.User.LastName,
			}
		}
		responses[i] = resp
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// AdminGetSubscriptionStats returns aggregate subscription statistics
func (h *SubscriptionHandler) AdminGetSubscriptionStats(c *gin.Context) {
	// Count by status
	statusCounts := make(map[string]int64)
	statuses := []models.SubscriptionStatus{
		models.SubStatusTrial, models.SubStatusActive, models.SubStatusPastDue,
		models.SubStatusSuspended, models.SubStatusCancelled,
	}
	for _, status := range statuses {
		var count int64
		database.DB.Model(&models.Subscription{}).Where("status = ?", status).Count(&count)
		statusCounts[string(status)] = count
	}

	// Count by subscriber type
	typeCounts := make(map[string]int64)
	for _, st := range []models.SubscriberType{models.SubscriberChef, models.SubscriberDriver} {
		var count int64
		database.DB.Model(&models.Subscription{}).Where("subscriber_type = ?", st).Count(&count)
		typeCounts[string(st)] = count
	}

	// Total revenue from paid invoices
	var totalRevenue float64
	database.DB.Model(&models.SubscriptionInvoice{}).
		Where("status = ?", models.InvoicePaid).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&totalRevenue)

	c.JSON(http.StatusOK, gin.H{
		"byStatus":         statusCounts,
		"bySubscriberType": typeCounts,
		"totalRevenue":     totalRevenue,
	})
}

// AdminGetSubscription returns a single subscription with invoices and recent earnings
func (h *SubscriptionHandler) AdminGetSubscription(c *gin.Context) {
	subID := c.Param("id")

	var sub models.Subscription
	if err := database.DB.Preload("User").Preload("Invoices").
		Where("id = ?", subID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}

	// Get recent earnings
	var recentEarnings []models.EarningsLedger
	database.DB.Where("subscription_id = ?", sub.ID).
		Order("created_at DESC").
		Limit(50).
		Find(&recentEarnings)

	// Get cycle earnings
	total, orderRev, deliveryFees, tips := services.GetCycleEarnings(sub.ID)

	// Build invoice responses
	invoiceResponses := make([]models.SubscriptionInvoiceResponse, len(sub.Invoices))
	for i, inv := range sub.Invoices {
		invoiceResponses[i] = inv.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription": sub.ToResponse(),
		"user": gin.H{
			"id":        sub.User.ID,
			"email":     sub.User.Email,
			"firstName": sub.User.FirstName,
			"lastName":  sub.User.LastName,
		},
		"invoices": invoiceResponses,
		"earnings": gin.H{
			"total":        total,
			"orderRevenue": orderRev,
			"deliveryFees": deliveryFees,
			"tips":         tips,
		},
		"recentEarnings": recentEarnings,
	})
}

// AdminListInvoices returns a paginated list of all subscription invoices
func (h *SubscriptionHandler) AdminListInvoices(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit
	status := c.Query("status")

	query := database.DB.Model(&models.SubscriptionInvoice{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var invoices []models.SubscriptionInvoice
	if err := query.Preload("Subscription").Preload("Subscription.User").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&invoices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch invoices"})
		return
	}

	responses := make([]gin.H, len(invoices))
	for i, inv := range invoices {
		resp := gin.H{
			"invoice": inv.ToResponse(),
		}
		if inv.Subscription.User.ID != uuid.Nil {
			resp["user"] = gin.H{
				"id":        inv.Subscription.User.ID,
				"email":     inv.Subscription.User.Email,
				"firstName": inv.Subscription.User.FirstName,
				"lastName":  inv.Subscription.User.LastName,
			}
		}
		responses[i] = resp
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// ---------- Helpers ----------

// resolveSubscriberType detects chef vs driver from the route prefix
func resolveSubscriberType(c *gin.Context) models.SubscriberType {
	path := c.FullPath()
	if strings.Contains(path, "/driver/") || strings.Contains(path, "/delivery/") {
		return models.SubscriberDriver
	}
	return models.SubscriberChef
}

// resolveCountryCode looks up the country for a user based on their profile
func resolveCountryCode(userID uuid.UUID, subType models.SubscriberType) string {
	if subType == models.SubscriberChef {
		var chef models.ChefProfile
		if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err == nil && chef.City != "" {
			return cityToCountry(chef.City)
		}
	} else {
		var partner models.DeliveryPartner
		if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err == nil && partner.City != "" {
			return cityToCountry(partner.City)
		}
	}
	return "IN" // Default to India
}

// cityToCountry maps a city to a country code by looking up delivery zones
func cityToCountry(city string) string {
	var zone models.DeliveryZone
	if err := database.DB.Where("city = ?", city).First(&zone).Error; err == nil && zone.Country != "" {
		return strings.ToUpper(zone.Country)
	}
	return "IN"
}
