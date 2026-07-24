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

// LedgerAccountKind identifies a normalized account in the double-entry ledger. A user's
// spendable wallet is `user_wallet` (scoped by UserID); the rest are system counterparty
// accounts so every transaction balances (Σdebit == Σcredit). Buckets (refund/referral/
// promo…) are layered on in a later phase as distinct user account kinds.
type LedgerAccountKind string

const (
	LedgerAcctUserWallet     LedgerAccountKind = "user_wallet"      // a customer's spendable balance
	LedgerAcctSystemRefund   LedgerAccountKind = "system_refund"    // funds a refund credit
	LedgerAcctSystemReferral LedgerAccountKind = "system_referral"  // funds a referral reward
	LedgerAcctSystemPromo    LedgerAccountKind = "system_promo"     // funds a promo/cashback credit
	LedgerAcctSystemSpend    LedgerAccountKind = "system_spend"     // counterparty when a wallet pays for an order
	LedgerAcctSystemAdjust   LedgerAccountKind = "system_adjust"    // admin adjustment counterparty
	LedgerAcctSystemOpening  LedgerAccountKind = "system_opening"   // migration opening balances
)

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
// their `user_wallet` entries.
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
