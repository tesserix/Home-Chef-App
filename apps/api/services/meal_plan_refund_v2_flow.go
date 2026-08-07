package services

// meal_plan_refund_v2_flow.go — the refund executor + chef/admin transitions
// (docs/refund-policy-v3-spec.md). ExecuteMealPlanV2Refund is the single money+state seam every
// path funnels through: the auto-approve path (top tier, no chef step), the customer-medium path,
// and the admin-executed gateway path.
//
// Money rules — v3 (#834). The percentage is a bounded 0–100 the chef agreed within the tier
// floor, applied to the day's refund base — food MINUS the platform commission, plus that day's
// GST and delivery (see MealPlanRefundAmount):
//   P% → customer gets P% of the day's base; the chef's held transfer is reversed by P% of
//        their net, so they keep (100−P)% as prep compensation.
//   0% → no customer refund; the chef keeps 100% (transfer released); the day is skipped.
// The platform retains its commission on a refunded day and is out of pocket only the GST and
// delivery it returns; a GST credit note is issued for the tax so filings stay correct.
// Destination: wallet (CreditWallet → dual-writes the ledger) or source (gateway refund, RBI).
// Idempotent on the day's refund_txn_id. No-op when escrow is off or the plan never captured.

import (
	"fmt"

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

// ExecuteMealPlanV2Refund resolves one day's refund at the chef-agreed percentage to the chosen
// destination, and drives the day terminal. Runs in the caller's tx.
func ExecuteMealPlanV2Refund(tx *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay, percent int, dest models.RefundDestination) error {
	percent = ClampRefundPercent(percent)
	if !MealPlanEscrowActive() || plan.EscrowPaymentID == "" {
		// Nothing captured → no money to move; just terminalize the state.
		return terminalizeV2Day(tx, day, percent, dest, false)
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

	amount := MealPlanRefundAmount(plan, day, percent)

	// 0% (or a zero base): no customer refund; the chef keeps their full payout and
	// the day is skipped (customer forfeits).
	if percent <= 0 || amount <= 0 {
		return terminalizeV2Day(tx, day, 0, dest, false)
	}

	// #937: a paid tiffin day being refunded back to the customer. Recorded inside the
	// caller's tx (savepoint-isolated) so a later gateway failure rolls the event back
	// with the refund — no refund, no ledger entry.
	mpChefID := plan.ChefID
	TrackRiskEvent(tx, RecordRiskEventInput{
		CustomerID: plan.CustomerID,
		Kind:       models.RiskMealPlanRefund,
		SourceKey:  "mealplan-day-refund:" + day.ID.String(),
		OrderID:    day.OrderID,
		ChefID:     &mpChefID,
		Amount:     amount,
		Reason:     fmt.Sprintf("%d%% day refund", percent),
	})

	// Drive the day's hold OUT of the payout-release queue so a refunded day can never also be
	// released to the chef (double-pay). Money-safe for every case; for Half, the chef's kept
	// slice is settled by the payout-reconcile path — this only guarantees no auto-release.
	if err := reverseRefundedDayHold(tx, day.ID); err != nil {
		return fmt.Errorf("v2 refund day %s: hold reversal: %w", day.ID, err)
	}

	reason := MealPlanRefundReason(plan, percent)
	if dest == models.RefundDestinationSource {
		// Original method: reverse the escrow charge to the customer's card/UPI (RBI ~5-7 days).
		refID, err := gatewayRefundToSource(plan, amount, reason, dayRefundKey(day.ID)+":src")
		if err != nil {
			return fmt.Errorf("v2 refund day %s to source: %w", day.ID, err)
		}
		day.RefundDestination = models.RefundDestinationSource
		if err := terminalizeV2DayWithGatewayRef(tx, day, percent, refID); err != nil {
			return err
		}
		return issueRefundCreditNote(tx, plan, day, percent, amount)
	}

	// Wallet (default, instant): CreditWallet dual-writes into the ledger's user_wallet_refund
	// bucket. Idempotent on dayRefundKey.
	txn, err := CreditWallet(tx, plan.CustomerID, amount, models.WalletSourceRefund, nil, reason, dayRefundKey(day.ID), nil)
	if err != nil {
		return fmt.Errorf("v2 refund day %s to wallet: %w", day.ID, err)
	}
	day.RefundTxnID = &txn.ID
	day.RefundDestination = models.RefundDestinationWallet
	if err := terminalizeV2Day(tx, day, percent, models.RefundDestinationWallet, true); err != nil {
		return err
	}
	return issueRefundCreditNote(tx, plan, day, percent, amount)
}

// issueRefundCreditNote records the GST credit note for a landed refund (#834 item 2). v3
// returns the tax the platform collected, so every refund must be matched by a credit note or
// the GST filing overstates output tax from the first refund onward. Idempotent per day.
//
// A zero-GST refund (legacy plan with no snapshotted tax) needs no note. Runs INSIDE the refund
// tx: a note that cannot be written rolls the refund back rather than returning tax silently.
func issueRefundCreditNote(tx *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay, percent int, refundAmount float64) error {
	gst := MealPlanRefundGSTComponent(plan, day, percent)
	if gst <= 0 {
		return nil
	}
	return IssueMealPlanDayCreditNote(tx, plan, day, refundAmount, gst)
}

// terminalizeV2Day persists the day's terminal state: refunded (a customer refund landed) or
// skipped (0% / nothing captured), stage resolved, with the agreed percentage + destination.
// The legacy chef_refund_choice enum is written alongside as a coarse label so pre-v3 readers
// (and the sqlite fixtures) keep working; refund_percent is the authoritative value.
func terminalizeV2Day(tx *gorm.DB, day *models.MealPlanDay, percent int, dest models.RefundDestination, refunded bool) error {
	percent = ClampRefundPercent(percent)
	day.RefundPercent = &percent
	day.ChefRefundChoice = models.RefundProportionLabel(percent)
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
		"refund_percent":     percent,
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
func terminalizeV2DayWithGatewayRef(tx *gorm.DB, day *models.MealPlanDay, percent int, _ string) error {
	percent = ClampRefundPercent(percent)
	day.RefundPercent = &percent
	day.ChefRefundChoice = models.RefundProportionLabel(percent)
	day.RefundStage = models.MPRefundResolved
	day.RefundDestination = models.RefundDestinationSource
	day.Status = models.MealPlanDayRefunded
	return tx.Model(&models.MealPlanDay{}).Where("id = ?", day.ID).Updates(map[string]any{
		"status":             day.Status,
		"refund_stage":       day.RefundStage,
		"refund_percent":     percent,
		"chef_refund_choice": day.ChefRefundChoice,
		"refund_destination": day.RefundDestination,
	}).Error
}

// gatewayRefundToSource issues a partial refund of the plan's captured escrow payment back
// to the customer's original method. Returns the gateway refund id. idemKey dedups a
// timeout-after-success retry so a day is never refunded twice at the gateway — on Cashfree
// it becomes the refund_id itself.
//
// Scoped to the plan's advance ORDER, not its payment: Cashfree refunds are order-scoped.
func gatewayRefundToSource(plan *models.MealPlan, amount float64, reason, idemKey string) (string, error) {
	cf := GetCashfreeFor(plan.Mode)
	if cf == nil {
		return "", fmt.Errorf("cashfree not configured")
	}
	note := plan.MealPlanNumber + ": " + reason
	if len(note) > 100 {
		note = note[:100]
	}
	resp, err := cf.CreateRefund(plan.GatewayOrderID, &CashfreeRefundRequest{
		AmountPaise:    cashfreeAmount(ToPaise(amount)),
		Note:           note,
		IdempotencyKey: idemKey,
	})
	if err != nil {
		return "", err
	}
	return resp.RefundID, nil
}
