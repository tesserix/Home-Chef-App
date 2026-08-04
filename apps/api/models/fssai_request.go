package models

import (
	"time"

	"github.com/google/uuid"
)

// FssaiRequest is a chef asking us to obtain their FSSAI registration for them —
// the in-app form behind the ₹50 + GST offer on fe3dr.com/fssai/.
//
// The chef pays us a filing fee, uploads their identity documents, and we file
// the application on FoSCoS on their behalf. The registration is issued in the
// CHEF's name: this row records the errand, never any claim over the licence.
//
// Payment comes FIRST (owner's call), so a row exists from the moment money is
// asked for rather than from the moment the request is complete. That ordering
// creates a real failure mode — a chef who pays and never uploads — so
// `awaiting_documents` is a first-class state with its own admin queue and
// refund path, not an edge case discovered later in a reconciliation.
type FssaiRequest struct {
	// Live/test data partition, so a sandbox kitchen's practice request can
	// never reach the onboarding inbox as real work.
	ModePartition

	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`

	Status string `gorm:"type:varchar(24);not null;default:awaiting_payment;index" json:"status"`

	// What the chef is registering — captured at request time rather than read
	// from the chef profile at send time, because the profile can change between
	// the request and the filing and the application must reflect what they
	// actually asked for.
	KitchenName    string `gorm:"type:varchar(160);not null;default:''" json:"kitchenName"`
	ApplicantName  string `gorm:"type:varchar(160);not null;default:''" json:"applicantName"`
	ContactPhone   string `gorm:"type:varchar(32);not null;default:''" json:"contactPhone"`
	ContactEmail   string `gorm:"type:varchar(160);not null;default:''" json:"contactEmail"`
	AddressLine1   string `gorm:"type:varchar(200);not null;default:''" json:"addressLine1"`
	AddressLine2   string `gorm:"type:varchar(200);not null;default:''" json:"addressLine2"`
	City           string `gorm:"type:varchar(80);not null;default:''" json:"city"`
	State          string `gorm:"type:varchar(80);not null;default:''" json:"state"`
	PostalCode     string `gorm:"type:varchar(16);not null;default:''" json:"postalCode"`
	// Years the chef wants to register for, 1–5. The government fee is ₹100 per
	// year and is paid to FSSAI directly by us on their behalf; it is NOT part of
	// the fee charged here.
	TermYears int `gorm:"not null;default:1" json:"termYears"`

	// The filing fee we charge, and the Cashfree charge that collected it. Frozen
	// on the row: a later price change must never restate what a chef was billed.
	FeeAmount     float64 `gorm:"type:numeric(12,2);not null;default:0" json:"feeAmount"`
	FeeTax        float64 `gorm:"type:numeric(12,2);not null;default:0" json:"feeTax"`
	FeeTotal      float64 `gorm:"type:numeric(12,2);not null;default:0" json:"feeTotal"`
	Currency      string  `gorm:"type:varchar(3);not null;default:INR" json:"currency"`
	PaymentRef    string  `gorm:"type:varchar(120);not null;default:''" json:"paymentRef,omitempty"`
	GatewayOrder  string  `gorm:"type:varchar(120);not null;default:''" json:"-"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`

	// The FoSCoS application reference, once we have actually filed. This is the
	// single most useful thing we can give a chef back — it is how they track
	// their own application with the state authority, independently of us.
	ApplicationRef string `gorm:"type:varchar(120);not null;default:''" json:"applicationRef,omitempty"`
	// The registration number once issued, which is what unlocks their menu.
	RegistrationNo string `gorm:"type:varchar(120);not null;default:''" json:"registrationNo,omitempty"`

	// Admin working notes and the reason shown to the chef on rejection. Kept
	// apart: an internal note must never surface on a chef's screen by accident.
	AdminNotes     string `gorm:"type:text;not null;default:''" json:"-"`
	RejectedReason string `gorm:"type:text;not null;default:''" json:"rejectedReason,omitempty"`

	// When the request (documents included) was handed to onboarding. Nil until
	// the email has actually been sent, so a resend is distinguishable from a
	// first send and a failed send does not look delivered.
	SubmittedAt *time.Time `json:"submittedAt,omitempty"`
	FiledAt     *time.Time `json:"filedAt,omitempty"`
	IssuedAt    *time.Time `json:"issuedAt,omitempty"`

	Documents []FssaiRequestDocument `gorm:"foreignKey:RequestID" json:"documents,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (FssaiRequest) TableName() string { return "fssai_requests" }

// FssaiRequestDocument is one uploaded file on a request. The file itself lives
// in object storage; this row is the reference plus what the chef said it is.
type FssaiRequestDocument struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RequestID uuid.UUID `gorm:"type:uuid;not null;index" json:"requestId"`
	// Kind is one of the FssaiDoc* constants — what FSSAI asks for, not a free
	// string, so an incomplete request can be detected rather than guessed at.
	Kind     string `gorm:"type:varchar(32);not null" json:"kind"`
	FileURL  string `gorm:"type:text;not null" json:"fileUrl"`
	FileName string `gorm:"type:varchar(255);not null;default:''" json:"fileName"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (FssaiRequestDocument) TableName() string { return "fssai_request_documents" }

// The three documents FSSAI asks for on a Registration Certificate, quoted from
// its own "Documents required for Registration Certificate": a photo, a
// government photo ID, and address proof ONLY where the business address differs
// from the one on that ID.
const (
	FssaiDocPhoto        = "photo"
	FssaiDocIdentity     = "identity"
	FssaiDocAddressProof = "address_proof"
)

// Request lifecycle.
//
// The chef's tracker collapses these into five steps; `awaiting_documents` is
// the honest extra state that pay-first creates and shows as "upload your
// documents" rather than pretending the request is already with us.
const (
	// FssaiAwaitingPayment — row minted, Cashfree order created, nothing paid.
	FssaiAwaitingPayment = "awaiting_payment"
	// FssaiAwaitingDocuments — paid, but we cannot file without the documents.
	FssaiAwaitingDocuments = "awaiting_documents"
	// FssaiSubmitted — paid AND documented; sent to onboarding.
	FssaiSubmitted = "submitted"
	// FssaiInProgress — an admin has picked it up and is preparing the form.
	FssaiInProgress = "in_progress"
	// FssaiFiled — lodged with FoSCoS; ApplicationRef is set.
	FssaiFiled = "filed"
	// FssaiIssued — the registration exists; RegistrationNo is set.
	FssaiIssued = "issued"
	// FssaiRejected — we could not proceed. RejectedReason is shown to the chef,
	// and anything already paid is refundable.
	FssaiRejected = "rejected"
	// FssaiRefunded — the fee was returned, whatever the reason.
	FssaiRefunded = "refunded"
)

// FssaiRequestOpen reports whether a request is still live work — used to stop a
// chef opening a second one while the first is in flight, and to scope the
// admin queue.
func FssaiRequestOpen(status string) bool {
	switch status {
	case FssaiIssued, FssaiRejected, FssaiRefunded:
		return false
	}
	return true
}

// NeedsDocuments reports whether the request still lacks a document FSSAI
// requires. Address proof is conditional: it is required only when the kitchen
// address differs from the one on the submitted photo ID, which only the chef
// can tell us — so it is never demanded here.
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
