package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	natsclient "github.com/nats-io/nats.go"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// order_track_sse.go — live driver location over Server-Sent Events, the same
// stream TrackOrderWS carries.
//
// The socket is unreachable from the mobile apps: React Native does TLS through
// SocketRocket/CFStream and that handshake fails outright against our edge
// (close 1006 / OSStatus -9836), so no HTTP request is ever issued and the
// attempts appear nowhere in the API logs. Notifications and order status were
// moved to SSE when that was found (#982/#983); live tracking was not, and kept
// failing until its budget was spent and the screen fell back to REST polling —
// 33 consecutive failures on one order in the 4 Aug run.
//
// Frames are the raw NATS payloads, byte-for-byte what the socket sends, so the
// two transports stay one stream and the client parses one shape.

// resolveTrackedDeliveryID authorises a tracking stream and returns the delivery
// whose location it may carry. Shared by both transports so they can never
// disagree about who is allowed to watch a driver move.
//
// Writes the error response itself; ok=false means the caller must simply return.
func resolveTrackedDeliveryID(c *gin.Context) (string, bool) {
	orderID := c.Param("id")
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	// Verify the customer owns this order and load the delivery relationship.
	var order models.Order
	if err := database.DB.Preload("Delivery").
		Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order_not_found", "message": "Order not found"})
		return "", false
	}
	if order.Delivery == nil || order.Delivery.ID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no_active_delivery", "message": "No active delivery for this order"})
		return "", false
	}
	return order.Delivery.ID.String(), true
}

// TrackOrderSSE streams the order's driver-location updates as SSE.
// GET /api/v1/orders/:id/track/sse
func (h *OrderHandler) TrackOrderSSE(c *gin.Context) {
	deliveryID, ok := resolveTrackedDeliveryID(c)
	if !ok {
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

	// Go sends no headers until the first write, so without an opening frame the
	// client sees nothing — not even a status — until the driver first moves.
	// Every layer in between reads that silence as a dead connection.
	fmt.Fprint(c.Writer, ": connected\n\n")
	flusher.Flush()

	// Buffered so a slow reader cannot block the shared NATS callback.
	frames := make(chan []byte, 32)

	subject := fmt.Sprintf("%s.%s", services.SubjectDeliveryLocation, deliveryID)
	sub, err := services.GetNATSClient().Subscribe(subject, func(msg *natsclient.Msg) {
		select {
		case frames <- msg.Data:
		default: // full — drop this position rather than stall the dispatcher
		}
	})
	if err != nil {
		log.Printf("tracking SSE subscribe failed for delivery %s: %v", deliveryID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stream unavailable"})
		return
	}
	defer sub.Unsubscribe()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	// The request context closes when the client disconnects — the only exit that
	// matters, or a dropped mobile connection leaks a goroutine and a subscription.
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
