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
	go updateChefRating(order.ChefID)

	// Per-dish ratings (#145): optional `dishRatings` JSON form field —
	// [{ "menuItemId": "...", "rating": 1-5 }]. Each rolls up into the dish's
	// MenuItem.Rating so customers see per-dish scores, not just a chef average.
	if raw := c.PostForm("dishRatings"); raw != "" {
		var dishes []struct {
			MenuItemID string `json:"menuItemId"`
			Rating     int    `json:"rating"`
		}
		if err := json.Unmarshal([]byte(raw), &dishes); err == nil {
			affected := map[uuid.UUID]bool{}
			for _, d := range dishes {
				mid, perr := uuid.Parse(d.MenuItemID)
				if perr != nil || d.Rating < 1 || d.Rating > 5 {
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

// updateChefRating recalculates a chef's average rating and total reviews.
func updateChefRating(chefID uuid.UUID) {
	var stats struct {
		AvgRating    float64
		TotalReviews int64
	}
	// Exclude hidden reviews (#35 moderation) alongside unapproved/deleted — a
	// hidden review must not count toward the public rating.
	database.DB.Model(&models.Review{}).
		Where("chef_id = ? AND is_approved = ? AND is_hidden = ? AND deleted_at IS NULL", chefID, true, false).
		Select("COALESCE(AVG(overall_rating), 0) as avg_rating, COUNT(*) as total_reviews").
		Scan(&stats)

	database.DB.Model(&models.ChefProfile{}).Where("id = ?", chefID).
		Updates(map[string]interface{}{
			"rating":        stats.AvgRating,
			"total_reviews": stats.TotalReviews,
		})
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
