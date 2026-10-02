package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func TestDefaultAddressPhoneCountry(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE addresses (id TEXT PRIMARY KEY, user_id TEXT, country TEXT, is_default INTEGER, created_at DATETIME)`).Error)

	au, legacy, none := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO addresses VALUES (?,?,?,?,?)`, uuid.NewString(), au, "IN", 0, "2026-01-01").Error)
	require.NoError(t, db.Exec(`INSERT INTO addresses VALUES (?,?,?,?,?)`, uuid.NewString(), au, "AU", 1, "2026-01-02").Error)
	require.NoError(t, db.Exec(`INSERT INTO addresses VALUES (?,?,?,?,?)`, uuid.NewString(), legacy, "US", 1, "2026-01-01").Error)

	require.Equal(t, "AU", DefaultAddressPhoneCountry(db, au))
	require.Equal(t, "IN", DefaultAddressPhoneCountry(db, legacy), "unsupported legacy country falls back to India")
	require.Equal(t, "IN", DefaultAddressPhoneCountry(db, none))
}
