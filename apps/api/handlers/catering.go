package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/lib/pq"
)

type CateringHandler struct{}

func NewCateringHandler() *CateringHandler {
	return &CateringHandler{}
}

// ---- Customer Endpoints ----

// CreateRequest creates a new catering request.
// POST /catering/requests
func (h *CateringHandler) CreateRequest(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var input struct {
		EventType    string   `json:"eventType" binding:"required"`
		EventDate    string   `json:"eventDate" binding:"required"`
		EventTime    string   `json:"eventTime"`
		GuestCount   int      `json:"guestCount" binding:"required,min=1"`
		Budget       float64  `json:"budget"`
		CuisineTypes []string `json:"cuisineTypes"`
		DietaryNeeds []string `json:"dietaryNeeds"`
		MenuStyle    string   `json:"menuStyle"`
		Description  string   `json:"description"`
		VenueName    string   `json:"venueName"`
		AddressLine1 string   `json:"addressLine1"`
		AddressLine2 string   `json:"addressLine2"`
		City         string   `json:"city" binding:"required"`
		State        string   `json:"state" binding:"required"`
		PostalCode   string   `json:"postalCode"`
		Latitude     float64  `json:"latitude"`
		Longitude    float64  `json:"longitude"`
		ContactName  string   `json:"contactName"`
		ContactPhone string   `json:"contactPhone"`
		ContactEmail string   `json:"contactEmail"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	eventDate, err := time.Parse("2006-01-02", input.EventDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event date format. Use YYYY-MM-DD."})
		return
	}

	if eventDate.Before(time.Now().Truncate(24 * time.Hour)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event date must be in the future"})
		return
	}

	// Set quote deadline to 7 days from now or 3 days before event, whichever is earlier
	sevenDays := time.Now().AddDate(0, 0, 7)
	threeDaysBefore := eventDate.AddDate(0, 0, -3)
	quoteDeadline := sevenDays
	if threeDaysBefore.Before(sevenDays) {
		quoteDeadline = threeDaysBefore
	}

	request := models.CateringRequest{
		CustomerID:    userID,
		Status:        models.CateringStatusOpen,
		EventType:     input.EventType,
		EventDate:     eventDate,
		EventTime:     input.EventTime,
		GuestCount:    input.GuestCount,
		Budget:        input.Budget,
		CuisineTypes:  pq.StringArray(input.CuisineTypes),
		DietaryNeeds:  pq.StringArray(input.DietaryNeeds),
		MenuStyle:     input.MenuStyle,
		Description:   input.Description,
		VenueName:     input.VenueName,
		AddressLine1:  input.AddressLine1,
		AddressLine2:  input.AddressLine2,
		City:          input.City,
		State:         input.State,
		PostalCode:    input.PostalCode,
		Latitude:      input.Latitude,
		Longitude:     input.Longitude,
		ContactName:   input.ContactName,
		ContactPhone:  input.ContactPhone,
		ContactEmail:  input.ContactEmail,
		QuoteDeadline: &quoteDeadline,
	}

	if err := database.DB.Create(&request).Error; err != nil {
		log.Printf("Failed to create catering request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create catering request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": request.ToResponse()})
}

// GetMyRequests returns the authenticated customer's catering requests.
// GET /catering/requests
func (h *CateringHandler) GetMyRequests(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	status := c.Query("status")

	var requests []models.CateringRequest
	var total int64

	query := database.DB.Where("customer_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Model(&models.CateringRequest{}).Count(&total)

	if err := query.
		Preload("Quotes").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch catering requests"})
		return
	}

	responses := make([]models.CateringRequestResponse, len(requests))
	for i, req := range requests {
		responses[i] = req.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetRequest returns a single catering request with its quotes.
// GET /catering/requests/:id
func (h *CateringHandler) GetRequest(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var request models.CateringRequest
	if err := database.DB.
		Preload("Quotes").
		Preload("Quotes.Chef").
		Where("id = ? AND customer_id = ?", requestID, userID).
		First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catering request not found"})
		return
	}

	// Build quote responses
	quotes := make([]models.CateringQuoteResponse, len(request.Quotes))
	for i, q := range request.Quotes {
		quotes[i] = q.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   request.ToResponse(),
		"quotes": quotes,
	})
}

// GetQuotes returns all quotes for a catering request.
// GET /catering/requests/:id/quotes
func (h *CateringHandler) GetQuotes(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	// Verify request belongs to this customer
	var request models.CateringRequest
	if err := database.DB.Where("id = ? AND customer_id = ?", requestID, userID).First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catering request not found"})
		return
	}

	var quotes []models.CateringQuote
	if err := database.DB.
		Preload("Chef").
		Where("request_id = ?", requestID).
		Order("created_at DESC").
		Find(&quotes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quotes"})
		return
	}

	responses := make([]models.CateringQuoteResponse, len(quotes))
	for i, q := range quotes {
		responses[i] = q.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"count": len(responses),
	})
}

// AcceptQuote accepts a quote for a catering request.
// POST /catering/quotes/:id/accept
func (h *CateringHandler) AcceptQuote(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	quoteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quote ID"})
		return
	}

	// Find the quote
	var quote models.CateringQuote
	if err := database.DB.Preload("Chef").First(&quote, "id = ?", quoteID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}

	// Verify the request belongs to this customer
	var request models.CateringRequest
	if err := database.DB.Where("id = ? AND customer_id = ?", quote.RequestID, userID).First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catering request not found"})
		return
	}

	if request.Status != models.CateringStatusOpen && request.Status != models.CateringStatusQuoted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This request is no longer accepting quotes"})
		return
	}

	if quote.Status != models.QuoteStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This quote cannot be accepted"})
		return
	}

	// Check if quote has expired
	if quote.ValidUntil != nil && quote.ValidUntil.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This quote has expired"})
		return
	}

	now := time.Now()

	// Accept this quote
	quote.Status = models.QuoteStatusAccepted
	quote.AcceptedAt = &now
	if err := database.DB.Save(&quote).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept quote"})
		return
	}

	// Update request status and link the accepted quote
	request.Status = models.CateringStatusAccepted
	request.AcceptedQuoteID = &quote.ID
	if err := database.DB.Save(&request).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update request"})
		return
	}

	// Reject all other pending quotes for this request
	database.DB.Model(&models.CateringQuote{}).
		Where("request_id = ? AND id != ? AND status = ?", request.ID, quote.ID, models.QuoteStatusPending).
		Updates(map[string]interface{}{
			"status":      models.QuoteStatusRejected,
			"rejected_at": now,
		})

	c.JSON(http.StatusOK, gin.H{
		"message": "Quote accepted",
		"data":    quote.ToResponse(),
	})
}

// ---- Chef Endpoints ----

// GetAvailableRequests returns open catering requests in the chef's area.
// GET /chef/catering/requests
func (h *CateringHandler) GetAvailableRequests(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	cuisine := c.Query("cuisine")
	eventType := c.Query("eventType")
	minGuests := c.Query("minGuests")
	maxGuests := c.Query("maxGuests")

	var requests []models.CateringRequest
	var total int64

	query := database.DB.Where("status = ? AND event_date > ?", models.CateringStatusOpen, time.Now())

	// Exclude requests where this chef has already quoted
	query = query.Where("id NOT IN (?)",
		database.DB.Model(&models.CateringQuote{}).Select("request_id").Where("chef_id = ?", chef.ID),
	)

	// Filter by cuisine if specified
	if cuisine != "" {
		query = query.Where("? = ANY(cuisine_types)", strings.ToLower(cuisine))
	}
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}
	if minGuests != "" {
		if v, err := strconv.Atoi(minGuests); err == nil {
			query = query.Where("guest_count >= ?", v)
		}
	}
	if maxGuests != "" {
		if v, err := strconv.Atoi(maxGuests); err == nil {
			query = query.Where("guest_count <= ?", v)
		}
	}

	// Check quote deadline
	query = query.Where("quote_deadline IS NULL OR quote_deadline > ?", time.Now())

	query.Model(&models.CateringRequest{}).Count(&total)

	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch catering requests"})
		return
	}

	responses := make([]models.CateringRequestResponse, len(requests))
	for i, req := range requests {
		responses[i] = req.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// SubmitQuote submits a quote for a catering request.
// POST /chef/catering/requests/:id/quote
func (h *CateringHandler) SubmitQuote(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	// Verify the request exists and is open
	var request models.CateringRequest
	if err := database.DB.First(&request, "id = ?", requestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catering request not found"})
		return
	}

	if request.Status != models.CateringStatusOpen {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This request is no longer accepting quotes"})
		return
	}

	// Check quote deadline
	if request.QuoteDeadline != nil && request.QuoteDeadline.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quote deadline has passed"})
		return
	}

	// Check if chef already submitted a quote
	var existingCount int64
	database.DB.Model(&models.CateringQuote{}).
		Where("request_id = ? AND chef_id = ?", requestID, chef.ID).
		Count(&existingCount)
	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "You have already submitted a quote for this request"})
		return
	}

	var input struct {
		ProposedMenu      string   `json:"proposedMenu" binding:"required"`
		MenuItems         []string `json:"menuItems"`
		PricePerPerson    float64  `json:"pricePerPerson" binding:"required,min=0"`
		TotalPrice        float64  `json:"totalPrice" binding:"required,min=0"`
		Notes             string   `json:"notes"`
		IncludesSetup     bool     `json:"includesSetup"`
		IncludesServing   bool     `json:"includesServing"`
		IncludesCleanup   bool     `json:"includesCleanup"`
		IncludesEquipment bool     `json:"includesEquipment"`
		ValidDays         int      `json:"validDays"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Set validity period (default 7 days)
	validDays := input.ValidDays
	if validDays <= 0 {
		validDays = 7
	}
	validUntil := time.Now().AddDate(0, 0, validDays)

	quote := models.CateringQuote{
		RequestID:         requestID,
		ChefID:            chef.ID,
		Status:            models.QuoteStatusPending,
		ProposedMenu:      input.ProposedMenu,
		MenuItems:         pq.StringArray(input.MenuItems),
		PricePerPerson:    input.PricePerPerson,
		TotalPrice:        input.TotalPrice,
		Notes:             input.Notes,
		IncludesSetup:     input.IncludesSetup,
		IncludesServing:   input.IncludesServing,
		IncludesCleanup:   input.IncludesCleanup,
		IncludesEquipment: input.IncludesEquipment,
		ValidUntil:        &validUntil,
	}

	if err := database.DB.Create(&quote).Error; err != nil {
		log.Printf("Failed to create catering quote: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit quote"})
		return
	}

	// Update request status to "quoted" if still open
	if request.Status == models.CateringStatusOpen {
		database.DB.Model(&request).Update("status", models.CateringStatusQuoted)
	}

	// Reload with chef data
	database.DB.Preload("Chef").First(&quote, "id = ?", quote.ID)

	c.JSON(http.StatusCreated, gin.H{"data": quote.ToResponse()})
}

// GetChefQuotes returns all quotes submitted by the authenticated chef.
// GET /chef/catering/quotes
func (h *CateringHandler) GetChefQuotes(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}
	offset := (page - 1) * limit

	status := c.Query("status")

	var quotes []models.CateringQuote
	var total int64

	query := database.DB.Where("chef_id = ?", chef.ID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Model(&models.CateringQuote{}).Count(&total)

	if err := query.
		Preload("Chef").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&quotes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quotes"})
		return
	}

	responses := make([]models.CateringQuoteResponse, len(quotes))
	for i, q := range quotes {
		responses[i] = q.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}
