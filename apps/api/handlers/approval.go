package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// ApprovalHandler manages admin approval workflows
type ApprovalHandler struct{}

func NewApprovalHandler() *ApprovalHandler {
	return &ApprovalHandler{}
}

// GetApprovalRequests returns paginated approval requests with filters
// GET /admin/approvals?page=1&limit=20&type=&status=&chefId=&priority=&search=
func (h *ApprovalHandler) GetApprovalRequests(c *gin.Context) {
	db := database.DB
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	filterType := c.Query("type")
	filterStatus := c.Query("status")
	chefID := c.Query("chefId")
	priority := c.Query("priority")
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := db.Model(&models.ApprovalRequest{}).Preload("Chef.User").Preload("Partner.User").Preload("SubmittedBy")

	// Category filter: "chef" for chef/vendor requests, "driver" for driver requests
	category := c.Query("category")
	if category == "chef" {
		query = query.Where("type IN ?", []string{
			string(models.ApprovalKitchenOnboarding),
			string(models.ApprovalDocumentVerification),
			string(models.ApprovalMenuItemNew),
			string(models.ApprovalMenuItemUpdate),
			string(models.ApprovalPricingChange),
			string(models.ApprovalKitchenUpdate),
		})
	} else if category == "driver" {
		query = query.Where("type IN ?", []string{
			string(models.ApprovalDriverOnboarding),
			string(models.ApprovalDriverDocument),
		})
	}

	if filterType != "" {
		query = query.Where("type = ?", filterType)
	}
	if filterStatus != "" {
		query = query.Where("status = ?", filterStatus)
	}
	if chefID != "" {
		query = query.Where("chef_id = ?", chefID)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var approvals []models.ApprovalRequest
	// Sort by priority (urgent first), then by created_at DESC
	query.Order(`
		CASE priority
			WHEN 'urgent' THEN 1
			WHEN 'high' THEN 2
			WHEN 'normal' THEN 3
			WHEN 'low' THEN 4
			ELSE 5
		END ASC, created_at DESC
	`).Offset(offset).Limit(limit).Find(&approvals)

	c.JSON(http.StatusOK, gin.H{
		"data": approvals,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// GetApprovalCounts returns counts by status and by type
// GET /admin/approvals/counts
func (h *ApprovalHandler) GetApprovalCounts(c *gin.Context) {
	db := database.DB

	// Counts by status
	type StatusCount struct {
		Status string
		Count  int
	}
	var statusCounts []StatusCount
	db.Model(&models.ApprovalRequest{}).Select("status, count(*) as count").Group("status").Scan(&statusCounts)

	byStatus := make(map[string]int)
	totalCount := 0
	for _, sc := range statusCounts {
		byStatus[sc.Status] = sc.Count
		totalCount += sc.Count
	}

	// Counts by type (only pending)
	type TypeCount struct {
		Type  string
		Count int
	}
	var typeCounts []TypeCount
	db.Model(&models.ApprovalRequest{}).Where("status = ?", models.ApprovalPending).
		Select("type, count(*) as count").Group("type").Scan(&typeCounts)

	byType := make(map[string]int)
	for _, tc := range typeCounts {
		byType[tc.Type] = tc.Count
	}

	c.JSON(http.StatusOK, gin.H{
		"byStatus": byStatus,
		"byType":   byType,
		"total":    totalCount,
	})
}

// GetApprovalRequest returns a single approval request with all relations
// GET /admin/approvals/:id
func (h *ApprovalHandler) GetApprovalRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval ID"})
		return
	}

	var approval models.ApprovalRequest
	if err := database.DB.
		Preload("Chef.User").
		Preload("Partner.User").
		Preload("SubmittedBy").
		Preload("ReviewedBy").
		First(&approval, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Approval request not found"})
		return
	}

	// Fetch documents based on approval type
	response := gin.H{
		"id":            approval.ID,
		"type":          approval.Type,
		"status":        approval.Status,
		"priority":      approval.Priority,
		"chefId":        approval.ChefID,
		"partnerId":     approval.PartnerID,
		"submittedById": approval.SubmittedByID,
		"reviewedById":  approval.ReviewedByID,
		"entityType":    approval.EntityType,
		"entityId":      approval.EntityID,
		"title":         approval.Title,
		"description":   approval.Description,
		"submittedData": approval.SubmittedData,
		"adminNotes":    approval.AdminNotes,
		"reviewedAt":    approval.ReviewedAt,
		"createdAt":     approval.CreatedAt,
		"updatedAt":     approval.UpdatedAt,
		"chef":          approval.Chef,
		"partner":       approval.Partner,
		"submittedBy":   approval.SubmittedBy,
		"reviewedBy":    approval.ReviewedBy,
	}

	// Fetch appropriate documents based on approval type
	if approval.Type == models.ApprovalDriverOnboarding || approval.Type == models.ApprovalDriverDocument {
		// Fetch partner (driver) documents
		var partnerDocs []models.DeliveryPartnerDocument
		database.DB.Where("partner_id = ?", approval.PartnerID).Order("created_at DESC").Find(&partnerDocs)
		docResponses := make([]map[string]interface{}, len(partnerDocs))
		for i, d := range partnerDocs {
			docResponses[i] = map[string]interface{}{
				"id":              d.ID,
				"type":            d.Type,
				"fileName":        d.FileName,
				"contentType":     d.ContentType,
				"fileSize":        d.FileSize,
				"status":          d.Status,
				"rejectionReason": d.RejectionReason,
				"createdAt":       d.CreatedAt,
			}
		}
		response["documents"] = docResponses
	} else {
		// Fetch chef documents
		var docs []models.ChefDocument
		database.DB.Where("chef_id = ?", approval.ChefID).Order("created_at DESC").Find(&docs)
		response["documents"] = docs
	}

	c.JSON(http.StatusOK, response)
}

// ApproveRequest approves an approval request and applies side effects
// PUT /admin/approvals/:id/approve
func (h *ApprovalHandler) ApproveRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval ID"})
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	c.ShouldBindJSON(&req)

	adminUserID, _ := middleware.GetUserID(c)

	var approval models.ApprovalRequest
	if err := database.DB.First(&approval, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Approval request not found"})
		return
	}

	if approval.Status != models.ApprovalPending && approval.Status != models.ApprovalInfoRequested {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot approve request with status '%s'", approval.Status)})
		return
	}

	previousStatus := string(approval.Status)
	now := time.Now()

	// Update approval request
	database.DB.Model(&approval).Updates(map[string]interface{}{
		"status":        models.ApprovalApproved,
		"reviewed_by_id": adminUserID,
		"reviewed_at":   &now,
		"admin_notes":   req.Notes,
	})

	// Create history entry
	history := models.ApprovalRequestHistory{
		ApprovalID:  approval.ID,
		FromStatus:  previousStatus,
		ToStatus:    string(models.ApprovalApproved),
		ChangedByID: adminUserID,
		Notes:       req.Notes,
	}
	database.DB.Create(&history)

	// Apply side effects based on type
	switch approval.Type {
	case models.ApprovalKitchenOnboarding:
		database.DB.Model(&models.ChefProfile{}).Where("id = ?", approval.ChefID).Updates(map[string]interface{}{
			"is_verified": true,
			"verified_at": &now,
			"is_active":   true,
		})
		// Update user role to chef
		var chef models.ChefProfile
		if err := database.DB.First(&chef, "id = ?", approval.ChefID).Error; err == nil {
			database.DB.Model(&models.User{}).Where("id = ?", chef.UserID).Update("role", models.RoleChef)
		}

	case models.ApprovalDocumentVerification:
		database.DB.Model(&models.ChefDocument{}).
			Where("chef_id = ? AND status = ?", approval.ChefID, models.DocStatusPending).
			Update("status", models.DocStatusVerified)

	case models.ApprovalMenuItemNew, models.ApprovalMenuItemUpdate, models.ApprovalPricingChange:
		// Approve the menu item - make it visible to customers
		database.DB.Model(&models.MenuItem{}).
			Where("id = ?", approval.EntityID).
			Update("is_approved", true)

	case models.ApprovalDriverOnboarding:
		if approval.PartnerID != nil {
			database.DB.Model(&models.DeliveryPartner{}).Where("id = ?", *approval.PartnerID).Updates(map[string]interface{}{
				"verification_status": models.VerificationApproved,
				"is_verified":         true,
				"verified_at":         &now,
				"verified_by_id":      adminUserID,
			})
			// Approve all pending partner documents
			database.DB.Model(&models.DeliveryPartnerDocument{}).
				Where("partner_id = ? AND status = ?", *approval.PartnerID, "pending").
				Update("status", "verified")
		}

	case models.ApprovalDriverDocument:
		if approval.PartnerID != nil {
			database.DB.Model(&models.DeliveryPartnerDocument{}).
				Where("partner_id = ? AND status = ?", *approval.PartnerID, "pending").
				Update("status", "verified")
		}
	}

	// Publish NATS event — include partner_id for driver approvals
	eventData := map[string]interface{}{
		"approval_id": approval.ID.String(),
		"type":        string(approval.Type),
		"title":       approval.Title,
		"notes":       req.Notes,
	}
	if approval.ChefID != nil {
		eventData["chef_id"] = approval.ChefID.String()
	}
	if approval.PartnerID != nil {
		eventData["partner_id"] = approval.PartnerID.String()
	}
	if err := services.PublishEvent(services.SubjectApprovalApproved, "approval.approved", adminUserID, eventData); err != nil {
		log.Printf("Failed to publish approval approved event: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Approval request approved"})
}

// RejectRequest rejects an approval request and applies side effects
// PUT /admin/approvals/:id/reject
func (h *ApprovalHandler) RejectRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval ID"})
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	c.ShouldBindJSON(&req)

	adminUserID, _ := middleware.GetUserID(c)

	var approval models.ApprovalRequest
	if err := database.DB.First(&approval, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Approval request not found"})
		return
	}

	if approval.Status != models.ApprovalPending && approval.Status != models.ApprovalInfoRequested {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot reject request with status '%s'", approval.Status)})
		return
	}

	previousStatus := string(approval.Status)
	now := time.Now()

	// Update approval request
	database.DB.Model(&approval).Updates(map[string]interface{}{
		"status":        models.ApprovalRejected,
		"reviewed_by_id": adminUserID,
		"reviewed_at":   &now,
		"admin_notes":   req.Notes,
	})

	// Create history entry
	history := models.ApprovalRequestHistory{
		ApprovalID:  approval.ID,
		FromStatus:  previousStatus,
		ToStatus:    string(models.ApprovalRejected),
		ChangedByID: adminUserID,
		Notes:       req.Notes,
	}
	database.DB.Create(&history)

	// Apply side effects based on type
	switch approval.Type {
	case models.ApprovalKitchenOnboarding:
		database.DB.Model(&models.ChefProfile{}).Where("id = ?", approval.ChefID).Updates(map[string]interface{}{
			"is_active": false,
		})

	case models.ApprovalDocumentVerification:
		database.DB.Model(&models.ChefDocument{}).
			Where("chef_id = ? AND status = ?", approval.ChefID, models.DocStatusPending).
			Updates(map[string]interface{}{
				"status":           models.DocStatusRejected,
				"rejection_reason": req.Notes,
			})

	case models.ApprovalMenuItemNew, models.ApprovalMenuItemUpdate, models.ApprovalPricingChange:
		// Rejected menu items stay unapproved (not visible to customers)
		database.DB.Model(&models.MenuItem{}).
			Where("id = ?", approval.EntityID).
			Update("is_approved", false)

	case models.ApprovalDriverOnboarding:
		if approval.PartnerID != nil {
			database.DB.Model(&models.DeliveryPartner{}).Where("id = ?", *approval.PartnerID).Updates(map[string]interface{}{
				"verification_status": models.VerificationRejected,
				"rejection_reason":    req.Notes,
				"is_verified":         false,
			})
		}

	case models.ApprovalDriverDocument:
		if approval.PartnerID != nil {
			database.DB.Model(&models.DeliveryPartnerDocument{}).
				Where("partner_id = ? AND status = ?", *approval.PartnerID, "pending").
				Updates(map[string]interface{}{
					"status":           "rejected",
					"rejection_reason": req.Notes,
				})
		}
	}

	// Publish NATS event — include partner_id for driver approvals
	eventData := map[string]interface{}{
		"approval_id": approval.ID.String(),
		"type":        string(approval.Type),
		"title":       approval.Title,
		"notes":       req.Notes,
	}
	if approval.ChefID != nil {
		eventData["chef_id"] = approval.ChefID.String()
	}
	if approval.PartnerID != nil {
		eventData["partner_id"] = approval.PartnerID.String()
	}
	if err := services.PublishEvent(services.SubjectApprovalRejected, "approval.rejected", adminUserID, eventData); err != nil {
		log.Printf("Failed to publish approval rejected event: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Approval request rejected"})
}

