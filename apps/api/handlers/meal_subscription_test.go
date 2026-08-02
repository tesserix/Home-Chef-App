package handlers

// meal_subscription_test.go — pins #904: every real subscribe request omits
// dayVariants, normaliseDayVariants returned the Go zero value "" for that case,
// and Postgres rejects '' as invalid JSON for a jsonb column, so DB.Create 500'd
// on literally every subscribe in production. The property the database actually
// enforces on this column is JSON validity, not string equality with any
// particular sentinel value — so these tests assert json.Valid, not `== "{}"`,
// deliberately so a future refactor that swaps the sentinel can't silently
// reintroduce the same class of bug.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormaliseDayVariants(t *testing.T) {
	t.Run("no overrides — nil map (real-world case, every current app request)", func(t *testing.T) {
		result := normaliseDayVariants(nil, []int64{1, 2, 3, 4, 5})
		assert.True(t, json.Valid([]byte(result)), "expected valid JSON, got %q", result)
	})

	t.Run("no overrides — empty map", func(t *testing.T) {
		result := normaliseDayVariants(map[string]string{}, []int64{1, 2, 3, 4, 5})
		assert.True(t, json.Valid([]byte(result)), "expected valid JSON, got %q", result)
	})

	t.Run("all entries filtered out — invalid day and invalid variant", func(t *testing.T) {
		result := normaliseDayVariants(map[string]string{"9": "veg", "1": "extra-spicy"}, []int64{1, 2, 3, 4, 5})
		assert.True(t, json.Valid([]byte(result)), "expected valid JSON, got %q", result)
	})

	t.Run("happy path — genuine overrides survive", func(t *testing.T) {
		result := normaliseDayVariants(map[string]string{"1": "veg", "6": "nonveg"}, []int64{1, 2, 3, 4, 5, 6})
		assert.True(t, json.Valid([]byte(result)), "expected valid JSON, got %q", result)

		var got map[string]string
		assert.NoError(t, json.Unmarshal([]byte(result), &got))
		assert.Equal(t, map[string]string{"1": "veg", "6": "nonveg"}, got)
	})
}
