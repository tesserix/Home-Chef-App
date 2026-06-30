package services

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// platformPolicyKey is the PlatformSettings row that holds the whole
// commerce-side policy as a single JSON blob. Same pattern as SecurityPolicy
// so updates are atomic and a single lookup gets the whole thing.
const platformPolicyKey = "platform_policy"

// PlatformPolicy is the live commerce configuration the order and checkout
// flows read on every request. Admins edit this from Settings → Platform.
type PlatformPolicy struct {
	// Fees applied at order creation. Values are plain floats (percent is 0-100,
	// fee is absolute). Chef payout = subtotal + chefTip + (subtotal * chefPayoutPercent / 100 - serviceFee).
	ServiceFeePercent   float64 `json:"serviceFeePercent"`
	TaxPercent          float64 `json:"taxPercent"`
	BaseDeliveryFee     float64 `json:"baseDeliveryFee"`
	PerKmDeliveryFee    float64 `json:"perKmDeliveryFee"`
	ChefPayoutPercent   float64 `json:"chefPayoutPercent"`   // of subtotal
	DriverPayoutPercent float64 `json:"driverPayoutPercent"` // of deliveryFee

	// Operating hours enforced at checkout. "" / 0 disables the check.
	Timezone      string `json:"timezone"`      // IANA, e.g. "Asia/Kolkata"
	OpeningTime   string `json:"openingTime"`   // "HH:MM" 24h
	ClosingTime   string `json:"closingTime"`   // "HH:MM" 24h
	OperatingDays []int  `json:"operatingDays"` // 0=Sunday..6=Saturday; empty = all days
	ClosedMessage string `json:"closedMessage"` // shown at checkout when closed

	// Feature flags — runtime overrides for env-based gates. When true the
	// feature is on even if its env var is off, so prod can toggle it from the
	// admin console without a redeploy. The gate is `env OR this`, so the env
	// default-on still works and this only ever turns a feature MORE on.
	GroupOrdersEnabled bool `json:"groupOrdersEnabled"` // group/office ("corporate") orders
}

// DefaultPlatformPolicy matches what was hardcoded in handlers/orders.go
// before this policy existed, so upgrading is a no-op for existing traffic.
func DefaultPlatformPolicy() PlatformPolicy {
	return PlatformPolicy{
		ServiceFeePercent:   10.0,
		TaxPercent:          8.0,
		BaseDeliveryFee:     2.99,
		PerKmDeliveryFee:    0.0,
		ChefPayoutPercent:   80.0,
		DriverPayoutPercent: 80.0,
		Timezone:            "Asia/Kolkata",
		OpeningTime:         "",
		ClosingTime:         "",
		OperatingDays:       nil,
		ClosedMessage:       "We're currently closed. Please come back during our operating hours.",
		// Default off here — the env var (GROUP_ORDERS_ENABLED) provides the
		// baseline; this policy field only adds a runtime override on top.
		GroupOrdersEnabled: false,
	}
}

var (
	platformPolicyCache     *PlatformPolicy
	platformPolicyFetchedAt time.Time
	platformPolicyMu        sync.RWMutex
)

// GetPlatformPolicy returns the active policy with a 5-min TTL cache. Admin
// writes call InvalidatePlatformPolicy so changes surface immediately.
func GetPlatformPolicy() PlatformPolicy {
	platformPolicyMu.RLock()
	if platformPolicyCache != nil && time.Since(platformPolicyFetchedAt) < platformConfigTTL {
		defer platformPolicyMu.RUnlock()
		return *platformPolicyCache
	}
	platformPolicyMu.RUnlock()

	platformPolicyMu.Lock()
	defer platformPolicyMu.Unlock()
	if platformPolicyCache != nil && time.Since(platformPolicyFetchedAt) < platformConfigTTL {
		return *platformPolicyCache
	}

	fresh := loadPlatformPolicyFromDB()
	platformPolicyCache = &fresh
	platformPolicyFetchedAt = time.Now()
	return fresh
}

// InvalidatePlatformPolicy drops the cache so the next read refetches from DB.
func InvalidatePlatformPolicy() {
	platformPolicyMu.Lock()
	defer platformPolicyMu.Unlock()
	platformPolicyCache = nil
}

