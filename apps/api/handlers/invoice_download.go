package handlers

// invoice_download.go — signed receipt-download links (#receipt mobile parity).
//
// Web downloads the PDF via an authenticated blob fetch. Mobile can't (no
// file-system module), so it opens the PDF in the in-app browser — which can't
// send the app's Bearer auth. These two endpoints bridge that:
//
//	GET /orders/:id/invoice-link   (authed)  → mint a short-lived signed URL
//	GET /invoice/:token            (public)  → the token IS the auth; serve the PDF
//
// The public route carries no session; the token is the whole authorisation, and
// it is order+user scoped, single-purpose, and minutes-lived (see
// services/invoice_download_token.go), so a leaked link can only fetch that one
// receipt and only briefly.

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// GetInvoiceDownloadLink returns a short-lived, self-authenticating URL the app
// can open in the browser to download the receipt PDF.
//
// GET /api/v1/orders/:id/invoice-link
func (h *OrderHandler) GetInvoiceDownloadLink(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderUUID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// Own-order + paid check up front, so the app gets a clean 404/400 rather than
	// a link that fails only when tapped.
	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderUUID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if !orderHasReceipt(order) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A receipt is available once payment is completed"})
		return
	}

	token, err := services.MintInvoiceDownloadToken(orderUUID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create the download link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": invoiceDownloadURL(c, token)})
}

// DownloadInvoiceByToken serves the receipt PDF for a valid signed token. PUBLIC
// — the token is the authorisation. It re-verifies ownership from the token's
// user id against the order, never trusting the id in the URL alone.
//
// GET /api/v1/invoice/:token
func (h *OrderHandler) DownloadInvoiceByToken(c *gin.Context) {
	orderID, userID, err := services.VerifyInvoiceDownloadToken(c.Param("token"))
	if err != nil {
		// One generic message — never leak whether a token is expired vs forged.
		c.JSON(http.StatusForbidden, gin.H{"error": "This download link is invalid or has expired"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if !orderHasReceipt(order) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A receipt is available once payment is completed"})
		return
	}

	pdfBytes, filename, err := services.GenerateOrderInvoicePDF(orderID)
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate receipt"})
		return
	}
	// inline so the browser renders it in place (with its own save/share), rather
	// than forcing a download the in-app browser handles awkwardly.
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// invoiceDownloadURL builds the absolute URL for the public download route from
// the incoming request, so it works whatever public host the API is served on.
func invoiceDownloadURL(c *gin.Context, token string) string {
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if c.Request.TLS == nil {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/api/v1/invoice/%s", scheme, c.Request.Host, token)
}
