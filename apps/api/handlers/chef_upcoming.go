package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// chef_upcoming.go — what a kitchen owes in the near future, whether or not it
// has become an order yet.
//
// The vendor dashboard only ever read /chef/orders, so it could not show a plan
// day until that day's order generated — which happens just 12h before service
// (mealPlanLockLead). A tiffin plan booked for the week was therefore invisible
// on the dashboard until the morning of each meal, and a chef checking "what's
// coming?" saw nothing. This endpoint answers from the plan days themselves, so
// a committed meal shows up the moment the plan is confirmed.

// upcomingDefaultHours is the dashboard window. A day is "upcoming" when its
// cook-start (MealPlanDayStartIST — the same anchor the order-lock and skip gate
// use) falls inside it, so this view can never disagree with when the kitchen is
// actually expected to cook.
const upcomingDefaultHours = 24

// upcomingMaxHours bounds the query so a caller cannot ask for the whole year and
// turn a dashboard widget into a table scan.
const upcomingMaxHours = 24 * 14

// upcomingMeal is one meal the chef owes inside the window.
type upcomingMeal struct {
	DayID uuid.UUID `json:"dayId"`
	// PlanID is what the client needs to open the plan; PlanNumber is only for
	// display. Without it the dashboard row had nothing to navigate to.
	PlanID     uuid.UUID `json:"planId"`
	PlanNumber string    `json:"planNumber"`
	Date       time.Time `json:"date"`
	// StartsAt is the cook-start this meal is measured against, so the client can
	// say "in 3 hours" without re-deriving the schedule rules.
	StartsAt     time.Time `json:"startsAt"`
	Slot         string    `json:"slot"`
	Variant      string    `json:"variant"`
	DishName     string    `json:"dishName"`
	Status       string    `json:"status"`
	CustomerName string    `json:"customerName"`
	// OrderNumber is empty until the day's order locks. Its presence is what tells
	// the chef this meal has moved from "committed" to "live in the kitchen".
	OrderNumber string `json:"orderNumber,omitempty"`
}

type upcomingResponse struct {
	Hours  int            `json:"hours"`
	Total  int            `json:"total"`
	Lunch  int            `json:"lunch"`
	Dinner int            `json:"dinner"`
	Meals  []upcomingMeal `json:"meals"`
}

// GetChefUpcoming — GET /chef/upcoming?hours=24.
//
// Returns every still-owed meal-plan day whose cook-start falls within the
// window, soonest first. Delivered/skipped/cancelled days are excluded: the
// question is what the kitchen still has to cook.
func (h *MealPlanHandler) GetChefUpcoming(c *gin.Context) {
	chef, ok := authedChef(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	hours := upcomingDefaultHours
	if raw := c.Query("hours"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			hours = v
		}
	}
	if hours > upcomingMaxHours {
		hours = upcomingMaxHours
	}

	now := time.Now()
	until := now.Add(time.Duration(hours) * time.Hour)

	// Pull a generous date range and filter precisely on cook-start below. The
	// stored Date is IST midnight, so a ±1 day margin covers a window that starts
	// late in the day without missing a dinner slot at the far edge.
	//
	// Columns are listed FLAT rather than embedding models.MealPlanDay: Scan does
	// not flatten an anonymously-embedded struct, so Date and Slot came back zero,
	// every cook-start resolved to year 1 — i.e. "already passed" — and the
	// endpoint returned 200 with an empty list while the SQL behind it matched
	// rows perfectly. A silent empty result is the worst shape this bug could take,
	// since it is indistinguishable from "nothing to cook".
	var rows []struct {
		ID            uuid.UUID
		MealPlanID    uuid.UUID
		Date          time.Time
		Slot          string
		Variant       string
		Status        string
		DishName      string
		PlanNumber    string
		CustomerFirst string
		CustomerLast  string
		OrderNumber   string
	}
	if err := database.DB.
		Table("meal_plan_days").
		Select(`meal_plan_days.*, meal_plans.meal_plan_number AS plan_number,
		        users.first_name AS customer_first, users.last_name AS customer_last,
		        COALESCE(orders.order_number, '') AS order_number`).
		Joins("JOIN meal_plans ON meal_plans.id = meal_plan_days.meal_plan_id").
		Joins("LEFT JOIN users ON users.id = meal_plans.customer_id").
		Joins("LEFT JOIN orders ON orders.id = meal_plan_days.order_id").
		Where("meal_plans.chef_id = ?", chef.ID).
		Where("meal_plans.status IN ?", []models.MealPlanStatus{
			models.MealPlanConfirmed, models.MealPlanActive,
		}).
		Where("meal_plan_days.status IN ?", prepStatuses).
		Where("meal_plan_days.date BETWEEN ? AND ?", now.AddDate(0, 0, -1), until.AddDate(0, 0, 1)).
		Order("meal_plan_days.date ASC").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load upcoming meals"})
		return
	}

	// One schedule read for the chef — MealPlanDayStartIST needs it per day, and
	// re-querying inside the loop would be N round-trips for a dashboard widget.
	var schedules []models.ChefSchedule
	database.DB.Where("chef_id = ?", chef.ID).Find(&schedules)

	resp := upcomingResponse{Hours: hours, Meals: []upcomingMeal{}}
	for i := range rows {
		r := &rows[i]
		day := models.MealPlanDay{Date: r.Date, Slot: models.MealSlot(r.Slot)}
		startsAt := services.MealPlanDayStartIST(schedules, &day)
		// Strictly inside the window. A meal whose cook-start has already passed is
		// the kitchen's current work, not something "upcoming".
		if startsAt.Before(now) || startsAt.After(until) {
			continue
		}
		name := r.CustomerFirst
		if r.CustomerLast != "" {
			if name != "" {
				name += " "
			}
			name += r.CustomerLast
		}
		if name == "" {
			name = "Customer"
		}
		resp.Meals = append(resp.Meals, upcomingMeal{
			DayID:        r.ID,
			PlanID:       r.MealPlanID,
			PlanNumber:   r.PlanNumber,
			Date:         r.Date,
			StartsAt:     startsAt,
			Slot:         r.Slot,
			Variant:      r.Variant,
			DishName:     r.DishName,
			Status:       r.Status,
			CustomerName: name,
			OrderNumber:  r.OrderNumber,
		})
		if models.MealSlot(r.Slot) == models.MealSlotDinner {
			resp.Dinner++
		} else {
			resp.Lunch++
		}
	}
	resp.Total = len(resp.Meals)

	// Soonest first — a dashboard is read top-down and the next thing to cook
	// matters most. The SQL orders by date, which is IST midnight and therefore
	// ties every lunch with its dinner; sorting on the resolved cook-start is what
	// actually separates them.
	for i := 1; i < len(resp.Meals); i++ {
		for j := i; j > 0 && resp.Meals[j].StartsAt.Before(resp.Meals[j-1].StartsAt); j-- {
			resp.Meals[j], resp.Meals[j-1] = resp.Meals[j-1], resp.Meals[j]
		}
	}

	c.JSON(http.StatusOK, resp)
}
