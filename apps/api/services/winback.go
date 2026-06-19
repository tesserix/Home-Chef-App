package services

// winback.go — the subscription / customer win-back offer engine (#42).
//
// When a customer lapses or a chef/driver cancels/suspends their subscription,
// OfferWinback mints a targeted, time-limited, single-use platform-funded promo
// (reusing the #39 promo engine) and enqueues a `subscription.winback_offered`
// event that the notification service turns into an in-app + push + email nudge.
// Reactivation = the offer's unique code being redeemed; ReconcileWinbackOffers
// flips offers to reactivated / expired. All config is admin-tunable via the
// `winback.*` PlatformSettings keys (mirrors the referral/order-issue patterns).

import (
	"crypto/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// WinbackConfig is the admin-tunable win-back program config.
type WinbackConfig struct {
	Enabled            bool    `json:"enabled"`
	DiscountPercent    float64 `json:"discountPercent"`    // win-back discount %
	MaxDiscount        float64 `json:"maxDiscount"`        // cap on the % discount (0 = none)
	ValidityDays       int     `json:"validityDays"`       // the code is valid for N days
	LapseThresholdDays int     `json:"lapseThresholdDays"` // no delivered order in N days → lapsed
	CooldownDays       int     `json:"cooldownDays"`       // don't re-offer the same user within N days
}

// GetWinbackConfig reads the program config from `winback.*` PlatformSettings,
// falling back to sane defaults so the feature works before an admin sets anything.
func GetWinbackConfig(db *gorm.DB) WinbackConfig {
	cfg := WinbackConfig{Enabled: true, DiscountPercent: 20, MaxDiscount: 0, ValidityDays: 14, LapseThresholdDays: 30, CooldownDays: 30}
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "winback.%").Find(&settings)
	for _, s := range settings {
		switch s.Key {
		case "winback.enabled":
			cfg.Enabled = s.Value == "true" || s.Value == "1"
		case "winback.discount_percent":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.DiscountPercent = v
			}
		case "winback.max_discount":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.MaxDiscount = v
			}
		case "winback.validity_days":
			if v, err := strconv.Atoi(s.Value); err == nil {
				cfg.ValidityDays = v
			}
		case "winback.lapse_threshold_days":
			if v, err := strconv.Atoi(s.Value); err == nil {
				cfg.LapseThresholdDays = v
			}
		case "winback.cooldown_days":
			if v, err := strconv.Atoi(s.Value); err == nil {
				cfg.CooldownDays = v
			}
		}
	}
	return cfg
}

// winbackAlphabet excludes ambiguous characters so codes are easy to read/type.
const winbackAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// generateWinbackCode mints a hard-to-guess single-use code like "WB-7K9PQR".
func generateWinbackCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	out := make([]byte, len(b))
	for i, c := range b {
		out[i] = winbackAlphabet[int(c)%len(winbackAlphabet)]
	}
	return "WB-" + string(out)
}

// HasRecentWinbackOffer reports whether the user already received an offer within
// the cooldown window — prevents spamming repeat offers across triggers/cron runs.
func HasRecentWinbackOffer(db *gorm.DB, userID uuid.UUID, cooldownDays int) bool {
	var count int64
	since := time.Now().AddDate(0, 0, -cooldownDays)
	db.Model(&models.WinbackOffer{}).
		Where("user_id = ? AND offered_at >= ?", userID, since).
		Count(&count)
	return count > 0
}

