package services

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// A chef stores its state as a full name, an address geocoded by Mappls/Photon
// stores the ISO code, and both are correct. These pin that the two spellings
// resolve to the same state — because when they didn't, the customer feed
// returned zero kitchens and a same-state order was invoiced as IGST.

func withSeededStates(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE states (
		id text PRIMARY KEY, country_code text, code text, name text, type text,
		created_at datetime, updated_at datetime)`).Error)
	for _, s := range [][3]string{
		{"IN-OR", "OR", "Odisha"},
		{"IN-MH", "MH", "Maharashtra"},
		{"IN-KA", "KA", "Karnataka"},
	} {
		require.NoError(t, db.Exec(
			`INSERT INTO states (id, country_code, code, name, type) VALUES (?,?,?,?,'state')`,
			s[0], "IN", s[1], s[2]).Error)
	}

	orig := database.DB
	database.DB = db
	InvalidateStateAliases()
	t.Cleanup(func() {
		database.DB = orig
		InvalidateStateAliases()
	})
}

func TestCanonicalState_CodeAndNameAgree(t *testing.T) {
	withSeededStates(t)

	// The exact pair that broke production: chef "Odisha", address "OR".
	require.Equal(t, CanonicalState("Odisha"), CanonicalState("OR"))
	require.True(t, SameState("Odisha", "OR"))
	require.True(t, SameState("  odisha ", "or"))

	require.False(t, SameState("Odisha", "Maharashtra"))
	require.False(t, SameState("OR", "MH"))
}

func TestSameState_BlankIsNeverAMatch(t *testing.T) {
	withSeededStates(t)
	require.False(t, SameState("", "OR"))
	require.False(t, SameState("OR", ""))
	require.False(t, SameState("", ""))
}

// An unseeded state must still compare equal to an identical spelling. Falling
// back to "" would make every unknown state match every other one.
func TestCanonicalState_UnknownFallsBackToItself(t *testing.T) {
	withSeededStates(t)
	require.True(t, SameState("Atlantis", "atlantis "))
	require.False(t, SameState("Atlantis", "Narnia"))
}

func TestStateMatchValues_CoversBothSpellings(t *testing.T) {
	withSeededStates(t)

	// Whichever spelling the customer's address holds, the SQL filter must match
	// a chef row written the other way.
	for _, in := range []string{"OR", "Odisha", " odisha "} {
		vals := StateMatchValues(in)
		require.Contains(t, vals, "or", "input %q must match a chef row storing the code", in)
		require.Contains(t, vals, "odisha", "input %q must match a chef row storing the name", in)
	}

	require.Nil(t, StateMatchValues(""), "no state means no filter, not an empty IN () that matches nothing")
	require.Equal(t, []string{"atlantis"}, StateMatchValues("Atlantis"))
}

// The GST split is the money-facing consumer of this. A Bhubaneswar chef and a
// Bhubaneswar customer are ONE state however each side spells it, so the invoice
// must show CGST+SGST — not IGST, which is what shipped.
func TestIsIntraStateSupply_ResolvesCodeAgainstName(t *testing.T) {
	withSeededStates(t)

	require.True(t, IsIntraStateSupply("Odisha", "OR"))
	require.True(t, IsIntraStateSupply("OR", "Odisha"))

	lines := gstLines(t, 14.18, 5, "Odisha", "OR")
	require.Len(t, lines, 2, "same state, different spelling → CGST+SGST")
	require.InDelta(t, 14.18, lines[0].Amount+lines[1].Amount, 1e-9)

	// A genuine inter-state supply is still IGST.
	inter := gstLines(t, 14.18, 5, "Odisha", "MH")
	require.Len(t, inter, 1)
	require.Equal(t, models.TaxLineIGST, inter[0].Code)
	require.InDelta(t, 14.18, inter[0].Amount, 1e-9)
}

// Blank on either side keeps defaulting to intra — the safe case for a home
// kitchen, and unchanged by this fix.
func TestIsIntraStateSupply_BlankStillDefaultsToIntra(t *testing.T) {
	withSeededStates(t)
	require.True(t, IsIntraStateSupply("", "OR"))
	require.True(t, IsIntraStateSupply("Odisha", ""))
}
