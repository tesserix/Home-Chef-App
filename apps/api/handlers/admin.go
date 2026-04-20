package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type AdminHandler struct{}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// boolField returns the field name when present, "" otherwise — cheap
// helper to build a list of which fields an update actually touched.
func boolField(name string, present bool) string {
	if present {
		return name
	}
	return ""
}

// GetStats returns dashboard statistics
func (h *AdminHandler) GetStats(c *gin.Context) {
	db := database.DB
	var stats models.AdminDashboardStats

	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	lastWeek := today.AddDate(0, 0, -7)
	prevWeek := lastWeek.AddDate(0, 0, -7)

	// Use int64 intermediaries for GORM Count()
	var totalUsers, newUsersToday, totalChefs, pendingVerifications, totalOrders, ordersToday int64

	db.Model(&models.User{}).Count(&totalUsers)
	db.Model(&models.User{}).Where("created_at >= ?", today).Count(&newUsersToday)
	db.Model(&models.ChefProfile{}).Count(&totalChefs)
	db.Model(&models.ChefProfile{}).Where("is_verified = ?", false).Count(&pendingVerifications)
	db.Model(&models.Order{}).Count(&totalOrders)
	db.Model(&models.Order{}).Where("created_at >= ?", today).Count(&ordersToday)

	stats.TotalUsers = int(totalUsers)
	stats.NewUsersToday = int(newUsersToday)
	stats.TotalChefs = int(totalChefs)
	stats.PendingVerifications = int(pendingVerifications)
	stats.TotalOrders = int(totalOrders)
	stats.OrdersToday = int(ordersToday)

	// Revenue (completed orders)
	db.Model(&models.Order{}).Where("payment_status = ?", "completed").Select("COALESCE(SUM(total), 0)").Scan(&stats.Revenue)
	db.Model(&models.Order{}).Where("payment_status = ? AND created_at >= ?", "completed", today).Select("COALESCE(SUM(total), 0)").Scan(&stats.RevenueToday)

	// Orders change (this week vs last week)
	var ordersThisWeek, ordersLastWeek int64
	db.Model(&models.Order{}).Where("created_at >= ?", lastWeek).Count(&ordersThisWeek)
	db.Model(&models.Order{}).Where("created_at >= ? AND created_at < ?", prevWeek, lastWeek).Count(&ordersLastWeek)
	if ordersLastWeek > 0 {
		stats.OrdersChange = float64(ordersThisWeek-ordersLastWeek) / float64(ordersLastWeek) * 100
	}

	// Revenue change (today vs yesterday)
	var revenueYesterday float64
	db.Model(&models.Order{}).Where("payment_status = ? AND created_at >= ? AND created_at < ?", "completed", yesterday, today).Select("COALESCE(SUM(total), 0)").Scan(&revenueYesterday)
	if revenueYesterday > 0 {
		stats.RevenueChange = (stats.RevenueToday - revenueYesterday) / revenueYesterday * 100
	}

	c.JSON(http.StatusOK, stats)
}

