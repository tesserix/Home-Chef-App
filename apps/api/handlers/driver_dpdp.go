package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

// driver_dpdp.go — DPDP Act 2023 data-subject endpoints for the delivery-driver
// role, on the shared dpdp_common.go scaffolding:
//   - GET  /driver/me/export → dump of the driver's personal + delivery data
//   - POST /driver/me/delete → soft-delete + 30-day retention (confirmEmail)
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

// DeleteMyAccount soft-deletes the driver's user account per the DPDP right to
// erasure (30-day retention on the user row) and removes the delivery-partner
// profile so the assigner can never pick this driver again — mirroring how the
// chef flow removes the chef profile. In-flight deliveries remain server-side
// (referenced by partner id) for the operational/financial record.
//
// POST /driver/me/delete   { "confirmEmail": "<user.email>" }
func (h *DriverDPDPHandler) DeleteMyAccount(c *gin.Context) {
	user, proceed := beginSelfDelete(c)
	if !proceed {
		return
	}
	userID, _ := middleware.GetUserID(c)
	finalizeSelfDelete(c, user, "driver.account.delete", func() {
		// DeliveryPartner has no soft-delete column, so this removes the row —
		// which is exactly what stops dispatch (the assigner queries the table
		// by is_online/is_active and the row is gone).
		database.DB.Where("user_id = ?", userID).Delete(&models.DeliveryPartner{})
	})
}
