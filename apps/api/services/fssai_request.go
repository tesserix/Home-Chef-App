package services

// fssai_request.go — the lifecycle of a chef asking us to obtain their FSSAI
// registration. THE one writer of fssai_requests.
//
//	awaiting_payment → awaiting_documents → submitted → in_progress → filed → issued
//	                                                                     └→ rejected → refunded
//
// Payment is taken FIRST, so the row exists from the moment money is asked for
// rather than from the moment the request is complete. That ordering creates a
// real failure mode — a chef who pays and never uploads — so awaiting_documents
// is a first-class state with its own admin queue and refund path, not an edge
// case discovered later in a reconciliation.
//
// See .planning/FSSAI-IN-APP-REQUEST-DESIGN.md.

import (
	"errors"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ErrFssaiRequestOpen — the chef already has one in flight. A second request
// would take a second payment for the same registration.
var ErrFssaiRequestOpen = errors.New("an FSSAI request is already in progress")

// OpenFssaiRequestFor returns the chef's live request, or nil when they have
// none. Closed requests (issued/rejected/refunded) do not count — a chef whose
// registration lapsed years later may legitimately ask again.
func OpenFssaiRequestFor(db *gorm.DB, chefID uuid.UUID) (*models.FssaiRequest, error) {
	var rows []models.FssaiRequest
	err := db.Preload("Documents").
		Where("chef_id = ?", chefID).
		Order("created_at DESC").
		Limit(5).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if models.FssaiRequestOpen(rows[i].Status) {
			return &rows[i], nil
		}
	}
	return nil, nil
}

// LatestFssaiRequestFor is what the app's tracker renders: the live request if
// there is one, else the most recent closed one so a chef can still see the
// registration number we obtained for them.
func LatestFssaiRequestFor(db *gorm.DB, chefID uuid.UUID) (*models.FssaiRequest, error) {
	var row models.FssaiRequest
	err := db.Preload("Documents").
		Where("chef_id = ?", chefID).
		Order("created_at DESC").First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// CreateFssaiRequest mints the row at its priced quote, refusing when the chef
// already has one open. The Cashfree order is minted by the caller against the
// row's own id, so a capture can never be attributed to the wrong request.
func CreateFssaiRequest(db *gorm.DB, chef *models.ChefProfile, r *models.FssaiRequest) error {
	open, err := OpenFssaiRequestFor(db, chef.ID)
	if err != nil {
		return err
	}
	if open != nil {
		// An UNPAID request is not a commitment — the chef opened the form,
		// thought better of it, and has come back. Nothing was charged, so the
		// abandoned row is discarded and they start again, possibly with a
		// different term or corrected details.
		//
		// Without this the guard below would lock a chef out of the service
		// permanently on the strength of a form they never paid for.
		if open.Status == models.FssaiAwaitingPayment {
			if err := DiscardFssaiRequest(db, open); err != nil {
				return err
			}
		} else {
			return ErrFssaiRequestOpen
		}
	}
	// Assign the id here rather than leaning on the column default. The default
	// is gen_random_uuid(), which only exists on Postgres — so under any other
	// engine the row is written with the nil UUID and every later
	// `WHERE id = ?` silently misses. Generating it in Go makes the row
	// addressable the moment it is created, on any database.
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	r.ModePartition = PartitionForChef(chef)
	r.ChefID = chef.ID
	r.UserID = chef.UserID
	r.Status = models.FssaiAwaitingPayment
	ApplyQuote(r, QuoteFssaiFiling(r.TermYears))
	return db.Create(r).Error
}

// DiscardFssaiRequest removes an UNPAID request and anything attached to it.
//
// Hard delete, not a status: nothing was charged and nothing was filed, so
// there is no history worth keeping — and leaving abandoned rows in the admin
// queue would bury the requests that are real work. Refuses anything paid,
// where the money makes the row a record.
func DiscardFssaiRequest(db *gorm.DB, r *models.FssaiRequest) error {
	if r.Status != models.FssaiAwaitingPayment {
		return fmt.Errorf("a %s request cannot be discarded", r.Status)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("request_id = ?", r.ID).
			Delete(&models.FssaiRequestDocument{}).Error; err != nil {
			return err
		}
		// Guard on status inside the delete too: a capture landing between the
		// check above and here must not lose a paid row.
		return tx.Where("id = ? AND status = ?", r.ID, models.FssaiAwaitingPayment).
			Delete(&models.FssaiRequest{}).Error
	})
}

