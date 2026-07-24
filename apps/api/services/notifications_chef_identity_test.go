package services

// notifications_chef_identity_test.go — OrderEvent.ChefID is a chef_profiles.id,
// NOT a users.id. Order.CustomerID *is* a users.id (models.Customer aliases
// models.User), and that asymmetry is the trap: passing ChefID straight into a
// user-keyed notification compiles fine (both are uuid.UUID) and fails only at
// runtime, as `push: user <id> not found: record not found`. Because
// notifications are best-effort the failure never reaches the caller — it
// retries 5x and dead-letters, so chefs simply never learn an order arrived.
//
// These tests pin the mapping: given an event carrying a PROFILE id, every
// chef-facing notification must be written against the chef's USER id.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// setupChefNotifDB extends setupNotifDB with the chef_profiles table the
// profile→user mapping needs.
func setupChefNotifDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupNotifDB(t)
	require.NoError(t, db.Exec(
		`CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', id text PRIMARY KEY, user_id text)`,
	).Error)
	return db
}

// seedChef inserts a chef profile and returns (profileID, userID) — deliberately
// different values, which is the whole point.
func seedChef(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	profileID, userID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id) VALUES (?, ?)`, profileID.String(), userID.String(),
	).Error)
	return profileID, userID
}

func TestChefUserID_MapsProfileIDToUserID(t *testing.T) {
	db := setupChefNotifDB(t)
	profileID, userID := seedChef(t, db)

	got, err := chefUserID(profileID)

	require.NoError(t, err)
	require.Equal(t, userID, got)
	require.NotEqual(t, profileID, got, "profile id and user id must not be conflated")
}

func TestChefUserID_UnknownProfile_Errors(t *testing.T) {
	setupChefNotifDB(t)

	_, err := chefUserID(uuid.New())

	require.Error(t, err, "an unresolvable chef must surface, not silently pass a bad id downstream")
}

func TestChefUserID_NilProfile_Errors(t *testing.T) {
	setupChefNotifDB(t)

	_, err := chefUserID(uuid.Nil)

	require.Error(t, err)
}

// handleOrderCreated is now a deliberate no-op: orders.created fires at order
// CREATION (before payment), and used to duplicate the chef's one true
// new-order alert — chef.new_order → handleChefNewOrder ("New Order!" + the
// actionable Accept/Reject push), which fires on payment success for regular
// orders, or alongside generated/group/subscription orders. Dedup means the
// chef must see NO notification from order_created, for either a resolvable or
// an unresolvable chef profile.
func TestHandleOrderCreated_NoOp_ChefNotNotified(t *testing.T) {
	db := setupChefNotifDB(t)
	profileID, _ := seedChef(t, db)

	s := GetNotificationService()
	require.NoError(t, s.handleOrderCreated(OrderEvent{
		OrderID:    uuid.New(),
		CustomerID: uuid.New(),
		ChefID:     profileID, // as published by handlers: order.ChefID
		Total:      80,
	}))

	var count int64
	require.NoError(t, db.Model(&models.Notification{}).Where("type = ?", "order_created").Count(&count).Error)
	require.Zero(t, count, "order_created must no longer write a chef notification — chef.new_order owns that alert now")
}

// Even an unresolvable chef profile must not error now that the handler never
// touches the chef at all.
func TestHandleOrderCreated_NoOp_UnresolvableChefDoesNotError(t *testing.T) {
	setupChefNotifDB(t)

	s := GetNotificationService()
	err := s.handleOrderCreated(OrderEvent{
		OrderID:    uuid.New(),
		CustomerID: uuid.New(),
		ChefID:     uuid.New(), // no such profile
		Total:      80,
	})

	require.NoError(t, err)
}

// Cancellation notifies BOTH parties: the customer by user id (already correct)
// and the chef via the profile→user mapping.
func TestHandleOrderCancelled_NotifiesCustomerAndChefUser(t *testing.T) {
	db := setupChefNotifDB(t)
	profileID, chefUser := seedChef(t, db)
	customerID := uuid.New()

	s := GetNotificationService()
	require.NoError(t, s.handleOrderCancelled(OrderEvent{
		OrderID:    uuid.New(),
		CustomerID: customerID,
		ChefID:     profileID,
	}))

	var rows []models.Notification
	require.NoError(t, db.Where("type = ?", "order_cancelled").Find(&rows).Error)
	require.Len(t, rows, 2, "both customer and chef are told")

	got := map[uuid.UUID]bool{}
	for _, r := range rows {
		got[r.UserID] = true
	}
	require.True(t, got[customerID], "customer notified by user id")
	require.True(t, got[chefUser], "chef notified by USER id, not profile id")
	require.False(t, got[profileID], "profile id must never be written as a notification user id")
}

func TestHandleChefNewOrder_NotifiesChefUserNotProfile(t *testing.T) {
	db := setupChefNotifDB(t)
	profileID, chefUser := seedChef(t, db)

	s := GetNotificationService()
	require.NoError(t, s.handleChefNewOrder(OrderEvent{
		OrderID: uuid.New(),
		ChefID:  profileID,
		Total:   80,
	}))

	var n models.Notification
	require.NoError(t, db.Where("type = ?", "new_order").First(&n).Error)
	require.Equal(t, chefUser, n.UserID)
	require.NotEqual(t, profileID, n.UserID)
}
