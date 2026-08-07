package services

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// easy_split_release.go — the split happens when the governor releases the
// order, not when the customer pays (ADR-0003, #1091).
//
// Splitting at capture allocated the chef's share before the maturation window,
// BlockRefundOpen, BlockRecoveryBalance, BlockNewChefRamp or
// BlockAboveReviewThreshold had said anything — a split order had no payout to
// release, so none of them ran. Cashfree's split-delay window lets the split be
// created after payment instead, which puts the same decision back in front of
// the money.
//
// Every path here fails toward the payout rail: not splitting is Cashfree's own
// documented fallback (the whole amount settles to the platform), which is
// exactly where the order was before Easy Split existed.

const (
	// SettingEasySplitDelayHours is Cashfree's account-level order split delay.
	// Their default is T+1; extending it is an account-manager conversation, so
	// this setting exists to be told what was agreed, never to request it.
	SettingEasySplitDelayHours = "easy_split_delay_hours"

	defaultEasySplitDelayHours = 24

	// easySplitSyncDelay is how long Cashfree needs after a payment before it
	// can be split at all. Documented as 2 minutes; the maturation window is
	// hours, so this only ever bites on a misconfiguration.
	easySplitSyncDelay = 2 * time.Minute
)

// EasySplitDelay reads the agreed split window, falling back on Cashfree's own
// default. A shorter window is safe (we split earlier); an over-long one would
// have us call an API that has already closed, so anything unparseable falls
// back rather than being trusted.
func EasySplitDelay(db *gorm.DB) time.Duration {
	raw := strings.TrimSpace(settingValue(db, SettingEasySplitDelayHours))
	if raw == "" {
		return defaultEasySplitDelayHours * time.Hour
	}
	hours, err := strconv.Atoi(raw)
	if err != nil || hours <= 0 {
		log.Printf("easy-split: %s=%q is unusable — falling back to Cashfree's %dh default",
			SettingEasySplitDelayHours, raw, defaultEasySplitDelayHours)
		return defaultEasySplitDelayHours * time.Hour
	}
	return time.Duration(hours) * time.Hour
}

// EasySplitWindowFits reports whether a released order can still be split.
//
// ADR-0003 (Consequences): the maturation window has to fit inside the split
// delay. If it does not, every order matures after Cashfree has already settled
// the capture, and the split rail degrades to the payout rail silently — for
// every chef, indefinitely. Asserted here so the two coupled values cannot
// drift apart unnoticed.
func EasySplitWindowFits(db *gorm.DB) bool {
	maturation, delay := maturationWindow(db), EasySplitDelay(db)
	if maturation+easySplitSyncDelay < delay {
		return true
	}
	log.Printf("easy-split: maturation window %s does not fit inside the %s split delay — orders will settle through the payout rail",
		maturation, delay)
	return false
}

// ReleaseOrderSplit pays the chef's share straight from the capture, and
// reports whether it did. False means "not this rail" — the caller must fall
// through to the payout rail, which is where every order was before Easy Split.
//
// An error means the outcome is unknown or merely early: nothing is stamped,
// the release does not complete, and the reconcile re-drives it. Re-driving is
// safe because the split carries the order's own id as its idempotency key and
// Cashfree reports a repeat as already-processed.
func ReleaseOrderSplit(db *gorm.DB, orderID uuid.UUID, now time.Time) (bool, error) {
	if db == nil {
		return false, nil
	}

	var order models.Order
	if err := db.Preload("Chef").First(&order, "id = ?", orderID).Error; err != nil {
		// Not knowing whether to split is not a reason to block the release: the
		// order settles through the payout rail, same as every other refusal here.
		log.Printf("easy-split: could not read order %s (%v) — falling back to the payout rail", orderID, err)
		return false, nil
	}
	// Already split — by an earlier release, or by a re-drive of this one. The
	// chef has been paid; saying otherwise would pay them again on the other rail.
	if order.GatewaySplitPaise > 0 {
		return true, nil
	}
	if order.PaymentProvider != models.PaymentProviderCashfree || order.RazorpayOrderID == "" {
		return false, nil
	}
	if !EasySplitWindowFits(db) {
		return false, nil
	}
	// Age is measured from order creation, which is never later than payment —
	// so a window this reads as open really is open.
	age := now.Sub(order.CreatedAt)
	if age < easySplitSyncDelay || age >= EasySplitDelay(db) {
		return false, nil
	}

	creditPaise := ToPaise(order.WalletApplied) + ToPaise(order.LoyaltyApplied)
	capturePaise := ToPaise(order.Total) - creditPaise
	split, skipped := BuildOrderSplitWithReason(db, &order, capturePaise, creditPaise)
	if split == nil {
		// Recorded, not just logged: "why did this order not split?" is asked
		// weeks later, by which time the chef's state has moved on (#1084).
		LogSystemAudit(nil, "order.payout.easy_split_skipped", "order", orderID.String(), nil, map[string]any{
			"reason": skipped, "orderNumber": order.OrderNumber,
		})
		return false, nil
	}
	cf := GetCashfreeFor(order.Mode)
	if cf == nil {
		return false, nil
	}

	if err := cf.SplitOrderAfterPayment(order.RazorpayOrderID,
		[]CashfreeVendorSplit{*split}, orderID.String()); err != nil {
		if errors.Is(err, ErrEasySplitRetryable) {
			return false, err
		}
		// A refusal Cashfree will repeat — an unregistered vendor, an amount it
		// will not accept. Nothing moved, so let the window lapse and pay the
		// chef through the payout rail rather than stranding them.
		log.Printf("easy-split: order %s not split (%v) — falling back to the payout rail", order.OrderNumber, err)
		return false, nil
	}

	// Conditional so a concurrent re-drive cannot double-count the share. The
	// column is what keeps the weekly statement off this order — the same
	// exclusion split-at-capture relied on.
	res := db.Model(&models.Order{}).
		Where("id = ? AND COALESCE(gateway_split_paise, 0) = 0", orderID).
		Update("gateway_split_paise", split.AmountPaise.Paise())
	if res.Error != nil {
		// The money has moved and we cannot record it. Returning the error keeps
		// the order unsettled so the reconcile re-drives — the split call itself
		// is idempotent, and the stamp is what stops the double payment.
		return false, res.Error
	}
	LogSystemAudit(nil, "order.payout.easy_split", "order", orderID.String(), nil, map[string]any{
		"vendorId":    split.VendorID,
		"amountPaise": split.AmountPaise.Paise(),
		"orderNumber": order.OrderNumber,
	})
	return true, nil
}
