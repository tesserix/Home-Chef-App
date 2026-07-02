package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// meal_plan.go — the tiffin meal-plan handshake (#194/#195/#196): customer creates
// a multi-day calendar request from one chef → chef accepts all or cherry-picks →
// customer approves any trim → confirmed. Integration: DB transactions, NATS
// transactional outbox (meal_plans.*), a Redis idempotency lock per mutation, and
// strict per-customer / per-vendor isolation (every query scoped to the authed
// owner; 404 — never 403 — to avoid leaking existence). The escrow money flow
// (capture/refund/release) is gated by config.MealPlanEscrowEnabled (#194) and the
// expiry sweeps run via the Temporal cron (services/meal_plan_cron.go, #197).

const (
	mealPlanMaxDays = 31
	// chefRespondWindow: the chef must accept/cherry-pick within 12h of the
	// customer's request, else the expiry sweep voids it. custApproveWindow: once
	// the chef cherry-picks, the customer must approve the revised plan within 6h.
	// A lapse of either window expires the plan; the customer can always book a
	// fresh request (no constraint blocks rebooking after expiry).
	chefRespondWindow = 12 * time.Hour
	custApproveWindow = 6 * time.Hour
	// A day must be booked at least this far ahead (lead time before the kitchen
	// needs to plan). Tunable; IST is the business timezone.
	mealPlanLeadTime = 12 * time.Hour
)

// istLoc is the business timezone (IST, no DST). Booking dates (YYYY-MM-DD) are
// interpreted as IST midnight so the lead-time cutoff is correct regardless of
// the server clock's zone (containers run UTC).
var istLoc = time.FixedZone("IST", 5*3600+30*60)

// errPlanConflict signals that a meal-plan transition lost a race — the row was
// no longer in the expected status (e.g. the expiry cron got there first). The
// handler maps it to 409 rather than 500.
var errPlanConflict = errors.New("meal plan state changed concurrently")

// parsePlanDate parses a YYYY-MM-DD booking date as IST midnight.
func parsePlanDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, istLoc)
}

// dayBeforeCutoff reports whether a booked day starts before the lead-time
// cutoff (now + mealPlanLeadTime). Extracted for unit testing.
func dayBeforeCutoff(dateStr string, now time.Time) (bool, error) {
	d, err := parsePlanDate(dateStr)
	if err != nil {
		return false, err
	}
	return d.Before(now.Add(mealPlanLeadTime)), nil
}

// MealPlanHandler owns the customer + chef meal-plan endpoints.
type MealPlanHandler struct{}

func NewMealPlanHandler() *MealPlanHandler { return &MealPlanHandler{} }

// idempotencyGuard takes a short-lived Redis lock so a double-submitted mutation
// (retry, double-tap) is processed once. Fails OPEN if Redis is down (the DB
// state-machine guards are the real backstop).
func idempotencyGuard(ctx context.Context, key string) (bool, func()) {
	r := services.GetRedisClient()
	if r == nil || !r.IsConnected() {
		return true, func() {}
	}
	ok, err := r.SetNX(ctx, "mealplan:lock:"+key, "1", 30*time.Second)
	if err != nil {
		return true, func() {} // fail open
	}
	if !ok {
		return false, func() {}
	}
	return true, func() { _ = r.Del(ctx, "mealplan:lock:"+key) }
}

func mealPlanNumber() string {
	return "MP-" + uuid.NewString()[:8]
}

// ───────────────────────── Customer ─────────────────────────

type createMealPlanDayInput struct {
	Date    string `json:"date"`    // YYYY-MM-DD
	Slot    string `json:"slot"`    // lunch|dinner
	Variant string `json:"variant"` // veg|nonveg
}

type createMealPlanRequest struct {
	ChefID string                   `json:"chefId"`
	Days   []createMealPlanDayInput `json:"days"`
}

