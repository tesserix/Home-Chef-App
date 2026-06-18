package handlers

// Unit test for the prep-manifest rollup (#50). The query is DB-backed; this
// covers the pure grouping/counting that turns flat days into the manifest.

import "testing"

func TestBuildPrepManifest(t *testing.T) {
	rows := []prepRow{
		{Slot: "lunch", Variant: "veg", DishName: "Paneer", Status: "confirmed"},
		{Slot: "lunch", Variant: "veg", DishName: "Paneer", Status: "prepared"},
		{Slot: "lunch", Variant: "nonveg", DishName: "Butter Chicken", Status: "confirmed"},
		{Slot: "dinner", Variant: "veg", DishName: "Dal", Status: "confirmed"},
		{Slot: "dinner", Variant: "veg", DishName: "Dal", Status: "confirmed"},
		{Slot: "dinner", Variant: "veg", DishName: "Dal", Status: "prepared"},
	}

	lines, totals := buildPrepManifest(rows)

	// 3 distinct (slot,variant,dish) groups.
	if len(lines) != 3 {
		t.Fatalf("expected 3 manifest lines, got %d: %+v", len(lines), lines)
	}

	// Totals: 6 days, 3 lunch + 3 dinner, 2 prepared.
	if totals.Total != 6 || totals.Lunch != 3 || totals.Dinner != 3 || totals.Prepared != 2 {
		t.Fatalf("totals wrong: %+v", totals)
	}

	// Find the Paneer line: 2 total, 1 prepared.
	var paneer *prepManifestLine
	for i := range lines {
		if lines[i].DishName == "Paneer" {
			paneer = &lines[i]
		}
	}
	if paneer == nil || paneer.Total != 2 || paneer.Prepared != 1 {
		t.Fatalf("paneer line wrong: %+v", paneer)
	}

	// Dal: 3 total, 1 prepared, dinner/veg.
	for i := range lines {
		if lines[i].DishName == "Dal" {
			if lines[i].Total != 3 || lines[i].Prepared != 1 || lines[i].Slot != "dinner" || lines[i].Variant != "veg" {
				t.Fatalf("dal line wrong: %+v", lines[i])
			}
		}
	}

	// Empty input → no lines, zero totals.
	emptyLines, emptyTotals := buildPrepManifest(nil)
	if len(emptyLines) != 0 || emptyTotals.Total != 0 {
		t.Fatalf("empty input should yield no lines/zero totals, got %d / %+v", len(emptyLines), emptyTotals)
	}
}
