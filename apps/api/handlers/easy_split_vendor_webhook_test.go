package handlers

// easy_split_vendor_webhook_test.go — #1083. Cashfree finishes verifying a
// chef's bank account hours after they saved it. Until this webhook existed the
// only thing that noticed was a 30-minute cron, so a chef who verified at 09:01
// stayed unpayable until 09:30 — and a chef whose verification FAILED was never
// told at all.

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/services"
)

// The webhook carries the status itself, so no gateway call is expected — the
// client exists only so the signature can be verified.
func withVendorWebhookSecret(t *testing.T) {
	t.Helper()
	withCashfreeGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("the vendor webhook must not call back to Cashfree")
		w.WriteHeader(http.StatusInternalServerError)
	})
}

func cfVendorWebhookBody(eventType, vendorID, status string) []byte {
	b, _ := json.Marshal(map[string]any{
		"type":       eventType,
		"event_time": time.Now().Format(time.RFC3339),
		"data":       map[string]any{"vendor_id": vendorID, "status": status},
	})
	return b
}

func vendorStatusOf(t *testing.T, db *gorm.DB, chefID uuid.UUID) string {
	t.Helper()
	var got string
	require.NoError(t, db.Raw(`SELECT cashfree_vendor_status FROM chef_profiles WHERE id = ?`, chefID.String()).Scan(&got).Error)
	return got
}

func registerVendor(t *testing.T, db *gorm.DB, chefID uuid.UUID, vendorID, status string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`UPDATE chef_profiles SET payout_method = 'bank_transfer', cashfree_vendor_id = ?, cashfree_vendor_status = ? WHERE id = ?`,
		vendorID, status, chefID.String()).Error)
}

func TestCashfreeVendorWebhook_ActivatesTheChefWithoutWaitingForTheCron(t *testing.T) {
	db := setupPayDB(t)
	h := &PaymentHandler{}
	withVendorWebhookSecret(t)
	chefID := payChef(t, db, payUser(t, db, "chef"))
	registerVendor(t, db, chefID, "hc_1", services.CashfreeVendorInBankValidation)

	w := cfWebhookRequest(t, h, cfVendorWebhookBody("VENDOR_STATUS_UPDATE", "hc_1", "ACTIVE"), true, nowUnix())

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, services.CashfreeVendorActive, vendorStatusOf(t, db, chefID))
}

// A failed verification is the case the chef most needs to hear about: nothing
// else on their screen would ever change.
func TestCashfreeVendorWebhook_RecordsAFailedVerification(t *testing.T) {
	db := setupPayDB(t)
	h := &PaymentHandler{}
	withVendorWebhookSecret(t)
	chefID := payChef(t, db, payUser(t, db, "chef"))
	registerVendor(t, db, chefID, "hc_2", services.CashfreeVendorInBankValidation)

	w := cfWebhookRequest(t, h, cfVendorWebhookBody("VENDOR_STATUS_UPDATE", "hc_2", "BLOCKED"), true, nowUnix())

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, services.CashfreeVendorBlocked, vendorStatusOf(t, db, chefID))
}

// An event for a vendor we have never registered is Cashfree's business, not
// ours. It must not error, and it must not touch anyone else's row.
func TestCashfreeVendorWebhook_IgnoresAnUnknownVendor(t *testing.T) {
	db := setupPayDB(t)
	h := &PaymentHandler{}
	withVendorWebhookSecret(t)
	chefID := payChef(t, db, payUser(t, db, "chef"))
	registerVendor(t, db, chefID, "hc_3", services.CashfreeVendorInBankValidation)

	w := cfWebhookRequest(t, h, cfVendorWebhookBody("VENDOR_STATUS_UPDATE", "hc_someone_else", "ACTIVE"), true, nowUnix())

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, services.CashfreeVendorInBankValidation, vendorStatusOf(t, db, chefID))
}

// The exact event-type string is Cashfree's to change, and it has changed
// before. Anything vendor-shaped carrying a vendor id and a status is acted on;
// the cron remains the backstop for anything that is not.
func TestCashfreeVendorWebhook_AcceptsTheEventUnderEitherName(t *testing.T) {
	for _, eventType := range []string{"VENDOR_STATUS_UPDATE", "VENDOR_STATUS_WEBHOOK", "EASY_SPLIT_VENDOR_UPDATE"} {
		t.Run(eventType, func(t *testing.T) {
			db := setupPayDB(t)
			h := &PaymentHandler{}
			withVendorWebhookSecret(t)
			chefID := payChef(t, db, payUser(t, db, "chef"))
			registerVendor(t, db, chefID, "hc_4", services.CashfreeVendorInBeneCreation)

			w := cfWebhookRequest(t, h, cfVendorWebhookBody(eventType, "hc_4", "ACTIVE"), true, nowUnix())

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, services.CashfreeVendorActive, vendorStatusOf(t, db, chefID))
		})
	}
}

// Cashfree names a failed validation in its own vocabulary; it is still a
// refusal, and the chef must be moved off pending.
func TestCashfreeVendorWebhook_RecordsABankValidationFailure(t *testing.T) {
	db := setupPayDB(t)
	h := &PaymentHandler{}
	withVendorWebhookSecret(t)
	chefID := payChef(t, db, payUser(t, db, "chef"))
	registerVendor(t, db, chefID, "hc_6", services.CashfreeVendorInBankValidation)

	body, _ := json.Marshal(map[string]any{
		"type":       "VENDOR_STATUS_UPDATE",
		"event_time": time.Now().Format(time.RFC3339),
		"data": map[string]any{
			"vendor_id": "hc_6", "status": services.CashfreeVendorBankValidationFailed,
			"remarks": "The name on the account does not match the PAN",
		},
	})
	w := cfWebhookRequest(t, h, body, true, nowUnix())

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, services.CashfreeVendorBankValidationFailed, vendorStatusOf(t, db, chefID))
}

// A truncated payload must leave a verified chef verified — reading a missing
// status as a change would strip their ability to be paid.
func TestCashfreeVendorWebhook_IgnoresAStatuslessPayload(t *testing.T) {
	db := setupPayDB(t)
	h := &PaymentHandler{}
	withVendorWebhookSecret(t)
	chefID := payChef(t, db, payUser(t, db, "chef"))
	registerVendor(t, db, chefID, "hc_5", services.CashfreeVendorActive)

	w := cfWebhookRequest(t, h, cfVendorWebhookBody("VENDOR_STATUS_UPDATE", "hc_5", ""), true, nowUnix())

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, services.CashfreeVendorActive, vendorStatusOf(t, db, chefID))
}
