package database

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connection-pool defaults, sized for the shared Cloud SQL db-f1-micro.
// Postgres caps total connections (~100 default) across EVERY client, and
// Knative can run up to maxScale pods — so each pod must stay modest. 20 open
// keeps 5 pods at ~100 worst-case, and idle conns are released after
// connMaxIdleTime so a quiet pod doesn't hoard slots. Override via env for
// load-tuning without a redeploy.
const (
	defaultMaxOpenConns = 20
	defaultMaxIdleConns = 5
	connMaxLifetime     = 30 * time.Minute
	connMaxIdleTime     = 5 * time.Minute
)

// envInt reads a positive int from env, falling back to def on unset/invalid.
func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

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

	// Connection pool — right-sized for the shared db-f1-micro so multiple
	// Knative pods can't collectively exhaust Postgres' connection cap.
	maxOpen := envInt("DB_MAX_OPEN_CONNS", defaultMaxOpenConns)
	maxIdle := envInt("DB_MAX_IDLE_CONNS", defaultMaxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	log.Printf("Database connection established (pool: maxOpen=%d maxIdle=%d)", maxOpen, maxIdle)
	return nil
}

func Migrate() error {
	log.Println("Running database migrations...")

	// Pre-AutoMigrate: location reference tables were rewritten from the
	// legacy UUID-keyed shape (countries.id uuid, states.id uuid, ...) to the
	// mark8ly-style ISO-coded shape (countries.code char(2), states.id
	// varchar(10), ...). GORM AutoMigrate cannot change a primary-key
	// column's type in place, so we drop the old tables first. None of them
	// were ever populated in homechef, so this is a no-op for data.
	//
	// Idempotent: the IF EXISTS guard makes this safe to run on first boot
	// (where the tables don't exist yet) and on every subsequent boot.
	preMigrateDrops := []string{
		`DROP TABLE IF EXISTS postcodes CASCADE`,
		`DROP TABLE IF EXISTS cities CASCADE`,
		`DROP TABLE IF EXISTS states CASCADE`,
		`DROP TABLE IF EXISTS countries CASCADE`,
	}
	for _, stmt := range preMigrateDrops {
		// Only drop tables that match the legacy shape. We detect by checking
		// for the `is_active` column that the new ISO-keyed schema doesn't
		// carry. This way a re-run after the new schema is already in place
		// does NOT clobber seeded data.
		tableName := ""
		switch stmt {
		case `DROP TABLE IF EXISTS postcodes CASCADE`:
			tableName = "postcodes"
		case `DROP TABLE IF EXISTS cities CASCADE`:
			tableName = "cities"
		case `DROP TABLE IF EXISTS states CASCADE`:
			tableName = "states"
		case `DROP TABLE IF EXISTS countries CASCADE`:
			tableName = "countries"
		}
		var hasIsActive bool
		probe := DB.Raw(`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = ? AND column_name = 'is_active'
		)`, tableName).Scan(&hasIsActive)
		if probe.Error != nil {
			return fmt.Errorf("location pre-migrate probe failed (%q): %w", tableName, probe.Error)
		}
		if !hasIsActive {
			continue // already on the new schema, leave seeded data alone
		}
		if err := DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("location pre-migrate drop failed (%q): %w", stmt, err)
		}
		log.Printf("dropped legacy location table %q (UUID-keyed schema)", tableName)
	}

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
		&models.ChefNotificationPreferences{},

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
		&models.DishRating{},

		// Wallet (store-credit ledger, #33)
		&models.Wallet{},
		&models.WalletTxn{},

		// Social
		&models.Post{},
		&models.PostLike{},
		&models.PostComment{},

		// Catering
		&models.CateringRequest{},
		&models.CateringQuote{},

		// Tiffin weekly menu (#192) + meal plans (#193)
		&models.WeeklyMenu{},
		&models.WeeklyMenuItem{},
		&models.MealPlan{},
		&models.MealPlanDay{},

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

		// Weekly settlement statements
		&models.WeeklyStatement{},

		// Support Tickets
		&models.SupportTicket{},
		&models.SupportMessage{},

		// Promo Codes
		&models.PromoCode{},
		&models.PromoCodeUsage{},

		// Chat
		&models.ChatRoom{},
		&models.ChatMessage{},

		// Reliable event backbone: transactional outbox + consumer idempotency
		&models.OutboxEvent{},
		&models.ProcessedEvent{},
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
		// 3PL deliveries have no internal delivery partner. The model field is
		// now *uuid.UUID (nullable), but AutoMigrate never drops an existing
		// NOT NULL constraint — do it here so provider-fulfilled deliveries can
		// be inserted with a null partner. No-op once already nullable.
		`ALTER TABLE deliveries ALTER COLUMN delivery_partner_id DROP NOT NULL`,
	}
	for _, stmt := range postMigrate {
		if err := DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("post-migration DDL failed (%q): %w", stmt, err)
		}
	}

	// Backfill SEO slugs for chefs created before the slug column existed (#58),
	// so they're resolvable by slug via GET /chefs/:slug. Idempotent — only fills
	// empty slugs; new/updated chefs get one from ChefProfile.BeforeSave.
	backfillChefSlugs()

	log.Println("Database migrations completed")
	return nil
}

// backfillChefSlugs stamps a slug on any chef missing one. Best-effort: logs and
// continues on error so startup is never blocked by it.
func backfillChefSlugs() {
	var chefs []models.ChefProfile
	if err := DB.Where("slug = '' OR slug IS NULL").Find(&chefs).Error; err != nil {
		log.Printf("chef-slug backfill: query failed: %v", err)
		return
	}
	filled := 0
	for i := range chefs {
		slug := models.ChefSlug(chefs[i].BusinessName)
		if slug == "" {
			continue
		}
		if err := DB.Model(&models.ChefProfile{}).Where("id = ?", chefs[i].ID).Update("slug", slug).Error; err != nil {
			log.Printf("chef-slug backfill: update %s failed: %v", chefs[i].ID, err)
			continue
		}
		filled++
	}
	if filled > 0 {
		log.Printf("chef-slug backfill: stamped %d chef slugs", filled)
	}
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
