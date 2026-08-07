package services

import (
	"strings"

	"github.com/homechef/api/models"
)

// easy_split_status.go — the chef's view of their payout registration.
//
// Cashfree's status strings are gateway vocabulary and must not reach a chef
// (#1082). The mapping lives here, server-side, so every surface — vendor app,
// admin, email — says the same thing, and so a state Cashfree adds tomorrow
// changes one function rather than each client's guesswork.

// Chef-facing payout registration stages.
const (
	PayoutRegistrationNone     = "none"
	PayoutRegistrationPending  = "pending"
	PayoutRegistrationVerified = "verified"
	PayoutRegistrationFailed   = "failed"
)

// PayoutRegistration is what the chef is told about their bank registration.
type PayoutRegistration struct {
	State   string `json:"state"`
	Message string `json:"message"`
}

// Verified reports whether order money can reach the chef straight from
// capture — the same test BuildOrderSplit applies.
func (r PayoutRegistration) Verified() bool { return r.State == PayoutRegistrationVerified }

// PayoutRegistrationFor translates a chef's stored Cashfree vendor state into
// plain words.
//
// Only BLOCKED and DELETED are terminal. An unrecognised state is reported as
// pending, not failed: telling a chef their details are wrong when Cashfree has
// merely added a state sends them to support over nothing.
func PayoutRegistrationFor(chef *models.ChefProfile) PayoutRegistration {
	if chef == nil || chef.CashfreeVendorID == "" {
		return PayoutRegistration{
			State:   PayoutRegistrationNone,
			Message: "Add your bank details to start receiving payouts",
		}
	}
	switch {
	case strings.EqualFold(chef.CashfreeVendorStatus, CashfreeVendorActive):
		return PayoutRegistration{State: PayoutRegistrationVerified, Message: "Verified — payouts active"}
	case strings.EqualFold(chef.CashfreeVendorStatus, CashfreeVendorBlocked),
		strings.EqualFold(chef.CashfreeVendorStatus, CashfreeVendorDeleted),
		strings.EqualFold(chef.CashfreeVendorStatus, CashfreeVendorBankValidationFailed):
		return PayoutRegistration{
			State:   PayoutRegistrationFailed,
			Message: "Couldn't verify — please check your details",
		}
	default:
		return PayoutRegistration{State: PayoutRegistrationPending, Message: "Pending verification"}
	}
}
