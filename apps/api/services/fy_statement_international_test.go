package services

import (
	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestFYStatementUsesFrozenInternationalTaxBasis(t *testing.T) {
	for _, country := range []string{"NZ", "AU", "IN"} {
		t.Run(country, func(t *testing.T) {
			db := setupStatementRowsDB(t)
			createTablesFor(t, db, &models.ChefExpense{})
			chefID := uuid.New()
			require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, state, payout_country) VALUES (?,?,?,?)`, chefID, uuid.New(), "Auckland", country).Error)
			require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal, tax, tax_food, tax_service, chef_tip, total, commission_rate, currency, tax_inclusive) VALUES (?,?,?,'delivered',?,100,16,15,1,5,121,0.06,?,true)`, uuid.New(), country, chefID, time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), ReportingCurrency(country)).Error)
			stmt, err := ComputeFYStatement(chefID, 2026)
			require.NoError(t, err)
			require.Equal(t, 15.0, stmt.GSTCollected, "only food GST belongs to the chef")
			require.Equal(t, 120.0, stmt.GrossReceipts)
			if country == "IN" {
				require.Equal(t, 1.2, stmt.TDSWithheld)
				require.Equal(t, 112.8, stmt.NetEarnings)
			} else {
				require.Zero(t, stmt.TDSWithheld)
				require.Zero(t, stmt.CommissionCGST+stmt.CommissionSGST+stmt.CommissionIGST)
				require.Equal(t, 114.0, stmt.NetEarnings)
			}
		})
	}
}
