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
	// ExistingStakeholderCreated marks that a prior attempt already got the
	// stakeholder attached to ExistingAccountID. The stakeholder step is
	// skipped ONLY when this is true — independent of whether AccountID is
	// freshly created or reused this call. Without a field of its own, the
	// stakeholder step used to be nested inside "account was just created",
	// so an account-created-but-stakeholder-failed retry would carry a
	// non-empty AccountID and skip the stakeholder forever, never able to
	// complete registration (review finding 3).
	ExistingStakeholderCreated bool

	Email        string
	Phone        string
	LegalName    string
	ContactName  string
	BusinessType string

	Account SettlementAccount
}

// SettlementRegistrationResult is what the caller persists on the chef.
type SettlementRegistrationResult struct {
	AccountID string
	ProductID string
	// StakeholderCreated records that the stakeholder step has succeeded —
	// either just now or on an earlier attempt, carried in via
	// ExistingStakeholderCreated — mirroring how AccountID/ProductID
	// themselves persist across retries.
	StakeholderCreated bool
	ActivationStatus   string
	Requirements       []RouteRequirement
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
		AccountID:          in.ExistingAccountID,
		ProductID:          in.ExistingProductID,
		StakeholderCreated: in.ExistingStakeholderCreated,
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
	}

	// Runs whenever the stakeholder hasn't succeeded yet — NOT nested inside
	// "account was just created". A fresh account whose stakeholder call then
	// fails must still be retried on the next attempt; gating this on
	// AccountID's freshness (the old shape) meant a stakeholder failure right
	// after account creation left AccountID non-empty forever, so every later
	// retry skipped BOTH steps and the chef could never complete registration.
	if !res.StakeholderCreated {
		_, err := rz.CreateStakeholder(res.AccountID, &StakeholderRequest{
			Name:  in.ContactName,
			Email: in.Email,
		})
		switch {
		case err == nil:
			res.StakeholderCreated = true
		case isStakeholderAlreadyExists(err):
			// Razorpay already has a stakeholder on this account — e.g. an
			// earlier attempt succeeded server-side but our client never saw
			// the response. That's the goal state, not a failure to report.
			res.StakeholderCreated = true
		default:
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

// isStakeholderAlreadyExists recognizes Razorpay's "stakeholder already
// exists" style rejection as the goal state rather than a failure. Without
// this, a retry that lands on an account whose stakeholder actually was
// created by an earlier attempt (e.g. the create call succeeded at Razorpay
// but the response never reached our client) would error on every
// subsequent attempt forever, even though nothing is actually wrong.
func isStakeholderAlreadyExists(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "already exist")
}
