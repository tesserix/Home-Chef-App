package handlers

import (
	"encoding/base64"
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
	"golang.org/x/crypto/bcrypt"
)

type SecurityHandler struct{}

func NewSecurityHandler() *SecurityHandler { return &SecurityHandler{} }

// ========================================================================
// Password policy — public so registration pages can render live hints
// ========================================================================

// GetPasswordPolicy returns the password rules currently in force. Safe to
// expose publicly — these are the same rules the server enforces at submit
// time, so the client already has to obey them.
func (h *SecurityHandler) GetPasswordPolicy(c *gin.Context) {
	p := services.GetSecurityPolicy()
	c.JSON(http.StatusOK, gin.H{
		"minLength":       p.PasswordMinLength,
		"requireUpper":    p.PasswordRequireUpper,
		"requireLower":    p.PasswordRequireLower,
		"requireNumber":   p.PasswordRequireNumber,
		"requireSpecial":  p.PasswordRequireSpecial,
	})
}

// ========================================================================
// Session management — user-facing
// ========================================================================

// ListMySessions returns the caller's currently active refresh tokens, which
// map 1:1 to logged-in devices.
func (h *SecurityHandler) ListMySessions(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sessions []models.RefreshToken
	database.DB.Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("created_at DESC").Find(&sessions)

	// Sanitize — never return the raw refresh token string.
	out := make([]gin.H, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, gin.H{
			"id":         s.ID,
			"userAgent":  s.UserAgent,
			"ipAddress":  s.IPAddress,
			"createdAt":  s.CreatedAt,
			"expiresAt":  s.ExpiresAt,
			"lastUsedAt": s.LastUsedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"sessions": out})
}

// RevokeMySession revokes one of the caller's sessions by its ID. Revoking
// the session the caller is currently using is allowed — they'll just need
// to log in again on next request.
func (h *SecurityHandler) RevokeMySession(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	sessionID, err := uuid.Parse(c.Param("sessionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session id"})
		return
	}
	now := time.Now()
	res := database.DB.Model(&models.RefreshToken{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", sessionID, userID).
		Update("revoked_at", now)
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Session revoked"})
}

// RevokeAllMySessions revokes every non-revoked refresh token for the caller.
// The current request will continue to work until its access token expires
// (short TTL) but no new access tokens will be issued.
func (h *SecurityHandler) RevokeAllMySessions(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	now := time.Now()
	database.DB.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now)
	c.JSON(http.StatusOK, gin.H{"message": "All sessions revoked"})
}

// ========================================================================
// 2FA — TOTP enrollment & verification
// ========================================================================

type totpEnrollVerifyRequest struct {
	EnrollmentToken string `json:"enrollmentToken" binding:"required"`
	Code            string `json:"code" binding:"required"`
}

// StartTOTPEnrollment begins enrollment for an already-authenticated user
// (Authorization: Bearer <JWT>). Returns the secret, otpauth URL, and a PNG
// QR code ready for <img src="data:image/png;base64,..." />.
func (h *SecurityHandler) StartTOTPEnrollment(c *gin.Context) {
	user, ok := middleware.GetUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	enrol, err := services.EnrollTOTP(c.Request.Context(), user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"secret":       enrol.Secret,
		"otpAuthUrl":   enrol.OtpAuthURL,
		"qrCodeBase64": base64.StdEncoding.EncodeToString(enrol.QRCodePNG),
	})
}

// VerifyTOTPEnrollment confirms the enrollment by checking a code generated
// from the stored secret. Once this succeeds, TOTPEnabled flips to true and
// subsequent logins require a code.
func (h *SecurityHandler) VerifyTOTPEnrollment(c *gin.Context) {
	user, ok := middleware.GetUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := services.VerifyTOTPCode(c.Request.Context(), user.ID, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid code"})
		return
	}
	now := time.Now()
	user.TOTPEnabled = true
	user.TOTPVerifiedAt = &now
	if err := database.DB.Save(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable 2FA"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "2FA enabled"})
}

// VerifyTOTP is the login-flow continuation — called after /auth/login
// returns twoFactorRequired. Accepts the challenge token + 6-digit code and
// issues real access/refresh tokens on success.
func (h *SecurityHandler) VerifyTOTP(c *gin.Context) {
	var req struct {
		ChallengeToken string `json:"challengeToken" binding:"required"`
		Code           string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := middleware.ParseTwoFactorChallenge(req.ChallengeToken, middleware.ChallengeVerify)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired challenge"})
		return
	}
	if err := services.VerifyTOTPCode(c.Request.Context(), userID, req.Code); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	now := time.Now()
	user.LastLoginAt = &now
	database.DB.Save(&user)

	accessToken, refreshToken, err := middleware.GenerateTokensWithContext(&user, c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}
	c.JSON(http.StatusOK, AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// ForcedEnrollTOTPStart is called when login returned twoFactorEnrollmentRequired
// (admin trying to log in without 2FA enrolled under an enforce-2FA policy).
// Consumes an enrollment-kind challenge token, returns the QR data.
func (h *SecurityHandler) ForcedEnrollTOTPStart(c *gin.Context) {
	var req struct {
		EnrollmentToken string `json:"enrollmentToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := middleware.ParseTwoFactorChallenge(req.EnrollmentToken, middleware.ChallengeEnroll)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired enrollment token"})
		return
	}
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	enrol, err := services.EnrollTOTP(c.Request.Context(), user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"secret":       enrol.Secret,
		"otpAuthUrl":   enrol.OtpAuthURL,
		"qrCodeBase64": base64.StdEncoding.EncodeToString(enrol.QRCodePNG),
	})
}

// ForcedEnrollTOTPVerify completes the forced-enrollment flow. On success it
// flips TOTPEnabled and issues real login tokens so the admin can proceed.
func (h *SecurityHandler) ForcedEnrollTOTPVerify(c *gin.Context) {
	var req totpEnrollVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := middleware.ParseTwoFactorChallenge(req.EnrollmentToken, middleware.ChallengeEnroll)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired enrollment token"})
		return
	}
	if err := services.VerifyTOTPCode(c.Request.Context(), userID, req.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid code"})
		return
	}
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	now := time.Now()
	user.TOTPEnabled = true
	user.TOTPVerifiedAt = &now
	user.LastLoginAt = &now
	database.DB.Save(&user)

	accessToken, refreshToken, err := middleware.GenerateTokensWithContext(&user, c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}
	c.JSON(http.StatusOK, AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// DisableTOTP turns 2FA off for the calling user. Requires re-verifying the
// current password AND a current TOTP code so a hijacked access token alone
// can't disable the second factor.
func (h *SecurityHandler) DisableTOTP(c *gin.Context) {
	user, ok := middleware.GetUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		Password string `json:"password" binding:"required"`
		Code     string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password is incorrect"})
		return
	}
	if err := services.VerifyTOTPCode(c.Request.Context(), user.ID, req.Code); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid code"})
		return
	}
	user.TOTPEnabled = false
	user.TOTPVerifiedAt = nil
	database.DB.Save(user)
	_ = services.DisableTOTPSecret(c.Request.Context(), user.ID)
	c.JSON(http.StatusOK, gin.H{"message": "2FA disabled"})
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
