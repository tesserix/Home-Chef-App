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

// parseClockHHMM parses an "HH:MM" 24-hour clock string into (hour, minute, ok). ok is
// false for any malformed or out-of-range value so callers fall back to a slot default.
func parseClockHHMM(s string) (int, int, bool) {
	var hh, mm int
	if n, err := fmt.Sscanf(s, "%d:%d", &hh, &mm); err != nil || n != 2 {
		return 0, 0, false
	}
	if hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return 0, 0, false
	}
	return hh, mm, true
}

// mealPlanDayStartIST approximates the IST start time of a plan day's cooking — the
// anchor for the 12h-before-start skip gate (#422). There is no exact per-slot start
// time stored, so it uses the chef's ChefSchedule OpenTime for that IST weekday (an
// open row carrying a valid HH:MM). Fallback order: (1) that schedule row's OpenTime;
// (2) a per-slot default — lunch 12:00 IST, dinner 19:00 IST — when there is no
// schedule row for the weekday, the row is marked closed, or its OpenTime is
// missing/malformed. Pure (no DB) so it is unit-tested directly.
func mealPlanDayStartIST(schedules []models.ChefSchedule, day *models.MealPlanDay) time.Time {
	d := day.Date.In(istLoc)
	weekday := int(d.Weekday()) // 0=Sunday .. 6=Saturday — matches ChefSchedule.DayOfWeek
	for i := range schedules {
		s := schedules[i]
		if s.DayOfWeek != weekday || s.IsClosed {
			continue
		}
		if hh, mm, ok := parseClockHHMM(s.OpenTime); ok {
			return time.Date(d.Year(), d.Month(), d.Day(), hh, mm, 0, 0, istLoc)
		}
	}
	hh := 12 // lunch default
	if day.Slot == models.MealSlotDinner {
		hh = 19 // dinner default
	}
	return time.Date(d.Year(), d.Month(), d.Day(), hh, 0, 0, 0, istLoc)
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
	Variant string `json:"variant"` // veg|nonveg (weekly-menu bookings)
	// DailyMenuItemID (#406) — when set, book this specific dish/combo from the
	// chef's PUBLISHED per-date menu for that date, overriding the weekly cell.
	// The variant is taken from the item.
	DailyMenuItemID string `json:"dailyMenuItemId"`
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
		if !validSlot(d.Slot) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "slot must be lunch|dinner"})
			return
		}
		if d.DailyMenuItemID == "" && !validVariant(d.Variant) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "variant must be veg|nonveg"})
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

		// Per-date override (#406): the customer picked a specific dish/combo from
		// the chef's PUBLISHED daily menu for that date. Book it directly and take
		// its price + variant; a combo carries its set bundle price.
		if d.DailyMenuItemID != "" {
			itemID, perr := uuid.Parse(d.DailyMenuItemID)
			if perr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dailyMenuItemId"})
				return
			}
			var item models.DailyMenuItem
			if err := database.DB.
				Joins("JOIN daily_menus ON daily_menus.id = daily_menu_items.daily_menu_id").
				Where("daily_menu_items.id = ? AND daily_menu_items.chef_id = ? AND daily_menu_items.date = ? AND daily_menu_items.slot = ? AND daily_menus.is_published = ?",
					itemID, chefID, date, d.Slot, true).
				First(&item).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("that dish isn't on the chef's published menu for %s %s", d.Slot, d.Date)})
				return
			}
			subtotal += item.Price
			days = append(days, models.MealPlanDay{
				Date:     date,
				Slot:     models.MealSlot(d.Slot),
				Variant:  item.Variant,
				DishName: item.Name,
				Price:    item.Price,
				Status:   models.MealPlanDayRequested,
			})
			if minDate.IsZero() || date.Before(minDate) {
				minDate = date
			}
			if date.After(maxDate) {
				maxDate = date
			}
			continue
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

	// #409: one active plan per (customer, chef, week). Block a second request that
	// overlaps an existing plan still in the pending→active lifecycle, so a
	// customer can't double-book a chef for the same dates. Re-allowed once the
	// prior plan is rejected/expired/cancelled or its dates pass.
	liveStatuses := []models.MealPlanStatus{
		models.MealPlanPendingChef, models.MealPlanChefAcceptedFull, models.MealPlanChefModified,
		models.MealPlanAwaitingCustomer, models.MealPlanConfirmed, models.MealPlanActive,
	}
	var existing models.MealPlan
	if err := database.DB.
		Where("customer_id = ? AND chef_id = ? AND status IN ? AND start_date <= ? AND end_date >= ?",
			customerID, chefID, liveStatuses, maxDate, minDate).
		First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":          "You already have a plan with this chef for these dates — it's with the chef for approval.",
			"code":           "duplicate_plan",
			"existingPlanId": existing.ID,
		})
		return
	}

	// Escrow (paid) plans charge the full amount upfront: food + GST + per-day
	// delivery. The chef is paid only the food price; the platform keeps GST +
	// delivery. Snapshot Tax/Total onto the plan so per-day refunds stay stable
	// even if platform policy changes after booking. Escrow off = unpaid handshake,
	// so Total stays the food subtotal (nothing is charged).
	planTax := 0.0
	planTotal := subtotal
	if services.MealPlanEscrowActive() {
		tax, delivery := services.MealPlanFeeTotals(subtotal, len(days))
		planTax = tax
		planTotal = services.Round2(subtotal + tax + delivery)
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
		Tax:            planTax,
		Total:          planTotal,
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

	// Payment now happens AFTER the chef responds and the customer APPROVES the
	// confirmed days (see ApproveMealPlan) — NOT at create. The request reaches the
	// chef with nothing charged; the advance order is created + collected only on
	// approval, for exactly the days the chef committed to. So create returns the
	// plan only (no razorpay order).
	c.JSON(http.StatusCreated, gin.H{"mealPlan": plan, "escrowEnabled": config.AppConfig.MealPlanEscrowEnabled})
}

