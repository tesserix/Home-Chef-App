package routes

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/handlers"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// bffAuth returns a fresh BFFAuth middleware configured with the loaded HMAC
// key and freshness window pulled from config. Called from each route group
// so the closure captures the key once at boot.
func bffAuth(key []byte, window time.Duration) gin.HandlerFunc {
	return middleware.BFFAuth(middleware.BFFAuthConfig{
		HMACKey:       key,
		Window:        window,
		BFFSessionURL: config.AppConfig.BFFSessionURL,
	})
}

// bffAuthOptional is the optional-auth counterpart to bffAuth — used for
// surfaces that render publicly but enrich the response when a signed
// identity is attached (chef listings, social feed).
func bffAuthOptional(key []byte, window time.Duration) gin.HandlerFunc {
	return middleware.BFFAuthOptional(middleware.BFFAuthConfig{
		HMACKey:       key,
		Window:        window,
		BFFSessionURL: config.AppConfig.BFFSessionURL,
	})
}

// allowedOrigins resolves the CORS allowlist. In production we ONLY trust the
// CORS_ORIGINS env var (comma-separated, https only — matches what argocd
// passes through to the helm chart). In dev we keep the localhost origins
// so the SPAs can hit the API directly.
func allowedOrigins() []string {
	if env := os.Getenv("CORS_ORIGINS"); env != "" {
		var out []string
		for o := range strings.SplitSeq(env, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				out = append(out, o)
			}
		}
		return out
	}
	if config.IsProduction() {
		// Production fallback — explicit, fixed list. No localhost. No wildcards.
		return []string{
			"https://fe3dr.com",
			"https://www.fe3dr.com",
			"https://vendors.fe3dr.com",
			"https://admin.fe3dr.com",
			"https://auth.fe3dr.com",
			"https://delivery.fe3dr.com",
		}
	}
	// Dev / staging
	return []string{
		"http://localhost:5173",
		"http://localhost:3000",
		"https://fe3dr.com",
		"https://www.fe3dr.com",
		"https://vendors.fe3dr.com",
		"https://admin.fe3dr.com",
		"https://auth.fe3dr.com",
		"https://delivery.fe3dr.com",
	}
}

