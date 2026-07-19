// Package piicrypto provides application-level encryption for PII columns
// (#710). It lives in its own package — importing neither models nor services —
// so both can use it without an import cycle.
//
// Design (envelope encryption):
//   - A GCP KMS key (homechef-pii/pii-dek) WRAPS a 32-byte data-encryption-key
//     (DEK). Only the wrapped DEK is stored (Secret Manager); the raw DEK exists
//     only in memory after unwrap.
//   - Columns are encrypted with AES-256-GCM under the DEK.
//   - Searchable/unique fields additionally get a blind index:
//     HMAC-SHA256(bidxKey, normalized value) — so equality lookups and uniqueness
//     still work over ciphertext.
//
// Everything is gated behind PII_ENCRYPTION_ENABLED. When disabled (the default),
// EncryptPII/DecryptPII are safe pass-throughs, so introducing the EncryptedString
// type changes no behavior until the flag flips AND columns are migrated.
package piicrypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// cipherPrefix tags an encrypted value so DecryptPII can tell ciphertext from a
// not-yet-migrated plaintext value (both coexist during the backfill phase).
const cipherPrefix = "enc:v1:"

// KMS key coordinates (region + names are fixed; project is passed in).
const (
	kmsLocation = "asia-south1"
	kmsKeyRing  = "homechef-pii"
	kmsKey      = "pii-dek"

	wrappedDEKSecret = "prod-homechef-pii-dek-wrapped"
	blindIndexSecret = "prod-homechef-pii-bidx-key"
)

type state struct {
	mu      sync.RWMutex
	active  bool
	dek     []byte // raw 32-byte AES key
	bidxKey []byte
}

var pii state

// Active reports whether encryption is initialized and on.
func Active() bool {
	pii.mu.RLock()
	defer pii.mu.RUnlock()
	return pii.active
}

// Init loads and unwraps the DEK + blind-index key. Call once at startup ONLY
// when the feature flag is on; a disabled deployment never calls it and every
// primitive stays a pass-through. projectID is the GCP project holding the KMS
// key + secrets.
func Init(ctx context.Context, projectID string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	sm, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("piicrypto: secret manager: %w", err)
	}
	defer sm.Close()

	// Both secrets are stored base64-encoded (binary payloads round-trip through
	// Secret Manager cleanly as text), so decode after fetching.
	wrapped, err := accessSecretB64(ctx, sm, projectID, wrappedDEKSecret)
	if err != nil {
		return err
	}
	bidx, err := accessSecretB64(ctx, sm, projectID, blindIndexSecret)
	if err != nil {
		return err
	}

	kc, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return fmt.Errorf("piicrypto: kms client: %w", err)
	}
	defer kc.Close()

	keyName := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s",
		projectID, kmsLocation, kmsKeyRing, kmsKey)
	resp, err := kc.Decrypt(ctx, &kmspb.DecryptRequest{Name: keyName, Ciphertext: wrapped})
	if err != nil {
		return fmt.Errorf("piicrypto: unwrap dek: %w", err)
	}
	if len(resp.Plaintext) != 32 {
		return fmt.Errorf("piicrypto: dek must be 32 bytes, got %d", len(resp.Plaintext))
	}

	pii.mu.Lock()
	pii.dek = resp.Plaintext
	pii.bidxKey = bidx
	pii.active = true
	pii.mu.Unlock()
	return nil
}

// accessSecretB64 fetches a secret whose payload is base64-encoded text and
// returns the decoded bytes.
func accessSecretB64(ctx context.Context, sm *secretmanager.Client, projectID, id string) ([]byte, error) {
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, id)
	res, err := sm.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("piicrypto: access %s: %w", id, err)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(res.Payload.Data)))
	if err != nil {
		return nil, fmt.Errorf("piicrypto: decode %s: %w", id, err)
	}
	return raw, nil
}

// EncryptPII returns the encrypted, prefixed, base64 form of plaintext. When
// encryption is disabled it returns plaintext unchanged (safe no-op).
func EncryptPII(plaintext string) (string, error) {
	pii.mu.RLock()
	active, dek := pii.active, pii.dek
	pii.mu.RUnlock()
	if !active {
		return plaintext, nil
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return cipherPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptPII reverses EncryptPII. A value WITHOUT the cipher prefix is returned
// as-is — during migration a column holds a mix of plaintext and ciphertext.
func DecryptPII(value string) (string, error) {
	if !strings.HasPrefix(value, cipherPrefix) {
		return value, nil // not-yet-migrated plaintext
	}
	pii.mu.RLock()
	active, dek := pii.active, pii.dek
	pii.mu.RUnlock()
	if !active {
		return "", errors.New("piicrypto: encrypted value but crypto not initialized")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, cipherPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("piicrypto: ciphertext too short")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("piicrypto: decrypt: %w", err)
	}
	return string(pt), nil
}

// BlindIndex returns a deterministic HMAC of a normalized value for equality
// lookups / uniqueness over encrypted columns. Empty string when disabled or the
// input is empty (so an absent optional field doesn't collide on "").
func BlindIndex(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return ""
	}
	pii.mu.RLock()
	key := pii.bidxKey
	pii.mu.RUnlock()
	if len(key) == 0 {
		return ""
	}
	m := hmac.New(sha256.New, key)
	m.Write([]byte(v))
	return hex.EncodeToString(m.Sum(nil))
}
