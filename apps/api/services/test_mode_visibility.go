package services

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ChefVisibilityDecision is how much of a kitchen a given viewer may see.
type ChefVisibilityDecision int

const (
	// VisibilityFull — the kitchen renders normally: listed, browsable and
	// orderable, subject to the usual open/closed and FSSAI rules.
	VisibilityFull ChefVisibilityDecision = iota

	// VisibilityClosed — the kitchen is listed by name and shown as closed, with
	// no menu, no prices and no ordering. Used for an established kitchen that
	// has been flipped to test for debugging: its regulars would read a sudden
	// disappearance as "they shut down", which is worse and less true than
	// "closed today".
	VisibilityClosed

	// VisibilityHidden — the kitchen does not exist for this viewer. Used for
	// born-test kitchens, which no real customer has ever heard of.
	VisibilityHidden
)

// ChefVisibility decides how much of a kitchen a viewer may see.
// viewerEmail is empty for anonymous callers, who never see a test kitchen.
func ChefVisibility(chef *models.ChefProfile, viewerEmail string) ChefVisibilityDecision {
	if chef == nil || !chef.IsTestMode() {
		return VisibilityFull
	}
	if GetTestModePolicy().MayViewTestChefs(viewerEmail) {
		return VisibilityFull
	}
	if chef.IsBornTest() {
		return VisibilityHidden
	}
	return VisibilityClosed
}

// TestChefVisibility is the set-based mirror of ChefVisibility for
// chef_profiles list queries — the same rule expressed in SQL, so pagination
// and counts stay correct instead of post-filtering a page down to three rows.
//
// Allowlisted viewers get no filter at all. For everyone else, born-test
// kitchens are removed in SQL; flipped kitchens deliberately survive the query
// and are reduced to their closed presentation by the response mapper, which is
// where per-row policy belongs.
func TestChefVisibility(viewerEmail string) func(*gorm.DB) *gorm.DB {
	allowed := GetTestModePolicy().MayViewTestChefs(viewerEmail)
	return func(db *gorm.DB) *gorm.DB {
		if allowed {
			return db
		}
		return db.Where("NOT (mode = ? AND first_live_at IS NULL)", models.ChefModeTest)
	}
}

// ModeScope filters any partitioned table to one mode.
func ModeScope(mode string) func(*gorm.DB) *gorm.DB {
	m := models.NormalizeMode(mode)
	return func(db *gorm.DB) *gorm.DB { return db.Where("mode = ?", m) }
}

// ExcludeTestOrders removes test-partition rows from a query.
//
// Applied to every real-money and reporting path — revenue analytics, chef
// earnings, GST/TDS, reconciliation, the ledger, payout selection. A fake order
// must never move a real number or reach a real bank.
func ExcludeTestOrders(db *gorm.DB) *gorm.DB {
	return db.Where("mode = ?", models.ChefModeLive)
}

// ExcludeClonedRows removes rows produced by the live→test clone.
//
// A cloned order is a replica of a REAL customer's order, copied into the
// sandbox for debugging. That customer never placed it there, so it must never
// appear in their order history — this is a hard rule at the query layer, not a
// UI concern.
func ExcludeClonedRows(db *gorm.DB) *gorm.DB {
	return db.Where("cloned_from_id IS NULL")
}

// CustomerVisibleModes scopes a customer-facing transactional query.
//
// Customers see live rows, plus test rows they placed themselves while on the
// viewer allowlist. Cloned rows are excluded unconditionally.
func CustomerVisibleModes(viewerEmail string) func(*gorm.DB) *gorm.DB {
	allowed := GetTestModePolicy().MayViewTestChefs(viewerEmail)
	return func(db *gorm.DB) *gorm.DB {
		db = db.Where("cloned_from_id IS NULL")
		if allowed {
			return db
		}
		return db.Where("mode = ?", models.ChefModeLive)
	}
}

// ChefOwnModeScope filters a chef-scoped table to the world the kitchen is
// CURRENTLY in, expressed as a correlated subquery so it costs no extra round
// trip and can never race a flip that happens mid-request.
//
// This is what makes the sandbox feel like a clean slate to the chef. While a
// kitchen is in test mode its dashboard, order queue, earnings and menu must
// show the sandbox's own rows and nothing else; the moment it returns to live,
// its real data reappears exactly as it was. Without this the chef would see
// live and sandbox work interleaved — which is both confusing and dangerous,
// since a chef could act on a real order believing it was a test one.
//
// Note it reads the CHEF's current mode, unlike the money paths which read each
// record's own snapshotted mode. That difference is deliberate: this is a view
// concern, not a settlement concern.
func ChefOwnModeScope(chefID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(
			"mode = COALESCE((SELECT mode FROM chef_profiles WHERE id = ?), ?)",
			chefID, models.ChefModeLive)
	}
}
