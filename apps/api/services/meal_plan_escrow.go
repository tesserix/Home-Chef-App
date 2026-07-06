package services

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// meal_plan_escrow.go — the tiffin meal-plan escrow money flow (#194).
//
// Flow (capture-at-create, because accept-all auto-confirms with no later
// customer-present step):
//   1. Booking: one Razorpay charge for the full requested total → platform
//      account. (CreateMealPlanAdvanceOrder + VerifyMealPlanAdvance.)
//   2. Confirm (accept-all or customer-approve): a per-day on-hold Route
//      transfer to the chef for each ACCEPTED day (HoldChefPayouts), and a
//      wallet refund of any declined days (RefundDays).
//   3. Per delivered day: release that day's held transfer (ReleaseDayPayout).
//   4. Skip / no-show / expiry / reject: refund the affected days to wallet.
//
// EVERYTHING here is gated by config.MealPlanEscrowEnabled and is a safe no-op
// when the flag is off — so the negotiation handshake (#195/#196) and the
// per-day fulfilment pipeline (#197) work unchanged until the Razorpay paths are
// sandbox-verified. Idempotency: wallet refunds key on "mealplan-refund:<dayID>"
// (unique-indexed in the ledger); releases DB-guard on PayoutTransferID.

// MealPlanEscrowActive reports whether the escrow money flow is switched on.
func MealPlanEscrowActive() bool {
	return config.AppConfig != nil && config.AppConfig.MealPlanEscrowEnabled
}

// dayRefundKey is the idempotency key for refunding a single meal-plan day.
func dayRefundKey(dayID uuid.UUID) string {
	return "mealplan-refund:" + dayID.String()
}

// MealPlanFeeTotals computes the GST + per-day delivery the customer pays ON TOP
// of the food subtotal, from the current platform policy. The chef is paid only
// the food price; the platform keeps GST + delivery (it remits the tax and covers
// logistics). These are snapshotted onto the plan (Subtotal/Tax/Total) at booking
// so later per-day refunds don't drift if policy changes. Returns (tax, delivery).
func MealPlanFeeTotals(subtotal float64, numDays int) (float64, float64) {
	policy := GetPlatformPolicy()
	tax := Round2(subtotal * (policy.TaxPercent / 100.0))
	delivery := Round2(policy.BaseDeliveryFee * float64(numDays))
	return tax, delivery
}

// perDayGross is the full amount the customer paid for a single day — food +
// proportional GST + flat per-day delivery — derived from the plan's snapshotted
// totals (not live policy), so a per-day refund makes the customer whole for the
// whole day they paid for regardless of later policy changes.
func perDayGross(plan *models.MealPlan, day *models.MealPlanDay) float64 {
	n := len(plan.Days)
	if plan.Subtotal <= 0 || n == 0 {
		return day.Price
	}
	tax := plan.Tax * (day.Price / plan.Subtotal)
	delivery := (plan.Total - plan.Subtotal - plan.Tax) / float64(n)
	return Round2(day.Price + tax + delivery)
}

// perDayNetPayout is the chef's NET payout for a single day — the amount the held
// Route transfer must carry (#518). It mirrors ComputeOrderEarnings per-day so the
// meal-plan escrow pays the chef on the SAME basis as regular orders:
//
//	dayFoodGST = plan.Tax × (day.Price / plan.Subtotal)   (proportional food GST)
//	gross      = day.Price + dayFoodGST                    (chef income; delivery is the driver's, excluded)
//	commission = rate × day.Price                          (platform commission on the food subtotal only)
//	tds        = RateTDS × gross                            (§194-O, on gross)
//	net        = gross − commission − tds
//
// A plan with no snapshotted subtotal falls back to food-only gross (no GST). This
// closes the leak where HoldChefPayouts paid the GROSS food price with no commission
// or TDS, unlike the order path (#390).
func perDayNetPayout(plan *models.MealPlan, day *models.MealPlanDay, rate float64) float64 {
	if rate <= 0 || rate >= 1 {
		rate = DefaultCommissionRate
	}
	dayFoodGST := 0.0
	if plan.Subtotal > 0 {
		dayFoodGST = plan.Tax * (day.Price / plan.Subtotal)
	}
	gross := day.Price + dayFoodGST
	commission := rate * day.Price
	tds := RateTDS * gross
	return Round2(gross - commission - tds)
}

