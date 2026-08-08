package handlers

// admin_chef_payout_profile_test.go — the per-chef admin payout surface.
// Same recipe as admin_payout_test.go: in-memory sqlite, hand-DDL'd tables,
// swap database.DB, restore in t.Cleanup. The rail happy paths (beneficiary
// registration, sandbox seeding) need Secret Manager and the Cashfree sandbox,
// so they live in the cfsandbox-tagged live test; here we pin the guards —
// which are the money-safety part of these endpoints.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	"github.com/homechef/api/services"
)

const profileChefDDL = `CREATE TABLE chef_profiles (mode text DEFAULT 'live', first_live_at datetime, active_test_session_id text,
	address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY,
	business_name TEXT DEFAULT '', payout_method TEXT DEFAULT '', payout_auto_release TEXT DEFAULT '', payout_auto_disburse TEXT DEFAULT '',
	easy_split_mode TEXT DEFAULT '', cashfree_vendor_id TEXT DEFAULT '', cashfree_vendor_status TEXT DEFAULT '',
	cashfree_test_vendor_id TEXT DEFAULT '', cashfree_test_vendor_status TEXT DEFAULT '',
	updated_at DATETIME)`

const profileSettingsDDL = `CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT UNIQUE, value TEXT,
	type TEXT DEFAULT 'string', updated_by TEXT, updated_at DATETIME)`

const profileMethodsDDL = `CREATE TABLE payout_methods (id TEXT PRIMARY KEY, tenant_id TEXT, payee_type TEXT, payee_id TEXT,
	kind TEXT, status TEXT, "primary" INTEGER DEFAULT 0, display_hint TEXT DEFAULT '', beneficiary_name TEXT DEFAULT '',
	rail TEXT DEFAULT '', rail_beneficiary_id TEXT DEFAULT '', rail_status_detail TEXT DEFAULT '',
	verified_at DATETIME, created_at DATETIME, updated_at DATETIME)`

func setupChefPayoutProfileDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(profileChefDDL).Error)
	require.NoError(t, db.Exec(profileMethodsDDL).Error)
	require.NoError(t, db.Exec(profileSettingsDDL).Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	// The secret-name builder consults the environment; without this the
	// profile's Secret Manager reads would nil-panic before failing cleanly.
	prevCfg := config.AppConfig
	config.AppConfig = &config.Config{Environment: "test"}
	t.Cleanup(func() { config.AppConfig = prevCfg })
	return db
}

func profileRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAdminPayoutRailHandler()
	r.GET("/admin/chefs/:id/payout-profile", h.GetChefPayoutProfile)
	r.POST("/admin/chefs/:id/payout-methods/refresh", h.RefreshChefPayoutMethod)
	r.POST("/admin/chefs/:id/payout-methods/test-bank", h.SeedChefTestBankAccount)
	r.PUT("/admin/chefs/:id/disburse-automation", h.SetChefDisburseAutomation)
	r.PUT("/admin/chefs/:id/easy-split-mode", h.SetChefEasySplitMode)
	r.PUT("/admin/chefs/easy-split-mode", h.SetEasySplitModeBulk)
	r.GET("/admin/payouts/settings", h.GetPayoutSettings)
	r.PUT("/admin/payouts/settings", h.UpdatePayoutSettings)
	return r
}

