package services

// confirm_receipt.go — the auto-confirm-delivery follow-up (see docs plan
// 2026-07-24-auto-confirm-delivery.md, Task 1). When a customer never taps
// "confirm receipt" after delivery, a durable reminder flow eventually
// confirms on their behalf. AutoConfirmOrderReceipt is the terminal action of
// that flow: it re-reads the order and, if it is still genuinely awaiting
// confirmation, delegates to the SAME ConfirmOrderHold transition the
// customer's own confirm button uses (→ release_eligible, or → disputed if an
// issue is open). It moves no money — the existing payout-release sweep does
// that on its own schedule.

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// AutoConfirmOrderReceipt confirms receipt on the customer's behalf once the
// reminder window has elapsed with no customer action. It re-reads the order
// fresh (the caller may be an async activity with a stale/no view of state)
// and only acts when the order is still genuinely awaiting confirmation: not
// already confirmed, not refunded, and not in a terminal cancelled/refunded
// status. Any of those conditions makes this a safe no-op.
//
// Returns (status, acted, err). acted=false with a nil error means "nothing
// to do" (already confirmed / not awaiting / terminal / refunded).
func AutoConfirmOrderReceipt(db *gorm.DB, orderID uuid.UUID) (models.PayoutHoldStatus, bool, error) {
	var order models.Order
	if err := db.First(&order, "id = ?", orderID).Error; err != nil {
		return "", false, err
	}
	if order.CustomerConfirmedAt != nil ||
		order.PayoutHoldStatus != models.PayoutHoldAwaitingConfirmation ||
		order.RefundedAt != nil ||
		order.Status == models.OrderStatusCancelled ||
		order.Status == models.OrderStatusRefunded {
		return order.PayoutHoldStatus, false, nil
	}
	status, err := ConfirmOrderHold(db, &order)
	if err != nil {
		return "", false, err
	}
	return status, true, nil
}

// confirmReminderSetting reads one `payout.*` PlatformSettings key, falling
// back to def when the key is absent or unparsable. Mirrors the fold pattern
// in GetCustomerConfirmWindowHours (payout_hold.go).
func confirmReminderSetting(db *gorm.DB, key string, def int) int {
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "payout.%").Find(&settings)
	for _, s := range settings {
		if s.Key == key {
			if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
				return v
			}
		}
	}
	return def
}

// ConfirmReminderIntervalMinutes is the gap between confirm-receipt
// reminders, read from PlatformSettings key
// `payout.confirm_reminder_interval_minutes` (default 10).
func ConfirmReminderIntervalMinutes(db *gorm.DB) int {
	return confirmReminderSetting(db, "payout.confirm_reminder_interval_minutes", 10)
}

// ConfirmReminderMaxCount is how many reminders fire before auto-confirm,
// read from PlatformSettings key `payout.confirm_reminder_max_count`
// (default 3).
func ConfirmReminderMaxCount(db *gorm.DB) int {
	return confirmReminderSetting(db, "payout.confirm_reminder_max_count", 3)
}

// SendConfirmReceiptReminder nudges the customer to confirm they received
// their order. It re-reads the order and skips (sent=false, nil error) once
// it is no longer awaiting confirmation — already confirmed, disputed, or
// otherwise moved on. Otherwise it fires a best-effort push (a missing/failed
// send is not a failure of the reminder itself) and stages a transactional
// outbox event. Returns (sent, err); sent=true means the reminder was
// attempted (outbox staged), regardless of whether the push actually reached
// a device.
func SendConfirmReceiptReminder(db *gorm.DB, orderID uuid.UUID, attempt int) (bool, error) {
	var order models.Order
	if err := db.First(&order, "id = ?", orderID).Error; err != nil {
		return false, err
	}
	if order.CustomerConfirmedAt != nil ||
		order.PayoutHoldStatus != models.PayoutHoldAwaitingConfirmation {
		return false, nil
	}

	chefName := "your chef"
	var chef models.ChefProfile
	if err := db.First(&chef, "id = ?", order.ChefID).Error; err == nil && chef.BusinessName != "" {
		chefName = chef.BusinessName
	}

	title := "Did your order arrive?"
	body := fmt.Sprintf("Tap to confirm you received your order from %s.", chefName)
	data := map[string]string{
		"order_id": order.ID.String(),
		"type":     "confirm_receipt",
	}
	// Best-effort: a missing/invalid token or send failure is not a failure
	// of the reminder itself.
	_ = SendPushNotification(order.CustomerID, title, body, data)

	if err := EnqueueEvent(db, "orders.confirm_reminder", "order.confirm_reminder", order.CustomerID, map[string]any{
		"order_id": order.ID.String(),
		"attempt":  attempt,
	}); err != nil {
		return true, err
	}
	return true, nil
}
