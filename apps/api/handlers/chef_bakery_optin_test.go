package handlers

// chef_bakery_optin_test.go — a kitchen decides to sell cakes months after it
// onboarded. That must be a plain profile edit: no re-onboarding, no loss of
// verification, and no change to how the store is listed (#1065).

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedLiveKitchen inserts a verified, active, meals-only kitchen.
func seedLiveKitchen(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	userID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, email, role) VALUES (?, ?, 'chef')`,
		userID.String(), userID.String()[:8]+"@chef.test").Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles
		(id, user_id, business_name, vertical, sells_bakery,
		 offers_pickup, offers_self_delivery, is_active, is_verified)
		VALUES (?, ?, ?, 'kitchen', 0, 1, 1, 1, 1)`,
		chefID.String(), userID.String(), "Six Month Old Kitchen").Error)
	return userID, chefID
}

type bakeryOptInRow struct {
	Vertical    string
	SellsBakery bool
	IsVerified  bool
	IsActive    bool
}

func readBakeryOptIn(t *testing.T, db *gorm.DB, chefID uuid.UUID) bakeryOptInRow {
	t.Helper()
	var row bakeryOptInRow
	require.NoError(t, db.Raw(
		`SELECT vertical, sells_bakery, is_verified, is_active FROM chef_profiles WHERE id = ?`,
		chefID.String()).Scan(&row).Error)
	return row
}

// TestUpdateChefProfile_KitchenCanStartSellingBakeryLater — the opt-in is
// available at any time, not just during onboarding.
func TestUpdateChefProfile_KitchenCanStartSellingBakeryLater(t *testing.T) {
	db := setupChefGuardDB(t)
	userID, chefID := seedLiveKitchen(t, db)

	w := putChefProfile(t, chefGuardRouter(userID), map[string]any{"sellsBakery": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	row := readBakeryOptIn(t, db, chefID)
	require.True(t, row.SellsBakery)
	require.Equal(t, "kitchen", row.Vertical, "the store is still listed as a kitchen")
	require.True(t, row.IsVerified, "opting in must not send the chef back through verification")
	require.True(t, row.IsActive)
}

// TestUpdateChefProfile_KitchenCanStopSellingBakery — and the shelf can be
// taken back down again.
func TestUpdateChefProfile_KitchenCanStopSellingBakery(t *testing.T) {
	db := setupChefGuardDB(t)
	userID, chefID := seedLiveKitchen(t, db)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET sells_bakery = 1 WHERE id = ?`, chefID.String()).Error)

	w := putChefProfile(t, chefGuardRouter(userID), map[string]any{"sellsBakery": false})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.False(t, readBakeryOptIn(t, db, chefID).SellsBakery)
}

// TestUpdateChefProfile_OtherEditsLeaveTheBakeryShelfAlone — sellsBakery is a
// pointer field, so a payload that doesn't mention it must not clear it.
func TestUpdateChefProfile_OtherEditsLeaveTheBakeryShelfAlone(t *testing.T) {
	db := setupChefGuardDB(t)
	userID, chefID := seedLiveKitchen(t, db)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET sells_bakery = 1 WHERE id = ?`, chefID.String()).Error)

	w := putChefProfile(t, chefGuardRouter(userID), map[string]any{"description": "Home food, and cakes."})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.True(t, readBakeryOptIn(t, db, chefID).SellsBakery)
}
