package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ErrFlipBlocked is returned when a live→test flip is refused because the
// kitchen still has money or orders in flight.
var ErrFlipBlocked = errors.New("kitchen has work in flight and cannot be moved to test mode")

// ErrSessionOpen is returned when a purge targets a session that is still open.
var ErrSessionOpen = errors.New("close the test session before purging it")

// TestFlipBlockers lists, in plain language, everything that currently prevents
// moving a kitchen from live into test mode.
//
// Flipping mid-service would strand real customers holding an order they can no
// longer see, and would leave real money mid-flight in a kitchen that has just
// been redefined as a sandbox. Each obstruction is counted and named so the
// admin UI can say "3 active orders, 1 pending payout" instead of failing with a
// generic error.
//
// An empty slice means the flip is safe.
func TestFlipBlockers(db *gorm.DB, chefID uuid.UUID) []string {
	var out []string

	count := func(model interface{}, query string, args ...interface{}) int64 {
		var n int64
		if err := db.Model(model).Where(query, args...).Count(&n).Error; err != nil {
			// A failed check must not silently read as "nothing blocking" — that
			// would let a flip through precisely when we can't verify it's safe.
			out = append(out, "could not verify kitchen state — try again")
			return 0
		}
		return n
	}

	if n := count(&models.Order{},
		"chef_id = ? AND mode = ? AND status NOT IN ?",
		chefID, models.ChefModeLive, []models.OrderStatus{
			models.OrderStatusDelivered, models.OrderStatusCancelled, models.OrderStatusRefunded,
		}); n > 0 {
		out = append(out, plural(n, "active order", "active orders"))
	}

	if n := count(&models.Order{},
		"chef_id = ? AND mode = ? AND status = ? AND payout_settled_at IS NULL AND payout_hold_status <> ''",
		chefID, models.ChefModeLive, models.OrderStatusDelivered); n > 0 {
		out = append(out, plural(n, "unsettled payout", "unsettled payouts"))
	}

	if n := count(&models.MealPlan{},
		"chef_id = ? AND mode = ? AND status NOT IN ?",
		chefID, models.ChefModeLive, []models.MealPlanStatus{
			models.MealPlanCompleted, models.MealPlanCancelled, models.MealPlanExpired,
		}); n > 0 {
		out = append(out, plural(n, "active meal plan", "active meal plans"))
	}

	return out
}

