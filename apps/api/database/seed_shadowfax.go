package database

import (
	"log"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// SeedShadowfaxProvider upserts the Shadowfax 3PL provider row. It ships DISABLED
// (IsEnabled=false) — both FindAvailableProvider and the inbound webhook require
// is_enabled=true, so nothing dispatches or accepts callbacks until the owner
// flips it on from the admin provider API after sandbox verification.
//
// The API token + webhook secret come from Secret Manager via env (config), never
// hard-coded in source (the repo is public). A blank token leaves the row created
// but unusable, which is fine — it stays dark.
func SeedShadowfaxProvider(db *gorm.DB) {
	token := config.AppConfig.ShadowfaxAPIToken
	webhookSecret := config.AppConfig.ShadowfaxWebhookSecret

	var existing models.DeliveryProvider
	err := db.Where("code = ?", "shadowfax").First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		provider := models.DeliveryProvider{
			Name:               "Shadowfax",
			Code:               "shadowfax",
			Description:        "Shadowfax hyperlocal 3PL (Unified API)",
			APIBaseURL:         "https://dale.staging.shadowfax.in/api",
			APIKey:             token,
			WebhookSecret:      webhookSecret,
			SupportedCities:    `["Bengaluru"]`, // owner edits via admin API for their test city
			SupportedCountries: `["IN"]`,
			MaxDistance:        50,
			Priority:           1,
			IsEnabled:          false, // dark until verified + flipped
			IsActive:           true,
		}
		if e := db.Create(&provider).Error; e != nil {
			log.Printf("seed shadowfax provider: %v", e)
		} else {
			log.Println("seeded Shadowfax provider (disabled)")
		}
		return
	}
	if err != nil {
		log.Printf("seed shadowfax provider lookup: %v", err)
		return
	}
	// Refresh credentials + base URL on an existing row but DO NOT auto-enable —
	// the owner controls IsEnabled. Keeps the token in sync with Secret Manager.
	if e := db.Model(&existing).Updates(map[string]any{
		"api_base_url":   "https://dale.staging.shadowfax.in/api",
		"api_key":        token,
		"webhook_secret": webhookSecret,
	}).Error; e != nil {
		log.Printf("refresh shadowfax provider: %v", e)
	}
}
