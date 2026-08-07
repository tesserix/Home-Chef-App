package handlers

// charge_legs_cashfree_only_test.go — #1086 Phase 5. The three non-order charges
// (featured listing, catering deposit, group-order share) mint their charge on
// Cashfree or refuse.
//
// Each of these used to end in a Razorpay leg reached whenever the resolved rail
// was not Cashfree. Selection can no longer return razorpay, so that leg was
// unreachable for an INR kitchen — but it was still reachable for a STRIPE one,
// which would have been charged in the wrong currency on a gateway the chef is
// not registered with. Refusing is the honest answer: none of these three flows
// has a Stripe implementation.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// chargeStub points the live Cashfree slot at a stub and records every
// create-order body it receives.
func chargeStub(t *testing.T) *[]map[string]any {
	t.Helper()
	bodies := &[]map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		if _, isCreate := body["order_amount"]; isCreate {
			*bodies = append(*bodies, body)
		}
		orderID, _ := body["order_id"].(string)
		amount, _ := body["order_amount"].(float64)
		_, _ = w.Write([]byte(fmt.Sprintf(
			`{"cf_order_id":1,"order_id":%q,"payment_session_id":"sess_x","order_status":"ACTIVE","order_amount":%.2f}`,
			orderID, amount)))
	}))
	t.Cleanup(srv.Close)
	services.SetCashfreeClient(
		services.NewCashfreeTestClient(srv.URL, "app_test", "secret_test", cfTestWebhookSecret, models.ChefModeLive))
	t.Cleanup(func() { services.SetCashfreeClient(nil) })
	return bodies
}

// setupChargeDB adds the tables the three charge endpoints touch on top of the
// shared payment harness.
func setupChargeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupPayDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE chef_promotions (mode text DEFAULT 'live', test_session_id text, cloned_from_id text,
		id TEXT PRIMARY KEY, chef_id TEXT, status TEXT DEFAULT 'pending', amount REAL DEFAULT 0,
		currency TEXT DEFAULT 'INR', duration INTEGER DEFAULT 30, starts_at DATETIME, expires_at DATETIME,
		gateway_order_id TEXT DEFAULT '', gateway_payment_id TEXT DEFAULT '',
		payment_provider TEXT DEFAULT 'razorpay', payment_method TEXT DEFAULT '',
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE catering_requests (mode text DEFAULT 'live', test_session_id text, cloned_from_id text,
		event_address_enc text DEFAULT '', contact_phone_enc text DEFAULT '',
		id TEXT PRIMARY KEY, customer_id TEXT, chef_id TEXT, accepted_quote_id TEXT,
		status TEXT DEFAULT 'accepted', deposit_amount REAL DEFAULT 0, deposit_status TEXT DEFAULT '',
		gateway_order_id TEXT DEFAULT '', gateway_payment_id TEXT DEFAULT '',
		payment_provider TEXT DEFAULT 'razorpay', deposit_paid_at DATETIME,
		event_type TEXT DEFAULT 'wedding', event_date DATETIME,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE catering_quotes (id TEXT PRIMARY KEY, request_id TEXT, chef_id TEXT,
		created_at DATETIME, updated_at DATETIME)`).Error)
	// setupPayDB's group_orders is a stub for a COUNT probe; PayGroupShare loads
	// the whole aggregate, so it is replaced here with the columns it reads.
	require.NoError(t, db.Exec(`DROP TABLE group_orders`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE group_orders (mode text DEFAULT 'live', test_session_id text, cloned_from_id text,
		delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '',
		id TEXT PRIMARY KEY, host_id TEXT, chef_id TEXT, order_id TEXT, type TEXT DEFAULT 'personal',
		split_mode TEXT DEFAULT 'split', join_token TEXT, status TEXT DEFAULT 'open',
		currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE group_order_participants (display_name_enc text DEFAULT '',
		id TEXT PRIMARY KEY, group_order_id TEXT, user_id TEXT, role TEXT DEFAULT 'guest',
		display_name TEXT DEFAULT '', share_amount REAL DEFAULT 0, payment_status TEXT DEFAULT 'pending',
		gateway_order_id TEXT DEFAULT '', gateway_payment_id TEXT DEFAULT '',
		payment_provider TEXT DEFAULT 'razorpay', joined_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE group_order_items (id TEXT PRIMARY KEY, group_order_id TEXT,
		participant_id TEXT, created_at DATETIME, updated_at DATETIME)`).Error)
	return db
}

