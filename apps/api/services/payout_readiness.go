package services

// payout_readiness.go — the gate that stops a chef taking a customer's money
// before there is somewhere to send the chef's share.
//
// Today nothing enforces this. Payout setup is a Settings action
// (handlers/chefs.go SavePayoutDetails); onboarding explicitly defers it; and
// no server-side check gates approval, go-live or order acceptance on it. A
// chef can therefore be approved, switch on accepting_orders, take real money,
// deliver, dual-confirm, and accrue released holds with no payable destination
// on file — the payout engine (#736) would then build batches for a payee it
// cannot pay.
//
// The gate is runtime-tunable via PlatformSettings and ships OFF, so enabling
// it is an ops decision rather than a deploy. It deliberately blocks only the
// transition into accepting orders; it never switches a trading chef off.

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// PlatformSettings keys.
const (
	// payoutGateLevelKey selects how strict the gate is: off, method_on_file
	// or verified. Missing or invalid values fall back to off.
	payoutGateLevelKey = "payout.setup_gate_level"

	// payoutGraceUntilKey is an RFC3339 instant before which the gate reports
	// but does not enforce. It exists so turning the gate on does not strand
	// the existing cohort of approved chefs who never had a reason to add a
	// payout method.
	payoutGraceUntilKey = "payout.setup_grace_until"
)

// PayoutGateLevel is how much payout setup a chef must have to accept orders.
type PayoutGateLevel string

const (
	// PayoutGateOff — no requirement. Launch value.
	PayoutGateOff PayoutGateLevel = "off"
	// PayoutGateMethodOnFile — a payout method must be saved.
	PayoutGateMethodOnFile PayoutGateLevel = "method_on_file"
	// PayoutGateVerified — the method must also have passed verification.
	// Not selectable until #740 implements verification; see resolveGateLevel.
	PayoutGateVerified PayoutGateLevel = "verified"
)

// Reason codes, stable across releases so clients can branch on them and
// analytics can group by them. The wording lives in the client.
const (
	ReasonPayoutMethodMissing    = "payout_method_missing"
	ReasonPayoutMethodUnverified = "payout_method_unverified"
)

// payoutVerificationSupported reports whether payout-method verification
// exists yet. #740 flips this to true when penny-drop verification lands.
// Until then the `verified` gate level is downgraded rather than obeyed,
// because obeying it would block every chef on the platform.
const payoutVerificationSupported = false

// PayoutReadiness is a chef's standing against the gate.
type PayoutReadiness struct {
	// Level is the effective gate level after any downgrade.
	Level PayoutGateLevel `json:"level"`
	// MethodOnFile reports whether a payout method has been saved.
	MethodOnFile bool `json:"methodOnFile"`
	// Verified reports whether that method passed verification (#740).
	Verified bool `json:"verified"`
	// Ready reports whether the chef satisfies the gate.
	Ready bool `json:"ready"`
	// Enforced reports whether failing the gate actually blocks. False during
	// the grace window and when the gate is off.
	Enforced bool `json:"enforced"`
	// ReasonCode explains a not-ready result; empty when ready.
	ReasonCode string `json:"reasonCode,omitempty"`
	// GraceActive reports that the grace window is suppressing enforcement,
	// so clients can nag rather than block.
	GraceActive bool `json:"graceActive"`
}

// Blocks reports whether this readiness should refuse the transition into
// accepting orders.
func (r PayoutReadiness) Blocks() bool { return r.Enforced && !r.Ready }

// ParsePayoutGateLevel parses a stored setting value. Unknown values are
// rejected so callers can fall back to off rather than guess.
func ParsePayoutGateLevel(raw string) (PayoutGateLevel, bool) {
	switch PayoutGateLevel(strings.ToLower(strings.TrimSpace(raw))) {
	case PayoutGateOff:
		return PayoutGateOff, true
	case PayoutGateMethodOnFile:
		return PayoutGateMethodOnFile, true
	case PayoutGateVerified:
		return PayoutGateVerified, true
	default:
		return "", false
	}
}

// resolveGateLevel downgrades a level the platform cannot yet satisfy.
func resolveGateLevel(level PayoutGateLevel, verificationSupported bool) PayoutGateLevel {
	if level == PayoutGateVerified && !verificationSupported {
		return PayoutGateMethodOnFile
	}
	return level
}

// EvaluatePayoutReadiness decides whether a chef satisfies a gate level.
// Pure — no DB, no clock — so every combination is table-testable.
//
// A chef with nothing on file is always reported as missing rather than
// unverified, even under the stricter gate: the wrong reason code sends them
// to the wrong screen.
func EvaluatePayoutReadiness(level PayoutGateLevel, methodOnFile, verified bool) (bool, string) {
	switch level {
	case PayoutGateOff:
		return true, ""
	case PayoutGateMethodOnFile:
		if methodOnFile {
			return true, ""
		}
		return false, ReasonPayoutMethodMissing
	case PayoutGateVerified:
		if !methodOnFile {
			return false, ReasonPayoutMethodMissing
		}
		if !verified {
			return false, ReasonPayoutMethodUnverified
		}
		return true, ""
	default:
		// An unrecognised level must not silently become a hard gate.
		return true, ""
	}
}

