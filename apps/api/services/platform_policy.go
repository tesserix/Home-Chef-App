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
	// fee is absolute). Chef payout = subtotal + chefTip + (subtotal * chefPayoutPercent / 100 - platformFee).
	PlatformFeePercent  float64 `json:"platformFeePercent"`
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

	// ConfirmReceiptFlowEnabled controls the durable reminder + auto-confirm
	// flow after delivery (#auto-confirm). Unlike GroupOrdersEnabled it defaults
	// ON and the admin toggle can turn it OFF — a real runtime kill switch — so
	// the gate reads this value directly (default true) rather than `env OR`.
	ConfirmReceiptFlowEnabled bool `json:"confirmReceiptFlowEnabled"`

	// PickupReadyFlowEnabled controls the durable ready-to-collect flow for
	// pickup orders (ready notice → reminders → chef escalation). Same shape as
	// ConfirmReceiptFlowEnabled: defaults ON, and the admin toggle turns it OFF.
	PickupReadyFlowEnabled bool `json:"pickupReadyFlowEnabled"`

	// ── Refund policy v3 (#834, docs/refund-policy-v3-spec.md) ────────────────
	// MealPlanRefundTiers prices a customer cancellation by lead time before
	// cook-start: each band carries a floor the chef may not refund below, and
	// the top band can resolve automatically. Ops retunes the whole table here
	// rather than through a deploy — including flipping the >12h band from
	// auto-100% to a chef-decided 75% floor. Empty ⇒ DefaultMealPlanRefundTiers.
	MealPlanRefundTiers []MealPlanRefundTier `json:"mealPlanRefundTiers"`
	// ── Stranded fulfilment (an order the chef accepted and never finished) ──────
	// StuckOrderReminderDays is how long an in-flight order sits before both sides are
	// nudged, and how long between repeats. StuckOrderRefundDays is when the platform
	// stops asking and refunds the customer in full. 0 ⇒ the defaults below; a negative
	// StuckOrderRefundDays disables the auto-refund, leaving the nudges only.
	StuckOrderReminderDays int `json:"stuckOrderReminderDays"`
	StuckOrderRefundDays   int `json:"stuckOrderRefundDays"`
	// StuckOrderMaxReminders caps the nudges so a dead order stops pestering both sides
	// while it waits out the refund deadline.
	StuckOrderMaxReminders int `json:"stuckOrderMaxReminders"`

	// MealPlanChefRefundDecisionMinutes is how long a chef has to price a refund before it
	// resolves at 100% (of the base, which already excludes the platform's commission).
	// The floor protects the chef's minimum; this protects the customer's maximum wait.
	// 0 ⇒ the default below. Negative disables the sweep, leaving days pending indefinitely.
	MealPlanChefRefundDecisionMinutes int `json:"mealPlanChefRefundDecisionMinutes"`

	// ChefCancelPenalty* levies a percentage of the cancelled order's value on a
	// chef who cancels close to service, deducted from their next weekly
	// settlement. Guarded so a genuine one-off emergency is not auto-fined:
	// GraceCancellations cancellations inside GraceWindowDays are exempt, and an
	// admin can waive any levy outright.
	ChefCancelPenaltyEnabled    bool    `json:"chefCancelPenaltyEnabled"`
	ChefCancelPenaltyPercent    float64 `json:"chefCancelPenaltyPercent"`    // of the order value
	ChefCancelPenaltyLeadHours  float64 `json:"chefCancelPenaltyLeadHours"`  // cancels with LESS lead than this are levied
	ChefCancelPenaltyGraceCount int     `json:"chefCancelPenaltyGraceCount"` // free cancellations per window
	ChefCancelPenaltyGraceDays  int     `json:"chefCancelPenaltyGraceDays"`  // rolling window length

	// GatewayFeeLevy* recovers the payment gateway's transaction-fee loss on a chef-fault
	// Cashfree refund (#885), deducted from the chef's next weekly settlement through the
	// SAME mechanism as ChefCancelPenalty above (raise → optional grace → admin waiver →
	// statement deduction). Cashfree only — Razorpay/Stripe fee recovery is a noted follow-up.
	GatewayFeeLevyEnabled bool `json:"gatewayFeeLevyEnabled"`
	// GatewayFeeLevyPercent is a flat rate of the amount actually REFUNDED (never the order's
	// original total) — a configured proxy for the gateway's real per-refund processing fee.
	GatewayFeeLevyPercent float64 `json:"gatewayFeeLevyPercent"`
	// GatewayFeeLevyGraceEnabled defaults false: a gateway fee is a pass-through cost actually
	// incurred, not an accountability penalty that needs an emergency exemption.
	GatewayFeeLevyGraceEnabled bool `json:"gatewayFeeLevyGraceEnabled"`
	GatewayFeeLevyGraceCount   int  `json:"gatewayFeeLevyGraceCount"` // only consulted when GraceEnabled
	GatewayFeeLevyGraceDays    int  `json:"gatewayFeeLevyGraceDays"`  // only consulted when GraceEnabled
	// GatewayFeeLevyStackWithCancelLevy defaults false: one cancellation is one penalty event —
	// an order that already raised a cancel_late levy does not also raise a gateway_fee levy.
	GatewayFeeLevyStackWithCancelLevy bool `json:"gatewayFeeLevyStackWithCancelLevy"`
}

