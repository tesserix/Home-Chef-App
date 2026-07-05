package services

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// ErrWebhookPermanent marks an inbound webhook that can never be processed no
// matter how many times the provider redelivers it — a malformed payload, an
// unmapped provider status, a missing id. The webhook handler ACKs these (200)
// and KEEPS the dedup claim rather than returning 500 into a retry storm.
var ErrWebhookPermanent = errors.New("webhook: permanent, un-processable event")

// maxWebhookEventID bounds the dedup key to processed_events.msg_id (varchar(64)).
// Postgres ERRORS on an over-length insert (→ 500 poison-loop); sqlite silently
// accepts it, so the cap is enforced here, not by the column.
const maxWebhookEventID = 64

// ClaimWebhookEvent atomically claims (consumer, eventID) in the processed_events
// idempotency ledger (the same table the NATS consumer backbone uses). Returns
// firstTime=true when this is the first time the event is seen — the caller should
// process it — and false when it's a replay the caller should skip. Atomic via
// INSERT ... ON CONFLICT DO NOTHING + RowsAffected, so there is no check-then-act
// race under concurrent deliveries.
func ClaimWebhookEvent(db *gorm.DB, consumer, eventID, subject string) (bool, error) {
	rec := &models.ProcessedEvent{Consumer: consumer, MsgID: eventID, Subject: subject}
	res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(rec)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// ReleaseWebhookEvent removes a claim so a provider retry re-processes the event
// after a TRANSIENT dispatch failure (instead of the retry being silently
// deduped). Best-effort: a failure here only means the retry is deduped (never a
// double-process), so it's logged rather than surfaced.
func ReleaseWebhookEvent(db *gorm.DB, consumer, eventID string) {
	if err := db.Where("consumer = ? AND msg_id = ?", consumer, eventID).
		Delete(&models.ProcessedEvent{}).Error; err != nil {
		log.Printf("webhook dedup: release claim failed for %s/%s: %v", consumer, eventID, err)
	}
}

// WebhookEventID picks a stable dedup key that always fits msg_id (varchar(64)). A
// provider-supplied event-id header wins when present and short enough; otherwise
// (empty, or long enough to overflow the column) it falls back to a hash of the
// raw body — so an EXACT replay dedups while any genuinely-distinct event (new
// status, new timestamp, moved coords) yields a different id and is not deduped.
func WebhookEventID(headerID string, body []byte) string {
	if headerID != "" && len(headerID) <= maxWebhookEventID {
		return headerID
	}
	sum := sha256.Sum256(body)
	return "h:" + base64.RawURLEncoding.EncodeToString(sum[:]) // 2 + 43 = 45 chars
}
