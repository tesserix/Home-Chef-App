package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/lib/pq"
)

type ReviewHandler struct{}

func NewReviewHandler() *ReviewHandler {
	return &ReviewHandler{}
}

// orderReviewResponse is the customer's own review of an order, with the
// per-dish stars the public review response omits — the app needs them to
// render back what the customer actually submitted.
type orderReviewResponse struct {
	models.ReviewResponse
	DishRatings []models.DishRating `json:"dishRatings"`
}

// GetOrderReview returns the calling customer's review of an order, or null if
// they have not reviewed it. Without this the app could only discover an
// existing review by attempting a create and reading the 409 (#1046).
func (h *ReviewHandler) GetOrderReview(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid orderId"})
		return
	}

	var review models.Review
	err = database.DB.
		Where("order_id = ? AND customer_id = ?", orderID, userID).
		First(&review).Error
	if err != nil {
		// Not reviewed yet is the normal case, not a failure.
		c.JSON(http.StatusOK, gin.H{"review": nil})
		return
	}

	dishRatings := []models.DishRating{}
	database.DB.Where("review_id = ?", review.ID).Find(&dishRatings)

	c.JSON(http.StatusOK, gin.H{"review": orderReviewResponse{
		ReviewResponse: review.ToResponse(),
		DishRatings:    dishRatings,
	}})
}

// updateReviewRequest is a partial edit — every field is a pointer so an omitted
// rating keeps its stored value rather than resetting to zero.
type updateReviewRequest struct {
	OverallRating   *int    `json:"overallRating"`
	FoodRating      *int    `json:"foodRating"`
	DeliveryRating  *int    `json:"deliveryRating"`
	ValueRating     *int    `json:"valueRating"`
	PackagingRating *int    `json:"packagingRating"`
	HygieneRating   *int    `json:"hygieneRating"`
	Title           *string `json:"title"`
	Comment         *string `json:"comment"`
}

// UpdateReview rewrites the calling customer's own review. A customer could
// previously only report or block themselves — never correct what they wrote
// (#1047).
func (h *ReviewHandler) UpdateReview(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	reviewID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review id"})
		return
	}

	var req updateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var review models.Review
	// Scoped by customer so another customer's review reads as absent, not denied.
	if err := database.DB.Where("id = ? AND customer_id = ?", reviewID, userID).First(&review).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	updates := map[string]any{}
	ratings := map[string]*int{
		"overall_rating":   req.OverallRating,
		"food_rating":      req.FoodRating,
		"delivery_rating":  req.DeliveryRating,
		"value_rating":     req.ValueRating,
		"packaging_rating": req.PackagingRating,
		"hygiene_rating":   req.HygieneRating,
	}
	for column, v := range ratings {
		if v == nil {
			continue
		}
		// Overall carries the chef's public score, so it may never be cleared;
		// the optional sub-scores may be (0 = not given).
		lowest := 0
		if column == "overall_rating" {
			lowest = 1
		}
		if *v < lowest || *v > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ratings must be between 1 and 5"})
			return
		}
		updates[column] = *v
	}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Comment != nil {
		updates["comment"] = *req.Comment
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&review).Updates(updates).Error; err != nil {
			log.Printf("Failed to update review %s: %v", reviewID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update review"})
			return
		}
		updateChefRating(review.ChefID, review.Mode)
	}

	database.DB.First(&review, "id = ?", reviewID)
	c.JSON(http.StatusOK, review.ToResponse())
}

