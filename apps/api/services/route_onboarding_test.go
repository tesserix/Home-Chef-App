package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// #740 — registering a chef's settlement bank account with Razorpay.
//
// Route pays a chef by settling their linked account's balance to a bank
// account REGISTERED ON THAT LINKED ACCOUNT. Until this exists, a released
// transfer lands in a balance with nowhere to go, which is why no chef could
// ever be paid: CreateLinkedAccount sends name/email/phone and no bank details.
//
// Razorpay's onboarding is a v2 flow — account, then stakeholder, then a route
// product configuration carrying the bank account — and our client is pinned to
// /v1, so these exercise the v2 base as well as the payloads.

// routeTestServer records every request so a test can assert on path, method
// and body without a live gateway.
type routeCall struct {
	Method string
	Path   string
	Body   map[string]any
}

func newRouteServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *[]routeCall) {
	t.Helper()
	var calls []routeCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		calls = append(calls, routeCall{Method: r.Method, Path: r.URL.Path, Body: body})
		w.Header().Set("Content-Type", "application/json")
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestCreateLinkedAccountV2_UsesTheV2Base(t *testing.T) {
	// The v1 /accounts endpoint cannot carry a settlement configuration. Hitting
	// it silently produces an account that can never be paid.
	srv, calls := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"acc_123","status":"created"}`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	acct, err := c.CreateLinkedAccountV2(&V2AccountRequest{
		Email:        "chef@example.com",
		Phone:        "9876543210",
		LegalName:    "Anita's Kitchen",
		BusinessType: "individual",
		ContactName:  "Anita R",
	})
	if err != nil {
		t.Fatalf("CreateLinkedAccountV2: %v", err)
	}
	if acct.ID != "acc_123" {
		t.Fatalf("account id = %q, want acc_123", acct.ID)
	}
	if got := (*calls)[0].Path; got != "/v2/accounts" {
		t.Fatalf("path = %q, want /v2/accounts", got)
	}
}

func TestCreateStakeholder_TargetsTheAccount(t *testing.T) {
	srv, calls := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"sth_1"}`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	if _, err := c.CreateStakeholder("acc_123", &StakeholderRequest{
		Name:  "Anita R",
		Email: "chef@example.com",
	}); err != nil {
		t.Fatalf("CreateStakeholder: %v", err)
	}
	if got := (*calls)[0].Path; got != "/v2/accounts/acc_123/stakeholders" {
		t.Fatalf("path = %q", got)
	}
}

func TestConfigureRouteSettlement_SendsTheBankAccount(t *testing.T) {
	// The whole point of #740: the bank account has to reach Razorpay. If this
	// payload loses the account number or IFSC, the chef never gets money and
	// nothing in our system reports a problem.
	srv, calls := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"acc_prd_1","activation_status":"activated"}`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	res, err := c.ConfigureRouteSettlement("acc_123", "acc_prd_1", &SettlementAccount{
		BeneficiaryName: "Anita R",
		AccountNumber:   "1234567890",
		IFSC:            "HDFC0000123",
	})
	if err != nil {
		t.Fatalf("ConfigureRouteSettlement: %v", err)
	}
	if res.ActivationStatus != "activated" {
		t.Fatalf("activation status = %q", res.ActivationStatus)
	}

	call := (*calls)[0]
	if call.Method != "PATCH" {
		t.Fatalf("method = %q, want PATCH", call.Method)
	}
	if call.Path != "/v2/accounts/acc_123/products/acc_prd_1" {
		t.Fatalf("path = %q", call.Path)
	}
	// settlements live under product configuration
	settings, _ := call.Body["settlements"].(map[string]any)
	if settings == nil {
		t.Fatalf("no settlements object in payload: %v", call.Body)
	}
	if settings["account_number"] != "1234567890" {
		t.Fatalf("account_number = %v", settings["account_number"])
	}
	if settings["ifsc_code"] != "HDFC0000123" {
		t.Fatalf("ifsc_code = %v", settings["ifsc_code"])
	}
	if settings["beneficiary_name"] != "Anita R" {
		t.Fatalf("beneficiary_name = %v", settings["beneficiary_name"])
	}
}

func TestConfigureRouteSettlement_SurfacesNeedsClarification(t *testing.T) {
	// Razorpay reviews the bank details asynchronously. A rejected account looks
	// exactly like a healthy one unless the requirements come back to the admin,
	// so the chef would sit unpaid with no visible reason.
	srv, _ := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"id":"acc_prd_1",
			"activation_status":"needs_clarification",
			"requirements":[{
				"field_reference":"settlements.beneficiary_name",
				"reason_code":"field_missing",
				"resolution_url":"https://api.razorpay.com/v2/accounts/acc_123/products/acc_prd_1"
			}]
		}`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	res, err := c.ConfigureRouteSettlement("acc_123", "acc_prd_1", &SettlementAccount{
		BeneficiaryName: "Anita R", AccountNumber: "1234567890", IFSC: "HDFC0000123",
	})
	if err != nil {
		t.Fatalf("a reviewable response is not a transport error: %v", err)
	}
	if res.ActivationStatus != "needs_clarification" {
		t.Fatalf("activation status = %q", res.ActivationStatus)
	}
	if len(res.Requirements) != 1 {
		t.Fatalf("requirements = %d, want 1 so the admin can act", len(res.Requirements))
	}
	if res.Requirements[0].FieldReference != "settlements.beneficiary_name" {
		t.Fatalf("field reference = %q", res.Requirements[0].FieldReference)
	}
}

