package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// Approving as Test must produce a kitchen that is in test mode and has NEVER
// been live — the born-test state, which is hidden from customers outright
// rather than shown as closed.
func TestApproveAsTestCreatesBornTestChef(t *testing.T) {
	db := setupOnboardingDB(t)
	chefID, approvalID := seedPendingOnboarding(t, db)

	require.NoError(t, ActivateChefOnboarding(db, approvalID, models.ChefModeTest))

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.True(t, chef.IsVerified, "approval must still verify the kitchen")
	require.Equal(t, models.ChefModeTest, chef.Mode)
	require.Nil(t, chef.FirstLiveAt, "a born-test kitchen has never been live")
	require.True(t, chef.IsBornTest())
}

// The default must be live. An admin who does not choose gets a real kitchen.
func TestApproveDefaultsToLive(t *testing.T) {
	db := setupOnboardingDB(t)
	chefID, approvalID := seedPendingOnboarding(t, db)

	require.NoError(t, ActivateChefOnboarding(db, approvalID, ""))

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.Equal(t, models.ChefModeLive, chef.Mode)
	require.NotNil(t, chef.FirstLiveAt, "going live stamps FirstLiveAt")
}

// The durable Temporal activity passes an empty mode and reads the admin's
// choice back off the approval row, so a crash mid-activation retries into the
// SAME world rather than silently defaulting to live.
func TestActivationReadsModeBackFromTheApproval(t *testing.T) {
	db := setupOnboardingDB(t)
	chefID, approvalID := seedPendingOnboarding(t, db)
	require.NoError(t, db.Exec(`UPDATE approval_requests SET approved_mode = ? WHERE id = ?`,
		models.ChefModeTest, approvalID.String()).Error)

	// Empty mode = "whatever the admin chose", the activity's calling convention.
	require.NoError(t, ActivateChefOnboarding(db, approvalID, ""))

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.Equal(t, models.ChefModeTest, chef.Mode, "a retried activation must land in the admin's chosen world")
	require.Nil(t, chef.FirstLiveAt)
}

// Activation is idempotent (it runs both inline and as a retried activity), so
// running it twice must not move FirstLiveAt or change the mode.
func TestActivationIsIdempotent(t *testing.T) {
	db := setupOnboardingDB(t)
	chefID, approvalID := seedPendingOnboarding(t, db)

	require.NoError(t, ActivateChefOnboarding(db, approvalID, models.ChefModeLive))
	var first models.ChefProfile
	require.NoError(t, db.First(&first, "id = ?", chefID).Error)
	require.NotNil(t, first.FirstLiveAt)
	stamped := *first.FirstLiveAt

	require.NoError(t, ActivateChefOnboarding(db, approvalID, models.ChefModeLive))
	var second models.ChefProfile
	require.NoError(t, db.First(&second, "id = ?", chefID).Error)
	require.True(t, second.FirstLiveAt.Equal(stamped), "FirstLiveAt must not move on re-activation")
	require.Equal(t, models.ChefModeLive, second.Mode)
}

func seedPendingOnboarding(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	userID, chefID, approvalID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, role) VALUES (?,?)`,
		userID.String(), "customer").Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id) VALUES (?,?)`,
		chefID.String(), userID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO approval_requests (id, type, chef_id, status) VALUES (?,?,?,?)`,
		approvalID.String(), string(models.ApprovalKitchenOnboarding), chefID.String(), "pending").Error)
	return chefID, approvalID
}
