package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The vendor dashboard sends every status its tab spans, comma separated:
//
//	GET /chef/orders?status=pending,accepted,preparing
//
// Compared with `=`, that whole string matched no row, so a chef with a paid
// order waiting was shown an empty dashboard and no error.

func TestSplitCSVParam_TheVendorDashboardTabs(t *testing.T) {
	require.Equal(t,
		[]string{"pending", "accepted", "preparing"},
		splitCSVParam("pending,accepted,preparing"),
		"the exact parameter the dashboard sends must become three statuses")

	require.Equal(t,
		[]string{"pending", "accepted", "preparing", "ready"},
		splitCSVParam("pending,accepted,preparing,ready"),
		"LiveOrdersPage spans four")
}

func TestSplitCSVParam_SingleValueStillWorks(t *testing.T) {
	require.Equal(t, []string{"delivered"}, splitCSVParam("delivered"))
}

func TestSplitCSVParam_TolerantOfWhitespaceAndStrayCommas(t *testing.T) {
	require.Equal(t, []string{"pending", "accepted"}, splitCSVParam(" pending , accepted "))
	require.Equal(t, []string{"pending"}, splitCSVParam("pending,"))
	require.Equal(t, []string{"pending"}, splitCSVParam(",,pending,,"))
}

// nil means "no filter". Returning an empty slice instead would build
// `status IN ()`, which excludes every row — the same silent-empty failure this
// fix exists to remove, just arriving from the other direction.
func TestSplitCSVParam_EmptyMeansNoFilterNotMatchNothing(t *testing.T) {
	require.Nil(t, splitCSVParam(""))
	require.Nil(t, splitCSVParam("   "))
	require.Nil(t, splitCSVParam(","))
	require.Nil(t, splitCSVParam(" , , "))
}

// A blank entry must never survive into an IN list, or it would match rows
// whose status is the empty string.
func TestSplitCSVParam_NeverEmitsBlankValues(t *testing.T) {
	for _, in := range []string{"pending,,accepted", "pending, ,accepted", ",pending,accepted,"} {
		for _, v := range splitCSVParam(in) {
			require.NotEmpty(t, v, "input %q leaked a blank status into the IN list", in)
		}
	}
}