// CreateMealPlan — POST /meal-plans. Customer books a calendar of days from one chef.
func (h *MealPlanHandler) CreateMealPlan(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)

	var req createMealPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	chefID, err := uuid.Parse(req.ChefID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chefId"})
		return
	}
	if len(req.Days) == 0 || len(req.Days) > mealPlanMaxDays {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("a plan needs between 1 and %d days", mealPlanMaxDays)})
		return
	}

	ok, release := idempotencyGuard(c.Request.Context(), fmt.Sprintf("create:%s:%s", customerID, chefID))
	if !ok {
		c.JSON(http.StatusConflict, gin.H{"error": "A request is already in progress"})
		return
	}
	defer release()

	// Chef must exist, be active, and hold a valid FSSAI licence (#91 gate).
	var chef models.ChefProfile
	if err := database.DB.Where("id = ? AND is_active = ?", chefID, true).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	if services.IsChefFSSAIExpired(&chef) {
		c.JSON(http.StatusConflict, gin.H{"error": "This chef isn't accepting orders right now"})
		return
	}

	// The chef must have a PUBLISHED weekly menu — draft cells exist in the items
	// table but are not bookable (the public read gates on is_published too).
	if err := database.DB.Where("chef_id = ? AND is_published = ?", chefID, true).
		First(&models.WeeklyMenu{}).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "This chef hasn't published a weekly menu yet"})
		return
	}

	// Resolve each booked cell against the chef's PUBLISHED weekly menu (#192).
	var cells []models.WeeklyMenuItem
	database.DB.Where("chef_id = ?", chefID).Find(&cells)
	cellBy := map[string]models.WeeklyMenuItem{}
	for _, cell := range cells {
		cellBy[fmt.Sprintf("%d|%s|%s", cell.DayOfWeek, cell.Slot, cell.Variant)] = cell
	}

	cutoff := time.Now().Add(mealPlanLeadTime)
	seen := map[string]bool{}
	days := make([]models.MealPlanDay, 0, len(req.Days))
	var subtotal float64
	var minDate, maxDate time.Time

	for _, d := range req.Days {
		date, derr := parsePlanDate(d.Date)
		if derr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must be YYYY-MM-DD"})
			return
		}
		if !validSlot(d.Slot) || !validVariant(d.Variant) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "slot must be lunch|dinner, variant veg|nonveg"})
			return
		}
		// One booking per (date, slot) — can't double-book a slot.
		key := d.Date + "|" + d.Slot
		if seen[key] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "duplicate day+slot: " + key})
			return
		}
		seen[key] = true
		if date.Before(cutoff) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "day " + d.Date + " is past the booking cutoff"})
			return
		}
		cell, has := cellBy[fmt.Sprintf("%d|%s|%s", int(date.Weekday()), d.Slot, d.Variant)]
		if !has {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("chef has no %s %s on %s", d.Variant, d.Slot, d.Date)})
			return
		}
		subtotal += cell.Price
		cellID := cell.ID
		days = append(days, models.MealPlanDay{
			Date:             date,
			Slot:             models.MealSlot(d.Slot),
			Variant:          models.MealVariant(d.Variant),
			WeeklyMenuItemID: &cellID,
			DishName:         cell.Name,
			Price:            cell.Price,
			Status:           models.MealPlanDayRequested,
		})
		if minDate.IsZero() || date.Before(minDate) {
			minDate = date
		}
		if date.After(maxDate) {
			maxDate = date
		}
	}

	respondBy := time.Now().Add(chefRespondWindow)
	plan := models.MealPlan{
		MealPlanNumber: mealPlanNumber(),
		CustomerID:     customerID,
		ChefID:         chefID,
		Status:         models.MealPlanPendingChef,
		StartDate:      minDate,
		EndDate:        maxDate,
		Subtotal:       subtotal,
		Total:          subtotal, // tax/GST + delivery resolved at per-day order generation (#197)
		Currency:       "INR",
		ChefRespondBy:  &respondBy,
		Days:           days,
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&plan).Error; err != nil {
			return err
		}
		return services.EnqueueEvent(tx, services.SubjectMealPlanCreated, "meal_plan.created", chef.UserID, map[string]any{
			"meal_plan_id": plan.ID.String(),
			"meal_plan_no": plan.MealPlanNumber,
			"customer_id":  customerID.String(),
			"chef_id":      chefID.String(),
			"day_count":    len(days),
			"total":        subtotal,
		})
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create meal plan"})
		return
	}

	// Escrow advance (#194) — gated. When enabled, the customer pays the advance
	// into the platform account here and the held chef payouts are created; until
	// then the plan proceeds to the chef without charging (handshake mode).
	resp := gin.H{"mealPlan": plan, "escrowEnabled": config.AppConfig.MealPlanEscrowEnabled}
	// Escrow (gated): create the Razorpay advance order for the full total and
	// hand the client what it needs to launch checkout, then POST verify-payment.
	if services.MealPlanEscrowActive() {
		orderID, oerr := services.CreateMealPlanAdvanceOrder(&plan)
		if oerr != nil {
			resp["paymentError"] = "Could not start the advance payment; please retry."
		} else if orderID != "" {
			database.DB.Model(&models.MealPlan{}).Where("id = ?", plan.ID).
				Update("razorpay_order_id", orderID)
			resp["razorpayOrderId"] = orderID
			if rz := services.GetRazorpay(); rz != nil {
				resp["razorpayKeyId"] = rz.GetKeyID()
			}
		}
	}
	c.JSON(http.StatusCreated, resp)
}

