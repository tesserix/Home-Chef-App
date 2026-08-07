package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// gatewayIDColumnRenames lists every table that carried a razorpay-named gateway
// id column before #1119, paired with the suffix to move it to.
func gatewayIDColumnRenames() [][2]string {
	var pairs [][2]string
	for _, table := range []string{
		"catering_requests", "chef_promotions", "group_order_participants",
		"meal_plans", "meal_trials", "orders", "tips",
	} {
		pairs = append(pairs, [2]string{table, "order_id"}, [2]string{table, "payment_id"})
	}
	return pairs
}

// migrateGatewayIDColumns moves razorpay_<suffix> onto gateway_<suffix>.
//
// The db-schema-bootstrap CronJob refuses to touch a provisioned database ("skipping
// DDL — application owns schema"), so the SQL rename in tesserix-k8s only ever runs
// for a fresh bootstrap. In production AutoMigrate added the new columns beside the
// old ones and left the ids behind; this is the path that actually repairs it.
//
// Two shapes, because both exist across environments: rename when the target is
// absent, otherwise copy across and drop. Never overwrites a populated target — an
// order minted after the new image rolled has the authoritative value there.
func migrateGatewayIDColumns(db *gorm.DB, pairs [][2]string) error {
	m := db.Migrator()
	for _, pair := range pairs {
		table, suffix := pair[0], pair[1]
		oldCol, newCol := "razorpay_"+suffix, "gateway_"+suffix

		if !m.HasTable(table) || !m.HasColumn(table, oldCol) {
			continue
		}

		// Raw DDL rather than the Migrator helpers: those want a parsed model, and
		// the models no longer declare the column being moved.
		if !m.HasColumn(table, newCol) {
			if err := db.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`,
				table, oldCol, newCol)).Error; err != nil {
				return fmt.Errorf("renaming %s.%s: %w", table, oldCol, err)
			}
			log.Printf("gateway id migration: renamed %s.%s -> %s", table, oldCol, newCol)
			continue
		}

		res := db.Exec(fmt.Sprintf(
			`UPDATE %s SET %s = %s WHERE coalesce(%s, '') = '' AND coalesce(%s, '') <> ''`,
			table, newCol, oldCol, newCol, oldCol))
		if res.Error != nil {
			return fmt.Errorf("backfilling %s.%s: %w", table, newCol, res.Error)
		}
		if err := db.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, table, oldCol)).Error; err != nil {
			return fmt.Errorf("dropping %s.%s: %w", table, oldCol, err)
		}
		log.Printf("gateway id migration: backfilled %d row(s) into %s.%s, dropped %s",
			res.RowsAffected, table, newCol, oldCol)
	}
	return nil
}
