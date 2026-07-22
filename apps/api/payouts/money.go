// Package payouts is the domain-free core of the vendor payout engine.
//
// It knows about payees, ledgers, batches and rules. It knows nothing about
// chefs, orders, meal plans or food. That boundary is enforced by
// TestNoDomainImports in boundary_test.go, and it is what keeps the eventual
// extraction to go-shared/payouts (#749) a module-path change rather than a
// rewrite.
//
// All amounts are integer minor units (paise for INR). The package never uses
// float64 — callers convert at the boundary until #396 migrates the rest of
// the codebase off floats.
package payouts

import (
	"fmt"
	"strconv"
)

// Currency is an ISO-4217 code. The engine is single-currency today but the
// ledger is keyed by currency so a second one does not require a migration.
type Currency string

const CurrencyINR Currency = "INR"

// minorUnitsPerMajor is the scale for every currency the engine supports.
// Currencies with a different exponent (JPY, KWD) would need this to become a
// per-currency lookup; asserting it here keeps that decision visible.
const minorUnitsPerMajor = 100

// Money is an exact amount in minor units. The zero value is a currency-less
// zero, which Add and Sub accept as an identity so accumulator loops work
// without seeding a currency.
type Money struct {
	Minor    int64    `json:"minor"`
	Currency Currency `json:"currency"`
}

// Zero returns a typed zero in the given currency.
func Zero(c Currency) Money { return Money{Minor: 0, Currency: c} }

// FromMinor builds Money from minor units.
func FromMinor(minor int64, c Currency) Money { return Money{Minor: minor, Currency: c} }

// IsZero reports whether the amount is exactly zero, regardless of currency.
func (m Money) IsZero() bool { return m.Minor == 0 }

// IsNegative reports whether the amount is below zero.
func (m Money) IsNegative() bool { return m.Minor < 0 }

// IsPositive reports whether the amount is above zero.
func (m Money) IsPositive() bool { return m.Minor > 0 }

// Neg returns the additive inverse, preserving currency.
func (m Money) Neg() Money { return Money{Minor: -m.Minor, Currency: m.Currency} }

// ClampAtZero floors the amount at zero. Used where a computed deduction must
// never turn into a payment to the platform.
func (m Money) ClampAtZero() Money {
	if m.Minor < 0 {
		return Money{Minor: 0, Currency: m.Currency}
	}
	return m
}

// compatible resolves the currency of a binary operation, tolerating a
// currency-less zero on either side.
func compatible(a, b Money) (Currency, error) {
	switch {
	case a.Currency == b.Currency:
		return a.Currency, nil
	case a.Currency == "" && a.Minor == 0:
		return b.Currency, nil
	case b.Currency == "" && b.Minor == 0:
		return a.Currency, nil
	default:
		return "", fmt.Errorf("payouts: currency mismatch %q vs %q", a.Currency, b.Currency)
	}
}

// Add returns a+b, erroring if the currencies differ.
func (m Money) Add(other Money) (Money, error) {
	c, err := compatible(m, other)
	if err != nil {
		return Money{}, err
	}
	return Money{Minor: m.Minor + other.Minor, Currency: c}, nil
}

// Sub returns a-b, erroring if the currencies differ.
func (m Money) Sub(other Money) (Money, error) {
	c, err := compatible(m, other)
	if err != nil {
		return Money{}, err
	}
	return Money{Minor: m.Minor - other.Minor, Currency: c}, nil
}

// Cmp returns -1, 0 or 1 as m is less than, equal to, or greater than other.
func (m Money) Cmp(other Money) (int, error) {
	if _, err := compatible(m, other); err != nil {
		return 0, err
	}
	switch {
	case m.Minor < other.Minor:
		return -1, nil
	case m.Minor > other.Minor:
		return 1, nil
	default:
		return 0, nil
	}
}

// ApplyBasisPoints returns bps/10000 of the amount, rounded half away from
// zero. Basis points keep percentage settings (commission, reserve, caps)
// exact integers, so a 6% rate is 600 and never 0.06.
//
// Half away from zero is chosen over banker's rounding because it is the
// convention Indian tax and settlement documents use, and because it is the
// rounding the existing services/earnings.go math already implies.
func (m Money) ApplyBasisPoints(bps int64) Money {
	const scale = 10000
	product := m.Minor * bps
	q := product / scale
	rem := product % scale
	if rem != 0 {
		// 2*|rem| >= scale means the fraction is at or past one half.
		if abs64(rem)*2 >= scale {
			if product < 0 {
				q--
			} else {
				q++
			}
		}
	}
	return Money{Minor: q, Currency: m.Currency}
}

// Sum adds any number of amounts. Summing nothing yields a currency-less zero.
func Sum(amounts ...Money) (Money, error) {
	total := Money{}
	for _, a := range amounts {
		var err error
		if total, err = total.Add(a); err != nil {
			return Money{}, err
		}
	}
	return total, nil
}

// String renders the amount in major units for logs and admin surfaces. It is
// deliberately not a formatting helper for user-facing UI, which localises.
func (m Money) String() string {
	sign := ""
	minor := m.Minor
	if minor < 0 {
		sign = "-"
		minor = -minor
	}
	major := minor / minorUnitsPerMajor
	frac := minor % minorUnitsPerMajor
	cur := m.Currency
	if cur == "" {
		cur = "???"
	}
	return sign + strconv.FormatInt(major, 10) + "." +
		fmt.Sprintf("%02d", frac) + " " + string(cur)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
