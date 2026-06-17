package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/homechef/api/models"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// msgIDHeader is the JetStream dedup header the outbox sets on every event and
// the consumer reads for idempotency.
const msgIDHeader = "Nats-Msg-Id"

// consumerDeadLetters counts messages dead-lettered after exhausting retries.
var consumerDeadLetters = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "homechef_consumer_dead_letters_total",
	Help: "Messages dead-lettered after exhausting durable-consumer retries, by durable.",
}, []string{"durable"})

// RecordConsumerDeadLetter increments the dead-letter counter for a durable.
func RecordConsumerDeadLetter(durable string) { consumerDeadLetters.WithLabelValues(durable).Inc() }

// EventHandler processes one event. Returning nil acks the message; returning an
// error triggers a backed-off redelivery, and after MaxDeliver attempts the
// message is dead-lettered. Handlers MUST be idempotent — JetStream is
// at-least-once (the framework also dedups via the ProcessedEvent ledger).
type EventHandler func(ctx context.Context, subject string, data []byte) error

// ConsumerSpec declaratively describes one durable JetStream consumer.
type ConsumerSpec struct {
	Stream     string        // stream to bind to (e.g. "ORDERS")
	Durable    string        // durable name; also the idempotency namespace
	Subjects   []string      // filter subjects (subset of the stream's subjects)
	Handler    EventHandler  // event processor
	AckWait    time.Duration // redelivery timeout (default 30s)
	MaxDeliver int           // attempts before dead-letter (default 5)
}

// ConsumerManager creates durable JetStream consumers and runs their consume
// loops with idempotency, retry and dead-lettering. One per process.
type ConsumerManager struct {
	nats *NATSClient
	db   *gorm.DB
	mu   sync.Mutex
	ccs  []jetstream.ConsumeContext
}

// NewConsumerManager builds a manager bound to the NATS client and DB.
func NewConsumerManager(n *NATSClient, db *gorm.DB) *ConsumerManager {
	return &ConsumerManager{nats: n, db: db}
}

