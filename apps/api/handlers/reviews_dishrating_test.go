package handlers

// reviews_dishrating_test.go — per-dish ratings roll-up (#145). A review can
// carry per-dish ratings (DishRating); each dish's MenuItem.Rating must reflect
// the average of its dish ratings so customers see per-dish scores.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

func TestRecomputeMenuItemRating(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	// MenuItem carries gorm.DeletedAt → the Updates query appends deleted_at IS
	// NULL, so the column must exist.
	require.NoError(t, db.Exec(`CREATE TABLE menu_items (
		id text PRIMARY KEY, rating real DEFAULT 0, total_reviews integer DEFAULT 0,
		updated_at datetime, deleted_at datetime
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE dish_ratings (
		id text PRIMARY KEY, review_id text, menu_item_id text, chef_id text, rating integer, created_at datetime
	)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })

	mid := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO menu_items (id, rating, total_reviews) VALUES (?, 0, 0)`, mid.String()).Error)
	for _, r := range []int{4, 5} {
		require.NoError(t, db.Exec(`INSERT INTO dish_ratings (id, menu_item_id, chef_id, rating) VALUES (?, ?, ?, ?)`,
			uuid.New().String(), mid.String(), uuid.New().String(), r).Error)
	}

	recomputeMenuItemRating(mid)

	var rating float64
	var total int
	db.Raw(`SELECT rating FROM menu_items WHERE id = ?`, mid.String()).Scan(&rating)
	db.Raw(`SELECT total_reviews FROM menu_items WHERE id = ?`, mid.String()).Scan(&total)
	if rating != 4.5 {
		t.Fatalf("dish rating avg should be 4.5, got %v", rating)
	}
	if total != 2 {
		t.Fatalf("dish review count should be 2, got %d", total)
	}

	// A dish with no ratings stays at zero.
	other := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO menu_items (id, rating, total_reviews) VALUES (?, 0, 0)`, other.String()).Error)
	recomputeMenuItemRating(other)
	var r2 float64
	db.Raw(`SELECT rating FROM menu_items WHERE id = ?`, other.String()).Scan(&r2)
	if r2 != 0 {
		t.Fatalf("dish with no ratings should stay 0, got %v", r2)
	}
}