// DeleteReview withdraws the calling customer's own review, along with the
// per-dish stars it carried (#1047).
func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	reviewID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review id"})
		return
	}

	var review models.Review
	if err := database.DB.Where("id = ? AND customer_id = ?", reviewID, userID).First(&review).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	var dishes []models.DishRating
	database.DB.Where("review_id = ?", reviewID).Find(&dishes)

	if err := database.DB.Delete(&review).Error; err != nil {
		log.Printf("Failed to delete review %s: %v", reviewID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete review"})
		return
	}
	database.DB.Where("review_id = ?", reviewID).Delete(&models.DishRating{})

	updateChefRating(review.ChefID, review.Mode)
	for _, d := range dishes {
		recomputeMenuItemRating(d.MenuItemID)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// CreateReview creates a new review for an order.
// Accepts multipart/form-data with optional image uploads (up to 3).
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Parse multipart form (32 MB max)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	orderID := c.PostForm("orderId")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
		return
	}

	parsedOrderID, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid orderId"})
		return
	}

	// Verify the order belongs to this user and is delivered
	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", parsedOrderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.Status != models.OrderStatusDelivered {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only review delivered orders"})
		return
	}

	// Check if already reviewed
	var existingCount int64
	database.DB.Model(&models.Review{}).Where("order_id = ?", parsedOrderID).Count(&existingCount)
	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "This order has already been reviewed"})
		return
	}

	// Parse ratings
	overallRating := parseIntFormValue(c.PostForm("overallRating"), 0)
	foodRating := parseIntFormValue(c.PostForm("foodRating"), 0)
	deliveryRating := parseIntFormValue(c.PostForm("deliveryRating"), 0)
	valueRating := parseIntFormValue(c.PostForm("valueRating"), 0)
	// Ratings 2.0 sub-scores (#35), optional.
	packagingRating := parseIntFormValue(c.PostForm("packagingRating"), 0)
	hygieneRating := parseIntFormValue(c.PostForm("hygieneRating"), 0)

	if overallRating < 1 || overallRating > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "overallRating must be between 1 and 5"})
		return
	}

	title := c.PostForm("title")
	comment := c.PostForm("comment")

	// Handle image uploads (optional, max 3)
	var imageURLs []string
	form := c.Request.MultipartForm
	if form != nil && form.File["images"] != nil {
		files := form.File["images"]
		if len(files) > 3 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 3 images per review"})
			return
		}

		for _, header := range files {
			if header.Size > 5*1024*1024 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Each image must be under 5 MB"})
				return
			}

			contentType := header.Header.Get("Content-Type")
			if !services.IsImageContentType(contentType) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image type. Allowed: JPEG, PNG, WebP."})
				return
			}

			file, err := header.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read image"})
				return
			}
			defer file.Close()
			sniffed, err := sniffContentType(file)
			if err != nil || !services.IsImageContentType(sniffed) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Image contents don't match an allowed image type."})
				return
			}
			contentType = sniffed

			folder := fmt.Sprintf("reviews/%s", parsedOrderID.String())
			fileURL, err := services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
			if err != nil {
				log.Printf("Failed to upload review image: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image"})
				return
			}
			imageURLs = append(imageURLs, fileURL)
		}
	}

	review := models.Review{
		// A review of a test order belongs to the test partition, so a fake
		// order can never move a real kitchen's public rating.
		ModePartition:   order.ModePartition,
		OrderID:         parsedOrderID,
		CustomerID:      userID,
		ChefID:          order.ChefID,
		OverallRating:   overallRating,
		FoodRating:      foodRating,
		DeliveryRating:  deliveryRating,
		ValueRating:     valueRating,
		PackagingRating: packagingRating,
		HygieneRating:   hygieneRating,
		Title:           title,
		Comment:         comment,
		Images:          pq.StringArray(imageURLs),
	}

	if err := database.DB.Create(&review).Error; err != nil {
		log.Printf("Failed to create review: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create review"})
		return
	}

	// Update chef's rating stats
	go updateChefRating(order.ChefID, order.Mode)

	// Notify the chef of the new review (#422). Best-effort: the review is
	// already committed, so a staging failure must never fail the request.
	// Pluck into a SLICE: GORM does not scan a single column into a scalar
	// uuid.UUID — it leaves it zero, so the guard below rejected every chef and
	// no review notification was ever sent. Same trap as meal_plan_cron.go.
	var chefUserIDs []uuid.UUID
	plErr := database.DB.Model(&models.ChefProfile{}).
		Where("id = ?", order.ChefID).
		Pluck("user_id", &chefUserIDs).Error
	if plErr == nil && len(chefUserIDs) > 0 && chefUserIDs[0] != uuid.Nil {
		chefUserID := chefUserIDs[0]
		if err := services.EnqueueEvent(database.DB, services.SubjectReviewPosted, "review_posted", chefUserID, map[string]any{
			"review_id": review.ID.String(),
			"order_id":  order.ID.String(),
			"rating":    review.OverallRating,
		}); err != nil {
			log.Printf("review_posted: enqueue for chef %s: %v", chefUserID, err)
		}
	}

	// Per-dish ratings (#145): optional `dishRatings` JSON form field —
	// [{ "menuItemId": "...", "rating": 1-5 }]. Each rolls up into the dish's
	// MenuItem.Rating so customers see per-dish scores, not just a chef average.
	if raw := c.PostForm("dishRatings"); raw != "" {
		var dishes []struct {
			MenuItemID string `json:"menuItemId"`
			Rating     int    `json:"rating"`
		}
		if err := json.Unmarshal([]byte(raw), &dishes); err == nil {
			// SECURITY (audit #8): a dish rating may only target a MenuItem that was
			// actually on THIS order. Without this, a customer could deflate any
			// chef's dish ratings by submitting arbitrary menuItemIds (any chef's
			// dishes) from one legitimate order. Build the allowed set from the
			// order's own items.
			validDish := map[uuid.UUID]bool{}
			var orderItems []models.OrderItem
			database.DB.Where("order_id = ?", parsedOrderID).Find(&orderItems)
			for _, oi := range orderItems {
				validDish[oi.MenuItemID] = true
			}
			affected := map[uuid.UUID]bool{}
			for _, d := range dishes {
				mid, perr := uuid.Parse(d.MenuItemID)
				if perr != nil || d.Rating < 1 || d.Rating > 5 || !validDish[mid] {
					continue
				}
				dr := models.DishRating{ReviewID: review.ID, MenuItemID: mid, ChefID: order.ChefID, Rating: d.Rating}
				if err := database.DB.Create(&dr).Error; err == nil {
					affected[mid] = true
				}
			}
			for mid := range affected {
				recomputeMenuItemRating(mid)
			}
		}
	}

	// Reload with customer for response
	database.DB.Preload("Customer").First(&review, review.ID)

	c.JSON(http.StatusCreated, review.ToResponse())
}

