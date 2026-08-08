package services

// easy_split_release_test.go — #1091 / ADR-0003. The split now happens when the
// release governor says so, inside Cashfree's split-delay window, instead of at
// capture where no control could reach it. Every case here is about which of
// the two rails an order ends up on, because ending up on both pays the chef
// twice and on neither strands them.

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

const releaseOrderDDL = `CREATE TABLE orders (
	id TEXT PRIMARY KEY, order_number TEXT, chef_id TEXT, mode TEXT DEFAULT 'live',
	payment_provider TEXT DEFAULT '', gateway_order_id TEXT DEFAULT '',
	subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, chef_tip REAL DEFAULT 0,
	delivery_fee REAL DEFAULT 0, commission_rate REAL DEFAULT 0, total REAL DEFAULT 0,
	wallet_applied REAL DEFAULT 0, loyalty_applied REAL DEFAULT 0,
	gateway_split_paise INTEGER DEFAULT 0,
	created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`

const releaseChefDDL = `CREATE TABLE chef_profiles (
	id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '', mode TEXT DEFAULT 'live',
	payout_method TEXT DEFAULT '', payout_country TEXT DEFAULT 'IN',
	easy_split_mode TEXT DEFAULT '',
	cashfree_vendor_id TEXT DEFAULT '', cashfree_vendor_status TEXT DEFAULT '',
	cashfree_test_vendor_id TEXT DEFAULT '', cashfree_test_vendor_status TEXT DEFAULT '',
	created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`

func setupReleaseSplitDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupEasySplitDB(t)
	require.NoError(t, db.Exec(releaseOrderDDL).Error)
	require.NoError(t, db.Exec(releaseChefDDL).Error)
	setEasySplitSetting(t, db, SettingEasySplitEnabled, "true")
	return db
}

// seedReleaseOrder writes a paid, Cashfree-captured order and its chef.
func seedReleaseOrder(t *testing.T, db *gorm.DB, paidAt time.Time) *models.Order {
	t.Helper()
	order := easySplitOrder()
	order.ID = uuid.New()
	order.OrderNumber = "HC-" + order.ID.String()[:8]
	order.ChefID = order.Chef.ID
	order.Mode = models.ChefModeLive
	order.PaymentProvider = models.PaymentProviderCashfree
	order.GatewayOrderID = "cf-" + order.ID.String()[:8]

	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, payout_country, cashfree_vendor_id, cashfree_vendor_status)
		 VALUES (?,?,?,?,?)`,
		order.Chef.ID.String(), uuid.New().String(), "IN",
		order.Chef.CashfreeVendorID, order.Chef.CashfreeVendorStatus).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, chef_id, mode, payment_provider, gateway_order_id,
		  subtotal, tax, chef_tip, delivery_fee, commission_rate, total, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		order.ID.String(), order.OrderNumber, order.ChefID.String(), order.Mode,
		order.PaymentProvider, order.GatewayOrderID,
		order.Subtotal, order.Tax, order.ChefTip, order.DeliveryFee, order.CommissionRate,
		order.Total, paidAt, paidAt).Error)
	return order
}

// withSplitGateway points the live-mode Cashfree client at a stub.
func withSplitGateway(t *testing.T, handler http.HandlerFunc) *int {
	t.Helper()
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	SetCashfreeClient(NewCashfreeTestClient(srv.URL, "TESTapp", "cf_test_secret", "wh", models.ChefModeLive))
	t.Cleanup(func() { SetCashfreeClient(nil) })
	return &calls
}

func splitPaiseOf(t *testing.T, db *gorm.DB, id uuid.UUID) int {
	t.Helper()
	var got int
	require.NoError(t, db.Raw(`SELECT gateway_split_paise FROM orders WHERE id = ?`, id.String()).Scan(&got).Error)
	return got
}

func TestReleaseOrderSplit_SplitsWhenTheGovernorHasReleasedTheOrder(t *testing.T) {
	db := setupReleaseSplitDB(t)
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	calls := withSplitGateway(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/easy-split/orders/"+order.GatewayOrderID+"/split", r.URL.Path)
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})

	split, err := ReleaseOrderSplit(db, order.ID, time.Now())

	require.NoError(t, err)
	require.True(t, split)
	require.Equal(t, 1, *calls)
	// The stamp is what keeps the statement off this order — it is the same
	// exclusion split-at-capture used.
	require.Positive(t, splitPaiseOf(t, db, order.ID))
}

// Past the split delay Cashfree has already settled the whole amount to the
// platform. That is the documented fail-safe: pay the chef through the payout
// rail instead, and never call an API that can only refuse.
func TestReleaseOrderSplit_FallsBackToTheOtherRailOnceTheWindowHasLapsed(t *testing.T) {
	db := setupReleaseSplitDB(t)
	order := seedReleaseOrder(t, db, time.Now().Add(-30*time.Hour))
	calls := withSplitGateway(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) })

	split, err := ReleaseOrderSplit(db, order.ID, time.Now())

	require.NoError(t, err)
	require.False(t, split)
	require.Zero(t, *calls)
	require.Zero(t, splitPaiseOf(t, db, order.ID))
}

// Cashfree cannot split a payment it has not finished syncing. Our maturation
// window makes that near-impossible, but if it happens the order must stay
// unsettled and be re-driven — not fall through and be paid twice.
func TestReleaseOrderSplit_ReturnsTheErrorWhenTheSplitIsMerelyEarly(t *testing.T) {
	db := setupReleaseSplitDB(t)
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	withSplitGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"Transaction is not synced, please retry after 2 mins"}`))
	})

	split, err := ReleaseOrderSplit(db, order.ID, time.Now())

	require.Error(t, err)
	require.False(t, split)
	require.Zero(t, splitPaiseOf(t, db, order.ID), "nothing may be stamped for a split that did not happen")
}

