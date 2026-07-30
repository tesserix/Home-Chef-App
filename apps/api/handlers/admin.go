package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
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

// GetFSSAILockedChefs lists India chefs whose FSSAI food-safety licence has
// lapsed — and who are therefore locked out of new orders and payouts (#32) —
// so ops can follow up. Reuses services.IsChefFSSAIExpired, so a verified
// renewal is correctly excluded and this view never disagrees with enforcement.
//
// Candidates (a verified FSSAI doc already past expiry) are sorted into two
// buckets: genuinely "locked", and "overridden" — chefs an admin has granted a
// time-boxed reprieve (#93). Overridden chefs are surfaced separately (with the
// reason + expiry) so ops can review or revoke a reprieve, rather than silently
// dropping them as if they had renewed. It also reports how many chefs have a
// verified FSSAI doc with no recorded expiry (the backfill target).
// GET /admin/chefs/fssai-locked
func (h *AdminHandler) GetFSSAILockedChefs(c *gin.Context) {
	cutoff := time.Now().AddDate(0, 0, -1)

	// Candidate chefs: those with a verified FSSAI doc already past expiry.
	var chefIDs []uuid.UUID
	database.DB.Model(&models.ChefDocument{}).
		Distinct("chef_id").
		Where("type = ? AND status = ? AND expiry_date IS NOT NULL AND expiry_date < ?",
			models.DocFSSAILicense, models.DocStatusVerified, cutoff).
		Pluck("chef_id", &chefIDs)

	type lockedChef struct {
		ChefID          uuid.UUID  `json:"chefId"`
		UserID          uuid.UUID  `json:"userId"`
		BusinessName    string     `json:"businessName"`
		FSSAIExpiry     *time.Time `json:"fssaiExpiry"`
		DaysSinceExpiry int        `json:"daysSinceExpiry"`
		OverrideUntil   *time.Time `json:"overrideUntil,omitempty"`
		OverrideReason  string     `json:"overrideReason,omitempty"`
		OverrideBy      *uuid.UUID `json:"overrideBy,omitempty"`
	}

	locked := make([]lockedChef, 0, len(chefIDs))
	overridden := make([]lockedChef, 0)
	for _, chefID := range chefIDs {
		var chef models.ChefProfile
		if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
			continue
		}
		var doc models.ChefDocument
		database.DB.
			Where("chef_id = ? AND type = ? AND status = ? AND expiry_date IS NOT NULL",
				chefID, models.DocFSSAILicense, models.DocStatusVerified).
			Order("expiry_date DESC").First(&doc)
		days := 0
		if doc.ExpiryDate != nil {
			days = int(time.Since(*doc.ExpiryDate).Hours() / 24)
		}
		row := lockedChef{
			ChefID:          chef.ID,
			UserID:          chef.UserID,
			BusinessName:    chef.BusinessName,
			FSSAIExpiry:     doc.ExpiryDate,
			DaysSinceExpiry: days,
		}

		// Active admin override → reprieved, not locked. Surface separately.
		if chef.FSSAIOverrideUntil != nil && time.Now().Before(*chef.FSSAIOverrideUntil) {
			row.OverrideUntil = chef.FSSAIOverrideUntil
			row.OverrideReason = chef.FSSAIOverrideReason
			row.OverrideBy = chef.FSSAIOverrideBy
			overridden = append(overridden, row)
			continue
		}
		if !services.IsChefFSSAIExpired(&chef) {
			continue // renewed / not actually locked
		}
		locked = append(locked, row)
	}

	// Backfill target: India chefs with a verified FSSAI doc but NULL expiry
	// (legacy uploads from before expiry capture). Ops can prompt these chefs to
	// confirm their licence via POST /admin/fssai-expiry-backfill.
	var missingExpiry int64
	database.DB.Model(&models.ChefDocument{}).
		Joins("JOIN chef_profiles ON chef_profiles.id = chef_documents.chef_id").
		Where("chef_documents.type = ? AND chef_documents.status = ? AND chef_documents.expiry_date IS NULL AND chef_profiles.payout_country = ?",
			models.DocFSSAILicense, models.DocStatusVerified, "IN").
		Distinct("chef_documents.chef_id").
		Count(&missingExpiry)

	c.JSON(http.StatusOK, gin.H{
		"locked":             locked,
		"overridden":         overridden,
		"lockedCount":        len(locked),
		"overriddenCount":    len(overridden),
		"missingExpiryCount": missingExpiry,
	})
}

