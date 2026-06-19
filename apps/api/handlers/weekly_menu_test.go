package handlers

import (
	"strings"
	"testing"

	"github.com/homechef/api/models"
)

// weekly_menu_test.go — #1. A publishable weekly menu must have "every offered
// (day × slot) filled" — no holes. validatePublishableGrid enforces that the
// filled cells form a complete rectangle over the days the chef uses × the slots
// they use: if any day is missing a slot that another day has, publishing is
// blocked. Drafts skip this; only publish enforces it.

func cell(day int, slot models.MealSlot, variant models.MealVariant) models.WeeklyMenuItem {
	return models.WeeklyMenuItem{DayOfWeek: day, Slot: slot, Variant: variant, Name: "Dish"}
}

func TestValidatePublishableGrid(t *testing.T) {
	const (
		sun = 0
		mon = 1
		tue = 2
		wed = 3
	)
	lunch, dinner := models.MealSlotLunch, models.MealSlotDinner
	veg, non := models.MealVariantVeg, models.MealVariantNonVeg

	t.Run("empty is rejected", func(t *testing.T) {
		if err := validatePublishableGrid(nil); err == nil {
			t.Fatal("expected an error for an empty menu")
		}
	})

	t.Run("complete rectangle (2 days × 2 slots) is ok", func(t *testing.T) {
		cells := []models.WeeklyMenuItem{
			cell(mon, lunch, veg), cell(mon, dinner, veg),
			cell(tue, lunch, non), cell(tue, dinner, non),
		}
		if err := validatePublishableGrid(cells); err != nil {
			t.Fatalf("expected ok, got %v", err)
		}
	})

	t.Run("lunch-only across days is ok (slot set is consistent)", func(t *testing.T) {
		cells := []models.WeeklyMenuItem{
			cell(mon, lunch, veg), cell(tue, lunch, veg), cell(wed, lunch, veg),
		}
		if err := validatePublishableGrid(cells); err != nil {
			t.Fatalf("expected ok, got %v", err)
		}
	})

	t.Run("single cell is ok", func(t *testing.T) {
		if err := validatePublishableGrid([]models.WeeklyMenuItem{cell(sun, dinner, veg)}); err != nil {
			t.Fatalf("expected ok, got %v", err)
		}
	})

	t.Run("a hole is rejected and names the missing cell", func(t *testing.T) {
		// Mon has lunch+dinner, but Tue only has lunch → Tue dinner is a hole.
		cells := []models.WeeklyMenuItem{
			cell(mon, lunch, veg), cell(mon, dinner, veg),
			cell(tue, lunch, veg),
		}
		err := validatePublishableGrid(cells)
		if err == nil {
			t.Fatal("expected an error for the Tuesday dinner hole")
		}
		msg := strings.ToLower(err.Error())
		if !strings.Contains(msg, "tue") || !strings.Contains(msg, "dinner") {
			t.Errorf("error should name the missing Tuesday dinner cell, got: %v", err)
		}
	})

	t.Run("variant doesn't create a hole (one variant per cell is enough)", func(t *testing.T) {
		// Mon-lunch is veg, Tue-lunch is nonveg — still a complete lunch-only grid.
		cells := []models.WeeklyMenuItem{cell(mon, lunch, veg), cell(tue, lunch, non)}
		if err := validatePublishableGrid(cells); err != nil {
			t.Fatalf("expected ok (variant is per-cell), got %v", err)
		}
	})
}
