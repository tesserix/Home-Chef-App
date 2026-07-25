package services

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// PaymentModeForChef resolves which world a NEW record for this chef belongs to.
//
// This is the single place ChefProfile.Mode is read for money purposes. Every
// operation on an already-created record reads that record's own Mode instead,
// because an admin may flip the chef at any time and a refund must still route
// to the gateway that took the payment.
//
// Fails safe to live on any error. The asymmetry matters: a wrong "live" answer
// produces a loud gateway failure someone will notice, while a wrong "test"
// answer produces a real order that quietly captured no money.
func PaymentModeForChef(chefID uuid.UUID) string {
	if database.DB == nil {
		return models.ChefModeLive
	}
	var mode string
	err := database.DB.Model(&models.ChefProfile{}).
		Where("id = ?", chefID).Select("mode").Scan(&mode).Error
	if err != nil {
		log.Printf("payment-mode: chef %s lookup failed (%v) — defaulting to live", chefID, err)
		return models.ChefModeLive
	}
	return models.NormalizeMode(mode)
}

// PaymentModeForOrder reads an EXISTING order's snapshotted mode by id, for the
// payout, refund and reconciliation paths that only carry an order id.
//
// Note the difference from PaymentModeForChef: this reads the order, not the
// chef, precisely because the chef may have been flipped since the payment was
// taken. Refunding a test-mode payment through live credentials would 400 at
// the gateway and strand the refund.
//
// Fails safe to live, for the same reason PaymentModeForChef does.
func PaymentModeForOrder(orderID uuid.UUID) string {
	if database.DB == nil {
		return models.ChefModeLive
	}
	var mode string
	err := database.DB.Model(&models.Order{}).
		Where("id = ?", orderID).Select("mode").Scan(&mode).Error
	if err != nil {
		log.Printf("payment-mode: order %s lookup failed (%v) — defaulting to live", orderID, err)
		return models.ChefModeLive
	}
	return models.NormalizeMode(mode)
}

// PartitionForChef returns the mode plus the chef's open test session, ready to
// stamp onto a new record. Callers set both together so a test row is always
// attributable to the session that produced it.
func PartitionForChef(chef *models.ChefProfile) models.ModePartition {
	if chef == nil {
		return models.ModePartition{Mode: models.ChefModeLive}
	}
	p := models.ModePartition{Mode: models.NormalizeMode(chef.Mode)}
	if models.IsTestMode(p.Mode) {
		p.TestSessionID = chef.ActiveTestSessionID
	}
	return p
}

// IsTestOrder reports whether an order belongs to the test partition.
func IsTestOrder(o *models.Order) bool {
	return o != nil && models.IsTestMode(o.Mode)
}

// ShouldDispatchToProvider reports whether an order may be handed to an external
// 3PL courier. Test orders never may: a real rider would be dispatched to a real
// address for an order nobody is cooking. Pickup and chef self-delivery still
// work, and our own platform drivers can still be assigned, so the driver flow
// stays fully testable end to end.
func ShouldDispatchToProvider(o *models.Order) bool { return !IsTestOrder(o) }

// WalletAllowedForOrder reports whether wallet may be used as a payment source
// or a refund destination for this order. Never for a test order: a refund into
// the real wallet would mint spendable balance out of sandbox money, and that
// balance is usable against live kitchens.
func WalletAllowedForOrder(o *models.Order) bool { return !IsTestOrder(o) }

// LoyaltyAllowedForOrder reports whether an order participates in loyalty.
// Never for a test order: points earned on a fake order would be redeemable
// against a real kitchen. Referral rewards and promo-usage counters follow the
// same rule for the same reason.
func LoyaltyAllowedForOrder(o *models.Order) bool { return !IsTestOrder(o) }

// PayoutAllowedForOrder reports whether an order enters the real payout and
// settlement engine. Test orders do not — their Route transfers are created and
// released inside the Razorpay TEST account, so the split-payment path is
// genuinely exercised without anything reaching a real bank.
func PayoutAllowedForOrder(o *models.Order) bool { return !IsTestOrder(o) }

// OrderIsTestMode reports whether an order id belongs to the test partition,
// read through the caller's transaction so a guard inside a tx sees the same
// snapshot as the write it is guarding.
//
// Fails safe to FALSE (live) when the row can't be read: a lookup blip must not
// start rejecting real wallet or loyalty movement on live orders.
func OrderIsTestMode(db *gorm.DB, orderID uuid.UUID) bool {
	if db == nil {
		return false
	}
	var mode string
	if err := db.Model(&models.Order{}).
		Where("id = ?", orderID).Select("mode").Scan(&mode).Error; err != nil {
		return false
	}
	return models.IsTestMode(mode)
}
