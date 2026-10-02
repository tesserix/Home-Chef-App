package services

// docs_deadline_cron.go — enforce the 30-day document window for chef
// onboarding. A chef may submit their application without uploading the
// required documents (ID proof + FSSAI licence) and keep building their
// kitchen and menus, but the application cannot be approved without them:
//
//   - day 25: one warning (push + in-app + email) — "5 days left".
//   - day 30: the pending application is withdrawn — the approval request is
//     rejected with an explanatory note, the clock is cleared, and the chef is
//     told to re-apply. The kitchen never became customer-visible (customer
//     listings filter is_verified) and could never open (verification gate on
//     accepting_orders), so withdrawal has no live-order blast radius.
//
// Same sweep discipline as accept_reminder_cron.go: claim the stamp with a
// guarded UPDATE first, notify only if the claim won, so two sweep instances
// can never double-send or double-withdraw.

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

const (
	docsDeadlineInterval = 1 * time.Hour
	// DocsUploadWindow — how long after submitting onboarding a chef has to
	// upload the required documents.
	DocsUploadWindow = 30 * 24 * time.Hour
	// docsWarningLead — how far before the deadline the single warning fires.
	docsWarningLead = 5 * 24 * time.Hour
)

// StartDocsDeadlineCron launches the document-deadline sweep.
func StartDocsDeadlineCron(ctx context.Context) {
	go func() {
		runDocsDeadlineScan(ctx)
		ticker := time.NewTicker(docsDeadlineInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("docs-deadline: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runDocsDeadlineScan(ctx)
			}
		}
	}()
	log.Printf("docs-deadline: cron started (interval=%s window=%s)", docsDeadlineInterval, DocsUploadWindow)
}

func runDocsDeadlineScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("docs-deadline: recovered from panic: %v", r)
		}
	}()
	if database.DB == nil {
		return
	}
	warned, withdrawn := sweepDocsDeadlines(database.DB, time.Now())
	nudged := sweepPayoutReminders(database.DB, time.Now())
	if warned+withdrawn+nudged > 0 {
		log.Printf("docs-deadline: warned=%d withdrawn=%d payout-nudged=%d", warned, withdrawn, nudged)
	}
	_ = ctx
}

// ChefDocsComplete reports whether the chef has uploaded every required
// document type. Shared with the onboarding-status endpoint so the app and
// the sweep agree on what "complete" means.
func ChefDocsComplete(db *gorm.DB, chefID uuid.UUID) bool {
	var chef models.ChefProfile
	if err := db.Select("payout_country").Take(&chef, "id = ?", chefID).Error; err != nil {
		return false
	}
	country, ok := NormalizeKitchenCountry(chef.PayoutCountry)
	if !ok {
		return false
	}
	required := RequiredChefDocTypes(country)
	var n int64
	err := db.Model(&models.ChefDocument{}).
		Where("chef_id = ? AND type IN ?", chefID, required).
		Where("status <> ?", models.DocStatusRejected).
		Where("expiry_date IS NULL OR expiry_date >= ?", time.Now().UTC().Truncate(24*time.Hour)).
		Distinct("type").
		Count(&n).Error
	return err == nil && n >= int64(len(required))
}

func sweepDocsDeadlines(db *gorm.DB, now time.Time) (warned, withdrawn int) {
	// Only unverified chefs with a running clock AND a pending kitchen
	// application are on the hook: admin-rejected applications restart the
	// clock on re-submit, verified chefs are done.
	var chefs []models.ChefProfile
	if err := db.Model(&models.ChefProfile{}).
		Select("chef_profiles.id, chef_profiles.user_id, chef_profiles.onboarded_at, chef_profiles.docs_warning_sent_at").
		Joins("JOIN approval_requests ar ON ar.chef_id = chef_profiles.id AND ar.type = ? AND ar.status = ?",
			models.ApprovalKitchenOnboarding, models.ApprovalPending).
		Where("chef_profiles.is_verified = ? AND chef_profiles.onboarded_at IS NOT NULL", false).
		Find(&chefs).Error; err != nil {
		log.Printf("docs-deadline: query failed: %v", err)
		return 0, 0
	}

	for _, chef := range chefs {
		if ChefDocsComplete(db, chef.ID) {
			continue // docs are in — the admin review queue takes it from here
		}
		deadline := chef.OnboardedAt.Add(DocsUploadWindow)

		switch {
		case !now.Before(deadline):
			if withdrawDocsExpiredApplication(db, &chef, deadline) {
				withdrawn++
			}
		case !now.Before(deadline.Add(-docsWarningLead)) && chef.DocsWarningSentAt == nil:
			if warnDocsDeadline(db, &chef, deadline, now) {
				warned++
			}
		}
	}
	return warned, withdrawn
}

