package handlers

// delivery_quote_test.go — the checkout quote endpoint. Pins the JSON contract
// the customer app reads (deliveryFee / pickupFee / pickupSaving) and that pickup
// is always free.

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

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

func setupQuoteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (mode text DEFAULT 'live', first_live_at datetime, active_test_session_id text, address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', 
		id text PRIMARY KEY, user_id text, business_name text, is_active integer DEFAULT 1,
		offers_pickup integer DEFAULT 0, offers_self_delivery integer DEFAULT 0,
		self_delivery_tiers text DEFAULT '[]',
		self_delivery_base_fee real DEFAULT 0, self_delivery_free_radius_km real DEFAULT 0,
		self_delivery_per_km real DEFAULT 0, self_delivery_max_fee real DEFAULT 0,
		self_delivery_max_distance_km real DEFAULT 0,
		latitude real DEFAULT 0, longitude real DEFAULT 0, payout_country text DEFAULT 'IN')`).Error)
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func quote(t *testing.T, chefID uuid.UUID, body string) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/chefs/:id/delivery-quote", (&OrderHandler{}).QuoteDeliveryFee)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chefs/"+chefID.String()+"/delivery-quote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}

// Delivery costs the flat policy fee (no coords → fallback), pickup is free, and
// the saving equals the delivery fee — the exact numbers the incentive shows.
func TestQuoteEndpoint_ReturnsDeliveryFeeAndPickupSaving(t *testing.T) {
	db := setupQuoteDB(t)
	// Force a known policy fee via a seeded platform_settings-free override: the
	// service reads GetPlatformPolicy, whose default BaseDeliveryFee is 2.99.
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, offers_pickup) VALUES (?,?,1)`,
		chefID.String(), uuid.NewString()).Error)

	out := quote(t, chefID, `{"city":"Pune","country":"IN"}`)

	require.Equal(t, services.GetPlatformPolicy().BaseDeliveryFee, out["deliveryFee"],
		"delivery falls back to the flat policy fee without coords — the same as CreateOrder")
	require.Equal(t, 0.0, out["pickupFee"], "pickup is always free")
	require.Equal(t, out["deliveryFee"], out["pickupSaving"], "the saving is the whole delivery fee")
	require.Equal(t, true, out["offersPickup"])
}

// A self-delivering chef gets an itemised, capped estimate (#702): the response
// carries selfDeliveryFee (the approx max ceiling) and a breakdown whose
// components the app renders. A plain-delivery chef gets neither.
func TestQuoteEndpoint_SelfDeliveryEstimate(t *testing.T) {
	db := setupQuoteDB(t)
	chefID := uuid.New()
	// Chef in central Bangalore; base 20, free 2 km, ₹10/km, capped ₹200.
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles
		(id, user_id, offers_self_delivery, self_delivery_base_fee, self_delivery_free_radius_km,
		 self_delivery_per_km, self_delivery_max_fee, latitude, longitude)
		VALUES (?,?,1,20,2,10,200,12.9716,77.5946)`,
		chefID.String(), uuid.NewString()).Error)

	// Drop ~5 km away → a real distance component beyond the free radius.
	out := quote(t, chefID, `{"latitude":12.9352,"longitude":77.6245,"city":"Bengaluru","country":"IN"}`)

	require.Equal(t, true, out["offersSelfDelivery"])
	fee, ok := out["selfDeliveryFee"].(float64)
	require.True(t, ok, "selfDeliveryFee present for a self-delivering chef")
	require.Greater(t, fee, 20.0, "base + distance beyond the free radius")
	require.LessOrEqual(t, fee, 200.0, "never above the chef's cap — the approx max")

	bd, ok := out["selfDeliveryBreakdown"].(map[string]any)
	require.True(t, ok, "breakdown present")
	require.Equal(t, 20.0, bd["baseFee"])
	require.Equal(t, true, bd["distanceKnown"])
	require.Greater(t, bd["distanceComponent"].(float64), 0.0)
}

// A chef who does NOT self-deliver must not get a self-delivery estimate — the
// order would go 3PL, so quoting one would mislead.
func TestQuoteEndpoint_NoSelfDeliveryEstimateWhenNotOffered(t *testing.T) {
	db := setupQuoteDB(t)
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, offers_self_delivery) VALUES (?,?,0)`,
		chefID.String(), uuid.NewString()).Error)

	out := quote(t, chefID, `{"latitude":12.9352,"longitude":77.6245}`)
	require.Equal(t, false, out["offersSelfDelivery"])
	_, has := out["selfDeliveryFee"]
	require.False(t, has, "no self-delivery estimate when the chef doesn't offer it")
}

