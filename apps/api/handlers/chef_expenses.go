package handlers

// chef_expenses.go — self-declared business expenses for the chef dashboard.
//   POST   /chef/expenses            → record an expense
//   GET    /chef/expenses            → list (?from&to&category&limit&offset)
//   PUT    /chef/expenses/:id        → edit one
//   DELETE /chef/expenses/:id        → remove one
//   GET    /chef/expenses/summary    → FY totals by category + month (?year=FY-start)
//
// Expenses are the chef's own bookkeeping (gas, ingredients, utensils …).
// They feed analytics and the FY statement only — never settlement math.

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

const (
	expensesDefaultLimit = 50
	expensesMaxLimit     = 200
	// expenseMaxAmount rejects fat-finger entries (₹10 lakh per line item).
	expenseMaxAmount = 1_000_000
)

// ChefExpensesHandler serves chef expense bookkeeping.
type ChefExpensesHandler struct{}

// NewChefExpensesHandler constructs the handler.
func NewChefExpensesHandler() *ChefExpensesHandler {
	return &ChefExpensesHandler{}
}

// expenseRequest is the create/update wire shape.
type expenseRequest struct {
	Category    models.ChefExpenseCategory `json:"category" binding:"required"`
	Amount      float64                    `json:"amount" binding:"required"`
	Note        string                     `json:"note"`
	ExpenseDate string                     `json:"expenseDate" binding:"required"` // YYYY-MM-DD
	ReceiptURL  string                     `json:"receiptUrl"`
}

// parseAndValidate normalises the request; returns the parsed date or an error string.
func (r *expenseRequest) parseAndValidate() (time.Time, string) {
	if !models.ValidExpenseCategory(r.Category) {
		return time.Time{}, "Invalid category"
	}
	if r.Amount <= 0 || r.Amount > expenseMaxAmount {
		return time.Time{}, "Amount must be between 0 and 10,00,000"
	}
	if len(r.Note) > 500 {
		return time.Time{}, "Note too long (max 500 characters)"
	}
	day, err := services.ParseISTDate(r.ExpenseDate)
	if err != nil {
		return time.Time{}, "expenseDate must be YYYY-MM-DD"
	}
	if day.After(time.Now()) {
		return time.Time{}, "expenseDate cannot be in the future"
	}
	return day, ""
}

// CreateExpense records one expense for the calling chef.
func (h *ChefExpensesHandler) CreateExpense(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var req expenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	day, msg := req.parseAndValidate()
	if msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	expense := models.ChefExpense{
		ChefID:      chef.ID,
		UserID:      userID,
		Category:    req.Category,
		Amount:      services.Round2(req.Amount),
		Currency:    "INR",
		Note:        req.Note,
		ExpenseDate: day,
		ReceiptURL:  req.ReceiptURL,
	}
	if err := database.DB.Create(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save expense"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"expense": expense})
}

// ListExpenses returns the chef's expenses, newest first, with optional
// date-range and category filters.
func (h *ChefExpensesHandler) ListExpenses(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	q := database.DB.Model(&models.ChefExpense{}).Where("chef_id = ?", chef.ID)
	if raw := c.Query("from"); raw != "" {
		if day, perr := services.ParseISTDate(raw); perr == nil {
			q = q.Where("expense_date >= ?", day)
		}
	}
	if raw := c.Query("to"); raw != "" {
		if day, perr := services.ParseISTDate(raw); perr == nil {
			// Inclusive end-of-day: strictly before the next IST midnight.
			q = q.Where("expense_date < ?", day.AddDate(0, 0, 1))
		}
	}
	if raw := c.Query("category"); raw != "" {
		if !models.ValidExpenseCategory(models.ChefExpenseCategory(raw)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
			return
		}
		q = q.Where("category = ?", raw)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch expenses"})
		return
	}

	limit := expensesDefaultLimit
	if raw := c.Query("limit"); raw != "" {
		if n, perr := strconv.Atoi(raw); perr == nil && n > 0 {
			limit = n
		}
	}
	if limit > expensesMaxLimit {
		limit = expensesMaxLimit
	}
	offset := 0
	if raw := c.Query("offset"); raw != "" {
		if n, perr := strconv.Atoi(raw); perr == nil && n > 0 {
			offset = n
		}
	}

	var expenses []models.ChefExpense
	if err := q.Order("expense_date DESC, created_at DESC").
		Limit(limit).Offset(offset).
		Find(&expenses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch expenses"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expenses": expenses, "total": total})
}

// UpdateExpense edits one of the chef's own expenses.
func (h *ChefExpensesHandler) UpdateExpense(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	expenseID, perr := uuid.Parse(c.Param("id"))
	if perr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expense ID"})
		return
	}

	var req expenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	day, msg := req.parseAndValidate()
	if msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// Ownership check — a chef may only edit their own rows.
	var expense models.ChefExpense
	if err := database.DB.
		Where("id = ? AND chef_id = ?", expenseID, chef.ID).
		First(&expense).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
		return
	}

	expense.Category = req.Category
	expense.Amount = services.Round2(req.Amount)
	expense.Note = req.Note
	expense.ExpenseDate = day
	expense.ReceiptURL = req.ReceiptURL
	if err := database.DB.Save(&expense).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update expense"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"expense": expense})
}

// DeleteExpense removes one of the chef's own expenses.
func (h *ChefExpensesHandler) DeleteExpense(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	expenseID, perr := uuid.Parse(c.Param("id"))
	if perr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expense ID"})
		return
	}

	res := database.DB.
		Where("id = ? AND chef_id = ?", expenseID, chef.ID).
		Delete(&models.ChefExpense{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete expense"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Expense not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// GetExpenseSummary aggregates one financial year's expenses by category and
// by month — the analytics view. ?year=FY-start (defaults to current FY).
func (h *ChefExpensesHandler) GetExpenseSummary(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	fyStartYear, msg := parseFYStartYear(c.Query("year"))
	if msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	summary, err := services.ComputeExpenseSummary(chef.ID, fyStartYear)
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute summary"})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// parseFYStartYear resolves the ?year= FY-start-year param, defaulting to the
// current Indian financial year. Bounds match the TDS certificate.
func parseFYStartYear(raw string) (int, string) {
	currentFY := services.CurrentFinancialYearStart(time.Now())
	if raw == "" {
		return currentFY, ""
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, "Invalid year"
	}
	if n < tdsEarliestFYStart || n > currentFY {
		return 0, "year out of range"
	}
	return n, ""
}
