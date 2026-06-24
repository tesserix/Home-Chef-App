package handlers

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChefHandler struct{}

func NewChefHandler() *ChefHandler {
	return &ChefHandler{}
}

// chefBoundingBox returns the lat/lng rectangle around (lat,lng) covering
// radiusKm, used as a cheap SQL-side "near me" prefilter (#36). ~1° latitude ≈
// 111 km; longitude degrees shrink by cos(lat). A rectangle is a good-enough
// nearby filter for v1 and, unlike a Go-side circle filter, paginates in SQL.
func chefBoundingBox(lat, lng, radiusKm float64) (minLat, maxLat, minLng, maxLng float64) {
	dLat := radiusKm / 111.0
	cosLat := math.Cos(lat * math.Pi / 180.0)
	if cosLat < 0.01 {
		cosLat = 0.01 // guard near the poles; India is mid-latitude so n/a
	}
	dLng := radiusKm / (111.0 * cosLat)
	return lat - dLat, lat + dLat, lng - dLng, lng + dLng
}

// ListChefs returns a paginated list of chefs
func (h *ChefHandler) ListChefs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	cuisine := c.Query("cuisine")
	dietary := c.Query("dietary")
	isOpen := c.Query("isOpen")
	ratingMin := c.Query("rating")
	sortOrder := c.DefaultQuery("order", "desc")

	// Accept both "sortBy" and "sort" (frontend sends "sort")
	sortBy := c.Query("sortBy")
	if sortBy == "" {
		sortBy = c.DefaultQuery("sort", "rating")
	}

	// Price range + near-me geo (#36).
	minPriceStr := c.Query("minPrice")
	maxPriceStr := c.Query("maxPrice")
	var geoLat, geoLng float64
	hasGeo := false
	if latS, lngS := c.Query("lat"), c.Query("lng"); latS != "" && lngS != "" {
		la, e1 := strconv.ParseFloat(latS, 64)
		lo, e2 := strconv.ParseFloat(lngS, 64)
		if e1 == nil && e2 == nil {
			geoLat, geoLng, hasGeo = la, lo, true
		}
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.ChefProfile{}).
		Where("is_active = ?", true).
		Scopes(services.ExcludeFSSAILocked) // FSSAI lockout (#91): hide lapsed-licence India chefs

	// Search filter
	if search != "" {
		query = query.Where("business_name ILIKE ? OR description ILIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	// Cuisine filter
	if cuisine != "" {
		query = query.Where("? = ANY(cuisines)", cuisine)
	}

	// isOpen filter
	if isOpen == "true" {
		query = query.Where("accepting_orders = ?", true)
	}

	// Minimum rating filter
	if ratingMin != "" {
		if r, err := strconv.ParseFloat(ratingMin, 64); err == nil {
			query = query.Where("rating >= ?", r)
		}
	}

	// Dietary filter — find chefs that have at least one menu item with this dietary tag
	if dietary != "" {
		query = query.Where("id IN (SELECT chef_id FROM menu_items WHERE ? = ANY(dietary_tags) AND deleted_at IS NULL)", dietary)
	}

	// Price-range filter on the chef's minimum order (#36).
	if minPriceStr != "" {
		if v, err := strconv.ParseFloat(minPriceStr, 64); err == nil {
			query = query.Where("minimum_order >= ?", v)
		}
	}
	if maxPriceStr != "" {
		if v, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
			query = query.Where("minimum_order <= ?", v)
		}
	}

	// Near-me: bounding-box prefilter around the customer's coords (#36). radius
	// defaults to 15km. Cheap, SQL-side, paginates correctly.
	if hasGeo {
		radiusKm := 15.0
		if rS := c.Query("radius"); rS != "" {
			if r, err := strconv.ParseFloat(rS, 64); err == nil && r > 0 {
				radiusKm = r
			}
		}
		minLat, maxLat, minLng, maxLng := chefBoundingBox(geoLat, geoLng, radiusKm)
		query = query.Where("latitude BETWEEN ? AND ? AND longitude BETWEEN ? AND ?",
			minLat, maxLat, minLng, maxLng)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Apply sorting — featured chefs always appear first
	dir := "DESC"
	if sortOrder == "asc" {
		dir = "ASC"
	}

	// Ranking priority (best first): premium chefs (#44), then active featured
	// chefs (paid promotion), then everyone — within each band by the chosen sort.
	// Premium gets top placement in search & discovery; the correlated EXISTS
	// keys off the chef's user's active/trial premium subscription.
	premiumOrder := "CASE WHEN EXISTS (" +
		"SELECT 1 FROM subscriptions s WHERE s.user_id = chef_profiles.user_id " +
		"AND s.tier = 'premium' AND s.status IN ('trial','active') AND s.deleted_at IS NULL" +
		") THEN 0 ELSE 1 END ASC"
	featuredOrder := "CASE WHEN is_featured = true AND featured_until > NOW() THEN 0 ELSE 1 END ASC"
	rankOrder := premiumOrder + ", " + featuredOrder

	switch sortBy {
	case "rating":
		query = query.Order(rankOrder + ", rating " + dir)
	case "orders":
		query = query.Order(rankOrder + ", total_orders " + dir)
	case "newest":
		query = query.Order(rankOrder + ", created_at " + dir)
	case "price":
		query = query.Order(rankOrder + ", minimum_order " + dir)
	case "distance":
		if hasGeo {
			// Squared distance on lat/lng — a monotonic proxy for true distance
			// at city scale; cheap (no trig) and orders nearby chefs correctly.
			// Closest first.
			query = query.Order(clause.Expr{
				SQL:  rankOrder + ", ((latitude - ?) * (latitude - ?) + (longitude - ?) * (longitude - ?)) ASC",
				Vars: []interface{}{geoLat, geoLat, geoLng, geoLng},
			})
		} else {
			query = query.Order(rankOrder + ", rating " + dir)
		}
	default:
		query = query.Order(rankOrder + ", rating " + dir)
	}

	// Get chefs
	var chefs []models.ChefProfile
	if err := query.Offset(offset).Limit(limit).Find(&chefs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chefs"})
		return
	}

	// Convert to response
	responses := make([]models.ChefProfileResponse, len(chefs))
	for i, chef := range chefs {
		responses[i] = chef.ToResponse()
		// Customer-facing: never expose the chef's exact coordinates. Show an
		// approximate area (deterministic per-chef offset) so the "chefs near
		// you" map can place them without revealing the kitchen address.
		responses[i].Latitude, responses[i].Longitude =
			services.FuzzCoordinate(chef.Latitude, chef.Longitude, chef.ID.String())
	}

	// Hygiene/food-safety badge (#35): one batched lookup for the whole page,
	// so the chef cards can show the badge without an N+1 per chef.
	if len(chefs) > 0 {
		chefIDs := make([]uuid.UUID, len(chefs))
		for i, chef := range chefs {
			chefIDs[i] = chef.ID
		}
		badged := services.ChefsWithValidFSSAI(chefIDs)
		// Verified-Pro badge (#44) — one batched lookup for the whole page.
		premium := services.PremiumChefIDs(chefIDs)
		for i := range responses {
			responses[i].FoodSafetyBadge = badged[responses[i].ID]
			responses[i].ProBadge = premium[responses[i].ID]
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
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

// SearchDishes searches menu items by name/description across all active,
// non-FSSAI-locked chefs (#36 — "search by dish"). Returns matching dishes with
// their chef id so the client can deep-link to the chef. GET /search/dishes
func (h *ChefHandler) SearchDishes(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query must be at least 2 characters"})
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

	// Restrict to dishes from active, non-locked chefs — mirrors the visibility
	// rule ListChefs applies (FSSAI lockout #91 included).
	visibleChefs := database.DB.Model(&models.ChefProfile{}).
		Where("is_active = ?", true).
		Scopes(services.ExcludeFSSAILocked).
		Select("id")

	// Gate on availability only — consistent with the chef-detail menu
	// (GetChefMenu) and order creation (CreateOrder), which both use is_available
	// as the single visibility gate. (Previously search also required is_approved,
	// so an available dish was orderable from a chef's page but never appeared in
	// search.) If admin moderation of dishes is desired, enforce is_approved in
	// all three paths instead — a product decision.
	base := database.DB.Model(&models.MenuItem{}).
		Where("is_available = ?", true).
		Where("(name ILIKE ? OR description ILIKE ?)", "%"+q+"%", "%"+q+"%").
		Where("chef_id IN (?)", visibleChefs)

	// Optional diet filter (#41) — case-insensitive match on a dietary tag.
	if dietary := strings.TrimSpace(c.Query("dietary")); dietary != "" {
		base = base.Where("EXISTS (SELECT 1 FROM unnest(dietary_tags) AS t WHERE lower(t) = lower(?))", dietary)
	}

	var total int64
	base.Count(&total)

	var items []models.MenuItem
	if err := base.Order("rating DESC, total_reviews DESC").
		Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
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

// GetChef returns a single chef by ID
func (h *ChefHandler) GetChef(c *gin.Context) {
	// Accept either a UUID or an SEO slug (#58), so app universal links and
	// landing pages can use /chefs/<slug> as well as /chefs/<uuid>.
	idOrSlug := c.Param("id")
	q := database.DB.Preload("User")
	if chefID, err := uuid.Parse(idOrSlug); err == nil {
		q = q.Where("id = ?", chefID)
	} else {
		// Oldest match wins on the (rare) slug collision — stable canonical target.
		q = q.Where("slug = ?", idOrSlug).Order("created_at")
	}

	var chef models.ChefProfile
	if err := q.First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	// FSSAI lockout (#91): a direct link to a chef whose food-safety licence has
	// lapsed should read as unavailable, matching their absence from listings.
	if services.IsChefFSSAIExpired(&chef) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	var schedules []models.ChefSchedule
	database.DB.Where("chef_id = ?", chef.ID).Find(&schedules)

	resp := chef.ToPublicResponse(schedules)
	resp.ProBadge = services.IsChefPremium(chef.ID) // Verified-Pro badge (#44)
	// Customer-facing: approximate the kitchen location (deterministic per-chef
	// offset) so the exact address is never exposed to customers.
	resp.Latitude, resp.Longitude =
		services.FuzzCoordinate(chef.Latitude, chef.Longitude, chef.ID.String())
	c.JSON(http.StatusOK, resp)
}

// GetChefMenu returns the menu items and categories for a chef
// resolveChefID maps a :id route param (a UUID or an SEO slug, #58) to the chef's
// UUID, so every chef endpoint can be addressed by /chefs/<slug>. Returns false
// when no chef matches.
func resolveChefID(idOrSlug string) (uuid.UUID, bool) {
	if id, err := uuid.Parse(idOrSlug); err == nil {
		return id, true
	}
	var chef models.ChefProfile
	if err := database.DB.Select("id").Where("slug = ?", idOrSlug).Order("created_at").First(&chef).Error; err != nil {
		return uuid.Nil, false
	}
	return chef.ID, true
}

func (h *ChefHandler) GetChefMenu(c *gin.Context) {
	chefID, ok := resolveChefID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	category := c.Query("category")

	query := database.DB.Where("chef_id = ? AND is_available = ?", chefID, true).
		Preload("Images").
		// Add-ons + combo composition (#52) so the customer can pick modifiers
		// and see what a combo includes.
		Preload("ModifierGroups", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Preload("ModifierGroups.Options", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") }).
		Preload("ComboItems", func(db *gorm.DB) *gorm.DB { return db.Order("sort_order") })

	if category != "" {
		query = query.Where("category_id = ?", category)
	}
	// Diet filter (#41) — case-insensitive match on a dietary tag (e.g. veg).
	if dietary := c.Query("dietary"); dietary != "" {
		query = query.Where("EXISTS (SELECT 1 FROM unnest(dietary_tags) AS t WHERE lower(t) = lower(?))", dietary)
	}

	var items []models.MenuItem
	if err := query.Order("sort_order, name").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch menu"})
		return
	}

	// Fetch categories for this chef
	var categories []models.MenuCategory
	database.DB.Where("chef_id = ? AND is_active = ?", chefID, true).
		Order("sort_order, name").Find(&categories)

	capDay := services.CapacityDay(time.Now())
	responses := make([]models.MenuItemResponse, len(items))
	for i := range items {
		// Surface today's remaining count + sold-out for capped dishes (#48).
		// Set on the model so ToResponse carries it (single source of truth).
		if rem, soldOut := services.RemainingToday(items[i].ID, items[i].DailyCapacity, capDay); rem != nil {
			items[i].RemainingToday = rem
			items[i].SoldOut = soldOut
		}
		responses[i] = items[i].ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
		"items":      responses,
	})
}

// GetChefReviews returns reviews for a chef
func (h *ChefHandler) GetChefReviews(c *gin.Context) {
	chefID, ok := resolveChefID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var reviews []models.Review
	var total int64

	database.DB.Model(&models.Review{}).Where("chef_id = ? AND is_approved = ?", chefID, true).Count(&total)

	if err := database.DB.Preload("Customer").
		Where("chef_id = ? AND is_approved = ?", chefID, true).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}

	responses := make([]models.ReviewResponse, len(reviews))
	for i, review := range reviews {
		responses[i] = review.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
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

// ---- Chef Dashboard Endpoints ----

// GetChefProfile returns the full chef profile for the authenticated chef
func (h *ChefHandler) GetChefProfile(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// Load schedules and convert to operatingHours map
	var schedules []models.ChefSchedule
	database.DB.Where("chef_id = ?", chef.ID).Find(&schedules)

	dayNames := map[int]string{
		0: "sunday", 1: "monday", 2: "tuesday", 3: "wednesday",
		4: "thursday", 5: "friday", 6: "saturday",
	}

	operatingHours := make(map[string]interface{})
	for _, s := range schedules {
		name, ok := dayNames[s.DayOfWeek]
		if !ok {
			continue
		}
		if s.IsClosed {
			// Don't include closed days — frontend treats missing = closed
			continue
		}
		operatingHours[name] = map[string]string{
			"open":  s.OpenTime,
			"close": s.CloseTime,
		}
	}

	resp := chef.ToResponse()
	// Merge profile response with operating hours. Address fields surface
	// the full street address so the chef can correct it post-onboarding.
	result := map[string]interface{}{
		"id":            resp.ID,
		"userId":        resp.UserID,
		"businessName":  resp.BusinessName,
		"description":   resp.Description,
		"profileImage":  resp.ProfileImage,
		"bannerImage":   resp.BannerImage,
		"cuisines":      resp.Cuisines,
		"specialties":   resp.Specialties,
		"prepTime":      resp.PrepTime,
		"minimumOrder":  resp.MinimumOrder,
		"serviceRadius": resp.ServiceRadius,
		"rating":        resp.Rating,
		"totalReviews":  resp.TotalReviews,
		"totalOrders":   resp.TotalOrders,
		// issueCount feeds the chef's issue rate (#37) shown in vendor analytics.
		"issueCount":      chef.IssueCount,
		"verified":        resp.IsVerified,
		"acceptingOrders": resp.AcceptingOrders,
		"kitchenPhotos":   resp.KitchenPhotos,
		"addressLine1":    chef.AddressLine1,
		"addressLine2":    chef.AddressLine2,
		"city":            chef.City,
		"state":           chef.State,
		"postalCode":      chef.PostalCode,
		"operatingHours":  operatingHours,
		// Fulfillment capabilities + self-delivery pricing. These MUST be
		// returned so the vendor profile editor reflects the saved state — when
		// they were omitted the toggles always re-read as OFF after a reload, and
		// the next save echoed that stale OFF straight back into the DB, silently
		// disabling pickup/self-delivery the chef had turned on.
		"offersPickup":              chef.OffersPickup,
		"offersSelfDelivery":        chef.OffersSelfDelivery,
		"selfDeliveryBaseFee":       chef.SelfDeliveryBaseFee,
		"selfDeliveryFreeRadiusKm":  chef.SelfDeliveryFreeRadiusKm,
		"selfDeliveryPerKm":         chef.SelfDeliveryPerKm,
		"selfDeliveryMaxFee":        chef.SelfDeliveryMaxFee,
		"selfDeliveryMaxDistanceKm": chef.SelfDeliveryMaxDistanceKm,
	}

	c.JSON(http.StatusOK, result)
}

// GetChefDashboard returns the chef's dashboard data
func (h *ChefHandler) GetChefDashboard(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// "Today"/"this week" are the chef's business day in IST. Containers run UTC,
	// so a UTC CURRENT_DATE missed IST-evening orders that the History list
	// (grouped client-side in IST) still shows under "Today" — the dashboard read
	// 0 while History showed the order. Reuse the IST day helper so the two agree.
	todayStart := services.CapacityDay(time.Now())
	weekStart := todayStart.AddDate(0, 0, -7)

	// Get today's stats
	var todayOrders int64
	var todayRevenue float64
	database.DB.Model(&models.Order{}).
		Where("chef_id = ? AND created_at >= ?", chef.ID, todayStart).
		Count(&todayOrders)
	database.DB.Model(&models.Order{}).
		Where("chef_id = ? AND created_at >= ? AND payment_status = ?", chef.ID, todayStart, models.PaymentCompleted).
		Select("COALESCE(SUM(total), 0)").Scan(&todayRevenue)

	// Get pending orders — only PAID ones. An order row is created before
	// payment, so unpaid/abandoned orders (payment_status pending/failed) must
	// not surface to the chef as new orders.
	var pendingOrders int64
	database.DB.Model(&models.Order{}).
		Where("chef_id = ? AND status = ? AND payment_status = ?", chef.ID, models.OrderStatusPending, models.PaymentCompleted).
		Count(&pendingOrders)

	// Get this week's stats (IST week window)
	var weekOrders int64
	var weekRevenue float64
	database.DB.Model(&models.Order{}).
		Where("chef_id = ? AND created_at >= ?", chef.ID, weekStart).
		Count(&weekOrders)
	database.DB.Model(&models.Order{}).
		Where("chef_id = ? AND created_at >= ? AND payment_status = ?", chef.ID, weekStart, models.PaymentCompleted).
		Select("COALESCE(SUM(total), 0)").Scan(&weekRevenue)

	// Recent orders for the dashboard's in-flight + history-glance section.
	// Mobile reads `recentOrders` and filters by status client-side. Preload
	// Customer so we can populate `customerName` (the mobile dashboard renders
	// it on the InFlightRow and the dead-screen reassurance copy reads
	// `lastOrderIso` from `createdAt`).
	var recent []models.Order
	database.DB.Where("chef_id = ? AND payment_status IN ?", chef.ID, []models.PaymentStatus{models.PaymentCompleted, models.PaymentRefunded}).
		Preload("Customer").
		Order("created_at DESC").
		Limit(10).
		Find(&recent)

	recentOrdersResp := make([]gin.H, len(recent))
	for i, o := range recent {
		fulfillment := o.FulfillmentType
		if fulfillment == "" {
			fulfillment = models.FulfillmentDelivery
		}
		recentOrdersResp[i] = gin.H{
			"id": o.ID,
			// First name only for the chef view (privacy; matches ToChefResponse).
			"customerName": o.Customer.FirstName,
			"total":        o.Total,
			"status":       o.Status,
			"createdAt":    o.CreatedAt,
			// Drives the dashboard in-flight card's pickup-vs-delivery stepper
			// + the chef's "Mark handed over" action on pickup orders.
			"fulfillmentType": fulfillment,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"todayOrders":  todayOrders,
		"todayRevenue": todayRevenue,
		// Mobile reads `todayEarnings`; keep `todayRevenue` for any other
		// consumer.
		"todayEarnings":   todayRevenue,
		"pendingOrders":   pendingOrders,
		"weekOrders":      weekOrders,
		"weekRevenue":     weekRevenue,
		"rating":          chef.Rating,
		"totalReviews":    chef.TotalReviews,
		"totalOrders":     chef.TotalOrders,
		"acceptingOrders": chef.AcceptingOrders,
		"pausedUntil":     chef.PausedUntil,
		// FSSAI lockout (#92): drives the vendor dashboard's "orders paused —
		// renew licence" banner. Same helper as the order/payout enforcement.
		"fssaiLocked":  services.IsChefFSSAIExpired(&chef),
		"recentOrders": recentOrdersResp,
	})
}

// UpdateChefProfileRequest represents chef profile update.
// Pointer types let callers send `null`/omit fields to skip the update,
// while sending an empty string or zero genuinely clears the value.
// This unblocks clearing the description / address fields from the app.
type UpdateChefProfileRequest struct {
	BusinessName    *string  `json:"businessName"`
	Description     *string  `json:"description"`
	ProfileImage    *string  `json:"profileImage"`
	BannerImage     *string  `json:"bannerImage"`
	Cuisines        []string `json:"cuisines"`
	Specialties     []string `json:"specialties"`
	PrepTime        *string  `json:"prepTime"`
	MinimumOrder    *float64 `json:"minimumOrder"`
	ServiceRadius   *float64 `json:"serviceRadius"`
	AcceptingOrders *bool    `json:"acceptingOrders"`
	OffersPickup    *bool    `json:"offersPickup"`
	// Chef self-delivery offering + pricing (Phase 2).
	OffersSelfDelivery        *bool                      `json:"offersSelfDelivery"`
	SelfDeliveryBaseFee       *float64                   `json:"selfDeliveryBaseFee"`
	SelfDeliveryFreeRadiusKm  *float64                   `json:"selfDeliveryFreeRadiusKm"`
	SelfDeliveryPerKm         *float64                   `json:"selfDeliveryPerKm"`
	SelfDeliveryMaxFee        *float64                   `json:"selfDeliveryMaxFee"`
	SelfDeliveryMaxDistanceKm *float64                   `json:"selfDeliveryMaxDistanceKm"`
	OperatingHours            map[string]*DayHoursUpdate `json:"operatingHours"`

	// Address fields — added so the chef can edit their kitchen address
	// post-onboarding. Backend previously only accepted these during the
	// onboarding submit flow, leaving no surface for corrections.
	AddressLine1 *string `json:"addressLine1"`
	AddressLine2 *string `json:"addressLine2"`
	City         *string `json:"city"`
	State        *string `json:"state"`
	PostalCode   *string `json:"postalCode"`
}

type DayHoursUpdate struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

// UpdateChefProfile updates the chef's profile
func (h *ChefHandler) UpdateChefProfile(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var req UpdateChefProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply each updatable field. Pointer types semantically mean
	// "present in the payload" — sending nil skips the update, sending
	// an empty string or zero clears the stored value. This is how the
	// chef can legitimately blank their description or postal code.
	if req.BusinessName != nil {
		chef.BusinessName = *req.BusinessName
	}
	if req.Description != nil {
		chef.Description = *req.Description
	}
	if req.ProfileImage != nil {
		chef.ProfileImage = *req.ProfileImage
	}
	if req.BannerImage != nil {
		chef.BannerImage = *req.BannerImage
	}
	if req.Cuisines != nil {
		chef.Cuisines = req.Cuisines
	}
	if req.Specialties != nil {
		chef.Specialties = req.Specialties
	}
	if req.PrepTime != nil {
		chef.PrepTime = *req.PrepTime
	}
	if req.MinimumOrder != nil {
		chef.MinimumOrder = *req.MinimumOrder
	}
	if req.ServiceRadius != nil {
		chef.ServiceRadius = *req.ServiceRadius
	}
	if req.AcceptingOrders != nil {
		chef.AcceptingOrders = *req.AcceptingOrders
	}
	if req.OffersPickup != nil {
		chef.OffersPickup = *req.OffersPickup
	}
	if req.OffersSelfDelivery != nil {
		chef.OffersSelfDelivery = *req.OffersSelfDelivery
	}
	if req.SelfDeliveryBaseFee != nil {
		chef.SelfDeliveryBaseFee = *req.SelfDeliveryBaseFee
	}
	if req.SelfDeliveryFreeRadiusKm != nil {
		chef.SelfDeliveryFreeRadiusKm = *req.SelfDeliveryFreeRadiusKm
	}
	if req.SelfDeliveryPerKm != nil {
		chef.SelfDeliveryPerKm = *req.SelfDeliveryPerKm
	}
	if req.SelfDeliveryMaxFee != nil {
		chef.SelfDeliveryMaxFee = *req.SelfDeliveryMaxFee
	}
	if req.SelfDeliveryMaxDistanceKm != nil {
		chef.SelfDeliveryMaxDistanceKm = *req.SelfDeliveryMaxDistanceKm
	}
	if req.AddressLine1 != nil {
		chef.AddressLine1 = *req.AddressLine1
	}
	if req.AddressLine2 != nil {
		chef.AddressLine2 = *req.AddressLine2
	}
	if req.City != nil {
		chef.City = *req.City
	}
	if req.State != nil {
		chef.State = *req.State
	}
	if req.PostalCode != nil {
		chef.PostalCode = *req.PostalCode
	}

	if err := database.DB.Save(&chef).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// Resolve kitchen coordinates from the address (best-effort) so pickup +
	// self-delivery distance work. Only when we have a street + city and the
	// address actually changed enough to matter; failures are non-fatal.
	if chef.AddressLine1 != "" && chef.City != "" {
		parts := []string{}
		for _, p := range []string{chef.AddressLine1, chef.AddressLine2, chef.City, chef.State, chef.PostalCode} {
			if strings.TrimSpace(p) != "" {
				parts = append(parts, p)
			}
		}
		full := strings.Join(parts, ", ")
		if lat, lng, ok := services.GeocodeAddress(full); ok {
			chef.Latitude, chef.Longitude = lat, lng
			database.DB.Model(&chef).Updates(map[string]any{"latitude": lat, "longitude": lng})
		}
	}

	// Update operating hours if provided
	if req.OperatingHours != nil {
		dayMap := map[string]int{
			"sunday": 0, "monday": 1, "tuesday": 2, "wednesday": 3,
			"thursday": 4, "friday": 5, "saturday": 6,
		}

		// Delete existing schedules and recreate
		database.DB.Where("chef_id = ?", chef.ID).Delete(&models.ChefSchedule{})

		for day, dayNum := range dayMap {
			schedule := models.ChefSchedule{
				ChefID:    chef.ID,
				DayOfWeek: dayNum,
				IsClosed:  true,
			}
			if dh, ok := req.OperatingHours[day]; ok && dh != nil {
				schedule.IsClosed = false
				schedule.OpenTime = dh.Open
				schedule.CloseTime = dh.Close
			}
			database.DB.Create(&schedule)
		}
	}

	c.JSON(http.StatusOK, chef.ToResponse())
}

// GetChefOrders returns orders for the chef
func (h *ChefHandler) GetChefOrders(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Only show PAID orders to the chef. Orders are created before payment, so
	// exclude unpaid/abandoned ones (payment_status pending/failed); keep
	// completed + refunded (the latter for history).
	query := database.DB.Where("chef_id = ? AND payment_status IN ?", chef.ID,
		[]models.PaymentStatus{models.PaymentCompleted, models.PaymentRefunded})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.Order{}).Count(&total)

	var orders []models.Order
	if err := query.Preload("Items").Preload("Customer").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	responses := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		// Chef view: area-only address, no phone, first name only (privacy).
		responses[i] = order.ToChefResponse()
	}

	// Mobile (`useVendorPendingOrders`, `useVendorOrderHistory`) consumes the
	// `OrdersResponse` shape `{ orders, total, page, limit }`. Keep the
	// envelope flat so the existing typed hooks don't need adapters.
	c.JSON(http.StatusOK, gin.H{
		"orders": responses,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// UpdateOrderStatus updates an order's status
func (h *ChefHandler) UpdateOrderStatus(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("orderId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var order models.Order
	if err := database.DB.Preload("Items").Where("id = ? AND chef_id = ?", orderID, chef.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
		// Optional carrier choice the chef makes at Mark Ready (self-delivery
		// chefs only): "chef_delivery" = I'll deliver, "delivery" = hand to a rider.
		Carrier string `json:"carrier"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply the chef's carrier choice — allowed only before the order is en route.
	if req.Carrier != "" {
		if order.Status == models.OrderStatusPickedUp || order.Status == models.OrderStatusDelivered {
			c.JSON(http.StatusBadRequest, gin.H{"error": "carrier is locked once the order is out for delivery"})
			return
		}
		ft, err := resolveReadyCarrier(order.FulfillmentType, req.Carrier, chef)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		order.FulfillmentType = ft
	}

	priorStatus := order.Status
	order.Status = models.OrderStatus(req.Status)

	// Stamp the lifecycle timestamp for this transition (idempotent — only set
	// once) so the chef/customer order timelines render real times instead of
	// blanks. Repeated updates to the same status are safe.
	now := time.Now()
	switch order.Status {
	case models.OrderStatusAccepted:
		if order.AcceptedAt == nil {
			order.AcceptedAt = &now
		}
	case models.OrderStatusReady:
		if order.PreparedAt == nil {
			order.PreparedAt = &now
		}
	case models.OrderStatusPickedUp:
		if order.PickedUpAt == nil {
			order.PickedUpAt = &now
		}
	case models.OrderStatusDelivered:
		if order.DeliveredAt == nil {
			order.DeliveredAt = &now
		}
	}

	// A chef cancelling or rejecting an order should release the reserved daily
	// capacity (#48), but only on the first transition out of a live state.
	// "rejected" is a chef declining a pending order — handled identically to a
	// cancellation (capacity release + cancel/refund saga signal below).
	isCancel := order.Status == models.OrderStatusCancelled || order.Status == models.OrderStatusRejected
	releaseCap := isCancel &&
		priorStatus != models.OrderStatusCancelled &&
		priorStatus != models.OrderStatusRejected &&
		priorStatus != models.OrderStatusRefunded &&
		priorStatus != models.OrderStatusDelivered

	// Determine which subject to publish to based on status.
	subject := services.SubjectOrderUpdated
	if order.Status == models.OrderStatusDelivered {
		subject = services.SubjectOrderDelivered
	}

	// Persist the status change and stage the event atomically (transactional
	// outbox) so the status update is delivered durably by the relay (#131).
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&order).Error; err != nil {
			return err
		}
		if releaseCap {
			capDay := services.CapacityDay(order.CreatedAt)
			for _, it := range order.Items {
				if err := services.ReleaseCapacity(tx, it.MenuItemID, it.Quantity, capDay); err != nil {
					return err
				}
			}
			// Release the scheduled delivery-slot booking too (#51), keyed to the
			// order's scheduled delivery day.
			if order.DeliverySlot != "" && order.ScheduledFor != nil {
				if err := services.ReleaseSlot(tx, order.ChefID, order.DeliverySlot, 1, services.CapacityDay(*order.ScheduledFor)); err != nil {
					return err
				}
			}
		}
		return services.EnqueueOrderEvent(tx, subject, services.OrderEvent{
			OrderID:     order.ID,
			OrderNumber: order.OrderNumber,
			CustomerID:  order.CustomerID,
			ChefID:      order.ChefID,
			Status:      string(order.Status),
			Total:       order.Total,
		})
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	// Forward the lifecycle transition to the Order saga (#122) when enabled —
	// the saga awaits these as signals. No-op when ORDER_SAGA_ENABLED is off, so
	// this is invisible until ops cuts over.
	switch order.Status {
	case models.OrderStatusAccepted:
		services.SignalOrderChefDecision(order.ID, true, "")
	case models.OrderStatusReady:
		services.SignalOrderReady(order.ID)
	case models.OrderStatusDelivered:
		services.SignalOrderDelivered(order.ID)
	case models.OrderStatusCancelled, models.OrderStatusRejected:
		services.SignalOrderCancelled(order.ID, "cancelled by chef")
	}

	// Auto-dispatch a 3PL delivery once the food is ready for pickup. Runs off
	// the request path; idempotent so repeated "ready" updates are safe. A
	// dispatch failure must not fail the chef's status update.
	// Pickup orders are collected by the customer; chef_delivery (Phase 2) is
	// carried by the chef. Neither dispatches a provider.
	if order.Status == models.OrderStatusReady && order.FulfillmentType == models.FulfillmentDelivery {
		// Durable dispatch via Temporal when enabled (retries the flaky 3PL
		// booking, survives crashes); falls back to the inline goroutine
		// otherwise. Idempotent by order ID, so repeated "ready" updates are safe.
		services.EnqueueDeliveryDispatch(order.ID)
	}

	// Chef view: area-only address, no customer PII (privacy).
	c.JSON(http.StatusOK, order.ToChefResponse())
}

// UploadOrderPhoto attaches a lifecycle photo to an order (multipart field
// `file`, plus `kind` = "ready" | "handover"). The vendor app requires the chef
// to capture this photo before advancing the matching status: the food-ready
// photo (shown to the customer) at "Mark ready", and the proof-of-handover
// photo (pickup dispute evidence) at "Mark handed over". Stores the URL on the
// order; the separate status update follows. Idempotent — re-uploading replaces
// the URL for that kind.
// POST /chef/orders/:orderId/photos
func (h *ChefHandler) UploadOrderPhoto(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("orderId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND chef_id = ?", orderID, chef.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	kind := c.PostForm("kind")
	if kind != "ready" && kind != "handover" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind must be 'ready' or 'handover'"})
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

	folder := fmt.Sprintf("orders/%s/%s", order.ID.String(), kind)
	fileURL, err := services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
	if err != nil {
		log.Printf("Failed to upload order photo (order=%s kind=%s): %v", order.ID, kind, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload photo"})
		return
	}

	column := "ready_photo_url"
	if kind == "handover" {
		column = "handover_photo_url"
	}
	if err := database.DB.Model(&order).Update(column, fileURL).Error; err != nil {
		log.Printf("Failed to persist order photo url (order=%s kind=%s): %v", order.ID, kind, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save photo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"kind": kind, "url": fileURL})
}

// GetChefCapacitySettings — GET /chef/capacity-settings (#48). Cutoffs + auto-sold-out.
func (h *ChefHandler) GetChefCapacitySettings(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}
	c.JSON(http.StatusOK, services.GetChefCapacitySettings(chef.ID))
}

type updateCapacitySettingsRequest struct {
	CutoffEnabled *bool   `json:"cutoffEnabled"`
	LunchCutoff   *string `json:"lunchCutoff"`
	DinnerCutoff  *string `json:"dinnerCutoff"`
	AutoSoldOut   *bool   `json:"autoSoldOut"`

	// Scheduled delivery slots (#51)
	SlotsEnabled       *bool   `json:"slotsEnabled"`
	LunchSlotStart     *string `json:"lunchSlotStart"`
	LunchSlotEnd       *string `json:"lunchSlotEnd"`
	DinnerSlotStart    *string `json:"dinnerSlotStart"`
	DinnerSlotEnd      *string `json:"dinnerSlotEnd"`
	LunchSlotCapacity  *int    `json:"lunchSlotCapacity"`
	DinnerSlotCapacity *int    `json:"dinnerSlotCapacity"`
}

// UpdateChefCapacitySettings — PUT /chef/capacity-settings (#48, upsert).
func (h *ChefHandler) UpdateChefCapacitySettings(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}
	var req updateCapacitySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// All "HH:MM" fields (cutoffs + slot windows) must parse when non-empty.
	for _, v := range []*string{
		req.LunchCutoff, req.DinnerCutoff,
		req.LunchSlotStart, req.LunchSlotEnd, req.DinnerSlotStart, req.DinnerSlotEnd,
	} {
		if v != nil && *v != "" {
			if _, _, ok := services.ParseCutoff(*v); !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "time must be HH:MM (24h), e.g. 10:00"})
				return
			}
		}
	}
	// Each slot window must have start before end when both are given.
	if !slotWindowOrdered(req.LunchSlotStart, req.LunchSlotEnd) ||
		!slotWindowOrdered(req.DinnerSlotStart, req.DinnerSlotEnd) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slot window start must be before end"})
		return
	}
	var s models.ChefCapacitySettings
	database.DB.Where("chef_id = ?", chef.ID).FirstOrInit(&s)
	s.ChefID = chef.ID
	if req.CutoffEnabled != nil {
		s.CutoffEnabled = *req.CutoffEnabled
	}
	if req.LunchCutoff != nil {
		s.LunchCutoff = *req.LunchCutoff
	}
	if req.DinnerCutoff != nil {
		s.DinnerCutoff = *req.DinnerCutoff
	}
	if req.AutoSoldOut != nil {
		s.AutoSoldOut = *req.AutoSoldOut
	}
	if req.SlotsEnabled != nil {
		s.SlotsEnabled = *req.SlotsEnabled
	}
	if req.LunchSlotStart != nil {
		s.LunchSlotStart = *req.LunchSlotStart
	}
	if req.LunchSlotEnd != nil {
		s.LunchSlotEnd = *req.LunchSlotEnd
	}
	if req.DinnerSlotStart != nil {
		s.DinnerSlotStart = *req.DinnerSlotStart
	}
	if req.DinnerSlotEnd != nil {
		s.DinnerSlotEnd = *req.DinnerSlotEnd
	}
	// Capacity: negative is rejected; 0 means "unlimited" (stored as nil).
	if req.LunchSlotCapacity != nil {
		if *req.LunchSlotCapacity < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "slot capacity cannot be negative"})
			return
		}
		s.LunchSlotCapacity = normalizeCapacity(*req.LunchSlotCapacity)
	}
	if req.DinnerSlotCapacity != nil {
		if *req.DinnerSlotCapacity < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "slot capacity cannot be negative"})
			return
		}
		s.DinnerSlotCapacity = normalizeCapacity(*req.DinnerSlotCapacity)
	}
	if err := database.DB.Save(&s).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save settings"})
		return
	}
	c.JSON(http.StatusOK, s)
}

// slotWindowOrdered reports whether a slot's start time precedes its end time.
// True when either bound is absent/blank (nothing to compare).
func slotWindowOrdered(start, end *string) bool {
	if start == nil || end == nil || *start == "" || *end == "" {
		return true
	}
	sh, sm, sok := services.ParseCutoff(*start)
	eh, em, eok := services.ParseCutoff(*end)
	if !sok || !eok {
		return true // already flagged by the HH:MM validation above
	}
	return sh*60+sm < eh*60+em
}

// normalizeCapacity maps a capacity input to storage: 0 → nil (unlimited),
// positive → a pointer to that value.
func normalizeCapacity(n int) *int {
	if n <= 0 {
		return nil
	}
	return &n
}

// GetChefDeliverySlots — GET /chefs/:id/delivery-slots (PUBLIC). Returns the
// chef's offered scheduled-delivery slots across the booking horizon with
// per-day remaining capacity and open/closed state, for the checkout picker (#51).
func (h *ChefHandler) GetChefDeliverySlots(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	s := services.GetChefCapacitySettings(chefID)
	slots := services.BuildSlotAvailability(s, chefID, time.Now())
	c.JSON(http.StatusOK, gin.H{
		"slotsEnabled": s.SlotsEnabled,
		"slots":        slots,
	})
}

// GetOrderDetail returns a single order's full detail for the authenticated chef.
// The chef must own the order (order.chef_id == authenticated chef ID); returns 404
// when the order doesn't exist or belongs to a different chef.
// GET /chef/orders/:orderId
func (h *ChefHandler) GetOrderDetail(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderIDStr := c.Param("orderId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var order models.Order
	if err := database.DB.
		Preload("Items").
		Preload("Items.MenuItem").
		Preload("Customer").
		Where("id = ? AND chef_id = ?", orderIDStr, chef.ID).
		First(&order).Error; err != nil {
		// 404 for both "not found" and "wrong chef" — don't leak existence
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Chef view: area-only address, no phone, first name only (privacy).
	resp := order.ToChefResponse()

	// Enrich items with isVeg from the live MenuItem and surfacespecialInstructions
	for i, item := range order.Items {
		resp.Items[i].SpecialInstructions = item.Notes
		if item.MenuItem.ID != (item.MenuItemID) {
			// MenuItem was not loaded (e.g., item refers to a soft-deleted item) — skip
			continue
		}
		resp.Items[i].IsVeg = item.MenuItem.IsVeg
	}

	detail := models.ChefOrderDetailResponse{
		OrderResponse:         resp,
		AcceptedAt:            order.AcceptedAt,
		PreparedAt:            order.PreparedAt,
		PickedUpAt:            order.PickedUpAt,
		DeliveredAt:           order.DeliveredAt,
		CancelledAt:           order.CancelledAt,
		CancelReason:          order.CancelReason,
		ScheduledFor:          order.ScheduledFor,
		EstimatedPrepTime:     order.EstimatedPrepTime,
		EstimatedDeliveryTime: order.EstimatedDeliveryTime,
		SpecialInstructions:   order.SpecialInstructions,
		// DeliveryInstructions (navigation, e.g. "3rd floor, blue door") is
		// intentionally omitted from the chef view — it can reveal building-level
		// address detail. The 3PL rider receives it server-to-server instead.
		PaymentMethod: order.PaymentMethod,
	}

	// The authoritative self-delivery CAPABILITY flag — the vendor app gates the
	// Mark-Ready carrier choice on this, NOT on the distance fields below (which
	// are 0 when the chef set no radius or coords are missing, so they can't tell
	// "can't self-deliver" apart from "no radius configured").
	detail.OffersSelfDelivery = chef.OffersSelfDelivery

	// Surface the chef→drop distance + comfort radius for the Mark-Ready carrier
	// decision: on chef_delivery orders AND on delivery orders the chef COULD
	// self-deliver (so the distance is visible before they choose "I'll deliver").
	// Soft only — the gate is the chef's per-order decision, not the distance.
	if order.FulfillmentType == models.FulfillmentChefDelivery ||
		(order.FulfillmentType == models.FulfillmentDelivery && chef.OffersSelfDelivery) {
		detail.SelfDeliveryDistanceKm = services.ComputeSelfDeliveryDistanceKm(
			chef, order.DeliveryLatitude, order.DeliveryLongitude,
		)
		detail.SelfDeliveryMaxDistanceKm = chef.SelfDeliveryMaxDistanceKm
	}

	c.JSON(http.StatusOK, detail)
}

// GetChefReviewsSummary returns aggregate review stats for the chef's
// reviews tab header. Mobile currently computes averageRating client-
// side from the paginated list; this endpoint gives the correct answer
// across ALL reviews (not just the first page) and a star distribution
// for any future histogram view.
// GET /chef/reviews/summary
func (h *ChefHandler) GetChefReviewsSummary(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"averageRating": 0,
			"totalReviews":  0,
			"distribution":  map[string]int{"5": 0, "4": 0, "3": 0, "2": 0, "1": 0},
		})
		return
	}

	type bucket struct {
		Rating int
		Count  int
	}
	var rows []bucket
	if err := database.DB.Model(&models.Review{}).
		Select("overall_rating AS rating, COUNT(*) AS count").
		Where("chef_id = ?", chef.ID).
		Group("overall_rating").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load review summary"})
		return
	}

	distribution := map[string]int{"5": 0, "4": 0, "3": 0, "2": 0, "1": 0}
	var totalReviews, ratingSum int
	for _, r := range rows {
		if r.Rating >= 1 && r.Rating <= 5 {
			distribution[strconv.Itoa(r.Rating)] = r.Count
			totalReviews += r.Count
			ratingSum += r.Rating * r.Count
		}
	}
	avg := 0.0
	if totalReviews > 0 {
		avg = float64(ratingSum) / float64(totalReviews)
	}

	c.JSON(http.StatusOK, gin.H{
		"averageRating": avg,
		"totalReviews":  totalReviews,
		"distribution":  distribution,
	})
}

// GetChefReviewsForDashboard returns all reviews for the authenticated chef
func (h *ChefHandler) GetChefReviewsForDashboard(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit

	var reviews []models.Review
	var total int64

	database.DB.Model(&models.Review{}).Where("chef_id = ?", chef.ID).Count(&total)

	if err := database.DB.Preload("Customer").
		Where("chef_id = ?", chef.ID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}

	responses := make([]models.ReviewResponse, len(reviews))
	for i, review := range reviews {
		responses[i] = review.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// ReplyToReview allows a chef to respond to a review
func (h *ChefHandler) ReplyToReview(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	reviewID := c.Param("reviewId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var review models.Review
	if err := database.DB.Where("id = ? AND chef_id = ?", reviewID, chef.ID).First(&review).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	var req struct {
		Response string `json:"response" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Response text is required"})
		return
	}

	now := time.Now()
	review.ChefResponse = req.Response
	review.ChefRespondedAt = &now

	if err := database.DB.Save(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save reply"})
		return
	}

	// Reload with customer for response DTO
	database.DB.Preload("Customer").First(&review, review.ID)

	c.JSON(http.StatusOK, review.ToResponse())
}

// GetChefSettings returns the chef's settings (creates defaults if not found)
func (h *ChefHandler) GetChefSettings(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var settings models.ChefSettings
	if err := database.DB.Where("chef_id = ?", chef.ID).First(&settings).Error; err != nil {
		// Create default settings
		settings = models.ChefSettings{
			ChefID:              chef.ID,
			AutoAcceptOrders:    false,
			AutoAcceptThreshold: 0,
			PushNewOrder:        true,
			PushOrderUpdate:     true,
			EmailDailySummary:   true,
			EmailWeeklyReport:   true,
			SmsNewOrder:         false,
		}
		database.DB.Create(&settings)
	}

	// Return in the shape the frontend expects. AuthProvider used to be
	// surfaced here for the legacy local-auth UI; with GIP owning auth the
	// frontend reads provider details from the auth-bff session instead.
	c.JSON(http.StatusOK, gin.H{
		"notifications": gin.H{
			"pushNewOrder":      settings.PushNewOrder,
			"pushOrderUpdate":   settings.PushOrderUpdate,
			"emailDailySummary": settings.EmailDailySummary,
			"emailWeeklyReport": settings.EmailWeeklyReport,
			"smsNewOrder":       settings.SmsNewOrder,
		},
		"autoAcceptOrders":    settings.AutoAcceptOrders,
		"autoAcceptThreshold": settings.AutoAcceptThreshold,
		"acceptingOrders":     chef.AcceptingOrders,
	})
}

// UpdateChefSettings updates the chef's settings
func (h *ChefHandler) UpdateChefSettings(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var req struct {
		Notifications struct {
			PushNewOrder      bool `json:"pushNewOrder"`
			PushOrderUpdate   bool `json:"pushOrderUpdate"`
			EmailDailySummary bool `json:"emailDailySummary"`
			EmailWeeklyReport bool `json:"emailWeeklyReport"`
			SmsNewOrder       bool `json:"smsNewOrder"`
		} `json:"notifications"`
		AutoAcceptOrders    bool    `json:"autoAcceptOrders"`
		AutoAcceptThreshold float64 `json:"autoAcceptThreshold"`
		AcceptingOrders     bool    `json:"acceptingOrders"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update acceptingOrders on chef profile. A manual toggle also clears any
	// active pause timer — the chef's explicit choice overrides "Back in X min".
	database.DB.Model(&chef).Updates(map[string]interface{}{
		"accepting_orders": req.AcceptingOrders,
		"paused_until":     nil,
	})

	// Upsert chef settings
	var settings models.ChefSettings
	if err := database.DB.Where("chef_id = ?", chef.ID).First(&settings).Error; err != nil {
		settings = models.ChefSettings{ChefID: chef.ID}
	}
	settings.AutoAcceptOrders = req.AutoAcceptOrders
	settings.AutoAcceptThreshold = req.AutoAcceptThreshold
	settings.PushNewOrder = req.Notifications.PushNewOrder
	settings.PushOrderUpdate = req.Notifications.PushOrderUpdate
	settings.EmailDailySummary = req.Notifications.EmailDailySummary
	settings.EmailWeeklyReport = req.Notifications.EmailWeeklyReport
	settings.SmsNewOrder = req.Notifications.SmsNewOrder
	database.DB.Save(&settings)

	c.JSON(http.StatusOK, gin.H{
		"notifications": gin.H{
			"pushNewOrder":      settings.PushNewOrder,
			"pushOrderUpdate":   settings.PushOrderUpdate,
			"emailDailySummary": settings.EmailDailySummary,
			"emailWeeklyReport": settings.EmailWeeklyReport,
			"smsNewOrder":       settings.SmsNewOrder,
		},
		"autoAcceptOrders":    settings.AutoAcceptOrders,
		"autoAcceptThreshold": settings.AutoAcceptThreshold,
		"acceptingOrders":     req.AcceptingOrders,
	})
}

// GetPayoutDetails returns the chef's payout configuration with masked bank details
func (h *ChefHandler) GetPayoutDetails(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Preload("User").Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// All sensitive payout data lives in GCP Secret Manager — read and mask for display
	vendorID := chef.ID.String()
	ctx := c.Request.Context()

	bankAccountName, _ := services.GetVendorSecret(ctx, vendorID, "bank-account-name")
	bankAccountNumber, _ := services.GetVendorSecret(ctx, vendorID, "bank-account-number")
	bankIFSC, _ := services.GetVendorSecret(ctx, vendorID, "bank-ifsc")
	upiID, _ := services.GetVendorSecret(ctx, vendorID, "upi-id")

	c.JSON(http.StatusOK, gin.H{
		"payoutMethod":      chef.PayoutMethod,
		"bankAccountName":   bankAccountName,
		"bankAccountNumber": maskBankAccount(bankAccountNumber),
		"bankIFSC":          bankIFSC,
		"upiId":             maskEmail(upiID),
		"razorpayConnected": chef.RazorpayAccountID != "",
		"razorpayAccountId": maskID(chef.RazorpayAccountID),
		"stripeConnected":   chef.StripeAccountID != "",
		"stripeAccountId":   maskID(chef.StripeAccountID),
		"paymentProvider":   chef.PaymentProvider,
		"payoutCountry":     chef.PayoutCountry,
	})
}

// SavePayoutDetails saves the chef's payout information.
// Sensitive fields (account number, UPI ID) are stored in GCP Secret Manager.
// Only masked values are stored in the database for display purposes.
// Also creates a Razorpay Route linked account so payments split directly to the chef.
func (h *ChefHandler) SavePayoutDetails(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		PayoutMethod      string `json:"payoutMethod" binding:"required"`
		BankAccountNumber string `json:"bankAccountNumber"`
		BankIFSC          string `json:"bankIFSC"`
		BankAccountName   string `json:"bankAccountName"`
		UpiID             string `json:"upiId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PayoutMethod != "bank_transfer" && req.PayoutMethod != "upi" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payoutMethod must be 'bank_transfer' or 'upi'"})
		return
	}

	if req.PayoutMethod == "bank_transfer" {
		if req.BankAccountNumber == "" || req.BankIFSC == "" || req.BankAccountName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bankAccountNumber, bankIFSC, and bankAccountName are required for bank_transfer"})
			return
		}
	}

	if req.PayoutMethod == "upi" {
		if req.UpiID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "upiId is required for upi payout method"})
			return
		}
	}

	var chef models.ChefProfile
	if err := database.DB.Preload("User").Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	vendorID := chef.ID.String()

	// DB stores ONLY the payout method (non-sensitive selector).
	// All sensitive fields (account number, IFSC, name, UPI) live exclusively in Secret Manager.
	chef.PayoutMethod = req.PayoutMethod
	chef.BankAccountNumber = "" // No sensitive data in DB
	chef.BankIFSC = ""          // Stored in Secret Manager
	chef.BankAccountName = ""   // Stored in Secret Manager
	chef.UpiID = ""             // Stored in Secret Manager

	if err := database.DB.Save(&chef).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save payout details"})
		return
	}

	log.Printf("Payout details saved for vendor %s (method: %s)", vendorID, req.PayoutMethod)

	// Store sensitive fields in GCP Secret Manager asynchronously
	// (Secret Manager creation can take several seconds on first call)
	go func() {
		ctx := context.Background()
		secretFields := map[string]string{
			"bank-account-number": req.BankAccountNumber,
			"bank-account-name":   req.BankAccountName,
			"bank-ifsc":           req.BankIFSC,
			"upi-id":              req.UpiID,
		}
		for field, value := range secretFields {
			if value != "" {
				if err := services.StoreVendorSecret(ctx, vendorID, field, value); err != nil {
					log.Printf("Warning: failed to store secret %s for vendor %s: %v", field, vendorID, err)
				}
			}
		}
		log.Printf("Secrets stored in Secret Manager for vendor %s", vendorID)
	}()

	// Create Razorpay Route linked account asynchronously if not already created
	razorpayConnected := chef.RazorpayAccountID != ""
	if !razorpayConnected {
		go func() {
			rz := services.GetRazorpay()
			if rz == nil {
				return
			}
			contactName := chef.User.FirstName + " " + chef.User.LastName
			linkedAcct, err := rz.CreateLinkedAccount(&services.LinkedAccountRequest{
				Email:        chef.User.Email,
				Phone:        chef.User.Phone,
				LegalName:    chef.BusinessName,
				BusinessType: "individual",
				ContactName:  contactName,
			})
			if err != nil {
				log.Printf("Failed to create Razorpay linked account for vendor %s", vendorID)
				return
			}
			database.DB.Model(&models.ChefProfile{}).Where("id = ?", chef.ID).Update("razorpay_account_id", linkedAcct.ID)
			log.Printf("Created Razorpay linked account for vendor %s", vendorID)
		}()
	}

	// Audit the payout change. NEVER store raw bank details in the audit row —
	// only the method + masked account so the trail is useful without leaking PII.
	services.LogAudit(c, "chef.payout.update", "chef", vendorID, nil, gin.H{
		"payoutMethod":      req.PayoutMethod,
		"bankAccountNumber": maskBankAccount(req.BankAccountNumber),
		"upiId":             maskEmail(req.UpiID),
	})

	resp := gin.H{
		"message":           "Payout details saved",
		"payoutMethod":      chef.PayoutMethod,
		"bankAccountName":   req.BankAccountName,
		"bankAccountNumber": maskBankAccount(req.BankAccountNumber),
		"bankIFSC":          req.BankIFSC,
		"upiId":             maskEmail(req.UpiID),
		"razorpayConnected": razorpayConnected,
		"razorpayAccountId": maskID(chef.RazorpayAccountID),
	}

	c.JSON(http.StatusOK, resp)
}

// GetChefAnalytics returns analytics data for the authenticated chef
func (h *ChefHandler) GetChefAnalytics(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// Parse period (7d, 30d, 90d)
	period := c.DefaultQuery("period", "7d")
	var days int
	switch period {
	case "30d":
		days = 30
	case "90d":
		days = 90
	default:
		days = 7
	}

	since := time.Now().AddDate(0, 0, -days)

	// ── Order & Revenue Trends (grouped by date) ──
	type dailyStat struct {
		Date    string  `json:"date"`
		Orders  int     `json:"orders"`
		Revenue float64 `json:"revenue"`
	}
	var dailyStats []dailyStat
	database.DB.Raw(`
		SELECT DATE(created_at) as date,
		       COUNT(*) as orders,
		       COALESCE(SUM(total), 0) as revenue
		FROM orders
		WHERE chef_id = ? AND created_at >= ? AND deleted_at IS NULL
		GROUP BY DATE(created_at)
		ORDER BY date
	`, chef.ID, since).Scan(&dailyStats)

	// Build label→value maps for the full date range
	dateMap := make(map[string]dailyStat)
	for _, ds := range dailyStats {
		dateMap[ds.Date] = ds
	}

	var orderLabels []string
	var orderData []int
	var revenueLabels []string
	var revenueData []float64

	for i := days - 1; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i)
		dateStr := d.Format("2006-01-02")
		var label string
		if days <= 7 {
			label = d.Format("Mon")
		} else {
			label = d.Format("Jan 2")
		}

		ds := dateMap[dateStr]
		orderLabels = append(orderLabels, label)
		orderData = append(orderData, ds.Orders)
		revenueLabels = append(revenueLabels, label)
		revenueData = append(revenueData, math.Round(ds.Revenue*100)/100)
	}

	// ── Popular Items (top 5 by order count) ──
	type popularItem struct {
		Name   string `json:"name"`
		Orders int    `json:"orders"`
	}
	var topItems []popularItem
	database.DB.Raw(`
		SELECT oi.name, SUM(oi.quantity) as orders
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE o.chef_id = ? AND o.created_at >= ? AND o.deleted_at IS NULL
		GROUP BY oi.name
		ORDER BY orders DESC
		LIMIT 5
	`, chef.ID, since).Scan(&topItems)

	// Calculate percentages
	var totalItemOrders int
	for _, it := range topItems {
		totalItemOrders += it.Orders
	}
	popularItemsResp := make([]gin.H, len(topItems))
	for i, it := range topItems {
		pct := 0
		if totalItemOrders > 0 {
			pct = int(math.Round(float64(it.Orders) / float64(totalItemOrders) * 100))
		}
		popularItemsResp[i] = gin.H{
			"name":       it.Name,
			"orders":     it.Orders,
			"percentage": pct,
		}
	}

	// ── Peak Hours ──
	type hourStat struct {
		Hour   int `json:"hour"`
		Orders int `json:"orders"`
	}
	var hourStats []hourStat
	database.DB.Raw(`
		SELECT EXTRACT(HOUR FROM created_at)::int as hour, COUNT(*) as orders
		FROM orders
		WHERE chef_id = ? AND created_at >= ? AND deleted_at IS NULL
		GROUP BY hour
		ORDER BY hour
	`, chef.ID, since).Scan(&hourStats)

	hourMap := make(map[int]int)
	for _, hs := range hourStats {
		hourMap[hs.Hour] = hs.Orders
	}
	peakHours := make([]gin.H, 0)
	for h := 8; h <= 22; h++ {
		label := fmt.Sprintf("%d %s", func() int {
			if h == 12 {
				return 12
			}
			if h > 12 {
				return h - 12
			}
			return h
		}(), func() string {
			if h >= 12 {
				return "PM"
			}
			return "AM"
		}())
		peakHours = append(peakHours, gin.H{
			"hour":   label,
			"orders": hourMap[h],
		})
	}

	// ── Revenue by Category ──
	type categoryStat struct {
		Category string  `json:"category"`
		Revenue  float64 `json:"revenue"`
	}
	var catStats []categoryStat
	database.DB.Raw(`
		SELECT COALESCE(mc.name, 'Uncategorized') as category,
		       COALESCE(SUM(oi.subtotal), 0) as revenue
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		LEFT JOIN menu_items mi ON mi.id = oi.menu_item_id
		LEFT JOIN menu_categories mc ON mc.id = mi.category_id
		WHERE o.chef_id = ? AND o.created_at >= ? AND o.deleted_at IS NULL
		GROUP BY mc.name
		ORDER BY revenue DESC
	`, chef.ID, since).Scan(&catStats)

	var totalCatRevenue float64
	for _, cs := range catStats {
		totalCatRevenue += cs.Revenue
	}
	revByCat := make([]gin.H, len(catStats))
	for i, cs := range catStats {
		pct := 0
		if totalCatRevenue > 0 {
			pct = int(math.Round(cs.Revenue / totalCatRevenue * 100))
		}
		revByCat[i] = gin.H{
			"category":   cs.Category,
			"revenue":    math.Round(cs.Revenue*100) / 100,
			"percentage": pct,
		}
	}

	// ── Summary headline metrics (#228): orders, revenue, AOV, repeat-rate ──
	totalOrders := 0
	var totalRevenue float64
	for i := range orderData {
		totalOrders += orderData[i]
		totalRevenue += revenueData[i]
	}
	aov := 0.0
	if totalOrders > 0 {
		aov = totalRevenue / float64(totalOrders)
	}

	// Prior period of equal length, for a trend delta.
	prevSince := time.Now().AddDate(0, 0, -2*days)
	var prevRevenue float64
	database.DB.Raw(`
		SELECT COALESCE(SUM(total), 0) FROM orders
		WHERE chef_id = ? AND created_at >= ? AND created_at < ? AND deleted_at IS NULL
	`, chef.ID, prevSince, since).Scan(&prevRevenue)

	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"orders":      totalOrders,
			"revenue":     math.Round(totalRevenue*100) / 100,
			"aov":         math.Round(aov*100) / 100,
			"repeatRate":  chefRepeatRate(chef.ID),
			"prevRevenue": math.Round(prevRevenue*100) / 100,
		},
		"orderTrends":       gin.H{"labels": orderLabels, "data": orderData},
		"revenueTrends":     gin.H{"labels": revenueLabels, "data": revenueData},
		"popularItems":      popularItemsResp,
		"peakHours":         peakHours,
		"revenueByCategory": revByCat,
	})
}

// chefRepeatRate returns the lifetime repeat-customer percentage for a chef:
// distinct customers with ≥2 orders ÷ distinct customers, ×100 (#228). Reused
// across the analytics surfaces.
func chefRepeatRate(chefID uuid.UUID) float64 {
	var row struct {
		Repeat int
		Total  int
	}
	database.DB.Raw(`
		SELECT
			COUNT(*) FILTER (WHERE cnt >= 2) AS repeat,
			COUNT(*) AS total
		FROM (
			SELECT customer_id, COUNT(*) AS cnt
			FROM orders
			WHERE chef_id = ? AND deleted_at IS NULL
			GROUP BY customer_id
		) t
	`, chefID).Scan(&row)
	if row.Total == 0 {
		return 0
	}
	return math.Round(float64(row.Repeat)/float64(row.Total)*1000) / 10
}
