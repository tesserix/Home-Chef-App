package services

// test_sync_classification_drift_test.go — #1147. classifyColumns refuses to
// clone a table carrying a column nobody classified, which is the right
// behaviour but only fires against a live database: the first person to learn a
// column was added is an admin whose flip has just failed in production. This
// test asks the models the same question at build time.

import (
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"

	"github.com/homechef/api/models"
)

// The model behind each cloned table. A new entry in clonedTableColumns without
// one here fails below, so the two cannot drift apart either.
var clonedTableModels = map[string]any{
	"menu_items":        &models.MenuItem{},
	"chef_schedules":    &models.ChefSchedule{},
	"weekly_menus":      &models.WeeklyMenu{},
	"weekly_menu_items": &models.WeeklyMenuItem{},
	"daily_menus":       &models.DailyMenu{},
	"daily_menu_items":  &models.DailyMenuItem{},
	"orders":            &models.Order{},
}

func modelColumns(t *testing.T, model any) []string {
	t.Helper()
	s, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)

	cols := make([]string, 0, len(s.DBNames))
	cols = append(cols, s.DBNames...)
	sort.Strings(cols)
	return cols
}

// A column added to a cloned table must be classified in the same change, not
// discovered by an admin whose flip failed in production.
func TestEveryClonedColumnIsClassified(t *testing.T) {
	// Collected across every table rather than failing at the first, so one run
	// names all the drift — the runtime guard can only ever report the table it
	// happens to reach first.
	drift := map[string][]string{}
	for table, classes := range clonedTableColumns {
		model, ok := clonedTableModels[table]
		require.Truef(t, ok, "table %s is cloned but has no model in clonedTableModels", table)

		for _, col := range modelColumns(t, model) {
			if _, ok := classes[col]; !ok {
				drift[table] = append(drift[table], col)
			}
		}
	}
	require.Emptyf(t, drift,
		"unclassified column(s) — classify them in test_sync_classification.go: %v", drift)
}
