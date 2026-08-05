package handlers

// fssai_webhook_test.go — the gateway's own account of a capture.
//
// Confirmation used to happen only when the chef came back from the hosted
// checkout. A chef who paid and closed the browser left us holding their money
// against a request that was never submitted, never emailed to onboarding, and
// invisible in the admin queue. These pin the fallback that closes that.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupFssaiWebhookDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE fssai_requests (mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
		id TEXT PRIMARY KEY, chef_id TEXT, user_id TEXT, status TEXT DEFAULT 'awaiting_payment',
		kitchen_name TEXT DEFAULT '', applicant_name TEXT DEFAULT '', contact_phone TEXT DEFAULT '',
		contact_email TEXT DEFAULT '', address_line1 TEXT DEFAULT '', address_line2 TEXT DEFAULT '',
		city TEXT DEFAULT '', state TEXT DEFAULT '', postal_code TEXT DEFAULT '', term_years INTEGER DEFAULT 1,
		fee_amount REAL DEFAULT 0, fee_tax REAL DEFAULT 0, fee_total REAL DEFAULT 0, currency TEXT DEFAULT 'INR',
		payment_ref TEXT DEFAULT '', gateway_order TEXT DEFAULT '', paid_at DATETIME,
		application_ref TEXT DEFAULT '', registration_no TEXT DEFAULT '', admin_notes TEXT DEFAULT '',
		rejected_reason TEXT DEFAULT '', info_requested TEXT DEFAULT '', info_requested_at DATETIME,
		license_file_url TEXT DEFAULT '', license_file_name TEXT DEFAULT '',
		submitted_at DATETIME, filed_at DATETIME, issued_at DATETIME,
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE fssai_request_documents (id TEXT PRIMARY KEY, request_id TEXT,
		kind TEXT, file_url TEXT, file_name TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME)`).Error)
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

// seedFssaiRequest writes an unpaid request awaiting the given gateway order,
// with the two documents FSSAI requires already attached — which is the only
// state checkout will mint an order for.
func seedFssaiRequest(t *testing.T, db *gorm.DB, gatewayOrder, mode string, withDocs bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO fssai_requests (id, chef_id, user_id, status, gateway_order, mode, fee_total, kitchen_name)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id.String(), uuid.New().String(), uuid.New().String(),
		models.FssaiAwaitingPayment, gatewayOrder, mode, 177.00, "Saffron").Error)
	if withDocs {
		for _, kind := range []string{models.FssaiDocPhoto, models.FssaiDocIdentity} {
			require.NoError(t, db.Exec(
				`INSERT INTO fssai_request_documents (id, request_id, kind, file_url) VALUES (?, ?, ?, ?)`,
				uuid.New().String(), id.String(), kind, "chefs/x/"+kind+".jpg").Error)
		}
	}
	return id
}

func fssaiStatusOf(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var row models.FssaiRequest
	require.NoError(t, db.First(&row, "id = ?", id).Error)
	return row.Status
}

// The whole point: a capture the chef never came back to confirm still submits
// the request.
func TestConfirmFssaiRequestFromWebhook_SubmitsAnAbandonedCheckout(t *testing.T) {
	db := setupFssaiWebhookDB(t)
	id := seedFssaiRequest(t, db, "cf_order_1", "live", true)

	h := &PaymentHandler{}
	confirmed, err := h.confirmFssaiRequestFromWebhook("cf_order_1", "cf_pay_1", "live")
	require.NoError(t, err)
	require.True(t, confirmed)
	require.Equal(t, models.FssaiSubmitted, fssaiStatusOf(t, db, id))
}

// A redelivered webhook, or one racing the chef's own confirm, must settle on
// one submission — not a second onboarding email.
func TestConfirmFssaiRequestFromWebhook_IsIdempotent(t *testing.T) {
	db := setupFssaiWebhookDB(t)
	id := seedFssaiRequest(t, db, "cf_order_2", "live", true)

	h := &PaymentHandler{}
	first, err := h.confirmFssaiRequestFromWebhook("cf_order_2", "cf_pay_1", "live")
	require.NoError(t, err)
	require.True(t, first)

	second, err := h.confirmFssaiRequestFromWebhook("cf_order_2", "cf_pay_2", "live")
	require.NoError(t, err)
	require.False(t, second, "the request is already submitted; there is nothing left to do")

	var row models.FssaiRequest
	require.NoError(t, db.First(&row, "id = ?", id).Error)
	require.Equal(t, models.FssaiSubmitted, row.Status)
	require.Equal(t, "cf_pay_1", row.PaymentRef, "the first capture stands")
}

// An order id belonging to something else must not be mistaken for a filing
// request — the fallback runs for every unmatched capture on the platform.
func TestConfirmFssaiRequestFromWebhook_IgnoresAnUnrelatedOrder(t *testing.T) {
	db := setupFssaiWebhookDB(t)
	seedFssaiRequest(t, db, "cf_order_3", "live", true)

	h := &PaymentHandler{}
	confirmed, err := h.confirmFssaiRequestFromWebhook("cf_order_somebody_else", "cf_pay_1", "live")
	require.NoError(t, err)
	require.False(t, confirmed)
}

// A sandbox capture must never settle a live request. The gateway order id is a
// UUID, but the mode filter is what makes that structural rather than lucky.
func TestConfirmFssaiRequestFromWebhook_IsScopedToMode(t *testing.T) {
	db := setupFssaiWebhookDB(t)
	id := seedFssaiRequest(t, db, "cf_order_4", "live", true)

	h := &PaymentHandler{}
	confirmed, err := h.confirmFssaiRequestFromWebhook("cf_order_4", "cf_pay_1", "test")
	require.NoError(t, err)
	require.False(t, confirmed)
	require.Equal(t, models.FssaiAwaitingPayment, fssaiStatusOf(t, db, id))
}

// Checkout will not mint an order without the documents, so this is unreachable
// — but if it ever fires we are holding money for a request we cannot file, and
// that needs a person, not a retry that hammers the webhook forever.
func TestConfirmFssaiRequestFromWebhook_DoesNotRetryWhenDocumentsAreMissing(t *testing.T) {
	db := setupFssaiWebhookDB(t)
	id := seedFssaiRequest(t, db, "cf_order_5", "live", false)

	h := &PaymentHandler{}
	confirmed, err := h.confirmFssaiRequestFromWebhook("cf_order_5", "cf_pay_1", "live")
	require.NoError(t, err, "a permanent problem must not be reported as transient")
	require.False(t, confirmed)
	require.Equal(t, models.FssaiAwaitingPayment, fssaiStatusOf(t, db, id))
}
