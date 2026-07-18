package config

import (
	"encoding/base64"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port        string
	Environment string

	// Database
	DatabaseURL string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string

	// BFF trust — apps/auth-bff signs every inbound request with this
	// shared HMAC; the API rejects anything missing or stale.
	BFFInternalHMACKey []byte
	BFFAuthTSWindow    time.Duration

	// BFFSessionURL, when set, enables a Bearer-token fallback in the
	// BFFAuth middleware. Dev convenience for mobile/SPA clients that
	// haven't been rebuilt to route every request through the BFF proxy.
	// Leave empty in production: enforces HMAC-only access.
	BFFSessionURL string

	// OAuth (public client IDs only — the OIDC handshake lives in
	// apps/auth-bff via Google Identity Platform). Kept here because
	// some frontends still receive the public ID from server config.
	GoogleClientID string
	FacebookAppID  string

	// GCS Storage
	GCSProjectID     string
	GCSPublicBucket  string
	GCSPrivateBucket string

	// Stripe (legacy)
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePublishableKey string

	// Razorpay
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string

	// Shadowfax 3PL (from Secret Manager; empty leaves the provider disabled)
	ShadowfaxAPIToken      string
	ShadowfaxWebhookSecret string

	// Email — SendGrid primary, Resend fallback
	SendGridAPIKey string
	ResendAPIKey   string
	FromEmail      string
	FromName       string

	// Twilio
	TwilioAccountSID  string
	TwilioAuthToken   string
	TwilioPhoneNumber string

	// Redis
	RedisURL string

	// MongoDB — backs in-app chat (admin-mediated messaging) + document uploads
	// (#53). Optional: empty MongoURI disables the Mongo-backed features.
	MongoURI    string
	MongoDBName string

	// NATS
	NATSURL string
	// NATSStreamReplicas is the JetStream stream replica factor. 1 for local /
	// single-node dev; set to 3 in prod (3-node JetStream HA cluster).
	NATSStreamReplicas int
	// OutboxRelayInterval is how often the transactional-outbox relay scans for
	// pending events to publish.
	OutboxRelayInterval time.Duration
	// OutboxBatchSize is the max number of outbox rows claimed per scan.
	OutboxBatchSize int
	// OutboxMaxAttempts is the publish attempt cap before a row is dead-lettered
	// (kept as status='failed' for inspection/replay).
	OutboxMaxAttempts int

	// Exchange Rates APIs
	OpenExchangeRatesAppID string
	ExchangeRatesAPIKey    string

	// Feature Flags
	EnableMockMode bool
	// WalletCheckoutEnabled gates applying wallet store-credit at checkout (#141).
	// Default OFF: the chef/driver top-up uses Razorpay direct transfers
	// (services.CreateTransfer) which must be verified in the Razorpay sandbox
	// before it touches live settlement. Enable with WALLET_CHECKOUT_ENABLED=true.
	WalletCheckoutEnabled bool
	// GroupOrdersEnabled gates the group / office orders feature (#46): shared
	// cart, split payment, and consolidation. Default OFF — the multi-payer money
	// flow should be verified in the Razorpay sandbox before going live.
	GroupOrdersEnabled bool
	// OrderPayoutAutoReleaseEnabled gates auto-releasing a delivered regular
	// order's held chef/rider Route transfers (#217). Default OFF — moves live
	// settlement; verify in the Razorpay sandbox (#218) before enabling.
	OrderPayoutAutoReleaseEnabled bool
	// MealPlanEscrowEnabled gates the tiffin meal-plan ESCROW money flow (#194):
	// upfront advance capture, refund of declined/expired days, and per-day payout
	// release on delivery. Default OFF — the negotiation handshake (#195/#196) works
	// without it; flip on only after the Razorpay escrow paths are sandbox-verified.
	MealPlanEscrowEnabled bool
	// MealSubscriptionAutoActivate gates whether a new tiffin subscription starts
	// ACTIVE (so the daily-order generator + pause/resume/skip work end-to-end) vs
	// TRIALING. Default OFF: real recurring CHARGING (Razorpay UPI-Autopay mandate,
	// #281) isn't wired yet, so an auto-activated sub would generate daily orders
	// WITHOUT auto-charging the customer. Turn ON in test/staging to exercise the
	// full flow; keep OFF in prod until #281 billing is live.
	MealSubscriptionAutoActivate bool
	// CateringDepositEnabled gates the catering deposit/advance money flow (#55):
	// creating a Razorpay deposit order to confirm a catering booking. Default OFF —
	// the request → quote → accept flow works without it; flip on only after the
	// Razorpay deposit path is sandbox-verified (#218).
	CateringDepositEnabled bool
	// OrderSagaEnabled gates orchestrating the post-payment order lifecycle as a
	// durable Temporal saga (#122): notify chef → await accept → await ready →
	// dispatch → await delivered → settle, with refund compensation. Default OFF —
	// the existing synchronous handlers are authoritative until ops validates the
	// saga end-to-end on the cluster; the saga's activities are idempotent so
	// enabling it never double-acts alongside residual handler logic.
	OrderSagaEnabled bool
	// OnboardingWorkflowEnabled gates running chef-onboarding activation as a
	// durable Temporal workflow (#126) instead of the inline approval side
	// effects. Default OFF — the inline activation stays authoritative until ops
	// validates the workflow; the activation op is idempotent so the durable path
	// re-runs safely. Closes the gap where a crash mid-approval left a chef
	// "approved" but never actually verified/activated.
	OnboardingWorkflowEnabled bool

	// DeliveryDistancePricePerCallUSD / DeliveryWeatherPricePerCallUSD are the
	// per-call prices of the metered delivery-intelligence providers (#699), used
	// ONLY to estimate spend for the admin cost view — they don't affect what a
	// customer is charged. Defaults track Google Maps Platform list pricing
	// (Routes ~$5 / 1000 = $0.005; Weather ~$0.001 / call). Override per provider.
	DeliveryDistancePricePerCallUSD float64
	DeliveryWeatherPricePerCallUSD  float64

	// GoogleMapsAPIKey enables the real Google Routes distance provider (#701/#700).
	// When set, self-delivery distances come from the Routes API (road distance,
	// cached so it's paid at most once per trip); when empty, the winding-factor
	// haversine fallback is used — so this is a pure accuracy upgrade, opt-in by key.
	GoogleMapsAPIKey string
}

