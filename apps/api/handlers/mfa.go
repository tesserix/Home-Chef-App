package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/piicrypto"
	"github.com/homechef/api/services"
)

// Two-factor HTTP surface.
//
// Every route here is exempt from MFAGate — they are the endpoints a challenged
// user must reach in order to become unchallenged. That makes them the one place
// in the API where an un-elevated session is allowed to do something, so each
// handler re-derives what it is allowed to do rather than trusting the caller.

type MFAHandler struct{}

func NewMFAHandler() *MFAHandler { return &MFAHandler{} }

// ---- requests ---------------------------------------------------------------

type mfaChallengeRequest struct {
	Channel string `json:"channel" binding:"required"`
}

type mfaVerifyRequest struct {
	Channel string `json:"channel"`
	Code    string `json:"code"`
	// BackupCode is the recovery path; supplying it makes Channel irrelevant.
	BackupCode string `json:"backupCode"`
	// FirebaseIDToken carries the phone leg: a token minted after the client
	// completed Firebase phone verification, whose claims hold the verified
	// number. Verified server-side — the client's word is worth nothing here.
	FirebaseIDToken string `json:"firebaseIdToken"`
	// RememberDevice persists trust until explicitly revoked. Without it the
	// caller gets a session-scoped elevation token instead.
	RememberDevice bool   `json:"rememberDevice"`
	DeviceLabel    string `json:"deviceLabel"`
	Platform       string `json:"platform"`
}

type mfaEnrollEmailVerify struct {
	Code string `json:"code" binding:"required"`
}

type mfaEnrollPhone struct {
	// FirebaseIDToken is minted after the client completes Firebase phone
	// verification; its claims carry the number Google verified.
	FirebaseIDToken string `json:"firebaseIdToken" binding:"required"`
}

// ---- helpers ----------------------------------------------------------------

func mfaEnabled() bool {
	return config.AppConfig != nil && config.AppConfig.MFAEnabled
}

func backupCodeKey() string {
	if config.AppConfig == nil {
		return ""
	}
	return config.AppConfig.MFABackupCodeKey
}

// currentUser resolves the caller and their account email in one step; almost
// every handler here needs both.
func currentUser(c *gin.Context) (uuid.UUID, models.User, bool) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return uuid.Nil, models.User{}, false
	}
	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return uuid.Nil, models.User{}, false
	}
	return userID, user, true
}

// upsertSettings creates the settings row on first touch. Returning the row
// keeps callers from having to distinguish "new" from "existing".
func upsertSettings(userID uuid.UUID) (models.UserMFASettings, error) {
	s, err := services.GetMFASettings(database.DB, userID)
	if err != nil {
		return s, err
	}
	var count int64
	if err := database.DB.Model(&models.UserMFASettings{}).
		Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return s, err
	}
	if count == 0 {
		s = models.UserMFASettings{UserID: userID, UpdatedAt: time.Now()}
		if err := database.DB.Create(&s).Error; err != nil {
			return s, err
		}
	}
	return s, nil
}

// ---- status -----------------------------------------------------------------

// GetStatus reports the caller's two-factor state, for the settings screen.
func (h *MFAHandler) GetStatus(c *gin.Context) {
	userID, user, ok := currentUser(c)
	if !ok {
		return
	}
	// Answer before touching the database when the feature is off.
	//
	// The two-factor tables ship separately (tesserix-k8s), so between a client
	// release and that schema landing this query hits a table that does not
	// exist. Ordering the flag check second turned "two-factor isn't available
	// yet" — which every client already renders cleanly — into a 500 and an
	// error screen. Nothing below is meaningful with the feature off anyway.
	if !mfaEnabled() {
		c.JSON(http.StatusOK, gin.H{
			"featureEnabled":       false,
			"enabled":              false,
			"emailEnrolled":        false,
			"phoneEnrolled":        false,
			"maskedEmail":          services.MaskEmail(user.Email),
			"maskedPhone":          "",
			"backupCodesRemaining": 0,
			"channels":             []services.MFAChannel{},
		})
		return
	}
	s, err := services.GetMFASettings(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load two-factor settings"})
		return
	}
	remaining, _ := services.CountUnusedBackupCodes(database.DB, userID)

	c.JSON(http.StatusOK, gin.H{
		"featureEnabled":       mfaEnabled(),
		"enabled":              s.Enabled,
		"emailEnrolled":        s.EmailEnrolled,
		"phoneEnrolled":        s.PhoneEnrolled,
		"maskedEmail":          services.MaskEmail(user.Email),
		"maskedPhone":          services.MaskPhone(string(s.PhoneE164Enc)),
		"backupCodesRemaining": remaining,
		"channels":             services.EnrolledChannels(s),
	})
}

