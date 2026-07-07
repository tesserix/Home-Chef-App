package handlers

// chef_statements.go — weekly settlement statements for the chef dashboard.
//   GET /chef/statements/weekly?limit=12   → list issued statements (JSON)
//   GET /chef/statements/:id/statement.pdf → download one statement (PDF)

import (
	"fmt"
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

// statementsDefaultLimit / statementsMaxLimit bound the list query.
const (
	statementsDefaultLimit = 12
	statementsMaxLimit     = 52
)

// ChefStatementsHandler serves weekly settlement statements.
type ChefStatementsHandler struct{}

// NewChefStatementsHandler constructs the handler.
func NewChefStatementsHandler() *ChefStatementsHandler {
	return &ChefStatementsHandler{}
}

// statementResponse is the per-statement wire shape.
type statementResponse struct {
	ID                 uuid.UUID `json:"id"`
	WeekStart          string    `json:"weekStart"`
	WeekEnd            string    `json:"weekEnd"`
	Currency           string    `json:"currency"`
	OrdersCount        int       `json:"ordersCount"`
	GrossRevenue       float64   `json:"grossRevenue"`
	PlatformCommission float64   `json:"platformCommission"`
	CGST               float64   `json:"cgst"`
	SGST               float64   `json:"sgst"`
	IGST               float64   `json:"igst"`
	TDS                float64   `json:"tds"`
	NetPayout          float64   `json:"netPayout"`
	// Disbursement state (#617) — the model tracked these but the DTO dropped them,
	// so the chef couldn't tell a paid statement from a pending one. PaidAt/PayoutRef
	// are set only once disbursed (manual weekly mark-paid at launch, RazorpayX later).
	Status    models.PayoutStatus `json:"status"`
	PaidAt    *time.Time          `json:"paidAt,omitempty"`
	PayoutRef string              `json:"payoutRef,omitempty"`
}

// GetWeeklyStatements lists the chef's issued weekly statements, newest first.
func (h *ChefStatementsHandler) GetWeeklyStatements(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	limit := statementsDefaultLimit
	if raw := c.Query("limit"); raw != "" {
		if n, perr := strconv.Atoi(raw); perr == nil && n > 0 {
			limit = n
		}
	}
	if limit > statementsMaxLimit {
		limit = statementsMaxLimit
	}

	var statements []models.WeeklyStatement
	if err := database.DB.
		Where("chef_id = ?", chef.ID).
		Order("week_start DESC").
		Limit(limit).
		Find(&statements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch statements"})
		return
	}

	out := make([]statementResponse, 0, len(statements))
	for _, s := range statements {
		out = append(out, statementResponse{
			ID:                 s.ID,
			WeekStart:          s.WeekStart.Format("2006-01-02"),
			WeekEnd:            s.WeekEnd.Format("2006-01-02"),
			Currency:           s.Currency,
			OrdersCount:        s.OrdersCount,
			GrossRevenue:       s.GrossRevenue,
			PlatformCommission: s.PlatformCommission,
			CGST:               s.CGST,
			SGST:               s.SGST,
			IGST:               s.IGST,
			TDS:                s.TDS,
			NetPayout:          s.NetPayout,
			Status:             s.Status,
			PaidAt:             s.PaidAt,
			PayoutRef:          s.PayoutRef,
		})
	}
	c.JSON(http.StatusOK, gin.H{"statements": out})
}

// GetWeeklyStatementPDF streams the PDF for one of the chef's statements.
func (h *ChefStatementsHandler) GetWeeklyStatementPDF(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	statementID, perr := uuid.Parse(c.Param("id"))
	if perr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid statement ID"})
		return
	}

	// Ownership check before generating — a chef may only pull their own.
	var stmt models.WeeklyStatement
	if err := database.DB.
		Where("id = ? AND chef_id = ?", statementID, chef.ID).
		First(&stmt).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Statement not found"})
		return
	}

	pdfBytes, filename, genErr := services.GenerateWeeklyStatementPDF(statementID)
	if genErr != nil {
		services.CaptureSentryError(c, genErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate statement"})
		return
	}
	services.LogAudit(c, "chef.statement.download", "weekly_statement", statementID.String(),
		nil, gin.H{"weekStart": stmt.WeekStart.Format("2006-01-02")})
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