func plural(n int64, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// OpenTestSession moves a kitchen into test mode, opening a numbered session and
// cloning the kitchen's setup plus a window of recent orders into it.
//
// Refused with ErrFlipBlocked when anything is still in flight. The whole
// operation — blocker check, session insert, clone, chef flip — runs in one
// transaction, so a failure anywhere leaves the kitchen live and unchanged
// rather than half-copied and inaccessible.
//
// Each flip opens a NEW session rather than reusing the last one, so the
// evidence from a previous investigation is never overwritten by the next.
func OpenTestSession(db *gorm.DB, chefID, adminID uuid.UUID, reason string, windowDays int) (*models.ChefTestSession, error) {
	if windowDays <= 0 {
		windowDays = models.DefaultTestCloneWindowDays
	}

	var session models.ChefTestSession
	err := db.Transaction(func(tx *gorm.DB) error {
		var chef models.ChefProfile
		if err := tx.First(&chef, "id = ?", chefID).Error; err != nil {
			return fmt.Errorf("test-session: load chef %s: %w", chefID, err)
		}
		if chef.IsTestMode() {
			return fmt.Errorf("test-session: chef %s is already in test mode", chefID)
		}
		if blockers := TestFlipBlockers(tx, chefID); len(blockers) > 0 {
			return fmt.Errorf("%w: %v", ErrFlipBlocked, blockers)
		}

		var maxNo int
		if err := tx.Model(&models.ChefTestSession{}).
			Where("chef_id = ?", chefID).
			Select("COALESCE(MAX(session_no), 0)").Scan(&maxNo).Error; err != nil {
			return fmt.Errorf("test-session: next session number: %w", err)
		}

		session = models.ChefTestSession{
			ID:              uuid.New(),
			ChefID:          chefID,
			SessionNo:       maxNo + 1,
			Status:          models.TestSessionOpen,
			Reason:          reason,
			OrderWindowDays: windowDays,
			OpenedByID:      adminID,
			OpenedAt:        time.Now(),
			CloneSummary:    "{}",
		}
		if err := tx.Create(&session).Error; err != nil {
			return fmt.Errorf("test-session: create session: %w", err)
		}

		// A born-test kitchen (never live) has nothing to clone; an established
		// one gets its setup and recent history copied so the sandbox behaves
		// like the real kitchen.
		if !chef.IsBornTest() {
			summary, err := CloneChefIntoSession(tx, chefID, &session, windowDays)
			if err != nil {
				return fmt.Errorf("test-session: clone chef %s: %w", chefID, err)
			}
			now := time.Now()
			if err := tx.Model(&session).Updates(map[string]any{
				"cloned_at":     &now,
				"clone_summary": summary,
			}).Error; err != nil {
				return fmt.Errorf("test-session: record clone summary: %w", err)
			}
			session.ClonedAt = &now
			session.CloneSummary = summary
		}

		if err := tx.Model(&models.ChefProfile{}).Where("id = ?", chefID).Updates(map[string]any{
			"mode":                   models.ChefModeTest,
			"active_test_session_id": session.ID,
		}).Error; err != nil {
			return fmt.Errorf("test-session: flip chef %s to test: %w", chefID, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// CloseTestSession returns a kitchen to live mode and closes its open session.
//
// There is no restore step, and deliberately so: while the kitchen was in test
// mode every write went to a test-partition row, so the live rows were never
// touched. Returning to live simply makes them visible again, exactly as they
// were at the moment of the flip.
//
// Closing when no session is open is a no-op that still sets the kitchen live,
// so a chef somehow stranded in an inconsistent state can always be recovered.
func CloseTestSession(db *gorm.DB, chefID, adminID uuid.UUID) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var chef models.ChefProfile
		if err := tx.First(&chef, "id = ?", chefID).Error; err != nil {
			return fmt.Errorf("test-session: load chef %s: %w", chefID, err)
		}

		now := time.Now()
		if err := tx.Model(&models.ChefTestSession{}).
			Where("chef_id = ? AND status = ?", chefID, models.TestSessionOpen).
			Updates(map[string]any{
				"status":       models.TestSessionClosed,
				"closed_at":    &now,
				"closed_by_id": adminID,
			}).Error; err != nil {
			return fmt.Errorf("test-session: close session for chef %s: %w", chefID, err)
		}

		updates := map[string]any{
			"mode":                   models.ChefModeLive,
			"active_test_session_id": nil,
		}
		// FirstLiveAt is stamped once and never moved. It is what turns a
		// born-test kitchen (hidden from customers) into an established one
		// (shown as closed when flipped), so re-stamping it on every close would
		// be harmless but misleading in the audit trail.
		if chef.FirstLiveAt == nil {
			updates["first_live_at"] = &now
		}
		if err := tx.Model(&models.ChefProfile{}).Where("id = ?", chefID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("test-session: flip chef %s to live: %w", chefID, err)
		}
		return nil
	})
}

// PurgeTestSession deletes every row a session produced and marks it purged.
//
// Refuses an OPEN session: purging one while the kitchen is still operating in
// it would delete rows out from under live traffic. Close first, then purge.
//
// Returns per-table deletion counts so the admin UI can show what was removed.
func PurgeTestSession(db *gorm.DB, sessionID uuid.UUID) (map[string]int, error) {
	deleted := map[string]int{}
	err := db.Transaction(func(tx *gorm.DB) error {
		var session models.ChefTestSession
		if err := tx.First(&session, "id = ?", sessionID).Error; err != nil {
			return fmt.Errorf("test-session: load session %s: %w", sessionID, err)
		}
		if session.Status == models.TestSessionOpen {
			return ErrSessionOpen
		}

		for _, table := range partitionedTables {
			// Skip tables absent from this environment. In production every
			// partitioned table exists, so nothing is skipped; the test fixtures
			// create only the tables a given case needs.
			if !tx.Migrator().HasTable(table) {
				continue
			}
			res := tx.Exec(
				fmt.Sprintf("DELETE FROM %s WHERE test_session_id = ?", table), sessionID)
			if res.Error != nil {
				return fmt.Errorf("test-session: purge %s: %w", table, res.Error)
			}
			if res.RowsAffected > 0 {
				deleted[table] = int(res.RowsAffected)
			}
		}

		now := time.Now()
		if err := tx.Model(&session).Updates(map[string]any{
			"status":    models.TestSessionPurged,
			"purged_at": &now,
		}).Error; err != nil {
			return fmt.Errorf("test-session: mark session %s purged: %w", sessionID, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return deleted, nil
}
