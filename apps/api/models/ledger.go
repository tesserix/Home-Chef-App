package models

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// Money is an amount in the smallest currency unit — paise for INR: ₹250.75 == 25075.
// Financial values are ALWAYS stored/computed as integers, never float64, so there is no
// rounding drift. Convert at the boundary with RupeesToMoney / Rupees.
type Money int64

// RupeesToMoney converts a rupee float (legacy representation) to paise, rounding to the
// nearest paise. Used only at the boundary while the legacy float wallet is migrated.
func RupeesToMoney(r float64) Money { return Money(math.Round(r * 100)) }

// Rupees renders the amount as a rupee float (for legacy interop / display only).
func (m Money) Rupees() float64 { return float64(m) / 100 }

// LedgerAccountKind identifies a normalized account in the double-entry ledger. User-side
// accounts (scoped by UserID) hold a customer's spendable value; the rest are system
// counterparty accounts so every transaction balances (Σdebit == Σcredit).
//
// A user's value is split into BUCKETS (Phase 2, docs/wallet-ledger-plan.md) — refund,
// referral, promo, goodwill, cashback — each a distinct user account kind with its own
// provenance and (later) rules. `user_wallet` is the generic/opening bucket: migration
// opening balances and any unclassified credit. The customer sees ONE balance (the sum of
// all their buckets, LedgerUserBalance); the per-bucket split is internal (spend priority,
// expiry, source-preserving refunds). Every user-side kind must satisfy IsUserWalletKind.
type LedgerAccountKind string

const (
	// User-side spendable buckets (all scoped by UserID; sum = available balance).
	LedgerAcctUserWallet   LedgerAccountKind = "user_wallet"          // generic / opening / unclassified
	LedgerAcctUserRefund   LedgerAccountKind = "user_wallet_refund"   // refunded order/meal-plan value
	LedgerAcctUserReferral LedgerAccountKind = "user_wallet_referral" // referral rewards
	LedgerAcctUserPromo    LedgerAccountKind = "user_wallet_promo"    // promo credits
	LedgerAcctUserGoodwill LedgerAccountKind = "user_wallet_goodwill" // admin/goodwill grants
	LedgerAcctUserCashback LedgerAccountKind = "user_wallet_cashback" // cashback / loyalty rewards
	// User-side RESERVED funds — held for an in-flight order/payment (Phase 3). Still the
	// user's money (counts toward total balance) but NOT spendable: a hold moves value here
	// from the spendable buckets; capture spends it, release restores it to its buckets.
	LedgerAcctUserHeld LedgerAccountKind = "user_wallet_held"

	// System counterparty accounts (UserID nil).
	LedgerAcctSystemRefund   LedgerAccountKind = "system_refund"   // funds a refund credit
	LedgerAcctSystemReferral LedgerAccountKind = "system_referral" // funds a referral reward
	LedgerAcctSystemPromo    LedgerAccountKind = "system_promo"    // funds a promo/cashback credit
	LedgerAcctSystemSpend    LedgerAccountKind = "system_spend"    // counterparty when a wallet pays for an order
	LedgerAcctSystemAdjust   LedgerAccountKind = "system_adjust"   // admin adjustment counterparty
	LedgerAcctSystemOpening  LedgerAccountKind = "system_opening"  // migration opening balances
)

// UserWalletKinds lists every user-side account kind (spendable buckets + held). Their
// per-user sum is the customer's TOTAL balance (available + held).
func UserWalletKinds() []LedgerAccountKind {
	return []LedgerAccountKind{
		LedgerAcctUserWallet, LedgerAcctUserRefund, LedgerAcctUserReferral,
		LedgerAcctUserPromo, LedgerAcctUserGoodwill, LedgerAcctUserCashback,
		LedgerAcctUserHeld,
	}
}

// IsUserWalletKind reports whether an account kind is user-side (scoped by UserID, counted
// toward the customer's balance) rather than a system counterparty. Includes the held account.
func IsUserWalletKind(k LedgerAccountKind) bool {
	switch k {
	case LedgerAcctUserWallet, LedgerAcctUserRefund, LedgerAcctUserReferral,
		LedgerAcctUserPromo, LedgerAcctUserGoodwill, LedgerAcctUserCashback,
		LedgerAcctUserHeld:
		return true
	}
	return false
}

// LedgerDirection is the side of an entry.
type LedgerDirection string

const (
	LedgerDebit  LedgerDirection = "debit"
	LedgerCredit LedgerDirection = "credit"
)

// LedgerTransaction is one atomic, IMMUTABLE posting — a set of balanced entries recorded
// together. It is never updated or deleted; a correction posts a new reversing transaction.
// Idempotent on IdempotencyKey (the same business event posts exactly once).
//
// NOTE: production schema lives in tesserix-k8s (db-schema-bootstrap); this GORM model is
// the app-side read/write surface + local-dev AutoMigrate.
type LedgerTransaction struct {
	ID             uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       string        `gorm:"type:varchar(64);index" json:"tenantId"`
	IdempotencyKey string        `gorm:"type:varchar(200);uniqueIndex;not null" json:"-"`
	Reason         string        `gorm:"type:text" json:"reason,omitempty"`
	RefType        string        `gorm:"type:varchar(32);index" json:"refType,omitempty"` // order / refund / referral / opening / adjustment
	RefID          string        `gorm:"type:varchar(64);index" json:"refId,omitempty"`
	CreatedAt      time.Time     `gorm:"autoCreateTime" json:"createdAt"`
	Entries        []LedgerEntry `gorm:"foreignKey:TransactionID" json:"entries,omitempty"`
}

// LedgerEntry is one leg (debit or credit) of a transaction. Immutable. AmountMinor is
// always positive; Direction carries the sign. A user's balance is Σcredits − Σdebits over
// all of their user-side (bucket) entries — every leg with their UserID set.
type LedgerEntry struct {
	ID            uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TransactionID uuid.UUID         `gorm:"type:uuid;index;not null" json:"transactionId"`
	AccountKind   LedgerAccountKind `gorm:"type:varchar(32);not null;index" json:"accountKind"`
	UserID        *uuid.UUID        `gorm:"type:uuid;index" json:"userId,omitempty"` // set for user_wallet accounts
	Direction     LedgerDirection   `gorm:"type:varchar(6);not null" json:"direction"`
	AmountMinor   Money             `gorm:"type:bigint;not null" json:"amountMinor"` // paise, > 0
	Currency      string            `gorm:"type:varchar(3);not null;default:'INR'" json:"currency"`
	CreatedAt     time.Time         `gorm:"autoCreateTime" json:"createdAt"`
}
