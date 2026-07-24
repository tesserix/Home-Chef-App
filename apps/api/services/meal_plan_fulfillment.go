package services

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// meal_plan_fulfillment.go — turns confirmed tiffin plans into the normal order
// pipeline (#197). A cron generates (LOCKS) one scheduled Order per confirmed day
// exactly mealPlanLockLead before that day's cook-start (MealPlanDayStartIST) —
// the SAME instant the customer's skip window closes (#422 unified boundary), so
// the chef only ever preps locked, un-skippable days. The existing prep/deliver
// flow then drives it, and the delivery hook (MarkMealPlanDayDelivered) marks the
// day delivered + releases its escrow payout. A plan flips active on first
// generation and completed once every day reaches a terminal state.
const mealPlanFulfillmentInterval = 15 * time.Minute

// StartMealPlanFulfillmentCron is the legacy in-process fallback (Temporal off).
func StartMealPlanFulfillmentCron(ctx context.Context) {
	go func() {
		runMealPlanFulfillment(ctx)
		t := time.NewTicker(mealPlanFulfillmentInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("meal-plan-fulfillment: shutting down on ctx cancel")
				return
			case <-t.C:
				runMealPlanFulfillment(ctx)
			}
		}
	}()
	log.Println("meal-plan-fulfillment: cron started (interval=15m)")
}

func runMealPlanFulfillment(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("meal-plan-fulfillment: panic recovered: %v", r)
		}
	}()
	generateDueDayOrders()
	sweepStuckDays()
	completeFinishedPlans()
}

// generateDueDayOrders creates an Order for each confirmed day coming due that
// doesn't yet have one, and flips the plan active.
func generateDueDayOrders() {
	now := time.Now()
	var plans []models.MealPlan
	if err := database.DB.
		Where("status IN ?", []models.MealPlanStatus{models.MealPlanConfirmed, models.MealPlanActive}).
		Preload("Days").Find(&plans).Error; err != nil {
		log.Printf("meal-plan-fulfillment: load plans failed: %v", err)
		return
	}
	for i := range plans {
		p := &plans[i]
		addr, hasAddr := defaultAddress(p.CustomerID)
		// The chef's schedule anchors each day's cook-start (MealPlanDayStartIST);
		// no rows → per-slot defaults. Loaded once per plan.
		var schedules []models.ChefSchedule
		database.DB.Where("chef_id = ?", p.ChefID).Find(&schedules)
		generated := 0
		for j := range p.Days {
			d := &p.Days[j]
			if d.OrderID != nil || d.Status != models.MealPlanDayConfirmed {
				continue
			}
			// UNIFIED skip/lock boundary (#422): generate (lock) the order exactly
			// mealPlanLockLead before the day's cook-start — the SAME instant the
			// customer's skip window closes. A day whose start is still further out
			// waits (the customer can still request a skip until then).
			if MealPlanDayStartIST(schedules, d).After(now.Add(mealPlanLockLead)) {
				continue
			}
			if !hasAddr {
				log.Printf("meal-plan-fulfillment: plan %s has no delivery address; skipping day %s", p.ID, d.ID)
				continue
			}
			if err := generateDayOrder(p, d, addr); err != nil {
				log.Printf("meal-plan-fulfillment: generate order for day %s failed: %v", d.ID, err)
				continue
			}
			generated++
		}
		if p.Status == models.MealPlanConfirmed && generated > 0 {
			database.DB.Model(&models.MealPlan{}).
				Where("id = ? AND status = ?", p.ID, models.MealPlanConfirmed).
				Update("status", models.MealPlanActive)
		}
	}
}

// stuckDaySweepGrace is how long past a day's date it may sit confirmed without a
// generated order (the address-missing case) before the sweep auto-refunds it — the
// window in which the customer can still add a default address and have the day
// generate normally.
const stuckDaySweepGrace = 24 * time.Hour

