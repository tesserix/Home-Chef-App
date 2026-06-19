package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// meal_plan_prep.go — the chef's bulk subscription prep view (#50). Rolls up the
// confirmed meal-plan days for a date into a per-dish manifest ("tomorrow you owe
// N veg lunches, M nonveg dinners") + a per-customer packing list, and lets the
// chef mark dishes/days prepared. The prepared signal is what the customer sees
// live (with a cooking animation).

// prepStatuses are the day statuses that count as "owed" for a date — locked in
// (confirmed) or already marked cooking (prepared). Delivered days are done.
var prepStatuses = []string{string(models.MealPlanDayConfirmed), string(models.MealPlanDayPrepared)}

// prepRow is one flattened meal-plan day joined with its plan + customer, the raw
// material for both the manifest rollup and the packing list.
type prepRow struct {
	DayID         uuid.UUID `json:"dayId"`
	Slot          string    `json:"slot"`
	Variant       string    `json:"variant"`
	DishName      string    `json:"dishName"`
	Status        string    `json:"status"`
	PlanNumber    string    `json:"planNumber"`
	CustomerFirst string    `json:"-"`
	CustomerLast  string    `json:"-"`
	CustomerName  string    `json:"customerName"`
}

// prepManifestLine is one aggregated dish for a (slot, variant).
type prepManifestLine struct {
	Slot     string `json:"slot"`
	Variant  string `json:"variant"`
	DishName string `json:"dishName"`
	Total    int    `json:"total"`
	Prepared int    `json:"prepared"`
}

// prepTotals are the headline counts the chef sees at a glance.
type prepTotals struct {
	Lunch    int `json:"lunch"`
	Dinner   int `json:"dinner"`
	Total    int `json:"total"`
	Prepared int `json:"prepared"`
}

// buildPrepManifest rolls flattened rows into per-dish manifest lines + totals.
// Pure (no DB) so it's unit-tested. Lines are keyed by slot|variant|dish; totals
// split by slot and track how many are already prepared.
func buildPrepManifest(rows []prepRow) ([]prepManifestLine, prepTotals) {
	type key struct{ slot, variant, dish string }
	idx := map[key]*prepManifestLine{}
	order := []key{}
	var totals prepTotals

	for _, r := range rows {
		k := key{r.Slot, r.Variant, r.DishName}
		line, ok := idx[k]
		if !ok {
			line = &prepManifestLine{Slot: r.Slot, Variant: r.Variant, DishName: r.DishName}
			idx[k] = line
			order = append(order, k)
		}
		line.Total++
		isPrepared := r.Status == string(models.MealPlanDayPrepared)
		if isPrepared {
			line.Prepared++
			totals.Prepared++
		}
		totals.Total++
		switch r.Slot {
		case string(models.MealSlotLunch):
			totals.Lunch++
		case string(models.MealSlotDinner):
			totals.Dinner++
		}
	}

	lines := make([]prepManifestLine, 0, len(order))
	for _, k := range order {
		lines = append(lines, *idx[k])
	}
	return lines, totals
}

// startOfDayIST returns IST midnight for an instant.
func startOfDayIST(t time.Time) time.Time {
	ist := t.In(istLoc)
	return time.Date(ist.Year(), ist.Month(), ist.Day(), 0, 0, 0, 0, istLoc)
}

// GetPrepManifest — GET /chef/meal-plans/prep?date=YYYY-MM-DD (default tomorrow IST).
func (h *MealPlanHandler) GetPrepManifest(c *gin.Context) {
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
	dayEnd := day.Add(24 * time.Hour)

	rows := queryPrepRows(chef.ID, day, dayEnd)
	manifest, totals := buildPrepManifest(rows)

	c.JSON(http.StatusOK, gin.H{
		"date":        day.In(istLoc).Format("2006-01-02"),
		"manifest":    manifest,
		"packingList": rows,
		"totals":      totals,
	})
}

// queryPrepRows loads the chef's owed meal-plan days for [day, dayEnd) joined with
// the plan + customer name.
func queryPrepRows(chefID uuid.UUID, day, dayEnd time.Time) []prepRow {
	rows := []prepRow{}
	database.DB.Table("meal_plan_days").
		Select(`meal_plan_days.id AS day_id, meal_plan_days.slot, meal_plan_days.variant,
			meal_plan_days.dish_name, meal_plan_days.status,
			meal_plans.meal_plan_number AS plan_number,
			users.first_name AS customer_first, users.last_name AS customer_last`).
		Joins("JOIN meal_plans ON meal_plans.id = meal_plan_days.meal_plan_id").
		Joins("LEFT JOIN users ON users.id = meal_plans.customer_id").
		Where("meal_plans.chef_id = ? AND meal_plan_days.date >= ? AND meal_plan_days.date < ? AND meal_plan_days.status IN ?",
			chefID, day, dayEnd, prepStatuses).
		Order("meal_plan_days.slot, meal_plan_days.variant, meal_plan_days.dish_name").
		Scan(&rows)
	for i := range rows {
		name := rows[i].CustomerFirst
		if rows[i].CustomerLast != "" {
			name += " " + rows[i].CustomerLast
		}
		rows[i].CustomerName = name
	}
	return rows
}

