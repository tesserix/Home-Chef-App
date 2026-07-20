package handlers

// order_receipt_test.go — the customer receipt/invoice gate (#receipt).
//
// The old gate was delivered-only, so a customer could never pull a receipt for
// an order they PAID for and had cancelled — the reported bug. The gate is now
// "money was captured", which these pin.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func TestOrderHasReceipt_OnlyWhenMoneyWasCaptured(t *testing.T) {
	// Paid and paid-then-refunded both have a real payment to receipt.
	require.True(t, orderHasReceipt(models.Order{PaymentStatus: models.PaymentCompleted}))
	require.True(t, orderHasReceipt(models.Order{PaymentStatus: models.PaymentRefunded}),
		"a refunded order was still paid — the receipt records the charge AND the refund")

	// Nothing captured → nothing to receipt.
	require.False(t, orderHasReceipt(models.Order{PaymentStatus: models.PaymentPending}))
	require.False(t, orderHasReceipt(models.Order{PaymentStatus: models.PaymentFailed}))
}

// The endpoint refuses an unpaid order with 400 rather than generating a
// meaningless receipt, and refuses another customer's order with 404.
func TestGetOrderInvoicePDF_Gate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id text PRIMARY KEY, order_number text, customer_id text,
		status text, payment_status text, total real, refund_amount real DEFAULT 0,
		created_at datetime, updated_at datetime, deleted_at datetime)`).Error)
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	customer := uuid.New()
	pendingOrder := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, status, payment_status, total)
		VALUES (?,?,?,?,?,?)`, pendingOrder.String(), "ORD-PEND", customer.String(),
		string(models.OrderStatusPending), string(models.PaymentPending), 100).Error)

	call := func(orderID, asUser uuid.UUID) int {
		gin.SetMode(gin.TestMode)
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set("userID", asUser); c.Next() })
		r.GET("/orders/:id/invoice.pdf", (&OrderHandler{}).GetOrderInvoicePDF)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String()+"/invoice.pdf", nil))
		return w.Code
	}

	require.Equal(t, http.StatusBadRequest, call(pendingOrder, customer),
		"an unpaid order has no receipt yet")
	require.Equal(t, http.StatusNotFound, call(pendingOrder, uuid.New()),
		"a customer must not pull someone else's receipt")
}
