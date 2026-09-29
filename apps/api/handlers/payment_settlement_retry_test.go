package handlers

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/homechef/api/services"
	"github.com/stretchr/testify/require"
)

func TestCashfreeSettleStatus_PersistenceFailureIsRetryable(t *testing.T) {
	require.Equal(t, http.StatusServiceUnavailable, cashfreeSettleStatus(fmt.Errorf("commit: %w", services.ErrPaymentSettlementPending)))
}