type fssaiOverrideRequest struct {
	Reason string `json:"reason" binding:"required,min=10,max=500"`
	Days   int    `json:"days" binding:"required,min=1,max=30"`
}

// OverrideFSSAILock grants a time-boxed, reason-logged reprieve from the FSSAI
// expiry lockout for one chef (#93). For genuine edge cases only — e.g. a
// government renewal backlog where the chef's renewal is filed but not yet
// processed — never a routine way to ship food on a lapsed licence; hence the
// mandatory reason and the hard 30-day cap. Fully audited (actor, reason,
// window). POST /admin/chefs/:id/fssai-override
func (h *AdminHandler) OverrideFSSAILock(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var req fssaiOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A reason (10–500 chars) and a duration of 1–30 days are required"})
		return
	}

	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	old := map[string]any{
		"overrideUntil":  chef.FSSAIOverrideUntil,
		"overrideReason": chef.FSSAIOverrideReason,
	}
	until := time.Now().AddDate(0, 0, req.Days)
	updates := map[string]any{
		"fssai_override_until":  until,
		"fssai_override_reason": req.Reason,
	}
	if v, ok := c.Get("userID"); ok {
		if uid, ok := v.(uuid.UUID); ok {
			updates["fssai_override_by"] = uid
		}
	}
	if err := database.DB.Model(&chef).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply override"})
		return
	}

	middleware.RecordFSSAILockout("admin_override_granted")
	services.LogAudit(c, "chef.fssai.override.grant", "chef", chefID.String(), old, map[string]any{
		"overrideUntil":  until,
		"overrideReason": req.Reason,
		"days":           req.Days,
	})

	c.JSON(http.StatusOK, gin.H{
		"chefId":         chefID,
		"overrideUntil":  until,
		"overrideReason": req.Reason,
	})
}

