package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/homechef/api/models"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOnboardingValidatesKitchenMarket(t *testing.T) {
	for _, tc := range []struct {
		country, postcode, phone string
		status                   int
	}{
		{"AU", "3000", "412345678", 428}, {"NZ", "1010", "211234567", 428},
		{"US", "10001", "9876543210", 400}, {"AU", "560001", "412345678", 400},
	} {
		t.Run(tc.country+tc.postcode, func(t *testing.T) {
			db := setupOnboardingPhoneDB(t)
			user := seedPhoneUser(t, db, "", models.PoolBusiness)
			body, err := json.Marshal(map[string]any{"fullName": "Test Chef", "phone": tc.phone, "email": "chef@fe3dr.example.test", "businessName": "Country Test", "description": "Home cooked food", "cuisines": []string{"Home cooked"}, "kitchenAddress": map[string]string{"country": tc.country, "line1": "10 Test Road", "city": "Test City", "state": "Test State", "postalCode": tc.postcode}})
			require.NoError(t, err)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Set("userID", user)
			c.Request = httptest.NewRequest(http.MethodPost, "/chef/onboarding", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			NewUploadHandler().Onboarding(c)
			require.Equal(t, tc.status, w.Code, w.Body.String())
		})
	}
}
