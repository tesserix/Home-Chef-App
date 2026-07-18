package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The home-tiffin accept handshake (#709): the chef confirms the customer's
// requested time or proposes a different one. These are the exact rules the
// customer's order detail renders, so the resolution must be precise.
func TestResolveAcceptFulfillmentTime(t *testing.T) {
	base := time.Date(2026, 7, 18, 13, 0, 0, 0, time.UTC)

	t.Run("no times → as soon as ready", func(t *testing.T) {
		got, status := ResolveAcceptFulfillmentTime(nil, nil)
		require.Nil(t, got)
		require.Equal(t, "", status)
	})

	t.Run("customer requested, chef silent → confirmed as-is", func(t *testing.T) {
		req := base
		got, status := ResolveAcceptFulfillmentTime(&req, nil)
		require.Equal(t, "confirmed", status)
		require.Equal(t, base, got.UTC())
	})

	t.Run("chef supplies same time → confirmed", func(t *testing.T) {
		req := base
		chef := base.Add(5 * time.Minute) // within 10-min tolerance
		got, status := ResolveAcceptFulfillmentTime(&req, &chef)
		require.Equal(t, "confirmed", status)
		require.Equal(t, chef.UTC(), got.UTC())
	})

	t.Run("chef supplies a clearly later time → proposed", func(t *testing.T) {
		req := base
		chef := base.Add(45 * time.Minute)
		got, status := ResolveAcceptFulfillmentTime(&req, &chef)
		require.Equal(t, "proposed", status)
		require.Equal(t, chef.UTC(), got.UTC())
	})

	t.Run("ASAP order, chef proposes a concrete time → confirmed (no request to differ from)", func(t *testing.T) {
		chef := base.Add(45 * time.Minute)
		got, status := ResolveAcceptFulfillmentTime(nil, &chef)
		require.Equal(t, "confirmed", status)
		require.Equal(t, chef.UTC(), got.UTC())
	})
}
