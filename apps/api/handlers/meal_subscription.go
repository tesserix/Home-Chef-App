package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// meal_subscription.go — customer tiffin meal-subscription endpoints (#2/#3/#280):
// chef offer config (#4 API) + the customer subscribe / pause / resume / skip /
// cancel lifecycle. The Razorpay UPI-Autopay mandate + first charge (#281) and the
// daily auto-order generation (#282) layer on top of this foundation. Cancel auto-
// fires the #42 win-back offer (#278).

type MealSubscriptionHandler struct{}

func NewMealSubscriptionHandler() *MealSubscriptionHandler { return &MealSubscriptionHandler{} }

// ── Chef offer config (#4) ──────────────────────────────────────────────────

// GetChefSubscriptionConfig returns the authenticated chef's subscription offer.
// GET /chef/subscription-config
func (h *MealSubscriptionHandler) GetChefSubscriptionConfig(c *gin.Context) {
	chefID, ok := h.chefIDFor(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}
	cfg := services.GetChefSubscriptionConfig(database.DB, chefID)
	if cfg == nil {
		cfg = &models.ChefSubscriptionConfig{ChefID: chefID, CutoffTime: "21:00", TrialDurationDays: 3}
	}
	c.JSON(http.StatusOK, gin.H{
		"config":             cfg,
		"hasPublishedMenu":   services.ChefHasPublishedWeeklyMenu(database.DB, chefID),
	})
}

