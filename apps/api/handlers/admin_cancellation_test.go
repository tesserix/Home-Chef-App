package handlers

// admin_cancellation_test.go — admin arbitration (#480). A timed-out request
// (never refunded) gets its refund issued at the admin's tier; a disputed request
// (already refunded) is topped up to the more-generous tier. Reuses the harness
// from cancellation_test.go (same package).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func adminPost(t *testing.T, adminID uuid.UUID, path string, body any) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", adminID) })
	h := NewCancellationHandler()
	r.POST("/admin/cancel-requests/:id/resolve", h.ResolveCancellationAdmin)
	r.GET("/admin/cancel-requests", h.GetAdminCancellationRequests)
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// admin_review (timeout, never refunded) → admin picks materials (40%) → the
// refund is executed for the first time; request resolved.
func TestAdminResolve_TimeoutExecutesRefund(t *testing.T) {
	db, custID, adminUserID, chefID := setupCancelDB(t)
	oid := seedPaidOrder(t, db, custID, chefID, "preparing")
	reqID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, refund_destination, refund_total_paise, refund_executed)
		VALUES (?,?,?,?,?,?,0,0)`, reqID.String(), oid.String(), custID.String(), chefID.String(), "admin_review", "wallet").Error)

	w := adminPost(t, adminUserID, "/admin/cancel-requests/"+reqID.String()+"/resolve", map[string]any{"reason": "materials_purchased", "note": "checked with chef"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var cr models.CancellationRequest
	require.NoError(t, db.First(&cr, "id = ?", reqID.String()).Error)
	require.Equal(t, models.CancelReqResolved, cr.Status)
	require.True(t, cr.RefundExecuted)
	require.Equal(t, 25200, cr.RefundTotalPaise) // ₹252 at the 40% tier
	require.InDelta(t, 252.0, walletBalance(t, db, custID), 0.001)
}

// disputed (already refunded ₹252 at 40%) → admin bumps to not_started (90%) →
// tops up the difference to the wallet; request resolved at the new snapshot.
func TestAdminResolve_DisputeTopsUpDifference(t *testing.T) {
	db, custID, adminUserID, chefID := setupCancelDB(t)
	oid := seedPaidOrder(t, db, custID, chefID, "preparing")
	reqID := uuid.New()
	// Simulate the already-issued 40% refund (₹252).
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, refund_destination, refund_total_paise, vendor_kept_paise, platform_kept_paise, refund_executed)
		VALUES (?,?,?,?,?,?,25200,30000,7800,1)`, reqID.String(), oid.String(), custID.String(), chefID.String(), "disputed", "wallet").Error)

	w := adminPost(t, adminUserID, "/admin/cancel-requests/"+reqID.String()+"/resolve", map[string]any{"reason": "not_started"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var cr models.CancellationRequest
	require.NoError(t, db.First(&cr, "id = ?", reqID.String()).Error)
	require.Equal(t, models.CancelReqResolved, cr.Status)
	require.Equal(t, 51450, cr.RefundTotalPaise) // now ₹514.50 at the 90% tier
	require.Equal(t, 5000, cr.VendorKeptPaise)   // vendor keeps 10% of food = ₹50
	require.Equal(t, 63000, cr.RefundTotalPaise+cr.VendorKeptPaise+cr.PlatformKeptPaise, "money conserved")
	// Only the DELTA (₹514.50 − ₹252 = ₹262.50) is credited now, not the whole refund again.
	require.InDelta(t, 262.50, walletBalance(t, db, custID), 0.001, "only the top-up is credited")
}

// #642 Finding-1: with the prior ₹252 refund REFLECTED in order.refund_amount (as
// ExecuteCancellationRefund really leaves it), the disputed re-snapshot must cap at
// Total − OTHER refunds (RemainingRefundable + this cancellation's own refund), NOT at the bare
// RemainingRefundable — otherwise it double-subtracts the ₹252 and silently shortchanges the
// admin's 90% ruling (₹126 top-up instead of ₹262.50).
func TestAdminResolve_Dispute_HonoursRulingWhenPriorRefundReflected(t *testing.T) {
	db, custID, adminUserID, chefID := setupCancelDB(t)
	oid := seedPaidOrder(t, db, custID, chefID, "preparing")
	// The prior 40% cancellation refund (₹252) is reflected on the order, as it would be in prod.
	require.NoError(t, db.Exec(`UPDATE orders SET refund_amount = 252 WHERE id = ?`, oid.String()).Error)
	reqID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, refund_destination, refund_total_paise, vendor_kept_paise, platform_kept_paise, refund_executed)
		VALUES (?,?,?,?,?,?,25200,30000,7800,1)`, reqID.String(), oid.String(), custID.String(), chefID.String(), "disputed", "wallet").Error)

	w := adminPost(t, adminUserID, "/admin/cancel-requests/"+reqID.String()+"/resolve", map[string]any{"reason": "not_started"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var cr models.CancellationRequest
	require.NoError(t, db.First(&cr, "id = ?", reqID.String()).Error)
	require.Equal(t, 51450, cr.RefundTotalPaise, "#642: the admin's full 90% ruling is honoured (not capped low by double-subtracting the prior refund)")
	require.InDelta(t, 262.50, walletBalance(t, db, custID), 0.001, "the full top-up is credited, not the shortchanged ₹126")
}
