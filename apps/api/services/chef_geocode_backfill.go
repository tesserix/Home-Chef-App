package services

import (
	"log"
	"strings"

	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// BackfillChefCoordinates fills lat/lng for chefs that have an address but no
// coordinates yet (legacy rows created before geocoding existed). Idempotent and
// best-effort — safe to run on every boot.
func BackfillChefCoordinates(db *gorm.DB) {
	var chefs []models.ChefProfile
	if err := db.Where("(latitude = 0 OR latitude IS NULL) AND address_line1 <> ''").
		Find(&chefs).Error; err != nil {
		log.Printf("chef-coord backfill: query failed: %v", err)
		return
	}
	for _, ch := range chefs {
		parts := []string{}
		for _, p := range []string{ch.AddressLine1, ch.AddressLine2, ch.City, ch.State, ch.PostalCode} {
			if strings.TrimSpace(p) != "" {
				parts = append(parts, p)
			}
		}
		full := strings.TrimSpace(strings.Join(parts, ", "))
		lat, lng, ok := GeocodeAddress(full)
		if !ok {
			continue
		}
		db.Model(&models.ChefProfile{}).Where("id = ?", ch.ID).
			Updates(map[string]any{"latitude": lat, "longitude": lng})
	}
	log.Printf("chef-coord backfill: processed %d chef(s)", len(chefs))
}
