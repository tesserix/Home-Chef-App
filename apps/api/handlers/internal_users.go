package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// InternalUsersHandler owns the BFF-only user upsert path. Called by
// apps/auth-bff on every successful Google Identity Platform sign-in via the
// HMAC-signed pathway from Task 1.6 + verified by BFFAuth (Task 2.1). New
// identities get inserted; returning ones only refresh last_login_at.
//
// Cross-pool same-email behavior (one person can exist as both a customer and
// a vendor under the same email) is enforced by the partial unique index in
// migration 20260514000002 — at the application layer we dispatch by gip_uid
// so this handler never has to see the conflict.
type InternalUsersHandler struct {
	DB *gorm.DB
}

// NewInternalUsersHandler builds a handler bound to the given GORM connection.
// We pass DB in explicitly (rather than reading the package-level
// database.DB) so unit tests can swap in an in-memory sqlite instance.
func NewInternalUsersHandler(db *gorm.DB) *InternalUsersHandler {
	return &InternalUsersHandler{DB: db}
}

// UpsertUserRequest mirrors the BFF's apiclient.UpsertUserRequest shape.
// Every field that maps to a GIP identity column is required so that we
// fail loudly if the BFF stops sending one.
type UpsertUserRequest struct {
	GIPUid      string `json:"gip_uid" binding:"required"`
	GIPTenantID string `json:"gip_tenant_id" binding:"required"`
	GIPProvider string `json:"gip_provider" binding:"required"`
	AuthPool    string `json:"auth_pool" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Name        string `json:"name"`
	// Avatar is the GIP token's "picture" claim URL. Backfilled onto the user
	// row only when the stored avatar is empty — never overwrites a user's
	// edited avatar on repeat login.
	Avatar string `json:"avatar"`
	// EmailVerified is the GIP token's email_verified claim. It gates same-email
	// account re-bind (see Upsert): an unverified password signup must not
	// hijack a verified social account's row. Not persisted — request-only.
	EmailVerified bool   `json:"email_verified"`
	Role          string `json:"role" binding:"required"`
	// MarketingConsent is the user's DPDP §6 opt-in for promotional email,
	// captured at registration. Optional in the request body so legacy
	// callers (and social sign-in paths that have no checkbox) default to
	// false. Only honored on NEW user creation — re-login does not flip
	// the flag in either direction.
	// TODO(CW-01b): expose a separate /users/:id/preferences endpoint for
	// updating consent post-registration.
	MarketingConsent bool `json:"marketing_consent"`
}

// UpsertUserResponse returns the canonical user_id (UUID string) so the BFF
// can mint its session cookie against a stable identifier.
type UpsertUserResponse struct {
	UserID string `json:"user_id"`
}

// Upsert idempotently materializes a user row for a GIP identity.
//
// Lookup is keyed on gip_uid (the GIP-issued subject claim). On first
// sign-in we create the row with email lowercased, the Name split into
// FirstName/LastName, every GIP column populated, LastLoginAt set, and
// IsActive=true. On subsequent sign-ins we only refresh LastLoginAt (and
// lazily backfill FirstName/LastName if they were previously blank).
//
// Errors:
//   - 400 on validation failures (bad email, missing required fields).
//   - 502 on any DB error — the BFF retries, so surfacing a gateway error
//     keeps the user-facing flow consistent with other upstream failures.
func (h *InternalUsersHandler) Upsert(c *gin.Context) {
	var req UpsertUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	email := strings.ToLower(req.Email)

	// Deleted-account handshake, before anything else.
	//
	// A soft-deleted row still owns its (lower(email), auth_pool) slot in
	// idx_users_email_per_pool — the index is not filtered on deleted_at. The
	// scoped lookups below cannot see that row, so they used to fall through to
	// INSERT, hit a duplicate key, and return 502: a deleted user could never
	// sign up again on the same email. Detect the ghost instead and hand back a
	// restore token so the app can offer "restore" or "start fresh".
	if h.handleDeletedAccount(c, email, req) {
		return
	}

	var u models.User
	res := h.DB.Where("gip_uid = ?", req.GIPUid).First(&u)
	switch {
	case errors.Is(res.Error, gorm.ErrRecordNotFound):
		// gip_uid miss — before creating a new row, look for an existing
		// row by email. The unique idx_users_email constraint would
		// otherwise reject the INSERT and surface a 502 to the BFF,
		// which is how this manifested for users who re-signed in via
		// a different identity provider or whose GIP tenant changed.
		// Scope the email lookup by auth_pool so we don't collapse a
		// customer-pool row into the chef-pool row when the same human uses
		// the same email across products. The unique constraint on email
		// is per-pool, not global — see TestUpsert_SameEmailDifferentPool.
		// SECURITY (T-due-01): only attempt a same-email re-bind when the
		// incoming token's email is verified. Otherwise an unverified password
		// signup using an email already owned by a verified social account
		// could hijack that existing row. Google/Apple GIP tokens are always
		// email_verified=true; only unverified password signups are false, so
		// they fall through to the new-user INSERT path below.
		if email != "" && req.AuthPool != "" && req.EmailVerified {
			byEmail := h.DB.Where("email = ? AND auth_pool = ?", email, req.AuthPool).First(&u)
			if byEmail.Error == nil {
				// Existing row found by email — re-bind it to the new
				// GIP identity and stamp last-login. This is the "same
				// human, new GIP uid" path.
				u.GIPUid = req.GIPUid
				u.GIPTenantID = req.GIPTenantID
				u.GIPProvider = req.GIPProvider
				if req.AuthPool != "" {
					u.AuthPool = models.AuthPool(req.AuthPool)
				}
				u.LastLoginAt = &now
				// Backfill-only: never clobber a name/avatar the user may have edited.
				if req.Name != "" && u.FirstName == "" && u.LastName == "" {
					u.FirstName, u.LastName = splitName(req.Name)
				}
				if req.Avatar != "" && u.Avatar == "" {
					u.Avatar = req.Avatar
				}
				if err := h.DB.Save(&u).Error; err != nil {
					c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, UpsertUserResponse{UserID: u.ID.String()})
				return
			}
			if !errors.Is(byEmail.Error, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadGateway, gin.H{"error": byEmail.Error.Error()})
				return
			}
		}

		first, last := splitName(req.Name)
		u = models.User{
			ID:               uuid.New(),
			Email:            email,
			FirstName:        first,
			LastName:         last,
			Avatar:           req.Avatar,
			GIPUid:           req.GIPUid,
			GIPTenantID:      req.GIPTenantID,
			GIPProvider:      req.GIPProvider,
			AuthPool:         models.AuthPool(req.AuthPool),
			Role:             models.UserRole(req.Role),
			LastLoginAt:      &now,
			IsActive:         true,
			MarketingConsent: req.MarketingConsent,
		}
		// Only stamp the consent timestamp if the user actually opted in.
		// Leaving it null preserves the "never granted" signal for DPDP audits.
		if req.MarketingConsent {
			u.MarketingConsentAt = &now
		}
		if err := h.DB.Create(&u).Error; err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	case res.Error != nil:
		c.JSON(http.StatusBadGateway, gin.H{"error": res.Error.Error()})
		return
	default:
		// Found by gip_uid — bump last_login_at, and lazily backfill the
		// name if the row was created before we started capturing one. We
		// never overwrite an existing FirstName/LastName here: the user
		// may have edited their profile and a re-login shouldn't clobber that.
		u.LastLoginAt = &now
		if req.Name != "" && u.FirstName == "" && u.LastName == "" {
			u.FirstName, u.LastName = splitName(req.Name)
		}
		// Backfill-only avatar: fill a blank avatar, never overwrite an edited one.
		if req.Avatar != "" && u.Avatar == "" {
			u.Avatar = req.Avatar
		}
		if err := h.DB.Save(&u).Error; err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, UpsertUserResponse{UserID: u.ID.String()})
}

// handleDeletedAccount looks for a soft-deleted account on this email+pool and,
// if one is still inside its restore window, answers the sign-in with a
// "restorable" response instead of creating a duplicate. Reports whether it has
// written the response.
//
// Requires req.EmailVerified. Without it an unverified password signup on a
// known address could probe for — or seize — someone's deleted account, the
// same hijack the live re-bind path below already guards against.
//
// A ghost whose window has already elapsed is purged here and now, which frees
// the unique email slot so the ordinary signup path can proceed.
func (h *InternalUsersHandler) handleDeletedAccount(c *gin.Context, email string, req UpsertUserRequest) bool {
	if email == "" || req.AuthPool == "" || !req.EmailVerified {
		return false
	}

	var ghost models.User
	err := h.DB.Unscoped().
		Where("email = ? AND auth_pool = ? AND deleted_at IS NOT NULL", email, req.AuthPool).
		First(&ghost).Error
	if err != nil {
		return false // no deleted account here — ordinary flow
	}

	// Window elapsed: the old account has no further claim on this address, so
	// release the email slot immediately rather than making the user wait for
	// the nightly sweeper to free their own address.
	//
	// This tombstones the email rather than purging inline. A full purge here
	// would run the whole role cascade on the sign-in path, and any failure in
	// it (a locked row, a missing child table) would 502 — re-creating exactly
	// the lockout this handshake exists to remove. One UPDATE cannot fail that
	// way, and the sweeper still erases the row properly on its next pass.
	if ghost.PurgeAfter != nil && time.Now().UTC().After(*ghost.PurgeAfter) {
		tombstone := fmt.Sprintf("purged-%s@deleted.invalid", ghost.ID)
		if err := h.DB.Unscoped().Model(&models.User{}).
			Where("id = ?", ghost.ID).
			UpdateColumns(map[string]any{"email": tombstone, "email_bidx": ""}).Error; err != nil {
			log.Printf("internal-users: could not release email for expired ghost user=%s: %v",
				ghost.ID, err)
			return false // fall through; a duplicate-key error is still surfaced below
		}
		return false // slot released — fall through and create a fresh account
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "restorable",
		"user_id":      ghost.ID.String(),
		"deletedAt":    ghost.DeletedAt.Time,
		"purgeAfter":   ghost.PurgeAfter,
		"restoreToken": services.MintRestoreToken(ghost.ID, req.GIPUid),
	})
	return true
}

// splitName splits "First Last [Middle...]" into FirstName + LastName.
// Empty input yields two empty strings. A single-token name goes into
// FirstName and LastName stays empty. Anything after the first space is
// kept verbatim as LastName (so "Jean-Paul van der Berg" becomes
// FirstName="Jean-Paul", LastName="van der Berg").
func splitName(full string) (first, last string) {
	full = strings.TrimSpace(full)
	if full == "" {
		return "", ""
	}
	parts := strings.SplitN(full, " ", 2)
	first = parts[0]
	if len(parts) == 2 {
		last = strings.TrimSpace(parts[1])
	}
	return
}
