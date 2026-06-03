package database

import (
	"fmt"
	"log"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() error {
	var dsn string
	cfg := config.AppConfig

	if cfg.DatabaseURL != "" {
		dsn = cfg.DatabaseURL
	} else {
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)
	}

	// Configure logger based on environment
	logLevel := logger.Silent
	if config.IsDevelopment() {
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Database connection established")
	return nil
}

func Migrate() error {
	log.Println("Running database migrations...")

	err := DB.AutoMigrate(
		// Users. Auth tokens (refresh/password-reset/email-verification) are
		// no longer owned by this service — apps/auth-bff handles all session
		// + verification flows via Google Identity Platform.
		&models.User{},
		&models.CustomerProfile{},
		&models.PreferenceOption{},

		// Chef
		&models.ChefProfile{},
		&models.ChefSchedule{},
		&models.ChefDocument{},

		// Menu
		&models.MenuCategory{},
		&models.MenuItem{},
		&models.MenuItemImage{},

		// Orders
		&models.Order{},
		&models.OrderItem{},

		// Cart
		&models.Cart{},
		&models.CartItem{},

		// Delivery
		&models.DeliveryPartner{},
		&models.Delivery{},
		&models.DeliveryPartnerDocument{},
		&models.DeliveryZone{},
		&models.DeliveryProvider{},
		&models.DriverReferral{},

		// Promotions
		&models.ChefPromotion{},

		// Reviews
		&models.Review{},

		// Social
		&models.Post{},
		&models.PostLike{},
		&models.PostComment{},

		// Catering
		&models.CateringRequest{},
		&models.CateringQuote{},

		// Addresses
		&models.Address{},

		// Payments
		&models.PaymentMethod{},
		&models.Transaction{},

		// Location reference tables
		&models.Country{},
		&models.State{},
		&models.City{},
		&models.Postcode{},

		// Chef Settings
		&models.ChefSettings{},

		// Favorites
		&models.FavoriteChef{},

		// Notifications
		&models.Notification{},
		&models.NotificationPreference{},

		// Admin
		&models.PlatformSettings{},
		&models.AuditLog{},
		&models.ApiKey{},

		// Currency
		&models.Currency{},
		&models.ExchangeRate{},

		// Per-country tax rules
		&models.TaxRate{},

		// Approvals
		&models.ApprovalRequest{},
		&models.ApprovalRequestHistory{},

		// Staff & Invitations
		&models.StaffMember{},
		&models.StaffInvitation{},

		// Subscriptions & Billing
		&models.Subscription{},
		&models.SubscriptionInvoice{},
		&models.EarningsLedger{},

		// Invoices
		&models.OrderInvoice{},

		// Support Tickets
		&models.SupportTicket{},
		&models.SupportMessage{},

		// Promo Codes
		&models.PromoCode{},
		&models.PromoCodeUsage{},

		// Chat
		&models.ChatRoom{},
		&models.ChatMessage{},
	)

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Post-AutoMigrate DDL that AutoMigrate can't express. Idempotent — every
	// statement is `IF EXISTS` / `IF NOT EXISTS` so it's a no-op after the
	// first successful run.
	//
	// Background: the GIP cutover introduced per-pool email uniqueness so the
	// same person can exist as both a customer and a chef. The legacy global
	// unique on users.email (idx_users_email, possibly also a users_email_key
	// constraint depending on table provenance) blocks that. AutoMigrate
	// won't drop the old index — it only adds, never removes. Migration
	// 20260514000002 was authored to do this swap but the API never ran
	// golang-migrate SQL files; the .up.sql sat dead in the repo. This block
	// is the live version that ships with the service.
	postMigrate := []string{
		// New schema. gip_uid is already covered by the model's uniqueIndex tag
		// but stating it here keeps this block self-documenting.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_gip_uid ON users (gip_uid) WHERE gip_uid IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_users_email_pool ON users (email, auth_pool)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_per_pool ON users (lower(email), auth_pool) WHERE email IS NOT NULL AND auth_pool IS NOT NULL`,
		// Drop the legacy global email uniqueness. Postgres unique constraints
		// own their backing index, so dropping the constraint first is required;
		// the bare DROP INDEX handles the case where the original was a plain
		// CREATE UNIQUE INDEX with no owning constraint.
		`ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key`,
		`ALTER TABLE users DROP CONSTRAINT IF EXISTS idx_users_email`,
		`DROP INDEX IF EXISTS users_email_key`,
		`DROP INDEX IF EXISTS idx_users_email`,
	}
	for _, stmt := range postMigrate {
		if err := DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("post-migration DDL failed (%q): %w", stmt, err)
		}
	}

	log.Println("Database migrations completed")
	return nil
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
