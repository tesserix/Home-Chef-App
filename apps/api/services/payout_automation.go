package services

import (
	"strconv"

	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// payout_automation.go — resolving whether a chef's payouts auto-release
// (#741).
//
// Three controls with distinct roles, and the precedence between them is the
// safety property:
//
//  1. payout.sweep_enabled is an absolute kill switch. Off means nothing
//     releases for anyone; a per-chef "on" does not override it. This is the
//     control reached for when something is going wrong with live money, so
//     it cannot have exceptions.
//  2. The chef's tri-state decides next.
//  3. payout.auto_release_default fills in for an unset chef.
//
// Everything fails closed: a missing or unparseable setting means no
// automation, because the failure mode of over-releasing is money that has
// left the platform.

// PlatformSettings keys.
const (
	payoutSweepEnabledKey       = "payout.sweep_enabled"
	payoutAutoReleaseDefaultKey = "payout.auto_release_default"
)

// Per-chef switch values. Empty string means follow the default.
const (
	PayoutAutoOn  = "on"
	PayoutAutoOff = "off"
)

// settingValue reads a raw platform setting, empty when absent.
func settingValue(db *gorm.DB, key string) string {
	var setting models.PlatformSettings
	if err := db.Where("key = ?", key).First(&setting).Error; err != nil {
		return ""
	}
	return setting.Value
}

// settingBool parses a boolean setting, defaulting to false on anything
// unparseable so a typo cannot switch money movement on.
func settingBool(db *gorm.DB, key string) bool {
	v, err := strconv.ParseBool(settingValue(db, key))
	return err == nil && v
}

// PayoutAutomationEnabled reports whether this chef's matured payouts may
// release without a human.
//
// It grants candidacy only — the guardrail chain still runs. A chef switched
// on with an open refund is still blocked.
func PayoutAutomationEnabled(db *gorm.DB, chef *models.ChefProfile) bool {
	if !settingBool(db, payoutSweepEnabledKey) {
		return false
	}
	if chef == nil {
		return false
	}
	switch chef.PayoutAutoRelease {
	case PayoutAutoOn:
		return true
	case PayoutAutoOff:
		return false
	default:
		return settingValue(db, payoutAutoReleaseDefaultKey) == PayoutAutoOn
	}
}
