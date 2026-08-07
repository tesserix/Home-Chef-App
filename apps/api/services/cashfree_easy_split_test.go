package services

// cashfree_easy_split_test.go — #1082. The chef self-service registration path,
// pinned against Cashfree's ACTUAL sandbox behaviour rather than its docs:
// a duplicate is a 400 not a 409, a status change mid-validation is refused,
// and IN_BANK_VALIDATION is a real state the docs do not list.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

type vendorCall struct {
	Method string
	Path   string
	Body   map[string]any
}

// vendorStub runs fn against a fake Easy Split API and returns every call made.
// handler decides the response per call index, so a test can say "the POST 400s
// as a duplicate, then the PATCH succeeds".
func vendorStub(t *testing.T, handler func(i int, w http.ResponseWriter, r *http.Request), fn func(*CashfreeClient)) []vendorCall {
	t.Helper()
	var calls []vendorCall

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		call := vendorCall{Method: r.Method, Path: r.URL.Path}
		_ = json.Unmarshal(raw, &call.Body)
		calls = append(calls, call)
		w.Header().Set("Content-Type", "application/json")
		handler(len(calls)-1, w, r)
	}))
	t.Cleanup(srv.Close)

	fn(NewCashfreeTestClient(srv.URL, "TESTapp", "cf_test_secret", "wh", models.ChefModeTest))
	return calls
}

func vendorOK(status string) string {
	return `{"vendor_id":"hc_1","status":"` + status + `","name":"Kitchen"}`
}

func sampleVendorRequest() *CashfreeVendorRequest {
	return &CashfreeVendorRequest{
		VendorID:      "hc_1",
		Name:          "Kitchen",
		Email:         "chef@fe3dr.com",
		Phone:         "9845012345",
		VerifyAccount: true,
		Bank:          &CashfreeVendorBank{AccountNumber: "00011020001772", AccountHolder: "Kitchen", IFSC: "HDFC0000001"},
		KYC:           CashfreeVendorKYC{AccountType: "savings", BusinessType: "Food and Beverages", PAN: "ABCDE1234F"},
	}
}

// A first registration is one POST that carries the account for verification.
func TestCreateVendor_RegistersWithAccountVerification(t *testing.T) {
	var vendor *CashfreeVendorResponse
	var err error
	calls := vendorStub(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(vendorOK(CashfreeVendorInBeneCreation)))
	}, func(c *CashfreeClient) { vendor, err = c.CreateVendor(sampleVendorRequest()) })

	require.NoError(t, err)
	require.Len(t, calls, 1)
	require.Equal(t, "POST", calls[0].Method)
	require.Equal(t, "/easy-split/vendors", calls[0].Path)
	require.Equal(t, true, calls[0].Body["verify_account"])
	require.Equal(t, "Food and Beverages", calls[0].Body["kyc_details"].(map[string]any)["business_type"],
		"a free-form business_type is rejected by Cashfree with a 400")
	require.Equal(t, CashfreeVendorInBeneCreation, vendor.Status)
	require.False(t, vendor.SplitPayable(), "verification is not finished, so no split may name this vendor")
}

// Re-saving with changed details must reach Cashfree, not be swallowed as
// "already registered" — otherwise the chef's money keeps going to the old
// account. The duplicate arrives as a 400, not the documented 409.
func TestCreateVendor_DuplicateBadRequestPatchesTheChangedDetails(t *testing.T) {
	var vendor *CashfreeVendorResponse
	var err error
	calls := vendorStub(t, func(i int, w http.ResponseWriter, _ *http.Request) {
		if i == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"vendor already exists"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(vendorOK(CashfreeVendorActive)))
	}, func(c *CashfreeClient) {
		req := sampleVendorRequest()
		req.Bank.AccountNumber = "00011020009999"
		vendor, err = c.CreateVendor(req)
	})

	require.NoError(t, err, "a duplicate is an existing vendor to update, not a failure")
	require.Len(t, calls, 2)
	require.Equal(t, "PATCH", calls[1].Method)
	require.Equal(t, "/easy-split/vendors/hc_1", calls[1].Path)
	require.Equal(t, "00011020009999", calls[1].Body["bank"].(map[string]any)["account_number"],
		"the new account must reach Cashfree or payouts keep going to the old one")
	_, sentStatus := calls[1].Body["status"]
	require.False(t, sentStatus, "a status change is refused mid-validation, so the update carries details only")
	require.True(t, vendor.SplitPayable())
}

// Saving again with nothing changed is an ordinary no-op: same vendor, no error.
func TestCreateVendor_UnchangedResaveIsANoOp(t *testing.T) {
	var vendor *CashfreeVendorResponse
	var err error
	vendorStub(t, func(i int, w http.ResponseWriter, _ *http.Request) {
		if i == 0 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"Vendor already exists"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(vendorOK(CashfreeVendorActive)))
	}, func(c *CashfreeClient) { vendor, err = c.CreateVendor(sampleVendorRequest()) })

	require.NoError(t, err)
	require.Equal(t, "hc_1", vendor.VendorID, "a re-save must not mint a second vendor")
	require.Equal(t, CashfreeVendorActive, vendor.Status)
}

// While the bank account is being validated the sandbox refuses PATCH outright.
// The vendor exists and is fine — report its live state, not a phantom error.
func TestCreateVendor_RefusedUpdateReturnsTheLiveState(t *testing.T) {
	var vendor *CashfreeVendorResponse
	var err error
	calls := vendorStub(t, func(i int, w http.ResponseWriter, _ *http.Request) {
		switch i {
		case 0:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"vendor already exists"}`))
		case 1:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"vendor update not allowed in current status"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(vendorOK(CashfreeVendorInBankValidation)))
		}
	}, func(c *CashfreeClient) { vendor, err = c.CreateVendor(sampleVendorRequest()) })

	require.NoError(t, err)
	require.Len(t, calls, 3)
	require.Equal(t, "GET", calls[2].Method)
	require.Equal(t, CashfreeVendorInBankValidation, vendor.Status)
	require.False(t, vendor.SplitPayable(),
		"IN_BANK_VALIDATION is undocumented but real — anything but ACTIVE is unpayable")
}

// A genuine rejection — wrong IFSC, name mismatch — is still an error. The
// duplicate handling must not swallow every 400.
func TestCreateVendor_RealRejectionIsStillAnError(t *testing.T) {
	var err error
	calls := vendorStub(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid ifsc"}`))
	}, func(c *CashfreeClient) { _, err = c.CreateVendor(sampleVendorRequest()) })

	require.Error(t, err)
	require.Len(t, calls, 1, "a rejection must not be retried as an update")
}