// GetActivities returns recent platform activities
func (h *AdminHandler) GetActivities(c *gin.Context) {
	db := database.DB
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}

	type Activity struct {
		ID          string    `json:"id"`
		Type        string    `json:"type"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Timestamp   time.Time `json:"timestamp"`
	}

	var activities []Activity

	// Get recent orders as activities
	var recentOrders []models.Order
	db.Order("created_at DESC").Limit(limit / 2).Find(&recentOrders)
	for _, o := range recentOrders {
		activities = append(activities, Activity{
			ID:          o.ID.String(),
			Type:        "order",
			Title:       fmt.Sprintf("Order #%s", o.OrderNumber),
			Description: fmt.Sprintf("Status: %s - ₹%.0f", o.Status, o.Total),
			Timestamp:   o.CreatedAt,
		})
	}

	// Get recent user signups
	var recentUsers []models.User
	db.Order("created_at DESC").Limit(limit / 4).Find(&recentUsers)
	for _, u := range recentUsers {
		activities = append(activities, Activity{
			ID:          u.ID.String(),
			Type:        "user",
			Title:       fmt.Sprintf("New user: %s %s", u.FirstName, u.LastName),
			Description: u.Email,
			Timestamp:   u.CreatedAt,
		})
	}

	// Get recent chef registrations
	var recentChefs []models.ChefProfile
	db.Preload("User").Order("created_at DESC").Limit(limit / 4).Find(&recentChefs)
	for _, ch := range recentChefs {
		verifiedStr := "Unverified"
		if ch.IsVerified {
			verifiedStr = "Verified"
		}
		activities = append(activities, Activity{
			ID:          ch.ID.String(),
			Type:        "chef",
			Title:       fmt.Sprintf("Chef: %s", ch.BusinessName),
			Description: verifiedStr,
			Timestamp:   ch.CreatedAt,
		})
	}

	// Sort by timestamp descending
	for i := 0; i < len(activities); i++ {
		for j := i + 1; j < len(activities); j++ {
			if activities[j].Timestamp.After(activities[i].Timestamp) {
				activities[i], activities[j] = activities[j], activities[i]
			}
		}
	}

	if len(activities) > limit {
		activities = activities[:limit]
	}

	c.JSON(http.StatusOK, activities)
}

// GetUsers returns paginated user list with order stats
func (h *AdminHandler) GetUsers(c *gin.Context) {
	db := database.DB
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	role := c.Query("role")

	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := db.Model(&models.User{})

	if search != "" {
		query = query.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	var total int64
	query.Count(&total)

	var users []models.User
	query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users)

	// Build enriched response with order stats per user
	type UserWithStats struct {
		models.User
		TotalOrders int     `json:"totalOrders"`
		TotalSpent  float64 `json:"totalSpent"`
		LastOrderAt *string `json:"lastOrderAt,omitempty"`
	}

	var response []UserWithStats
	for _, u := range users {
		uw := UserWithStats{User: u}

		// Get order count and total spent
		var orderCount int64
		var totalSpent float64
		db.Model(&models.Order{}).Where("customer_id = ?", u.ID).Count(&orderCount)
		db.Model(&models.Order{}).Where("customer_id = ? AND payment_status = ?", u.ID, "completed").
			Select("COALESCE(SUM(total), 0)").Scan(&totalSpent)

		uw.TotalOrders = int(orderCount)
		uw.TotalSpent = totalSpent

		// Get last order date
		var lastOrder models.Order
		if err := db.Where("customer_id = ?", u.ID).Order("created_at DESC").First(&lastOrder).Error; err == nil {
			ts := lastOrder.CreatedAt.Format("2006-01-02T15:04:05Z")
			uw.LastOrderAt = &ts
		}

		response = append(response, uw)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
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

// GetUser returns a single user by ID with order stats
func (h *AdminHandler) GetUser(c *gin.Context) {
	db := database.DB
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := db.Preload("ChefProfile").Preload("CustomerProfile").First(&user, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Enrich with order stats
	type UserWithStats struct {
		models.User
		TotalOrders int     `json:"totalOrders"`
		TotalSpent  float64 `json:"totalSpent"`
		LastOrderAt *string `json:"lastOrderAt,omitempty"`
	}

	uw := UserWithStats{User: user}
	var orderCount int64
	var totalSpent float64
	db.Model(&models.Order{}).Where("customer_id = ?", user.ID).Count(&orderCount)
	db.Model(&models.Order{}).Where("customer_id = ? AND payment_status = ?", user.ID, "completed").
		Select("COALESCE(SUM(total), 0)").Scan(&totalSpent)
	uw.TotalOrders = int(orderCount)
	uw.TotalSpent = totalSpent

	var lastOrder models.Order
	if err := db.Where("customer_id = ?", user.ID).Order("created_at DESC").First(&lastOrder).Error; err == nil {
		ts := lastOrder.CreatedAt.Format("2006-01-02T15:04:05Z")
		uw.LastOrderAt = &ts
	}

	c.JSON(http.StatusOK, uw)
}

// SuspendUser deactivates a user
func (h *AdminHandler) SuspendUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	result := database.DB.Model(&models.User{}).Where("id = ?", id).Update("is_active", false)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	services.LogAudit(c, "user.suspend", "user", id.String(), nil, map[string]any{"isActive": false})
	c.JSON(http.StatusOK, gin.H{"message": "User suspended"})
}

// ActivateUser reactivates a user
func (h *AdminHandler) ActivateUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	result := database.DB.Model(&models.User{}).Where("id = ?", id).Update("is_active", true)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	services.LogAudit(c, "user.activate", "user", id.String(), nil, map[string]any{"isActive": true})
	c.JSON(http.StatusOK, gin.H{"message": "User activated"})
}

// GetChefs returns paginated chef list with filters
func (h *AdminHandler) GetChefs(c *gin.Context) {
	db := database.DB
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := db.Model(&models.ChefProfile{}).Preload("User")

	if search != "" {
		query = query.Where("business_name ILIKE ? OR cuisines::text ILIKE ?",
			"%"+search+"%", "%"+search+"%")
	}
	if status != "" {
		switch status {
		case "submitted", "pending":
			query = query.Where("is_verified = ?", false)
		case "approved":
			query = query.Where("is_verified = ?", true)
		}
	}

	var total int64
	query.Count(&total)

	var chefs []models.ChefProfile
	query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&chefs)

	// Enrich with stats
	type ChefWithStats struct {
		models.ChefProfile
		OwnerName      string  `json:"ownerName"`
		OwnerEmail     string  `json:"ownerEmail"`
		OwnerPhone     string  `json:"ownerPhone"`
		TotalOrders    int     `json:"totalOrders"`
		TotalRevenue   float64 `json:"totalRevenue"`
		MenuItemCount  int     `json:"menuItemCount"`
		DocumentCount  int     `json:"documentCount"`
		OnlineStatus   string  `json:"onlineStatus"`
	}

	var response []ChefWithStats
	for _, ch := range chefs {
		cws := ChefWithStats{ChefProfile: ch}
		cws.OwnerName = ch.User.FirstName + " " + ch.User.LastName
		cws.OwnerEmail = ch.User.Email
		cws.OwnerPhone = ch.User.Phone

		var orderCount int64
		var revenue float64
		db.Model(&models.Order{}).Where("chef_id = ?", ch.ID).Count(&orderCount)
		db.Model(&models.Order{}).Where("chef_id = ? AND payment_status = ?", ch.ID, "completed").
			Select("COALESCE(SUM(total), 0)").Scan(&revenue)
		cws.TotalOrders = int(orderCount)
		cws.TotalRevenue = revenue

		var menuCount, docCount int64
		db.Model(&models.MenuItem{}).Where("chef_id = ?", ch.ID).Count(&menuCount)
		db.Model(&models.ChefDocument{}).Where("chef_id = ?", ch.ID).Count(&docCount)
		cws.MenuItemCount = int(menuCount)
		cws.DocumentCount = int(docCount)

		if ch.AcceptingOrders && ch.IsActive {
			cws.OnlineStatus = "online"
		} else if ch.IsActive {
			cws.OnlineStatus = "away"
		} else {
			cws.OnlineStatus = "offline"
		}

		response = append(response, cws)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
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

// VerifyChef approves a chef
func (h *AdminHandler) VerifyChef(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef ID"})
		return
	}

	now := time.Now()
	result := database.DB.Model(&models.ChefProfile{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_verified": true,
		"verified_at": &now,
		"is_active":   true,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	// Update user role to chef
	var chef models.ChefProfile
	database.DB.First(&chef, "id = ?", id)
	database.DB.Model(&models.User{}).Where("id = ?", chef.UserID).Update("role", "chef")

	services.LogAudit(c, "chef.verify", "chef", id.String(), nil, map[string]any{"isVerified": true})
	c.JSON(http.StatusOK, gin.H{"message": "Chef verified"})
}

// RejectChef rejects a chef application
func (h *AdminHandler) RejectChef(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef ID"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	result := database.DB.Model(&models.ChefProfile{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_verified": false,
		"is_active":   false,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	services.LogAudit(c, "chef.reject", "chef", id.String(), nil, map[string]any{"reason": req.Reason})
	c.JSON(http.StatusOK, gin.H{"message": "Chef rejected", "reason": req.Reason})
}

// SuspendChef suspends a chef
func (h *AdminHandler) SuspendChef(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef ID"})
		return
	}

	result := database.DB.Model(&models.ChefProfile{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_active":   false,
		"is_verified": false,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	services.LogAudit(c, "chef.suspend", "chef", id.String(), nil, map[string]any{"isActive": false})
	c.JSON(http.StatusOK, gin.H{"message": "Chef suspended"})
}

// GetAllOrders returns paginated orders for admin
func (h *AdminHandler) GetAllOrders(c *gin.Context) {
	db := database.DB
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := db.Model(&models.Order{}).Preload("Customer").Preload("Chef")

	if search != "" {
		query = query.Where("order_number ILIKE ?", "%"+search+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var orders []models.Order
	query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&orders)

	// Build response with customer/chef names
	type OrderResponse struct {
		models.Order
		CustomerName string `json:"customerName"`
		ChefName     string `json:"chefName"`
		ItemCount    int    `json:"itemCount"`
	}

	var response []OrderResponse
	for _, o := range orders {
		name := ""
		if o.Customer.FirstName != "" {
			name = o.Customer.FirstName + " " + o.Customer.LastName
		} else {
			name = o.Customer.Email
		}
		chefName := ""
		if o.Chef.BusinessName != "" {
			chefName = o.Chef.BusinessName
		}

		var itemCount int64
		db.Model(&models.OrderItem{}).Where("order_id = ?", o.ID).Count(&itemCount)

		response = append(response, OrderResponse{
			Order:        o,
			CustomerName: name,
			ChefName:     chefName,
			ItemCount:    int(itemCount),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": response,
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

// GetOrderDetails returns a single order with all details. Customer and
// Chef are NOT serialized inline (they're json:"-" on the Order model so
// admin views can't accidentally leak the customer's 2FA state or auth
// provider); a hand-picked subset is added next to the order body.
func (h *AdminHandler) GetOrderDetails(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := database.DB.Preload("Customer").Preload("Chef").Preload("Items").First(&order, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	customer := gin.H{
		"id":        order.Customer.ID,
		"name":      strings.TrimSpace(order.Customer.FirstName + " " + order.Customer.LastName),
		"email":     order.Customer.Email,
		"phone":     order.Customer.Phone,
		"createdAt": order.Customer.CreatedAt,
	}
	chef := gin.H{
		"id":           order.Chef.ID,
		"businessName": order.Chef.BusinessName,
		"city":         order.Chef.City,
	}

	c.JSON(http.StatusOK, gin.H{
		"order":    order,
		"customer": customer,
		"chef":     chef,
	})
}

// GetAnalytics returns platform analytics
func (h *AdminHandler) GetAnalytics(c *gin.Context) {
	db := database.DB
	var analytics models.AdminAnalytics

	today := time.Now().Truncate(24 * time.Hour)
	thirtyDaysAgo := today.AddDate(0, 0, -30)

	// Overview
	db.Model(&models.Order{}).Where("payment_status = ?", "completed").Select("COALESCE(SUM(total), 0)").Scan(&analytics.Overview.TotalRevenue)

	var totalOrders, activeUsers int64
	db.Model(&models.Order{}).Count(&totalOrders)
	analytics.Overview.TotalOrders = int(totalOrders)

	if analytics.Overview.TotalOrders > 0 {
		analytics.Overview.AvgOrderValue = analytics.Overview.TotalRevenue / float64(analytics.Overview.TotalOrders)
	}

	db.Model(&models.User{}).Where("last_login_at >= ?", thirtyDaysAgo).Count(&activeUsers)
	analytics.Overview.ActiveUsers = int(activeUsers)

	// Orders by status
	analytics.OrdersByStatus = make(map[string]int)
	type StatusCount struct {
		Status string
		Count  int
	}
	var statusCounts []StatusCount
	db.Model(&models.Order{}).Select("status, count(*) as count").Group("status").Scan(&statusCounts)
	for _, sc := range statusCounts {
		analytics.OrdersByStatus[sc.Status] = sc.Count
	}

	c.JSON(http.StatusOK, analytics)
}

// GetSettings returns platform settings
func (h *AdminHandler) GetSettings(c *gin.Context) {
	var settings []models.PlatformSettings
	database.DB.Find(&settings)
	c.JSON(http.StatusOK, settings)
}

// GetPaymentGatewayStatus reports whether Razorpay is configured and reachable.
// Credentials come straight from the live client (which reads GCP Secret
// Manager at runtime) — there's no hidden env/config path that can show stale
// or placeholder values here.
func (h *AdminHandler) GetPaymentGatewayStatus(c *gin.Context) {
	client := services.GetRazorpay()

	webhookURL := "https://api.fe3dr.com/webhooks/razorpay"

	if client == nil {
		c.JSON(http.StatusOK, gin.H{
			"configured":       false,
			"mode":             "unknown",
			"webhookUrl":       webhookURL,
			"webhookSecretSet": false,
			"keyPrefix":        "",
			"error":            "Razorpay is not configured. Enter your keys to set up the gateway.",
		})
		return
	}

	keyID := client.GetKeyID()
	mode := "unknown"
	if strings.HasPrefix(keyID, "rzp_test_") {
		mode = "test"
	} else if strings.HasPrefix(keyID, "rzp_live_") {
		mode = "live"
	}

	// Key ID is a publishable identifier — safe to surface a short prefix.
	keyPrefix := keyID
	if len(keyID) > 12 {
		keyPrefix = keyID[:12] + "..."
	}

	// Validate credentials by making a lightweight call to Razorpay.
	healthErr := ""
	configured := true
	if err := client.HealthCheck(); err != nil {
		healthErr = err.Error()
		configured = false
	}

	c.JSON(http.StatusOK, gin.H{
		"configured":       configured,
		"mode":             mode,
		"webhookUrl":       webhookURL,
		"webhookSecretSet": client.HasWebhookSecret(),
		"keyPrefix":        keyPrefix,
		"error":            healthErr,
	})
}

// UpdatePaymentGatewayKeys writes the Razorpay credentials to GCP Secret
// Manager, invalidates the in-memory cache so the next use picks them up, and
// runs an immediate health check so the admin sees pass/fail right in the UI
// response (no second round trip needed).
//
// Secrets are never persisted to the DB or to config. The app reads them
// dynamically from Secret Manager on demand.
func (h *AdminHandler) UpdatePaymentGatewayKeys(c *gin.Context) {
	var req struct {
		KeyID         string `json:"keyId"`
		KeySecret     string `json:"keySecret"`
		WebhookSecret string `json:"webhookSecret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.KeyID == "" && req.KeySecret == "" && req.WebhookSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one field is required"})
		return
	}

	ctx := c.Request.Context()

	// Writing the key ID and key secret together is the only sensible mode —
	// a mismatched pair guarantees a 401 from Razorpay. Require both or neither.
	if (req.KeyID == "") != (req.KeySecret == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "keyId and keySecret must be provided together"})
		return
	}

	// Persist each provided value to GCP Secret Manager. StorePlatformSecret
	// creates the secret on first call (idempotent) and appends a new version
	// on subsequent calls, so the most recent version is always "latest".
	secretMap := map[string]string{
		"prod-razorpay-key-id":         req.KeyID,
		"prod-razorpay-key-secret":     req.KeySecret,
		"prod-razorpay-webhook-secret": req.WebhookSecret,
	}
	for secretName, value := range secretMap {
		if value == "" {
			continue
		}
		if err := services.StorePlatformSecret(ctx, secretName, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to store %s: %v", secretName, err)})
			return
		}
	}

	// Drop the cached client so the next GetRazorpay() rereads from SM.
	services.InvalidateRazorpay()
	services.LogAudit(c, "payment.keys.update", "payment_gateway", "razorpay", nil, map[string]any{
		"updatedFields": []string{
			boolField("keyId", req.KeyID != ""),
			boolField("keySecret", req.KeySecret != ""),
			boolField("webhookSecret", req.WebhookSecret != ""),
		},
	})

	// Validate by actually calling Razorpay. If the keys are wrong, surface
	// the exact error to the UI so the admin can fix it immediately.
	client := services.GetRazorpay()
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Saved to Secret Manager, but client failed to initialize"})
		return
	}
	if err := client.HealthCheck(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message":   "Keys saved, but validation failed",
			"testError": err.Error(),
			"verified":  false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Payment gateway keys saved and verified",
		"verified": true,
	})
}

