package handlers

// fssai.go — the chef's side of the FSSAI filing request.
//
// A chef asks us to obtain their FSSAI registration: they pay, upload their
// documents, and track it. Payment comes first, so every endpoint here is
// written to survive a chef who pays and stops.
//
// See .planning/FSSAI-IN-APP-REQUEST-DESIGN.md.

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type FssaiHandler struct{}

func NewFssaiHandler() *FssaiHandler { return &FssaiHandler{} }

// chefFor resolves the authenticated chef, answering 404 when there is none.
func (h *FssaiHandler) chefFor(c *gin.Context) (*models.ChefProfile, bool) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return nil, false
	}
	return &chef, true
}

// GetFssaiQuote prices the filing before a chef commits to anything, and says
// whether the service is open at all.
// GET /chef/fssai/quote?years=1
func (h *FssaiHandler) GetFssaiQuote(c *gin.Context) {
	years := 1
	if n, err := strconv.Atoi(c.Query("years")); err == nil && n > 0 {
		years = n
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled": services.FssaiFilingEnabled(),
		"quote":   services.QuoteFssaiFiling(years),
		// Every term, so the app can render the ladder without a call per year.
		"terms": fssaiTermLadder(),
		// Stated by the SERVER so the disclosure and the rule it describes can
		// never drift apart: the app must show this before taking the payment.
		"nonRefundable": true,
		"nonRefundableNotice": "Once you pay, this fee is non-refundable and the " +
			"request cannot be cancelled. You can decide not to go ahead at any " +
			"point before paying.",
	})
}

func fssaiTermLadder() []services.FssaiQuote {
	out := make([]services.FssaiQuote, 0, services.FssaiTermYearsMax)
	for y := services.FssaiTermYearsMin; y <= services.FssaiTermYearsMax; y++ {
		out = append(out, services.QuoteFssaiFiling(y))
	}
	return out
}

type createFssaiRequest struct {
	KitchenName   string `json:"kitchenName" binding:"required"`
	ApplicantName string `json:"applicantName" binding:"required"`
	ContactPhone  string `json:"contactPhone" binding:"required"`
	ContactEmail  string `json:"contactEmail" binding:"required,email"`
	AddressLine1  string `json:"addressLine1" binding:"required"`
	AddressLine2  string `json:"addressLine2"`
	City          string `json:"city" binding:"required"`
	State         string `json:"state" binding:"required"`
	PostalCode    string `json:"postalCode" binding:"required"`
	TermYears     int    `json:"termYears" binding:"required"`
}

// CreateFssaiRequest mints the request and the Cashfree order that pays for it.
// POST /chef/fssai/requests
func (h *FssaiHandler) CreateFssaiRequest(c *gin.Context) {
	if !services.FssaiFilingEnabled() {
		c.JSON(http.StatusForbidden, gin.H{"error": "FSSAI filing is not available right now"})
		return
	}
	chef, ok := h.chefFor(c)
	if !ok {
		return
	}
	var req createFssaiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.TermYears < services.FssaiTermYearsMin || req.TermYears > services.FssaiTermYearsMax {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Choose a term between 1 and 5 years"})
		return
	}

	row := models.FssaiRequest{
		KitchenName:   strings.TrimSpace(req.KitchenName),
		ApplicantName: strings.TrimSpace(req.ApplicantName),
		ContactPhone:  strings.TrimSpace(req.ContactPhone),
		ContactEmail:  strings.TrimSpace(req.ContactEmail),
		AddressLine1:  strings.TrimSpace(req.AddressLine1),
		AddressLine2:  strings.TrimSpace(req.AddressLine2),
		City:          strings.TrimSpace(req.City),
		State:         strings.TrimSpace(req.State),
		PostalCode:    strings.TrimSpace(req.PostalCode),
		TermYears:     req.TermYears,
	}
	if err := services.CreateFssaiRequest(database.DB, chef, &row); err != nil {
		if err == services.ErrFssaiRequestOpen {
			// 409, not 400: the chef did nothing wrong, they simply already have
			// one — and the app routes them to the tracker instead.
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		log.Printf("FSSAI request create failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create the request"})
		return
	}

	var payer models.User
	_ = database.DB.First(&payer, "id = ?", chef.UserID).Error
	cfOrder, cerr := services.CreateCashfreeCharge(
		chef.Mode, row.ID, row.FeeTotal, row.Currency,
		services.CashfreeCustomerFor(&payer),
		map[string]string{
			"type": "fssai_filing", "request_id": row.ID.String(), "chef_id": chef.ID.String(),
		},
		"Fe3dr FSSAI filing", "fssai",
	)
	if cerr != nil {
		// The row stays in awaiting_payment with no gateway order. It is visible
		// to admin and harmless: nothing was charged, and the chef can retry.
		log.Printf("FSSAI request %s: cashfree order failed: %v", row.ID, cerr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start the payment"})
		return
	}
	database.DB.Model(&row).Update("gateway_order", cfOrder.OrderID)

	c.JSON(http.StatusOK, gin.H{
		"request":                  fssaiRequestResponse(c, &row),
		"cashfreeOrderId":          cfOrder.OrderID,
		"cashfreePaymentSessionId": cfOrder.PaymentSessionID,
		// The app cannot infer sandbox vs production — the hosts differ and the
		// session id looks alike — so the server states it.
		"cashfreeEnv": services.CashfreeEnvLabel(chef.Mode),
	})
}

// ConfirmFssaiPayment verifies the capture with Cashfree and advances the
// request. The client is never trusted for this: it hands us nothing signed, so
// the gateway fetch IS the verification.
// POST /chef/fssai/requests/:id/confirm
func (h *FssaiHandler) ConfirmFssaiPayment(c *gin.Context) {
	chef, ok := h.chefFor(c)
	if !ok {
		return
	}
	var row models.FssaiRequest
	if err := database.DB.Preload("Documents").
		Where("id = ? AND chef_id = ?", c.Param("id"), chef.ID).First(&row).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}
	// Already past payment — idempotent success, so a retried confirm or a
	// double tap does not read as an error to the chef.
	if row.Status != models.FssaiAwaitingPayment {
		c.JSON(http.StatusOK, gin.H{"request": fssaiRequestResponse(c, &row)})
		return
	}

	pay, err := services.VerifyCashfreeCharge(chef.Mode, row.GatewayOrder, row.FeeTotal)
	if err != nil {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Payment not confirmed yet"})
		return
	}
	if err := services.MarkFssaiPaid(database.DB, &row, pay.CFPaymentID.String()); err != nil {
		log.Printf("FSSAI request %s: mark paid failed: %v", row.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record the payment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"request": fssaiRequestResponse(c, &row)})
}

