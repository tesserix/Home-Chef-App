package handlers

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// takenInPool reports whether a *different* user in the caller's own auth pool
// already holds the value matched by q.
//
// Contact details are unique per pool, not globally — the live constraint is
// UNIQUE (lower(email), auth_pool), so one human may hold a customer account and
// a business account on the same address. Legacy rows store NULL auth_pool and
// form their own bucket, which COALESCE keeps out of a pooled caller's way.
func takenInPool(q *gorm.DB, self models.User) bool {
	var other models.User
	return q.Where("id != ? AND COALESCE(auth_pool, '') = ?", self.ID, string(self.AuthPool)).
		First(&other).Error == nil
}

// loadUser fetches the caller's row, which every pool-scoped check needs for its
// auth_pool.
func loadUser(userID uuid.UUID) (models.User, error) {
	var user models.User
	err := database.DB.Where("id = ?", userID).First(&user).Error
	return user, err
}
