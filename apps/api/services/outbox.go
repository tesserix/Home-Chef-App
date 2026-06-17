package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// outboxFailuresTotal counts events dead-lettered after exhausting publish
// retries, by subject — so event loss is alertable (issue #137).
var outboxFailuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "homechef_outbox_publish_failures_total",
	Help: "Outbox events dead-lettered after exhausting publish retries, by subject.",
}, []string{"subject"})

// RecordOutboxFailure increments the dead-letter counter for a subject.
func RecordOutboxFailure(subject string) { outboxFailuresTotal.WithLabelValues(subject).Inc() }

// ─────────────────────────────────────────────────────────────────────────────
// Transactional outbox — producer API
//
// Stage an event in the SAME database transaction as the state change that
// produced it, then let OutboxRelay publish it to JetStream with PubAck. This
// removes the dual-write race where a process dies between tx.Commit() and a
// fire-and-forget publish, so events are never silently lost (issue #131/#121).
//
// Prefer the Enqueue*Tx helpers (atomic with a business write). Pass database.DB
// to enqueue outside a transaction — still durable, just not atomically coupled.
// ─────────────────────────────────────────────────────────────────────────────

// EnqueueOutbox stages a raw payload on subject within tx.
func EnqueueOutbox(tx *gorm.DB, subject, aggregateType, aggregateID string, payload any) error {
	return enqueueOutbox(tx, subject, aggregateType, aggregateID, generateEventID(), payload)
}

func enqueueOutbox(tx *gorm.DB, subject, aggType, aggID, msgID string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("outbox marshal %s: %w", subject, err)
	}
	row := &models.OutboxEvent{
		Subject:       subject,
		MsgID:         msgID,
		AggregateType: aggType,
		AggregateID:   aggID,
		Payload:       string(raw),
		Status:        models.OutboxPending,
		NextRetryAt:   time.Now().UTC(),
	}
	if err := tx.Create(row).Error; err != nil {
		return fmt.Errorf("outbox enqueue %s: %w", subject, err)
	}
	return nil
}

// EnqueueEvent stages a generic Event — same wire shape PublishEvent produced,
// so consumers are unchanged. The event ID doubles as the Nats-Msg-Id.
func EnqueueEvent(tx *gorm.DB, subject, eventType string, userID uuid.UUID, data map[string]any) error {
	id := generateEventID()
	event := Event{ID: id, Type: eventType, Timestamp: time.Now().UTC(), UserID: userID, Data: data}
	return enqueueOutbox(tx, subject, "event", userID.String(), id, event)
}

// EnqueueOrderEvent stages an OrderEvent.
func EnqueueOrderEvent(tx *gorm.DB, subject string, order OrderEvent) error {
	return EnqueueOutbox(tx, subject, "order", order.OrderID.String(), order)
}

// EnqueueNotificationEvent stages a NotificationEvent on its channel subject.
func EnqueueNotificationEvent(tx *gorm.DB, notif NotificationEvent) error {
	return EnqueueOutbox(tx, notificationSubject(notif.Type), "notification", notif.UserID.String(), notif)
}

// ─────────────────────────────────────────────────────────────────────────────
// OutboxRelay — drains the outbox to JetStream
//
// Safe to run on every API replica: rows are claimed with SELECT … FOR UPDATE
// SKIP LOCKED, and each publish carries the row's MsgID as Nats-Msg-Id so
// JetStream dedups any double publish (at-least-once → effectively once).
// ─────────────────────────────────────────────────────────────────────────────

const (
	outboxStuckAfter = 2 * time.Minute // reclaim rows stuck in 'publishing'
	outboxKeepAfter  = 48 * time.Hour  // prune published rows after this
	outboxReapEvery  = 1 * time.Minute
)

// OutboxRelay publishes staged events and self-heals stuck/old rows.
type OutboxRelay struct {
	db          *gorm.DB
	nats        *NATSClient
	interval    time.Duration
	batchSize   int
	maxAttempts int
}

// NewOutboxRelay builds a relay from config with safe fallbacks.
func NewOutboxRelay(db *gorm.DB, n *NATSClient) *OutboxRelay {
	cfg := config.AppConfig
	r := &OutboxRelay{db: db, nats: n, interval: cfg.OutboxRelayInterval, batchSize: cfg.OutboxBatchSize, maxAttempts: cfg.OutboxMaxAttempts}
	if r.interval <= 0 {
		r.interval = time.Second
	}
	if r.batchSize <= 0 {
		r.batchSize = 100
	}
	if r.maxAttempts <= 0 {
		r.maxAttempts = 10
	}
	return r
}

