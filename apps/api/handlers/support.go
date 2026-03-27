package handlers

import (
	"fmt"
	"math/rand"
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

type SupportHandler struct{}

func NewSupportHandler() *SupportHandler {
	return &SupportHandler{}
}

// ---------- Customer / Chef / Delivery endpoints ----------

// CreateTicket creates a new support ticket.
// POST /support/tickets
func (h *SupportHandler) CreateTicket(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	userRole, _ := middleware.GetUserRole(c)

	var req struct {
		OrderID     *uuid.UUID `json:"orderId"`
		Category    string     `json:"category" binding:"required"`
		Priority    string     `json:"priority"`
		Subject     string     `json:"subject" binding:"required"`
		Description string     `json:"description" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate category
	validCategories := map[string]bool{
		"order_issue": true, "payment_issue": true, "account_issue": true,
		"chef_complaint": true, "delivery_complaint": true, "technical": true, "other": true,
	}
	if !validCategories[req.Category] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		return
	}

	// Default priority
	priority := models.TicketPriorityMedium
	if req.Priority != "" {
		validPriorities := map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
		if !validPriorities[req.Priority] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid priority"})
			return
		}
		priority = models.TicketPriority(req.Priority)
	}

	// Verify linked order belongs to the user
	if req.OrderID != nil {
		var order models.Order
		if err := database.DB.Where("id = ? AND customer_id = ?", *req.OrderID, userID).
			First(&order).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order not found"})
			return
		}
	}

	// PII filter the description
	filteredDesc, _, _ := services.FilterChatMessage(req.Description)

	// Determine reporter role
	reporterRole := "customer"
	switch userRole {
	case models.RoleChef:
		reporterRole = "chef"
	case models.RoleDelivery:
		reporterRole = "delivery"
	}

	ticket := models.SupportTicket{
		TicketNumber: generateTicketNumber(),
		ReporterID:   userID,
		ReporterRole: reporterRole,
		OrderID:      req.OrderID,
		Category:     models.TicketCategory(req.Category),
		Priority:     priority,
		Status:       models.TicketStatusOpen,
		Subject:      req.Subject,
		Description:  filteredDesc,
	}

	if err := database.DB.Create(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ticket"})
		return
	}

	c.JSON(http.StatusCreated, ticket)
}

// GetMyTickets lists tickets for the authenticated user.
// GET /support/tickets
func (h *SupportHandler) GetMyTickets(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := database.DB.Where("reporter_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.SupportTicket{}).Count(&total)

	var tickets []models.SupportTicket
	if err := query.Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tickets,
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

// GetTicket returns a single ticket with its messages.
// GET /support/tickets/:id
func (h *SupportHandler) GetTicket(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	ticketID := c.Param("id")

	var ticket models.SupportTicket
	if err := database.DB.Preload("Messages", "is_internal = ?", false).
		Preload("Messages.Sender").
		Where("id = ? AND reporter_id = ?", ticketID, userID).
		First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

// AddMessage adds a message to a ticket.
// POST /support/tickets/:id/messages
func (h *SupportHandler) AddMessage(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	userRole, _ := middleware.GetUserRole(c)
	ticketID := c.Param("id")

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ticket models.SupportTicket
	if err := database.DB.Where("id = ? AND reporter_id = ?", ticketID, userID).
		First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	if ticket.Status == models.TicketStatusClosed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticket is closed"})
		return
	}

	// PII filter the content
	filteredContent, piiDetected, _ := services.FilterChatMessage(req.Content)

	senderRole := "customer"
	switch userRole {
	case models.RoleChef:
		senderRole = "chef"
	case models.RoleDelivery:
		senderRole = "delivery"
	}

	msg := models.SupportMessage{
		TicketID:    ticket.ID,
		SenderID:    userID,
		SenderRole:  senderRole,
		Content:     filteredContent,
		PIIDetected: piiDetected,
		IsInternal:  false,
	}

	if err := database.DB.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add message"})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// CloseTicket closes a ticket (reporter or admin).
// PUT /support/tickets/:id/close
func (h *SupportHandler) CloseTicket(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	ticketID := c.Param("id")

	var ticket models.SupportTicket
	if err := database.DB.Where("id = ? AND reporter_id = ?", ticketID, userID).
		First(&ticket).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	if ticket.Status == models.TicketStatusClosed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ticket is already closed"})
		return
	}

	now := time.Now()
	ticket.Status = models.TicketStatusClosed
	ticket.ClosedAt = &now

	if err := database.DB.Save(&ticket).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close ticket"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

// ---------- Admin endpoints ----------

// AdminGetTickets returns all tickets with filters.
// GET /admin/support/tickets
func (h *SupportHandler) AdminGetTickets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")
	priority := c.Query("priority")
	category := c.Query("category")
	assignee := c.Query("assignee")

	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.SupportTicket{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if assignee != "" {
		assigneeID, err := uuid.Parse(assignee)
		if err == nil {
			query = query.Where("assigned_to_id = ?", assigneeID)
		}
	}

	var total int64
	query.Count(&total)

	var tickets []models.SupportTicket
	if err := query.Preload("Reporter").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&tickets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tickets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tickets,
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

// AdminGetTicket returns full ticket detail including internal notes.
// GET /admin/support/tickets/:id
func (h *SupportHandler) AdminGetTicket(c *gin.Context) {
	ticketID := c.Param("id")

	var ticket models.SupportTicket
	if err := database.DB.Preload("Messages").Preload("Messages.Sender").
		Preload("Reporter").Preload("Order").
		First(&ticket, "id = ?", ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

// AdminAssignTicket assigns a ticket to an admin staff member.
// PUT /admin/support/tickets/:id/assign
func (h *SupportHandler) AdminAssignTicket(c *gin.Context) {
	ticketID := c.Param("id")

	var req struct {
		AssignedToID uuid.UUID `json:"assignedToId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify assignee exists
	var assignee models.User
	if err := database.DB.First(&assignee, "id = ?", req.AssignedToID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Assignee not found"})
		return
	}

	result := database.DB.Model(&models.SupportTicket{}).Where("id = ?", ticketID).
		Updates(map[string]interface{}{
			"assigned_to_id": req.AssignedToID,
			"status":         models.TicketStatusInProgress,
		})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ticket assigned"})
}

// AdminUpdateTicketStatus updates the status of a ticket.
// PUT /admin/support/tickets/:id/status
func (h *SupportHandler) AdminUpdateTicketStatus(c *gin.Context) {
	ticketID := c.Param("id")

	var req struct {
		Status     string `json:"status" binding:"required"`
		Resolution string `json:"resolution"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validStatuses := map[string]bool{
		"open": true, "in_progress": true, "waiting_on_customer": true,
		"waiting_on_chef": true, "resolved": true, "closed": true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	updates := map[string]interface{}{
		"status": req.Status,
	}

	if req.Resolution != "" {
		updates["resolution"] = req.Resolution
	}

	now := time.Now()
	if req.Status == "resolved" {
		updates["resolved_at"] = &now
	}
	if req.Status == "closed" {
		updates["closed_at"] = &now
	}

	result := database.DB.Model(&models.SupportTicket{}).Where("id = ?", ticketID).Updates(updates)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ticket status updated"})
}

// AdminAddMessage adds a message to a ticket (can be internal).
// POST /admin/support/tickets/:id/messages
func (h *SupportHandler) AdminAddMessage(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	ticketID := c.Param("id")

	var req struct {
		Content    string `json:"content" binding:"required"`
		IsInternal bool   `json:"isInternal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ticket models.SupportTicket
	if err := database.DB.First(&ticket, "id = ?", ticketID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	msg := models.SupportMessage{
		TicketID:    ticket.ID,
		SenderID:    userID,
		SenderRole:  "admin",
		Content:     req.Content,
		PIIDetected: false,
		IsInternal:  req.IsInternal,
	}

	if err := database.DB.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add message"})
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// AdminGetSupportStats returns ticket statistics.
// GET /admin/support/stats
func (h *SupportHandler) AdminGetSupportStats(c *gin.Context) {
	db := database.DB

	type StatusCount struct {
		Status string
		Count  int
	}

	var statusCounts []StatusCount
	db.Model(&models.SupportTicket{}).Select("status, count(*) as count").Group("status").Scan(&statusCounts)

	statusMap := make(map[string]int)
	totalOpen := 0
	totalResolved := 0
	for _, sc := range statusCounts {
		statusMap[sc.Status] = sc.Count
		if sc.Status == "open" || sc.Status == "in_progress" || sc.Status == "waiting_on_customer" || sc.Status == "waiting_on_chef" {
			totalOpen += sc.Count
		}
		if sc.Status == "resolved" || sc.Status == "closed" {
			totalResolved += sc.Count
		}
	}

	// Average resolution time (hours) for resolved tickets
	var avgResolutionHours float64
	db.Model(&models.SupportTicket{}).
		Where("resolved_at IS NOT NULL").
		Select("COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600), 0)").
		Scan(&avgResolutionHours)

	var totalTickets int64
	db.Model(&models.SupportTicket{}).Count(&totalTickets)

	c.JSON(http.StatusOK, gin.H{
		"total":              totalTickets,
		"open":               totalOpen,
		"resolved":           totalResolved,
		"byStatus":           statusMap,
		"avgResolutionHours": avgResolutionHours,
	})
}

// ---------- Helpers ----------

func generateTicketNumber() string {
	now := time.Now()
	random := rand.Intn(9999)
	return fmt.Sprintf("TKT-%s-%04d", now.Format("200601"), random)
}
