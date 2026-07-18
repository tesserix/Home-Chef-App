package handlers

// invoice_download_test.go — the signed receipt-link endpoints (#receipt parity).
//
// The public download route trusts a token instead of a session, so its guards
// are the whole security boundary. These pin them: a link is only minted for the
// owner of a paid order, and the download re-checks ownership + payment from the
// token, refusing forged/foreign/unpaid tokens before it ever renders a PDF.

import (
	"encoding/json"
	"strings"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func setupInvoiceLinkDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE orders (id text PRIMARY KEY, order_number text, customer_id text,
		status text, payment_status text, total real, refund_amount real DEFAULT 0,
		created_at datetime, updated_at datetime, deleted_at datetime)`).Error)
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	prevCfg := config.AppConfig
	config.AppConfig = &config.Config{BFFInternalHMACKey: []byte("test-hmac-key")}
	t.Cleanup(func() { config.AppConfig = prevCfg })
	return db
}

func seedInvoiceOrder(t *testing.T, db *gorm.DB, customer uuid.UUID, pay models.PaymentStatus) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, status, payment_status, total)
		VALUES (?,?,?,?,?,?)`, id.String(), "ORD-1", customer.String(),
		string(models.OrderStatusCancelled), string(pay), 150).Error)
	return id
}

func getLink(t *testing.T, order, asUser uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", asUser); c.Next() })
	r.GET("/orders/:id/invoice-link", (&OrderHandler{}).GetInvoiceDownloadLink)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/orders/"+order.String()+"/invoice-link", nil))
	return w
}

func TestInvoiceLink_MintedForOwnerOfAPaidOrder(t *testing.T) {
	db := setupInvoiceLinkDB(t)
	customer := uuid.New()
	order := seedInvoiceOrder(t, db, customer, models.PaymentRefunded) // paid-then-refunded

	w := getLink(t, order, customer)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct{ URL string }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Contains(t, body.URL, "/api/v1/invoice/", "returns a public download URL")

	// The minted token (last path segment) must verify back to this order + user.
	token := body.URL[strings.LastIndexByte(body.URL, '/')+1:]
	gotOrder, gotUser, err := services.VerifyInvoiceDownloadToken(token)
	require.NoError(t, err)
	require.Equal(t, order, gotOrder)
	require.Equal(t, customer, gotUser)
}

func TestInvoiceLink_UnpaidOrder_NoLink(t *testing.T) {
	db := setupInvoiceLinkDB(t)
	customer := uuid.New()
	order := seedInvoiceOrder(t, db, customer, models.PaymentPending)
	require.Equal(t, http.StatusBadRequest, getLink(t, order, customer).Code)
}

func TestInvoiceLink_OtherUsersOrder_NotFound(t *testing.T) {
	db := setupInvoiceLinkDB(t)
	order := seedInvoiceOrder(t, db, uuid.New(), models.PaymentCompleted)
	require.Equal(t, http.StatusNotFound, getLink(t, order, uuid.New()).Code)
}

func download(t *testing.T, token string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/invoice/:token", (&OrderHandler{}).DownloadInvoiceByToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/invoice/"+token, nil))
	return w
}

func TestInvoiceDownload_ForgedToken_Forbidden(t *testing.T) {
	setupInvoiceLinkDB(t)
	require.Equal(t, http.StatusForbidden, download(t, "not-a-real-token").Code)
}

// A validly-signed token whose user does NOT own the order (order deleted, or a
// crafted mismatch) must not serve — the download re-checks ownership.
func TestInvoiceDownload_ValidTokenButOrderNotOwned_NotFound(t *testing.T) {
	setupInvoiceLinkDB(t)
	// Token for an order that was never inserted (or belongs to someone else).
	token, err := services.MintInvoiceDownloadToken(uuid.New(), uuid.New())
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, download(t, token).Code)
}

// A valid token for an UNPAID order stops at the gate, before any PDF render.
func TestInvoiceDownload_ValidTokenUnpaidOrder_BadRequest(t *testing.T) {
	db := setupInvoiceLinkDB(t)
	customer := uuid.New()
	order := seedInvoiceOrder(t, db, customer, models.PaymentPending)
	token, _ := services.MintInvoiceDownloadToken(order, customer)
	require.Equal(t, http.StatusBadRequest, download(t, token).Code)
}
