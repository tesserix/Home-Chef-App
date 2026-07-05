package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

// customer_dpdp.go — DPDP Act 2023 data-subject endpoints for the customer role,
// mirroring chef_dpdp.go on the shared dpdp_common.go scaffolding:
//   - GET  /me/export  → machine-readable dump of the customer's personal data
//   - POST /me/delete  → soft-delete + 30-day retention (confirmEmail required)
type CustomerDPDPHandler struct{}

func NewCustomerDPDPHandler() *CustomerDPDPHandler { return &CustomerDPDPHandler{} }

// ExportMyData returns every row Home Chef holds for the authenticated customer,
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

	writeExportJSON(c, dump)
}

// DeleteMyAccount soft-deletes the customer account per the DPDP right to
// erasure. Existing orders remain server-side through the retention window for
// legal/financial holds; the account itself is hidden immediately.
//
// POST /me/delete   { "confirmEmail": "<user.email>" }
func (h *CustomerDPDPHandler) DeleteMyAccount(c *gin.Context) {
	user, proceed := beginSelfDelete(c)
	if !proceed {
		return
	}
	userID, _ := middleware.GetUserID(c)
	finalizeSelfDelete(c, user, "customer.account.delete", func() {
		// Detach saved addresses from the marketplace immediately so a stale
		// address can't be surfaced during the retention window. Soft-delete via
		// the model so admin tooling can still reach them for a legal hold.
		database.DB.Where("user_id = ?", userID).Delete(&models.Address{})
	})
}
