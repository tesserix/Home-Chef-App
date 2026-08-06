package models

import (
	"encoding/json"
	"testing"
)

// An ordinary dish must leave BakeryDetails nil so the insert writes SQL NULL.
// An empty string reaches jsonb as '' and Postgres rejects it (22P02), which
// broke every order containing a non-bakery line.
func TestParsedBakeryNilForOrdinaryLine(t *testing.T) {
	var line OrderItem
	if line.BakeryDetails != nil {
		t.Fatalf("an unconfigured line must be nil, got %q", *line.BakeryDetails)
	}
	if line.ParsedBakery() != nil {
		t.Fatal("ParsedBakery must be nil for an ordinary dish")
	}

	empty := ""
	line.BakeryDetails = &empty
	if line.ParsedBakery() != nil {
		t.Fatal("ParsedBakery must be nil for a blank snapshot")
	}
}

func TestParsedBakeryDecodesSnapshot(t *testing.T) {
	b, _ := json.Marshal(OrderItemBakery{
		WeightKg:   1.5,
		Selections: []BakerySelection{{Kind: "shape", Label: "Shape", Name: "Heart"}},
	})
	s := string(b)
	line := OrderItem{BakeryDetails: &s}

	got := line.ParsedBakery()
	if got == nil {
		t.Fatal("a configured line must decode")
	}
	if got.WeightKg != 1.5 || len(got.Selections) != 1 {
		t.Fatalf("snapshot lost detail: %+v", got)
	}
}