func SetupRouter() *gin.Engine {
	// Set Gin mode
	if config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Trust only proxies on the loopback / private mesh addresses so c.ClientIP()
	// can't be spoofed by an upstream client.
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})

	// Prometheus metrics middleware
	r.Use(middleware.PrometheusMiddleware())

	// Defense-in-depth security headers on every response.
	r.Use(middleware.SecurityHeaders())

	// CORS configuration — allowlist driven by env in prod; never wildcards
	// because AllowCredentials=true requires a specific origin.
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = allowedOrigins()
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Auth-Token", "X-Request-ID", "X-Razorpay-Signature", "Stripe-Signature", "x-jwt-claim-sub", "x-jwt-claim-tenant-id", "x-jwt-claim-tenant-slug", "x-jwt-claim-email", "x-jwt-claim-name", "x-jwt-claim-given-name", "x-jwt-claim-family-name"}
	corsConfig.ExposeHeaders = []string{"X-Request-ID", "Retry-After"}
	corsConfig.AllowCredentials = true
	corsConfig.MaxAge = 600
	r.Use(cors.New(corsConfig))

	// Shared HMAC key for verifying BFF-signed identity headers. Loaded once
	// at config.Load() so the same []byte is captured by every protected
	// route group; the matching value is wired into apps/auth-bff via the
	// same k8s ExternalSecret.
	bffKey := config.AppConfig.BFFInternalHMACKey
	bffWindow := config.AppConfig.BFFAuthTSWindow

	// Initialize handlers. Note: the legacy AuthHandler was removed in the
	// GIP migration (Task 2.6) — all auth flows now live in apps/auth-bff.
	chefHandler := handlers.NewChefHandler()
	chefEarningsHandler := handlers.NewChefEarningsHandler()
	orderHandler := handlers.NewOrderHandler()
	healthHandler := handlers.NewHealthHandler()
	uploadHandler := handlers.NewUploadHandler()
	menuHandler := handlers.NewMenuHandler()
	locationHandler := handlers.NewLocationHandler()
	reviewHandler := handlers.NewReviewHandler()
	favoriteHandler := handlers.NewFavoriteHandler()
	customerHandler := handlers.NewCustomerHandler()
	addressHandler := handlers.NewAddressHandler()
	preferenceHandler := handlers.NewPreferenceHandler()
	currencyHandler := handlers.NewCurrencyHandler()
	adminHandler := handlers.NewAdminHandler()
	approvalHandler := handlers.NewApprovalHandler()
	notificationHandler := handlers.NewNotificationHandler()
	deliveryHandler := handlers.NewDeliveryHandler()
	driverHandler := handlers.NewDriverOnboardingHandler()
	staffHandler := handlers.NewStaffHandler()
	subscriptionHandler := handlers.NewSubscriptionHandler()
	paymentHandler := handlers.NewPaymentHandler()
	promotionHandler := handlers.NewPromotionHandler()
	providerHandler := handlers.NewDeliveryProviderHandler()
	socialHandler := handlers.NewSocialHandler()
	cateringHandler := handlers.NewCateringHandler()
	supportHandler := handlers.NewSupportHandler()
	promoHandler := handlers.NewPromoHandler()
	chatHandler := handlers.NewChatHandler()
	securityHandler := handlers.NewSecurityHandler()
	platformHandler := handlers.NewPlatformHandler()
	exportsHandler := handlers.NewExportsHandler()

	// Health check endpoints
	r.GET("/health", healthHandler.Health)
	r.GET("/health/live", healthHandler.Liveness)
	r.GET("/health/ready", healthHandler.Readiness)
	r.GET("/health/stats", healthHandler.SystemStats)

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Webhook endpoints — HMAC verified, but cap inbound rate as a DoS guard.
	webhookLimit := middleware.RateLimit(20, 40) // 20 rps sustained, 40 burst per IP
	r.POST("/webhooks/razorpay", webhookLimit, paymentHandler.RazorpayWebhook)
	r.POST("/webhooks/stripe", webhookLimit, paymentHandler.StripeWebhook)
	r.POST("/webhooks/delivery/:provider", webhookLimit, providerHandler.HandleWebhook)

	// WebSocket endpoints live at top-level /ws/* so the per-portal Istio
	// VirtualServices (fe3dr.com, vendors.fe3dr.com, admin.fe3dr.com, …)
	// can route /ws/ directly to the API at the 3600s upgrade timeout,
	// bypassing the auth-bff which can't proxy WS upgrades. The BFF still
	// signs the upgrade request via its mesh path; we verify the same HMAC
	// here so a hijacked browser session can't open a socket directly.
	wsGroup := r.Group("/ws")
	wsGroup.Use(bffAuth(bffKey, bffWindow))
	{
		wsGroup.GET("/notifications", notificationHandler.StreamNotificationsWS)
		wsGroup.GET("/orders/:id/track", orderHandler.TrackOrderWS)
	}

	// Internal endpoints — invoked by apps/auth-bff to materialize user
	// rows when a new GIP identity logs in. The BFF signs every request
	// with the shared HMAC; nothing else should ever reach this group.
	internalUsersHandler := handlers.NewInternalUsersHandler(database.DB)
	internal := r.Group("/internal")
	internal.Use(bffAuth(bffKey, bffWindow))
	{
		internal.POST("/users/upsert", internalUsersHandler.Upsert)
	}

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Location reference routes (public)
		locations := v1.Group("/locations")
		{
			locations.GET("/countries", locationHandler.GetCountries)
			locations.GET("/countries/:countryCode/states", locationHandler.GetStates)
			locations.GET("/states/:stateCode/cities", locationHandler.GetCities)
			locations.GET("/cities/:cityName/postcodes", locationHandler.GetPostcodes)
			locations.GET("/postcodes/search", locationHandler.SearchPostcodes)
			// Photon-backed worldwide address autocomplete, India-filtered.
			// Complements /postcodes/search (which only knows our seeded PINs)
			// so the mobile picker can resolve any Indian street address.
			locations.GET("/autocomplete", locationHandler.AutocompleteAddresses)
		}

		// Preference options (public)
		v1.GET("/preferences", preferenceHandler.GetPreferenceOptions)

		// Public platform config — fees + operating-hours status for checkout
		v1.GET("/platform/config", platformHandler.GetPublicConfig)
		// Check if a lat/lon falls inside a configured delivery zone.
		v1.GET("/platform/zone-coverage", platformHandler.CheckZoneCoverage)

		// Currency routes (public)
		currencies := v1.Group("/currencies")
		{
			currencies.GET("", currencyHandler.ListCurrencies)
			currencies.GET("/rates", currencyHandler.GetRates)
			currencies.GET("/detect", currencyHandler.DetectCurrency)
		}

		// Tax rate lookup (public — used by checkout to preview tax line
		// before the order is created)
		taxHandler := handlers.NewTaxHandler()
		v1.GET("/tax-rates/lookup", taxHandler.GetPublicTaxRate)

		// Auth routes (public). All credential, session, OAuth, password-reset,
		// email-verification, and TOTP flows moved to apps/auth-bff in the
		// GIP migration (Task 2.6). The only thing that stays here is the
		// public password-policy lookup — the registration UI in the BFF
		// still calls into our SecurityPolicy table to render live rules.
		authLimit := middleware.RateLimit(5, 10)
		auth := v1.Group("/auth")
		auth.Use(authLimit)
		{
			auth.GET("/password-policy", securityHandler.GetPasswordPolicy)
		}

		// Staff invitation routes (public - token validates)
		v1.GET("/staff/invitations/validate", staffHandler.ValidateInvitation)

		// Staff invitation acceptance — the auth-bff session is required so
		// the invitee is bound to a real GIP identity at accept time.
		staffInvite := v1.Group("/staff/invitations")
		staffInvite.Use(bffAuth(bffKey, bffWindow))
		{
			staffInvite.POST("/accept", staffHandler.AcceptInvitation)
		}

		// Public chef routes — anonymous browsing is allowed, but a signed
		// identity from the BFF lets us surface per-user state (favorites,
		// hidden chefs, etc.) when present.
		chefs := v1.Group("/chefs")
		chefs.Use(bffAuthOptional(bffKey, bffWindow))
		{
			chefs.GET("", chefHandler.ListChefs)
			chefs.GET("/:id", chefHandler.GetChef)
			chefs.GET("/:id/menu", chefHandler.GetChefMenu)
			chefs.GET("/:id/reviews", chefHandler.GetChefReviews)
		}

		// Chef onboarding (authenticated, but no chef role required — user is becoming a chef)
		chefOnboarding := v1.Group("/chef")
		chefOnboarding.Use(bffAuth(bffKey, bffWindow))
		{
			chefOnboarding.GET("/onboarding/status", uploadHandler.GetOnboardingStatus)
		chefOnboarding.POST("/onboarding", uploadHandler.Onboarding)
			chefOnboarding.POST("/documents", uploadHandler.UploadDocument)
			chefOnboarding.GET("/documents", uploadHandler.GetDocuments)
			chefOnboarding.POST("/profile-image", uploadHandler.UploadProfileImage)
			chefOnboarding.POST("/banner-image", uploadHandler.UploadBannerImage)
			chefOnboarding.POST("/kitchen-photos", uploadHandler.UploadKitchenPhoto)
			chefOnboarding.DELETE("/kitchen-photos", uploadHandler.DeleteKitchenPhoto)
		}

		// Menu management routes (authenticated — accessible during onboarding and after)
		chefMenu := v1.Group("/chef/menu")
		chefMenu.Use(bffAuth(bffKey, bffWindow))
		{
			chefMenu.GET("", menuHandler.GetChefMenuItems)
			chefMenu.GET("/categories", menuHandler.GetCategories)
			chefMenu.POST("/categories", menuHandler.CreateCategory)
			chefMenu.PUT("/categories/:categoryId", menuHandler.UpdateCategory)
			chefMenu.DELETE("/categories/:categoryId", menuHandler.DeleteCategory)
			chefMenu.GET("/items/:itemId", menuHandler.GetMenuItem)
			chefMenu.POST("/items", menuHandler.CreateMenuItem)
			chefMenu.PUT("/items/:itemId", menuHandler.UpdateMenuItem)
			chefMenu.DELETE("/items/:itemId", menuHandler.DeleteMenuItem)
			chefMenu.POST("/items/:itemId/images", menuHandler.UploadMenuItemImage)
			chefMenu.DELETE("/items/:itemId/images/:imageId", menuHandler.DeleteMenuItemImage)
			// PUT /chef/menu/items/:itemId/availability — toggle out-of-stock per item.
			// Param name MUST match the sibling routes (:itemId) — Gin's radix
			// tree panics on conflicting wildcard names at the same position.
			chefMenu.PUT("/items/:itemId/availability", menuHandler.ToggleMenuItemAvailability)
		}

		// Chef dashboard routes (chef only)
		chefDashboard := v1.Group("/chef")
		chefDashboard.Use(bffAuth(bffKey, bffWindow), middleware.RequireChef())
		{
			chefDashboard.GET("/dashboard", chefHandler.GetChefDashboard)
			chefDashboard.GET("/profile", chefHandler.GetChefProfile)
			chefDashboard.PUT("/profile", chefHandler.UpdateChefProfile)
			chefDashboard.GET("/orders", chefHandler.GetChefOrders)
			// GET /chef/orders/:orderId — full order detail for the vendor
			chefDashboard.GET("/orders/:orderId", chefHandler.GetOrderDetail)
			chefDashboard.PUT("/orders/:orderId/status", chefHandler.UpdateOrderStatus)
			// GET /chef/earnings/breakdown?period=week|month|cycle
			chefDashboard.GET("/earnings/breakdown", chefEarningsHandler.GetEarningsBreakdown)
			// GET /chef/documents/expiring?withinDays=30
			chefDashboard.GET("/documents/expiring", uploadHandler.GetExpiringDocuments)
			chefDashboard.GET("/reviews", chefHandler.GetChefReviewsForDashboard)
			chefDashboard.POST("/reviews/:reviewId/reply", chefHandler.ReplyToReview)
			chefDashboard.GET("/settings", chefHandler.GetChefSettings)
			chefDashboard.PUT("/settings", chefHandler.UpdateChefSettings)
			chefDashboard.GET("/analytics", chefHandler.GetChefAnalytics)
			chefDashboard.GET("/payout", chefHandler.GetPayoutDetails)
			chefDashboard.POST("/payout", chefHandler.SavePayoutDetails)
			chefDashboard.GET("/admin-requests", approvalHandler.GetChefApprovalRequests)
			chefDashboard.PUT("/admin-requests/:id/respond", approvalHandler.RespondToApprovalRequest)

			// Stripe Connect onboarding (international chefs)
			stripeConnectHandler := handlers.NewStripeConnectHandler()
			chefDashboard.POST("/stripe/connect", stripeConnectHandler.CreateStripeConnectAccount)
			chefDashboard.GET("/stripe/status", stripeConnectHandler.GetStripeConnectStatus)
			chefDashboard.POST("/stripe/onboarding-link", stripeConnectHandler.RefreshStripeOnboardingLink)
			chefDashboard.PUT("/payment-provider", stripeConnectHandler.SetPaymentProvider)
		}

		// Customer order routes
		orders := v1.Group("/orders")
		orders.Use(bffAuth(bffKey, bffWindow))
		{
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("", orderHandler.GetOrders)
			orders.GET("/:id", orderHandler.GetOrder)
			orders.POST("/:id/cancel", orderHandler.CancelOrder)
			orders.GET("/:id/track", orderHandler.TrackOrder)
			orders.GET("/:id/track/ws", orderHandler.TrackOrderWS)
			orders.GET("/:id/invoice", orderHandler.GetOrderInvoice)
			// Order-specific chat rooms
			orders.GET("/:id/chat/:type", chatHandler.GetOrCreateChatRoom)
		}

		// Chat routes (authenticated)
		chat := v1.Group("/chat")
		chat.Use(bffAuth(bffKey, bffWindow))
		{
			chat.GET("/rooms", chatHandler.ListChatRooms)
			chat.GET("/:roomId/messages", chatHandler.GetMessages)
			chat.POST("/:roomId/messages", chatHandler.SendMessage)
			chat.PUT("/:roomId/messages/:messageId/read", chatHandler.MarkAsRead)
		}

		// Support ticket routes (authenticated)
		support := v1.Group("/support")
		support.Use(bffAuth(bffKey, bffWindow))
		{
			support.POST("/tickets", supportHandler.CreateTicket)
			support.GET("/tickets", supportHandler.GetMyTickets)
			support.GET("/tickets/:id", supportHandler.GetTicket)
			support.POST("/tickets/:id/messages", supportHandler.AddMessage)
			support.PUT("/tickets/:id/close", supportHandler.CloseTicket)
		}

		// Promo code validation (authenticated)
		promo := v1.Group("/promo")
		promo.Use(bffAuth(bffKey, bffWindow))
		{
			promo.POST("/validate", promoHandler.ValidatePromoCode)
		}

		// Cart routes
		cart := v1.Group("/cart")
		cart.Use(bffAuth(bffKey, bffWindow))
		{
			// cart.GET("", cartHandler.GetCart)
			// cart.POST("/items", cartHandler.AddItem)
			// cart.PUT("/items/:itemId", cartHandler.UpdateItem)
			// cart.DELETE("/items/:itemId", cartHandler.RemoveItem)
			// cart.DELETE("", cartHandler.ClearCart)
		}

		// Social feed routes — anonymous-friendly; like/comment are gated
		// inside the handlers by checking for a userID on the context.
		social := v1.Group("/social")
		social.Use(bffAuthOptional(bffKey, bffWindow))
		{
			social.GET("/feed", socialHandler.GetFeed)
			social.GET("/posts/:id", socialHandler.GetPost)
			social.POST("/posts/:id/like", socialHandler.LikePost)
			social.POST("/posts/:id/comments", socialHandler.AddComment)
		}

		// Chef social posts (chef only)
		chefSocial := v1.Group("/chef/posts")
		chefSocial.Use(bffAuth(bffKey, bffWindow), middleware.RequireChef())
		{
			chefSocial.GET("", socialHandler.GetChefPosts)
			chefSocial.POST("", socialHandler.CreatePost)
			chefSocial.PUT("/:id", socialHandler.UpdatePost)
			chefSocial.DELETE("/:id", socialHandler.DeletePost)
		}

		// Catering routes
		catering := v1.Group("/catering")
		catering.Use(bffAuth(bffKey, bffWindow))
		{
			catering.POST("/requests", cateringHandler.CreateRequest)
			catering.GET("/requests", cateringHandler.GetMyRequests)
			catering.GET("/requests/:id", cateringHandler.GetRequest)
			catering.GET("/requests/:id/quotes", cateringHandler.GetQuotes)
			catering.POST("/quotes/:id/accept", cateringHandler.AcceptQuote)
		}

		// Chef catering (chef only)
		chefCatering := v1.Group("/chef/catering")
		chefCatering.Use(bffAuth(bffKey, bffWindow), middleware.RequireChef())
		{
			chefCatering.GET("/requests", cateringHandler.GetAvailableRequests)
			chefCatering.POST("/requests/:id/quote", cateringHandler.SubmitQuote)
			chefCatering.GET("/quotes", cateringHandler.GetChefQuotes)
		}

		// Delivery partner onboarding (authenticated, no delivery role required)
		deliveryOnboarding := v1.Group("/delivery")
		deliveryOnboarding.Use(bffAuth(bffKey, bffWindow))
		{
			deliveryOnboarding.GET("/onboarding/status", deliveryHandler.GetOnboardingStatus)
			deliveryOnboarding.POST("/onboarding", deliveryHandler.Onboarding)
		}

		// Driver onboarding (authenticated, no role required — user is becoming a driver)
		driverOnboarding := v1.Group("/driver")
		driverOnboarding.Use(bffAuth(bffKey, bffWindow))
		{
			driverOnboarding.GET("/onboarding/status", driverHandler.GetDriverOnboardingStatus)
			driverOnboarding.POST("/onboarding/personal", driverHandler.DriverOnboardingPersonal)
			driverOnboarding.POST("/onboarding/vehicle", driverHandler.DriverOnboardingVehicle)
			driverOnboarding.POST("/onboarding/documents", deliveryHandler.UploadPartnerDocument)
			driverOnboarding.GET("/onboarding/documents", deliveryHandler.GetPartnerDocuments)
			driverOnboarding.POST("/onboarding/payout", driverHandler.DriverOnboardingPayout)
			driverOnboarding.POST("/onboarding/submit", driverHandler.DriverOnboardingSubmit)
			driverOnboarding.POST("/referral/validate", driverHandler.ValidateReferralCode)
			// Subscription plan selection during onboarding (no delivery role needed yet)
			driverOnboarding.GET("/subscription/plans", subscriptionHandler.GetAvailablePlans)
			driverOnboarding.POST("/subscription/plan", subscriptionHandler.ChoosePlan)
		}

		// Driver referral routes (delivery role required)
		driverReferral := v1.Group("/driver/referral")
		driverReferral.Use(bffAuth(bffKey, bffWindow), middleware.RequireDelivery())
		{
			driverReferral.GET("/code", driverHandler.GetReferralCode)
			driverReferral.GET("/stats", driverHandler.GetReferralStats)
		}

		// Delivery staff routes — enforced with granular staff permissions
		deliveryStaff := v1.Group("/delivery/staff")
		deliveryStaff.Use(bffAuth(bffKey, bffWindow), middleware.RequireDelivery())
		{
			deliveryStaff.GET("/me", staffHandler.GetMyStaffProfile)   // No permission needed — own profile
			deliveryStaff.GET("/roles", staffHandler.GetStaffRoles)    // No permission needed — role definitions
			deliveryStaff.GET("", middleware.RequireStaffPermission(models.SPViewStaff), staffHandler.ListStaff)
			deliveryStaff.POST("/invitations", middleware.RequireStaffPermission(models.SPInviteStaff), staffHandler.CreateInvitation)
			deliveryStaff.GET("/invitations", middleware.RequireStaffPermission(models.SPViewStaff), staffHandler.ListInvitations)
			deliveryStaff.PUT("/invitations/:id/revoke", middleware.RequireStaffPermission(models.SPManageStaff), staffHandler.RevokeInvitation)
			deliveryStaff.PUT("/invitations/:id/resend", middleware.RequireStaffPermission(models.SPManageStaff), staffHandler.ResendInvitation)

			// Fleet management — third-party providers
			deliveryStaff.GET("/fleet/providers", middleware.RequireStaffPermission(models.SPViewFleet), providerHandler.ListProviders)
			deliveryStaff.GET("/fleet/providers/:id", middleware.RequireStaffPermission(models.SPViewFleet), providerHandler.GetProvider)

			// Fleet management
			deliveryStaff.GET("/fleet/overview", middleware.RequireStaffPermission(models.SPViewFleet), deliveryHandler.FleetOverview)
			deliveryStaff.GET("/fleet/partners", middleware.RequireStaffPermission(models.SPViewDeliveryPartners), deliveryHandler.AdminGetDeliveryPartners)
			deliveryStaff.GET("/fleet/partners/:id", middleware.RequireStaffPermission(models.SPViewDeliveryPartners), deliveryHandler.GetPartnerDetail)
			deliveryStaff.PUT("/fleet/partners/:id/verify", middleware.RequireStaffPermission(models.SPVerifyDeliveryPartners), deliveryHandler.AdminVerifyPartner)
			deliveryStaff.PUT("/fleet/partners/:id/suspend", middleware.RequireStaffPermission(models.SPManageDeliveryPartners), deliveryHandler.AdminSuspendPartner)
			deliveryStaff.POST("/fleet/partners/:id/assign", middleware.RequireStaffPermission(models.SPAssignDeliveries), deliveryHandler.ManualAssignDelivery)
		}

		// Delivery partner routes (delivery role required)
		delivery := v1.Group("/delivery")
		delivery.Use(bffAuth(bffKey, bffWindow), middleware.RequireDelivery())
		{
			delivery.GET("/stats", deliveryHandler.GetStats)
			delivery.GET("/profile", deliveryHandler.GetProfile)
			delivery.PUT("/profile", deliveryHandler.UpdateProfile)
			delivery.PUT("/online", deliveryHandler.ToggleOnline)
			delivery.PUT("/location", deliveryHandler.UpdateLocation)
			delivery.GET("/current", deliveryHandler.GetCurrentDelivery)
			delivery.GET("/available", deliveryHandler.GetAvailableDeliveries)
			delivery.POST("/:id/accept", deliveryHandler.AcceptDelivery)
			delivery.PUT("/:id/status", deliveryHandler.UpdateDeliveryStatus)
			delivery.GET("/orders", deliveryHandler.GetDeliveryHistory)
			delivery.GET("/earnings", deliveryHandler.GetEarnings)
			delivery.POST("/documents", deliveryHandler.UploadPartnerDocument)
			delivery.GET("/documents", deliveryHandler.GetPartnerDocuments)

			// Stripe Connect onboarding for international drivers. Same
			// handler type as the chef version — different endpoints.
			driverStripeHandler := handlers.NewStripeConnectHandler()
			delivery.POST("/stripe/connect", driverStripeHandler.CreateDriverStripeAccount)
			delivery.GET("/stripe/status", driverStripeHandler.GetDriverStripeStatus)
			delivery.POST("/stripe/onboarding-link", driverStripeHandler.RefreshDriverOnboardingLink)
			delivery.PUT("/payment-provider", driverStripeHandler.SetDriverPaymentProvider)
		}

		// Chef promotion routes (featured ads)
		chefPromotion := v1.Group("/chef/promotion")
		chefPromotion.Use(bffAuth(bffKey, bffWindow), middleware.RequireChef())
		{
			chefPromotion.GET("/pricing", promotionHandler.GetFeaturedAdPricing)
			chefPromotion.POST("/purchase", promotionHandler.PurchaseFeaturedAd)
			chefPromotion.POST("/confirm", promotionHandler.ConfirmFeaturedAd)
			chefPromotion.GET("/history", promotionHandler.GetMyPromotions)
		}

		// Chef subscription routes (chef role required)
		chefSubscription := v1.Group("/chef/subscription")
		chefSubscription.Use(bffAuth(bffKey, bffWindow), middleware.RequireChef())
		{
			chefSubscription.GET("", subscriptionHandler.GetSubscription)
			chefSubscription.GET("/plans", subscriptionHandler.GetAvailablePlans)
			chefSubscription.POST("/choose-plan", subscriptionHandler.ChoosePlan)
			chefSubscription.POST("/plan", subscriptionHandler.ChoosePlan) // Alias
			chefSubscription.POST("/cancel", subscriptionHandler.CancelSubscription)
			chefSubscription.PUT("/change-plan", subscriptionHandler.ChangePlan)
			chefSubscription.GET("/invoices", subscriptionHandler.GetInvoices)
			chefSubscription.GET("/earnings", subscriptionHandler.GetEarningsSummary)
		}

		// Driver subscription routes (delivery role required — post-onboarding management)
		// Note: /plans and /plan are registered in the onboarding group above (auth-only)
		// to allow plan selection during onboarding before delivery role is assigned
		driverSubscription := v1.Group("/driver/subscription")
		driverSubscription.Use(bffAuth(bffKey, bffWindow), middleware.RequireDelivery())
		{
			driverSubscription.GET("", subscriptionHandler.GetSubscription)
			driverSubscription.POST("/choose-plan", subscriptionHandler.ChoosePlan)
			driverSubscription.POST("/cancel", subscriptionHandler.CancelSubscription)
			driverSubscription.PUT("/change-plan", subscriptionHandler.ChangePlan)
			driverSubscription.GET("/invoices", subscriptionHandler.GetInvoices)
			driverSubscription.GET("/earnings", subscriptionHandler.GetEarningsSummary)
		}

		// Payment routes (authenticated). Rate-limited per-user to defend
		// against credential-theft abuse: stolen tokens shouldn't be able to
		// fire bulk refund/verify floods even though Razorpay itself is
		// idempotent.
		paymentLimit := middleware.RateLimitByUser(2, 5) // 2 rps sustained, 5 burst per user
		orderPayments := v1.Group("/payments")
		orderPayments.Use(bffAuth(bffKey, bffWindow), paymentLimit)
		{
			orderPayments.POST("/order/:orderId/create", paymentHandler.CreateOrderPayment)
			orderPayments.POST("/order/:orderId/verify", paymentHandler.VerifyPayment)
			orderPayments.POST("/order/:orderId/refund", paymentHandler.InitiateRefund)
		}

		// Admin routes
		admin := v1.Group("/admin")
		admin.Use(bffAuth(bffKey, bffWindow), middleware.RequireAdmin())
		{
			// Dashboard
			admin.GET("/stats", adminHandler.GetStats)
			admin.GET("/activities", adminHandler.GetActivities)
			admin.GET("/analytics", adminHandler.GetAnalytics)

			// User management
			admin.GET("/users", adminHandler.GetUsers)
			admin.GET("/users/:id", adminHandler.GetUser)
			admin.PUT("/users/:id/suspend", adminHandler.SuspendUser)
			admin.PUT("/users/:id/activate", adminHandler.ActivateUser)

			// Chef management
			admin.GET("/chefs", adminHandler.GetChefs)
			admin.PUT("/chefs/:id/verify", adminHandler.VerifyChef)
			admin.PUT("/chefs/:id/reject", adminHandler.RejectChef)
			admin.PUT("/chefs/:id/suspend", adminHandler.SuspendChef)

			// Order management
			admin.GET("/orders", adminHandler.GetAllOrders)
			admin.GET("/orders/:id", adminHandler.GetOrderDetails)

			// Promotions (featured ads)
			admin.GET("/promotions", promotionHandler.AdminListPromotions)
			admin.GET("/promotions/stats", promotionHandler.AdminGetPromotionStats)

			// Delivery management
			admin.GET("/delivery/stats", deliveryHandler.AdminGetDeliveryStats)
			admin.GET("/delivery/list", deliveryHandler.AdminListDeliveries)
			admin.GET("/delivery/partners", deliveryHandler.AdminGetDeliveryPartners)
			admin.GET("/delivery/partners/:id", deliveryHandler.GetPartnerDetail)
			admin.PUT("/delivery/partners/:id/verify", deliveryHandler.AdminVerifyPartner)
			admin.PUT("/delivery/partners/:id/suspend", deliveryHandler.AdminSuspendPartner)

			// Delivery provider management
			admin.GET("/delivery/providers", providerHandler.ListProviders)
			admin.GET("/delivery/providers/:id", providerHandler.GetProvider)
			admin.POST("/delivery/providers", providerHandler.CreateProvider)
			admin.PUT("/delivery/providers/:id", providerHandler.UpdateProvider)
			admin.DELETE("/delivery/providers/:id", providerHandler.DeleteProvider)
			admin.PUT("/delivery/providers/:id/toggle", providerHandler.ToggleProvider)
			admin.POST("/delivery/providers/:id/test", providerHandler.TestConnection)
			admin.GET("/delivery/providers/:id/stats", providerHandler.GetProviderStats)

			// Delivery zone management
			admin.GET("/delivery/zones", deliveryHandler.ListZones)
			admin.POST("/delivery/zones", deliveryHandler.CreateZone)
			admin.PUT("/delivery/zones/:id", deliveryHandler.UpdateZone)
			admin.DELETE("/delivery/zones/:id", deliveryHandler.DeleteZone)

			// Staff management — enforced with granular staff permissions
			admin.GET("/staff/me", staffHandler.GetMyStaffProfile) // No permission needed — own profile
			admin.GET("/staff/roles", staffHandler.GetStaffRoles)  // No permission needed — role definitions
			admin.GET("/staff", middleware.RequireStaffPermission(models.SPViewStaff), staffHandler.ListStaff)
			admin.GET("/staff/:id", middleware.RequireStaffPermission(models.SPViewStaff), staffHandler.GetStaffMember)
			admin.PUT("/staff/:id/role", middleware.RequireStaffPermission(models.SPManageStaff), staffHandler.UpdateStaffRole)
			admin.PUT("/staff/:id/deactivate", middleware.RequireStaffPermission(models.SPManageStaff), staffHandler.DeactivateStaff)
			admin.PUT("/staff/:id/reactivate", middleware.RequireStaffPermission(models.SPManageStaff), staffHandler.ReactivateStaff)
			admin.POST("/staff/invitations", middleware.RequireStaffPermission(models.SPInviteStaff), staffHandler.CreateInvitation)
			admin.GET("/staff/invitations", middleware.RequireStaffPermission(models.SPViewStaff), staffHandler.ListInvitations)
			admin.PUT("/staff/invitations/:id/revoke", middleware.RequireStaffPermission(models.SPManageStaff), staffHandler.RevokeInvitation)
			admin.PUT("/staff/invitations/:id/resend", middleware.RequireStaffPermission(models.SPManageStaff), staffHandler.ResendInvitation)

			// Payment gateway — Razorpay (India)
			admin.GET("/payment-gateway/status", adminHandler.GetPaymentGatewayStatus)
			admin.PUT("/payment-gateway/keys", adminHandler.UpdatePaymentGatewayKeys)

			// Payment gateway — Stripe (international)
			admin.GET("/payment-gateway/stripe/status", adminHandler.GetStripeGatewayStatus)
			admin.PUT("/payment-gateway/stripe/keys", adminHandler.UpdateStripeGatewayKeys)

			// Tax rule CRUD — per-country, optional per-region rules used
			// by the order pipeline and rendered on the invoice.
			admin.GET("/tax-rates", taxHandler.AdminListTaxRates)
			admin.POST("/tax-rates", taxHandler.AdminUpsertTaxRate)
			admin.DELETE("/tax-rates/:id", taxHandler.AdminDeleteTaxRate)

			// Security policy + API keys
			admin.GET("/security/policy", securityHandler.AdminGetSecurityPolicy)
			admin.PUT("/security/policy", securityHandler.AdminUpdateSecurityPolicy)
			admin.GET("/security/api-keys", securityHandler.AdminListApiKeys)
			admin.POST("/security/api-keys", securityHandler.AdminCreateApiKey)
			admin.DELETE("/security/api-keys/:id", securityHandler.AdminRevokeApiKey)

			// Platform policy — commission, delivery fees, operating hours
			admin.GET("/platform/policy", platformHandler.AdminGetPolicy)
			admin.PUT("/platform/policy", platformHandler.AdminUpdatePolicy)

			// CSV exports for reporting / data review
			admin.GET("/exports/users.csv", exportsHandler.ExportUsers)
			admin.GET("/exports/orders.csv", exportsHandler.ExportOrders)
			admin.GET("/exports/revenue.csv", exportsHandler.ExportRevenue)

			// Audit log viewer
			admin.GET("/audit-logs", securityHandler.AdminListAuditLogs)

			// Settings
			admin.GET("/settings", adminHandler.GetSettings)
			admin.PUT("/settings", adminHandler.UpdateSettings)

			// Approval/Review workflow
			admin.GET("/approvals", approvalHandler.GetApprovalRequests)
			admin.GET("/approvals/counts", approvalHandler.GetApprovalCounts)
			admin.GET("/approvals/:id", approvalHandler.GetApprovalRequest)
			admin.PUT("/approvals/:id/approve", approvalHandler.ApproveRequest)
			admin.PUT("/approvals/:id/reject", approvalHandler.RejectRequest)
			admin.PUT("/approvals/:id/request-info", approvalHandler.RequestMoreInfo)
			admin.GET("/approvals/:id/history", approvalHandler.GetApprovalHistory)
			admin.GET("/approvals/:id/documents/:docId", approvalHandler.GetDocumentDownload)

			// Support ticket management
			admin.GET("/support/tickets", supportHandler.AdminGetTickets)
			admin.GET("/support/tickets/:id", supportHandler.AdminGetTicket)
			admin.PUT("/support/tickets/:id/assign", supportHandler.AdminAssignTicket)
			admin.PUT("/support/tickets/:id/status", supportHandler.AdminUpdateTicketStatus)
			admin.POST("/support/tickets/:id/messages", supportHandler.AdminAddMessage)
			admin.GET("/support/stats", supportHandler.AdminGetSupportStats)

			// Promo code management
			admin.GET("/promos", promoHandler.AdminListPromos)
			admin.POST("/promos", promoHandler.AdminCreatePromo)
			admin.PUT("/promos/:id", promoHandler.AdminUpdatePromo)
			admin.DELETE("/promos/:id", promoHandler.AdminDeletePromo)
			admin.GET("/promos/:id/usage", promoHandler.AdminGetPromoUsage)

			// Subscription management
			admin.GET("/subscriptions", subscriptionHandler.AdminGetSubscriptions)
			admin.GET("/subscriptions/stats", subscriptionHandler.AdminGetSubscriptionStats)
			admin.GET("/subscriptions/:id", subscriptionHandler.AdminGetSubscription)
			admin.GET("/invoices", subscriptionHandler.AdminListInvoices)
		}

		// Addresses
		addresses := v1.Group("/addresses")
		addresses.Use(bffAuth(bffKey, bffWindow))
		{
			addresses.GET("", addressHandler.GetAddresses)
			addresses.POST("", addressHandler.CreateAddress)
			addresses.PUT("/:id", addressHandler.UpdateAddress)
			addresses.DELETE("/:id", addressHandler.DeleteAddress)
		}

		// Payment methods
		payments := v1.Group("/payment-methods")
		payments.Use(bffAuth(bffKey, bffWindow))
		{
			// payments.GET("", paymentHandler.GetPaymentMethods)
			// payments.POST("", paymentHandler.AddPaymentMethod)
			// payments.DELETE("/:id", paymentHandler.RemovePaymentMethod)
			// payments.PUT("/:id/default", paymentHandler.SetDefaultPaymentMethod)
		}

		// Customer profile & onboarding
		customer := v1.Group("/customer")
		customer.Use(bffAuth(bffKey, bffWindow))
		{
			customer.GET("/profile", customerHandler.GetCustomerProfile)
			customer.PUT("/profile", customerHandler.UpdateCustomerProfile)
			customer.POST("/avatar", customerHandler.UploadAvatar)
			customer.PUT("/currency", currencyHandler.SetPreferredCurrency)
			customer.GET("/onboarding/status", customerHandler.GetOnboardingStatus)
			customer.POST("/onboarding/complete", customerHandler.CompleteOnboarding)
			customer.POST("/onboarding/skip", customerHandler.SkipOnboarding)
		}

		// Reviews (authenticated customers)
		reviews := v1.Group("/reviews")
		reviews.Use(bffAuth(bffKey, bffWindow))
		{
			reviews.POST("", reviewHandler.CreateReview)
		}

		// Favorites (authenticated customers)
		favorites := v1.Group("/favorites")
		favorites.Use(bffAuth(bffKey, bffWindow))
		{
			favorites.GET("/chefs", favoriteHandler.ListFavoriteChefs)
			favorites.GET("/chefs/ids", favoriteHandler.ListFavoriteChefIDs)
			favorites.POST("/chefs", favoriteHandler.AddFavoriteChef)
			favorites.DELETE("/chefs/:chefId", favoriteHandler.RemoveFavoriteChef)
		}

		// Notifications
		notifications := v1.Group("/notifications")
		notifications.Use(bffAuth(bffKey, bffWindow))
		{
			notifications.GET("", notificationHandler.GetNotifications)
			notifications.GET("/unread-count", notificationHandler.GetUnreadCount)
			notifications.GET("/ws", notificationHandler.StreamNotificationsWS)
			notifications.PUT("/:id/read", notificationHandler.MarkAsRead)
			notifications.PUT("/read-all", notificationHandler.MarkAllAsRead)
			// Per-user opt-in/out by category — dispatch checks these
			// before email/push so users can mute non-critical channels.
			notifications.GET("/preferences", notificationHandler.GetPreferences)
			notifications.PUT("/preferences", notificationHandler.UpdatePreference)
		}
	}

	return r
}

// RoleToPermissions returns permissions for debugging
func RoleToPermissions(role models.UserRole) []middleware.Permission {
	return middleware.RolePermissions[role]
}