// ---- enrollment -------------------------------------------------------------

// RequestEmailEnrollment sends a code to the account email to prove the user
// controls it before it becomes a second factor.
func (h *MFAHandler) RequestEmailEnrollment(c *gin.Context) {
	userID, user, ok := currentUser(c)
	if !ok {
		return
	}
	email := services.NormalizeEmail(user.Email)
	if !services.IsValidEmailFormat(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Your account email is invalid. Please contact support."})
		return
	}
	code, err := services.IssueOTP(c.Request.Context(), services.PurposeMFAEnroll, userID.String(), email, "")
	if err != nil {
		c.JSON(otpStatus(err), gin.H{"error": err.Error()})
		return
	}
	// Delivered inline, not over NATS: if this fails the user must be told now,
	// rather than being left waiting for a code that is not coming.
	if err := services.GetEmailService().SendEmailOTP(email, user.FirstName, code); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "We could not send the code. Please try again."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sent": true, "maskedEmail": services.MaskEmail(email), "expiresInSeconds": 600})
}

// VerifyEmailEnrollment marks email usable as a second factor. It does NOT turn
// two-factor on — see Enable.
func (h *MFAHandler) VerifyEmailEnrollment(c *gin.Context) {
	userID, user, ok := currentUser(c)
	if !ok {
		return
	}
	var req mfaEnrollEmailVerify
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter the 6-digit code", "field": "code"})
		return
	}
	email := services.NormalizeEmail(user.Email)
	if err := services.RedeemOTP(c.Request.Context(), services.PurposeMFAEnroll, userID.String(), email, "", req.Code); err != nil {
		c.JSON(otpStatus(err), gin.H{"error": err.Error(), "field": "code"})
		return
	}
	if _, err := upsertSettings(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save two-factor settings"})
		return
	}
	if err := database.DB.Model(&models.UserMFASettings{}).Where("user_id = ?", userID).
		Updates(map[string]any{"email_enrolled": true, "updated_at": time.Now()}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save two-factor settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"emailEnrolled": true})
}

// EnrollPhone records a phone number as a second factor, but only after Google
// says the user proved they hold it.
//
// The client runs Firebase phone verification, then posts the resulting ID
// token. Everything up to that point happened on a device we do not control, so
// the signature check here is what makes the phone factor a factor at all.
func (h *MFAHandler) EnrollPhone(c *gin.Context) {
	userID, user, ok := currentUser(c)
	if !ok {
		return
	}
	var req mfaEnrollPhone
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verify your phone first", "field": "firebaseIdToken"})
		return
	}
	verified, err := services.VerifyFirebasePhoneToken(c.Request.Context(), req.FirebaseIDToken, user.GIPUid)
	if err != nil {
		c.JSON(phoneVerifyStatus(err), gin.H{"error": err.Error(), "field": "firebaseIdToken"})
		return
	}
	if _, err := upsertSettings(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save two-factor settings"})
		return
	}
	// Any previous enrollment marker for this channel is cleared: a stale marker
	// must not vouch for a number the user has just replaced.
	services.ClearOTPVerified(c.Request.Context(), services.PurposeMFAEnroll, userID.String(), verified.E164)

	if err := database.DB.Model(&models.UserMFASettings{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"phone_enrolled": true,
			"phone_e164_enc": models.EncryptedString(verified.E164),
			// A _bidx column holds a KEYED HASH, never the value. This wrote the
			// normalized number in the clear, one column over from the ciphertext
			// it was supposed to make unnecessary — so encrypting phone_e164_enc
			// bought nothing for any user who enrolled a phone (#925).
			//
			// Normalize first, then hash: BlindIndex only lowercases and trims, so
			// hashing an unnormalized number would make "+91 98…" and "+9198…"
			// index differently and never match each other.
			"phone_e164_bidx": piicrypto.BlindIndex(services.NormalizeE164(verified.E164)),
			"updated_at":      time.Now(),
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save two-factor settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"phoneEnrolled": true, "maskedPhone": services.MaskPhone(verified.E164)})
}

// ---- enable / disable -------------------------------------------------------

