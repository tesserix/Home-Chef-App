package handlers

// driver_payout_upi_test.go — #767. UPI is not an accepted payout destination
// for drivers either: Razorpay Route settles by NEFT/IMPS to a bank account and
// has no VPA destination, so a driver who nominated UPI could never be paid.
// DriverOnboardingPayout must reject the request at the edge (400) before it
// touches the database.

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

func setupDriverPayoutDB(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	// A minimal table is enough: a rejected UPI request must be turned away
	// before the handler ever reads a row. Seeding a real partner proves the
	// rejection is not just "profile not found" masquerading as success.
	require.NoError(t, db.Exec(
		`CREATE TABLE delivery_partners (id TEXT PRIMARY KEY, user_id TEXT, payout_method TEXT DEFAULT '', onboarding_step INTEGER DEFAULT 0)`).Error)

	userID, driverID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO delivery_partners (id, user_id, onboarding_step) VALUES (?,?,3)`,
		driverID.String(), userID.String()).Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return userID, driverID
}

func postDriverPayout(t *testing.T, userID uuid.UUID, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.POST("/driver/onboarding/payout", (&DriverOnboardingHandler{}).DriverOnboardingPayout)

	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/driver/onboarding/payout", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDriverOnboardingPayout_UpiRejected(t *testing.T) {
	userID, driverID := setupDriverPayoutDB(t)

	w := postDriverPayout(t, userID, map[string]any{
		"payoutMethod": "upi",
		"upiId":        "driver@upi",
	})
	require.Equal(t, http.StatusBadRequest, w.Code, "UPI is not payable on Route — the driver payout save must reject it")

	var method string
	require.NoError(t, database.DB.Raw(
		`SELECT payout_method FROM delivery_partners WHERE id = ?`, driverID.String()).Row().Scan(&method))
	require.NotEqual(t, "upi", method, "the rejected UPI method must never be persisted")
}
