package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// customer_dpdp.go — DPDP Act 2023 data-subject endpoints for the customer role,
// mirroring chef_dpdp.go on the shared dpdp_common.go scaffolding:
//   - GET  /me/export  → machine-readable dump of the customer's personal data
//
// Deletion is NOT here: it lives in account_lifecycle.go, which adds blocker
// checks, the restore window (services.RestoreWindow) and GIP credential teardown.
type CustomerDPDPHandler struct{}

func NewCustomerDPDPHandler() *CustomerDPDPHandler { return &CustomerDPDPHandler{} }

// ExportMyData returns every row Fe3dr holds for the authenticated customer,
// each table scoped by the token-derived user/customer id so no other user's
// data can leak. Orders are projected through ToResponse (the customer's own
// view) rather than dumped raw.
//
// GET /me/export
func (h *CustomerDPDPHandler) ExportMyData(c *gin.Context) {
	user, ok := loadExportUser(c)
	if !ok {
		return
	}
	userID := user.ID

	dump := newExportEnvelope(user)

	var addresses []models.Address
	database.DB.Where("user_id = ?", userID).Find(&addresses)
	dump["addresses"] = addresses

	var orders []models.Order
	database.DB.Where("customer_id = ?", userID).Preload("Items").Find(&orders)
	orderExports := make([]models.OrderResponse, 0, len(orders))
	for i := range orders {
		orderExports = append(orderExports, orders[i].ToResponse())
	}
	dump["orders"] = orderExports

	var wallet models.Wallet
	if err := database.DB.Where("user_id = ?", userID).First(&wallet).Error; err == nil {
		dump["wallet"] = wallet
	}
	var walletTxns []models.WalletTxn
	database.DB.Where("user_id = ?", userID).Find(&walletTxns)
	dump["walletTransactions"] = walletTxns

	var mealPlans []models.MealPlan
	database.DB.Where("customer_id = ?", userID).Find(&mealPlans)
	dump["mealPlans"] = mealPlans

	var reviews []models.Review
	database.DB.Where("customer_id = ?", userID).Find(&reviews)
	dump["reviews"] = reviews

	var tips []models.Tip
	database.DB.Where("customer_id = ?", userID).Find(&tips)
	dump["tips"] = tips

	var catering []models.CateringRequest
	database.DB.Where("customer_id = ?", userID).Find(&catering)
	dump["cateringRequests"] = catering

	// #937: the refund-abuse profile and the events behind it are personal data and an
	// automated assessment of the person, so the export has to show both — the score AND
	// what it was derived from. Anything less is a decision the subject cannot contest.
	if profile := services.GetCustomerRiskProfile(database.DB, userID); profile != nil {
		dump["riskProfile"] = profile
	}
	var riskEvents []models.CustomerRiskEvent
	database.DB.Where("customer_id = ?", userID).Order("occurred_at DESC").Find(&riskEvents)
	dump["riskEvents"] = riskEvents

	writeExportJSON(c, dump)
}
