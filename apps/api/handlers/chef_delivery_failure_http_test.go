package handlers

// chef_delivery_failure_http_test.go — #393 chef parity. HTTP-level guards on
// ReportChefDeliveryFailure: chef_id scoping (a chef can't act on another chef's order),
// the fulfillment/status gate, and reason validation. Reuses setupChefVisDB/seedVisChef.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func chefFailureRouter(userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID) })
	r.POST("/chef/orders/:orderId/delivery-failed", NewChefHandler().ReportChefDeliveryFailure)
	return r
}

func seedChefFailureOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, status, ft string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, fulfillment_type, total)
		VALUES (?,?,?,?,?,?,100)`,
		id.String(), "ORD-"+id.String()[:6], uuid.NewString(), chefID.String(), status, ft).Error)
	return id
}

func postChefFailure(t *testing.T, r *gin.Engine, orderID, reason string) int {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chef/orders/"+orderID+"/delivery-failed",
		strings.NewReader(`{"failureReason":"`+reason+`"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w.Code
}

func TestReportChefDeliveryFailure_Guards(t *testing.T) {
	db := setupChefVisDB(t)
	chefAUser, chefA := seedVisChef(t, db)
	_, chefB := seedVisChef(t, db)

	orderB := seedChefFailureOrder(t, db, chefB, "delivering", "chef_delivery") // another chef's order
	pickup := seedChefFailureOrder(t, db, chefA, "delivering", "pickup")        // not self-delivery
	notOut := seedChefFailureOrder(t, db, chefA, "preparing", "chef_delivery")  // not out for delivery
	okOrder := seedChefFailureOrder(t, db, chefA, "delivering", "chef_delivery")

	r := chefFailureRouter(chefAUser)

	// Cross-chef abuse: chef A cannot report a failure on chef B's order.
	require.Equal(t, http.StatusNotFound, postChefFailure(t, r, orderB.String(), "customer_unavailable"))
	// Wrong fulfillment (pickup is customer-collected; 3PL uses the courier pipeline).
	require.Equal(t, http.StatusUnprocessableEntity, postChefFailure(t, r, pickup.String(), "customer_unavailable"))
	// Not out for delivery yet.
	require.Equal(t, http.StatusUnprocessableEntity, postChefFailure(t, r, notOut.String(), "customer_unavailable"))
	// Invalid reason.
	require.Equal(t, http.StatusBadRequest, postChefFailure(t, r, okOrder.String(), "banana"))
	// Unknown order id.
	require.Equal(t, http.StatusNotFound, postChefFailure(t, r, uuid.NewString(), "customer_unavailable"))
}
