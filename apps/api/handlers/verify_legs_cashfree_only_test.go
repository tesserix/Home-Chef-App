package handlers

// verify_legs_cashfree_only_test.go — #1086 Phase 5. The three non-order charges
// are confirmed by asking Cashfree what was captured, never by trusting a
// client-supplied payment id.
//
// The charge legs went Cashfree-only in #1102, so every one of these rows is now
// stamped cashfree — but the verify endpoints still demanded a
// gatewayPaymentId as a REQUIRED field, which no client can supply for a
// Cashfree charge. The group-share leg was worse: its Cashfree branch confirmed
// the capture and then never marked the participant paid, so the group could
// never consolidate.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// verifyStub answers GET /orders/:id/payments with the given attempts.
func verifyStub(t *testing.T, payments string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/payments") {
			_, _ = w.Write([]byte(payments))
			return
		}
		_, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	services.SetCashfreeClient(
		services.NewCashfreeTestClient(srv.URL, "app_test", "secret_test", cfTestWebhookSecret, models.ChefModeLive))
	t.Cleanup(func() { services.SetCashfreeClient(nil) })
}

func capture(orderID string, rupees float64) string {
	b, _ := json.Marshal([]map[string]any{{
		"cf_payment_id": 4242, "order_id": orderID, "payment_status": "SUCCESS", "payment_amount": rupees,
	}})
	return string(b)
}

// ── group-order share ────────────────────────────────────────────────────────

// A share whose charge is captured must be marked paid. Its Cashfree branch
// verified the capture and then returned without recording it, so the group sat
// unpaid forever with the customer's money taken.
func TestVerifyGroupShare_CapturedShareIsMarkedPaid(t *testing.T) {
	db := setupChargeDB(t)
	_, chefID := chargeChef(t, db, models.PaymentProviderCashfree)
	payer := payUser(t, db, "customer")
	groupID, partID := seedGroupShare(t, db, chefID, payer)
	stampShare(t, db, partID, "grp-abc")
	// A second unpaid guest keeps the group short of consolidation, so this test
	// stays about recording the share.
	seedUnpaidGuest(t, db, groupID)
	verifyStub(t, capture("grp-abc", 250))

	w := callAs(payer, http.MethodPost, "/group-orders/"+groupID.String()+"/pay/verify", func(r *gin.Engine) {
		r.POST("/group-orders/:id/pay/verify", NewGroupOrderHandler().VerifyGroupShare)
	}, map[string]any{})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, string(models.GroupPayCompleted), shareStatusOf(t, db, partID))
}

// The amount binding (#395·4) holds on the Cashfree rail: an under-amount
// capture must not settle a share in full.
func TestVerifyGroupShare_UnderAmountCaptureDoesNotSettleTheShare(t *testing.T) {
	db := setupChargeDB(t)
	_, chefID := chargeChef(t, db, models.PaymentProviderCashfree)
	payer := payUser(t, db, "customer")
	groupID, partID := seedGroupShare(t, db, chefID, payer)
	stampShare(t, db, partID, "grp-abc")
	verifyStub(t, capture("grp-abc", 50))

	w := callAs(payer, http.MethodPost, "/group-orders/"+groupID.String()+"/pay/verify", func(r *gin.Engine) {
		r.POST("/group-orders/:id/pay/verify", NewGroupOrderHandler().VerifyGroupShare)
	}, map[string]any{})

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Equal(t, "pending", shareStatusOf(t, db, partID))
}

func stampShare(t *testing.T, db *gorm.DB, partID uuid.UUID, gatewayOrderID string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`UPDATE group_order_participants SET gateway_order_id = ?, payment_provider = ? WHERE id = ?`,
		gatewayOrderID, models.PaymentProviderCashfree, partID.String()).Error)
}

func seedUnpaidGuest(t *testing.T, db *gorm.DB, groupID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO group_order_participants
		(id, group_order_id, user_id, role, share_amount, payment_status, joined_at, updated_at)
		VALUES (?, ?, ?, 'guest', 250, 'pending', ?, ?)`,
		uuid.NewString(), groupID.String(), uuid.NewString(), time.Now(), time.Now()).Error)
}

func shareStatusOf(t *testing.T, db *gorm.DB, partID uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payment_status FROM group_order_participants WHERE id = ?`,
		partID.String()).Scan(&s).Error)
	return s
}

// ── catering deposit ─────────────────────────────────────────────────────────

