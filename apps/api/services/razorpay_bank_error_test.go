package services

// razorpay_bank_error_test.go — a Razorpay validation error must never leak
// the bank details it was validating (review finding 4, payout-release-
// governor fix wave).
//
// doURL's default error format is `razorpay API error (HTTP %d): %s` with the
// ENTIRE raw response body appended — fine for most Razorpay calls, but
// ConfigureRouteSettlement PATCHes account_number/ifsc_code/beneficiary_name,
// and a validation rejection commonly echoes the offending value back
// somewhere in the body. handlers/chefs.go log.Printf's that error and sends
// it to Sentry, so a bank account number could ride a rejected save straight
// into logs and an error-tracking dashboard.

import (
	"net/http"
	"strings"
	"testing"
)

func TestConfigureRouteSettlement_GatewayErrorDoesNotLeakTheAccountNumber(t *testing.T) {
	const leakedAccountNumber = "1234567890123"
	srv, _ := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		// A realistic Razorpay-shaped validation error whose body echoes the
		// offending value outside error.description — exactly the shape
		// doURL's raw-body error would leak today.
		_, _ = w.Write([]byte(`{
			"error": {
				"code": "BAD_REQUEST_VALIDATION_ERROR",
				"description": "The account number is invalid"
			},
			"input": {"settlements": {"account_number": "` + leakedAccountNumber + `"}}
		}`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	_, err := c.ConfigureRouteSettlement("acc_123", "acc_prd_1", &SettlementAccount{
		BeneficiaryName: "Anita R",
		AccountNumber:   leakedAccountNumber,
		IFSC:            "HDFC0000123",
	})
	if err == nil {
		t.Fatal("a rejected settlement configuration must still surface as an error")
	}
	if strings.Contains(err.Error(), leakedAccountNumber) {
		t.Fatalf("error leaked the bank account number: %v", err)
	}
	// The structured reason must still reach the caller — sanitizing must not
	// turn every failure into an opaque blob nobody can act on.
	if !strings.Contains(err.Error(), "account number is invalid") {
		t.Fatalf("error should still carry Razorpay's description, got: %v", err)
	}
}

func TestConfigureRouteSettlement_ErrorSurvivesUnparseableBody(t *testing.T) {
	// A non-JSON or unexpected-shape error body must still produce a
	// reasonable error, not a panic or an empty message.
	srv, _ := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`not json`))
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	_, err := c.ConfigureRouteSettlement("acc_123", "acc_prd_1", &SettlementAccount{
		BeneficiaryName: "Anita R", AccountNumber: "1", IFSC: "IFSC0001",
	})
	if err == nil {
		t.Fatal("a 500 must surface as an error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("error should mention the HTTP status, got: %v", err)
	}
}

func TestRegisterSettlementAccount_SettlementErrorDoesNotLeakTheAccountNumber(t *testing.T) {
	// The same guarantee through the orchestration entrypoint every caller
	// (SavePayoutDetails) actually uses.
	const leakedAccountNumber = "9988776655"
	srv, _ := newRouteServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/stakeholders"):
			_, _ = w.Write([]byte(`{"id":"sth_1"}`))
		case strings.Contains(r.URL.Path, "/products/"):
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{
				"error": {"code": "BAD_REQUEST_VALIDATION_ERROR", "description": "invalid account number"},
				"input": {"settlements": {"account_number": "` + leakedAccountNumber + `"}}
			}`))
		case strings.HasSuffix(r.URL.Path, "/products"):
			_, _ = w.Write([]byte(`{"id":"acc_prd_1","activation_status":"created"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"acc_123","status":"created"}`))
		}
	})
	c := NewRazorpayTestClient(srv.URL, "k", "s", "")

	in := registration()
	in.Account.AccountNumber = leakedAccountNumber

	_, err := RegisterSettlementAccount(c, in)
	if err == nil {
		t.Fatal("a rejected settlement configuration must surface as an error")
	}
	if strings.Contains(err.Error(), leakedAccountNumber) {
		t.Fatalf("error leaked the bank account number all the way through RegisterSettlementAccount: %v", err)
	}
}
