package handlers

import (
	"fmt"
	"log"
	"math"
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

	// Accept the quote, confirm the booking's deposit terms, reject the rest, and
	// stage the chef notification — atomically (transactional outbox).
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		quote.Status = models.QuoteStatusAccepted
		quote.AcceptedAt = &now
		if err := tx.Save(&quote).Error; err != nil {
			return err
		}

		// The accepted quote sets the deposit the customer now owes to confirm.
		request.Status = models.CateringStatusAccepted
		request.AcceptedQuoteID = &quote.ID
		request.DepositAmount = quote.DepositAmount
		request.DepositStatus = "pending"
		if err := tx.Save(&request).Error; err != nil {
			return err
		}

		// Reject all other pending quotes for this request.
		if err := tx.Model(&models.CateringQuote{}).
			Where("request_id = ? AND id != ? AND status = ?", request.ID, quote.ID, models.QuoteStatusPending).
			Updates(map[string]interface{}{
				"status":      models.QuoteStatusRejected,
				"rejected_at": now,
			}).Error; err != nil {
			return err
		}

		// Notify the chef their quote was accepted (best-effort via the outbox).
		return services.EnqueueEvent(tx, services.SubjectCateringQuote, "catering.quote_accepted",
			cateringChefUserID(quote.ChefID), map[string]any{
				"requestId":     request.ID.String(),
				"quoteId":       quote.ID.String(),
				"eventType":     request.EventType,
				"guestCount":    request.GuestCount,
				"depositAmount": request.DepositAmount,
			})
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept quote"})
		return
	}

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
		DepositAmount     float64  `json:"depositAmount"`
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

	// Deposit defaults to 25% of the total when the chef doesn't set one, and is
	// clamped to the total. This is the advance the customer pays to confirm (#55).
	deposit := defaultCateringDeposit(input.DepositAmount, input.TotalPrice)

	quote := models.CateringQuote{
		RequestID:         requestID,
		ChefID:            chef.ID,
		Status:            models.QuoteStatusPending,
		ProposedMenu:      input.ProposedMenu,
		MenuItems:         pq.StringArray(input.MenuItems),
		PricePerPerson:    input.PricePerPerson,
		TotalPrice:        input.TotalPrice,
		DepositAmount:     deposit,
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

// ---- Helpers ----

// defaultCateringDeposit returns the deposit to store on a quote: the chef's value
// when positive (clamped to total), else 25% of the total (#55).
func defaultCateringDeposit(deposit, total float64) float64 {
	if deposit <= 0 {
		deposit = total * 0.25
	}
	if total > 0 && deposit > total {
		deposit = total
	}
	return math.Round(deposit*100) / 100
}

// cateringChefUserID resolves a ChefProfile ID to its owning user ID (for
// notification targeting). Returns uuid.Nil if not found — the event still
// records, just unaddressed.
func cateringChefUserID(chefID uuid.UUID) uuid.UUID {
	var chef models.ChefProfile
	if err := database.DB.Select("user_id").First(&chef, "id = ?", chefID).Error; err != nil {
		return uuid.Nil
	}
	return chef.UserID
}

// DeclineQuote rejects a single pending quote without accepting another.
// POST /catering/quotes/:id/decline
func (h *CateringHandler) DeclineQuote(c *gin.Context) {
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
	var quote models.CateringQuote
	if err := database.DB.First(&quote, "id = ?", quoteID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quote not found"})
		return
	}
	// Ownership: the quote's request must belong to this customer.
	var request models.CateringRequest
	if err := database.DB.Where("id = ? AND customer_id = ?", quote.RequestID, userID).First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catering request not found"})
		return
	}
	if quote.Status != models.QuoteStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only a pending quote can be declined"})
		return
	}
	now := time.Now()
	quote.Status = models.QuoteStatusRejected
	quote.RejectedAt = &now
	if err := database.DB.Save(&quote).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decline quote"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Quote declined", "data": quote.ToResponse()})
}

// CancelRequest cancels a catering request that hasn't been paid/confirmed yet.
// POST /catering/requests/:id/cancel
func (h *CateringHandler) CancelRequest(c *gin.Context) {
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
	if err := database.DB.Where("id = ? AND customer_id = ?", requestID, userID).First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catering request not found"})
		return
	}
	// A confirmed (deposit-paid) booking can't be self-cancelled — that needs a
	// refund path (out of scope; route to support).
	switch request.Status {
	case models.CateringStatusConfirmed, models.CateringStatusCompleted, models.CateringStatusCancelled:
		c.JSON(http.StatusBadRequest, gin.H{"error": "This request can no longer be cancelled"})
		return
	}
	now := time.Now()
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		request.Status = models.CateringStatusCancelled
		request.CancelledAt = &now
		if err := tx.Save(&request).Error; err != nil {
			return err
		}
		// Reject any still-pending quotes so chefs stop expecting a decision.
		return tx.Model(&models.CateringQuote{}).
			Where("request_id = ? AND status = ?", request.ID, models.QuoteStatusPending).
			Updates(map[string]interface{}{"status": models.QuoteStatusRejected, "rejected_at": now}).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Request cancelled", "data": request.ToResponse()})
}

