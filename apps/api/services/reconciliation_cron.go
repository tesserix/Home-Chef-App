package services

// reconciliation_cron.go — daily settlement reconciliation.
//
// Mirrors the FSSAI / weekly-statement crons: fire on startup, then every 24h.
// Each run reconciles the previous full day's (IST) payment activity against
// the gateways and logs any drift at ERROR level (so it surfaces in Cloud
// Logging alerts) plus Sentry. Read-only — it never mutates orders; a human /
// finance ops decides the remediation.

import (
	"context"
	"fmt"
	"log"
	"time"
)

const reconciliationInterval = 24 * time.Hour

// StartReconciliationCron launches the background reconciliation job. Returns
// immediately; lives for the life of ctx. Pass main.go's root context.
func StartReconciliationCron(ctx context.Context) {
	go func() {
		runReconciliationScan(ctx)

		ticker := time.NewTicker(reconciliationInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("reconciliation: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runReconciliationScan(ctx)
			}
		}
	}()
	log.Println("reconciliation: cron started (interval=24h, window=previous IST day)")
}

func runReconciliationScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("reconciliation: panic recovered: %v", r)
		}
	}()

	start, end := previousISTDay(time.Now())
	drifts, checked, err := ReconcileSettlements(ctx, start, end)
	if err != nil {
		log.Printf("reconciliation: scan failed for %s: %v", start.Format("2006-01-02"), err)
		return
	}

	if len(drifts) == 0 {
		log.Printf("reconciliation: clean — %d orders checked for %s",
			checked, start.In(istLocation).Format("2006-01-02"))
		return
	}

	// Drift found — log each at ERROR level and raise one aggregate alert.
	for _, d := range drifts {
		log.Printf("reconciliation DRIFT: order=%s (%s) provider=%s kind=%s platform=%.2f gateway=%.2f detail=%q",
			d.OrderNumber, d.OrderID, d.Provider, d.Kind, d.PlatformAmt, d.GatewayAmt, d.Detail)
	}
	CaptureBackgroundError(fmt.Errorf(
		"settlement reconciliation found %d drift(s) across %d orders for %s",
		len(drifts), checked, start.In(istLocation).Format("2006-01-02")))
}

// previousISTDay returns [start, end) in UTC for the full IST calendar day
// before now (00:00 IST yesterday .. 00:00 IST today).
func previousISTDay(now time.Time) (time.Time, time.Time) {
	ist := now.In(istLocation)
	todayStart := time.Date(ist.Year(), ist.Month(), ist.Day(), 0, 0, 0, 0, istLocation)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	return yesterdayStart.UTC(), todayStart.UTC()
}
