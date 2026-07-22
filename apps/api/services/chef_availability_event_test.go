package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// Closing a kitchen currently tells nobody. handlers/chef_availability.go flips
// accepting_orders and returns; there is no subject for it in nats.go, and the
// customer app has WebSockets only for order tracking and notifications. With
// useChefs on a 2-minute staleTime, a closed kitchen keeps showing as open —
// long enough for a customer to order into a kitchen that has shut.
//
// The fix is an outbox event so the change is durable and fans out like every
// other state change in the system.

const availOutboxDDL = `CREATE TABLE outbox_events (
	id text PRIMARY KEY, subject text, aggregate_type text, aggregate_id text,
	msg_id text, payload text, status text, attempts integer DEFAULT 0,
	last_error text, next_retry_at datetime, published_at datetime,
	created_at datetime, updated_at datetime
)`

func availDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(availOutboxDDL).Error)
	return db
}

func outboxRows(t *testing.T, db *gorm.DB) []struct {
	Subject     string
	AggregateID string
	Payload     string
} {
	t.Helper()
	var rows []struct {
		Subject     string
		AggregateID string
		Payload     string
	}
	require.NoError(t, db.Raw(`SELECT subject, aggregate_id, payload FROM outbox_events`).Scan(&rows).Error)
	return rows
}

func TestChefAvailabilityChanged_StagesAnOutboxEvent(t *testing.T) {
	db := availDB(t)
	chefID := uuid.New()

	require.NoError(t, EnqueueChefAvailabilityChanged(db, chefID, false))

	rows := outboxRows(t, db)
	require.Len(t, rows, 1, "closing a kitchen must stage exactly one event")
	require.Equal(t, SubjectChefAvailabilityChanged, rows[0].Subject)
	require.Equal(t, chefID.String(), rows[0].AggregateID,
		"aggregate id must be the chef, so consumers can fan out per chef")
}

func TestChefAvailabilityChanged_CarriesTheNewState(t *testing.T) {
	// A consumer must be able to act on the payload alone without re-reading
	// the row, or it races the next change.
	db := availDB(t)
	chefID := uuid.New()

	require.NoError(t, EnqueueChefAvailabilityChanged(db, chefID, false))
	require.Contains(t, outboxRows(t, db)[0].Payload, `"acceptingOrders":false`)

	db2 := availDB(t)
	require.NoError(t, EnqueueChefAvailabilityChanged(db2, chefID, true))
	require.Contains(t, outboxRows(t, db2)[0].Payload, `"acceptingOrders":true`)
}

func TestChefAvailabilityChanged_IdentifiesTheChefInThePayload(t *testing.T) {
	db := availDB(t)
	chefID := uuid.New()

	require.NoError(t, EnqueueChefAvailabilityChanged(db, chefID, true))
	require.Contains(t, outboxRows(t, db)[0].Payload, chefID.String())
}

func TestChefAvailabilityChanged_SubjectIsOnTheChefStream(t *testing.T) {
	// Stream routing is by subject prefix; a subject outside chef.* would be
	// published to a stream that does not exist and silently dropped.
	require.Regexp(t, `^chef\.`, SubjectChefAvailabilityChanged)
}
