package services

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// walletTopUpConsumer namespaces wallet top-up claims in the processed_events
// idempotency ledger (shared with the webhook dedup + NATS consumer backbone).
const walletTopUpConsumer = "wallet-topup"

// walletTopUpKey is a stable per-(order, leg-index, account) claim key bounded to
// processed_events.msg_id (varchar(64)). A uuid + leg + ":" + acc id normally fits; a
// pathologically long account id falls back to a hash so the insert can't overflow.
// #558: the leg index keeps each top-up leg independently idempotent even when a chef and
// driver share one Razorpay payout account (else the 2nd leg's claim collides and is skipped —
// that leg never gets paid).
func walletTopUpKey(orderID uuid.UUID, leg int, account string) string {
	k := fmt.Sprintf("%s:%d:%s", orderID, leg, account)
	if len(k) <= maxWebhookEventID {
		return k
	}
	sum := sha256.Sum256([]byte(k))
	return "h:" + base64.RawURLEncoding.EncodeToString(sum[:])
}

// ClaimWalletTopUp atomically claims the platform-funded top-up transfer for one
// (order, chef/driver account) so a repeated settlement — e.g. a retried
// VerifyPayment — cannot issue the same real money transfer twice (#554). firstTime
// is true only for the caller that first claims it (which should create the
// transfer); every later caller gets false and must skip. Atomic via
// INSERT ... ON CONFLICT DO NOTHING, so there is no check-then-act race.
func ClaimWalletTopUp(db *gorm.DB, orderID uuid.UUID, leg int, account string) (bool, error) {
	return ClaimWebhookEvent(db, walletTopUpConsumer, walletTopUpKey(orderID, leg, account), "wallet.topup")
}

// ReleaseWalletTopUp drops a top-up claim so a retry re-attempts the transfer after a
// gateway failure (instead of the retry being deduped into a no-op). Best-effort: a
// failed release only means the transfer won't retry (never a double transfer).
func ReleaseWalletTopUp(db *gorm.DB, orderID uuid.UUID, leg int, account string) {
	ReleaseWebhookEvent(db, walletTopUpConsumer, walletTopUpKey(orderID, leg, account))
}
