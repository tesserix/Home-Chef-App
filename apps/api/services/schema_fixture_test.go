package services

// schema_fixture_test.go — build sqlite test tables FROM the GORM model rather
// than hand-rolling the DDL.
//
// Hand-rolled fixtures are the recurring failure mode in this package: a test
// table that omits a column the code writes makes the test pass while confirming
// the bug, and a table that omits a column the code READS (refund_amount,
// refunded_at, cancelled_at) hides money defects entirely. Every new column on
// models.Order has to be remembered in a dozen CREATE TABLE strings, and it never
// is.
//
// AutoMigrate cannot be used directly — the Postgres `gen_random_uuid()` column
// default does not parse on sqlite (that is why the hand-rolled convention
// exists). So this derives the column list from the same schema GORM writes
// through, and emits sqlite-flavoured DDL without the offending defaults. The
// fixture therefore tracks the model automatically.

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// createTableFor issues a sqlite CREATE TABLE covering every column GORM knows
// about for the model, so the fixture can never drift from the struct.
func createTableFor(t *testing.T, db *gorm.DB, model any) {
	t.Helper()
	s, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err, "parse schema for fixture")

	seen := map[string]bool{}
	cols := make([]string, 0, len(s.Fields))
	for _, f := range s.Fields {
		if f.DBName == "" || seen[f.DBName] {
			continue
		}
		seen[f.DBName] = true

		typ := "TEXT"
		switch f.FieldType.Kind().String() {
		case "bool", "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64":
			typ = "INTEGER"
		case "float32", "float64":
			typ = "REAL"
		}
		if strings.Contains(f.FieldType.String(), "time.Time") {
			typ = "DATETIME"
		}

		def := fmt.Sprintf("%s %s", f.DBName, typ)
		if f.PrimaryKey {
			def += " PRIMARY KEY"
		}
		cols = append(cols, def)
	}
	ddl := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", s.Table, strings.Join(cols, ", "))
	require.NoError(t, db.Exec(ddl).Error, "create fixture table %s", s.Table)
}

// createTablesFor is createTableFor over several models.
func createTablesFor(t *testing.T, db *gorm.DB, models ...any) {
	t.Helper()
	for _, m := range models {
		createTableFor(t, db, m)
	}
}
