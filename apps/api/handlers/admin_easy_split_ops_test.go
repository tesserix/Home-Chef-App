package handlers

// admin_easy_split_ops_test.go — #1085. The operator surface lives in
// tesserix-home; these are the two reads it is built on. Both answer questions
// that cannot be answered from the chef list alone: "which chefs can actually
// be split-paid, and if not, what is stopping them", and "for the orders that
// have been paid, which rail settled them and does the amount agree".

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

const opsChefDDL = `CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, user_id TEXT, mode text DEFAULT 'live',
	business_name TEXT DEFAULT '', state TEXT DEFAULT '', payout_country TEXT DEFAULT 'IN',
	payout_method TEXT DEFAULT '', easy_split_mode TEXT DEFAULT '',
	cashfree_vendor_id TEXT DEFAULT '', cashfree_vendor_status TEXT DEFAULT '',
	fssai_override_until DATETIME, is_active INTEGER DEFAULT 1,
	created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`

const opsOrderDDL = `CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT, chef_id TEXT, mode text DEFAULT 'live',
	status TEXT DEFAULT 'delivered', payment_status TEXT DEFAULT 'paid',
	payment_provider TEXT DEFAULT 'cashfree', razorpay_order_id TEXT DEFAULT '',
	subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, total REAL DEFAULT 0,
	tax_food REAL DEFAULT 0, tax_service REAL DEFAULT 0, tax_delivery REAL DEFAULT 0,
	chef_tip REAL DEFAULT 0, driver_tip REAL DEFAULT 0, delivery_fee REAL DEFAULT 0,
	delivery_fee_final REAL, fulfillment_type TEXT DEFAULT 'chef_delivery',
	chef_funded_discount REAL DEFAULT 0, commission_rate REAL DEFAULT 0,
	delivery_address_state TEXT DEFAULT '', wallet_applied REAL DEFAULT 0, loyalty_applied REAL DEFAULT 0,
	gateway_split_paise INTEGER DEFAULT 0,
	created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`

const opsAuditDDL = `CREATE TABLE audit_logs (id TEXT PRIMARY KEY, user_id TEXT, action TEXT, entity_type TEXT,
	entity_id TEXT, old_value TEXT, new_value TEXT, ip_address TEXT, user_agent TEXT,
	correlation_id TEXT, created_at DATETIME)`

const opsDocsDDL = `CREATE TABLE chef_documents (id TEXT PRIMARY KEY, chef_id TEXT, type TEXT, status TEXT,
	expiry_date DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`

func setupEasySplitOpsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	require.NoError(t, err)
	for _, ddl := range []string{opsChefDDL, opsOrderDDL, opsAuditDDL, opsDocsDDL, profileSettingsDDL} {
		require.NoError(t, db.Exec(ddl).Error)
	}

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	prevCfg := config.AppConfig
	config.AppConfig = &config.Config{Environment: "test"}
	t.Cleanup(func() { config.AppConfig = prevCfg })
	return db
}

func opsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAdminPayoutRailHandler()
	r.GET("/admin/payouts/easy-split/roster", h.GetEasySplitRoster)
	r.GET("/admin/payouts/easy-split/orders", h.GetEasySplitOrders)
	return r
}

func opsGet(t *testing.T, r http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	return w
}

func setOpsSetting(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO platform_settings (id, key, value, type, updated_at) VALUES (?,?,?,'string',?)`,
		uuid.NewString(), key, value, time.Now()).Error)
}

func seedOpsChef(t *testing.T, db *gorm.DB, name, vendorStatus, mode string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	vendorID := ""
	if vendorStatus != "" {
		vendorID = services.EasySplitVendorIDFor(id)
	}
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, business_name, state, payout_country, cashfree_vendor_id, cashfree_vendor_status, easy_split_mode, created_at, updated_at)
		 VALUES (?, ?, 'KA', 'IN', ?, ?, ?, ?, ?)`,
		id.String(), name, vendorID, vendorStatus, mode, time.Now(), time.Now()).Error)
	return id
}

func seedOpsOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, number string, splitPaise int, age time.Duration) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, chef_id, status, payment_status, payment_provider, razorpay_order_id,
		 subtotal, total, tax_food, commission_rate, delivery_address_state, gateway_split_paise, created_at, updated_at)
		 VALUES (?, ?, ?, 'delivered', 'completed', 'cashfree', ?, 1000, 1180, 50, 15, 'KA', ?, ?, ?)`,
		id.String(), number, chefID.String(), "cf_"+number, splitPaise,
		time.Now().Add(-age), time.Now().Add(-age)).Error)
	return id
}

type rosterResponse struct {
	GlobalEnabled bool `json:"globalEnabled"`
	WindowFits    bool `json:"windowFits"`
	Chefs         []struct {
		ChefID       string `json:"chefId"`
		BusinessName string `json:"businessName"`
		VendorStatus string `json:"vendorStatus"`
		Mode         string `json:"easySplitMode"`
		Effective    bool   `json:"effective"`
		Payable      bool   `json:"payable"`
		Blocker      string `json:"blocker"`
		SplitOrders  int    `json:"splitOrders"`
		SplitPaise   int    `json:"splitPaise"`
	} `json:"chefs"`
}

// The roster's whole job: not "is the flag on" but "would an order for this
// chef actually split", and when it would not, which guard says no.
func TestGetEasySplitRoster_NamesTheBlockerPerChef(t *testing.T) {
	db := setupEasySplitOpsDB(t)
	setOpsSetting(t, db, services.SettingEasySplitEnabled, "true")
	ready := seedOpsChef(t, db, "Ammas Kitchen", services.CashfreeVendorActive, "")
	pending := seedOpsChef(t, db, "Bakers Nook", services.CashfreeVendorInBankValidation, "")
	held := seedOpsChef(t, db, "Coastal Curry", services.CashfreeVendorActive, services.PayoutAutoOff)
	seedOpsOrder(t, db, ready, "HC-1", 74000, time.Hour)
	seedOpsOrder(t, db, ready, "HC-2", 0, 2*time.Hour)

	var resp rosterResponse
	require.NoError(t, json.Unmarshal(opsGet(t, opsRouter(), "/admin/payouts/easy-split/roster").Body.Bytes(), &resp))

	require.True(t, resp.GlobalEnabled)
	byID := map[string]int{}
	for i, row := range resp.Chefs {
		byID[row.ChefID] = i
	}
	require.Len(t, resp.Chefs, 3)

	r := resp.Chefs[byID[ready.String()]]
	require.True(t, r.Payable)
	require.Empty(t, r.Blocker)
	require.Equal(t, 1, r.SplitOrders)
	require.Equal(t, 74000, r.SplitPaise)

	p := resp.Chefs[byID[pending.String()]]
	require.False(t, p.Payable)
	require.Equal(t, services.EasySplitSkipVendorNotActive, p.Blocker)

	h := resp.Chefs[byID[held.String()]]
	require.False(t, h.Payable)
	require.False(t, h.Effective)
	require.Equal(t, services.EasySplitSkipDisabled, h.Blocker)
}

// A chef nobody has enabled is not an exception — the roster has to say so
// without inventing a vendor problem that does not exist.
func TestGetEasySplitRoster_ReportsTheGlobalFlagAsTheBlocker(t *testing.T) {
	db := setupEasySplitOpsDB(t)
	setOpsSetting(t, db, services.SettingEasySplitEnabled, "false")
	seedOpsChef(t, db, "Ammas Kitchen", services.CashfreeVendorActive, "")

	var resp rosterResponse
	require.NoError(t, json.Unmarshal(opsGet(t, opsRouter(), "/admin/payouts/easy-split/roster").Body.Bytes(), &resp))

	require.False(t, resp.GlobalEnabled)
	require.Len(t, resp.Chefs, 1)
	require.False(t, resp.Chefs[0].Payable)
	require.Equal(t, services.EasySplitSkipDisabled, resp.Chefs[0].Blocker)
}

type ordersResponse struct {
	Summary struct {
		SplitCount     int `json:"splitCount"`
		SplitPaise     int `json:"splitPaise"`
		PayoutCount    int `json:"payoutCount"`
		ExceptionCount int `json:"exceptionCount"`
		DeltaPaise     int `json:"deltaPaise"`
	} `json:"summary"`
	Orders []struct {
		OrderNumber   string `json:"orderNumber"`
		ChefName      string `json:"chefName"`
		Rail          string `json:"rail"`
		ExpectedPaise int    `json:"expectedPaise"`
		SplitPaise    int    `json:"splitPaise"`
		DeltaPaise    int    `json:"deltaPaise"`
		Reason        string `json:"reason"`
		Exception     bool   `json:"exception"`
	} `json:"orders"`
}

func opsOrdersByNumber(t *testing.T, resp ordersResponse, number string) int {
	t.Helper()
	for i, o := range resp.Orders {
		if o.OrderNumber == number {
			return i
		}
	}
	t.Fatalf("order %s missing from the settlement ledger", number)
	return -1
}

// Per order: which rail paid it, what the chef's share should have been, and
// what actually left the capture. A delta is the thing an operator is looking
// for, so it has to be a listed row and not a summary tick.
func TestGetEasySplitOrders_ShowsTheRailAndTheDelta(t *testing.T) {
	db := setupEasySplitOpsDB(t)
	setOpsSetting(t, db, services.SettingEasySplitEnabled, "true")
	chef := seedOpsChef(t, db, "Ammas Kitchen", services.CashfreeVendorActive, "")

	agreed := seedOpsOrder(t, db, chef, "HC-1", 0, time.Hour)
	var order models.Order
	require.NoError(t, db.Preload("Chef").First(&order, "id = ?", agreed.String()).Error)
	expected := services.ToPaise(services.ChefNetPayoutFor(&order))
	require.NoError(t, db.Exec(`UPDATE orders SET gateway_split_paise = ? WHERE id = ?`,
		expected, agreed.String()).Error)
	seedOpsOrder(t, db, chef, "HC-2", expected-2500, 2*time.Hour)

	var resp ordersResponse
	require.NoError(t, json.Unmarshal(opsGet(t, opsRouter(), "/admin/payouts/easy-split/orders?days=7").Body.Bytes(), &resp))

	ok := resp.Orders[opsOrdersByNumber(t, resp, "HC-1")]
	require.Equal(t, "split", ok.Rail)
	require.Equal(t, expected, ok.ExpectedPaise)
	require.Zero(t, ok.DeltaPaise)
	require.False(t, ok.Exception)
	require.Equal(t, "Ammas Kitchen", ok.ChefName)

	short := resp.Orders[opsOrdersByNumber(t, resp, "HC-2")]
	require.Equal(t, "split", short.Rail)
	require.Equal(t, -2500, short.DeltaPaise)
	require.True(t, short.Exception)

	require.Equal(t, 2, resp.Summary.SplitCount)
	require.Equal(t, 1, resp.Summary.ExceptionCount)
	require.Equal(t, -2500, resp.Summary.DeltaPaise)
}

// An order that took the payout rail carries the recorded reason, so the
// operator reads why from what we stored rather than re-deriving it.
func TestGetEasySplitOrders_CarriesTheRecordedSkipReason(t *testing.T) {
	db := setupEasySplitOpsDB(t)
	setOpsSetting(t, db, services.SettingEasySplitEnabled, "true")
	chef := seedOpsChef(t, db, "Ammas Kitchen", services.CashfreeVendorActive, "")
	orderID := seedOpsOrder(t, db, chef, "HC-9", 0, time.Hour)
	require.NoError(t, db.Exec(
		`INSERT INTO audit_logs (id, action, entity_type, entity_id, new_value, created_at) VALUES (?,?,?,?,?,?)`,
		uuid.NewString(), "order.payout.easy_split_skipped", "order", orderID.String(),
		`{"reason":"`+services.EasySplitSkipFSSAIExpired+`"}`, time.Now()).Error)

	var resp ordersResponse
	require.NoError(t, json.Unmarshal(opsGet(t, opsRouter(), "/admin/payouts/easy-split/orders").Body.Bytes(), &resp))

	row := resp.Orders[opsOrdersByNumber(t, resp, "HC-9")]
	require.Equal(t, "payout", row.Rail)
	require.Equal(t, services.EasySplitSkipFSSAIExpired, row.Reason)
	require.Zero(t, row.SplitPaise)
	require.Equal(t, 1, resp.Summary.PayoutCount)
}

// The exception queue is the same read, filtered — an operator working the
// queue must not have to page through the orders that are fine.
func TestGetEasySplitOrders_FiltersToTheExceptionQueue(t *testing.T) {
	db := setupEasySplitOpsDB(t)
	setOpsSetting(t, db, services.SettingEasySplitEnabled, "true")
	chef := seedOpsChef(t, db, "Ammas Kitchen", services.CashfreeVendorActive, "")

	clean := seedOpsOrder(t, db, chef, "HC-1", 0, time.Hour)
	var order models.Order
	require.NoError(t, db.Preload("Chef").First(&order, "id = ?", clean.String()).Error)
	require.NoError(t, db.Exec(`UPDATE orders SET gateway_split_paise = ? WHERE id = ?`,
		services.ToPaise(services.ChefNetPayoutFor(&order)), clean.String()).Error)
	seedOpsOrder(t, db, chef, "HC-2", 1, time.Hour)

	var resp ordersResponse
	require.NoError(t, json.Unmarshal(
		opsGet(t, opsRouter(), "/admin/payouts/easy-split/orders?rail=exception").Body.Bytes(), &resp))

	require.Len(t, resp.Orders, 1)
	require.Equal(t, "HC-2", resp.Orders[0].OrderNumber)
}

// Unpaid orders have nothing to reconcile — including them would fill the
// queue with orders that have not reached the split decision yet.
func TestGetEasySplitOrders_IgnoresUnpaidOrders(t *testing.T) {
	db := setupEasySplitOpsDB(t)
	setOpsSetting(t, db, services.SettingEasySplitEnabled, "true")
	chef := seedOpsChef(t, db, "Ammas Kitchen", services.CashfreeVendorActive, "")
	orderID := seedOpsOrder(t, db, chef, "HC-3", 0, time.Hour)
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = 'pending' WHERE id = ?`, orderID.String()).Error)

	var resp ordersResponse
	require.NoError(t, json.Unmarshal(opsGet(t, opsRouter(), "/admin/payouts/easy-split/orders").Body.Bytes(), &resp))

	require.Empty(t, resp.Orders)
	require.Zero(t, resp.Summary.PayoutCount)
}
