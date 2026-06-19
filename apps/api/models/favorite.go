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

// FavoriteDish represents a customer's saved/favorited menu item (#237). Unlike
// FavoriteChef (curated, max 7), dishes are more numerous so the cap is higher.
type FavoriteDish struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_favorite_dishes_user_item" json:"userId"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_favorite_dishes_user_item;index" json:"menuItemId"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`

	User     User     `gorm:"foreignKey:UserID" json:"-"`
	MenuItem MenuItem `gorm:"foreignKey:MenuItemID" json:"-"`
}

// FavoriteDishChef is the lightweight chef summary carried on a favorited dish
// so the Favorites surface can show "from <chef>" + link without an extra fetch.
type FavoriteDishChef struct {
	ID           uuid.UUID `json:"id"`
	BusinessName string    `json:"businessName"`
	ProfileImage string    `json:"profileImage,omitempty"`
}

type FavoriteDishResponse struct {
	ID         uuid.UUID        `json:"id"`
	MenuItemID uuid.UUID        `json:"menuItemId"`
	MenuItem   MenuItemResponse `json:"menuItem"`
	Chef       FavoriteDishChef `json:"chef"`
	CreatedAt  time.Time        `json:"createdAt"`
}
