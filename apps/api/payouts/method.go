package payouts

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// method.go — where a payee's money is sent, and the rail-side registration of it.
//
// Batch.MethodID has pointed at this since the engine was designed; this is the
// row it points to.
//
// THE CENTRAL RULE OF THIS FILE: a PayoutMethod never holds the account number,
// the IFSC or the VPA. Those live in GCP Secret Manager under the vendor's own
// secrets (services.GetVendorSecret with "bank-account-number", "bank-ifsc",
// "upi-id"), and that is deliberate — they are the most sensitive data the
// platform holds, and keeping them out of Postgres keeps them out of every
// backup, replica, analytics export and admin list query.
//
// What is stored here is only what the platform needs to REASON about the
// destination without being able to reconstruct it: which kind of instrument it
// is, a masked hint for a human to recognise it, whether the rail has verified
// it, and the opaque id the rail gave us. Resolving the real instrument is a
// deliberate, audited read from Secret Manager at the moment of disbursement.

// MethodKind is the instrument type money is sent to.
type MethodKind string

const (
	// MethodBankAccount settles by IMPS/NEFT/RTGS to an account + IFSC.
	MethodBankAccount MethodKind = "bank_account"
	// MethodUPI settles to a VPA. Faster and cheaper, but capped per transfer
	// by NPCI limits, so the engine may have to fall back to a bank account.
	MethodUPI MethodKind = "upi"
)

// IsValid reports whether k is a kind the engine can disburse to.
func (k MethodKind) IsValid() bool {
	return k == MethodBankAccount || k == MethodUPI
}

// MethodStatus is how far a destination has got toward being payable.
//
// The states are deliberately about the RAIL's opinion, not ours: a bank account
// we believe is fine but that the rail has rejected is not payable, and the
// engine must never override that judgement locally.
type MethodStatus string

const (
	// MethodPending — registered with the rail, awaiting its verdict. Cashfree
	// returns beneficiary_status INITIATED here.
	MethodPending MethodStatus = "pending"
	// MethodVerified — the rail accepted it; money may be sent.
	MethodVerified MethodStatus = "verified"
	// MethodInvalid — the rail rejected it (bad IFSC, name mismatch, closed
	// account). Terminal until the payee supplies new details.
	MethodInvalid MethodStatus = "invalid"
	// MethodDisabled — an admin or the payee retired it. Never auto-selected.
	MethodDisabled MethodStatus = "disabled"
)

// Payable reports whether money may be sent to a method in this state.
//
// Only `verified` qualifies. Pending is NOT payable — sending to an unverified
// destination is how money reaches a mistyped account number and has to be
// chased back through a bank dispute rather than a reversal.
func (s MethodStatus) Payable() bool { return s == MethodVerified }

// PayoutMethod is one registered destination for one payee.
//
// A payee may have several (a bank account and a UPI id), but exactly one is
// Primary per (payee, kind) — enforced by a partial unique index rather than
// only in code, because two primaries would make destination selection depend on
// row order, and row order is not a thing to pay people by.
type PayoutMethod struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID string    `gorm:"type:varchar(64);not null;index:idx_payout_method_payee" json:"tenantId"`

	PayeeType PayeeType `gorm:"type:varchar(32);not null;index:idx_payout_method_payee" json:"payeeType"`
	PayeeID   uuid.UUID `gorm:"type:uuid;not null;index:idx_payout_method_payee" json:"payeeId"`

	Kind   MethodKind   `gorm:"type:varchar(24);not null" json:"kind"`
	Status MethodStatus `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`

	// Primary marks the method the engine picks by default for this payee.
	Primary bool `gorm:"not null;default:false" json:"primary"`

	// DisplayHint is a MASKED, human-recognisable fragment — "HDFC ••••4821" or
	// "sam@okhdfcbank". It exists so an admin can tell two destinations apart in
	// the payout queue without the platform storing anything that could
	// reconstruct the instrument. Never render anything else about the account.
	DisplayHint string `gorm:"type:varchar(64)" json:"displayHint"`

	// BeneficiaryName is the account holder's name as registered with the rail.
	// Stored (unlike the account number) because it is not secret, and because a
	// name mismatch is the single most common cause of a rejected transfer — an
	// admin needs to see what was actually sent.
	BeneficiaryName string `gorm:"type:varchar(120)" json:"beneficiaryName"`

	// Rail is which disbursement rail this registration belongs to
	// ("cashfree_payouts"). A method is rail-specific: the same bank account
	// registered with two rails is two rows, because each rail issues its own
	// beneficiary id and forms its own opinion of validity.
	Rail string `gorm:"type:varchar(32);not null;index" json:"rail"`

	// RailBeneficiaryID is the rail's opaque handle. Derived deterministically
	// from the payee (see BeneficiaryIDFor) so a re-registration after a lost
	// response resolves to the same beneficiary rather than creating a second.
	RailBeneficiaryID string `gorm:"type:varchar(64);index" json:"railBeneficiaryId,omitempty"`

	// RailStatusDetail carries the rail's own rejection wording, so an admin can
	// act on "IFSC not found" rather than a generic invalid.
	RailStatusDetail string `gorm:"type:text" json:"railStatusDetail,omitempty"`

	// VerifiedAt is when the rail last confirmed the destination.
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (PayoutMethod) TableName() string { return "payout_methods" }

// BeneficiaryIDFor derives the rail-side beneficiary id for a payee.
//
// Deterministic on purpose, and this is a money-safety property rather than a
// convenience: registering a beneficiary can time out AFTER the rail created it.
// A random id would mint a second beneficiary on retry, and the payee could then
// end up with two registrations whose verification states diverge — one verified,
// one rejected — with nothing to say which the next transfer used.
//
// The format satisfies Cashfree's constraint (alphanumeric, underscore, pipe and
// dot only, max 50 chars): "hc_chef_<uuid-without-hyphens>" is 8 + 32 = 40.
func BeneficiaryIDFor(ref PayeeRef) string {
	return fmt.Sprintf("hc_%s_%s",
		strings.ReplaceAll(string(ref.Type), "-", "_"),
		strings.ReplaceAll(ref.ID.String(), "-", ""))
}

// MaskAccountNumber renders the DisplayHint fragment for a bank account.
//
// Shows at most the last four digits and never the rest. Short or malformed
// values collapse to a bare mask rather than leaking what little they have — a
// six-digit "account number" is a data-entry error, and echoing it back in an
// admin list is not worth the diagnostic value.
func MaskAccountNumber(acct string) string {
	acct = strings.TrimSpace(acct)
	if len(acct) < 8 {
		return "••••"
	}
	return "••••" + acct[len(acct)-4:]
}

// MaskVPA renders the DisplayHint fragment for a UPI id.
//
// The handle (@okhdfcbank) is kept because it identifies the bank and is not
// secret; the local part is truncated. A VPA is quasi-public — it is what you
// hand someone to pay you — but it is still an identifier tied to a person, so
// it is not stored whole.
func MaskVPA(vpa string) string {
	vpa = strings.TrimSpace(vpa)
	at := strings.LastIndex(vpa, "@")
	if at <= 0 {
		return "••••"
	}
	local, handle := vpa[:at], vpa[at:]
	if len(local) <= 2 {
		return "••" + handle
	}
	return local[:2] + "••••" + handle
}

// Ref returns the method's payee identity.
func (m PayoutMethod) Ref() PayeeRef {
	return PayeeRef{Type: m.PayeeType, ID: m.PayeeID}
}

// Payable reports whether this method may receive money right now.
func (m PayoutMethod) Payable() bool {
	return m.Status.Payable() && m.RailBeneficiaryID != ""
}
