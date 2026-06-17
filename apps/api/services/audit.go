package services

import (
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/logger"
	"github.com/homechef/api/models"
)

// LogAudit writes an AuditLog row. Non-blocking on error — audit failures
// must never break the request that triggered them, so callers ignore the
// return value and we log internally.
//
// action     short verb, e.g. "chef.verify", "user.suspend", "security.policy.update"
// entityType the kind of resource being acted on: "chef", "user", "policy"
// entityID   opaque string id of the resource; optional
// oldValue   JSON-marshalable previous state; optional (nil skips)
// newValue   JSON-marshalable next state; optional (nil skips)
func LogAudit(c *gin.Context, action, entityType, entityID string, oldValue, newValue any) {
	logAudit(c, true, action, entityType, entityID, oldValue, newValue)
}

// LogSystemAudit writes an AuditLog row for a SYSTEM-triggered action — one with
// no human actor, e.g. the automatic FSSAI payout freeze that fires during a
// customer's order payment (#93). It mirrors LogAudit but deliberately never
// records a UserID, so the entry isn't misattributed to whichever customer's
// request happened to trigger it. IP / correlation-id are still captured from c
// (when present) so the event stays traceable back to the request that caused it.
func LogSystemAudit(c *gin.Context, action, entityType, entityID string, oldValue, newValue any) {
	logAudit(c, false, action, entityType, entityID, oldValue, newValue)
}

func logAudit(c *gin.Context, recordActor bool, action, entityType, entityID string, oldValue, newValue any) {
	if action == "" {
		return
	}
	entry := models.AuditLog{
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
	}

	if c != nil {
		// ClientIP / UserAgent both touch c.Request — guard so callers on a
		// background goroutine with a faked context don't panic.
		if c.Request != nil {
			entry.IPAddress = c.ClientIP()
			entry.UserAgent = c.Request.UserAgent()
		}
		if recordActor {
			if v, ok := c.Get("userID"); ok {
				if uid, ok := v.(uuid.UUID); ok {
					entry.UserID = &uid
				}
			}
		}
		if c.Request != nil {
			entry.CorrelationID = logger.CorrelationIDFromContext(c.Request.Context())
		}
	}

	if oldValue != nil {
		if b, err := json.Marshal(oldValue); err == nil {
			entry.OldValue = string(b)
		}
	}
	if newValue != nil {
		if b, err := json.Marshal(newValue); err == nil {
			entry.NewValue = string(b)
		}
	}

	if err := database.DB.Create(&entry).Error; err != nil {
		log.Printf("audit: failed to write log entry %q: %v", action, err)
	}
}
