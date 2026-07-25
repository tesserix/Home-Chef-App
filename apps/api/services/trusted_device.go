package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// Trusted devices — the "remember this device" half of two-factor.
//
// The token is a bearer credential: whoever holds it skips the challenge on
// that app. So it is generated from crypto/rand, handed to the client exactly
// once, and only its SHA-256 lands in the database. A dump of trusted_devices
// yields nothing usable.
//
// No HMAC or salt here, deliberately: the token is 256 bits of uniform
// randomness, so it is not guessable and there is no dictionary to defend
// against. A plain hash is the right tool; a KDF would only add latency to
// every request.

var (
	ErrDeviceTokenInvalid = errors.New("unknown device")
	ErrDeviceNotFound     = errors.New("device not found")
)

const trustedDeviceTokenBytes = 32

// ValidMFAApps are the app scopes a device token can be issued for.
var ValidMFAApps = map[string]bool{
	"customer": true,
	"vendor":   true,
	"delivery": true,
	"admin":    true,
}

func hashDeviceToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// IssueTrustedDevice mints a token for (user, app) and returns the plaintext.
// The caller must return it to the client immediately; it cannot be recovered.
func IssueTrustedDevice(db *gorm.DB, userID uuid.UUID, app, label, platform string) (string, error) {
	if !ValidMFAApps[app] {
		return "", errors.New("unknown app scope")
	}
	raw := make([]byte, trustedDeviceTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	now := time.Now()
	rec := models.TrustedDevice{
		ID:         uuid.New(),
		UserID:     userID,
		App:        app,
		TokenHash:  hashDeviceToken(token),
		Label:      strings.TrimSpace(label),
		Platform:   strings.TrimSpace(platform),
		CreatedAt:  now,
		LastSeenAt: now,
	}
	if err := db.Create(&rec).Error; err != nil {
		return "", err
	}
	return token, nil
}

// IsDeviceTrusted reports whether this token still lets the user skip the
// challenge on this app, and refreshes last_seen_at when it does.
//
// The lookup is scoped by user AND app, so a token that is genuinely valid for
// one app cannot be replayed against another, and a token belonging to a
// different user cannot be replayed at all.
func IsDeviceTrusted(db *gorm.DB, userID uuid.UUID, app, token string) bool {
	if token == "" || !ValidMFAApps[app] {
		return false
	}
	var rec models.TrustedDevice
	err := db.Where(
		"user_id = ? AND app = ? AND token_hash = ? AND revoked_at IS NULL",
		userID, app, hashDeviceToken(token),
	).First(&rec).Error
	if err != nil {
		return false
	}
	// Best-effort: a failed touch must not cost the user their trusted device.
	db.Model(&models.TrustedDevice{}).Where("id = ?", rec.ID).
		Update("last_seen_at", time.Now())
	return true
}

// ListTrustedDevices returns the user's active devices, newest first, for the
// settings screen where they revoke them.
func ListTrustedDevices(db *gorm.DB, userID uuid.UUID) ([]models.TrustedDevice, error) {
	var out []models.TrustedDevice
	err := db.Where("user_id = ? AND revoked_at IS NULL", userID).
		Order("last_seen_at DESC").Find(&out).Error
	return out, err
}

// RevokeTrustedDevice revokes one device the user owns. Scoping the update by
// user_id means a guessed device ID from another account is a no-op.
func RevokeTrustedDevice(db *gorm.DB, userID, deviceID uuid.UUID) error {
	res := db.Model(&models.TrustedDevice{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", deviceID, userID).
		Update("revoked_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

// RevokeAllTrustedDevices drops every remembered device for a user. Called on
// password change, on two-factor disable, on admin reset, and from the user's
// own "sign out everywhere" — the events that mean previously-granted trust
// should no longer stand.
func RevokeAllTrustedDevices(db *gorm.DB, userID uuid.UUID) (int64, error) {
	res := db.Model(&models.TrustedDevice{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now())
	return res.RowsAffected, res.Error
}
