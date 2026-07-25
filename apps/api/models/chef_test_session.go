package models

import (
	"time"

	"github.com/google/uuid"
)

// Test session lifecycle. A session is open while the chef is in test mode,
// closed when they flip back to live, and purged once an admin deletes its rows.
const (
	TestSessionOpen   = "open"
	TestSessionClosed = "closed"
	TestSessionPurged = "purged"
)

// DefaultTestCloneWindowDays is how far back the live→test clone reaches for
// order history. Deep enough to reproduce essentially any production issue,
// shallow enough that the clone completes in seconds.
const DefaultTestCloneWindowDays = 30

// ChefTestSession is one debugging episode for one kitchen.
//
// Every live→test flip opens a new session with a fresh clone, so the evidence
// from a previous investigation is never overwritten by the next one. Sessions
// are retained indefinitely and removed only by an explicit admin purge.
type ChefTestSession struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`

	// SessionNo is a per-chef counter starting at 1, for human reference in the
	// admin UI ("session 3") — the UUID is not memorable enough to talk about.
	SessionNo int `gorm:"not null" json:"sessionNo"`

	Status string `gorm:"type:varchar(10);not null;default:'open';index" json:"status"`

	// Reason is the admin's free-text note for why this kitchen was flipped.
	// Required at open time: six months later nobody remembers.
	Reason string `gorm:"type:text;default:''" json:"reason"`

	// OrderWindowDays records how far back the clone reached, so a session's
	// contents are self-describing even after the default changes.
	OrderWindowDays int `gorm:"default:30" json:"orderWindowDays"`

	ClonedAt *time.Time `gorm:"" json:"clonedAt,omitempty"`

	// CloneSummary is a per-table row count as a JSON object, e.g.
	// {"menu_items":42,"orders":118}. Surfaced in the admin UI so an admin can
	// tell at a glance whether the clone actually caught the data they needed.
	// Stored as a JSON string, matching how PlatformSettings stores its blobs.
	CloneSummary string `gorm:"type:jsonb;default:'{}'" json:"cloneSummary,omitempty"`

	OpenedByID uuid.UUID  `gorm:"type:uuid" json:"openedById"`
	OpenedAt   time.Time  `gorm:"autoCreateTime" json:"openedAt"`
	ClosedByID *uuid.UUID `gorm:"type:uuid" json:"closedById,omitempty"`
	ClosedAt   *time.Time `gorm:"" json:"closedAt,omitempty"`
	PurgedAt   *time.Time `gorm:"" json:"purgedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (ChefTestSession) TableName() string { return "chef_test_sessions" }

// ChefModeStats holds a kitchen's aggregate counters for ONE mode.
//
// chef_profiles keeps the live numbers verbatim so every customer-facing
// surface reads them unchanged and a fake order can never move a real rating.
// This table is what the vendor and admin dashboards read for whichever mode is
// currently active.
type ChefModeStats struct {
	ChefID uuid.UUID `gorm:"type:uuid;primaryKey" json:"chefId"`
	Mode   string    `gorm:"type:varchar(4);primaryKey;default:'live'" json:"mode"`

	TotalOrders  int     `gorm:"default:0" json:"totalOrders"`
	Rating       float64 `gorm:"default:0" json:"rating"`
	TotalReviews int     `gorm:"default:0" json:"totalReviews"`
	IssueCount   int     `gorm:"default:0" json:"issueCount"`

	UpdatedAt time.Time `json:"updatedAt"`
}

func (ChefModeStats) TableName() string { return "chef_mode_stats" }
