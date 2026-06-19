package handlers

// Unit tests for the add-on modifier validation + pricing (#232). Pure logic;
// the order handler that calls it is DB-backed.

import (
	"testing"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

func opt(name string, price float64, avail bool) models.ModifierOption {
	return models.ModifierOption{ID: uuid.New(), Name: name, PriceDelta: price, IsAvailable: avail}
}

func TestValidateAndPriceModifiers(t *testing.T) {
	// "Spice level": required single-choice.
	mild := opt("Mild", 0, true)
	hot := opt("Hot", 0, true)
	spice := models.ModifierGroup{Name: "Spice level", Required: true, MinSelect: 1, MaxSelect: 1, Options: []models.ModifierOption{mild, hot}}

	// "Extras": optional multi-choice (max 2).
	roti := opt("Extra roti", 10, true)
	cheese := opt("Extra cheese", 20, true)
	soldOut := opt("Paneer (sold out)", 30, false)
	extras := models.ModifierGroup{Name: "Extras", Required: false, MinSelect: 0, MaxSelect: 2, Options: []models.ModifierOption{roti, cheese, soldOut}}

	groups := []models.ModifierGroup{spice, extras}

	t.Run("required group, nothing selected → error", func(t *testing.T) {
		if _, _, err := validateAndPriceModifiers(groups, nil); err == nil {
			t.Fatal("expected error for missing required selection")
		}
	})

	t.Run("valid: spice + two extras prices the deltas", func(t *testing.T) {
		delta, snap, err := validateAndPriceModifiers(groups, []uuid.UUID{hot.ID, roti.ID, cheese.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if delta != 30 {
			t.Fatalf("delta = %v, want 30", delta)
		}
		if len(snap) != 3 {
			t.Fatalf("snapshot len = %d, want 3", len(snap))
		}
	})

	t.Run("exceeds max in a group → error", func(t *testing.T) {
		// spice ok (hot), but 3 extras > max 2 — need a 3rd available extra.
		extra3 := opt("Extra raita", 15, true)
		g := []models.ModifierGroup{spice, {Name: "Extras", MaxSelect: 2, Options: []models.ModifierOption{roti, cheese, extra3}}}
		if _, _, err := validateAndPriceModifiers(g, []uuid.UUID{hot.ID, roti.ID, cheese.ID, extra3.ID}); err == nil {
			t.Fatal("expected error for exceeding max selections")
		}
	})

	t.Run("unavailable option → error", func(t *testing.T) {
		if _, _, err := validateAndPriceModifiers(groups, []uuid.UUID{hot.ID, soldOut.ID}); err == nil {
			t.Fatal("expected error for selecting an unavailable option")
		}
	})

	t.Run("unknown option id → error", func(t *testing.T) {
		if _, _, err := validateAndPriceModifiers(groups, []uuid.UUID{hot.ID, uuid.New()}); err == nil {
			t.Fatal("expected error for an option not in the item's groups")
		}
	})

	t.Run("no groups, no selection → no error, zero delta", func(t *testing.T) {
		delta, snap, err := validateAndPriceModifiers(nil, nil)
		if err != nil || delta != 0 || len(snap) != 0 {
			t.Fatalf("got (%v, %v, %v), want (0, [], nil)", delta, snap, err)
		}
	})
}
