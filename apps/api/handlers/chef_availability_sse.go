package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	natsclient "github.com/nats-io/nats.go"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// chef_availability_sse.go — a kitchen's live open/closed state over SSE, the
// same stream StreamChefAvailabilityWS carries.
//
// The socket is unreachable from the mobile apps: React Native does TLS through
// SocketRocket/CFStream and that handshake fails outright against our edge
// (#982), so the customer app spent its retry budget on calls that never landed
// and fell back to a 60-second poll — which is most of the staleness #970 set
// out to remove. Notifications, order status and driver tracking are on SSE for
// the same reason; this is the last of the four.
//
// Like the socket, this route authenticates OPTIONALLY: browsing is guest-
// friendly and a kitchen's open/closed state is public.

// availabilityFrame is the frame both transports send, so clients parse one shape.
func availabilityFrame(chefID uuid.UUID, acceptingOrders bool) []byte {
	payload, _ := json.Marshal(map[string]any{
		"type":            "availability",
		"chefId":          chefID.String(),
		"acceptingOrders": acceptingOrders,
	})
	return payload
}

// StreamChefAvailabilitySSE streams one chef's availability changes as SSE.
// GET /api/v1/chefs/:id/availability/sse
func (h *ChefAvailabilityHandler) StreamChefAvailabilitySSE(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}

	// Must be set before the first flush or intermediaries buffer the whole response.
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	fmt.Fprint(c.Writer, ": connected\n\n")
	flusher.Flush()

	// Current state on connect, so a client that joined after a toggle is right
	// immediately rather than waiting for the next one — same as the socket.
	var chef models.ChefProfile
	if err := database.DB.Select("id", "accepting_orders").First(&chef, "id = ?", chefID).Error; err == nil {
		fmt.Fprintf(c.Writer, "data: %s\n\n", availabilityFrame(chefID, chef.AcceptingOrders))
		flusher.Flush()
	}

	frames := make(chan []byte, 16)

	sub, err := services.GetNATSClient().Subscribe(services.SubjectChefAvailabilityChanged, func(msg *natsclient.Msg) {
		var ev services.ChefAvailabilityEvent
		if jerr := json.Unmarshal(msg.Data, &ev); jerr != nil || ev.ChefID != chefID {
			return
		}
		select {
		case frames <- availabilityFrame(ev.ChefID, ev.AcceptingOrders):
		default: // full — drop rather than stall the shared dispatcher
		}
	})
	if err != nil {
		log.Printf("chef availability SSE subscribe failed for chef %s: %v", chefID, err)
		return
	}
	defer sub.Unsubscribe()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-frames:
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", frame); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprint(c.Writer, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
