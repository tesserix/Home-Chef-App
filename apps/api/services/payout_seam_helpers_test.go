package services

// payout_seam_helpers_test.go — shared harness for the payout seams. The gateway
// stubs that lived here went with the retired gateway client in #1086; the Cashfree seam
// is stubbed by withCashfreeServer (cashfree_test.go).

import (
	"testing"

	"github.com/homechef/api/config"
)

// orderPayoutFlagOn turns ORDER_PAYOUT_AUTO_RELEASE_ENABLED on for the test.
func orderPayoutFlagOn(t *testing.T) {
	t.Helper()
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: true}
}
