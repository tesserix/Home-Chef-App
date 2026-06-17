package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
)

// TestGenerateEventIDUnique guards the #138 fix: the old time-based generator
// emitted repeated characters under concurrency. IDs must be unique UUIDs.
func TestGenerateEventIDUnique(t *testing.T) {
	const n = 5000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := generateEventID()
		if _, err := uuid.Parse(id); err != nil {
			t.Fatalf("event id %q is not a valid uuid: %v", id, err)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate event id generated: %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestOutboxBackoffGrowsAndCaps(t *testing.T) {
	prev := time.Duration(0)
	for attempt := 1; attempt <= 6; attempt++ {
		d := outboxBackoff(attempt)
		if d <= 0 || d > time.Minute {
			t.Fatalf("attempt %d: backoff %s out of bounds (0, 60s]", attempt, d)
		}
		if d < prev {
			t.Fatalf("attempt %d: backoff %s should be non-decreasing (prev %s)", attempt, d, prev)
		}
		prev = d
	}
	if got := outboxBackoff(100); got != time.Minute {
		t.Fatalf("large attempt should cap at 60s, got %s", got)
	}
}

func TestConsumerBackoffCaps(t *testing.T) {
	for _, n := range []uint64{0, 1, 2, 5, 50} {
		d := consumerBackoff(&jetstream.MsgMetadata{NumDelivered: n})
		if d <= 0 || d > 30*time.Second {
			t.Fatalf("NumDelivered=%d: backoff %s out of bounds (0, 30s]", n, d)
		}
	}
	if got := consumerBackoff(nil); got <= 0 {
		t.Fatalf("nil metadata should still yield a positive backoff, got %s", got)
	}
}

func TestNotificationSubject(t *testing.T) {
	cases := map[string]string{
		"email": SubjectNotificationEmail,
		"sms":   SubjectNotificationSMS,
		"push":  SubjectNotificationPush,
		"":      SubjectNotificationPush, // default
		"weird": SubjectNotificationPush, // default
	}
	for channel, want := range cases {
		if got := notificationSubject(channel); got != want {
			t.Errorf("notificationSubject(%q) = %q, want %q", channel, got, want)
		}
	}
}

func TestDecodeEvent(t *testing.T) {
	in := OrderEvent{OrderNumber: "HC-1", Status: "created"}
	raw, _ := json.Marshal(in)

	got, err := decodeEvent[OrderEvent](raw)
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if got.OrderNumber != "HC-1" || got.Status != "created" {
		t.Fatalf("decodeEvent round-trip mismatch: %+v", got)
	}

	if _, err := decodeEvent[OrderEvent]([]byte("not json")); err == nil {
		t.Fatal("expected decode error on malformed json (poison message)")
	}
}
