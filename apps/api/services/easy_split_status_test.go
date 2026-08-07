package services

// easy_split_status_test.go — #1082. A chef must never read Cashfree's internal
// vocabulary. "IN_BENE_CREATION" tells a home cook nothing, and the vendor app's
// own guesswork rendered BLOCKED as "verification in progress" — a state that
// will never resolve, described as one that will.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// The profile itself is marshalled straight into several chef-facing responses,
// so a status field left on it leaks past the mapper (#1082).
func TestChefProfile_DoesNotMarshalTheRawVendorStatus(t *testing.T) {
	raw, err := json.Marshal(chefWithVendor("hc_1", CashfreeVendorInBeneCreation))
	require.NoError(t, err)
	require.NotContains(t, string(raw), CashfreeVendorInBeneCreation)
	require.NotContains(t, string(raw), "cashfreeVendorStatus")
}

func chefWithVendor(id, status string) *models.ChefProfile {
	return &models.ChefProfile{CashfreeVendorID: id, CashfreeVendorStatus: status}
}

func TestPayoutRegistrationFor_ReportsTheThreeChefFacingStages(t *testing.T) {
	cases := []struct {
		name    string
		chef    *models.ChefProfile
		state   string
		message string
	}{
		{"no details on file", chefWithVendor("", ""), PayoutRegistrationNone, "Add your bank details to start receiving payouts"},
		{"registered, awaiting Cashfree", chefWithVendor("hc_1", CashfreeVendorInBeneCreation), PayoutRegistrationPending, "Pending verification"},
		{"bank account being validated", chefWithVendor("hc_1", CashfreeVendorInBankValidation), PayoutRegistrationPending, "Pending verification"},
		{"registered, status not yet read back", chefWithVendor("hc_1", ""), PayoutRegistrationPending, "Pending verification"},
		{"verified", chefWithVendor("hc_1", CashfreeVendorActive), PayoutRegistrationVerified, "Verified — payouts active"},
		{"blocked", chefWithVendor("hc_1", CashfreeVendorBlocked), PayoutRegistrationFailed, "Couldn't verify — please check your details"},
		{"deleted", chefWithVendor("hc_1", CashfreeVendorDeleted), PayoutRegistrationFailed, "Couldn't verify — please check your details"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := PayoutRegistrationFor(tc.chef)
			require.Equal(t, tc.state, got.State)
			require.Equal(t, tc.message, got.Message)
		})
	}
}

// Cashfree adds states without announcing them — IN_BANK_VALIDATION was one.
// An unrecognised state is still in flight, so it reads as pending: telling a
// chef their details are wrong when they are not sends them to support.
func TestPayoutRegistrationFor_TreatsAnUnknownStateAsStillInFlight(t *testing.T) {
	got := PayoutRegistrationFor(chefWithVendor("hc_1", "SOME_FUTURE_STATE"))
	require.Equal(t, PayoutRegistrationPending, got.State)
}

// The chef-facing state is a summary, never the raw string. If a Cashfree token
// leaks into either field the UI will render it.
func TestPayoutRegistrationFor_NeverEchoesTheRawCashfreeStatus(t *testing.T) {
	for _, raw := range []string{
		CashfreeVendorActive, CashfreeVendorInBeneCreation, CashfreeVendorInBankValidation,
		CashfreeVendorBlocked, CashfreeVendorDeleted, "SOME_FUTURE_STATE",
	} {
		got := PayoutRegistrationFor(chefWithVendor("hc_1", raw))
		require.NotContains(t, got.State, "_", "%q leaked into the chef-facing state", raw)
		require.NotContains(t, got.Message, raw)
	}
}

// Verified is the only state that means "your money arrives from the gateway",
// and it must agree with what BuildOrderSplit will actually do.
func TestPayoutRegistrationFor_VerifiedMatchesSplitPayability(t *testing.T) {
	for _, raw := range []string{
		CashfreeVendorActive, CashfreeVendorInBeneCreation, CashfreeVendorInBankValidation,
		CashfreeVendorBlocked, CashfreeVendorDeleted, "SOME_FUTURE_STATE", "",
	} {
		verified := PayoutRegistrationFor(chefWithVendor("hc_1", raw)).State == PayoutRegistrationVerified
		payable := (&CashfreeVendorResponse{Status: raw}).SplitPayable()
		require.Equal(t, payable, verified, "status %q: the chef is told one thing and paid another", raw)
	}
}
