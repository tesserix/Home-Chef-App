package services

// audit_retention_cron.go — prunes old audit_logs rows.
//
// The audit table grows unbounded; this cron caps it at a retention window
// (default ~13 months) so it stays queryable and cheap on the shared
// db-f1-micro. 13 months comfortably covers a financial year + a quarter of
// margin for any DPDP / dispute lookback. Override via AUDIT_RETENTION_DAYS.
//
// Same shape as the other crons: fire on startup, then daily, panic-recover.

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const (
	auditRetentionInterval    = 24 * time.Hour
	defaultAuditRetentionDays = 400 // ~13 months
)

// StartAuditRetentionCron launches the audit-log pruner. Returns immediately;
// lives for the life of ctx.
func StartAuditRetentionCron(ctx context.Context) {
	go func() {
		runAuditRetentionScan(ctx)

		ticker := time.NewTicker(auditRetentionInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("audit-retention: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runAuditRetentionScan(ctx)
			}
		}
	}()
	log.Printf("audit-retention: cron started (interval=24h, keep=%dd)", auditRetentionDays())
}

func runAuditRetentionScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("audit-retention: panic recovered: %v", r)
		}
	}()

	cutoff := time.Now().UTC().AddDate(0, 0, -auditRetentionDays())
	res := database.DB.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&models.AuditLog{})
	if res.Error != nil {
		log.Printf("audit-retention: prune failed (cutoff=%s): %v",
			cutoff.Format("2006-01-02"), res.Error)
		return
	}
	if res.RowsAffected > 0 {
		log.Printf("audit-retention: pruned %d row(s) older than %s",
			res.RowsAffected, cutoff.Format("2006-01-02"))
	}
}

// auditRetentionDays reads AUDIT_RETENTION_DAYS (positive int) or the default.
func auditRetentionDays() int {
	if v := os.Getenv("AUDIT_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultAuditRetentionDays
}
