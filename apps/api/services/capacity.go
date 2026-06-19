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

// ── Scheduled delivery slots (#51) ──────────────────────────────────────────
//
// Named lunch/dinner delivery windows with per-day capacity. Built on the same
// IST day-math + atomic reserve/release as the per-dish caps above. The chef's
// Lunch/DinnerCutoff (#48) doubles as the per-slot order cutoff. Booking counts
// live in ChefSlotDailyBookings, keyed by (chef, slot, IST delivery day) so they
// reset each day with no cron.

const (
	// SlotLunch / SlotDinner are the only two slot identifiers.
	SlotLunch  = "lunch"
	SlotDinner = "dinner"
	// slotMaxHorizonDays bounds how far ahead a customer can schedule (and how
	// many days the availability endpoint enumerates). Today = day 0.
	slotMaxHorizonDays = 6
)

var (
	// ErrSlotFull is returned when a booking would exceed a slot's daily capacity.
	ErrSlotFull = errors.New("delivery slot is full")
	// ErrSlotNotOffered is returned for a slot the chef hasn't configured.
	ErrSlotNotOffered = errors.New("delivery slot not offered")
	// ErrSlotClosed is returned when today's slot is past its cutoff / window.
	ErrSlotClosed = errors.New("delivery slot closed for that day")
	// ErrSlotDateInvalid is returned for a delivery date in the past or beyond
	// the booking horizon, or an unparseable date string.
	ErrSlotDateInvalid = errors.New("delivery date out of range")
)

// IsValidSlot reports whether slot is one of the two known slot identifiers.
func IsValidSlot(slot string) bool { return slot == SlotLunch || slot == SlotDinner }

// SlotWindow returns the configured display window (start, end "HH:MM") for a
// slot and whether it is offered (a non-empty, valid start time).
func SlotWindow(s *models.ChefCapacitySettings, slot string) (start, end string, ok bool) {
	if s == nil {
		return "", "", false
	}
	switch slot {
	case SlotLunch:
		start, end = s.LunchSlotStart, s.LunchSlotEnd
	case SlotDinner:
		start, end = s.DinnerSlotStart, s.DinnerSlotEnd
	default:
		return "", "", false
	}
	if _, _, valid := ParseCutoff(start); !valid {
		return "", "", false
	}
	return start, end, true
}

// SlotCapacity returns the per-day capacity for a slot (nil/0 = unlimited).
func SlotCapacity(s *models.ChefCapacitySettings, slot string) *int {
	if s == nil {
		return nil
	}
	switch slot {
	case SlotLunch:
		return s.LunchSlotCapacity
	case SlotDinner:
		return s.DinnerSlotCapacity
	}
	return nil
}

// slotCutoff returns the order cutoff "HH:MM" for a slot (reuses the #48 meal
// cutoffs). "" = no cutoff.
func slotCutoff(s *models.ChefCapacitySettings, slot string) string {
	switch slot {
	case SlotLunch:
		return s.LunchCutoff
	case SlotDinner:
		return s.DinnerCutoff
	}
	return ""
}

// istDateAt builds the IST instant for an IST calendar day at "HH:MM".
func istDateAt(day time.Time, hhmm string) time.Time {
	h, m, ok := ParseCutoff(hhmm)
	if !ok {
		h, m = 0, 0
	}
	ist := day.In(capacityIST)
	return time.Date(ist.Year(), ist.Month(), ist.Day(), h, m, 0, 0, capacityIST)
}

// ParseSlotDateIST parses a "YYYY-MM-DD" string into that IST calendar day
// (midnight). Empty → today IST. ok=false on a malformed string.
func ParseSlotDateIST(s string, now time.Time) (time.Time, bool) {
	if s == "" {
		return CapacityDay(now), true
	}
	t, err := time.ParseInLocation("2006-01-02", s, capacityIST)
	if err != nil {
		return time.Time{}, false
	}
	return CapacityDay(t), true
}

// slotOpenForDay reports whether a slot can still be ordered for the given IST
// delivery day: future days are always open; today is open only while `now` is
// before the slot's cutoff (if set) AND before the slot's window end.
func slotOpenForDay(s *models.ChefCapacitySettings, slot string, day, now time.Time) bool {
	today := CapacityDay(now)
	if day.After(today) {
		return true
	}
	if day.Before(today) {
		return false
	}
	// Same IST day — gate on cutoff then on the window end.
	if IsPastCutoff(slotCutoff(s, slot), now) {
		return false
	}
	_, end, _ := SlotWindow(s, slot)
	if _, _, ok := ParseCutoff(end); ok {
		return now.In(capacityIST).Before(istDateAt(day, end))
	}
	return true
}

// ResolveSlotSchedule validates a requested slot + delivery date against the
// chef's slot config and returns the order's ScheduledFor (the slot window start
// on that day, IST) and the IST booking day used for capacity. dateStr is
// "YYYY-MM-DD" (empty = today). Does not touch capacity — the caller reserves
// inside its transaction. Returns ErrSlotNotOffered / ErrSlotDateInvalid /
// ErrSlotClosed.
func ResolveSlotSchedule(s *models.ChefCapacitySettings, slot, dateStr string, now time.Time) (scheduledFor, day time.Time, err error) {
	if !IsValidSlot(slot) {
		return time.Time{}, time.Time{}, ErrSlotNotOffered
	}
	start, _, ok := SlotWindow(s, slot)
	if !s.SlotsEnabled || !ok {
		return time.Time{}, time.Time{}, ErrSlotNotOffered
	}
	day, parsed := ParseSlotDateIST(dateStr, now)
	if !parsed {
		return time.Time{}, time.Time{}, ErrSlotDateInvalid
	}
	today := CapacityDay(now)
	maxDay := today.AddDate(0, 0, slotMaxHorizonDays)
	if day.Before(today) || day.After(maxDay) {
		return time.Time{}, time.Time{}, ErrSlotDateInvalid
	}
	if !slotOpenForDay(s, slot, day, now) {
		return time.Time{}, time.Time{}, ErrSlotClosed
	}
	return istDateAt(day, start), day, nil
}

