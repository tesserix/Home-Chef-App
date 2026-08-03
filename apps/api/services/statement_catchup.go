package services

// statement_catchup.go — make the weekly statement and the escrow hold machine
// reconcilable, and settle what the statement had to skip (#927).
//
// Two independent paths can pay a chef for the same order — the hold state
// machine (#387/#388) releasing a Route transfer, and the weekly statement
// feeding a Cashfree payout batch — and nothing connected them. The statement
// builder selected on delivery alone, so an order whose payout was withheld,
// disputed or clawed back on the hold path was still a billable statement line.
//
// Fixing that by narrowing the statement predicate is only half safe on its own.
// loadStatementOrderRows is windowed on delivered_at and upsertWeeklyStatement
// generates each (chef, week) exactly once and then FREEZES it, so an order the
// statement skips is never billed by any later statement. Excluding a state the
// order can still LEAVE — awaiting_customer_confirmation, or disputed and later
// resolved in the chef's favour — would therefore silently cost the chef the
// whole order. That is why the shape suggested on the issue is not safe as
// written, and why this file exists.
//
// So the two halves are:
//
//   - orders.billed_statement_id records which statement settled an order,
//     turning "has this already paid the chef?" from an inference into a fact;
//   - reconcileStatementCatchup credits, via ChefBonus, any order that has since
//     become payable but whose own week's statement has already closed.
//
// The result is that an order is paid AT MOST ONCE (the statement skips anything
// already stamped) and AT LEAST ONCE once it is genuinely owed (the catch-up
// settles anything a closed week missed).

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// statementCatchupKeyPrefix namespaces the catch-up ChefBonus source key.
const statementCatchupKeyPrefix = "stmtcatchup:"

// StatementCatchupSourceKey is the natural key of one order's catch-up credit.
func StatementCatchupSourceKey(orderID uuid.UUID) string {
	return statementCatchupKeyPrefix + orderID.String()
}

// payableHoldStates are the hold states a chef is genuinely owed on: no hold was
// ever set (non-gateway order), or the hold has been confirmed/cleared. Kept in
// lockstep with the predicate in loadStatementOrderRows.
func payableHoldStates() []string {
	return []string{"", string(models.PayoutHoldReleaseEligible), string(models.PayoutHoldReleased)}
}

// stampBilledOrders records which orders a statement settled, inside the caller's
// transaction so the stamp and the statement commit together. Conditional on
// billed_statement_id IS NULL so a re-drive can never re-attribute an order that
// some other statement already settled.
func stampBilledOrders(tx *gorm.DB, statementID uuid.UUID, orderIDs []uuid.UUID) error {
	if len(orderIDs) == 0 {
		return nil
	}
	res := tx.Model(&models.Order{}).
		Where("id IN ? AND billed_statement_id IS NULL", orderIDs).
		Update("billed_statement_id", statementID)
	if res.Error != nil {
		return fmt.Errorf("statement-catchup: stamp billed orders for %s: %w", statementID, res.Error)
	}
	return nil
}

// BackfillBilledStatementIDs stamps orders covered by statements that were
// generated BEFORE stamping existed.
//
// Without this every historical order looks unbilled, and the catch-up would
// credit orders the chef has already been billed for — paying them twice. It
// reproduces the predicate those statements were generated under (delivered in
// the window, not gateway-split, not soft-deleted) rather than today's narrower
// one, because that is what those statements actually billed.
//
// Idempotent: only ever fills a NULL stamp. Drains to a no-op once every
// pre-existing statement is covered.
func BackfillBilledStatementIDs(db *gorm.DB) (int64, error) {
	var stmts []models.WeeklyStatement
	if err := db.Find(&stmts).Error; err != nil {
		return 0, fmt.Errorf("statement-catchup: list statements: %w", err)
	}
	var stamped int64
	for i := range stmts {
		s := stmts[i]
		// Only ever touch a statement generated BEFORE stamping existed. Since
		// upsertWeeklyStatement now creates the statement and stamps its orders in one
		// transaction, and a statement is only created when it bills at least one
		// order, "zero stamped orders" identifies exactly the pre-change statements.
		//
		// The scoping matters: this reproduces the OLD, broader predicate, so running
		// it against a new statement would stamp orders that statement deliberately
		// SKIPPED (still awaiting confirmation, or disputed) — marking them settled
		// when nobody has paid for them, and permanently suppressing their catch-up.
		var already int64
		if err := db.Model(&models.Order{}).
			Where("billed_statement_id = ?", s.ID).Count(&already).Error; err != nil {
			return stamped, fmt.Errorf("statement-catchup: count stamps for statement %s: %w", s.ID, err)
		}
		if already > 0 {
			continue
		}
		res := db.Model(&models.Order{}).
			Where("chef_id = ? AND billed_statement_id IS NULL", s.ChefID).
			Where("status = ?", models.OrderStatusDelivered).
			Where("delivered_at >= ? AND delivered_at < ?", s.WeekStart, s.WeekEnd).
			Where("COALESCE(gateway_split_paise, 0) = 0").
			Update("billed_statement_id", s.ID)
		if res.Error != nil {
			return stamped, fmt.Errorf("statement-catchup: backfill stamp for statement %s: %w", s.ID, res.Error)
		}
		stamped += res.RowsAffected
	}
	return stamped, nil
}

