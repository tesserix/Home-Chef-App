package handlers

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Catering deposits are minted in INR on Cashfree, so kitchens outside India
// must not see or quote on requests until the market has its own rail.
func TestCatering_NonRupeeChefCannotBrowseOrQuote(t *testing.T) {
	db := setupPayDB(t)
	chefUser := payUser(t, db, "chef")
	chefID := payChef(t, db, chefUser)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET payout_country = 'AU' WHERE id = ?`, chefID.String()).Error)

	w := callAs(chefUser, http.MethodGet, "/chef/catering/requests", func(r *gin.Engine) {
		r.GET("/chef/catering/requests", NewCateringHandler().GetAvailableRequests)
	}, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"data":[],"total":0,"page":1,"limit":20}`, w.Body.String())

	w = callAs(chefUser, http.MethodPost, "/chef/catering/requests/"+uuid.NewString()+"/quote", func(r *gin.Engine) {
		r.POST("/chef/catering/requests/:id/quote", NewCateringHandler().SubmitQuote)
	}, map[string]any{"proposedMenu": "x", "pricePerPerson": 20, "totalPrice": 400})
	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	require.Contains(t, w.Body.String(), "market_not_supported")
}
