package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// driver_dpdp.go — DPDP Act 2023 data-subject endpoints for the delivery-driver
// role, on the shared dpdp_common.go scaffolding:
//   - GET  /driver/me/export → dump of the driver's personal + delivery data
//
// Deletion is NOT here: it lives in account_lifecycle.go, which adds blocker
// checks, the restore window (services.RestoreWindow) and GIP credential teardown.
type DriverDPDPHandler struct{}

func NewDriverDPDPHandler() *DriverDPDPHandler { return &DriverDPDPHandler{} }

// ExportMyData returns every row Home Chef holds for the authenticated driver,
// each table scoped by the token-derived id. The driver's own partner profile is
// projected full (it's their data); deliveries are scoped to their partner id.
//
// GET /driver/me/export
func (h *DriverDPDPHandler) ExportMyData(c *gin.Context) {
	user, ok := loadExportUser(c)
	if !ok {
		return
	}
	userID := user.ID

	dump := newExportEnvelope(user)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err == nil {
		dump["deliveryPartner"] = partner.ToDetailResponse()

		var deliveries []models.Delivery
		database.DB.Where("delivery_partner_id = ?", partner.ID).Find(&deliveries)
		dump["deliveries"] = deliveries

		var documents []models.DeliveryPartnerDocument
		database.DB.Where("partner_id = ?", partner.ID).Find(&documents)
		dump["documents"] = documents
	}

	// Referrals this driver made (scoped by referrer user id).
	var referrals []models.DriverReferral
	database.DB.Where("referrer_id = ?", userID).Find(&referrals)
	dump["referrals"] = referrals

	writeExportJSON(c, dump)
}