// chargeChef seeds a chef on the given provider and returns (userID, chefID).
func chargeChef(t *testing.T, db *gorm.DB, provider string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	userID := payUser(t, db, "chef")
	chefID := payChef(t, db, userID)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET payment_provider = ? WHERE id = ?`,
		provider, chefID.String()).Error)
	return userID, chefID
}

func callAs(userID uuid.UUID, method, path string, register func(*gin.Engine), body any) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	register(r)
	var reader io.Reader = http.NoBody
	if body != nil {
		buf, _ := json.Marshal(body)
		reader = bytes.NewReader(buf)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── featured listing ─────────────────────────────────────────────────────────

func TestPurchaseFeaturedAd_MintsACashfreeCharge(t *testing.T) {
	db := setupChargeDB(t)
	bodies := chargeStub(t)
	userID, chefID := chargeChef(t, db, models.PaymentProviderCashfree)

	w := callAs(userID, http.MethodPost, "/chef/promotion/purchase", func(r *gin.Engine) {
		r.POST("/chef/promotion/purchase", NewPromotionHandler().PurchaseFeaturedAd)
	}, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Len(t, *bodies, 1)

	// The response is the assertion here rather than the stored row: the id column
	// carries a Postgres gen_random_uuid() default that sqlite does not apply, so
	// the handler's update-by-primary-key finds nothing in this harness.
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, models.PaymentProviderCashfree, body["provider"])
	require.NotEmpty(t, body["cashfreePaymentSessionId"])

	var count int
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM chef_promotions WHERE chef_id = ?`,
		chefID.String()).Scan(&count).Error)
	require.Equal(t, 1, count)
}