// sweepStuckDays auto-refunds confirmed meal-plan days that came due but could never
// generate an order: generateDueDayOrders silently skips days whose customer has no
// default address, so they sit confirmed forever, strand their held escrow, and block
// completeFinishedPlans (allDaysTerminal never true). Past stuckDaySweepGrace beyond
// the day's date, each such day is refunded to the customer's wallet (RefundDay —
// escrow-gated) and marked refunded so the plan can finish. #398.
//
// MONEY-SAFE ordering: the guarded conditional UPDATE (status=confirmed AND
// order_id IS NULL → refunded) is CLAIMED FIRST inside the tx. If a concurrent
// generate/deliver already gave the day an order (RowsAffected==0), we return before
// RefundDay so NO money moves — the day is being fulfilled, not voided. Only after we
// own the void does RefundDay run; if it errors (e.g. the gateway is unreachable) the
// whole tx — including the claim — rolls back, so the day returns to confirmed and is
// retried next sweep. A day is thus never refunded-and-fulfilled, nor marked refunded
// without the refund succeeding (or being a no-op because escrow is off).
func sweepStuckDays() {
	cutoff := time.Now().Add(-stuckDaySweepGrace)
	var plans []models.MealPlan
	if err := database.DB.
		Where("status IN ?", []models.MealPlanStatus{models.MealPlanConfirmed, models.MealPlanActive}).
		Preload("Days").Find(&plans).Error; err != nil {
		log.Printf("meal-plan-fulfillment: load plans for stuck-day sweep failed: %v", err)
		return
	}
	for i := range plans {
		p := &plans[i]
		for j := range p.Days {
			d := &p.Days[j]
			if d.Status != models.MealPlanDayConfirmed || d.OrderID != nil || d.RefundTxnID != nil || !d.Date.Before(cutoff) {
				continue
			}
			if err := database.DB.Transaction(func(tx *gorm.DB) error {
				// Claim the void FIRST — the concurrency gate. If a generate/deliver
				// already gave the day an order, RowsAffected==0 → return before any
				// money moves (the day is being fulfilled, not voided).
				res := tx.Model(&models.MealPlanDay{}).
					Where("id = ? AND status = ? AND order_id IS NULL", d.ID, models.MealPlanDayConfirmed).
					Update("status", models.MealPlanDayRefunded)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected == 0 {
					return nil // raced — a generate/deliver claimed the day first
				}
				// Now that we own the void, refund (gated + idempotent). On failure the
				// whole tx — including the claim above — rolls back and the day returns
				// to confirmed, retried next sweep. Never refunded without the money.
				if err := RefundDay(tx, p, d, "meal plan: no delivery address on file — day auto-refunded"); err != nil {
					return err
				}
				// Notify the customer only when money actually moved back (escrow on).
				if d.RefundTxnID != nil {
					return EnqueueEvent(tx, SubjectMealPlanDayRefunded, "meal_plan.day_refunded", p.CustomerID, map[string]any{
						"meal_plan_id": p.ID.String(), "day_id": d.ID.String(), "reason": "no delivery address on file",
					})
				}
				return nil
			}); err != nil {
				log.Printf("meal-plan-fulfillment: stuck-day sweep for day %s failed: %v", d.ID, err)
			}
		}
	}
}

