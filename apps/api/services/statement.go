package services

// statement.go — weekly settlement statement generation.
//
// Once a Mon–Sun week (IST) closes, every chef who had delivered orders in
// that week gets an immutable WeeklyStatement row freezing their settlement
// totals, plus a "statement ready" push. Generation is idempotent: the DB
// unique index on (chef_id, week_start) is the hard guard, Redis SETNX the
// soft guard that also suppresses the duplicate push across pods.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// statementOrderRow is the join row scanned when generating statements —
// order financials plus the owning chef's identity + home state (for the
// intra/inter-state GST split).
type statementOrderRow struct {
	OrderID            uuid.UUID `gorm:"column:id"`
	OrderNumber        string    `gorm:"column:order_number"`
	CompletedAt        time.Time `gorm:"column:delivered_at"`
	ItemRevenue        float64   `gorm:"column:subtotal"`
	Tax                float64   `gorm:"column:tax"`
	TaxFood            float64   `gorm:"column:tax_food"`
	TaxService         float64   `gorm:"column:tax_service"`
	ChefFundedDiscount float64   `gorm:"column:chef_funded_discount"`
	DeliveryFee        float64   `gorm:"column:delivery_fee"`
	ChefTip            float64   `gorm:"column:chef_tip"`
	DriverTip          float64   `gorm:"column:driver_tip"`
	DeliveryState      string    `gorm:"column:delivery_address_state"`
	// CommissionRate is the rate FROZEN on the order at checkout (#390); 0 for
	// legacy rows → callers fall back to the live/default rate via rowRate.
	CommissionRate float64   `gorm:"column:commission_rate"`
	ChefID         uuid.UUID `gorm:"column:chef_id"`
	UserID         uuid.UUID `gorm:"column:user_id"`
	ChefState      string    `gorm:"column:chef_state"`
	// FulfillmentType decides who the delivery fee belongs to; DeliveryFeeFinal
	// is the #703 lowered-at-accept figure the customer was actually charged.
	FulfillmentType  string   `gorm:"column:fulfillment_type"`
	DeliveryFeeFinal *float64 `gorm:"column:delivery_fee_final"`
}

// earningsInput maps a scanned row to the settlement engine. Every statement
// surface goes through here so none of them can disagree about what a chef is
// owed. flatRate is the legacy fallback for rows placed before #390 froze a rate.
func (r statementOrderRow) earningsInput(flatRate float64) EarningsInput {
	fee := r.DeliveryFee
	if r.DeliveryFeeFinal != nil {
		fee = *r.DeliveryFeeFinal
	}
	return EarningsInput{
		OrderID:              r.OrderID,
		OrderNumber:          r.OrderNumber,
		CompletedAt:          r.CompletedAt,
		ItemRevenue:          r.ItemRevenue,
		Tax:                  ChefTaxOf(r.Tax, r.TaxFood, r.TaxService),
		ChefFundedDiscount:   r.ChefFundedDiscount,
		DeliveryFee:          fee,
		ChefEarnsDeliveryFee: SettledChefEarnsDeliveryFee(r.FulfillmentType),
		ChefTip:              r.ChefTip,
		DriverTip:            r.DriverTip,
		DeliveryState:        r.DeliveryState,
		CommissionRate:       rowRate(r.CommissionRate, flatRate),
	}
}

// MostRecentClosedWeek returns the [start, end) bounds — in UTC — of the most
// recently completed Mon–Sun week relative to now, reckoned in IST. WeekStart
// is Monday 00:00 IST; WeekEnd is the following Monday 00:00 IST (exclusive).
func MostRecentClosedWeek(now time.Time) (time.Time, time.Time) {
	thisMonday := BusinessWeekStart(now)
	return thisMonday.AddDate(0, 0, -7).UTC(), thisMonday.UTC()
}

