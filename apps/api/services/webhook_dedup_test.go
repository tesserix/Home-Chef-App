package services

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// setupDedupDB spins an in-memory sqlite with just the processed_events ledger.
func setupDedupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.ProcessedEvent{}))
	return db
}

func TestClaimWebhookEvent_FirstThenReplay(t *testing.T) {
	db := setupDedupDB(t)

	first, err := ClaimWebhookEvent(db, "webhook:razorpay", "evt_1", "payment.captured")
	require.NoError(t, err)
	assert.True(t, first, "first claim of a fresh event must be firstTime=true")

	replay, err := ClaimWebhookEvent(db, "webhook:razorpay", "evt_1", "payment.captured")
	require.NoError(t, err)
	assert.False(t, replay, "identical (consumer,id) replay must be firstTime=false")

	// A different id under the same consumer is independent.
	other, err := ClaimWebhookEvent(db, "webhook:razorpay", "evt_2", "payment.captured")
	require.NoError(t, err)
	assert.True(t, other, "distinct event id must claim fresh")

	// Same id under a different consumer is independent.
	otherConsumer, err := ClaimWebhookEvent(db, "webhook:delivery:borzo", "evt_1", "delivery:borzo")
	require.NoError(t, err)
	assert.True(t, otherConsumer, "same id under a different consumer must claim fresh")
}

func TestClaimWebhookEvent_Concurrent(t *testing.T) {
	db := setupDedupDB(t)
	// :memory: opens a fresh empty DB per pooled connection; pin to 1 so all
	// goroutines share the one DB that has the table. Serializes the claims —
	// this validates the idiom returns a single winner, not raw DB arbitration
	// (that atomicity guarantee lives in Postgres ON CONFLICT DO NOTHING).
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	const n = 40
	var winners int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			first, err := ClaimWebhookEvent(db, "webhook:razorpay", "evt_race", "payment.captured")
			if err == nil && first {
				atomic.AddInt32(&winners, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&winners), "exactly one concurrent claim may win")

	var rows int64
	require.NoError(t, db.Model(&models.ProcessedEvent{}).
		Where("consumer = ? AND msg_id = ?", "webhook:razorpay", "evt_race").Count(&rows).Error)
	assert.Equal(t, int64(1), rows, "exactly one ledger row for the raced event")
}

func TestReleaseWebhookEvent_AllowsReprocess(t *testing.T) {
	db := setupDedupDB(t)

	first, err := ClaimWebhookEvent(db, "webhook:delivery:borzo", "evt_x", "delivery:borzo")
	require.NoError(t, err)
	require.True(t, first)

	// A transient dispatch failure releases the claim so the provider retry
	// re-processes rather than being silently deduped.
	ReleaseWebhookEvent(db, "webhook:delivery:borzo", "evt_x")

	again, err := ClaimWebhookEvent(db, "webhook:delivery:borzo", "evt_x", "delivery:borzo")
	require.NoError(t, err)
	assert.True(t, again, "after release the same event must claim fresh again")
}

func TestWebhookEventID(t *testing.T) {
	body := []byte(`{"event":"payment.captured","id":"pay_123"}`)

	// A sane header id wins verbatim.
	assert.Equal(t, "evt_abc123", WebhookEventID("evt_abc123", body))

	// Empty header → deterministic body hash, stable across calls.
	h1 := WebhookEventID("", body)
	h2 := WebhookEventID("", body)
	assert.Equal(t, h1, h2, "body hash must be deterministic")
	assert.NotEmpty(t, h1)
	assert.LessOrEqual(t, len(h1), 64, "event id must fit processed_events.msg_id varchar(64)")

	// Different bodies → different ids.
	assert.NotEqual(t, h1, WebhookEventID("", []byte(`{"event":"payment.failed"}`)))

	// An over-long header id must NOT be used verbatim (would overflow varchar(64)
	// on Postgres → 500 poison-loop); it falls back to the capped body hash.
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'a'
	}
	got := WebhookEventID(string(long), body)
	assert.LessOrEqual(t, len(got), 64, "over-long header id must be capped to ≤ 64")
}
