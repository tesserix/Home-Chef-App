package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// chef_profiles.business_name carries a unique index, so the doc-upload
// placeholder cannot reuse the empty string — two concurrent users would
// otherwise collide on `”` and the second upload would fail. We embed the
// userID so each placeholder is unique-per-user; updateOnboarding overwrites
// it with the real kitchen name on submit.
const draftBusinessNamePrefix = "__draft__"

func draftBusinessName(userID uuid.UUID) string {
	return draftBusinessNamePrefix + userID.String()
}

func isPlaceholderBusinessName(n string) bool {
	return strings.HasPrefix(n, draftBusinessNamePrefix)
}

// sniffContentType detects the real content type of an uploaded file from its
// first 512 bytes (http.DetectContentType) and rewinds the reader so the full
// file still uploads afterwards. Multipart files implement io.Seeker. This
// guards against a spoofed multipart Content-Type header (e.g. an executable or
// SVG sent as image/png) since the byte-level sniff can't be faked by the header.
func sniffContentType(file io.ReadSeeker) (string, error) {
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	ct := http.DetectContentType(buf[:n])
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return ct, nil
}

// getOrCreateChefProfile finds the chef profile for the user, creating a placeholder if needed.
// During onboarding, documents may be uploaded before the full profile is submitted.
func getOrCreateChefProfile(userID uuid.UUID) (*models.ChefProfile, error) {
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		chef = models.ChefProfile{
			UserID:       userID,
			BusinessName: draftBusinessName(userID),
			IsActive:     false,
		}
		if err := database.DB.Create(&chef).Error; err != nil {
			return nil, fmt.Errorf("failed to create placeholder chef profile: %w", err)
		}
		log.Printf("Auto-created placeholder chef profile %s for user %s", chef.ID, userID)
	}
	return &chef, nil
}

// UploadDocument handles file uploads for chef documents
// POST /chef/documents — multipart/form-data with fields: file, type
func (h *UploadHandler) UploadDocument(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	chef, err := getOrCreateChefProfile(userID)
	if err != nil {
		log.Printf("Failed to get/create chef profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize chef profile"})
		return
	}

	docType := models.DocumentType(c.PostForm("type"))
	if docType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document type is required"})
		return
	}

	// Optional expiry date — ISO-8601 string in the multipart field "expiryDate"
	// (e.g. "2026-07-10"). Omitting the field leaves ExpiryDate as nil.
	var expiryDate *time.Time
	if ed := c.PostForm("expiryDate"); ed != "" {
		parsed, err := time.Parse("2006-01-02", ed)
		if err != nil {
			// Also try RFC3339 for callers that send a full timestamp
			parsed, err = time.Parse(time.RFC3339, ed)
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiryDate format. Use YYYY-MM-DD or RFC3339."})
			return
		}
		expiryDate = &parsed
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	// Validate file size (5MB max)
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum 5 MB."})
		return
	}

	contentType := header.Header.Get("Content-Type")
	isPrivate := models.IsPrivateDoc(docType)
	isPhoto := models.IsPhotoDoc(docType)

	// Validate content type based on document category
	if isPhoto && !services.IsImageContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed: JPEG, PNG, WebP."})
		return
	}
	if !isPhoto && !services.IsDocContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed: JPEG, PNG, PDF."})
		return
	}

	// Also validate the real bytes so a spoofed Content-Type header can't slip
	// a disallowed payload through. Rewinds the reader so the full file uploads.
	sniffed, serr := sniffContentType(file)
	if serr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read file"})
		return
	}
	if isPhoto && !services.IsImageContentType(sniffed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File contents don't match an allowed image type."})
		return
	}
	if !isPhoto && !services.IsDocContentType(sniffed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File contents don't match an allowed file type."})
		return
	}

	// Upload to appropriate bucket with per-chef directory isolation:
	// chefs/{chefID}/{documentType}/{uuid}.{ext}
	folder := fmt.Sprintf("chefs/%s/%s", chef.ID.String(), string(docType))
	var fileURL, filePath, bucket string

	if isPrivate {
		bucket = config.AppConfig.GCSPrivateBucket
		filePath, err = services.UploadPrivateFile(c.Request.Context(), folder, header.Filename, file, contentType)
		if err != nil {
			log.Printf("Failed to upload private file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
			return
		}
	} else {
		bucket = config.AppConfig.GCSPublicBucket
		fileURL, err = services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
		if err != nil {
			log.Printf("Failed to upload public file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
			return
		}
		filePath = fileURL // For public files, path IS the URL
	}

	// Delete previous document of same type if exists
	database.DB.Where("chef_id = ? AND type = ?", chef.ID, docType).Delete(&models.ChefDocument{})

	// Save document record
	doc := models.ChefDocument{
		ChefID:      chef.ID,
		Type:        docType,
		FileName:    header.Filename,
		FilePath:    filePath,
		FileURL:     fileURL,
		Bucket:      bucket,
		ContentType: contentType,
		FileSize:    header.Size,
		Status:      models.DocStatusPending,
		ExpiryDate:  expiryDate,
	}

	if err := database.DB.Create(&doc).Error; err != nil {
		log.Printf("Failed to save document record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save document"})
		return
	}

	// If it's a profile image, also update the chef profile
	if docType == models.DocProfileImage {
		database.DB.Model(chef).Update("profile_image", fileURL)
	}

	// Create approval request for document verification (skip for profile images)
	if docType != models.DocProfileImage {
		approvalReq := models.ApprovalRequest{
			Type:          models.ApprovalDocumentVerification,
			Status:        models.ApprovalPending,
			Priority:      "normal",
			ChefID:        &chef.ID,
			SubmittedByID: userID,
			EntityType:    "chef_document",
			EntityID:      doc.ID,
			Title:         fmt.Sprintf("Document Verification: %s", string(docType)),
			Description:   fmt.Sprintf("Document uploaded for verification: %s (%s)", header.Filename, string(docType)),
			SubmittedData: fmt.Sprintf(`{"document_id":"%s","type":"%s","file_name":"%s"}`, doc.ID.String(), string(docType), header.Filename),
		}
		// Cancel any existing pending request for the same doc type+chef
		database.DB.Model(&models.ApprovalRequest{}).
			Where("chef_id = ? AND type = ? AND entity_id = ? AND status = ?", chef.ID, models.ApprovalDocumentVerification, doc.ID, models.ApprovalPending).
			Update("status", models.ApprovalCancelled)
		database.DB.Create(&approvalReq)

		if err := services.EnqueueEvent(database.DB, services.SubjectApprovalCreated, "approval.created", userID, map[string]interface{}{
			"approval_id": approvalReq.ID.String(),
			"type":        string(approvalReq.Type),
			"chef_id":     chef.ID.String(),
			"title":       approvalReq.Title,
		}); err != nil {
			log.Printf("failed to enqueue approval.created event: %v", err)
		}
	}

	c.JSON(http.StatusOK, doc.ToResponse())
}

