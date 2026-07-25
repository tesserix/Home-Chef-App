package services

// loyalty_admin_test.go — admin manual grant/adjust of a customer's points
// balance (#40 Task 8). Positive points credit, negative points debit; both
// paths run through the same dated-lot ledger as earn/redeem so balance and
// FIFO expiry stay consistent no matter how the points arrived.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAdminAdjustLoyalty_CreditAndDebit(t *testing.T) {
	db := setupLoyaltyDB(t)
	u, admin := uuid.New(), uuid.New()

	_, err := AdminAdjustLoyalty(db, u, 500, "goodwill", admin)
	require.NoError(t, err)
	acct, err := LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 500.0, acct.Balance)

	_, err = AdminAdjustLoyalty(db, u, -200, "correction", admin)
	require.NoError(t, err)
	acct, err = LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 300.0, acct.Balance)
}
