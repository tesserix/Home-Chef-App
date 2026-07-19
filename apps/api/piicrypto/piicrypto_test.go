package piicrypto

import (
	"strings"
	"testing"
)

// activateForTest injects fixed keys so tests exercise the real AES-GCM / HMAC
// paths without needing KMS or Secret Manager.
func activateForTest(t *testing.T) {
	t.Helper()
	pii.mu.Lock()
	pii.dek = []byte("0123456789abcdef0123456789abcdef") // 32 bytes
	pii.bidxKey = []byte("blind-index-test-key")
	pii.active = true
	pii.mu.Unlock()
	t.Cleanup(func() {
		pii.mu.Lock()
		pii.dek, pii.bidxKey, pii.active = nil, nil, false
		pii.mu.Unlock()
	})
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	activateForTest(t)
	for _, pt := range []string{"9876543210", "chef@fe3dr.com", "Samyak Rout", ""} {
		ct, err := EncryptPII(pt)
		if err != nil {
			t.Fatalf("encrypt %q: %v", pt, err)
		}
		if pt != "" && !strings.HasPrefix(ct, cipherPrefix) {
			t.Errorf("ciphertext for %q missing prefix: %q", pt, ct)
		}
		if ct == pt && pt != "" {
			t.Errorf("value %q was not encrypted", pt)
		}
		got, err := DecryptPII(ct)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if got != pt {
			t.Errorf("roundtrip: got %q want %q", got, pt)
		}
	}
}

func TestEncryptNonDeterministic(t *testing.T) {
	activateForTest(t)
	a, _ := EncryptPII("9876543210")
	b, _ := EncryptPII("9876543210")
	if a == b {
		t.Error("two encryptions of the same value should differ (random nonce)")
	}
}

func TestDisabledIsPassthrough(t *testing.T) {
	// No activation → disabled.
	ct, err := EncryptPII("9876543210")
	if err != nil || ct != "9876543210" {
		t.Errorf("disabled encrypt should pass through, got %q err %v", ct, err)
	}
	// Plaintext (unprefixed) decrypts to itself even when disabled — migration safety.
	got, err := DecryptPII("9876543210")
	if err != nil || got != "9876543210" {
		t.Errorf("plaintext passthrough failed: %q err %v", got, err)
	}
}

func TestBlindIndexDeterministicAndNormalized(t *testing.T) {
	activateForTest(t)
	a := BlindIndex("9876543210")
	b := BlindIndex("9876543210")
	if a == "" || a != b {
		t.Errorf("blind index should be deterministic: %q vs %q", a, b)
	}
	// Normalization: case + surrounding space collapse to the same index.
	if BlindIndex("  Chef@Fe3dr.com ") != BlindIndex("chef@fe3dr.com") {
		t.Error("blind index should normalize case + whitespace")
	}
	if BlindIndex("9876543210") == BlindIndex("9876543211") {
		t.Error("different values must not collide")
	}
	if BlindIndex("") != "" {
		t.Error("empty value should give empty index")
	}
}
