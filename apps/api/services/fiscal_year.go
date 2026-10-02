package services

import (
	"fmt"
	"strings"
	"time"
	// Embedded so AU/NZ zones resolve even in an image without system tzdata.
	_ "time/tzdata"

	"github.com/homechef/api/internal/markets"
)

// FiscalCalendar is a market's tax year: the zone its days are drawn in and the
// month it starts. India runs Apr–Mar, Australia Jul–Jun, New Zealand Apr–Mar.
type FiscalCalendar struct {
	Loc        *time.Location
	StartMonth time.Month
}

// FiscalCalendarFor returns the kitchen country's tax year, India when unknown.
func FiscalCalendarFor(country string) FiscalCalendar {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "AU":
		return FiscalCalendar{Loc: mustLoadLocation("Australia/Sydney"), StartMonth: time.July}
	case "NZ":
		return FiscalCalendar{Loc: mustLoadLocation("Pacific/Auckland"), StartMonth: time.April}
	default:
		return FiscalCalendar{Loc: istLoc, StartMonth: time.April}
	}
}

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// Window is the [start, end) UTC range of the tax year beginning in startYear.
func (f FiscalCalendar) Window(startYear int) (time.Time, time.Time) {
	start := time.Date(startYear, f.StartMonth, 1, 0, 0, 0, 0, f.Loc)
	end := time.Date(startYear+1, f.StartMonth, 1, 0, 0, 0, 0, f.Loc)
	return start.UTC(), end.UTC()
}

// CurrentStart is the start year of the tax year containing now.
func (f FiscalCalendar) CurrentStart(now time.Time) int {
	local := now.In(f.Loc)
	if local.Month() < f.StartMonth {
		return local.Year() - 1
	}
	return local.Year()
}

// ParseDate parses YYYY-MM-DD as a local calendar day, returned in UTC.
func (f FiscalCalendar) ParseDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, f.Loc)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// Month is the local YYYY-MM that t falls in.
func (f FiscalCalendar) Month(t time.Time) string {
	return t.In(f.Loc).Format("2006-01")
}

// ReportingCurrency is the ISO code statements are kept in, INR when unknown.
func ReportingCurrency(country string) string {
	if m, ok := markets.Lookup(country); ok {
		return m.Currency
	}
	return EarningsCurrency
}

// Quarter is t's 0-based quarter within its tax year.
func (f FiscalCalendar) Quarter(t time.Time) int {
	return (int(t.In(f.Loc).Month()) - int(f.StartMonth) + 12) % 12 / 3
}

// QuarterLabels names the four quarters, e.g. "Q1 (Jul–Sep)".
func (f FiscalCalendar) QuarterLabels() []string {
	labels := make([]string, 4)
	for q := range labels {
		first := time.Month((int(f.StartMonth)-1+3*q)%12 + 1)
		last := time.Month((int(first)-1+2)%12 + 1)
		labels[q] = fmt.Sprintf("Q%d (%s–%s)", q+1, first.String()[:3], last.String()[:3])
	}
	return labels
}

// PeriodLabel renders the tax year as "1 Jul 2025 – 30 Jun 2026".
func (f FiscalCalendar) PeriodLabel(startYear int) string {
	first := time.Date(startYear, f.StartMonth, 1, 0, 0, 0, 0, f.Loc)
	last := first.AddDate(1, 0, -1)
	return first.Format("2 Jan 2006") + " – " + last.Format("2 Jan 2006")
}
