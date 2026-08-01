package services

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// chef_referral.go — chef-refers-chef program. Codes come from the shared
// referral_codes table (GetOrCreateReferralCode — same minting as the customer
// program). Both rewards are CASH: raised as pending ChefBonus rows when the
// referee kitchen hits the delivered-orders milestone, then added onto each
// chef's next weekly settlement by ApplyChefBonusesToStatement.

// ChefReferralConfig is the admin-tunable program config, stored as
// PlatformSettings `chef_referral.*` keys.
type ChefReferralConfig struct {
	Enabled bool `json:"enabled"`
	// Rewards are RUPEES — chefs are paid cash through settlement, not points.
	ReferrerAmount float64 `json:"referrerAmount"`
	RefereeAmount  float64 `json:"refereeAmount"`
	// MilestoneOrders is how many delivered orders the referee kitchen must
	// complete before both rewards vest.
	MilestoneOrders int `json:"milestoneOrders"`
	// MonthlySpendCap (₹) bounds referral bonuses raised per calendar month;
	// referrals past the cap stay pending and vest in a later month's re-check.
	MonthlySpendCap float64 `json:"monthlySpendCap"`
}

// GetChefReferralConfig reads the program config with working defaults.
func GetChefReferralConfig(db *gorm.DB) ChefReferralConfig {
	cfg := ChefReferralConfig{Enabled: true, ReferrerAmount: 500, RefereeAmount: 250, MilestoneOrders: 10, MonthlySpendCap: 50000}
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "chef_referral.%").Find(&settings)
	for _, s := range settings {
		switch s.Key {
		case "chef_referral.enabled":
			cfg.Enabled = s.Value == "true" || s.Value == "1"
		case "chef_referral.referrer_amount":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.ReferrerAmount = v
			}
		case "chef_referral.referee_amount":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.RefereeAmount = v
			}
		case "chef_referral.milestone_orders":
			if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
				cfg.MilestoneOrders = v
			}
		case "chef_referral.monthly_spend_cap":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.MonthlySpendCap = v
			}
		}
	}
	return cfg
}

// ChefReferralLink is the shareable vendor-onboarding link for a code.
func ChefReferralLink(code string) string {
	return "https://vendors.fe3dr.com/onboarding?ref=" + code
}

var (
	ErrChefReferralNotAChef      = errors.New("that referral code doesn't belong to an active kitchen")
	ErrChefReferralSelf          = errors.New("you can't use your own referral code")
	ErrChefReferralAlreadyUsed   = errors.New("this kitchen was already referred")
	ErrChefReferralNotNewKitchen = errors.New("referral codes are for new kitchens only")
)

// AcceptChefReferralInput carries a redemption made during kitchen onboarding.
type AcceptChefReferralInput struct {
	RefereeChefID uuid.UUID
	RefereeUserID uuid.UUID
	Code          string
}

// ResolveChefReferrer validates a code for the chef program and returns the
// referrer's chef profile. Used both to pre-validate during onboarding (before
// the profile exists) and inside AcceptChefReferral.
func ResolveChefReferrer(db *gorm.DB, code string) (*models.ChefProfile, error) {
	code = NormalizeReferralCode(code)
	if code == "" {
		return nil, ErrReferralCodeInvalid
	}
	var rc models.ReferralCode
	if err := db.Where("code = ?", code).First(&rc).Error; err != nil {
		return nil, ErrReferralCodeInvalid
	}
	// The code must belong to a verified, active kitchen — a customer's code
	// can't recruit chefs.
	var referrer models.ChefProfile
	if err := db.Where("user_id = ? AND is_verified = ? AND is_active = ?", rc.UserID, true, true).
		First(&referrer).Error; err != nil {
		return nil, ErrChefReferralNotAChef
	}
	return &referrer, nil
}

// AcceptChefReferral records that a new kitchen onboarded with an existing
// chef's code. Guards: code belongs to a verified kitchen, not self, referee
// kitchen not already referred, referee has no delivered orders yet.
// Re-submitting the same code is idempotent.
func AcceptChefReferral(db *gorm.DB, in AcceptChefReferralInput) (*models.ChefReferral, error) {
	code := NormalizeReferralCode(in.Code)
	referrer, err := ResolveChefReferrer(db, code)
	if err != nil {
		return nil, err
	}
	if referrer.UserID == in.RefereeUserID {
		return nil, ErrChefReferralSelf
	}

	var existing models.ChefReferral
	if err := db.Where("referee_chef_id = ?", in.RefereeChefID).First(&existing).Error; err == nil {
		if existing.Code == code {
			return &existing, nil // idempotent re-accept
		}
		return nil, ErrChefReferralAlreadyUsed
	}

	// New-kitchen guard: a kitchen that already delivered orders isn't "new".
	var delivered int64
	db.Model(&models.Order{}).
		Where("chef_id = ? AND status = ?", in.RefereeChefID, models.OrderStatusDelivered).
		Count(&delivered)
	if delivered > 0 {
		return nil, ErrChefReferralNotNewKitchen
	}

	cfg := GetChefReferralConfig(db)
	ref := models.ChefReferral{
		ReferrerChefID:  referrer.ID,
		ReferrerUserID:  referrer.UserID,
		RefereeChefID:   in.RefereeChefID,
		RefereeUserID:   in.RefereeUserID,
		Code:            code,
		Status:          models.ChefReferralPending,
		MilestoneOrders: cfg.MilestoneOrders,
	}
	if err := db.Create(&ref).Error; err != nil {
		return nil, err
	}
	return &ref, nil
}

