package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/models"
)

// The legacy Keycloak-era authentication middleware lived here. It has been
// removed as part of the GIP migration (Task 2.6). All session validation is
// now performed by apps/auth-bff which forwards verified-identity headers to
// us via BFFAuth (see bff_auth.go).
//
// This file retains only the small context-accessor helpers that handlers
// rely on. The new BFFAuth middleware seeds both the new context keys
// (user_id, user_role, ...) and the legacy aliases (userID as uuid.UUID,
// userRole as models.UserRole, user as *models.User) so handlers keep
// working without per-handler edits.

// GetUserID pulls the authenticated user's UUID off the Gin context. Returns
// (uuid.Nil, false) when no user is attached — middleware MUST set "userID"
// on protected routes.
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}
	switch v := userID.(type) {
	case uuid.UUID:
		return v, true
	case string:
		// Defensive: if some future middleware sets the legacy key as a
		// string, parse it here rather than panicking on a bad type
		// assertion in a hot handler path.
		if parsed, err := uuid.Parse(v); err == nil {
			return parsed, true
		}
	}
	return uuid.Nil, false
}

// GetUserRole pulls the authenticated user's role off the Gin context.
func GetUserRole(c *gin.Context) (models.UserRole, bool) {
	role, exists := c.Get("userRole")
	if !exists {
		return "", false
	}
	switch v := role.(type) {
	case models.UserRole:
		return v, true
	case string:
		return models.UserRole(v), true
	}
	return "", false
}

// GetAuthPool pulls the caller's GIP identity pool (customer/business/internal)
// off the Gin context. BFFAuth seeds it from the signed X-Auth-Pool header.
// Returns ("", false) when no pool is attached.
func GetAuthPool(c *gin.Context) (models.AuthPool, bool) {
	pool, exists := c.Get(CtxAuthPool)
	if !exists {
		return "", false
	}
	switch v := pool.(type) {
	case models.AuthPool:
		return v, true
	case string:
		if v == "" {
			return "", false
		}
		return models.AuthPool(v), true
	}
	return "", false
}

// GetUser returns the hydrated *models.User attached by BFFAuth, or
// (nil, false) when the user row could not be loaded.
func GetUser(c *gin.Context) (*models.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}
	u, ok := user.(*models.User)
	if !ok {
		return nil, false
	}
	return u, true
}
