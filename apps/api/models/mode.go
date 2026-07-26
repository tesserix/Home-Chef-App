package models

import (
	"strings"

	"github.com/google/uuid"
)

// Mode partitions the platform into two worlds that share one database.
//
// "live" is the real marketplace: real customers, real money, live Razorpay
// credentials. "test" is a sandbox kitchen — visible only to the test-mode
// viewer allowlist, paid for with Razorpay test credentials, and excluded from
// every real-money and reporting path.
//
// Mode appears in two places with two different meanings. On ChefProfile it is
// configuration: which world the kitchen currently inhabits, freely flippable
// by an admin. On a transactional row it is a snapshot taken at creation and
// never changed, so a refund on an order paid in test mode still routes to the
// test gateway years after the chef went live.
const (
	ChefModeLive = "live"
	ChefModeTest = "test"
)

// NormalizeMode coerces any stored or user-supplied value to a known mode.
//
// Anything that is not recognisably "test" becomes "live". This asymmetry is
// deliberate and load-bearing: a corrupt or missing value must never hide a
// real kitchen from customers, and must never send a real payment through
// sandbox credentials (which would silently capture no money). The failure
// direction is always toward live.
func NormalizeMode(m string) string {
	if strings.EqualFold(strings.TrimSpace(m), ChefModeTest) {
		return ChefModeTest
	}
	return ChefModeLive
}

// IsTestMode reports whether a stored mode value means test.
func IsTestMode(m string) bool { return NormalizeMode(m) == ChefModeTest }

// ModePartition is embedded in every chef-scoped row that participates in the
// live/test split. GORM flattens anonymous embedded structs into the parent
// table, so these become ordinary `mode`, `test_session_id` and
// `cloned_from_id` columns — identical to writing them out per model, but
// impossible to get subtly wrong in one of seventeen places.
//
// Embedding rather than repeating also means a scope written against one
// partitioned table works unchanged against any other.
type ModePartition struct {
	// Mode is the data partition this row belongs to, snapshotted at creation
	// from the chef's mode and never changed afterwards. Money operations read
	// THIS field, not the chef's current mode — a chef flipped test→live must
	// still be able to refund an order that was paid with test credentials.
	Mode string `gorm:"type:varchar(4);not null;default:'live';index" json:"mode"`

	// TestSessionID ties a test row to the debugging session it belongs to, so
	// sessions can be listed and purged independently.
	TestSessionID *uuid.UUID `gorm:"type:uuid;index" json:"testSessionId,omitempty"`

	// ClonedFromID is set on rows produced by the live→test clone. A cloned row
	// is a historical replica belonging to a real customer who never placed it
	// in the sandbox, so it must never surface to any customer.
	ClonedFromID *uuid.UUID `gorm:"type:uuid;index" json:"-"`
}

// IsTest reports whether this row belongs to the test partition.
func (m ModePartition) IsTest() bool { return IsTestMode(m.Mode) }

// IsClone reports whether this row was produced by the live→test clone rather
// than by activity actually performed in the sandbox.
func (m ModePartition) IsClone() bool { return m.ClonedFromID != nil }
