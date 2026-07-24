package handlers

// chef_payout_settlement_test.go — #740 wiring gap (SDD task 5b).
//
// SavePayoutDetails used to fire a goroutine calling the legacy v1
// rz.CreateLinkedAccount, which sends identity only (email, phone, legal
// name, contact name) and NO bank details. Every linked account created that
// way can receive Route transfers but has nowhere to settle them — the root
// cause of "no chef has ever been paid". These tests drive the handler
// end-to-end against an httptest Razorpay stub (services.SetRazorpayClient +
// NewRazorpayTestClient, the same seam route_registration_test.go uses) and
// pin:
//
//   - a bank-transfer chef's RazorpaySettlementStatus is persisted from the
//     real registration result, not left blank
//   - an existing RazorpayAccountID is reused rather than minting a second
//     linked account (a second account would split the chef's money with no
//     way to merge it back)
//   - a needs_clarification outcome persists both the status and the
//     requirements JSON, so the admin queue can show what Razorpay objected to
//   - a chef who selected UPI never ends up with a status implying they can
//     be paid — Route has no VPA destination, so the gateway must not even be
//     called
//
// DDL reuse: chefProfilesGuardDDL / chefGuardAuditDDL (chef_fulfillment_guard_test.go)
// already list every column gorm's tx.Save(&chef)/Updates(&chef) touches for
// models.ChefProfile, including the four settlement columns this task writes to.
// setupDB (internal_users_test.go) provides the users table SavePayoutDetails
// preloads.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

// settlementRouteCall records one request the handler test's stub gateway saw.
type settlementRouteCall struct {
	Method string
	Path   string
}

