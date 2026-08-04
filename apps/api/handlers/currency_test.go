package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

// withEmptyCurrencyDB points database.DB at a store with no currencies row, so
// these tests exercise the built-in fallback map rather than seeded data.
func withEmptyCurrencyDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
}

// D-04: the minimum-order rejection was the one customer-facing string that
// hardcoded a dollar sign on an INR-only marketplace — a customer under the
// threshold was told "Minimum order is $199.00". Both rejection sites (a solo
// order and a group order, which also disagreed on decimal places) now resolve
// the symbol through getCurrencySymbol.
func TestCurrencySymbol_PlatformCurrencyIsTheRupee(t *testing.T) {
	withEmptyCurrencyDB(t)
	require.Equal(t, "₹", getCurrencySymbol(services.EarningsCurrency))
	require.Equal(t, "₹", getCurrencySymbol("INR"))
}

// An unknown code falls back to the code itself rather than to a wrong symbol:
// "Minimum order is ZZZ199.00" is confusing, "$199.00" is simply untrue.
func TestCurrencySymbol_UnknownCodePassesThrough(t *testing.T) {
	withEmptyCurrencyDB(t)
	require.Equal(t, "ZZZ", getCurrencySymbol("ZZZ"))
}
