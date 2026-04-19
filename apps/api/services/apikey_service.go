package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// apiKeyPrefix is the human-readable prefix on every generated key, making
// leaked credentials easy to spot in logs / secret scanners.
const apiKeyPrefix = "hc_"

// apiKeyEntropyBytes is the random material behind each key. 32 bytes = 64
// hex chars = plenty of entropy for an auth token.
const apiKeyEntropyBytes = 32

// GeneratedApiKey is returned by CreateApiKey. The FullKey is plaintext and
// is shown to the admin exactly once — we never store it ourselves.
type GeneratedApiKey struct {
	Record  *models.ApiKey
	FullKey string
}

// CreateApiKey generates a fresh API key, hashes it with SHA-256, and saves
// the hash + metadata. Returns the plaintext key so the handler can show it
// to the admin one time.
func CreateApiKey(name string, scopes []string, createdBy uuid.UUID, expiresAt *time.Time) (*GeneratedApiKey, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	raw := make([]byte, apiKeyEntropyBytes)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	full := apiKeyPrefix + hex.EncodeToString(raw)
	hash := hashApiKey(full)

	// The prefix column is what the admin sees in listings ("hc_abcd1234…").
	// 12 chars is enough to disambiguate but not enough to let someone
	// recover the full key.
	prefixDisplay := full
	if len(full) > 12 {
		prefixDisplay = full[:12]
	}

	scopeStr := ""
	for i, s := range scopes {
		if i > 0 {
			scopeStr += ","
		}
		scopeStr += s
	}

	record := &models.ApiKey{
		Name:      name,
		Prefix:    prefixDisplay,
		KeyHash:   hash,
		Scopes:    scopeStr,
		CreatedBy: createdBy,
		ExpiresAt: expiresAt,
	}
	if err := database.DB.Create(record).Error; err != nil {
		return nil, fmt.Errorf("save api key: %w", err)
	}

	return &GeneratedApiKey{Record: record, FullKey: full}, nil
}

// LookupApiKey finds a non-revoked, non-expired key matching the given
// plaintext credential. Updates LastUsedAt on hit (fire-and-forget).
func LookupApiKey(full string) (*models.ApiKey, error) {
	if !isLikelyApiKey(full) {
		return nil, fmt.Errorf("not an api key")
	}
	hash := hashApiKey(full)

	var key models.ApiKey
	if err := database.DB.Where("key_hash = ?", hash).First(&key).Error; err != nil {
		return nil, fmt.Errorf("api key not recognized")
	}
	if !key.IsActive(time.Now()) {
		return nil, fmt.Errorf("api key inactive")
	}

	// Best-effort LastUsedAt bump — don't block on it, don't fail the auth
	// if the UPDATE hits a transient error.
	go func(id uuid.UUID) {
		now := time.Now()
		database.DB.Model(&models.ApiKey{}).Where("id = ?", id).Update("last_used_at", now)
	}(key.ID)

	return &key, nil
}

// RevokeApiKey marks a key revoked. Idempotent — revoking twice is a no-op.
func RevokeApiKey(id uuid.UUID) error {
	now := time.Now()
	return database.DB.Model(&models.ApiKey{}).Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
}

// isLikelyApiKey is a cheap pre-check so we don't hash arbitrary strings.
func isLikelyApiKey(s string) bool {
	if len(s) < len(apiKeyPrefix)+16 {
		return false
	}
	return s[:len(apiKeyPrefix)] == apiKeyPrefix
}

func hashApiKey(full string) string {
	sum := sha256.Sum256([]byte(full))
	return hex.EncodeToString(sum[:])
}