// CreateMealPlanAdvanceOrder creates the Razorpay order for the full plan total
// at booking time and stamps RazorpayOrderID. Returns the order id for the
// client checkout. No-op (empty id) when escrow is off.
func CreateMealPlanAdvanceOrder(plan *models.MealPlan) (string, error) {
	if !MealPlanEscrowActive() {
		return "", nil
	}
	rz := GetRazorpay()
	if rz == nil {
		return "", fmt.Errorf("razorpay not configured")
	}
	order, err := rz.CreateOrder(&OrderRequest{
		Amount:   ToPaise(plan.Total),
		Currency: plan.Currency,
		Receipt:  plan.MealPlanNumber,
		Notes:    map[string]string{"meal_plan_id": plan.ID.String(), "kind": "tiffin_advance"},
	})
	if err != nil {
		return "", fmt.Errorf("create advance order: %w", err)
	}
	return order.ID, nil
}

// VerifyMealPlanAdvance confirms the customer's advance payment was captured and
// stamps EscrowPaymentID on the plan. No-op when escrow is off.
//
// SECURITY: binds the fetched payment to THIS plan's advance order and amount and
// verifies the Checkout signature — without this any captured payment (e.g. a ₹1
// payment reused across plans) could mark a large plan "paid" from the platform
// escrow. Mirrors the order payment-verify binding in handlers/payment.go.
func VerifyMealPlanAdvance(tx *gorm.DB, plan *models.MealPlan, paymentID, signature string) error {
	if !MealPlanEscrowActive() {
		return nil
	}
	rz := GetRazorpay()
	if rz == nil {
		return fmt.Errorf("razorpay not configured")
	}
	if plan.RazorpayOrderID == "" {
		return fmt.Errorf("no advance order on this plan")
	}
	pay, err := rz.FetchPayment(paymentID)
	if err != nil {
		return fmt.Errorf("fetch advance payment: %w", err)
	}
	if pay.Status != "captured" {
		return fmt.Errorf("advance payment not captured (status=%s)", pay.Status)
	}
	if pay.OrderID != plan.RazorpayOrderID {
		return fmt.Errorf("advance payment does not belong to this plan")
	}
	if pay.Amount < ToPaise(plan.Total) {
		return fmt.Errorf("advance payment amount does not match the plan total")
	}
	// Enforced when the client sends it (the customer app always does); the
	// order+amount binding above is the hard gate and doesn't rely on the client.
	if signature != "" && !VerifyPaymentSignature(plan.RazorpayOrderID, paymentID, signature) {
		return fmt.Errorf("advance payment signature verification failed")
	}
	plan.EscrowPaymentID = paymentID
	return tx.Model(&models.MealPlan{}).Where("id = ?", plan.ID).
		Update("escrow_payment_id", paymentID).Error
}