// chefReferralSpendThisMonth sums referral bonuses raised since the start of
// the current (UTC) month — the figure the monthly cap is checked against.
func chefReferralSpendThisMonth(db *gorm.DB) float64 {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var total float64
	db.Model(&models.ChefBonus{}).
		Where("kind IN ? AND status <> ? AND created_at >= ?",
			[]models.ChefBonusKind{models.ChefBonusReferralReferrer, models.ChefBonusReferralReferee},
			models.ChefBonusVoided, monthStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&total)
	return total
}

// MaybeGrantChefReferralReward vests both referral rewards once the referee
// kitchen has completed its milestone of delivered orders. Called from every
// delivered transition; cheap no-op when the chef wasn't referred. Idempotent:
// the ChefBonus unique SourceKey makes re-runs no-ops, and the referral status
// flip is guarded on `pending`.
func MaybeGrantChefReferralReward(db *gorm.DB, chefID uuid.UUID) {
	var ref models.ChefReferral
	if err := db.Where("referee_chef_id = ? AND status = ?", chefID, models.ChefReferralPending).
		First(&ref).Error; err != nil {
		return // not a referred kitchen (or already rewarded/rejected)
	}

	var delivered int64
	db.Model(&models.Order{}).
		Where("chef_id = ? AND status = ?", chefID, models.OrderStatusDelivered).
		Count(&delivered)
	if delivered < int64(ref.MilestoneOrders) {
		return
	}

	cfg := GetChefReferralConfig(db)
	if !cfg.Enabled {
		return
	}
	grant := Round2(cfg.ReferrerAmount + cfg.RefereeAmount)
	if cfg.MonthlySpendCap > 0 && chefReferralSpendThisMonth(db)+grant > cfg.MonthlySpendCap {
		log.Printf("chef referral reward skipped (monthly cap): referral=%s grant=%.2f cap=%.2f", ref.ID, grant, cfg.MonthlySpendCap)
		return
	}

	if cfg.ReferrerAmount > 0 {
		if err := raiseChefBonus(db, ref.ReferrerChefID, ref.ReferrerUserID, models.ChefBonusReferralReferrer,
			"chefref-referrer:"+ref.ID.String(), cfg.ReferrerAmount, &ref.ID,
			fmt.Sprintf("Referral reward — the kitchen you referred completed %d orders", ref.MilestoneOrders)); err != nil {
			log.Printf("chef referral: raise referrer bonus failed (will retry): %v", err)
			return
		}
	}
	if cfg.RefereeAmount > 0 {
		if err := raiseChefBonus(db, ref.RefereeChefID, ref.RefereeUserID, models.ChefBonusReferralReferee,
			"chefref-referee:"+ref.ID.String(), cfg.RefereeAmount, &ref.ID,
			fmt.Sprintf("Welcome bonus — you completed your first %d orders", ref.MilestoneOrders)); err != nil {
			// Leave the referral PENDING so a later delivered order retries the
			// referee half; the referrer bonus above is idempotent on its key.
			log.Printf("chef referral: raise referee bonus failed (will retry): %v", err)
			return
		}
	}

	now := time.Now()
	db.Model(&models.ChefReferral{}).
		Where("id = ? AND status = ?", ref.ID, models.ChefReferralPending).
		Updates(map[string]any{
			"status":          models.ChefReferralRewarded,
			"referrer_amount": cfg.ReferrerAmount,
			"referee_amount":  cfg.RefereeAmount,
			"rewarded_at":     now,
		})

	// Best-effort pushes — a failed notification must never fail an order flow.
	if err := SendPushNotification(ref.ReferrerUserID, "Referral reward earned",
		fmt.Sprintf("The kitchen you referred completed %d orders. ₹%.0f will be added to your next payout.", ref.MilestoneOrders, cfg.ReferrerAmount),
		map[string]string{"type": "chef_referral_reward", "deeplink": "homechef-vendor:///referral"}); err != nil {
		log.Printf("chef referral: referrer push failed: %v", err)
	}
	if err := SendPushNotification(ref.RefereeUserID, "Milestone bonus earned",
		fmt.Sprintf("Congrats on your first %d orders! ₹%.0f will be added to your next payout.", ref.MilestoneOrders, cfg.RefereeAmount),
		map[string]string{"type": "chef_referral_reward", "deeplink": "homechef-vendor:///referral"}); err != nil {
		log.Printf("chef referral: referee push failed: %v", err)
	}
}

