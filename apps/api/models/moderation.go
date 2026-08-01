package models

// moderation.go — user-facing content reports and user blocks.
//
// App Review guideline 1.2 requires that any app carrying user-generated
// content ship four things: a way to filter objectionable content, a way for
// users to report it, a way to block abusive users, and published contact
// details. Home Chef carries three UGC surfaces — chef social posts and their
// comments, customer reviews of chefs, and order-scoped messaging — and until
// now had none of the first three from a user's point of view. Admin-side
// review hiding and message blocking existed, but a reviewer cannot see those,
// and neither can a customer being harassed.
//
// The design is deliberately generic over content type rather than one report
// table per surface: the triage queue, the rate limiting and the audit trail
// are the same work every time, and a fourth UGC surface should not need a
// fourth table.

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReportableType enumerates the content a user can report. Adding a surface
// means adding a constant here and a resolver in services/moderation.go —
// nothing else.
type ReportableType string

const (
	ReportableReview   ReportableType = "review"
	ReportableMessage  ReportableType = "message"
	ReportableUser     ReportableType = "user"
	ReportableChef     ReportableType = "chef"
	ReportableMenuItem ReportableType = "menu_item"
)

// ValidReportableType reports whether t is a type this API accepts. Callers
// validate at the edge so an unknown value never reaches the triage queue,
// where it would be untriageable.
func ValidReportableType(t ReportableType) bool {
	switch t {
	case ReportableReview, ReportableMessage, ReportableUser,
		ReportableChef, ReportableMenuItem:
		return true
	}
	return false
}

// ReportReason is the reporter's category. Kept short and concrete: a long list
// produces worse signal, because reporters pick the first plausible option.
type ReportReason string

const (
	ReasonSpam           ReportReason = "spam"
	ReasonHarassment     ReportReason = "harassment"
	ReasonHateSpeech     ReportReason = "hate_speech"
	ReasonSexualContent  ReportReason = "sexual_content"
	ReasonViolence       ReportReason = "violence"
	ReasonFoodSafety     ReportReason = "food_safety"
	ReasonMisinformation ReportReason = "misinformation"
	ReasonIllegal        ReportReason = "illegal"
	ReasonOther          ReportReason = "other"
)

// ValidReportReason reports whether r is an accepted reason.
func ValidReportReason(r ReportReason) bool {
	switch r {
	case ReasonSpam, ReasonHarassment, ReasonHateSpeech, ReasonSexualContent,
		ReasonViolence, ReasonFoodSafety, ReasonMisinformation, ReasonIllegal,
		ReasonOther:
		return true
	}
	return false
}

// ReportStatus tracks triage. Apple asks for reports to be acted on, not merely
// collected, so the queue distinguishes "nobody has looked" from "looked and
// decided".
type ReportStatus string

const (
	ReportPending  ReportStatus = "pending"
	ReportUpheld   ReportStatus = "upheld"   // content removed / user actioned
	ReportRejected ReportStatus = "rejected" // reviewed, no violation found
)

// ContentReport is one user's report of one piece of content.
type ContentReport struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// Who reported it. Not null: anonymous reports cannot be rate limited or
	// followed up, and are overwhelmingly noise.
	ReporterID uuid.UUID `gorm:"type:uuid;not null;index" json:"reporterId"`

	// What was reported. TargetID is the row id in the table implied by
	// TargetType; there is no FK because the target lives in one of several
	// tables, and because a report must survive the content being deleted —
	// that is exactly the case an auditor asks about.
	TargetType ReportableType `gorm:"type:varchar(24);not null;index:ix_content_reports_target" json:"targetType"`
	TargetID   uuid.UUID      `gorm:"type:uuid;not null;index:ix_content_reports_target" json:"targetId"`

	// TargetOwnerID is the user who authored the reported content, resolved at
	// report time. Denormalised on purpose: it lets the queue show "this author
	// has 9 upheld reports" without a per-type join, and it still answers after
	// the content row is gone.
	TargetOwnerID *uuid.UUID `gorm:"type:uuid;index" json:"targetOwnerId,omitempty"`

	Reason  ReportReason `gorm:"type:varchar(24);not null" json:"reason"`
	Details string       `gorm:"type:text" json:"details,omitempty"`

	Status ReportStatus `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`

	// Triage outcome.
	ReviewedBy     *uuid.UUID `gorm:"type:uuid" json:"reviewedBy,omitempty"`
	ReviewedAt     *time.Time `gorm:"" json:"reviewedAt,omitempty"`
	ResolutionNote string     `gorm:"type:text" json:"resolutionNote,omitempty"`

	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ContentReport) TableName() string { return "content_reports" }

// UserBlock is one user choosing not to see another.
//
// Blocking is symmetric in effect but not in record: A blocking B hides B's
// content from A and prevents B from messaging A, without telling B. One row
// per (blocker, blocked) pair.
type UserBlock struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	BlockerID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_user_blocks_pair" json:"blockerId"`
	BlockedID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:ux_user_blocks_pair;index" json:"blockedId"`

	// Optional: blocks raised from a report carry the reason through, so the
	// triage queue can weigh "blocked by many people" alongside report counts.
	Reason ReportReason `gorm:"type:varchar(24)" json:"reason,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

func (UserBlock) TableName() string { return "user_blocks" }