// HoldChefPayouts creates one on-hold Route transfer per accepted day to the
// chef's linked account and stamps PayoutTransferID on each day. Called inside
// the confirm transaction. No-op when escrow is off.
func HoldChefPayouts(tx *gorm.DB, plan *models.MealPlan, chefAccount string) error {
	if !MealPlanEscrowActive() {
		return nil
	}
	rz := GetRazorpay()
	if rz == nil {
		return fmt.Errorf("razorpay not configured")
	}
	if chefAccount == "" {
		return fmt.Errorf("chef has no Razorpay linked account")
	}
	// SECURITY: never hold chef payouts for a plan whose advance wasn't captured
	// — the on-hold transfers draw from the platform balance, so holding against
	// an unpaid plan would pay the chef with the platform's own money.
	if plan.EscrowPaymentID == "" {
		return fmt.Errorf("cannot hold payouts: advance payment not captured for plan %s", plan.ID)
	}
	// #518: the held transfer must be the chef's NET (food + per-day GST − commission
	// − TDS), the SAME basis as the order path, NOT the gross food price. Resolve the
	// commission rate once for the whole plan so every day is held at one consistent
	// rate.
	rate := GetCommissionRate(tx)
	for i := range plan.Days {
		d := &plan.Days[i]
		if d.PayoutTransferID != "" {
			continue // already held (idempotent)
		}
		if !isPayableDayStatus(d.Status) {
			continue
		}
		heldPaise := ToPaise(perDayNetPayout(plan, d, rate))
		tr, err := rz.CreateTransfer(&DirectTransferRequest{
			Account:  chefAccount,
			Amount:   heldPaise,
			Currency: plan.Currency,
			OnHold:   true,
			Notes:    map[string]string{"meal_plan_id": plan.ID.String(), "day_id": d.ID.String()},
			// One hold per day (DB-guarded by PayoutTransferID); a retry re-derives the
			// same per-day key so Razorpay dedups it. #574.
			IdempotencyKey: HoldPayoutIdempotencyKey("mealplanday", d.ID),
		})
		if err != nil {
			return fmt.Errorf("hold payout for day %s: %w", d.ID, err)
		}
		d.PayoutTransferID = tr.ID
		if err := tx.Model(&models.MealPlanDay{}).Where("id = ?", d.ID).
			Update("payout_transfer_id", tr.ID).Error; err != nil {
			return err
		}
		auditTransferMovement(auditTransferHold, aggTypeMealPlanDay, d.ID, tr.ID, heldPaise, "meal-plan day confirmed — chef payout held")
	}
	return nil
}

// ReleaseDayPayout releases the held transfer for a delivered day. DB-guarded:
// only acts if a transfer id is present. No-op when escrow is off. Since #387 the
// delivery hook parks a hold instead of calling this; it remains the seam the
// admin payout queue (#388) drives off release_eligible and the payout-reconcile
// cron (#459) re-drives for drift rows.
//
// IDEMPOTENT RE-DRIVE (#459): ReleaseTransfer is a PATCH on_hold:false that
// Razorpay treats as a no-op on an already-released transfer, but a partially-
// applied release can still surface an "already released" gateway error. Tolerate
// it (log + return nil) so a re-drive cannot loop forever. TRADEOFF: string-
// matching the gateway message is the fragile part — the durable guard is the
// reconcile only re-driving rows with payout_settled_at IS NULL (a settled day is
// never re-driven); a single-transfer FetchTransfer on the client would let us
// verify on-hold state instead (out of scope, noted as a follow-up).
func ReleaseDayPayout(tx *gorm.DB, day *models.MealPlanDay) error {
	if !MealPlanEscrowActive() || day.PayoutTransferID == "" {
		return nil
	}
	rz := GetRazorpay()
	if rz == nil {
		return fmt.Errorf("razorpay not configured")
	}
	if _, err := rz.ReleaseTransfer(day.PayoutTransferID); err != nil {
		if isAlreadyReleasedErr(err) {
			log.Printf("meal-plan release: transfer %s already released (idempotent re-drive): %v", day.PayoutTransferID, err)
			return nil
		}
		return fmt.Errorf("release payout %s: %w", day.PayoutTransferID, err)
	}
	auditTransferMovement(auditTransferRelease, aggTypeMealPlanDay, day.ID, day.PayoutTransferID, 0, "meal-plan day delivered — chef payout released")
	return nil
}

// isAlreadyReleasedErr reports whether a gateway error indicates the transfer was
// already released / no longer on hold — the idempotent re-drive case
// ReleaseDayPayout tolerates so the reconcile cannot loop on a settled transfer.
func isAlreadyReleasedErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already released") ||
		strings.Contains(msg, "already settled") ||
		strings.Contains(msg, "not on hold") ||
		strings.Contains(msg, "not_on_hold")
}

// isAlreadyReversedErr reports whether a gateway error indicates the transfer was
// already reversed — the idempotent case the reverse seam tolerates so a residual
// concurrent double-dispatch (#508) or a reconcile re-drive can't loop/fail on a
// transfer that's already been clawed back.
func isAlreadyReversedErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already reversed") ||
		strings.Contains(msg, "fully reversed") ||
		strings.Contains(msg, "already refunded") ||
		strings.Contains(msg, "not_reversible") ||
		strings.Contains(msg, "not reversible")
}

