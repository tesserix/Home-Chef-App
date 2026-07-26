package services

// moderation.go — the user-facing half of content moderation: reporting
// content, blocking users, and filtering blocked authors out of what a user
// sees.
//
// Required by App Review guideline 1.2 for any app with user-generated content.
// Home Chef's UGC surfaces are chef social posts, comments on those posts,
// customer reviews of chefs, and order-scoped messaging.
//
// Admin-side tooling (hide a review, block a message) already existed, but a
// user could neither report nor block, which is what 1.2 actually asks for and
// what a reviewer will look for.

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

var (
	// ErrSelfBlock is returned when a user tries to block themselves. Harmless
	// but always a client bug, so it surfaces rather than silently succeeding.
	ErrSelfBlock = errors.New("moderation: cannot block yourself")

	// ErrTargetNotFound means the reported content does not exist.
	ErrTargetNotFound = errors.New("moderation: reported content not found")

	// ErrDuplicateReport means this user already has an open report against
	// this exact content.
	ErrDuplicateReport = errors.New("moderation: already reported")
)

// ResolveTargetOwner finds the user who authored a piece of content.
//
// The owner is snapshotted onto the report so triage can rank authors by upheld
// reports without a per-type join, and so the report still names someone after
// the content itself is deleted.
//
// Returns ErrTargetNotFound if the content does not exist — reporting a
// non-existent row is either a stale client or someone probing ids, and neither
// should create a queue entry.
func ResolveTargetOwner(db *gorm.DB, t models.ReportableType, id uuid.UUID) (uuid.UUID, error) {
	var ownerID uuid.UUID

	switch t {
	case models.ReportableReview:
		var row models.Review
		if err := db.Select("customer_id").First(&row, "id = ?", id).Error; err != nil {
			return uuid.Nil, wrapLookup(err)
		}
		ownerID = row.CustomerID

	case models.ReportableSocialPost:
		// Posts are owned by a chef profile; reports are about people, so this
		// resolves through to the underlying user.
		var row models.Post
		if err := db.Select("chef_id").First(&row, "id = ?", id).Error; err != nil {
			return uuid.Nil, wrapLookup(err)
		}
		return chefOwnerUserID(db, row.ChefID)

	case models.ReportablePostComment:
		var row models.PostComment
		if err := db.Select("user_id").First(&row, "id = ?", id).Error; err != nil {
			return uuid.Nil, wrapLookup(err)
		}
		ownerID = row.UserID

	case models.ReportableMessage:
		var row models.ChatMessage
		if err := db.Select("sender_id").First(&row, "id = ?", id).Error; err != nil {
			return uuid.Nil, wrapLookup(err)
		}
		ownerID = row.SenderID

	case models.ReportableChef:
		return chefOwnerUserID(db, id)

	case models.ReportableMenuItem:
		var row models.MenuItem
		if err := db.Select("chef_id").First(&row, "id = ?", id).Error; err != nil {
			return uuid.Nil, wrapLookup(err)
		}
		return chefOwnerUserID(db, row.ChefID)

	case models.ReportableUser:
		var row models.User
		if err := db.Select("id").First(&row, "id = ?", id).Error; err != nil {
			return uuid.Nil, wrapLookup(err)
		}
		ownerID = row.ID

	default:
		return uuid.Nil, fmt.Errorf("moderation: unsupported target type %q", t)
	}

	return ownerID, nil
}

func chefOwnerUserID(db *gorm.DB, chefID uuid.UUID) (uuid.UUID, error) {
	var chef models.ChefProfile
	if err := db.Select("user_id").First(&chef, "id = ?", chefID).Error; err != nil {
		return uuid.Nil, wrapLookup(err)
	}
	return chef.UserID, nil
}

func wrapLookup(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrTargetNotFound
	}
	return err
}

// CreateReport records a report, resolving and snapshotting the content owner.
//
// Idempotent per (reporter, target) while a report is still pending: tapping
// Report twice must not create two queue entries, and must not look like a
// failure to the user either.
func CreateReport(
	db *gorm.DB,
	reporterID uuid.UUID,
	targetType models.ReportableType,
	targetID uuid.UUID,
	reason models.ReportReason,
	details string,
) (models.ContentReport, error) {
	ownerID, err := ResolveTargetOwner(db, targetType, targetID)
	if err != nil {
		return models.ContentReport{}, err
	}

	var existing models.ContentReport
	err = db.Where(
		"reporter_id = ? AND target_type = ? AND target_id = ? AND status = ?",
		reporterID, targetType, targetID, models.ReportPending,
	).First(&existing).Error
	if err == nil {
		return existing, ErrDuplicateReport
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return models.ContentReport{}, err
	}

	report := models.ContentReport{
		ReporterID: reporterID,
		TargetType: targetType,
		TargetID:   targetID,
		Reason:     reason,
		Details:    details,
		Status:     models.ReportPending,
	}
	if ownerID != uuid.Nil {
		report.TargetOwnerID = &ownerID
	}

	if err := db.Create(&report).Error; err != nil {
		return models.ContentReport{}, err
	}
	return report, nil
}

