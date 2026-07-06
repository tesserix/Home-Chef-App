package services

// gateway_idempotency.go — #574. Stable idempotency keys for the Razorpay money-moving
// POSTs so a timeout-AFTER-success retry is deduped by the gateway instead of issuing a
// SECOND real refund/transfer.
//
// Two invariants every key must hold (getting either wrong moves real money):
//  1. IDENTICAL across retries of the SAME logical operation — else a
//     timeout-after-success retry issues a second refund/transfer.
//  2. DISTINCT across genuinely-different operations — else the gateway silently drops
//     the second one (shorting the customer/chef; the #549-class trap).
//
// Because two different operations must never collide, keys are built from an
// operation's IMMUTABLE identity (order id + line id, or order id + prior-refunded
// paise, or destination account) — never from mutable gateway state that advances on a
// successful call (which would change the key on retry and defeat dedup).
//
// Razorpay uses endpoint-specific headers with charset/length limits (verified against
// razorpay.com/docs 2026-07): refunds → X-Refund-Idempotency (>=10 chars, [A-Za-z0-9-_],
// Normal + Instant), direct transfers → X-Transfer-Idempotency (4-36 chars,
// [A-Za-z0-9-_ ]). Colons are illegal and a UUID alone is already 36 chars, so the
// human-readable logical key is normalized to a 32-char hex digest that is valid for
// BOTH endpoints. (Reversals — /transfers/{id}/reversals — do NOT support a gateway
// idempotency key per Razorpay, so ReverseTransfer relies on the AmountReversed cap +
// local persist gate instead; tracked separately.)

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// normalizeIdempotencyKey maps an arbitrary logical operation string to a 32-char
// lowercase hex token: within Razorpay's [A-Za-z0-9-_] charset and both the refund
// (>=10) and transfer (4-36) length windows. Deterministic, so a retry of the same
// logical key produces the same header value.
func normalizeIdempotencyKey(logical string) string {
	sum := sha256.Sum256([]byte(logical))
	return hex.EncodeToString(sum[:])[:32]
}

// RefundFullIdempotencyKey keys a once-per-order full refund (cancellation paths). The
// order transitions to a terminal cancelled/refunded state so exactly one full refund is
// issued; a shared key across the cancellation code paths is deliberate — if two paths
// ever both fire for the same order, the gateway dedups them (never a double refund).
func RefundFullIdempotencyKey(orderID uuid.UUID) string {
	return fmt.Sprintf("refund:%s:full", orderID)
}

// RefundLineIdempotencyKey keys a per-line partial refund by the immutable line-item id.
// Line id (not prior-refunded) is the correct identity here: concurrent refunds of two
// different lines could both read the same prior-refunded amount before either persists,
// so a prior-based key would collide and drop one line's refund.
func RefundLineIdempotencyKey(orderID, lineItemID uuid.UUID) string {
	return fmt.Sprintf("refund:%s:line:%s", orderID, lineItemID)
}

// RefundPartialIdempotencyKey keys an arbitrary-amount (goodwill/partial) refund by the
// order's PRIOR cumulative refunded paise. Stable on retry (RefundAmount only advances on
// commit, after the gateway call) and distinct across sequential partials. Matches the
// wallet-credit idempotency basis (refundWalletIdempotencyKey) so both legs of the same
// refund instance share one identity.
func RefundPartialIdempotencyKey(orderID uuid.UUID, priorRefundedPaise int) string {
	return fmt.Sprintf("refund:%s:%d", orderID, priorRefundedPaise)
}

// TopupIdempotencyKey keys a platform-balance top-up transfer per (order, destination
// account) — the same (order, account) identity the processed_events claim uses (#554),
// so a retried settlement re-derives the same key for each of the chef/driver top-ups.
func TopupIdempotencyKey(orderID uuid.UUID, account string) string {
	return fmt.Sprintf("topup:%s:%s", orderID, account)
}

// HoldPayoutIdempotencyKey keys a per-aggregate on-hold chef payout (meal-plan day or
// group order). One hold per aggregate, guarded by a DB PayoutTransferID; on a
// timeout-after-success retry the DB is still unstamped so the loop re-derives the same
// aggregate key. scope disambiguates the aggregate type ("mealplanday" / "group").
func HoldPayoutIdempotencyKey(scope string, aggID uuid.UUID) string {
	return fmt.Sprintf("hold-payout:%s:%s", scope, aggID)
}
