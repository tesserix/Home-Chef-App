package handlers

// easy_split_capture_test.go — #1091 / ADR-0003. Checkout must no longer
// allocate the chef's share.
//
// THE BUG THIS PINS: order_splits on the create-order call committed the money
// before the release governor had run, so an order that later turned out to be
// refund-open, recovery-owing or from a chef still on the new-chef ramp had
// already been paid out. A split order had no payout to hold, so none of the
// blocks could reach it. The split now happens at release; checkout mints a
// plain full capture.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// cfCheckoutCapture points the live Cashfree slot at a stub and returns the
// bodies of every create-order call it receives.
func cfCheckoutCapture(t *testing.T) *[]map[string]any {
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

// splitEligibleChef is a chef every gate in BuildOrderSplit would pass: a
// verified Easy Split vendor, on Cashfree, with a payout destination.
func splitEligibleChef(t *testing.T, db *gorm.DB, chefID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(
		`UPDATE chef_profiles SET payment_provider = 'cashfree', payout_method = 'bank',
		   cashfree_vendor_id = ?, cashfree_vendor_status = ? WHERE id = ?`,
		services.EasySplitVendorIDFor(chefID), services.CashfreeVendorActive, chefID.String()).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO platform_settings (id, key, value, type, updated_at) VALUES (?, ?, 'true', 'bool', ?)`,
		uuid.NewString(), services.SettingEasySplitEnabled, time.Now()).Error)
}

func TestCreateOrderPayment_DoesNotSplitAtCapture(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	quoteFlagsOn(t)
	bodies := cfCheckoutCapture(t)
	cust := payUser(t, db, "customer")
	chefID := payChef(t, db, payUser(t, db, "chef"))
	splitEligibleChef(t, db, chefID)
	orderID := payOrder(t, db, cust, chefID, "pending", 1000, "", "")

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create", regCreate, map[string]any{})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Len(t, *bodies, 1)
	require.NotContains(t, (*bodies)[0], "order_splits",
		"the chef's share is allocated at release, after the governor has run")

	// And nothing may claim the share was paid — that stamp is what excludes the
	// order from the weekly statement.
	require.Zero(t, orderIntColumn(t, db, orderID, "gateway_split_paise"))
}

func orderIntColumn(t *testing.T, db *gorm.DB, id uuid.UUID, col string) int {
	t.Helper()
	var v int
	require.NoError(t, db.Raw(`SELECT COALESCE(`+col+`, 0) FROM orders WHERE id = ?`, id.String()).Scan(&v).Error)
	return v
}