// UpdateChefSubscriptionConfig upserts the chef's offer. Enabling requires a
// published weekly menu (#1).
// PUT /chef/subscription-config
func (h *MealSubscriptionHandler) UpdateChefSubscriptionConfig(c *gin.Context) {
	chefID, ok := h.chefIDFor(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}
	var req struct {
		Enabled           bool     `json:"enabled"`
		Slots             []string `json:"slots"`
		Cadences          []string `json:"cadences"`
		PerMealPrice      float64  `json:"perMealPrice"`
		DeliveryFee       float64  `json:"deliveryFee"`
		DailyCapacity     int      `json:"dailyCapacity"`
		CutoffTime        string   `json:"cutoffTime"`
		TrialEnabled      bool     `json:"trialEnabled"`
		TrialDurationDays int      `json:"trialDurationDays"`
		TrialPrice        float64  `json:"trialPrice"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Enabled && !services.ChefHasPublishedWeeklyMenu(database.DB, chefID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Publish a weekly menu before enabling subscriptions"})
		return
	}
	if req.Enabled && req.PerMealPrice <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Set a per-meal price"})
		return
	}
	cutoff := req.CutoffTime
	if cutoff == "" {
		cutoff = "21:00"
	}

	cfg := models.ChefSubscriptionConfig{
		ChefID:            chefID,
		Enabled:           req.Enabled,
		Slots:             req.Slots,
		Cadences:          req.Cadences,
		PerMealPrice:      req.PerMealPrice,
		DeliveryFee:       req.DeliveryFee,
		DailyCapacity:     req.DailyCapacity,
		CutoffTime:        cutoff,
		TrialEnabled:      req.TrialEnabled,
		TrialDurationDays: req.TrialDurationDays,
		TrialPrice:        req.TrialPrice,
	}
	// Upsert on chef_id.
	var existing models.ChefSubscriptionConfig
	if err := database.DB.Where("chef_id = ?", chefID).First(&existing).Error; err == nil {
		cfg.ID = existing.ID
		// Select("*") so unchecking Enabled / clearing a fee (zero-values) persists.
		database.DB.Model(&existing).Select("*").Omit("id", "chef_id", "created_at").Updates(cfg)
		database.DB.Where("chef_id = ?", chefID).First(&cfg)
	} else if err := database.DB.Create(&cfg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"config": cfg})
}

// ── Customer: preview a chef's offer + price ────────────────────────────────

// GetChefOffer returns a chef's public subscription offer for the customer app.
// GET /chefs/:id/subscription
func (h *MealSubscriptionHandler) GetChefOffer(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	cfg := services.GetChefSubscriptionConfig(database.DB, chefID)
	if cfg == nil || !cfg.Enabled || !services.ChefHasPublishedWeeklyMenu(database.DB, chefID) {
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"available":         true,
		"slots":             cfg.Slots,
		"cadences":          cfg.Cadences,
		"perMealPrice":      cfg.PerMealPrice,
		"deliveryFee":       cfg.DeliveryFee,
		"cutoffTime":        cfg.CutoffTime,
		"trialEnabled":      cfg.TrialEnabled,
		"trialPrice":        cfg.TrialPrice,
		"trialDurationDays": cfg.TrialDurationDays,
	})
}

// PreviewPrice computes the live per-cycle price for a selection.
// POST /meal-subscriptions/preview
func (h *MealSubscriptionHandler) PreviewPrice(c *gin.Context) {
	var req mealSubSelection
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	chefID, err := uuid.Parse(req.ChefID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	cfg := services.GetChefSubscriptionConfig(database.DB, chefID)
	if vErr := services.ValidateMealSelection(cfg, req.Slots, req.Days, req.Cadence); vErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": vErr.Error()})
		return
	}
	amount := services.ComputeMealCycleAmount(cfg.PerMealPrice, len(req.Slots), len(req.Days), req.Cadence, cfg.DeliveryFee)
	c.JSON(http.StatusOK, gin.H{"cycleAmount": amount, "currency": "INR", "deliveryFee": cfg.DeliveryFee})
}

// ── Customer: subscribe + lifecycle ─────────────────────────────────────────

type mealSubSelection struct {
	ChefID    string   `json:"chefId" binding:"required"`
	Slots     []string `json:"slots" binding:"required"`
	Days      []int64  `json:"days" binding:"required"`
	Variant   string   `json:"variant"`
	Cadence   string   `json:"cadence" binding:"required"`
	AddressID string   `json:"addressId"`
}

// Subscribe creates a meal subscription. NOTE (#281): this foundation activates the
// subscription directly; the Razorpay UPI-Autopay mandate + first charge gate
// activation in the billing phase.
// POST /meal-subscriptions
func (h *MealSubscriptionHandler) Subscribe(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var req mealSubSelection
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	chefID, err := uuid.Parse(req.ChefID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	cfg := services.GetChefSubscriptionConfig(database.DB, chefID)
	if vErr := services.ValidateMealSelection(cfg, req.Slots, req.Days, req.Cadence); vErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": vErr.Error()})
		return
	}
	variant := models.MealVariant(req.Variant)
	if variant != models.MealVariantVeg && variant != models.MealVariantNonVeg {
		variant = models.MealVariantVeg
	}
	// One live subscription per (customer, chef) — block duplicates so #282's
	// order-generation can't produce double tiffins/charges. A DB partial unique
	// index backstops this against concurrent submits.
	var dup int64
	database.DB.Model(&models.MealSubscription{}).
		Where("customer_id = ? AND chef_id = ? AND status <> ?", userID, chefID, models.MealSubStatusCancelled).
		Count(&dup)
	if dup > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "You already have a subscription with this chef"})
		return
	}

	amount := services.ComputeMealCycleAmount(cfg.PerMealPrice, len(req.Slots), len(req.Days), req.Cadence, cfg.DeliveryFee)

	now := time.Now()
	periodEnd := services.AddMealCadence(now, req.Cadence)
	sub := models.MealSubscription{
		CustomerID:   userID,
		ChefID:       chefID,
		Slots:        req.Slots,
		Days:         req.Days,
		Variant:      variant,
		Cadence:      req.Cadence,
		PerMealPrice: cfg.PerMealPrice,
		DeliveryFee:  cfg.DeliveryFee,
		CycleAmount:  amount,
		Currency:     "INR",
		// Start trialing — NOT active. The Razorpay UPI-Autopay mandate + first
		// charge (#281) flip it to active; until then it generates no orders (#282).
		Status:             models.MealSubStatusTrialing,
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   &periodEnd,
	}
	if req.AddressID != "" {
		if aid, e := uuid.Parse(req.AddressID); e == nil {
			sub.DefaultAddressID = &aid
		}
	}
	if err := database.DB.Create(&sub).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create subscription"})
		return
	}
	_ = services.EnqueueEvent(database.DB, services.SubjectMealSubscriptionCreated, "subscription.meal.created", userID, map[string]any{
		"meal_subscription_id": sub.ID.String(),
		"chef_id":              chefID.String(),
		"cycle_amount":         amount,
		"cadence":              req.Cadence,
	})
	c.JSON(http.StatusCreated, gin.H{"subscription": sub})
}

// GetMySubscriptions lists the customer's meal subscriptions.
// GET /meal-subscriptions
func (h *MealSubscriptionHandler) GetMySubscriptions(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var subs []models.MealSubscription
	database.DB.Where("customer_id = ?", userID).Order("created_at DESC").Find(&subs)
	c.JSON(http.StatusOK, gin.H{"data": subs, "count": len(subs)})
}

// Pause halts order generation. POST /meal-subscriptions/:id/pause
func (h *MealSubscriptionHandler) Pause(c *gin.Context) {
	h.transition(c, services.CanPauseMealSub, models.MealSubStatusPaused, services.SubjectMealSubscriptionPaused, "subscription.meal.paused")
}

// Resume restarts an active subscription. POST /meal-subscriptions/:id/resume
func (h *MealSubscriptionHandler) Resume(c *gin.Context) {
	h.transition(c, services.CanResumeMealSub, models.MealSubStatusActive, services.SubjectMealSubscriptionResumed, "subscription.meal.resumed")
}

// Skip records a skipped delivery date (credited to the next cycle).
// POST /meal-subscriptions/:id/skip  { "date": "YYYY-MM-DD" }
func (h *MealSubscriptionHandler) Skip(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	sub, ok := h.ownedSub(c, userID)
	if !ok {
		return
	}
	if sub.Status != models.MealSubStatusActive && sub.Status != models.MealSubStatusTrialing {
		c.JSON(http.StatusConflict, gin.H{"error": "You can only skip days on an active subscription"})
		return
	}
	var req struct {
		Date string `json:"date" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Cutoffs are IST — compare the skip date against the current IST calendar day
	// so a customer can only skip a future day (full cutoff-time enforcement is #282).
	ist, lerr := time.LoadLocation("Asia/Kolkata")
	if lerr != nil {
		ist = time.UTC
	}
	d, err := time.ParseInLocation("2006-01-02", req.Date, ist)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date (use YYYY-MM-DD)"})
		return
	}
	nowIST := time.Now().In(ist)
	todayIST := time.Date(nowIST.Year(), nowIST.Month(), nowIST.Day(), 0, 0, 0, 0, ist)
	if !d.After(todayIST) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You can only skip a future day before its cutoff"})
		return
	}
	skip := models.MealSubscriptionSkip{MealSubscriptionID: sub.ID, Date: d}
	if err := database.DB.Create(&skip).Error; err != nil {
		// Unique (sub, date) — already skipped is fine (idempotent).
		c.JSON(http.StatusOK, gin.H{"message": "Day already skipped"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Day skipped", "date": req.Date})
}

// Cancel terminates the subscription and fires the win-back offer (#278).
// POST /meal-subscriptions/:id/cancel
func (h *MealSubscriptionHandler) Cancel(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	sub, ok := h.ownedSub(c, userID)
	if !ok {
		return
	}
	if !services.CanCancelMealSub(sub.Status) {
		c.JSON(http.StatusConflict, gin.H{"error": "This subscription can't be cancelled"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	now := time.Now()
	database.DB.Model(&sub).Updates(map[string]any{
		"status":        models.MealSubStatusCancelled,
		"cancelled_at":  &now,
		"cancel_reason": req.Reason,
	})
	_ = services.EnqueueEvent(database.DB, services.SubjectMealSubscriptionCancelled, "subscription.meal.cancelled", userID, map[string]any{
		"meal_subscription_id": sub.ID.String(),
		"chef_id":              sub.ChefID.String(),
	})
	// Win-back (#278): offer a discounted return to the cancelling customer.
	if _, werr := services.OfferWinback(database.DB, userID, models.WinbackAudienceCustomer, models.WinbackTriggerMealSubCancelled, &sub.ID); werr != nil {
		log.Printf("winback: offer on meal-sub cancel failed for user=%s: %v", userID, werr)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Subscription cancelled"})
}

// ── helpers ─────────────────────────────────────────────────────────────────

func (h *MealSubscriptionHandler) transition(c *gin.Context, allowed func(string) bool, to, subject, eventType string) {
	userID, _ := middleware.GetUserID(c)
	sub, ok := h.ownedSub(c, userID)
	if !ok {
		return
	}
	if !allowed(sub.Status) {
		c.JSON(http.StatusConflict, gin.H{"error": "Not allowed from the current status"})
		return
	}
	updates := map[string]any{"status": to}
	if to == models.MealSubStatusPaused {
		now := time.Now()
		updates["paused_at"] = &now
	}
	database.DB.Model(&sub).Updates(updates)
	_ = services.EnqueueEvent(database.DB, subject, eventType, userID, map[string]any{
		"meal_subscription_id": sub.ID.String(),
	})
	c.JSON(http.StatusOK, gin.H{"status": to})
}

func (h *MealSubscriptionHandler) ownedSub(c *gin.Context, userID uuid.UUID) (models.MealSubscription, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subscription id"})
		return models.MealSubscription{}, false
	}
	var sub models.MealSubscription
	if err := database.DB.Where("id = ? AND customer_id = ?", id, userID).First(&sub).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
		return models.MealSubscription{}, false
	}
	return sub, true
}

func (h *MealSubscriptionHandler) chefIDFor(c *gin.Context) (uuid.UUID, bool) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Select("id").Where("user_id = ?", userID).First(&chef).Error; err != nil {
		return uuid.Nil, false
	}
	return chef.ID, true
}
