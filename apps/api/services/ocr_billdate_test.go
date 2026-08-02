package services

// ocr_billdate_test.go — the address-proof recency helpers: bill-date
// extraction from OCR text and the current-or-last-3-calendar-months window.

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestExtractBillDate_PicksLatestPastDate(t *testing.T) {
	recent := time.Now().AddDate(0, 0, -20)
	older := time.Now().AddDate(0, -8, 0)
	text := fmt.Sprintf(
		"STATE ELECTRICITY BOARD\nConsumer No 12345\nPrevious reading %s\nBill Date %s\nUnits consumed 214",
		older.Format("02/01/2006"), recent.Format("02/01/2006"))

	got, ok := ExtractBillDate(text)
	require.True(t, ok)
	require.Equal(t, recent.Format("2006-01-02"), got.Format("2006-01-02"))
}

func TestExtractBillDate_IgnoresFarFutureDates(t *testing.T) {
	// A licence-style "valid upto" years ahead must not read as the bill date.
	future := time.Now().AddDate(3, 0, 0)
	billed := time.Now().AddDate(0, -1, 0)
	text := fmt.Sprintf("Valid upto %s\nBill Date %s",
		future.Format("02/01/2006"), billed.Format("02/01/2006"))

	got, ok := ExtractBillDate(text)
	require.True(t, ok)
	require.Equal(t, billed.Format("2006-01-02"), got.Format("2006-01-02"))
}

func TestExtractBillDate_NoDates(t *testing.T) {
	_, ok := ExtractBillDate("ELECTRICITY BILL — no dates printed here")
	require.False(t, ok)
}

func TestAddressProofTooOld_Window(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)

	require.False(t, AddressProofTooOld(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), now), "current month passes")
	require.False(t, AddressProofTooOld(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), now), "3 months back passes")
	require.True(t, AddressProofTooOld(time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC), now), "4 months back fails")
	require.True(t, AddressProofTooOld(time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC), now), "years old fails")
}
