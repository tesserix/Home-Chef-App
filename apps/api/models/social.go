package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusPublished PostStatus = "published"
	PostStatusArchived  PostStatus = "archived"
	PostStatusFlagged   PostStatus = "flagged"
)

type Post struct {
	ID       uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"chefId"`
	Status   PostStatus     `gorm:"type:varchar(20);default:'published'" json:"status"`
	Content  string         `gorm:"type:text;not null" json:"content"`
	Images   pq.StringArray `gorm:"type:text[]" json:"images"`
	Hashtags pq.StringArray `gorm:"type:text[]" json:"hashtags"`

	// Linked Menu Item (optional)
	MenuItemID *uuid.UUID `gorm:"type:uuid" json:"menuItemId,omitempty"`

	// Engagement
	LikesCount    int `gorm:"default:0" json:"likesCount"`
	CommentsCount int `gorm:"default:0" json:"commentsCount"`
	SharesCount   int `gorm:"default:0" json:"sharesCount"`

	// Moderation
	IsModerated    bool   `gorm:"default:false" json:"isModerated"`
	ModeratedAt    *time.Time `gorm:"" json:"moderatedAt,omitempty"`
	ModeratorNote  string `gorm:"" json:"moderatorNote,omitempty"`
	ContactInfoDetected bool `gorm:"default:false" json:"contactInfoDetected"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Chef     ChefProfile   `gorm:"foreignKey:ChefID" json:"chef,omitempty"`
	MenuItem *MenuItem     `gorm:"foreignKey:MenuItemID" json:"menuItem,omitempty"`
	Likes    []PostLike    `gorm:"foreignKey:PostID" json:"likes,omitempty"`
	Comments []PostComment `gorm:"foreignKey:PostID" json:"comments,omitempty"`
}

type PostLike struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PostID    uuid.UUID `gorm:"type:uuid;not null;index" json:"postId"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`

	Post Post `gorm:"foreignKey:PostID" json:"-"`
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type PostComment struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PostID    uuid.UUID `gorm:"type:uuid;not null;index" json:"postId"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`
	ParentID  *uuid.UUID `gorm:"type:uuid;index" json:"parentId,omitempty"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	IsHidden  bool      `gorm:"default:false" json:"isHidden"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Post    Post          `gorm:"foreignKey:PostID" json:"-"`
	User    User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Replies []PostComment `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}

// DTOs
type PostResponse struct {
	ID            uuid.UUID       `json:"id"`
	ChefID        uuid.UUID       `json:"chefId"`
	Chef          ChefPostInfo    `json:"chef"`
	Status        PostStatus      `json:"status"`
	Content       string          `json:"content"`
	Images        []string        `json:"images"`
	Hashtags      []string        `json:"hashtags"`
	MenuItemID    *uuid.UUID      `json:"menuItemId,omitempty"`
	MenuItem      *MenuItemBasic  `json:"menuItem,omitempty"`
	LikesCount    int             `json:"likesCount"`
	CommentsCount int             `json:"commentsCount"`
	IsLiked       bool            `json:"isLiked"`
	CreatedAt     time.Time       `json:"createdAt"`
}

type ChefPostInfo struct {
	ID           uuid.UUID `json:"id"`
	BusinessName string    `json:"businessName"`
	ProfileImage string    `json:"profileImage"`
	IsVerified   bool      `json:"verified"`
}

type MenuItemBasic struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Price float64   `json:"price"`
	Image string    `json:"image"`
}

type CommentResponse struct {
	ID        uuid.UUID         `json:"id"`
	Content   string            `json:"content"`
	User      CommentUserInfo   `json:"user"`
	Replies   []CommentResponse `json:"replies,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
}

type CommentUserInfo struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	Avatar    string    `json:"avatar,omitempty"`
}

func (p *Post) ToResponse(userID *uuid.UUID) PostResponse {
	images := []string{}
	if p.Images != nil {
		images = p.Images
	}
	hashtags := []string{}
	if p.Hashtags != nil {
		hashtags = p.Hashtags
	}

	response := PostResponse{
		ID:            p.ID,
		ChefID:        p.ChefID,
		Status:        p.Status,
		Content:       p.Content,
		Images:        images,
		Hashtags:      hashtags,
		MenuItemID:    p.MenuItemID,
		LikesCount:    p.LikesCount,
		CommentsCount: p.CommentsCount,
		CreatedAt:     p.CreatedAt,
	}

	if p.Chef.ID != uuid.Nil {
		response.Chef = ChefPostInfo{
			ID:           p.Chef.ID,
			BusinessName: p.Chef.BusinessName,
			ProfileImage: p.Chef.ProfileImage,
			IsVerified:   p.Chef.IsVerified,
		}
	}

	if p.MenuItem != nil {
		response.MenuItem = &MenuItemBasic{
			ID:    p.MenuItem.ID,
			Name:  p.MenuItem.Name,
			Price: p.MenuItem.Price,
			Image: p.MenuItem.ImageURL,
		}
	}

	// Check if user liked the post
	if userID != nil {
		for _, like := range p.Likes {
			if like.UserID == *userID {
				response.IsLiked = true
				break
			}
		}
	}

	return response
}
