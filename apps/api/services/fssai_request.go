package services

// fssai_request.go — the lifecycle of a chef asking us to obtain their FSSAI
// registration. THE one writer of fssai_requests.
//
//	awaiting_payment → submitted → in_progress → filed → issued
//	                                              └→ rejected → refunded
//
// The chef assembles the request — details and documents — while it is unpaid,
// and paying submits it. So there is no paid-but-incomplete state to reconcile,
// and the fee is non-refundable from the moment it is taken.
//
// See .planning/FSSAI-IN-APP-REQUEST-DESIGN.md.

import (
	"context"
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

// ErrFssaiDocumentsMissing — payment was asked for before FSSAI's required
// documents were attached.
var ErrFssaiDocumentsMissing = errors.New("attach your photo and photo ID before paying")

// OpenFssaiRequestFor returns the chef's live request, or nil. Closed requests
// do not count — a chef whose registration lapses may legitimately ask again.
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

// LatestFssaiRequestFor is what the tracker renders: the live request if there
// is one, else the most recent closed one so the registration number we
// obtained stays visible.
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

// CreateFssaiRequest mints the draft at its priced quote, refusing when the
// chef already has a PAID one in flight.
func CreateFssaiRequest(db *gorm.DB, chef *models.ChefProfile, r *models.FssaiRequest) error {
	open, err := OpenFssaiRequestFor(db, chef.ID)
	if err != nil {
		return err
	}
	if open != nil {
		// An unpaid draft is not a commitment — the chef opened the form, thought
		// better of it, and has come back. Discard it rather than locking them out
		// of the service on the strength of a form they never paid for.
		if open.Status != models.FssaiAwaitingPayment {
			return ErrFssaiRequestOpen
		}
		if err := DiscardFssaiRequest(db, open); err != nil {
			return err
		}
	}
	// Assign the id here rather than leaning on the column default: gen_random_uuid()
	// is Postgres-only, so under any other engine the row is written with the nil
	// UUID and every later `WHERE id = ?` silently misses.
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

// DiscardFssaiRequest removes an UNPAID draft, the documents attached to it,
// and the FILES those documents point at.
//
// Hard delete: nothing was charged and nothing was filed, so there is no
// history worth keeping and abandoned rows would bury the real work. Deleting
// the objects matters more than the rows — they are Aadhaar and PAN images, and
// a file left behind after its row is gone outlives every record that it exists,
// including the account-deletion purge that walks those rows.
func DiscardFssaiRequest(db *gorm.DB, r *models.FssaiRequest) error {
	if r.Status != models.FssaiAwaitingPayment {
		return fmt.Errorf("a %s request cannot be discarded", r.Status)
	}
	// Read the object paths BEFORE the rows go, or there is nothing left to
	// tell us what to delete.
	var docs []models.FssaiRequestDocument
	if err := db.Where("request_id = ?", r.ID).Find(&docs).Error; err != nil {
		return err
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("request_id = ?", r.ID).
			Delete(&models.FssaiRequestDocument{}).Error; err != nil {
			return err
		}
		// Guard on status inside the delete too: a capture landing between the
		// check above and here must not lose a paid row.
		return tx.Where("id = ? AND status = ?", r.ID, models.FssaiAwaitingPayment).
			Delete(&models.FssaiRequest{}).Error
	})
	if err != nil {
		return err
	}
	purgeFssaiObjects(r.ID, docs)
	return nil
}

// FssaiDocumentsOfKind returns the rows about to be superseded or removed, so
// their stored files can be purged once the rows are gone.
func FssaiDocumentsOfKind(db *gorm.DB, requestID uuid.UUID, kind string) ([]models.FssaiRequestDocument, error) {
	var docs []models.FssaiRequestDocument
	err := db.Where("request_id = ? AND kind = ?", requestID, kind).Find(&docs).Error
	return docs, err
}

// PurgeFssaiObjects is purgeFssaiObjects for callers outside this package —
// the handler that removes an optional document.
func PurgeFssaiObjects(requestID uuid.UUID, docs []models.FssaiRequestDocument) {
	purgeFssaiObjects(requestID, docs)
}

// purgeFssaiObjects deletes the stored files for documents whose rows are gone.
//
// After the transaction, deliberately: object storage cannot participate in it,
// and a delete that fails must not roll back a discard the chef has been told
// happened. A failure is logged loudly because what is left behind is somebody's
// identity document.
func purgeFssaiObjects(requestID uuid.UUID, docs []models.FssaiRequestDocument) {
	ctx := context.Background()
	for _, d := range docs {
		if err := DeletePrivateFile(ctx, d.FileURL); err != nil {
			log.Printf("FSSAI request %s: ORPHANED %s document at %s: %v",
				requestID, d.Kind, d.FileURL, err)
		}
	}
}

// MarkFssaiPaid records the capture and submits the request, because the
// documents were gathered before payment was possible.
//
// Idempotent: a retried confirm, a webhook arriving after the client confirmed,
// or two taps land on one row. The status guard is in the WHERE clause rather
// than in Go, so two concurrent confirms cannot both apply — and only the one
// that wins sends the email.
func MarkFssaiPaid(db *gorm.DB, r *models.FssaiRequest, paymentRef string) error {
	if r.NeedsDocuments() {
		return ErrFssaiDocumentsMissing
	}
	now := time.Now().UTC()
	res := db.Model(&models.FssaiRequest{}).
		Where("id = ? AND status = ?", r.ID, models.FssaiAwaitingPayment).
		Updates(map[string]any{
			"status":       models.FssaiSubmitted,
			"payment_ref":  paymentRef,
			"paid_at":      now,
			"submitted_at": now,
			"updated_at":   now,
		})
	if res.Error != nil {
		return res.Error
	}
	// No row updated means it was already paid — the desired end state either
	// way, so this is a success, not a conflict.
	if res.RowsAffected == 0 {
		return nil
	}
	r.Status = models.FssaiSubmitted
	r.PaymentRef = paymentRef
	r.PaidAt = &now
	r.SubmittedAt = &now

	// Best-effort: the request is real and paid whether or not the mail lands. A
	// failure is logged loudly because it means work is sitting in the admin
	// queue that nobody has been told about.
	if err := SendFssaiRequestToOnboarding(r); err != nil {
		log.Printf("FSSAI request %s: onboarding email FAILED: %v", r.ID, err)
	}
	// Starts the SLA workflow. Published after the transition commits, so a
	// consumer that reads the row back always finds it submitted.
	PublishFssaiRequestEvent(r)
	return nil
}

// AttachFssaiDocument records one uploaded file, replacing any earlier file of
// the same kind — a chef who photographed their Aadhaar badly must be able to
// fix it, and two rows of one kind leave the filer guessing which is current.
func AttachFssaiDocument(db *gorm.DB, r *models.FssaiRequest, kind, fileURL, fileName string) error {
	if !models.IsFssaiDocKind(kind) {
		return fmt.Errorf("unknown document kind %q", kind)
	}
	if !r.AcceptsDocuments() {
		return fmt.Errorf("documents cannot be attached to a %s request", r.Status)
	}
	// The file being replaced has to go with its row. A chef re-photographing a
	// blurry Aadhaar would otherwise leave the blurry one in the bucket with
	// nothing pointing at it.
	superseded, err := FssaiDocumentsOfKind(db, r.ID, kind)
	if err != nil {
		return err
	}
	if err := db.Where("request_id = ? AND kind = ?", r.ID, kind).
		Delete(&models.FssaiRequestDocument{}).Error; err != nil {
		return err
	}
	purgeFssaiObjects(r.ID, superseded)

	doc := models.FssaiRequestDocument{
		ID:        uuid.New(),
		RequestID: r.ID, Kind: kind, FileURL: fileURL, FileName: fileName,
	}
	if err := db.Create(&doc).Error; err != nil {
		return err
	}
	// Re-read rather than mutating the caller's slice: "is this request
	// complete" must be answered from the database the payment will be checked
	// against.
	var docs []models.FssaiRequestDocument
	if err := db.Where("request_id = ?", r.ID).Find(&docs).Error; err != nil {
		return err
	}
	r.Documents = docs
	return nil
}

// FssaiOnboardingRecipient is where a submitted request is sent. Deliberately
// constant and not configurable: an admin able to redirect chefs' identity
// documents to an arbitrary address is a bigger hole than the convenience.
const FssaiOnboardingRecipient = "chef-onboarding@fe3dr.com"

// SendFssaiRequestToOnboarding emails a submitted request to the onboarding
// team. Documents are NAMED but never linked or attached: they are Aadhaar and
// PAN images, and a link mailed to an inbox outlives the screen it was minted
// for. Staff open them from the admin queue, behind authentication, on a URL
// that expires in fifteen minutes.
func SendFssaiRequestToOnboarding(r *models.FssaiRequest) error {
	svc := GetEmailService()
	if svc == nil {
		return errors.New("email service not configured")
	}

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

// fssaiEventSubjects maps a status to the subject announcing arrival at it.
// Terminal states share one `closed` subject: a consumer that only needs to
// stop watching should not have to enumerate every way a request can end.
var fssaiEventSubjects = map[string]string{
	models.FssaiSubmitted:        SubjectFssaiRequestSubmitted,
	models.FssaiInProgress:       SubjectFssaiRequestInProgress,
	models.FssaiMoreInfoRequired: SubjectFssaiRequestInfoNeeded,
	models.FssaiFiled:            SubjectFssaiRequestFiled,
	models.FssaiIssued:           SubjectFssaiRequestIssued,
	models.FssaiRejected:         SubjectFssaiRequestClosed,
	models.FssaiRefunded:         SubjectFssaiRequestClosed,
}

// PublishFssaiRequestEvent announces a status change.
//
// Best-effort by design: NATS being down must never roll back a transition an
// admin has already made and a chef has already been shown. The payload carries
// the status explicitly so a `closed` consumer can still tell rejected from
// refunded without a second lookup.
func PublishFssaiRequestEvent(r *models.FssaiRequest) {
	subject, ok := fssaiEventSubjects[r.Status]
	if !ok {
		return // drafts and unknown states are not worth announcing
	}
	err := PublishEvent(subject, "fssai.request", r.UserID, map[string]any{
		"requestId":      r.ID.String(),
		"chefId":         r.ChefID.String(),
		"status":         r.Status,
		"kitchenName":    r.KitchenName,
		"termYears":      r.TermYears,
		"mode":           models.NormalizeMode(r.Mode),
		"applicationRef": r.ApplicationRef,
		"registrationNo": r.RegistrationNo,
		"infoRequested":  r.InfoRequested,
		"awaitingChef":   models.FssaiAwaitingChef(r.Status),
	})
	if err != nil {
		log.Printf("FSSAI request %s: publish %s failed: %v", r.ID, subject, err)
	}
}

// fssaiChefNotice is what the chef is told on arriving at a status. Statuses
// absent from here are internal bookkeeping the chef does not need pinged about.
func fssaiChefNotice(r *models.FssaiRequest) (title, message string) {
	switch r.Status {
	case models.FssaiInProgress:
		return "We're preparing your FSSAI form",
			"Your application is with our team. We'll tell you when it's filed."
	case models.FssaiMoreInfoRequired:
		// The admin's words, verbatim — a generic "action needed" would make the
		// chef open the app to find out what, which is the whole cost of the state.
		return "We need something from you", r.InfoRequested
	case models.FssaiFiled:
		return "Filed with FSSAI",
			"Your application is lodged. Reference " + r.ApplicationRef +
				" — you can track it on the government portal yourself."
	case models.FssaiIssued:
		return "Your FSSAI registration is ready",
			"Registration " + r.RegistrationNo + ". Open the app to download your certificate."
	case models.FssaiRejected:
		return "We couldn't proceed with your FSSAI request", r.RejectedReason
	}
	return "", ""
}

// NotifyFssaiRequestStatus tells the chef what changed.
//
// Best-effort: a push service being down must not undo a transition an admin
// has already made. The tracker is the source of truth either way.
func NotifyFssaiRequestStatus(db *gorm.DB, r *models.FssaiRequest) {
	title, message := fssaiChefNotice(r)
	if title == "" {
		return
	}
	svc := GetNotificationService()
	if svc == nil {
		log.Printf("FSSAI request %s: no notification service — chef not told about %s",
			r.ID, r.Status)
		return
	}
	if r.UserID == uuid.Nil {
		log.Printf("FSSAI request %s: no user on the row — chef not told about %s",
			r.ID, r.Status)
		return
	}
	if err := svc.SaveUserNotification(&models.Notification{
		UserID:  r.UserID,
		Type:    "fssai_request_" + r.Status,
		Title:   title,
		Message: message,
	}); err != nil {
		log.Printf("FSSAI request %s: chef notification FAILED for %s: %v", r.ID, r.Status, err)
		return
	}
	log.Printf("FSSAI request %s: chef notified of %s", r.ID, r.Status)
}

// fssaiAdminTransitions is what an admin may move a request to. Forward-only
// through the working states. Unpaid drafts are absent: they are the chef's,
// and admin has nothing to move until money has been taken.
// more_info_required is reachable from every working state and returns to
// in_progress, because "we need something from you" can happen at any point up
// to issue and does not undo the work already done.
var fssaiAdminTransitions = map[string][]string{
	models.FssaiSubmitted: {
		models.FssaiInProgress, models.FssaiMoreInfoRequired,
		models.FssaiRejected, models.FssaiRefunded,
	},
	models.FssaiInProgress: {
		models.FssaiFiled, models.FssaiMoreInfoRequired,
		models.FssaiRejected, models.FssaiRefunded,
	},
	models.FssaiMoreInfoRequired: {
		models.FssaiInProgress, models.FssaiFiled,
		models.FssaiRejected, models.FssaiRefunded,
	},
	models.FssaiFiled: {
		models.FssaiIssued, models.FssaiMoreInfoRequired,
		models.FssaiRejected, models.FssaiRefunded,
	},
	models.FssaiRejected: {models.FssaiRefunded},
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
// status, or "" when none is needed. `filed` without a FoSCoS reference leaves
// the chef unable to track their own application.
func FssaiTransitionRequires(to string) string {
	switch to {
	case models.FssaiFiled:
		return "applicationRef"
	case models.FssaiIssued:
		return "registrationNo"
	case models.FssaiRejected:
		return "rejectedReason"
	case models.FssaiMoreInfoRequired:
		// Parking a request without saying what is missing leaves the chef
		// staring at a status they cannot act on.
		return "infoRequested"
	}
	return ""
}
