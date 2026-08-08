package services

// #1151. Cashfree allows ONE beneficiary per account/IFSC across the merchant
// account. When an account is already registered under some other beneficiary id,
// the create 409s and our own id is a 404 — which we read as "rejected", leaving
// the payee permanently unpayable even though the destination is registered and
// perfectly good. It is the same bank account either way, so the registration is
// adopted instead.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

const (
	conflictAccount = "000100289877623"
	conflictIFSC    = "SBIN0008752"
	// The id the account is already registered under — not one we would mint.
	incumbentBeneficiary = "hc_chef_e2e000000000400080000000cafe.9911aabb"
)

// conflictingRail answers like Cashfree when the instrument is already taken:
// 409 on create, 404 on our beneficiary id, and the incumbent on an
// instrument lookup.
func conflictingRail(t *testing.T) *CashfreePayoutClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"type":"conflict_error","code":"conflict_with_existing_beneficiary",` +
				`"message":"Beneficiary with the given bank_account_number and bank_ifsc already exists"}`))
			return
		}
		q := r.URL.Query()
		if q.Get("bank_account_number") == conflictAccount && q.Get("bank_ifsc") == conflictIFSC {
			_, _ = w.Write([]byte(`{"beneficiary_id":"` + incumbentBeneficiary +
				`","beneficiary_status":"VERIFIED"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"type":"not_found_error","message":"beneficiary does not exist"}`))
	}))
	t.Cleanup(srv.Close)
	return NewCashfreePayoutTestClient(srv.URL, "id", "secret", "", models.ChefModeTest)
}

func TestBeneficiaryConflictAdoptsTheIncumbentRegistration(t *testing.T) {
	c := conflictingRail(t)

	res, err := c.CreateBeneficiary(context.Background(), payouts.BeneficiaryRequest{
		BeneficiaryID: "hc_chef_e150c72a42e24beb8cb1389666dd813c.4040e6f8",
		Name:          "Saffron Home Kitchen",
		Instrument: payouts.Instrument{
			Kind: payouts.MethodBankAccount, AccountNumber: conflictAccount, IFSC: conflictIFSC,
		},
	})

	require.NoError(t, err, "an account already registered is a usable destination, not a rejection")
	require.Equal(t, incumbentBeneficiary, res.BeneficiaryID,
		"the incumbent id is what a transfer must be addressed to")
	require.Equal(t, payouts.MethodVerified, res.Status)
}

// A conflict with nothing behind it is still a rejection — adopting must never
// invent a destination.
func TestBeneficiaryConflictWithNoIncumbentIsStillRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"conflict_with_existing_beneficiary","message":"already exists"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"beneficiary does not exist"}`))
	}))
	t.Cleanup(srv.Close)
	c := NewCashfreePayoutTestClient(srv.URL, "id", "secret", "", models.ChefModeTest)

	res, err := c.CreateBeneficiary(context.Background(), payouts.BeneficiaryRequest{
		BeneficiaryID: "hc_chef_whatever.1234",
		Name:          "Nobody",
		Instrument: payouts.Instrument{
			Kind: payouts.MethodBankAccount, AccountNumber: "999", IFSC: "HDFC0000001",
		},
	})

	require.ErrorIs(t, err, payouts.ErrBeneficiaryRejected)
	require.Equal(t, payouts.MethodInvalid, res.Status)
}

// The lookup is by instrument, so both halves must reach Cashfree — an account
// number alone matches a different bank's account.
func TestInstrumentLookupSendsAccountAndIFSC(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"beneficiary_id":"bene_1","beneficiary_status":"VERIFIED"}`))
	}))
	t.Cleanup(srv.Close)
	c := NewCashfreePayoutTestClient(srv.URL, "id", "secret", "", models.ChefModeTest)

	_, err := c.FetchBeneficiaryByInstrument(context.Background(), payouts.Instrument{
		Kind: payouts.MethodBankAccount, AccountNumber: conflictAccount, IFSC: conflictIFSC,
	})
	require.NoError(t, err)
	require.Equal(t, conflictAccount, got.Get("bank_account_number"))
	require.Equal(t, conflictIFSC, got.Get("bank_ifsc"))
}