// RequestMoreInfo requests additional information from the chef
// PUT /admin/approvals/:id/request-info
func (h *ApprovalHandler) RequestMoreInfo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval ID"})
		return
	}

	var req struct {
		Notes string `json:"notes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Notes are required when requesting more info"})
		return
	}

	adminUserID, _ := middleware.GetUserID(c)

	var approval models.ApprovalRequest
	if err := database.DB.First(&approval, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Approval request not found"})
		return
	}

	if approval.Status != models.ApprovalPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cannot request info for request with status '%s'", approval.Status)})
		return
	}

	previousStatus := string(approval.Status)

	// Update approval request
	database.DB.Model(&approval).Updates(map[string]interface{}{
		"status":     models.ApprovalInfoRequested,
		"admin_notes": req.Notes,
	})

	// Create history entry
	history := models.ApprovalRequestHistory{
		ApprovalID:  approval.ID,
		FromStatus:  previousStatus,
		ToStatus:    string(models.ApprovalInfoRequested),
		ChangedByID: adminUserID,
		Notes:       req.Notes,
	}
	database.DB.Create(&history)

	// Publish NATS event
	if err := services.PublishEvent(services.SubjectApprovalInfoRequested, "approval.info_requested", adminUserID, map[string]interface{}{
		"approval_id": approval.ID.String(),
		"type":        string(approval.Type),
		"chef_id":     approval.ChefID,
		"title":       approval.Title,
		"notes":       req.Notes,
	}); err != nil {
		log.Printf("Failed to publish approval info_requested event: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "More information requested"})
}

// GetApprovalHistory returns all history entries for a request
// GET /admin/approvals/:id/history
func (h *ApprovalHandler) GetApprovalHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval ID"})
		return
	}

	// Verify the approval request exists
	var approval models.ApprovalRequest
	if err := database.DB.First(&approval, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Approval request not found"})
		return
	}

	var history []models.ApprovalRequestHistory
	database.DB.Where("approval_id = ?", id).
		Preload("ChangedBy").
		Order("created_at ASC").
		Find(&history)

	c.JSON(http.StatusOK, gin.H{"data": history})
}

// GetDocumentDownload returns a download URL for a document associated with an approval
// GET /admin/approvals/:id/documents/:docId
func (h *ApprovalHandler) GetDocumentDownload(c *gin.Context) {
	_, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval ID"})
		return
	}

	docID, err := uuid.Parse(c.Param("docId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	var doc models.ChefDocument
	if err := database.DB.First(&doc, "id = ?", docID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// For public files, return the stored URL directly
	if !models.IsPrivateDoc(doc.Type) {
		c.JSON(http.StatusOK, gin.H{
			"url":         doc.FilePath,
			"fileName":    doc.FileName,
			"contentType": doc.ContentType,
		})
		return
	}

	// For private files, generate a signed URL
	signedURL, err := services.GenerateSignedURL(c.Request.Context(), doc.FilePath, 15*time.Minute)
	if err != nil {
		log.Printf("Failed to generate signed URL for document %s: %v", doc.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":         signedURL,
		"fileName":    doc.FileName,
		"contentType": doc.ContentType,
	})
}

// GetChefApprovalRequests returns approval requests for the authenticated chef's kitchen
// GET /chef/admin-requests — accessible to chefs (not admin-only)
func (h *ApprovalHandler) GetChefApprovalRequests(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Find chef profile for this user
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "pagination": gin.H{
			"page": 1, "limit": 20, "total": 0, "totalPages": 0, "hasNext": false, "hasPrev": false,
		}})
		return
	}

	// Get all approval requests for this chef's kitchen
	var requests []models.ApprovalRequest
	database.DB.Where("chef_id = ?", chef.ID).
		Preload("ReviewedBy").
		Order("created_at DESC").
		Find(&requests)

	c.JSON(http.StatusOK, gin.H{
		"data": requests,
		"pagination": gin.H{
			"page":       1,
			"limit":      100,
			"total":      len(requests),
			"totalPages": 1,
			"hasNext":    false,
			"hasPrev":    false,
		},
	})
}

// RespondToApprovalRequest allows a chef to respond to an admin request
// PUT /chef/admin-requests/:id/respond — accessible to chefs
func (h *ApprovalHandler) RespondToApprovalRequest(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var req struct {
		Response string `json:"response" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Response message is required"})
		return
	}

	// Verify the approval belongs to this chef
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No kitchen profile found"})
		return
	}

	var approval models.ApprovalRequest
	if err := database.DB.Where("id = ? AND chef_id = ?", requestID, chef.ID).First(&approval).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	// Update status back to pending (chef has responded)
	oldStatus := string(approval.Status)
	database.DB.Model(&approval).Updates(map[string]interface{}{
		"status": models.ApprovalPending,
	})

	// Create history entry for the chef's response
	history := models.ApprovalRequestHistory{
		ApprovalID:  approval.ID,
		FromStatus:  oldStatus,
		ToStatus:    string(models.ApprovalPending),
		ChangedByID: userID,
		Notes:       req.Response,
	}
	database.DB.Create(&history)

	// Publish NATS event to notify admins
	if err := services.PublishEvent(services.SubjectApprovalCreated, "approval.chef_responded", userID, map[string]interface{}{
		"approval_id": approval.ID.String(),
		"type":        string(approval.Type),
		"chef_id":     chef.ID.String(),
		"title":       approval.Title,
		"response":    req.Response,
	}); err != nil {
		log.Printf("Failed to publish chef response event: %v", err)
	}

	// Create notification for all admin users
	var admins []models.User
	database.DB.Where("role = ?", models.RoleAdmin).Find(&admins)
	for _, admin := range admins {
		notif := &models.Notification{
			UserID:  admin.ID,
			Type:    "chef_responded",
			Title:   "Chef Responded: " + approval.Title,
			Message: req.Response,
		}
		database.DB.Create(notif)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Response sent to admin"})
}