// Start runs the dispatch loop and the reaper until ctx is cancelled.
func (r *OutboxRelay) Start(ctx context.Context) {
	go tick(ctx, r.interval, r.dispatchBatch)
	go tick(ctx, outboxReapEvery, r.reap)
	log.Printf("Outbox relay started (interval=%s batch=%d maxAttempts=%d)", r.interval, r.batchSize, r.maxAttempts)
}

// tick calls fn every d until ctx is done.
func tick(ctx context.Context, d time.Duration, fn func(context.Context)) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			fn(ctx)
		}
	}
}

func (r *OutboxRelay) dispatchBatch(ctx context.Context) {
	if !r.nats.IsConnected() {
		return
	}
	batch, err := r.claim(ctx)
	if err != nil {
		log.Printf("outbox: claim failed: %v", err)
		return
	}
	for i := range batch {
		e := &batch[i]
		if perr := r.nats.PublishJSRaw(ctx, e.Subject, e.MsgID, []byte(e.Payload)); perr != nil {
			r.markRetry(e, perr)
			continue
		}
		r.markPublished(e)
	}
}

// claim atomically moves up to batchSize due pending rows to 'publishing'.
func (r *OutboxRelay) claim(ctx context.Context) ([]models.OutboxEvent, error) {
	var batch []models.OutboxEvent
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND next_retry_at <= ?", models.OutboxPending, time.Now().UTC()).
			Order("created_at asc").Limit(r.batchSize).Find(&batch).Error; err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, len(batch))
		for i := range batch {
			ids[i] = batch[i].ID
		}
		return tx.Model(&models.OutboxEvent{}).Where("id IN ?", ids).
			Update("status", models.OutboxPublishing).Error
	})
	return batch, err
}

func (r *OutboxRelay) markPublished(e *models.OutboxEvent) {
	now := time.Now().UTC()
	if err := r.db.Model(&models.OutboxEvent{}).Where("id = ?", e.ID).
		Updates(map[string]any{"status": models.OutboxPublished, "published_at": now, "attempts": e.Attempts + 1}).Error; err != nil {
		log.Printf("outbox: mark published %s failed: %v", e.ID, err)
	}
}

func (r *OutboxRelay) markRetry(e *models.OutboxEvent, cause error) {
	attempts := e.Attempts + 1
	updates := map[string]any{"attempts": attempts, "last_error": cause.Error()}
	if attempts >= r.maxAttempts {
		updates["status"] = models.OutboxFailed
		log.Printf("outbox: event %s (%s) dead-lettered after %d attempts: %v", e.ID, e.Subject, attempts, cause)
		RecordOutboxFailure(e.Subject)
	} else {
		updates["status"] = models.OutboxPending
		updates["next_retry_at"] = time.Now().UTC().Add(outboxBackoff(attempts))
	}
	if err := r.db.Model(&models.OutboxEvent{}).Where("id = ?", e.ID).Updates(updates).Error; err != nil {
		log.Printf("outbox: mark retry %s failed: %v", e.ID, err)
	}
}

// reap reclaims rows stuck in 'publishing' (relay crashed mid-flight) and prunes
// old published rows so the table stays bounded.
func (r *OutboxRelay) reap(ctx context.Context) {
	now := time.Now().UTC()
	if err := r.db.WithContext(ctx).Model(&models.OutboxEvent{}).
		Where("status = ? AND updated_at < ?", models.OutboxPublishing, now.Add(-outboxStuckAfter)).
		Updates(map[string]any{"status": models.OutboxPending, "next_retry_at": now}).Error; err != nil {
		log.Printf("outbox: reclaim stuck rows failed: %v", err)
	}
	if err := r.db.WithContext(ctx).
		Where("status = ? AND published_at < ?", models.OutboxPublished, now.Add(-outboxKeepAfter)).
		Delete(&models.OutboxEvent{}).Error; err != nil {
		log.Printf("outbox: prune published rows failed: %v", err)
	}
}

// outboxBackoff is exponential with a 60s ceiling: 2s, 4s, 8s … 60s.
func outboxBackoff(attempt int) time.Duration {
	d := time.Second * time.Duration(int64(1)<<min(attempt, 6))
	if d > time.Minute {
		d = time.Minute
	}
	return d
}