// GetMyMealPlans — GET /meal-plans. Customer's own plans only (isolation).
func (h *MealPlanHandler) GetMyMealPlans(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	var plans []models.MealPlan
	database.DB.Where("customer_id = ?", customerID).
		Preload("Days").Preload("Chef").Order("created_at DESC").Find(&plans)
	for i := range plans {
		plans[i].ProjectForCustomer()
	}
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
	plan.ProjectForCustomer()
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

	// Escrow-on APPROVE does not confirm yet: it snapshots the accepted-days charge
	// (food + GST + per-accepted-day delivery), mints the Razorpay advance order,
	// and hands the client checkout. The plan STAYS awaiting_customer until the
	// payment verifies — VerifyMealPlanPayment then flips it to confirmed and holds
	// the chef payouts. (Escrow-OFF approve is the unpaid handshake → confirmed
	// directly, and reject cancels — both handled by the tx below.)
	if approve && services.MealPlanEscrowActive() {
		accSub := plan.AcceptedTotal()
		tax, delivery := services.MealPlanFeeTotals(accSub, plan.AcceptedDayCount())
		total := services.Round2(accSub + tax + delivery)

		// Snapshot the accepted-days charge, guarded so only one approval mints an
		// order and only while still awaiting_customer with none minted yet
		// (idempotent against a double-approve / retry).
		res := database.DB.Model(&models.MealPlan{}).
			Where("id = ? AND status = ? AND (razorpay_order_id IS NULL OR razorpay_order_id = '')",
				plan.ID, models.MealPlanAwaitingCustomer).
			Updates(map[string]any{"subtotal": accSub, "tax": tax, "total": total})
		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize meal plan"})
			return
		}
		if res.RowsAffected == 0 {
			// The guarded snapshot didn't apply — RESUME vs CONFLICT. If the plan is
			// still awaiting_customer with an advance order already minted but NOT yet
			// paid (escrow_payment_id blank), the customer backed out of Razorpay and is
			// retrying: hand back the SAME order to resume checkout, rather than 409
			// (which would strand a minted-but-unpaid advance and block re-payment).
			// Any other state (paid/confirmed/cancelled) is a genuine conflict.
			var cur models.MealPlan
			if err := database.DB.Preload("Days").First(&cur, "id = ?", plan.ID).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize meal plan"})
				return
			}
			if cur.Status == models.MealPlanAwaitingCustomer && cur.RazorpayOrderID != "" && cur.EscrowPaymentID == "" {
				resp := gin.H{"razorpayOrderId": cur.RazorpayOrderID}
				if rz := services.GetRazorpay(); rz != nil {
					resp["razorpayKeyId"] = rz.GetKeyID()
				}
				cur.ProjectForCustomer()
				resp["mealPlan"] = cur
				c.JSON(http.StatusOK, resp)
				return
			}
			c.JSON(http.StatusConflict, gin.H{"error": "This plan is no longer awaiting your approval"})
			return
		}
		plan.Subtotal, plan.Tax, plan.Total = accSub, tax, total

		// Mint the advance order OUTSIDE a tx (external Razorpay call), then stamp it.
		resp := gin.H{}
		orderID, oerr := services.CreateMealPlanAdvanceOrder(&plan)
		if oerr != nil {
			resp["paymentError"] = "Could not start the advance payment; please retry."
		} else if orderID != "" {
			database.DB.Model(&models.MealPlan{}).Where("id = ?", plan.ID).
				Update("razorpay_order_id", orderID)
			plan.RazorpayOrderID = orderID
			resp["razorpayOrderId"] = orderID
			if rz := services.GetRazorpay(); rz != nil {
				resp["razorpayKeyId"] = rz.GetKeyID()
			}
		}
		plan.ProjectForCustomer()
		resp["mealPlan"] = plan
		c.JSON(http.StatusOK, resp)
		return
	}

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
	plan.ProjectForCustomer()
	c.JSON(http.StatusOK, gin.H{"mealPlan": plan})
}

