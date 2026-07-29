package services

import (
	"log"

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
	filled := 0
	for _, ch := range chefs {
		lat, lng, ok := GeocodeAddressParts(ch.AddressLine1, ch.AddressLine2, ch.City, ch.State, ch.PostalCode)
		if !ok {
			log.Printf("chef-coord backfill: no geocode match for chef %s (%s, %s) — still uncoordinated, so it stays hidden from located customers", ch.ID, ch.City, ch.State)
			continue
		}
		filled++
		db.Model(&models.ChefProfile{}).Where("id = ?", ch.ID).
			Updates(map[string]any{"latitude": lat, "longitude": lng})
	}
	// Report both numbers: "processed N" alone hid the failure mode that left every
	// kitchen at 0,0 and therefore undiscoverable.
	log.Printf("chef-coord backfill: processed %d chef(s), filled %d", len(chefs), filled)
}
