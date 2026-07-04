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

// istLocation is UTC+5:30. Settlement weeks are reckoned in IST so a week
// boundary lands at the Indian Monday midnight, not UTC midnight.
var istLocation = time.FixedZone("IST", 5*3600+1800)

// statementOrderRow is the join row scanned when generating statements —
// order financials plus the owning chef's identity + home state (for the
// intra/inter-state GST split).
type statementOrderRow struct {
	OrderID            uuid.UUID `gorm:"column:id"`
	OrderNumber        string    `gorm:"column:order_number"`
	CompletedAt        time.Time `gorm:"column:delivered_at"`
	ItemRevenue        float64   `gorm:"column:subtotal"`
	Tax                float64   `gorm:"column:tax"`
	ChefFundedDiscount float64   `gorm:"column:chef_funded_discount"`
	DeliveryFee        float64   `gorm:"column:delivery_fee"`
	ChefTip            float64   `gorm:"column:chef_tip"`
	DeliveryState      string    `gorm:"column:delivery_address_state"`
	// CommissionRate is the rate FROZEN on the order at checkout (#390); 0 for
	// legacy rows → callers fall back to the live/default rate via rowRate.
	CommissionRate float64   `gorm:"column:commission_rate"`
	ChefID         uuid.UUID `gorm:"column:chef_id"`
	UserID         uuid.UUID `gorm:"column:user_id"`
	ChefState      string    `gorm:"column:chef_state"`
}

// MostRecentClosedWeek returns the [start, end) bounds — in UTC — of the most
// recently completed Mon–Sun week relative to now, reckoned in IST. WeekStart
// is Monday 00:00 IST; WeekEnd is the following Monday 00:00 IST (exclusive).
func MostRecentClosedWeek(now time.Time) (time.Time, time.Time) {
	ist := now.In(istLocation)
	// Weekday: Sunday=0 … Saturday=6. Days since this week's Monday.
	daysSinceMonday := (int(ist.Weekday()) + 6) % 7
	thisMonday := time.Date(ist.Year(), ist.Month(), ist.Day(), 0, 0, 0, 0, istLocation).
		AddDate(0, 0, -daysSinceMonday)
	start := thisMonday.AddDate(0, 0, -7)
	return start.UTC(), thisMonday.UTC()
}

// GenerateWeeklyStatements computes and persists a WeeklyStatement for every
// chef with delivered orders in [weekStart, weekEnd), then pushes each chef.
// Safe to call repeatedly — already-issued statements are skipped.
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
	}
	buckets := make(map[uuid.UUID]*chefBucket)
	for _, r := range rows {
		b := buckets[r.ChefID]
		if b == nil {
			b = &chefBucket{userID: r.UserID, chefState: r.ChefState, commissionRate: flatRate}
			buckets[r.ChefID] = b
		}
		b.totals.Add(ComputeOrderEarnings(EarningsInput{
			OrderID:            r.OrderID,
			OrderNumber:        r.OrderNumber,
			CompletedAt:        r.CompletedAt,
			ItemRevenue:        r.ItemRevenue,
			Tax:                r.Tax,
			ChefFundedDiscount: r.ChefFundedDiscount,
			DeliveryFee:        r.DeliveryFee,
			ChefTip:            r.ChefTip,
			DeliveryState:      r.DeliveryState,
			// Per-row frozen rate (#390); the once-resolved flatRate is the legacy
			// fallback for orders placed before commission_rate was stamped.
			CommissionRate: rowRate(r.CommissionRate, flatRate),
		}, b.chefState))
	}

	issued := 0
	for chefID, b := range buckets {
		if !claimStatementGeneration(ctx, chefID, weekStart) {
			continue
		}
		b.totals.Round()
		created, err := upsertWeeklyStatement(chefID, b.userID, weekStart, weekEnd, b.totals)
		if err != nil {
			log.Printf("weekly-statement: persist failed for chef=%s week=%s: %v",
				chefID, weekStart.Format("2006-01-02"), err)
			continue
		}
		if !created {
			continue // already existed — no duplicate push
		}
		if err := sendStatementReadyPush(b.userID, weekStart, weekEnd, b.totals.NetPayout); err != nil {
			log.Printf("weekly-statement: push failed for chef=%s: %v", chefID, err)
		}
		issued++
	}
	return issued, nil
}

func loadStatementOrderRows(weekStart, weekEnd time.Time) ([]statementOrderRow, error) {
	var rows []statementOrderRow
	err := database.DB.Raw(`
		SELECT o.id, o.order_number, o.delivered_at, o.subtotal, o.tax, o.chef_funded_discount,
		       o.delivery_fee, o.chef_tip, o.delivery_address_state, o.commission_rate,
		       o.chef_id, c.user_id, c.state AS chef_state
		FROM   orders o
		JOIN   chef_profiles c ON c.id = o.chef_id
		WHERE  o.status        = 'delivered'
		AND    o.delivered_at >= ?
		AND    o.delivered_at  < ?
		AND    o.deleted_at    IS NULL
		ORDER  BY o.chef_id, o.delivered_at ASC
	`, weekStart, weekEnd).Scan(&rows).Error
	return rows, err
}

// upsertWeeklyStatement creates the statement row, returning created=false if
// one already exists for (chef, week). The DB unique index makes the insert
// the authoritative race-winner across pods.
func upsertWeeklyStatement(
	chefID, userID uuid.UUID, weekStart, weekEnd time.Time, t EarningsTotals,
) (bool, error) {
	var existing models.WeeklyStatement
	err := database.DB.
		Where("chef_id = ? AND week_start = ?", chefID, weekStart).
		First(&existing).Error
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
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
	if err := database.DB.Create(&stmt).Error; err != nil {
		// Lost the race to a concurrent pod — treat as "already issued".
		return false, nil
	}
	return true, nil
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
		weekStart.In(istLocation).Format("2 Jan"),
		weekEnd.In(istLocation).AddDate(0, 0, -1).Format("2 Jan"),
		netPayout,
	)
	data := map[string]string{
		"type":      "weekly_statement",
		"deeplink":  "homechef-vendor:///earnings",
		"weekStart": weekStart.Format("2006-01-02"),
	}
	return SendPushNotification(userID, title, body, data)
}
