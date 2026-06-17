package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// AdminListReviews lists reviews for moderation (#35). `?hidden=true` filters to
// hidden ones; otherwise returns recent reviews newest-first. GET /admin/reviews
func (h *AdminHandler) AdminListReviews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	q := database.DB.Model(&models.Review{})
	if c.Query("hidden") == "true" {
		q = q.Where("is_hidden = ?", true)
	}

	var total int64
	q.Count(&total)

	var reviews []models.Review
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load reviews"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": reviews,
		"pagination": gin.H{
			"page": page, "limit": limit, "total": total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total, "hasPrev": page > 1,
		},
	})
}

// AdminHideReview hides a review and recomputes the chef's rating so the hidden
// review no longer counts. Audited. PUT /admin/reviews/:id/hide
func (h *AdminHandler) AdminHideReview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review id"})
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	var review models.Review
	if err := database.DB.First(&review, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}
	if err := database.DB.Model(&review).Updates(map[string]interface{}{
		"is_hidden": true, "hidden_reason": req.Reason,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hide review"})
		return
	}
	updateChefRating(review.ChefID)
	services.LogAudit(c, "review.hide", "review", id.String(), nil, map[string]any{
		"reason": req.Reason, "chefId": review.ChefID.String(),
	})
	c.JSON(http.StatusOK, gin.H{"message": "Review hidden"})
}

// AdminUnhideReview restores a hidden review and recomputes the chef's rating.
// Audited. PUT /admin/reviews/:id/unhide
func (h *AdminHandler) AdminUnhideReview(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid review id"})
		return
	}
	var review models.Review
	if err := database.DB.First(&review, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}
	if err := database.DB.Model(&review).Updates(map[string]interface{}{
		"is_hidden": false, "hidden_reason": "",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore review"})
		return
	}
	updateChefRating(review.ChefID)
	services.LogAudit(c, "review.unhide", "review", id.String(), nil, map[string]any{
		"chefId": review.ChefID.String(),
	})
	c.JSON(http.StatusOK, gin.H{"message": "Review restored"})
}
