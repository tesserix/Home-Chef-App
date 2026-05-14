package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// SecurityHandler serves the admin-facing security surfaces that the auth
// migration to GIP did NOT subsume: platform security policy CRUD, platform
// API key CRUD, the audit-log viewer, and the public password policy lookup.
//
// The Keycloak-era session list/revoke, TOTP enroll/verify, and password-
// verify endpoints that used to live here were removed as part of Task 2.6 —
// session lifecycle and 2FA are now owned by apps/auth-bff via Google
// Identity Platform.
type SecurityHandler struct{}

func NewSecurityHandler() *SecurityHandler { return &SecurityHandler{} }

// ========================================================================
// Password policy — public so registration pages can render live hints
// ========================================================================

// GetPasswordPolicy returns the password rules currently in force. Safe to
// expose publicly — these are the same rules the auth-bff enforces at submit
// time, so the client already has to obey them.
func (h *SecurityHandler) GetPasswordPolicy(c *gin.Context) {
	p := services.GetSecurityPolicy()
	c.JSON(http.StatusOK, gin.H{
		"minLength":      p.PasswordMinLength,
		"requireUpper":   p.PasswordRequireUpper,
		"requireLower":   p.PasswordRequireLower,
		"requireNumber":  p.PasswordRequireNumber,
		"requireSpecial": p.PasswordRequireSpecial,
	})
}

// ========================================================================
// Admin: SecurityPolicy CRUD
// ========================================================================

// AdminGetSecurityPolicy returns the full policy (admin-only).
func (h *SecurityHandler) AdminGetSecurityPolicy(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetSecurityPolicy())
}

// AdminUpdateSecurityPolicy upserts the policy. Unspecified fields are left
// at their current values — this plays nicely with per-section saves from
// the UI (e.g. the Sessions tab only sends session fields).
func (h *SecurityHandler) AdminUpdateSecurityPolicy(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	current := services.GetSecurityPolicy()
	// Accept every field as optional; only overwrite when the client sent it.
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if v, ok := req["passwordMinLength"].(float64); ok {
		current.PasswordMinLength = int(v)
	}
	if v, ok := req["passwordRequireUpper"].(bool); ok {
		current.PasswordRequireUpper = v
	}
	if v, ok := req["passwordRequireLower"].(bool); ok {
		current.PasswordRequireLower = v
	}
	if v, ok := req["passwordRequireNumber"].(bool); ok {
		current.PasswordRequireNumber = v
	}
	if v, ok := req["passwordRequireSpecial"].(bool); ok {
		current.PasswordRequireSpecial = v
	}
	if v, ok := req["sessionAccessTtlHours"].(float64); ok {
		current.SessionAccessTTLHours = int(v)
	}
	if v, ok := req["sessionRefreshTtlDays"].(float64); ok {
		current.SessionRefreshTTLDays = int(v)
	}
	if v, ok := req["twoFactorRequiredForAdmins"].(bool); ok {
		current.TwoFactorRequiredForAdmins = v
	}
	if v, ok := req["twoFactorExemptEmails"].([]any); ok {
		emails := make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok && strings.TrimSpace(s) != "" {
				emails = append(emails, strings.TrimSpace(s))
			}
		}
		current.TwoFactorExemptEmails = emails
	}

	if err := services.SaveSecurityPolicy(current, &userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.LogAudit(c, "security.policy.update", "security_policy", "", nil, current)
	c.JSON(http.StatusOK, current)
}

// ========================================================================
// Admin: API keys CRUD
// ========================================================================

// AdminListApiKeys returns all keys (without the secret).
func (h *SecurityHandler) AdminListApiKeys(c *gin.Context) {
	var keys []models.ApiKey
	database.DB.Order("created_at DESC").Find(&keys)
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

// AdminCreateApiKey mints a new key. The plaintext secret appears in this
// response exactly once — the admin must copy it immediately.
func (h *SecurityHandler) AdminCreateApiKey(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		Name      string   `json:"name" binding:"required"`
		Scopes    []string `json:"scopes"`
		ExpiresIn int      `json:"expiresInDays"` // 0 = never
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var expiresAt *time.Time
	if req.ExpiresIn > 0 {
		t := time.Now().AddDate(0, 0, req.ExpiresIn)
		expiresAt = &t
	}
	gen, err := services.CreateApiKey(strings.TrimSpace(req.Name), req.Scopes, userID, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"key":     gen.Record,
		"fullKey": gen.FullKey, // shown once
	})
}

// AdminRevokeApiKey marks a key revoked — it stops authenticating immediately.
func (h *SecurityHandler) AdminRevokeApiKey(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}
	if err := services.RevokeApiKey(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.LogAudit(c, "api_key.revoke", "api_key", id.String(), nil, nil)
	c.JSON(http.StatusOK, gin.H{"message": "Key revoked"})
}

// ========================================================================
// Admin: Audit log viewer
// ========================================================================

// AdminListAuditLogs returns paginated audit entries with optional filters.
// Query params: action (prefix match), entityType, userId, from, to, page, limit.
func (h *SecurityHandler) AdminListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 50
	}

	q := database.DB.Model(&models.AuditLog{}).Preload("User")
	if action := c.Query("action"); action != "" {
		q = q.Where("action LIKE ?", action+"%")
	}
	if et := c.Query("entityType"); et != "" {
		q = q.Where("entity_type = ?", et)
	}
	if uid := c.Query("userId"); uid != "" {
		if parsed, err := uuid.Parse(uid); err == nil {
			q = q.Where("user_id = ?", parsed)
		}
	}
	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			q = q.Where("created_at < ?", t.Add(24*time.Hour))
		}
	}

	var total int64
	q.Count(&total)

	var logs []models.AuditLog
	q.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&logs)

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
