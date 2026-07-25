package services

// account_lifecycle.go — the account state machine behind deactivate, delete,
// restore and start-fresh.
//
//	active ──deactivate──▶ deactivated ──reactivate──▶ active
//	   │                        │
//	   └────────delete──────────┴──▶ pending_deletion ──restore──▶ active (pending re-approval)
//	                                        │
//	                                        └──purge_after elapses──▶ purged
//
// The transitions are role-agnostic; everything role-specific lives behind
// RoleCascade so the ordering, idempotency and audit rules are written and
// tested once rather than three times.
//
// Deletion is a *soft* delete: the row stays in Postgres for the restore window
// so a returning user gets their history back at full fidelity. Only the purge
// sweeper (account_purge_cron.go) erases anything.

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// RestoreWindow is how long a deleted account can be restored before the
// sweeper erases it. The privacy policy and both stores' Data Safety forms
// quote this figure — change them together.
const RestoreWindow = 180 * 24 * time.Hour

// ErrBlocked is returned by RequestDeletion when preconditions are unmet. The
// handler turns it into a 409 carrying the blockers.
var ErrBlocked = errors.New("account: deletion blocked by unfinished business")

// RoleCascade is the role-specific half of each transition. Every method runs
// inside the caller's transaction, so returning an error rolls the whole
// transition back rather than leaving the account half-changed.
type RoleCascade interface {
	// OnDeactivate takes the account's public surfaces offline reversibly.
	OnDeactivate(tx *gorm.DB, userID uuid.UUID) error
	// OnDelete takes them offline for the retention window.
	OnDelete(tx *gorm.DB, userID uuid.UUID) error
	// OnRestore brings data back but resets standing — approval flags cleared,
	// identity documents dropped, re-approval queued.
	OnRestore(tx *gorm.DB, userID uuid.UUID) error
	// Purge hard-deletes the role's rows. Must be idempotent: the sweeper
	// retries, and a partially-purged user must converge.
	Purge(tx *gorm.DB, userID uuid.UUID) error
}

// CascadeFor returns the cascade for a user's role. An unknown role gets the
// customer cascade, which touches only rows every user can own — never a no-op
// that would silently skip cleanup.
func CascadeFor(role models.UserRole) RoleCascade {
	switch role {
	case models.RoleChef:
		return chefCascade{}
	case models.RoleDelivery:
		return driverCascade{}
	default:
		return customerCascade{}
	}
}

// Deactivate reversibly pauses an account. Idempotent: deactivating an already
// deactivated account is a no-op success.
//
// This flips is_active, which middleware/bff_auth.go already rejects with a 403
// on the next request — no separate enforcement path is needed.
func Deactivate(db *gorm.DB, user *models.User, reason string) error {
	if !user.IsActive {
		return nil
	}
	now := time.Now().UTC()

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).
			Where("id = ?", user.ID).
			Updates(map[string]any{
				"is_active":       false,
				"deactivated_at":  now,
				"deletion_reason": reason,
			}).Error; err != nil {
			return fmt.Errorf("account: deactivate user: %w", err)
		}
		if err := CascadeFor(user.Role).OnDeactivate(tx, user.ID); err != nil {
			return err
		}
		user.IsActive = false
		user.DeactivatedAt = &now
		return nil
	})
}

// Reactivate lifts a deactivation. Approval is deliberately NOT reset — a pause
// is not a deletion, and the account never left the platform's trust boundary.
// The chef's accepting_orders stays off until they choose to reopen.
func Reactivate(db *gorm.DB, user *models.User) error {
	if user.IsActive {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).
			Where("id = ?", user.ID).
			Updates(map[string]any{
				"is_active":       true,
				"deactivated_at":  nil,
				"deletion_reason": "",
			}).Error; err != nil {
			return fmt.Errorf("account: reactivate user: %w", err)
		}
		user.IsActive = true
		user.DeactivatedAt = nil
		return nil
	})
}

