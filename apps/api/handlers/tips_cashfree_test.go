package handlers

// tips_cashfree_test.go — the tip must ride the gateway the order rode.
//
// D-18: the tip flow was written against the retired gateway's split product and never moved when
// payouts did, so it demanded a linked account that NO chef on the
// platform has. Every post-delivery tip answered 409 "This chef can't receive
// tips right now" — a whole surface, unreachable, promising "100% goes straight
// to your chef".

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// The tip's gateway order id is derived from the tip id, so a retry lands on the
// same Cashfree order rather than minting a parallel charge. Cashfree constrains
// the format, and an id it rejects would fail the charge outright.
func TestCashfreeTipOrderID_CarriesTheDispatchPrefix(t *testing.T) {
	id := "tip-" + strings.ReplaceAll("0f5a366c-c73e-e626-d6e7-14323a021c80", "-", "")
	require.True(t, strings.HasPrefix(id, "tip-"))
	// Cashfree order ids are 3–45 chars of alphanumerics, - and _.
	require.LessOrEqual(t, len(id), 45)
	require.NotContains(t, strings.TrimPrefix(id, "tip-"), "-")
}

// A tip carries no commission and no tax (INV-6), so unlike an order's Easy
// Split there is no platform fee to subtract: the chef's split is the WHOLE tip.
func TestCashfreeTipSplit_IsTheWholeTip(t *testing.T) {
	tipPaise := services.ToPaise(50.0)
	split := services.CashfreeVendorSplit{
		VendorID:    "hc_abc",
		AmountPaise: services.CashfreeAmountFromPaise(tipPaise),
	}
	require.Equal(t, 5000, tipPaise)
	require.Equal(t, tipPaise, split.AmountPaise.Paise(),
		"the platform must keep nothing — the screen promises 100% to the chef")
}

// --- #1080: rider tips on a chef-delivered order ---

func chefDeliveredOrder(vendorStatus string) *models.Order {
	o := &models.Order{FulfillmentType: models.FulfillmentChefDelivery}
	o.Chef = models.ChefProfile{
		UserID:               uuid.New(),
		CashfreeVendorID:     "hc_chef1",
		CashfreeVendorStatus: vendorStatus,
	}
	return o
}

// The defect: the chef IS the driver today, so a rider tip on a chef-delivered
// order has a valid destination and was refused anyway.
func TestPlanCashfreeTip_RiderTipOnChefDeliveredOrderGoesToTheChefsVendor(t *testing.T) {
	order := chefDeliveredOrder(services.CashfreeVendorActive)

	plan, status, msg := planCashfreeTip(order, createTipRequest{RiderAmount: 40})

	require.Equal(t, 0, status, "refused with %q", msg)
	require.Equal(t, "hc_chef1", plan.VendorID)
	require.Equal(t, 40.0, plan.RiderAmount)
	require.Zero(t, plan.ChefAmount)
	require.Equal(t, order.Chef.UserID, *plan.RiderUserID,
		"the chef carried the order, so the chef is the rider being tipped")
}

// The gate is the ORDER's fulfilment, not a global flag — the day a fleet
// exists, its rider's tip must not be silently paid to the kitchen.
func TestPlanCashfreeTip_RiderTipOnAThirdPartyDeliveryIsStillRefused(t *testing.T) {
	for _, ft := range []models.FulfillmentType{models.FulfillmentDelivery, models.FulfillmentPickup, ""} {
		order := chefDeliveredOrder(services.CashfreeVendorActive)
		order.FulfillmentType = ft

		_, status, msg := planCashfreeTip(order, createTipRequest{RiderAmount: 40})

		require.Equal(t, http.StatusConflict, status, "fulfilment %q", ft)
		require.Equal(t, "Rider tips aren't available on this payment method yet", msg)
	}
}

// Both legs settle to the one vendor, and the total charged is their sum.
func TestPlanCashfreeTip_ChefAndRiderLegsBothReachTheVendor(t *testing.T) {
	order := chefDeliveredOrder(services.CashfreeVendorActive)

	plan, status, msg := planCashfreeTip(order, createTipRequest{ChefAmount: 60, RiderAmount: 40})

	require.Equal(t, 0, status, "refused with %q", msg)
	require.Equal(t, 60.0, plan.ChefAmount)
	require.Equal(t, 40.0, plan.RiderAmount)
	require.Equal(t, 100.0, plan.Total(), "the whole tip is charged once")
	require.Equal(t, order.Chef.UserID, *plan.ChefUserID)
	require.Equal(t, order.Chef.UserID, *plan.RiderUserID)
}

// A rider tip refused for a dormant payout account must say so, rather than
// reusing the "not available on this payment method" wording — that is the
// misdiagnosis the original defect caused.
func TestPlanCashfreeTip_InactiveVendorIsRefusedForTheRealReason(t *testing.T) {
	order := chefDeliveredOrder("PENDING")

	_, status, msg := planCashfreeTip(order, createTipRequest{RiderAmount: 40})

	require.Equal(t, http.StatusConflict, status)
	require.Contains(t, msg, "payout account isn't active")
}