// catchupRow is an order that became payable after its own week had closed.
type catchupRow struct {
	ID          uuid.UUID
	OrderNumber string
	ChefID      uuid.UUID
	StatementID uuid.UUID
	Subtotal    float64
	Tax         float64
	ChefTip     float64
	// ChefFundedDiscount reduces the chef's revenue before commission (#39).
	ChefFundedDiscount float64
	CommissionRate     float64
	ChefState          string
	DeliveryState      string
}

// loadCatchupOrders finds delivered, payable, unsettled orders whose week already
// has a statement — i.e. orders the statement had to skip and will never revisit.
//
// The covering-statement join is the whole point: an order with NO statement for
// its week yet needs nothing, because the next statement generated for that week
// will bill it normally. Crediting it here as well would pay it twice.
func loadCatchupOrders(db *gorm.DB, limit int) ([]catchupRow, error) {
	var rows []catchupRow
	err := db.Table("orders o").
		Select(`o.id, o.order_number, o.chef_id, s.id AS statement_id,
			o.subtotal, o.tax, o.chef_tip, o.chef_funded_discount, o.commission_rate,
			c.state AS chef_state, o.delivery_address_state AS delivery_state`).
		Joins("JOIN chef_profiles c ON c.id = o.chef_id").
		Joins(`JOIN weekly_statements s ON s.chef_id = o.chef_id
		       AND o.delivered_at >= s.week_start AND o.delivered_at < s.week_end`).
		Where("o.status = ?", models.OrderStatusDelivered).
		Where("o.deleted_at IS NULL").
		Where("o.billed_statement_id IS NULL").
		Where("o.refunded_at IS NULL").
		Where("COALESCE(o.gateway_split_paise, 0) = 0").
		Where("COALESCE(o.payout_hold_status, '') IN ?", payableHoldStates()).
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// reconcileStatementCatchup settles orders whose statement week closed while they
// were still held. Safe to run repeatedly.
func reconcileStatementCatchup() {
	db := database.DB
	if db == nil {
		return
	}
	// Historical statements must be stamped BEFORE any catch-up runs, or every
	// order they already billed reads as unsettled and gets credited a second time.
	if n, err := BackfillBilledStatementIDs(db); err != nil {
		log.Printf("statement-catchup: backfill of billed stamps failed — skipping catch-up this run: %v", err)
		return
	} else if n > 0 {
		log.Printf("statement-catchup: stamped %d order(s) already billed by an earlier statement", n)
	}

	rows, err := loadCatchupOrders(db, 500)
	if err != nil {
		log.Printf("statement-catchup: query skipped orders failed: %v", err)
		return
	}
	credited := 0.0
	for i := range rows {
		r := rows[i]
		net := ComputeOrderEarnings(EarningsInput{
			ItemRevenue:        r.Subtotal,
			Tax:                r.Tax,
			ChefTip:            r.ChefTip,
			ChefFundedDiscount: r.ChefFundedDiscount,
			CommissionRate:     r.CommissionRate,
			DeliveryState:      r.DeliveryState,
		}, r.ChefState).NetPayout
		if net <= 0 {
			continue
		}
		if err := raiseStatementCatchup(db, r, net); err != nil {
			log.Printf("statement-catchup: credit for order %s failed (will retry): %v", r.ID, err)
			continue
		}
		credited += net
	}
	if credited > 0 {
		log.Printf("statement-catchup: credited ₹%.2f for order(s) held past their statement week", credited)
	}
}

// raiseStatementCatchup credits one order's net payout onto the chef's next
// statement and stamps the order settled, so it leaves the candidate set.
//
// The ChefBonus source key is unique, so the credit itself is idempotent even if
// the stamp write fails and the order is retried.
func raiseStatementCatchup(db *gorm.DB, r catchupRow, net float64) error {
	var chef models.ChefProfile
	if err := db.Select("id", "user_id").First(&chef, "id = ?", r.ChefID).Error; err != nil {
		return fmt.Errorf("load chef %s: %w", r.ChefID, err)
	}
	if chef.UserID == uuid.Nil {
		return fmt.Errorf("chef %s has no user", r.ChefID)
	}
	reason := fmt.Sprintf("Order %s — payout was still held when its statement closed", r.OrderNumber)
	if err := raiseChefBonus(db, r.ChefID, chef.UserID, models.ChefBonusStatementCatchup,
		StatementCatchupSourceKey(r.ID), net, nil, reason); err != nil {
		return err
	}
	// Attribute the settlement to the statement whose week it belonged to. The
	// money rides a LATER statement as a bonus line, but this records that the
	// order's own period is now settled — and takes it out of the candidate set.
	return db.Model(&models.Order{}).
		Where("id = ? AND billed_statement_id IS NULL", r.ID).
		Update("billed_statement_id", r.StatementID).Error
}

// statementCatchupInterval is deliberately slower than the money-critical sweeps:
// nothing here is time-sensitive (the credit rides the next weekly statement
// either way) and it walks every statement on each run.
const statementCatchupInterval = time.Hour

// runStatementCatchupScan is the cron entry point. Panic-guarded so one bad row
// never stops the sweep.
func runStatementCatchupScan(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("statement-catchup: panic recovered: %v", r)
		}
	}()
	reconcileStatementCatchup()
}

// StartStatementCatchupCron is the in-process ticker fallback (Temporal disabled).
func StartStatementCatchupCron(ctx context.Context) {
	go func() {
		runStatementCatchupScan(ctx)
		t := time.NewTicker(statementCatchupInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runStatementCatchupScan(ctx)
			}
		}
	}()
	log.Println("statement-catchup: cron started (interval=1h)")
}
