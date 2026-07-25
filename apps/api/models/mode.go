package models

import "strings"

// Mode partitions the platform into two worlds that share one database.
//
// "live" is the real marketplace: real customers, real money, live Razorpay
// credentials. "test" is a sandbox kitchen — visible only to the test-mode
// viewer allowlist, paid for with Razorpay test credentials, and excluded from
// every real-money and reporting path.
//
// Mode appears in two places with two different meanings. On ChefProfile it is
// configuration: which world the kitchen currently inhabits, freely flippable
// by an admin. On a transactional row it is a snapshot taken at creation and
// never changed, so a refund on an order paid in test mode still routes to the
// test gateway years after the chef went live.
const (
	ChefModeLive = "live"
	ChefModeTest = "test"
)

// NormalizeMode coerces any stored or user-supplied value to a known mode.
//
// Anything that is not recognisably "test" becomes "live". This asymmetry is
// deliberate and load-bearing: a corrupt or missing value must never hide a
// real kitchen from customers, and must never send a real payment through
// sandbox credentials (which would silently capture no money). The failure
// direction is always toward live.
func NormalizeMode(m string) string {
	if strings.EqualFold(strings.TrimSpace(m), ChefModeTest) {
		return ChefModeTest
	}
	return ChefModeLive
}

// IsTestMode reports whether a stored mode value means test.
func IsTestMode(m string) bool { return NormalizeMode(m) == ChefModeTest }
