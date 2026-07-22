package services

import (
	"net/http"
	"strings"
	"testing"
)

// #740 — the orchestration that turns a chef's saved bank details into a
// payable Razorpay linked account.
//
// Kept free of the database so the sequencing is testable on its own: the
// caller persists whatever comes back. The ordering matters (an account before
// a stakeholder before a product before its settlement configuration) and so
// does not repeating steps, because creating a second linked account for a chef
// who already has one splits their money across two accounts with no way to
// merge them.

func registration() SettlementRegistration {
	return SettlementRegistration{
		Email:        "chef@example.com",
		Phone:        "9876543210",
		LegalName:    "Anita's Kitchen",
		ContactName:  "Anita R",
		BusinessType: "individual",
		Account: SettlementAccount{
			BeneficiaryName: "Anita R",
			AccountNumber:   "1234567890",
			IFSC:            "HDFC0000123",
		},
	}
}

// routeStub answers the whole onboarding sequence with canned success.
func routeStub(t *testing.T) (*RazorpayClient, *[]routeCall) {
	t.Helper()
	srv, calls := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/stakeholders"):
			_, _ = w.Write([]byte(`{"id":"sth_1"}`))
		case strings.Contains(r.URL.Path, "/products/"):
			_, _ = w.Write([]byte(`{"id":"acc_prd_1","activation_status":"activated"}`))
		case strings.HasSuffix(r.URL.Path, "/products"):
			_, _ = w.Write([]byte(`{"id":"acc_prd_1","activation_status":"created"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"acc_123","status":"created"}`))
		}
	})
	return NewRazorpayTestClient(srv.URL, "k", "s", ""), calls
}

func paths(calls []routeCall) []string {
	out := make([]string, 0, len(calls))
	for _, c := range calls {
		out = append(out, c.Method+" "+c.Path)
	}
	return out
}

func TestRegisterSettlementAccount_RunsTheFullSequenceInOrder(t *testing.T) {
	c, calls := routeStub(t)

	res, err := RegisterSettlementAccount(c, registration())
	if err != nil {
		t.Fatalf("RegisterSettlementAccount: %v", err)
	}
	if res.AccountID != "acc_123" || res.ProductID != "acc_prd_1" {
		t.Fatalf("ids = %q / %q", res.AccountID, res.ProductID)
	}
	if res.ActivationStatus != "activated" {
		t.Fatalf("activation status = %q, want activated", res.ActivationStatus)
	}

	want := []string{
		"POST /v2/accounts",
		"POST /v2/accounts/acc_123/stakeholders",
		"POST /v2/accounts/acc_123/products",
		"PATCH /v2/accounts/acc_123/products/acc_prd_1",
	}
	got := paths(*calls)
	if len(got) != len(want) {
		t.Fatalf("calls = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRegisterSettlementAccount_ReusesAnExistingAccount(t *testing.T) {
	// Re-saving bank details must not mint a second linked account: the chef's
	// money would then be split across two accounts with no way to merge them.
	c, calls := routeStub(t)

	in := registration()
	in.ExistingAccountID = "acc_existing"

	res, err := RegisterSettlementAccount(c, in)
	if err != nil {
		t.Fatalf("RegisterSettlementAccount: %v", err)
	}
	if res.AccountID != "acc_existing" {
		t.Fatalf("account id = %q, want the existing one", res.AccountID)
	}
	for _, p := range paths(*calls) {
		if p == "POST /v2/accounts" {
			t.Fatal("must not create a second linked account for a chef who has one")
		}
	}
}

func TestRegisterSettlementAccount_ReusesAnExistingProduct(t *testing.T) {
	// Same reasoning one level down — re-requesting the product on every save
	// churns the activation state Razorpay is reviewing.
	c, calls := routeStub(t)

	in := registration()
	in.ExistingAccountID = "acc_existing"
	in.ExistingProductID = "acc_prd_existing"

	if _, err := RegisterSettlementAccount(c, in); err != nil {
		t.Fatalf("RegisterSettlementAccount: %v", err)
	}
	for _, p := range paths(*calls) {
		if strings.HasSuffix(p, "/products") {
			t.Fatal("must not re-request the route product when one exists")
		}
	}
}

func TestRegisterSettlementAccount_CarriesRequirementsBack(t *testing.T) {
	srv, _ := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/products/"):
			_, _ = w.Write([]byte(`{
				"id":"acc_prd_1",
				"activation_status":"needs_clarification",
				"requirements":[{"field_reference":"settlements.ifsc_code","reason_code":"invalid_value"}]
			}`))
		case strings.HasSuffix(r.URL.Path, "/stakeholders"):
			_, _ = w.Write([]byte(`{"id":"sth_1"}`))
		case strings.HasSuffix(r.URL.Path, "/products"):
			_, _ = w.Write([]byte(`{"id":"acc_prd_1","activation_status":"created"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"acc_123"}`))
		}
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	res, err := RegisterSettlementAccount(c, registration())
	if err != nil {
		t.Fatalf("a review outcome is not an error: %v", err)
	}
	if res.ActivationStatus != "needs_clarification" {
		t.Fatalf("activation status = %q", res.ActivationStatus)
	}
	if len(res.Requirements) != 1 {
		t.Fatalf("requirements = %d, want 1", len(res.Requirements))
	}
	if res.Payable() {
		t.Fatal("a chef under review must not be reported as payable")
	}
}

func TestRegisterSettlementAccount_RejectsAMissingBankAccount(t *testing.T) {
	// Route settles by NEFT/IMPS to a bank account — there is no UPI VPA
	// destination for a linked account. Registering without one would create an
	// account that looks configured and can never pay out.
	c, calls := routeStub(t)

	in := registration()
	in.Account = SettlementAccount{}

	if _, err := RegisterSettlementAccount(c, in); err == nil {
		t.Fatal("registration without a bank account must be refused")
	}
	if len(*calls) != 0 {
		t.Fatalf("must fail before touching the gateway, made %d call(s)", len(*calls))
	}
}

func TestRegisterSettlementAccount_StopsOnGatewayFailure(t *testing.T) {
	// A failure partway through must not be reported as a registered account,
	// or the chef is marked payout-ready with money stranded behind it.
	srv, _ := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stakeholders") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"description":"pan required"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"acc_123"}`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	res, err := RegisterSettlementAccount(c, registration())
	if err == nil {
		t.Fatal("a failed stakeholder step must surface as an error")
	}
	if !strings.Contains(err.Error(), "pan required") {
		t.Fatalf("error should carry the gateway reason: %v", err)
	}
	// The account id created before the failure is still worth keeping so a
	// retry reuses it rather than creating another.
	if res == nil || res.AccountID != "acc_123" {
		t.Fatalf("partial result should retain the created account id, got %+v", res)
	}
}