// CancelMealPlan — PUT /meal-plans/:id/cancel. The customer cancels a plan that
// has NOT started or been served: withdraw a pending request, back out while it's
// with the chef / awaiting their own approval, or cancel a confirmed plan whose
// first day hasn't been prepared. FULL refund of any advance collected (nothing
// was served), so no penalty math. Mid-week cancellation once days have been
// served (the >=20% penalty flow, #411) is out of scope and rejected here.
// Cancelling frees the customer to rebook — `cancelled` is not a "live" status,
// so the duplicate-plan guard (#409) no longer blocks a fresh request.
// mealPlanCancellableBeforeStart reports whether a customer may cancel this plan
// for a FULL refund — it must still be in flight or confirmed (not a terminal
// state), and no day may have been prepared or delivered. Once the chef has begun
// serving, cancellation is the mid-week penalty flow (#411), not this free one.
func mealPlanCancellableBeforeStart(status models.MealPlanStatus, days []models.MealPlanDay) (bool, string) {
	cancellable := map[models.MealPlanStatus]bool{
		models.MealPlanPendingChef:      true,
		models.MealPlanChefAcceptedFull: true,
		models.MealPlanChefModified:     true,
		models.MealPlanAwaitingCustomer: true,
		models.MealPlanConfirmed:        true,
	}
	if !cancellable[status] {
		return false, "This plan can no longer be cancelled"
	}
	for i := range days {
		if s := days[i].Status; s == models.MealPlanDayPrepared || s == models.MealPlanDayDelivered {
			return false, "This plan has already started — meals have been served. Skip upcoming days instead."
		}
	}
	return true, ""
}