// PayoutGraceActive reports whether the grace window is still open.
//
// It fails closed: an unset, malformed or past value grants no grace. Failing
// open would mean a typo in a settings row silently lets unpayable chefs trade.
func PayoutGraceActive(raw string, now time.Time) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}
	until, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return false
	}
	return now.Before(until)
}

// buildReadiness assembles the result. Split out so the decision logic is
// testable without a database.
func buildReadiness(level PayoutGateLevel, methodOnFile, verified, graceActive bool) PayoutReadiness {
	ready, reason := EvaluatePayoutReadiness(level, methodOnFile, verified)
	return PayoutReadiness{
		Level:        level,
		MethodOnFile: methodOnFile,
		Verified:     verified,
		Ready:        ready,
		Enforced:     level != PayoutGateOff && !graceActive,
		ReasonCode:   reason,
		GraceActive:  graceActive,
	}
}

// GetPayoutGateLevel resolves the effective gate level from PlatformSettings.
// A missing or invalid setting yields off, so a bad value can never halt
// trading platform-wide.
func GetPayoutGateLevel(db *gorm.DB) PayoutGateLevel {
	var setting models.PlatformSettings
	if err := db.Where("key = ?", payoutGateLevelKey).First(&setting).Error; err != nil {
		return PayoutGateOff
	}
	level, ok := ParsePayoutGateLevel(setting.Value)
	if !ok {
		return PayoutGateOff
	}
	return resolveGateLevel(level, payoutVerificationSupported)
}

// payoutGraceUntilRaw reads the grace setting, empty when unset.
func payoutGraceUntilRaw(db *gorm.DB) string {
	var setting models.PlatformSettings
	if err := db.Where("key = ?", payoutGraceUntilKey).First(&setting).Error; err != nil {
		return ""
	}
	return setting.Value
}

// ChefHasPayoutMethod reports whether a chef has saved a payout destination.
//
// The selector lives in chef_profiles.payout_method; the sensitive fields live
// in Secret Manager (SavePayoutDetails blanks the DB columns deliberately), so
// the selector is the only queryable signal and is what the gate reads.
func ChefHasPayoutMethod(chef *models.ChefProfile) bool {
	if chef == nil {
		return false
	}
	return strings.TrimSpace(chef.PayoutMethod) != ""
}

// chefPayoutVerified reports whether the chef's method passed verification.
// Always false until #740 lands; kept as a seam so the gate composes with it
// without rework.
func chefPayoutVerified(_ *models.ChefProfile) bool { return false }

// ChefPayoutReadiness evaluates a chef against the live gate.
func ChefPayoutReadiness(db *gorm.DB, chef *models.ChefProfile, now time.Time) PayoutReadiness {
	level := GetPayoutGateLevel(db)
	grace := PayoutGraceActive(payoutGraceUntilRaw(db), now)
	return buildReadiness(level, ChefHasPayoutMethod(chef), chefPayoutVerified(chef), grace)
}

// PayoutGateOpenFilter returns a SQL predicate restricting a bulk
// accepting_orders=true update to chefs that satisfy the gate, and whether the
// gate is enforced at all.
//
// The automated paths that reopen kitchens — the pause-expiry resume cron and
// the scheduled open/close cron — write accepting_orders in bulk. Without this
// they would be a hole straight through the gate: a chef with no payout method
// would be switched back on by a timer. Closing a kitchen is never filtered;
// a chef must always be able to stop trading.
func PayoutGateOpenFilter(db *gorm.DB, now time.Time) (predicate string, enforced bool) {
	level := GetPayoutGateLevel(db)
	if level == PayoutGateOff {
		return "", false
	}
	if PayoutGraceActive(payoutGraceUntilRaw(db), now) {
		return "", false
	}
	// Only method_on_file is reachable here: resolveGateLevel downgrades
	// `verified` until #740 lands, and will extend this predicate when it does.
	return "COALESCE(payout_method, '') <> ''", true
}

// ErrPayoutSetupRequired is returned when the gate refuses a transition into
// accepting orders. Handlers map it to 409 with the reason code, so the vendor
// app can deep-link to the payout screen instead of showing a dead end.
type ErrPayoutSetupRequired struct {
	ReasonCode string
}

func (e ErrPayoutSetupRequired) Error() string {
	return fmt.Sprintf("payout setup required: %s", e.ReasonCode)
}

// RequirePayoutReadyToAcceptOrders guards every path that switches
// accepting_orders on.
//
// It guards only the transition into accepting orders. It never switches a
// trading chef off, and it never blocks turning accepting_orders off — a chef
// must always be able to stop taking orders, gate or no gate.
func RequirePayoutReadyToAcceptOrders(db *gorm.DB, chef *models.ChefProfile, accepting bool, now time.Time) error {
	if !accepting {
		return nil
	}
	r := ChefPayoutReadiness(db, chef, now)
	if r.Blocks() {
		return ErrPayoutSetupRequired{ReasonCode: r.ReasonCode}
	}
	return nil
}
