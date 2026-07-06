// Package money is the integer minor-unit (paise) foundation for #396 Phase 2 — the
// migration of all money math off float64 to exact integer paise, so the escrow
// invariant "captured = released + refunded + retained" holds at the paise level with
// NO epsilon tolerance.
//
// It deliberately depends on nothing in the rest of the codebase (so `services` and
// `handlers` can import it without a cycle) and holds ONLY pure arithmetic — no DB, no
// gateway. Slice 1 introduces the type + helpers + the largest-remainder allocator and
// routes services.ToPaise/FromPaise through FromRupees/Rupees (behaviour-preserving).
// Later slices add paise columns (dual-write → backfill → flip reads) per aggregate.
package money

import "math"

// Paise is a monetary amount in integer minor units (1 rupee = 100 paise). int64 so a
// realistic marketplace sum (crores of rupees) never overflows, and an intermediate
// total×weight product in Allocate (≤ ~10^14 for real amounts) stays well inside int64.
type Paise int64

// FromRupees converts a rupee float to paise, rounding half-away-from-zero — the SAME
// convention as services.ToMinor(_, "INR"), so every gateway/ledger path mints the
// identical minor-unit value. Round (not truncate) fixes the IEEE-754 near-integer loss
// (0.29*100 == 28.999… → 28); see #524/#396.
func FromRupees(r float64) Paise {
	return Paise(math.Round(r * 100))
}

// Rupees renders paise back to a rupee float for display / legacy float arithmetic.
func (p Paise) Rupees() float64 {
	return float64(p) / 100.0
}

// Sum totals a slice of paise amounts (exact — integer addition).
func Sum(parts []Paise) Paise {
	var t Paise
	for _, p := range parts {
		t += p
	}
	return t
}

// Allocate splits `total` paise across len(weights) parts in proportion to `weights`,
// using the largest-remainder (Hamilton) method so the parts ALWAYS sum EXACTLY to
// `total` — the property float proportional splits can't guarantee (three ways of a
// ₹10.00 order would leave a stray paise). Each part gets floor(total*wᵢ/Σw); the
// leftover paise (0 ≤ leftover < n) go one each to the parts with the largest
// fractional remainder, ties broken by lowest index (deterministic).
//
// Edge cases: nil/empty weights → nil. A non-positive Σw (all-zero or negative weights)
// → the total is split as evenly as possible (equal weights), largest-remainder giving
// the first `leftover` parts the extra paise. A negative total is allocated with the
// same rule (remainders compared by magnitude), so a claw-back split also conserves.
func Allocate(total Paise, weights []Paise) []Paise {
	n := len(weights)
	if n == 0 {
		return nil
	}

	// Normalise weights: use them as given when Σ>0, else fall back to equal weights so
	// an all-zero weight vector still produces a conserving even split.
	var sumW int64
	for _, w := range weights {
		if w > 0 {
			sumW += int64(w)
		}
	}
	norm := make([]int64, n)
	if sumW > 0 {
		for i, w := range weights {
			if w > 0 {
				norm[i] = int64(w)
			} // negative/zero weights get 0 share of a positive pool
		}
	} else {
		for i := range norm {
			norm[i] = 1
		}
		sumW = int64(n)
	}

	out := make([]Paise, n)
	rem := make([]int64, n) // fractional remainder numerator for each part
	var allocated int64     // sum of the floors
	t := int64(total)
	for i := range norm {
		// product = t * norm[i]; floor division toward negative infinity is NOT what we
		// want for negatives — use Go's truncated division and keep the remainder sign
		// consistent so allocated + leftover == t regardless of sign.
		q := t * norm[i] / sumW
		r := t*norm[i] - q*sumW // == (t*norm[i]) % sumW, sign follows the dividend
		out[i] = Paise(q)
		rem[i] = r
		allocated += q
	}

	leftover := t - allocated // paise still to distribute; sign matches `total`
	step := int64(1)
	if leftover < 0 {
		step = -1
		leftover = -leftover
	}
	// Hand out `leftover` single-paise steps to the largest remainders (by magnitude),
	// ties → lowest index, so the split is deterministic and conserving.
	used := make([]bool, n)
	for k := int64(0); k < leftover; k++ {
		best := -1
		var bestRem int64
		for i := 0; i < n; i++ {
			if used[i] {
				continue
			}
			mag := rem[i]
			if mag < 0 {
				mag = -mag
			}
			if best == -1 || mag > bestRem {
				best, bestRem = i, mag
			}
		}
		if best == -1 {
			break // more leftover than parts shouldn't happen (leftover < n), guard anyway
		}
		out[best] += Paise(step)
		used[best] = true
	}
	return out
}
