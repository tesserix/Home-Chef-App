package services

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
)

// withPinKey installs a signing key for the duration of one test. config.AppConfig
// is process-global, so it must be restored or later tests inherit it.
func withPinKey(t *testing.T, key string) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig = &config.Config{DeliverySurgePinKey: key, DeliverySurgeChargeEnabled: true}
	t.Cleanup(func() { config.AppConfig = prev })
}

func TestSurgePin_RoundTrips(t *testing.T) {
	withPinKey(t, "test-pin-key")
	chef := uuid.New()
	lat, lng := 12.9716, 77.5946

	pin := SignSurgePin(chef, lat, lng, 1.35)
	if pin == "" {
		t.Fatal("expected a pin for a surged quote")
	}
	got, ok := VerifySurgePin(pin, chef, lat, lng)
	if !ok {
		t.Fatal("expected the pin to verify")
	}
	if got != 1.35 {
		t.Fatalf("surge = %v, want 1.35", got)
	}
}

func TestSurgePin_NeutralSurgeIsNotPinned(t *testing.T) {
	withPinKey(t, "test-pin-key")
	// Nothing to hold the charge to when conditions are neutral.
	if pin := SignSurgePin(uuid.New(), 12.97, 77.59, 1.0); pin != "" {
		t.Fatalf("expected no pin for neutral surge, got %q", pin)
	}
}

func TestSurgePin_RejectsTampering(t *testing.T) {
	withPinKey(t, "test-pin-key")
	chef := uuid.New()
	pin := SignSurgePin(chef, 12.97, 77.59, 1.5)

	// Raising the multiplier inside the body must invalidate the signature —
	// otherwise a client could inflate its own delivery fee.
	tampered := "" + pin
	tampered = tampered[:len(tampered)-1] + "X"
	if _, ok := VerifySurgePin(tampered, chef, 12.97, 77.59); ok {
		t.Fatal("tampered pin verified")
	}
}

func TestSurgePin_RejectsOtherChefOrAddress(t *testing.T) {
	withPinKey(t, "test-pin-key")
	chef := uuid.New()
	pin := SignSurgePin(chef, 12.97, 77.59, 1.5)

	if _, ok := VerifySurgePin(pin, uuid.New(), 12.97, 77.59); ok {
		t.Fatal("pin verified for a different chef")
	}
	// A materially different drop must not reuse the pin.
	if _, ok := VerifySurgePin(pin, chef, 13.50, 77.59); ok {
		t.Fatal("pin verified for a different address")
	}
}

func TestSurgePin_RejectsExpired(t *testing.T) {
	withPinKey(t, "test-pin-key")
	chef := uuid.New()
	lat, lng := 12.97, 77.59

	// Sign with an already-past expiry via the same body construction.
	body := surgePinBody(chef, lat, lng, 1.4, time.Now().Add(-time.Minute).Unix())
	expired := body + "." + surgePinSignature(body)

	if _, ok := VerifySurgePin(expired, chef, lat, lng); ok {
		t.Fatal("expired pin verified")
	}
}

func TestSurgePin_RejectsForeignKey(t *testing.T) {
	withPinKey(t, "key-one")
	chef := uuid.New()
	pin := SignSurgePin(chef, 12.97, 77.59, 1.5)

	withPinKey(t, "key-two")
	if _, ok := VerifySurgePin(pin, chef, 12.97, 77.59); ok {
		t.Fatal("pin signed with another key verified")
	}
}

func TestSurgeChargeEnabled_RequiresPinKey(t *testing.T) {
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })

	// Flag on but no key: charging a multiplier we cannot pin would reintroduce
	// exactly the quote/charge drift the pin exists to prevent.
	config.AppConfig = &config.Config{DeliverySurgeChargeEnabled: true}
	if SurgeChargeEnabled() {
		t.Fatal("surge charging enabled without a pin key")
	}

	config.AppConfig = &config.Config{DeliverySurgeChargeEnabled: true, DeliverySurgePinKey: "k"}
	if !SurgeChargeEnabled() {
		t.Fatal("surge charging disabled despite flag + key")
	}
}
