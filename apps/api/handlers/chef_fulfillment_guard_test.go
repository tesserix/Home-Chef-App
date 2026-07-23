package handlers

// chef_fulfillment_guard_test.go — a chef offering neither pickup nor
// self-delivery is unorderable: the customer chef-discovery gate only shows
// chefs that offer pickup or can self-deliver in range, so a chef with both
// offers false is active/discoverable but produces empty/partial results
// wherever it surfaces. A prod data bug found active chefs in exactly this
// state; these tests pin the guard added to close it:
//
//   - ChefHandler.UpdateChefProfile refuses to save a chef that would end up
//     active with both OffersPickup and OffersSelfDelivery false.
//   - AdminHandler.VerifyChef refuses to activate a chef offering neither.
//
// SQLite harness mirrors chef_orders_visibility_test.go / internal_users_test.go
// (reuses the shared `setupDB` users-table helper): in-memory DB, hand-DDL'd
// `chef_profiles` table with the exact column set gorm's Save() writes for
// models.ChefProfile (a Postgres-only default:gen_random_uuid() means we
// can't AutoMigrate it), swap database.DB, restore in t.Cleanup.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
)

// chefProfilesGuardDDL lists every column gorm's tx.Save(&chef) writes for
// models.ChefProfile (verified via schema.Parse against the live struct) so
// the full-record UPDATE the handler issues never hits "no such column".
const chefProfilesGuardDDL = `CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', 
	id text PRIMARY KEY, user_id text, business_name text DEFAULT '',
	slug text DEFAULT '', description text DEFAULT '',
	profile_image text DEFAULT '', banner_image text DEFAULT '',
	cuisines text DEFAULT '{}', specialties text DEFAULT '{}',
	prep_time text DEFAULT '', minimum_order real DEFAULT 0,
	delivery_radius real DEFAULT 10, service_radius real DEFAULT 10,
	offers_pickup integer DEFAULT 0, offers_self_delivery integer DEFAULT 0,
	self_delivery_base_fee real DEFAULT 0, self_delivery_free_radius_km real DEFAULT 0,
	self_delivery_per_km real DEFAULT 0, self_delivery_max_fee real DEFAULT 0,
	self_delivery_max_distance_km real DEFAULT 0,
	rating real DEFAULT 0, total_reviews integer DEFAULT 0, total_orders integer DEFAULT 0,
	issue_count integer DEFAULT 0,
	is_verified integer DEFAULT 0, verified_at datetime,
	is_active integer DEFAULT 1, accepting_orders integer DEFAULT 1, paused_until datetime,
	auto_schedule_enabled integer DEFAULT 0,
	kitchen_photos text DEFAULT '{}', kitchen_type text DEFAULT 'home_kitchen',
	address_line1 text DEFAULT '', address_line2 text DEFAULT '',
	city text DEFAULT '', state text DEFAULT '', postal_code text DEFAULT '',
	latitude real DEFAULT 0, longitude real DEFAULT 0,
	is_featured integer DEFAULT 0, featured_until datetime,
	stripe_account_id text DEFAULT '', razorpay_account_id text DEFAULT '',
	payment_provider text DEFAULT 'razorpay', payout_country text DEFAULT 'IN',
	stripe_charges_enabled integer DEFAULT 0, stripe_payouts_enabled integer DEFAULT 0,
	razorpay_product_id text DEFAULT '', razorpay_settlement_status text DEFAULT '',
	razorpay_settlement_requirements text DEFAULT '', razorpay_stakeholder_created integer DEFAULT 0,
	payout_auto_release text DEFAULT '',
	payout_method text DEFAULT '', bank_account_number text DEFAULT '',
	bank_ifsc text DEFAULT '', bank_account_name text DEFAULT '', upi_id text DEFAULT '',
	pan_number text DEFAULT '', fssai_license_number text DEFAULT '', gstin text DEFAULT '',
	fssai_override_until datetime, fssai_override_reason text DEFAULT '', fssai_override_by text,
	created_at datetime, updated_at datetime)`

const chefGuardAuditDDL = `CREATE TABLE audit_logs (
	id text DEFAULT '00000000-0000-0000-0000-000000000000',
	user_id text, action text, entity_type text, entity_id text,
	old_value text, new_value text, ip_address text, user_agent text,
	correlation_id text, created_at datetime)`

// setupChefGuardDB wires users + chef_profiles + audit_logs and points
// database.DB at it for the duration of the test.
func setupChefGuardDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupDB(t) // users table, shared with internal_users_test.go
	require.NoError(t, db.Exec(chefProfilesGuardDDL).Error)
	require.NoError(t, db.Exec(chefGuardAuditDDL).Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

// seedGuardChef inserts a user + chef_profiles row with the given offer/active
// state and returns (userID, chefID).
func seedGuardChef(t *testing.T, db *gorm.DB, offersPickup, offersSelfDelivery, isActive bool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	userID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, email, role) VALUES (?, ?, 'chef')`,
		userID.String(), userID.String()[:8]+"@chef.test").Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles
		(id, user_id, business_name, offers_pickup, offers_self_delivery, is_active, is_verified)
		VALUES (?, ?, ?, ?, ?, ?, 0)`,
		chefID.String(), userID.String(), "Guard Kitchen "+chefID.String()[:8],
		offersPickup, offersSelfDelivery, isActive).Error)
	return userID, chefID
}

