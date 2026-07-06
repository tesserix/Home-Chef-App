package services

// delivery_retry.go — #579 (owner's 2-attempt cap). A failed courier delivery gets ONE
// retry before terminal money resolution. Two seams:
//
//   - RetryOrTerminalizeFailedDelivery: the failure decision. Under the cap → re-dispatch
//     (reset the SAME Delivery row, re-open the order); at the cap → freeze the money for
//     admin fault resolution (the slice-1 always-terminalize behaviour).
//   - AssignDeliveryForOrder: the assignment upsert. A retry (or a cancellation) leaves a
//     Delivery row in place, and Delivery.OrderID is a hard uniqueIndex with NO
//     soft-delete — so a re-dispatch MUST reuse that row, never insert a second (which
//     500s forever). Used by both AcceptDelivery and ManualAssignDelivery. This also
//     fixes the pre-existing latent clash on the cancelled → re-accept path.

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// MaxDeliveryAttempts is the owner's cap: a delivery may be attempted at most twice (one
// retry after the first failure), then a further failure terminalizes into admin money
// resolution. The decision uses this constant, not the per-row MaxAttempts column (whose
// model default is a stale 3), so the cap is authoritative regardless of row data.
const MaxDeliveryAttempts = 2

// ErrDeliveryStateChanged signals the delivery row moved out of its expected in-flight
// state between load and the guarded retry UPDATE (a concurrent status change) — the
// caller should not treat it as a successful retry.
var ErrDeliveryStateChanged = errors.New("delivery state changed before retry could be applied")

// RetryOrTerminalizeFailedDelivery decides a failed courier delivery's fate under the
// attempt cap and applies it atomically. `delivery` is the just-failed row (its
// AttemptNumber is the attempt that failed). Returns retried=true when it re-dispatched
// (no money moved), false when it terminalized (money frozen for admin resolution). On the
// retry path the delivery row and its order are reset here — the caller must NOT then
// persist the stale (failed) struct. The terminal path loads the order fresh (not from a
// possibly-stale delivery.Order) so the gateway freeze sees the real razorpay_order_id.
func RetryOrTerminalizeFailedDelivery(db *gorm.DB, delivery *models.Delivery, reason models.DeliveryFailureReason, reportedBy string) (bool, error) {
	// A retry is allowed only while an attempt remains (the failed attempt was < cap).
	if delivery.AttemptNumber < MaxDeliveryAttempts {
		err := db.Transaction(func(tx *gorm.DB) error {
			// Re-dispatch the SAME row (order_id unique index forbids a second): clear the
			// partner + the just-failed attempt's residue, bump the attempt, back to pending
			// so the assignment machinery can re-offer it. Guard EXCLUDES the terminal states
			// (not the caller's already-mutated in-memory Status — the failure isn't persisted
			// yet; the DB row still holds the pre-failure in-flight status): a concurrent
			// delivered/cancelled/terminal transition must not be overwritten by a retry.
			res := tx.Model(&models.Delivery{}).
				Where("id = ? AND status NOT IN ?", delivery.ID, []models.DeliveryStatus{
					models.DeliveryDelivered, models.DeliveryCancelled,
					models.DeliveryFailed, models.DeliveryReturned,
				}).
				Updates(map[string]any{
					"status":              models.DeliveryPending,
					"delivery_partner_id": nil,
					"attempt_number":      delivery.AttemptNumber + 1,
					"failure_reason":      "",
					"picked_up_at":        nil,
				})
			if res.Error != nil {
				return fmt.Errorf("delivery-retry: reset delivery %s: %w", delivery.ID, res.Error)
			}
			if res.RowsAffected == 0 {
				return ErrDeliveryStateChanged
			}
			// Re-open the order for re-dispatch. Targeted update — never clobber the
			// payout-hold columns (#460 race 1). No money moves on a retry.
			if err := tx.Model(&models.Order{}).Where("id = ?", delivery.OrderID).
				Updates(map[string]any{"status": models.OrderStatusReady, "delivery_id": nil}).Error; err != nil {
				return fmt.Errorf("delivery-retry: re-open order %s: %w", delivery.OrderID, err)
			}
			return nil
		})
		if err != nil {
			return false, err
		}
		// Reflect the applied reset on the passed struct for the caller/response.
		delivery.Status = models.DeliveryPending
		delivery.DeliveryPartnerID = nil
		delivery.AttemptNumber++
		delivery.FailureReason = ""
		return true, nil
	}
	// Cap reached → terminal money freeze (open the delivery_failed issue + dispute the
	// hold; no money moves until an admin confirms fault). Mirrors the courier slice-1 path.
	var order models.Order
	if err := db.First(&order, "id = ?", delivery.OrderID).Error; err != nil {
		return false, fmt.Errorf("delivery-retry: load order %s for terminalize: %w", delivery.OrderID, err)
	}
	_, err := TerminalizeDeliveryFailure(db, &order, reason, reportedBy,
		map[string]any{"delivery_id": delivery.ID.String()})
	return false, err
}

