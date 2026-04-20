package handlers

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// StripeConnectHandler hosts the chef-facing endpoints for Stripe Connect
// onboarding. Kept separate from the main ChefHandler so the Razorpay-era
// payout endpoints (GetPayoutDetails / SavePayoutDetails) don't grow a mix
// of India-specific and international-specific concerns.
type StripeConnectHandler struct{}

func NewStripeConnectHandler() *StripeConnectHandler {
	return &StripeConnectHandler{}
}

// vendorPortalBaseURL returns the URL Stripe redirects the chef back to
// after onboarding completes (or needs a fresh link). Configurable via
// VENDOR_PORTAL_BASE_URL so staging/dev work too; defaults to production.
func vendorPortalBaseURL() string {
	if v := os.Getenv("VENDOR_PORTAL_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://vendors.fe3dr.com"
}

// CreateStripeConnectAccount creates (or reuses) a Connect account for the
// chef and returns a one-time onboarding URL. The chef opens that URL,
// finishes KYC on Stripe's hosted pages, and is redirected back.
//
// POST /chef/stripe/connect
// Body: { "country": "US" }   // ISO-3166 alpha-2, optional (defaults to chef's PayoutCountry)
func (h *StripeConnectHandler) CreateStripeConnectAccount(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	st := services.GetStripe()
	if st == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Stripe gateway not configured by platform admin"})
		return
	}

	var req struct {
		Country string `json:"country"`
	}
	_ = c.ShouldBindJSON(&req) // body is optional

	var chef models.ChefProfile
	if err := database.DB.Preload("User").Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	country := strings.ToUpper(req.Country)
	if country == "" {
		country = strings.ToUpper(chef.PayoutCountry)
	}
	if country == "" {
		country = "US"
	}

	// Reuse an existing Connect account if the chef has already started
	// onboarding — Stripe rejects a second "Create Account" on the same
	// chef anyway, so short-circuit here and just mint a fresh link.
	accountID := chef.StripeAccountID
	if accountID == "" {
		acct, err := st.CreateConnectAccount(&services.StripeConnectAccountRequest{
			Type:    "express",
			Country: country,
			Email:   chef.User.Email,
		})
		if err != nil {
			log.Printf("Failed to create Stripe Connect account for chef %s: %v", chef.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Connect account"})
			return
		}
		accountID = acct.ID

		database.DB.Model(&chef).Updates(map[string]interface{}{
			"stripe_account_id": accountID,
			"payout_country":    country,
			"payment_provider":  "stripe",
		})
	}

	link, err := st.CreateAccountLink(
		accountID,
		vendorPortalBaseURL()+"/settings/stripe?refresh=1",
		vendorPortalBaseURL()+"/settings/stripe?done=1",
	)
	if err != nil {
		log.Printf("Failed to create Stripe account link for chef %s: %v", chef.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate onboarding link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accountId":     accountID,
		"onboardingUrl": link.URL,
		"expiresAt":     link.ExpiresAt,
		"country":       country,
	})
}

// GetStripeConnectStatus fetches live capability state from Stripe and
// returns a compact status the vendor portal can render ("onboarding
// complete", "action required", "not started"). Also refreshes the cached
// `stripe_charges_enabled` / `stripe_payouts_enabled` flags on the chef
// profile so order creation's cached check stays current.
//
// GET /chef/stripe/status
func (h *StripeConnectHandler) GetStripeConnectStatus(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	if chef.StripeAccountID == "" {
		c.JSON(http.StatusOK, gin.H{
			"connected":        false,
			"accountId":        "",
			"chargesEnabled":   false,
			"payoutsEnabled":   false,
			"detailsSubmitted": false,
			"country":          chef.PayoutCountry,
			"paymentProvider":  chef.PaymentProvider,
		})
		return
	}

	st := services.GetStripe()
	if st == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Stripe gateway not configured"})
		return
	}

	acct, err := st.FetchConnectAccount(chef.StripeAccountID)
	if err != nil {
		log.Printf("Failed to fetch Stripe account %s: %v", chef.StripeAccountID, err)
		c.JSON(http.StatusOK, gin.H{
			"connected":        true,
			"accountId":        chef.StripeAccountID,
			"chargesEnabled":   chef.StripeChargesEnabled,
			"payoutsEnabled":   chef.StripePayoutsEnabled,
			"detailsSubmitted": false,
			"country":          chef.PayoutCountry,
			"paymentProvider":  chef.PaymentProvider,
			"warning":          "Stripe API unreachable — showing cached status",
		})
		return
	}

	// Keep the cached flags in sync so CreateOrderPayment can trust them
	// without making a Stripe call on every order.
	database.DB.Model(&chef).Updates(map[string]interface{}{
		"stripe_charges_enabled": acct.Charges,
		"stripe_payouts_enabled": acct.Payouts,
	})

	c.JSON(http.StatusOK, gin.H{
		"connected":        true,
		"accountId":        acct.ID,
		"chargesEnabled":   acct.Charges,
		"payoutsEnabled":   acct.Payouts,
		"detailsSubmitted": acct.Details,
		"country":          acct.Country,
		"paymentProvider":  chef.PaymentProvider,
	})
}

// RefreshStripeOnboardingLink mints a new account-link URL for a chef who
// already has an account but needs to resume onboarding (e.g. Stripe
// flagged extra verification). Separate endpoint so the vendor-portal
// doesn't have to POST the connect-creation endpoint just to re-link.
//
// POST /chef/stripe/onboarding-link
func (h *StripeConnectHandler) RefreshStripeOnboardingLink(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}
	if chef.StripeAccountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No Stripe account — call /chef/stripe/connect first"})
		return
	}

	st := services.GetStripe()
	if st == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Stripe gateway not configured"})
		return
	}

	link, err := st.CreateAccountLink(
		chef.StripeAccountID,
		vendorPortalBaseURL()+"/settings/stripe?refresh=1",
		vendorPortalBaseURL()+"/settings/stripe?done=1",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate onboarding link"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"onboardingUrl": link.URL,
		"expiresAt":     link.ExpiresAt,
	})
}

// SetPaymentProvider lets a chef toggle which gateway their orders settle
// through. Used when a chef switches regions or when the platform wants to
// migrate a chef between providers. The chef must have completed onboarding
// with the target provider (Razorpay account ID or Stripe account ID
// non-empty) before switching to it.
//
// PUT /chef/payment-provider
// Body: { "provider": "stripe" | "razorpay" }
func (h *StripeConnectHandler) SetPaymentProvider(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		Provider string `json:"provider" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Provider = strings.ToLower(req.Provider)
	if req.Provider != "razorpay" && req.Provider != "stripe" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider must be 'razorpay' or 'stripe'"})
		return
	}

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	if req.Provider == "stripe" && chef.StripeAccountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Complete Stripe onboarding before switching to Stripe"})
		return
	}
	if req.Provider == "razorpay" && chef.RazorpayAccountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Complete Razorpay payout setup before switching to Razorpay"})
		return
	}

	database.DB.Model(&chef).Update("payment_provider", req.Provider)

	c.JSON(http.StatusOK, gin.H{
		"paymentProvider": req.Provider,
	})
}
