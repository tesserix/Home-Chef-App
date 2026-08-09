package services

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every table in partitionedTables carries live and test rows side by side, so
// any read that serves a customer or a chef their "own" world must carry a mode
// scope. Two shipped without one:
//
//   - menu_items: customers were served the live menu plus the sandbox copy, so
//     every dish appeared twice (32 rows where the chef's app showed 16).
//   - weekly_menu_items: the same, on the tiffin grid — and the chef-facing
//     getter was unscoped too, so the chef saw each cell twice as well.
//
// Both were found by eye in the app, not by a test, and the second was missed
// even while fixing the first. This asserts the scope is present in the handlers
// that read these tables, so the next partitioned table cannot repeat it.
//
// It is deliberately a source check rather than a query test: the failure mode
// is a MISSING call, and only reading the source can prove absence.

// readScopeRequired maps a handler file to the partitioned models it reads on a
// user-facing path. Money and admin paths are excluded on purpose — they read
// each row's own snapshotted mode (ExcludeTestOrders), not the chef's current
// one, and admin tooling must be able to see sandbox rows.
var readScopeRequired = map[string][]string{
	"weekly_menu.go": {"WeeklyMenu", "WeeklyMenuItem"},
	"menu.go":        {"MenuItem"},
}

var scopeCall = regexp.MustCompile(`Scopes\([^)]*(?:ChefOwnModeScope|MatchOwningChefModeScope|CustomerVisibleModes|ModeScope|ExcludeTestOrders)`)

func TestPartitionedReadsCarryAModeScope(t *testing.T) {
	for file, models := range readScopeRequired {
		path := filepath.Join("..", "handlers", file)
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v (did the handler move? update readScopeRequired)", file, err)
		}
		body := string(src)

		for _, m := range models {
			// Prove the file actually reads the model, so a rename cannot make
			// this test pass by checking nothing.
			if !strings.Contains(body, "models."+m+"{}") && !strings.Contains(body, "models."+m+"\n") &&
				!strings.Contains(body, "[]models."+m) {
				t.Errorf("%s: expected a read of models.%s — update readScopeRequired if it moved", file, m)
			}
		}
		if !scopeCall.MatchString(body) {
			t.Errorf("%s reads %v but calls no mode scope. "+
				"A partitioned table read without one serves live and sandbox rows together.",
				file, models)
		}
	}
}

// The clone/purge list is the source of truth for what is partitioned. If a
// table joins it, someone must decide whether its user-facing reads need a
// scope — this fails when the list grows so that decision is forced.
func TestPartitionedTableListIsUnchanged(t *testing.T) {
	const known = 17
	if len(partitionedTables) != known {
		t.Fatalf("partitionedTables is now %d tables (was %d). "+
			"Audit the new table's customer- and chef-facing reads for a mode scope, "+
			"then update this count. menu_items and weekly_menu_items both shipped unscoped.",
			len(partitionedTables), known)
	}
}