func chefGuardRouter(userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.PUT("/chef/profile", (&ChefHandler{}).UpdateChefProfile)
	return r
}

func putChefProfile(t *testing.T, r *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/chef/profile", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func chefIsActive(t *testing.T, db *gorm.DB, chefID uuid.UUID) bool {
	t.Helper()
	var active bool
	require.NoError(t, db.Raw(`SELECT is_active FROM chef_profiles WHERE id = ?`, chefID.String()).Scan(&active).Error)
	return active
}

// TestUpdateChefProfile_RejectsActiveChefWithNoFulfillmentMethod — an active
// chef flipping both offers to false must be rejected with 400, and the
// row must stay unchanged (both offers still whatever they were before —
// here: at least one true, proving the save never happened).
func TestUpdateChefProfile_RejectsActiveChefWithNoFulfillmentMethod(t *testing.T) {
	db := setupChefGuardDB(t)
	userID, chefID := seedGuardChef(t, db, true, false, true) // active, pickup-only

	r := chefGuardRouter(userID)
	w := putChefProfile(t, r, map[string]any{
		"offersPickup":       false,
		"offersSelfDelivery": false,
	})

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotEmpty(t, body["error"])
	require.NotContains(t, body["error"], "500", "must be a human-readable message, not a raw code")

	var row struct {
		OffersPickup bool
	}
	require.NoError(t, db.Raw(`SELECT offers_pickup FROM chef_profiles WHERE id = ?`, chefID.String()).Scan(&row).Error)
	require.True(t, row.OffersPickup, "the blocked update must never persist")
}

// TestUpdateChefProfile_AllowsWhenAtLeastOneOfferTrue — flipping pickup off
// is fine as long as self-delivery stays on (or vice versa).
func TestUpdateChefProfile_AllowsWhenAtLeastOneOfferTrue(t *testing.T) {
	db := setupChefGuardDB(t)
	userID, chefID := seedGuardChef(t, db, true, true, true) // active, both offers on

	r := chefGuardRouter(userID)
	w := putChefProfile(t, r, map[string]any{
		"offersPickup":       false,
		"offersSelfDelivery": true,
	})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var row struct {
		OffersPickup       bool
		OffersSelfDelivery bool
	}
	require.NoError(t, db.Raw(`SELECT offers_pickup, offers_self_delivery FROM chef_profiles WHERE id = ?`, chefID.String()).Scan(&row).Error)
	require.False(t, row.OffersPickup)
	require.True(t, row.OffersSelfDelivery)
}

// TestUpdateChefProfile_AllowsNoFulfillmentWhenInactive — a chef that is
// already inactive (e.g. suspended, or paused pending onboarding) may
// legitimately have no fulfillment method; the guard only bites chefs that
// would end up active. IsActive isn't part of this request — only the admin
// endpoints flip it — so the guard here must read the chef's existing state.
func TestUpdateChefProfile_AllowsNoFulfillmentWhenInactive(t *testing.T) {
	db := setupChefGuardDB(t)
	userID, chefID := seedGuardChef(t, db, false, false, false) // already inactive, no offers

	r := chefGuardRouter(userID)
	w := putChefProfile(t, r, map[string]any{
		"offersPickup":       false,
		"offersSelfDelivery": false,
	})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.False(t, chefIsActive(t, db, chefID))
}

// ── VerifyChef (admin) ───────────────────────────────────────────────────────

func chefGuardAdminRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/admin/chefs/:id/verify", (&AdminHandler{}).VerifyChef)
	return r
}

func postVerifyChef(t *testing.T, r *gin.Engine, chefID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/admin/chefs/"+chefID.String()+"/verify", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestVerifyChef_RefusesToActivateChefWithNoFulfillmentMethod — an
// unverified chef offering neither pickup nor self-delivery must not be
// verified/activated by the admin endpoint.
func TestVerifyChef_RefusesToActivateChefWithNoFulfillmentMethod(t *testing.T) {
	db := setupChefGuardDB(t)
	_, chefID := seedGuardChef(t, db, false, false, false)

	r := chefGuardAdminRouter()
	w := postVerifyChef(t, r, chefID)

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var row struct {
		IsVerified bool
		IsActive   bool
	}
	require.NoError(t, db.Raw(`SELECT is_verified, is_active FROM chef_profiles WHERE id = ?`, chefID.String()).Scan(&row).Error)
	require.False(t, row.IsVerified, "refused verification must not persist")
	require.False(t, row.IsActive)
}

// TestVerifyChef_SucceedsWhenChefOffersAFulfillmentMethod — the happy path
// stays intact: a chef offering at least one method verifies + activates
// normally, and the linked user's role flips to chef.
func TestVerifyChef_SucceedsWhenChefOffersAFulfillmentMethod(t *testing.T) {
	db := setupChefGuardDB(t)
	userID, chefID := seedGuardChef(t, db, true, false, false)

	r := chefGuardAdminRouter()
	w := postVerifyChef(t, r, chefID)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var row struct {
		IsVerified bool
		IsActive   bool
	}
	require.NoError(t, db.Raw(`SELECT is_verified, is_active FROM chef_profiles WHERE id = ?`, chefID.String()).Scan(&row).Error)
	require.True(t, row.IsVerified)
	require.True(t, row.IsActive)

	var role string
	require.NoError(t, db.Raw(`SELECT role FROM users WHERE id = ?`, userID.String()).Scan(&role).Error)
	require.Equal(t, "chef", role)
}