var AppConfig *Config

func Load() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	enableMock, _ := strconv.ParseBool(getEnv("ENABLE_MOCK_MODE", "false"))
	walletCheckout, _ := strconv.ParseBool(getEnv("WALLET_CHECKOUT_ENABLED", "false"))
	mealPlanEscrow, _ := strconv.ParseBool(getEnv("MEAL_PLAN_ESCROW_ENABLED", "false"))
	mealSubAutoActivate, _ := strconv.ParseBool(getEnv("MEAL_SUBSCRIPTION_AUTO_ACTIVATE", "false"))
	groupOrders, _ := strconv.ParseBool(getEnv("GROUP_ORDERS_ENABLED", "false"))
	orderPayoutAutoRelease, _ := strconv.ParseBool(getEnv("ORDER_PAYOUT_AUTO_RELEASE_ENABLED", "false"))
	cateringDeposit, _ := strconv.ParseBool(getEnv("CATERING_DEPOSIT_ENABLED", "false"))
	orderSaga, _ := strconv.ParseBool(getEnv("ORDER_SAGA_ENABLED", "false"))
	onboardingWorkflow, _ := strconv.ParseBool(getEnv("ONBOARDING_WORKFLOW_ENABLED", "false"))
	distancePricePerCall, _ := strconv.ParseFloat(getEnv("DELIVERY_DISTANCE_PRICE_PER_CALL_USD", "0.005"), 64)
	weatherPricePerCall, _ := strconv.ParseFloat(getEnv("DELIVERY_WEATHER_PRICE_PER_CALL_USD", "0.001"), 64)
	env := getEnv("ENVIRONMENT", "development")
	isProd := env == "production"

	dbPassword := os.Getenv("DB_PASSWORD")
	dbURL := os.Getenv("DATABASE_URL")
	if isProd && dbPassword == "" && dbURL == "" {
		log.Fatal("DB_PASSWORD or DATABASE_URL is required in production")
	}

	// BFF HMAC key — required in every environment. apps/auth-bff and the
	// API must share the exact same value; the k8s ExternalSecret
	// (prod-homechef-bff-internal-hmac-key) wires the same secret into
	// both pods.
	hmacRaw := os.Getenv("BFF_INTERNAL_HMAC_KEY")
	if hmacRaw == "" {
		log.Fatal("BFF_INTERNAL_HMAC_KEY required (base64, 16+ bytes)")
	}
	hmacKey, err := base64.StdEncoding.DecodeString(hmacRaw)
	if err != nil {
		log.Fatalf("BFF_INTERNAL_HMAC_KEY must be valid base64: %v", err)
	}
	if len(hmacKey) < 16 {
		log.Fatal("BFF_INTERNAL_HMAC_KEY must decode to at least 16 bytes")
	}

	// Clock-skew window for the X-Auth-Ts header. Default 60s — same as
	// what auth-bff stamps with. Operators can widen it for laggy
	// cross-region environments.
	windowSecs := 60
	if v := os.Getenv("BFF_AUTH_TS_WINDOW_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			log.Fatal("BFF_AUTH_TS_WINDOW_SECONDS must be a positive integer")
		}
		windowSecs = n
	}

	AppConfig = &Config{
		// Server
		Port:        getEnv("PORT", "8080"),
		Environment: env,

		// Database
		DatabaseURL: dbURL,
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  dbPassword,
		DBName:      getEnv("DB_NAME", "homechef"),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),

		// BFF trust
		BFFInternalHMACKey: hmacKey,
		BFFAuthTSWindow:    time.Duration(windowSecs) * time.Second,
		BFFSessionURL:      getEnv("BFF_SESSION_URL", ""),

		// OAuth public IDs (secrets removed — owned by auth-bff/GIP)
		GoogleClientID:   getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleMapsAPIKey: getEnv("GOOGLE_MAPS_API_KEY", ""),
		FacebookAppID:    getEnv("FACEBOOK_APP_ID", ""),

		// GCS Storage
		GCSProjectID:     getEnv("GCS_PROJECT_ID", "tesseracthub-480811"),
		GCSPublicBucket:  getEnv("GCS_PUBLIC_BUCKET", "homechef-prod-assets-in"),
		GCSPrivateBucket: getEnv("GCS_PRIVATE_BUCKET", "homechef-prod-docs-in"),

		// Stripe (legacy)
		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),

		// Razorpay
		RazorpayKeyID:         getEnv("RAZORPAY_KEY_ID", ""),
		RazorpayKeySecret:     getEnv("RAZORPAY_KEY_SECRET", ""),
		RazorpayWebhookSecret: getEnv("RAZORPAY_WEBHOOK_SECRET", ""),

		ShadowfaxAPIToken:      getEnv("SHADOWFAX_API_TOKEN", ""),
		ShadowfaxWebhookSecret: getEnv("SHADOWFAX_WEBHOOK_SECRET", ""),

		// Email — SendGrid primary, Resend fallback
		SendGridAPIKey: getEnv("SENDGRID_API_KEY", ""),
		ResendAPIKey:   getEnv("RESEND_API_KEY", ""),
		FromEmail:      getEnv("FROM_EMAIL", "noreply@homechef.com"),
		FromName:       getEnv("FROM_NAME", "HomeChef"),

		// Twilio
		TwilioAccountSID:  getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:   getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioPhoneNumber: getEnv("TWILIO_PHONE_NUMBER", ""),

		// Redis
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
		MongoURI:    getEnv("MONGODB_URI", ""),
		MongoDBName: getEnv("MONGODB_DB", "homechef_chat"),

		// NATS
		NATSURL:             getEnv("NATS_URL", "nats://localhost:4222"),
		NATSStreamReplicas:  getEnvInt("NATS_STREAM_REPLICAS", 1),
		OutboxRelayInterval: time.Duration(getEnvInt("OUTBOX_RELAY_INTERVAL_MS", 1000)) * time.Millisecond,
		OutboxBatchSize:     getEnvInt("OUTBOX_BATCH_SIZE", 100),
		OutboxMaxAttempts:   getEnvInt("OUTBOX_MAX_ATTEMPTS", 10),

		// Exchange Rates APIs
		OpenExchangeRatesAppID: getEnv("OPENEXCHANGERATES_APP_ID", ""),
		ExchangeRatesAPIKey:    getEnv("EXCHANGERATES_API_KEY", ""),

		// Feature Flags
		EnableMockMode:                  enableMock,
		WalletCheckoutEnabled:           walletCheckout,
		MealPlanEscrowEnabled:           mealPlanEscrow,
		MealSubscriptionAutoActivate:    mealSubAutoActivate,
		GroupOrdersEnabled:              groupOrders,
		DeliveryDistancePricePerCallUSD: distancePricePerCall,
		DeliveryWeatherPricePerCallUSD:  weatherPricePerCall,
		OrderPayoutAutoReleaseEnabled:   orderPayoutAutoRelease,
		CateringDepositEnabled:          cateringDeposit,
		OrderSagaEnabled:                orderSaga,
		OnboardingWorkflowEnabled:       onboardingWorkflow,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvInt reads an integer env var, falling back to defaultValue when unset or
// unparseable.
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}
	return defaultValue
}

func IsDevelopment() bool {
	return AppConfig.Environment == "development"
}

func IsProduction() bool {
	return AppConfig.Environment == "production"
}