func TestPlanCashfreeTip_UnregisteredVendorIsRefused(t *testing.T) {
	order := chefDeliveredOrder(services.CashfreeVendorActive)
	order.Chef.CashfreeVendorID = ""

	_, status, msg := planCashfreeTip(order, createTipRequest{ChefAmount: 40})

	require.Equal(t, http.StatusConflict, status)
	require.Contains(t, msg, "payout account isn't active")
}

func TestPlanCashfreeTip_NothingToTip(t *testing.T) {
	order := chefDeliveredOrder(services.CashfreeVendorActive)

	_, status, msg := planCashfreeTip(order, createTipRequest{})

	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "Nothing to tip", msg)
}

// Vendor status is compared case-insensitively, as everywhere else Cashfree's
// status strings are read.
func TestPlanCashfreeTip_VendorStatusIsCaseInsensitive(t *testing.T) {
	order := chefDeliveredOrder(strings.ToLower(services.CashfreeVendorActive))

	_, status, msg := planCashfreeTip(order, createTipRequest{ChefAmount: 40})

	require.Equal(t, 0, status, "refused with %q", msg)
}

const chefTipsDDL = `CREATE TABLE tips (mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
	id TEXT PRIMARY KEY, order_id TEXT, customer_id TEXT, chef_user_id TEXT, rider_user_id TEXT,
	amount REAL DEFAULT 0, chef_amount REAL DEFAULT 0, rider_amount REAL DEFAULT 0,
	currency TEXT DEFAULT 'INR', status TEXT, gateway_order_id TEXT DEFAULT '',
	gateway_payment_id TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME)`

func setupChefTipsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(chefTipsDDL).Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func getChefTips(t *testing.T, userID uuid.UUID) []models.Tip {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/chef/tips", func(c *gin.Context) {
		c.Set("userID", userID)
		NewTipHandler().GetChefTips(c)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/chef/tips", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Data []models.Tip `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Data
}

// A chef who carried their own order and was tipped as the rider has been paid
// real money. If "tips received" only ever reads chef_amount, that money arrives
// with no record of where it came from.
func TestGetChefTips_IncludesATipReceivedAsTheRider(t *testing.T) {
	db := setupChefTipsDB(t)
	chefUserID := uuid.New()
	require.NoError(t, db.Create(&models.Tip{
		ID: uuid.New(), OrderID: uuid.New(), CustomerID: uuid.New(), Status: models.TipPaid,
		Amount: 40, RiderAmount: 40, RiderUserID: &chefUserID,
	}).Error)

	tips := getChefTips(t, chefUserID)

	require.Len(t, tips, 1)
	require.Equal(t, 40.0, tips[0].RiderAmount)
}

// The widened read must not leak: on a third-party delivery the rider is a
// different person, and their tip is not the chef's to see.
func TestGetChefTips_ExcludesAnotherPartysRiderTip(t *testing.T) {
	db := setupChefTipsDB(t)
	chefUserID := uuid.New()
	riderUserID := uuid.New()
	require.NoError(t, db.Create(&models.Tip{
		ID: uuid.New(), OrderID: uuid.New(), CustomerID: uuid.New(), Status: models.TipPaid,
		Amount: 40, RiderAmount: 40, RiderUserID: &riderUserID,
	}).Error)

	require.Empty(t, getChefTips(t, chefUserID))
}

// An unpaid tip is a charge the customer abandoned — it must not read as income.
func TestGetChefTips_ExcludesUnpaidTips(t *testing.T) {
	db := setupChefTipsDB(t)
	chefUserID := uuid.New()
	require.NoError(t, db.Create(&models.Tip{
		ID: uuid.New(), OrderID: uuid.New(), CustomerID: uuid.New(), Status: models.TipPending,
		Amount: 40, RiderAmount: 40, RiderUserID: &chefUserID,
	}).Error)

	require.Empty(t, getChefTips(t, chefUserID))
}

// Both legs of one chef-delivered tip are one row, listed once, not twice.
func TestGetChefTips_ListsABothLegTipOnce(t *testing.T) {
	db := setupChefTipsDB(t)
	chefUserID := uuid.New()
	require.NoError(t, db.Create(&models.Tip{
		ID: uuid.New(), OrderID: uuid.New(), CustomerID: uuid.New(), Status: models.TipPaid,
		Amount: 100, ChefAmount: 60, ChefUserID: &chefUserID, RiderAmount: 40, RiderUserID: &chefUserID,
	}).Error)

	tips := getChefTips(t, chefUserID)

	require.Len(t, tips, 1)
	require.Equal(t, 100.0, tips[0].Amount)
}
