package handlers

// chef_payout_save_test.go — what SavePayoutDetails does with a chef's bank
// details (#740/#1086).
//
// It was chef_payout_settlement_test.go, pinning the Razorpay v2 onboarding
// sequence (POST /accounts, /stakeholders, /products, PATCH /products/:pid) the
// save used to drive synchronously. #1086 removed that rail: the destination a
// chef is actually paid through is their Cashfree Easy Split vendor and the
// Cashfree Payouts beneficiary, both registered further down the same handler.
// What survives here is the part no rail change can move — UPI is refused, and
// the response advertises no Route settlement the chef could act on.
//
// DDL reuse: chefProfilesGuardDDL / chefGuardAuditDDL (chef_fulfillment_guard_test.go);
// setupDB (internal_users_test.go) provides the users table the handler preloads.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
)

// setupChefPayoutSettlementDB wires users + chef_profiles + audit_logs and
// seeds one chef/user pair, returning the db handle plus both ids.
func setupChefPayoutSettlementDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db := setupDB(t) // users table, shared with internal_users_test.go
	require.NoError(t, db.Exec(chefProfilesGuardDDL).Error)
	require.NoError(t, db.Exec(chefGuardAuditDDL).Error)

	userID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, first_name, last_name, phone, role) VALUES (?,?,?,?,?, 'chef')`,
		userID.String(), "chef@example.com", "Anita", "Rao", "9876543210").Error)
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name) VALUES (?,?,?)`,
		chefID.String(), userID.String(), "Anita's Kitchen").Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	// SavePayoutDetails' secret-storage goroutine calls config.IsDevelopment(),
	// which nil-derefs unless AppConfig is set. Set it once and never restore to
	// nil: that goroutine is fire-and-forget and can still be running after this
	// test function returns, so a save/restore-to-nil in t.Cleanup would race the
	// goroutine's read against the next test's restore. A permanent non-nil
	// zero-value Config is behaviourally identical to nil for every other
	// handler's `config.AppConfig == nil || !config.AppConfig.XEnabled`
	// feature-flag checks, so this can't affect unrelated tests in this package.
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{Environment: "test"}
	}

	return db, userID, chefID
}

func payoutSettlementRouter(userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.POST("/chef/payout", (&ChefHandler{}).SavePayoutDetails)
	return r
}

func postPayout(t *testing.T, userID uuid.UUID, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/chef/payout", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	payoutSettlementRouter(userID).ServeHTTP(w, req)
	return w
}

func bankTransferPayload() map[string]any {
	return map[string]any{
		"payoutMethod":      "bank_transfer",
		"bankAccountNumber": "1234567890",
		"bankIFSC":          "HDFC0000123",
		"bankAccountName":   "Anita Rao",
	}
}

// The chef's screen reads the Cashfree verdict (#1082). A Route connection flag
// alongside it could only ever contradict it.
func TestSavePayoutDetails_ResponseCarriesNoRouteFields(t *testing.T) {
	_, userID, _ := setupChefPayoutSettlementDB(t)
	w := postPayout(t, userID, bankTransferPayload())
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	for _, k := range []string{"razorpayConnected", "razorpayAccountId", "razorpaySettlementStatus", "razorpaySettlementError"} {
		require.NotContains(t, body, k, "the response must not advertise a Route settlement")
	}
	require.Contains(t, body, "payoutRegistration", "the chef still gets the verdict that decides whether they are paid")
}

// TestSavePayoutDetails_UpiRejected pins #767: UPI is not an accepted payout
// method. Neither Easy Split nor the Payouts rail disburses to a VPA, so a chef
// who picks UPI could never be paid — accepting it only strands their earnings
// behind an "onboarded" facade. The handler must reject the request at the edge
// (400) before touching the DB, and must never persist "upi".
func TestSavePayoutDetails_UpiRejected(t *testing.T) {
	db, userID, chefID := setupChefPayoutSettlementDB(t)

	w := postPayout(t, userID, map[string]any{
		"payoutMethod": "upi",
		"upiId":        "chef@upi",
	})
	require.Equal(t, http.StatusBadRequest, w.Code, "UPI is not payable — it must be rejected, not accepted")

	var method string
	require.NoError(t, db.Raw(`SELECT payout_method FROM chef_profiles WHERE id = ?`, chefID.String()).Row().Scan(&method))
	require.NotEqual(t, "upi", method, "the rejected UPI method must never be persisted")
}

// TestSavePayoutDetails_CannotSwitchToUpi pins #767 for a chef already set up on
// bank transfer: the switch is rejected and their existing method left intact,
// so they stay payable rather than being stranded on a rail nothing settles to.
func TestSavePayoutDetails_CannotSwitchToUpi(t *testing.T) {
	db, userID, chefID := setupChefPayoutSettlementDB(t)
	require.NoError(t, db.Exec(
		`UPDATE chef_profiles SET payout_method = 'bank_transfer', cashfree_vendor_id = 'vend_1', cashfree_vendor_status = 'ACTIVE' WHERE id = ?`,
		chefID.String()).Error)

	w := postPayout(t, userID, map[string]any{
		"payoutMethod": "upi",
		"upiId":        "chef@upi",
	})
	require.Equal(t, http.StatusBadRequest, w.Code, "switching a payable chef to UPI must be rejected")

	var method, vendorStatus string
	require.NoError(t, db.Raw(`SELECT payout_method, cashfree_vendor_status FROM chef_profiles WHERE id = ?`,
		chefID.String()).Row().Scan(&method, &vendorStatus))
	require.Equal(t, "bank_transfer", method, "the chef's payout method must remain bank_transfer")
	require.Equal(t, "ACTIVE", vendorStatus, "a rejected switch must leave the chef's verified vendor intact")
}

// The bank details are persisted to Secret Manager and the payout method to the
// DB even when no Cashfree client is configured (the test environment): the save
// itself must not depend on a gateway being reachable.
func TestSavePayoutDetails_PersistsMethodWithoutAGateway(t *testing.T) {
	db, userID, chefID := setupChefPayoutSettlementDB(t)

	w := postPayout(t, userID, bankTransferPayload())
	require.Equal(t, http.StatusOK, w.Code)

	var method string
	require.NoError(t, db.Raw(`SELECT payout_method FROM chef_profiles WHERE id = ?`, chefID.String()).Row().Scan(&method))
	require.Equal(t, "bank_transfer", method)
}
