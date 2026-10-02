package services

import (
	"github.com/homechef/api/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestChefHasPayoutMethodUsesMarketDestination(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE chef_profiles (payout_country TEXT, payout_method TEXT, stripe_account_id TEXT, stripe_charges_enabled BOOLEAN, stripe_payouts_enabled BOOLEAN)").Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Exec("CREATE TABLE platform_settings (id TEXT, key TEXT, value TEXT, deleted_at DATETIME)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO platform_settings (id,key,value) VALUES ('00000000-0000-0000-0000-000000000001',?,?)", payoutGateLevelKey, PayoutGateMethodOnFile).Error; err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, country, bank, account string
		charges, payouts, want       bool
	}{
		{"AU ready", "AU", "", "acct_test", true, true, true},
		{"NZ ready", "NZ", "", "acct_test", true, true, true},
		{"normalized AU", " au ", "", "acct_test", true, true, true},
		{"AU bank is not Stripe", "AU", "bank_transfer", "", false, false, false},
		{"NZ charges disabled", "NZ", "", "acct_test", false, true, false},
		{"NZ payouts disabled", "NZ", "", "acct_test", true, false, false},
		{"NZ missing account", "NZ", "", "", true, true, false},
		{"IN bank", "IN", "bank_transfer", "", false, false, true},
		{"legacy IN bank", "", "bank_transfer", "", false, false, true},
		{"IN Stripe is not bank", "IN", "", "acct_test", true, true, false},
		{"unsupported country", "US", "bank_transfer", "acct_test", true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chef := &models.ChefProfile{PayoutCountry: tc.country, PayoutMethod: tc.bank, StripeAccountID: tc.account, StripeChargesEnabled: tc.charges, StripePayoutsEnabled: tc.payouts}
			if got := ChefHasPayoutMethod(chef); got != tc.want {
				t.Fatalf("ready=%v, want %v", got, tc.want)
			}
			if err := db.Exec("DELETE FROM chef_profiles").Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Exec("INSERT INTO chef_profiles VALUES (?, ?, ?, ?, ?)", tc.country, tc.bank, tc.account, tc.charges, tc.payouts).Error; err != nil {
				t.Fatal(err)
			}
			for _, mode := range []PayoutGateLevel{PayoutGateMethodOnFile, PayoutGateVerified} {
				if err := db.Exec("UPDATE platform_settings SET value = ? WHERE key = ?", mode, payoutGateLevelKey).Error; err != nil {
					t.Fatal(err)
				}
				predicate, enforced := PayoutGateOpenFilter(db, time.Now())
				if !enforced {
					t.Fatal("configured gate must be enforced")
				}
				var count int64
				if err := db.Table("chef_profiles").Where(predicate).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if (count == 1) != tc.want {
					t.Fatalf("SQL gate %s ready=%v, want %v", mode, count == 1, tc.want)
				}
			}
		})
	}
	if ChefHasPayoutMethod(nil) {
		t.Fatal("nil chef must not be ready")
	}
}
