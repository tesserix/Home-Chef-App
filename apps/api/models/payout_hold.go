package models

// payout_hold.go — the payout hold state machine (#387). Decouples "delivered"
// from "chef paid": on delivery an order/day hold becomes
// awaiting_customer_confirmation instead of releasing funds. Only an explicit
// customer confirmation advances it to release_eligible, and only when no open
// OrderIssue disputes it. release_eligible moves NO money by itself — the real
// Razorpay ReleaseTransfer is driven later, off release_eligible, by the admin
// payout queue (#388).

// PayoutHoldStatus is the lifecycle of a delivered order/day's held chef payout.
type PayoutHoldStatus string

const (
	// PayoutHoldNone is the pre-delivery / not-applicable state (also the value
	// for meal-plan/consolidated orders that settle through their own paths).
	PayoutHoldNone PayoutHoldStatus = ""
	// PayoutHoldAwaitingConfirmation is set on delivery: the payout is parked
	// until the customer confirms receipt (or a follow-up timeout auto-confirms).
	PayoutHoldAwaitingConfirmation PayoutHoldStatus = "awaiting_customer_confirmation"
	// PayoutHoldReleaseEligible means the customer confirmed and no dispute is
	// open; the admin payout queue (#388) may now release the real transfer.
	PayoutHoldReleaseEligible PayoutHoldStatus = "release_eligible"
	// PayoutHoldReleased is set later by #388 once ReleaseTransfer has run.
	PayoutHoldReleased PayoutHoldStatus = "released"
	// PayoutHoldDisputed blocks release: an open OrderIssue forced it at confirm
	// time. A disputed hold can never transition to release_eligible.
	PayoutHoldDisputed PayoutHoldStatus = "disputed"
)
