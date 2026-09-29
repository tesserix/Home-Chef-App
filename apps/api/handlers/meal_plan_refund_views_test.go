package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"github.com/stretchr/testify/require"
)

func TestRefundListsPreserveIndianMealDate(t *testing.T) {
	for _, role := range []string{"customer", "chef", "admin"} {
		t.Run(role, func(t *testing.T) {
			db := setupOrchestrationDB(t)
			old := config.AppConfig
			config.AppConfig = &config.Config{MealPlanRefundFlowV2Enabled: true}
			t.Cleanup(func() { config.AppConfig = old })
			for _, column := range []string{"refund_stage text", "refund_percent integer", "refund_floor_percent integer", "chef_refund_choice text", "slot text", "dish_name text", "variant text", "weekly_menu_item_id text", "prepared_at datetime", "delivered_at datetime", "refund_destination text", "refund_decision_by datetime", "customer_confirmed_at datetime", "payout_settled_at datetime", "payout_settle_attempts integer"} {
				require.NoError(t, db.Exec("ALTER TABLE meal_plan_days ADD COLUMN "+column).Error)
			}
			chefID, chefUserID := seedOrchChef(t, db)
			customerID := uuid.New()
			planID := seedOrchPlan(t, db, models.MealPlanConfirmed, customerID, chefID)
			require.NoError(t, db.Exec(`UPDATE meal_plans SET subtotal=200, platform_fee=8, tax=15.94, tax_food=10, tax_service=1.44, tax_delivery=4.5, total=248.94 WHERE id=?`, planID.String()).Error)
			stage := models.MPRefundPendingCustomer
			principal := customerID
			handler := (&MealPlanHandler{}).GetCustomerPendingRefundChoices
			if role == "chef" {
				stage = models.MPRefundPendingChef
				principal = chefUserID
				handler = (&MealPlanHandler{}).GetChefPendingRefundDecisions
			}
			if role == "admin" {
				stage = models.MPRefundPendingAdmin
				handler = (&MealPlanHandler{}).GetAdminPendingRefunds
			}
			date := time.Date(2026, 10, 1, 0, 0, 0, 0, istLoc).UTC()
			require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, date, slot, price, refund_stage, refund_percent, refund_floor_percent, commission_rate) VALUES (?,?,?,?,?,?,?,?,?)`, uuid.NewString(), planID.String(), date, "lunch", 200, stage, 100, 75, 0.1).Error)
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(func(c *gin.Context) { c.Set("userID", principal); c.Next() })
			r.GET("/refunds", handler)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/refunds", nil))
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			var response struct {
				Data []struct {
					Date         string  `json:"date"`
					Amount       float64 `json:"amount"`
					FullRefund   float64 `json:"fullRefund"`
					RefundAmount float64 `json:"refundAmount"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
			require.Len(t, response.Data, 1, w.Body.String())
			require.Equal(t, "2026-10-01", response.Data[0].Date)
			got := response.Data[0].Amount
			if role == "chef" {
				got = response.Data[0].FullRefund
			}
			if role == "admin" {
				got = response.Data[0].RefundAmount
			}
			require.Equal(t, 215.0, got, "food net of commission + food GST + delivery; exclude platform fee and other taxes")
		})
	}
}
