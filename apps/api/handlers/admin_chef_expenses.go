package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type AdminChefExpensesHandler struct{}

func NewAdminChefExpensesHandler() *AdminChefExpensesHandler {
	return &AdminChefExpensesHandler{}
}

// GetChefExpenses returns one chef's expense book for the requested FY with
// signed receipt URLs, plus the category/month summary the admin panel charts.
func (h *AdminChefExpensesHandler) GetChefExpenses(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef ID"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	fyStartYear, msg := parseFYStartYear(c.Query("year"))
	if msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	limit := 100
	if raw := c.Query("limit"); raw != "" {
		if n, perr := strconv.Atoi(raw); perr == nil && n > 0 && n <= 500 {
			limit = n
		}
	}

	start, end := services.FinancialYearWindow(fyStartYear)
	var expenses []models.ChefExpense
	if err := database.DB.
		Where("chef_id = ? AND expense_date >= ? AND expense_date < ?", chefID, start, end).
		Order("expense_date DESC, created_at DESC").
		Limit(limit).
		Find(&expenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch expenses"})
		return
	}

	summary, err := services.ComputeExpenseSummary(chefID, fyStartYear)
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chefId":       chefID,
		"businessName": chef.BusinessName,
		"expenses":     buildExpenseResponses(c, expenses),
		"summary":      summary,
	})
}