// AssignDeliveryForOrder persists a delivery assignment, REUSING any existing Delivery row
// for the order rather than inserting a second — Delivery.OrderID is a hard uniqueIndex
// with no soft-delete, so a re-dispatch (retry-reset or post-cancel row) MUST reuse. On
// reuse it overwrites the assignment columns, PRESERVES the attempt counter + created_at,
// and CLEARS the prior attempt's terminal residue (failure/cancel fields, stale
// timestamps). On a first assignment it inserts, stamping the effective cap. `want` is
// mutated to reflect the persisted row (ID, AttemptNumber, MaxAttempts). Runs in the
// caller's tx.
func AssignDeliveryForOrder(tx *gorm.DB, want *models.Delivery) error {
	var existing models.Delivery
	err := tx.Select("id", "attempt_number", "max_attempts").
		Where("order_id = ?", want.OrderID).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// First assignment for this order — fresh row.
		if want.ID == uuid.Nil {
			want.ID = uuid.New() // app-generate so the caller has the id on any backend
		}
		if want.MaxAttempts == 0 {
			want.MaxAttempts = MaxDeliveryAttempts
		}
		if want.AttemptNumber == 0 {
			want.AttemptNumber = 1
		}
		return tx.Create(want).Error
	case err != nil:
		return fmt.Errorf("delivery-assign: look up existing delivery for order %s: %w", want.OrderID, err)
	}
	// Re-dispatch onto the existing row. Preserve the attempt counter (the retry path
	// already bumped it; a cancellation left it unchanged).
	now := time.Now()
	want.ID = existing.ID
	want.AttemptNumber = existing.AttemptNumber
	want.MaxAttempts = existing.MaxAttempts
	want.AssignedAt = now // re-dispatch = a new assignment time (Delivery has no separate created_at)
	updates := map[string]any{
		"delivery_partner_id":   want.DeliveryPartnerID,
		"status":                want.Status,
		"assignment_type":       want.AssignmentType,
		"assigned_by_id":        want.AssignedByID,
		"pickup_address_line1":  want.PickupAddressLine1,
		"pickup_address_city":   want.PickupAddressCity,
		"pickup_latitude":       want.PickupLatitude,
		"pickup_longitude":      want.PickupLongitude,
		"dropoff_address_line1": want.DropoffAddressLine1,
		"dropoff_address_city":  want.DropoffAddressCity,
		"dropoff_latitude":      want.DropoffLatitude,
		"dropoff_longitude":     want.DropoffLongitude,
		"distance":              want.Distance,
		"estimated_duration":    want.EstimatedDuration,
		"delivery_fee":          want.DeliveryFee,
		"tip":                   want.Tip,
		"total_payout":          want.TotalPayout,
		"assigned_at":           now,
		// Clear the prior attempt's terminal residue so the reused row is clean.
		"failure_reason":   "",
		"cancel_reason":    "",
		"cancelled_at":     nil,
		"picked_up_at":     nil,
		"delivered_at":     nil,
		"actual_duration":  0,
		"offer_expires_at": nil,
		// Clear any 3PL/provider residue: a reuse here is always an own-fleet
		// (AcceptDelivery / ManualAssignDelivery) re-dispatch, so stale provider/rider
		// fields from a prior third-party attempt must not linger (a replayed provider
		// webhook keys off external_delivery_id; ToResponse shows stale rider coords).
		"provider_id":           nil,
		"external_delivery_id":  "",
		"external_tracking_id":  "",
		"external_tracking_url": "",
		"provider_cost":         0,
		"provider_status":       "",
		"rider_name":            "",
		"rider_phone":           "",
		"rider_latitude":        0,
		"rider_longitude":       0,
	}
	// Guard the reuse to RE-DISPATCHABLE states only. A row that is active
	// (assigned/picked_up/…) or delivered must never be stolen — this closes the
	// two-concurrent-accept race the old tx.Create relied on the unique index to catch
	// (an UPDATE is not protected by that index). RowsAffected==0 → the row is live/done.
	res := tx.Model(&models.Delivery{}).
		Where("id = ? AND status IN ?", existing.ID, []models.DeliveryStatus{
			models.DeliveryPending, models.DeliveryCancelled,
			models.DeliveryFailed, models.DeliveryReturned,
		}).
		Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("delivery-assign: reuse delivery %s for order %s: %w", existing.ID, want.OrderID, res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("delivery-assign: order %s already has an active delivery: %w", want.OrderID, ErrDeliveryStateChanged)
	}
	return nil
}
