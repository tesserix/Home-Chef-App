package services

// delivery_fee_from_test.go — D-01. The chef card advertised "Free delivery"
// while the order charged 39.12, because the API sent a hardcoded zero. The card
// cannot know the real fee (it is distance-based), so what it states must be a
// floor the checkout quote can only rise from — and only a fee that cannot rise
// at all may be called free outright.

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/homechef/api/models"
)

func TestDeliveryFeeFrom(t *testing.T) {
	t.Run("self-delivering chef quotes their own base", func(t *testing.T) {
		fee, flat := DeliveryFeeFrom(models.ChefProfile{
			OffersSelfDelivery: true, SelfDeliveryBaseFee: 30, SelfDeliveryPerKm: 8,
		})
		assert.Equal(t, 30.0, fee)
		assert.False(t, flat, "distance can lift it, so it is a floor and not a promise")
	})

	t.Run("no per-km charge is a fee that cannot rise", func(t *testing.T) {
		fee, flat := DeliveryFeeFrom(models.ChefProfile{
			OffersSelfDelivery: true, SelfDeliveryBaseFee: 0, SelfDeliveryPerKm: 0,
		})
		assert.Zero(t, fee)
		assert.True(t, flat, "the only case that earns an unqualified \"Free delivery\"")
	})

	t.Run("a zero base with a per-km charge is not free", func(t *testing.T) {
		// The shape that would otherwise reproduce D-01 exactly: a 0 fee shown as
		// "Free delivery" on a chef who charges by distance.
		_, flat := DeliveryFeeFrom(models.ChefProfile{
			OffersSelfDelivery: true, SelfDeliveryBaseFee: 0, SelfDeliveryPerKm: 8,
		})
		assert.False(t, flat)
	})

	t.Run("platform-carried delivery quotes the platform base", func(t *testing.T) {
		fee, flat := DeliveryFeeFrom(models.ChefProfile{OffersSelfDelivery: false})
		assert.Equal(t, GetPlatformPolicy().BaseDeliveryFee, fee)
		assert.False(t, flat, "a carrier prices the leg; no floor is a promise")
	})
}
