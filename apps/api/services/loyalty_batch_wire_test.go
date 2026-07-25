package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestApplyTxn_CreditWritesBatch_DebitConsumes(t *testing.T) {
	db := setupBatchDB(t)
	u := uuid.New()
	cfg := GetLoyaltyConfig(db)

	_, created, err := applyLoyaltyTxnInTx(db, u, 300, models.LoyaltyCredit, models.LoyaltySourceOrder, nil, "earn", "credit-1", nil, cfg)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, 300.0, batchRemaining(t, db, u)) // one lot of 300

	_, _, err = applyLoyaltyTxnInTx(db, u, 120, models.LoyaltyDebit, models.LoyaltySourceRedeem, nil, "redeem", "debit-1", nil, cfg)
	require.NoError(t, err)
	require.Equal(t, 180.0, batchRemaining(t, db, u)) // 300 − 120
}