// GetMyMealPlans — GET /meal-plans. Customer's own plans only (isolation).
func (h *MealPlanHandler) GetMyMealPlans(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	var plans []models.MealPlan
	database.DB.Where("customer_id = ?", customerID).
		Preload("Days").Preload("Chef").Order("created_at DESC").Find(&plans)
	c.JSON(http.StatusOK, gin.H{"data": plans})
}

// GetMealPlan — GET /meal-plans/:id. Scoped to the authed customer (404 if not theirs).
func (h *MealPlanHandler) GetMealPlan(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	plan, ok := loadScopedPlan(c.Param("id"), "customer_id", customerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal plan not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mealPlan": plan})
}

// ApproveMealPlan — PUT /meal-plans/:id/approve. Customer accepts the chef's
// trimmed set; declined days are refunded (escrow), the rest is confirmed.
func (h *MealPlanHandler) ApproveMealPlan(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	h.finalizeByCustomer(c, customerID, true)
}

// RejectMealPlan — PUT /meal-plans/:id/reject. Customer declines the trim → full
// cancel + refund.
func (h *MealPlanHandler) RejectMealPlan(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	h.finalizeByCustomer(c, customerID, false)
}

func (h *MealPlanHandler) finalizeByCustomer(c *gin.Context, customerID uuid.UUID, approve bool) {
	plan, ok := loadScopedPlan(c.Param("id"), "customer_id", customerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal plan not found"})
		return
	}
	if plan.Status != models.MealPlanAwaitingCustomer {
		c.JSON(http.StatusConflict, gin.H{"error": "This plan is not awaiting your approval"})
		return
	}

	guardKey := fmt.Sprintf("finalize:%s", plan.ID)
	okLock, release := idempotencyGuard(c.Request.Context(), guardKey)
	if !okLock {
		c.JSON(http.StatusConflict, gin.H{"error": "Already processing"})
		return
	}
	defer release()

	// Load the chef for the recipient User.ID (events target User.ID, not
	// ChefProfile.ID) and the Razorpay linked account (escrow payouts).
	var chefProfile models.ChefProfile
	database.DB.First(&chefProfile, "id = ?", plan.ChefID)
	chefUserID := chefProfile.UserID

	now := time.Now()
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		newStatus := models.MealPlanConfirmed
		if !approve {
			newStatus = models.MealPlanCancelled
		}
		// Status-guarded transition: only act if still awaiting_customer. A
		// concurrent expiry sweep (or double-submit) that already moved the row
		// loses here, so we never overwrite a terminal state.
		res := tx.Model(&models.MealPlan{}).
			Where("id = ? AND status = ?", plan.ID, models.MealPlanAwaitingCustomer).
			Update("status", newStatus)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errPlanConflict
		}

		if approve {
			// Accepted days → confirmed; declined days stay declined (refunded by #194).
			if err := tx.Model(&models.MealPlanDay{}).
				Where("meal_plan_id = ? AND status = ?", plan.ID, models.MealPlanDayAccepted).
				Update("status", models.MealPlanDayConfirmed).Error; err != nil {
				return err
			}
			for i := range plan.Days {
				if plan.Days[i].Status == models.MealPlanDayAccepted {
					plan.Days[i].Status = models.MealPlanDayConfirmed
				}
			}
			plan.Status = models.MealPlanConfirmed
			plan.ConfirmedAt = &now
			// Charge basis = only the accepted days, not the original full request.
			accepted := plan.AcceptedTotal()
			plan.Subtotal, plan.Total = accepted, accepted
			if err := tx.Model(&models.MealPlan{}).Where("id = ?", plan.ID).
				Updates(map[string]any{"confirmed_at": now, "subtotal": accepted, "total": accepted}).Error; err != nil {
				return err
			}
			// Escrow (gated): refund the declined days, hold the chef's per-day payouts.
			if err := services.RefundDeclinedDays(tx, &plan, "chef could not cook this day"); err != nil {
				return err
			}
			if err := services.HoldChefPayouts(tx, &plan, chefProfile.RazorpayAccountID); err != nil {
				return err
			}
		} else {
			plan.Status = models.MealPlanCancelled
			plan.CancelledAt = &now
			plan.CancelReason = "customer rejected the chef's revised plan"
			if err := tx.Model(&models.MealPlan{}).Where("id = ?", plan.ID).
				Updates(map[string]any{"cancelled_at": now, "cancel_reason": plan.CancelReason}).Error; err != nil {
				return err
			}
			// Escrow (gated): full refund of everything still in scope.
			if err := services.RefundUndeliveredDays(tx, &plan, "customer rejected the revised plan"); err != nil {
				return err
			}
		}

		subj := services.SubjectMealPlanConfirmed
		if !approve {
			subj = services.SubjectMealPlanCancelled
		}
		return services.EnqueueEvent(tx, subj, "meal_plan.finalized", chefUserID, map[string]any{
			"meal_plan_id": plan.ID.String(), "meal_plan_no": plan.MealPlanNumber,
			"approved": approve, "customer_id": customerID.String(), "chef_id": plan.ChefID.String(),
		})
	}); err != nil {
		if errors.Is(err, errPlanConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "This plan is no longer awaiting your approval"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize meal plan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mealPlan": plan})
}

// SkipMealPlanDay — PUT /meal-plans/:id/days/:dayId/skip. The customer skips a
// confirmed day whose order hasn't been generated yet, before the lead-time
// cutoff; the day is refunded to wallet (escrow; gated).
func (h *MealPlanHandler) SkipMealPlanDay(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	plan, ok := loadScopedPlan(c.Param("id"), "customer_id", customerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal plan not found"})
		return
	}
	dayID, err := uuid.Parse(c.Param("dayId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day id"})
		return
	}
	var day *models.MealPlanDay
	for i := range plan.Days {
		if plan.Days[i].ID == dayID {
			day = &plan.Days[i]
			break
		}
	}
	if day == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Day not found in this plan"})
		return
	}
	// Only a confirmed day whose order hasn't been generated, before the cutoff.
	if day.Status != models.MealPlanDayConfirmed || day.OrderID != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "This day can no longer be skipped"})
		return
	}
	if day.Date.Before(time.Now().Add(mealPlanLeadTime)) {
		c.JSON(http.StatusConflict, gin.H{"error": "Too late to skip this day"})
		return
	}

	okLock, release := idempotencyGuard(c.Request.Context(), fmt.Sprintf("skipday:%s", day.ID))
	if !okLock {
		c.JSON(http.StatusConflict, gin.H{"error": "Already processing"})
		return
	}
	defer release()

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.MealPlanDay{}).
			Where("id = ? AND status = ?", day.ID, models.MealPlanDayConfirmed).
			Update("status", models.MealPlanDaySkipped)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errPlanConflict
		}
		day.Status = models.MealPlanDaySkipped
		if err := services.RefundDay(tx, &plan, day, "customer skipped this day"); err != nil {
			return err
		}
		return services.EnqueueEvent(tx, services.SubjectMealPlanDayRefunded, "meal_plan_day.skipped", customerID, map[string]any{
			"meal_plan_id": plan.ID.String(), "day_id": day.ID.String(),
		})
	}); err != nil {
		if errors.Is(err, errPlanConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "This day can no longer be skipped"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to skip day"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mealPlan": plan})
}

