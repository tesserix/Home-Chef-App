package services

// loyalty.go — points engine for the loyalty program (#40). Customers earn
// points on delivered orders and meal-subscription streaks (#2), and redeem
// them to wallet store credit (#33). The points ledger mirrors the wallet
// ledger exactly: LoyaltyAccount.Balance is a cached aggregate kept consistent
// with the append-only LoyaltyTransaction ledger inside one DB transaction, and
// every mutation is idempotent on a key so a redelivered event or a
// double-tapped redeem can never double-apply. Redeem is the only money-touching
// path — it debits points and credits the wallet atomically.
//
// All program parameters are admin-tunable at runtime via loyalty.*
// PlatformSettings, mirroring the referral/win-back config pattern, so the
// platform owner changes earn/redeem economics without a deploy.

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// LoyaltyConfig is the admin-tunable program config, stored as PlatformSettings
// `loyalty.*` keys.
type LoyaltyConfig struct {
	Enabled         bool    `json:"enabled"`
	PointsPerRupee  float64 `json:"pointsPerRupee"`  // points earned per ₹1 of order value
	RedeemRate      float64 `json:"redeemRate"`      // ₹ of wallet credit per 1 point redeemed
	MinRedeem       float64 `json:"minRedeem"`       // minimum points per redemption
	StreakThreshold int     `json:"streakThreshold"` // consecutive meal-sub days for a bonus
	StreakBonus     float64 `json:"streakBonus"`     // points awarded each time the streak is hit
	StreakGraceDays int     `json:"streakGraceDays"` // max gap (days) between delivered days before the run breaks
	TierSilverAt    float64 `json:"tierSilverAt"`    // lifetime points for silver
	TierGoldAt      float64 `json:"tierGoldAt"`      // lifetime points for gold
}

// defaultLoyaltyConfig — sane defaults so the feature works before an admin sets
// anything. Defaults give ~1% earn (₹10 → 1 pt) and a 1:1 ₹/10-pt redeem
// (100 pts → ₹10), i.e. roughly 1% back, with a 7-day streak bonus.
func defaultLoyaltyConfig() LoyaltyConfig {
	return LoyaltyConfig{
		Enabled:         true,
		PointsPerRupee:  0.1,
		RedeemRate:      0.1,
		MinRedeem:       100,
		StreakThreshold: 7,
		StreakBonus:     50,
		StreakGraceDays: 1, // strictly consecutive calendar days by default
		TierSilverAt:    1000,
		TierGoldAt:      5000,
	}
}

// GetLoyaltyConfig reads the program config, falling back to defaults per field.
func GetLoyaltyConfig(db *gorm.DB) LoyaltyConfig {
	cfg := defaultLoyaltyConfig()
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "loyalty.%").Find(&settings)
	for _, s := range settings {
		switch s.Key {
		case "loyalty.enabled":
			cfg.Enabled = s.Value == "true" || s.Value == "1"
		case "loyalty.points_per_rupee":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.PointsPerRupee = v
			}
		case "loyalty.redeem_rate":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.RedeemRate = v
			}
		case "loyalty.min_redeem":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.MinRedeem = v
			}
		case "loyalty.streak_threshold":
			if v, err := strconv.Atoi(s.Value); err == nil {
				cfg.StreakThreshold = v
			}
		case "loyalty.streak_bonus":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.StreakBonus = v
			}
		case "loyalty.streak_grace_days":
			if v, err := strconv.Atoi(s.Value); err == nil {
				cfg.StreakGraceDays = v
			}
		case "loyalty.tier_silver_at":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.TierSilverAt = v
			}
		case "loyalty.tier_gold_at":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.TierGoldAt = v
			}
		}
	}
	return cfg
}

var (
	ErrLoyaltyDisabled           = errors.New("loyalty program is disabled")
	ErrInsufficientLoyaltyPoints = errors.New("insufficient loyalty points")
	ErrLoyaltyBelowMinRedeem     = errors.New("redemption is below the minimum points")
)