// creditRailsOff pins both credit rails closed. The tip assertions below are
// about the TOTAL, not about allocation, so keeping the rails shut means the
// quote never reaches for wallet/loyalty tables this in-memory DB doesn't have.
func creditRailsOff(t *testing.T) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig = &config.Config{Environment: "test"}
	t.Cleanup(func() { config.AppConfig = prev })
}

// quoteAs runs the endpoint with a signed-in viewer, so the response carries the
// `credit` block (the anonymous helper above never does).
func quoteAs(t *testing.T, chefID, userID uuid.UUID, body string) map[string]any {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/chefs/:id/delivery-quote", func(c *gin.Context) {
		c.Set("userID", userID)
	}, (&OrderHandler{}).QuoteDeliveryFee)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chefs/"+chefID.String()+"/delivery-quote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}

// A tip lands in the previewed payable pound-for-pound, and nowhere else: it is
// not redeemable against credit and it does not move the fee or tax lines. Before
// the tip was sent, checkout previewed a payable short by the tip and the gateway
// then asked for more than the screen had shown.
func TestQuoteEndpoint_TipRaisesPayableByExactlyTheTip(t *testing.T) {
	db := setupQuoteDB(t)
	creditRailsOff(t)
	chefID, userID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id) VALUES (?,?)`,
		chefID.String(), uuid.NewString()).Error)

	base := quoteAs(t, chefID, userID, `{"subtotal":270,"city":"Pune","country":"IN"}`)
	tipped := quoteAs(t, chefID, userID, `{"subtotal":270,"city":"Pune","country":"IN","tip":25}`)

	baseCredit, ok := base["credit"].(map[string]any)
	require.True(t, ok, "a signed-in viewer gets a credit block")
	tippedCredit, ok := tipped["credit"].(map[string]any)
	require.True(t, ok)

	require.InDelta(t, baseCredit["payable"].(float64)+25, tippedCredit["payable"].(float64), 0.001,
		"the tip is added to what the customer pays, in full")
	require.Equal(t, baseCredit["redeemableCap"], tippedCredit["redeemableCap"],
		"a tip is never redeemable against wallet or points")
	require.Equal(t, base["platformFee"], tipped["platformFee"], "a tip is not fee-bearing")
}

// A negative tip is a client bug, not a discount — it must never shrink the bill.
func TestQuoteEndpoint_NegativeTipIsIgnored(t *testing.T) {
	db := setupQuoteDB(t)
	creditRailsOff(t)
	chefID, userID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id) VALUES (?,?)`,
		chefID.String(), uuid.NewString()).Error)

	base := quoteAs(t, chefID, userID, `{"subtotal":270,"country":"IN"}`)
	negative := quoteAs(t, chefID, userID, `{"subtotal":270,"country":"IN","tip":-100}`)

	require.Equal(t,
		base["credit"].(map[string]any)["payable"],
		negative["credit"].(map[string]any)["payable"])
}