type verifyMealPlanPaymentRequest struct {
	RazorpayPaymentID string `json:"razorpayPaymentId"`
	RazorpaySignature string `json:"razorpaySignature"`
}

// VerifyMealPlanPayment — POST /meal-plans/:id/verify-payment. Confirms the
// customer's advance payment was captured and stamps the escrow payment id.
// No-op acknowledgement when escrow is off.
func (h *MealPlanHandler) VerifyMealPlanPayment(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	plan, ok := loadScopedPlan(c.Param("id"), "customer_id", customerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal plan not found"})
		return
	}
	var req verifyMealPlanPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return services.VerifyMealPlanAdvance(tx, &plan, req.RazorpayPaymentID, req.RazorpaySignature)
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment verification failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mealPlan": plan, "paymentVerified": true})
}

// ───────────────────────── Chef ─────────────────────────

// GetChefMealPlanRequests — GET /chef/meal-plans. Pending requests for the authed chef only.
func (h *MealPlanHandler) GetChefMealPlanRequests(c *gin.Context) {
	chef, ok := authedChef(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	status := c.DefaultQuery("status", string(models.MealPlanPendingChef))
	var plans []models.MealPlan
	q := database.DB.Where("chef_id = ? AND status = ?", chef.ID, status)
	// When escrow is on, only surface plans whose advance the customer has paid —
	// the chef shouldn't act on (or be held liable to cook for) an unpaid request.
	if services.MealPlanEscrowActive() {
		q = q.Where("escrow_payment_id <> ''")
	}
	q.Preload("Days").Preload("Customer").Order("created_at DESC").Find(&plans)
	c.JSON(http.StatusOK, gin.H{"data": plans})
}

type respondMealPlanRequest struct {
	AcceptAll      bool     `json:"acceptAll"`
	AcceptedDayIDs []string `json:"acceptedDayIds"` // when not acceptAll: the days the chef will cook
}

// RespondMealPlan — POST /chef/meal-plans/:id/respond. Chef accepts all days or
// cherry-picks a subset. All accepted → auto-confirm (customer just gets notified);
// a trim → back to the customer for approval.
func (h *MealPlanHandler) RespondMealPlan(c *gin.Context) {
	chef, ok := authedChef(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	plan, ok := loadScopedPlan(c.Param("id"), "chef_id", chef.ID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal plan not found"})
		return
	}
	if plan.Status != models.MealPlanPendingChef {
		c.JSON(http.StatusConflict, gin.H{"error": "This plan has already been responded to"})
		return
	}

	var req respondMealPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accepted := map[uuid.UUID]bool{}
	for _, raw := range req.AcceptedDayIDs {
		if id, perr := uuid.Parse(raw); perr == nil {
			accepted[id] = true
		}
	}
	if !req.AcceptAll && len(accepted) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accept at least one day, or set acceptAll"})
		return
	}

	okLock, release := idempotencyGuard(c.Request.Context(), fmt.Sprintf("respond:%s", plan.ID))
	if !okLock {
		c.JSON(http.StatusConflict, gin.H{"error": "Already processing"})
		return
	}
	defer release()

	allAccepted := true
	for i := range plan.Days {
		if req.AcceptAll || accepted[plan.Days[i].ID] {
			plan.Days[i].Status = models.MealPlanDayAccepted
		} else {
			plan.Days[i].Status = models.MealPlanDayDeclined
			allAccepted = false
		}
	}

	now := time.Now()
	subj := services.SubjectMealPlanModified
	if allAccepted {
		plan.Status = models.MealPlanConfirmed
		plan.ConfirmedAt = &now
		// Accepted-all → confirm the days immediately too.
		for i := range plan.Days {
			plan.Days[i].Status = models.MealPlanDayConfirmed
		}
		subj = services.SubjectMealPlanAcceptedFull
	} else {
		plan.Status = models.MealPlanAwaitingCustomer
		approveBy := now.Add(custApproveWindow)
		plan.CustomerApproveBy = &approveBy
	}

	// Charge basis = the accepted days only (declined days are excluded).
	acceptedTotal := plan.AcceptedTotal()
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Status-guarded transition from pending_chef — a concurrent expiry sweep
		// (or double-submit) that already moved the row loses here.
		res := tx.Model(&models.MealPlan{}).
			Where("id = ? AND status = ?", plan.ID, models.MealPlanPendingChef).
			Update("status", plan.Status)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errPlanConflict
		}
		for i := range plan.Days {
			if err := tx.Model(&models.MealPlanDay{}).
				Where("id = ? AND meal_plan_id = ?", plan.Days[i].ID, plan.ID).
				Update("status", plan.Days[i].Status).Error; err != nil {
				return err
			}
		}
		upd := map[string]any{"subtotal": acceptedTotal, "total": acceptedTotal}
		if allAccepted {
			upd["confirmed_at"] = now
		} else {
			upd["customer_approve_by"] = plan.CustomerApproveBy
		}
		if err := tx.Model(&models.MealPlan{}).Where("id = ?", plan.ID).Updates(upd).Error; err != nil {
			return err
		}
		// Accept-all auto-confirms → hold the chef's per-day payouts now (escrow;
		// gated). A cherry-pick waits for the customer's approval (finalizeByCustomer).
		if allAccepted {
			if err := services.HoldChefPayouts(tx, &plan, chef.RazorpayAccountID); err != nil {
				return err
			}
		}
		return services.EnqueueEvent(tx, subj, "meal_plan.responded", plan.CustomerID, map[string]any{
			"meal_plan_id": plan.ID.String(), "all_accepted": allAccepted, "chef_id": chef.ID.String(),
		})
	}); err != nil {
		if errors.Is(err, errPlanConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "This plan is no longer pending your response"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record response"})
		return
	}
	plan.Subtotal, plan.Total = acceptedTotal, acceptedTotal
	c.JSON(http.StatusOK, gin.H{"mealPlan": plan, "allAccepted": allAccepted})
}