// PointsForOrder returns the whole points earned on an order of the given total.
// Fractional points are floored so a customer always sees clean integer points.
func PointsForOrder(cfg LoyaltyConfig, orderTotal float64) float64 {
	if orderTotal <= 0 || cfg.PointsPerRupee <= 0 {
		return 0
	}
	return math.Floor(orderTotal * cfg.PointsPerRupee)
}

// ComputeTier maps lifetime points earned to a status tier. Thresholds of 0
// disable that tier.
func ComputeTier(lifetime float64, cfg LoyaltyConfig) string {
	switch {
	case cfg.TierGoldAt > 0 && lifetime >= cfg.TierGoldAt:
		return models.LoyaltyTierGold
	case cfg.TierSilverAt > 0 && lifetime >= cfg.TierSilverAt:
		return models.LoyaltyTierSilver
	default:
		return models.LoyaltyTierBronze
	}
}

// GetOrCreateLoyaltyAccount returns the customer's account, lazily creating it on
// first touch (first earn or first balance read) — mirrors GetOrCreateWallet.
func GetOrCreateLoyaltyAccount(db *gorm.DB, userID uuid.UUID) (*models.LoyaltyAccount, error) {
	var acct models.LoyaltyAccount
	err := db.Where("user_id = ?", userID).First(&acct).Error
	if err == nil {
		return &acct, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	acct = models.LoyaltyAccount{UserID: userID, Tier: models.LoyaltyTierBronze}
	if err := db.Create(&acct).Error; err != nil {
		// Lost a race to create — re-read the winner's row.
		if err2 := db.Where("user_id = ?", userID).First(&acct).Error; err2 == nil {
			return &acct, nil
		}
		return nil, err
	}
	return &acct, nil
}

// LoyaltyBalance returns the customer's account (created if absent) for read.
func LoyaltyBalance(db *gorm.DB, userID uuid.UUID) (*models.LoyaltyAccount, error) {
	return GetOrCreateLoyaltyAccount(db, userID)
}

// ListLoyaltyTxns returns a page of a user's points ledger, newest first.
func ListLoyaltyTxns(db *gorm.DB, userID uuid.UUID, limit, offset int) ([]models.LoyaltyTransaction, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var txns []models.LoyaltyTransaction
	var total int64
	if err := db.Model(&models.LoyaltyTransaction{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&txns).Error; err != nil {
		return nil, 0, err
	}
	return txns, total, nil
}

// applyLoyaltyTxnInTx is the single mutate path for the points ledger, run
// inside a caller-provided transaction so it can be composed with a wallet
// credit (redeem) atomically. It locks the account row (FOR UPDATE on Postgres),
// writes the immutable ledger entry, and updates the cached balance + lifetime +
// tier. The idempotency check (plus the unique index) makes a retried event a
// no-op; the returned bool reports whether a NEW entry was written (false on a
// dedup hit) so callers can avoid double-notifying.
func applyLoyaltyTxnInTx(tx *gorm.DB, userID uuid.UUID, points float64, txnType models.LoyaltyTxnType, source models.LoyaltyTxnSource, orderID *uuid.UUID, reason, idempotencyKey string, createdBy *uuid.UUID, cfg LoyaltyConfig) (*models.LoyaltyTransaction, bool, error) {
	if points <= 0 {
		return nil, false, fmt.Errorf("loyalty points must be positive, got %v", points)
	}
	if idempotencyKey == "" {
		idempotencyKey = "ltx:" + uuid.NewString()
	}

	// Idempotency: if this logical event already landed, return it untouched.
	var existing models.LoyaltyTransaction
	err := tx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error
	if err == nil {
		return &existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	acct, err := GetOrCreateLoyaltyAccount(tx, userID)
	if err != nil {
		return nil, false, err
	}
	// Re-read under a row lock so a concurrent txn can't race the balance.
	reread := tx
	if tx.Dialector.Name() == "postgres" {
		reread = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := reread.First(acct, "id = ?", acct.ID).Error; err != nil {
		return nil, false, err
	}

	newBalance := acct.Balance
	newLifetime := acct.LifetimePoints
	if txnType == models.LoyaltyCredit {
		newBalance += points
		newLifetime += points // lifetime only ever grows — drives tier
	} else {
		if acct.Balance < points {
			return nil, false, ErrInsufficientLoyaltyPoints
		}
		newBalance -= points
	}
	newTier := ComputeTier(newLifetime, cfg)

	entry := &models.LoyaltyTransaction{
		ID:               uuid.New(),
		LoyaltyAccountID: acct.ID,
		UserID:           userID,
		Type:             txnType,
		Source:           source,
		Points:           points,
		PointsAfter:      newBalance,
		OrderID:          orderID,
		Reason:           reason,
		CreatedBy:        createdBy,
		IdempotencyKey:   idempotencyKey,
	}
	if err := tx.Create(entry).Error; err != nil {
		return nil, false, err
	}
	if err := tx.Model(acct).Updates(map[string]any{
		"balance":         newBalance,
		"lifetime_points": newLifetime,
		"tier":            newTier,
	}).Error; err != nil {
		return nil, false, err
	}
	return entry, true, nil
}

// EarnLoyalty credits points (delivered order, streak bonus, admin grant).
// Idempotent on idempotencyKey.
func EarnLoyalty(db *gorm.DB, userID uuid.UUID, points float64, source models.LoyaltyTxnSource, orderID *uuid.UUID, reason, idempotencyKey string) (*models.LoyaltyTransaction, error) {
	cfg := GetLoyaltyConfig(db)
	var result *models.LoyaltyTransaction
	err := db.Transaction(func(tx *gorm.DB) error {
		txn, _, err := applyLoyaltyTxnInTx(tx, userID, points, models.LoyaltyCredit, source, orderID, reason, idempotencyKey, nil, cfg)
		if err != nil {
			return err
		}
		result = txn
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// AwardOrderLoyalty earns points for a delivered order and enqueues a
// points-earned notification event — both in the same transaction, and only
// when the points were newly awarded. A redelivered order-delivered event is a
// no-op (idempotent on "loyalty:order:<id>"). Returns the points awarded (0 if
// disabled, already awarded, or a zero-value order). Called from the
// order-delivered consumer.
func AwardOrderLoyalty(db *gorm.DB, userID, orderID uuid.UUID, orderTotal float64) (float64, error) {
	cfg := GetLoyaltyConfig(db)
	if !cfg.Enabled {
		return 0, nil
	}
	points := PointsForOrder(cfg, orderTotal)
	if points <= 0 {
		return 0, nil
	}
	var awarded float64
	err := db.Transaction(func(tx *gorm.DB) error {
		oid := orderID
		txn, created, err := applyLoyaltyTxnInTx(tx, userID, points, models.LoyaltyCredit, models.LoyaltySourceOrder, &oid, "Earned on a delivered order", "loyalty:order:"+orderID.String(), nil, cfg)
		if err != nil {
			return err
		}
		if !created {
			return nil // redelivered event — already awarded
		}
		awarded = points
		return EnqueueEvent(tx, SubjectLoyaltyEarned, "loyalty_earned", userID, map[string]any{
			"points":   points,
			"balance":  txn.PointsAfter,
			"order_id": orderID.String(),
			"source":   string(models.LoyaltySourceOrder),
		})
	})
	if err != nil {
		return 0, err
	}
	return awarded, nil
}

// RedeemLoyalty converts points to wallet store credit. It debits the points and
// credits the wallet (WalletSourceLoyalty) in one transaction so the two ledgers
// never diverge, then enqueues a redeemed notification event. Enforces the
// configured minimum and rejects an overdraw without partial application.
func RedeemLoyalty(db *gorm.DB, userID uuid.UUID, points float64) (*models.LoyaltyTransaction, *models.WalletTxn, error) {
	cfg := GetLoyaltyConfig(db)
	if !cfg.Enabled {
		return nil, nil, ErrLoyaltyDisabled
	}
	if points < cfg.MinRedeem {
		return nil, nil, ErrLoyaltyBelowMinRedeem
	}
	rupees := round2Money(points * cfg.RedeemRate)
	if rupees <= 0 {
		return nil, nil, fmt.Errorf("redemption resolves to zero wallet credit")
	}

	var lt *models.LoyaltyTransaction
	var wt *models.WalletTxn
	err := db.Transaction(func(tx *gorm.DB) error {
		reason := fmt.Sprintf("Redeemed %.0f points for ₹%.2f wallet credit", points, rupees)
		txn, _, err := applyLoyaltyTxnInTx(tx, userID, points, models.LoyaltyDebit, models.LoyaltySourceRedeem, nil, reason, "loyalty-redeem-debit:"+uuid.NewString(), nil, cfg)
		if err != nil {
			return err
		}
		lt = txn
		// The wallet credit's idempotency key is derived from the debit row, so
		// the credit can never duplicate against its debit.
		wtxn, err := CreditWallet(tx, userID, rupees, models.WalletSourceLoyalty, nil, "Loyalty points redeemed", "loyalty-redeem:"+txn.ID.String(), nil)
		if err != nil {
			return err
		}
		wt = wtxn
		return EnqueueEvent(tx, SubjectLoyaltyRedeemed, "loyalty_redeemed", userID, map[string]any{
			"points": points,
			"amount": rupees,
		})
	})
	if err != nil {
		return nil, nil, err
	}
	return lt, wt, nil
}

// round2Money rounds to 2 decimal places (paise) for wallet money amounts.
func round2Money(v float64) float64 { return math.Round(v*100) / 100 }

// ─────────────────────────────────────────────────────────────────────────────
// Streaks — meal-subscription adherence (#2 / #291)
//
// A streak is the run of consecutive delivered meal-subscription days. Each
// delivered day advances it by one (two slots on the same calendar day count
// once). The run breaks on a GAP: when the next delivered day is more than
// StreakGraceDays after the previous one, the streak resets to 1 for that day.
// A missed day therefore breaks the streak automatically — no meal was
// delivered, so the gap to the next delivery exceeds the grace — without
// depending on any explicit "mark missed" sweep. (ResetLoyaltyStreak still gives
// callers an explicit reset for an immediate break.) When the streak crosses a
// multiple of the configured threshold the customer earns a bonus — awarded
// exactly once per milestone via an idempotency key keyed on the streak count.
// ─────────────────────────────────────────────────────────────────────────────

// truncateToDay drops the time-of-day so streak comparisons are by calendar day.
func truncateToDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// graceDays is the configured max gap between delivered days, floored at 1 so a
// misconfigured 0/negative value can't make the streak never break.
func graceDays(cfg LoyaltyConfig) int {
	if cfg.StreakGraceDays < 1 {
		return 1
	}
	return cfg.StreakGraceDays
}

// AdvanceLoyaltyStreak records a delivered meal-subscription day for the customer
// and returns the new streak length and any bonus points awarded this call. Same
// calendar day twice (e.g. lunch + dinner) counts once. Safe to call on a
// redelivered event when paired with a delivered-transition guard upstream.
func AdvanceLoyaltyStreak(db *gorm.DB, userID uuid.UUID, deliveredOn time.Time) (int, float64, error) {
	cfg := GetLoyaltyConfig(db)
	var newStreak int
	var bonus float64
	err := db.Transaction(func(tx *gorm.DB) error {
		s, b, err := advanceStreakInTx(tx, userID, deliveredOn, cfg)
		newStreak, bonus = s, b
		return err
	})
	if err != nil {
		return 0, 0, err
	}
	return newStreak, bonus, nil
}

func advanceStreakInTx(tx *gorm.DB, userID uuid.UUID, deliveredOn time.Time, cfg LoyaltyConfig) (int, float64, error) {
	acct, err := GetOrCreateLoyaltyAccount(tx, userID)
	if err != nil {
		return 0, 0, err
	}
	reread := tx
	if tx.Dialector.Name() == "postgres" {
		reread = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := reread.First(acct, "id = ?", acct.ID).Error; err != nil {
		return 0, 0, err
	}

	d := truncateToDay(deliveredOn)
	if acct.LastStreakDay != nil {
		last := truncateToDay(*acct.LastStreakDay)
		// Whole-day delta (rounded so a DST transition's ±1h can't skew it).
		daysSince := int(math.Round(d.Sub(last).Hours() / 24))
		if daysSince <= 0 {
			// Same calendar day already counted (second slot), or an
			// out-of-order older delivery — already reflected; don't
			// double-count or rewind the streak.
			return acct.CurrentStreak, 0, nil
		}
		if daysSince > graceDays(cfg) {
			acct.CurrentStreak = 0 // a gap (missed day) broke the run
		}
	}

	acct.CurrentStreak++
	if acct.CurrentStreak > acct.LongestStreak {
		acct.LongestStreak = acct.CurrentStreak
	}
	if err := tx.Model(acct).Updates(map[string]any{
		"current_streak":  acct.CurrentStreak,
		"longest_streak":  acct.LongestStreak,
		"last_streak_day": d,
	}).Error; err != nil {
		return 0, 0, err
	}

	// Bonus on every threshold crossing (7, 14, 21, …), once per milestone.
	var bonus float64
	if cfg.StreakThreshold > 0 && cfg.StreakBonus > 0 && acct.CurrentStreak%cfg.StreakThreshold == 0 {
		key := fmt.Sprintf("loyalty:streak:%s:%d", acct.ID, acct.CurrentStreak)
		reason := fmt.Sprintf("%d-day meal-subscription streak bonus", acct.CurrentStreak)
		txn, created, err := applyLoyaltyTxnInTx(tx, userID, cfg.StreakBonus, models.LoyaltyCredit, models.LoyaltySourceStreak, nil, reason, key, nil, cfg)
		if err != nil {
			return 0, 0, err
		}
		if created {
			bonus = cfg.StreakBonus
			if err := EnqueueEvent(tx, SubjectLoyaltyEarned, "loyalty_earned", userID, map[string]any{
				"points":  cfg.StreakBonus,
				"balance": txn.PointsAfter,
				"streak":  acct.CurrentStreak,
				"source":  string(models.LoyaltySourceStreak),
			}); err != nil {
				return 0, 0, err
			}
		}
	}
	return acct.CurrentStreak, bonus, nil
}

// ResetLoyaltyStreak breaks the customer's current run (a missed day). The
// longest-streak record is preserved. No-op if the account doesn't exist yet.
func ResetLoyaltyStreak(db *gorm.DB, userID uuid.UUID) error {
	return db.Model(&models.LoyaltyAccount{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{"current_streak": 0, "last_streak_day": nil}).Error
}

// LoyaltyAnalytics is the admin overview of the program (#40).
type LoyaltyAnalytics struct {
	Members        int64   `json:"members"`        // accounts that have ever earned
	OutstandingPts float64 `json:"outstandingPts"` // unredeemed points held by customers
	PointsEarned   float64 `json:"pointsEarned"`   // all-time credited points
	PointsRedeemed float64 `json:"pointsRedeemed"` // all-time points converted to wallet
	ActiveStreaks  int64   `json:"activeStreaks"`  // accounts with a live streak
	LongestStreak  int64   `json:"longestStreak"`  // best streak across all customers
}

// GetLoyaltyAnalytics summarises program activity for the admin dashboard.
func GetLoyaltyAnalytics(db *gorm.DB) LoyaltyAnalytics {
	var a LoyaltyAnalytics
	db.Model(&models.LoyaltyAccount{}).Where("lifetime_points > 0").Count(&a.Members)
	db.Model(&models.LoyaltyAccount{}).Select("COALESCE(SUM(balance), 0)").Scan(&a.OutstandingPts)
	db.Model(&models.LoyaltyAccount{}).Where("current_streak > 0").Count(&a.ActiveStreaks)
	db.Model(&models.LoyaltyAccount{}).Select("COALESCE(MAX(longest_streak), 0)").Scan(&a.LongestStreak)
	db.Model(&models.LoyaltyTransaction{}).Where("type = ?", models.LoyaltyCredit).Select("COALESCE(SUM(points), 0)").Scan(&a.PointsEarned)
	db.Model(&models.LoyaltyTransaction{}).Where("type = ? AND source = ?", models.LoyaltyDebit, models.LoyaltySourceRedeem).Select("COALESCE(SUM(points), 0)").Scan(&a.PointsRedeemed)
	return a
}