func TestRequestRouteProduct_AsksForRoute(t *testing.T) {
	srv, calls := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"acc_prd_1","activation_status":"created"}`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	prod, err := c.RequestRouteProduct("acc_123")
	if err != nil {
		t.Fatalf("RequestRouteProduct: %v", err)
	}
	if prod.ID != "acc_prd_1" {
		t.Fatalf("product id = %q", prod.ID)
	}
	call := (*calls)[0]
	if call.Path != "/v2/accounts/acc_123/products" {
		t.Fatalf("path = %q", call.Path)
	}
	if call.Body["product_name"] != "route" {
		t.Fatalf("product_name = %v, want route", call.Body["product_name"])
	}
}

// TestCreateLinkedAccountV2_SanitizesGatewayError pins the final-review
// finding: CreateLinkedAccountV2 must not let a gateway 4xx echo the chef's
// name/email into whatever logs/Sentry the caller feeds the returned error
// into (services.CaptureBackgroundError) — the same protection
// ConfigureRouteSettlement's bank-detail call already has via doV2Sanitized.
// The structured error.code/description must still come through, since a
// caller (isStakeholderAlreadyExists-style checks) may need it.
func TestCreateLinkedAccountV2_SanitizesGatewayError(t *testing.T) {
	srv, _ := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		// Razorpay's own validation error, PLUS a hypothetical echo of the
		// submitted identity fields elsewhere in the body — exactly the shape
		// doURL's unsanitized error (which includes the WHOLE raw body) would
		// leak verbatim.
		_, _ = w.Write([]byte(`{"error":{"code":"BAD_REQUEST_ERROR","description":"invalid legal_business_name"},"input":{"email":"chef@example.com","contact_name":"Anita R"}}`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	_, err := c.CreateLinkedAccountV2(&V2AccountRequest{
		Email:        "chef@example.com",
		Phone:        "9876543210",
		LegalName:    "Anita's Kitchen",
		BusinessType: "individual",
		ContactName:  "Anita R",
	})
	if err == nil {
		t.Fatal("a rejected account creation must surface as an error")
	}
	if strings.Contains(err.Error(), "chef@example.com") {
		t.Fatalf("error must not echo the chef's email, got: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid legal_business_name") {
		t.Fatalf("error should still carry the structured gateway reason, got: %v", err)
	}
}

// TestCreateStakeholder_SanitizesGatewayError is the same protection for the
// stakeholder call, whose request body also carries the chef's name/email.
func TestCreateStakeholder_SanitizesGatewayError(t *testing.T) {
	srv, _ := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"BAD_REQUEST_ERROR","description":"invalid phone"},"input":{"email":"chef@example.com","name":"Anita R"}}`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	_, err := c.CreateStakeholder("acc_123", &StakeholderRequest{
		Name:  "Anita R",
		Email: "chef@example.com",
	})
	if err == nil {
		t.Fatal("a rejected stakeholder creation must surface as an error")
	}
	if strings.Contains(err.Error(), "chef@example.com") {
		t.Fatalf("error must not echo the chef's email, got: %v", err)
	}
	if !strings.Contains(err.Error(), "invalid phone") {
		t.Fatalf("error should still carry the structured gateway reason, got: %v", err)
	}
}

func TestRouteOnboarding_ReportsGatewayFailures(t *testing.T) {
	// A 4xx from Razorpay must not be mistaken for a registered account — that
	// would mark the chef payout-ready and silently strand their money.
	srv, _ := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"description":"invalid ifsc"}}`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	_, err := c.ConfigureRouteSettlement("acc_123", "acc_prd_1", &SettlementAccount{
		BeneficiaryName: "Anita R", AccountNumber: "1", IFSC: "BAD",
	})
	if err == nil {
		t.Fatal("a rejected settlement configuration must surface as an error")
	}
	if !strings.Contains(err.Error(), "invalid ifsc") {
		t.Fatalf("error should carry the gateway reason, got: %v", err)
	}
}