// raiseChefBonus inserts a pending settlement credit, treating a SourceKey
// uniqueness collision as success (already raised).
func raiseChefBonus(db *gorm.DB, chefID, userID uuid.UUID, kind models.ChefBonusKind,
	sourceKey string, amount float64, referralID *uuid.UUID, reason string) error {
	bonus := models.ChefBonus{
		ChefID:     chefID,
		UserID:     userID,
		Kind:       kind,
		Status:     models.ChefBonusPending,
		SourceKey:  sourceKey,
		ReferralID: referralID,
		Currency:   EarningsCurrency,
		Amount:     Round2(amount),
		Reason:     reason,
	}
	if err := db.Create(&bonus).Error; err != nil {
		var existing models.ChefBonus
		if e := db.Where("source_key = ?", sourceKey).First(&existing).Error; e == nil {
			return nil // already raised by an earlier run
		}
		return err
	}
	return nil
}

// ApplyChefBonusesToStatement adds every pending ChefBonus for the statement's
// chef onto its net payout — the credit mirror of ApplyChefPenaltiesToStatement.
// Claim-guarded on `pending` so concurrent statement runs credit each bonus at
// most once. Returns the total credited.
func ApplyChefBonusesToStatement(db *gorm.DB, stmt *models.WeeklyStatement) (float64, error) {
	var credited float64
	err := db.Transaction(func(tx *gorm.DB) error {
		var pending []models.ChefBonus
		if err := tx.Where("chef_id = ? AND status = ?", stmt.ChefID, models.ChefBonusPending).
			Find(&pending).Error; err != nil {
			return err
		}
		if len(pending) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, 0, len(pending))
		var total float64
		for i := range pending {
			ids = append(ids, pending[i].ID)
			total += pending[i].Amount
		}
		total = Round2(total)
		now := time.Now().UTC()
		res := tx.Model(&models.ChefBonus{}).
			Where("id IN ? AND status = ?", ids, models.ChefBonusPending).
			Updates(map[string]any{
				"status": models.ChefBonusCredited, "credited_statement_id": stmt.ID, "credited_at": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // a concurrent run took them all
		}
		if res.RowsAffected != int64(len(ids)) {
			// Partially lost the race — recount exactly what THIS statement claimed.
			var claimed []models.ChefBonus
			if err := tx.Where("credited_statement_id = ?", stmt.ID).Find(&claimed).Error; err != nil {
				return err
			}
			total = 0
			for i := range claimed {
				total += claimed[i].Amount
			}
			total = Round2(total)
		}
		net := Round2(stmt.NetPayout + total)
		if err := tx.Model(&models.WeeklyStatement{}).Where("id = ?", stmt.ID).
			Updates(map[string]any{"bonus_additions": total, "net_payout": net}).Error; err != nil {
			return err
		}
		stmt.BonusAdditions = total
		stmt.NetPayout = net
		credited = total
		return nil
	})
	return credited, err
}

// ChefReferralProgress is one row of the vendor "Refer a chef" screen.
type ChefReferralProgress struct {
	KitchenName     string     `json:"kitchenName"`
	Status          string     `json:"status"`
	DeliveredOrders int64      `json:"deliveredOrders"`
	MilestoneOrders int        `json:"milestoneOrders"`
	Reward          float64    `json:"reward"`
	CreatedAt       time.Time  `json:"createdAt"`
	RewardedAt      *time.Time `json:"rewardedAt,omitempty"`
}

// GetChefReferralProgress lists a referrer's referrals with live milestone
// progress for the pending ones.
func GetChefReferralProgress(db *gorm.DB, referrerChefID uuid.UUID) []ChefReferralProgress {
	var refs []models.ChefReferral
	db.Where("referrer_chef_id = ?", referrerChefID).Order("created_at DESC").Find(&refs)
	out := make([]ChefReferralProgress, 0, len(refs))
	for _, r := range refs {
		row := ChefReferralProgress{
			Status:          string(r.Status),
			MilestoneOrders: r.MilestoneOrders,
			Reward:          r.ReferrerAmount,
			CreatedAt:       r.CreatedAt,
			RewardedAt:      r.RewardedAt,
		}
		var chef models.ChefProfile
		if err := db.Select("business_name").First(&chef, "id = ?", r.RefereeChefID).Error; err == nil {
			row.KitchenName = chef.BusinessName
		}
		if r.Status == models.ChefReferralPending {
			db.Model(&models.Order{}).
				Where("chef_id = ? AND status = ?", r.RefereeChefID, models.OrderStatusDelivered).
				Count(&row.DeliveredOrders)
		} else {
			row.DeliveredOrders = int64(r.MilestoneOrders)
		}
		out = append(out, row)
	}
	return out
}
