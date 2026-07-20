package services

// notifications_void_test.go — #694. The consumer end of the auto-void.
//
// The sweeps STAGE orders.voided and orders.accept_reminder; these prove the
// notification service actually consumes them: decodes the exact payload the
// producer emits, routes it to the right handler, and writes the customer's
// apology / the chef's nudge. A staged event nobody consumes is a silent gap, so
// this is the "integrated and working" half.

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func setupVoidNotifDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupNotifDB(t)
	// chefUserID (the accept-reminder path) maps a profile id to a user id.
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', id text PRIMARY KEY, user_id text)`).Error)
	// handleOrderVoided may fall back to the order for the customer id.
	require.NoError(t, db.Exec(`CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id text PRIMARY KEY, customer_id text, deleted_at datetime)`).Error)
	return db
}

func notifRows(t *testing.T, db *gorm.DB, userID uuid.UUID) []models.Notification {
	t.Helper()
	var rows []models.Notification
	require.NoError(t, db.Where("user_id = ?", userID).Find(&rows).Error)
	return rows
}

// The customer gets an apology + refund confirmation, decoded from the SAME
// payload refundUnacceptedOrders emits.
func TestHandleOrderVoided_ApologisesToTheCustomer(t *testing.T) {
	db := setupVoidNotifDB(t)
	customerID := uuid.New()

	// Byte-identical to the sweep's EnqueueEvent map, routed through the real
	// subject dispatch — so a field rename on either side breaks this test.
	payload, _ := json.Marshal(map[string]any{
		"order_id":        uuid.New().String(),
		"order_number":    "ORD-VOID",
		"chef_id":         uuid.New().String(),
		"reason":          "the kitchen closed before this order was accepted",
		"refunded":        true,
		"initiated_by":    "system",
		"slot":            "dinner",
		"refunded_amount": 250.0,
		"customer_id":     customerID.String(),
	})

	s := GetNotificationService()
	require.NoError(t, s.handleBySubject(nil, SubjectOrderVoided, payload))

	rows := notifRows(t, db, customerID)
	require.Len(t, rows, 1, "the customer must be told — a silent refund reads as theft")
	require.Equal(t, "order_voided", rows[0].Type)
	require.Contains(t, rows[0].Message, "dinner", "the apology names the meal")
	require.Contains(t, rows[0].Message, "250", "and confirms the amount refunded")
}

// No customer_id in the payload → resolve it from the order rather than skip the
// apology.
func TestHandleOrderVoided_FallsBackToTheOrderForCustomer(t *testing.T) {
	db := setupVoidNotifDB(t)
	customerID, orderID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id) VALUES (?,?)`,
		orderID.String(), customerID.String()).Error)

	payload, _ := json.Marshal(map[string]any{
		"order_id": orderID.String(), "slot": "lunch", "refunded_amount": 120.0,
	})
	s := GetNotificationService()
	require.NoError(t, s.handleBySubject(nil, SubjectOrderVoided, payload))

	require.Len(t, notifRows(t, db, customerID), 1, "a missing customer_id must not lose the apology")
}

// The chef gets the nudge, keyed to their USER id (the payload carries a PROFILE
// id — the #134 trap).
func TestHandleOrderAcceptReminder_NudgesTheChefByUserID(t *testing.T) {
	db := setupVoidNotifDB(t)
	profileID, userID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id) VALUES (?,?)`,
		profileID.String(), userID.String()).Error)

	payload, _ := json.Marshal(map[string]any{
		"order_id":     uuid.New().String(),
		"order_number": "ORD-NUDGE",
		"chef_id":      profileID.String(),
		"slot":         "lunch",
		"minutes_left": 25,
		"final_call":   true,
	})
	s := GetNotificationService()
	require.NoError(t, s.handleBySubject(nil, SubjectOrderAcceptReminder, payload))

	require.Empty(t, notifRows(t, db, profileID), "must not key the nudge on the profile id")
	rows := notifRows(t, db, userID)
	require.Len(t, rows, 1, "the chef learns their order is about to be auto-cancelled")
	require.Equal(t, "accept_reminder", rows[0].Type)
	require.Contains(t, rows[0].Title, "Last chance", "final_call sharpens the tone")
}