// markDaysPrepared transitions the given confirmed days (owned by the chef) to
// prepared, stamps PreparedAt, and notifies each customer. Returns the count
// actually transitioned. Idempotent: already-prepared days are skipped.
func markDaysPrepared(chefID uuid.UUID, where *gorm.DB) (int, error) {
	var days []models.MealPlanDay
	// Only the chef's own confirmed days are eligible.
	q := database.DB.
		Joins("JOIN meal_plans ON meal_plans.id = meal_plan_days.meal_plan_id").
		Where("meal_plans.chef_id = ? AND meal_plan_days.status = ?", chefID, models.MealPlanDayConfirmed)
	if where != nil {
		q = q.Where(where)
	}
	if err := q.Find(&days).Error; err != nil {
		return 0, err
	}
	if len(days) == 0 {
		return 0, nil
	}

	now := time.Now()
	count := 0
	for i := range days {
		d := days[i]
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// Status-guarded update so a concurrent transition can't double-fire.
			res := tx.Model(&models.MealPlanDay{}).
				Where("id = ? AND status = ?", d.ID, models.MealPlanDayConfirmed).
				Updates(map[string]any{"status": models.MealPlanDayPrepared, "prepared_at": now})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return nil // lost the race / already prepared — skip silently
			}
			count++
			// Notify the customer their dish is being cooked.
			var customerID uuid.UUID
			database.DB.Table("meal_plans").Select("customer_id").
				Where("id = ?", d.MealPlanID).Scan(&customerID)
			return services.EnqueueEvent(tx, services.SubjectMealPlanDayPrepared, "meal_plan_day.prepared",
				customerID, map[string]any{
					"mealPlanId": d.MealPlanID.String(),
					"dayId":      d.ID.String(),
					"dishName":   d.DishName,
					"slot":       string(d.Slot),
				})
		})
		if err != nil {
			return count, err
		}
	}
	return count, nil
}

// MarkDayPrepared — POST /chef/meal-plans/days/:dayId/prepare. Single day.
func (h *MealPlanHandler) MarkDayPrepared(c *gin.Context) {
	chef, ok := authedChef(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	dayID, err := uuid.Parse(c.Param("dayId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day id"})
		return
	}
	count, err := markDaysPrepared(chef.ID, database.DB.Where("meal_plan_days.id = ?", dayID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark prepared"})
		return
	}
	if count == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Day not found or not in a preparable state"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prepared": count})
}

type markPrepBulkRequest struct {
	Date     string   `json:"date"`
	Slot     string   `json:"slot"`
	Variant  string   `json:"variant"`
	DishName string   `json:"dishName"`
	DayIDs   []string `json:"dayIds"`
}

// MarkPrepBulk — POST /chef/meal-plans/prep/mark. Marks every matching confirmed
// day prepared: either an explicit dayIds list, or all days matching
// date(+slot+variant+dishName) — the "mark this whole dish prepared" action.
func (h *MealPlanHandler) MarkPrepBulk(c *gin.Context) {
	chef, ok := authedChef(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	var req markPrepBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	where := database.DB.Session(&gorm.Session{NewDB: true})
	if len(req.DayIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(req.DayIDs))
		for _, s := range req.DayIDs {
			if id, err := uuid.Parse(s); err == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no valid dayIds"})
			return
		}
		where = where.Where("meal_plan_days.id IN ?", ids)
	} else {
		if req.Date == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date or dayIds required"})
			return
		}
		day, err := parsePlanDate(req.Date)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "date must be YYYY-MM-DD"})
			return
		}
		where = where.Where("meal_plan_days.date >= ? AND meal_plan_days.date < ?", day, day.Add(24*time.Hour))
		if req.Slot != "" {
			where = where.Where("meal_plan_days.slot = ?", req.Slot)
		}
		if req.Variant != "" {
			where = where.Where("meal_plan_days.variant = ?", req.Variant)
		}
		if req.DishName != "" {
			where = where.Where("meal_plan_days.dish_name = ?", req.DishName)
		}
	}

	count, err := markDaysPrepared(chef.ID, where)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark prepared"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prepared": count})
}
