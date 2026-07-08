package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymentDrift is one row of the escrow conservation ledger (#398). The reconcile
// sweep compares the platform's recorded payout-hold state against what the payment
// gateway reports for the hold's transfer, and writes a drift row whenever they
// disagree — most importantly when an aggregate is refunded yet its chef transfer is
// NOT reversed at the gateway (the "chef paid AND customer refunded" alarm). Rows are
// queryable by ops/admin; an OPEN row means an unreconciled discrepancy, a resolved
// row (ResolvedAt set) means a later scan found it consistent again.
//
// Detection only — the sweep never moves money (the payout-reconcile cron is the
// self-heal). One OPEN row per (agg_type, agg_id, kind): a partial unique index makes
// re-detection an idempotent no-op instead of a duplicate.
type PaymentDrift struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// The aggregate the drift belongs to: order / meal-plan-day / group-order + its id.
	AggType string    `gorm:"type:varchar(24);not null;index:idx_payment_drift_open,priority:1" json:"aggType"`
	AggID   uuid.UUID `gorm:"type:uuid;not null;index:idx_payment_drift_open,priority:2" json:"aggId"`

	// Kind classifies the discrepancy (see the PaymentDriftKind constants).
	Kind string `gorm:"type:varchar(40);not null;index:idx_payment_drift_open,priority:3" json:"kind"`

	// Detail is a human-readable one-liner for the ops triage view.
	Detail string `gorm:"type:text" json:"detail"`

	// ExpectedPaise is what the platform's recorded state implies; GatewayPaise is what
	// the gateway reported. Both in paise; 0 when the drift is purely a status mismatch.
	ExpectedPaise int64 `gorm:"default:0" json:"expectedPaise"`
	GatewayPaise  int64 `gorm:"default:0" json:"gatewayPaise"`

	DetectedAt time.Time  `gorm:"autoCreateTime" json:"detectedAt"`
	// ResolvedAt is stamped when a later scan finds the aggregate consistent again — an
	// OPEN drift is ResolvedAt IS NULL. UpdatedAt tracks the last scan that touched it.
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}

// PaymentDriftKind classifies an escrow-ledger discrepancy.
type PaymentDriftKind = string

const (
	// DriftChefPaidOnRefund: the aggregate is (fully) refunded, but its chef transfer is
	// NOT reversed at the gateway (still held or already settled) — the dangerous
	// "chef paid AND customer refunded" case. Highest severity.
	DriftChefPaidOnRefund PaymentDriftKind = "chef_paid_on_refund"
	// DriftHeldButReleasedAtGateway: platform state says the payout is still held
	// (awaiting/eligible/disputed) but the gateway transfer is no longer on hold —
	// the chef may have been paid before the escrow released it.
	DriftHeldButReleasedAtGateway PaymentDriftKind = "held_but_released_at_gateway"
	// DriftReleasedButHeldAtGateway: platform state says released, but the gateway
	// transfer is still on hold — the chef has NOT actually been paid.
	DriftReleasedButHeldAtGateway PaymentDriftKind = "released_but_held_at_gateway"
	// DriftReversedButNotAtGateway: platform state says reversed, but the gateway shows
	// the transfer not (fully) reversed.
	DriftReversedButNotAtGateway PaymentDriftKind = "reversed_but_not_at_gateway"
)
