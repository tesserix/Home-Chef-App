package handlers

// admin_fssai.go — the operator side of the FSSAI filing request, consumed by
// tesserix-home through the existing /admin/* proxy.
//
// Staff see the queue, open the chef's documents, and move the request as they
// actually do the work. Every state after `submitted` is moved from here — the
// chef's tracker is a mirror of what an admin has done, never a guess.

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type AdminFssaiHandler struct{}

func NewAdminFssaiHandler() *AdminFssaiHandler { return &AdminFssaiHandler{} }

// ListFssaiRequests is the admin queue.
//
// Defaults to PAID, open requests — the actual work. Finished registrations
// would bury the two that need doing today, and an unpaid draft is a chef
// halfway through a form, not something staff can act on. `?status=` narrows
// further; `?all=true` shows everything including drafts.
// GET /admin/fssai/requests
func (h *AdminFssaiHandler) ListFssaiRequests(c *gin.Context) {
	q := database.DB.Model(&models.FssaiRequest{}).Preload("Documents")

	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	} else if c.Query("all") != "true" {
		q = q.Where("status NOT IN ?", []string{
			models.FssaiAwaitingPayment,
			models.FssaiIssued, models.FssaiRejected, models.FssaiRefunded,
		})
	}

	var total int64
	q.Count(&total)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}

	var rows []models.FssaiRequest
	if err := q.Order("created_at ASC"). // oldest first: it is a work list
						Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load requests"})
		return
	}

	out := make([]gin.H, 0, len(rows))
	for i := range rows {
		out = append(out, adminFssaiRow(c, &rows[i]))
	}
	c.JSON(http.StatusOK, gin.H{
		"requests": out, "total": total, "page": page, "limit": limit,
		// So the admin UI can render the pricing it is verifying against without
		// hardcoding a second copy of the figures.
		"pricing": services.QuoteFssaiFiling(1),
	})
}

// GetFssaiRequest is one request in full, for the admin detail view.
// GET /admin/fssai/requests/:id
func (h *AdminFssaiHandler) GetFssaiRequest(c *gin.Context) {
	var row models.FssaiRequest
	if err := database.DB.Preload("Documents").
		Where("id = ?", c.Param("id")).First(&row).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"request": adminFssaiRow(c, &row)})
}

type updateFssaiRequest struct {
	Status         string `json:"status"`
	ApplicationRef string `json:"applicationRef"`
	RegistrationNo string `json:"registrationNo"`
	RejectedReason string `json:"rejectedReason"`
	AdminNotes     string `json:"adminNotes"`
	// What we need from the chef. Shown to them verbatim, so it is written for
	// a chef and not for the queue.
	InfoRequested string `json:"infoRequested"`
}

// UpdateFssaiRequest moves a request and records what came back from FoSCoS.
//
// The transition table is enforced here rather than trusted from the client:
// this is the only writer of these states, and a request marked `issued`
// without a registration number tells a chef their licence exists when nobody
// has recorded one.
// PATCH /admin/fssai/requests/:id
func (h *AdminFssaiHandler) UpdateFssaiRequest(c *gin.Context) {
	var req updateFssaiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var row models.FssaiRequest
	if err := database.DB.Preload("Documents").
		Where("id = ?", c.Param("id")).First(&row).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	now := time.Now().UTC()
	updates := map[string]any{"updated_at": now}
	if s := strings.TrimSpace(req.ApplicationRef); s != "" {
		updates["application_ref"] = s
		row.ApplicationRef = s
	}
	if s := strings.TrimSpace(req.RegistrationNo); s != "" {
		updates["registration_no"] = s
		row.RegistrationNo = s
	}
	if s := strings.TrimSpace(req.RejectedReason); s != "" {
		updates["rejected_reason"] = s
		row.RejectedReason = s
	}
	if s := strings.TrimSpace(req.InfoRequested); s != "" {
		updates["info_requested"] = s
		updates["info_requested_at"] = now
		row.InfoRequested = s
	}
	if req.AdminNotes != "" {
		updates["admin_notes"] = req.AdminNotes
	}

	if s := strings.TrimSpace(req.Status); s != "" && s != row.Status {
		if !services.FssaiAdminMayTransition(row.Status, s) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Cannot move a " + row.Status + " request to " + s,
			})
			return
		}
		// The field that makes the new status meaningful must be present — either
		// supplied now or already on the row.
		switch services.FssaiTransitionRequires(s) {
		case "applicationRef":
			if row.ApplicationRef == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "An FoSCoS application reference is required to mark a request filed",
				})
				return
			}
		case "registrationNo":
			if row.RegistrationNo == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "A registration number is required to mark a request issued",
				})
				return
			}
		case "rejectedReason":
			if row.RejectedReason == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "A reason is required to reject a request — the chef is shown it",
				})
				return
			}
		case "infoRequested":
			if row.InfoRequested == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Say what you need from the chef — they are shown it, and " +
						"a request parked without it is one they cannot act on",
				})
				return
			}
		}
		updates["status"] = s
		switch s {
		case models.FssaiFiled:
			updates["filed_at"] = now
		case models.FssaiIssued:
			updates["issued_at"] = now
		}
	}

	statusChanged := updates["status"] != nil
	if err := database.DB.Model(&models.FssaiRequest{}).
		Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the request"})
		return
	}
	_ = database.DB.Preload("Documents").First(&row, "id = ?", row.ID).Error

	// After the write, so a consumer reading the row back sees the new status.
	// Both are best-effort: the transition an admin made and a chef will see
	// must not be undone by a broker or a push service being down.
	if statusChanged {
		services.PublishFssaiRequestEvent(&row)
		services.NotifyFssaiRequestStatus(database.DB, &row)
	}
	c.JSON(http.StatusOK, gin.H{"request": adminFssaiRow(c, &row)})
}