func (h *MealPlanHandler) CancelMealPlan(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	plan, ok := loadScopedPlan(c.Param("id"), "customer_id", customerID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal plan not found"})
		return
	}

	if ok, reason := mealPlanCancellableBeforeStart(plan.Status, plan.Days); !ok {
		c.JSON(http.StatusConflict, gin.H{"error": reason})
		return
	}

	okLock, release := idempotencyGuard(c.Request.Context(), fmt.Sprintf("cancelplan:%s", plan.ID))
	if !okLock {
		c.JSON(http.StatusConflict, gin.H{"error": "Already processing"})
		return
	}
	defer release()

	var chefProfile models.ChefProfile
	database.DB.First(&chefProfile, "id = ?", plan.ChefID)
	chefUserID := chefProfile.UserID

	now := time.Now()
	fromStatuses := []models.MealPlanStatus{
		models.MealPlanPendingChef, models.MealPlanChefAcceptedFull,
		models.MealPlanChefModified, models.MealPlanAwaitingCustomer, models.MealPlanConfirmed,
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Status-guarded: lose to a concurrent expiry/respond/skip that already moved it.
		res := tx.Model(&models.MealPlan{}).
			Where("id = ? AND status IN ?", plan.ID, fromStatuses).
			Update("status", models.MealPlanCancelled)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errPlanConflict
		}
		plan.Status = models.MealPlanCancelled
		plan.CancelledAt = &now
		plan.CancelReason = "cancelled by customer before start"
		if err := tx.Model(&models.MealPlan{}).Where("id = ?", plan.ID).
			Updates(map[string]any{"cancelled_at": now, "cancel_reason": plan.CancelReason}).Error; err != nil {
			return err
		}
		// Escrow (gated): full refund — nothing was served. No-op when escrow is
		// off (unpaid handshake) or no advance was collected.
		if err := services.RefundUndeliveredDays(tx, &plan, "cancelled by customer before start"); err != nil {
			return err
		}
		// Notify the chef the request/plan was cancelled.
		return services.EnqueueEvent(tx, services.SubjectMealPlanCancelled, "meal_plan.cancelled_by_customer", chefUserID, map[string]any{
			"meal_plan_id": plan.ID.String(), "meal_plan_no": plan.MealPlanNumber,
			"customer_id": customerID.String(), "chef_id": plan.ChefID.String(),
		})
	}); err != nil {
		if errors.Is(err, errPlanConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "This plan can no longer be cancelled"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel meal plan"})
		return
	}
	plan.ProjectForCustomer()
	c.JSON(http.StatusOK, gin.H{"mealPlan": plan})
}

