package services

import (
	"errors"
	"log"
	"strings"
	"time"

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
