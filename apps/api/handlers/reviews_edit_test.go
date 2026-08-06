package handlers

// reviews_edit_test.go — #1047. A customer's own review offered only "Report" and
// "Block yourself": there was no way to correct or withdraw what they had just
// written, because the API had no edit or delete route at all.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func reviewMutationReq(t *testing.T, userID uuid.UUID, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	h := NewReviewHandler()
	r.PATCH("/v1/reviews/:id", h.UpdateReview)
	r.DELETE("/v1/reviews/:id", h.DeleteReview)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdateReview_RewritesTheAuthorsOwnReview(t *testing.T) {
	db := setupOrderReviewDB(t)
	customerID := uuid.New()
	reviewID := seedOrderReview(t, db, uuid.New(), customerID)

	w := reviewMutationReq(t, customerID, http.MethodPatch, "/v1/reviews/"+reviewID.String(),
		`{"overallRating":3,"foodRating":3,"title":"Good, not great","comment":"Reheated well though"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body models.ReviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 3, body.OverallRating)
	assert.Equal(t, "Good, not great", body.Title)
	assert.Equal(t, "Reheated well though", body.Comment)

	var stored models.Review
	require.NoError(t, db.First(&stored, "id = ?", reviewID).Error)
	assert.Equal(t, 3, stored.OverallRating)
	assert.Equal(t, 3, stored.FoodRating)
	// Untouched sub-scores survive a partial edit.
	assert.Equal(t, 3, stored.DeliveryRating)
}

// Someone else's review is not editable, and the failure must not disclose that
// the review exists.
func TestUpdateReview_RefusesAnotherCustomersReview(t *testing.T) {
	db := setupOrderReviewDB(t)
	reviewID := seedOrderReview(t, db, uuid.New(), uuid.New())

	w := reviewMutationReq(t, uuid.New(), http.MethodPatch, "/v1/reviews/"+reviewID.String(),
		`{"overallRating":1}`)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var stored models.Review
	require.NoError(t, db.First(&stored, "id = ?", reviewID).Error)
	assert.Equal(t, 5, stored.OverallRating, "the review must be untouched")
}

func TestUpdateReview_RejectsAnOutOfRangeRating(t *testing.T) {
	db := setupOrderReviewDB(t)
	customerID := uuid.New()
	reviewID := seedOrderReview(t, db, uuid.New(), customerID)

	w := reviewMutationReq(t, customerID, http.MethodPatch, "/v1/reviews/"+reviewID.String(),
		`{"overallRating":9}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteReview_WithdrawsTheAuthorsOwnReview(t *testing.T) {
	db := setupOrderReviewDB(t)
	customerID := uuid.New()
	orderID := uuid.New()
	reviewID := seedOrderReview(t, db, orderID, customerID)
	require.NoError(t, db.Exec(`INSERT INTO dish_ratings (id, review_id, menu_item_id, chef_id, rating)
		VALUES (?,?,?,?,?)`, uuid.New().String(), reviewID.String(), uuid.New().String(), uuid.New().String(), 4).Error)

	w := reviewMutationReq(t, customerID, http.MethodDelete, "/v1/reviews/"+reviewID.String(), "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var count int64
	require.NoError(t, db.Model(&models.Review{}).Where("id = ?", reviewID).Count(&count).Error)
	assert.Zero(t, count, "a withdrawn review must stop counting towards the chef's rating")

	// The per-dish stars went with it — a deleted review must not keep pulling a
	// dish's score down.
	var dishes int64
	require.NoError(t, db.Table("dish_ratings").Where("review_id = ?", reviewID).Count(&dishes).Error)
	assert.Zero(t, dishes)
}

func TestDeleteReview_RefusesAnotherCustomersReview(t *testing.T) {
	db := setupOrderReviewDB(t)
	reviewID := seedOrderReview(t, db, uuid.New(), uuid.New())

	w := reviewMutationReq(t, uuid.New(), http.MethodDelete, "/v1/reviews/"+reviewID.String(), "")
	assert.Equal(t, http.StatusNotFound, w.Code)

	var count int64
	require.NoError(t, db.Model(&models.Review{}).Where("id = ?", reviewID).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestUpdateReview_RejectsAMalformedID(t *testing.T) {
	var _ *gorm.DB = setupOrderReviewDB(t)

	w := reviewMutationReq(t, uuid.New(), http.MethodPatch, "/v1/reviews/not-a-uuid", `{"overallRating":3}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
