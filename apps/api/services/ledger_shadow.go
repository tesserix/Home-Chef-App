package services

// ledger_shadow.go — Phase 1 shadow layer (docs/wallet-ledger-plan.md). Runs the
// double-entry ledger ALONGSIDE the legacy float wallet so it can be proven before any
// read flips over. Gated by config.LedgerShadowEnabled (default OFF) — zero behaviour
// change until enabled, and the ledger schema (ledger_transactions/ledger_entries) must
// be deployed via tesserix-k8s first.
//
//   1. Dual-write: mirrorWalletTxnToLedger posts every wallet credit/debit into the
//      ledger in the SAME tx as the legacy write, so they commit atomically.
//   2. Backfill: BackfillLedgerOpeningBalances seeds each wallet's current balance as an
//      opening entry so the ledger projection starts equal to the legacy balance.
//   3. Reconcile: ReconcileLedgerVsWallet asserts ledger balance == legacy balance and
//      reports drift (never auto-corrects), run periodically by the ledger-reconcile cron.

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// LedgerShadowActive reports whether the wallet→ledger dual-write is switched on.
func LedgerShadowActive() bool {
	return config.AppConfig != nil && config.AppConfig.LedgerShadowEnabled
}

// ledgerCounterpartyFor maps a legacy wallet source to the ledger system account that
// balances the user-side leg (so every transaction has Σdebit == Σcredit).
func ledgerCounterpartyFor(s models.WalletTxnSource) models.LedgerAccountKind {
	switch s {
	case models.WalletSourceRefund:
		return models.LedgerAcctSystemRefund
	case models.WalletSourceReferral:
		return models.LedgerAcctSystemReferral
	case models.WalletSourcePromo, models.WalletSourceCashback, models.WalletSourceLoyalty:
		return models.LedgerAcctSystemPromo
	case models.WalletSourceOrderPayment:
		return models.LedgerAcctSystemSpend
	default: // admin_adjustment + any future source
		return models.LedgerAcctSystemAdjust
	}
}

// ledgerBucketFor maps a wallet CREDIT's source to the user bucket it lands in (Phase 2). The
// bucket preserves provenance so value can later be spent by priority, expired, or restored to
// its original tender on refund. Only ever consulted for credits — a debit drains buckets by
// priority instead (userWalletDebitLegs).
func ledgerBucketFor(s models.WalletTxnSource) models.LedgerAccountKind {
	switch s {
	case models.WalletSourceRefund:
		return models.LedgerAcctUserRefund
	case models.WalletSourceReferral:
		return models.LedgerAcctUserReferral
	case models.WalletSourcePromo:
		return models.LedgerAcctUserPromo
	case models.WalletSourceCashback, models.WalletSourceLoyalty:
		return models.LedgerAcctUserCashback
	case models.WalletSourceAdminAdjust:
		return models.LedgerAcctUserGoodwill
	default: // OrderPayment is never a credit; anything unknown → generic bucket
		return models.LedgerAcctUserWallet
	}
}

// walletSpendPriority is the order buckets are drained when the wallet is spent: most
// promotional / expiring value first, refund value (the most cash-like, most likely to become
// withdrawable) last. A policy-driven priority (platform_settings) replaces this default in a
// later phase; it is centralized here so that change is one function.
func walletSpendPriority() []models.LedgerAccountKind {
	return []models.LedgerAccountKind{
		models.LedgerAcctUserPromo,
		models.LedgerAcctUserCashback,
		models.LedgerAcctUserReferral,
		models.LedgerAcctUserGoodwill,
		models.LedgerAcctUserWallet, // generic / opening
		models.LedgerAcctUserRefund,
	}
}

// userWalletDebitLegs splits a debit across the user's buckets in spending priority, taking as
// much as each bucket holds before moving to the next, so no bucket goes negative and expiring
// value is spent first. Returns one debit leg per drained bucket; the caller balances them with
// the system counterparty. Runs in the caller's tx (after the wallet row lock), so the bucket
// reads are consistent with the balance being debited.
func userWalletDebitLegs(tx *gorm.DB, userID uuid.UUID, amount models.Money) ([]LedgerLeg, error) {
	balances, err := LedgerUserBucketBalances(tx, userID)
	if err != nil {
		return nil, err
	}
	uid := userID
	remaining := amount
	var legs []LedgerLeg
	for _, kind := range walletSpendPriority() {
		if remaining <= 0 {
			break
		}
		avail := balances[kind]
		if avail <= 0 {
			continue
		}
		take := avail
		if take > remaining {
			take = remaining
		}
		legs = append(legs, LedgerLeg{AccountKind: kind, UserID: &uid, Direction: models.LedgerDebit, Amount: take})
		remaining -= take
	}
	// Buckets should always cover the debit — the legacy balance check already guaranteed
	// sufficient funds and the ledger tracks it to the paise. If they somehow don't (pre-existing
	// drift), put the shortfall on the generic bucket so the transaction still balances; the
	// reconcile sweep surfaces the drift. We never strand a leg or post an unbalanced tx.
	if remaining > 0 {
		legs = append(legs, LedgerLeg{AccountKind: models.LedgerAcctUserWallet, UserID: &uid, Direction: models.LedgerDebit, Amount: remaining})
	}
	return legs, nil
}