// GetStripeGatewayStatus reports whether Stripe is configured and reachable.
// Mirrors GetPaymentGatewayStatus (the Razorpay one) so the admin UI can
// render a parallel card for the Stripe provider.
func (h *AdminHandler) GetStripeGatewayStatus(c *gin.Context) {
	client := services.GetStripe()

	webhookURL := "https://api.fe3dr.com/webhooks/stripe"

	if client == nil {
		c.JSON(http.StatusOK, gin.H{
			"configured":        false,
			"mode":              "unknown",
			"webhookUrl":        webhookURL,
			"webhookSecretSet":  false,
			"keyPrefix":         "",
			"publishableKeySet": false,
			"error":             "Stripe is not configured. Enter your keys to enable international payouts.",
		})
		return
	}

	secretKey := client.GetSecretKeyID()
	mode := "unknown"
	if strings.HasPrefix(secretKey, "sk_test_") {
		mode = "test"
	} else if strings.HasPrefix(secretKey, "sk_live_") {
		mode = "live"
	}

	keyPrefix := secretKey
	if len(secretKey) > 12 {
		keyPrefix = secretKey[:12] + "..."
	}

	healthErr := ""
	configured := true
	if err := client.HealthCheck(); err != nil {
		healthErr = err.Error()
		configured = false
	}

	c.JSON(http.StatusOK, gin.H{
		"configured":        configured,
		"mode":              mode,
		"webhookUrl":        webhookURL,
		"webhookSecretSet":  client.HasWebhookSecret(),
		"keyPrefix":         keyPrefix,
		"publishableKeySet": client.GetPublishableKey() != "",
		"error":             healthErr,
	})
}

