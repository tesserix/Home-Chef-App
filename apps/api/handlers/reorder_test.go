package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// reorder_test.go — #238. Reorder re-adds a past order's lines to the cart. The
// tricky part is the #232 add-ons: the order stores a NAME snapshot, but the
// cart/order-create needs current option IDs. resolveModifierOptions matches the
// snapshot against the dish's live groups by (group name, option name) and flags
// any option that was renamed/removed/disabled so the client can ask the user to
// re-pick. These tests pin that matching so a reorder never silently changes the
// add-ons or carries a stale, invalid option ID into a new order.

// group builds a modifier group; opt() is shared with modifier_pricing_test.go.
func group(name string, opts ...models.ModifierOption) models.ModifierGroup {
	return models.ModifierGroup{Name: name, Options: opts}
}

func TestResolveModifierOptions(t *testing.T) {
	t.Run("exact match resolves ids and sums delta", func(t *testing.T) {
		groups := []models.ModifierGroup{
			group("Spice level", opt("Mild", 0, true), opt("Hot", 0, true)),
			group("Extras", opt("Cheese", 20, true), opt("Egg", 15, true)),
		}
		snap := []models.OrderItemModifier{
			{GroupName: "Spice level", OptionName: "Hot", PriceDelta: 0},
			{GroupName: "Extras", OptionName: "Cheese", PriceDelta: 20},
		}
		mods, allMatched := resolveModifierOptions(snap, groups)
		require.Len(t, mods, 2)
		assert.Equal(t, "Cheese", mods[1].OptionName)
		assert.Equal(t, 20.0, mods[1].PriceDelta)
		assert.NotEqual(t, uuid.Nil, mods[1].OptionID)
		assert.True(t, allMatched)
	})

	t.Run("renamed/removed option is dropped and flagged", func(t *testing.T) {
		groups := []models.ModifierGroup{
			group("Spice level", opt("Medium", 0, true)),
		}
		snap := []models.OrderItemModifier{{GroupName: "Spice level", OptionName: "Mild"}}
		mods, allMatched := resolveModifierOptions(snap, groups)
		assert.Empty(t, mods)
		assert.False(t, allMatched, "an unmatched option must flag the line for review")
	})

	t.Run("unavailable option is not carried over", func(t *testing.T) {
		groups := []models.ModifierGroup{
			group("Extras", opt("Cheese", 20, false)), // now unavailable
		}
		snap := []models.OrderItemModifier{{GroupName: "Extras", OptionName: "Cheese", PriceDelta: 20}}
		mods, allMatched := resolveModifierOptions(snap, groups)
		assert.Empty(t, mods)
		assert.False(t, allMatched)
	})

	t.Run("empty snapshot is fully matched", func(t *testing.T) {
		mods, allMatched := resolveModifierOptions(nil, []models.ModifierGroup{group("Extras", opt("Cheese", 20, true))})
		assert.Empty(t, mods)
		assert.True(t, allMatched)
	})
}
