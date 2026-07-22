package handlers

// chef_availability.go — timed pause for a chef's kitchen.
//   POST /chef/availability/pause   {minutes: 15|30|60}
//   POST /chef/availability/resume
//
// "Pause" sets accepting_orders=false AND paused_until=now+minutes, so every
// existing accepting_orders check keeps blocking orders for the duration. The
// auto-resume cron (services.StartAvailabilityResumeCron) flips it back on when
// the timer elapses; the chef can also resume early via /resume.

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// allowedPauseMinutes are the only durations the UI offers; validated here so
// a client can't pause indefinitely.
var allowedPauseMinutes = map[int]bool{15: true, 30: true, 60: true}

// ChefAvailabilityHandler handles timed pause/resume.
type ChefAvailabilityHandler struct{}

// NewChefAvailabilityHandler constructs the handler.
func NewChefAvailabilityHandler() *ChefAvailabilityHandler {
	return &ChefAvailabilityHandler{}
}

type availabilityResponse struct {
	AcceptingOrders bool       `json:"acceptingOrders"`
	PausedUntil     *time.Time `json:"pausedUntil,omitempty"`
}

// PauseReceiving temporarily closes the kitchen for {15,30,60} minutes.
func (h *ChefAvailabilityHandler) PauseReceiving(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var req struct {
		Minutes int `json:"minutes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !allowedPauseMinutes[req.Minutes] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "minutes must be 15, 30, or 60"})
		return
	}

	until := time.Now().UTC().Add(time.Duration(req.Minutes) * time.Minute)
	if err := database.DB.Model(&models.ChefProfile{}).
		Where("id = ?", chef.ID).
		Updates(map[string]interface{}{
			"accepting_orders": false,
			"paused_until":     until,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to pause"})
		return
	}

	services.LogAudit(c, "chef.availability.pause", "chef", chef.ID.String(),
		nil, gin.H{"minutes": req.Minutes, "pausedUntil": until})

	c.JSON(http.StatusOK, availabilityResponse{AcceptingOrders: false, PausedUntil: &until})
}

// ResumeReceiving reopens the kitchen immediately, clearing any pause timer.
func (h *ChefAvailabilityHandler) ResumeReceiving(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	if err := database.DB.Model(&models.ChefProfile{}).
		Where("id = ?", chef.ID).
		Updates(map[string]interface{}{
			"accepting_orders": true,
			"paused_until":     nil,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resume"})
		return
	}

	if err := services.EnqueueChefAvailabilityChanged(database.DB, chef.ID, true); err != nil {
		log.Printf("chef availability event (resume) for %s: %v", chef.ID, err)
	}

	services.LogAudit(c, "chef.availability.resume", "chef", chef.ID.String(), nil, nil)

	c.JSON(http.StatusOK, availabilityResponse{AcceptingOrders: true, PausedUntil: nil})
}
