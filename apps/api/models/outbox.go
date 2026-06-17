package models

import (
	"time"

	"github.com/google/uuid"
)

// OutboxStatus is the lifecycle state of a row in the transactional outbox.
type OutboxStatus string

const (
	// OutboxPending — written, not yet published to the broker.
	OutboxPending OutboxStatus = "pending"
	// OutboxPublishing — claimed by a relay worker, publish in flight.
	OutboxPublishing OutboxStatus = "publishing"
	// OutboxPublished — confirmed durably stored by JetStream (PubAck received).
	OutboxPublished OutboxStatus = "published"
	// OutboxFailed — exhausted max attempts; retained as a dead-letter for
	// inspection/replay. This is the publisher-side DLQ.
	OutboxFailed OutboxStatus = "failed"
)

// OutboxEvent is a single domain event durably staged in the database so it can
// be published to NATS JetStream exactly once, in the same transaction as the
// business state change that produced it. This eliminates the dual-write race
// where a process crashes between `tx.Commit()` and a fire-and-forget publish
// (see issue #131 / #121 — transactional outbox).
//
// The production DDL lives in tesserix-k8s (db-schema-bootstrap); this GORM
// model drives local-dev AutoMigrate and is the read/write surface in code.
type OutboxEvent struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// Subject is the NATS subject the event is published to (e.g. orders.created).
	Subject string `gorm:"type:varchar(255);not null" json:"subject"`

	// MsgID is the unique idempotency key. It is set as the JetStream
	// `Nats-Msg-Id` header so the broker dedups duplicate publishes (e.g. a relay
	// that crashed after publishing but before marking the row), and consumers
	// dedup redelivery against it.
	MsgID string `gorm:"type:varchar(64);not null;uniqueIndex" json:"msgId"`

	// AggregateType/AggregateID identify the entity that emitted the event
	// (e.g. "order"/<uuid>) — for tracing, debugging and replay.
	AggregateType string `gorm:"type:varchar(64);index" json:"aggregateType"`
	AggregateID   string `gorm:"type:varchar(64);index" json:"aggregateId"`

	// Payload is the JSON-encoded event body (opaque to the DB). Stored as text
	// so it round-trips through any driver without a jsonb cast.
	Payload string `gorm:"type:text;not null" json:"payload"`

	Status    OutboxStatus `gorm:"type:varchar(16);not null;default:'pending';index:idx_outbox_dispatch,priority:1" json:"status"`
	Attempts  int          `gorm:"not null;default:0" json:"attempts"`
	LastError string       `gorm:"type:text" json:"lastError,omitempty"`

	// NextRetryAt gates when a pending row becomes eligible for (re)dispatch.
	NextRetryAt time.Time `gorm:"not null;index:idx_outbox_dispatch,priority:2" json:"nextRetryAt"`

	CreatedAt   time.Time  `gorm:"autoCreateTime;index" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
}

// TableName pins the table name so it reads naturally in SQL/admin tooling.
func (OutboxEvent) TableName() string { return "outbox_events" }
