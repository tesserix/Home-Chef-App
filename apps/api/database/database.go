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
		// Users & Auth
		&models.User{},
		&models.RefreshToken{},
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

		// Admin
		&models.PlatformSettings{},
		&models.AuditLog{},

		// Currency
		&models.Currency{},
		&models.ExchangeRate{},

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
	)

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
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
