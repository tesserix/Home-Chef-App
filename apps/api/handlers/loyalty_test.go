package handlers

// loyalty_test.go — HTTP-level checks for the customer loyalty endpoints (#40):
// the balance/tier payload, redeem-to-wallet happy path, and the guard rails
// (below-minimum, insufficient points). The ledger math itself is pinned in the
// services package; here we verify the handler wiring, status codes, and that a
// redeem actually moves both ledgers.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func setupLoyaltyHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE loyalty_accounts (
			id text PRIMARY KEY, user_id text UNIQUE, balance real DEFAULT 0,
			lifetime_points real DEFAULT 0, tier text DEFAULT 'bronze',
			current_streak integer DEFAULT 0, longest_streak integer DEFAULT 0,
			last_streak_day datetime, created_at datetime, updated_at datetime)`,
		`CREATE TABLE loyalty_transactions (
			id text PRIMARY KEY, loyalty_account_id text, user_id text, type text, source text,
			points real, points_after real, order_id text, reason text,
			created_by text, idempotency_key text UNIQUE, created_at datetime)`,
		`CREATE TABLE wallets (
			id text PRIMARY KEY, user_id text UNIQUE, balance real DEFAULT 0,
			currency text DEFAULT 'INR', created_at datetime, updated_at datetime)`,
		`CREATE TABLE wallet_txns (
			id text PRIMARY KEY, wallet_id text, user_id text, type text, source text,
			amount real, balance_after real, currency text, order_id text, reason text,
			created_by text, idempotency_key text UNIQUE, created_at datetime)`,
		`CREATE TABLE platform_settings (
			id text PRIMARY KEY, key text UNIQUE, value text, type text, updated_by text, updated_at datetime)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT, aggregate_id TEXT,
			payload TEXT, status TEXT, attempts INT, last_error TEXT, next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func loyaltyReq(t *testing.T, userID uuid.UUID, method, path string, register func(*gin.Engine, *LoyaltyHandler), body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	register(r, NewLoyaltyHandler())
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetLoyalty_ReturnsBalanceAndConfig(t *testing.T) {
	db := setupLoyaltyHandlerDB(t)
	uid := uuid.New()
	_, err := services.EarnLoyalty(db, uid, 1200, models.LoyaltySourceOrder, nil, "seed", "loyalty:order:seed")
	require.NoError(t, err)

	w := loyaltyReq(t, uid, http.MethodGet, "/loyalty",
		func(r *gin.Engine, h *LoyaltyHandler) { r.GET("/loyalty", h.GetLoyalty) }, nil)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1200.0, resp["balance"])
	assert.Equal(t, models.LoyaltyTierSilver, resp["tier"]) // 1200 ≥ 1000 default
	assert.NotNil(t, resp["config"])
}

func TestRedeemLoyalty_HappyPath(t *testing.T) {
	db := setupLoyaltyHandlerDB(t)
	uid := uuid.New()
	_, err := services.EarnLoyalty(db, uid, 500, models.LoyaltySourceOrder, nil, "seed", "loyalty:order:seed")
	require.NoError(t, err)

	w := loyaltyReq(t, uid, http.MethodPost, "/loyalty/redeem",
		func(r *gin.Engine, h *LoyaltyHandler) { r.POST("/loyalty/redeem", h.RedeemLoyalty) },
		map[string]float64{"points": 300})
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 300.0, resp["pointsRedeemed"])
	assert.Equal(t, 200.0, resp["pointsBalance"])
	assert.Equal(t, 30.0, resp["walletCredited"]) // 300 × 0.1 ₹/pt

	// Both ledgers moved.
	acct, _ := services.LoyaltyBalance(db, uid)
	assert.Equal(t, 200.0, acct.Balance)
	wallet, _ := services.WalletBalance(db, uid)
	assert.Equal(t, 30.0, wallet.Balance)
}

func TestRedeemLoyalty_BelowMinimum(t *testing.T) {
	db := setupLoyaltyHandlerDB(t)
	uid := uuid.New()
	_, err := services.EarnLoyalty(db, uid, 500, models.LoyaltySourceOrder, nil, "seed", "loyalty:order:seed")
	require.NoError(t, err)

	w := loyaltyReq(t, uid, http.MethodPost, "/loyalty/redeem",
		func(r *gin.Engine, h *LoyaltyHandler) { r.POST("/loyalty/redeem", h.RedeemLoyalty) },
		map[string]float64{"points": 50})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRedeemLoyalty_InsufficientPoints(t *testing.T) {
	db := setupLoyaltyHandlerDB(t)
	uid := uuid.New()
	_, err := services.EarnLoyalty(db, uid, 120, models.LoyaltySourceOrder, nil, "seed", "loyalty:order:seed")
	require.NoError(t, err)

	w := loyaltyReq(t, uid, http.MethodPost, "/loyalty/redeem",
		func(r *gin.Engine, h *LoyaltyHandler) { r.POST("/loyalty/redeem", h.RedeemLoyalty) },
		map[string]float64{"points": 200})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
