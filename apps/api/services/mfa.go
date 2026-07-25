package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// Two-factor orchestration: who is challenged, on which channel, and what
// counts as having passed.
//
// The gate (middleware/mfa_gate.go) asks exactly one question — may this
// request through? — and everything behind that question lives here.

var (
	ErrMFANotEnrolled        = errors.New("two-factor is not set up")
	ErrMFAChannelUnknown     = errors.New("unknown channel")
	ErrMFAChannelNotEnrolled = errors.New("that channel is not set up")
)

// MFAChannel is a delivery route for a challenge.
type MFAChannel string

// Prefixed to stay clear of the notification-preference Channel constants,
// which already own the bare ChannelEmail name in this package.
const (
	MFAChannelEmail MFAChannel = "email"
	MFAChannelPhone MFAChannel = "phone"
)

// elevationTTL bounds a session that passed the challenge but declined to be
// remembered. Long enough to cover a working day, short enough that a shared
// browser does not stay elevated indefinitely.
const elevationTTL = 12 * time.Hour

func elevationKey(userID uuid.UUID, app, token string) string {
	return fmt.Sprintf("mfa:elev:%s:%s:%s", userID, app, token)
}

// GetMFASettings loads a user's two-factor state. A missing row is not an
// error — it means the user has never touched the feature, which reads the same
// as disabled. Returning a zero-valued struct keeps every caller from having to
// special-case gorm.ErrRecordNotFound.
func GetMFASettings(db *gorm.DB, userID uuid.UUID) (models.UserMFASettings, error) {
	var s models.UserMFASettings
	err := db.Where("user_id = ?", userID).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.UserMFASettings{UserID: userID}, nil
	}
	if err != nil {
		return models.UserMFASettings{}, err
	}
	return s, nil
}

// EnrolledChannels lists the channels a user can actually be challenged on.
func EnrolledChannels(s models.UserMFASettings) []MFAChannel {
	var out []MFAChannel
	if s.EmailEnrolled {
		out = append(out, MFAChannelEmail)
	}
	if s.PhoneEnrolled {
		out = append(out, MFAChannelPhone)
	}
	return out
}

// MaskEmail renders an address as a hint the owner recognises but a stranger
// cannot use: "samyak@gmail.com" → "s•••k@gmail.com".
//
// The challenge response has to tell the user where the code went, but it is
// served to whoever holds the password — so it must not hand over a full
// address to someone probing an account.
func MaskEmail(email string) string {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "•••"
	}
	local, domain := email[:at], email[at:]
	switch {
	case len(local) <= 2:
		return strings.Repeat("•", len(local)) + domain
	default:
		return string(local[0]) + "•••" + string(local[len(local)-1]) + domain
	}
}

// MaskPhone keeps the last two digits only: "+919876543210" → "+91•••••••10".
func MaskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	digits := []rune(phone)
	if len(digits) <= 4 {
		return "•••"
	}
	keepFront := 3
	if strings.HasPrefix(phone, "+") {
		keepFront = 3
	}
	if keepFront > len(digits)-2 {
		keepFront = 0
	}
	return string(digits[:keepFront]) + strings.Repeat("•", len(digits)-keepFront-2) + string(digits[len(digits)-2:])
}

// MFARequirement is what the gate needs in order to answer a request.
type MFARequirement struct {
	// Required is true when this request must be refused until challenged.
	Required bool
	// Channels the user may be challenged on, for the client to offer.
	Channels []MFAChannel
}

// EvaluateMFA decides whether a request must be challenged.
//
// Order matters: the cheap checks come first so the common path (feature off,
// or user has not enrolled) costs nothing, and the database is only touched
// once the flag is on.
func EvaluateMFA(
	ctx context.Context,
	db *gorm.DB,
	enabled bool,
	userID uuid.UUID,
	app, deviceToken string,
) (MFARequirement, error) {
	if !enabled {
		return MFARequirement{}, nil
	}
	s, err := GetMFASettings(db, userID)
	if err != nil {
		return MFARequirement{}, err
	}
	if !s.Enabled {
		return MFARequirement{}, nil
	}
	channels := EnrolledChannels(s)
	if len(channels) == 0 {
		// Enabled with nothing to challenge on would lock the user out of their
		// own account with no way back. Treat it as not-required and let the
		// settings screen repair it.
		return MFARequirement{}, nil
	}
	// A remembered device, or a session that already passed, goes straight
	// through. Both are checked against the same presented token.
	if deviceToken != "" {
		if IsDeviceTrusted(db, userID, app, deviceToken) {
			return MFARequirement{}, nil
		}
		if IsSessionElevated(ctx, userID, app, deviceToken) {
			return MFARequirement{}, nil
		}
	}
	return MFARequirement{Required: true, Channels: channels}, nil
}

// ElevateSession marks this login as having passed the challenge without
// remembering the device. Returns an opaque token the client presents on
// subsequent requests, exactly like a device token but expiring.
func ElevateSession(ctx context.Context, userID uuid.UUID, app string) (string, error) {
	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		return "", ErrOTPUnavailable
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	if err := r.Set(ctx, elevationKey(userID, app, token), "1", elevationTTL); err != nil {
		return "", ErrOTPUnavailable
	}
	return token, nil
}

// IsSessionElevated reports whether this token still represents a passed
// challenge. Fails closed on a Redis outage, for the same reason OTPVerified
// does: a cache blip must not become a way past the second factor.
func IsSessionElevated(ctx context.Context, userID uuid.UUID, app, token string) bool {
	if token == "" {
		return false
	}
	r := GetRedisClient()
	if r == nil || !r.IsConnected() {
		return false
	}
	v, err := r.Get(ctx, elevationKey(userID, app, token))
	return err == nil && v == "1"
}

// ChallengeSubject returns the address or number a challenge on this channel
// would be delivered to, along with its masked form for display.
func ChallengeSubject(s models.UserMFASettings, userEmail string, ch MFAChannel) (subject, masked string, err error) {
	switch ch {
	case MFAChannelEmail:
		if !s.EmailEnrolled {
			return "", "", ErrMFAChannelNotEnrolled
		}
		return NormalizeEmail(userEmail), MaskEmail(userEmail), nil
	case MFAChannelPhone:
		if !s.PhoneEnrolled {
			return "", "", ErrMFAChannelNotEnrolled
		}
		phone := string(s.PhoneE164Enc)
		return phone, MaskPhone(phone), nil
	default:
		return "", "", ErrMFAChannelUnknown
	}
}

// DisableMFA switches two-factor off and clears everything that depended on it.
//
// Revoking trusted devices and burning unused backup codes is the point: a user
// turning 2FA off then on again must not inherit trust granted under the old
// setup, and a leaked backup sheet must not survive the reset.
func DisableMFA(db *gorm.DB, userID uuid.UUID) error {
	now := time.Now()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.UserMFASettings{}).
			Where("user_id = ?", userID).
			Updates(map[string]any{
				"enabled":        false,
				"email_enrolled": false,
				"phone_enrolled": false,
				"disabled_at":    now,
				"updated_at":     now,
			}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.MFABackupCode{}).Error; err != nil {
			return err
		}
		return tx.Model(&models.TrustedDevice{}).
			Where("user_id = ? AND revoked_at IS NULL", userID).
			Update("revoked_at", now).Error
	})
}
