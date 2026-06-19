package services

import (
	"context"
	"log"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// group_order_cron.go — expires abandoned OPEN group/office carts past their TTL
// (#46). Only open carts are swept (no money has moved yet — payment happens
// after lock). A locked-but-unpaid group is left for the host to cancel, which
// refunds any partial payers. Runs as a durable Temporal Schedule with the
// in-process ticker as the legacy fallback.
const groupOrderSweepInterval = 30 * time.Minute

// StartGroupOrderCron is the legacy in-process fallback (Temporal off).
func StartGroupOrderCron(ctx context.Context) {
	go func() {
		runGroupOrderSweep(ctx)
		t := time.NewTicker(groupOrderSweepInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("group-order-sweep: shutting down on ctx cancel")
				return
			case <-t.C:
				runGroupOrderSweep(ctx)
			}
		}
	}()
	log.Println("group-order-sweep: cron started (interval=30m)")
}

func runGroupOrderSweep(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("group-order-sweep: panic recovered: %v", r)
		}
	}()
	res := database.DB.Model(&models.GroupOrder{}).
		Where("status = ? AND expires_at <= ?", models.GroupOrderOpen, time.Now()).
		Update("status", models.GroupOrderExpired)
	if res.Error != nil {
		log.Printf("group-order-sweep: expire query failed: %v", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		log.Printf("group-order-sweep: expired %d abandoned open carts", res.RowsAffected)
	}
}