// A refusal that will never change — an unregistered vendor — is not worth
// stranding the chef over. The window lapses and the payout rail pays them.
func TestReleaseOrderSplit_FallsBackOnARefusalCashfreeWillRepeat(t *testing.T) {
	db := setupReleaseSplitDB(t)
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	withSplitGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"vendor does not exist"}`))
	})

	split, err := ReleaseOrderSplit(db, order.ID, time.Now())

	require.NoError(t, err)
	require.False(t, split)
	require.Zero(t, splitPaiseOf(t, db, order.ID))
}

// A re-drive of an order already split must not split it again.
func TestReleaseOrderSplit_IsIdempotentOnAnAlreadySplitOrder(t *testing.T) {
	db := setupReleaseSplitDB(t)
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	require.NoError(t, db.Exec(`UPDATE orders SET gateway_split_paise = 38000 WHERE id = ?`, order.ID.String()).Error)
	calls := withSplitGateway(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) })

	split, err := ReleaseOrderSplit(db, order.ID, time.Now())

	require.NoError(t, err)
	require.True(t, split, "the chef has already been paid at the gateway")
	require.Zero(t, *calls)
}

// With the flag off nothing about the payout path may change.
func TestReleaseOrderSplit_IsInertWhileTheFlagIsOff(t *testing.T) {
	db := setupReleaseSplitDB(t)
	setEasySplitSetting(t, db, SettingEasySplitEnabled, "false")
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	calls := withSplitGateway(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) })

	split, err := ReleaseOrderSplit(db, order.ID, time.Now())

	require.NoError(t, err)
	require.False(t, split)
	require.Zero(t, *calls)
}

// An order we cannot read is not a reason to block its release — the chef is
// paid through the payout rail instead, which is where every refusal here ends.
func TestReleaseOrderSplit_FallsBackWhenTheOrderCannotBeRead(t *testing.T) {
	db := setupReleaseSplitDB(t)

	split, err := ReleaseOrderSplit(db, uuid.New(), time.Now())

	require.NoError(t, err)
	require.False(t, split)
}

// ADR-0003, Consequences: the maturation window has to fit inside the split
// delay, or every order silently degrades to the payout rail. The two values
// are coupled, so the coupling is asserted in code rather than remembered.
func TestEasySplitWindowFits_RefusesAMaturationLongerThanTheDelay(t *testing.T) {
	db := setupReleaseSplitDB(t)
	require.True(t, EasySplitWindowFits(db), "2h maturation inside a T+1 delay")

	setEasySplitSetting(t, db, "payout.maturation_minutes", "2880") // 48h
	require.False(t, EasySplitWindowFits(db))

	// An operator who has had the delay extended can raise it to match.
	setEasySplitSetting(t, db, SettingEasySplitDelayHours, "72")
	require.True(t, EasySplitWindowFits(db))
}

// The governor is the only caller that matters: releasing an order's money has
// to be what triggers the split, or the rail is written and never used.
func TestReleaseMoney_SplitsTheOrderItReleases(t *testing.T) {
	db := setupReleaseSplitDB(t)
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	calls := withSplitGateway(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/easy-split/orders/"+order.GatewayOrderID+"/split", r.URL.Path)
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})

	require.NoError(t, releaseMoney(db, aggTypeOrder, order.ID))

	require.Equal(t, 1, *calls)
	require.Positive(t, splitPaiseOf(t, db, order.ID))
}

// A split that is merely early must fail the release rather than complete it:
// completing would stamp the hold settled and nothing would ever re-drive the
// split, leaving the chef to be paid a week later on the statement instead.
func TestReleaseMoney_FailsTheReleaseWhenTheSplitIsEarly(t *testing.T) {
	db := setupReleaseSplitDB(t)
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	withSplitGateway(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"Transaction is not synced, please retry after 2 mins"}`))
	})

	require.Error(t, releaseMoney(db, aggTypeOrder, order.ID))
	require.Zero(t, splitPaiseOf(t, db, order.ID))
}

func TestReleaseOrderSplit_RefusesWhenTheMaturationOutlivesTheDelay(t *testing.T) {
	db := setupReleaseSplitDB(t)
	setEasySplitSetting(t, db, "payout.maturation_minutes", "2880")
	order := seedReleaseOrder(t, db, time.Now().Add(-2*time.Hour))
	calls := withSplitGateway(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) })

	split, err := ReleaseOrderSplit(db, order.ID, time.Now())

	require.NoError(t, err)
	require.False(t, split)
	require.Zero(t, *calls)
}
