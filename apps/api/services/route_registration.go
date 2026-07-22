package services

import (
	"errors"
	"fmt"
	"strings"
)

// route_registration.go — turning a chef's saved bank details into a payable
// Razorpay linked account (#740).
//
// Deliberately free of the database: the caller persists whatever comes back.
// That keeps the sequencing — account, stakeholder, product, settlement — and
// its idempotency testable on their own, and lets a retry resume from whatever
// the last attempt managed to create.

// ErrNoSettlementAccount is returned when a registration carries no bank
// account. Route settles by NEFT/IMPS to a bank account; there is no UPI VPA
// destination for a linked account, so a chef who nominated UPI cannot be
// registered on this rail.
var ErrNoSettlementAccount = errors.New("route: a bank account is required — Route cannot settle to a UPI id")

// SettlementRegistration is everything needed to make one chef payable.
//
// The Existing* fields make the call idempotent: a chef who already has a
// linked account or product configuration reuses it rather than minting a
// second, which would split their money across accounts with no way to merge.
type SettlementRegistration struct {
	ExistingAccountID string
	ExistingProductID string

	Email        string
	Phone        string
	LegalName    string
	ContactName  string
	BusinessType string

	Account SettlementAccount
}

// SettlementRegistrationResult is what the caller persists on the chef.
type SettlementRegistrationResult struct {
	AccountID        string
	ProductID        string
	ActivationStatus string
	Requirements     []RouteRequirement
}

// Payable reports whether a released transfer can actually reach the chef's
// bank. Only an activated configuration has a working destination.
func (r *SettlementRegistrationResult) Payable() bool {
	return r != nil && r.ActivationStatus == routeActivated
}

const routeActivated = "activated"

// RegisterSettlementAccount runs the onboarding sequence for one chef.
//
// On failure it still returns the partial result, so whatever was created
// before the error is persisted and a retry resumes rather than starting a
// second account.
func RegisterSettlementAccount(rz *RazorpayClient, in SettlementRegistration) (*SettlementRegistrationResult, error) {
	if strings.TrimSpace(in.Account.AccountNumber) == "" || strings.TrimSpace(in.Account.IFSC) == "" {
		return nil, ErrNoSettlementAccount
	}
	if rz == nil {
		return nil, errors.New("route: no Razorpay client configured")
	}

	res := &SettlementRegistrationResult{
		AccountID: in.ExistingAccountID,
		ProductID: in.ExistingProductID,
	}

	if res.AccountID == "" {
		acct, err := rz.CreateLinkedAccountV2(&V2AccountRequest{
			Email:        in.Email,
			Phone:        in.Phone,
			LegalName:    in.LegalName,
			BusinessType: in.BusinessType,
			ContactName:  in.ContactName,
		})
		if err != nil {
			return res, fmt.Errorf("route: create linked account: %w", err)
		}
		res.AccountID = acct.ID

		// Only for a brand-new account: Razorpay rejects a duplicate
		// stakeholder, and an existing account already has one.
		if _, err := rz.CreateStakeholder(res.AccountID, &StakeholderRequest{
			Name:  in.ContactName,
			Email: in.Email,
		}); err != nil {
			return res, fmt.Errorf("route: create stakeholder: %w", err)
		}
	}

	if res.ProductID == "" {
		prod, err := rz.RequestRouteProduct(res.AccountID)
		if err != nil {
			return res, fmt.Errorf("route: request route product: %w", err)
		}
		res.ProductID = prod.ID
	}

	cfg, err := rz.ConfigureRouteSettlement(res.AccountID, res.ProductID, &in.Account)
	if err != nil {
		return res, fmt.Errorf("route: configure settlement: %w", err)
	}
	res.ActivationStatus = cfg.ActivationStatus
	res.Requirements = cfg.Requirements
	return res, nil
}
