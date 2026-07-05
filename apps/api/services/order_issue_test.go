package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// order_issue_test.go — #37. The refund math (per-line + capped sum), the
// auto/assisted decision, and the wallet-refund grant. Money correctness is the
// point: a refund must credit exactly once, resolve the issue exactly once, and
// never exceed the order's remaining refundable amount.

func TestLineRefundAmount(t *testing.T) {
	// Line is 200 of a 1000 subtotal with 100 tax → 200 + 100*(200/1000) = 220.
	assert.InDelta(t, 220.0, LineRefundAmount(200, 1000, 100), 0.001)
	// Zero order subtotal → just the line subtotal (no tax share).
	assert.Equal(t, 50.0, LineRefundAmount(50, 0, 100))
}

func TestComputeIssueRefund(t *testing.T) {
	t.Run("sums affected lines with tax share", func(t *testing.T) {
		// Two lines (200 + 300) of a 1000 subtotal, 100 tax, total 1100, none refunded.
		// 220 + 330 = 550.
		got := ComputeIssueRefund(1000, 100, 1100, 0, []float64{200, 300})
		assert.InDelta(t, 550.0, got, 0.01)
	})
	t.Run("caps at remaining refundable", func(t *testing.T) {
		// Remaining = total 1100 - already 1000 = 100, even though lines sum higher.
		got := ComputeIssueRefund(1000, 100, 1100, 1000, []float64{200, 300})
		assert.InDelta(t, 100.0, got, 0.01)
	})
	t.Run("no affected items → zero (assisted)", func(t *testing.T) {
		assert.Equal(t, 0.0, ComputeIssueRefund(1000, 100, 1100, 0, nil))
	})
}

func TestShouldAutoRefund(t *testing.T) {
	cfg := IssueConfig{Enabled: true, AutoApproveCap: 300}
	assert.True(t, ShouldAutoRefund(cfg, 250))
	assert.True(t, ShouldAutoRefund(cfg, 300)) // boundary inclusive
	assert.False(t, ShouldAutoRefund(cfg, 301))
	assert.False(t, ShouldAutoRefund(cfg, 0))
	assert.False(t, ShouldAutoRefund(IssueConfig{Enabled: false, AutoApproveCap: 300}, 100))
}

func setupIssueDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, chef_id TEXT, customer_id TEXT,
			reason TEXT, description TEXT, photo_urls TEXT, affected_item_ids TEXT,
			requested_amount REAL DEFAULT 0, refund_amount REAL DEFAULT 0, status TEXT DEFAULT 'pending',
			resolved_by TEXT, resolved_at DATETIME, refund_txn_id TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE orders (id TEXT PRIMARY KEY, total REAL, refund_amount REAL DEFAULT 0, refund_reason TEXT,
			refund_initiated_by TEXT, refunded_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE order_items (id TEXT PRIMARY KEY, order_id TEXT, is_cancelled BOOLEAN DEFAULT 0, refund_amount REAL DEFAULT 0)`,
		`CREATE TABLE wallets (id TEXT PRIMARY KEY, user_id TEXT UNIQUE, balance REAL DEFAULT 0, currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE wallet_txns (id TEXT PRIMARY KEY, wallet_id TEXT, user_id TEXT, type TEXT, source TEXT,
			amount REAL, balance_after REAL, currency TEXT, order_id TEXT, reason TEXT, created_by TEXT,
			idempotency_key TEXT UNIQUE, created_at DATETIME)`,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT UNIQUE, value TEXT, type TEXT, updated_by TEXT, updated_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func seedIssue(t *testing.T, db *gorm.DB, customer uuid.UUID, orderID uuid.UUID) *models.OrderIssue {
	t.Helper()
	iss := &models.OrderIssue{
		OrderID: orderID, ChefID: uuid.New(), CustomerID: customer,
		Reason: models.IssueMissingItem, Status: models.IssuePending,
	}
	require.NoError(t, db.Create(iss).Error)
	return iss
}

func balanceOfUser(t *testing.T, db *gorm.DB, userID uuid.UUID) float64 {
	t.Helper()
	var bal float64
	db.Raw(`SELECT COALESCE(balance,0) FROM wallets WHERE user_id = ?`, userID.String()).Scan(&bal)
	return bal
}

func TestRefundIssueToWallet(t *testing.T) {
	t.Run("credits wallet, resolves issue, bumps order refund", func(t *testing.T) {
		db := setupIssueDB(t)
		customer, orderID := uuid.New(), uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 500, 0)`, orderID.String()).Error)
		iss := seedIssue(t, db, customer, orderID)

		require.NoError(t, RefundIssueToWallet(db, iss, 120, "system", nil))

		assert.Equal(t, 120.0, balanceOfUser(t, db, customer))
		assert.Equal(t, models.IssueAutoRefunded, iss.Status)

		var status string
		var orderRefund float64
		db.Raw(`SELECT status FROM order_issues WHERE id = ?`, iss.ID.String()).Scan(&status)
		db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&orderRefund)
		assert.Equal(t, "auto_refunded", status)
		assert.Equal(t, 120.0, orderRefund)
	})

	t.Run("idempotent — a duplicate call never double-refunds", func(t *testing.T) {
		db := setupIssueDB(t)
		customer, orderID := uuid.New(), uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 500, 0)`, orderID.String()).Error)
		iss := seedIssue(t, db, customer, orderID)

		require.NoError(t, RefundIssueToWallet(db, iss, 120, "system", nil))
		require.NoError(t, RefundIssueToWallet(db, iss, 120, "system", nil)) // retry

		assert.Equal(t, 120.0, balanceOfUser(t, db, customer), "wallet credited once")
		var orderRefund float64
		db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&orderRefund)
		assert.Equal(t, 120.0, orderRefund, "order refund incremented once")
	})

	// #560/#527: after a per-line cancel (Total reduced AND RefundAmount bumped by the
	// refunded line), the naive Total − RefundAmount caps to 0 and strands a legitimate
	// issue refund on the remaining LIVE item. RemainingRefundable adds the per-line
	// refund back, so the live-item value stays refundable.
	t.Run("per-line cancel doesn't strand a legit issue refund", func(t *testing.T) {
		db := setupIssueDB(t)
		customer, orderID := uuid.New(), uuid.New()
		// 2×₹110 order, per-line cancel of item 1 → Total 110, RefundAmount 110, cancelled item refund 110.
		require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 110, 110)`, orderID.String()).Error)
		require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
			uuid.NewString(), orderID.String(), true, 110).Error)
		iss := seedIssue(t, db, customer, orderID)

		// The remaining live item (₹110) is still owed — must NOT be capped to 0.
		require.NoError(t, RefundIssueToWallet(db, iss, 110, "system", nil))
		assert.Equal(t, 110.0, balanceOfUser(t, db, customer), "the live-item value is refundable, not stranded")
	})

	t.Run("admin resolve marks resolved (not auto)", func(t *testing.T) {
		db := setupIssueDB(t)
		customer, orderID := uuid.New(), uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 500, 0)`, orderID.String()).Error)
		iss := seedIssue(t, db, customer, orderID)
		admin := uuid.New()

		require.NoError(t, RefundIssueToWallet(db, iss, 90, "admin", &admin))
		assert.Equal(t, models.IssueResolved, iss.Status)
	})

	t.Run("two issues on one order never over-refund past the total", func(t *testing.T) {
		db := setupIssueDB(t)
		customer, orderID := uuid.New(), uuid.New()
		// Order total 300. Two separate reports each ask for 200 (e.g. a double
		// submit racing on a stale RefundAmount of 0). The second must be capped.
		require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 300, 0)`, orderID.String()).Error)
		iss1 := seedIssue(t, db, customer, orderID)
		iss2 := seedIssue(t, db, customer, orderID)

		require.NoError(t, RefundIssueToWallet(db, iss1, 200, "system", nil))
		require.NoError(t, RefundIssueToWallet(db, iss2, 200, "system", nil))

		// First refunds 200; second is capped at the remaining 100.
		assert.Equal(t, 200.0, iss1.RefundAmount)
		assert.Equal(t, 100.0, iss2.RefundAmount)
		assert.Equal(t, 300.0, balanceOfUser(t, db, customer), "wallet credited at most the order total")
		var orderRefund float64
		db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&orderRefund)
		assert.Equal(t, 300.0, orderRefund, "order refund never exceeds the total")
	})

	t.Run("fully-refunded order → ErrNothingToRefund, no money moves", func(t *testing.T) {
		db := setupIssueDB(t)
		customer, orderID := uuid.New(), uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 300, 300)`, orderID.String()).Error)
		iss := seedIssue(t, db, customer, orderID)

		err := RefundIssueToWallet(db, iss, 50, "admin", nil)
		require.ErrorIs(t, err, ErrNothingToRefund)
		assert.Equal(t, 0.0, balanceOfUser(t, db, customer))

		// Issue stays pending (an admin can reject it); nothing was resolved.
		var status string
		db.Raw(`SELECT status FROM order_issues WHERE id = ?`, iss.ID.String()).Scan(&status)
		assert.Equal(t, "pending", status)
	})
}

func TestGetIssueConfig(t *testing.T) {
	db := setupIssueDB(t)
	// Defaults.
	cfg := GetIssueConfig(db)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, 300.0, cfg.AutoApproveCap)
	// Admin override.
	require.NoError(t, db.Exec(`INSERT INTO platform_settings (id, key, value) VALUES (?, 'order_issue.auto_approve_cap', '500')`, uuid.New().String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO platform_settings (id, key, value) VALUES (?, 'order_issue.enabled', 'false')`, uuid.New().String()).Error)
	cfg = GetIssueConfig(db)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, 500.0, cfg.AutoApproveCap)
}