// RegisterAll registers a batch of specs, returning the first error.
func (m *ConsumerManager) RegisterAll(ctx context.Context, specs ...ConsumerSpec) error {
	for _, s := range specs {
		if err := m.Register(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// Register creates/updates the durable consumer and starts consuming. Idempotent
// — re-registering the same durable just rebinds the consume loop.
func (m *ConsumerManager) Register(ctx context.Context, spec ConsumerSpec) error {
	if spec.AckWait <= 0 {
		spec.AckWait = 30 * time.Second
	}
	if spec.MaxDeliver <= 0 {
		spec.MaxDeliver = 5
	}

	cons, err := m.nats.CreateConsumer(ctx, spec.Stream, jetstream.ConsumerConfig{
		Durable:        spec.Durable,
		AckPolicy:      jetstream.AckExplicitPolicy,
		AckWait:        spec.AckWait,
		MaxDeliver:     spec.MaxDeliver,
		FilterSubjects: spec.Subjects,
		DeliverPolicy:  jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return fmt.Errorf("create consumer %s/%s: %w", spec.Stream, spec.Durable, err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) { m.handle(ctx, spec, msg) })
	if err != nil {
		return fmt.Errorf("consume %s/%s: %w", spec.Stream, spec.Durable, err)
	}

	m.mu.Lock()
	m.ccs = append(m.ccs, cc)
	m.mu.Unlock()
	log.Printf("[consumer] %s/%s consuming %v", spec.Stream, spec.Durable, spec.Subjects)
	return nil
}

// handle runs one message through idempotency → handler → ack/retry/dead-letter.
func (m *ConsumerManager) handle(ctx context.Context, spec ConsumerSpec, msg jetstream.Msg) {
	msgID := msgIDOf(msg)

	if msgID != "" && m.alreadyProcessed(spec.Durable, msgID) {
		ack(msg)
		return
	}

	if err := spec.Handler(ctx, msg.Subject(), msg.Data()); err != nil {
		md, _ := msg.Metadata()
		if md != nil && md.NumDelivered >= uint64(spec.MaxDeliver) {
			m.deadLetter(ctx, spec, msg, err)
			ack(msg) // terminal: remove from the source stream (now in the DLQ)
			return
		}
		log.Printf("[consumer] %s/%s handler error (delivery %d/%d), retrying: %v",
			spec.Stream, spec.Durable, deliveries(md), spec.MaxDeliver, err)
		if nerr := msg.NakWithDelay(consumerBackoff(md)); nerr != nil {
			log.Printf("[consumer] %s/%s nak failed: %v", spec.Stream, spec.Durable, nerr)
		}
		return
	}

	if msgID != "" {
		m.recordProcessed(spec.Durable, msgID, msg.Subject())
	}
	ack(msg)
}

// deadLetter republishes a poison message to dlq.<stream>.<durable> (captured by
// the DLQ stream) for inspection and replay.
func (m *ConsumerManager) deadLetter(ctx context.Context, spec ConsumerSpec, msg jetstream.Msg, cause error) {
	subject := fmt.Sprintf("%s.%s.%s", DLQSubjectPrefix, spec.Stream, spec.Durable)
	envelope := map[string]any{
		"stream":          spec.Stream,
		"durable":         spec.Durable,
		"originalSubject": msg.Subject(),
		"error":           cause.Error(),
		"deadLetteredAt":  time.Now().UTC(),
		"payload":         json.RawMessage(msg.Data()),
	}
	if err := m.nats.PublishJS(ctx, subject, "dlq-"+msgIDOf(msg), envelope); err != nil {
		log.Printf("[consumer] %s/%s DLQ publish failed (original error: %v): %v", spec.Stream, spec.Durable, cause, err)
	} else {
		log.Printf("[consumer] %s/%s dead-lettered to %s after %d deliveries: %v", spec.Stream, spec.Durable, subject, spec.MaxDeliver, cause)
	}
	RecordConsumerDeadLetter(spec.Durable)
}

// Stop halts all consume loops. Safe on a nil manager.
func (m *ConsumerManager) Stop() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cc := range m.ccs {
		cc.Stop()
	}
	m.ccs = nil
	log.Println("[consumer] all consumers stopped")
}

// alreadyProcessed reports whether this durable already handled msgID.
func (m *ConsumerManager) alreadyProcessed(durable, msgID string) bool {
	var count int64
	m.db.Model(&models.ProcessedEvent{}).
		Where("consumer = ? AND msg_id = ?", durable, msgID).Count(&count)
	return count > 0
}

// recordProcessed records a successful handle (idempotent insert).
func (m *ConsumerManager) recordProcessed(durable, msgID, subject string) {
	rec := &models.ProcessedEvent{Consumer: durable, MsgID: msgID, Subject: subject}
	if err := m.db.Clauses(clause.OnConflict{DoNothing: true}).Create(rec).Error; err != nil {
		log.Printf("[consumer] %s record-processed failed for %s: %v", durable, msgID, err)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func msgIDOf(msg jetstream.Msg) string {
	if h := msg.Headers(); h != nil {
		return h.Get(msgIDHeader)
	}
	return ""
}

func ack(msg jetstream.Msg) {
	if err := msg.Ack(); err != nil {
		log.Printf("[consumer] ack failed for %s: %v", msg.Subject(), err)
	}
}

func deliveries(md *jetstream.MsgMetadata) uint64 {
	if md == nil {
		return 0
	}
	return md.NumDelivered
}

// consumerBackoff grows with delivery count, capped at 30s.
func consumerBackoff(md *jetstream.MsgMetadata) time.Duration {
	n := deliveries(md)
	if n < 1 {
		n = 1
	}
	d := time.Second * time.Duration(int64(1)<<min(int(n), 5)) // 2,4,…32 → capped
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

// decodeEvent unmarshals msg data into T. A decode error is a poison message
// (bad JSON never becomes good) — it retries then dead-letters, by design.
func decodeEvent[T any](data []byte) (T, error) {
	var v T
	err := json.Unmarshal(data, &v)
	return v, err
}