func TestVerifyCateringDeposit_CapturedDepositConfirmsTheBooking(t *testing.T) {
	db := setupChargeDB(t)
	cateringDepositsOn(t)
	_, chefID := chargeChef(t, db, models.PaymentProviderCashfree)
	customerID := payUser(t, db, "customer")
	reqID := seedCateringDeposit(t, db, customerID, chefID)
	require.NoError(t, db.Exec(
		`UPDATE catering_requests SET gateway_order_id = ?, payment_provider = ? WHERE id = ?`,
		"cat-abc", models.PaymentProviderCashfree, reqID.String()).Error)
	verifyStub(t, capture("cat-abc", 500))

	w := callAs(customerID, http.MethodPost, "/catering/requests/"+reqID.String()+"/deposit/verify",
		func(r *gin.Engine) {
			r.POST("/catering/requests/:id/deposit/verify", NewCateringHandler().VerifyDeposit)
		}, map[string]any{})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var status string
	require.NoError(t, db.Raw(`SELECT deposit_status FROM catering_requests WHERE id = ?`,
		reqID.String()).Scan(&status).Error)
	require.Equal(t, "paid", status)
}

func TestVerifyCateringDeposit_UncapturedDepositIsRefused(t *testing.T) {
	db := setupChargeDB(t)
	cateringDepositsOn(t)
	_, chefID := chargeChef(t, db, models.PaymentProviderCashfree)
	customerID := payUser(t, db, "customer")
	reqID := seedCateringDeposit(t, db, customerID, chefID)
	require.NoError(t, db.Exec(
		`UPDATE catering_requests SET gateway_order_id = ?, payment_provider = ? WHERE id = ?`,
		"cat-abc", models.PaymentProviderCashfree, reqID.String()).Error)
	verifyStub(t, `[]`)

	w := callAs(customerID, http.MethodPost, "/catering/requests/"+reqID.String()+"/deposit/verify",
		func(r *gin.Engine) {
			r.POST("/catering/requests/:id/deposit/verify", NewCateringHandler().VerifyDeposit)
		}, map[string]any{})

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	var status string
	require.NoError(t, db.Raw(`SELECT deposit_status FROM catering_requests WHERE id = ?`,
		reqID.String()).Scan(&status).Error)
	require.NotEqual(t, "paid", status)
}

// ── featured listing ─────────────────────────────────────────────────────────

func TestConfirmFeaturedAd_CapturedPurchaseActivatesTheListing(t *testing.T) {
	db := setupChargeDB(t)
	userID, chefID := chargeChef(t, db, models.PaymentProviderCashfree)
	promoID := seedPendingPromotion(t, db, chefID, "promo-abc", 999)
	verifyStub(t, capture("promo-abc", 999))

	w := callAs(userID, http.MethodPost, "/chef/promotion/confirm", func(r *gin.Engine) {
		r.POST("/chef/promotion/confirm", NewPromotionHandler().ConfirmFeaturedAd)
	}, map[string]any{"promotionId": promoID.String()})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var status string
	require.NoError(t, db.Raw(`SELECT status FROM chef_promotions WHERE id = ?`,
		promoID.String()).Scan(&status).Error)
	require.Equal(t, string(models.PromotionActive), status)
}

func TestConfirmFeaturedAd_UncapturedPurchaseStaysPending(t *testing.T) {
	db := setupChargeDB(t)
	userID, chefID := chargeChef(t, db, models.PaymentProviderCashfree)
	promoID := seedPendingPromotion(t, db, chefID, "promo-abc", 999)
	verifyStub(t, `[]`)

	w := callAs(userID, http.MethodPost, "/chef/promotion/confirm", func(r *gin.Engine) {
		r.POST("/chef/promotion/confirm", NewPromotionHandler().ConfirmFeaturedAd)
	}, map[string]any{"promotionId": promoID.String()})

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	var status string
	require.NoError(t, db.Raw(`SELECT status FROM chef_promotions WHERE id = ?`,
		promoID.String()).Scan(&status).Error)
	require.Equal(t, string(models.PromotionPending), status)
}

func seedPendingPromotion(t *testing.T, db *gorm.DB, chefID uuid.UUID, gatewayOrderID string, amount float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_promotions
		(id, chef_id, status, amount, currency, duration, gateway_order_id, payment_provider, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'INR', 30, ?, ?, ?, ?)`,
		id.String(), chefID.String(), models.PromotionPending, amount, gatewayOrderID,
		models.PaymentProviderCashfree, time.Now(), time.Now()).Error)
	return id
}
