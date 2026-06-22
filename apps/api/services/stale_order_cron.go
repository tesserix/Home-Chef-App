package services

// stale_order_cron.go — expire abandoned unpaid orders.
//
// An order row is created BEFORE payment, reserving the chef's daily dish
// capacity (#48). If the customer never completes payment (dismisses the
// Razorpay sheet, app killed, etc.) the order lingers as payment_status=pending
// forever — it clutters the customer's active-order view and holds capacity the
// chef can't sell. This cron cancels unpaid orders older than the grace window
// and releases their reserved capacity + slot. Fire-on-start also catches up
// anything missed during a deploy.

import (
	"context"
	"log"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

const (
	staleOrderInterval = 10 * time.Minute
	// Payment normally completes within seconds; 30 min comfortably covers the
	// checkout sheet, retries, and the webhook fallback before we give up.
	staleOrderThreshold = 30 * time.Minute
	// Bound the batch so a backlog can't hold a transaction open too long.
	staleOrderBatch = 100
)

// StartStaleOrderCron launches the abandoned-order sweep. Returns immediately;
// lives for the life of ctx.
func StartStaleOrderCron(ctx context.Context) {
	go func() {
		runStaleOrderScan(ctx)
		ticker := time.NewTicker(staleOrderInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("stale-order: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runStaleOrderScan(ctx)
			}
		}
	}()
	log.Println("stale-order: cron started (interval=10m, grace=30m)")
}

func runStaleOrderScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("stale-order: panic recovered: %v", r)
		}
	}()

	cutoff := time.Now().Add(-staleOrderThreshold)

	var stale []models.Order
	if err := database.DB.
		Preload("Items").
		Where("payment_status = ? AND status NOT IN ? AND created_at < ?",
			models.PaymentPending,
			[]models.OrderStatus{
				models.OrderStatusCancelled,
				models.OrderStatusRejected,
				models.OrderStatusRefunded,
				models.OrderStatusDelivered,
			},
			cutoff).
		Order("created_at ASC").
		Limit(staleOrderBatch).
		Find(&stale).Error; err != nil {
		log.Printf("stale-order: query failed: %v", err)
		return
	}
	if len(stale) == 0 {
		return
	}

	expired := 0
	for i := range stale {
		select {
		case <-ctx.Done():
			return
		default:
		}
		order := stale[i]
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// Mark cancelled (no payment was captured, so nothing to refund).
			now := time.Now()
			if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
				"status":         models.OrderStatusCancelled,
				"payment_status": models.PaymentFailed,
				"cancel_reason":  "payment not completed",
				"cancelled_at":   now,
			}).Error; err != nil {
				return err
			}
			// Release the reserved daily capacity — these dishes won't be made.
			capDay := CapacityDay(order.CreatedAt)
			for _, it := range order.Items {
				if err := ReleaseCapacity(tx, it.MenuItemID, it.Quantity, capDay); err != nil {
					return err
				}
			}
			// Release the scheduled delivery-slot booking too (#51), if any.
			if order.DeliverySlot != "" && order.ScheduledFor != nil {
				if err := ReleaseSlot(tx, order.ChefID, order.DeliverySlot, 1, CapacityDay(*order.ScheduledFor)); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Printf("stale-order: failed to expire order %s: %v", order.ID, err)
			continue
		}
		expired++
	}
	log.Printf("stale-order: expired %d abandoned unpaid order(s)", expired)
}