func TestGetChefPayoutProfile(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()

	chefID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, mode, business_name, payout_method) VALUES (?, 'test', 'Ammas Kitchen', 'bank_transfer')`,
		chefID.String()).Error)
	verifiedAt := time.Now().UTC()
	require.NoError(t, db.Exec(
		`INSERT INTO payout_methods (id, tenant_id, payee_type, payee_id, kind, status, "primary", display_hint, beneficiary_name, rail, rail_beneficiary_id, verified_at)
		 VALUES (?, 'homechef', 'chef', ?, 'bank_account', 'verified', 1, '••••1772', 'Ammas Kitchen', 'cashfree_payouts', 'hc_chef_abc', ?)`,
		uuid.New().String(), chefID.String(), verifiedAt).Error)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/chefs/"+chefID.String()+"/payout-profile", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Chef struct {
			BusinessName string `json:"businessName"`
			Mode         string `json:"mode"`
		} `json:"chef"`
		PayoutMethod string `json:"payoutMethod"`
		Methods      []struct {
			Kind        string `json:"kind"`
			Status      string `json:"status"`
			DisplayHint string `json:"displayHint"`
			Payable     bool   `json:"payable"`
		} `json:"methods"`
		Rail struct {
			Configured bool `json:"configured"`
		} `json:"rail"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "Ammas Kitchen", resp.Chef.BusinessName)
	require.Equal(t, "bank_transfer", resp.PayoutMethod)
	require.Len(t, resp.Methods, 1)
	require.Equal(t, "bank_account", resp.Methods[0].Kind)
	require.Equal(t, "verified", resp.Methods[0].Status)
	require.Equal(t, "••••1772", resp.Methods[0].DisplayHint)
	require.True(t, resp.Methods[0].Payable)
	// No Secret Manager in tests — the rail must report unconfigured, never panic.
	require.False(t, resp.Rail.Configured)
}

func TestGetChefPayoutProfile_UnknownChef(t *testing.T) {
	setupChefPayoutProfileDB(t)
	r := profileRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/chefs/"+uuid.NewString()+"/payout-profile", nil))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestRefreshChefPayoutMethod_RailUnconfigured(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?, 'test')`, chefID.String()).Error)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/chefs/"+chefID.String()+"/payout-methods/refresh", nil))
	require.Equal(t, http.StatusConflict, w.Code)
	require.Contains(t, w.Body.String(), "not configured")
}

