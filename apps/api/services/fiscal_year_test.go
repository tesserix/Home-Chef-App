package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFiscalCalendarWindow(t *testing.T) {
	for _, tc := range []struct {
		country    string
		start, end string
	}{
		{"IN", "2026-03-31T18:30:00Z", "2027-03-31T18:30:00Z"},
		{"AU", "2026-06-30T14:00:00Z", "2027-06-30T14:00:00Z"},
		{"NZ", "2026-03-31T11:00:00Z", "2027-03-31T11:00:00Z"},
	} {
		start, end := FiscalCalendarFor(tc.country).Window(2026)
		require.Equal(t, tc.start, start.Format(time.RFC3339), tc.country)
		require.Equal(t, tc.end, end.Format(time.RFC3339), tc.country)
	}
}

func TestFiscalCalendarCurrentStart(t *testing.T) {
	may := time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC)
	require.Equal(t, 2026, FiscalCalendarFor("IN").CurrentStart(may))
	require.Equal(t, 2025, FiscalCalendarFor("AU").CurrentStart(may), "May is still the AU year that began last July")
	require.Equal(t, 2026, FiscalCalendarFor("NZ").CurrentStart(may))

	// 30 Jun 15:00 UTC is already 1 Jul in Sydney.
	require.Equal(t, 2026, FiscalCalendarFor("AU").CurrentStart(time.Date(2026, time.June, 30, 15, 0, 0, 0, time.UTC)))
}

func TestFiscalCalendarParseDateAndMonth(t *testing.T) {
	cal := FiscalCalendarFor("NZ")
	day, err := cal.ParseDate("2026-07-01")
	require.NoError(t, err)
	require.Equal(t, "2026-06-30T12:00:00Z", day.Format(time.RFC3339))
	require.Equal(t, "2026-07", cal.Month(day))
	require.Equal(t, "2026-06", FiscalCalendarFor("IN").Month(day))
}

func TestFiscalCalendarUnknownFallsBackToIndia(t *testing.T) {
	a, b := FiscalCalendarFor("ZZ").Window(2026)
	c, d := FinancialYearWindow(2026)
	require.Equal(t, c, a)
	require.Equal(t, d, b)
}

func TestFiscalCalendarQuarters(t *testing.T) {
	au := FiscalCalendarFor("AU")
	require.Equal(t, []string{"Q1 (Jul–Sep)", "Q2 (Oct–Dec)", "Q3 (Jan–Mar)", "Q4 (Apr–Jun)"}, au.QuarterLabels())
	require.Equal(t, 0, au.Quarter(time.Date(2026, time.June, 30, 15, 0, 0, 0, time.UTC)), "1 Jul in Sydney")
	require.Equal(t, 3, au.Quarter(time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)))

	in := FiscalCalendarFor("IN")
	require.Equal(t, []string{"Q1 (Apr–Jun)", "Q2 (Jul–Sep)", "Q3 (Oct–Dec)", "Q4 (Jan–Mar)"}, in.QuarterLabels())
	for m := time.January; m <= time.December; m++ {
		ts := time.Date(2026, m, 10, 0, 0, 0, 0, time.UTC)
		require.Equal(t, financialQuarterIndex(ts), in.Quarter(ts), m.String())
	}
}

func TestReportingCurrency(t *testing.T) {
	require.Equal(t, "AUD", ReportingCurrency("au"))
	require.Equal(t, "NZD", ReportingCurrency("NZ"))
	require.Equal(t, "INR", ReportingCurrency("IN"))
	require.Equal(t, "INR", ReportingCurrency(""))
}

func TestFiscalCalendarPeriodLabel(t *testing.T) {
	require.Equal(t, "1 Jul 2025 – 30 Jun 2026", FiscalCalendarFor("AU").PeriodLabel(2025))
	require.Equal(t, "1 Apr 2025 – 31 Mar 2026", FiscalCalendarFor("NZ").PeriodLabel(2025))
	require.Equal(t, "1 Apr 2025 – 31 Mar 2026", FiscalCalendarFor("IN").PeriodLabel(2025))
}