// Enable arms two-factor and issues the backup codes.
//
// The codes are returned here and nowhere else, and enabling is refused unless a
// channel is already enrolled — arming the gate with nothing to challenge on
// would lock the user out of their own account.
func (h *MFAHandler) Enable(c *gin.Context) {
	userID, _, ok := currentUser(c)
	if !ok {
		return
	}
	s, err := services.GetMFASettings(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load two-factor settings"})
		return
	}
	if len(services.EnrolledChannels(s)) == 0 {
		c.JSON(http.StatusPreconditionRequired, gin.H{
			"error": "Set up email or phone verification before turning two-factor on.",
		})
		return
	}
	if backupCodeKey() == "" {
		// Arming without a recovery path is how people lose accounts.
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Two-factor is not fully configured on the server. Please try later.",
		})
		return
	}

	codes, err := services.GenerateBackupCodes(database.DB, userID, backupCodeKey())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate backup codes"})
		return
	}
	now := time.Now()
	if err := database.DB.Model(&models.UserMFASettings{}).Where("user_id = ?", userID).
		Updates(map[string]any{"enabled": true, "enrolled_at": now, "updated_at": now}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not enable two-factor"})
		return
	}

	services.PublishMFAEvent(services.SubjectMFAEnabled, userID)
	c.JSON(http.StatusOK, gin.H{"enabled": true, "backupCodes": codes})
}

// Disable turns two-factor off, revoking trusted devices and burning unused
// backup codes with it.
func (h *MFAHandler) Disable(c *gin.Context) {
	userID, _, ok := currentUser(c)
	if !ok {
		return
	}
	if err := services.DisableMFA(database.DB, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not disable two-factor"})
		return
	}
	services.PublishMFAEvent(services.SubjectMFADisabled, userID)
	c.JSON(http.StatusOK, gin.H{"enabled": false})
}

// RegenerateBackupCodes replaces the current sheet. The old codes stop working.
func (h *MFAHandler) RegenerateBackupCodes(c *gin.Context) {
	userID, _, ok := currentUser(c)
	if !ok {
		return
	}
	if backupCodeKey() == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Two-factor is not fully configured on the server."})
		return
	}
	codes, err := services.GenerateBackupCodes(database.DB, userID, backupCodeKey())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate backup codes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"backupCodes": codes})
}

// ---- login challenge --------------------------------------------------------

// challengeDevice scopes a login challenge to the handset that raised it.
//
// Without it, two devices on one account share a single code: the second to ask
// overwrites the first's code, and either is told to wait out the other's resend
// cooldown. The device that cannot get in then looks dead rather than
// challenged (#1164 finding 3). Empty for clients predating the header, which
// keeps the old account-wide behaviour for them.
//
// Scoped alongside the app, not by it alone, so two installs of the same app
// still get their own challenge.
func challengeDevice(c *gin.Context) string {
	deviceID := strings.TrimSpace(c.GetHeader(HdrDeviceID))
	if deviceID == "" {
		return ""
	}
	return middleware.ClientAppFrom(c) + ":" + deviceID
}

// Challenge sends a login code on the requested channel.
func (h *MFAHandler) Challenge(c *gin.Context) {
	userID, user, ok := currentUser(c)
	if !ok {
		return
	}
	var req mfaChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pick a channel", "field": "channel"})
		return
	}
	s, err := services.GetMFASettings(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load two-factor settings"})
		return
	}
	channel := services.MFAChannel(req.Channel)
	subject, masked, err := services.ChallengeSubject(s, user.Email, channel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "field": "channel"})
		return
	}

	if channel == services.MFAChannelPhone {
		// Firebase owns SMS delivery and verification for the phone leg; the
		// client drives it and posts the resulting credential to Verify. There is
		// no server-issued code to send here.
		c.JSON(http.StatusOK, gin.H{
			"channel": channel, "masked": masked, "delivery": "firebase",
		})
		return
	}

	code, err := services.IssueOTP(c.Request.Context(), services.PurposeMFALogin, userID.String(), subject, challengeDevice(c))
	if err != nil {
		c.JSON(otpStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := services.GetEmailService().SendEmailOTP(subject, user.FirstName, code); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "We could not send the code. Please try again."})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"channel": channel, "masked": masked, "delivery": "email", "expiresInSeconds": 600,
	})
}

