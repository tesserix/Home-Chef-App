package handlers

// meal_plan_guards_test.go — #200 (tiffin E2E). HTTP-level checks for two CreateMealPlan booking
// guards the audit flagged as call-site-untested: the FSSAI food-licence lockout (#91) and the
// per-day booking cutoff. Both reject BEFORE the GORM insert, so they run cleanly on SQLite (like
// the #406/#409 guard tests). Reuses setupBookingDB / mealPlanReq / bookDate from
// meal_plan_booking_test.go.

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// A chef in India whose verified FSSAI licence has expired is locked out of new bookings (409).
func TestCreateMealPlan_RejectsExpiredFSSAIChef(t *testing.T) {
	db, _, _ := setupBookingDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE chef_documents (id text PRIMARY KEY, chef_id text, type text,
		file_name text, file_path text, bucket text, content_type text, file_size integer, status text,
		rejection_reason text, expiry_date datetime, created_at datetime, updated_at datetime)`).Error)

	fssaiChef := uuid.New()
	// payout_country IN → FSSAI applies (the setup chef is "", so this is the only locked one).
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, payout_country, is_active) VALUES (?,?,?,?,1)`,
		fssaiChef.String(), uuid.NewString(), "Expired Licence Kitchen", "IN").Error)
	// A verified FSSAI licence that expired 2 days ago → locked from yesterday.
	require.NoError(t, db.Exec(
		`INSERT INTO chef_documents (id, chef_id, type, file_name, file_path, bucket, status, expiry_date)
		 VALUES (?,?,?,?,?,?,?,?)`,
		uuid.NewString(), fssaiChef.String(), string(models.DocFSSAILicense),
		"fssai.pdf", "path", "private", string(models.DocStatusVerified), time.Now().AddDate(0, 0, -2)).Error)

	w := mealPlanReq(t, uuid.New(), map[string]any{
		"chefId": fssaiChef.String(),
		"days":   []map[string]any{{"date": bookDate, "slot": "lunch", "variant": "veg"}},
	})
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "isn't accepting orders")
}

// A booked day earlier than the lead-time cutoff is rejected (400) — you can't book a past date.
func TestCreateMealPlan_RejectsPastCutoffDay(t *testing.T) {
	_, userID, chefID := setupBookingDB(t)
	w := mealPlanReq(t, userID, map[string]any{
		"chefId": chefID.String(),
		"days":   []map[string]any{{"date": "2024-06-01", "slot": "lunch", "variant": "veg"}},
	})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "past the booking cutoff")
}
