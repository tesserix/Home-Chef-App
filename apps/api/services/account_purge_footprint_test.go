package services

// account_purge_footprint_test.go — pins the fix for an erasure gap found by
// purging a real production account: customerCascade.Purge deleted only
// addresses, leaving rows in ten tables — including customer_profiles, which
// holds date_of_birth and food_allergies — still keyed to the "erased" user,
// and leaving the street address readable on every retained order row.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedPersonalFootprint creates the personal tables the production gap was
// found in, with one row each for the user.
func seedPersonalFootprint(t *testing.T, db *gorm.DB, userID uuid.UUID) {
	t.Helper()
	// IF NOT EXISTS: setupAccountDB already creates some of these (the deletion
	// blockers read wallets), and this seed must compose with it either way.
	for _, ddl := range []string{
		`CREATE TABLE IF NOT EXISTS customer_profiles (id TEXT PRIMARY KEY, user_id TEXT,
			date_of_birth DATETIME, food_allergies TEXT, dietary_preferences TEXT)`,
		`CREATE TABLE IF NOT EXISTS notifications (id TEXT PRIMARY KEY, user_id TEXT, title TEXT)`,
		`CREATE TABLE IF NOT EXISTS wallets (id TEXT PRIMARY KEY, user_id TEXT, balance REAL DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS wallet_txns (id TEXT PRIMARY KEY, user_id TEXT, amount REAL DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS loyalty_accounts (id TEXT PRIMARY KEY, user_id TEXT, points INTEGER DEFAULT 0)`,
	} {
		require.NoError(t, db.Exec(ddl).Error)
	}
	uid := userID.String()
	require.NoError(t, db.Exec(`INSERT INTO customer_profiles (id, user_id, food_allergies) VALUES (?,?,?)`,
		uuid.NewString(), uid, "peanuts").Error)
	require.NoError(t, db.Exec(`INSERT INTO notifications (id, user_id, title) VALUES (?,?,?)`,
		uuid.NewString(), uid, "Your order is on the way").Error)
	require.NoError(t, db.Exec(`INSERT INTO wallets (id, user_id, balance) VALUES (?,?,0)`,
		uuid.NewString(), uid).Error)
	require.NoError(t, db.Exec(`INSERT INTO wallet_txns (id, user_id, amount) VALUES (?,?,10)`,
		uuid.NewString(), uid).Error)
	require.NoError(t, db.Exec(`INSERT INTO loyalty_accounts (id, user_id, points) VALUES (?,?,5)`,
		uuid.NewString(), uid).Error)
}

func TestPurge_ErasesPersonalFootprint(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleCustomer)
	seedPersonalFootprint(t, db, user.ID)

	_, err := RequestDeletion(db, &user, "")
	require.NoError(t, err)
	require.NoError(t, PurgeUser(db, user.ID, models.RoleCustomer))

	for _, table := range []string{
		"customer_profiles", "notifications", "wallets", "wallet_txns", "loyalty_accounts",
	} {
		var n int64
		require.NoError(t, db.Raw(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`,
			user.ID.String()).Scan(&n).Error)
		require.Zerof(t, n, "%s must hold no rows for a purged user", table)
	}
}

// TestPurge_ScrubsOrderAddressButKeepsTheRecord: the order row survives (it is
// the chef's financial record too) but the parts that identify the customer —
// street address, coordinates, free-text instructions — do not.
func TestPurge_ScrubsOrderAddressButKeepsTheRecord(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleCustomer)

	// Replace the fixture's minimal orders table with one that carries the PII
	// columns the scrub targets — the whole point of this test.
	require.NoError(t, db.Exec(`DROP TABLE IF EXISTS orders`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE orders (id TEXT PRIMARY KEY, customer_id TEXT,
		chef_id TEXT, status TEXT DEFAULT 'delivered', mode TEXT DEFAULT 'live',
		deleted_at DATETIME,
		total REAL DEFAULT 0, delivery_address_line1 TEXT, delivery_address_line2 TEXT,
		delivery_address_postal_code TEXT, delivery_latitude REAL DEFAULT 0,
		delivery_longitude REAL DEFAULT 0, delivery_instructions TEXT,
		special_instructions TEXT)`).Error)
	orderID := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, customer_id, total, delivery_address_line1, delivery_address_postal_code,
		 delivery_latitude, delivery_longitude, delivery_instructions)
		VALUES (?,?,154.19,'12 Harbour Lane','751024',20.35,85.82,'gate code 4411')`,
		orderID, user.ID.String()).Error)

	_, err := RequestDeletion(db, &user, "")
	require.NoError(t, err)
	require.NoError(t, PurgeUser(db, user.ID, models.RoleCustomer))

	var row struct {
		Total    float64
		Line1    string
		Postal   string
		Lat, Lng float64
		Instr    string
	}
	require.NoError(t, db.Raw(`SELECT total, delivery_address_line1 AS line1,
		delivery_address_postal_code AS postal, delivery_latitude AS lat,
		delivery_longitude AS lng, delivery_instructions AS instr
		FROM orders WHERE id = ?`, orderID).Scan(&row).Error)

	require.Equal(t, 154.19, row.Total, "the financial record must survive the purge")
	require.Empty(t, row.Line1, "street address must be scrubbed")
	require.Empty(t, row.Postal, "postal code must be scrubbed")
	require.Zero(t, row.Lat, "coordinates must be scrubbed")
	require.Zero(t, row.Lng, "coordinates must be scrubbed")
	require.Empty(t, row.Instr, "free-text instructions must be scrubbed")
}
