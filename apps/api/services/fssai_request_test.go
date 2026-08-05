package services

// fssai_request_test.go — the filing service a chef pays for.
//
// Every branch here either takes a chef's money or decides whether they can
// have it back, so the arithmetic, the pay-first failure mode, the idempotency
// of a retried capture, and the admin transition guards are all pinned.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func setupFssaiDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE fssai_requests (mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
		id TEXT PRIMARY KEY, chef_id TEXT, user_id TEXT, status TEXT DEFAULT 'awaiting_payment',
		kitchen_name TEXT DEFAULT '', applicant_name TEXT DEFAULT '', contact_phone TEXT DEFAULT '',
		contact_email TEXT DEFAULT '', address_line1 TEXT DEFAULT '', address_line2 TEXT DEFAULT '',
		city TEXT DEFAULT '', state TEXT DEFAULT '', postal_code TEXT DEFAULT '', term_years INTEGER DEFAULT 1,
		fee_amount REAL DEFAULT 0, fee_tax REAL DEFAULT 0, fee_total REAL DEFAULT 0, currency TEXT DEFAULT 'INR',
		payment_ref TEXT DEFAULT '', gateway_order TEXT DEFAULT '', paid_at DATETIME,
		application_ref TEXT DEFAULT '', registration_no TEXT DEFAULT '', admin_notes TEXT DEFAULT '',
		rejected_reason TEXT DEFAULT '', submitted_at DATETIME, filed_at DATETIME, issued_at DATETIME,
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE fssai_request_documents (id TEXT PRIMARY KEY, request_id TEXT,
		kind TEXT, file_url TEXT, file_name TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME)`).Error)
	return db
}

// fssaiPolicy installs the owner's pricing: ₹50 service fee, ₹100/yr to FSSAI, 18% GST.
func fssaiPolicy(t *testing.T) {
	t.Helper()
	p := DefaultPlatformPolicy()
	p.FssaiFilingEnabled = true
	p.FssaiServiceFee = 50
	p.FssaiGovernmentFeePerYear = 100
	p.FssaiGstPercent = 18
	withPlatformPolicy(t, p)
}

func fssaiChef() *models.ChefProfile {
	return &models.ChefProfile{ID: uuid.New(), UserID: uuid.New()}
}

// The figure a chef is actually charged for one year: ₹100 to FSSAI + ₹18 GST
// on it + ₹50 to us + ₹9 GST on ours.
func TestQuoteFssaiFiling_OneYear(t *testing.T) {
	fssaiPolicy(t)
	q := QuoteFssaiFiling(1)
	require.Equal(t, 100.00, q.GovernmentFee)
	require.Equal(t, 18.00, q.GovernmentTax)
	require.Equal(t, 50.00, q.ServiceFee)
	require.Equal(t, 9.00, q.ServiceTax)
	require.Equal(t, 177.00, q.Total)
}

// The government fee multiplies by the term; ours never does — it is one form
// whether the chef registers for one year or five.
func TestQuoteFssaiFiling_ServiceFeeDoesNotRepeatPerYear(t *testing.T) {
	fssaiPolicy(t)
	five := QuoteFssaiFiling(5)
	require.Equal(t, 500.00, five.GovernmentFee)
	require.Equal(t, 90.00, five.GovernmentTax)
	require.Equal(t, 50.00, five.ServiceFee, "the filing fee is per application, not per year")
	require.Equal(t, 649.00, five.Total)
}

// The total must be the exact sum of the lines the chef reads, to the paise.
func TestQuoteFssaiFiling_TotalIsTheSumOfItsLines(t *testing.T) {
	fssaiPolicy(t)
	for y := 1; y <= 5; y++ {
		q := QuoteFssaiFiling(y)
		require.InDelta(t, q.GovernmentFee+q.GovernmentTax+q.ServiceFee+q.ServiceTax, q.Total, 0.0001,
			"the breakdown must add up to what is charged (term %d)", y)
	}
}

// A term outside 1–5 is clamped, never priced as nonsense.
func TestQuoteFssaiFiling_ClampsTheTerm(t *testing.T) {
	fssaiPolicy(t)
	require.Equal(t, QuoteFssaiFiling(1).Total, QuoteFssaiFiling(0).Total)
	require.Equal(t, QuoteFssaiFiling(5).Total, QuoteFssaiFiling(99).Total)
}

// Pricing is policy, not code: an admin correcting the government fee changes
// what is charged without a deploy.
func TestQuoteFssaiFiling_FollowsThePolicy(t *testing.T) {
	p := DefaultPlatformPolicy()
	p.FssaiServiceFee = 75
	p.FssaiGovernmentFeePerYear = 118
	p.FssaiGstPercent = 0 // e.g. if the CA rules it a pure-agent pass-through
	withPlatformPolicy(t, p)

	q := QuoteFssaiFiling(1)
	require.Equal(t, 118.00, q.GovernmentFee)
	require.Equal(t, 0.00, q.GovernmentTax)
	require.Equal(t, 193.00, q.Total)
}

// The charged figures are FROZEN on the row: a later price change must never
// restate what a chef already paid.
func TestCreateFssaiRequest_FreezesThePrice(t *testing.T) {
	fssaiPolicy(t)
	db := setupFssaiDB(t)
	chef := fssaiChef()
	r := models.FssaiRequest{TermYears: 2, KitchenName: "Saffron"}
	require.NoError(t, CreateFssaiRequest(db, chef, &r))

	require.Equal(t, models.FssaiAwaitingPayment, r.Status)
	require.Equal(t, 295.00, r.FeeTotal) // 200 + 36 + 50 + 9

	p := DefaultPlatformPolicy()
	p.FssaiServiceFee = 500
	withPlatformPolicy(t, p)

	var stored models.FssaiRequest
	require.NoError(t, db.First(&stored, "id = ?", r.ID).Error)
	require.Equal(t, 295.00, stored.FeeTotal, "a price rise must not restate a charged request")
}

// A chef may open the form, see the price and walk away — then come back and
// start again. The abandoned UNPAID row must not lock them out of the service.
func TestCreateFssaiRequest_UnpaidRequestIsSuperseded(t *testing.T) {
	fssaiPolicy(t)
	db := setupFssaiDB(t)
	chef := fssaiChef()

	first := models.FssaiRequest{TermYears: 1}
	require.NoError(t, CreateFssaiRequest(db, chef, &first))

	second := models.FssaiRequest{TermYears: 5}
	require.NoError(t, CreateFssaiRequest(db, chef, &second),
		"an unpaid request is not a commitment; the chef can change their mind")

	var n int64
	db.Model(&models.FssaiRequest{}).Where("chef_id = ?", chef.ID).Count(&n)
	require.Equal(t, int64(1), n, "the abandoned row must not linger in the admin queue")
	require.Equal(t, 5, second.TermYears)
}

// Once PAID it is a commitment: a second request would take a second payment
// for the same registration.
func TestCreateFssaiRequest_PaidRequestBlocksASecond(t *testing.T) {
	fssaiPolicy(t)
	db := setupFssaiDB(t)
	chef := fssaiChef()

	first := models.FssaiRequest{TermYears: 1}
	require.NoError(t, CreateFssaiRequest(db, chef, &first))
	require.NoError(t, MarkFssaiPaid(db, &first, "pay_1"))

	second := models.FssaiRequest{TermYears: 1}
	require.ErrorIs(t, CreateFssaiRequest(db, chef, &second), ErrFssaiRequestOpen)
}

// A paid request is never discarded — the money makes the row a record.
func TestDiscardFssaiRequest_RefusesAPaidRequest(t *testing.T) {
	fssaiPolicy(t)
	db := setupFssaiDB(t)
	chef := fssaiChef()
	r := models.FssaiRequest{TermYears: 1}
	require.NoError(t, CreateFssaiRequest(db, chef, &r))
	require.NoError(t, MarkFssaiPaid(db, &r, "pay_1"))

	require.Error(t, DiscardFssaiRequest(db, &r))
	var n int64
	db.Model(&models.FssaiRequest{}).Where("id = ?", r.ID).Count(&n)
	require.Equal(t, int64(1), n)
}

// A retried confirm, a webhook landing after the client already confirmed, or
// a double tap must all settle on one paid row.
func TestMarkFssaiPaid_IsIdempotent(t *testing.T) {
	fssaiPolicy(t)
	db := setupFssaiDB(t)
	chef := fssaiChef()
	r := models.FssaiRequest{TermYears: 1}
	require.NoError(t, CreateFssaiRequest(db, chef, &r))

	require.NoError(t, MarkFssaiPaid(db, &r, "pay_1"))
	require.NoError(t, MarkFssaiPaid(db, &r, "pay_2"))

	var stored models.FssaiRequest
	require.NoError(t, db.First(&stored, "id = ?", r.ID).Error)
	require.Equal(t, models.FssaiAwaitingDocuments, stored.Status)
	require.Equal(t, "pay_1", stored.PaymentRef, "the first capture stands")
}

// The request only reaches onboarding once it is paid AND has the documents
// FSSAI requires — there is nothing to send before that.
func TestAttachFssaiDocument_SubmitsOnlyWhenComplete(t *testing.T) {
	fssaiPolicy(t)
	db := setupFssaiDB(t)
	chef := fssaiChef()
	r := models.FssaiRequest{TermYears: 1, KitchenName: "Saffron"}
	require.NoError(t, CreateFssaiRequest(db, chef, &r))
	require.NoError(t, MarkFssaiPaid(db, &r, "pay_1"))

	require.NoError(t, AttachFssaiDocument(db, &r, models.FssaiDocPhoto, "https://x/p.jpg", "p.jpg"))
	require.Equal(t, models.FssaiAwaitingDocuments, r.Status, "a photo alone is not a filing")

	require.NoError(t, AttachFssaiDocument(db, &r, models.FssaiDocIdentity, "https://x/a.jpg", "a.jpg"))
	require.Equal(t, models.FssaiSubmitted, r.Status)
	require.NotNil(t, r.SubmittedAt)
}

// Address proof is conditional — required only when the kitchen address differs
// from the ID — so it must never hold a request back.
func TestAttachFssaiDocument_AddressProofIsOptional(t *testing.T) {
	fssaiPolicy(t)
	db := setupFssaiDB(t)
	chef := fssaiChef()
	r := models.FssaiRequest{TermYears: 1}
	require.NoError(t, CreateFssaiRequest(db, chef, &r))
	require.NoError(t, MarkFssaiPaid(db, &r, "pay_1"))
	require.NoError(t, AttachFssaiDocument(db, &r, models.FssaiDocPhoto, "https://x/p.jpg", ""))
	require.NoError(t, AttachFssaiDocument(db, &r, models.FssaiDocIdentity, "https://x/a.jpg", ""))
	require.Equal(t, models.FssaiSubmitted, r.Status)
}

// A chef who photographed their Aadhaar badly must be able to replace it —
// and two rows of the same kind would leave the filer guessing which is current.
func TestAttachFssaiDocument_ReplacesTheSameKind(t *testing.T) {
	fssaiPolicy(t)
	db := setupFssaiDB(t)
	chef := fssaiChef()
	r := models.FssaiRequest{TermYears: 1}
	require.NoError(t, CreateFssaiRequest(db, chef, &r))
	require.NoError(t, MarkFssaiPaid(db, &r, "pay_1"))
	require.NoError(t, AttachFssaiDocument(db, &r, models.FssaiDocPhoto, "https://x/blurry.jpg", ""))
	require.NoError(t, AttachFssaiDocument(db, &r, models.FssaiDocPhoto, "https://x/clear.jpg", ""))

	var docs []models.FssaiRequestDocument
	require.NoError(t, db.Where("request_id = ? AND kind = ?", r.ID, models.FssaiDocPhoto).Find(&docs).Error)
	require.Len(t, docs, 1)
	require.Equal(t, "https://x/clear.jpg", docs[0].FileURL)
}

// Documents cannot be attached before the money clears — the pay-first ordering
// must not be bypassable by uploading straight into an unpaid request.
func TestAttachFssaiDocument_RefusesAnUnpaidRequest(t *testing.T) {
	fssaiPolicy(t)
	db := setupFssaiDB(t)
	chef := fssaiChef()
	r := models.FssaiRequest{TermYears: 1}
	require.NoError(t, CreateFssaiRequest(db, chef, &r))
	require.Error(t, AttachFssaiDocument(db, &r, models.FssaiDocPhoto, "https://x/p.jpg", ""))
}

// Filed without a FoSCoS reference tells a chef we lodged their application
// while leaving them nothing to track it with.
func TestFssaiAdminTransitions(t *testing.T) {
	require.True(t, FssaiAdminMayTransition(models.FssaiSubmitted, models.FssaiInProgress))
	require.True(t, FssaiAdminMayTransition(models.FssaiInProgress, models.FssaiFiled))
	require.True(t, FssaiAdminMayTransition(models.FssaiFiled, models.FssaiIssued))

	require.False(t, FssaiAdminMayTransition(models.FssaiSubmitted, models.FssaiIssued),
		"a request cannot be issued without being filed first")
	require.False(t, FssaiAdminMayTransition(models.FssaiIssued, models.FssaiInProgress),
		"issued is terminal")

	require.Equal(t, "applicationRef", FssaiTransitionRequires(models.FssaiFiled))
	require.Equal(t, "registrationNo", FssaiTransitionRequires(models.FssaiIssued))
	require.Equal(t, "rejectedReason", FssaiTransitionRequires(models.FssaiRejected))
}

// The tracker shows the live request; once closed it still shows the last one
// so the registration number we obtained stays visible.
func TestLatestFssaiRequestFor(t *testing.T) {
	fssaiPolicy(t)
	db := setupFssaiDB(t)
	chef := fssaiChef()
	require.Nil(t, mustLatest(t, db, chef.ID))

	r := models.FssaiRequest{TermYears: 1}
	require.NoError(t, CreateFssaiRequest(db, chef, &r))
	require.Equal(t, r.ID, mustLatest(t, db, chef.ID).ID)

	require.NoError(t, db.Model(&models.FssaiRequest{}).Where("id = ?", r.ID).
		Updates(map[string]any{"status": models.FssaiIssued, "registration_no": "12345"}).Error)
	latest := mustLatest(t, db, chef.ID)
	require.Equal(t, "12345", latest.RegistrationNo)

	open, err := OpenFssaiRequestFor(db, chef.ID)
	require.NoError(t, err)
	require.Nil(t, open, "an issued request is finished; the chef may ask again years later")
}

func mustLatest(t *testing.T, db *gorm.DB, chefID uuid.UUID) *models.FssaiRequest {
	t.Helper()
	r, err := LatestFssaiRequestFor(db, chefID)
	require.NoError(t, err)
	return r
}