// mirrorWalletTxnToLedger posts a legacy wallet_txn into the double-entry ledger, in the
// caller's tx. A credit lands in its source bucket (refund→refund, promo→promo…), balanced by
// the system counterparty that funded it; a debit drains buckets in spending priority, balanced
// by one system-counterparty credit. Reuses the wallet_txn's idempotency key so it posts exactly
// once even across retries (and, being in the same tx, it commits atomically with the legacy
// write).
func mirrorWalletTxnToLedger(tx *gorm.DB, e *models.WalletTxn) error {
	counter := ledgerCounterpartyFor(e.Source)
	amt := models.RupeesToMoney(e.Amount)
	uid := e.UserID
	refID := ""
	if e.OrderID != nil {
		refID = e.OrderID.String()
	}

	var legs []LedgerLeg
	if e.Type == models.WalletCredit {
		legs = []LedgerLeg{
			{AccountKind: ledgerBucketFor(e.Source), UserID: &uid, Direction: models.LedgerCredit, Amount: amt},
			{AccountKind: counter, Direction: models.LedgerDebit, Amount: amt},
		}
	} else {
		debitLegs, err := userWalletDebitLegs(tx, uid, amt)
		if err != nil {
			return err
		}
		legs = append(debitLegs, LedgerLeg{AccountKind: counter, Direction: models.LedgerCredit, Amount: amt})
	}

	_, err := PostLedgerTransaction(tx, e.IdempotencyKey, e.Reason, string(e.Source), refID, legs)
	return err
}

// BackfillLedgerOpeningBalances seeds an OPENING entry per wallet equal to the gap between
// its legacy balance and its current ledger balance, so the ledger projection becomes
// exactly the legacy balance. The `legacy − ledger` delta makes it correct regardless of
// whether dual-write started before or after this runs, and it is idempotent (one opening
// per user, keyed ledger-opening:<uid>). Returns how many openings it posted.
func BackfillLedgerOpeningBalances(db *gorm.DB) (int, error) {
	var wallets []models.Wallet
	if err := db.Find(&wallets).Error; err != nil {
		return 0, err
	}
	n := 0
	for i := range wallets {
		w := wallets[i]
		ledgerBal, err := LedgerUserBalance(db, w.UserID)
		if err != nil {
			log.Printf("ledger-backfill: ledger balance for %s failed: %v", w.UserID, err)
			continue
		}
		delta := models.RupeesToMoney(w.Balance) - ledgerBal
		if delta <= 0 {
			continue // already reconciled (or nothing to open)
		}
		uid := w.UserID
		legs := []LedgerLeg{
			{AccountKind: models.LedgerAcctUserWallet, UserID: &uid, Direction: models.LedgerCredit, Amount: delta},
			{AccountKind: models.LedgerAcctSystemOpening, Direction: models.LedgerDebit, Amount: delta},
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			_, e := PostLedgerTransaction(tx, "ledger-opening:"+w.UserID.String(),
				"opening balance backfill", "opening", w.UserID.String(), legs)
			return e
		}); err != nil {
			log.Printf("ledger-backfill: post opening for %s failed: %v", w.UserID, err)
			continue
		}
		n++
	}
	return n, nil
}

// LedgerDrift is a per-wallet mismatch between the legacy float balance and the ledger.
type LedgerDrift struct {
	UserID      uuid.UUID
	LegacyMinor models.Money
	LedgerMinor models.Money
}

// ReconcileLedgerVsWallet returns every wallet whose ledger projection differs from its
// legacy balance. It NEVER corrects — a mismatch is surfaced for investigation (SEV-1).
func ReconcileLedgerVsWallet(db *gorm.DB) ([]LedgerDrift, error) {
	var wallets []models.Wallet
	if err := db.Find(&wallets).Error; err != nil {
		return nil, err
	}
	var drift []LedgerDrift
	for i := range wallets {
		w := wallets[i]
		lb, err := LedgerUserBalance(db, w.UserID)
		if err != nil {
			return nil, err
		}
		if legacy := models.RupeesToMoney(w.Balance); lb != legacy {
			drift = append(drift, LedgerDrift{UserID: w.UserID, LegacyMinor: legacy, LedgerMinor: lb})
		}
	}
	return drift, nil
}

const ledgerReconcileInterval = 30 * time.Minute

func runLedgerReconcileScan(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ledger-reconcile: panic recovered: %v", r)
		}
	}()
	if !LedgerShadowActive() {
		return // shadow off → nothing to reconcile
	}
	// Self-seed openings so a wallet that existed before dual-write reconciles. Idempotent (one
	// opening per user, keyed ledger-opening:<uid>) — it posts nothing once every wallet is
	// seeded — so running it each cycle just guarantees the shadow phase CONVERGES on its own,
	// without a separate manual backfill call. (POST /admin/ledger/backfill triggers it
	// immediately when the operator wants the seed the moment the flag flips.)
	if n, err := BackfillLedgerOpeningBalances(database.DB); err != nil {
		log.Printf("ledger-reconcile: opening backfill failed: %v", err)
	} else if n > 0 {
		log.Printf("ledger-reconcile: seeded %d wallet opening balance(s)", n)
	}
	drift, err := ReconcileLedgerVsWallet(database.DB)
	if err != nil {
		log.Printf("ledger-reconcile: scan failed: %v", err)
		return
	}
	for _, d := range drift {
		log.Printf("ledger-reconcile: DRIFT user=%s legacy=%dp ledger=%dp", d.UserID, d.LegacyMinor, d.LedgerMinor)
	}
	if len(drift) > 0 {
		log.Printf("ledger-reconcile: %d wallet(s) drifted between legacy balance and ledger — investigate, do NOT auto-correct", len(drift))
	}
}

func StartLedgerReconcileCron(ctx context.Context) {
	go func() {
		t := time.NewTicker(ledgerReconcileInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runLedgerReconcileScan(ctx)
			}
		}
	}()
}
