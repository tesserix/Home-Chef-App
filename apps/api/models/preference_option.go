package models

import (
	"time"

	"github.com/google/uuid"
)

type PreferenceOption struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Category    string    `gorm:"type:varchar(50);not null;index" json:"category"` // dietary, allergy, cuisine, spice_level, household_size
	Value       string    `gorm:"type:varchar(50);not null" json:"value"`
	Label       string    `gorm:"type:varchar(100);not null" json:"label"`
	Description string    `gorm:"type:varchar(255)" json:"description,omitempty"`
	SortOrder   int       `gorm:"default:0" json:"sortOrder"`
	IsActive    bool      `gorm:"default:true" json:"isActive"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
}
