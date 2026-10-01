package markets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCountryPolicy(t *testing.T) {
	for _, tc := range []struct{ country, provider, currency, status string }{
		{"IN", "cashfree", "INR", "active"},
		{"AU", "stripe", "AUD", "planned"},
		{"NZ", "stripe", "NZD", "planned"},
	} {
		t.Run(tc.country, func(t *testing.T) {
			p, ok := Lookup(tc.country)
			if !ok || p.PaymentProvider != tc.provider || p.Currency != tc.currency || p.Status != tc.status {
				t.Fatalf("unexpected policy: %+v, found=%v", p, ok)
			}
		})
	}
	for _, country := range []string{"", "US", "AUSTRALIA", "IN,AU", "123"} {
		if _, ok := Lookup(country); ok {
			t.Errorf("accepted unsupported market %q", country)
		}
	}
	if p, ok := Lookup(" au "); !ok || p.CountryCode != "AU" {
		t.Fatal("normalization failed")
	}
}

func TestMarketAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"))
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/api/v1/markets", 200},
		{"/api/v1/markets/AU", 200},
		{"/api/v1/markets/nz", 200},
		{"/api/v1/markets/US", 404},
		{"/api/v1/markets/INDIA", 404},
	} {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if tc.status == 404 {
				var result struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Code != "unsupported_market" {
					t.Fatalf("unstable error contract: %s", w.Body.String())
				}
			}
			if tc.path == "/api/v1/markets" {
				var result struct {
					Markets []Policy `json:"markets"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if len(result.Markets) != 3 {
					t.Fatalf("markets=%v", result.Markets)
				}
				for _, p := range result.Markets {
					if p.CountryCode != "IN" && p.Status != "planned" {
						t.Fatal("Stripe market enabled before launch gates")
					}
				}
			}
			if tc.path == "/api/v1/markets/AU" {
				var result struct {
					Market Policy `json:"market"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Market.PaymentProvider != "stripe" || result.Market.Currency != "AUD" || result.Market.Status != "planned" {
					t.Fatalf("unexpected AU response: %s", w.Body.String())
				}
			}
		})
	}
}