// sweepPayoutReminders nudges chefs who onboarded ≥25 days ago and still have
// no payout destination. Softer than the documents window: the account is
// never removed — earnings simply cannot be paid out (payout gate #739) —
// so this is one reminder, not an ultimatum. Applies to verified chefs too:
// they are the ones actually accruing earnings with nowhere to send them.
func sweepPayoutReminders(db *gorm.DB, now time.Time) int {
	cutoff := now.Add(-(DocsUploadWindow - docsWarningLead)) // day 25
	var chefs []models.ChefProfile
	if err := db.Model(&models.ChefProfile{}).
		Select("id, user_id, onboarded_at").
		Where("onboarded_at IS NOT NULL AND onboarded_at <= ?", cutoff).
		Where("(payout_method IS NULL OR TRIM(payout_method) = '')").
		Where("payout_reminder_sent_at IS NULL").
		Find(&chefs).Error; err != nil {
		log.Printf("docs-deadline: payout-reminder query failed: %v", err)
		return 0
	}

	nudged := 0
	for _, chef := range chefs {
		claimed := false
		err := db.Transaction(func(tx *gorm.DB) error {
			res := tx.Model(&models.ChefProfile{}).
				Where("id = ? AND payout_reminder_sent_at IS NULL", chef.ID).
				Update("payout_reminder_sent_at", now)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return nil // another sweep instance won
			}
			claimed = true
			return EnqueueEvent(tx, SubjectChefPayoutReminder, "chef.payout_details.reminder", chef.UserID, map[string]any{
				"chef_id": chef.ID.String(),
			})
		})
		if err != nil {
			log.Printf("docs-deadline: payout reminder tx failed for %s (will retry): %v", chef.ID, err)
			CaptureBackgroundError(err)
			continue
		}
		if claimed {
			nudged++
		}
	}
	return nudged
}

func warnDocsDeadline(db *gorm.DB, chef *models.ChefProfile, deadline, now time.Time) bool {
	// Claim and enqueue in ONE transaction: the stamp and the outbox row
	// commit together, so a crash can never burn the claim without the
	// notification (the outbox relay then delivers it to JetStream with
	// PubAck — durable end to end). The guarded UPDATE still makes exactly
	// one concurrent sweep instance win.
	claimed := false
	err := db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.ChefProfile{}).
			Where("id = ? AND docs_warning_sent_at IS NULL", chef.ID).
			Update("docs_warning_sent_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // another sweep instance won this nudge
		}
		claimed = true
		daysLeft := max(int(time.Until(deadline).Hours()/24), 1)
		return EnqueueEvent(tx, SubjectChefDocsDeadlineWarning, "chef.docs_deadline.warning", chef.UserID, map[string]any{
			"chef_id":   chef.ID.String(),
			"deadline":  deadline,
			"days_left": daysLeft,
		})
	})
	if err != nil {
		// Rolled back — the stamp is NOT burned, the next sweep retries.
		log.Printf("docs-deadline: warning tx failed for %s (will retry): %v", chef.ID, err)
		CaptureBackgroundError(err)
		return false
	}
	return claimed
}

func withdrawDocsExpiredApplication(db *gorm.DB, chef *models.ChefProfile, deadline time.Time) bool {
	claimed := false
	err := db.Transaction(func(tx *gorm.DB) error {
		// Clearing onboarded_at is the claim: it removes the chef from every
		// future sweep, and only one instance's guarded UPDATE can win it.
		res := tx.Model(&models.ChefProfile{}).
			Where("id = ? AND onboarded_at = ?", chef.ID, chef.OnboardedAt).
			Updates(map[string]any{
				"onboarded_at":         nil,
				"docs_warning_sent_at": nil,
				"accepting_orders":     false,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // another sweep won, or the chef state moved under us
		}
		claimed = true
		if err := tx.Model(&models.ApprovalRequest{}).
			Where("chef_id = ? AND type = ? AND status = ?",
				chef.ID, models.ApprovalKitchenOnboarding, models.ApprovalPending).
			Updates(map[string]any{
				"status":      models.ApprovalRejected,
				"admin_notes": "Application withdrawn automatically: the required documents (ID proof, address proof and FSSAI licence) were not uploaded within 30 days of submission. You can re-apply from the app at any time.",
			}).Error; err != nil {
			return err
		}
		// Outbox row rides the same commit: the withdrawal and its
		// notification are atomic — neither can exist without the other.
		return EnqueueEvent(tx, SubjectChefDocsDeadlineExpired, "chef.docs_deadline.expired", chef.UserID, map[string]any{
			"chef_id":  chef.ID.String(),
			"deadline": deadline,
		})
	})
	if err != nil {
		log.Printf("docs-deadline: withdraw failed for %s (will retry): %v", chef.ID, err)
		CaptureBackgroundError(err)
		return false
	}
	return claimed
}
