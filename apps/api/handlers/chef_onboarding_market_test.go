package handlers

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

func postMarketOnboarding(t *testing.T, userID uuid.UUID, phone string, address map[string]any, extra map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	payload := map[string]any{
		"fullName":       "Chef Tester",
		"phone":          phone,
		"email":          "chef.tester@fe3dr.com",
		"businessName":   "Kitchen " + uuid.NewString()[:8],
		"description":    "Home cooked meals",
		"kitchenType":    "home_kitchen",
		"cuisines":       []string{"modern_australian"},
		"kitchenAddress": address,
	}
	for k, v := range extra {
		payload[k] = v
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/chef/onboarding", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userID", userID)
	(&UploadHandler{}).Onboarding(c)
	return w
}

func TestChefOnboarding_ValidatesTheKitchensMarket(t *testing.T) {
	db := setupOnboardingPhoneDB(t)
	melbourne := map[string]any{"line1": "1 Collins St", "city": "Melbourne", "state": "VIC", "postalCode": "3000", "country": "AU"}
	auckland := map[string]any{"line1": "1 Queen St", "city": "Auckland", "state": "Auckland", "postalCode": "1010", "country": "NZ"}

	cases := []struct {
		name      string
		phone     string
		address   map[string]any
		extra     map[string]any
		wantCode  int
		wantField string
	}{
		{"australian kitchen clears locale checks", "412345678", melbourne, map[string]any{"businessNumber": "51 824 753 556"}, http.StatusPreconditionRequired, ""},
		{"new zealand kitchen clears locale checks", "211234567", auckland, nil, http.StatusPreconditionRequired, ""},
		{"indian mobile on an australian kitchen", "9876543210", melbourne, nil, http.StatusBadRequest, "phone"},
		{"indian PIN on a new zealand kitchen", "211234567", map[string]any{"line1": "x", "city": "Auckland", "state": "Auckland", "postalCode": "560001", "country": "NZ"}, nil, http.StatusBadRequest, "postalCode"},
		{"unserved country", "412345678", map[string]any{"line1": "x", "city": "NYC", "state": "NY", "postalCode": "10001", "country": "US"}, nil, http.StatusBadRequest, "country"},
		{"bad ABN", "412345678", melbourne, map[string]any{"businessNumber": "51824753557"}, http.StatusBadRequest, "businessNumber"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vendor := seedPhoneUser(t, db, "", models.PoolBusiness)
			w := postMarketOnboarding(t, vendor, tc.phone, tc.address, tc.extra)
			require.Equal(t, tc.wantCode, w.Code, "body: %s", w.Body.String())
			if tc.wantField != "" {
				var body map[string]any
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
				require.Equal(t, tc.wantField, body["field"])
			}
		})
	}
}