type attachFssaiDocRequest struct {
	Kind     string `json:"kind" binding:"required"`
	FileURL  string `json:"fileUrl" binding:"required"`
	FileName string `json:"fileName"`
}

// AttachFssaiDocument records an uploaded document and, once the request has
// everything FSSAI requires, submits it to onboarding.
// POST /chef/fssai/requests/:id/documents
func (h *FssaiHandler) AttachFssaiDocument(c *gin.Context) {
	chef, ok := h.chefFor(c)
	if !ok {
		return
	}
	var req attachFssaiDocRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	switch req.Kind {
	case models.FssaiDocPhoto, models.FssaiDocIdentity, models.FssaiDocAddressProof:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown document kind"})
		return
	}

	var row models.FssaiRequest
	if err := database.DB.Preload("Documents").
		Where("id = ? AND chef_id = ?", c.Param("id"), chef.ID).First(&row).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}
	if err := services.AttachFssaiDocument(
		database.DB, &row, req.Kind, req.FileURL, req.FileName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"request": fssaiRequestResponse(c, &row)})
}

// CancelFssaiRequest lets a chef back out of a request they have not paid for.
//
// The service is optional and a chef may change their mind: they can open the
// form, see the price, and walk away — then come back weeks later and start
// again. Without this, an abandoned unpaid row would sit in their tracker
// forever and block a fresh request.
//
// A PAID request is not cancellable here: money has moved, so backing out is a
// refund an admin has to make, not something the app does silently.
// DELETE /chef/fssai/requests/:id
func (h *FssaiHandler) CancelFssaiRequest(c *gin.Context) {
	chef, ok := h.chefFor(c)
	if !ok {
		return
	}
	var row models.FssaiRequest
	if err := database.DB.Where("id = ? AND chef_id = ?", c.Param("id"), chef.ID).
		First(&row).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}
	if row.Status != models.FssaiAwaitingPayment {
		// Owner policy: once paid, the filing fee is non-refundable and the
		// request cannot be withdrawn. The chef is told this before they pay
		// (see the quote endpoint's `nonRefundable`), so it is not a surprise
		// discovered at the point they try to back out.
		c.JSON(http.StatusConflict, gin.H{
			"error": "This request has been paid for and cannot be cancelled. " +
				"We have started work on your application.",
		})
		return
	}
	if err := services.DiscardFssaiRequest(database.DB, &row); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel the request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cancelled": true})
}

