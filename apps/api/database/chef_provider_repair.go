package database

import (
	"fmt"
	"log"
	"strings"

	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// normalizeConfiguredProviders moves every chef and delivery partner onto a
// gateway that can still take money (#1122).
//
// A chef's payment_provider is CONFIGURATION — it picks the rail the next order
// is routed through — unlike an order's, which is the factual record of who took
// the money and must never be restamped. Since #1086 the retired gateway can settle nothing,
// so a chef left on it is a payout that silently cannot happen.
//
// Idempotent and re-run every boot, because AutoMigrate cannot express it and the
// bootstrap CronJob does not replay DDL against a provisioned database (#1127).
func normalizeConfiguredProviders(db *gorm.DB) error {
	allowed := models.SelectableChefProviders()
	placeholders := "?" + strings.Repeat(",?", len(allowed)-1)
	args := make([]any, 0, len(allowed)+1)
	args = append(args, models.PreferredChefPaymentProvider)
	for _, p := range allowed {
		args = append(args, p)
	}

	for _, table := range []string{"chef_profiles", "delivery_partners"} {
		if !db.Migrator().HasTable(table) {
			continue
		}
		res := db.Exec(fmt.Sprintf(
			`UPDATE %s SET payment_provider = ? WHERE LOWER(TRIM(COALESCE(payment_provider, ''))) NOT IN (%s)`,
			table, placeholders), args...)
		if res.Error != nil {
			return fmt.Errorf("normalizing %s.payment_provider: %w", table, res.Error)
		}
		if res.RowsAffected > 0 {
			log.Printf("provider repair: moved %d %s row(s) to %s",
				res.RowsAffected, table, models.PreferredChefPaymentProvider)
		}
	}
	return nil
}
