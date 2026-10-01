package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"github.com/stretchr/testify/require"
)

func TestChefNetPayoutForInternationalInclusiveOrder(t *testing.T) {
	for _, currency := range []string{"AUD", "NZD"} {
		t.Run(currency, func(t *testing.T) {
			pricing := models.ComputeOrderPricing(models.PricingInput{
				Subtotal: 110, Tip: 5, Rates: models.TaxRates{Food: 10, FoodInclusive: true},
			})
			order := &models.Order{
				Currency: currency, Subtotal: pricing.Subtotal, Tax: pricing.Tax, TaxFood: pricing.TaxFood,
				TaxInclusive: true, CommissionRate: 0.06, ChefTip: 5,
			}
			require.Equal(t, 109.0, ChefNetPayoutFor(order), "saved subtotal is net of GST; restore food GST without Indian TDS")
		})
	}
}

func TestChefNetPayoutForIndiaPreservesWithholding(t *testing.T) {
	order := &models.Order{Currency: "INR", Subtotal: 100, Tax: 5, TaxFood: 5, CommissionRate: 0.06, ChefTip: 5}
	require.Equal(t, 102.9, ChefNetPayoutFor(order))
}

func TestInternationalStatementReadsFrozenTaxBasis(t *testing.T) {
	db := setupStatementRowsDB(t)
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, state) VALUES (?,?,?)`, chefID, uuid.New(), "VIC").Error)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	for _, currency := range []string{"AUD", "NZD"} {
		require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, chef_id, status, delivered_at, subtotal, tax, tax_food, chef_tip, total,
		payout_hold_status, commission_rate, currency, tax_inclusive)
		VALUES (?,?,?,'delivered',?,100,10,10,5,115,'release_eligible',0.06,?,true)`,
			uuid.New(), currency, chefID, start.Add(time.Hour), currency).Error)
	}
	rows, err := loadStatementOrderRows(start, start.AddDate(0, 0, 7))
	require.NoError(t, err)
	require.Len(t, rows, 2)
	for _, row := range rows {
		result := ComputeOrderEarnings(row.earningsInput(0.06), row.ChefState)
		require.Equal(t, 115.0, result.Gross)
		require.Equal(t, 109.0, result.NetPayout)
		require.Zero(t, result.TDS)
		require.Zero(t, result.CGST+result.SGST+result.IGST)
	}
}

func TestInternationalPendingPayoutMatchesOrder(t *testing.T) {
	db := setupReleaseDB(t)
	id := seedOrderHoldNet(t, db, models.PayoutHoldReleaseEligible, time.Now(), 115, 100, 10, 5, 0.06)
	require.NoError(t, db.Exec("UPDATE orders SET currency='NZD', tax_inclusive=true WHERE id=?", id).Error)
	rows, err := ListPendingPayouts(db, PendingFilter{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 109.0, rows[0].NetPayout)
}
