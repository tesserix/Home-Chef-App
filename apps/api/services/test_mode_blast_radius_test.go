package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func testOrder(mode string) *models.Order {
	o := &models.Order{}
	o.Mode = mode
	return o
}

// A real courier must never be dispatched for a sandbox order — a rider would
// be sent to a real address to collect food nobody is cooking.
func TestTestOrderNeverDispatchesToProvider(t *testing.T) {
	if ShouldDispatchToProvider(testOrder(models.ChefModeTest)) {
		t.Fatal("a test order must never reach a 3PL provider")
	}
	if !ShouldDispatchToProvider(testOrder(models.ChefModeLive)) {
		t.Fatal("a live order must still dispatch normally")
	}
}

// Sandbox money must not become spendable balance, and sandbox points must not
// be redeemable against a real kitchen.
func TestTestOrderIsExcludedFromWalletLoyaltyAndPayouts(t *testing.T) {
	test, live := testOrder(models.ChefModeTest), testOrder(models.ChefModeLive)

	if WalletAllowedForOrder(test) {
		t.Fatal("a test order must not use wallet as payment source or refund destination")
	}
	if LoyaltyAllowedForOrder(test) {
		t.Fatal("a test order must neither earn nor redeem loyalty points")
	}
	if PayoutAllowedForOrder(test) {
		t.Fatal("a test order must never enter the real payout engine")
	}

	for name, ok := range map[string]bool{
		"wallet":  WalletAllowedForOrder(live),
		"loyalty": LoyaltyAllowedForOrder(live),
		"payout":  PayoutAllowedForOrder(live),
	} {
		if !ok {
			t.Fatalf("a live order must be unaffected for %s", name)
		}
	}
}

func setupBlastRadiusDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	// deleted_at is required: Order is soft-deletable, so GORM appends
	// "deleted_at IS NULL" to every query and the fixture must carry the column
	// or the read errors out (and OrderIsTestMode then fails safe to live).
	require.NoError(t, db.Exec(`CREATE TABLE orders (id TEXT PRIMARY KEY, mode TEXT DEFAULT 'live',
		test_session_id TEXT, cloned_from_id TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error)
	return db
}

func TestOrderIsTestModeReadsTheRow(t *testing.T) {
	db := setupBlastRadiusDB(t)
	live, test := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, mode) VALUES (?,?),(?,?)`,
		live.String(), "live", test.String(), "test").Error)

	if !OrderIsTestMode(db, test) {
		t.Fatal("a test-partition order must read as test")
	}
	if OrderIsTestMode(db, live) {
		t.Fatal("a live order must not read as test")
	}
	// Fails safe to live: a lookup blip must not start rejecting real wallet or
	// loyalty movement on live orders.
	if OrderIsTestMode(db, uuid.New()) {
		t.Fatal("a missing order must fail safe to live, not test")
	}
	if OrderIsTestMode(nil, live) {
		t.Fatal("a nil db must fail safe to live")
	}
}

// The scopes are what keep sandbox rows out of reporting and out of real
// customers' order history.
func TestModeScopesFilterCorrectly(t *testing.T) {
	db := setupBlastRadiusDB(t)
	live, test, clone := uuid.New(), uuid.New(), uuid.New()
	src := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, mode) VALUES (?,?),(?,?)`,
		live.String(), "live", test.String(), "test").Error)
	require.NoError(t, db.Exec(`INSERT INTO orders (id, mode, cloned_from_id) VALUES (?,?,?)`,
		clone.String(), "test", src.String()).Error)

	var n int64
	require.NoError(t, db.Table("orders").Scopes(ExcludeTestOrders).Count(&n).Error)
	require.EqualValues(t, 1, n, "reporting must see only the live order")

	require.NoError(t, db.Table("orders").Scopes(ExcludeClonedRows).Count(&n).Error)
	require.EqualValues(t, 2, n, "clones must be excluded, the two real rows kept")

	require.NoError(t, db.Table("orders").Scopes(ModeScope(models.ChefModeTest)).Count(&n).Error)
	require.EqualValues(t, 2, n, "the test partition holds the sandbox order and the clone")

	// A stranger sees only live rows, and never a clone.
	require.NoError(t, db.Table("orders").
		Scopes(CustomerVisibleModes("stranger@example.com")).Count(&n).Error)
	require.EqualValues(t, 1, n, "a non-allowlisted customer sees only their live orders")

	// A tester sees their own sandbox order too — but still never a clone,
	// because a clone belongs to a real customer who never placed it.
	require.NoError(t, db.Table("orders").
		Scopes(CustomerVisibleModes("samyak.rout@gmail.com")).Count(&n).Error)
	require.EqualValues(t, 2, n, "a tester sees live + test, never clones")
}
