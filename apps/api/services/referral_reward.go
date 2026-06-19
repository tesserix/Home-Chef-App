package services

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// referral_reward.go — the referral reward engine (#38). Called from every
// order-paid path; idempotent end-to-end so duplicate webhooks / retries never
// double-pay. The reward is a plain in-tx wallet credit (no Temporal) — the
// wallet's idempotency key + the referral's status make it exactly-once.

// referralSpendThisMonth sums all referral wallet credits issued since the start
// of the current (UTC) month — the figure the monthly spend cap is checked against.
func referralSpendThisMonth(db *gorm.DB) float64 {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var total float64
	db.Model(&models.WalletTxn{}).
		Where("source = ? AND type = ? AND created_at >= ?", models.WalletSourceReferral, models.WalletCredit, monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&total)
	return total
}

// MaybeGrantReward grants the referral reward when `orderID` is the referee's
// FIRST paid order. It's a no-op (logged, never fatal) when there's no eligible
// referral, the order isn't paid/first, a fraud guard trips, the program is
// disabled, or the monthly spend cap would be exceeded. Safe to call from a
// payment webhook — failures here must never fail the payment.
func MaybeGrantReward(db *gorm.DB, orderID uuid.UUID) {
	var order models.Order
	if err := db.Select("id, customer_id, payment_status").First(&order, "id = ?", orderID).Error; err != nil {
		return
	}
	if order.PaymentStatus != models.PaymentCompleted {
		return
	}

	// A pending referral where this customer is the referee.
	var ref models.Referral
	if err := db.Where("referee_user_id = ? AND status = ?", order.CustomerID, models.ReferralStatePending).
		First(&ref).Error; err != nil {
		return // not a referred customer (or already rewarded/rejected)
	}

	// First-paid-order guard: any OTHER completed order means this isn't the first.
	var priorPaid int64
	db.Model(&models.Order{}).
		Where("customer_id = ? AND payment_status = ? AND id <> ?", order.CustomerID, models.PaymentCompleted, order.ID).
		Count(&priorPaid)
	if priorPaid > 0 {
		return
	}

	cfg := GetReferralConfig(db)
	if !cfg.Enabled {
		return
	}

	// Fraud guard: the referrer and referee must not share a device (best-effort
	// FCM-token dedupe — the most common self-referral signal we have).
	if ref.RefereeDevice != "" {
		var referrer models.User
		if err := db.Select("fcm_token").First(&referrer, "id = ?", ref.ReferrerUserID).Error; err == nil {
			if referrer.FCMToken != "" && referrer.FCMToken == ref.RefereeDevice {
				rejectReferral(db, &ref, "device shared with referrer")
				return
			}
		}
	}

	grant := cfg.ReferrerReward + cfg.RefereeReward

	// Monthly spend cap — skip (leave pending for manual review) if exceeding.
	if cfg.MonthlySpendCap > 0 && referralSpendThisMonth(db)+grant > cfg.MonthlySpendCap {
		log.Printf("referral reward skipped (monthly cap reached): referral=%s grant=%.2f cap=%.2f", ref.ID, grant, cfg.MonthlySpendCap)
		return
	}

	// Credit both wallets idempotently (per-referral keys → re-runs are no-ops).
	if cfg.ReferrerReward > 0 {
		if _, err := CreditWallet(db, ref.ReferrerUserID, cfg.ReferrerReward, models.WalletSourceReferral, &order.ID,
			"Referral reward — your friend placed their first order", "referral-referrer:"+ref.ID.String(), nil); err != nil {
			log.Printf("referral reward: credit referrer %s failed: %v", ref.ReferrerUserID, err)
			return
		}
	}
	if cfg.RefereeReward > 0 {
		if _, err := CreditWallet(db, ref.RefereeUserID, cfg.RefereeReward, models.WalletSourceReferral, &order.ID,
			"Welcome credit for joining via a referral", "referral-referee:"+ref.ID.String(), nil); err != nil {
			// Leave the referral PENDING so a retry completes the "get" half. The
			// referrer credit above is idempotent on its key, so a retry won't
			// double-pay it. Marking rewarded here would silently drop the
			// referee's credit forever.
			log.Printf("referral reward: credit referee %s failed (will retry): %v", ref.RefereeUserID, err)
			return
		}
	}

	now := time.Now()
	db.Model(&models.Referral{}).Where("id = ?", ref.ID).Updates(map[string]any{
		"status":          models.ReferralStateRewarded,
		"order_id":        order.ID,
		"referrer_reward": cfg.ReferrerReward,
		"referee_reward":  cfg.RefereeReward,
		"rewarded_at":     now,
	})

	// Notify the referrer their reward landed (durable via the outbox).
	if err := EnqueueEvent(db, SubjectReferralRewarded, "referral.reward.granted", ref.ReferrerUserID, map[string]any{
		"referral_id":     ref.ID.String(),
		"referee_id":      ref.RefereeUserID.String(),
		"referrer_reward": cfg.ReferrerReward,
		"referee_reward":  cfg.RefereeReward,
	}); err != nil {
		log.Printf("referral reward: enqueue notification failed: %v", err)
	}
}

func rejectReferral(db *gorm.DB, ref *models.Referral, reason string) {
	log.Printf("referral %s rejected: %s", ref.ID, reason)
	db.Model(&models.Referral{}).Where("id = ?", ref.ID).Update("status", models.ReferralStateRejected)
}
