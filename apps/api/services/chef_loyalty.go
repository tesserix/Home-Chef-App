package services

import (
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// chef_loyalty.go — chef-side loyalty points. Chefs earn points on every
// DELIVERED order (rate × order subtotal); once the balance crosses the
// conversion threshold they convert it into a cashback ChefBonus that
// ApplyChefBonusesToStatement adds onto their next weekly settlement.

// ChefLoyaltyConfig is the admin-tunable program config, stored as
// PlatformSettings `chef_loyalty.*` keys.
type ChefLoyaltyConfig struct {
	Enabled bool `json:"enabled"`
	// EarnRate is points per ₹1 of delivered-order subtotal.
	EarnRate float64 `json:"earnRate"`
	// RedeemRate is ₹ per point at conversion (10,000 pts × 0.005 = ₹50).
	RedeemRate float64 `json:"redeemRate"`
	// MinConvertPoints is the balance a chef must reach before converting.
	MinConvertPoints float64 `json:"minConvertPoints"`
}

// GetChefLoyaltyConfig reads the program config with working defaults.
//
// What the programme COSTS is earn × redeem, not either number alone. At 1 point
// per ₹1 and ₹0.05 a point it was 5% of subtotal — against a 6% commission, so
// it consumed five sixths of the platform's take on every delivered order and
// lost money once the ~2% payment MDR is counted.
//
// The earn rate stays at 1 point per ₹1 deliberately: a ₹400 order showing 400
// points is what makes the programme feel worth chasing, and it is the point
// VALUE that decides the cost. At ₹0.005 the programme is 0.5% of subtotal —
// the same giveaway the customer programme runs at, and one twelfth of
// commission rather than five sixths.
//
// Every field is a PlatformSettings key, so the rate is an admin decision at
// runtime and this is only where it starts.
func GetChefLoyaltyConfig(db *gorm.DB) ChefLoyaltyConfig {
	cfg := ChefLoyaltyConfig{Enabled: true, EarnRate: 1, RedeemRate: 0.005, MinConvertPoints: 10000}
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "chef_loyalty.%").Find(&settings)
	for _, s := range settings {
		switch s.Key {
		case "chef_loyalty.enabled":
			cfg.Enabled = s.Value == "true" || s.Value == "1"
		case "chef_loyalty.earn_rate":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil && v >= 0 {
				cfg.EarnRate = v
			}
		case "chef_loyalty.redeem_rate":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil && v >= 0 {
				cfg.RedeemRate = v
			}
		case "chef_loyalty.min_convert_points":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil && v > 0 {
				cfg.MinConvertPoints = v
			}
		}
	}
	return cfg
}

// EarnChefLoyaltyForOrder credits the chef points for one delivered order.
// Idempotent via the unique "cheforder:<orderID>" source key — a re-stamped
// delivered status never double-earns. Never fatal to the caller.
func EarnChefLoyaltyForOrder(db *gorm.DB, orderID uuid.UUID) {
	cfg := GetChefLoyaltyConfig(db)
	if !cfg.Enabled || cfg.EarnRate <= 0 {
		return
	}

	var order models.Order
	if err := db.Select("id, chef_id, status, subtotal").First(&order, "id = ?", orderID).Error; err != nil {
		return
	}
	if order.Status != models.OrderStatusDelivered || order.Subtotal <= 0 {
		return
	}
	var chef models.ChefProfile
	if err := db.Select("id, user_id").First(&chef, "id = ?", order.ChefID).Error; err != nil {
		return
	}

	points := Round2(order.Subtotal * cfg.EarnRate)
	if points <= 0 {
		return
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		txn := models.ChefLoyaltyTxn{
			ChefID:      chef.ID,
			UserID:      chef.UserID,
			Kind:        models.ChefLoyaltyEarnOrder,
			Points:      points,
			SourceKey:   "cheforder:" + orderID.String(),
			OrderID:     &order.ID,
			Description: "Points for a delivered order",
		}
		if err := tx.Create(&txn).Error; err != nil {
			// A source-key collision means this order already earned — success.
			var existing models.ChefLoyaltyTxn
			if e := tx.Where("source_key = ?", txn.SourceKey).First(&existing).Error; e == nil {
				return nil
			}
			return err
		}
		return addChefLoyaltyPoints(tx, chef.ID, chef.UserID, points)
	})
	if err != nil {
		log.Printf("chef loyalty: earn for order %s failed: %v", orderID, err)
	}
}

