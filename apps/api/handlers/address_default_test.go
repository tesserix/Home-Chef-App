package handlers

// address_default_test.go — a prod data bug (multiple default addresses per
// user) silently broke the customer chef-discovery gate: the storefront picks
// ONE "default" address to source delivery coords, and when a user somehow had
// two, downstream behavior became nondeterministic. The un-transactioned
// "clear other defaults" + "create/save this one" pair in CreateAddress /
// UpdateAddress could be interleaved by a concurrent request, tripping the new
// partial unique index (idx_addresses_one_default_per_user, migration
// 20260705000002). These tests pin the handler-level invariant: setting a new
// default address always unsets any prior one, for both create and update.
//
// SQLite harness mirrors the other handler tests: in-memory DB, hand-DDL'd
// `addresses` table (the model's gen_random_uuid() default can't run on
// SQLite — models.Address.BeforeCreate mints the UUID in Go instead), swap
// database.DB, restore in t.Cleanup. The table also carries the real partial
// unique index so a regression that reintroduces two defaults for the same
// user fails loudly here, not in prod.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

const addressesDDL = `CREATE TABLE addresses (
	id text PRIMARY KEY, user_id text, label text DEFAULT '',
	line1 text, line2 text DEFAULT '', city text, state text, postal_code text,
	country text DEFAULT 'US', latitude real DEFAULT 0, longitude real DEFAULT 0,
	is_default integer DEFAULT 0,
	created_at datetime, updated_at datetime)`

// addressesUniqueDefaultIndex mirrors migrations/20260705000002 — SQLite
// supports partial unique indexes with the same syntax as Postgres.
const addressesUniqueDefaultIndex = `CREATE UNIQUE INDEX idx_addresses_one_default_per_user ON addresses (user_id) WHERE is_default`

func setupAddressDB(t *testing.T) (*gorm.DB, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(addressesDDL).Error)
	require.NoError(t, db.Exec(addressesUniqueDefaultIndex).Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	return db, uuid.New()
}

func addressRouter(userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	h := NewAddressHandler()
	r.POST("/addresses", h.CreateAddress)
	r.PUT("/addresses/:id", h.UpdateAddress)
	return r
}

func postAddress(t *testing.T, r *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/addresses", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func putAddress(t *testing.T, r *gin.Engine, id string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/addresses/"+id, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func newAddressBody(label string, isDefault bool) map[string]any {
	return map[string]any{
		"label":      label,
		"line1":      "1 Test St",
		"city":       "Testville",
		"state":      "TS",
		"postalCode": "000001",
		"isDefault":  isDefault,
	}
}

func countDefaults(t *testing.T, db *gorm.DB, userID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM addresses WHERE user_id = ? AND is_default = 1`, userID.String()).Scan(&n).Error)
	return n
}

// TestCreateAddress_NewDefaultUnsetsPriorDefault — creating a second default
// address must leave exactly one default row (the new one).
func TestCreateAddress_NewDefaultUnsetsPriorDefault(t *testing.T) {
	db, userID := setupAddressDB(t)
	r := addressRouter(userID)

	w1 := postAddress(t, r, newAddressBody("Home", true))
	require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())
	require.EqualValues(t, 1, countDefaults(t, db, userID))

	w2 := postAddress(t, r, newAddressBody("Work", true))
	require.Equal(t, http.StatusCreated, w2.Code, w2.Body.String())

	require.EqualValues(t, 1, countDefaults(t, db, userID), "only the newest address should be default")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &body))
	require.Equal(t, true, body["isDefault"])
}

// TestCreateAddress_NonDefaultDoesNotDisturbExistingDefault — creating a
// non-default address must not touch the existing default.
func TestCreateAddress_NonDefaultDoesNotDisturbExistingDefault(t *testing.T) {
	db, userID := setupAddressDB(t)
	r := addressRouter(userID)

	w1 := postAddress(t, r, newAddressBody("Home", true))
	require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())

	w2 := postAddress(t, r, newAddressBody("Cabin", false))
	require.Equal(t, http.StatusCreated, w2.Code, w2.Body.String())

	require.EqualValues(t, 1, countDefaults(t, db, userID))
}

// TestUpdateAddress_PromotingToDefaultUnsetsOthers — flipping an existing
// non-default address to default via UpdateAddress must unset the other one.
func TestUpdateAddress_PromotingToDefaultUnsetsOthers(t *testing.T) {
	db, userID := setupAddressDB(t)
	r := addressRouter(userID)

	w1 := postAddress(t, r, newAddressBody("Home", true))
	require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())
	var home map[string]any
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &home))

	w2 := postAddress(t, r, newAddressBody("Work", false))
	require.Equal(t, http.StatusCreated, w2.Code, w2.Body.String())
	var work map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &work))

	// Promote Work to default.
	workBody := newAddressBody("Work", true)
	w3 := putAddress(t, r, work["id"].(string), workBody)
	require.Equal(t, http.StatusOK, w3.Code, w3.Body.String())

	require.EqualValues(t, 1, countDefaults(t, db, userID), "promoting Work must unset Home's default")

	var homeAfter struct {
		IsDefault bool
	}
	require.NoError(t, db.Raw(`SELECT is_default FROM addresses WHERE id = ?`, home["id"]).Scan(&homeAfter).Error)
	require.False(t, homeAfter.IsDefault, "the previous default must be cleared")
}

// TestUpdateAddress_ReSavingExistingDefaultIsANoop — saving an already-default
// address again (no promotion) must not error or duplicate the clear.
func TestUpdateAddress_ReSavingExistingDefaultIsANoop(t *testing.T) {
	db, userID := setupAddressDB(t)
	r := addressRouter(userID)

	w1 := postAddress(t, r, newAddressBody("Home", true))
	require.Equal(t, http.StatusCreated, w1.Code, w1.Body.String())
	var home map[string]any
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &home))

	w2 := putAddress(t, r, home["id"].(string), newAddressBody("Home Renamed", true))
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
	require.EqualValues(t, 1, countDefaults(t, db, userID))
}