// A Stripe kitchen has no featured-listing implementation. It used to fall
// through to the Razorpay leg and be charged in the wrong currency on a gateway
// it is not registered with.
func TestPurchaseFeaturedAd_RefusesANonCashfreeRail(t *testing.T) {
	db := setupChargeDB(t)
	chargeStub(t)
	userID, chefID := chargeChef(t, db, models.PaymentProviderStripe)

	w := callAs(userID, http.MethodPost, "/chef/promotion/purchase", func(r *gin.Engine) {
		r.POST("/chef/promotion/purchase", NewPromotionHandler().PurchaseFeaturedAd)
	}, nil)
	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())

	var count int
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM chef_promotions WHERE chef_id = ?`,
		chefID.String()).Scan(&count).Error)
	require.Zero(t, count, "a refused purchase must not leave a pending promotion behind")
}

// ── catering deposit ─────────────────────────────────────────────────────────

func seedCateringDeposit(t *testing.T, db *gorm.DB, customerID, chefID uuid.UUID) uuid.UUID {
	t.Helper()
	reqID, quoteID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO catering_quotes (id, request_id, chef_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, quoteID.String(), reqID.String(), chefID.String(), time.Now(), time.Now()).Error)
	require.NoError(t, db.Exec(`INSERT INTO catering_requests
		(id, customer_id, chef_id, accepted_quote_id, status, deposit_amount, deposit_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'accepted', 500, '', ?, ?)`,
		reqID.String(), customerID.String(), chefID.String(), quoteID.String(), time.Now(), time.Now()).Error)
	return reqID
}

func TestCreateCateringDeposit_MintsACashfreeCharge(t *testing.T) {
	db := setupChargeDB(t)
	bodies := chargeStub(t)
	cateringDepositsOn(t)
	_, chefID := chargeChef(t, db, models.PaymentProviderCashfree)
	customerID := payUser(t, db, "customer")
	reqID := seedCateringDeposit(t, db, customerID, chefID)

	w := callAs(customerID, http.MethodPost, "/catering/requests/"+reqID.String()+"/deposit",
		func(r *gin.Engine) {
			r.POST("/catering/requests/:id/deposit", NewCateringHandler().CreateDeposit)
		}, nil)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Len(t, *bodies, 1)
}

func TestCreateCateringDeposit_RefusesANonCashfreeRail(t *testing.T) {
	db := setupChargeDB(t)
	chargeStub(t)
	cateringDepositsOn(t)
	_, chefID := chargeChef(t, db, models.PaymentProviderStripe)
	customerID := payUser(t, db, "customer")
	reqID := seedCateringDeposit(t, db, customerID, chefID)

	w := callAs(customerID, http.MethodPost, "/catering/requests/"+reqID.String()+"/deposit",
		func(r *gin.Engine) {
			r.POST("/catering/requests/:id/deposit", NewCateringHandler().CreateDeposit)
		}, nil)
	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())

	var stamped string
	require.NoError(t, db.Raw(`SELECT gateway_order_id FROM catering_requests WHERE id = ?`,
		reqID.String()).Scan(&stamped).Error)
	require.Empty(t, stamped, "a refused deposit must not stamp a gateway order id")
}

// ── group-order share ────────────────────────────────────────────────────────

func seedGroupShare(t *testing.T, db *gorm.DB, chefID, participantUser uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()
	groupID, partID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO group_orders (id, host_id, chef_id, status, currency, join_token, created_at, updated_at)
		VALUES (?, ?, ?, 'locked', 'INR', ?, ?, ?)`,
		groupID.String(), participantUser.String(), chefID.String(), uuid.NewString(), time.Now(), time.Now()).Error)
	require.NoError(t, db.Exec(`INSERT INTO group_order_participants
		(id, group_order_id, user_id, role, share_amount, payment_status, joined_at, updated_at)
		VALUES (?, ?, ?, 'host', 250, 'pending', ?, ?)`,
		partID.String(), groupID.String(), participantUser.String(), time.Now(), time.Now()).Error)
	return groupID, partID
}

func TestPayGroupShare_MintsACashfreeCharge(t *testing.T) {
	db := setupChargeDB(t)
	bodies := chargeStub(t)
	_, chefID := chargeChef(t, db, models.PaymentProviderCashfree)
	payer := payUser(t, db, "customer")
	groupID, partID := seedGroupShare(t, db, chefID, payer)

	w := callAs(payer, http.MethodPost, "/group-orders/"+groupID.String()+"/pay", func(r *gin.Engine) {
		r.POST("/group-orders/:id/pay", NewGroupOrderHandler().PayGroupShare)
	}, nil)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Len(t, *bodies, 1)

	var provider string
	require.NoError(t, db.Raw(`SELECT payment_provider FROM group_order_participants WHERE id = ?`,
		partID.String()).Scan(&provider).Error)
	require.Equal(t, models.PaymentProviderCashfree, provider)
}

func TestPayGroupShare_RefusesANonCashfreeRail(t *testing.T) {
	db := setupChargeDB(t)
	chargeStub(t)
	_, chefID := chargeChef(t, db, models.PaymentProviderStripe)
	payer := payUser(t, db, "customer")
	groupID, partID := seedGroupShare(t, db, chefID, payer)

	w := callAs(payer, http.MethodPost, "/group-orders/"+groupID.String()+"/pay", func(r *gin.Engine) {
		r.POST("/group-orders/:id/pay", NewGroupOrderHandler().PayGroupShare)
	}, nil)
	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())

	var stamped string
	require.NoError(t, db.Raw(`SELECT gateway_order_id FROM group_order_participants WHERE id = ?`,
		partID.String()).Scan(&stamped).Error)
	require.Empty(t, stamped, "a refused share must not stamp a gateway order id")
}

func cateringDepositsOn(t *testing.T) {
	t.Helper()
	prev := config.AppConfig
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{}
	}
	config.AppConfig.CateringDepositEnabled = true
	t.Cleanup(func() { config.AppConfig = prev })
	_ = database.DB
}
