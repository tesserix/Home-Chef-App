package models

// delivery_distance_cache.go — the DURABLE road-distance cache (#699 cost
// control). Road distance between a chef and a drop address never changes, so we
// pay the routing provider for a given (chef → address) pair at most ONCE, ever,
// and persist the answer here for every future order — even across Redis flushes
// and restarts. Redis is only the hot layer in front of this.
//
// Per repo policy this is an ORM model only (AutoMigrate creates the table);
// schema/permissions live in tesserix-k8s.

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryDistanceCache is one computed chef→drop road distance, keyed by the
// rounded coordinate pair so a repeat order from the same address reuses it.
type DeliveryDistanceCache struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	// CacheKey is the rounded "chefLat,chefLng:dropLat,dropLng" identity. Unique
	// so two concurrent first-orders for the same trip collapse to one row.
	CacheKey string `gorm:"type:varchar(80);uniqueIndex;not null" json:"cacheKey"`

	ChefLat  float64 `gorm:"not null" json:"chefLat"`
	ChefLng  float64 `gorm:"not null" json:"chefLng"`
	DropLat  float64 `gorm:"not null" json:"dropLat"`
	DropLng  float64 `gorm:"not null" json:"dropLng"`

	DistanceKm float64 `gorm:"not null" json:"distanceKm"`
	// Provider records which router produced it ("google" | "osrm" | "fallback"),
	// so a later provider change can invalidate just the rows it needs to.
	Provider string `gorm:"type:varchar(20)" json:"provider"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (DeliveryDistanceCache) TableName() string { return "delivery_distance_cache" }