// SkipMealPlanDay — PUT /meal-plans/:id/days/:dayId/skip. The customer REQUESTS to skip
// a confirmed day whose order hasn't been generated yet, at least 12h before that day's
// cooking start (approximated from the chef's schedule). This no longer auto-credits the
// customer (#422 policy change): it flips the day to `skip_req`, freezes the chef's payout
// hold to `disputed`, and notifies the chef the skip is pending. An admin then approves
// (partial refund — food minus the platform fee) or rejects it (day returns to confirmed).
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
	// Only a confirmed day whose order hasn't been generated yet.
	if day.Status != models.MealPlanDayConfirmed || day.OrderID != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "This day can no longer be skipped"})
		return
	}
	// 12h-before-START gate (#422): a skip must land at least the lead time before that
	// day's cooking begins. There is no exact per-slot start, so approximate it from the
	// chef's weekly schedule OpenTime (a per-slot default when there's no usable row).
	var schedules []models.ChefSchedule
	database.DB.Where("chef_id = ?", plan.ChefID).Find(&schedules)
	start := mealPlanDayStartIST(schedules, day)
	if !time.Now().Before(start.Add(-mealPlanLeadTime)) {
		c.JSON(http.StatusConflict, gin.H{"error": "Too late to skip this day — the kitchen is about to start cooking it"})
		return
	}

	okLock, release := idempotencyGuard(c.Request.Context(), fmt.Sprintf("skipday:%s", day.ID))
	if !okLock {
		c.JSON(http.StatusConflict, gin.H{"error": "Already processing"})
		return
	}
	defer release()

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Confirmed → skip_req: RAISE the request (no auto-credit). Guarded so a race
		// can't re-request or transition past a concurrent change (#422 policy). The
		// `order_id IS NULL` re-check closes the (already time-gate-disjoint) window where
		// the fulfilment cron locks + generates this day's order between the handler's plan
		// load and this tx — a generated day must never become skippable.
		res := tx.Model(&models.MealPlanDay{}).
			Where("id = ? AND status = ? AND order_id IS NULL", day.ID, models.MealPlanDayConfirmed).
			Update("status", models.MealPlanDaySkipRequested)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errPlanConflict
		}
		day.Status = models.MealPlanDaySkipRequested
		// Freeze the chef's payout so it can't be released while the skip is under review.
		if err := services.SetMealPlanDayHoldDisputed(tx, day.ID); err != nil {
			return err
		}
		// Tell the chef a skip is PENDING review (hold off cooking it). No RefundDay here —
		// money moves only if an admin approves. Load the chef through a struct (not a bare
		// Scan into uuid.UUID) so GORM handles the uuid column on Postgres and sqlite alike.
		var chef models.ChefProfile
		if err := tx.Select("id", "user_id").First(&chef, "id = ?", plan.ChefID).Error; err == nil && chef.UserID != uuid.Nil {
			return services.EnqueueEvent(tx, services.SubjectMealPlanDaySkipRequested, "meal_plan_day.skip_requested", chef.UserID, map[string]any{
				"meal_plan_id": plan.ID.String(), "day_id": day.ID.String(),
				"date": day.Date.Format("2006-01-02"),
			})
		}
		return nil
	}); err != nil {
		if errors.Is(err, errPlanConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "This day can no longer be skipped"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to request skip"})
		return
	}
	plan.ProjectForCustomer()
	c.JSON(http.StatusOK, gin.H{
		"status":   "skip_requested",
		"message":  "Skip requested — pending review",
		"mealPlan": plan,
	})
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
		// Validate the captured advance (order + amount + Checkout signature) and, on
		// the single awaiting_customer → confirmed transition, confirm the plan + hold
		// the chef payouts. This is the SAME durable seam the payment.captured webhook
		// uses (services.ConfirmMealPlanAdvance), so a client verify and a webhook
		// confirm can never diverge, and either one alone is sufficient. Idempotent +
		// status-guarded; escrow-off is a no-op ack.
		_, err := services.ConfirmMealPlanAdvance(tx, &plan, req.RazorpayPaymentID, req.RazorpaySignature)
		return err
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment verification failed"})
		return
	}
	plan.ProjectForCustomer()
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
	// Payment now happens AFTER the customer approves the chef's response (not at
	// create), so a pending request is legitimately unpaid — the chef reviews and
	// responds first. Do NOT gate on escrow_payment_id here or the chef would never
	// see a request to act on.
	q := database.DB.Where("chef_id = ? AND status = ?", chef.ID, status)
	q.Preload("Days").Preload("Customer").Order("created_at DESC").Find(&plans)
	for i := range plans {
		plans[i].ProjectForChef()
	}
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
	// Payment-after-approval: whether the chef accepts every day or trims some, the
	// plan ALWAYS goes to the customer for approval + payment — it is NOT
	// auto-confirmed, and NO payout is held here. The advance is created + collected
	// only after the customer approves (ApproveMealPlan → advance order → verify →
	// HoldChefPayouts). AcceptedFull vs Modified only changes the customer copy.
	subj := services.SubjectMealPlanAcceptedFull
	if !allAccepted {
		subj = services.SubjectMealPlanModified
	}
	plan.Status = models.MealPlanAwaitingCustomer
	approveBy := now.Add(custApproveWindow)
	plan.CustomerApproveBy = &approveBy

	// Estimate basis = the accepted days only (declined days are excluded). The
	// final charge is recomputed with GST + delivery at approval.
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
		if err := tx.Model(&models.MealPlan{}).Where("id = ?", plan.ID).Updates(map[string]any{
			"subtotal": acceptedTotal, "total": acceptedTotal, "customer_approve_by": plan.CustomerApproveBy,
		}).Error; err != nil {
			return err
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
	plan.ProjectForChef()
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
	for i := range plans {
		plans[i].ProjectForAdmin()
	}

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
	plan.ProjectForAdmin()
	c.JSON(http.StatusOK, gin.H{"mealPlan": plan})
}