// UpdateStripeGatewayKeys persists the Stripe credentials to GCP Secret
// Manager, invalidates the cached client, and runs an immediate health
// check so the admin sees pass/fail in one response. Same contract as
// UpdatePaymentGatewayKeys (Razorpay).
func (h *AdminHandler) UpdateStripeGatewayKeys(c *gin.Context) {
	var req struct {
		SecretKey      string `json:"secretKey"`
		PublishableKey string `json:"publishableKey"`
		WebhookSecret  string `json:"webhookSecret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SecretKey == "" && req.PublishableKey == "" && req.WebhookSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one field is required"})
		return
	}

	ctx := c.Request.Context()

	// The secret key alone is enough to authenticate API calls, but the
	// publishable key is what the frontend loads Elements with — callers
	// that set one usually set the other, but we don't enforce it here.
	secretMap := map[string]string{
		"prod-homechef-stripe-secret-key":      req.SecretKey,
		"prod-homechef-stripe-publishable-key": req.PublishableKey,
		"prod-homechef-stripe-webhook-secret":  req.WebhookSecret,
	}
	for secretName, value := range secretMap {
		if value == "" {
			continue
		}
		if err := services.StorePlatformSecret(ctx, secretName, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to store %s: %v", secretName, err)})
			return
		}
	}

	services.InvalidateStripe()
	services.LogAudit(c, "payment.keys.update", "payment_gateway", "stripe", nil, map[string]any{
		"updatedFields": []string{
			boolField("secretKey", req.SecretKey != ""),
			boolField("publishableKey", req.PublishableKey != ""),
			boolField("webhookSecret", req.WebhookSecret != ""),
		},
	})

	client := services.GetStripe()
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Saved to Secret Manager, but client failed to initialize"})
		return
	}
	if err := client.HealthCheck(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message":   "Keys saved, but validation failed",
			"testError": err.Error(),
			"verified":  false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Stripe gateway keys saved and verified",
		"verified": true,
	})
}

// UpdateSettings updates platform settings
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("userID")
	uid, _ := userID.(uuid.UUID)

	result := database.DB.Model(&models.PlatformSettings{}).
		Where("key = ?", req.Key).
		Updates(map[string]interface{}{
			"value":      req.Value,
			"updated_by": uid,
		})

	if result.RowsAffected == 0 {
		setting := models.PlatformSettings{
			Key:       req.Key,
			Value:     req.Value,
			UpdatedBy: &uid,
		}
		database.DB.Create(&setting)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Setting updated"})
}
