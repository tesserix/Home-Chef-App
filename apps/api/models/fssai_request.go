package models

import (
	"time"

	"github.com/google/uuid"
)

// FssaiRequest is a chef asking us to obtain their FSSAI registration for them.
// The registration is issued in the CHEF's name: this row records the errand,
// never any claim over the licence.
//
// Documents are collected BEFORE payment, so a paid request is always complete
// and the fee is non-refundable from the moment it is taken.
type FssaiRequest struct {
	// Live/test data partition, so a sandbox kitchen's practice request can
	// never reach the onboarding inbox as real work.
	ModePartition

	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`

	Status string `gorm:"type:varchar(24);not null;default:awaiting_payment;index" json:"status"`

	// Captured at request time, not read from the chef profile at filing time:
	// the profile can change in between and the application must reflect what
	// the chef actually asked for.
	KitchenName   string `gorm:"type:varchar(160);not null;default:''" json:"kitchenName"`
	ApplicantName string `gorm:"type:varchar(160);not null;default:''" json:"applicantName"`
	ContactPhone  string `gorm:"type:varchar(32);not null;default:''" json:"contactPhone"`
	ContactEmail  string `gorm:"type:varchar(160);not null;default:''" json:"contactEmail"`
	AddressLine1  string `gorm:"type:varchar(200);not null;default:''" json:"addressLine1"`
	AddressLine2  string `gorm:"type:varchar(200);not null;default:''" json:"addressLine2"`
	City          string `gorm:"type:varchar(80);not null;default:''" json:"city"`
	State         string `gorm:"type:varchar(80);not null;default:''" json:"state"`
	PostalCode    string `gorm:"type:varchar(16);not null;default:''" json:"postalCode"`
	TermYears     int    `gorm:"not null;default:1" json:"termYears"`

	// Frozen at creation: a later price change must never restate what a chef
	// was billed.
	FeeAmount    float64    `gorm:"type:numeric(12,2);not null;default:0" json:"feeAmount"`
	FeeTax       float64    `gorm:"type:numeric(12,2);not null;default:0" json:"feeTax"`
	FeeTotal     float64    `gorm:"type:numeric(12,2);not null;default:0" json:"feeTotal"`
	Currency     string     `gorm:"type:varchar(3);not null;default:INR" json:"currency"`
	PaymentRef   string     `gorm:"type:varchar(120);not null;default:''" json:"paymentRef,omitempty"`
	GatewayOrder string     `gorm:"type:varchar(120);not null;default:''" json:"-"`
	PaidAt       *time.Time `json:"paidAt,omitempty"`

	// The FoSCoS reference is what lets a chef track their own application with
	// the state authority, independently of us.
	ApplicationRef string `gorm:"type:varchar(120);not null;default:''" json:"applicationRef,omitempty"`
	RegistrationNo string `gorm:"type:varchar(120);not null;default:''" json:"registrationNo,omitempty"`

	// Kept apart so an internal note can never surface on a chef's screen.
	AdminNotes     string `gorm:"type:text;not null;default:''" json:"-"`
	RejectedReason string `gorm:"type:text;not null;default:''" json:"rejectedReason,omitempty"`

	SubmittedAt *time.Time `json:"submittedAt,omitempty"`
	FiledAt     *time.Time `json:"filedAt,omitempty"`
	IssuedAt    *time.Time `json:"issuedAt,omitempty"`

	Documents []FssaiRequestDocument `gorm:"foreignKey:RequestID" json:"documents,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (FssaiRequest) TableName() string { return "fssai_requests" }

// FssaiRequestDocument is one uploaded file on a request. FileURL is an object
// path in the private bucket, never a durable link.
type FssaiRequestDocument struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RequestID uuid.UUID `gorm:"type:uuid;not null;index" json:"requestId"`
	Kind      string    `gorm:"type:varchar(32);not null" json:"kind"`
	FileURL   string    `gorm:"type:text;not null" json:"fileUrl"`
	FileName  string    `gorm:"type:varchar(255);not null;default:''" json:"fileName"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (FssaiRequestDocument) TableName() string { return "fssai_request_documents" }

// What FSSAI asks for on a Registration Certificate: a photo, a government
// photo ID, and address proof ONLY where the kitchen address differs from the
// one on that ID.
const (
	FssaiDocPhoto        = "photo"
	FssaiDocIdentity     = "identity"
	FssaiDocAddressProof = "address_proof"
)

// IsFssaiDocKind reports whether a kind is one FSSAI recognises. The one place
// the set is checked, so an upload path cannot drift from an attach path.
func IsFssaiDocKind(kind string) bool {
	switch kind {
	case FssaiDocPhoto, FssaiDocIdentity, FssaiDocAddressProof:
		return true
	}
	return false
}

// Request lifecycle. Documents are gathered during awaiting_payment, so paying
// submits the request outright.
//
//	awaiting_payment → submitted → in_progress → filed → issued
//	                                              └→ rejected → refunded
const (
	// FssaiAwaitingPayment — the chef's draft: details captured, documents
	// being gathered, nothing charged and nothing owed.
	FssaiAwaitingPayment = "awaiting_payment"
	// FssaiSubmitted — paid and complete; sent to onboarding.
	FssaiSubmitted = "submitted"
	// FssaiInProgress — an admin has picked it up and is preparing the form.
	FssaiInProgress = "in_progress"
	// FssaiFiled — lodged with FoSCoS; ApplicationRef is set.
	FssaiFiled = "filed"
	// FssaiIssued — the registration exists; RegistrationNo is set.
	FssaiIssued = "issued"
	// FssaiRejected — we could not proceed; RejectedReason is shown to the chef.
	FssaiRejected = "rejected"
	// FssaiRefunded — admin-only, for a chargeback or a refund we are obliged to
	// make. The chef-facing fee is non-refundable and no chef action reaches it.
	FssaiRefunded = "refunded"
)

// FssaiRequestOpen reports whether a request is still live work — used to stop
// a chef opening a second one, and to scope the admin queue.
func FssaiRequestOpen(status string) bool {
	switch status {
	case FssaiIssued, FssaiRejected, FssaiRefunded:
		return false
	}
	return true
}

// FssaiPaid reports whether money has been taken. Past this point the chef
// cannot cancel and the fee is not refundable.
func FssaiPaid(status string) bool {
	return status != FssaiAwaitingPayment && FssaiRequestOpen(status)
}

// AcceptsDocuments reports whether a document may still be attached. Open until
// we file: before payment the chef is assembling the request, and afterwards
// they may still be asked to replace something unreadable.
func (r *FssaiRequest) AcceptsDocuments() bool {
	switch r.Status {
	case FssaiAwaitingPayment, FssaiSubmitted, FssaiInProgress:
		return true
	}
	return false
}

// NeedsDocuments reports whether the request still lacks a document FSSAI
// requires. Address proof is conditional on the kitchen address differing from
// the one on the photo ID, which only the chef can tell us, so it is never
// demanded here.
func (r *FssaiRequest) NeedsDocuments() bool {
	var photo, identity bool
	for _, d := range r.Documents {
		switch d.Kind {
		case FssaiDocPhoto:
			photo = true
		case FssaiDocIdentity:
			identity = true
		}
	}
	return !photo || !identity
}
