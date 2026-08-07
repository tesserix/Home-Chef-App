package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// retired_gateway_columns.go — drops the payout columns the retired gateway
// owned (#1086). The tesserix-k8s schema already declares them dropped, but the
// bootstrap CronJob skips a provisioned database, and AutoMigrate never removes
// a column the models stopped declaring — so production kept carrying them.

type retiredColumn struct{ table, column string }

// zeroValueLiterals is what a row that was never written looks like once cast to
// text: the empty string, the JSON literal a nil payload serialised to, and a
// boolean false — 'false' on Postgres, '0' on SQLite. A flag nobody ever set
// records nothing, and keeping the column on that basis kept it forever.
const zeroValueLiterals = `'', 'null', 'false', 'f', '0'`

// retiredGatewayColumns is what the models dropped in #1086/#1105 and the
// database did not.
func retiredGatewayColumns() []retiredColumn {
	return []retiredColumn{
		{"chef_profiles", "razorpay_account_id"},
		{"chef_profiles", "razorpay_product_id"},
		{"chef_profiles", "razorpay_settlement_status"},
		{"chef_profiles", "razorpay_settlement_requirements"},
		{"chef_profiles", "razorpay_stakeholder_created"},
		{"delivery_partners", "razorpay_account_id"},
	}
}

// dropRetiredGatewayColumns removes each column only once it holds nothing.
//
// The guard is the point: an account id still sitting in one of these is a
// payout destination somebody may yet need to read, and a dropped column cannot
// be asked again. A column that fails the guard is logged and kept, so the boot
// still succeeds and the row can be looked at by hand.
func dropRetiredGatewayColumns(db *gorm.DB, cols []retiredColumn) error {
	m := db.Migrator()
	for _, c := range cols {
		if !m.HasTable(c.table) || !m.HasColumn(c.table, c.column) {
			continue
		}
		var populated int64
		if err := db.Raw(fmt.Sprintf(
			`SELECT count(*) FROM %s WHERE %s IS NOT NULL AND CAST(%s AS TEXT) NOT IN (%s)`,
			c.table, c.column, c.column, zeroValueLiterals)).Scan(&populated).Error; err != nil {
			return fmt.Errorf("inspecting %s.%s: %w", c.table, c.column, err)
		}
		if populated > 0 {
			log.Printf("retired gateway columns: %s.%s still holds %d value(s) — kept", c.table, c.column, populated)
			continue
		}
		if err := db.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, c.table, c.column)).Error; err != nil {
			return fmt.Errorf("dropping %s.%s: %w", c.table, c.column, err)
		}
		log.Printf("retired gateway columns: dropped %s.%s", c.table, c.column)
	}
	return nil
}
