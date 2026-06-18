package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// capacity.go — chef capacity & cutoff controls (#48). Atomic per-dish daily caps
// (oversell-safe under concurrency) + IST per-meal cutoff helpers.
//
// Scope: caps + cutoffs apply to à-la-carte CreateOrder (the dominant channel).
// Group/office orders (#46) and meal-plan/subscription orders (#193/#197) are
// SEPARATE channels — group items are reserved/released through their own
// lifecycle and meal-plan days book weekly-menu cells (distinct inventory), so
// neither consumes the à-la-carte MenuItem cap today. Extending caps to those
// channels (reserve at group-lock / plan-confirm, with their own release) is a
// tracked follow-up. IsPastSlotCutoff is exported for that subscription-timing
// use.

// capacityIST is the business timezone for the daily-cap calendar day and the
// "HH:MM" cutoff comparisons (IST has no DST). Containers run UTC, so all
// day/time math must go through this zone.
var capacityIST = time.FixedZone("IST", 5*3600+30*60)

// ErrSoldOut is returned when a reservation would exceed a dish's daily cap.
var ErrSoldOut = errors.New("sold out for today")

// CapacityDay returns the IST calendar day (midnight) for an instant — the key
// for the daily-sales counter and for releasing against the order's original day.
func CapacityDay(t time.Time) time.Time {
	ist := t.In(capacityIST)
	return time.Date(ist.Year(), ist.Month(), ist.Day(), 0, 0, 0, 0, capacityIST)
}

// ReserveCapacity atomically reserves qty of a capped dish for the given IST day
// inside the caller's transaction. No-op when cap <= 0 (unlimited). Returns
// ErrSoldOut if the reservation would exceed the cap. Concurrency-safe: the
// counter row is ensured (idempotent), then a single conditional UPDATE does the
// check-and-increment atomically.
func ReserveCapacity(tx *gorm.DB, chefID, menuItemID uuid.UUID, qty, capLimit int, day time.Time) error {
	if capLimit <= 0 || qty <= 0 {
		return nil
	}
	// Ensure the counter row exists without disturbing an existing count.
	if err := tx.Exec(`
		INSERT INTO menu_item_daily_sales (id, menu_item_id, chef_id, sale_date, sold_qty, created_at, updated_at)
		VALUES (gen_random_uuid(), ?, ?, ?, 0, now(), now())
		ON CONFLICT (menu_item_id, sale_date) DO NOTHING`,
		menuItemID, chefID, day).Error; err != nil {
		return err
	}
	// Atomic check-and-increment: the WHERE makes it oversell-safe under load.
	res := tx.Exec(`
		UPDATE menu_item_daily_sales SET sold_qty = sold_qty + ?, updated_at = now()
		WHERE menu_item_id = ? AND sale_date = ? AND sold_qty + ? <= ?`,
		qty, menuItemID, day, qty, capLimit)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSoldOut
	}
	return nil
}

// ReleaseCapacity returns qty back to a dish's counter for the order's original
// IST day (floored at 0). Idempotency is the caller's responsibility — only call
// once per cancel transition (the cancel paths are status-guarded).
func ReleaseCapacity(tx *gorm.DB, menuItemID uuid.UUID, qty int, day time.Time) error {
	if qty <= 0 {
		return nil
	}
	return tx.Exec(`
		UPDATE menu_item_daily_sales SET sold_qty = GREATEST(0, sold_qty - ?), updated_at = now()
		WHERE menu_item_id = ? AND sale_date = ?`,
		qty, menuItemID, day).Error
}

// SoldToday returns how many of a dish have sold on the given IST day.
func SoldToday(menuItemID uuid.UUID, day time.Time) int {
	var row models.MenuItemDailySales
	if err := database.DB.Where("menu_item_id = ? AND sale_date = ?", menuItemID, day).
		First(&row).Error; err != nil {
		return 0
	}
	return row.SoldQty
}

// RemainingToday returns the remaining count for a capped dish (nil when uncapped),
// and whether it is sold out.
func RemainingToday(menuItemID uuid.UUID, capLimit *int, day time.Time) (*int, bool) {
	if capLimit == nil || *capLimit <= 0 {
		return nil, false
	}
	rem := *capLimit - SoldToday(menuItemID, day)
	if rem < 0 {
		rem = 0
	}
	return &rem, rem == 0
}

// ── Cutoffs (pure, unit-tested) ──────────────────────────────────────────────

// ParseCutoff parses an "HH:MM" 24h string into (hour, minute, ok). Empty or
// malformed → ok=false (treated as "no cutoff").
func ParseCutoff(s string) (int, int, bool) {
	if len(s) != 5 || s[2] != ':' {
		return 0, 0, false
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	if s[0] < '0' || s[0] > '9' || s[1] < '0' || s[1] > '9' || s[3] < '0' || s[3] > '9' || s[4] < '0' || s[4] > '9' {
		return 0, 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

// IsPastCutoff reports whether `now` is at/after today's "HH:MM" cutoff in IST.
// A blank/invalid cutoff is never past (returns false). Pure for testing.
func IsPastCutoff(cutoff string, now time.Time) bool {
	h, m, ok := ParseCutoff(cutoff)
	if !ok {
		return false
	}
	ist := now.In(capacityIST)
	deadline := time.Date(ist.Year(), ist.Month(), ist.Day(), h, m, 0, 0, capacityIST)
	return !ist.Before(deadline)
}

// IsPastDailyClose reports whether the chef has closed for the rest of the day —
// cutoffs enabled and `now` is past BOTH the lunch and dinner cutoffs (whichever
// are set). Used to auto-close à-la-carte ordering. A meal with no cutoff doesn't
// gate the close. Returns false when cutoffs are disabled or none are set.
func IsPastDailyClose(s *models.ChefCapacitySettings, now time.Time) bool {
	if s == nil || !s.CutoffEnabled {
		return false
	}
	_, _, lunchOK := ParseCutoff(s.LunchCutoff)
	_, _, dinnerOK := ParseCutoff(s.DinnerCutoff)
	if !lunchOK && !dinnerOK {
		return false
	}
	if lunchOK && !IsPastCutoff(s.LunchCutoff, now) {
		return false
	}
	if dinnerOK && !IsPastCutoff(s.DinnerCutoff, now) {
		return false
	}
	return true
}

// IsPastSlotCutoff reports whether a meal slot's cutoff has passed (for meal-plan
// per-slot enforcement). slot is "lunch" or "dinner".
func IsPastSlotCutoff(s *models.ChefCapacitySettings, slot string, now time.Time) bool {
	if s == nil || !s.CutoffEnabled {
		return false
	}
	switch slot {
	case "lunch":
		return IsPastCutoff(s.LunchCutoff, now)
	case "dinner":
		return IsPastCutoff(s.DinnerCutoff, now)
	}
	return false
}

// GetChefCapacitySettings loads a chef's capacity settings, returning a default
// (cutoffs off, auto-sold-out on) when none exists.
func GetChefCapacitySettings(chefID uuid.UUID) *models.ChefCapacitySettings {
	var s models.ChefCapacitySettings
	if err := database.DB.Where("chef_id = ?", chefID).First(&s).Error; err != nil {
		return &models.ChefCapacitySettings{ChefID: chefID, AutoSoldOut: true}
	}
	return &s
}