// addChefLoyaltyPoints upserts the account row and adds `points` to both the
// live balance and the lifetime total.
func addChefLoyaltyPoints(tx *gorm.DB, chefID, userID uuid.UUID, points float64) error {
	res := tx.Model(&models.ChefLoyaltyAccount{}).
		Where("chef_id = ?", chefID).
		Updates(map[string]any{
			"points":          gorm.Expr("points + ?", points),
			"lifetime_points": gorm.Expr("lifetime_points + ?", points),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}
	acct := models.ChefLoyaltyAccount{ChefID: chefID, UserID: userID, Points: points, LifetimePoints: points}
	if err := tx.Create(&acct).Error; err != nil {
		// Lost the create race — retry the increment against the winner's row.
		return tx.Model(&models.ChefLoyaltyAccount{}).
			Where("chef_id = ?", chefID).
			Updates(map[string]any{
				"points":          gorm.Expr("points + ?", points),
				"lifetime_points": gorm.Expr("lifetime_points + ?", points),
			}).Error
	}
	return nil
}

var (
	ErrChefLoyaltyDisabled     = errors.New("the chef rewards program is currently disabled")
	ErrChefLoyaltyBelowMinimum = errors.New("not enough points to convert yet")
)

// ConvertChefLoyalty converts the chef's ENTIRE current points balance into a
// cashback ChefBonus once it crosses the configured threshold. The guarded
// decrement (`points >= balance`) makes a concurrent double-convert impossible;
// the cashback lands on the next weekly settlement.
func ConvertChefLoyalty(db *gorm.DB, chefID, userID uuid.UUID) (*models.ChefBonus, float64, error) {
	cfg := GetChefLoyaltyConfig(db)
	if !cfg.Enabled {
		return nil, 0, ErrChefLoyaltyDisabled
	}

	var acct models.ChefLoyaltyAccount
	if err := db.Where("chef_id = ?", chefID).First(&acct).Error; err != nil {
		return nil, 0, ErrChefLoyaltyBelowMinimum
	}
	points := acct.Points
	if points < cfg.MinConvertPoints {
		return nil, 0, ErrChefLoyaltyBelowMinimum
	}
	amount := Round2(points * cfg.RedeemRate)
	if amount <= 0 {
		return nil, 0, ErrChefLoyaltyBelowMinimum
	}

	var bonus *models.ChefBonus
	err := db.Transaction(func(tx *gorm.DB) error {
		// Claim the points first: the balance guard loses cleanly if a
		// concurrent conversion (or a fresher earn changing the sum) got there
		// before us.
		res := tx.Model(&models.ChefLoyaltyAccount{}).
			Where("chef_id = ? AND points >= ?", chefID, points).
			Update("points", gorm.Expr("points - ?", points))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrChefLoyaltyBelowMinimum
		}

		b := models.ChefBonus{
			ChefID:    chefID,
			UserID:    userID,
			Kind:      models.ChefBonusLoyaltyCashback,
			Status:    models.ChefBonusPending,
			SourceKey: "chefconv:" + uuid.New().String(),
			Currency:  EarningsCurrency,
			Amount:    amount,
			Reason:    fmt.Sprintf("Cashback — converted %.0f reward points", points),
		}
		if err := tx.Create(&b).Error; err != nil {
			return err
		}
		txn := models.ChefLoyaltyTxn{
			ChefID:      chefID,
			UserID:      userID,
			Kind:        models.ChefLoyaltyConvert,
			Points:      -points,
			SourceKey:   "chefconvtxn:" + b.ID.String(),
			BonusID:     &b.ID,
			Description: fmt.Sprintf("Converted to ₹%.2f cashback", amount),
		}
		if err := tx.Create(&txn).Error; err != nil {
			return err
		}
		bonus = &b
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return bonus, points, nil
}

// OnChefOrderDelivered runs every chef-side reward side-effect of a delivered
// order: loyalty earn + referral milestone check. Call it from each delivered
// transition; both halves are idempotent and never fatal.
func OnChefOrderDelivered(db *gorm.DB, orderID uuid.UUID) {
	var order models.Order
	if err := db.Select("id, chef_id").First(&order, "id = ?", orderID).Error; err != nil {
		return
	}
	EarnChefLoyaltyForOrder(db, orderID)
	MaybeGrantChefReferralReward(db, order.ChefID)
}
