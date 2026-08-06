package handlers

// reviews_order_test.go — #1046. A customer who had already reviewed an order was
// still shown an empty "Leave a review" form, re-entered every rating, and only
// then hit the 409 that the create path raises. The client cannot render the
// review it already has without a way to read it back.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

func setupOrderReviewDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE reviews (mode text DEFAULT 'live', test_session_id text, cloned_from_id text,
		id text PRIMARY KEY, order_id text, customer_id text, chef_id text,
		overall_rating integer, food_rating integer, delivery_rating integer, value_rating integer,
		packaging_rating integer, hygiene_rating integer,
		title text, comment text, images text, is_approved integer DEFAULT 1, is_hidden integer DEFAULT 0,
		hidden_reason text, chef_response text, chef_responded_at datetime, helpful_count integer DEFAULT 0,
		created_at datetime, updated_at datetime, deleted_at datetime)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE dish_ratings (
		id text PRIMARY KEY, review_id text, menu_item_id text, chef_id text, rating integer, created_at datetime)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func orderReviewReq(t *testing.T, userID uuid.UUID, orderID string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.GET("/v1/reviews/order/:orderId", NewReviewHandler().GetOrderReview)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/v1/reviews/order/"+orderID, nil))
	return w
}

func seedOrderReview(t *testing.T, db *gorm.DB, orderID, customerID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO reviews (id, order_id, customer_id, chef_id,
		overall_rating, food_rating, delivery_rating, value_rating, packaging_rating, hygiene_rating,
		title, comment, mode) VALUES (?,?,?,?,?,?,?,?,?,?,?,?, 'live')`,
		id.String(), orderID.String(), customerID.String(), uuid.New().String(),
		5, 4, 3, 5, 4, 5, "Excellent", "Butter chicken was superb").Error)
	return id
}

func TestGetOrderReview_ReturnsTheCustomersOwnReview(t *testing.T) {
	db := setupOrderReviewDB(t)
	orderID, customerID := uuid.New(), uuid.New()
	reviewID := seedOrderReview(t, db, orderID, customerID)

	menuItemID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO dish_ratings (id, review_id, menu_item_id, chef_id, rating)
		VALUES (?,?,?,?,?)`, uuid.New().String(), reviewID.String(), menuItemID.String(), uuid.New().String(), 4).Error)

	w := orderReviewReq(t, customerID, orderID.String())
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Review struct {
			ID              string `json:"id"`
			OverallRating   int    `json:"overallRating"`
			FoodRating      int    `json:"foodRating"`
			PackagingRating int    `json:"packagingRating"`
			HygieneRating   int    `json:"hygieneRating"`
			Title           string `json:"title"`
			Comment         string `json:"comment"`
			DishRatings     []struct {
				MenuItemID string `json:"menuItemId"`
				Rating     int    `json:"rating"`
			} `json:"dishRatings"`
		} `json:"review"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	assert.Equal(t, reviewID.String(), body.Review.ID)
	assert.Equal(t, 5, body.Review.OverallRating)
	assert.Equal(t, 4, body.Review.FoodRating)
	assert.Equal(t, 4, body.Review.PackagingRating)
	assert.Equal(t, 5, body.Review.HygieneRating)
	assert.Equal(t, "Excellent", body.Review.Title)
	assert.Equal(t, "Butter chicken was superb", body.Review.Comment)
	require.Len(t, body.Review.DishRatings, 1, "the per-dish stars must come back too")
	assert.Equal(t, menuItemID.String(), body.Review.DishRatings[0].MenuItemID)
	assert.Equal(t, 4, body.Review.DishRatings[0].Rating)
}

// No review yet is the normal case, not an error: the client uses it to decide
// whether to offer the form. A 404 body would make every un-reviewed order log
// as a failure.
func TestGetOrderReview_NoReviewYetIsNotAnError(t *testing.T) {
	setupOrderReviewDB(t)

	w := orderReviewReq(t, uuid.New(), uuid.New().String())
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"review":null}`, w.Body.String())
}

// Another customer's review is not this customer's to read.
func TestGetOrderReview_ScopedToTheCallingCustomer(t *testing.T) {
	db := setupOrderReviewDB(t)
	orderID := uuid.New()
	seedOrderReview(t, db, orderID, uuid.New())

	w := orderReviewReq(t, uuid.New(), orderID.String())
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"review":null}`, w.Body.String())
}

func TestGetOrderReview_RejectsAMalformedOrderID(t *testing.T) {
	setupOrderReviewDB(t)

	w := orderReviewReq(t, uuid.New(), "not-a-uuid")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