// MarkFssaiPaid moves a request out of awaiting_payment once the gateway
// confirms the capture.
//
// Idempotent: a retried confirm, a webhook arriving after the client already
// confirmed, or two taps land on one row. The status guard is in the WHERE
// clause rather than in Go, so two concurrent confirms cannot both apply.
func MarkFssaiPaid(db *gorm.DB, r *models.FssaiRequest, paymentRef string) error {
	now := time.Now().UTC()
	res := db.Model(&models.FssaiRequest{}).
		Where("id = ? AND status = ?", r.ID, models.FssaiAwaitingPayment).
		Updates(map[string]any{
			"status":      models.FssaiAwaitingDocuments,
			"payment_ref": paymentRef,
			"paid_at":     now,
			"updated_at":  now,
		})
	if res.Error != nil {
		return res.Error
	}
	// No row updated means it was already paid — the desired end state either
	// way, so this is a success, not a conflict.
	if res.RowsAffected == 1 {
		r.Status = models.FssaiAwaitingDocuments
		r.PaymentRef = paymentRef
		r.PaidAt = &now
	}
	return nil
}

// AttachFssaiDocument records one uploaded file and, when the request now has
// everything FSSAI requires, advances it to submitted and sends it to
// onboarding.
//
// Re-uploading a kind REPLACES it: a chef who photographed their Aadhaar badly
// must be able to fix it, and two rows of the same kind would leave whoever
// files the application guessing which is current.
func AttachFssaiDocument(db *gorm.DB, r *models.FssaiRequest, kind, fileURL, fileName string) error {
	if r.Status != models.FssaiAwaitingDocuments && r.Status != models.FssaiSubmitted {
		return fmt.Errorf("documents cannot be attached to a %s request", r.Status)
	}
	if err := db.Where("request_id = ? AND kind = ?", r.ID, kind).
		Delete(&models.FssaiRequestDocument{}).Error; err != nil {
		return err
	}
	doc := models.FssaiRequestDocument{
		ID:        uuid.New(),
		RequestID: r.ID, Kind: kind, FileURL: fileURL, FileName: fileName,
	}
	if err := db.Create(&doc).Error; err != nil {
		return err
	}

	// Re-read the documents rather than mutating the in-memory slice: the caller
	// may hold a stale copy, and "is this request complete" must be answered from
	// the database that the onboarding email will be built from.
	var docs []models.FssaiRequestDocument
	if err := db.Where("request_id = ?", r.ID).Find(&docs).Error; err != nil {
		return err
	}
	r.Documents = docs
	if r.NeedsDocuments() || r.Status == models.FssaiSubmitted {
		return nil
	}
	return submitFssaiRequest(db, r)
}