// ───────────────────────── Admin (oversight, #199) ─────────────────────────

// AdminListMealPlans — GET /admin/meal-plans. Platform-wide, read-only oversight.
// The route is gated by middleware.RequireAdmin (no per-row owner scoping here —
// that's the whole point of admin oversight). Paginated, optional status filter.
func (h *MealPlanHandler) AdminListMealPlans(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := database.DB.Model(&models.MealPlan{})
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	q.Count(&total)

	var plans []models.MealPlan
	q.Preload("Days").Preload("Customer").Preload("Chef").
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&plans)

	c.JSON(http.StatusOK, gin.H{
		"data": plans,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
		},
	})
}

// AdminGetMealPlan — GET /admin/meal-plans/:id. Full plan (days + parties) for
// oversight; admin-gated, not owner-scoped.
func (h *MealPlanHandler) AdminGetMealPlan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	var plan models.MealPlan
	if err := database.DB.Preload("Days").Preload("Customer").Preload("Chef").
		First(&plan, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal plan not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mealPlan": plan})
}

// ───────────────────────── helpers ─────────────────────────

// loadScopedPlan loads a plan by id ONLY if it belongs to the given owner column
// (customer_id or chef_id) = ownerID — per-customer / per-vendor isolation.
func loadScopedPlan(idParam, ownerCol string, ownerID uuid.UUID) (models.MealPlan, bool) {
	id, err := uuid.Parse(idParam)
	if err != nil {
		return models.MealPlan{}, false
	}
	var plan models.MealPlan
	if err := database.DB.Preload("Days").Preload("Chef").Preload("Customer").
		Where("id = ? AND "+ownerCol+" = ?", id, ownerID).First(&plan).Error; err != nil {
		return models.MealPlan{}, false
	}
	return plan, true
}

func authedChef(c *gin.Context) (models.ChefProfile, bool) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		return models.ChefProfile{}, false
	}
	return chef, true
}
