package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A full flat-level query must yield progressively-broader fallbacks so the
// address search doesn't return an empty box (#address-search).
func TestPhotonQueryVariants(t *testing.T) {
	v := photonQueryVariants("Flat 104 Asima Residency Kalarahanga Bhubaneswar Odisha")
	require.Equal(t, "Flat 104 Asima Residency Kalarahanga Bhubaneswar Odisha", v[0], "full query first")
	// The leading flat number is stripped.
	require.Contains(t, v, "Asima Residency Kalarahanga Bhubaneswar Odisha")
	// A trailing city+state fallback that Photon can actually match.
	require.Contains(t, v, "Bhubaneswar Odisha")
	require.LessOrEqual(t, len(v), 5, "attempts are bounded for latency")

	// "No. 5, MG Road, Pune" → strips the house number, keeps the area.
	v2 := photonQueryVariants("No. 5 MG Road Pune Maharashtra")
	require.Contains(t, v2, "MG Road Pune Maharashtra")
	require.Contains(t, v2, "Pune Maharashtra")

	// A plain area query stays as-is (no needless extra calls beyond dedup).
	v3 := photonQueryVariants("Koramangala Bengaluru")
	require.Equal(t, "Koramangala Bengaluru", v3[0])
}
