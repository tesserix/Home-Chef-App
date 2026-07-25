package services

// account_blockers.go — the preconditions that must be clear before an account
// may be deleted.
//
// Deletion is refused (409) while the user still has work or money in flight:
// an active order, meal-plan escrow the platform is holding, store credit, an
// unreleased chef payout, or a delivery in hand. The alternative — deleting
// anyway — either silently forfeits the user's money or strands escrow with no
// party left to release it to. Deactivation stays available as the immediate
// escape hatch, which is why blocking here is not a dead end for the user.
//
// The same computation backs both GET /me/deletion-eligibility (so the UI can
// warn before the confirm screen) and POST /me/delete (enforcement), so the app
// can never present a delete button that then 409s unexpectedly.

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// Blocker is one reason an account cannot be deleted yet. Code is stable and
// machine-readable so the mobile apps can deep-link to the right screen; Label
// is the human sentence shown when they have no specific handling for a code.
type Blocker struct {
	Code   string  `json:"code"`
	Label  string  `json:"label"`
	Count  int64   `json:"count,omitempty"`
	Amount float64 `json:"amount,omitempty"`
}

// Blocker codes. Stable API surface — the mobile apps switch on these.
const (
	BlockerActiveOrders   = "active_orders"
	BlockerMealPlanEscrow = "mealplan_escrow"
	BlockerWalletBalance  = "wallet_balance"
	BlockerPendingPayout  = "pending_payout"
	BlockerActiveDelivery = "active_delivery"
)

// activeOrderStatuses are the order states that still need someone to act. An
// order outside this set is finished (delivered) or dead (cancelled) and never
// blocks deletion.
func activeOrderStatuses() []models.OrderStatus {
	return []models.OrderStatus{
		models.OrderStatusPending,
		models.OrderStatusAccepted,
		models.OrderStatusPreparing,
		models.OrderStatusReady,
		models.OrderStatusPickedUp,
		models.OrderStatusDelivering,
	}
}

// liveMealPlanStatuses are plans where the platform is holding the customer's
// money in escrow, or still owes the chef a decision. Completed, cancelled and
// expired plans have already settled.
func liveMealPlanStatuses() []models.MealPlanStatus {
	return []models.MealPlanStatus{
		models.MealPlanPendingChef,
		models.MealPlanChefAcceptedFull,
		models.MealPlanChefModified,
		models.MealPlanAwaitingCustomer,
		models.MealPlanConfirmed,
		models.MealPlanActive,
	}
}

// unreleasedPayoutStatuses are holds where the chef is owed money the platform
// has not yet transferred. Withheld and reversed are terminal with no money
// owed; released has already paid out.
func unreleasedPayoutStatuses() []models.PayoutHoldStatus {
	return []models.PayoutHoldStatus{
		models.PayoutHoldAwaitingConfirmation,
		models.PayoutHoldReleaseEligible,
		models.PayoutHoldDisputed,
	}
}

// CustomerDeletionBlockers returns everything stopping this customer from
// deleting: orders in flight, escrowed meal plans, and any store credit they
// would otherwise silently forfeit.
func CustomerDeletionBlockers(db *gorm.DB, userID uuid.UUID) []Blocker {
	blockers := make([]Blocker, 0, 3)

	var activeOrders int64
	db.Model(&models.Order{}).
		Where("customer_id = ? AND status IN ?", userID, activeOrderStatuses()).
		Count(&activeOrders)
	if activeOrders > 0 {
		blockers = append(blockers, Blocker{
			Code:  BlockerActiveOrders,
			Label: "You have orders still in progress.",
			Count: activeOrders,
		})
	}

	var escrowCount int64
	var escrowTotal float64
	db.Model(&models.MealPlan{}).
		Where("customer_id = ? AND status IN ?", userID, liveMealPlanStatuses()).
		Count(&escrowCount)
	if escrowCount > 0 {
		// COALESCE so a plan set with a NULL total scans into 0 rather than
		// failing the whole eligibility check.
		db.Model(&models.MealPlan{}).
			Select("COALESCE(SUM(total), 0)").
			Where("customer_id = ? AND status IN ?", userID, liveMealPlanStatuses()).
			Scan(&escrowTotal)
		blockers = append(blockers, Blocker{
			Code:   BlockerMealPlanEscrow,
			Label:  "You have an active meal plan we are still holding payment for.",
			Count:  escrowCount,
			Amount: escrowTotal,
		})
	}

	var wallet models.Wallet
	if err := db.Where("user_id = ?", userID).First(&wallet).Error; err == nil && wallet.Balance > 0 {
		blockers = append(blockers, Blocker{
			Code:   BlockerWalletBalance,
			Label:  "You still have store credit in your wallet.",
			Amount: wallet.Balance,
		})
	}

	return blockers
}

