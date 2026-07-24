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
// balances the user-wallet leg (so every transaction has Σdebit == Σcredit).
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

// mirrorWalletTxnToLedger posts a legacy wallet_txn into the double-entry ledger, in the
// caller's tx. A credit → user_wallet CREDIT + counterparty DEBIT; a debit → the reverse.
// Reuses the wallet_txn's idempotency key so it posts exactly once even across retries
// (and, being in the same tx, it commits atomically with the legacy write).
func mirrorWalletTxnToLedger(tx *gorm.DB, e *models.WalletTxn) error {
	counter := ledgerCounterpartyFor(e.Source)
	amt := models.RupeesToMoney(e.Amount)
	uid := e.UserID
	userLeg := LedgerLeg{AccountKind: models.LedgerAcctUserWallet, UserID: &uid, Amount: amt}
	sysLeg := LedgerLeg{AccountKind: counter, Amount: amt}
	if e.Type == models.WalletCredit {
		userLeg.Direction, sysLeg.Direction = models.LedgerCredit, models.LedgerDebit
	} else {
		userLeg.Direction, sysLeg.Direction = models.LedgerDebit, models.LedgerCredit
	}
	refID := ""
	if e.OrderID != nil {
		refID = e.OrderID.String()
	}
	_, err := PostLedgerTransaction(tx, e.IdempotencyKey, e.Reason, string(e.Source), refID,
		[]LedgerLeg{userLeg, sysLeg})
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
