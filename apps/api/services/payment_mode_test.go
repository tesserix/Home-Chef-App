package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupPaymentModeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, user_id TEXT,
		mode TEXT DEFAULT 'live', first_live_at DATETIME, active_test_session_id TEXT,
		is_verified BOOLEAN DEFAULT 0, is_active BOOLEAN DEFAULT 1,
		created_at DATETIME, updated_at DATETIME)`).Error)
	old := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = old })
	return db
}

func TestPaymentModeForChef(t *testing.T) {
	db := setupPaymentModeDB(t)
	live, test := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?,?),(?,?)`,
		live.String(), "live", test.String(), "test").Error)

	if got := PaymentModeForChef(live); got != models.ChefModeLive {
		t.Fatalf("live chef = %q, want live", got)
	}
	if got := PaymentModeForChef(test); got != models.ChefModeTest {
		t.Fatalf("test chef = %q, want test", got)
	}
}

// A missing chef, a DB error, or a garbage stored value must ALL resolve to
// live. Resolving to test would route a real payment through sandbox
// credentials, which appears to succeed while capturing no money — a silent
// revenue loss. Resolving wrongly to live merely produces a loud gateway error.
func TestPaymentModeFailsSafeToLive(t *testing.T) {
	db := setupPaymentModeDB(t)
	garbage := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?,?)`,
		garbage.String(), "sandbox").Error)

	if got := PaymentModeForChef(uuid.New()); got != models.ChefModeLive {
		t.Fatalf("missing chef = %q, want live", got)
	}
	if got := PaymentModeForChef(garbage); got != models.ChefModeLive {
		t.Fatalf("garbage mode = %q, want live", got)
	}

	// An empty stored value (a row written before the column existed) is live.
	blank := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?,?)`,
		blank.String(), "").Error)
	if got := PaymentModeForChef(blank); got != models.ChefModeLive {
		t.Fatalf("blank mode = %q, want live", got)
	}
}

func TestIsTestOrder(t *testing.T) {
	o := &models.Order{}
	o.Mode = models.ChefModeTest
	if !IsTestOrder(o) {
		t.Fatal("a test-mode order is a test order")
	}
	for _, m := range []string{"", "live", "garbage"} {
		x := &models.Order{}
		x.Mode = m
		if IsTestOrder(x) {
			t.Fatalf("mode %q must not read as a test order", m)
		}
	}
	if IsTestOrder(nil) {
		t.Fatal("a nil order must not read as a test order")
	}
}