// RequestDeletion soft-deletes the account and starts the restore window.
// Returns ErrBlocked plus the blockers when money or work is still in flight.
//
// The GIP credential is torn down by the caller *after* this commits: an
// Identity Platform hiccup must never roll back a committed erasure.
func RequestDeletion(db *gorm.DB, user *models.User, reason string) ([]Blocker, error) {
	if user.DeletedAt.Valid {
		return nil, nil // already pending deletion — idempotent
	}

	// Fail CLOSED. The blocker queries swallow their errors, so a database
	// problem would otherwise read as "nothing outstanding" and delete an
	// account still holding escrow, store credit or an unreleased payout.
	if err := CanDetermineBlockers(db, *user); err != nil {
		return nil, err
	}
	if blockers := DeletionBlockers(db, *user); len(blockers) > 0 {
		return blockers, ErrBlocked
	}

	now := time.Now().UTC()
	purgeAfter := now.Add(RestoreWindow)

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := CascadeFor(user.Role).OnDelete(tx, user.ID); err != nil {
			return err
		}
		// purge_after and the reason must land before the soft delete: once
		// deleted_at is set, a plain Updates() is scoped out by GORM and would
		// silently affect zero rows.
		if err := tx.Model(&models.User{}).
			Where("id = ?", user.ID).
			Updates(map[string]any{
				"purge_after":     purgeAfter,
				"deletion_reason": reason,
				"is_active":       false,
			}).Error; err != nil {
			return fmt.Errorf("account: stamp purge window: %w", err)
		}
		if err := tx.Delete(&models.User{}, "id = ?", user.ID).Error; err != nil {
			return fmt.Errorf("account: soft delete user: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	user.PurgeAfter = &purgeAfter
	user.IsActive = false
	// Reflect the soft delete on the caller's copy, so a retry with the same
	// struct short-circuits above instead of re-running and pushing the restore
	// window further out each time.
	user.DeletedAt = gorm.DeletedAt{Time: now, Valid: true}
	return nil, nil
}

// Restore brings a pending-deletion account back. Data returns at full
// fidelity; standing does not — the cascade resets approval flags, drops
// identity documents and queues re-approval, so a restored chef is invisible to
// customers until an admin re-approves them.
//
// newGIPUid/newTenant/newProvider rebind the row to the identity the user just
// created, since the original credential was deleted.
func Restore(db *gorm.DB, userID uuid.UUID, newGIPUid, newTenant, newProvider string) (*models.User, error) {
	var user models.User
	if err := db.Unscoped().First(&user, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("account: load for restore: %w", err)
	}
	if !user.DeletedAt.Valid {
		return &user, nil // already live — idempotent
	}
	if user.PurgeAfter != nil && time.Now().UTC().After(*user.PurgeAfter) {
		return nil, fmt.Errorf("account: restore window expired")
	}

	now := time.Now().UTC()
	err := db.Transaction(func(tx *gorm.DB) error {
		// Raw SQL, not Updates(map): the row is soft-deleted (so the ordinary
		// scope matches nothing) AND this must write real NULLs. GORM drops nil
		// map values rather than emitting NULL, which would silently leave
		// deleted_at and purge_after set — the account would look restored in
		// the response while staying invisible and still queued for purge.
		if err := tx.Exec(`UPDATE users SET deleted_at = NULL, purge_after = NULL,
				deactivated_at = NULL, deletion_reason = '', is_active = ?,
				restored_at = ?, gip_uid = ?, gip_tenant_id = ?, gip_provider = ?
			WHERE id = ?`,
			true, now, newGIPUid, newTenant, newProvider, userID).Error; err != nil {
			return fmt.Errorf("account: restore user: %w", err)
		}
		return CascadeFor(user.Role).OnRestore(tx, userID)
	})
	if err != nil {
		return nil, err
	}

	// Reload into a FRESH struct, not the one loaded above: scanning a NULL
	// column into an already-populated pointer field leaves the stale value, so
	// reusing `user` would report the old purge_after and make a successful
	// restore look like it had not cleared the window.
	var restored models.User
	if err := db.First(&restored, "id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("account: reload after restore: %w", err)
	}
	log.Printf("account: restored user=%s role=%s (re-approval required)", userID, restored.Role)
	return &restored, nil
}

// PurgeUser hard-erases one account: the role cascade first (children), then
// the user row. Idempotent — a re-run over an already-purged user deletes
// nothing and reports success, which is what lets the sweeper retry freely.
func PurgeUser(db *gorm.DB, userID uuid.UUID, role models.UserRole) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := CascadeFor(role).Purge(tx, userID); err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&models.User{}, "id = ?", userID).Error; err != nil {
			return fmt.Errorf("account: purge user row: %w", err)
		}
		return nil
	})
}
