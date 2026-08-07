package services

import (
	"github.com/homechef/api/config"
)

// order_payout.go — the flag that gates live payout movement.
//
// The Route transfer layer this file used to hold (release / reverse / partial
// claw-back of an order's held chef and rider transfers) went with the retired
// gateway in #1086. Cashfree captures the whole order to the platform and splits through
// Easy Split (ReleaseOrderSplit), with the remainder settled on the weekly
// statement → payout batch path; neither has a transfer to hold or reverse.

// payoutMovementEnabled reports whether live payout movement is on.
func payoutMovementEnabled() bool {
	return config.AppConfig != nil && config.AppConfig.OrderPayoutAutoReleaseEnabled
}