// GenerateWeeklyStatements computes and persists a WeeklyStatement for every
// chef with delivered orders in [weekStart, weekEnd), then pushes each chef.
// Safe to call repeatedly — already-issued statements are skipped.
//
// #390: statements now use gross = itemRevenue + Tax + chefTip (food GST in,
// delivery out) and read the per-order FROZEN commission_rate (rowRate falls back
// to the live rate for legacy rows that lack one). Pre-#390 rows were computed on
// the old delivery-in / tax-out basis; they are NOT rewritten — see the plan's
// MIGRATION-NOTE.md for the historical-rows and TDS-certificate caveats.
func GenerateWeeklyStatements(ctx context.Context, weekStart, weekEnd time.Time) (int, error) {
	rows, err := loadStatementOrderRows(weekStart, weekEnd)
	if err != nil {
		return 0, fmt.Errorf("load statement rows: %w", err)
	}

	// Flat platform commission (ADR-0001 / #390) — a single runtime rate for all
	// chefs, resolved once before the bucket loop.
	flatRate := GetCommissionRate(database.DB)

	// Group rows by chef so each chef's totals are computed in one pass.
	type chefBucket struct {
		userID         uuid.UUID
		chefState      string
		commissionRate float64
		totals         EarningsTotals
		// billed are the orders this statement bills and the net payout each is
		// billed for, stamped onto them when it is created so the catch-up can tell
		// settled orders from skipped ones (#927) and under-billed ones from orders
		// billed in full (D-15).
		billed []billedOrder
	}
	buckets := make(map[uuid.UUID]*chefBucket)
	for _, r := range rows {
		b := buckets[r.ChefID]
		if b == nil {
			b = &chefBucket{userID: r.UserID, chefState: r.ChefState, commissionRate: flatRate}
			buckets[r.ChefID] = b
		}
		earnings := ComputeOrderEarnings(r.earningsInput(flatRate), b.chefState)
		b.billed = append(b.billed, billedOrder{ID: r.OrderID, Net: earnings.NetPayout})
		b.totals.Add(earnings)
	}

	issued := 0
	for chefID, b := range buckets {
		if !claimStatementGeneration(ctx, chefID, weekStart) {
			continue
		}
		b.totals.Round()
		created, stmt, err := upsertWeeklyStatement(chefID, b.userID, weekStart, weekEnd, b.totals, b.billed)
		if err != nil {
			log.Printf("weekly-statement: persist failed for chef=%s week=%s: %v",
				chefID, weekStart.Format("2006-01-02"), err)
			continue
		}
		if !created {
			continue // already existed — no duplicate push
		}
		// #834: net any outstanding cancellation levies off this settlement and record them
		// as its penalty line. Claim-guarded, so a re-run can't deduct twice. A failure here
		// leaves the levies pending for the NEXT statement rather than losing them — the
		// statement is still valid, it just didn't collect this week.
		if _, pErr := ApplyChefPenaltiesToStatement(database.DB, stmt); pErr != nil {
			log.Printf("weekly-statement: penalty deduction failed for chef=%s week=%s (levies stay pending): %v",
				chefID, weekStart.Format("2006-01-02"), pErr)
		}
		// Referral / cashback credits ride the same settlement: a failure leaves
		// them pending for the next statement, never lost.
		if _, bErr := ApplyChefBonusesToStatement(database.DB, stmt); bErr != nil {
			log.Printf("weekly-statement: bonus credit failed for chef=%s week=%s (bonuses stay pending): %v",
				chefID, weekStart.Format("2006-01-02"), bErr)
		}
		// #1092: collect any outstanding debt off the settled figure, last, so the
		// recovery line reflects what is left after levies and credits. Shares the
		// ledger with the per-order collection site, so the debt is paid once.
		if _, rErr := ApplyChefRecoveryToStatement(database.DB, stmt); rErr != nil {
			log.Printf("weekly-statement: recovery collection failed for chef=%s week=%s (debt stays owed): %v",
				chefID, weekStart.Format("2006-01-02"), rErr)
		}
		// Push the payout AFTER deductions — the number the chef will actually receive.
		if err := sendStatementReadyPush(b.userID, weekStart, weekEnd, stmt.NetPayout); err != nil {
			log.Printf("weekly-statement: push failed for chef=%s: %v", chefID, err)
		}
		issued++
	}
	return issued, nil
}

