package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

// GetNotifications returns paginated notifications for the authenticated user
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	unreadOnly := c.Query("unread") == "true"

	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.Notification{}).Where("user_id = ?", userID)
	if unreadOnly {
		query = query.Where("is_read = ?", false)
	}

	var total int64
	query.Count(&total)

	var notifications []models.Notification
	query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&notifications)

	c.JSON(http.StatusOK, gin.H{
		"data": notifications,
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

// GetUnreadCount returns the count of unread notifications
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var count int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count)

	c.JSON(http.StatusOK, gin.H{"unreadCount": count})
}

// MarkAsRead marks a single notification as read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	notifID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	now := time.Now()
	result := database.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notifID, userID).
		Updates(map[string]interface{}{"is_read": true, "read_at": &now})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}

// MarkAllAsRead marks all notifications as read for the user
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	now := time.Now()

	database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{"is_read": true, "read_at": &now})

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}