// generateDayOrder creates a pending Order + item for one confirmed day, links
// it back to the day, and stages the order-created + chef-new-order events.
func generateDayOrder(p *models.MealPlan, d *models.MealPlanDay, addr models.Address) error {
	// FK constraints are disabled (DisableForeignKeyConstraintWhenMigrating), so
	// linking the weekly-menu item id is safe when there's no backing MenuItem.
	menuItemID := uuid.Nil
	if d.WeeklyMenuItemID != nil {
		menuItemID = *d.WeeklyMenuItemID
		var wmi models.WeeklyMenuItem
		if err := database.DB.First(&wmi, "id = ?", *d.WeeklyMenuItemID).Error; err == nil && wmi.MenuItemID != nil {
			menuItemID = *wmi.MenuItemID
		}
	}
	// The advance was captured at booking (escrow), so generated orders are paid.
	paymentStatus := models.PaymentPending
	if MealPlanEscrowActive() && p.EscrowPaymentID != "" {
		paymentStatus = models.PaymentCompleted
	}
	scheduled := d.Date

	// Per-day breakdown mirrors the advance: food + GST + delivery (chef is paid
	// food only; platform keeps GST + delivery). Informational on the order record
	// — escrow money conservation is handled by the plan total + per-day refunds.
	dayTax := 0.0
	dayDelivery := 0.0
	dayTotal := d.Price
	if MealPlanEscrowActive() {
		policy := GetPlatformPolicy()
		// #540: report the SAME proportional food-GST basis the chef day-transfer is withheld on
		// (perDayFoodGST), not a live-policy-rate recompute — so orders-based reporting
		// (statement / earnings / Form-16A) matches withheld-TDS exactly, no sub-rupee drift.
		dayTax = Round2(perDayFoodGST(p, d))
		dayDelivery = Round2(policy.BaseDeliveryFee)
		dayTotal = Round2(d.Price + dayTax + dayDelivery)
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		order := models.Order{
			OrderNumber:               mealPlanOrderNumber(),
			CustomerID:                p.CustomerID,
			ChefID:                    p.ChefID,
			Status:                    models.OrderStatusPending,
			PaymentStatus:             paymentStatus,
			Currency:                  p.Currency,
			Subtotal:                  d.Price,
			Tax:                       dayTax,
			DeliveryFee:               dayDelivery,
			Total:                     dayTotal,
			DeliveryAddressLine1:      addr.Line1,
			DeliveryAddressLine2:      addr.Line2,
			DeliveryAddressCity:       addr.City,
			DeliveryAddressState:      addr.State,
			DeliveryAddressPostalCode: addr.PostalCode,
			DeliveryAddressCountry:    addr.Country,
			DeliveryLatitude:          addr.Latitude,
			DeliveryLongitude:         addr.Longitude,
			EstimatedPrepTime:         30,
			ScheduledFor:              &scheduled,
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		item := models.OrderItem{
			OrderID:    order.ID,
			MenuItemID: menuItemID,
			Name:       d.DishName,
			Price:      d.Price,
			Quantity:   1,
			Subtotal:   d.Price,
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.MealPlanDay{}).Where("id = ?", d.ID).
			Update("order_id", order.ID).Error; err != nil {
			return err
		}
		ev := OrderEvent{
			OrderID: order.ID, OrderNumber: order.OrderNumber,
			CustomerID: order.CustomerID, ChefID: order.ChefID,
			Status: string(order.Status), Total: order.Total,
		}
		if err := EnqueueOrderEvent(tx, SubjectOrderCreated, ev); err != nil {
			return err
		}
		return EnqueueOrderEvent(tx, SubjectChefNewOrder, ev)
	})
}

// completeFinishedPlans marks an active plan completed once every day is terminal.
func completeFinishedPlans() {
	// Include Confirmed as well as Active: a fully address-stuck plan never flips to
	// Active (generateDueDayOrders only does that when it generates a day), so once
	// the stuck-day sweep terminates all its days it would otherwise sit Confirmed
	// forever (#398). allDaysTerminal is false for a not-yet-resolved plan, so a
	// freshly-confirmed plan with pending days never completes prematurely.
	var plans []models.MealPlan
	if err := database.DB.
		Where("status IN ?", []models.MealPlanStatus{models.MealPlanConfirmed, models.MealPlanActive}).
		Preload("Days").Find(&plans).Error; err != nil {
		return
	}
	for i := range plans {
		p := &plans[i]
		if allDaysTerminal(p) {
			// Flip status + stage the completion notification ATOMICALLY in one tx
			// (transactional outbox), so a crash between them can't leave a plan
			// Completed with the customer never told — once Completed, the sweep never
			// re-selects it (it only loads Confirmed/Active), so a lost emit is
			// permanent. The guarded UPDATE (status = the loaded status) means only the
			// goroutine that wins the flip (RowsAffected>0) emits — exactly once under
			// concurrent cron runs. If the outbox insert fails the whole tx rolls back
			// and the next tick retries the completion cleanly.
			if err := database.DB.Transaction(func(tx *gorm.DB) error {
				res := tx.Model(&models.MealPlan{}).
					Where("id = ? AND status = ?", p.ID, p.Status).
					Update("status", models.MealPlanCompleted)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected == 0 {
					return nil // a concurrent action moved it first
				}
				return EnqueueEvent(tx, SubjectMealPlanCompleted, "meal_plan.completed", p.CustomerID, map[string]any{
					"meal_plan_id": p.ID.String(), "meal_plan_no": p.MealPlanNumber,
				})
			}); err != nil {
				log.Printf("meal-plan fulfillment: complete plan %s: %v", p.ID, err)
			}
		}
	}
}