func TestQuoteEndpoint_UnknownChef_Is404(t *testing.T) {
	setupQuoteDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/chefs/:id/delivery-quote", (&OrderHandler{}).QuoteDeliveryFee)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/chefs/"+uuid.NewString()+"/delivery-quote", strings.NewReader(`{}`)))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestQuoteEndpoint_BadChefID_Is400(t *testing.T) {
	setupQuoteDB(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/chefs/:id/delivery-quote", (&OrderHandler{}).QuoteDeliveryFee)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/chefs/not-a-uuid/delivery-quote", strings.NewReader(`{}`)))
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQuoteEndpointUsesSameMarketGatewayAsPayment(t *testing.T) {
	for _, tc := range []struct{ country, currency, configured, provider string }{
		{"IN", "inr", "stripe", "cashfree"},
		{"AU", "aud", "cashfree", "stripe"},
		{"NZ", "nzd", "cashfree", "stripe"},
	} {
		t.Run(tc.country, func(t *testing.T) {
			db := setupQuoteDB(t)
			creditRailsOff(t)
			require.NoError(t, db.Exec("ALTER TABLE chef_profiles ADD COLUMN payment_provider text").Error)
			chefID := uuid.New()
			require.NoError(t, db.Exec("INSERT INTO chef_profiles (id, user_id, payout_country, payment_provider) VALUES (?,?,?,?)", chefID, uuid.New(), tc.country, tc.configured).Error)
			out := quoteAs(t, chefID, uuid.New(), `{"subtotal":50,"fulfillmentType":"pickup"}`)
			require.Equal(t, tc.provider, out["paymentProvider"])
			require.Equal(t, tc.currency, out["currency"])
			if tc.provider == "stripe" {
				require.NotContains(t, out, "credit", "Stripe must not preview unsupported wallet/loyalty discounts")
			}
		})
	}
}

func TestQuoteEndpoint_KitchenTaxCountry(t *testing.T) {
	for _, country := range []string{"AU", "NZ"} {
		t.Run(country, func(t *testing.T) {
			db := setupQuoteDB(t)
			services.InvalidateTaxCache()
			t.Cleanup(services.InvalidateTaxCache)
			rate := 10.0
			if country == "NZ" {
				rate = 15
			}
			require.NoError(t, db.Exec(`CREATE TABLE tax_rates (country_code text, region text, tax_name text, rate real, is_active boolean, inclusive boolean)`).Error)
			require.NoError(t, db.Exec(`INSERT INTO tax_rates VALUES (?, '', 'GST', ?, true, true)`, country, rate).Error)
			chefID := uuid.New()
			require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, payout_country) VALUES (?,?,?)`, chefID.String(), uuid.NewString(), country).Error)
			for _, body := range []string{`{"subtotal":100}`, `{"subtotal":100,"fulfillment":"pickup","country":"IN"}`} {
				out := quote(t, chefID, body)
				require.Equal(t, country, out["taxCountry"])
				require.Equal(t, rate, out["taxRatePercent"])
				require.Equal(t, "GST", out["taxName"])
				require.Contains(t, out, "subtotal")
				require.Less(t, out["subtotal"].(float64), 100.0)
				require.InDelta(t, out["total"], out["subtotal"].(float64)+out["effectiveDeliveryFee"].(float64)+out["platformFee"].(float64)+out["tax"].(float64), 0.001)
				lines, err := json.Marshal(out["taxLines"])
				require.NoError(t, err)
				require.NotContains(t, string(lines), "CGST")
				require.NotContains(t, string(lines), "SGST")
				require.Greater(t, out["tax"].(float64), 0.0)
			}
		})
	}
}

func TestQuoteEndpoint_RejectsCrossMarketDelivery(t *testing.T) {
	db := setupQuoteDB(t)
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, payout_country) VALUES (?,?,?)`, chefID.String(), uuid.NewString(), "NZ").Error)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/chefs/:id/delivery-quote", (&OrderHandler{}).QuoteDeliveryFee)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/chefs/"+chefID.String()+"/delivery-quote", strings.NewReader(`{"country":"AU"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	require.Contains(t, w.Body.String(), "address_market_mismatch")
}
