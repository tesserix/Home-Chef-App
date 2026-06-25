package database

import (
	"log"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// SeedBorzoProvider upserts the Borzo 3PL provider row. It ships DISABLED
// (IsEnabled=false) — both FindAvailableProvider and the inbound webhook require
// is_enabled=true, so nothing dispatches or accepts callbacks until the owner
// flips it on after validating the sandbox token.
//
// The API token comes from Secret Manager via env BORZO_API_TOKEN, never source
// (the repo is public). On a deploy the token is only refreshed when the env is
// non-empty, so a token set directly for sandbox testing survives redeploys.
func SeedBorzoProvider(db *gorm.DB) {
	token := config.AppConfig.BorzoAPIToken

	var existing models.DeliveryProvider
	err := db.Where("code = ?", "borzo").First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		provider := models.DeliveryProvider{
			Name:               "Borzo",
			Code:               "borzo",
			Description:        "Borzo on-demand intracity courier (Business API)",
			APIBaseURL:         "https://robotapitest-in.borzodelivery.com/api/business/1.8",
			APIKey:             token,
			SupportedCities:    `["Bengaluru"]`, // owner edits via admin API for their test city
			SupportedCountries: `["IN"]`,
			MaxDistance:        50,
			Priority:           1,
			IsEnabled:          false, // dark until verified + flipped
			IsActive:           true,
		}
		if e := db.Create(&provider).Error; e != nil {
			log.Printf("seed borzo provider: %v", e)
		} else {
			log.Println("seeded Borzo provider (disabled)")
		}
		return
	}
	if err != nil {
		log.Printf("seed borzo provider lookup: %v", err)
		return
	}
	// Refresh base URL always; refresh the token only when env supplies one, so a
	// hand-set sandbox token isn't blanked on the next deploy. Never auto-enable.
	updates := map[string]any{
		"api_base_url": "https://robotapitest-in.borzodelivery.com/api/business/1.8",
	}
	if token != "" {
		updates["api_key"] = token
	}
	if e := db.Model(&existing).Updates(updates).Error; e != nil {
		log.Printf("refresh borzo provider: %v", e)
	}
}
