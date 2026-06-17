package models

import "time"

// ProcessedEvent is the consumer-side idempotency ledger. Before a durable
// JetStream consumer runs a handler it checks for a (Consumer, MsgID) row; after
// success it records one. This makes redelivery (after a crash, a Nak, or an
// AckWait expiry) a no-op instead of a double-send / double-write — JetStream is
// at-least-once, so consumers must be idempotent. See issue #138 / #134.
//
// The composite primary key is (Consumer, MsgID): the same event may legitimately
// be processed once per independent consumer group (e.g. notification-workers and
// push-workers both act on orders.updated).
type ProcessedEvent struct {
	// Consumer is the durable consumer / worker-group name.
	Consumer string `gorm:"type:varchar(64);primaryKey" json:"consumer"`
	// MsgID mirrors the OutboxEvent.MsgID carried in the Nats-Msg-Id header.
	MsgID string `gorm:"type:varchar(64);primaryKey" json:"msgId"`

	Subject     string    `gorm:"type:varchar(255)" json:"subject"`
	ProcessedAt time.Time `gorm:"autoCreateTime;index" json:"processedAt"`
}

// TableName pins the table name.
func (ProcessedEvent) TableName() string { return "processed_events" }
