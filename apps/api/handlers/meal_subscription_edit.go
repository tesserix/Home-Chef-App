package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// meal_subscription_edit.go — change a live subscription instead of replacing it.
//
// One live subscription per (customer, chef) is enforced at create time, so a
// customer who wanted different days or slots had NO way to get them: creating
// again is refused, and the only escape was cancel-and-resubscribe, which throws
// away the billing period they had already paid for. Editing is the missing half
// of that rule — the restriction only makes sense if the thing you already have
// can be adjusted.

// editableSubStatuses are the states where a change is meaningful.
//
// Cancelled is terminal — editing it would silently resurrect a subscription the
// customer ended, so they must subscribe afresh (which the duplicate guard now
// permits, precisely because the old one is cancelled). past_due is excluded too:
// the money problem must be settled before the shape of the plan moves, or the
// retry would charge for a cycle the customer never agreed to.
var editableSubStatuses = map[string]bool{
	models.MealSubStatusTrialing: true,
	models.MealSubStatusActive:   true,
	models.MealSubStatusPaused:   true,
}

type updateMealSubRequest struct {
	Slots       []string          `json:"slots" binding:"required"`
	Days        []int64           `json:"days" binding:"required"`
	Variant     string            `json:"variant"`
	DayVariants map[string]string `json:"dayVariants"`
	AddressID   string            `json:"addressId"`
}

// UpdateSubscription — PUT /meal-subscriptions/:id.
//
// Changes what the customer receives (slots, days, variant, address) on their
// existing subscription. Cadence is deliberately NOT editable: it defines the
// billing period, and changing it mid-cycle would desynchronise the charge from
// the mandate. To change cadence, cancel and resubscribe.
//
// The cycle amount is recomputed from the chef's CURRENT config so the price
// always matches what is actually being delivered.
func (h *MealSubscriptionHandler) UpdateSubscription(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription id"})
		return
	}

	// Scoped to the authed customer — 404 rather than 403 so this cannot be used
	// to probe whether a subscription id exists.
	var sub models.MealSubscription
	if err := database.DB.Where("id = ? AND customer_id = ?", id, userID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return
	}
	if !editableSubStatuses[sub.Status] {
		c.JSON(http.StatusConflict, gin.H{
			"error": "This subscription can't be edited — cancel it and subscribe again to start a new plan.",
		})
		return
	}

	var req updateMealSubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Slots) == 0 || len(req.Days) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pick at least one slot and one day"})
		return
	}

	// Price from the chef's CURRENT config, not the amount snapshotted at
	// subscribe time — otherwise a customer could add days at a stale rate.
	var cfg models.ChefSubscriptionConfig
	if err := database.DB.Where("chef_id = ?", sub.ChefID).First(&cfg).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "This chef isn't offering subscriptions right now"})
		return
	}

	variant := models.MealVariant(req.Variant)
	if variant != models.MealVariantVeg && variant != models.MealVariantNonVeg {
		variant = sub.Variant
	}

	updates := map[string]any{
		"slots":          req.Slots,
		"days":           req.Days,
		"variant":        variant,
		"day_variants":   normaliseDayVariants(req.DayVariants, req.Days),
		"per_meal_price": cfg.PerMealPrice,
		"delivery_fee":   cfg.DeliveryFee,
		"cycle_amount": services.ComputeMealCycleAmount(
			cfg.PerMealPrice, len(req.Slots), len(req.Days), sub.Cadence, cfg.DeliveryFee),
	}
	if req.AddressID != "" {
		if aid, e := uuid.Parse(req.AddressID); e == nil {
			updates["default_address_id"] = aid
		}
	}

	// Status-guarded so a concurrent cancel wins rather than being overwritten by
	// an edit that was already in flight.
	res := database.DB.Model(&models.MealSubscription{}).
		Where("id = ? AND customer_id = ? AND status = ?", id, userID, sub.Status).
		Updates(updates)
	if res.Error != nil {
		log.Printf("meal-subscription: update failed for subscription=%s customer=%s: %v", id, userID, res.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update subscription"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "This subscription changed while you were editing — reload and try again."})
		return
	}

	database.DB.First(&sub, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"subscription": sub})
}
