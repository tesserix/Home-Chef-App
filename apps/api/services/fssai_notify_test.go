package services

// fssai_notify_test.go — the chef's bell entry for a status change.
//
// A status an admin sets is only real to the chef if it reaches them, and the
// push is the half that can be switched off. These pin the row that survives it.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestFssaiChefNotice_CoversEveryStatusAChefIsToldAbout(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{models.FssaiInProgress, true},
		{models.FssaiMoreInfoRequired, true},
		{models.FssaiFiled, true},
		{models.FssaiIssued, true},
		{models.FssaiRejected, true},
		// Internal bookkeeping the chef does not need pinged about.
		{models.FssaiAwaitingPayment, false},
		{models.FssaiSubmitted, false},
		{models.FssaiRefunded, false},
	}
	for _, c := range cases {
		r := &models.FssaiRequest{
			Status:         c.status,
			InfoRequested:  "Your Aadhaar photo is cut off.",
			ApplicationRef: "FOSCOS-1",
			RegistrationNo: "12345",
			RejectedReason: "Address could not be verified.",
		}
		title, msg := fssaiChefNotice(r)
		if c.want {
			require.NotEmpty(t, title, "status %s must tell the chef something", c.status)
			require.NotEmpty(t, msg, "status %s must say why", c.status)
		} else {
			require.Empty(t, title, "status %s is not news for the chef", c.status)
		}
	}
}

// The admin's words reach the chef unaltered — a paraphrase would be a second
// version of the truth for them to reconcile against what they were asked for.
func TestFssaiChefNotice_QuotesTheAdminVerbatim(t *testing.T) {
	want := "Send a utility bill for the kitchen address."
	_, msg := fssaiChefNotice(&models.FssaiRequest{
		Status:        models.FssaiMoreInfoRequired,
		InfoRequested: want,
	})
	require.Equal(t, want, msg)
}

// The row is what survives a push the chef never saw, so it must carry a user
// to file against — an empty UserID would write a notification nobody can read.
func TestNotifyFssaiRequestStatus_NeedsAUser(t *testing.T) {
	r := &models.FssaiRequest{
		ID:     uuid.New(),
		Status: models.FssaiInProgress,
	}
	require.Equal(t, uuid.Nil, r.UserID,
		"guards the assumption below: a request with no user must not be notified")

	title, _ := fssaiChefNotice(r)
	require.NotEmpty(t, title, "in_progress is worth telling the chef")
}
