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

// statementAdjustKeyPrefix namespaces the under-billed adjustment source key.
const statementAdjustKeyPrefix = "stmtadjust:"

// StatementAdjustSourceKey is the natural key of one under-billed settlement
// adjustment. The settled-to figure is part of the key, so re-running against an
// unchanged order dedups to the credit already raised, while a genuine further
// increase mints a new one.
func StatementAdjustSourceKey(orderID uuid.UUID, settledToPaise int) string {
	return fmt.Sprintf("%s%s:%d", statementAdjustKeyPrefix, orderID, settledToPaise)
}

// underbilledEpsilon is the smallest gap worth a settlement line. Below half a
// rupee the difference is per-row rounding, not money owed.
const underbilledEpsilon = 0.50

// payableHoldStates are the hold states a chef is genuinely owed on: no hold was
// ever set (non-gateway order), or the hold has been confirmed/cleared. Kept in
// lockstep with the predicate in loadStatementOrderRows.
func payableHoldStates() []string {
	return []string{"", string(models.PayoutHoldReleaseEligible), string(models.PayoutHoldReleased)}
}

// billedOrder is one order a statement bills, with the net payout it was billed
// for. The amount is recorded alongside the stamp so a later increase in the
// order's value is detectable (D-15) rather than silently unpayable.
type billedOrder struct {
	ID  uuid.UUID
	Net float64
}

