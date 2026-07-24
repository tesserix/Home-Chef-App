package services

// meal_plan_refund_v2_flow.go — the v2 refund executor + chef/admin transitions
// (docs/meal-plan-refund-flow-design.md). ExecuteMealPlanV2Refund is the single money+state seam
// every v2 path funnels through: the >12h auto path (Full → wallet), the admin-pay path (chef's
// Full/Half after admin picks a destination), and the None path (no customer refund, chef paid).
//
// Money rules (all off the fee/GST-EXCLUDED base — see MealPlanRefundAmount):
//   Full → customer gets 100% of base; chef's held transfer fully reversed (chef 0 for the day).
//   Half → customer gets 50%; chef keeps 50% of their net payout (transfer half-reversed).
//   None → customer gets 0; chef keeps 100% (transfer released); day skipped.
// Destination: wallet (CreditWallet → dual-writes the ledger) or source (gateway refund, RBI).
// Idempotent on the day's refund_txn_id. No-op when escrow is off or the plan never captured.

import (
	"fmt"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// dayCommissionRate resolves a day's commission rate (frozen rate, else the live/default).
func dayCommissionRate(tx *gorm.DB, day *models.MealPlanDay) float64 {
	rate := day.CommissionRate
	if rate <= 0 || rate >= 1 {
		rate = GetCommissionRate(tx)
	}
	return rate
}

// ExecuteMealPlanV2Refund resolves one day's refund at the chef-chosen proportion to the chosen
// destination, and drives the day terminal. Runs in the caller's tx.
func ExecuteMealPlanV2Refund(tx *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay, proportion models.RefundProportion, dest models.RefundDestination) error {
	if !MealPlanEscrowActive() || plan.EscrowPaymentID == "" {
		// Nothing captured → no money to move; just terminalize the state.
		return terminalizeV2Day(tx, day, proportion, dest, false)
	}

	// Idempotency: re-read refund_txn_id under a row lock; a prior writer already refunded.
	lockTx := tx
	if tx.Dialector.Name() == "postgres" {
		lockTx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var locked models.MealPlanDay
	if err := lockTx.Select("refund_txn_id", "refund_stage").First(&locked, "id = ?", day.ID).Error; err != nil {
		return fmt.Errorf("v2 refund day %s: lock-read: %w", day.ID, err)
	}
	if locked.RefundTxnID != nil || locked.RefundStage == models.MPRefundResolved {
		day.RefundTxnID = locked.RefundTxnID
		return nil // already resolved by a prior/concurrent writer
	}

	amount := MealPlanRefundAmount(plan, day, proportion)

	// NONE (or a zero base): no customer refund; the chef keeps their full payout (release the
	// held transfer) and the day is skipped (customer forfeits).
	if proportion == models.RefundProportionNone || amount <= 0 {
		if err := ReleaseDayPayout(tx, day); err != nil {
			return fmt.Errorf("v2 refund day %s: release chef payout (none): %w", day.ID, err)
		}
		return terminalizeV2Day(tx, day, models.RefundProportionNone, dest, false)
	}

	// Reverse the chef's held transfer by the refunded proportion (Full → full, Half → half); the
	// chef keeps (1 − proportion) of their net payout. Best-effort on the gateway (a failed
	// reverse is left as re-drivable drift for the payout-reconcile cron), never blocking the
	// customer refund. No-op when the chef has no Route transfer for the day.
	reverseChefTransferForV2(tx, plan, day, proportion)

	reason := fmt.Sprintf("Tiffin %s — refund (%s)", plan.MealPlanNumber, proportion)
	if dest == models.RefundDestinationSource {
		// Original method: reverse the escrow charge to the customer's card/UPI (RBI ~5-7 days).
		refID, err := gatewayRefundToSource(plan, amount, reason, dayRefundKey(day.ID)+":src")
		if err != nil {
			return fmt.Errorf("v2 refund day %s to source: %w", day.ID, err)
		}
		day.RefundDestination = models.RefundDestinationSource
		return terminalizeV2DayWithGatewayRef(tx, day, proportion, refID)
	}

	// Wallet (default, instant): CreditWallet dual-writes into the ledger's user_wallet_refund
	// bucket. Idempotent on dayRefundKey.
	txn, err := CreditWallet(tx, plan.CustomerID, amount, models.WalletSourceRefund, nil, reason, dayRefundKey(day.ID), nil)
	if err != nil {
		return fmt.Errorf("v2 refund day %s to wallet: %w", day.ID, err)
	}
	day.RefundTxnID = &txn.ID
	day.RefundDestination = models.RefundDestinationWallet
	return terminalizeV2Day(tx, day, proportion, models.RefundDestinationWallet, true)
}

// terminalizeV2Day persists the day's terminal v2 state: refunded (a customer refund landed) or
// skipped (none / nothing captured), stage resolved, with the chef's choice + destination.
func terminalizeV2Day(tx *gorm.DB, day *models.MealPlanDay, proportion models.RefundProportion, dest models.RefundDestination, refunded bool) error {
	day.ChefRefundChoice = proportion
	day.RefundStage = models.MPRefundResolved
	if dest != "" {
		day.RefundDestination = dest
	}
	if refunded {
		day.Status = models.MealPlanDayRefunded
	} else {
		day.Status = models.MealPlanDaySkipped
	}
	updates := map[string]any{
		"status":             day.Status,
		"refund_stage":       day.RefundStage,
		"chef_refund_choice": day.ChefRefundChoice,
		"refund_destination": day.RefundDestination,
	}
	if day.RefundTxnID != nil {
		updates["refund_txn_id"] = *day.RefundTxnID
	}
	return tx.Model(&models.MealPlanDay{}).Where("id = ?", day.ID).Updates(updates).Error
}

// terminalizeV2DayWithGatewayRef records a source (gateway) refund: no wallet txn id, but a
// refund reference so the day reads refunded.
func terminalizeV2DayWithGatewayRef(tx *gorm.DB, day *models.MealPlanDay, proportion models.RefundProportion, gatewayRefID string) error {
	day.ChefRefundChoice = proportion
	day.RefundStage = models.MPRefundResolved
	day.RefundDestination = models.RefundDestinationSource
	day.Status = models.MealPlanDayRefunded
	return tx.Model(&models.MealPlanDay{}).Where("id = ?", day.ID).Updates(map[string]any{
		"status":             day.Status,
		"refund_stage":       day.RefundStage,
		"chef_refund_choice": day.ChefRefundChoice,
		"refund_destination": day.RefundDestination,
	}).Error
}

// reverseChefTransferForV2 reverses the chef's held Route transfer by the refunded proportion.
// Best-effort: a failed reverse is logged and left for the payout-reconcile cron. No-op when the
// day has no transfer (e.g. a chef without a Route account) or Razorpay is unavailable.
func reverseChefTransferForV2(tx *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay, proportion models.RefundProportion) {
	if day.PayoutTransferID == "" {
		return
	}
	rz := GetRazorpay()
	if rz == nil {
		return
	}
	factor := refundProportionFactor(proportion)
	if factor <= 0 {
		return
	}
	net := perDayNetPayout(plan, day, dayCommissionRate(tx, day))
	reversePaise := 0 // 0 = full reverse for a Full refund
	if factor < 1 {
		reversePaise = ToPaise(Round2(net * factor))
	}
	if _, err := rz.ReverseTransfer(day.PayoutTransferID, reversePaise); err != nil {
		if !isAlreadyReversedErr(err) {
			log.Printf("v2 refund: reverse transfer %s (proportion %s) failed — reconcile cron will re-drive: %v", day.PayoutTransferID, proportion, err)
		}
	}
}

// gatewayRefundToSource issues a partial refund of the plan's captured escrow payment back to the
// customer's original method (Razorpay). Returns the gateway refund id. idemKey dedups a
// timeout-after-success retry so a day is never refunded twice at the gateway.
func gatewayRefundToSource(plan *models.MealPlan, amount float64, reason, idemKey string) (string, error) {
	rz := GetRazorpay()
	if rz == nil {
		return "", fmt.Errorf("razorpay not configured")
	}
	resp, err := rz.CreateRefund(plan.EscrowPaymentID, &RefundRequest{
		Amount: ToPaise(amount),
		Speed:  "normal",
		Notes: map[string]string{
			"meal_plan": plan.MealPlanNumber, "reason": reason,
		},
		IdempotencyKey: idemKey,
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}