// Verify completes the challenge and hands back the token the client presents on
// later requests — a persistent device token when the user asked to be
// remembered, a session-scoped elevation token otherwise.
func (h *MFAHandler) Verify(c *gin.Context) {
	userID, user, ok := currentUser(c)
	if !ok {
		return
	}
	var req mfaVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter the code"})
		return
	}
	s, err := services.GetMFASettings(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load two-factor settings"})
		return
	}

	switch {
	case req.BackupCode != "":
		if backupCodeKey() == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Two-factor is not fully configured on the server."})
			return
		}
		remaining, err := services.RedeemBackupCode(database.DB, userID, req.BackupCode, backupCodeKey())
		if err != nil {
			c.JSON(mfaVerifyStatus(err), gin.H{"error": "That code is not valid.", "field": "backupCode"})
			return
		}
		if remaining <= services.LowBackupCodeThreshold {
			services.PublishMFAEvent(services.SubjectMFABackupCodesLow, userID)
		}

	case services.MFAChannel(req.Channel) == services.MFAChannelPhone:
		if !s.PhoneEnrolled {
			c.JSON(http.StatusBadRequest, gin.H{"error": services.ErrMFAChannelNotEnrolled.Error(), "field": "channel"})
			return
		}
		verified, err := services.VerifyFirebasePhoneToken(
			c.Request.Context(), req.FirebaseIDToken, user.GIPUid,
		)
		if err != nil {
			c.JSON(phoneVerifyStatus(err), gin.H{"error": err.Error(), "field": "firebaseIdToken"})
			return
		}
		// The number Google just verified must be the one enrolled as this
		// account's factor. Skipping this would let anyone verify any phone they
		// hold and walk past the challenge with it.
		if verified.E164 != services.NormalizeE164(string(s.PhoneE164Enc)) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "That is not the phone number registered for this account.",
				"field": "firebaseIdToken",
			})
			return
		}

	default:
		subject, _, err := services.ChallengeSubject(s, user.Email, services.MFAChannelEmail)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "field": "channel"})
			return
		}
		if err := services.RedeemOTP(c.Request.Context(), services.PurposeMFALogin, userID.String(), subject, challengeDevice(c), req.Code); err != nil {
			c.JSON(otpStatus(err), gin.H{"error": err.Error(), "field": "code"})
			return
		}
	}

	app := middleware.ClientAppFrom(c)
	if req.RememberDevice {
		token, err := services.IssueTrustedDevice(database.DB, userID, app, req.DeviceLabel, req.Platform)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not remember this device"})
			return
		}
		services.PublishMFAEvent(services.SubjectMFADeviceTrusted, userID)
		c.JSON(http.StatusOK, gin.H{"verified": true, "deviceToken": token, "remembered": true})
		return
	}

	token, err := services.ElevateSession(c.Request.Context(), userID, app)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Could not complete sign-in. Please try again."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"verified": true, "deviceToken": token, "remembered": false})
}

// ---- trusted devices --------------------------------------------------------

// ListDevices returns the caller's remembered devices.
func (h *MFAHandler) ListDevices(c *gin.Context) {
	userID, _, ok := currentUser(c)
	if !ok {
		return
	}
	// Same short-circuit as GetStatus. The shared settings screen calls this
	// from a hook, which React runs unconditionally — so with the feature off
	// and the tables not yet deployed, every visit logged a background 500 for
	// a list that is empty by definition.
	if !mfaEnabled() {
		c.JSON(http.StatusOK, gin.H{"devices": []models.TrustedDevice{}})
		return
	}
	devices, err := services.ListTrustedDevices(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load devices"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices": devices})
}

// RevokeDevice drops one remembered device.
func (h *MFAHandler) RevokeDevice(c *gin.Context) {
	userID, _, ok := currentUser(c)
	if !ok {
		return
	}
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown device"})
		return
	}
	if err := services.RevokeTrustedDevice(database.DB, userID, deviceID); err != nil {
		if errors.Is(err, services.ErrDeviceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Unknown device"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not revoke device"})
		return
	}
	services.PublishMFAEvent(services.SubjectMFADeviceRevoked, userID)
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}

// RevokeAllDevices is the "sign out everywhere" action.
func (h *MFAHandler) RevokeAllDevices(c *gin.Context) {
	userID, _, ok := currentUser(c)
	if !ok {
		return
	}
	n, err := services.RevokeAllTrustedDevices(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not revoke devices"})
		return
	}
	services.PublishMFAEvent(services.SubjectMFADeviceRevoked, userID)
	c.JSON(http.StatusOK, gin.H{"revoked": n})
}

func mfaVerifyStatus(err error) int {
	switch {
	case errors.Is(err, services.ErrBackupCodeInvalid), errors.Is(err, services.ErrBackupCodesMissing):
		return http.StatusBadRequest
	case errors.Is(err, gorm.ErrRecordNotFound):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func phoneVerifyStatus(err error) int {
	switch {
	case errors.Is(err, services.ErrPhoneVerifyUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, services.ErrPhoneTokenWrongUser):
		return http.StatusForbidden
	default:
		// Invalid token and phone-less token both read as a bad request; keeping
		// them indistinguishable from each other gives a prober nothing.
		return http.StatusBadRequest
	}
}
