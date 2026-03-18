package config

import (
	"log"
	"os"
	"strconv"

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

	// JWT
	JWTSecret          string
	JWTExpirationHours int
	RefreshTokenDays   int

	// OAuth
	GoogleClientID     string
	GoogleClientSecret string
	FacebookAppID      string
	FacebookAppSecret  string
	AppleClientID      string
	AppleTeamID        string
	AppleKeyID         string

	// GCS Storage
	GCSProjectID     string
	GCSPublicBucket  string
	GCSPrivateBucket string

	// Stripe (legacy)
	StripeSecretKey      string
	StripeWebhookSecret  string
	StripePublishableKey string

	// Razorpay
	RazorpayKeyID        string
	RazorpayKeySecret    string
	RazorpayWebhookSecret string

	// SendGrid
	SendGridAPIKey   string
	FromEmail        string
	FromName         string

	// Twilio
	TwilioAccountSID  string
	TwilioAuthToken   string
	TwilioPhoneNumber string

	// Redis
	RedisURL string

	// NATS
	NATSURL string

	// Exchange Rates APIs
	OpenExchangeRatesAppID string
	ExchangeRatesAPIKey    string

	// Feature Flags
	EnableMockMode bool
}

var AppConfig *Config

func Load() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	jwtExpiration, _ := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))
	refreshTokenDays, _ := strconv.Atoi(getEnv("REFRESH_TOKEN_DAYS", "30"))
	enableMock, _ := strconv.ParseBool(getEnv("ENABLE_MOCK_MODE", "false"))

	AppConfig = &Config{
		// Server
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),

		// Database
		DatabaseURL: getEnv("DATABASE_URL", ""),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", ""),
		DBName:      getEnv("DB_NAME", "homechef"),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),

		// JWT
		JWTSecret:          getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		JWTExpirationHours: jwtExpiration,
		RefreshTokenDays:   refreshTokenDays,

		// OAuth
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		FacebookAppID:      getEnv("FACEBOOK_APP_ID", ""),
		FacebookAppSecret:  getEnv("FACEBOOK_APP_SECRET", ""),
		AppleClientID:      getEnv("APPLE_CLIENT_ID", ""),
		AppleTeamID:        getEnv("APPLE_TEAM_ID", ""),
		AppleKeyID:         getEnv("APPLE_KEY_ID", ""),

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

		// SendGrid
		SendGridAPIKey: getEnv("SENDGRID_API_KEY", ""),
		FromEmail:      getEnv("FROM_EMAIL", "noreply@homechef.com"),
		FromName:       getEnv("FROM_NAME", "HomeChef"),

		// Twilio
		TwilioAccountSID:  getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:   getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioPhoneNumber: getEnv("TWILIO_PHONE_NUMBER", ""),

		// Redis
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		// NATS
		NATSURL: getEnv("NATS_URL", "nats://localhost:4222"),

		// Exchange Rates APIs
		OpenExchangeRatesAppID: getEnv("OPENEXCHANGERATES_APP_ID", ""),
		ExchangeRatesAPIKey:    getEnv("EXCHANGERATES_API_KEY", ""),

		// Feature Flags
		EnableMockMode: enableMock,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func IsDevelopment() bool {
	return AppConfig.Environment == "development"
}

func IsProduction() bool {
	return AppConfig.Environment == "production"
}
