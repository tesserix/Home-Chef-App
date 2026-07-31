package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	natsclient "github.com/nats-io/nats.go"

	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// notifications_sse.go — the same per-user stream as StreamNotificationsWS, over
// Server-Sent Events.
//
// Two transports for one stream because they fail in different places: a WebSocket needs an
// upgrade that corporate proxies and some mobile networks drop, while SSE is an ordinary
// long-lived GET that anything able to serve HTTP can carry. Clients try the socket and fall
// back here, so a blocked upgrade costs latency rather than live updates.
//
// Frames are identical to the socket's, so the client's routing is shared between them.

// sseHeartbeatInterval keeps the connection warm. Idle SSE connections are reaped by proxies
// (and by iOS when nothing arrives), and a comment frame is the cheapest way to say alive.
const sseHeartbeatInterval = 25 * time.Second

// StreamNotificationsSSE streams the authenticated user's notifications as SSE.
// GET /api/v1/notifications/sse
func (h *NotificationHandler) StreamNotificationsSSE(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Must be set before the first flush or intermediaries buffer the whole response.
	// X-Accel-Buffering is for nginx-class proxies, which otherwise hold frames back.
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
		return
	}

	// Buffered so a slow reader cannot block the NATS callback, which is shared.
	frames := make(chan []byte, 32)

	subject := fmt.Sprintf("%s.%s", services.SubjectNotificationUser, userID.String())
	sub, err := services.GetNATSClient().Subscribe(subject, func(msg *natsclient.Msg) {
		var notif map[string]any
		if err := json.Unmarshal(msg.Data, &notif); err != nil {
			return
		}
		// Same envelope the socket sends, so clients parse one shape.
		notif["type"] = "new_notification"
		payload, err := json.Marshal(notif)
		if err != nil {
			return
		}
		select {
		case frames <- payload:
		default: // full — drop rather than stall every other subscriber
		}
	})
	if err != nil {
		log.Printf("SSE subscribe failed for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stream unavailable"})
		return
	}
	defer sub.Unsubscribe()

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	// c.Request.Context() closes when the client disconnects, which is the only exit that
	// matters — without it a dropped mobile connection leaks a goroutine and a subscription.
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
			// A comment line: valid SSE, ignored by clients, keeps proxies from reaping us.
			if _, err := fmt.Fprint(c.Writer, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
