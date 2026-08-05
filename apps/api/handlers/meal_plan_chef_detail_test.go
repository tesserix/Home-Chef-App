package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/database"
)

// meal_plan_chef_detail_test.go — GET /chef/meal-plans/:id (#1029).
//
// The vendor detail screen resolved a plan out of the pending_chef LIST, so every
// plan the chef had already accepted opened onto "no longer pending" — a chef with
// a confirmed week could not see a single day of it. This endpoint answers for a
// plan in ANY status, and carries the settlement breakdown with it.

func seedChefPlan(t *testing.T, status string, dayStatuses []string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	db, userID, chefID := setupBookingDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE users (id text PRIMARY KEY, first_name text, last_name text, email text, phone text, deleted_at datetime)`).Error)
	customerID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, first_name, last_name, email) VALUES (?,?,?,?)`,
		customerID.String(), "Asha", "Rao", "asha@example.com").Error)

	// The booking snapshot the breakdown reads: ₹200/day food, 4% platform fee,
	// 5% food GST, ₹25/day delivery, 18% GST on both the fee and delivery.
	n := float64(len(dayStatuses))
	subtotal := 200.0 * n
	fee := subtotal * 0.04
	taxFood := subtotal * 0.05
	taxService := fee * 0.18
	delivery := 25.0 * n
	taxDelivery := delivery * 0.18
	tax := taxFood + taxService + taxDelivery

	planID := uuid.New()
	require.NoError(t, db.Exec(`ALTER TABLE meal_plans ADD COLUMN platform_fee real DEFAULT 0`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE meal_plans ADD COLUMN tax_food real DEFAULT 0`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE meal_plans ADD COLUMN tax_service real DEFAULT 0`).Error)
	require.NoError(t, db.Exec(`ALTER TABLE meal_plans ADD COLUMN tax_delivery real DEFAULT 0`).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plans
		(id, meal_plan_number, customer_id, chef_id, status, start_date, end_date,
		 subtotal, platform_fee, tax, tax_food, tax_service, tax_delivery, total, currency)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		planID.String(), "MP-2001", customerID.String(), chefID.String(), status,
		bookDate, bookDate2, subtotal, fee, tax, taxFood, taxService, taxDelivery,
		subtotal+fee+tax+delivery, "INR").Error)
	for i, ds := range dayStatuses {
		date := bookDateUTC
		if i > 0 {
			date = bookDate2UTC
		}
		require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, date, slot, variant, status, dish_name, price)
			VALUES (?,?,?,?,?,?,?,?)`,
			uuid.NewString(), planID.String(), date, "lunch", "veg", ds, "Rajma Chawal", 200.0).Error)
	}
	return userID, planID
}

func getChefPlan(t *testing.T, userID, planID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	h := &MealPlanHandler{}
	r.GET("/chef/meal-plans/:id", h.GetChefMealPlan)
	req := httptest.NewRequest(http.MethodGet, "/chef/meal-plans/"+planID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type chefPlanResponse struct {
	Data struct {
		ID       string `json:"id"`
		Status   string `json:"status"`
		Total    float64
		Days     []map[string]any `json:"days"`
		Customer *struct {
			FirstName string `json:"firstName"`
			Email     string `json:"email"`
		} `json:"customer"`
	} `json:"data"`
	Earnings struct {
		PayableDays         int     `json:"payableDays"`
		ExcludedDays        int     `json:"excludedDays"`
		FoodSubtotal        float64 `json:"foodSubtotal"`
		Gross               float64 `json:"gross"`
		PlatformCommission  float64 `json:"platformCommission"`
		TDS                 float64 `json:"tds"`
		NetPayout           float64 `json:"netPayout"`
		CustomerTotal       float64 `json:"customerTotal"`
		CustomerPlatformFee float64 `json:"customerPlatformFee"`
		CustomerDelivery    float64 `json:"customerDelivery"`
		RefundedToCustomer  float64 `json:"refundedToCustomer"`
		Days                []struct {
			NetPayout float64 `json:"netPayout"`
		} `json:"days"`
	} `json:"earnings"`
}

// The regression: a CONFIRMED plan must load, not 404 the way the pending-only
// list did.
func TestGetChefMealPlan_ReturnsConfirmedPlanWithBreakdown(t *testing.T) {
	userID, planID := seedChefPlan(t, "confirmed", []string{"confirmed", "confirmed"})

	w := getChefPlan(t, userID, planID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var out chefPlanResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Equal(t, planID.String(), out.Data.ID)
	require.Equal(t, "confirmed", out.Data.Status)
	require.Len(t, out.Data.Days, 2)

	require.Equal(t, 2, out.Earnings.PayableDays)
	require.Equal(t, 400.0, out.Earnings.FoodSubtotal)
	require.Greater(t, out.Earnings.NetPayout, 0.0)
	// Delivery and the platform fee are the platform's, never the chef's.
	require.Less(t, out.Earnings.Gross, out.Earnings.CustomerTotal)
	require.InDelta(t, 50.0, out.Earnings.CustomerDelivery, 0.01)
	require.InDelta(t, 16.0, out.Earnings.CustomerPlatformFee, 0.01)
	// The headline must be the sum of the rows shown beneath it.
	var sum float64
	for _, d := range out.Earnings.Days {
		sum += d.NetPayout
	}
	require.InDelta(t, out.Earnings.NetPayout, sum, 0.001)
	require.InDelta(t, out.Earnings.Gross-out.Earnings.PlatformCommission-out.Earnings.TDS,
		out.Earnings.NetPayout, 0.011)
}

// A day the chef declined pays them nothing and refunds the customer.
func TestGetChefMealPlan_ExcludesDeclinedDays(t *testing.T) {
	userID, planID := seedChefPlan(t, "active", []string{"delivered", "declined"})

	w := getChefPlan(t, userID, planID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var out chefPlanResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Equal(t, 1, out.Earnings.PayableDays)
	require.Equal(t, 1, out.Earnings.ExcludedDays)
	require.Equal(t, 200.0, out.Earnings.FoodSubtotal)
	require.Greater(t, out.Earnings.RefundedToCustomer, 0.0)
}

// A chef must never receive the customer's contact details (audit H1/M1).
func TestGetChefMealPlan_HidesCustomerContact(t *testing.T) {
	userID, planID := seedChefPlan(t, "confirmed", []string{"confirmed"})

	w := getChefPlan(t, userID, planID)
	require.Equal(t, http.StatusOK, w.Code)

	var out chefPlanResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.NotNil(t, out.Data.Customer)
	require.Equal(t, "Asha", out.Data.Customer.FirstName)
	require.Empty(t, out.Data.Customer.Email, "the chef must not see the customer's email")
}

// Another chef's plan is not found, not merely unauthorised — the id itself must
// not confirm the plan exists.
func TestGetChefMealPlan_ScopedToOwningChef(t *testing.T) {
	_, planID := seedChefPlan(t, "confirmed", []string{"confirmed"})
	otherUserID := uuid.New()
	require.NoError(t, database.DB.Exec(`INSERT INTO chef_profiles (id, user_id, business_name, payout_country, is_active) VALUES (?,?,?,?,1)`,
		uuid.NewString(), otherUserID.String(), "Someone Else", "").Error)

	w := getChefPlan(t, otherUserID, planID)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetChefMealPlan_RejectsMalformedID(t *testing.T) {
	userID, _ := seedChefPlan(t, "confirmed", []string{"confirmed"})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	h := &MealPlanHandler{}
	r.GET("/chef/meal-plans/:id", h.GetChefMealPlan)
	req := httptest.NewRequest(http.MethodGet, "/chef/meal-plans/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}
