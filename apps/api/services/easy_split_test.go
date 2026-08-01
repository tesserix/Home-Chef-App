package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupEasySplitDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT UNIQUE, value TEXT,
		type TEXT DEFAULT 'string', updated_by TEXT, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_documents (id TEXT PRIMARY KEY, chef_id TEXT, type TEXT,
		file_name TEXT DEFAULT '', file_path TEXT DEFAULT '', bucket TEXT DEFAULT '', content_type TEXT DEFAULT '',
		file_size INTEGER DEFAULT 0, status TEXT DEFAULT 'pending', rejection_reason TEXT DEFAULT '',
		expiry_date DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error)

	// IsChefFSSAIExpired reads the GLOBAL handle, so the split guard needs it
	// pointed at this test database.
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func setEasySplitSetting(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO platform_settings (id, key, value) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		uuid.NewString(), key, value).Error)
}

func easySplitOrder() *models.Order {
	return &models.Order{
		Subtotal: 500, Tax: 25, ChefTip: 20, DeliveryFee: 40,
		CommissionRate: 6, Total: 585,
		Chef: models.ChefProfile{
			ID:                   uuid.New(),
			CashfreeVendorID:     "hc_vendor",
			CashfreeVendorStatus: CashfreeVendorActive,
			PayoutCountry:        "IN",
		},
	}
}

func TestBuildOrderSplit(t *testing.T) {
	db := setupEasySplitDB(t)
	order := easySplitOrder()
	capture := ToPaise(order.Total)

	// Flag off → no split.
	require.Nil(t, BuildOrderSplit(db, order, capture, 0))

	setEasySplitSetting(t, db, SettingEasySplitEnabled, "true")
	setEasySplitSetting(t, db, SettingPlatformFeeFlatMinor, "1000") // ₹10 flat

	split := BuildOrderSplit(db, order, capture, 0)
	require.NotNil(t, split)
	require.Equal(t, "hc_vendor", split.VendorID)
	expected := int64(ToPaise(ChefNetPayoutFor(order))) - 1000
	require.EqualValues(t, expected, int64(split.AmountPaise.Paise()))
	require.Less(t, split.AmountPaise.Paise(), capture)

	// Credit-funded orders settle through statements, never a partial split.
	require.Nil(t, BuildOrderSplit(db, order, capture-5000, 5000))

	// Vendor not yet verified → full capture.
	pending := easySplitOrder()
	pending.Chef.CashfreeVendorStatus = CashfreeVendorInBeneCreation
	require.Nil(t, BuildOrderSplit(db, pending, capture, 0))
	unregistered := easySplitOrder()
	unregistered.Chef.CashfreeVendorID = ""
	require.Nil(t, BuildOrderSplit(db, unregistered, capture, 0))

	// Lapsed FSSAI: the payout is withheld, so the money must stay platform-side.
	lapsed := easySplitOrder()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_documents (id, chef_id, type, status, expiry_date) VALUES (?, ?, ?, ?, ?)`,
		uuid.NewString(), lapsed.Chef.ID.String(), string(models.DocFSSAILicense),
		string(models.DocStatusVerified), time.Now().Add(-24*time.Hour)).Error)
	require.Nil(t, BuildOrderSplit(db, lapsed, capture, 0))

	// Unreadable fee fails closed to full capture.
	setEasySplitSetting(t, db, SettingPlatformFeeFlatMinor, "ten rupees")
	require.Nil(t, BuildOrderSplit(db, order, capture, 0))

	// Fee larger than the share → nothing sensible to split.
	setEasySplitSetting(t, db, SettingPlatformFeeFlatMinor, "99999900")
	require.Nil(t, BuildOrderSplit(db, order, capture, 0))
}

func TestPlatformFeeFlatMinor(t *testing.T) {
	db := setupEasySplitDB(t)

	fee, ok := PlatformFeeFlatMinor(db)
	require.True(t, ok)
	require.EqualValues(t, 0, fee)

	setEasySplitSetting(t, db, SettingPlatformFeeFlatMinor, "2500")
	fee, ok = PlatformFeeFlatMinor(db)
	require.True(t, ok)
	require.EqualValues(t, 2500, fee)

	setEasySplitSetting(t, db, SettingPlatformFeeFlatMinor, "-5")
	_, ok = PlatformFeeFlatMinor(db)
	require.False(t, ok)
}