// UploadFssaiLicense attaches the issued certificate to a request.
//
// The registration number alone is not the licence — this is the document the
// chef produces when an inspector asks. Stored in the private bucket like every
// other document here and handed out only as a short-lived signed URL.
// POST /admin/fssai/requests/:id/license  (multipart: file)
func (h *AdminFssaiHandler) UploadFssaiLicense(c *gin.Context) {
	var row models.FssaiRequest
	if err := database.DB.Preload("Documents").
		Where("id = ?", c.Param("id")).First(&row).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}
	if !models.FssaiPaid(row.Status) && row.Status != models.FssaiIssued {
		c.JSON(http.StatusConflict, gin.H{
			"error": "A certificate can only be attached to a request that is being worked",
		})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum 10 MB."})
		return
	}
	// Sniffed, not declared — same rule as the chef's uploads.
	contentType, serr := sniffContentType(file)
	if serr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read the file"})
		return
	}
	if contentType != "application/pdf" && !services.IsImageContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Allowed: PDF, JPEG, PNG or WebP."})
		return
	}

	folder := fmt.Sprintf("chefs/%s/fssai/%s/license", row.ChefID.String(), row.ID.String())
	objectPath, uerr := services.UploadPrivateFile(
		c.Request.Context(), folder, fssaiFileExtension(contentType), file, contentType)
	if uerr != nil {
		log.Printf("FSSAI request %s: licence upload failed: %v", row.ID, uerr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload the certificate"})
		return
	}

	now := time.Now().UTC()
	if err := database.DB.Model(&models.FssaiRequest{}).Where("id = ?", row.ID).
		Updates(map[string]any{
			"license_file_url":  objectPath,
			"license_file_name": sanitiseFileLabel(header.Filename),
			"updated_at":        now,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record the certificate"})
		return
	}
	_ = database.DB.Preload("Documents").First(&row, "id = ?", row.ID).Error
	c.JSON(http.StatusOK, gin.H{"request": adminFssaiRow(c, &row)})
}

// fssaiLicenseView mints a short-lived link to the issued certificate, or "" if
// there is none. Never a durable URL: it carries the chef's name and address.
func fssaiLicenseView(c *gin.Context, r *models.FssaiRequest) string {
	if r.LicenseFileURL == "" {
		return ""
	}
	url, err := services.GenerateSignedURL(c.Request.Context(), r.LicenseFileURL, 15*time.Minute)
	if err != nil {
		return ""
	}
	return url
}

// adminFssaiRow is the operator shape: everything the chef sees plus the notes
// and the money trail staff need to verify a payment against.
func adminFssaiRow(c *gin.Context, r *models.FssaiRequest) gin.H {
	// Signed, short-lived, same as every other admin document surface — the
	// objects are in the private bucket and staff get a link that expires.
	docs := fssaiDocumentViews(c, r.Documents)
	return gin.H{
		"id": r.ID, "chefId": r.ChefID, "status": r.Status,
		"kitchenName": r.KitchenName, "applicantName": r.ApplicantName,
		"contactPhone": r.ContactPhone, "contactEmail": r.ContactEmail,
		"addressLine1": r.AddressLine1, "addressLine2": r.AddressLine2,
		"city": r.City, "state": r.State, "postalCode": r.PostalCode,
		"termYears": r.TermYears,
		"feeAmount": r.FeeAmount, "feeTax": r.FeeTax, "feeTotal": r.FeeTotal,
		"currency": r.Currency, "paymentRef": r.PaymentRef, "paidAt": r.PaidAt,
		"applicationRef": r.ApplicationRef, "registrationNo": r.RegistrationNo,
		"rejectedReason": r.RejectedReason, "adminNotes": r.AdminNotes,
		"submittedAt": r.SubmittedAt, "filedAt": r.FiledAt, "issuedAt": r.IssuedAt,
		"documents": docs, "needsDocuments": r.NeedsDocuments(),
		"mode":      models.NormalizeMode(r.Mode),
		"createdAt": r.CreatedAt, "updatedAt": r.UpdatedAt,
	}
}