// OfferWinback mints a targeted win-back promo + offer for a user and enqueues the
// notification event, atomically. Returns (nil, nil) when the program is disabled
// or the user is within the cooldown window, so callers (cron + lifecycle hooks)
// can fire it freely without their own guards.
func OfferWinback(db *gorm.DB, userID uuid.UUID, audience, trigger string, subscriptionID *uuid.UUID) (*models.WinbackOffer, error) {
	cfg := GetWinbackConfig(db)
	if !cfg.Enabled {
		return nil, nil
	}
	if HasRecentWinbackOffer(db, userID, cfg.CooldownDays) {
		return nil, nil
	}

	now := time.Now()
	validUntil := now.AddDate(0, 0, cfg.ValidityDays)

	var offer models.WinbackOffer
	err := db.Transaction(func(tx *gorm.DB) error {
		// Mint a unique, single-use, platform-funded promo (reuse #39). ApplicableTo
		// is "all" — targeting is enforced by only notifying this user; the unique
		// random code + single-use limit keep it from leaking broadly.
		promo := models.PromoCode{
			ID:            uuid.New(),
			Code:          generateWinbackCode(),
			Description:   "Win-back offer — we miss you!",
			DiscountType:  PromoDiscountPercentage,
			DiscountValue: cfg.DiscountPercent,
			MaxDiscount:   cfg.MaxDiscount,
			UsageLimit:    1,
			PerUserLimit:  1,
			ValidFrom:     now,
			ValidUntil:    &validUntil,
			IsActive:      true,
			ApplicableTo:  "all",
			FundingSource: models.PromoFundingPlatform,
			CreatedByID:   userID,
		}
		if err := tx.Create(&promo).Error; err != nil {
			return err
		}
		offer = models.WinbackOffer{
			UserID:          userID,
			AudienceType:    audience,
			Trigger:         trigger,
			PromoCodeID:     promo.ID,
			Code:            promo.Code,
			DiscountPercent: cfg.DiscountPercent,
			Status:          models.WinbackStatusOffered,
			SubscriptionID:  subscriptionID,
			ExpiresAt:       validUntil,
		}
		if err := tx.Create(&offer).Error; err != nil {
			return err
		}
		return EnqueueEvent(tx, SubjectSubscriptionWinbackOffered, "subscription.winback_offered", userID, map[string]any{
			"offer_id":         offer.ID.String(),
			"code":             promo.Code,
			"discount_percent": cfg.DiscountPercent,
			"expires_at":       validUntil.Format(time.RFC3339),
			"trigger":          trigger,
			"audience":         audience,
		})
	})
	if err != nil {
		return nil, err
	}
	return &offer, nil
}

// ReconcileWinbackOffers advances open offers: → reactivated once the offer's promo
// code has been redeemed, or → expired once past ExpiresAt. Idempotent (guarded by
// a conditional WHERE status='offered'); run periodically by the win-back cron.
// Returns (reactivated, expired) counts for logging/metrics.
func ReconcileWinbackOffers(db *gorm.DB) (int, int) {
	now := time.Now()
	var open []models.WinbackOffer
	db.Where("status = ?", models.WinbackStatusOffered).Find(&open)

	reactivated, expired := 0, 0
	for _, o := range open {
		var redemptions int64
		db.Model(&models.PromoCodeUsage{}).
			Where("promo_code_id = ? AND user_id = ?", o.PromoCodeID, o.UserID).
			Count(&redemptions)
		if redemptions > 0 {
			res := db.Model(&models.WinbackOffer{}).
				Where("id = ? AND status = ?", o.ID, models.WinbackStatusOffered).
				Updates(map[string]any{"status": models.WinbackStatusReactivated, "reactivated_at": now})
			if res.RowsAffected == 1 {
				reactivated++
			}
			continue
		}
		if now.After(o.ExpiresAt) {
			res := db.Model(&models.WinbackOffer{}).
				Where("id = ? AND status = ?", o.ID, models.WinbackStatusOffered).
				Update("status", models.WinbackStatusExpired)
			if res.RowsAffected == 1 {
				expired++
			}
		}
	}
	return reactivated, expired
}

// GetActiveWinbackOffer returns the user's current open, non-expired win-back offer
// (most recent first), or nil. Powers the in-app banner (#276).
func GetActiveWinbackOffer(db *gorm.DB, userID uuid.UUID) *models.WinbackOffer {
	var offer models.WinbackOffer
	if err := db.Where("user_id = ? AND status = ? AND expires_at > ?", userID, models.WinbackStatusOffered, time.Now()).
		Order("offered_at DESC").First(&offer).Error; err != nil {
		return nil
	}
	return &offer
}