// UploadProfileImage handles profile picture uploads
// POST /chef/profile-image — multipart/form-data with field: file
func (h *UploadHandler) UploadProfileImage(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	chef, err := getOrCreateChefProfile(userID)
	if err != nil {
		log.Printf("Failed to get/create chef profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize chef profile"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum 5 MB."})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !services.IsImageContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed: JPEG, PNG, WebP."})
		return
	}
	sniffed, serr := sniffContentType(file)
	if serr != nil || !services.IsImageContentType(sniffed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File contents don't match an allowed image type."})
		return
	}

	// Per-chef directory: chefs/{chefID}/avatar/{uuid}.{ext}
	folder := fmt.Sprintf("chefs/%s/avatar", chef.ID.String())
	fileURL, err := services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
	if err != nil {
		log.Printf("Failed to upload profile image: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	database.DB.Model(chef).Update("profile_image", fileURL)

	c.JSON(http.StatusOK, gin.H{"url": fileURL})
}

// UploadBannerImage handles banner/cover image uploads
// POST /chef/banner-image — multipart/form-data with field: file
func (h *UploadHandler) UploadBannerImage(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	chef, err := getOrCreateChefProfile(userID)
	if err != nil {
		log.Printf("Failed to get/create chef profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize chef profile"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum 5 MB."})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !services.IsImageContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed: JPEG, PNG, WebP."})
		return
	}
	sniffed, serr := sniffContentType(file)
	if serr != nil || !services.IsImageContentType(sniffed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File contents don't match an allowed image type."})
		return
	}

	folder := fmt.Sprintf("chefs/%s/banner", chef.ID.String())
	fileURL, err := services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
	if err != nil {
		log.Printf("Failed to upload banner image: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	database.DB.Model(chef).Update("banner_image", fileURL)

	c.JSON(http.StatusOK, gin.H{"url": fileURL})
}

// UploadKitchenPhoto uploads a kitchen photo (max 5 photos per chef)
// POST /chef/kitchen-photos — multipart/form-data with field: file
func (h *UploadHandler) UploadKitchenPhoto(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	chef, err := getOrCreateChefProfile(userID)
	if err != nil {
		log.Printf("Failed to get/create chef profile: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize chef profile"})
		return
	}

	// Check current count
	if len(chef.KitchenPhotos) >= 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 5 kitchen photos allowed. Remove one before adding another."})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum 5 MB."})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !services.IsImageContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed: JPEG, PNG, WebP."})
		return
	}
	sniffed, serr := sniffContentType(file)
	if serr != nil || !services.IsImageContentType(sniffed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File contents don't match an allowed image type."})
		return
	}

	folder := fmt.Sprintf("chefs/%s/kitchen", chef.ID.String())
	fileURL, err := services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
	if err != nil {
		log.Printf("Failed to upload kitchen photo: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Append to kitchen_photos array
	updatedPhotos := append(chef.KitchenPhotos, fileURL)
	database.DB.Model(chef).Update("kitchen_photos", updatedPhotos)

	c.JSON(http.StatusOK, gin.H{"url": fileURL, "kitchenPhotos": updatedPhotos})
}

// DeleteKitchenPhoto removes a kitchen photo by URL
// DELETE /chef/kitchen-photos — JSON body: { "url": "..." }
func (h *UploadHandler) DeleteKitchenPhoto(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var req struct {
		URL string `json:"url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	// Filter out the photo
	updatedPhotos := make([]string, 0, len(chef.KitchenPhotos))
	found := false
	for _, photo := range chef.KitchenPhotos {
		if photo == req.URL {
			found = true
			continue
		}
		updatedPhotos = append(updatedPhotos, photo)
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	database.DB.Model(&chef).Update("kitchen_photos", updatedPhotos)

	c.JSON(http.StatusOK, gin.H{"kitchenPhotos": updatedPhotos})
}

// GetOnboardingStatus checks if the authenticated user has completed chef onboarding.
// Returns the full chef profile data so the frontend can hydrate the onboarding form.
// GET /chef/onboarding/status — accessible to any authenticated user (no RequireChef)
func (h *UploadHandler) GetOnboardingStatus(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).Preload("User").First(&chef).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":    "not_started",
			"completed": false,
		})
		return
	}

	// Determine onboarding step based on what data is filled
	step := 0
	if chef.BusinessName != "" && !isPlaceholderBusinessName(chef.BusinessName) {
		step = 2 // Kitchen details done
	}
	if chef.PrepTime != "" || chef.ServiceRadius > 0 {
		step = 3 // Operations done
	}

	// Check if documents are uploaded
	var docCount int64
	database.DB.Model(&models.ChefDocument{}).Where("chef_id = ?", chef.ID).Count(&docCount)
	if docCount > 0 {
		step = 4 // Documents done
	}

	// Check if there's a pending or rejected approval request
	var latestApproval models.ApprovalRequest
	hasApproval := false
	approvalStatus := ""
	approvalNotes := ""
	if err := database.DB.Where("chef_id = ? AND type = ?", chef.ID, models.ApprovalKitchenOnboarding).
		Order("created_at DESC").First(&latestApproval).Error; err == nil {
		hasApproval = true
		approvalStatus = string(latestApproval.Status)
		approvalNotes = latestApproval.AdminNotes
	}

	// Determine onboarding completion. The full submit flow always creates an
	// ApprovalRequest (transactionally, alongside the chef row). If no approval
	// exists, the chef row is either a doc-upload stub or a half-saved wizard
	// record — not a completed application, so keep the user on /onboarding.
	completed := false
	onboardingStatus := "not_started"
	switch {
	case chef.IsVerified:
		completed = true
		onboardingStatus = "verified"
	case chef.BusinessName == "" || isPlaceholderBusinessName(chef.BusinessName):
		// Stub profile from document upload or partial wizard save.
		onboardingStatus = "in_progress"
	case hasApproval && (approvalStatus == "rejected" || approvalStatus == "info_requested"):
		onboardingStatus = approvalStatus
	case hasApproval:
		completed = true
		onboardingStatus = "pending_review"
		if approvalStatus == "approved" {
			onboardingStatus = "submitted"
		}
	default:
		// Business name set but no approval row — means the submit tx failed.
		// Treat as in_progress so the user re-submits rather than silently hanging.
		onboardingStatus = "in_progress"
	}

	// Hide the per-user draft sentinel from the wire — the wizard's hydration
	// path would otherwise pre-fill the kitchen-name input with `__draft__<uuid>`.
	wireBusinessName := chef.BusinessName
	if isPlaceholderBusinessName(wireBusinessName) {
		wireBusinessName = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         onboardingStatus,
		"completed":      completed,
		"step":           step,
		"chefId":         chef.ID,
		"approvalStatus": approvalStatus,
		"adminNotes":     approvalNotes,
		"profile": gin.H{
			"businessName":   wireBusinessName,
			"description":    chef.Description,
			"cuisines":       chef.Cuisines,
			"specialties":    chef.Specialties,
			"profileImage":   chef.ProfileImage,
			"bannerImage":    chef.BannerImage,
			"kitchenPhotos":  chef.KitchenPhotos,
			"prepTime":       chef.PrepTime,
			"serviceRadius":  chef.ServiceRadius,
			"minimumOrder":   chef.MinimumOrder,
			"deliveryRadius": chef.DeliveryRadius,
			"addressLine1":   chef.AddressLine1,
			"addressLine2":   chef.AddressLine2,
			"city":           chef.City,
			"state":          chef.State,
			"postalCode":     chef.PostalCode,
			"latitude":       chef.Latitude,
			"longitude":      chef.Longitude,
			"fullName":       chef.User.FirstName + " " + chef.User.LastName,
			"email":          chef.User.Email,
			"phone":          chef.User.Phone,
		},
	})
}

// Onboarding handles the chef onboarding form submission (JSON only — files uploaded separately)
// POST /chef/onboarding
// normalizeKitchenType enforces the home-chefs-only rule at onboarding. Fe3dr
// does not onboard commercial vendors, so any kitchen type other than home is
// rejected outright; a blank value (older clients that don't send the field)
// defaults to home. Returns (normalized, ok) — ok=false means "reject this".
func normalizeKitchenType(kt string) (string, bool) {
	if kt == "" || kt == models.KitchenTypeHome {
		return models.KitchenTypeHome, true
	}
	return "", false
}

const notHomeKitchenMsg = "Fe3dr is for individual home chefs only — cloud kitchens, shared or commercial kitchens cannot be onboarded."

func (h *UploadHandler) Onboarding(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	log.Printf("[onboarding] POST /chef/onboarding user=%s", userID)

	// Check if profile already exists (may have been auto-created by upload handler
	// via document upload, or by a prior partial-save from the wizard's Step 3 hand-off).
	var existing models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&existing).Error; err == nil {
		// Block re-submission for chefs whose application is already in flight or
		// approved. Without this guard the wizard's per-step `ensureProfile()` POST
		// silently rewrote the live profile (business name, cuisines, address,
		// schedules) and bumped the user back to a fresh pending approval —
		// effectively wiping their onboarded kitchen on every Step-2 → Step-3 nav.
		if existing.IsVerified {
			c.JSON(http.StatusConflict, gin.H{
				"error":  "Your kitchen is already verified. Use the dashboard to update kitchen details.",
				"status": "verified",
			})
			return
		}
		var latest models.ApprovalRequest
		if err := database.DB.Where("chef_id = ? AND type = ?", existing.ID, models.ApprovalKitchenOnboarding).
			Order("created_at DESC").First(&latest).Error; err == nil {
			switch latest.Status {
			case models.ApprovalPending, models.ApprovalApproved:
				c.JSON(http.StatusConflict, gin.H{
					"error":  "Your kitchen application is already under review. Please wait for the admin's response before re-submitting.",
					"status": string(latest.Status),
				})
				return
			}
		}
		h.updateOnboarding(c, &existing)
		return
	}

	var req OnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[onboarding] bind failed user=%s err=%v", userID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Home chefs only — reject commercial kitchen types before doing any work.
	kitchenType, okKT := normalizeKitchenType(req.KitchenType)
	if !okKT {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": notHomeKitchenMsg, "field": "kitchenType"})
		return
	}

	// Validate uniqueness before opening the transaction so clashes return 409 cleanly.
	if req.BusinessName != "" {
		var existingByName models.ChefProfile
		if err := database.DB.Where("business_name = ? AND user_id != ?", req.BusinessName, userID).First(&existingByName).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "A kitchen with this name already exists. Please choose a different name.", "field": "businessName"})
			return
		}
	}
	if req.Phone != "" {
		var existingByPhone models.User
		if err := database.DB.Where("phone = ? AND id != ?", req.Phone, userID).First(&existingByPhone).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "This phone number is already registered with another account.", "field": "phone"})
			return
		}
	}

	chef := models.ChefProfile{
		UserID:         userID,
		BusinessName:   req.BusinessName,
		Description:    req.Description,
		Cuisines:       pq.StringArray(req.Cuisines),
		Specialties:    pq.StringArray(req.Specialties),
		PrepTime:       req.PrepTime,
		MinimumOrder:   req.MinimumOrder,
		DeliveryRadius: req.ServiceRadius,
		ServiceRadius:  req.ServiceRadius,
		AddressLine1:   req.KitchenAddress.Line1,
		AddressLine2:   req.KitchenAddress.Line2,
		City:           req.KitchenAddress.City,
		State:          req.KitchenAddress.State,
		PostalCode:     req.KitchenAddress.PostalCode,
		// Persist the regulatory IDs as structured columns. Previously only
		// captured in the approval submittedData JSON blob (audit-only),
		// which left admin queries + Wave 3 invoicing without a queryable
		// FSSAI number to print on customer invoices.
		PanNumber:          req.PanNumber,
		FSSAILicenseNumber: req.FSSAINumber,
		GSTIN:              req.GSTIN,
		KitchenType:        kitchenType,
		IsActive:           true,
		AcceptingOrders:    false,
	}

	approvalReq := models.ApprovalRequest{
		Type:          models.ApprovalKitchenOnboarding,
		Status:        models.ApprovalPending,
		Priority:      "high",
		SubmittedByID: userID,
		EntityType:    "chef_profile",
	}

	// Single transaction — profile, user role, schedules, categories, approval row
	// must all succeed together, or none of them persist. Prior code silently
	// swallowed errors on approval_request.Create which let the user see a 200
	// response while the admin never saw the submission.
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&chef).Error; err != nil {
			return fmt.Errorf("create chef profile: %w", err)
		}

		if err := tx.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
			"role":       models.RoleChef,
			"first_name": req.FullName,
			"phone":      req.Phone,
		}).Error; err != nil {
			return fmt.Errorf("update user role: %w", err)
		}

		if err := h.createSchedulesTx(tx, &chef, req.OperatingHours); err != nil {
			return fmt.Errorf("create schedules: %w", err)
		}

		if err := h.seedDefaultCategoriesTx(tx, chef.ID); err != nil {
			return fmt.Errorf("seed categories: %w", err)
		}

		submittedData, _ := json.Marshal(map[string]interface{}{
			"businessName": req.BusinessName,
			"fullName":     req.FullName,
			"phone":        maskPhone(req.Phone),
			"city":         req.KitchenAddress.City,
			"cuisines":     req.Cuisines,
			"kitchenType":  kitchenType,
			"panNumber":    maskPAN(req.PanNumber),
			"fssaiNumber":  maskID(req.FSSAINumber),
		})
		approvalReq.ChefID = &chef.ID
		approvalReq.EntityID = chef.ID
		approvalReq.Title = fmt.Sprintf("Kitchen Onboarding: %s", req.BusinessName)
		approvalReq.Description = fmt.Sprintf("%s submitted kitchen onboarding for review", req.FullName)
		approvalReq.SubmittedData = string(submittedData)

		if err := tx.Model(&models.ApprovalRequest{}).
			Where("chef_id = ? AND type = ? AND status = ?", chef.ID, models.ApprovalKitchenOnboarding, models.ApprovalPending).
			Update("status", models.ApprovalCancelled).Error; err != nil {
			return fmt.Errorf("cancel prior approvals: %w", err)
		}
		if err := tx.Create(&approvalReq).Error; err != nil {
			return fmt.Errorf("create approval request: %w", err)
		}
		return nil
	})
	if err != nil {
		log.Printf("[onboarding] tx failed user=%s err=%v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit onboarding. Please try again."})
		return
	}

	// Durable event publication via the transactional outbox.
	if err := services.EnqueueEvent(database.DB, services.SubjectApprovalCreated, "approval.created", userID, map[string]interface{}{
		"approval_id": approvalReq.ID.String(),
		"type":        string(approvalReq.Type),
		"chef_id":     chef.ID.String(),
		"title":       approvalReq.Title,
	}); err != nil {
		log.Printf("failed to enqueue approval.created event: %v", err)
	}

	log.Printf("[onboarding] created user=%s chef=%s approval=%s", userID, chef.ID, approvalReq.ID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Onboarding submitted successfully",
		"chefId":  chef.ID,
	})
}

func (h *UploadHandler) updateOnboarding(c *gin.Context, chef *models.ChefProfile) {
	log.Printf("[onboarding] update user=%s chef=%s", chef.UserID, chef.ID)

	var req OnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[onboarding] bind failed user=%s err=%v", chef.UserID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Home chefs only — reject commercial kitchen types before doing any work.
	kitchenType, okKT := normalizeKitchenType(req.KitchenType)
	if !okKT {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": notHomeKitchenMsg, "field": "kitchenType"})
		return
	}

	if req.BusinessName != "" && req.BusinessName != chef.BusinessName {
		var existingByName models.ChefProfile
		if err := database.DB.Where("business_name = ? AND id != ?", req.BusinessName, chef.ID).First(&existingByName).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "A kitchen with this name already exists. Please choose a different name.", "field": "businessName"})
			return
		}
	}
	if req.Phone != "" {
		var existingByPhone models.User
		if err := database.DB.Where("phone = ? AND id != ?", req.Phone, chef.UserID).First(&existingByPhone).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "This phone number is already registered with another account.", "field": "phone"})
			return
		}
	}

	chef.BusinessName = req.BusinessName
	chef.Description = req.Description
	chef.Cuisines = pq.StringArray(req.Cuisines)
	chef.Specialties = pq.StringArray(req.Specialties)
	chef.PrepTime = req.PrepTime
	chef.MinimumOrder = req.MinimumOrder
	chef.DeliveryRadius = req.ServiceRadius
	chef.ServiceRadius = req.ServiceRadius
	chef.AddressLine1 = req.KitchenAddress.Line1
	chef.AddressLine2 = req.KitchenAddress.Line2
	chef.City = req.KitchenAddress.City
	chef.State = req.KitchenAddress.State
	chef.PostalCode = req.KitchenAddress.PostalCode
	chef.KitchenType = kitchenType
	chef.IsActive = true

	approvalReq := models.ApprovalRequest{
		Type:          models.ApprovalKitchenOnboarding,
		Status:        models.ApprovalPending,
		Priority:      "high",
		ChefID:        &chef.ID,
		SubmittedByID: chef.UserID,
		EntityType:    "chef_profile",
		EntityID:      chef.ID,
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(chef).Error; err != nil {
			return fmt.Errorf("save chef: %w", err)
		}

		if err := tx.Model(&models.User{}).Where("id = ?", chef.UserID).Updates(map[string]interface{}{
			"role":       models.RoleChef,
			"first_name": req.FullName,
			"phone":      req.Phone,
		}).Error; err != nil {
			return fmt.Errorf("update user role: %w", err)
		}

		if err := tx.Where("chef_id = ?", chef.ID).Delete(&models.ChefSchedule{}).Error; err != nil {
			return fmt.Errorf("delete schedules: %w", err)
		}
		if err := h.createSchedulesTx(tx, chef, req.OperatingHours); err != nil {
			return fmt.Errorf("create schedules: %w", err)
		}

		submittedData, _ := json.Marshal(map[string]interface{}{
			"businessName": req.BusinessName,
			"fullName":     req.FullName,
			"phone":        maskPhone(req.Phone),
			"city":         req.KitchenAddress.City,
			"cuisines":     req.Cuisines,
			"kitchenType":  kitchenType,
			"panNumber":    maskPAN(req.PanNumber),
			"fssaiNumber":  maskID(req.FSSAINumber),
		})
		approvalReq.Title = fmt.Sprintf("Kitchen Onboarding: %s", req.BusinessName)
		approvalReq.Description = fmt.Sprintf("%s submitted kitchen onboarding for review", req.FullName)
		approvalReq.SubmittedData = string(submittedData)

		if err := tx.Model(&models.ApprovalRequest{}).
			Where("chef_id = ? AND type = ? AND status = ?", chef.ID, models.ApprovalKitchenOnboarding, models.ApprovalPending).
			Update("status", models.ApprovalCancelled).Error; err != nil {
			return fmt.Errorf("cancel prior approvals: %w", err)
		}
		if err := tx.Create(&approvalReq).Error; err != nil {
			return fmt.Errorf("create approval request: %w", err)
		}
		return nil
	})
	if err != nil {
		log.Printf("[onboarding] update tx failed user=%s err=%v", chef.UserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update onboarding. Please try again."})
		return
	}

	if err := services.EnqueueEvent(database.DB, services.SubjectApprovalCreated, "approval.created", chef.UserID, map[string]interface{}{
		"approval_id": approvalReq.ID.String(),
		"type":        string(approvalReq.Type),
		"chef_id":     chef.ID.String(),
		"title":       approvalReq.Title,
	}); err != nil {
		log.Printf("failed to enqueue approval.created event: %v", err)
	}

	log.Printf("[onboarding] updated user=%s chef=%s approval=%s", chef.UserID, chef.ID, approvalReq.ID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Onboarding updated successfully",
		"chefId":  chef.ID,
	})
}

func (h *UploadHandler) createSchedules(chef *models.ChefProfile, hours map[string]*DayHoursReq) {
	_ = h.createSchedulesTx(database.DB, chef, hours)
}

func (h *UploadHandler) createSchedulesTx(tx *gorm.DB, chef *models.ChefProfile, hours map[string]*DayHoursReq) error {
	dayMap := map[string]int{
		"monday": 1, "tuesday": 2, "wednesday": 3, "thursday": 4,
		"friday": 5, "saturday": 6, "sunday": 0,
	}
	for day, dayNum := range dayMap {
		schedule := models.ChefSchedule{
			ChefID:    chef.ID,
			DayOfWeek: dayNum,
			IsClosed:  true,
		}
		if dh, ok := hours[day]; ok && dh != nil {
			schedule.IsClosed = false
			schedule.OpenTime = dh.Open
			schedule.CloseTime = dh.Close
		}
		if err := tx.Create(&schedule).Error; err != nil {
			return err
		}
	}
	return nil
}

// seedDefaultCategories inserts the standard starter menu categories for a new chef.
// Skips any category whose name already exists (case-insensitive) for this chef.
func (h *UploadHandler) seedDefaultCategories(chefID uuid.UUID) {
	_ = h.seedDefaultCategoriesTx(database.DB, chefID)
}

func (h *UploadHandler) seedDefaultCategoriesTx(tx *gorm.DB, chefID uuid.UUID) error {
	categories := []string{
		"Starters & Snacks",
		"Main Course",
		"Rice & Biryani",
		"Breads & Rotis",
		"Dal & Curries",
		"Thalis & Combos",
		"Desserts & Sweets",
		"Beverages",
		"Breakfast & Tiffin",
		"Specials of the Day",
	}
	for i, name := range categories {
		var existing models.MenuCategory
		if err := tx.Where("chef_id = ? AND LOWER(name) = LOWER(?)", chefID, name).First(&existing).Error; err == nil {
			continue
		}
		cat := models.MenuCategory{
			ChefID:    chefID,
			Name:      name,
			SortOrder: i,
			IsActive:  true,
		}
		if err := tx.Create(&cat).Error; err != nil {
			return fmt.Errorf("seed category %q: %w", name, err)
		}
	}
	return nil
}

// GetDocuments returns the chef's uploaded documents
// GET /chef/documents
func (h *UploadHandler) GetDocuments(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		// No profile yet = no documents
		c.JSON(http.StatusOK, []models.ChefDocumentResponse{})
		return
	}

	var docs []models.ChefDocument
	database.DB.Where("chef_id = ?", chef.ID).Order("created_at DESC").Find(&docs)

	responses := make([]models.ChefDocumentResponse, len(docs))
	for i, doc := range docs {
		resp := doc.ToResponse()
		if models.IsPrivateDoc(doc.Type) {
			// Private files: generate a short-lived signed URL so the chef can view their own docs
			signedURL, err := services.GenerateSignedURL(c.Request.Context(), doc.FilePath, 15*time.Minute)
			if err != nil {
				log.Printf("Failed to generate signed URL for doc %s: %v", doc.ID, err)
				// Don't expose the raw path — just leave URL empty
			} else {
				resp.FileURL = signedURL
			}
		} else {
			// Public files: URL is the public GCS path
			resp.FileURL = doc.FilePath
		}
		responses[i] = resp
	}

	c.JSON(http.StatusOK, responses)
}

// GetExpiringDocuments returns the chef's documents whose expiryDate is within
// the next N days (default 30). Intended for the mobile dashboard banner that
// warns chefs to renew their FSSAI license before it lapses.
//
// GET /chef/documents/expiring?withinDays=30
//
// TODO(v2): Replace polling with a server-push via NATS/notification-service
// so the mobile app is notified pro-actively instead of checking on every
// app open.
func (h *UploadHandler) GetExpiringDocuments(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	withinDays := 30
	if wd := c.Query("withinDays"); wd != "" {
		if n, err := strconv.Atoi(wd); err == nil && n > 0 && n <= 365 {
			withinDays = n
		}
	}

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		// No profile = no documents, return empty rather than 404
		c.JSON(http.StatusOK, gin.H{"documents": []interface{}{}})
		return
	}

	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, withinDays)

	// Fetch documents whose expiryDate is in the future but within the window.
	// We intentionally exclude already-expired documents (expiry_date <= now)
	// because those would need a different warning ("expired") rather than
	// "expiring soon". Handle both in v2.
	var docs []models.ChefDocument
	if err := database.DB.Where(
		"chef_id = ? AND expiry_date > ? AND expiry_date <= ?",
		chef.ID, now, cutoff,
	).Order("expiry_date ASC").Find(&docs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch documents"})
		return
	}

	type expiringDocResponse struct {
		ID              string    `json:"id"`
		Type            string    `json:"type"`
		ExpiryDate      time.Time `json:"expiryDate"`
		DaysUntilExpiry int       `json:"daysUntilExpiry"`
	}

	result := make([]expiringDocResponse, 0, len(docs))
	for _, doc := range docs {
		if doc.ExpiryDate == nil {
			continue
		}
		days := int(doc.ExpiryDate.Sub(now).Hours() / 24)
		result = append(result, expiringDocResponse{
			ID:              doc.ID.String(),
			Type:            string(doc.Type),
			ExpiryDate:      *doc.ExpiryDate,
			DaysUntilExpiry: days,
		})
	}

	c.JSON(http.StatusOK, gin.H{"documents": result})
}

// Request types

type OnboardingRequest struct {
	FullName       string                  `json:"fullName" binding:"required"`
	Phone          string                  `json:"phone" binding:"required"`
	Email          string                  `json:"email"`
	KitchenAddress KitchenAddressReq       `json:"kitchenAddress" binding:"required"`
	BusinessName   string                  `json:"businessName" binding:"required"`
	Description    string                  `json:"description" binding:"required"`
	KitchenType    string                  `json:"kitchenType"`
	Cuisines       []string                `json:"cuisines" binding:"required"`
	Specialties    []string                `json:"specialties"`
	YearsOfExp     string                  `json:"yearsOfExperience"`
	MealsPerDay    string                  `json:"mealsPerDay"`
	PrepTime       string                  `json:"prepTime"`
	ServiceRadius  float64                 `json:"serviceRadius"`
	MinimumOrder   float64                 `json:"minimumOrder"`
	DeliveryFee    float64                 `json:"deliveryFee"`
	OperatingHours map[string]*DayHoursReq `json:"operatingHours"`
	PanNumber      string                  `json:"panNumber"`
	FSSAINumber    string                  `json:"fssaiLicenseNumber"`
	// GSTIN is optional — chefs below the GST threshold don't need one.
	// When provided, persisted to chef_profiles.gstin and printed on
	// customer invoices alongside the FSSAI number.
	GSTIN         string `json:"gstin"`
	AcceptedTerms bool   `json:"acceptedTerms"`
}

type KitchenAddressReq struct {
	Line1      string `json:"line1" binding:"required"`
	Line2      string `json:"line2"`
	City       string `json:"city" binding:"required"`
	State      string `json:"state" binding:"required"`
	PostalCode string `json:"postalCode" binding:"required"`
	Landmark   string `json:"landmark"`
}

type DayHoursReq struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

// ReplaceDocument swaps an existing document's file (and optionally its
// expiry date) and resets verification status to pending so admins
// re-review. Used by the chef-side "renew expired document" flow —
// the chef has already submitted onboarding and just needs to refresh
// a single doc (FSSAI most often) before/after expiry.
//
// Differs from UploadDocument by ID-based addressing: UploadDocument
// is keyed by (chef, type) and deletes any prior doc of that type;
// this one updates the specific record so the audit log of which
// approvals were tied to which document remains coherent.
//
// POST /chef/documents/:id/replace — multipart: file, optional expiryDate
func (h *UploadHandler) ReplaceDocument(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	docID := c.Param("id")

	chef, err := getOrCreateChefProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize chef profile"})
		return
	}

	var doc models.ChefDocument
	if err := database.DB.Where("id = ? AND chef_id = ?", docID, chef.ID).First(&doc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	var newExpiry *time.Time
	if ed := c.PostForm("expiryDate"); ed != "" {
		parsed, perr := time.Parse("2006-01-02", ed)
		if perr != nil {
			parsed, perr = time.Parse(time.RFC3339, ed)
		}
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiryDate format. Use YYYY-MM-DD or RFC3339."})
			return
		}
		newExpiry = &parsed
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum 5 MB."})
		return
	}

	contentType := header.Header.Get("Content-Type")
	isPrivate := models.IsPrivateDoc(doc.Type)
	isPhoto := models.IsPhotoDoc(doc.Type)
	if isPhoto && !services.IsImageContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed: JPEG, PNG, WebP."})
		return
	}
	if !isPhoto && !services.IsDocContentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed: JPEG, PNG, PDF."})
		return
	}

	// Also validate the real bytes so a spoofed Content-Type can't slip through.
	sniffed, serr := sniffContentType(file)
	if serr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read file"})
		return
	}
	if isPhoto && !services.IsImageContentType(sniffed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File contents don't match an allowed image type."})
		return
	}
	if !isPhoto && !services.IsDocContentType(sniffed) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File contents don't match an allowed file type."})
		return
	}

	// Upload the replacement to the same bucket the original lived in.
	// We don't delete the previous GCS object — keeping it around lets
	// support investigate any approval dispute against the prior file.
	// Cost is negligible at our scale.
	folder := fmt.Sprintf("chefs/%s/%s", chef.ID.String(), string(doc.Type))
	var newFileURL, newFilePath, newBucket string
	if isPrivate {
		newBucket = config.AppConfig.GCSPrivateBucket
		newFilePath, err = services.UploadPrivateFile(c.Request.Context(), folder, header.Filename, file, contentType)
	} else {
		newBucket = config.AppConfig.GCSPublicBucket
		newFileURL, err = services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
		newFilePath = newFileURL
	}
	if err != nil {
		log.Printf("Failed to upload replacement file: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Atomic update — file metadata + (optional) expiry + status reset
	// land together, so a half-saved doc never gets shown as still
	// "approved" with the wrong file backing it.
	updates := map[string]interface{}{
		"file_name": header.Filename,
		"file_path": newFilePath,
		// NOTE: no "file_url" — ChefDocument.FileURL is `gorm:"-"` (computed
		// at read time by signing file_path), so there is no file_url column.
		// Writing it via a raw update map throws SQLSTATE 42703 and fails the
		// whole replace. For public docs the URL already lives in file_path.
		"bucket":           newBucket,
		"content_type":     contentType,
		"file_size":        header.Size,
		"status":           models.DocStatusPending,
		"rejection_reason": "",
	}
	if newExpiry != nil {
		updates["expiry_date"] = newExpiry
	}
	if err := database.DB.Model(&doc).Updates(updates).Error; err != nil {
		log.Printf("Failed to update document record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save document"})
		return
	}

	// Cancel any in-flight approval against the old file + create a
	// fresh one for the new submission. Same pattern as UploadDocument
	// so the admin dashboard shows exactly one pending request per
	// chef document.
	database.DB.Model(&models.ApprovalRequest{}).
		Where("chef_id = ? AND type = ? AND entity_id = ? AND status = ?",
			chef.ID, models.ApprovalDocumentVerification, doc.ID, models.ApprovalPending).
		Update("status", models.ApprovalCancelled)

	approvalReq := models.ApprovalRequest{
		Type:          models.ApprovalDocumentVerification,
		Status:        models.ApprovalPending,
		Priority:      "normal",
		ChefID:        &chef.ID,
		SubmittedByID: userID,
		EntityType:    "chef_document",
		EntityID:      doc.ID,
		Title:         fmt.Sprintf("Document Re-Verification: %s", string(doc.Type)),
		Description:   fmt.Sprintf("Document replaced for re-verification: %s (%s)", header.Filename, string(doc.Type)),
		SubmittedData: fmt.Sprintf(`{"document_id":"%s","type":"%s","file_name":"%s","replaced":true}`, doc.ID.String(), string(doc.Type), header.Filename),
	}
	database.DB.Create(&approvalReq)

	if err := services.EnqueueEvent(database.DB, services.SubjectApprovalCreated, "approval.created", userID, map[string]interface{}{
		"approval_id": approvalReq.ID.String(),
		"type":        string(approvalReq.Type),
		"chef_id":     chef.ID.String(),
		"title":       approvalReq.Title,
		"replaced":    true,
	}); err != nil {
		log.Printf("failed to enqueue approval.created event: %v", err)
	}

	// Refresh + return so the client sees the new expiry + status.
	_ = database.DB.First(&doc, "id = ?", doc.ID).Error
	c.JSON(http.StatusOK, doc.ToResponse())
}

// OCRDocument runs Cloud Vision OCR on an image and returns the detected
// FSSAI number + expiry date for the chef to confirm/edit. Stateless — it
// stores nothing; the actual upload still goes through UploadDocument /
// ReplaceDocument with the confirmed values.
// POST /chef/documents/ocr — multipart: file (image only)
func (h *UploadHandler) OCRDocument(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum 5 MB."})
		return
	}
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read file"})
		return
	}

	res, err := services.DetectDocumentFields(c.Request.Context(), data)
	if err != nil {
		// Soft-fail: OCR is a convenience. The chef types the fields manually.
		log.Printf("OCR failed (non-fatal): %v", err)
		c.JSON(http.StatusOK, services.OCRResult{})
		return
	}
	c.JSON(http.StatusOK, res)
}
