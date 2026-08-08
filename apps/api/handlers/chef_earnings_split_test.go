package handlers

// chef_earnings_split_test.go — #1087. A split order is settled at capture, and
// the amount that reached the chef's vendor account is stored on the order. The
// earnings screen recomputed the figure instead of reading it, so it showed a
// number the chef never received: the split share is the computed net MINUS the
// flat platform fee, and is capped at what was actually captured.
//
// This is the chef-facing half of the statement sunset. Once the statement stops
// being their payment instrument, the Earnings screen is the only place they are
// told what they were paid, so it has to be the settled figure.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChefSettledEarnings_SplitOrderReportsWhatWasActuallySettled(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()
	at := now.Add(-time.Hour)

	// ₹420.00 reached the vendor account — the computed net less the flat fee.
	seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 500, taxFood: 25, deliveredAt: at, splitPaise: 42_000,
	})

	totals, orders, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Len(t, orders, 1)

	assert.Equal(t, 420.0, orders[0].NetPayout,
		"the chef must be shown the money that reached their account, not a recomputation of it")
	assert.Equal(t, 420.0, totals.NetPayout)
}

func TestChefSettledEarnings_UnsplitOrderStillComputesItsNet(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()
	at := now.Add(-time.Hour)

	seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 500, taxFood: 25, deliveredAt: at,
	})

	_, orders, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Len(t, orders, 1)

	assert.NotEqual(t, 0.0, orders[0].NetPayout)
	assert.Greater(t, orders[0].NetPayout, 400.0,
		"Rail B has not settled yet, so the computed net is the only figure there is")
}

func TestChefSettledEarnings_MixedChefSeesBothRailsInOneTotal(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()
	at := now.Add(-time.Hour)

	seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 500, taxFood: 25, deliveredAt: at, splitPaise: 42_000,
	})
	seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 800, taxFood: 40, deliveredAt: at,
	})

	totals, orders, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	require.Len(t, orders, 2)
	require.Equal(t, 2, totals.OrdersCount)

	var sum float64
	for _, o := range orders {
		sum += o.NetPayout
	}
	assert.InDelta(t, sum, totals.NetPayout, 0.005,
		"the headline stays the sum of the rows even when the rows come from two rails")
}

// A split order is settled money. Escrow buckets describe money the platform is
// still holding, so a split order belongs in neither.
func TestChefSettledEarnings_SplitOrderIsNotHeldInEscrow(t *testing.T) {
	db, chef := setupSettledEarningsDB(t)
	now := time.Now()

	seedSettledOrder(t, db, chef.ID, settledOrderSeed{
		subtotal: 500, taxFood: 25, deliveredAt: now.Add(-time.Hour), splitPaise: 42_000,
	})

	totals, _, err := chefSettledEarnings(chef, now.AddDate(0, 0, -7), now, 0.06)
	require.NoError(t, err)
	assert.Zero(t, totals.Held, "the platform is not holding money it already paid out")
}
