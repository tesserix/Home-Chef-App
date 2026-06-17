package services

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// IsChefFSSAIExpired reports whether an India chef's FSSAI food-safety licence
// has lapsed, which means the chef must be locked out of taking new orders and
// of receiving payouts until a renewal is verified. See issue #32 for the full
// policy and scenario matrix.
//
// Lockout semantics (food-safety gate — deliberately conservative):
//   - India-only. FSSAI is an Indian regulation; chefs whose PayoutCountry is
//     not "IN" are never FSSAI-locked.
//   - Coverage requires a VERIFIED FSSAI document. An unverified/pending upload
//     does not lift the lock, so a chef cannot bypass the gate with a fake or
//     back-dated document — only an admin-verified renewal unlocks.
//   - "Expired" = the chef's most-recent verified FSSAI ExpiryDate is in the
//     past. A licence is valid through its expiry date, so we only lock from the
//     day AFTER the recorded expiry (this honours the full expiry day and
//     absorbs minor tz drift between the stored UTC-midnight date and IST).
//   - A chef with no verified FSSAI on record, or one whose ExpiryDate is NULL
//     (legacy rows from before expiry capture), is NOT treated as expired here:
//     that is an onboarding/verification gap handled by the approval flow plus a
//     one-time backfill, not the expiry lockout. Locking them here would wrongly
//     take every pre-tracking chef offline.
//
// Fail-safe: on a transient DB error we fail OPEN (return false) and log an
// alert — a database blip must not halt the entire marketplace. The reminder
// cron and admin dashboard are the backstop for genuinely-expired chefs.
func IsChefFSSAIExpired(chef *models.ChefProfile) bool {
	if chef == nil {
		return false
	}
	if !strings.EqualFold(chef.PayoutCountry, "IN") {
		return false // FSSAI does not apply outside India.
	}

	// An active, time-boxed admin override (audited; #93) suspends the lockout
	// for genuine edge cases — e.g. a government renewal backlog where the chef's
	// paperwork is filed but not yet processed. Once it lapses the expiry gate
	// below re-applies automatically, so there is nothing to clean up.
	if chef.FSSAIOverrideUntil != nil && time.Now().Before(*chef.FSSAIOverrideUntil) {
		return false
	}

	var doc models.ChefDocument
	err := database.DB.
		Where("chef_id = ? AND type = ? AND status = ? AND expiry_date IS NOT NULL",
			chef.ID, models.DocFSSAILicense, models.DocStatusVerified).
		Order("expiry_date DESC").
		First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No verified FSSAI with a recorded expiry → not an expiry lockout.
			return false
		}
		// Transient DB error: fail open so the marketplace keeps serving, but
		// record it so the genuinely-expired backstop (cron/admin) can act.
		log.Printf("fssai-lockout: expiry check DB error for chef=%s: %v — failing open", chef.ID, err)
		return false
	}
	if doc.ExpiryDate == nil {
		return false
	}
	// Valid through the expiry date; locked from the following day.
	return time.Now().After(doc.ExpiryDate.AddDate(0, 0, 1))
}

// ExcludeFSSAILocked is a GORM scope that removes India chefs whose FSSAI licence
// has lapsed from a chef_profiles query — the set-based mirror of
// IsChefFSSAIExpired, for efficient list filtering (no per-row N+1 check).
//
// A chef is excluded iff PayoutCountry is "IN" AND they have no active admin
// override AND the MAX expiry across their verified fssai_license documents is
// before the same (now - 1 day) cutoff that IsChefFSSAIExpired uses. Using MAX
// (not "any expired doc") is essential: a chef who renewed still has the old,
// expired document, and must NOT be hidden — their latest verified expiry is in
// the future. The override clause mirrors the early-return in IsChefFSSAIExpired
// so a time-boxed admin reprieve (#93) keeps the chef visible everywhere.
//
// This query and IsChefFSSAIExpired express the identical rule and MUST stay in
// sync; the unit tests assert both, including the renewal and override cases.
func ExcludeFSSAILocked(db *gorm.DB) *gorm.DB {
	now := time.Now()
	cutoff := now.AddDate(0, 0, -1)
	return db.Where(
		`NOT (payout_country = ?
			AND (fssai_override_until IS NULL OR fssai_override_until <= ?)
			AND id IN (
				SELECT chef_id FROM chef_documents
				WHERE type = ? AND status = ? AND expiry_date IS NOT NULL
				GROUP BY chef_id HAVING MAX(expiry_date) < ?
			))`,
		"IN", now, models.DocFSSAILicense, models.DocStatusVerified, cutoff,
	)
}

// ChefsWithValidFSSAI returns, for a set of chef ids, which ones currently hold a
// verified, non-expired FSSAI licence — the basis for the hygiene/food-safety
// badge (#35). Batched into a single query so listing pages don't N+1. "Valid"
// uses the same "through the expiry day" cutoff as the lockout, so the badge and
// the lockout never contradict each other.
func ChefsWithValidFSSAI(chefIDs []uuid.UUID) map[uuid.UUID]bool {
	out := make(map[uuid.UUID]bool, len(chefIDs))
	if len(chefIDs) == 0 {
		return out
	}
	cutoff := time.Now().AddDate(0, 0, -1)
	var ids []uuid.UUID
	database.DB.Model(&models.ChefDocument{}).
		Where("type = ? AND status = ? AND expiry_date IS NOT NULL AND chef_id IN ?",
			models.DocFSSAILicense, models.DocStatusVerified, chefIDs).
		Group("chef_id").
		Having("MAX(expiry_date) >= ?", cutoff).
		Pluck("chef_id", &ids)
	for _, id := range ids {
		out[id] = true
	}
	return out
}

// ChefHasValidFSSAI is the single-chef form of ChefsWithValidFSSAI.
func ChefHasValidFSSAI(chefID uuid.UUID) bool {
	return ChefsWithValidFSSAI([]uuid.UUID{chefID})[chefID]
}