// TestSeedChefTestBankAccount_RefusesLiveRail is the guard that matters: a chef
// whose mode resolves to the PRODUCTION rail must never get a fabricated bank
// account registered — that is a destination real money can be sent to.
func TestSeedChefTestBankAccount_RefusesLiveRail(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?, 'live')`, chefID.String()).Error)

	// A client whose base URL is NOT the Cashfree sandbox — i.e. production.
	services.SetCashfreePayoutClientFor("live",
		services.NewCashfreePayoutTestClient("https://api.cashfree.com/payout", "CFLIVE", "secret", "", "live"))
	t.Cleanup(func() { services.InvalidateCashfreePayoutFor("live") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/chefs/"+chefID.String()+"/payout-methods/test-bank", nil))
	require.Equal(t, http.StatusConflict, w.Code)
	require.Contains(t, w.Body.String(), "LIVE")
}

func TestSetChefDisburseAutomation(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?, 'test')`, chefID.String()).Error)

	put := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut,
			"/admin/chefs/"+chefID.String()+"/disburse-automation", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	w := put(`{"value":"on"}`)
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		PayoutAutoDisburse string `json:"payoutAutoDisburse"`
		Effective          bool   `json:"effective"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "on", resp.PayoutAutoDisburse)
	require.True(t, resp.Effective)

	var stored string
	require.NoError(t, db.Raw(`SELECT payout_auto_disburse FROM chef_profiles WHERE id = ?`, chefID.String()).Scan(&stored).Error)
	require.Equal(t, "on", stored)

	// "off" wins over any global default; "" follows it (global unset → false).
	require.Equal(t, http.StatusOK, put(`{"value":"off"}`).Code)
	require.Equal(t, http.StatusOK, put(`{"value":""}`).Code)

	// An unrecognised value must never be stored — it would read back as
	// "follow the default" and silently re-enable automation.
	require.Equal(t, http.StatusBadRequest, put(`{"value":"maybe"}`).Code)
	require.NoError(t, db.Raw(`SELECT payout_auto_disburse FROM chef_profiles WHERE id = ?`, chefID.String()).Scan(&stored).Error)
	require.Equal(t, "", stored)
}

func TestSetChefDisburseAutomation_UnknownChef(t *testing.T) {
	setupChefPayoutProfileDB(t)
	r := profileRouter()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut,
		"/admin/chefs/"+uuid.NewString()+"/disburse-automation", strings.NewReader(`{"value":"on"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetChefPayoutProfile_AutomationBlock(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	chefID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, mode, payout_auto_disburse) VALUES (?, 'test', 'off')`,
		chefID.String()).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO platform_settings (id, key, value) VALUES (?, 'payout_auto_disburse_enabled', 'true'), (?, 'payout_auto_disburse_max_minor', '2500000')`,
		uuid.NewString(), uuid.NewString()).Error)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/chefs/"+chefID.String()+"/payout-profile", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Automation struct {
			Disburse           string `json:"disburse"`
			GlobalAutoDisburse bool   `json:"globalAutoDisburse"`
			Effective          bool   `json:"effective"`
			AutoCapMinor       int64  `json:"autoCapMinor"`
			AutoCapUnreadable  bool   `json:"autoCapUnreadable"`
		} `json:"automation"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "off", resp.Automation.Disburse)
	require.True(t, resp.Automation.GlobalAutoDisburse)
	// The per-chef "off" beats the global "on" — that asymmetry is the point.
	require.False(t, resp.Automation.Effective)
	require.EqualValues(t, 2500000, resp.Automation.AutoCapMinor)
	require.False(t, resp.Automation.AutoCapUnreadable)
}

func TestPayoutSettings_CapRoundTrip(t *testing.T) {
	setupChefPayoutProfileDB(t)
	r := profileRouter()

	put := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/admin/payouts/settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}

	w := put(`{"autoDisburseEnabled":true,"autoCapMinor":2500000}`)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		AutoDisburseEnabled bool  `json:"autoDisburseEnabled"`
		AutoCapMinor        int64 `json:"autoCapMinor"`
		AutoCapUnreadable   bool  `json:"autoCapUnreadable"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.AutoDisburseEnabled)
	require.EqualValues(t, 2500000, resp.AutoCapMinor)
	require.False(t, resp.AutoCapUnreadable)

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/payouts/settings", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.AutoDisburseEnabled)
	require.EqualValues(t, 2500000, resp.AutoCapMinor)

	require.Equal(t, http.StatusBadRequest, put(`{"autoCapMinor":-1}`).Code)
	require.Equal(t, http.StatusBadRequest, put(`{}`).Code)

	// Zero clears the cap — "no amount limit", not "block everything".
	w = put(`{"autoCapMinor":0}`)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.EqualValues(t, 0, resp.AutoCapMinor)
	require.False(t, resp.AutoCapUnreadable)
}

func TestSeedChefTestBankAccount_RailUnconfigured(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, mode) VALUES (?, 'test')`, chefID.String()).Error)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/admin/chefs/"+chefID.String()+"/payout-methods/test-bank", nil))
	require.Equal(t, http.StatusConflict, w.Code)
}

// The platform's cut of every order is the flat commission rate — already
// configured, already frozen onto each order, already netted off inside
// ChefNetPayoutFor before a split is built. The settings endpoint must report
// it, because an operator looking at an empty "flat platform fee" box otherwise
// reads it as the platform taking nothing and invents a second figure.
func TestGetPayoutSettings_ReportsTheCommissionRateOrdersAlreadyUse(t *testing.T) {
	db := setupChefPayoutProfileDB(t)
	r := profileRouter()

	read := func() float64 {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/payouts/settings", nil))
		require.Equal(t, http.StatusOK, w.Code)
		var resp struct {
			CommissionRatePercent float64 `json:"commissionRatePercent"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		return resp.CommissionRatePercent
	}

	// Unconfigured falls back to the launch default rather than reading as zero.
	require.InDelta(t, services.DefaultCommissionRate*100, read(), 0.001)

	require.NoError(t, db.Exec(
		`INSERT INTO platform_settings (id, key, value, type) VALUES (?, 'payout.commission_rate', '0.085', 'number')`,
		uuid.New().String()).Error)
	require.InDelta(t, 8.5, read(), 0.001)
}
