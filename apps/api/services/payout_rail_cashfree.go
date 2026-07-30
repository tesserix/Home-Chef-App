package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/homechef/api/payouts"
)

// payout_rail_cashfree.go — Cashfree Payouts as a payouts.Rail, plus the
// resolution of a payee's actual bank/UPI instrument at the moment of use.
//
// The rail itself is thin on purpose: the client already speaks the engine's
// types, so this adds only mode selection and the nil-client guard. The
// interesting part of the file is ResolvePayeeInstrument, below.

// CashfreeRail adapts CashfreePayoutClient to payouts.Rail for one mode.
//
// Mode is bound at construction rather than read per call, because a payout must
// execute against the SAME environment the batch was authorised in. Resolving it
// per call would let a mode flip mid-flight send a sandbox-approved batch to the
// live rail.
type CashfreeRail struct {
	mode string
}

// compile-time proof this satisfies the engine's port.
var _ payouts.Rail = (*CashfreeRail)(nil)

// NewCashfreeRail builds the rail for a mode.
func NewCashfreeRail(mode string) *CashfreeRail {
	return &CashfreeRail{mode: mode}
}

// Name is what Batch.Provider records.
func (r *CashfreeRail) Name() string { return CashfreePayoutRailName }

func (r *CashfreeRail) client() (*CashfreePayoutClient, error) {
	c := GetCashfreePayoutFor(r.mode)
	if c == nil {
		return nil, fmt.Errorf("%w: cashfree payouts [%s]", payouts.ErrRailNotConfigured, r.mode)
	}
	return c, nil
}

// EnsureBeneficiary registers a destination, idempotently on beneficiary id.
func (r *CashfreeRail) EnsureBeneficiary(ctx context.Context, req payouts.BeneficiaryRequest) (payouts.BeneficiaryResult, error) {
	c, err := r.client()
	if err != nil {
		return payouts.BeneficiaryResult{}, err
	}
	return c.CreateBeneficiary(ctx, req)
}

// Disburse sends one batch. See the interface contract: an ambiguous result
// must be resolved with GetPayoutByReference, never retried.
func (r *CashfreeRail) Disburse(ctx context.Context, req payouts.DisburseRequest) (payouts.DisburseResult, error) {
	c, err := r.client()
	if err != nil {
		return payouts.DisburseResult{}, err
	}
	return c.CreateTransfer(ctx, req)
}

// GetPayoutByReference resolves what happened to a disbursement.
func (r *CashfreeRail) GetPayoutByReference(ctx context.Context, idempotencyKey string) (payouts.DisburseResult, error) {
	c, err := r.client()
	if err != nil {
		return payouts.DisburseResult{}, err
	}
	return c.GetTransfer(ctx, idempotencyKey)
}

// --- Instrument resolution ---

// ResolvePayeeInstrument reads a payee's real bank/UPI details from GCP Secret
// Manager, immediately before they are needed.
//
// This is the ONLY place the platform assembles a full instrument, and it is
// deliberately a function rather than a cached value: the details are the most
// sensitive data held anywhere in the system, they live in Secret Manager
// precisely so they are not in Postgres, and the correct lifetime for one in
// memory is the length of a single rail call.
//
// Preference order is bank account first, UPI second — not the other way round,
// even though UPI is faster and cheaper. A bank account is what a chef's KYC and
// PAN are matched against, UPI VPAs are re-assignable between people, and a
// mis-sent NEFT is traceable through the banking system in a way a UPI transfer
// to a recycled handle is not. UPI is the fallback for payees who never supplied
// a bank account, which is common for drivers.
func ResolvePayeeInstrument(ctx context.Context, payeeID uuid.UUID, payeeType payouts.PayeeType) (payouts.Instrument, string, error) {
	getter := GetVendorSecret
	if payeeType == payouts.PayeeDeliveryPartner {
		getter = GetDriverSecret
	}
	id := payeeID.String()

	// Errors are swallowed per-field on purpose: a missing secret is the normal
	// case for a payee who supplied one instrument and not the other, and
	// GetVendorSecret cannot distinguish "absent" from "unreadable" without
	// leaking which. The completeness check below is what decides.
	accountName, _ := getter(ctx, id, "bank-account-name")
	accountNumber, _ := getter(ctx, id, "bank-account-number")
	ifsc, _ := getter(ctx, id, "bank-ifsc")

	if strings.TrimSpace(accountNumber) != "" && strings.TrimSpace(ifsc) != "" {
		return payouts.Instrument{
			Kind:          payouts.MethodBankAccount,
			AccountNumber: strings.TrimSpace(accountNumber),
			IFSC:          strings.ToUpper(strings.TrimSpace(ifsc)),
		}, strings.TrimSpace(accountName), nil
	}

	vpa, _ := getter(ctx, id, "upi-id")
	if strings.TrimSpace(vpa) != "" {
		return payouts.Instrument{
			Kind: payouts.MethodUPI,
			VPA:  strings.TrimSpace(vpa),
		}, strings.TrimSpace(accountName), nil
	}

	// Deliberately does not say WHICH field is missing. This error reaches an
	// admin screen, and "no payout destination" is all an admin needs; naming
	// the field would start describing the payee's banking setup in a UI that
	// is not otherwise allowed to know it.
	return payouts.Instrument{}, "", fmt.Errorf("payouts: payee %s has no usable payout destination on file", payeeID)
}

// DisplayHintFor renders the masked fragment stored on a PayoutMethod.
func DisplayHintFor(in payouts.Instrument) string {
	switch in.Kind {
	case payouts.MethodBankAccount:
		return payouts.MaskAccountNumber(in.AccountNumber)
	case payouts.MethodUPI:
		return payouts.MaskVPA(in.VPA)
	default:
		return "••••"
	}
}
