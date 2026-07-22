package handlers

// chef_payout_gate_test.go — #739. Proves the payout-setup gate actually
// refuses the transition into accepting orders at the HTTP edge, not merely
// that the policy function returns false.
//
// The scenario that motivates the gate: today a chef can be approved, switch
// on accepting_orders, take a customer's money, deliver, dual-confirm and
// accrue released holds with no payout destination on file.

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

	"github.com/homechef/api/database"
)

const payoutGateChefDDL = `CREATE TABLE chef_profiles (
	id text PRIMARY KEY, user_id text, business_name text,
	is_active integer DEFAULT 1, accepting_orders integer DEFAULT 0,
	paused_until datetime, payout_method text DEFAULT '',
	created_at datetime, updated_at datetime
)`

const platformSettingsDDL = `CREATE TABLE platform_settings (
	id text PRIMARY KEY, key text UNIQUE, value text,
	type text DEFAULT 'string', updated_by text, updated_at datetime
)`

func setupPayoutGateDB(t *testing.T, payoutMethod string) (*gorm.DB, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{payoutGateChefDDL, platformSettingsDDL, auditDDL} {
		require.NoError(t, db.Exec(s).Error)
	}

	userID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, is_active, accepting_orders, payout_method)
		 VALUES (?,?,?,1,0,?)`,
		chefID.String(), userID.String(), "Test Kitchen", payoutMethod).Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db, userID
}

func setSetting(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO platform_settings (id, key, value) VALUES (?,?,?)`,
		uuid.NewString(), key, value).Error)
}

// postResume drives ResumeReceiving, the simplest path that switches
// accepting_orders on.
func postResume(t *testing.T, userID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.POST("/chef/availability/resume", (&ChefAvailabilityHandler{}).ResumeReceiving)

	req := httptest.NewRequest(http.MethodPost, "/chef/availability/resume", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func acceptingOrders(t *testing.T, db *gorm.DB, userID uuid.UUID) bool {
	t.Helper()
	var accepting bool
	require.NoError(t, db.Raw(
		`SELECT accepting_orders FROM chef_profiles WHERE user_id = ?`, userID.String()).
		Scan(&accepting).Error)
	return accepting
}

func TestPayoutGate_BlocksGoingOnlineWithNoPayoutMethod(t *testing.T) {
	db, userID := setupPayoutGateDB(t, "")
	setSetting(t, db, "payout.setup_gate_level", "method_on_file")

	w := postResume(t, userID)

	require.Equal(t, http.StatusConflict, w.Code, "a chef with no payout destination must not be able to start taking money")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "payout_method_missing", body["reasonCode"],
		"the reason code is the contract the vendor app deep-links on")
	require.Equal(t, "setup_payout", body["action"])

	require.False(t, acceptingOrders(t, db, userID), "the block must not have partially applied")
}

func TestPayoutGate_AllowsGoingOnlineWithAPayoutMethod(t *testing.T) {
	db, userID := setupPayoutGateDB(t, "upi")
	setSetting(t, db, "payout.setup_gate_level", "method_on_file")

	w := postResume(t, userID)

	require.Equal(t, http.StatusOK, w.Code)
	require.True(t, acceptingOrders(t, db, userID))
}

func TestPayoutGate_OffByDefault(t *testing.T) {
	// No setting row at all — the gate must ship inert, so deploying it
	// cannot take anyone offline.
	db, userID := setupPayoutGateDB(t, "")

	w := postResume(t, userID)

	require.Equal(t, http.StatusOK, w.Code, "the gate must be off when unconfigured")
	require.True(t, acceptingOrders(t, db, userID))
}

func TestPayoutGate_InvalidSettingFailsOpen(t *testing.T) {
	// A typo in the settings row must not halt trading platform-wide. This is
	// the one place the gate deliberately fails open rather than closed.
	db, userID := setupPayoutGateDB(t, "")
	setSetting(t, db, "payout.setup_gate_level", "yes-please")

	w := postResume(t, userID)

	require.Equal(t, http.StatusOK, w.Code, "an unparseable gate level must not block every chef")
}

func TestPayoutGate_GraceWindowNagsRatherThanBlocks(t *testing.T) {
	// Turning the gate on must not strand the existing cohort of approved
	// chefs who never had a reason to add a payout method.
	db, userID := setupPayoutGateDB(t, "")
	setSetting(t, db, "payout.setup_gate_level", "method_on_file")
	setSetting(t, db, "payout.setup_grace_until", time.Now().Add(48*time.Hour).UTC().Format(time.RFC3339))

	w := postResume(t, userID)

	require.Equal(t, http.StatusOK, w.Code, "during grace the gate reports but does not enforce")
	require.True(t, acceptingOrders(t, db, userID))
}

func TestPayoutGate_ExpiredGraceEnforces(t *testing.T) {
	db, userID := setupPayoutGateDB(t, "")
	setSetting(t, db, "payout.setup_gate_level", "method_on_file")
	setSetting(t, db, "payout.setup_grace_until", time.Now().Add(-time.Hour).UTC().Format(time.RFC3339))

	w := postResume(t, userID)

	require.Equal(t, http.StatusConflict, w.Code, "once grace expires the gate must bite")
}

func TestPayoutGate_ReadinessEndpointReportsDuringGrace(t *testing.T) {
	// The vendor app needs to know it should nag even while enforcement is
	// suppressed, so readiness and enforcement are reported separately.
	db, userID := setupPayoutGateDB(t, "")
	setSetting(t, db, "payout.setup_gate_level", "method_on_file")
	setSetting(t, db, "payout.setup_grace_until", time.Now().Add(48*time.Hour).UTC().Format(time.RFC3339))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.GET("/chef/payout/readiness", (&ChefHandler{}).GetPayoutReadiness)
	req := httptest.NewRequest(http.MethodGet, "/chef/payout/readiness", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, false, body["ready"], "readiness must still report the gap")
	require.Equal(t, false, body["enforced"], "but enforcement is suppressed during grace")
	require.Equal(t, true, body["graceActive"])
	require.Equal(t, "payout_method_missing", body["reasonCode"])
	_ = db
}
