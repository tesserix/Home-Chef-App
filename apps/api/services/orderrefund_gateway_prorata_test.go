package services

// orderrefund_gateway_prorata_test.go — the funding-rail split on refund.
//
// Every rail that funded the order gets back its exact proportion of whatever is
// refunded. The previous behaviour was gateway-first: it refunded the provider up
// to the captured amount and only the EXCESS returned as store credit, so on a
// partial refund a credit-funded order returned pure cash and the customer's
// wallet slice came back only once the cash ran out.

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services/orderrefund"
)

// seedCreditFundedOrder builds a paid Cashfree order funded by wallet + loyalty + card.
func seedCreditFundedOrder(t *testing.T, db *gorm.DB, total, wallet, loyalty float64) *models.Order {
	t.Helper()
	o := &models.Order{
		ID: uuid.New(), OrderNumber: "ORD-C", CustomerID: uuid.New(), ChefID: uuid.New(),
		Status: models.OrderStatusCancelled, PaymentStatus: models.PaymentCompleted,
		PaymentProvider: "cashfree", GatewayOrderID: "cf_ord_credit",
		Total: total, WalletApplied: wallet, LoyaltyApplied: loyalty,
	}
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status,
		payment_provider, gateway_order_id, total, wallet_applied, loyalty_applied,
		wallet_refunded, loyalty_refunded, refund_amount) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		o.ID.String(), o.OrderNumber, o.CustomerID.String(), o.ChefID.String(), string(o.Status),
		string(models.PaymentCompleted), "cashfree", "cf_ord_credit", total, wallet, loyalty,
		0.0, 0.0, 0.0).Error)
	return o
}

func walletBalanceOf(t *testing.T, db *gorm.DB, userID uuid.UUID) float64 {
	t.Helper()
	var b float64
	require.NoError(t, db.Raw(`SELECT COALESCE(balance,0) FROM wallets WHERE user_id = ?`,
		userID.String()).Scan(&b).Error)
	return b
}

// The owner's worked example, end to end: wallet 400 + loyalty 100 + card 500,
// refunding half. Each rail gets exactly its share.
func TestOrderRefundGateway_SplitsProRataAcrossRails(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedCreditFundedOrder(t, db, 1000, 400, 100)
	spy := withCashfreeRefundSpy(t, http.StatusOK)

	_, err := OrderRefundGateway{}.RefundPayment(context.Background(), orderrefund.GatewayRequest{
		OrderID: o.ID, Amount: 500, Reason: "cancelled", Actor: "customer",
		IdempotencyKey: "refund:" + o.ID.String() + ":full",
	})
	require.NoError(t, err)

	require.Equal(t, 25000, spy.amountPaise, "card slice only — 250.00")
	require.Equal(t, 250.0, walletBalanceOf(t, db, o.CustomerID),
		"wallet 200.00 + loyalty 100.00-share 50.00, both to the wallet")

	var got models.Order
	require.NoError(t, db.First(&got, "id = ?", o.ID).Error)
	require.Equal(t, 200.0, got.WalletRefunded)
	require.Equal(t, 50.0, got.LoyaltyRefunded)
}

// The loyalty slice returns as wallet RUPEES, not as restored points.
func TestOrderRefundGateway_LoyaltySliceReturnsAsWalletCredit(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedCreditFundedOrder(t, db, 1000, 0, 200)
	withCashfreeRefundSpy(t, http.StatusOK)

	_, err := OrderRefundGateway{}.RefundPayment(context.Background(), orderrefund.GatewayRequest{
		OrderID: o.ID, Amount: 1000, Reason: "cancelled", Actor: "customer",
		IdempotencyKey: "refund:" + o.ID.String() + ":full",
	})
	require.NoError(t, err)
	require.Equal(t, 200.0, walletBalanceOf(t, db, o.CustomerID))

	acct, err := LoyaltyBalance(db, o.CustomerID)
	require.NoError(t, err)
	require.Equal(t, 0.0, acct.Balance, "points are NOT restored — their value came back as credit")
}

// An order with no credit behaves exactly as before: everything to the gateway.
func TestOrderRefundGateway_NoCreditGoesEntirelyToTheGateway(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedCreditFundedOrder(t, db, 500, 0, 0)
	spy := withCashfreeRefundSpy(t, http.StatusOK)

	_, err := OrderRefundGateway{}.RefundPayment(context.Background(), orderrefund.GatewayRequest{
		OrderID: o.ID, Amount: 500, Reason: "cancelled", Actor: "customer",
		IdempotencyKey: "refund:" + o.ID.String() + ":full",
	})
	require.NoError(t, err)
	require.Equal(t, 50000, spy.amountPaise)
	require.Equal(t, 0.0, walletBalanceOf(t, db, o.CustomerID))
}

// Two successive partial refunds must EACH credit the wallet — a per-order
// idempotency key would dedup the second into silence and lose the money.
func TestOrderRefundGateway_SuccessivePartialsEachCreditTheWallet(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedCreditFundedOrder(t, db, 1000, 500, 0)
	withCashfreeRefundSpy(t, http.StatusOK)

	for _, scope := range []string{"issue:a", "issue:b"} {
		var fresh models.Order
		require.NoError(t, db.First(&fresh, "id = ?", o.ID).Error)
		_, err := OrderRefundGateway{}.RefundPayment(context.Background(), orderrefund.GatewayRequest{
			OrderID: o.ID, Amount: 200, Reason: "issue", Actor: "admin",
			IdempotencyKey: "refund:" + o.ID.String() + ":" + scope,
		})
		require.NoError(t, err)
	}
	require.Equal(t, 200.0, walletBalanceOf(t, db, o.CustomerID),
		"two 200.00 refunds x 50% wallet funding = 100.00 each")
}

// Re-driving the SAME logical refund must not credit twice.
func TestOrderRefundGateway_SameKeyReplayCreditsOnce(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedCreditFundedOrder(t, db, 1000, 500, 0)
	withCashfreeRefundSpy(t, http.StatusOK)

	key := "refund:" + o.ID.String() + ":full"
	for i := 0; i < 2; i++ {
		var fresh models.Order
		require.NoError(t, db.First(&fresh, "id = ?", o.ID).Error)
		_, err := OrderRefundGateway{}.RefundPayment(context.Background(), orderrefund.GatewayRequest{
			OrderID: o.ID, Amount: 400, Reason: "cancelled", Actor: "customer", IdempotencyKey: key,
		})
		require.NoError(t, err)
	}
	require.Equal(t, 200.0, walletBalanceOf(t, db, o.CustomerID), "credited once, not twice")
}

// The card slice can never exceed what was actually captured at the provider.
func TestOrderRefundGateway_CardSliceNeverExceedsTheCapture(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedCreditFundedOrder(t, db, 1000, 700, 200) // only 100.00 ever captured
	spy := withCashfreeRefundSpy(t, http.StatusOK)

	_, err := OrderRefundGateway{}.RefundPayment(context.Background(), orderrefund.GatewayRequest{
		OrderID: o.ID, Amount: 1000, Reason: "cancelled", Actor: "customer",
		IdempotencyKey: "refund:" + o.ID.String() + ":full",
	})
	require.NoError(t, err)
	require.LessOrEqual(t, spy.amountPaise, 10000, "cannot refund more than was captured")
	require.Equal(t, 900.0, walletBalanceOf(t, db, o.CustomerID))
}