// BlockUser hides blocked's content from blocker and stops blocked messaging
// them. Idempotent — blocking twice is a no-op success, because the client
// cannot always know the current state.
func BlockUser(db *gorm.DB, blockerID, blockedID uuid.UUID, reason models.ReportReason) error {
	if blockerID == blockedID {
		return ErrSelfBlock
	}

	var existing models.UserBlock
	err := db.Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return db.Create(&models.UserBlock{
		BlockerID: blockerID,
		BlockedID: blockedID,
		Reason:    reason,
	}).Error
}

// UnblockUser reverses a block. Idempotent.
func UnblockUser(db *gorm.DB, blockerID, blockedID uuid.UUID) error {
	return db.Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Delete(&models.UserBlock{}).Error
}

// BlockedUserIDs returns everyone this user has blocked.
//
// Returns an empty slice, never nil, so callers can use it in a NOT IN clause
// without a length check — an accidental `NOT IN (NULL)` matches nothing and
// would silently empty the feed.
func BlockedUserIDs(db *gorm.DB, userID uuid.UUID) ([]uuid.UUID, error) {
	ids := []uuid.UUID{}
	if userID == uuid.Nil {
		return ids, nil // anonymous browsing — nobody is blocked
	}
	if err := db.Model(&models.UserBlock{}).
		Where("blocker_id = ?", userID).
		Pluck("blocked_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// IsBlockedEitherWay reports whether either user has blocked the other.
//
// Messaging checks both directions: a blocked sender must not reach the person
// who blocked them, and someone who blocked another user should not be able to
// message them either — a one-way check leaves an obvious harassment path open.
func IsBlockedEitherWay(db *gorm.DB, a, b uuid.UUID) (bool, error) {
	var count int64
	err := db.Model(&models.UserBlock{}).
		Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)", a, b, b, a).
		Count(&count).Error
	return count > 0, err
}

// AutoHideThreshold is how many distinct pending reports a piece of content
// needs before it is hidden pending triage.
//
// This is the "filtering objectionable content" half of guideline 1.2. Pure
// human triage is not a filter — it is a queue, and it is empty at 3am. Three
// distinct reporters is high enough that one angry customer cannot suppress a
// chef, and low enough to act before a moderator wakes up.
const AutoHideThreshold = 3

// PendingReportCount counts distinct reporters with an open report against a
// piece of content.
func PendingReportCount(db *gorm.DB, t models.ReportableType, id uuid.UUID) (int64, error) {
	var count int64
	err := db.Model(&models.ContentReport{}).
		Where("target_type = ? AND target_id = ? AND status = ?", t, id, models.ReportPending).
		Distinct("reporter_id").
		Count(&count).Error
	return count, err
}

// ApplyAutoHide hides content that has crossed AutoHideThreshold.
//
// Best-effort and idempotent: hiding an already-hidden row is a no-op, and a
// failure here must not fail the user's report — the report is the record that
// matters, and triage will catch what the threshold missed.
func ApplyAutoHide(db *gorm.DB, t models.ReportableType, id uuid.UUID) (bool, error) {
	count, err := PendingReportCount(db, t, id)
	if err != nil || count < AutoHideThreshold {
		return false, err
	}

	const note = "Automatically hidden pending review after multiple reports."
	switch t {
	case models.ReportableReview:
		err = db.Model(&models.Review{}).Where("id = ?", id).
			Updates(map[string]any{"is_hidden": true, "hidden_reason": note}).Error
	case models.ReportableSocialPost:
		err = db.Model(&models.Post{}).Where("id = ?", id).
			Updates(map[string]any{"is_moderated": true, "moderator_note": note}).Error
	case models.ReportablePostComment:
		err = db.Model(&models.PostComment{}).Where("id = ?", id).
			Update("is_hidden", true).Error
	default:
		// Users, chefs and messages are not auto-hidden: suppressing a whole
		// account or a delivery conversation on report volume alone is a denial
		// of service against the reported party. Those go to human triage.
		return false, nil
	}
	return err == nil, err
}
