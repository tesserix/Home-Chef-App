package services

// ledger_buckets_test.go — Phase 2 (docs/wallet-ledger-plan.md). A user's value is split into
// provenance buckets (refund/referral/promo/goodwill/cashback), yet the customer still has ONE
// balance. Credits land in their source bucket; debits drain buckets in spending priority
// (promo/cashback first, refund last) so no bucket goes negative — and through it all the summed
// balance keeps tracking the legacy float balance to the paise.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// Credits route to the bucket their source dictates; the per-bucket breakdown reflects it; the
// single balance is the sum and equals the legacy float balance.
func TestLedgerBuckets_CreditsRouteToBuckets(t *testing.T) {
	ledgerShadow(t, true)
	db := setupShadowDB(t)
	u := uuid.New()

	credit := func(amt float64, src models.WalletTxnSource, key string) {
		t.Helper()
		_, err := CreditWallet(db, u, amt, src, nil, string(src), key, nil)
		require.NoError(t, err)
	}
	credit(200, models.WalletSourceRefund, "c-refund")
	credit(100, models.WalletSourceReferral, "c-referral")
	credit(50, models.WalletSourcePromo, "c-promo")
	credit(30, models.WalletSourceAdminAdjust, "c-admin")
	credit(20, models.WalletSourceCashback, "c-cashback")
	credit(10, models.WalletSourceLoyalty, "c-loyalty")

	buckets, err := LedgerUserBucketBalances(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(20000), buckets[models.LedgerAcctUserRefund])
	require.Equal(t, models.Money(10000), buckets[models.LedgerAcctUserReferral])
	require.Equal(t, models.Money(5000), buckets[models.LedgerAcctUserPromo])
	require.Equal(t, models.Money(3000), buckets[models.LedgerAcctUserGoodwill], "admin adjustment → goodwill")
	require.Equal(t, models.Money(3000), buckets[models.LedgerAcctUserCashback], "cashback + loyalty share the cashback bucket")

	// One balance = sum of every bucket.
	bal, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(41000), bal, "200+100+50+30+20+10 = ₹410")

	// And it still tracks the legacy float balance to the paise.
	var w models.Wallet
	require.NoError(t, db.First(&w, "user_id = ?", u).Error)
	require.Equal(t, models.RupeesToMoney(w.Balance), bal)
	drift, err := ReconcileLedgerVsWallet(db)
	require.NoError(t, err)
	require.Empty(t, drift)
}

// A spend drains the highest-priority bucket first, spilling into the next only for the
// remainder — promo before refund — and no bucket goes negative.
func TestLedgerBuckets_DebitDrainsPromoBeforeRefund(t *testing.T) {
	ledgerShadow(t, true)
	db := setupShadowDB(t)
	u := uuid.New()

	_, err := CreditWallet(db, u, 100, models.WalletSourcePromo, nil, "promo", "c1", nil)
	require.NoError(t, err)
	_, err = CreditWallet(db, u, 200, models.WalletSourceRefund, nil, "refund", "c2", nil)
	require.NoError(t, err)

	// Spend ₹150: promo (₹100) drains fully, then ₹50 from refund.
	_, err = DebitWallet(db, u, 150, models.WalletSourceOrderPayment, nil, "order", "d1", nil)
	require.NoError(t, err)

	buckets, err := LedgerUserBucketBalances(db, u)
	require.NoError(t, err)
	require.Zero(t, buckets[models.LedgerAcctUserPromo], "promo fully spent first")
	require.Equal(t, models.Money(15000), buckets[models.LedgerAcctUserRefund], "refund keeps ₹200−₹50")

	bal, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(15000), bal)

	d, c, err := LedgerTotals(db)
	require.NoError(t, err)
	require.Equal(t, d, c, "double-entry holds after a split debit")

	drift, err := ReconcileLedgerVsWallet(db)
	require.NoError(t, err)
	require.Empty(t, drift, "summed balance still tracks legacy after a bucketed spend")
}

// A single spend that exceeds two buckets spans a third — draining strictly in priority order
// (promo → cashback → … → refund) and leaving only refund with a remainder.
func TestLedgerBuckets_DebitSpansBucketsInPriority(t *testing.T) {
	ledgerShadow(t, true)
	db := setupShadowDB(t)
	u := uuid.New()

	credit := func(amt float64, src models.WalletTxnSource, key string) {
		t.Helper()
		_, err := CreditWallet(db, u, amt, src, nil, string(src), key, nil)
		require.NoError(t, err)
	}
	credit(30, models.WalletSourcePromo, "c1")    // priority 1
	credit(20, models.WalletSourceCashback, "c2") // priority 2
	credit(100, models.WalletSourceRefund, "c3")  // last

	// Spend ₹70: promo ₹30 + cashback ₹20 + refund ₹20.
	_, err := DebitWallet(db, u, 70, models.WalletSourceOrderPayment, nil, "order", "d1", nil)
	require.NoError(t, err)

	buckets, err := LedgerUserBucketBalances(db, u)
	require.NoError(t, err)
	require.Zero(t, buckets[models.LedgerAcctUserPromo])
	require.Zero(t, buckets[models.LedgerAcctUserCashback])
	require.Equal(t, models.Money(8000), buckets[models.LedgerAcctUserRefund], "refund keeps ₹100−₹20")

	bal, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(8000), bal)

	d, c, err := LedgerTotals(db)
	require.NoError(t, err)
	require.Equal(t, d, c)
}
