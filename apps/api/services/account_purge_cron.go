package services

// account_purge_cron.go — erases accounts whose restore window has elapsed.
//
// This is the half of the deletion contract the original DPDP work left
// unbuilt: rows were soft-deleted and a retention window was promised, but
// nothing ever purged, so "deleted" accounts accumulated indefinitely. Without
// this cron the restore-window promise is only half true — data is hidden, never
// erased — which is exactly what a DPDP audit or a store review would fault.
//
// Personal data is erased outright. Only PII-stripped financial records are
// archived (see account_archive.go), because Indian tax law requires the
// financial trail while DPDP does not permit keeping the identity beside it.

import (
	"context"
	"log"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const (
	accountPurgeInterval = 24 * time.Hour
	// accountPurgeBatch caps one scan so a large backlog cannot hold a long
	// transaction open against the shared instance. The remainder is picked up
	// on the next run.
	accountPurgeBatch = 200
)

// StartAccountPurgeCron launches the purge sweeper. Returns immediately; lives
// for the life of ctx. Used only when Temporal Schedules are unavailable — see
// cronJobs() in cron_temporal.go.
func StartAccountPurgeCron(ctx context.Context) {
	go func() {
		runAccountPurgeScan(ctx)

		ticker := time.NewTicker(accountPurgeInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("account-purge: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runAccountPurgeScan(ctx)
			}
		}
	}()
	log.Printf("account-purge: cron started (interval=24h, window=%dd)",
		int(RestoreWindow.Hours()/24))
}

func runAccountPurgeScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("account-purge: panic recovered: %v", r)
		}
	}()

	if database.DB == nil {
		return
	}
	now := time.Now().UTC()

	var due []models.User
	if err := database.DB.WithContext(ctx).Unscoped().
		Where("deleted_at IS NOT NULL AND purge_after IS NOT NULL AND purge_after <= ?", now).
		Limit(accountPurgeBatch).
		Find(&due).Error; err != nil {
		log.Printf("account-purge: could not list due accounts: %v", err)
		return
	}
	if len(due) == 0 {
		return
	}

	var purged, failed int
	for i := range due {
		if purgeOneAccount(ctx, due[i]) {
			purged++
		} else {
			failed++
		}
	}

	log.Printf("account-purge: scan complete (due=%d purged=%d failed=%d)",
		len(due), purged, failed)
}

// purgeOneAccount erases a single due account, reporting whether it succeeded.
//
// It recovers per USER. The scan-level recover above only stops a panic from
// killing the cron goroutine — it cannot resume the loop, so a single panicking
// account silently abandoned every remaining account in the batch. That is how a
// nil GCS client in the worker (InitStorage was never called there) turned one
// bad archive into "no account is ever erased", with a single recovered-panic
// line as the only evidence. The retention window is a legal commitment, so the
// batch must survive one bad row.
func purgeOneAccount(ctx context.Context, user models.User) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			// User id only — never the email. This is the erasure path.
			log.Printf("account-purge: panic purging user=%s (batch continues): %v", user.ID, r)
			ok = false
		}
	}()

	// Refuse to erase an account that still holds money (#948). Deleting the user
	// row does not delete their financial position: production carries one user
	// with no `users` row, no wallet, a ₹145.87 ledger credit that was never
	// materialised, and ₹297.58 captured on cancelled orders and never refunded.
	// The ledger-reconcile cron then reports that as DRIFT forever and refuses to
	// auto-correct.
	//
	// Deliberately NOT silent and NOT permanent: the row keeps its purge_after so
	// it re-surfaces every scan until someone settles or writes it off. Erasure is
	// a legal commitment, so this must be noisy enough to action rather than a
	// quiet block. A guard that cannot read the position lets the purge proceed —
	// failing erasure on a database error would be the worse trade.
	if owed, err := AccountUnsettledMoney(database.DB, user.ID); err != nil {
		log.Printf("account-purge: could not check unsettled money for user=%s (continuing): %v", user.ID, err)
	} else if owed.Any() {
		log.Printf("account-purge: SKIPPING user=%s — account still holds money (%s); settle or write off before erasure",
			user.ID, owed)
		return false
	}

	// Archive before erasing. A failed archive must not stop the erasure: the
	// user asked to be deleted, and holding their data back because a bucket
	// write failed would be the worse outcome of the two.
	if err := ArchiveAccountFinancials(ctx, user); err != nil {
		log.Printf("account-purge: archive failed for user=%s (continuing): %v", user.ID, err)
	}

	// Last chance to kill the credential, in case the post-commit attempt at
	// deletion time failed.
	if user.GIPUid != "" {
		if err := DeleteGIPAccount(ctx, user.GIPTenantID, user.GIPUid); err != nil {
			log.Printf("account-purge: GIP delete failed for user=%s: %v", user.ID, err)
		}
	}

	// Same for the Apple grant — this is the last moment the refresh token still
	// exists, since PurgeUser erases the row that holds it.
	RevokeAppleGrantForUser(ctx, &user)

	if err := PurgeUser(database.DB.WithContext(ctx), user.ID, user.Role); err != nil {
		log.Printf("account-purge: purge failed for user=%s: %v", user.ID, err)
		return false
	}
	log.Printf("account-purge: erased user=%s role=%s", user.ID, user.Role)
	return true
}
