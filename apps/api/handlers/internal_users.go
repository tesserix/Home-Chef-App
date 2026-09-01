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

// InternalUsersHandler owns the BFF-only user upsert path, called by
// apps/auth-bff on every successful sign-in via the HMAC-signed pathway and
// verified by BFFAuth. Identities are keyed by (provider, subject) in the
// user_identities table; a verified-email match links a new identity to an
// existing user, which is what migrates GIP-era accounts to Zitadel seamlessly.
type InternalUsersHandler struct {
	DB *gorm.DB
}

// NewInternalUsersHandler builds a handler bound to the given GORM connection.
// DB is passed explicitly so unit tests can swap in an in-memory sqlite instance.
func NewInternalUsersHandler(db *gorm.DB) *InternalUsersHandler {
	return &InternalUsersHandler{DB: db}
}

// UpsertUserRequest mirrors the BFF's apiclient.UpsertUserRequest shape.
type UpsertUserRequest struct {
	// Provider + Subject identify the upstream identity ("zitadel" + OIDC sub).
	Provider string `json:"provider"`
	Subject  string `json:"subject"`
	// Legacy GIP shape — accepted during rollout so an older BFF build can
	// still upsert; normalise() maps it onto Provider="gip", Subject=gip_uid.
	GIPUid      string `json:"gip_uid"`
	GIPTenantID string `json:"gip_tenant_id"`
	GIPProvider string `json:"gip_provider"`
	AuthPool    string `json:"auth_pool" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Name        string `json:"name"`
	// Avatar is the id_token's "picture" claim URL. Backfilled onto the user
	// row only when the stored avatar is empty — never overwrites an edit.
	Avatar string `json:"avatar"`
	// EmailVerified gates same-email account linking (see Upsert): an
	// unverified signup must not hijack a verified account's row. Request-only.
	EmailVerified bool   `json:"email_verified"`
	Role          string `json:"role" binding:"required"`
	// MarketingConsent is the DPDP §6 opt-in captured at registration. Only
	// honored on NEW user creation — re-login does not flip the flag.
	// TODO(CW-01b): expose a /users/:id/preferences endpoint for updates.
	MarketingConsent bool `json:"marketing_consent"`
	// Device describes the install this sign-in came from (#1164). Optional.
	DeviceID    string `json:"device_id"`
	Platform    string `json:"platform"`
	DeviceLabel string `json:"device_label"`
	AppVersion  string `json:"app_version"`
	IP          string `json:"ip"`
}

// normalise folds the legacy GIP shape into provider/subject and validates.
func (req *UpsertUserRequest) normalise() error {
	if req.Provider == "" && req.GIPUid != "" {
		req.Provider, req.Subject = "gip", req.GIPUid
	}
	if req.Provider == "" || req.Subject == "" {
		return errors.New("provider and subject are required")
	}
	return nil
}

// loginDeviceNotifier runs the device sighting off the request path: a sign-in
// must not wait on geolocation or on the mail provider. A var so tests can
// observe the sighting without racing the goroutine.
var loginDeviceNotifier = func(db *gorm.DB, u models.User, in services.LoginSighting) {
	go services.NoteLoginDevice(db, u, in)
}

func (req UpsertUserRequest) sighting(firstLogin bool) services.LoginSighting {
	return services.LoginSighting{
		DeviceID:   req.DeviceID,
		App:        req.Role,
		Platform:   req.Platform,
		Label:      req.DeviceLabel,
		AppVersion: req.AppVersion,
		IP:         req.IP,
		FirstLogin: firstLogin,
	}
}

// UpsertUserResponse returns the canonical user_id (UUID string) so the BFF
// can mint its session cookie against a stable identifier.
type UpsertUserResponse struct {
	UserID string `json:"user_id"`
}

// Upsert idempotently materializes a user row for an upstream identity.
//
// Resolution order:
//  1. user_identities (provider, subject) hit → returning login.
//  2. legacy gip_uid column hit (provider "gip" only) → GIP-era row that
//     predates the identities table; link an identity row to it.
//  3. verified-email + same-pool match → same human on a new identity
//     (the GIP→Zitadel migration path); link, never duplicate.
//  4. otherwise → create user + identity.
//
// Errors: 400 on validation failures, 502 on any DB error (the BFF retries).
func (h *InternalUsersHandler) Upsert(c *gin.Context) {
	var req UpsertUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := req.normalise(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	email := strings.ToLower(req.Email)

	// Deleted-account handshake, before anything else. A soft-deleted row still
	// owns its (lower(email), auth_pool) slot, so signup would 502 on the
	// unique index forever; offer restore instead (see handleDeletedAccount).
	if h.handleDeletedAccount(c, email, req) {
		return
	}

	var u models.User
	firstLogin := false
	var ident models.UserIdentity
	identErr := h.DB.Where("provider = ? AND subject = ?", req.Provider, req.Subject).First(&ident).Error
	switch {
	case identErr == nil:
		if err := h.DB.First(&u, "id = ?", ident.UserID).Error; err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		if !h.refreshOnLogin(c, &u, req, now) {
			return
		}
		_ = h.DB.Model(&models.UserIdentity{}).Where("id = ?", ident.ID).
			UpdateColumn("last_login_at", now).Error
	case errors.Is(identErr, gorm.ErrRecordNotFound):
		linked := false
		// Legacy rows predate user_identities — match GIP logins on the old
		// gip_uid column so they land on the same row and gain an identity row.
		if req.Provider == "gip" {
			err := h.DB.Where("gip_uid = ?", req.Subject).First(&u).Error
			switch {
			case err == nil:
				linked = true
			case !errors.Is(err, gorm.ErrRecordNotFound):
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}
		}
		// SECURITY (T-due-01): only link by email when the incoming token's
		// email is verified — an unverified password signup on an address owned
		// by a verified account must not hijack that row. Scoped by auth_pool:
		// the same human may exist separately as customer and chef.
		if !linked && email != "" && req.AuthPool != "" && req.EmailVerified {
			err := h.DB.Where("email = ? AND auth_pool = ?", email, req.AuthPool).First(&u).Error
			switch {
			case err == nil:
				linked = true
			case !errors.Is(err, gorm.ErrRecordNotFound):
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}
		}
		if linked {
			// Same human, new identity — keep the legacy GIP columns current
			// only for legacy callers; Zitadel identities live in their own row.
			if req.Provider == "gip" {
				u.GIPUid = req.Subject
				u.GIPTenantID = req.GIPTenantID
				u.GIPProvider = req.GIPProvider
			}
			if req.AuthPool != "" {
				u.AuthPool = models.AuthPool(req.AuthPool)
			}
			if !h.refreshOnLogin(c, &u, req, now) {
				return
			}
		} else {
			first, last := splitName(req.Name)
			u = models.User{
				ID:               uuid.New(),
				Email:            email,
				FirstName:        first,
				LastName:         last,
				Avatar:           req.Avatar,
				AuthPool:         models.AuthPool(req.AuthPool),
				Role:             models.UserRole(req.Role),
				LastLoginAt:      &now,
				IsActive:         true,
				MarketingConsent: req.MarketingConsent,
			}
			if req.Provider == "gip" {
				u.GIPUid = req.Subject
				u.GIPTenantID = req.GIPTenantID
				u.GIPProvider = req.GIPProvider
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
			firstLogin = true
		}
		// Hard-fail: without the identity row the next login would re-run the
		// email link (fine when verified) or create a duplicate (502 forever).
		if err := h.linkIdentity(u.ID, req, now); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
	default:
		c.JSON(http.StatusBadGateway, gin.H{"error": identErr.Error()})
		return
	}
	loginDeviceNotifier(h.DB, u, req.sighting(firstLogin))
	c.JSON(http.StatusOK, UpsertUserResponse{UserID: u.ID.String()})
}

// refreshOnLogin bumps last_login_at and lazily backfills name/avatar — never
// overwriting values the user may have edited. Writes the 502 itself on error.
func (h *InternalUsersHandler) refreshOnLogin(c *gin.Context, u *models.User, req UpsertUserRequest, now time.Time) bool {
	u.LastLoginAt = &now
	if req.Name != "" && u.FirstName == "" && u.LastName == "" {
		u.FirstName, u.LastName = splitName(req.Name)
	}
	if req.Avatar != "" && u.Avatar == "" {
		u.Avatar = req.Avatar
	}
	if err := h.DB.Save(u).Error; err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return false
	}
	return true
}

func (h *InternalUsersHandler) linkIdentity(userID uuid.UUID, req UpsertUserRequest, now time.Time) error {
	return h.DB.Create(&models.UserIdentity{
		ID:          uuid.New(),
		UserID:      userID,
		Provider:    req.Provider,
		Subject:     req.Subject,
		Email:       strings.ToLower(req.Email),
		LastLoginAt: &now,
	}).Error
}

// handleDeletedAccount looks for a soft-deleted account on this email+pool and,
// if one is still inside its restore window, answers the sign-in with a
// "restorable" response instead of creating a duplicate. Reports whether it has
// written the response.
//
// Requires req.EmailVerified. Without it an unverified password signup on a
// known address could probe for — or seize — someone's deleted account, the
// same hijack the live linking path above already guards against.
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
		"restoreToken": services.MintRestoreToken(ghost.ID, req.Subject),
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
