package routes

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/homechef/api/config"
	"github.com/homechef/api/handlers"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func SetupRouter() *gin.Engine {
	// Set Gin mode
	if config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Prometheus metrics middleware
	r.Use(middleware.PrometheusMiddleware())

	// CORS configuration
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{
		"http://localhost:5173",
		"http://localhost:3000",
		"https://homechef.app",
		"https://fe3dr.com",
		"https://www.fe3dr.com",
		"https://vendors.fe3dr.com",
		"https://admin.fe3dr.com",
		"https://auth.fe3dr.com",
		"https://delivery.fe3dr.com",
	}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "x-jwt-claim-sub", "x-jwt-claim-tenant-id", "x-jwt-claim-tenant-slug", "x-jwt-claim-email", "x-jwt-claim-name", "x-jwt-claim-given-name", "x-jwt-claim-family-name"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	// Initialize handlers
	authHandler := handlers.NewAuthHandler()
	chefHandler := handlers.NewChefHandler()
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

	// Health check endpoints
	r.GET("/health", healthHandler.Health)
	r.GET("/health/live", healthHandler.Liveness)
	r.GET("/health/ready", healthHandler.Readiness)
	r.GET("/health/stats", healthHandler.SystemStats)

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Razorpay webhook (no auth — uses HMAC signature verification)
	r.POST("/webhooks/razorpay", paymentHandler.RazorpayWebhook)

	// Provider webhooks (public, verified by webhook secret)
	r.POST("/webhooks/delivery/:provider", providerHandler.HandleWebhook)

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
		}

		// Preference options (public)
		v1.GET("/preferences", preferenceHandler.GetPreferenceOptions)

		// Currency routes (public)
		currencies := v1.Group("/currencies")
		{
			currencies.GET("", currencyHandler.ListCurrencies)
			currencies.GET("/rates", currencyHandler.GetRates)
			currencies.GET("/detect", currencyHandler.DetectCurrency)
		}

		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/oauth", authHandler.OAuthLogin)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
		}

		// Staff invitation routes (public - token validates)
		v1.GET("/staff/invitations/validate", staffHandler.ValidateInvitation)

		// Staff invitation acceptance (requires auth via OptionalAuth for info, Auth for accept)
		staffInvite := v1.Group("/staff/invitations")
		staffInvite.Use(middleware.OptionalAuthMiddleware())
		{
			staffInvite.POST("/accept", staffHandler.AcceptInvitation)
		}

		// Profile routes (authenticated)
		profile := v1.Group("/profile")
		profile.Use(middleware.AuthMiddleware())
		{
			profile.GET("", authHandler.GetProfile)
			profile.PUT("", authHandler.UpdateProfile)
			profile.PUT("/password", authHandler.ChangePassword)
		}

		// Public chef routes
		chefs := v1.Group("/chefs")
		chefs.Use(middleware.OptionalAuthMiddleware())
		{
			chefs.GET("", chefHandler.ListChefs)
			chefs.GET("/:id", chefHandler.GetChef)
			chefs.GET("/:id/menu", chefHandler.GetChefMenu)
			chefs.GET("/:id/reviews", chefHandler.GetChefReviews)
		}

		// Chef onboarding (authenticated, but no chef role required — user is becoming a chef)
		chefOnboarding := v1.Group("/chef")
		chefOnboarding.Use(middleware.AuthMiddleware())
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
		chefMenu.Use(middleware.AuthMiddleware())
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
		}

		// Chef dashboard routes (chef only)
		chefDashboard := v1.Group("/chef")
		chefDashboard.Use(middleware.AuthMiddleware(), middleware.RequireChef())
		{
			chefDashboard.GET("/dashboard", chefHandler.GetChefDashboard)
			chefDashboard.GET("/profile", chefHandler.GetChefProfile)
			chefDashboard.PUT("/profile", chefHandler.UpdateChefProfile)
			chefDashboard.GET("/orders", chefHandler.GetChefOrders)
			chefDashboard.PUT("/orders/:orderId/status", chefHandler.UpdateOrderStatus)
			chefDashboard.GET("/reviews", chefHandler.GetChefReviewsForDashboard)
			chefDashboard.POST("/reviews/:reviewId/reply", chefHandler.ReplyToReview)
			chefDashboard.GET("/settings", chefHandler.GetChefSettings)
			chefDashboard.PUT("/settings", chefHandler.UpdateChefSettings)
			chefDashboard.GET("/analytics", chefHandler.GetChefAnalytics)
			chefDashboard.GET("/payout", chefHandler.GetPayoutDetails)
			chefDashboard.POST("/payout", chefHandler.SavePayoutDetails)
			chefDashboard.GET("/admin-requests", approvalHandler.GetChefApprovalRequests)
			chefDashboard.PUT("/admin-requests/:id/respond", approvalHandler.RespondToApprovalRequest)
		}

		// Customer order routes
		orders := v1.Group("/orders")
		orders.Use(middleware.AuthMiddleware())
		{
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("", orderHandler.GetOrders)
			orders.GET("/:id", orderHandler.GetOrder)
			orders.POST("/:id/cancel", orderHandler.CancelOrder)
			orders.GET("/:id/track", orderHandler.TrackOrder)
			orders.GET("/:id/invoice", orderHandler.GetOrderInvoice)
		}

		// Cart routes
		cart := v1.Group("/cart")
		cart.Use(middleware.AuthMiddleware())
		{
			// cart.GET("", cartHandler.GetCart)
			// cart.POST("/items", cartHandler.AddItem)
			// cart.PUT("/items/:itemId", cartHandler.UpdateItem)
			// cart.DELETE("/items/:itemId", cartHandler.RemoveItem)
			// cart.DELETE("", cartHandler.ClearCart)
		}

		// Social feed routes
		social := v1.Group("/social")
		social.Use(middleware.OptionalAuthMiddleware())
		{
			// social.GET("/feed", socialHandler.GetFeed)
			// social.GET("/posts/:id", socialHandler.GetPost)
			// social.POST("/posts/:id/like", socialHandler.LikePost) // requires auth
			// social.POST("/posts/:id/comments", socialHandler.AddComment) // requires auth
		}

		// Chef social posts (chef only)
		chefSocial := v1.Group("/chef/posts")
		chefSocial.Use(middleware.AuthMiddleware(), middleware.RequireChef())
		{
			// chefSocial.GET("", socialHandler.GetChefPosts)
			// chefSocial.POST("", socialHandler.CreatePost)
			// chefSocial.PUT("/:id", socialHandler.UpdatePost)
			// chefSocial.DELETE("/:id", socialHandler.DeletePost)
		}

		// Catering routes
		catering := v1.Group("/catering")
		catering.Use(middleware.AuthMiddleware())
		{
			// Customer catering
			// catering.POST("/requests", cateringHandler.CreateRequest)
			// catering.GET("/requests", cateringHandler.GetMyRequests)
			// catering.GET("/requests/:id", cateringHandler.GetRequest)
			// catering.GET("/requests/:id/quotes", cateringHandler.GetQuotes)
			// catering.POST("/quotes/:id/accept", cateringHandler.AcceptQuote)
		}

		// Chef catering (chef only)
		chefCatering := v1.Group("/chef/catering")
		chefCatering.Use(middleware.AuthMiddleware(), middleware.RequireChef())
		{
			// chefCatering.GET("/requests", cateringHandler.GetAvailableRequests)
			// chefCatering.POST("/requests/:id/quote", cateringHandler.SubmitQuote)
			// chefCatering.GET("/quotes", cateringHandler.GetChefQuotes)
		}

		// Delivery partner onboarding (authenticated, no delivery role required)
		deliveryOnboarding := v1.Group("/delivery")
		deliveryOnboarding.Use(middleware.AuthMiddleware())
		{
			deliveryOnboarding.GET("/onboarding/status", deliveryHandler.GetOnboardingStatus)
			deliveryOnboarding.POST("/onboarding", deliveryHandler.Onboarding)
		}

		// Driver onboarding (authenticated, no role required — user is becoming a driver)
		driverOnboarding := v1.Group("/driver")
		driverOnboarding.Use(middleware.AuthMiddleware())
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
		driverReferral.Use(middleware.AuthMiddleware(), middleware.RequireDelivery())
		{
			driverReferral.GET("/code", driverHandler.GetReferralCode)
			driverReferral.GET("/stats", driverHandler.GetReferralStats)
		}

		// Delivery staff routes — enforced with granular staff permissions
		deliveryStaff := v1.Group("/delivery/staff")
		deliveryStaff.Use(middleware.AuthMiddleware(), middleware.RequireDelivery())
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
		delivery.Use(middleware.AuthMiddleware(), middleware.RequireDelivery())
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
		}

		// Chef promotion routes (featured ads)
		chefPromotion := v1.Group("/chef/promotion")
		chefPromotion.Use(middleware.AuthMiddleware(), middleware.RequireChef())
		{
			chefPromotion.GET("/pricing", promotionHandler.GetFeaturedAdPricing)
			chefPromotion.POST("/purchase", promotionHandler.PurchaseFeaturedAd)
			chefPromotion.POST("/confirm", promotionHandler.ConfirmFeaturedAd)
			chefPromotion.GET("/history", promotionHandler.GetMyPromotions)
		}

		// Chef subscription routes (chef role required)
		chefSubscription := v1.Group("/chef/subscription")
		chefSubscription.Use(middleware.AuthMiddleware(), middleware.RequireChef())
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
		driverSubscription.Use(middleware.AuthMiddleware(), middleware.RequireDelivery())
		{
			driverSubscription.GET("", subscriptionHandler.GetSubscription)
			driverSubscription.POST("/choose-plan", subscriptionHandler.ChoosePlan)
			driverSubscription.POST("/cancel", subscriptionHandler.CancelSubscription)
			driverSubscription.PUT("/change-plan", subscriptionHandler.ChangePlan)
			driverSubscription.GET("/invoices", subscriptionHandler.GetInvoices)
			driverSubscription.GET("/earnings", subscriptionHandler.GetEarningsSummary)
		}

		// Payment routes (authenticated)
		orderPayments := v1.Group("/payments")
		orderPayments.Use(middleware.AuthMiddleware())
		{
			orderPayments.POST("/order/:orderId/create", paymentHandler.CreateOrderPayment)
			orderPayments.POST("/order/:orderId/verify", paymentHandler.VerifyPayment)
			orderPayments.POST("/order/:orderId/refund", paymentHandler.InitiateRefund)
		}

		// Admin routes
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(), middleware.RequireAdmin())
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

			// Payment gateway
			admin.GET("/payment-gateway/status", adminHandler.GetPaymentGatewayStatus)
			admin.PUT("/payment-gateway/keys", adminHandler.UpdatePaymentGatewayKeys)

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

			// Subscription management
			admin.GET("/subscriptions", subscriptionHandler.AdminGetSubscriptions)
			admin.GET("/subscriptions/stats", subscriptionHandler.AdminGetSubscriptionStats)
			admin.GET("/subscriptions/:id", subscriptionHandler.AdminGetSubscription)
			admin.GET("/invoices", subscriptionHandler.AdminListInvoices)
		}

		// Addresses
		addresses := v1.Group("/addresses")
		addresses.Use(middleware.AuthMiddleware())
		{
			addresses.GET("", addressHandler.GetAddresses)
			addresses.POST("", addressHandler.CreateAddress)
			addresses.PUT("/:id", addressHandler.UpdateAddress)
			addresses.DELETE("/:id", addressHandler.DeleteAddress)
		}

		// Payment methods
		payments := v1.Group("/payment-methods")
		payments.Use(middleware.AuthMiddleware())
		{
			// payments.GET("", paymentHandler.GetPaymentMethods)
			// payments.POST("", paymentHandler.AddPaymentMethod)
			// payments.DELETE("/:id", paymentHandler.RemovePaymentMethod)
			// payments.PUT("/:id/default", paymentHandler.SetDefaultPaymentMethod)
		}

		// Customer profile & onboarding
		customer := v1.Group("/customer")
		customer.Use(middleware.AuthMiddleware())
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
		reviews.Use(middleware.AuthMiddleware())
		{
			reviews.POST("", reviewHandler.CreateReview)
		}

		// Favorites (authenticated customers)
		favorites := v1.Group("/favorites")
		favorites.Use(middleware.AuthMiddleware())
		{
			favorites.GET("/chefs", favoriteHandler.ListFavoriteChefs)
			favorites.GET("/chefs/ids", favoriteHandler.ListFavoriteChefIDs)
			favorites.POST("/chefs", favoriteHandler.AddFavoriteChef)
			favorites.DELETE("/chefs/:chefId", favoriteHandler.RemoveFavoriteChef)
		}

		// Notifications
		notifications := v1.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware())
		{
			notifications.GET("", notificationHandler.GetNotifications)
			notifications.GET("/unread-count", notificationHandler.GetUnreadCount)
			notifications.PUT("/:id/read", notificationHandler.MarkAsRead)
			notifications.PUT("/read-all", notificationHandler.MarkAllAsRead)
		}
	}

	return r
}

// RoleToPermissions returns permissions for debugging
func RoleToPermissions(role models.UserRole) []middleware.Permission {
	return middleware.RolePermissions[role]
}
