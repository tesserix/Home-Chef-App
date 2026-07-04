package services

// cancellation_cron.go — the durable sweep that makes cancellation refunds
// never-lost and vendor timeouts self-resolving (#475). Runs via a Temporal
// Schedule (exactly-once, survives restarts, catches missed windows) when
// Temporal is enabled, else an in-process ticker — same pattern as the meal-plan
// and payout-reconcile crons.

import (
	"context"
	"log"
	"time"
)

const cancellationSweepInterval = 2 * time.Minute

// runCancellationSweep retries owed-but-unexecuted refunds and times out stuck
// vendor requests. Idempotent + panic-guarded so one bad row never stops the sweep.
func runCancellationSweep(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("cancellation-sweep: panic recovered: %v", r)
		}
	}()
	_ = ctx
	SweepCancellationRefunds()
	SweepCancellationTimeouts(time.Now())
}

// StartCancellationCron is the in-process ticker fallback (Temporal disabled).
func StartCancellationCron(ctx context.Context) {
	go func() {
		runCancellationSweep(ctx)
		ticker := time.NewTicker(cancellationSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("cancellation-sweep: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runCancellationSweep(ctx)
			}
		}
	}()
	log.Println("cancellation-sweep: cron started (interval=2m)")
}