// submitFssaiRequest advances a complete, paid request and hands it to
// onboarding. SubmittedAt is stamped only once, and the email is sent AFTER the
// transition commits — a mail failure must not roll the request back to a state
// the chef has already been told they are past.
func submitFssaiRequest(db *gorm.DB, r *models.FssaiRequest) error {
	now := time.Now().UTC()
	res := db.Model(&models.FssaiRequest{}).
		Where("id = ? AND status = ?", r.ID, models.FssaiAwaitingDocuments).
		Updates(map[string]any{
			"status": models.FssaiSubmitted, "submitted_at": now, "updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil // already submitted by a concurrent upload
	}
	r.Status = models.FssaiSubmitted
	r.SubmittedAt = &now

	// Best-effort: the request is real and paid whether or not the mail lands.
	// A failure is logged loudly because it means work is sitting in the admin
	// queue that nobody has been told about.
	if err := SendFssaiRequestToOnboarding(r); err != nil {
		log.Printf("FSSAI request %s: onboarding email FAILED: %v", r.ID, err)
	}
	return nil
}

// FssaiOnboardingRecipient is where a completed request is sent. Deliberately a
// constant and not configurable: it is an internal work queue, and an admin
// able to redirect chefs' identity documents to an arbitrary address is a
// bigger hole than any convenience it would buy.
const FssaiOnboardingRecipient = "chef-onboarding@fe3dr.com"

// SendFssaiRequestToOnboarding emails a submitted request to the onboarding
// team, with LINKS to the documents rather than attachments — these are Aadhaar
// and PAN images, and a mailbox is outside every control the platform has over
// PII at rest.
func SendFssaiRequestToOnboarding(r *models.FssaiRequest) error {
	svc := GetEmailService()
	if svc == nil {
		return errors.New("email service not configured")
	}

	// The documents are NAMED here but never linked. They are Aadhaar and PAN
	// images in a private bucket, and a link mailed to an inbox outlives the
	// screen it was minted for — staff open them from the admin queue, behind
	// authentication, on a URL that expires in fifteen minutes.
	var docs strings.Builder
	for _, d := range r.Documents {
		label := d.Kind
		if d.FileName != "" {
			label = d.Kind + " — " + d.FileName
		}
		docs.WriteString(fmt.Sprintf(`<li>%s</li>`, html.EscapeString(label)))
	}
	if docs.Len() == 0 {
		docs.WriteString("<li>none attached</li>")
	}

	addr := strings.TrimSpace(strings.Join([]string{
		r.AddressLine1, r.AddressLine2, r.City, r.State, r.PostalCode,
	}, ", "))

	body := fmt.Sprintf(`
<h2>FSSAI filing request</h2>
<p><strong>%s</strong> has paid for us to obtain their FSSAI registration.</p>
<table cellpadding="6">
<tr><td>Kitchen</td><td>%s</td></tr>
<tr><td>Applicant</td><td>%s</td></tr>
<tr><td>Phone</td><td>%s</td></tr>
<tr><td>Email</td><td>%s</td></tr>
<tr><td>Address</td><td>%s</td></tr>
<tr><td>Term</td><td>%d year(s)</td></tr>
<tr><td>Paid</td><td>%s %.2f (ref %s)</td></tr>
<tr><td>Request</td><td>%s</td></tr>
</table>
<h3>Documents</h3>
<ul>%s</ul>
<p>Kind of business: <strong>Home Based Canteens / Dabba Wallas</strong>,
turnover up to Rs. 1.5Cr [Registration]. File on FoSCoS, then set the
application reference on the request in admin so the chef can track it.</p>
<p>Open the documents from the admin queue — they are not attached or linked
here, because they are identity documents and this mailbox is not the place for
them.</p>`,
		html.EscapeString(r.KitchenName), html.EscapeString(r.KitchenName),
		html.EscapeString(r.ApplicantName), html.EscapeString(r.ContactPhone),
		html.EscapeString(r.ContactEmail), html.EscapeString(addr),
		r.TermYears, html.EscapeString(r.Currency), r.FeeTotal,
		html.EscapeString(r.PaymentRef), r.ID,
		docs.String(),
	)

	subject := fmt.Sprintf(
		"REQUEST TO APPLY FOR THE FSSAI LICENSE FOR HOME BASED COOK — %s", r.KitchenName)
	return svc.Send(FssaiOnboardingRecipient, subject, body)
}

// fssaiAdminTransitions is what an admin may move a request to. Forward-only
// through the working states; `refunded` is reachable from anywhere money is
// still held, because a request can be abandoned at any point before issue.
var fssaiAdminTransitions = map[string][]string{
	models.FssaiAwaitingPayment:   {models.FssaiRefunded},
	models.FssaiAwaitingDocuments: {models.FssaiRejected, models.FssaiRefunded},
	models.FssaiSubmitted:         {models.FssaiInProgress, models.FssaiRejected, models.FssaiRefunded},
	models.FssaiInProgress:        {models.FssaiFiled, models.FssaiRejected, models.FssaiRefunded},
	models.FssaiFiled:             {models.FssaiIssued, models.FssaiRejected, models.FssaiRefunded},
	models.FssaiRejected:          {models.FssaiRefunded},
}

// FssaiAdminMayTransition reports whether an admin may move a request from one
// status to another. Unknown or terminal statuses allow nothing.
func FssaiAdminMayTransition(from, to string) bool {
	for _, allowed := range fssaiAdminTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// FssaiTransitionRequires reports the field an admin must supply to reach a
// status, or "" when none is needed. `filed` without a FoSCoS reference is the
// one that matters: the reference is the whole point of the status to a chef,
// and a filed request without one leaves them unable to track their own
// application.
func FssaiTransitionRequires(to string) string {
	switch to {
	case models.FssaiFiled:
		return "applicationRef"
	case models.FssaiIssued:
		return "registrationNo"
	case models.FssaiRejected:
		return "rejectedReason"
	}
	return ""
}
