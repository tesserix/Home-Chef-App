package services

// statement_delivery_fee_test.go — the fee on a chef_delivery order is priced
// from the chef's OWN published rates and charged to the customer for a leg the
// chef drove. The settlement path treated it as the driver's money, so it never
// reached the chef the statement actually pays.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatementEarningsInput_ChefCarriedLegKeepsTheFee(t *testing.T) {
	row := statementOrderRow{
		ItemRevenue:     1000,
		Tax:             50,
		DeliveryFee:     70,
		ChefTip:         20,
		FulfillmentType: "chef_delivery",
		CommissionRate:  0.06,
	}
	in := row.earningsInput(0.10)

	assert.True(t, in.ChefEarnsDeliveryFee, "a chef-carried leg is the chef's income")
	assert.Equal(t, 70.0, in.DeliveryFee)
	assert.Equal(t, 0.06, in.CommissionRate, "the frozen per-row rate wins over the flat fallback")
}

func TestStatementEarningsInput_PlatformCarriedLegDoesNot(t *testing.T) {
	row := statementOrderRow{ItemRevenue: 1000, DeliveryFee: 70, FulfillmentType: "delivery"}
	assert.False(t, row.earningsInput(0.06).ChefEarnsDeliveryFee)

	pickup := statementOrderRow{ItemRevenue: 1000, FulfillmentType: "pickup"}
	assert.False(t, pickup.earningsInput(0.06).ChefEarnsDeliveryFee)
}

func TestStatementEarningsInput_UsesTheFeeTheCustomerActuallyPaid(t *testing.T) {
	// #703 lowered the fee at accept and already refunded the difference; paying
	// the chef the original estimate would over-pay them.
	final := 40.0
	row := statementOrderRow{DeliveryFee: 70, DeliveryFeeFinal: &final, FulfillmentType: "chef_delivery"}
	assert.Equal(t, 40.0, row.earningsInput(0.06).DeliveryFee)
}

func TestLoadStatementOrderRows_ProjectsDeliveryAttribution(t *testing.T) {
	db := setupStatementRowsDB(t)
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name, state) VALUES (?,?,?,?)`,
		chefID.String(), uuid.New().String(), "Saffron Home Kitchen", "KA").Error)

	weekStart := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal, tax, total,
		   delivery_fee, delivery_fee_final, fulfillment_type, commission_rate)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		uuid.NewString(), "SELF-DELIVERED", chefID.String(), "delivered", weekStart.Add(36*time.Hour),
		500.0, 25.0, 630.0, 70.0, 40.0, "chef_delivery", 0.06).Error)

	rows, err := loadStatementOrderRows(weekStart, weekStart.AddDate(0, 0, 7))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	assert.Equal(t, "chef_delivery", rows[0].FulfillmentType)
	require.NotNil(t, rows[0].DeliveryFeeFinal)
	assert.Equal(t, 40.0, *rows[0].DeliveryFeeFinal)
	assert.Equal(t, 40.0, rows[0].earningsInput(0.06).DeliveryFee)
}