// DefaultPlatformPolicy matches what was hardcoded in handlers/orders.go
// before this policy existed, so upgrading is a no-op for existing traffic.
func DefaultPlatformPolicy() PlatformPolicy {
	return PlatformPolicy{
		// Customer-facing platform fee: a nominal 4.99% of subtotal — enough to
		// cover the payment gateway (~2%) + a small margin, without over-charging
		// customers. Runtime-tunable via the admin (SavePlatformPolicy).
		PlatformFeePercent: 4.99,
		// TaxPercent no longer prices anything. Tax is resolved per supply from
		// tax_rates (CreateOrder, MealPlanFeeTotals, the simulator), so this is a
		// display-only leftover the admin console still round-trips. Do not wire a
		// price back onto it — an 8% flat rate is what a tiffin plan was charged
		// while the same meal à la carte was charged 5% (D-08).
		TaxPercent: 8.0,
		// The flat fallback delivery fee, in RUPEES. It was 2.99 — a figure carried
		// over from a dollar-denominated template, on a marketplace that has only
		// ever charged in INR and whose own delivery pricing starts at 39. A meal
		// plan bills this per day, so a week of tiffin was charged 20.93 of
		// delivery (the other half of D-08).
		BaseDeliveryFee: 39.0,
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
		// Default ON — the auto-confirm flow ships enabled; an admin disables it
		// from the console (or ops via CONFIRM_RECEIPT_FLOW_ENABLED=false).
		ConfirmReceiptFlowEnabled: true,
		// Default ON — before this flow existed a pickup customer got one generic
		// push and then silence, so shipping it disabled would preserve the bug.
		PickupReadyFlowEnabled: true,
		// Refund policy v3 (#834). The tier table ships with the recommended
		// shape; see DefaultMealPlanRefundTiers for why >12h stays automatic.
		MealPlanRefundTiers: DefaultMealPlanRefundTiers(),
		// One hour to price a refund. Long enough for a chef mid-service to answer,
		// short enough that a cancelled customer is not waiting on a silent kitchen.
		MealPlanChefRefundDecisionMinutes: 60,
		// Nudge from day 3, repeat every 3 days up to 5 times, refund at 30. Nothing was
		// delivered, so at the deadline the customer is made whole rather than left to
		// chase an order both sides have plainly abandoned.
		StuckOrderReminderDays: 3,
		StuckOrderRefundDays:   30,
		StuckOrderMaxReminders: 5,
		// 6% of the cancelled order's value, on cancellations inside 4h of
		// service, with the first cancellation in a rolling 30 days exempt.
		ChefCancelPenaltyEnabled:    true,
		ChefCancelPenaltyPercent:    6.0,
		ChefCancelPenaltyLeadHours:  4.0,
		ChefCancelPenaltyGraceCount: 1,
		ChefCancelPenaltyGraceDays:  30,
		// GatewayFeeLevy (#885) defaults OFF — a brand-new money-charging mechanism must not
		// start charging chefs the moment this deploys; ops opts in explicitly through the
		// admin console, unlike ChefCancelPenaltyEnabled which defaulted true because it
		// matched pre-existing hardcoded behaviour. Percent is Cashfree's typical
		// processing-fee ballpark — retune freely. No grace, no stacking with cancel_late.
		GatewayFeeLevyEnabled:             false,
		GatewayFeeLevyPercent:             2.0,
		GatewayFeeLevyGraceEnabled:        false,
		GatewayFeeLevyGraceCount:          0,
		GatewayFeeLevyGraceDays:           0,
		GatewayFeeLevyStackWithCancelLevy: false,
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
	// No DB configured (unit tests, early startup) → unconfigured defaults (always open).
	if database.DB == nil {
		return def
	}
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
	// platformFeePercent=0 (promo) or taxPercent=0 (tax-exempt) must not
	// have the value silently replaced by the default. We use a pointer
	// struct to distinguish "missing" from "zero", then merge onto defaults.
	type partial struct {
		PlatformFeePercent *float64 `json:"platformFeePercent"`
		// LegacyServiceFeePercent reads blobs written before the platform fee was
		// renamed from "service fee". Without it, an admin-configured value would be
		// seen as absent and silently reset to the 4.99 default on first read after
		// deploy. Preferred only when the new key is missing; the next
		// SavePlatformPolicy rewrites the blob with the new key.
		LegacyServiceFeePercent *float64 `json:"serviceFeePercent"`
		TaxPercent              *float64 `json:"taxPercent"`
		BaseDeliveryFee         *float64 `json:"baseDeliveryFee"`
		PerKmDeliveryFee        *float64 `json:"perKmDeliveryFee"`
		ChefPayoutPercent       *float64 `json:"chefPayoutPercent"`
		DriverPayoutPercent     *float64 `json:"driverPayoutPercent"`
		Timezone                *string  `json:"timezone"`
		OpeningTime             *string  `json:"openingTime"`
		ClosingTime             *string  `json:"closingTime"`
		OperatingDays           *[]int   `json:"operatingDays"`
		ClosedMessage           *string  `json:"closedMessage"`
		GroupOrdersEnabled      *bool    `json:"groupOrdersEnabled"`
		// Pointer so an explicit admin `false` turns the default-on flow OFF
		// (a plain bool couldn't distinguish "unset" from "disabled").
		ConfirmReceiptFlowEnabled *bool `json:"confirmReceiptFlowEnabled"`
		PickupReadyFlowEnabled    *bool `json:"pickupReadyFlowEnabled"`
		// Refund policy v3 (#834). All pointers for the same reason: an admin
		// setting the penalty to 0% (suspended) or the grace count to 0 (no free
		// cancellation) must not be silently overwritten by the default.
		MealPlanRefundTiers         *[]MealPlanRefundTier `json:"mealPlanRefundTiers"`
		ChefCancelPenaltyEnabled    *bool                 `json:"chefCancelPenaltyEnabled"`
		ChefCancelPenaltyPercent    *float64              `json:"chefCancelPenaltyPercent"`
		ChefCancelPenaltyLeadHours  *float64              `json:"chefCancelPenaltyLeadHours"`
		ChefCancelPenaltyGraceCount *int                  `json:"chefCancelPenaltyGraceCount"`
		ChefCancelPenaltyGraceDays  *int                  `json:"chefCancelPenaltyGraceDays"`
		// GatewayFeeLevy (#885). Pointers for the same reason: an admin explicitly disabling
		// the levy or setting the rate/grace to 0 must not be silently reset by the default.
		GatewayFeeLevyEnabled             *bool    `json:"gatewayFeeLevyEnabled"`
		GatewayFeeLevyPercent             *float64 `json:"gatewayFeeLevyPercent"`
		GatewayFeeLevyGraceEnabled        *bool    `json:"gatewayFeeLevyGraceEnabled"`
		GatewayFeeLevyGraceCount          *int     `json:"gatewayFeeLevyGraceCount"`
		GatewayFeeLevyGraceDays           *int     `json:"gatewayFeeLevyGraceDays"`
		GatewayFeeLevyStackWithCancelLevy *bool    `json:"gatewayFeeLevyStackWithCancelLevy"`
	}
	var p partial
	if err := json.Unmarshal([]byte(setting.Value), &p); err != nil {
		log.Printf("platform_policy: failed to parse, using defaults: %v", err)
		return def
	}
	out := def
	if p.PlatformFeePercent != nil {
		out.PlatformFeePercent = *p.PlatformFeePercent
	} else if p.LegacyServiceFeePercent != nil {
		out.PlatformFeePercent = *p.LegacyServiceFeePercent
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
	if p.ConfirmReceiptFlowEnabled != nil {
		out.ConfirmReceiptFlowEnabled = *p.ConfirmReceiptFlowEnabled
	}
	if p.PickupReadyFlowEnabled != nil {
		out.PickupReadyFlowEnabled = *p.PickupReadyFlowEnabled
	}
	// Refund policy v3. normalizeRefundTiers falls back to the default table for an
	// empty or wholly-invalid configured one, so a bad save can't leave cancellations
	// unpriced.
	if p.MealPlanRefundTiers != nil {
		out.MealPlanRefundTiers = normalizeRefundTiers(*p.MealPlanRefundTiers)
	}
	if p.ChefCancelPenaltyEnabled != nil {
		out.ChefCancelPenaltyEnabled = *p.ChefCancelPenaltyEnabled
	}
	if p.ChefCancelPenaltyPercent != nil {
		out.ChefCancelPenaltyPercent = *p.ChefCancelPenaltyPercent
	}
	if p.ChefCancelPenaltyLeadHours != nil {
		out.ChefCancelPenaltyLeadHours = *p.ChefCancelPenaltyLeadHours
	}
	if p.ChefCancelPenaltyGraceCount != nil {
		out.ChefCancelPenaltyGraceCount = *p.ChefCancelPenaltyGraceCount
	}
	if p.ChefCancelPenaltyGraceDays != nil {
		out.ChefCancelPenaltyGraceDays = *p.ChefCancelPenaltyGraceDays
	}
	if p.GatewayFeeLevyEnabled != nil {
		out.GatewayFeeLevyEnabled = *p.GatewayFeeLevyEnabled
	}
	if p.GatewayFeeLevyPercent != nil {
		out.GatewayFeeLevyPercent = *p.GatewayFeeLevyPercent
	}
	if p.GatewayFeeLevyGraceEnabled != nil {
		out.GatewayFeeLevyGraceEnabled = *p.GatewayFeeLevyGraceEnabled
	}
	if p.GatewayFeeLevyGraceCount != nil {
		out.GatewayFeeLevyGraceCount = *p.GatewayFeeLevyGraceCount
	}
	if p.GatewayFeeLevyGraceDays != nil {
		out.GatewayFeeLevyGraceDays = *p.GatewayFeeLevyGraceDays
	}
	if p.GatewayFeeLevyStackWithCancelLevy != nil {
		out.GatewayFeeLevyStackWithCancelLevy = *p.GatewayFeeLevyStackWithCancelLevy
	}
	return out
}
