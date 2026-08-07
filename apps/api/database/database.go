package database

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
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
		// Test-chef mode: sandbox sessions and per-mode aggregates.
		&models.ChefTestSession{},
		// FSSAI filing requests — a chef paying us to obtain their registration.
		&models.FssaiRequest{},
		&models.FssaiRequestDocument{},
		&models.ChefModeStats{},

		// Menu
		&models.MenuCategory{},
		&models.MenuItem{},
		&models.MenuItemImage{},

		// Add-ons / combos (#52)
		&models.ModifierGroup{},
		&models.ModifierOption{},
		&models.BakerySpec{},
		&models.BakeryOption{},
		&models.ComboItem{},

		// Orders
		&models.Order{},
		// One row per order: what the chef is paid for it (D-21).
		&models.OrderChefPayout{},
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

		// Refund ledger (#689) — one row per gateway refund ATTEMPT, written
		// pending BEFORE the gateway call so a crash mid-call leaves a claim a
		// sweep can reconcile rather than money moving off our books silently.
		&models.RefundTransaction{},
		// Durable road-distance cache (#699) — a (chef → address) distance is paid
		// for once, ever; repeat orders reuse it across restarts / Redis flushes.
		&models.DeliveryDistanceCache{},

		// Loyalty (points ledger + streaks, #40)
		&models.LoyaltyAccount{},
		&models.LoyaltyTransaction{},

		// Marketing campaigns (#56)
		&models.Campaign{},
		&models.CampaignDelivery{},

		// Catering
		&models.CateringRequest{},
		&models.CateringQuote{},

		// Tiffin weekly menu (#192) + per-date menu (#405) + meal plans (#193)
		&models.WeeklyMenu{},
		&models.WeeklyMenuItem{},
		&models.DailyMenu{},
		&models.DailyMenuItem{},
		&models.MealPlan{},
		&models.MealPlanDay{},

		// Cancellation with vendor arbitration + tiered refund (#475)
		&models.CancellationRequest{},

		// Post-delivery tips (#45)
		&models.Tip{},

		// Group / office orders (#46)
		&models.GroupOrder{},
		&models.GroupOrderParticipant{},
		&models.GroupOrderItem{},

		// Chef capacity & cutoff controls (#48)
		&models.MenuItemDailySales{},
		&models.ChefCapacitySettings{},

		// Scheduled delivery slots (#51)
		&models.ChefSlotDailyBookings{},

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
		&models.FavoriteDish{},

		// Likes + subscriptions (discovery ranking, notification audience)
		&models.ChefLike{},
		&models.ChefSubscription{},

		&models.ReferralCode{},
		&models.Referral{},
		&models.OrderIssue{},

		// Customer refund-abuse detection (#937): append-only event ledger, the
		// derived score the admin queue reads, and the audit of every decision.
		&models.CustomerRiskEvent{},
		&models.CustomerRiskProfile{},
		&models.CustomerRiskAction{},

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
		// GST credit notes — refund policy v3 returns collected tax, so every refund
		// must be matched by a note or the filing overstates output tax (#834).
		&models.CreditNote{},

		// Weekly settlement statements
		&models.WeeklyStatement{},
		&models.ChefExpense{},
		// Chef cancellation levies, netted off the next settlement (#834).
		&models.ChefPenalty{},
		// Chef-refers-chef program: redemptions + settlement credits.
		&models.ChefReferral{},
		&models.ChefBonus{},
		// Chef loyalty points (earn per delivered order → cashback conversion).
		&models.ChefLoyaltyAccount{},
		&models.ChefLoyaltyTxn{},

		// Support Tickets
		&models.SupportTicket{},
		&models.SupportMessage{},

		// Promo Codes
		&models.PromoCode{},
		&models.PromoCodeUsage{},
		// Win-back offers (#42)
		&models.WinbackOffer{},
		// Customer meal subscriptions (tiffin, #2/#3)
		&models.ChefSubscriptionConfig{},
		&models.MealTrial{},
		&models.MealSubscription{},
		&models.MealSubscriptionSkip{},
		&models.MealSubscriptionInvoice{},
		&models.MealSubscriptionFulfillment{},

		// Chat
		&models.ChatRoom{},
		&models.ChatMessage{},

		// Reliable event backbone: transactional outbox + consumer idempotency
		&models.OutboxEvent{},
		&models.ProcessedEvent{},

		// Escrow conservation ledger: gateway-vs-platform payout drift report (#398)
		&models.PaymentDrift{},

		// Automated vendor payouts (#736, ADR 0002). These live in the
		// domain-free payouts core rather than models/ — the package must not
		// depend on this application's domain types so it stays extractable
		// (#749). Registering them here is the one place the two meet.
		&payouts.LedgerEntry{},
		&payouts.Batch{},
		&payouts.BatchItem{},
		// Batch.MethodID has always pointed here; this is the row it points to.
		// Holds no account number, IFSC or VPA — those stay in Secret Manager
		// (see payouts/method.go), so this table carries only a masked hint, the
		// rail's verdict, and the rail's opaque beneficiary id.
		&payouts.PayoutMethod{},
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
		// One PRIMARY payout destination per (tenant, payee, kind). A partial
		// unique index rather than only application logic, because two primaries
		// would make destination selection depend on row order — and row order is
		// not a thing to pay people by. AutoMigrate cannot express the WHERE.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payout_method_primary ON payout_methods (tenant_id, payee_type, payee_id, kind) WHERE "primary" = true`,
		// One registration per payee per rail — what makes EnsurePayoutMethod's
		// upsert safe under concurrency.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_payout_method_payee_rail ON payout_methods (tenant_id, payee_type, payee_id, rail)`,
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
		// A promo redemption is tied to EITHER an order OR a subscription invoice
		// (#269), so order_id is now nullable. AutoMigrate won't drop the legacy
		// NOT NULL — do it here or subscription redemptions (order_id NULL) fail.
		`ALTER TABLE promo_code_usages ALTER COLUMN order_id DROP NOT NULL`,
		// At most ONE open win-back offer per user (#42) — the DB-level backstop
		// against concurrent triggers (lapse cron + a cancel/suspend webhook, or a
		// retried Razorpay delivery) double-minting offers + platform-funded promos.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_winback_one_open ON winback_offers (user_id) WHERE status = 'offered'`,
		// One live meal subscription per (customer, chef) (#280) — backstops the
		// handler check against concurrent submits so #282 can't double-generate.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_sub_one_live ON meal_subscriptions (customer_id, chef_id) WHERE status <> 'cancelled' AND deleted_at IS NULL`,
		// At most ONE default address per user — the DB-level backstop behind the
		// address handlers' clear-then-set tx, so a customer's active discovery
		// location is never ambiguous. Mirrors migration
		// 20260705000002 (golang-migrate files don't run here; this is the live copy).
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_addresses_one_default_per_user ON addresses (user_id) WHERE is_default`,
		// Mode-aware uniqueness for the tiffin menu tables. The live→test clone
		// (services/test_session_clone.go) copies a kitchen's rows back into the
		// SAME table changing only mode/test_session_id/cloned_from_id, so a
		// unique index that ignores mode rejects the copy: opening a session for
		// any chef holding a weekly menu failed with 23505 on
		// idx_weekly_menus_chef_id.
		//
		// SPLIT rather than simply adding mode to the key, because a closed
		// session's rows survive until an explicit purge — one index keeps the
		// live invariant strict (exactly one live weekly menu per chef), the
		// other scopes test rows to their session so consecutive sessions don't
		// collide with each other's leftovers.
		//
		// AutoMigrate only ever adds indexes, so the mode-blind originals must be
		// dropped here; the GORM tags that created them are gone from
		// models/weekly_menu.go and models/daily_menu.go.
		`DROP INDEX IF EXISTS idx_weekly_menus_chef_id`,
		`DROP INDEX IF EXISTS idx_weekly_cell`,
		`DROP INDEX IF EXISTS idx_daily_menu_chef_date`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_weekly_menus_chef_live ON weekly_menus (chef_id) WHERE mode = 'live'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_weekly_menus_chef_test ON weekly_menus (chef_id, test_session_id) WHERE mode = 'test'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_weekly_cell_live ON weekly_menu_items (chef_id, day_of_week, slot, variant) WHERE mode = 'live'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_weekly_cell_test ON weekly_menu_items (chef_id, day_of_week, slot, variant, test_session_id) WHERE mode = 'test'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_menu_chef_date_live ON daily_menus (chef_id, date) WHERE mode = 'live'`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_menu_chef_date_test ON daily_menus (chef_id, date, test_session_id) WHERE mode = 'test'`,
		// The stranded-delivery sweep (services/delivery_failure_reconcile.go)
		// scans failed/returned rows oldest-first inside a grace window. Without
		// this it is a full scan of every delivery ever made on every tick.
		// AutoMigrate creates the deliveries.updated_at column this indexes.
		`CREATE INDEX IF NOT EXISTS ix_deliveries_status_updated ON deliveries (status, updated_at)`,
	}
	for _, stmt := range postMigrate {
		if err := DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("post-migration DDL failed (%q): %w", stmt, err)
		}
	}

	// #1119 renamed the gateway id fields; AutoMigrate cannot rename, it only ADDS,
	// so an existing database ends up with both columns and every historical id
	// stranded in the one nothing selects. Repair it here, before the unique indexes
	// below so they cover the recovered ids (#1127).
	if err := migrateGatewayIDColumns(DB, gatewayIDColumnRenames()); err != nil {
		return fmt.Errorf("gateway id column migration failed: %w", err)
	}

	if err := normalizeConfiguredProviders(DB); err != nil {
		return err
	}

	// #1125 — a row inserted without an explicit provider must not claim a gateway
	// that can no longer take money. Stated here rather than left to AutoMigrate,
	// which is not reliable about changing an EXISTING column's default. Existing
	// rows are untouched: every one already carries an explicit provider.
	for _, col := range [][2]string{
		{"orders", "payment_provider"},
		{"chef_profiles", "payment_provider"},
		{"delivery_partners", "payment_provider"},
		{"meal_plans", "payment_provider"},
		{"catering_requests", "payment_provider"},
		{"chef_promotions", "payment_provider"},
		{"group_order_participants", "payment_provider"},
		{"meal_subscriptions", "payment_gateway"},
	} {
		if !DB.Migrator().HasColumn(col[0], col[1]) {
			continue
		}
		stmt := fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s SET DEFAULT '%s'`,
			col[0], col[1], models.PreferredChefPaymentProvider)
		if err := DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("post-migration DDL failed (%q): %w", stmt, err)
		}
	}

	// Payment-id uniqueness backstop (#395·1): a DB-level guard against ONE gateway
	// payment being stamped on two orders (the app-logic binding alone can't stop it).
	// PARTIAL indexes (WHERE col <> '') because wallet-only / unpaid / Stripe orders and
	// meal-plan-day shell orders legitimately share the empty default — only real gateway
	// ids must be unique. Group participants (group_order_participants) and catering
	// (catering_requests) live in other tables and are unaffected.
	//
	// Applied NON-FATALLY, unlike the constraints above: those are already handler-enforced
	// (a dup is near-impossible), but payment-id uniqueness was NEVER enforced before, so a
	// legacy duplicate could pre-exist and would make CREATE UNIQUE INDEX fail. A duplicate
	// payment id is a real money bug to investigate + dedup, but crash-looping the whole API
	// over it is disproportionate — log a loud ALERT for ops instead; the settlement
	// reconciliation cron (refund_mismatch / conservation) backstops until it's cleaned and
	// the index is re-attempted on the next boot.
	paymentIDIndexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_gateway_order_id ON orders (gateway_order_id) WHERE gateway_order_id <> ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_gateway_payment_id ON orders (gateway_payment_id) WHERE gateway_payment_id <> ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_stripe_payment_intent_id ON orders (stripe_payment_intent_id) WHERE stripe_payment_intent_id <> ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_plans_gateway_order_id ON meal_plans (gateway_order_id) WHERE gateway_order_id <> ''`,
	}
	for _, stmt := range paymentIDIndexes {
		if err := DB.Exec(stmt).Error; err != nil {
			log.Printf("post-migration ALERT: payment-id unique index not created — a pre-existing DUPLICATE payment id likely exists; investigate + dedup, then redeploy (%q): %v", stmt, err)
		}
	}

	// Backfill SEO slugs for chefs created before the slug column existed (#58),
	// so they're resolvable by slug via GET /chefs/:slug. Idempotent — only fills
	// empty slugs; new/updated chefs get one from ChefProfile.BeforeSave.
	backfillChefSlugs()

	// Seed the Shadowfax 3PL provider (disabled until the owner flips it on).
	SeedShadowfaxProvider(DB)

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
