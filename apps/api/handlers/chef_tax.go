package handlers

// chef_tax.go — chef tax documents.
//   GET /chef/tax/certificate?year=YYYY → annual TDS summary PDF (Form 16A style)
//
// `year` is the FINANCIAL-year start year (Indian FY runs 1 Apr – 31 Mar):
// year=2025 → FY 2025-26. Defaults to the current financial year.

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// tdsEarliestFYStart bounds the earliest financial year a certificate can be
// requested for — guards against absurd inputs hammering the order query.
const tdsEarliestFYStart = 2020

// ChefTaxHandler serves chef tax documents.
type ChefTaxHandler struct{}

// NewChefTaxHandler constructs the handler.
func NewChefTaxHandler() *ChefTaxHandler {
	return &ChefTaxHandler{}
}

// GetTDSCertificate streams the annual TDS summary PDF for the requested FY.
func (h *ChefTaxHandler) GetTDSCertificate(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	currentFY := services.CurrentFinancialYearStart(time.Now())
	fyStartYear := currentFY
	if raw := c.Query("year"); raw != "" {
		n, perr := strconv.Atoi(raw)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
			return
		}
		fyStartYear = n
	}
	if fyStartYear < tdsEarliestFYStart || fyStartYear > currentFY {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("year must be between %d and %d", tdsEarliestFYStart, currentFY),
		})
		return
	}

	pdfBytes, filename, genErr := services.GenerateTDSCertificatePDF(chef.ID, fyStartYear)
	if genErr != nil {
		services.CaptureSentryError(c, genErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate certificate"})
		return
	}
	services.LogAudit(c, "chef.tax_certificate.download", "tds_certificate",
		strconv.Itoa(fyStartYear), nil, gin.H{"fyStartYear": fyStartYear})
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
