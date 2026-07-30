package services

import (
	"context"
	"log"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// payout_status_cron.go — the sweep that resolves in-flight disbursements.
//
// THIS IS NOT A BACKSTOP. Cashfree Payouts transfers are asynchronous: creating
// one returns RECEIVED or PENDING and nothing further. Webhooks are deliberately
// not in use, so this sweep is the ONLY way a batch reaches a terminal state and
// the only way a WeeklyStatement is ever marked paid.
//
// If it stops running, nothing appears broken: no error is logged, no request
// fails, and money continues to leave the platform correctly. The only symptom is
// that chefs' statements sit "pending" forever while their banks show the credit.
// That asymmetry is the reason this file exists separately from
// payout_reconcile_cron.go, rather than being folded into it — so it is visible
// in the cron registry as its own line with its own name.

// payoutStatusInterval is how often in-flight transfers are re-checked.
//
// Two minutes: fast enough that a UPI transfer (which settles in seconds) is
// reflected almost immediately, and slow enough to stay far inside any rail rate
// limit given the sweep is capped at 200 batches per run.
const payoutStatusInterval = 2 * time.Minute

// StartPayoutStatusCron is the in-process fallback used when Temporal Schedules
// are not driving the crons.
func StartPayoutStatusCron(ctx context.Context) {
	go func() {
		runPayoutStatusScan(ctx)
		ticker := time.NewTicker(payoutStatusInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("payout-status: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runPayoutStatusScan(ctx)
			}
		}
	}()
	log.Println("payout-status: cron started (interval=2m)")
}

// runPayoutStatusScan resolves every executing batch against the rail.
//
// Both modes are swept, live first. A test-mode batch is a real transfer in
// Cashfree's sandbox and reaches a terminal state exactly as a live one does —
// leaving it unresolved would make sandbox testing of the whole flow impossible,
// which is the one thing this integration most needs to be able to do.
func runPayoutStatusScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("payout-status: PANIC recovered: %v", r)
		}
	}()

	if database.DB == nil {
		return
	}

	for _, mode := range []string{models.ChefModeLive, models.ChefModeTest} {
		// Skip a mode with no credentials rather than logging a failure per
		// batch: an unconfigured slot is the normal state before go-live.
		if GetCashfreePayoutFor(mode) == nil {
			continue
		}
		settled, err := PollExecutingPayouts(ctx, database.DB, mode)
		if err != nil {
			log.Printf("payout-status[%s]: sweep failed: %v", mode, err)
			CaptureBackgroundError(err)
			continue
		}
		if settled > 0 {
			log.Printf("payout-status[%s]: %d batch(es) reached a terminal state", mode, settled)
		}
	}
}