// CreateDeposit creates a Razorpay order for the accepted quote's deposit, so the
// customer can pay the advance that confirms the booking (#55).
// POST /catering/requests/:id/deposit
func (h *CateringHandler) CreateDeposit(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	if config.AppConfig == nil || !config.AppConfig.CateringDepositEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Catering deposits aren't available yet"})
		return
	}
	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}
	var request models.CateringRequest
	if err := database.DB.Where("id = ? AND customer_id = ?", requestID, userID).First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catering request not found"})
		return
	}
	if request.Status != models.CateringStatusAccepted {
		c.JSON(http.StatusConflict, gin.H{"error": "Accept a quote before paying the deposit"})
		return
	}
	if request.DepositStatus == "paid" {
		c.JSON(http.StatusConflict, gin.H{"error": "Deposit already paid"})
		return
	}
	if request.DepositAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No deposit is due for this booking"})
		return
	}
	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}
	rzOrder, err := rz.CreateOrder(&services.OrderRequest{
		Amount:   services.ToPaise(request.DepositAmount),
		Currency: "INR",
		Receipt:  fmt.Sprintf("CAT-%s", request.ID.String()[:8]),
		Notes:    map[string]string{"purpose": "catering_deposit", "catering_request_id": request.ID.String()},
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not start payment"})
		return
	}
	database.DB.Model(&request).Update("razorpay_order_id", rzOrder.ID)
	c.JSON(http.StatusCreated, gin.H{
		"razorpayOrderId": rzOrder.ID,
		"razorpayKeyId":   rz.GetKeyID(),
		"amount":          rzOrder.Amount,
		"currency":        "INR",
	})
}

// VerifyDeposit confirms a captured deposit payment and moves the booking to
// confirmed. POST /catering/requests/:id/deposit/verify
func (h *CateringHandler) VerifyDeposit(c *gin.Context) {
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
	var req struct {
		RazorpayPaymentID string `json:"razorpayPaymentId" binding:"required"`
		RazorpayOrderID   string `json:"razorpayOrderId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var request models.CateringRequest
	if err := database.DB.Where("id = ? AND customer_id = ?", requestID, userID).First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catering request not found"})
		return
	}
	if request.DepositStatus == "paid" {
		c.JSON(http.StatusOK, gin.H{"message": "Deposit already confirmed", "data": request.ToResponse()})
		return
	}
	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}
	payment, err := rz.FetchPayment(req.RazorpayPaymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify payment"})
		return
	}
	if payment.Status != "captured" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment not captured"})
		return
	}
	if request.RazorpayOrderID != "" && payment.OrderID != request.RazorpayOrderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID mismatch"})
		return
	}
	now := time.Now()
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		request.Status = models.CateringStatusConfirmed
		request.DepositStatus = "paid"
		request.RazorpayPaymentID = req.RazorpayPaymentID
		request.DepositPaidAt = &now
		if err := tx.Save(&request).Error; err != nil {
			return err
		}
		// Notify the chef the booking is confirmed (best-effort via the outbox).
		var chefUserID uuid.UUID
		if request.AcceptedQuoteID != nil {
			var q models.CateringQuote
			if err := tx.Select("chef_id").First(&q, "id = ?", *request.AcceptedQuoteID).Error; err == nil {
				chefUserID = cateringChefUserID(q.ChefID)
			}
		}
		return services.EnqueueEvent(tx, services.SubjectCateringRequest, "catering.confirmed",
			chefUserID, map[string]any{
				"requestId":     request.ID.String(),
				"eventType":     request.EventType,
				"eventDate":     request.EventDate.Format("2006-01-02"),
				"depositAmount": request.DepositAmount,
			})
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment captured but confirmation failed; please contact support"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deposit confirmed", "data": request.ToResponse()})
}

// GetChefBookings returns the confirmed/completed catering bookings whose accepted
// quote belongs to the authenticated chef. GET /chef/catering/bookings
func (h *CateringHandler) GetChefBookings(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// Accepted quotes by this chef whose request is confirmed or completed.
	var quotes []models.CateringQuote
	if err := database.DB.
		Where("chef_id = ? AND status = ?", chef.ID, models.QuoteStatusAccepted).
		Find(&quotes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookings"})
		return
	}

	type booking struct {
		Request models.CateringRequestResponse `json:"request"`
		Quote   models.CateringQuoteResponse   `json:"quote"`
	}
	bookings := []booking{}
	for i := range quotes {
		var request models.CateringRequest
		if err := database.DB.First(&request, "id = ?", quotes[i].RequestID).Error; err != nil {
			continue
		}
		if request.Status != models.CateringStatusConfirmed && request.Status != models.CateringStatusCompleted {
			continue
		}
		bookings = append(bookings, booking{Request: request.ToResponse(), Quote: quotes[i].ToResponse()})
	}
	c.JSON(http.StatusOK, gin.H{"data": bookings})
}

// CompleteBooking marks a confirmed catering booking completed (chef-side, after
// the event). POST /chef/catering/requests/:id/complete
func (h *CateringHandler) CompleteBooking(c *gin.Context) {
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
	var request models.CateringRequest
	if err := database.DB.First(&request, "id = ?", requestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Catering request not found"})
		return
	}
	// The chef must own the accepted quote on this request.
	if request.AcceptedQuoteID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This booking has no accepted quote"})
		return
	}
	var quote models.CateringQuote
	if err := database.DB.First(&quote, "id = ?", *request.AcceptedQuoteID).Error; err != nil || quote.ChefID != chef.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "This booking isn't yours"})
		return
	}
	if request.Status != models.CateringStatusConfirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only a confirmed booking can be completed"})
		return
	}
	now := time.Now()
	request.Status = models.CateringStatusCompleted
	request.CompletedAt = &now
	if err := database.DB.Save(&request).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete booking"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Booking completed", "data": request.ToResponse()})
}
