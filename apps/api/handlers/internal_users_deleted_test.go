package handlers

// internal_users_deleted_test.go — the re-signup handshake for a deleted
// account.
//
// Before this existed, deleting an account permanently locked its email out:
// idx_users_email_per_pool is not filtered on deleted_at, so the soft-deleted
// row kept owning the (lower(email), auth_pool) slot while the scoped lookups
// in Upsert could not see it. Sign-up fell through to INSERT, hit a duplicate
// key and returned 502 forever. These tests pin the replacement behaviour.

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// withUpsertRestoreKey installs a signing key for the restore token.
func withUpsertRestoreKey(t *testing.T) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig = &config.Config{
		BFFInternalHMACKey: []byte("test-key-at-least-16-bytes-long!"),
	}
	t.Cleanup(func() { config.AppConfig = prev })
}

// softDelete marks a user deleted with a purge window offsetFromNow away. A
// negative offset represents an account whose restore window has elapsed.
func softDelete(t *testing.T, db *gorm.DB, userID string, offsetFromNow time.Duration) {
	t.Helper()
	now := time.Now().UTC()
	require.NoError(t, db.Exec(
		`UPDATE users SET deleted_at = ?, purge_after = ?, is_active = 0 WHERE id = ?`,
		now, now.Add(offsetFromNow), userID).Error)
}

func TestUpsert_DeletedAccount_OffersRestoreInsteadOfLockingOut(t *testing.T) {
	withUpsertRestoreKey(t)
	db := setupDB(t)
	h := NewInternalUsersHandler(db)

	w1 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-1", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "gone@example.com", Name: "Gone User",
		EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w1.Code, "body: %s", w1.Body.String())
	var original models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-1").First(&original).Error)

	softDelete(t, db, original.ID.String(), 180*24*time.Hour)

	// Same human signs up again — new GIP identity, same email.
	w2 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-2", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "gone@example.com", Name: "Gone User",
		EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w2.Code,
		"a returning user must not be locked out; body: %s", w2.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &body))
	assert.Equal(t, "restorable", body["status"])
	assert.Equal(t, original.ID.String(), body["user_id"])
	assert.NotEmpty(t, body["restoreToken"], "app needs a token to authorise the restore")

	// Critically: no duplicate account was created.
	var count int64
	require.NoError(t, db.Unscoped().Model(&models.User{}).
		Where("email = ?", "gone@example.com").Count(&count).Error)
	assert.Equal(t, int64(1), count, "must not create a second row for the same email")
}

func TestUpsert_DeletedAccount_PastWindowPurgesAndSignsUpFresh(t *testing.T) {
	withUpsertRestoreKey(t)
	db := setupDB(t)
	h := NewInternalUsersHandler(db)

	w1 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-1", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "expired@example.com", Name: "Expired User",
		EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	var original models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-1").First(&original).Error)

	// Window elapsed — the old account has no claim on the email any more.
	softDelete(t, db, original.ID.String(), -time.Hour)

	w2 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-2", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "expired@example.com", Name: "Expired User",
		EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w2.Code, "body: %s", w2.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &body))
	assert.NotEqual(t, "restorable", body["status"],
		"an expired account must not be offered for restore")

	// The ghost's email is tombstoned so the unique slot is free immediately.
	// The row itself is left for the purge sweeper: running the full role
	// cascade on the sign-in path risks a failure that would 502 and re-create
	// the very lockout this handshake removes.
	var ghost models.User
	require.NoError(t, db.Unscoped().First(&ghost, "id = ?", original.ID).Error)
	assert.NotEqual(t, "expired@example.com", ghost.Email,
		"expired ghost must release the email address")
	assert.Contains(t, ghost.Email, "deleted.invalid")

	// And a genuinely new account now owns the address.
	var fresh models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-2").First(&fresh).Error)
	assert.NotEqual(t, original.ID, fresh.ID, "a brand-new account, not a revival")
	assert.Equal(t, "expired@example.com", fresh.Email)
}

func TestUpsert_DeletedAccount_UnverifiedEmailIsNotTold(t *testing.T) {
	withUpsertRestoreKey(t)
	db := setupDB(t)
	h := NewInternalUsersHandler(db)

	w1 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-1", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "victim@example.com", Name: "Victim",
		EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	var original models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-1").First(&original).Error)
	softDelete(t, db, original.ID.String(), 180*24*time.Hour)

	// An unverified password signup on the same address must not learn that a
	// deleted account exists, let alone receive a token that could restore it.
	w2 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-attacker", GIPTenantID: "T", GIPProvider: "password",
		AuthPool: "customer", Email: "victim@example.com", Name: "Attacker",
		EmailVerified: false, Role: "customer",
	})

	var body map[string]any
	_ = json.Unmarshal(w2.Body.Bytes(), &body)
	assert.NotEqual(t, "restorable", body["status"],
		"unverified signup must not be offered someone else's deleted account")
	assert.Empty(t, body["restoreToken"], "no restore token without a verified email")
}

func TestUpsert_DeletedAccount_OtherPoolIsUnaffected(t *testing.T) {
	withUpsertRestoreKey(t)
	db := setupDB(t)
	h := NewInternalUsersHandler(db)

	w1 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-cust", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "dual@example.com", Name: "Dual Role",
		EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w1.Code)
	var cust models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-cust").First(&cust).Error)
	softDelete(t, db, cust.ID.String(), 180*24*time.Hour)

	// The same human's business-pool account is a different row entirely and
	// must sign up normally, not inherit the customer-pool deletion.
	w2 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-chef", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "business", Email: "dual@example.com", Name: "Dual Role",
		EmailVerified: true, Role: "chef",
	})
	require.Equal(t, http.StatusOK, w2.Code, "body: %s", w2.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &body))
	assert.NotEqual(t, "restorable", body["status"],
		"a deletion in one pool must not block the other pool")
}
