package services

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// platformConfigTTL controls how long a fetched SecurityPolicy stays cached.
// Admin writes call InvalidateSecurityPolicy() so changes surface immediately;
// the TTL exists only so the happy-path read doesn't hit the DB every request.
const platformConfigTTL = 5 * time.Minute

// securityPolicyKey is the PlatformSettings row we store the policy blob in.
const securityPolicyKey = "security_policy"

// SecurityPolicy is the platform-wide security configuration. It's persisted
// as a single JSON blob in the PlatformSettings table so updates are atomic
// and the whole policy is one round-trip to fetch.
type SecurityPolicy struct {
	// Password rules — enforced at register / change-password / reset-password
	// (for paths that hit our API directly; BFF/Keycloak logins enforce their own).
	PasswordMinLength      int  `json:"passwordMinLength"`
	PasswordRequireUpper   bool `json:"passwordRequireUpper"`
	PasswordRequireLower   bool `json:"passwordRequireLower"`
	PasswordRequireNumber  bool `json:"passwordRequireNumber"`
	PasswordRequireSpecial bool `json:"passwordRequireSpecial"`

	// Session lifetimes — consumed by GenerateTokens.
	SessionAccessTTLHours int `json:"sessionAccessTtlHours"`
	SessionRefreshTTLDays int `json:"sessionRefreshTtlDays"`

	// 2FA enforcement — when true, admins without TOTP enrolled are
	// forced through setup before they can use the admin portal.
	TwoFactorRequiredForAdmins bool `json:"twoFactorRequiredForAdmins"`

	// Emails exempt from the forced-2FA flow even when the policy is on.
	// Kept for service accounts (E2E tests, automation) that can't scan a QR.
	// Match is case-insensitive.
	TwoFactorExemptEmails []string `json:"twoFactorExemptEmails,omitempty"`
}

// DefaultSecurityPolicy returns sensible built-in defaults so the app is
// always usable, even before an admin configures the policy.
func DefaultSecurityPolicy() SecurityPolicy {
	return SecurityPolicy{
		PasswordMinLength:          8,
		PasswordRequireUpper:       true,
		PasswordRequireLower:       true,
		PasswordRequireNumber:      true,
		PasswordRequireSpecial:     false,
		SessionAccessTTLHours:      24,
		SessionRefreshTTLDays:      30,
		TwoFactorRequiredForAdmins: false,
		// E2E service accounts ship exempt by default so automated tests
		// don't get forced through a QR-scan flow they can't complete.
		// Any real admin added to this list bypasses 2FA enforcement too,
		// so admins should only extend it for non-interactive accounts.
		TwoFactorExemptEmails: []string{
			"e2e-admin@fe3dr.com",
			"e2e-test@fe3dr.com",
		},
	}
}

// IsTwoFactorExempt reports whether an email should skip forced 2FA
// enrollment even when the platform policy requires it. Case-insensitive.
func (p SecurityPolicy) IsTwoFactorExempt(email string) bool {
	for _, e := range p.TwoFactorExemptEmails {
		if strings.EqualFold(strings.TrimSpace(e), strings.TrimSpace(email)) {
			return true
		}
	}
	return false
}

var (
	policyCache     *SecurityPolicy
	policyFetchedAt time.Time
	policyMu        sync.RWMutex
)

// GetSecurityPolicy returns the current platform security policy, cached for
// platformConfigTTL. If the DB row is missing or malformed we fall back to
// DefaultSecurityPolicy() so the policy is always a valid value.
func GetSecurityPolicy() SecurityPolicy {
	policyMu.RLock()
	if policyCache != nil && time.Since(policyFetchedAt) < platformConfigTTL {
		defer policyMu.RUnlock()
		return *policyCache
	}
	policyMu.RUnlock()

	policyMu.Lock()
	defer policyMu.Unlock()
	// Double-check after acquiring the write lock.
	if policyCache != nil && time.Since(policyFetchedAt) < platformConfigTTL {
		return *policyCache
	}

	fresh := loadSecurityPolicyFromDB()
	policyCache = &fresh
	policyFetchedAt = time.Now()
	return fresh
}

// InvalidateSecurityPolicy drops the cache so the next read refetches from DB.
// Call this after writing the policy via the admin API.
func InvalidateSecurityPolicy() {
	policyMu.Lock()
	defer policyMu.Unlock()
	policyCache = nil
}

// SaveSecurityPolicy persists the policy as a single JSON blob in
// PlatformSettings under securityPolicyKey and invalidates the cache.
func SaveSecurityPolicy(p SecurityPolicy, updatedBy *uuid.UUID) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}

	var setting models.PlatformSettings
	err = database.DB.Where("key = ?", securityPolicyKey).First(&setting).Error
	if err != nil {
		setting = models.PlatformSettings{
			Key:       securityPolicyKey,
			Value:     string(raw),
			Type:      "json",
			UpdatedBy: updatedBy,
		}
		if err := database.DB.Create(&setting).Error; err != nil {
			return err
		}
	} else {
		setting.Value = string(raw)
		setting.Type = "json"
		setting.UpdatedBy = updatedBy
		if err := database.DB.Save(&setting).Error; err != nil {
			return err
		}
	}

	InvalidateSecurityPolicy()
	return nil
}

func loadSecurityPolicyFromDB() SecurityPolicy {
	def := DefaultSecurityPolicy()
	var setting models.PlatformSettings
	if err := database.DB.Where("key = ?", securityPolicyKey).First(&setting).Error; err != nil {
		return def
	}
	if setting.Value == "" {
		return def
	}
	var parsed SecurityPolicy
	if err := json.Unmarshal([]byte(setting.Value), &parsed); err != nil {
		log.Printf("platform_config: failed to parse security policy, using defaults: %v", err)
		return def
	}
	// Backfill any zero-valued fields with defaults so partially-stored
	// policies don't accidentally disable a rule.
	if parsed.PasswordMinLength == 0 {
		parsed.PasswordMinLength = def.PasswordMinLength
	}
	if parsed.SessionAccessTTLHours == 0 {
		parsed.SessionAccessTTLHours = def.SessionAccessTTLHours
	}
	if parsed.SessionRefreshTTLDays == 0 {
		parsed.SessionRefreshTTLDays = def.SessionRefreshTTLDays
	}
	return parsed
}
