package services

import (
	"sync"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// state_resolve.go — one canonical answer to "are these two states the same?".
//
// Nothing guarantees the two sides of a state comparison are written the same
// way. A chef profile is filled in from the vendor onboarding form and stores
// the full name ("Odisha"); a customer address is filled from the geocoder,
// which returns whatever the provider hands back — for Mappls/Photon that is
// frequently the ISO code ("OR"). Both are correct; neither is wrong; they are
// simply not equal as strings.
//
// Two systems compared them as strings anyway, and both broke:
//
//   - The chef feed's region gate (LOWER(state) = LOWER(?)) returned ZERO
//     kitchens for a customer whose address said "OR" while the only kitchen in
//     that state said "Odisha".
//   - IsIntraStateSupply classified a Bhubaneswar → Bhubaneswar order as
//     INTER-state and put IGST on the invoice where CGST+SGST was due.
//
// The mapping to resolve it was already in the database — seed_locations.go
// seeds `states` with {Code: "OR", Name: "Odisha"} for all 36 Indian states and
// territories. It just wasn't being used for comparison.

// Hand the resolver to models so a DTO can decide place of supply without models
// importing services, which would be an import cycle. See models/pricing.go.
func init() { models.SameStateFunc = SameState }

// stateAliases maps every known spelling (lowercased code AND lowercased name)
// to a single canonical key per state — the state's ID, e.g. "IN-OR".
var (
	stateAliasMu        sync.RWMutex
	stateAliasCache     map[string]string
	stateAliasFetchedAt time.Time
)

// loadStateAliases builds the alias map from the seeded states table.
//
// A nil map (no DB, empty table) is cached like any other result: the caller
// degrades to plain string comparison, which is exactly today's behaviour, so a
// missing seed can never be worse than not having this at all.
func loadStateAliases() map[string]string {
	aliases := map[string]string{}
	if database.DB == nil {
		return aliases
	}
	var states []models.State
	if err := database.DB.Find(&states).Error; err != nil {
		return aliases
	}
	for _, s := range states {
		key := s.ID
		if key == "" {
			key = s.CountryCode + "-" + s.Code
		}
		if c := normalizeState(s.Code); c != "" {
			aliases[c] = key
		}
		if n := normalizeState(s.Name); n != "" {
			aliases[n] = key
		}
	}
	return aliases
}

func stateAliases() map[string]string {
	stateAliasMu.RLock()
	if stateAliasCache != nil && time.Since(stateAliasFetchedAt) < platformConfigTTL {
		defer stateAliasMu.RUnlock()
		return stateAliasCache
	}
	stateAliasMu.RUnlock()

	stateAliasMu.Lock()
	defer stateAliasMu.Unlock()
	if stateAliasCache != nil && time.Since(stateAliasFetchedAt) < platformConfigTTL {
		return stateAliasCache
	}
	stateAliasCache = loadStateAliases()
	stateAliasFetchedAt = time.Now()
	return stateAliasCache
}

// InvalidateStateAliases drops the cache. Call after seeding or editing states.
func InvalidateStateAliases() {
	stateAliasMu.Lock()
	defer stateAliasMu.Unlock()
	stateAliasCache = nil
}

// CanonicalState resolves a state written as either a code ("OR") or a name
// ("Odisha"), in any case or padding, to a single stable key ("IN-OR").
//
// An unrecognised value returns its normalized self rather than "", so a state
// we've never seeded still compares equal to an identical spelling instead of
// collapsing to match everything.
func CanonicalState(s string) string {
	n := normalizeState(s)
	if n == "" {
		return ""
	}
	if key, ok := stateAliases()[n]; ok {
		return key
	}
	return n
}

// SameState reports whether two state strings name the same state, whichever
// spelling each side happens to use. Blank on either side is "unknown", never a
// match — callers decide what unknown means for them.
func SameState(a, b string) bool {
	ca, cb := CanonicalState(a), CanonicalState(b)
	if ca == "" || cb == "" {
		return false
	}
	return ca == cb
}

// StateMatchValues returns every spelling that identifies the same state as the
// input — its code and its name — for use in a SQL `IN` filter.
//
// Returned lowercased to pair with a LOWER(column) comparison, and always
// includes the caller's own normalized input so an unseeded state still matches
// itself.
func StateMatchValues(s string) []string {
	n := normalizeState(s)
	if n == "" {
		return nil
	}
	canonical, ok := stateAliases()[n]
	if !ok {
		return []string{n}
	}
	out := []string{}
	seen := map[string]bool{}
	for alias, key := range stateAliases() {
		if key == canonical && !seen[alias] {
			seen[alias] = true
			out = append(out, alias)
		}
	}
	if !seen[n] {
		out = append(out, n)
	}
	// Deterministic order keeps the generated SQL (and its query plan cache)
	// stable across requests.
	sortStrings(out)
	return out
}

// sortStrings is a tiny insertion sort — the slice is at most a handful of
// aliases, so pulling in sort for it would be heavier than the work itself.
func sortStrings(v []string) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}
