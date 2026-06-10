package services

// statement_cron.go — daily cron that issues weekly settlement statements.
//
// Mirrors the FSSAI reminder cron: fire once on startup (so a fresh deploy /
// cold start catches up immediately), then every 24h. Each run generates
// statements for the most recently closed Mon–Sun week (IST). Because
// homechef-api can scale to zero, "fire on startup" is what makes this
// reliable — every wake re-checks the closed week, and idempotency keeps it
// from double-issuing. (Wave 4 min_scale:1 removes the cold-start dependence.)

import (
	"context"
	"log"
	"time"
)

const statementCronInterval = 24 * time.Hour

// StartWeeklyStatementCron launches the background statement generator. Returns
// immediately; the cron lives for the life of ctx. Pass main.go's root context
// so SIGTERM shuts it down cleanly.
func StartWeeklyStatementCron(ctx context.Context) {
	go func() {
		runWeeklyStatementScan(ctx)

		ticker := time.NewTicker(statementCronInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("weekly-statement: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runWeeklyStatementScan(ctx)
			}
		}
	}()
	log.Println("weekly-statement: cron started (interval=24h, week=Mon–Sun IST)")
}

func runWeeklyStatementScan(ctx context.Context) {
	defer func() {
		// A cron must never take down a serving pod — recover and log.
		if r := recover(); r != nil {
			log.Printf("weekly-statement: panic recovered: %v", r)
		}
	}()

	weekStart, weekEnd := MostRecentClosedWeek(time.Now())
	issued, err := GenerateWeeklyStatements(ctx, weekStart, weekEnd)
	if err != nil {
		log.Printf("weekly-statement: scan failed for week=%s: %v",
			weekStart.Format("2006-01-02"), err)
		return
	}
	if issued > 0 {
		log.Printf("weekly-statement: scan complete, issued=%d for week=%s",
			issued, weekStart.Format("2006-01-02"))
	}
}