// updateChefRating recalculates a chef's average rating and total reviews for
// the given mode.
//
// Recomputed per mode (services.RecalcChefRating) rather than over all reviews:
// the live figures still land on chef_profiles, which every customer-facing
// query already reads, while sandbox reviews accumulate only in chef_mode_stats.
// A fake order therefore cannot move a real kitchen's public rating.
// Hidden, unapproved and deleted reviews are excluded exactly as before.
func updateChefRating(chefID uuid.UUID, mode string) {
	if err := services.RecalcChefRating(database.DB, chefID, mode); err != nil {
		log.Printf("reviews: recompute %s rating for chef %s: %v", models.NormalizeMode(mode), chefID, err)
	}
}

// recomputeMenuItemRating averages a dish's per-dish ratings (#145) into the
// MenuItem's Rating + TotalReviews, so each dish carries its own score.
func recomputeMenuItemRating(menuItemID uuid.UUID) {
	var stats struct {
		AvgRating    float64
		TotalReviews int64
	}
	database.DB.Model(&models.DishRating{}).
		Where("menu_item_id = ?", menuItemID).
		Select("COALESCE(AVG(rating), 0) as avg_rating, COUNT(*) as total_reviews").
		Scan(&stats)

	database.DB.Model(&models.MenuItem{}).Where("id = ?", menuItemID).
		Updates(map[string]interface{}{
			"rating":        stats.AvgRating,
			"total_reviews": stats.TotalReviews,
		})
}

func parseIntFormValue(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		return fallback
	}
	return v
}
