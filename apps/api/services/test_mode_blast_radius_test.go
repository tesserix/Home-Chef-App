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

// setupMenuModeDB builds the two tables the menu scopes correlate across.
// chef_profiles carries only what the subquery reads.
func setupMenuModeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, mode TEXT DEFAULT 'live')`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE menu_items (id TEXT PRIMARY KEY, chef_id TEXT,
		name TEXT, mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error)
	return db
}

// menu_items is mode-partitioned and CloneChefIntoSession copies the whole menu
// into a test session, so an unscoped customer read returns the live menu AND
// its sandbox clone — every dish twice. That shipped: customers saw 32 items on
// a kitchen whose chef app correctly showed 16, because the three customer
// paths (GetChefMenu, SearchDishes, CreateOrder) filtered on availability and
// approval but never on mode.
//
// The blast-radius suite covered orders, wallet, loyalty, payouts and dispatch.
// menu_items was not in it, which is exactly why nobody noticed.
func TestMenuItemsAreScopedToTheOwningChefsMode(t *testing.T) {
	db := setupMenuModeDB(t)
	liveChef, testChef := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?,?),(?,?)`,
		liveChef.String(), models.ChefModeLive, testChef.String(), models.ChefModeTest).Error)

	// The live kitchen carries its real dish plus the sandbox clone left behind
	// by a past session — the exact shape that produced the duplicate menu.
	require.NoError(t, db.Exec(`INSERT INTO menu_items (id, chef_id, name, mode) VALUES (?,?,?,?),(?,?,?,?)`,
		uuid.New().String(), liveChef.String(), "Butter Chicken", models.ChefModeLive,
		uuid.New().String(), liveChef.String(), "Butter Chicken", models.ChefModeTest).Error)
	// The sandboxed kitchen's menu lives entirely in the test partition.
	require.NoError(t, db.Exec(`INSERT INTO menu_items (id, chef_id, name, mode) VALUES (?,?,?,?)`,
		uuid.New().String(), testChef.String(), "Paneer Tikka", models.ChefModeTest).Error)

	var n int64

	// Fixed-chef scope: GetChefMenu and CreateOrder.
	require.NoError(t, db.Table("menu_items").
		Where("chef_id = ?", liveChef.String()).
		Scopes(ChefOwnModeScope(liveChef)).Count(&n).Error)
	require.EqualValues(t, 1, n, "a live kitchen must serve its live dish only, not the sandbox clone")

	// A sandboxed kitchen shows its sandbox menu — the clone IS the menu there,
	// so scoping to live would leave a tester browsing an empty kitchen.
	require.NoError(t, db.Table("menu_items").
		Where("chef_id = ?", testChef.String()).
		Scopes(ChefOwnModeScope(testChef)).Count(&n).Error)
	require.EqualValues(t, 1, n, "a sandboxed kitchen must still show its own test-partition menu")

	// Correlated scope: SearchDishes spans kitchens, so each row is judged
	// against its own chef. One live dish + one sandbox dish on a sandbox chef.
	require.NoError(t, db.Table("menu_items").
		Scopes(MatchOwningChefModeScope("menu_items")).Count(&n).Error)
	require.EqualValues(t, 2, n, "each dish is judged against the mode of the kitchen that owns it")

	// The regression itself, stated plainly.
	require.NoError(t, db.Table("menu_items").Where("chef_id = ?", liveChef.String()).Count(&n).Error)
	require.EqualValues(t, 2, n, "unscoped, the duplicate is still there — the scope is what removes it")
}
