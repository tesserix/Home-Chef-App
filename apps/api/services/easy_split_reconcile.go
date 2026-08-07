package services

import (
	"context"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// easy_split_reconcile.go — finish the chef payout onboarding that the save path
// only STARTS (#1029).
//
// SavePayoutDetails registers the chef as an Easy Split vendor in a detached
// goroutine and logs failures. That is a reasonable choice for the save itself —
// a Cashfree hiccup must not fail a chef's payout-details save, and registration
// moves no money. What was missing is the other half: nothing ever retried, and
// nothing re-read a vendor still verifying. A chef whose one attempt failed —
// bad IFSC, missing PAN, no phone on the user row, a 5xx from Cashfree — stayed
// unregistered forever, silently. Their kitchen kept trading (the payout gate
// only checks that a method is on file), but split-at-capture fell back to full
// capture and tips answered 409 "This chef's payout account isn't active yet".
//
// This sweep closes the loop:
//   - vendor id missing  → re-run registration from the stored bank details
//   - vendor id present but not ACTIVE → re-read the status from Cashfree
//
// Both are idempotent: CreateVendor is keyed on a deterministic vendor id
// (EasySplitVendorIDFor) and PATCHes an existing one, and FetchVendor is a read.
// Safe to run repeatedly and concurrently with the save path.

const easySplitReconcileInterval = 30 * time.Minute

// easySplitReconcileBatch bounds one pass. Each chef costs a Cashfree API call,
// and the backlog only needs to drain steadily, not instantly.
const easySplitReconcileBatch = 50

// StartEasySplitReconcileCron runs the sweep on a ticker.
func StartEasySplitReconcileCron(ctx context.Context) {
	go func() {
		runEasySplitReconcile(ctx)
		t := time.NewTicker(easySplitReconcileInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("easy-split-reconcile: shutting down on ctx cancel")
				return
			case <-t.C:
				runEasySplitReconcile(ctx)
			}
		}
	}()
}

func runEasySplitReconcile(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("easy-split-reconcile: panic recovered: %v", r)
		}
	}()
	if database.DB == nil {
		return
	}
	ReconcileEasySplitVendors(ctx, database.DB, easySplitReconcileBatch)
}

// easySplitReconcileCandidates selects the chefs one pass is worth spending
// Cashfree calls on: a payout destination on file, and a registration that can
// still move. Terminal and long-dead ones are excluded in SQL rather than in Go
// so the batch limit isn't spent on rows that would be skipped anyway — the
// same rule EasySplitNeedsReconcile states for a single chef (#1083).
func easySplitReconcileCandidates(db *gorm.DB, limit int) ([]models.ChefProfile, error) {
	var chefs []models.ChefProfile
	err := db.Preload("User").
		Where("COALESCE(payout_method, '') <> ''").
		Where("UPPER(COALESCE(cashfree_vendor_status, '')) NOT IN ?",
			[]string{CashfreeVendorActive, CashfreeVendorDeleted, CashfreeVendorBankValidationFailed}).
		Where("COALESCE(cashfree_vendor_status, '') = '' OR updated_at > ?",
			time.Now().Add(-easySplitDeadAfter)).
		Limit(limit).
		Find(&chefs).Error
	return chefs, err
}

// ReconcileEasySplitVendors retries registration / refreshes status for chefs
// that have payout details but no ACTIVE Easy Split vendor. Returns how many
// chefs it moved to ACTIVE, and how many it attempted.
//
// Exported and db-injected so it is testable and so an admin path can force a
// pass without waiting for the ticker.
func ReconcileEasySplitVendors(ctx context.Context, db *gorm.DB, limit int) (activated, attempted int) {
	if db == nil || limit <= 0 {
		return 0, 0
	}

	chefs, err := easySplitReconcileCandidates(db, limit)
	if err != nil {
		log.Printf("easy-split-reconcile: query failed: %v", err)
		return 0, 0
	}

	for i := range chefs {
		chef := &chefs[i]
		if GetCashfreeFor(chef.Mode) == nil {
			continue // gateway not configured for this chef's mode
		}
		attempted++

		var (
			vendor *CashfreeVendorResponse
			err    error
		)
		if chef.CashfreeVendorID == "" {
			// Never registered, or the save-path goroutine failed. Bank details
			// live in Secret Manager, which is what the admin variant reads.
			vendor, err = EnsureEasySplitVendor(ctx, db, chef)
		} else {
			// Registered but still verifying at Cashfree — just re-read it.
			vendor, err = RefreshEasySplitVendor(ctx, db, chef)
		}
		if err != nil {
			// Expected and recurring for a chef with genuinely bad details (name
			// mismatch, dead IFSC). Logged, not escalated: the sweep will keep
			// trying, and the chef-facing surface reports the status separately.
			log.Printf("easy-split-reconcile: chef %s not activated: %v", chef.ID, err)
			continue
		}
		if vendor != nil && strings.EqualFold(strings.TrimSpace(vendor.Status), CashfreeVendorActive) {
			activated++
			log.Printf("easy-split-reconcile: chef %s vendor %s is now ACTIVE", chef.ID, vendor.VendorID)
		}
	}

	if attempted > 0 {
		log.Printf("easy-split-reconcile: attempted %d, activated %d", attempted, activated)
	}
	return activated, attempted
}
