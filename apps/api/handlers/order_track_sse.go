package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
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
// An empty deliveryID with ok=true means the order is legitimately trackable but
// has no driver yet — the common case for most of an order's life. Refusing that
// with a 400 was wrong: the client opened the stream on every focused order,
// spent its retry budget on refusals, and was on REST polling by the time a
// driver actually appeared.
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
		return "", true
	}
	return order.Delivery.ID.String(), true
}

// deliveryIDForOrder re-reads the order's delivery. Callers must already have
// authorised the order.
func deliveryIDForOrder(orderID string) string {
	var d models.Delivery
	if err := database.DB.Select("id").Where("order_id = ?", orderID).First(&d).Error; err != nil {
		return ""
	}
	if d.ID == uuid.Nil {
		return ""
	}
	return d.ID.String()
}

// subscribeTracking relays an order's driver-location frames to send, and is
// shared by both transports so they carry the identical stream.
//
// When the order has no driver yet it waits on delivery.assigned instead of
// refusing, so the map goes live the moment a driver accepts. Returns a stop
// func; an error means NATS is unavailable and the stream cannot be served.
func subscribeTracking(orderID, deliveryID string, send func([]byte)) (func(), error) {
	var mu sync.Mutex
	var locSub, assignedSub *natsclient.Subscription

	// Idempotent: the assignment event and the re-read below can race, and only
	// one location subscription may win.
	subscribeLocation := func(id string) error {
		mu.Lock()
		defer mu.Unlock()
		if locSub != nil {
			return nil
		}
		s, err := services.GetNATSClient().Subscribe(
			fmt.Sprintf("%s.%s", services.SubjectDeliveryLocation, id),
			func(msg *natsclient.Msg) { send(msg.Data) },
		)
		if err != nil {
			return err
		}
		locSub = s
		return nil
	}

	stop := func() {
		mu.Lock()
		defer mu.Unlock()
		if assignedSub != nil {
			assignedSub.Unsubscribe()
		}
		if locSub != nil {
			locSub.Unsubscribe()
		}
	}

	if deliveryID != "" {
		if err := subscribeLocation(deliveryID); err != nil {
			return nil, err
		}
		return stop, nil
	}

	s, err := services.GetNATSClient().Subscribe(services.SubjectDeliveryAssigned, func(msg *natsclient.Msg) {
		id, ok := assignedDeliveryForOrder(msg.Data, orderID)
		if !ok {
			return
		}
		if err := subscribeLocation(id); err != nil {
			log.Printf("tracking: location subscribe failed for delivery %s: %v", id, err)
		}
	})
	if err != nil {
		return nil, err
	}
	mu.Lock()
	assignedSub = s
	mu.Unlock()

	// A driver can be assigned between the ownership read and the subscription
	// above; re-read now that such an event would be caught either way.
	if id := deliveryIDForOrder(orderID); id != "" {
		if err := subscribeLocation(id); err != nil {
			stop()
			return nil, err
		}
	}
	return stop, nil
}

// assignedDeliveryForOrder reads a delivery.assigned event and reports the
// delivery it created, when that assignment belongs to orderID.
func assignedDeliveryForOrder(payload []byte, orderID string) (string, bool) {
	var evt services.Event
	if err := json.Unmarshal(payload, &evt); err != nil {
		return "", false
	}
	if id, _ := evt.Data["order_id"].(string); id != orderID {
		return "", false
	}
	deliveryID, _ := evt.Data["delivery_id"].(string)
	if deliveryID == "" {
		return "", false
	}
	return deliveryID, true
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

	stop, err := subscribeTracking(c.Param("id"), deliveryID, func(data []byte) {
		select {
		case frames <- data:
		default: // full — drop this position rather than stall the dispatcher
		}
	})
	if err != nil {
		log.Printf("tracking SSE subscribe failed for order %s: %v", c.Param("id"), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stream unavailable"})
		return
	}
	defer stop()

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