// ClearFSSAILockOverride revokes an active FSSAI override immediately, so the
// expiry lockout re-applies at once (#93). Audited.
// DELETE /admin/chefs/:id/fssai-override
func (h *AdminHandler) ClearFSSAILockOverride(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	old := map[string]any{
		"overrideUntil":  chef.FSSAIOverrideUntil,
		"overrideReason": chef.FSSAIOverrideReason,
	}
	// Map-based Updates so the nil values are written as NULL (a struct update
	// would skip zero values and leave the override in place).
	if err := database.DB.Model(&chef).Updates(map[string]any{
		"fssai_override_until":  nil,
		"fssai_override_reason": "",
		"fssai_override_by":     nil,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear override"})
		return
	}
	services.LogAudit(c, "chef.fssai.override.clear", "chef", chefID.String(), old, nil)
	c.JSON(http.StatusOK, gin.H{"chefId": chefID, "cleared": true})
}

// FSSAIExpiryBackfill surfaces (GET) or notifies (POST) India chefs who have a
// verified FSSAI document on file with NO recorded expiry date — legacy uploads
// from before expiry capture (#93). The one-time prompt asks them to confirm
// their licence expiry so the lockout can protect customers going forward. GET
// is a safe dry run (count + list); POST sends the "confirm your licence" push
// and is audited.
//
//	GET  /admin/fssai-expiry-backfill  → dry run: count + chef list
//	POST /admin/fssai-expiry-backfill  → send the confirm-licence push
func (h *AdminHandler) FSSAIExpiryBackfill(c *gin.Context) {
	var chefIDs []uuid.UUID
	database.DB.Model(&models.ChefDocument{}).
		Joins("JOIN chef_profiles ON chef_profiles.id = chef_documents.chef_id").
		Where("chef_documents.type = ? AND chef_documents.status = ? AND chef_documents.expiry_date IS NULL AND chef_profiles.payout_country = ?",
			models.DocFSSAILicense, models.DocStatusVerified, "IN").
		Distinct().
		Pluck("chef_documents.chef_id", &chefIDs)

	type backfillChef struct {
		ChefID       uuid.UUID `json:"chefId"`
		UserID       uuid.UUID `json:"userId"`
		BusinessName string    `json:"businessName"`
	}
	rows := make([]backfillChef, 0, len(chefIDs))
	execute := c.Request.Method == http.MethodPost
	notified := 0
	for _, chefID := range chefIDs {
		var chef models.ChefProfile
		if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
			continue
		}
		rows = append(rows, backfillChef{ChefID: chef.ID, UserID: chef.UserID, BusinessName: chef.BusinessName})
		if execute {
			if err := services.SendFSSAIConfirmLicencePush(chef.UserID); err != nil {
				log.Printf("fssai-backfill: confirm push failed chef=%s: %v", chefID, err)
				continue
			}
			notified++
		}
	}

	if execute {
		services.LogAudit(c, "chef.fssai.expiry_backfill", "chef", "", nil, map[string]any{
			"candidates": len(rows),
			"notified":   notified,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"count":    len(rows),
		"chefs":    rows,
		"executed": execute,
		"notified": notified,
	})
}

// GetStats returns dashboard statistics
func (h *AdminHandler) GetStats(c *gin.Context) {
	db := database.DB
	// Every order figure below is scoped to the console's active Live/Test
	// toggle, so a sandbox order can never move a real revenue number — and an
	// admin debugging in Test sees the sandbox's own figures rather than zeroes.
	orders := func() *gorm.DB { return db.Model(&models.Order{}).Scopes(adminModeScope(c)) }
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
	orders().Count(&totalOrders)
	orders().Where("created_at >= ?", today).Count(&ordersToday)

	stats.TotalUsers = int(totalUsers)
	stats.NewUsersToday = int(newUsersToday)
	stats.TotalChefs = int(totalChefs)
	stats.PendingVerifications = int(pendingVerifications)
	stats.TotalOrders = int(totalOrders)
	stats.OrdersToday = int(ordersToday)

	// Revenue (completed orders)
	orders().Where("payment_status = ?", "completed").Select("COALESCE(SUM(total), 0)").Scan(&stats.Revenue)
	orders().Where("payment_status = ? AND created_at >= ?", "completed", today).Select("COALESCE(SUM(total), 0)").Scan(&stats.RevenueToday)

	// Orders change (this week vs last week)
	var ordersThisWeek, ordersLastWeek int64
	orders().Where("created_at >= ?", lastWeek).Count(&ordersThisWeek)
	orders().Where("created_at >= ? AND created_at < ?", prevWeek, lastWeek).Count(&ordersLastWeek)
	if ordersLastWeek > 0 {
		stats.OrdersChange = float64(ordersThisWeek-ordersLastWeek) / float64(ordersLastWeek) * 100
	}

	// Revenue change (today vs yesterday)
	var revenueYesterday float64
	orders().Where("payment_status = ? AND created_at >= ? AND created_at < ?", "completed", yesterday, today).Select("COALESCE(SUM(total), 0)").Scan(&revenueYesterday)
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

	// Initialised, not declared nil: a nil slice serialises as JSON `null`
	// rather than `[]`, and every client that does `data ?? []` or `.map()` on
	// the result breaks on an account with no activity yet.
	activities := []Activity{}

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

	// Initialised, not nil: a nil slice marshals to `null`, so a page with no
	// rows would send {"data": null} and break any client that maps over it.
	response := []UserWithStats{}
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

	// Admins see every kitchen regardless of mode — they are the people who need
	// to find a sandbox kitchen — but can filter to one world when they want it.
	// Deliberately NOT tied to the console's global toggle: the chef list is how
	// you locate a kitchen in order to flip it, so hiding the other side would
	// make the flip action unreachable.
	if mode := c.Query("mode"); mode != "" {
		query = query.Where("mode = ?", models.NormalizeMode(mode))
	}

	if search != "" {
		query = query.Where("business_name ILIKE ? OR cuisines::text ILIKE ?",
			"%"+search+"%", "%"+search+"%")
	}
	if status != "" {
		switch status {
		case "submitted", "pending":
			query = query.Where("is_verified = ?", false)
		case "approved", "verified":
			query = query.Where("is_verified = ? AND is_active = ?", true, true)
		case "suspended":
			query = query.Where("is_verified = ? AND is_active = ?", true, false)
		}
	}

	var total int64
	query.Count(&total)

	var chefs []models.ChefProfile
	query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&chefs)

	// Enrich with stats
	type ChefWithStats struct {
		models.ChefProfile
		// Admin UI reads `isVerified`; the embedded model only marshals `verified`.
		IsVerifiedAlias bool    `json:"isVerified"`
		OwnerName       string  `json:"ownerName"`
		OwnerEmail      string  `json:"ownerEmail"`
		OwnerPhone      string  `json:"ownerPhone"`
		TotalOrders     int     `json:"totalOrders"`
		TotalRevenue    float64 `json:"totalRevenue"`
		MenuItemCount   int     `json:"menuItemCount"`
		DocumentCount   int     `json:"documentCount"`
		OnlineStatus    string  `json:"onlineStatus"`
	}

	// Initialised, not nil: a nil slice marshals to `null`, so a page with no
	// rows would send {"data": null} and break any client that maps over it.
	response := []ChefWithStats{}
	for _, ch := range chefs {
		cws := ChefWithStats{ChefProfile: ch}
		cws.IsVerifiedAlias = ch.IsVerified
		cws.OwnerName = ch.User.FirstName + " " + ch.User.LastName
		cws.OwnerEmail = ch.User.Email
		cws.OwnerPhone = ch.User.Phone

		// Every per-kitchen figure is scoped to the world that kitchen is
		// currently in. Unscoped, a sandboxed kitchen shows its live and test rows
		// SUMMED — 1 real dish plus 1 cloned dish reading as "2 items", and, once
		// sandbox orders exist, real revenue blended with fake. That is precisely
		// the mixing this feature exists to prevent, and it is most misleading on
		// the very screen an admin uses to decide whether to flip a kitchen.
		modeScoped := services.ChefOwnModeScope(ch.ID)

		var orderCount int64
		var revenue float64
		db.Model(&models.Order{}).Where("chef_id = ?", ch.ID).Scopes(modeScoped).Count(&orderCount)
		db.Model(&models.Order{}).Where("chef_id = ? AND payment_status = ?", ch.ID, "completed").
			Scopes(modeScoped).Select("COALESCE(SUM(total), 0)").Scan(&revenue)
		cws.TotalOrders = int(orderCount)
		cws.TotalRevenue = revenue

		var menuCount, docCount int64
		db.Model(&models.MenuItem{}).Where("chef_id = ?", ch.ID).Scopes(modeScoped).Count(&menuCount)
		// Documents are compliance artefacts of the REAL kitchen (FSSAI, ID proof)
		// and are deliberately not partitioned — the same licence applies in both
		// worlds — so this count stays unscoped.
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

	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	// A chef offering neither pickup nor self-delivery is unorderable — refuse
	// to activate it so verification never produces an active/discoverable
	// chef with no fulfillment method (mirrors the guard in UpdateChefProfile).
	if !chef.OffersPickup && !chef.OffersSelfDelivery {
		// Tell the chef what's blocking them — from the vendor side a "Pending"
		// kitchen with no reason is a dead end. New applications collect this in
		// onboarding, so this mainly reaches chefs who onboarded before that.
		_ = services.SendPushNotification(chef.UserID,
			"One step to go live",
			"Choose how customers get their food — offer pickup or delivery in your kitchen settings — so we can activate your kitchen.",
			map[string]string{"type": "chef_action_required", "reason": "fulfillment_required"})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chef must offer pickup or delivery before it can be activated"})
		return
	}

	now := time.Now()
	result := database.DB.Model(&models.ChefProfile{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_verified": true,
		"verified_at": &now,
		"is_active":   true,
		// Kitchen starts CLOSED on approval: the chef finishes setup (menu, photos,
		// hours) then opens it themselves, or lets their schedule auto-open it.
		// Going live open the instant an admin approves risks orders before the
		// kitchen is actually ready.
		"accepting_orders": false,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	// Update user role to chef
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
		"is_active": false,
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	services.LogAudit(c, "chef.suspend", "chef", id.String(), nil, map[string]any{"isActive": false})
	c.JSON(http.StatusOK, gin.H{"message": "Chef suspended"})
}

// GetChefDocuments returns a chef's compliance documents (viewable/downloadable
// URLs, signed for private docs) plus the kitchen photos/video, for admin review.
// GET /admin/chefs/:id/documents
func (h *AdminHandler) GetChefDocuments(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef ID"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	var docs []models.ChefDocument
	database.DB.Where("chef_id = ?", id).Order("created_at DESC").Find(&docs)
	out := make([]models.ChefDocumentResponse, len(docs))
	signingFailed := 0
	for i, doc := range docs {
		resp := doc.ToResponse()
		if models.IsPrivateDoc(doc.Type) {
			url, err := services.GenerateSignedURL(c.Request.Context(), doc.FilePath, 15*time.Minute)
			if err != nil {
				// Previously this error was discarded, leaving FileURL as the raw
				// private-bucket path — which the admin's browser cannot fetch. The
				// reviewer saw a broken image with no explanation and nothing was
				// logged, so a signing misconfiguration looked like a UI bug.
				// Blank the URL so the client can render "unavailable" honestly,
				// and say loudly why in the logs.
				signingFailed++
				log.Printf("admin: signed URL failed for chef=%s doc=%s type=%s: %v "+
					"(the API service account needs roles/iam.serviceAccountTokenCreator "+
					"on itself for signBlob under Workload Identity)",
					id, doc.ID, doc.Type, err)
				resp.FileURL = ""
			} else {
				resp.FileURL = url
			}
		} else {
			resp.FileURL = doc.FilePath
		}
		out[i] = resp
	}

	// Surfaced so the admin UI can tell the reviewer the documents exist but
	// could not be signed, rather than implying the chef never uploaded them.
	c.JSON(http.StatusOK, gin.H{
		"documents":         out,
		"kitchenPhotos":     []string(chef.KitchenPhotos),
		"signedUrlFailures": signingFailed,
	})
}

// VerifyChefDocument marks a single document verified or rejected. The vendor
// sees the new status on their next /chef/documents fetch and gets a push.
// PUT /admin/documents/:docId/verify  body: {verified: bool, reason?: string}
func (h *AdminHandler) VerifyChefDocument(c *gin.Context) {
	docID, err := uuid.Parse(c.Param("docId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}
	var req struct {
		Verified bool   `json:"verified"`
		Reason   string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	var doc models.ChefDocument
	if err := database.DB.First(&doc, "id = ?", docID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	status := models.DocStatusVerified
	reason := ""
	if !req.Verified {
		status = models.DocStatusRejected
		reason = req.Reason
	}
	database.DB.Model(&models.ChefDocument{}).Where("id = ?", docID).
		Updates(map[string]interface{}{"status": status, "rejection_reason": reason})

	var chef models.ChefProfile
	if database.DB.Select("user_id").First(&chef, "id = ?", doc.ChefID).Error == nil {
		title := "Document verified"
		body := fmt.Sprintf("Your %s was verified.", strings.ReplaceAll(string(doc.Type), "_", " "))
		if !req.Verified {
			title = "Document needs attention"
			body = fmt.Sprintf("Your %s was rejected. Please re-upload.", strings.ReplaceAll(string(doc.Type), "_", " "))
		}
		_ = services.SendPushNotification(chef.UserID, title, body,
			map[string]string{"type": "document_reviewed", "docId": docID.String(), "status": string(status)})
	}

	services.LogAudit(c, "chef.document.review", "chef_document", docID.String(), nil, map[string]any{"status": status})
	c.JSON(http.StatusOK, gin.H{"message": "Document " + string(status), "status": status})
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

	// Initialised, not nil: a nil slice marshals to `null`, so a page with no
	// rows would send {"data": null} and break any client that maps over it.
	response := []OrderResponse{}
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

	// Same Live/Test scoping as the dashboard: analytics must never blend
	// sandbox orders into real revenue, and must show the sandbox's own numbers
	// when the console is in Test.
	orders := func() *gorm.DB { return db.Model(&models.Order{}).Scopes(adminModeScope(c)) }

	// Overview
	orders().Where("payment_status = ?", "completed").Select("COALESCE(SUM(total), 0)").Scan(&analytics.Overview.TotalRevenue)

	var totalOrders, activeUsers int64
	orders().Count(&totalOrders)
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
	// Which credential slot the admin is asking about. Absent means live, so the
	// pre-existing admin UI keeps working through the deploy window.
	slot := models.NormalizeMode(c.Query("mode"))
	client := services.GetRazorpayFor(slot)

	webhookURL := "https://api.fe3dr.com/webhooks/razorpay"

	if client == nil {
		c.JSON(http.StatusOK, gin.H{
			"configured":       false,
			"slot":             slot,
			"mode":             "unknown",
			"webhookUrl":       webhookURL,
			"webhookSecretSet": false,
			"keyPrefix":        "",
			"slotWarning":      "",
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
		"slot":             slot,
		"mode":             mode,
		"webhookUrl":       webhookURL,
		"webhookSecretSet": client.HasWebhookSecret(),
		"keyPrefix":        keyPrefix,
		"slotWarning":      razorpaySlotWarning(slot, keyID),
		"error":            healthErr,
	})
}

// razorpaySlotWarning flags a credential slot holding a key of the wrong kind.
//
// Deliberately a WARNING and not a hard error: the live slot legitimately holds
// a test key until a real live key is issued, and refusing to save that would
// make the interim state unreachable. The banner stays up until it's fixed.
func razorpaySlotWarning(slot, keyID string) string {
	switch {
	case keyID == "":
		return ""
	case !models.IsTestMode(slot) && strings.HasPrefix(keyID, "rzp_test_"):
		return "The Live slot is holding a TEST key — no real payment will be captured until a live key is entered."
	case models.IsTestMode(slot) && strings.HasPrefix(keyID, "rzp_live_"):
		return "The Test slot is holding a LIVE key — sandbox orders would charge real cards. Replace it before using test mode."
	default:
		return ""
	}
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
		// Mode picks the credential slot: "live" (default) or "test".
		Mode string `json:"mode"`
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
	slot := models.NormalizeMode(req.Mode)
	idName, secretName, webhookName := services.RazorpaySecretNames(slot)
	secretMap := map[string]string{
		idName:      req.KeyID,
		secretName:  req.KeySecret,
		webhookName: req.WebhookSecret,
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

	// Drop only THIS slot's cached client, so saving test keys can't knock a
	// healthy live gateway offline.
	services.InvalidateRazorpayFor(slot)
	services.LogAudit(c, "payment.keys.update", "payment_gateway", "razorpay", nil, map[string]any{
		"updatedFields": []string{
			boolField("keyId", req.KeyID != ""),
			boolField("keySecret", req.KeySecret != ""),
			boolField("webhookSecret", req.WebhookSecret != ""),
		},
	})

	// Validate by actually calling Razorpay. If the keys are wrong, surface
	// the exact error to the UI so the admin can fix it immediately.
	client := services.GetRazorpayFor(slot)
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Saved to Secret Manager, but client failed to initialize"})
		return
	}
	warning := razorpaySlotWarning(slot, client.GetKeyID())
	if err := client.HealthCheck(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message":     "Keys saved, but validation failed",
			"testError":   err.Error(),
			"slot":        slot,
			"slotWarning": warning,
			"verified":    false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Payment gateway keys saved and verified",
		"slot":        slot,
		"slotWarning": warning,
		"verified":    true,
	})
}

// GetCashfreeGatewayStatus reports whether one Cashfree credential slot is
// configured and reachable.
//
// Mirrors GetPaymentGatewayStatus (the Razorpay one) including the ?mode=
// slot selector, so the admin UI renders a parallel card with the same
// live/test toggle. The two gateways' slots are wholly independent: a broken
// test slot says nothing about live, and vice versa.
//
// GET /admin/payment-gateway/cashfree/status?mode=live|test
func (h *AdminHandler) GetCashfreeGatewayStatus(c *gin.Context) {
	slot := models.NormalizeMode(c.Query("mode"))
	client := services.GetCashfreeFor(slot)

	// One dashboard-configured URL per environment, and it must be the /api form:
	// publicly only /api and /ws are routed to this service, so a root /webhooks/*
	// path falls through to the web frontend and 404s.
	webhookURL := "https://api.fe3dr.com/api/webhooks/cashfree"

	if client == nil {
		c.JSON(http.StatusOK, gin.H{
			"configured":       false,
			"slot":             slot,
			"mode":             slot,
			"environment":      "unknown",
			"webhookUrl":       webhookURL,
			"webhookSecretSet": false,
			"keyPrefix":        "",
			"error":            fmt.Sprintf("Cashfree %s slot is not configured. Enter the App ID and Secret Key to enable it.", slot),
		})
		return
	}

	appID := client.GetAppID()
	keyPrefix := appID
	if len(appID) > 12 {
		keyPrefix = appID[:12] + "..."
	}

	// Environment comes from the resolved HOST, not from the slot name — that is
	// the only thing that determines whether real money moves. A test slot
	// pointing at production would be reported as PRODUCTION here, which is
	// exactly the alarm an operator needs.
	environment := "production"
	if client.IsSandbox() {
		environment = "sandbox"
	}

	healthErr := ""
	configured := true
	if err := client.HealthCheck(); err != nil {
		healthErr = err.Error()
		configured = false
	}

	c.JSON(http.StatusOK, gin.H{
		"configured":       configured,
		"slot":             slot,
		"mode":             slot,
		"environment":      environment,
		"webhookUrl":       webhookURL,
		"webhookSecretSet": client.HasWebhookSecret(),
		"keyPrefix":        keyPrefix,
		"slotWarning":      cashfreeSlotWarning(slot, environment, client.GetAppID()),
		"error":            healthErr,
	})
}

// cashfreeSlotWarning flags a slot whose resolved environment contradicts its
// name.
//
// With Razorpay this can only be detected from a key prefix; with Cashfree the
// environment IS the hostname, so a mismatch is structurally impossible unless
// someone injects a client — which means in practice this warns only about the
// genuinely dangerous direction and stays silent otherwise. Kept as a warning
// rather than an error for the same reason razorpaySlotWarning is: the interim
// state while real credentials are pending must remain reachable.
func cashfreeSlotWarning(slot, environment, appID string) string {
	// Cashfree sandbox App IDs are prefixed "TEST". A test App ID in the LIVE slot
	// is a hard failure, not a degraded state, and it is worth calling out
	// explicitly: the live slot resolves to api.cashfree.com, where test
	// credentials return 401 — so live checkout would fail outright rather than
	// quietly capture nothing.
	//
	// This is a normal interim state while a merchant account is still in review,
	// which is why it is a WARNING and the save is still allowed (mirroring
	// razorpaySlotWarning). SelectCheckoutGateway is what keeps real orders
	// flowing meanwhile, by degrading those checkouts to Razorpay.
	testAppID := strings.HasPrefix(strings.ToUpper(appID), "TEST")
	switch {
	case !models.IsTestMode(slot) && testAppID:
		return "The Live slot is holding a TEST App ID — Cashfree separates environments by host, so live payments will fail with 401 until real live credentials are entered. Live checkout falls back to Razorpay meanwhile."
	case models.IsTestMode(slot) && appID != "" && !testAppID:
		return "The Test slot is holding what looks like a LIVE App ID — sandbox orders could charge real cards. Replace it before using test mode."
	case !models.IsTestMode(slot) && environment == "sandbox":
		return "The Live slot is resolving to the Cashfree SANDBOX — no real payment will be captured until live credentials are entered."
	case models.IsTestMode(slot) && environment == "production":
		return "The Test slot is resolving to Cashfree PRODUCTION — sandbox orders would charge real cards. Fix this before using test mode."
	default:
		return ""
	}
}

// UpdateCashfreeGatewayKeys writes one Cashfree credential slot to GCP Secret
// Manager, invalidates that slot's cached client, and runs an immediate health
// check so the admin sees pass/fail in the same response.
//
// Same contract as UpdatePaymentGatewayKeys (Razorpay), including per-slot
// invalidation: saving test keys can never knock a healthy live gateway offline.
// Secrets are never written to the DB or to config — the app reads them from
// Secret Manager on demand.
//
// PUT /admin/payment-gateway/cashfree/keys
func (h *AdminHandler) UpdateCashfreeGatewayKeys(c *gin.Context) {
	var req struct {
		AppID         string `json:"appId"`
		SecretKey     string `json:"secretKey"`
		WebhookSecret string `json:"webhookSecret"`
		// Mode picks the credential slot: "live" (default) or "test".
		Mode string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.AppID == "" && req.SecretKey == "" && req.WebhookSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one field is required"})
		return
	}

	// Writing the app id and secret key together is the only sensible mode — a
	// mismatched pair guarantees a 401 from Cashfree. Require both or neither.
	if (req.AppID == "") != (req.SecretKey == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "appId and secretKey must be provided together"})
		return
	}

	ctx := c.Request.Context()
	slot := models.NormalizeMode(req.Mode)
	// Read the slot's secret names from the SAME helper the client reads them
	// with, so the slot an admin saves into and the slot the app loads from cannot
	// drift apart. A drift of exactly this kind once made admin-entered Razorpay
	// keys silently invisible to the app.
	appIDName, secretName, webhookName := services.CashfreeSecretNames(slot)
	for secretName, value := range map[string]string{
		appIDName:   req.AppID,
		secretName:  req.SecretKey,
		webhookName: req.WebhookSecret,
	} {
		if value == "" {
			continue
		}
		if err := services.StorePlatformSecret(ctx, secretName, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to store %s: %v", secretName, err)})
			return
		}
	}

	services.InvalidateCashfreeFor(slot)
	services.LogAudit(c, "payment.keys.update", "payment_gateway", models.PaymentProviderCashfree, nil, map[string]any{
		"slot": slot,
		"updatedFields": []string{
			boolField("appId", req.AppID != ""),
			boolField("secretKey", req.SecretKey != ""),
			boolField("webhookSecret", req.WebhookSecret != ""),
		},
	})

	// Validate by actually calling Cashfree, so wrong keys surface immediately
	// rather than at a customer's checkout.
	client := services.GetCashfreeFor(slot)
	if client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Saved to Secret Manager, but client failed to initialize"})
		return
	}
	environment := "production"
	if client.IsSandbox() {
		environment = "sandbox"
	}
	warning := cashfreeSlotWarning(slot, environment, client.GetAppID())
	if err := client.HealthCheck(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message":     "Keys saved, but validation failed",
			"testError":   err.Error(),
			"slot":        slot,
			"environment": environment,
			"slotWarning": warning,
			"verified":    false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Cashfree gateway keys saved and verified",
		"slot":        slot,
		"environment": environment,
		"slotWarning": warning,
		"verified":    true,
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