// SavePlatformPolicy upserts the policy as a single JSON blob and invalidates
// the cache so the next call to GetPlatformPolicy() sees the new values.
func SavePlatformPolicy(p PlatformPolicy, updatedBy *uuid.UUID) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}

	var setting models.PlatformSettings
	err = database.DB.Where("key = ?", platformPolicyKey).First(&setting).Error
	if err != nil {
		setting = models.PlatformSettings{
			Key:       platformPolicyKey,
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

	InvalidatePlatformPolicy()
	return nil
}

// IsPlatformOpen evaluates OperatingDays + OpeningTime/ClosingTime in the
// configured timezone. Returns (true, "") when open or unconfigured.
// Returns (false, closedMessage) when outside hours so the caller can
// surface a helpful message to the user.
func IsPlatformOpen() (bool, string) {
	p := GetPlatformPolicy()

	// Empty opening/closing = unconfigured → always open.
	if p.OpeningTime == "" || p.ClosingTime == "" {
		return true, ""
	}

	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		log.Printf("platform_policy: invalid timezone %q, treating as open: %v", p.Timezone, err)
		return true, ""
	}
	now := time.Now().In(loc)

	// Day-of-week check (0=Sunday..6=Saturday). Empty list = all days.
	if len(p.OperatingDays) > 0 {
		ok := false
		today := int(now.Weekday())
		for _, d := range p.OperatingDays {
			if d == today {
				ok = true
				break
			}
		}
		if !ok {
			return false, p.ClosedMessage
		}
	}

	open, err := parseHHMM(p.OpeningTime)
	if err != nil {
		log.Printf("platform_policy: bad openingTime %q: %v", p.OpeningTime, err)
		return true, ""
	}
	closeT, err := parseHHMM(p.ClosingTime)
	if err != nil {
		log.Printf("platform_policy: bad closingTime %q: %v", p.ClosingTime, err)
		return true, ""
	}

	nowMinutes := now.Hour()*60 + now.Minute()
	openMinutes := open.h*60 + open.m
	closeMinutes := closeT.h*60 + closeT.m

	// Identical open/close is almost certainly an admin mistake. Treating
	// it as an overnight window would keep the platform open 24/7, which
	// is the exact opposite of "closed" most operators expect. Fail closed.
	if openMinutes == closeMinutes {
		return false, p.ClosedMessage
	}
	// Overnight window (e.g. 22:00 → 02:00) spans midnight.
	if closeMinutes < openMinutes {
		if nowMinutes >= openMinutes || nowMinutes < closeMinutes {
			return true, ""
		}
		return false, p.ClosedMessage
	}
	if nowMinutes >= openMinutes && nowMinutes < closeMinutes {
		return true, ""
	}
	return false, p.ClosedMessage
}

type hhmm struct{ h, m int }

func parseHHMM(s string) (hhmm, error) {
	var out hhmm
	t, err := time.Parse("15:04", s)
	if err != nil {
		return out, err
	}
	out.h = t.Hour()
	out.m = t.Minute()
	return out, nil
}

func loadPlatformPolicyFromDB() PlatformPolicy {
	def := DefaultPlatformPolicy()
	var setting models.PlatformSettings
	err := database.DB.Where("key = ?", platformPolicyKey).First(&setting).Error
	if err != nil {
		// Only ErrRecordNotFound should silently fall through to defaults;
		// real DB errors (connection loss, etc.) deserve at least a log.
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("platform_policy: db read failed, using defaults: %v", err)
		}
		return def
	}
	if setting.Value == "" {
		return def
	}

	// Decode presence-aware: an admin who legitimately sets
	// serviceFeePercent=0 (promo) or taxPercent=0 (tax-exempt) must not
	// have the value silently replaced by the default. We use a pointer
	// struct to distinguish "missing" from "zero", then merge onto defaults.
	type partial struct {
		ServiceFeePercent   *float64 `json:"serviceFeePercent"`
		TaxPercent          *float64 `json:"taxPercent"`
		BaseDeliveryFee     *float64 `json:"baseDeliveryFee"`
		PerKmDeliveryFee    *float64 `json:"perKmDeliveryFee"`
		ChefPayoutPercent   *float64 `json:"chefPayoutPercent"`
		DriverPayoutPercent *float64 `json:"driverPayoutPercent"`
		Timezone            *string  `json:"timezone"`
		OpeningTime         *string  `json:"openingTime"`
		ClosingTime         *string  `json:"closingTime"`
		OperatingDays       *[]int   `json:"operatingDays"`
		ClosedMessage       *string  `json:"closedMessage"`
		GroupOrdersEnabled  *bool    `json:"groupOrdersEnabled"`
	}
	var p partial
	if err := json.Unmarshal([]byte(setting.Value), &p); err != nil {
		log.Printf("platform_policy: failed to parse, using defaults: %v", err)
		return def
	}
	out := def
	if p.ServiceFeePercent != nil {
		out.ServiceFeePercent = *p.ServiceFeePercent
	}
	if p.TaxPercent != nil {
		out.TaxPercent = *p.TaxPercent
	}
	if p.BaseDeliveryFee != nil {
		out.BaseDeliveryFee = *p.BaseDeliveryFee
	}
	if p.PerKmDeliveryFee != nil {
		out.PerKmDeliveryFee = *p.PerKmDeliveryFee
	}
	if p.ChefPayoutPercent != nil {
		out.ChefPayoutPercent = *p.ChefPayoutPercent
	}
	if p.DriverPayoutPercent != nil {
		out.DriverPayoutPercent = *p.DriverPayoutPercent
	}
	if p.Timezone != nil && *p.Timezone != "" {
		out.Timezone = *p.Timezone
	}
	if p.OpeningTime != nil {
		out.OpeningTime = *p.OpeningTime
	}
	if p.ClosingTime != nil {
		out.ClosingTime = *p.ClosingTime
	}
	if p.OperatingDays != nil {
		out.OperatingDays = *p.OperatingDays
	}
	if p.ClosedMessage != nil && *p.ClosedMessage != "" {
		out.ClosedMessage = *p.ClosedMessage
	}
	if p.GroupOrdersEnabled != nil {
		out.GroupOrdersEnabled = *p.GroupOrdersEnabled
	}
	return out
}
