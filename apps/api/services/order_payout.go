package services

import (
	"github.com/homechef/api/config"
)

// order_payout.go — the flag that gates live payout movement.
//
// The transfer layer this file used to hold (release / reverse / partial claw-back
// of an order's held chef and rider shares) went with the retired gateway in #1086.
// Cashfree captures the whole order to the platform and the chef's share is split
// on release through Easy Split (ReleaseOrderSplit), with the remainder settled on
// the weekly statement → payout batch path; neither has a transfer to hold.

// payoutMovementEnabled reports whether live payout movement is on.
func payoutMovementEnabled() bool {
	return config.AppConfig != nil && config.AppConfig.OrderPayoutAutoReleaseEnabled
}
