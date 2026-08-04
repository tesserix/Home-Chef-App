package services

// chef_tip_backfill.go — pay chefs the tips that were charged before #964.
//
// Checkout wrote the customer's tip to the LEGACY `tip` column and never to
// `chef_tip`. Every payout surface — ComputeOrderEarnings' gross, the weekly
// statement, the FY statement, the statement PDF, the Form 16A TDS certificate,
// the payout queue, chef earnings — reads `chef_tip`. So the tip was charged,
// carried in order.total, and reached nobody.
//
// Fixing checkout stops the bleeding but does not settle what is already owed,
// and the reason is the frozen statement: WeeklyStatement totals are FROZEN at
// generation and each (chef, week) is generated exactly once. Simply setting
// chef_tip on a historical order changes no statement and therefore pays nobody.
//
// So this does two different things depending on whether the order has already
// been billed:
//
//   - ALWAYS set chef_tip = tip, so every re-derived surface (earnings, PDF, TDS
//     certificate) reports the chef's real income.
//   - ONLY where a statement already covers that order's delivered_at, raise a
//     ChefBonus catch-up — because that statement is frozen and will never pick
//     the tip up. An order not yet on a statement needs nothing: the next
//     statement reads the backfilled chef_tip naturally, and raising a bonus too
//     would pay the tip twice.
//
// Self-limiting: once checkout writes chef_tip, no new order matches, and this
// drains to a no-op. Idempotent on both halves — the chef_tip update is
// conditional on chef_tip = 0, and the bonus is unique on its source key.

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// chefTipCatchupKeyPrefix namespaces the catch-up ChefBonus source key.
const chefTipCatchupKeyPrefix = "tipcatchup:"

// ChefTipCatchupSourceKey is the natural key of one order's tip catch-up credit.
func ChefTipCatchupSourceKey(orderID uuid.UUID) string {
	return chefTipCatchupKeyPrefix + orderID.String()
}

// tipBackfillRow is an order whose tip never reached its chef.
type tipBackfillRow struct {
	ID          uuid.UUID
	OrderNumber string
	ChefID      uuid.UUID
	Tip         float64
	Status      string
	// AlreadyBilled — a weekly statement already covers this order's delivered_at,
	// so its totals are frozen and cannot absorb the tip.
	AlreadyBilled bool
}

// loadUnpaidTipOrders finds orders carrying a tip that never reached chef_tip, and
// reports for each whether a frozen statement already covers it.
func loadUnpaidTipOrders(db *gorm.DB, limit int) ([]tipBackfillRow, error) {
	var rows []tipBackfillRow
	err := db.Table("orders").
		Select(`orders.id, orders.order_number, orders.chef_id, orders.tip, orders.status,
			EXISTS (SELECT 1 FROM weekly_statements s
			        WHERE s.chef_id = orders.chef_id
			          AND orders.delivered_at >= s.week_start
			          AND orders.delivered_at < s.week_end) AS already_billed`).
		Where("orders.tip > 0 AND COALESCE(orders.chef_tip, 0) = 0").
		Where("orders.deleted_at IS NULL").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// BackfillChefTips settles tips charged before #964. Safe to run repeatedly.
func BackfillChefTips() {
	db := database.DB
	if db == nil {
		return
	}
	rows, err := loadUnpaidTipOrders(db, 500)
	if err != nil {
		log.Printf("chef-tip-backfill: query unpaid tips failed: %v", err)
		return
	}
	if len(rows) == 0 {
		return // drained — the steady state once checkout writes chef_tip
	}

	repaired, credited := 0, 0.0
	for i := range rows {
		r := rows[i]
		// Conditional so a concurrent run (or a re-drive) cannot double-apply.
		// chef_tip_at records that this write is the backfill's, not checkout's — the
		// distinction D-15 needed and could not make.
		res := db.Model(&models.Order{}).
			Where("id = ? AND COALESCE(chef_tip, 0) = 0", r.ID).
			Updates(map[string]any{"chef_tip": r.Tip, "chef_tip_at": time.Now().UTC()})
		if res.Error != nil {
			log.Printf("chef-tip-backfill: set chef_tip on order %s failed: %v", r.ID, res.Error)
			continue
		}
		if res.RowsAffected > 0 {
			repaired++
		}
		if !r.AlreadyBilled {
			continue // no statement has billed it yet — the next one includes the tip
		}
		if err := raiseChefTipCatchup(db, r); err != nil {
			log.Printf("chef-tip-backfill: catch-up credit for order %s failed (will retry): %v", r.ID, err)
			continue
		}
		credited += r.Tip
	}
	if repaired > 0 || credited > 0 {
		log.Printf("chef-tip-backfill: repaired %d order(s); credited ₹%.2f of tips billed on frozen statements",
			repaired, credited)
	}
}

// raiseChefTipCatchup credits one order's tip onto the chef's next statement.
func raiseChefTipCatchup(db *gorm.DB, r tipBackfillRow) error {
	var chef models.ChefProfile
	if err := db.Select("id", "user_id").First(&chef, "id = ?", r.ChefID).Error; err != nil {
		return fmt.Errorf("load chef %s: %w", r.ChefID, err)
	}
	if chef.UserID == uuid.Nil {
		return fmt.Errorf("chef %s has no user", r.ChefID)
	}
	reason := fmt.Sprintf("Customer tip on order %s (not included in the statement it was billed on)", r.OrderNumber)
	return raiseChefBonus(db, r.ChefID, chef.UserID, models.ChefBonusTipCatchup,
		ChefTipCatchupSourceKey(r.ID), r.Tip, nil, reason)
}
