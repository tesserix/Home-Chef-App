package services

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewDeviceLoginHTML_ShowsWhatTheUserNeedsToJudgeIt(t *testing.T) {
	when := time.Date(2026, 8, 8, 14, 5, 0, 0, time.UTC)
	subject, body := NewDeviceLoginHTML("Asha", NewDeviceLogin{
		App:      "customer",
		Platform: "ios",
		Label:    "iPhone 17 Pro",
		Location: IPLocation{City: "Bengaluru", Country: "India"},
		IP:       "49.207.1.1",
		DeviceID: "3F2C9A11-77D4-4B0E-9E42-9C1A55C0FFEE",
		Signed:   when,
	})

	require.Contains(t, subject, "New sign-in")
	for _, want := range []string{"Asha", "iPhone 17 Pro", "Bengaluru, India", "49.207.1.1", "8 Aug 2026"} {
		require.Contains(t, body, want)
	}
	require.Contains(t, body, "https://fe3dr.com/forgot-password", "the user needs a way to act on a sign-in they don't recognize")
}

// The device id is an opaque install identifier; printing it whole invites
// someone to copy it, and it tells the user nothing. Show only the tail.
func TestNewDeviceLoginHTML_TruncatesTheDeviceID(t *testing.T) {
	_, body := NewDeviceLoginHTML("", NewDeviceLogin{
		DeviceID: "3F2C9A11-77D4-4B0E-9E42-9C1A55C0FFEE",
		Signed:   time.Now(),
	})

	require.NotContains(t, body, "3F2C9A11-77D4-4B0E-9E42-9C1A55C0FFEE")
	require.Contains(t, body, "C0FFEE")
}

func TestNewDeviceLoginHTML_FallsBackWhenNothingIsKnown(t *testing.T) {
	_, body := NewDeviceLoginHTML("", NewDeviceLogin{Signed: time.Now()})

	require.Contains(t, body, "Unknown location")
	require.Contains(t, body, "there", "a nameless account still gets a greeting")
}

// These templates are assembled with Sprintf, so a display name carried over
// from a social profile must not be able to inject markup (audit #7).
func TestNewDeviceLoginHTML_EscapesUserControlledText(t *testing.T) {
	_, body := NewDeviceLoginHTML(`<script>x</script>`, NewDeviceLogin{
		Label:  `<img src=x onerror=alert(1)>`,
		Signed: time.Now(),
	})

	require.False(t, strings.Contains(body, "<script>x</script>"))
	require.False(t, strings.Contains(body, "<img src=x"))
}
