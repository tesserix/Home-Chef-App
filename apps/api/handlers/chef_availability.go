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
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	natsclient "github.com/nats-io/nats.go"
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

	if verificationGateBlocks(c, &chef, true) {
		return
	}
	if payoutGateBlocks(c, &chef, true) {
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

// StreamChefAvailabilityWS pushes a chef's open/closed state to customers who are
// looking at that kitchen right now.
//
// chef.availability_changed was published but had nothing subscribed to it, so a
// customer's screen kept saying "Open" until they pulled to refresh — and they
// only found out at Place Order (#970). This subscribes to the same subject the
// outbox relay publishes and forwards the events for this chef.
// GET /ws/chefs/:id/availability
func (h *ChefAvailabilityHandler) StreamChefAvailabilityWS(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}

	conn, err := notifWSUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Chef availability WS upgrade failed for chef %s: %v", chefID, err)
		return
	}
	defer conn.Close()

	// Send the current state on connect so a client that joined after a toggle
	// is correct immediately rather than waiting for the next one.
	var chef models.ChefProfile
	if err := database.DB.Select("id", "accepting_orders").First(&chef, "id = ?", chefID).Error; err == nil {
		conn.WriteMessage(websocket.TextMessage, availabilityFrame(chefID, chef.AcceptingOrders))
	}

	writeCh := make(chan []byte, 16)
	defer close(writeCh)

	go func() {
		for msg := range writeCh {
			if werr := conn.WriteMessage(websocket.TextMessage, msg); werr != nil {
				return
			}
		}
	}()

	sub, err := services.GetNATSClient().Subscribe(services.SubjectChefAvailabilityChanged, func(msg *natsclient.Msg) {
		var ev services.ChefAvailabilityEvent
		if jerr := json.Unmarshal(msg.Data, &ev); jerr != nil || ev.ChefID != chefID {
			return
		}
		select {
		case writeCh <- availabilityFrame(ev.ChefID, ev.AcceptingOrders):
		default:
		}
	})
	if err != nil {
		log.Printf("NATS subscribe failed for chef availability %s: %v", chefID, err)
		return
	}
	defer sub.Unsubscribe()

	// Block until the client goes away; reads double as the disconnect signal.
	for {
		if _, _, rerr := conn.ReadMessage(); rerr != nil {
			return
		}
	}
}