// MarkMealPlanDayDelivered is the delivery-pipeline hook: when an order linked to
// a meal-plan day is delivered, mark the day delivered, park its payout in a
// customer-confirmation hold (#387; no release), and emit the day-delivered
// event. Safe + idempotent on any order (no-op if the order isn't a meal-plan
// order or the day is already delivered).
func MarkMealPlanDayDelivered(orderID uuid.UUID) {
	var day models.MealPlanDay
	if err := database.DB.Where("order_id = ?", orderID).First(&day).Error; err != nil {
		return // not a meal-plan order
	}
	if day.Status == models.MealPlanDayDelivered || day.RefundTxnID != nil {
		return
	}
	now := time.Now()
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Exclude all terminal day states, not just delivered (#534), AND `failed`
		// (#393): a late/duplicate delivered event must not resurrect a
		// cancelled/skipped/declined/refunded day and park its hold (defeating the day
		// refund), nor flip a delivery-FAILED day (whose hold is frozen disputed) back to
		// delivered — that would orphan the disputed hold and let the plan silently
		// complete with the day's money stuck. A failed day is resolved only by the admin
		// day-resolution path. RowsAffected==0 → no-op.
		//
		// #631: ALSO exclude any day already refunded (refund_txn_id set) even if its STATUS was
		// not terminalized — RefundUndeliveredDays/RefundDeclinedDays refund via RefundDay but
		// leave the day status non-terminal (e.g. `confirmed`), so a status-only guard would
		// resurrect it here and re-park a hold on money already returned. The refund_txn_id is the
		// durable "this day was refunded" marker.
		res := tx.Model(&models.MealPlanDay{}).
			Where("id = ? AND refund_txn_id IS NULL AND status NOT IN ?", day.ID, []models.MealPlanDayStatus{
				models.MealPlanDayDelivered, models.MealPlanDayCancelled,
				models.MealPlanDaySkipped, models.MealPlanDayDeclined, models.MealPlanDayRefunded,
				models.MealPlanDayFailed,
			}).
			Updates(map[string]any{"status": models.MealPlanDayDelivered, "delivered_at": now})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		if err := SetMealPlanDayHoldAwaitingConfirmation(tx, day.ID); err != nil {
			return err
		}
		// The event's userID MUST be the CUSTOMER's users.id — handleMealPlanDayDelivered
		// saves + pushes to event.UserID. Passing day.MealPlanID (a meal_plans.id) sent
		// the notification to a non-user. Resolve the customer from the plan, mirroring
		// the day-prepared producer (handlers/meal_plan_prep.go).
		var customerID uuid.UUID
		tx.Model(&models.MealPlan{}).Select("customer_id").Where("id = ?", day.MealPlanID).Scan(&customerID)
		return EnqueueEvent(tx, SubjectMealPlanDayDelivered, "meal_plan_day.delivered", customerID, map[string]any{
			"meal_plan_id": day.MealPlanID.String(), "day_id": day.ID.String(),
		})
	})
	if err != nil {
		log.Printf("meal-plan: mark day delivered for order %s failed: %v", orderID, err)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func defaultAddress(userID uuid.UUID) (models.Address, bool) {
	var a models.Address
	if err := database.DB.Where("user_id = ?", userID).
		Order("is_default DESC, created_at ASC").First(&a).Error; err != nil {
		return models.Address{}, false
	}
	return a, true
}

func mealPlanOrderNumber() string {
	return "HCMP" + uuid.NewString()[:10]
}

// allDaysTerminal reports whether every day has reached a terminal state.
func allDaysTerminal(p *models.MealPlan) bool {
	if len(p.Days) == 0 {
		return false
	}
	for i := range p.Days {
		switch p.Days[i].Status {
		case models.MealPlanDayDelivered, models.MealPlanDaySkipped,
			models.MealPlanDayCancelled, models.MealPlanDayRefunded, models.MealPlanDayDeclined:
			// terminal
		default:
			return false
		}
	}
	return true
}
