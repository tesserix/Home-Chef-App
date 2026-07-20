package orderrefund

// fixture_test.go — test scaffolding for the coordinator (#689).
//
// The gateway is faked through the Gateway interface rather than an httptest
// server. The existing service tests spin up httptest + SetRazorpayClient, which
// is right for THEM (they exercise the real Razorpay client's request shaping).
// The coordinator's job is the saga — ledger, cap, reservation lifecycle — so a
// hand-rolled fake keeps those tests about the saga and lets us assert ordering
// (`before`) that an httptest handler can't express as cleanly.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

type fakeGateway struct {
	calls      int
	lastAmount float64
	lastKey    string
	err        error
	// before runs immediately before the fake "calls" the gateway — lets a test
	// observe committed state at exactly that moment.
	before func()
}

func (g *fakeGateway) RefundPayment(_ context.Context, req GatewayRequest) (string, error) {
	if g.before != nil {
		g.before()
	}
	g.calls++
	g.lastAmount = req.Amount
	g.lastKey = req.IdempotencyKey
	if g.err != nil {
		return "", g.err
	}
	return "rfnd_test", nil
}

type fixture struct {
	t       *testing.T
	db      *gorm.DB
	ctx     context.Context
	coord   *Coordinator
	gateway *fakeGateway
}

type fixtureOpt func(*fixtureCfg)

type fixtureCfg struct{ enabled bool }

func withEnabled(v bool) fixtureOpt { return func(c *fixtureCfg) { c.enabled = v } }

func newFixture(t *testing.T, opts ...fixtureOpt) *fixture {
	t.Helper()
	cfg := fixtureCfg{enabled: true}
	for _, o := range opts {
		o(&cfg)
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Only the tables the coordinator touches. deleted_at is required because
	// models.Order carries gorm.DeletedAt, so GORM appends
	// "orders.deleted_at IS NULL" to every query — same as the existing refund
	// suites' DDL.
	require.NoError(t, db.Exec(`CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', 
		id TEXT PRIMARY KEY, order_number TEXT DEFAULT '', customer_id TEXT,
		total REAL, refund_amount REAL DEFAULT 0, wallet_applied REAL DEFAULT 0,
		payment_status TEXT, payment_provider TEXT, razorpay_payment_id TEXT,
		stripe_payment_intent_id TEXT DEFAULT '',
		currency TEXT DEFAULT 'INR',
		refunded_at DATETIME,
		created_at DATETIME, updated_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	// The coordinator reserves through refundreserve (#690), which reads the columns
	// above and sums per-line refunds off order_items (#527). Both are part of the
	// shared reservation contract now, so the fixture has to model them.
	require.NoError(t, db.Exec(`CREATE TABLE order_items (
		id TEXT PRIMARY KEY, order_id TEXT, is_cancelled BOOLEAN DEFAULT 0,
		refund_amount REAL DEFAULT 0, subtotal REAL DEFAULT 0, created_at DATETIME
	)`).Error)
	// Raw DDL, not AutoMigrate: the model defaults ID to gen_random_uuid(), a
	// Postgres function sqlite can't parse. Every existing refund test does the
	// same (setupCancelRefundDB, setupIssueDB) — the model stays the source of
	// truth for the real schema via AutoMigrate in production.
	require.NoError(t, db.Exec(`CREATE TABLE refund_transactions (
		id TEXT PRIMARY KEY, order_id TEXT NOT NULL, provider TEXT NOT NULL,
		provider_payment_id TEXT, provider_refund_id TEXT,
		amount REAL NOT NULL, currency_code TEXT NOT NULL DEFAULT 'INR',
		status TEXT NOT NULL DEFAULT 'pending', reason TEXT,
		idempotency_key TEXT NOT NULL UNIQUE, scope_id TEXT NOT NULL,
		actor TEXT, failure_reason TEXT,
		created_at DATETIME, updated_at DATETIME, completed_at DATETIME
	)`).Error)

	g := &fakeGateway{}
	return &fixture{
		t:       t,
		db:      db,
		ctx:     context.Background(),
		gateway: g,
		coord:   NewCoordinator(db, g, cfg.enabled),
	}
}

func (f *fixture) seedPaidOrder(total float64) *models.Order {
	f.t.Helper()
	o := &models.Order{ID: uuid.New(), Total: total}
	require.NoError(f.t, f.db.Exec(
		`INSERT INTO orders (id, total, refund_amount, payment_status, payment_provider, razorpay_payment_id)
		 VALUES (?, ?, 0, 'completed', 'razorpay', 'pay_test')`,
		o.ID.String(), total,
	).Error)
	return o
}

func (f *fixture) seedUnpaidOrder(total float64) *models.Order {
	f.t.Helper()
	o := &models.Order{ID: uuid.New(), Total: total}
	require.NoError(f.t, f.db.Exec(
		`INSERT INTO orders (id, total, refund_amount, payment_status, payment_provider)
		 VALUES (?, ?, 0, 'pending', 'razorpay')`,
		o.ID.String(), total,
	).Error)
	return o
}

func (f *fixture) seedLedger(orderID uuid.UUID, amount float64, scope string, status models.RefundTxnStatus) {
	f.t.Helper()
	row := models.RefundTransaction{
		ID: uuid.New(), OrderID: orderID, Provider: "razorpay",
		ProviderPaymentID: "pay_test", Amount: amount, CurrencyCode: "INR",
		Status: status, ScopeID: scope,
		IdempotencyKey: "seed-" + scope + "-" + uuid.NewString(),
	}
	if status != models.RefundTxnPending {
		now := time.Now()
		row.CompletedAt = &now
	}
	require.NoError(f.t, f.db.Create(&row).Error)
}

func (f *fixture) seedPendingLedger(orderID uuid.UUID, amount float64, scope string) {
	f.seedLedger(orderID, amount, scope, models.RefundTxnPending)
}

// seedInFlightRefund models a refund that is mid-gateway RIGHT NOW, the way the
// coordinator actually leaves the DB: the shared reservation is taken (refund_amount
// incremented + the payment_status claim held) AND the pending ledger row exists —
// both committed together. A pending row on its own is not a state production can
// reach, so seeding one would test fiction.
func (f *fixture) seedInFlightRefund(orderID uuid.UUID, amount float64, scope string) {
	f.t.Helper()
	f.seedPendingLedger(orderID, amount, scope)
	require.NoError(f.t, f.db.Exec(
		`UPDATE orders SET refund_amount = COALESCE(refund_amount,0) + ?, payment_status = ? WHERE id = ?`,
		amount, string(models.PaymentRefunded), orderID.String(),
	).Error)
}

func (f *fixture) seedFailedLedger(orderID uuid.UUID, amount float64, scope string) {
	f.seedLedger(orderID, amount, scope, models.RefundTxnFailed)
}

func (f *fixture) ledgerRows(orderID uuid.UUID) []models.RefundTransaction {
	f.t.Helper()
	var rows []models.RefundTransaction
	require.NoError(f.t, f.db.Where("order_id = ?", orderID).Order("created_at").Find(&rows).Error)
	return rows
}

func (f *fixture) orderRefundAmount(orderID uuid.UUID) float64 {
	f.t.Helper()
	var v float64
	f.db.Raw(`SELECT COALESCE(refund_amount, 0) FROM orders WHERE id = ?`, orderID.String()).Scan(&v)
	return v
}
