package handlers

import (
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
		OrderID:        parsedOrderID,
		CustomerID:     userID,
		ChefID:         order.ChefID,
		OverallRating:  overallRating,
		FoodRating:     foodRating,
		DeliveryRating: deliveryRating,
		ValueRating:    valueRating,
		Title:          title,
		Comment:        comment,
		Images:         pq.StringArray(imageURLs),
	}

	if err := database.DB.Create(&review).Error; err != nil {
		log.Printf("Failed to create review: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create review"})
		return
	}

	// Update chef's rating stats
	go updateChefRating(order.ChefID)

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
	database.DB.Model(&models.Review{}).
		Where("chef_id = ? AND is_approved = ? AND deleted_at IS NULL", chefID, true).
		Select("COALESCE(AVG(overall_rating), 0) as avg_rating, COUNT(*) as total_reviews").
		Scan(&stats)

	database.DB.Model(&models.ChefProfile{}).Where("id = ?", chefID).
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