func loadStatementOrderRows(weekStart, weekEnd time.Time) ([]statementOrderRow, error) {
	var rows []statementOrderRow
	err := database.DB.Raw(`
		-- tax_food/tax_service are NOT optional here: ChefTaxOf falls back to the
		-- whole order tax when both scan as zero, which credits the chef the
		-- platform's own GST on the fee and delivery. Omitting them from this
		-- list is invisible to a test that exercises the helper directly.
		SELECT o.id, o.order_number, o.delivered_at, o.subtotal, o.tax,
		       o.tax_food, o.tax_service, o.chef_funded_discount,
		       o.delivery_fee, o.chef_tip, o.driver_tip, o.delivery_address_state, o.commission_rate,
		       o.fulfillment_type, o.delivery_fee_final,
		       o.chef_id, c.user_id, c.state AS chef_state
		FROM   orders o
		JOIN   chef_profiles c ON c.id = o.chef_id
		WHERE  o.status        = 'delivered'
		AND    o.delivered_at >= ?
		AND    o.delivered_at  < ?
		AND    o.deleted_at    IS NULL
		-- Easy Split orders settled the chef's share at the gateway; putting
		-- them on the weekly statement would pay that share a second time.
		AND    COALESCE(o.gateway_split_paise, 0) = 0
		-- #927: a refunded order must never bill the chef. status stays 'delivered'
		-- on the order-issue refund path (it stamps refunded_at instead), so
		-- filtering on status alone lets a fully-refunded order onto the statement.
		-- Mirrors the payout-release guard (payout_release.go), which has always had
		-- this predicate — the statement path simply never grew one.
		AND    o.refunded_at   IS NULL
		-- #927: bill ONLY a hold the chef is actually owed on.
		--
		--   ''                → no hold was ever set (non-gateway order) — payable
		--   release_eligible  → customer confirmed, cleared for payout      — payable
		--   released          → already cleared                             — payable
		--
		-- Everything else is withheld/reversed (terminal, never payable) or awaiting/
		-- disputed (the customer has not confirmed, or is contesting — the money is
		-- still held and can still go back to them).
		--
		-- Excluding a TRANSIENT state is only safe because of the catch-up credit:
		-- this query is windowed on delivered_at and each (chef, week) statement is
		-- generated exactly once then frozen, so an order skipped here is never billed
		-- by any later statement. reconcileStatementCatchup settles those once they
		-- clear. Without it, a dispute resolved in the chef's favour after their week
		-- closed would silently cost them the whole order.
		AND    COALESCE(o.payout_hold_status, '') IN ('', 'release_eligible', 'released')
		-- #927: and never bill an order already settled — by an earlier statement, or
		-- by the catch-up credit. This is the half that makes the two payout paths
		-- reconcilable rather than merely independent.
		AND    o.billed_statement_id IS NULL
		ORDER  BY o.chef_id, o.delivered_at ASC
	`, weekStart, weekEnd).Scan(&rows).Error
	return rows, err
}

// upsertWeeklyStatement creates the statement row, returning created=false if
// one already exists for (chef, week). The DB unique index makes the insert
// the authoritative race-winner across pods. The created row is returned so the
// caller can apply post-creation adjustments (penalty deductions, #834).
func upsertWeeklyStatement(
	chefID, userID uuid.UUID, weekStart, weekEnd time.Time, t EarningsTotals, billed []billedOrder,
) (bool, *models.WeeklyStatement, error) {
	var existing models.WeeklyStatement
	err := database.DB.
		Where("chef_id = ? AND week_start = ?", chefID, weekStart).
		First(&existing).Error
	if err == nil {
		return false, &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil, err
	}

	stmt := models.WeeklyStatement{
		ChefID:             chefID,
		UserID:             userID,
		WeekStart:          weekStart,
		WeekEnd:            weekEnd,
		Currency:           EarningsCurrency,
		OrdersCount:        t.OrdersCount,
		GrossRevenue:       t.GrossRevenue,
		PlatformCommission: t.PlatformCommission,
		CGST:               t.CGST,
		SGST:               t.SGST,
		IGST:               t.IGST,
		TDS:                t.TDS,
		NetPayout:          t.NetPayout,
	}
	// #927: create the statement and stamp the orders it bills in ONE transaction.
	// The atomicity is load-bearing, not tidiness: a crash between the two would
	// leave a statement whose orders still read as unsettled, and the catch-up
	// would then credit orders this statement had already billed — paying the chef
	// twice for them.
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if cErr := tx.Create(&stmt).Error; cErr != nil {
			return cErr
		}
		return stampBilledOrders(tx, stmt.ID, billed)
	})
	if err != nil {
		// Lost the race to a concurrent pod — treat as "already issued".
		return false, nil, nil
	}
	return true, &stmt, nil
}

// claimStatementGeneration gates each (chef, week) tuple through Redis SETNX
// so two pods don't both push. Fails OPEN on Redis outage (the DB unique
// index still prevents a duplicate row; worst case is a duplicate push).
func claimStatementGeneration(ctx context.Context, chefID uuid.UUID, weekStart time.Time) bool {
	r := GetRedisClient()
	if !r.IsConnected() {
		return true
	}
	key := fmt.Sprintf("weekly_statement:%s:%s", chefID, weekStart.Format("2006-01-02"))
	dedupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	acquired, err := r.SetNX(dedupCtx, key, "1", 8*24*time.Hour)
	if err != nil {
		return true
	}
	return acquired
}

func sendStatementReadyPush(userID uuid.UUID, weekStart, weekEnd time.Time, netPayout float64) error {
	title := "Weekly statement ready"
	body := fmt.Sprintf(
		"Your settlement statement for %s–%s is ready. Net payout ₹%.2f.",
		weekStart.In(istLoc).Format("2 Jan"),
		weekEnd.In(istLoc).AddDate(0, 0, -1).Format("2 Jan"),
		netPayout,
	)
	data := map[string]string{
		"type":      "weekly_statement",
		"deeplink":  "homechef-vendor:///earnings",
		"weekStart": weekStart.Format("2006-01-02"),
	}
	return SendPushNotification(userID, title, body, data)
}