// settlementRouteServer stands in for the Razorpay v2 onboarding sequence
// (POST /accounts, POST /accounts/:id/stakeholders, POST /accounts/:id/products,
// PATCH /accounts/:id/products/:pid) that services.RegisterSettlementAccount
// drives. productPatchBody supplies the PATCH response — the call whose
// activation_status decides whether the chef is payable.
func settlementRouteServer(t *testing.T, productPatchBody string) (*httptest.Server, *[]settlementRouteCall) {
	t.Helper()
	var calls []settlementRouteCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, settlementRouteCall{Method: r.Method, Path: r.URL.Path})
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/stakeholders"):
			_, _ = w.Write([]byte(`{"id":"sth_1"}`))
		case strings.Contains(r.URL.Path, "/products/"):
			_, _ = w.Write([]byte(productPatchBody))
		case strings.HasSuffix(r.URL.Path, "/products"):
			_, _ = w.Write([]byte(`{"id":"acc_prd_1","activation_status":"created"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"acc_123","status":"created"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

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

	// SavePayoutDetails' (unmodified, out-of-scope) secret-storage goroutine
	// calls config.IsDevelopment(), which nil-derefs unless AppConfig is set.
	// Set it once and never restore to nil: that goroutine is fire-and-forget
	// and can still be running after this test function returns, so a
	// save/restore-to-nil in t.Cleanup would race the goroutine's read against
	// the next test's restore. A permanent non-nil zero-value Config is
	// behaviourally identical to nil for every other handler's
	// `config.AppConfig == nil || !config.AppConfig.XEnabled` feature-flag
	// checks, so this can't affect unrelated tests in this package.
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

// chefSettlementRow reads back the four columns this task's registration
// result is persisted into.
func chefSettlementRow(t *testing.T, db *gorm.DB, chefID uuid.UUID) (accountID, productID, status, requirements string) {
	t.Helper()
	require.NoError(t, db.Raw(
		`SELECT razorpay_account_id, razorpay_product_id, razorpay_settlement_status, razorpay_settlement_requirements
		 FROM chef_profiles WHERE id = ?`, chefID.String()).
		Row().Scan(&accountID, &productID, &status, &requirements))
	return
}

func bankTransferPayload() map[string]any {
	return map[string]any{
		"payoutMethod":      "bank_transfer",
		"bankAccountNumber": "1234567890",
		"bankIFSC":          "HDFC0000123",
		"bankAccountName":   "Anita Rao",
	}
}

func TestSavePayoutDetails_BankTransferPersistsSettlementStatus(t *testing.T) {
	db, userID, chefID := setupChefPayoutSettlementDB(t)
	srv, calls := settlementRouteServer(t, `{"id":"acc_prd_1","activation_status":"activated"}`)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "k", "s", ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })

	w := postPayout(t, userID, bankTransferPayload())
	require.Equal(t, http.StatusOK, w.Code)

	accountID, productID, status, _ := chefSettlementRow(t, db, chefID)
	require.Equal(t, "acc_123", accountID, "the registration result must be persisted, not the legacy call's response")
	require.Equal(t, "acc_prd_1", productID)
	require.Equal(t, "activated", status, "an activated settlement is what makes the chef payable")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "activated", body["razorpaySettlementStatus"], "the chef must see the real outcome, not a blank status")

	require.NotEmpty(t, *calls, "the v2 onboarding sequence must actually run")
}

func TestSavePayoutDetails_ReusesExistingRazorpayAccount(t *testing.T) {
	db, userID, chefID := setupChefPayoutSettlementDB(t)
	require.NoError(t, db.Exec(
		`UPDATE chef_profiles SET razorpay_account_id = ? WHERE id = ?`,
		"acc_existing", chefID.String()).Error)

	srv, calls := settlementRouteServer(t, `{"id":"acc_prd_1","activation_status":"activated"}`)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "k", "s", ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })

	w := postPayout(t, userID, bankTransferPayload())
	require.Equal(t, http.StatusOK, w.Code)

	for _, c := range *calls {
		require.NotEqual(t, "/v2/accounts", c.Path,
			"re-saving bank details must never mint a second linked account — it would split the chef's money with no way to merge it back")
	}

	accountID, _, _, _ := chefSettlementRow(t, db, chefID)
	require.Equal(t, "acc_existing", accountID, "the existing account id must be preserved, not overwritten by a fresh one")
}

func TestSavePayoutDetails_NeedsClarificationPersistsRequirements(t *testing.T) {
	db, userID, chefID := setupChefPayoutSettlementDB(t)
	srv, _ := settlementRouteServer(t, `{
		"id":"acc_prd_1",
		"activation_status":"needs_clarification",
		"requirements":[{"field_reference":"settlements.beneficiary_name","reason_code":"field_missing","resolution_url":"https://api.razorpay.com/v2/accounts/acc_123/products/acc_prd_1"}]
	}`)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "k", "s", ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })

	w := postPayout(t, userID, bankTransferPayload())
	require.Equal(t, http.StatusOK, w.Code, "a review outcome is not a crash")

	_, _, status, requirementsJSON := chefSettlementRow(t, db, chefID)
	require.Equal(t, "needs_clarification", status)

	var reqs []map[string]any
	require.NoError(t, json.Unmarshal([]byte(requirementsJSON), &reqs), "requirements must be persisted as valid JSON")
	require.Len(t, reqs, 1)
	require.Equal(t, "settlements.beneficiary_name", reqs[0]["field_reference"])

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "needs_clarification", body["razorpaySettlementStatus"])
}

// TestSavePayoutDetails_UpiRejected pins #767: UPI is not an accepted payout
// method. Route settles by NEFT/IMPS to a bank account and has no VPA
// destination, so a chef who picks UPI could never be paid — accepting it only
// strands their earnings behind an "onboarded" facade. The handler must reject
// the request at the edge (400) before touching the DB or the gateway, and must
// never persist "upi" as the payout method.
func TestSavePayoutDetails_UpiRejected(t *testing.T) {
	db, userID, chefID := setupChefPayoutSettlementDB(t)
	// If the handler wrongly proceeded, this stub would report "activated" —
	// assert the gateway is never reached via calls, not via a refusing stub.
	srv, calls := settlementRouteServer(t, `{"id":"acc_prd_1","activation_status":"activated"}`)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "k", "s", ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })

	w := postPayout(t, userID, map[string]any{
		"payoutMethod": "upi",
		"upiId":        "chef@upi",
	})
	require.Equal(t, http.StatusBadRequest, w.Code, "UPI is not payable on Route — it must be rejected, not accepted")

	require.Empty(t, *calls, "a rejected request must never reach the gateway")

	_, _, status, _ := chefSettlementRow(t, db, chefID)
	require.Empty(t, status, "a rejected UPI save must not persist any settlement status")

	var method string
	require.NoError(t, db.Raw(`SELECT payout_method FROM chef_profiles WHERE id = ?`, chefID.String()).Row().Scan(&method))
	require.NotEqual(t, "upi", method, "the rejected UPI method must never be persisted")
}

// TestSavePayoutDetails_CannotSwitchToUpi pins #767 for an already-payable
// chef: a chef activated on bank transfer cannot switch to UPI. The switch is
// rejected (400) and their existing bank settlement — linked account, product
// and "activated" status — is left fully intact, so they stay payable rather
// than being silently stranded on a rail Route can never settle.
func TestSavePayoutDetails_CannotSwitchToUpi(t *testing.T) {
	db, userID, chefID := setupChefPayoutSettlementDB(t)
	require.NoError(t, db.Exec(
		`UPDATE chef_profiles SET razorpay_account_id = ?, razorpay_product_id = ?, razorpay_settlement_status = ?, payout_method = 'bank_transfer' WHERE id = ?`,
		"acc_existing", "prd_existing", "activated", chefID.String()).Error)

	// No stub server needed — a rejected switch must never touch the gateway.
	services.SetRazorpayClient(services.NewRazorpayTestClient("http://unused.invalid", "k", "s", ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })

	w := postPayout(t, userID, map[string]any{
		"payoutMethod": "upi",
		"upiId":        "chef@upi",
	})
	require.Equal(t, http.StatusBadRequest, w.Code, "switching an activated chef to UPI must be rejected")

	accountID, productID, status, _ := chefSettlementRow(t, db, chefID)
	require.Equal(t, "acc_existing", accountID, "a rejected switch must leave the linked account intact")
	require.Equal(t, "prd_existing", productID, "a rejected switch must leave the product configuration intact")
	require.Equal(t, "activated", status, "the chef stays payable on bank transfer — a rejected switch must not clear their status")

	var method string
	require.NoError(t, db.Raw(`SELECT payout_method FROM chef_profiles WHERE id = ?`, chefID.String()).Row().Scan(&method))
	require.Equal(t, "bank_transfer", method, "the chef's payout method must remain bank_transfer")
}

func TestSavePayoutDetails_PartialFailurePersistsWhatWasCreated(t *testing.T) {
	// A stakeholder-step rejection leaves an account created but no product/
	// status. That partial result has to be persisted anyway — otherwise a
	// retry mints a second linked account instead of resuming this one.
	db, userID, chefID := setupChefPayoutSettlementDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/stakeholders") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"description":"pan required"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"acc_123"}`))
	}))
	t.Cleanup(srv.Close)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "k", "s", ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })

	w := postPayout(t, userID, bankTransferPayload())
	require.Equal(t, http.StatusOK, w.Code, "a gateway failure saves the payout method; it must not 500 the whole request")

	accountID, productID, status, _ := chefSettlementRow(t, db, chefID)
	require.Equal(t, "acc_123", accountID, "the account created before the failure must be kept so a retry resumes rather than duplicating it")
	require.Empty(t, productID)
	require.NotEqual(t, "activated", status, "a chef must never read as payable when registration failed partway through")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "settlement_registration_failed", body["razorpaySettlementError"])
}
