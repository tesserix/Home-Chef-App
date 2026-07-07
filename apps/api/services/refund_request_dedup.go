package services

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// refund_request_dedup.go — #600. Dedup a CLIENT-VISIBLE refund submission so a retry AFTER a
// successful end-to-end round-trip doesn't issue a SECOND real refund.
//
// The gateway idempotency key (#574) is derived from the order's PRIOR cumulative refunded paise,
// which dedups only the narrow server↔gateway timeout window (a retry re-reads the same
// UNPERSISTED RefundAmount → same key). Once a partial refund fully succeeds, RefundAmount
// advances, so the app re-submitting the identical "refund ₹100" (its own HTTP response was
// dropped) computes a NEW gateway key AND re-runs the local reserve → a genuinely new double
// refund. This claim closes that gap BEFORE the reserve, so a duplicate issues neither a second
// gateway refund nor a second local refund_amount increment.

// refundDedupConsumer namespaces refund-request claims in the shared processed_events ledger.
const refundDedupConsumer = "refund-request"

// refundDedupWindow bounds the FALLBACK (order+amount+reason) dedup. It must comfortably cover a
// client retry-after-dropped-response (seconds to a minute or two of user re-tap / app backoff)
// while still allowing a genuinely-intended IDENTICAL repeat refund afterwards. A client-supplied
// Idempotency-Key is not windowed — it dedups exactly, so distinct keys are always distinct
// requests. Over-blocking a legit identical repeat within the window is recoverable (retry after
// it, or send a distinct Idempotency-Key); a second real refund is not — so the window errs long.
const refundDedupWindow = 5 * time.Minute

// maxRefundClientKey caps the accepted Idempotency-Key length (it is hashed regardless, but a
// pathologically long header shouldn't be copied around).
const maxRefundClientKey = 200

// refundDedupKey builds the processed_events.msg_id (varchar(64)). A client Idempotency-Key keys
// exactly + permanently; the (order, amount, reason) fallback keys within refundDedupWindow.
// Both are SHA-256 hashed so they always fit the column. Returns windowed=false for a client key.
func refundDedupKey(orderID uuid.UUID, clientKey string, amountPaise int64, reason string) (key string, windowed bool) {
	if clientKey != "" {
		sum := sha256.Sum256([]byte("c:" + orderID.String() + ":" + clientKey))
		return "rc:" + base64.RawURLEncoding.EncodeToString(sum[:]), false
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("a:%s:%d:%s", orderID, amountPaise, strings.TrimSpace(reason))))
	return "ra:" + base64.RawURLEncoding.EncodeToString(sum[:]), true
}

// NormalizeRefundClientKey trims + length-caps a client Idempotency-Key header. Empty ⇒ no client
// key (the caller falls back to the windowed order+amount+reason dedup).
func NormalizeRefundClientKey(header string) string {
	k := strings.TrimSpace(header)
	if len(k) > maxRefundClientKey {
		k = k[:maxRefundClientKey]
	}
	return k
}

// ClaimRefundRequest claims a refund submission so a duplicate (client retry) is deduped before it
// reaches the reserve + gateway. Returns proceed=true for the FIRST submission (the caller does
// the refund) and false for a duplicate (the caller returns the current state without refunding).
// dedupKey is the claim id — pass it to ReleaseRefundRequestClaim on a PRE-COMMIT failure so a
// legit retry can re-attempt. Atomic (INSERT … ON CONFLICT DO NOTHING + a guarded windowed
// refresh) — no check-then-act race under a concurrent double-submit.
func ClaimRefundRequest(db *gorm.DB, orderID uuid.UUID, clientKey string, amountPaise int64, reason string) (proceed bool, dedupKey string, err error) {
	key, windowed := refundDedupKey(orderID, clientKey, amountPaise, reason)
	rec := &models.ProcessedEvent{Consumer: refundDedupConsumer, MsgID: key, Subject: "order.refund.request"}
	res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(rec)
	if res.Error != nil {
		return false, key, res.Error
	}
	if res.RowsAffected == 1 {
		return true, key, nil // first claim — proceed
	}
	// Conflict: a prior claim exists.
	if !windowed {
		return false, key, nil // a client Idempotency-Key was already used → duplicate
	}
	// Windowed fallback: allow (and refresh) only if the prior claim is older than the window — a
	// genuinely-intended identical repeat. Guarded UPDATE (WHERE processed_at is stale) so two
	// concurrent stale repeats serialize: the first wins the refresh, the rest see 0 rows → dedup.
	upd := db.Model(&models.ProcessedEvent{}).
		Where("consumer = ? AND msg_id = ? AND processed_at <= ?",
			refundDedupConsumer, key, time.Now().Add(-refundDedupWindow)).
		Update("processed_at", time.Now())
	if upd.Error != nil {
		return false, key, upd.Error
	}
	return upd.RowsAffected == 1, key, nil
}

// ReleaseRefundRequestClaim drops a refund-request claim so a retry re-attempts after a PRE-COMMIT
// failure (reserve/gateway/wallet error) — instead of the retry being deduped into a no-op. Never
// called once the money has moved (a persisted or gateway-succeeded refund KEEPS its claim so a
// retry dedups). Best-effort (a failed release only means a retry is deduped, never a double
// refund).
func ReleaseRefundRequestClaim(db *gorm.DB, dedupKey string) {
	ReleaseWebhookEvent(db, refundDedupConsumer, dedupKey)
}