// stampBilledOrders records which orders a statement settled and for how much,
// inside the caller's transaction so the stamps and the statement commit together.
// Conditional on billed_statement_id IS NULL so a re-drive can never re-attribute
// an order that some other statement already settled.
func stampBilledOrders(tx *gorm.DB, statementID uuid.UUID, orders []billedOrder) error {
	for _, o := range orders {
		res := tx.Model(&models.Order{}).
			Where("id = ? AND billed_statement_id IS NULL", o.ID).
			Updates(map[string]any{
				"billed_statement_id": statementID,
				"settled_net_payout":  Round2(o.Net),
			})
		if res.Error != nil {
			return fmt.Errorf("statement-catchup: stamp billed order %s for %s: %w", o.ID, statementID, res.Error)
		}
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
	Currency     string `gorm:"column:currency"`
	TaxInclusive bool   `gorm:"column:tax_inclusive"`
	ID           uuid.UUID
	OrderNumber  string
	ChefID       uuid.UUID
	StatementID  uuid.UUID
	Subtotal     float64
	Tax          float64
	TaxFood      float64
	TaxService   float64
	ChefTip      float64
	// ChefFundedDiscount reduces the chef's revenue before commission (#39).
	ChefFundedDiscount float64
	CommissionRate     float64
	ChefState          string
	DeliveryState      string
	// SettledNetPayout is how much has already been credited for this order.
	// Selected only by loadUnderbilledOrders; the catch-up path deals in orders
	// nothing has been credited for yet, and leaves it zero.
	SettledNetPayout float64
	// FulfillmentType decides who the delivery fee belongs to; DeliveryFeeFinal
	// is the #703 lowered-at-accept figure the customer was actually charged.
	FulfillmentType  string
	DeliveryFee      float64
	DeliveryFeeFinal *float64
}

// earningsInput maps a catch-up row to the settlement engine, on the same basis
// as the statement that skipped or under-billed it.
func (r catchupRow) earningsInput() EarningsInput {
	fee := r.DeliveryFee
	if r.DeliveryFeeFinal != nil {
		fee = *r.DeliveryFeeFinal
	}
	return EarningsInput{
		Currency:             r.Currency,
		TaxInclusive:         r.TaxInclusive,
		ItemRevenue:          r.Subtotal,
		Tax:                  ChefTaxOf(r.Tax, r.TaxFood, r.TaxService),
		ChefTip:              r.ChefTip,
		ChefFundedDiscount:   r.ChefFundedDiscount,
		CommissionRate:       r.CommissionRate,
		DeliveryState:        r.DeliveryState,
		DeliveryFee:          fee,
		ChefEarnsDeliveryFee: SettledChefEarnsDeliveryFee(r.FulfillmentType),
	}
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
			o.subtotal, o.tax, o.currency, o.tax_inclusive, o.tax_food, o.tax_service, o.chef_tip, o.driver_tip, o.chef_funded_discount, o.commission_rate,
			o.fulfillment_type, o.delivery_fee, o.delivery_fee_final,
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

// loadUnderbilledOrders finds SETTLED orders whose current net payout exceeds what
// was actually credited for them (D-15). The statement that billed them is frozen
// and each (chef, week) is generated exactly once, so nothing else will ever revisit
// the difference.
//
// settled_net_payout IS NOT NULL is the safety predicate, not an optimisation: an
// order stamped by BackfillBilledStatementIDs has no recorded figure, so "what is
// still owed" is unknowable for it and any answer risks paying twice.
//
// The tip-catchup exclusion is defence in depth. BackfillChefTips only matches the
// pre-#964 shape (tip > 0 AND chef_tip = 0), which no order billed since this column
// existed can have — but it credits the same money on its own key, so the two paths
// are kept explicitly disjoint rather than by argument.
func loadUnderbilledOrders(db *gorm.DB, limit int) ([]catchupRow, error) {
	var rows []catchupRow
	err := db.Table("orders o").
		Select(`o.id, o.order_number, o.chef_id, o.billed_statement_id AS statement_id,
			o.subtotal, o.tax, o.currency, o.tax_inclusive, o.tax_food, o.tax_service, o.chef_tip, o.driver_tip, o.chef_funded_discount, o.commission_rate,
			o.fulfillment_type, o.delivery_fee, o.delivery_fee_final,
			o.settled_net_payout, c.state AS chef_state, o.delivery_address_state AS delivery_state`).
		Joins("JOIN chef_profiles c ON c.id = o.chef_id").
		Where("o.status = ?", models.OrderStatusDelivered).
		Where("o.deleted_at IS NULL").
		Where("o.billed_statement_id IS NOT NULL").
		Where("o.settled_net_payout IS NOT NULL").
		Where("o.refunded_at IS NULL").
		Where("COALESCE(o.gateway_split_paise, 0) = 0").
		Where("NOT EXISTS (SELECT 1 FROM chef_bonuses b WHERE b.source_key = ?)",
			gorm.Expr("? || o.id", chefTipCatchupKeyPrefix)).
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// reconcileUnderbilledOrders pays the difference on orders whose value grew after
// their statement froze. Safe to run repeatedly: the credit is unique on a source
// key carrying the settled-to figure, and the order's settled_net_payout is advanced
// to match, so an unchanged order stops matching on the next pass.
func reconcileUnderbilledOrders(db *gorm.DB) {
	rows, err := loadUnderbilledOrders(db, 500)
	if err != nil {
		log.Printf("statement-catchup: query under-billed orders failed: %v", err)
		return
	}
	adjusted := 0.0
	for i := range rows {
		r := rows[i]
		net := ComputeOrderEarnings(r.earningsInput(), r.ChefState).NetPayout
		delta := Round2(net - r.SettledNetPayout)
		if delta < underbilledEpsilon {
			// Never negative: a settlement is not clawed back here. An order whose value
			// FELL was refunded or adjusted, and those paths have their own machinery.
			continue
		}
		if err := raiseStatementAdjust(db, r, net, delta); err != nil {
			log.Printf("statement-catchup: adjustment for order %s failed (will retry): %v", r.ID, err)
			continue
		}
		adjusted += delta
	}
	if adjusted > 0 {
		log.Printf("statement-catchup: credited ₹%.2f owed on order(s) billed for less than they are now worth", adjusted)
	}
}

// raiseStatementAdjust credits one order's shortfall and advances its settled
// figure, so the order leaves the candidate set once the credit exists.
func raiseStatementAdjust(db *gorm.DB, r catchupRow, net, delta float64) error {
	var chef models.ChefProfile
	if err := db.Select("id", "user_id").First(&chef, "id = ?", r.ChefID).Error; err != nil {
		return fmt.Errorf("load chef %s: %w", r.ChefID, err)
	}
	if chef.UserID == uuid.Nil {
		return fmt.Errorf("chef %s has no user", r.ChefID)
	}
	reason := fmt.Sprintf("Order %s — value added after its statement closed", r.OrderNumber)
	if err := raiseChefBonus(db, r.ChefID, chef.UserID, models.ChefBonusStatementAdjust,
		StatementAdjustSourceKey(r.ID, ToPaise(net)), delta, nil, reason); err != nil {
		return err
	}
	// Advance only from the figure this credit was computed against, so a concurrent
	// run that already advanced it cannot be walked backwards.
	return db.Model(&models.Order{}).
		Where("id = ? AND settled_net_payout = ?", r.ID, r.SettledNetPayout).
		Update("settled_net_payout", Round2(net)).Error
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
		net := ComputeOrderEarnings(r.earningsInput(), r.ChefState).NetPayout
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
	// The other half of "paid at least once": an order billed for LESS than it is
	// now worth. Runs after the catch-up so an order settled above leaves with its
	// settled figure already recorded and does not also read as under-billed.
	reconcileUnderbilledOrders(db)
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
	// The credited figure is recorded alongside so a later increase in the order's
	// value is still detectable (D-15).
	return db.Model(&models.Order{}).
		Where("id = ? AND billed_statement_id IS NULL", r.ID).
		Updates(map[string]any{
			"billed_statement_id": r.StatementID,
			"settled_net_payout":  Round2(net),
		}).Error
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
