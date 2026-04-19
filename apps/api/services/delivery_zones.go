package services

import (
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// FindZoneForAddress returns the first active DeliveryZone whose bounding box
// contains (lat, lon), or nil if the address is outside every configured zone.
// Callers that get nil when (lat, lon) are both zero should treat that as
// "no coordinates — skip the check" rather than "unserviceable".
func FindZoneForAddress(lat, lon float64) *models.DeliveryZone {
	if lat == 0 && lon == 0 {
		return nil
	}
	var zone models.DeliveryZone
	// Overlapping zones are allowed; we pick the oldest match so the
	// "first" zone is stable across reads regardless of DB backend.
	err := database.DB.
		Where("is_active = ? AND min_latitude <= ? AND max_latitude >= ? AND min_longitude <= ? AND max_longitude >= ?",
			true, lat, lat, lon, lon).
		Order("created_at ASC").
		First(&zone).Error
	if err != nil {
		return nil
	}
	return &zone
}

// HasActiveZones reports whether any DeliveryZone rows exist so the order
// flow knows whether the zone check should be enforced at all. If no zones
// are configured we don't want to block deliveries — the feature is opt-in.
func HasActiveZones() bool {
	var count int64
	database.DB.Model(&models.DeliveryZone{}).Where("is_active = ?", true).Count(&count)
	return count > 0
}