// ReserveSlot atomically reserves one (or qty) booking for a chef's slot on the
// given IST day inside the caller's transaction. No-op when capLimit <= 0
// (unlimited). Returns ErrSlotFull if the booking would exceed the capacity.
// Oversell-safe — same ensure-row + conditional-UPDATE pattern as ReserveCapacity.
func ReserveSlot(tx *gorm.DB, chefID uuid.UUID, slot string, qty, capLimit int, day time.Time) error {
	if capLimit <= 0 || qty <= 0 {
		return nil
	}
	if err := tx.Exec(`
		INSERT INTO chef_slot_daily_bookings (id, chef_id, slot, booking_date, booked_qty, created_at, updated_at)
		VALUES (gen_random_uuid(), ?, ?, ?, 0, now(), now())
		ON CONFLICT (chef_id, slot, booking_date) DO NOTHING`,
		chefID, slot, day).Error; err != nil {
		return err
	}
	res := tx.Exec(`
		UPDATE chef_slot_daily_bookings SET booked_qty = booked_qty + ?, updated_at = now()
		WHERE chef_id = ? AND slot = ? AND booking_date = ? AND booked_qty + ? <= ?`,
		qty, chefID, slot, day, qty, capLimit)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSlotFull
	}
	return nil
}

// ReleaseSlot returns qty bookings back to a chef's slot counter for the order's
// scheduled IST day (floored at 0). Idempotency is the caller's responsibility —
// only call once per cancel/refund transition (the paths are status-guarded).
func ReleaseSlot(tx *gorm.DB, chefID uuid.UUID, slot string, qty int, day time.Time) error {
	if qty <= 0 || !IsValidSlot(slot) {
		return nil
	}
	return tx.Exec(`
		UPDATE chef_slot_daily_bookings SET booked_qty = GREATEST(0, booked_qty - ?), updated_at = now()
		WHERE chef_id = ? AND slot = ? AND booking_date = ?`,
		qty, chefID, slot, day).Error
}

// SlotBooked returns how many bookings a chef's slot has on the given IST day.
func SlotBooked(chefID uuid.UUID, slot string, day time.Time) int {
	var row models.ChefSlotDailyBookings
	if err := database.DB.Where("chef_id = ? AND slot = ? AND booking_date = ?", chefID, slot, day).
		First(&row).Error; err != nil {
		return 0
	}
	return row.BookedQty
}

// SlotRemaining returns the remaining bookings for a capped slot (nil when
// uncapped) and whether it is full.
func SlotRemaining(chefID uuid.UUID, slot string, capLimit *int, day time.Time) (*int, bool) {
	if capLimit == nil || *capLimit <= 0 {
		return nil, false
	}
	rem := *capLimit - SlotBooked(chefID, slot, day)
	if rem < 0 {
		rem = 0
	}
	return &rem, rem == 0
}

// SlotAvailability is one offered slot on one day, for the public availability
// endpoint the checkout pickers consume.
type SlotAvailability struct {
	Date         string    `json:"date"`   // "YYYY-MM-DD" IST
	Slot         string    `json:"slot"`   // "lunch" | "dinner"
	Label        string    `json:"label"`  // "Lunch" | "Dinner"
	Window       string    `json:"window"` // "12:00–14:00"
	Start        string    `json:"start"`  // "12:00"
	End          string    `json:"end"`    // "14:00"
	Cutoff       string    `json:"cutoff,omitempty"`
	Remaining    *int      `json:"remaining"` // nil = unlimited
	Available    bool      `json:"available"` // open (not past cutoff/window) and not full
	ScheduledFor time.Time `json:"scheduledFor"`
}

// BuildSlotAvailability enumerates a chef's offered slots across today..+horizon
// with per-day remaining capacity and open/closed state. Empty when the chef
// hasn't enabled scheduled slots. `now` is injected for testability.
func BuildSlotAvailability(s *models.ChefCapacitySettings, chefID uuid.UUID, now time.Time) []SlotAvailability {
	out := []SlotAvailability{}
	if s == nil || !s.SlotsEnabled {
		return out
	}
	today := CapacityDay(now)
	labels := map[string]string{SlotLunch: "Lunch", SlotDinner: "Dinner"}
	for d := 0; d <= slotMaxHorizonDays; d++ {
		day := today.AddDate(0, 0, d)
		for _, slot := range []string{SlotLunch, SlotDinner} {
			start, end, ok := SlotWindow(s, slot)
			if !ok {
				continue
			}
			open := slotOpenForDay(s, slot, day, now)
			rem, full := SlotRemaining(chefID, slot, SlotCapacity(s, slot), day)
			window := start
			if end != "" {
				window = start + "–" + end
			}
			out = append(out, SlotAvailability{
				Date:         day.Format("2006-01-02"),
				Slot:         slot,
				Label:        labels[slot],
				Window:       window,
				Start:        start,
				End:          end,
				Cutoff:       slotCutoff(s, slot),
				Remaining:    rem,
				Available:    open && !full,
				ScheduledFor: istDateAt(day, start),
			})
		}
	}
	return out
}
