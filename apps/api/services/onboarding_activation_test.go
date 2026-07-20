package services

// onboarding_activation_test.go — the chef-onboarding activation (#126). Pins
// that it verifies + activates the chef and promotes the user to the chef role,
// that it's idempotent (a retried durable activity converges to the same state),
// and that it no-ops for a non-onboarding approval.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func setupOnboardingDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE approval_requests (id text PRIMARY KEY, type text, chef_id text, partner_id text, status text,
			created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', id text PRIMARY KEY, user_id text, is_verified integer DEFAULT 0,
			verified_at datetime, is_active integer DEFAULT 1, created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE users (email_enc text DEFAULT '', email_bidx text DEFAULT '', first_name_enc text DEFAULT '', last_name_enc text DEFAULT '', phone_enc text DEFAULT '', phone_bidx text DEFAULT '', id text PRIMARY KEY, role text, created_at datetime, updated_at datetime, deleted_at datetime)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func TestActivateChefOnboarding_VerifiesAndPromotes(t *testing.T) {
	db := setupOnboardingDB(t)
	userID, chefID, approvalID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, role) VALUES (?, ?)`, userID.String(), "customer").Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, is_verified, is_active) VALUES (?,?,0,0)`,
		chefID.String(), userID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO approval_requests (id, type, chef_id, status) VALUES (?,?,?,?)`,
		approvalID.String(), string(models.ApprovalKitchenOnboarding), chefID.String(), "pending").Error)

	// Run twice — idempotent.
	require.NoError(t, ActivateChefOnboarding(db, approvalID))
	require.NoError(t, ActivateChefOnboarding(db, approvalID))

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.True(t, chef.IsVerified)
	require.True(t, chef.IsActive)
	require.NotNil(t, chef.VerifiedAt)

	var role string
	db.Raw(`SELECT role FROM users WHERE id = ?`, userID.String()).Scan(&role)
	require.Equal(t, string(models.RoleChef), role)
}

func TestActivateChefOnboarding_NoOpForOtherType(t *testing.T) {
	db := setupOnboardingDB(t)
	approvalID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO approval_requests (id, type, status) VALUES (?,?,?)`,
		approvalID.String(), string(models.ApprovalMenuItemNew), "pending").Error)

	// Menu-item approval has no chef onboarding to activate → clean no-op.
	require.NoError(t, ActivateChefOnboarding(db, approvalID))
}