// ChefDeletionBlockers returns everything stopping this chef from deleting:
// orders they still owe a customer, live meal plans, and payouts the platform
// has not yet transferred to them.
//
// chefID is the ChefProfile id (orders and plans reference the profile, not the
// user). A user with no chef profile has no chef-side blockers.
func ChefDeletionBlockers(db *gorm.DB, chefID uuid.UUID) []Blocker {
	blockers := make([]Blocker, 0, 3)

	var activeOrders int64
	db.Model(&models.Order{}).
		Where("chef_id = ? AND status IN ?", chefID, activeOrderStatuses()).
		Count(&activeOrders)
	if activeOrders > 0 {
		blockers = append(blockers, Blocker{
			Code:  BlockerActiveOrders,
			Label: "You have orders you still need to fulfil.",
			Count: activeOrders,
		})
	}

	var planCount int64
	db.Model(&models.MealPlan{}).
		Where("chef_id = ? AND status IN ?", chefID, liveMealPlanStatuses()).
		Count(&planCount)
	if planCount > 0 {
		blockers = append(blockers, Blocker{
			Code:  BlockerMealPlanEscrow,
			Label: "You have meal plans with days still to cook.",
			Count: planCount,
		})
	}

	// Count only, no Amount: the chef's cut is computed at settlement from the
	// subtotal, delivery-fee-final and commission rules — there is no stored
	// earnings column, and quoting a total derived here would drift from what
	// the payout engine actually pays.
	var payoutCount int64
	db.Model(&models.Order{}).
		Where("chef_id = ? AND payout_hold_status IN ?", chefID, unreleasedPayoutStatuses()).
		Count(&payoutCount)
	if payoutCount > 0 {
		blockers = append(blockers, Blocker{
			Code:  BlockerPendingPayout,
			Label: "You have earnings that have not been paid out yet.",
			Count: payoutCount,
		})
	}

	return blockers
}

// DriverDeletionBlockers returns any delivery this driver is still carrying.
// partnerID is the DeliveryPartner id.
func DriverDeletionBlockers(db *gorm.DB, partnerID uuid.UUID) []Blocker {
	blockers := make([]Blocker, 0, 1)

	var active int64
	db.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status NOT IN ?", partnerID,
			[]models.DeliveryStatus{
				models.DeliveryDelivered,
				models.DeliveryFailed,
				models.DeliveryReturned,
				models.DeliveryCancelled,
			}).
		Count(&active)
	if active > 0 {
		blockers = append(blockers, Blocker{
			Code:  BlockerActiveDelivery,
			Label: "You have a delivery still in progress.",
			Count: active,
		})
	}

	return blockers
}

// DeletionBlockers dispatches to the role-specific checks and always includes
// the customer-side checks: every user can place orders and hold wallet credit
// regardless of the role they signed up under, so a chef with leftover store
// credit must not lose it silently.
func DeletionBlockers(db *gorm.DB, user models.User) []Blocker {
	blockers := CustomerDeletionBlockers(db, user.ID)

	switch user.Role {
	case models.RoleChef:
		var chef models.ChefProfile
		if err := db.Where("user_id = ?", user.ID).First(&chef).Error; err == nil {
			blockers = append(blockers, ChefDeletionBlockers(db, chef.ID)...)
		}
	case models.RoleDelivery:
		var partner models.DeliveryPartner
		if err := db.Where("user_id = ?", user.ID).First(&partner).Error; err == nil {
			blockers = append(blockers, DriverDeletionBlockers(db, partner.ID)...)
		}
	}

	return blockers
}

// CanDetermineBlockers reports whether the blocker queries can actually run.
//
// The Count calls above ignore their errors, which means a failed query is
// indistinguishable from "nothing outstanding" — a guard protecting escrow,
// wallet credit and unreleased payouts must not fail OPEN like that. Callers
// check this first and refuse the deletion if eligibility cannot be
// established, rather than deleting an account with money still attached.
func CanDetermineBlockers(db *gorm.DB, user models.User) error {
	var n int64
	if err := db.Model(&models.Order{}).
		Where("customer_id = ?", user.ID).Count(&n).Error; err != nil {
		return fmt.Errorf("account: cannot read orders to check deletion eligibility: %w", err)
	}
	if err := db.Model(&models.MealPlan{}).
		Where("customer_id = ?", user.ID).Count(&n).Error; err != nil {
		return fmt.Errorf("account: cannot read meal plans to check deletion eligibility: %w", err)
	}
	return nil
}
