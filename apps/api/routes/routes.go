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
	"github.com/homechef/api/services"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
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

	// gin.New() (not Default) so we own the middleware chain: gin's text
	// logger is replaced by middleware.RequestLogger (structured JSON), and
	// gin.Recovery stays as the outermost panic backstop.
	r := gin.New()
	r.Use(gin.Recovery())

	// Trust only proxies on the loopback / private mesh addresses so c.ClientIP()
	// can't be spoofed by an upstream client.
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})

	// Correlation ID first — mints/propagates X-Request-ID so every
	// downstream log line, trace span, and audit row shares one id.
	r.Use(middleware.RequestID())

	// OpenTelemetry span per request (extracts inbound traceparent), then
	// bridge the trace id into the logging/audit context. No-ops cleanly
	// when tracing is disabled (no GCP project / creds).
	r.Use(otelgin.Middleware("homechef-api"))
	r.Use(middleware.TraceContext())

	// Structured access log (JSON) carrying the correlation id + trace id.
	r.Use(middleware.RequestLogger())

	// Sentry — recovers panics + tags them onto the per-request Hub so
	// handlers can attach user context. Comes before the business
	// middleware so anything that panics downstream lands in Sentry.
	// No-ops cleanly when SENTRY_DSN_API is unset.
	r.Use(services.SentryGinMiddleware())

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
	chefStatementsHandler := handlers.NewChefStatementsHandler()
	chefTaxHandler := handlers.NewChefTaxHandler()
	chefRefundsHandler := handlers.NewChefRefundsHandler()
	chefAvailabilityHandler := handlers.NewChefAvailabilityHandler()
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
	dietaryHandler := handlers.NewDietaryHandler()
	currencyHandler := handlers.NewCurrencyHandler()
	adminHandler := handlers.NewAdminHandler()
	approvalHandler := handlers.NewApprovalHandler()
	notificationHandler := handlers.NewNotificationHandler()
	walletHandler := handlers.NewWalletHandler()
	loyaltyHandler := handlers.NewLoyaltyHandler()
	referralHandler := handlers.NewReferralHandler()
	winbackHandler := handlers.NewWinbackHandler()
	orderIssueHandler := handlers.NewOrderIssueHandler()
	payoutHoldHandler := handlers.NewPayoutHoldHandler()
	deliveryHandler := handlers.NewDeliveryHandler()
	staffHandler := handlers.NewStaffHandler()
	subscriptionHandler := handlers.NewSubscriptionHandler()
	mealSubHandler := handlers.NewMealSubscriptionHandler()
	paymentHandler := handlers.NewPaymentHandler()
	tipHandler := handlers.NewTipHandler()
	groupOrderHandler := handlers.NewGroupOrderHandler()
	promotionHandler := handlers.NewPromotionHandler()
	providerHandler := handlers.NewDeliveryProviderHandler()
	socialHandler := handlers.NewSocialHandler()
	cateringHandler := handlers.NewCateringHandler()
	mealPlanHandler := handlers.NewMealPlanHandler()
	supportHandler := handlers.NewSupportHandler()
	promoHandler := handlers.NewPromoHandler()
	chatHandler := handlers.NewChatHandler()
	messagingHandler := handlers.NewMessagingHandler()
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

	// Same webhooks under /api/* too. Publicly, only /api and /ws are routed
	// to this service (per-host Istio VirtualServices + Cloudflare); the
	// root /webhooks/* paths fall through to the web frontend and 404. So the
	// externally-registered webhook URLs use the /api form
	// (e.g. https://fe3dr.com/api/webhooks/razorpay).
	r.POST("/api/webhooks/razorpay", webhookLimit, paymentHandler.RazorpayWebhook)
	r.POST("/api/webhooks/stripe", webhookLimit, paymentHandler.StripeWebhook)
	r.POST("/api/webhooks/delivery/:provider", webhookLimit, providerHandler.HandleWebhook)

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
	// Mobile force-upgrade gate. No-op for any request missing
	// X-App-Version / X-Platform (i.e. web + admin + delivery portals,
	// curl, internal tooling all pass through untouched). Mobile sets
	// both, so a too-old build receives 426 + storeUrl on every API
	// call as a defense-in-depth backstop to the explicit min-version
	// poll the client does on focus. Excludes the min-version endpoint
	// itself (would be circular) and webhook routes (no version
	// header, but already gated by HMAC).
	v1.Use(middleware.VersionCheck([]string{
		"/api/v1/mobile/min-version",
	}))
	// Redis-backed rate limit on all of v1. 60req/min per
	// authenticated user (chef), 30req/min per IP otherwise. Excluded:
	// the min-version poll (mobile hits it every 30 min — but other
	// clients might too, so don't count it against budgets). Fails
	// open if Redis is unreachable — see services/redis.go +
	// middleware/ratelimit.go for the fail-mode rationale.
	v1.Use(middleware.RateLimitRedis(middleware.RateLimitRedisConfig{
		AuthedPerMin:   60,
		UnauthedPerMin: 30,
		ExcludedPaths: []string{
			"/api/v1/mobile/min-version",
		},
	}))
	// Idempotency-Key dedup on chef-side mutations only. Opt-in via
	// the Idempotency-Key request header (web doesn't send it today;
	// mobile will start sending it on the retry-prone flows — accept
	// order, place order, file payout, upload doc). Non-listed paths
	// and missing headers pass through. 24h TTL; 5xx responses are
	// NOT cached so a retry can re-execute. Fails open on Redis down.
	v1.Use(middleware.Idempotency(middleware.IdempotencyConfig{
		IncludedPathPrefixes: []string{
			"/api/v1/chef/orders",
			"/api/v1/chef/menu",
			"/api/v1/chef/documents",
			"/api/v1/chef/payments",
		},
		ResponseTTL: 24 * time.Hour,
	}))
	{
		// Mobile-only routes (public — no auth)
		mobileHandler := handlers.NewMobileHandler()
		v1.GET("/mobile/min-version", mobileHandler.GetMinVersion)

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

		// Marketing campaign tracking (#56) — public: email open pixel + one-click
		// unsubscribe, hit directly by the recipient's email client.
		campaignHandler := handlers.NewCampaignHandler()
		v1.GET("/campaigns/track/open/:id", campaignHandler.TrackOpen)
		v1.GET("/campaigns/unsubscribe/:id", campaignHandler.Unsubscribe)

		// Dietary & allergen taxonomy (public) + per-cart conflict check (#41).
		v1.GET("/dietary/options", dietaryHandler.GetDietaryOptions)
		dietary := v1.Group("/dietary")
		dietary.Use(bffAuth(bffKey, bffWindow))
		{
			dietary.POST("/check", dietaryHandler.CheckDietary)
		}

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
			chefs.GET("/:id/weekly-menu", chefHandler.GetPublicWeeklyMenu)     // #192 tiffin menu
			chefs.GET("/:id/daily-menu", chefHandler.GetPublicDailyMenu)       // #405 per-date menu
			chefs.GET("/:id/subscription", mealSubHandler.GetChefOffer)        // #280 tiffin offer
			chefs.GET("/:id/delivery-slots", chefHandler.GetChefDeliverySlots) // #51 scheduled slots
		}

		// Dish search across chefs (#36) — its own path so it doesn't collide
		// with the /chefs/:id param route.
		search := v1.Group("/search")
		search.Use(bffAuthOptional(bffKey, bffWindow))
		{
			search.GET("/dishes", chefHandler.SearchDishes)
		}

		// Chef onboarding (authenticated, but no chef role required — user is becoming a chef)
		chefOnboarding := v1.Group("/chef")
		chefOnboarding.Use(bffAuth(bffKey, bffWindow))
		{
			chefOnboarding.GET("/onboarding/status", uploadHandler.GetOnboardingStatus)
			chefOnboarding.POST("/onboarding", uploadHandler.Onboarding)
			chefOnboarding.POST("/documents", uploadHandler.UploadDocument)
			// Wave 2: chef-side doc renewal. Existing doc by ID, swap file
			// + (optional) expiry, reset status to pending, fresh approval
			// request created so admins re-verify.
			chefOnboarding.POST("/documents/:id/replace", uploadHandler.ReplaceDocument)
			chefOnboarding.GET("/documents", uploadHandler.GetDocuments)
			chefOnboarding.POST("/profile-image", uploadHandler.UploadProfileImage)
			chefOnboarding.POST("/banner-image", uploadHandler.UploadBannerImage)
			// OCR a document image → detected FSSAI number + expiry (pre-fill).
			chefOnboarding.POST("/documents/ocr", uploadHandler.OCRDocument)
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
			// Per-item daily capacity cap (#48).
			chefMenu.PUT("/items/:itemId/capacity", menuHandler.SetMenuItemCapacity)
		}

		// Chef dashboard routes (chef only)
		chefDashboard := v1.Group("/chef")
		chefDashboard.Use(bffAuth(bffKey, bffWindow), middleware.RequireChef())
		{
			chefDashboard.GET("/dashboard", chefHandler.GetChefDashboard)
			chefDashboard.GET("/profile", chefHandler.GetChefProfile)
			chefDashboard.PUT("/profile", chefHandler.UpdateChefProfile)
			// Tiffin meal-subscription offer config (#4/#280).
			chefDashboard.GET("/subscription-config", mealSubHandler.GetChefSubscriptionConfig)
			chefDashboard.PUT("/subscription-config", mealSubHandler.UpdateChefSubscriptionConfig)
			// Weekly fixed menu (#192) — veg/nonveg dish per (day × slot).
			chefDashboard.GET("/weekly-menu", chefHandler.GetMyWeeklyMenu)
			chefDashboard.PUT("/weekly-menu", chefHandler.PutWeeklyMenu)
			// Per-date dynamic menu (#405) — multiple dishes per (date × slot).
			chefDashboard.GET("/daily-menu", chefHandler.GetMyDailyMenu)
			chefDashboard.PUT("/daily-menu/:date", chefHandler.PutDailyMenu)
			chefDashboard.GET("/orders", chefHandler.GetChefOrders)
			// GET /chef/orders/:orderId — full order detail for the vendor
			chefDashboard.GET("/orders/:orderId", chefHandler.GetOrderDetail)
			chefDashboard.PUT("/orders/:orderId/status", chefHandler.UpdateOrderStatus)
			// Lifecycle photos (food-ready, proof-of-handover) — required by the
			// vendor app before the matching status transition.
			chefDashboard.POST("/orders/:orderId/photos", chefHandler.UploadOrderPhoto)
			// In-app messaging (#53) — admin-mediated, order-scoped.
			chefDashboard.POST("/orders/:orderId/messages", messagingHandler.ChefSendMessage)
			chefDashboard.GET("/orders/:orderId/messages", messagingHandler.ChefListMessages)
			chefDashboard.POST("/orders/:orderId/attachments", messagingHandler.ChefUploadAttachment)
			// Chef-side cancellation (Wave 2). Whole-order issues a full
			// Razorpay refund; per-line issues a partial and recomputes
			// the order totals so the remaining items continue prep.
			chefOrderCancelHandler := handlers.NewChefOrderCancelHandler()
			chefDashboard.POST("/orders/:orderId/cancel", chefOrderCancelHandler.CancelOrder)
			chefDashboard.POST("/orders/:orderId/items/:itemId/cancel", chefOrderCancelHandler.CancelOrderItem)
			// Post-delivery refund — partial or full goodwill remedy on
			// an already-delivered order. Distinct from cancel.
			chefDashboard.POST("/orders/:orderId/refund", chefOrderCancelHandler.RefundOrder)
			// Chef can download the same GSTIN tax invoice the customer
			// gets — used for their own bookkeeping. Wave 3 §invoicing.
			chefDashboard.GET("/orders/:orderId/invoice.pdf", chefOrderCancelHandler.GetOrderInvoicePDF)
			// GET /chef/earnings/breakdown?period=week|month|cycle
			chefDashboard.GET("/earnings/breakdown", chefEarningsHandler.GetEarningsBreakdown)
			// Weekly settlement statements — list + per-statement PDF.
			chefDashboard.GET("/statements/weekly", chefStatementsHandler.GetWeeklyStatements)
			chefDashboard.GET("/statements/:id/statement.pdf", chefStatementsHandler.GetWeeklyStatementPDF)
			// Annual TDS summary (Form 16A style) — ?year=FY-start.
			chefDashboard.GET("/tax/certificate", chefTaxHandler.GetTDSCertificate)
			// Refund history — one entry per refunded order, item breakdown.
			chefDashboard.GET("/refunds", chefRefundsHandler.GetRefunds)
			// Timed pause ("Back in {15,30,60} min") + manual resume.
			chefDashboard.POST("/availability/pause", chefAvailabilityHandler.PauseReceiving)
			chefDashboard.POST("/availability/resume", chefAvailabilityHandler.ResumeReceiving)
			// GET /chef/documents/expiring?withinDays=30
			chefDashboard.GET("/documents/expiring", uploadHandler.GetExpiringDocuments)
			chefDashboard.GET("/reviews", chefHandler.GetChefReviewsForDashboard)
			// Aggregate stats for the reviews header — averageRating across
			// ALL reviews, total count, star distribution.
			chefDashboard.GET("/reviews/summary", chefHandler.GetChefReviewsSummary)
			chefDashboard.POST("/reviews/:reviewId/reply", chefHandler.ReplyToReview)
			chefDashboard.GET("/settings", chefHandler.GetChefSettings)
			chefDashboard.PUT("/settings", chefHandler.UpdateChefSettings)
			chefDashboard.GET("/analytics", chefHandler.GetChefAnalytics)
			chefDashboard.GET("/analytics/subscriptions", chefHandler.GetSubscriptionMetrics) // #229
			chefDashboard.GET("/analytics/forecast", chefHandler.GetDemandForecast)           // #230
			chefDashboard.GET("/analytics/advanced", chefHandler.GetAdvancedAnalytics)        // #44 premium-gated
			chefDashboard.GET("/payout", chefHandler.GetPayoutDetails)
			chefDashboard.POST("/payout", chefHandler.SavePayoutDetails)
			chefDashboard.GET("/admin-requests", approvalHandler.GetChefApprovalRequests)
			chefDashboard.PUT("/admin-requests/:id/respond", approvalHandler.RespondToApprovalRequest)

			// Post-delivery tips received (#45)
			chefDashboard.GET("/tips", tipHandler.GetChefTips)

			// Capacity & cutoff controls (#48)
			chefDashboard.GET("/capacity-settings", chefHandler.GetChefCapacitySettings)
			chefDashboard.PUT("/capacity-settings", chefHandler.UpdateChefCapacitySettings)

			// Wave 2: chef-side notification gating. GET returns defaults
			// when no row exists; PUT upserts and reconciles FCM topic
			// subscriptions in the background.
			chefNotifPrefsHandler := handlers.NewChefNotificationPreferencesHandler()
			chefDashboard.GET("/notification-preferences", chefNotifPrefsHandler.GetPreferences)
			chefDashboard.PUT("/notification-preferences", chefNotifPrefsHandler.UpdatePreferences)

			// Wave 3 DPDP Act 2023 data-subject endpoints. /me/export
			// returns a JSON dump of all rows tied to the chef; /me/delete
			// soft-deletes + queues hard-delete after the retention window.
			chefDPDPHandler := handlers.NewChefDPDPHandler()
			chefDashboard.GET("/me/export", chefDPDPHandler.ExportMyData)
			chefDashboard.POST("/me/delete", chefDPDPHandler.DeleteMyAccount)

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
			orders.POST("/:id/reorder", orderHandler.ReorderOrder)          // #238
			orders.POST("/:id/report-issue", orderIssueHandler.ReportIssue)              // #37
			orders.GET("/:id/issues", orderIssueHandler.GetMyOrderIssues)                // #37
			orders.POST("/:id/confirm-received", payoutHoldHandler.ConfirmOrderReceived) // #387
			orders.GET("/:id/track", orderHandler.TrackOrder)
			orders.GET("/:id/track/ws", orderHandler.TrackOrderWS)
			orders.GET("/:id/invoice", orderHandler.GetOrderInvoice)
			// PDF tax invoice — customer-facing. Streams directly back as
			// application/pdf with Content-Disposition: attachment.
			orders.GET("/:id/invoice.pdf", orderHandler.GetOrderInvoicePDF)
			// Order-specific chat rooms
			orders.GET("/:id/chat/:type", chatHandler.GetOrCreateChatRoom)
		}

		// Customer tiffin meal subscriptions (#2/#3/#280).
		mealSubs := v1.Group("/meal-subscriptions")
		mealSubs.Use(bffAuth(bffKey, bffWindow))
		{
			mealSubs.POST("/preview", mealSubHandler.PreviewPrice)
			mealSubs.POST("", mealSubHandler.Subscribe)
			mealSubs.GET("", mealSubHandler.GetMySubscriptions)
			mealSubs.GET("/:id/fulfillments", mealSubHandler.GetFulfillments)
			mealSubs.POST("/:id/pause", mealSubHandler.Pause)
			mealSubs.POST("/:id/resume", mealSubHandler.Resume)
			mealSubs.POST("/:id/skip", mealSubHandler.Skip)
			mealSubs.POST("/:id/cancel", mealSubHandler.Cancel)
		}

		// Shared authenticated profile routes (any logged-in role). The mobile
		// apps register their FCM push token here after login; without this the
		// server has no token and every push is skipped.
		profile := v1.Group("/profile")
		profile.Use(bffAuth(bffKey, bffWindow))
		{
			deviceTokenHandler := handlers.NewDeviceTokenHandler()
			profile.PUT("/device-token", deviceTokenHandler.UpdateDeviceToken)
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

		// Mediated-chat attachment download (#53/#304) — separate prefix so it
		// doesn't collide with the legacy /chat/:roomId wildcard. Any
		// authenticated participant; authorization is enforced in the handler.
		chatAttachments := v1.Group("/chat-attachments")
		chatAttachments.Use(bffAuth(bffKey, bffWindow))
		{
			chatAttachments.GET("/:id", messagingHandler.DownloadAttachment)
		}

		// Support ticket routes (authenticated)
		support := v1.Group("/support")
		support.Use(bffAuth(bffKey, bffWindow))
		{
			// Conservative per-user throttle on the write paths so a single
			// account can't flood ticket creation / replies.
			supportLimit := middleware.RateLimitByUser(1, 3) // 1 rps sustained, 3 burst per user
			support.POST("/tickets", supportLimit, supportHandler.CreateTicket)
			support.GET("/tickets", supportHandler.GetMyTickets)
			support.GET("/tickets/:id", supportHandler.GetTicket)
			support.POST("/tickets/:id/messages", supportLimit, supportHandler.AddMessage)
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
			catering.POST("/requests/:id/cancel", cateringHandler.CancelRequest)
			catering.POST("/quotes/:id/accept", cateringHandler.AcceptQuote)
			catering.POST("/quotes/:id/decline", cateringHandler.DeclineQuote)
			// Deposit / advance payment (#55) — flag-gated (CATERING_DEPOSIT_ENABLED).
			catering.POST("/requests/:id/deposit", cateringHandler.CreateDeposit)
			catering.POST("/requests/:id/deposit/verify", cateringHandler.VerifyDeposit)
		}

		// Chef catering (chef only)
		chefCatering := v1.Group("/chef/catering")
		chefCatering.Use(bffAuth(bffKey, bffWindow), middleware.RequireChef())
		{
			chefCatering.GET("/requests", cateringHandler.GetAvailableRequests)
			chefCatering.POST("/requests/:id/quote", cateringHandler.SubmitQuote)
			chefCatering.GET("/quotes", cateringHandler.GetChefQuotes)
			chefCatering.GET("/bookings", cateringHandler.GetChefBookings)
			chefCatering.POST("/requests/:id/complete", cateringHandler.CompleteBooking)
		}

		// Tiffin meal plans — customer side (#195/#196). Scoped to the authed
		// customer inside the handlers (per-customer isolation).
		mealPlans := v1.Group("/meal-plans")
		mealPlans.Use(bffAuth(bffKey, bffWindow))
		{
			mealPlans.POST("", mealPlanHandler.CreateMealPlan)
			mealPlans.GET("", mealPlanHandler.GetMyMealPlans)
			mealPlans.GET("/:id", mealPlanHandler.GetMealPlan)
			mealPlans.PUT("/:id/approve", mealPlanHandler.ApproveMealPlan)
			mealPlans.PUT("/:id/reject", mealPlanHandler.RejectMealPlan)
			mealPlans.PUT("/:id/days/:dayId/skip", mealPlanHandler.SkipMealPlanDay)
			mealPlans.POST("/:id/days/:dayId/confirm-received", payoutHoldHandler.ConfirmMealPlanDayReceived) // #387
			mealPlans.POST("/:id/verify-payment", mealPlanHandler.VerifyMealPlanPayment)
		}

		// Tiffin bulk confirm (#387) — its own group because a static segment can't
		// share the /orders/:id or /meal-plans/:id wildcard slot (httprouter).
		tiffin := v1.Group("/tiffin")
		tiffin.Use(bffAuth(bffKey, bffWindow))
		{
			tiffin.POST("/confirm-today", payoutHoldHandler.ConfirmTodaysTiffin) // #387
		}

		// Group / office orders (#46). Public invite preview by token + authed
		// accept; the rest scoped to participants inside the handlers.
		v1.GET("/group-invites/:token", groupOrderHandler.GroupJoinPreview)
		v1.POST("/group-invites/:token/accept", bffAuth(bffKey, bffWindow), groupOrderHandler.JoinGroupOrder)
		groupOrders := v1.Group("/group-orders")
		groupOrders.Use(bffAuth(bffKey, bffWindow))
		{
			groupOrders.POST("", groupOrderHandler.CreateGroupOrder)
			groupOrders.GET("", groupOrderHandler.GetMyGroupOrders)
			groupOrders.GET("/:id", groupOrderHandler.GetGroupOrder)
			groupOrders.POST("/:id/items", groupOrderHandler.AddGroupItem)
			groupOrders.DELETE("/:id/items/:itemId", groupOrderHandler.RemoveGroupItem)
			groupOrders.POST("/:id/lock", groupOrderHandler.LockGroupOrder)
			groupOrders.POST("/:id/pay", groupOrderHandler.PayGroupShare)
			groupOrders.POST("/:id/pay/verify", groupOrderHandler.VerifyGroupShare)
			groupOrders.POST("/:id/cancel", groupOrderHandler.CancelGroupOrder)
			groupOrders.POST("/:id/leave", groupOrderHandler.LeaveGroupOrder)
		}

		// Tiffin meal plans — chef side (#195). Scoped to the authed chef.
		chefMealPlans := v1.Group("/chef/meal-plans")
		chefMealPlans.Use(bffAuth(bffKey, bffWindow), middleware.RequireChef())
		{
			chefMealPlans.GET("", mealPlanHandler.GetChefMealPlanRequests)
			chefMealPlans.POST("/:id/respond", mealPlanHandler.RespondMealPlan)
		}

		// Bulk subscription prep view (#50) — own group so its static paths don't
		// collide with the meal-plans :id routes.
		chefPrep := v1.Group("/chef/prep")
		chefPrep.Use(bffAuth(bffKey, bffWindow), middleware.RequireChef())
		{
			chefPrep.GET("", mealPlanHandler.GetPrepManifest)
			chefPrep.POST("/mark", mealPlanHandler.MarkPrepBulk)
			chefPrep.POST("/day/:dayId", mealPlanHandler.MarkDayPrepared)
		}

		// Delivery staff routes — enforced with granular staff permissions
		deliveryStaff := v1.Group("/delivery/staff")
		deliveryStaff.Use(bffAuth(bffKey, bffWindow), middleware.RequireDelivery())
		{
			deliveryStaff.GET("/me", staffHandler.GetMyStaffProfile) // No permission needed — own profile
			deliveryStaff.GET("/roles", staffHandler.GetStaffRoles)  // No permission needed — role definitions
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
			chefSubscription.PUT("/tier", subscriptionHandler.ChangeTier) // #44 premium upgrade/downgrade
			chefSubscription.GET("/invoices", subscriptionHandler.GetInvoices)
			chefSubscription.GET("/earnings", subscriptionHandler.GetEarningsSummary)
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
			// Post-delivery tips (#45) — 100% pass-through to chef/rider.
			orderPayments.POST("/order/:orderId/tip", tipHandler.CreateOrderTip)
			orderPayments.POST("/tip/:tipId/verify", tipHandler.VerifyTip)
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
			admin.GET("/chefs/fssai-locked", adminHandler.GetFSSAILockedChefs)

			// Customer wallet — admin view + audited adjustment (#33)
			admin.GET("/wallet/:userId", adminHandler.GetCustomerWallet)
			admin.POST("/wallet/:userId/adjust", adminHandler.AdjustWallet)

			// Review moderation (#35) — list, hide, unhide (audited; recomputes rating)
			admin.GET("/reviews", adminHandler.AdminListReviews)
			admin.PUT("/reviews/:id/hide", adminHandler.AdminHideReview)
			admin.PUT("/reviews/:id/unhide", adminHandler.AdminUnhideReview)
			admin.POST("/chefs/:id/fssai-override", adminHandler.OverrideFSSAILock)
			admin.DELETE("/chefs/:id/fssai-override", adminHandler.ClearFSSAILockOverride)
			admin.GET("/fssai-expiry-backfill", adminHandler.FSSAIExpiryBackfill)
			admin.POST("/fssai-expiry-backfill", adminHandler.FSSAIExpiryBackfill)
			admin.PUT("/chefs/:id/verify", adminHandler.VerifyChef)
			admin.PUT("/chefs/:id/reject", adminHandler.RejectChef)
			admin.PUT("/chefs/:id/suspend", adminHandler.SuspendChef)

			// Order management
			admin.GET("/orders", adminHandler.GetAllOrders)
			admin.GET("/orders/:id", adminHandler.GetOrderDetails)

			// Tiffin meal-plan oversight (#199) — read-only, platform-wide
			admin.GET("/meal-plans", mealPlanHandler.AdminListMealPlans)
			admin.GET("/meal-plans/:id", mealPlanHandler.AdminGetMealPlan)

			// Promotions (featured ads)
			admin.GET("/promotions", promotionHandler.AdminListPromotions)
			admin.GET("/promotions/stats", promotionHandler.AdminGetPromotionStats)

			// Delivery management
			admin.GET("/delivery/stats", deliveryHandler.AdminGetDeliveryStats)
			admin.GET("/delivery/list", deliveryHandler.AdminListDeliveries)
			admin.GET("/delivery/partners", deliveryHandler.AdminGetDeliveryPartners)
			admin.GET("/delivery/partners/:id", deliveryHandler.GetPartnerDetail)

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

			// Subscription pricing (#44) — structured editor over the
			// subscription.{CC}.chef.* settings keys (standard + premium).
			admin.GET("/subscription-pricing", adminHandler.GetSubscriptionPricing)
			admin.PUT("/subscription-pricing", adminHandler.UpdateSubscriptionPricing)

			// Referral program config (#38) — reward amounts + monthly spend cap.
			// Order issues (#37) — review queue + assisted refund approval + config.
			admin.GET("/order-issues", orderIssueHandler.AdminListIssues)
			admin.POST("/order-issues/:issueId/resolve", orderIssueHandler.AdminResolveIssue)
			admin.POST("/order-issues/:issueId/reject", orderIssueHandler.AdminRejectIssue)
			admin.GET("/order-issue/config", adminHandler.GetOrderIssueConfig)
			admin.PUT("/order-issue/config", adminHandler.UpdateOrderIssueConfig)

			admin.GET("/referral/config", adminHandler.GetReferralConfig)
			admin.PUT("/referral/config", adminHandler.UpdateReferralConfig)
			// Loyalty program (#40) — config + analytics.
			admin.GET("/loyalty/config", adminHandler.GetLoyaltyConfig)
			admin.PUT("/loyalty/config", adminHandler.UpdateLoyaltyConfig)
			admin.GET("/loyalty/analytics", adminHandler.GetLoyaltyAnalytics)
			// In-app messaging mediation (#53) — the relay inbox.
			admin.GET("/messages/inbox", messagingHandler.AdminInbox)
			admin.POST("/messages/:id/relay", messagingHandler.AdminRelayMessage)
			admin.POST("/messages/:id/block", messagingHandler.AdminBlockMessage)
			admin.POST("/conversations/:id/send", messagingHandler.AdminSendMessage)
			// Communications audit read (#312) — list all conversations + full
			// transcripts (every status) + compliance export. Admin-only.
			admin.GET("/conversations", messagingHandler.AdminListConversations)
			admin.GET("/conversations/:id", messagingHandler.AdminConversationTranscript)
			admin.GET("/conversations/:id/export", messagingHandler.AdminExportConversation)
			// Marketing campaigns (#56) — compose, segment, lifecycle.
			admin.GET("/campaigns", adminHandler.ListCampaigns)
			admin.POST("/campaigns", adminHandler.CreateCampaign)
			admin.POST("/campaigns/preview", adminHandler.PreviewCampaignSegment)
			admin.GET("/campaigns/:id", adminHandler.GetCampaign)
			admin.PUT("/campaigns/:id", adminHandler.UpdateCampaign)
			admin.DELETE("/campaigns/:id", adminHandler.DeleteCampaign)
			admin.POST("/campaigns/:id/cancel", adminHandler.CancelCampaign)
			admin.POST("/campaigns/:id/schedule", adminHandler.ScheduleCampaign)
			admin.POST("/campaigns/:id/send", adminHandler.SendCampaignNow)
			admin.POST("/campaigns/:id/test", adminHandler.TestSendCampaign)
			admin.GET("/campaigns/:id/metrics", adminHandler.GetCampaignMetrics)
			// Win-back program (#42) — config + reactivation analytics.
			admin.GET("/winback/config", adminHandler.GetWinbackConfig)
			admin.PUT("/winback/config", adminHandler.UpdateWinbackConfig)
			admin.GET("/winback/analytics", adminHandler.GetWinbackAnalytics)

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
			admin.GET("/promos/:id/analytics", promoHandler.AdminGetPromoAnalytics)

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

			// Store-credit wallet (#33)
			customer.GET("/wallet", walletHandler.GetWallet)
			customer.GET("/wallet/transactions", walletHandler.GetWalletTransactions)

			// Loyalty points & streaks (#40)
			customer.GET("/loyalty", loyaltyHandler.GetLoyalty)
			customer.GET("/loyalty/transactions", loyaltyHandler.GetLoyaltyTransactions)
			customer.POST("/loyalty/redeem", loyaltyHandler.RedeemLoyalty)

			// In-app messaging (#53) — admin-mediated, order-scoped.
			customer.POST("/orders/:id/messages", messagingHandler.CustomerSendMessage)
			customer.GET("/orders/:id/messages", messagingHandler.CustomerListMessages)
			customer.POST("/orders/:id/attachments", messagingHandler.CustomerUploadAttachment)

			// Referral program (#38)
			customer.GET("/referral", referralHandler.GetMyReferral)
			customer.GET("/referral/history", referralHandler.GetReferralHistory)
			customer.POST("/referral/accept", referralHandler.AcceptReferral)

			// Win-back offer (#42) — active offer for the in-app banner.
			customer.GET("/winback/active", winbackHandler.GetActiveWinback)
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

			// Favorite dishes / menu items (#237)
			favorites.GET("/dishes", favoriteHandler.ListFavoriteDishes)
			favorites.GET("/dishes/ids", favoriteHandler.ListFavoriteDishIDs)
			favorites.POST("/dishes", favoriteHandler.AddFavoriteDish)
			favorites.DELETE("/dishes/:menuItemId", favoriteHandler.RemoveFavoriteDish)
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
