package models

import (
	"time"

	"github.com/google/uuid"
)

// WalletTxnType is the direction of a ledger entry.
type WalletTxnType string

const (
	WalletCredit WalletTxnType = "credit" // money into the wallet
	WalletDebit  WalletTxnType = "debit"  // money out of the wallet
)

// WalletTxnSource classifies why a ledger entry exists. Credits come from
// refunds, referrals, promos, cashback, or admin adjustments; debits come from
// applying credit at checkout or an admin clawback.
type WalletTxnSource string

const (
	WalletSourceRefund       WalletTxnSource = "refund"
	WalletSourceReferral     WalletTxnSource = "referral"
	WalletSourcePromo        WalletTxnSource = "promo"
	WalletSourceCashback     WalletTxnSource = "cashback"
	WalletSourceOrderPayment WalletTxnSource = "order_payment"
	WalletSourceAdminAdjust  WalletTxnSource = "admin_adjustment"
)

// Wallet is a customer's store-credit balance — one per user. Balance is a
// cached aggregate kept consistent with the WalletTxn ledger inside the same DB
// transaction: the ledger is the source of truth, Balance is the fast read. All
// amounts are in the wallet's Currency (India-first → INR default), stored as
// float64 to match Order totals; convert via services.ToMinor when handing to a
// payment gateway.
type Wallet struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	Balance   float64   `gorm:"not null;default:0" json:"balance"`
	Currency  string    `gorm:"type:varchar(3);not null;default:'INR'" json:"currency"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// WalletTxn is an immutable, append-only ledger entry (no updates, no soft
// delete). The balance is auditable by replaying entries — BalanceAfter is the
// running snapshot after this entry. IdempotencyKey dedupes retried operations
// (e.g. a refund webhook delivered twice, or a double-tapped checkout) so the
// same logical event can never double-credit/double-debit: it is a unique index
// and every entry carries a non-empty key (semantic where one exists, else a
// generated UUID).
type WalletTxn struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	WalletID     uuid.UUID       `gorm:"type:uuid;not null;index" json:"walletId"`
	UserID       uuid.UUID       `gorm:"type:uuid;not null;index" json:"userId"`
	Type         WalletTxnType   `gorm:"type:varchar(10);not null" json:"type"`
	Source       WalletTxnSource `gorm:"type:varchar(20);not null" json:"source"`
	Amount       float64         `gorm:"not null" json:"amount"`       // always positive
	BalanceAfter float64         `gorm:"not null" json:"balanceAfter"` // running snapshot
	Currency     string          `gorm:"type:varchar(3);not null;default:'INR'" json:"currency"`
	OrderID      *uuid.UUID      `gorm:"type:uuid;index" json:"orderId,omitempty"`
	Reason       string          `gorm:"type:text" json:"reason,omitempty"`
	// CreatedBy is the admin user id for admin adjustments; nil for
	// system/customer-initiated entries.
	CreatedBy      *uuid.UUID `gorm:"type:uuid" json:"createdBy,omitempty"`
	IdempotencyKey string     `gorm:"type:varchar(160);uniqueIndex;not null" json:"-"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"createdAt"`
}