// AdminResolveDayDeliveryFailure executes the admin-confirmed money policy for a
// delivery-FAILED meal-plan day (#393 slice B). The failed day is frozen `disputed` by
// the slice-A freeze; the admin confirms a concrete fault (customer / platform / chef)
// and the matching outcome runs — customer-fault → chef paid + day terminalized;
// platform/chef-fault → full day refund + day → refunded — then the plan completes if
// every day is terminal. Mirrors the gateway-order resolver (AdminResolveDeliveryFailure)
// for the meal-plan-day aggregate.
func (h *MealPlanHandler) AdminResolveDayDeliveryFailure(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	dayID, err := uuid.Parse(c.Param("dayId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day id"})
		return
	}
	var req struct {
		Fault models.DeliveryFaultClass `json:"fault" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A fault class (customer, platform, or chef) is required"})
		return
	}

	var day models.MealPlanDay
	if err := database.DB.First(&day, "id = ?", dayID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal-plan day not found"})
		return
	}
	// Load the plan WITH its days + snapshotted totals — RefundDay derives the per-day
	// gross refund from the plan's Subtotal/Tax/Total and day count.
	var plan models.MealPlan
	if err := database.DB.Preload("Days").First(&plan, "id = ?", day.MealPlanID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal plan not found"})
		return
	}

	switch err := services.ResolveMealPlanDayFailure(database.DB, &plan, &day, req.Fault, adminID); {
	case errors.Is(err, services.ErrAmbiguousFault):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Confirm a concrete fault: customer, platform, or chef"})
	case errors.Is(err, services.ErrNotDeliveryFailure):
		c.JSON(http.StatusBadRequest, gin.H{"error": "This day is not in a delivery-failed state"})
	case errors.Is(err, services.ErrIssueAlreadyHandled):
		c.JSON(http.StatusConflict, gin.H{"error": "This day has already been resolved"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not resolve the delivery failure"})
	default:
		c.JSON(http.StatusOK, gin.H{"status": "resolved", "fault": string(req.Fault)})
	}
}

// AdminApproveMealPlanDaySkip — POST /admin/meal-plan-days/:dayId/approve-skip. Approves
// a customer's pending skip (#422): the day becomes terminally `skipped`, the customer is
// refunded the FOOD price minus the platform commission (they forfeit GST + delivery +
// commission), the chef's held transfer is fully reversed, and the plan completes if every
// day is now terminal. Mirrors AdminResolveDayDeliveryFailure.
func (h *MealPlanHandler) AdminApproveMealPlanDaySkip(c *gin.Context) {
	h.resolveDaySkip(c, true)
}

// AdminRejectMealPlanDaySkip — POST /admin/meal-plan-days/:dayId/reject-skip. Rejects a
// pending skip: the day returns to `confirmed`, its frozen hold is restored, and the
// customer is told the skip was declined. No money moves.
func (h *MealPlanHandler) AdminRejectMealPlanDaySkip(c *gin.Context) {
	h.resolveDaySkip(c, false)
}

// resolveDaySkip is the shared body for the approve/reject admin skip endpoints: it loads
// the day + its plan (with snapshotted totals for the refund basis) and runs
// ResolveMealPlanDaySkip in a transaction.
func (h *MealPlanHandler) resolveDaySkip(c *gin.Context, approve bool) {
	adminID, _ := middleware.GetUserID(c)
	dayID, err := uuid.Parse(c.Param("dayId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day id"})
		return
	}
	var day models.MealPlanDay
	if err := database.DB.First(&day, "id = ?", dayID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal-plan day not found"})
		return
	}
	// Load the plan WITH its days + snapshotted totals — perDaySkipRefund derives from the
	// day's price and the plan is needed for completion re-check.
	var plan models.MealPlan
	if err := database.DB.Preload("Days").First(&plan, "id = ?", day.MealPlanID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal plan not found"})
		return
	}

	switch err := services.ResolveMealPlanDaySkip(database.DB, &plan, &day, approve, adminID); {
	case errors.Is(err, services.ErrNotSkipRequest):
		c.JSON(http.StatusConflict, gin.H{"error": "This day has no pending skip request"})
	case errors.Is(err, services.ErrIssueAlreadyHandled):
		c.JSON(http.StatusConflict, gin.H{"error": "This skip request has already been resolved"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not resolve the skip request"})
	default:
		decision := "approved"
		if !approve {
			decision = "rejected"
		}
		c.JSON(http.StatusOK, gin.H{"status": "resolved", "decision": decision})
	}
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
