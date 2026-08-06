package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// #1052 — #1035 removed every customer surface for recurring tiffin, but the
// rows and endpoints stayed. These pin the two things that keep the leftovers
// inert: nothing new can be created, and nothing existing can generate orders
// (and so, once #281 lands, nothing can charge).

func withMealSubsEnabled(t *testing.T, enabled bool) {
	t.Helper()
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })
	config.AppConfig = &config.Config{MealSubscriptionsEnabled: enabled}
}

func TestMealSubscriptionsEnabled_DefaultsOffAndFailsClosed(t *testing.T) {
	// An unconfigured process must never be the one that starts billing people.
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })
	config.AppConfig = nil
	require.False(t, MealSubscriptionsEnabled(), "nil config must read as OFF")

	withMealSubsEnabled(t, false)
	require.False(t, MealSubscriptionsEnabled())

	withMealSubsEnabled(t, true)
	require.True(t, MealSubscriptionsEnabled())
}

func TestMealSubGeneratesOrders_InertWhileTheProductIsOff(t *testing.T) {
	withMealSubsEnabled(t, false)

	// The dangerous case: a row left ACTIVE by the pre-#1035 world. Without the
	// flag gate this would keep producing daily orders for a customer who has no
	// screen to stop it on.
	require.False(t, MealSubGeneratesOrders(models.MealSubStatusActive),
		"an ACTIVE legacy row must not generate orders while the product is off")

	for _, st := range []string{
		models.MealSubStatusTrialing,
		models.MealSubStatusPaused,
		models.MealSubStatusPastDue,
		models.MealSubStatusCancelled,
	} {
		require.False(t, MealSubGeneratesOrders(st))
	}
}

func TestMealSubGeneratesOrders_StillStatusScopedWhenOn(t *testing.T) {
	// Turning the product back on must not turn the status rules off with it.
	withMealSubsEnabled(t, true)

	require.True(t, MealSubGeneratesOrders(models.MealSubStatusActive))
	require.False(t, MealSubGeneratesOrders(models.MealSubStatusTrialing),
		"trialing never generated orders and still must not")
	require.False(t, MealSubGeneratesOrders(models.MealSubStatusPaused))
	require.False(t, MealSubGeneratesOrders(models.MealSubStatusPastDue))
	require.False(t, MealSubGeneratesOrders(models.MealSubStatusCancelled))
}

func TestCancelStaysAllowedWhileTheProductIsOff(t *testing.T) {
	// Whatever else is disabled, a customer or admin must always be able to STOP
	// a subscription — that is the whole point of #1052.
	withMealSubsEnabled(t, false)
	for _, st := range []string{
		models.MealSubStatusActive,
		models.MealSubStatusTrialing,
		models.MealSubStatusPaused,
		models.MealSubStatusPastDue,
	} {
		require.True(t, CanCancelMealSub(st), "cancel must remain reachable for %s", st)
	}
}
