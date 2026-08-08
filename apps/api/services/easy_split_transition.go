package services

import (
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// easy_split_transition.go — the one place a chef's Easy Split vendor status
// changes (#1083).
//
// Cashfree verifies a bank account asynchronously and reports the outcome
// twice: on a vendor webhook, and again on the next reconcile tick. Both call
// through here, and the write is conditional on the value they claim to be
// replacing — so the second caller sees no change, and the chef is told once.

// easySplitDeadAfter stops polling a registration that has not moved in this
// long. Verification takes hours; a quarter of stillness is a dead
// registration, and re-saving payout details touches the row, which puts the
// chef straight back in the sweep.
const easySplitDeadAfter = 60 * 24 * time.Hour

// EasySplitNeedsReconcile reports whether a chef is worth a Cashfree call.
func EasySplitNeedsReconcile(status string, updatedAt time.Time) bool {
	s := strings.ToUpper(strings.TrimSpace(status))
	switch s {
	case CashfreeVendorActive, CashfreeVendorDeleted, CashfreeVendorBankValidationFailed:
		return false
	}
	// A chef with no status at all has never been registered, so the age of the
	// row says nothing about a registration — always retry those.
	if s == "" {
		return true
	}
	return time.Since(updatedAt) < easySplitDeadAfter
}

// ApplyEasySplitVendorStatus persists a status Cashfree reported, and reports
// whether it was actually a change. A false return means "already recorded" —
// no notification, no audit, no write.
//
// An empty status is refused: that is what a truncated webhook payload looks
// like, and taking it at face value would mark a verified chef unpayable.
func ApplyEasySplitVendorStatus(db *gorm.DB, chef *models.ChefProfile, vendorID, status string) bool {
	return ApplyEasySplitVendorStatusWithReason(db, chef, vendorID, status, "")
}

// ApplyEasySplitVendorStatusWithReason is the same transition carrying whatever
// Cashfree said about it — only the webhook has that, a status re-read does not.
func ApplyEasySplitVendorStatusWithReason(db *gorm.DB, chef *models.ChefProfile, vendorID, status, reason string) bool {
	if db == nil || chef == nil {
		return false
	}
	next := strings.ToUpper(strings.TrimSpace(status))
	if next == "" {
		return false
	}
	// Everything here reads and writes the partition the chef is in, so a
	// sandbox registration can never become the live payout identity (#1145).
	idCol, statusCol := chef.VendorColumns()
	prev := strings.ToUpper(strings.TrimSpace(chef.VendorStatus()))
	if vendorID == "" {
		vendorID = chef.VendorID()
	}
	if next == prev && vendorID == chef.VendorID() {
		return false
	}

	// Conditional on the value being replaced: two callers racing the same
	// transition, only one gets a row.
	res := db.Model(&models.ChefProfile{}).
		Where("id = ? AND COALESCE("+statusCol+", '') = ?", chef.ID, chef.VendorStatus()).
		Updates(map[string]any{idCol: vendorID, statusCol: next})
	if res.Error != nil {
		log.Printf("easy-split: persisting status %s for chef %s failed: %v", next, chef.ID, res.Error)
		return false
	}
	if res.RowsAffected == 0 {
		return false
	}

	chef.SetVendorIdentity(vendorID, next)
	LogSystemAudit(nil, "chef.payout.vendor_status", "chef", chef.ID.String(),
		map[string]any{"status": prev}, map[string]any{"status": next, "vendorId": vendorID})

	if notice, ok := easySplitStatusNotice(prev, next, reason); ok {
		notifyEasySplitStatus(chef.UserID, notice)
	}
	return true
}

// easySplitNotice is what the chef is told. Split from delivery so the wording
// is assertable without a notification service.
type easySplitNotice struct {
	Kind  string
	Title string
	Body  string
}

// easySplitStatusNotice decides whether a transition is worth telling the chef
// about. Movement between two pending states is not: it changes nothing they
// can act on, and a notification per tick trains them to ignore the channel.
func easySplitStatusNotice(prev, next, reason string) (easySplitNotice, bool) {
	switch next {
	case CashfreeVendorActive:
		return easySplitNotice{
			Kind:  "easy_split_payouts_active",
			Title: "Payouts are active",
			Body:  "Your bank account is verified. Money from your orders now goes straight to your account.",
		}, true
	case CashfreeVendorBlocked, CashfreeVendorDeleted, CashfreeVendorBankValidationFailed:
		body := "Please check your account number, IFSC and the name on the account, then save them again. You'll keep being paid on the weekly statement in the meantime."
		if why := easySplitReason(reason); why != "" {
			body = why + ". " + body
		}
		return easySplitNotice{
			Kind:  "easy_split_payouts_failed",
			Title: "We couldn't verify your bank account",
			Body:  body,
		}, true
	}
	_ = prev
	return easySplitNotice{}, false
}

// easySplitReason keeps Cashfree's remark only when it reads as a sentence a
// chef can act on. A status code passed through says nothing to them and reads
// like a fault in our app.
func easySplitReason(reason string) string {
	r := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(reason), "."))
	if r == "" || len(r) > 160 || strings.Contains(r, "_") || r == strings.ToUpper(r) {
		return ""
	}
	return r
}

func notifyEasySplitStatus(userID uuid.UUID, notice easySplitNotice) {
	if svc := GetNotificationService(); svc != nil {
		if err := svc.SaveUserNotification(&models.Notification{
			UserID: userID, Type: notice.Kind, Title: notice.Title, Message: notice.Body,
		}); err != nil {
			log.Printf("easy-split: notification row failed for user %s: %v", userID, err)
		}
	}
	if err := SendPushNotification(userID, notice.Title, notice.Body,
		map[string]string{"type": notice.Kind, "deeplink": "homechef-vendor:///payout"}); err != nil {
		log.Printf("easy-split: push failed for user %s: %v", userID, err)
	}
}
