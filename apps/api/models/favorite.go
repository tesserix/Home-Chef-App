package models

import (
	"time"

	"github.com/google/uuid"
)

// FavoriteChef represents a customer's favorite/wishlisted chef (max 7 per user).
type FavoriteChef struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_favorite_chefs_user_chef" json:"userId"`
	ChefID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_favorite_chefs_user_chef;index" json:"chefId"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`

	User User        `gorm:"foreignKey:UserID" json:"-"`
	Chef ChefProfile `gorm:"foreignKey:ChefID" json:"-"`
}

type FavoriteChefResponse struct {
	ID        uuid.UUID            `json:"id"`
	ChefID    uuid.UUID            `json:"chefId"`
	Chef      ChefProfileResponse  `json:"chef"`
	CreatedAt time.Time            `json:"createdAt"`
}