// UploadFssaiDocument takes the file itself, stores it, and attaches it in one
// call — the app should not have to upload somewhere and then separately tell
// us about it, which leaves an orphaned file whenever the second call fails.
//
// Same validation as every other chef upload: size capped and the bytes sniffed,
// so a spoofed Content-Type cannot slip a payload through.
// POST /chef/fssai/requests/:id/upload  (multipart: file, kind)
func (h *FssaiHandler) UploadFssaiDocument(c *gin.Context) {
	chef, ok := h.chefFor(c)
	if !ok {
		return
	}
	kind := c.PostForm("kind")
	switch kind {
	case models.FssaiDocPhoto, models.FssaiDocIdentity, models.FssaiDocAddressProof:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown document kind"})
		return
	}

	var row models.FssaiRequest
	if err := database.DB.Preload("Documents").
		Where("id = ? AND chef_id = ?", c.Param("id"), chef.ID).First(&row).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum 5 MB."})
		return
	}

	// Sniff the BYTES for every accepted type, PDFs included. Trusting the
	// declared Content-Type for one branch is the whole bypass: a caller simply
	// declares application/pdf and uploads whatever they like.
	sniffed, serr := sniffContentType(file)
	if serr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read the file"})
		return
	}
	// A PDF is only acceptable as an address proof — a utility bill usually is
	// one. A photo or an ID has to be an image we can actually look at.
	isPDF := sniffed == "application/pdf"
	switch {
	case isPDF && kind == models.FssaiDocAddressProof:
	case services.IsImageContentType(sniffed):
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Allowed: JPEG, PNG or WebP — or a PDF for an address proof.",
		})
		return
	}
	// Store the SNIFFED type, never the declared one, so the object cannot be
	// served back as something it is not.
	contentType = sniffed

	// PRIVATE bucket. These are Aadhaar and PAN images: the public bucket hands
	// out a permanent unauthenticated URL, which for identity documents is an
	// exposure, not a convenience. What is stored is the object path; every
	// surface that shows a document mints a short-lived signed URL for it.
	//
	// The stored object name is a UUID with an extension derived from the
	// SNIFFED type — the uploader's filename is never used to build a path, so
	// it cannot traverse out of the folder or choose its own extension.
	folder := fmt.Sprintf("chefs/%s/fssai/%s", chef.ID.String(), row.ID.String())
	objectPath, uerr := services.UploadPrivateFile(
		c.Request.Context(), folder, fssaiFileExtension(contentType), file, contentType)
	if uerr != nil {
		log.Printf("FSSAI request %s: upload failed: %v", row.ID, uerr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload the file"})
		return
	}

	// The chef's own filename is kept only as a LABEL, sanitised, never as a path.
	if err := services.AttachFssaiDocument(
		database.DB, &row, kind, objectPath, sanitiseFileLabel(header.Filename)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"request": fssaiRequestResponse(c, &row)})
}

// fssaiFileExtension maps a VERIFIED content type to the extension the stored
// object gets. Derived from the sniffed bytes rather than the uploader's
// filename, so the extension is never attacker-chosen.
func fssaiFileExtension(contentType string) string {
	switch contentType {
	case "image/png":
		return "f.png"
	case "image/webp":
		return "f.webp"
	case "application/pdf":
		return "f.pdf"
	default:
		return "f.jpg"
	}
}

// sanitiseFileLabel keeps the chef's filename only as something to read in the
// admin queue: no path separators, no traversal, length-capped. It never
// reaches a filesystem or an object path.
func sanitiseFileLabel(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "..", "")
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, name)
	if len(name) > 80 {
		name = name[:80]
	}
	if name == "" || name == "." {
		return "document"
	}
	return name
}

// fssaiDocumentViews turns stored object paths into short-lived signed URLs.
// Nothing that renders a document ever gets a durable link: the objects live in
// the private bucket and a link that outlives the screen is a copy of someone's
// Aadhaar loose in a log or a chat.
func fssaiDocumentViews(c *gin.Context, docs []models.FssaiRequestDocument) []gin.H {
	out := make([]gin.H, 0, len(docs))
	for _, d := range docs {
		row := gin.H{"kind": d.Kind, "fileName": d.FileName}
		if url, err := services.GenerateSignedURL(
			c.Request.Context(), d.FileURL, 15*time.Minute); err == nil {
			row["fileUrl"] = url
		}
		out = append(out, row)
	}
	return out
}

// GetFssaiRequest is the chef's tracker: their live request, or the most recent
// closed one so the registration number we obtained stays visible.
// GET /chef/fssai/request
func (h *FssaiHandler) GetFssaiRequest(c *gin.Context) {
	chef, ok := h.chefFor(c)
	if !ok {
		return
	}
	row, err := services.LatestFssaiRequestFor(database.DB, chef.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load the request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"request": fssaiRequestResponse(c, row),
		"enabled": services.FssaiFilingEnabled(),
	})
}

// fssaiRequestResponse is the chef-facing shape. AdminNotes are deliberately
// absent — internal working notes must never reach a chef's screen.
func fssaiRequestResponse(c *gin.Context, r *models.FssaiRequest) any {
	if r == nil {
		return nil
	}
	docs := fssaiDocumentViews(c, r.Documents)
	return gin.H{
		"id": r.ID, "status": r.Status, "termYears": r.TermYears,
		"kitchenName": r.KitchenName, "applicantName": r.ApplicantName,
		"feeAmount": r.FeeAmount, "feeTax": r.FeeTax, "feeTotal": r.FeeTotal,
		"currency": r.Currency, "paidAt": r.PaidAt,
		"applicationRef": r.ApplicationRef, "registrationNo": r.RegistrationNo,
		"rejectedReason": r.RejectedReason,
		"submittedAt":    r.SubmittedAt, "filedAt": r.FiledAt, "issuedAt": r.IssuedAt,
		"documents": docs,
		// What the app still needs from the chef, so the screen does not have to
		// re-derive FSSAI's document rules.
		"needsDocuments": r.NeedsDocuments(),
		"createdAt":      r.CreatedAt,
	}
}
