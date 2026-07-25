package handlers

// checkout_credit_test.go — the read-only quote endpoint.

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func regQuote(r *gin.Engine, h *PaymentHandler) {
	r.POST("/payments/order/:orderId/quote", h.QuoteOrderCredit)
}

// quoteFlagsOn forces both rails on for the duration of a test, restoring the
// previous config afterwards.
func quoteFlagsOn(t *testing.T) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig = &config.Config{
		Environment:            "test",
		WalletCheckoutEnabled:  true,
		LoyaltyCheckoutEnabled: true,
	}
	t.Cleanup(func() { config.AppConfig = prev })
}

// seedQuoteOrder creates a customer with a funded wallet and an unpaid order.
func seedQuoteOrder(t *testing.T, db *gorm.DB, walletAmount float64) (uuid.UUID, uuid.UUID) {
	t.Helper()
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "pending", 1000, "", "")
	if walletAmount > 0 {
		_, err := services.CreditWallet(db, cust, walletAmount, models.WalletSourcePromo,
			nil, "seed", "seed-"+uuid.NewString(), nil)
		require.NoError(t, err)
	}
	return cust, orderID
}

// A customer may only quote their OWN order.
func TestQuoteOrderCredit_RejectsAnotherCustomersOrder(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	quoteFlagsOn(t)
	_, orderID := seedQuoteOrder(t, db, 500)

	w := callPay(uuid.New(), http.MethodPost, "/payments/order/"+orderID.String()+"/quote",
		regQuote, map[string]any{"useWallet": true})
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestQuoteOrderCredit_InvalidOrderID_400(t *testing.T) {
	setupPayDB(t)
	quoteFlagsOn(t)
	w := callPay(uuid.New(), http.MethodPost, "/payments/order/not-a-uuid/quote", regQuote, nil)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// The quote is read-only: calling it repeatedly must never move the wallet.
func TestQuoteOrderCredit_DebitsNothing(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	quoteFlagsOn(t)
	cust, orderID := seedQuoteOrder(t, db, 500)

	for i := 0; i < 3; i++ {
		w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/quote",
			regQuote, map[string]any{"useWallet": true, "useLoyalty": true})
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	}

	var balance float64
	require.NoError(t, db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, cust.String()).
		Scan(&balance).Error)
	require.Equal(t, 500.0, balance, "quoting must never debit")
}

// The order seeded by payOrder is subtotal 900 + tax 100 = total 1000, so credit
// may fund at most the 900 of food and the tax stays in cash.
func TestQuoteOrderCredit_ReturnsCapAndPayable(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	quoteFlagsOn(t)
	cust, orderID := seedQuoteOrder(t, db, 5000)

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/quote",
		regQuote, map[string]any{"useWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 900.0, body["redeemableCap"], "food only — tax is never redeemable")
	require.Equal(t, 900.0, body["walletApplied"])
	require.Equal(t, 100.0, body["payable"], "the tax, in cash")
	require.Equal(t, true, body["walletEnabled"])
}

// The client learns rail availability from the server, not from a compiled-in flag.
func TestQuoteOrderCredit_ReportsDisabledRails(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	quoteFlagsOn(t)
	config.AppConfig.WalletCheckoutEnabled = false
	cust, orderID := seedQuoteOrder(t, db, 500)

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/quote",
		regQuote, map[string]any{"useWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, false, body["walletEnabled"])
	require.Equal(t, 0.0, body["walletApplied"])
	require.Equal(t, 1000.0, body["payable"])
}
