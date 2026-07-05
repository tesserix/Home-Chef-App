package services

// group_net_payout_test.go — #546. The chef's held group-order transfer must be NET
// (food subtotal + tax − commission − TDS), the SAME basis as the order path
// (ComputeOrderEarnings) and the meal-plan-day path (perDayNetPayout). It previously
// paid the GROSS chef slice (subtotal + tax) with no commission or TDS — the platform
// earned nothing on group-order food and withheld no §194-O TDS.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestGroupNetPayout_NetsCommissionAndTDS(t *testing.T) {
	// subtotal 1000, tax 50 → gross = 1050
	// commission = 0.06 * 1000 = 60.00   (on food subtotal only)
	// tds        = 0.01 * 1050 = 10.50
	// net        = 1050 − 60 − 10.50 = 979.50
	g := &models.GroupOrder{Subtotal: 1000, Tax: 50}
	require.Equal(t, 979.50, groupNetPayout(g, 0.06))
}

func TestGroupNetPayout_IsBelowGrossSlice(t *testing.T) {
	// The whole point of #546: the chef is no longer paid the gross subtotal+tax.
	g := &models.GroupOrder{Subtotal: 200, Tax: 20}
	net := groupNetPayout(g, 0.06)
	require.Less(t, net, GroupChefPayout(g), "net below the gross chef slice (commission+TDS deducted)")
}

func TestGroupNetPayout_ClampsRate(t *testing.T) {
	// Out-of-range rate falls back to the flat default, like the order/day paths.
	g := &models.GroupOrder{Subtotal: 100, Tax: 10}
	require.Equal(t, groupNetPayout(g, DefaultCommissionRate), groupNetPayout(g, 0))
	require.Equal(t, groupNetPayout(g, DefaultCommissionRate), groupNetPayout(g, 1.5))
}