// RefundDay refunds a single day to the customer's wallet (idempotent on the
// day id) and stamps RefundTxnID. If the day's payout was already held, the
// held transfer is reversed first so the chef isn't paid for a refunded day.
// No-op when escrow is off.
func RefundDay(tx *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay, reason string) error {
	if !MealPlanEscrowActive() || day.RefundTxnID != nil {
		return nil
	}
	rz := GetRazorpay()
	if rz == nil {
		return fmt.Errorf("razorpay not configured")
	}
	// Attempt the gateway claw-back of the held transfer. The customer refund below
	// MUST proceed even if this fails, so we DON'T abort — but we record whether it
	// landed (reverseOK) so reverseRefundedDayHold can leave a failed claw-back as
	// re-drivable drift for the reconcile cron rather than stamping it settled and
	// stranding the chef's transfer (#398). An already-reversed transfer counts as
	// success (idempotent), not drift.
	reverseOK := true
	if day.PayoutTransferID != "" {
		if _, err := rz.ReverseTransfer(day.PayoutTransferID, 0); err != nil {
			if !isAlreadyReversedErr(err) {
				log.Printf("meal-plan refund: reverse transfer %s failed — leaving day hold as re-drivable drift for the reconcile cron: %v", day.PayoutTransferID, err)
				reverseOK = false
			}
			// already-reversed → idempotent no-op, no new money moved (no audit row)
		} else {
			auditTransferMovement(auditTransferReverse, aggTypeMealPlanDay, day.ID, day.PayoutTransferID, 0, "meal-plan day refunded — chef payout clawed back")
		}
	}
	// Refund the FULL amount the customer paid for the day (food + GST + delivery),
	// not just the food price — otherwise they'd lose the fees on an unserved day.
	txn, err := CreditWallet(tx, plan.CustomerID, perDayGross(plan, day), models.WalletSourceRefund, nil,
		fmt.Sprintf("Tiffin %s — %s", plan.MealPlanNumber, reason), dayRefundKey(day.ID), nil)
	if err != nil {
		return fmt.Errorf("refund day %s to wallet: %w", day.ID, err)
	}
	day.RefundTxnID = &txn.ID
	if err := tx.Model(&models.MealPlanDay{}).Where("id = ?", day.ID).
		Update("refund_txn_id", txn.ID).Error; err != nil {
		return err
	}
	// #498/#398: drive the day's payout hold out of the releasable set so the admin
	// queue can't release a refunded day (double-pay). STATE-ONLY — the claw-back was
	// attempted above; reverseOK tells it whether to stamp terminal-settled or leave
	// re-drivable drift for the reconcile cron on a failed reverse.
	return reverseRefundedDayHold(tx, day.ID, reverseOK)
}

// RefundDeclinedDays refunds the days the chef declined (cherry-picked out) once
// the customer approves the trim. No-op when escrow is off.
func RefundDeclinedDays(tx *gorm.DB, plan *models.MealPlan, reason string) error {
	if !MealPlanEscrowActive() {
		return nil
	}
	for i := range plan.Days {
		d := &plan.Days[i]
		if d.Status != models.MealPlanDayDeclined || d.RefundTxnID != nil {
			continue
		}
		if err := RefundDay(tx, plan, d, reason); err != nil {
			return err
		}
	}
	return nil
}

// RefundUndeliveredDays refunds every still-in-scope (not delivered, not already
// refunded) day — used on expiry / customer-reject for a full refund. No-op when
// escrow is off.
func RefundUndeliveredDays(tx *gorm.DB, plan *models.MealPlan, reason string) error {
	if !MealPlanEscrowActive() {
		return nil
	}
	for i := range plan.Days {
		d := &plan.Days[i]
		if d.Status == models.MealPlanDayDelivered || d.RefundTxnID != nil {
			continue
		}
		if err := RefundDay(tx, plan, d, reason); err != nil {
			return err
		}
	}
	return nil
}

// isPayableDayStatus reports whether a day is in a state that should hold a chef
// payout (accepted/confirmed — not declined/skipped/cancelled/refunded).
func isPayableDayStatus(s models.MealPlanDayStatus) bool {
	switch s {
	case models.MealPlanDayAccepted, models.MealPlanDayConfirmed,
		models.MealPlanDayPrepared, models.MealPlanDayDelivered:
		return true
	default:
		return false
	}
}
