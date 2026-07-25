package services

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// testModePolicyKey is the PlatformSettings row we store the policy blob in.
const testModePolicyKey = "test_mode_policy"

// TestModePolicy controls who can see test-mode kitchens.
//
// Test kitchens live on production alongside real ones. The only thing standing
// between a real customer and a sandbox kitchen is this list, so it is
// deliberately an explicit allowlist: no wildcards, no domain matching, no
// "all admins" shortcut. Anything looser turns a config mistake into a real
// customer placing an order that will never be cooked.
type TestModePolicy struct {
	// ViewerEmails may browse, open and order from test-mode kitchens.
	// Matched case-insensitively with surrounding whitespace trimmed.
	ViewerEmails []string `json:"viewerEmails"`
}

// DefaultTestModePolicy seeds the three platform testers so the feature works
// the moment it deploys, before anyone opens the admin UI.
//
// This differs deliberately from DefaultSecurityPolicy's exempt list, which is
// intentionally empty because it bypasses 2FA and was once a weak-default
// credential vector. Membership here grants no privilege over real data — only
// visibility of fake data — so a working default is safe and saves a
// chicken-and-egg setup step.
func DefaultTestModePolicy() TestModePolicy {
	return TestModePolicy{ViewerEmails: []string{
		"samyak.rout@gmail.com",
		"unidevidp@gmail.com",
		"mahesh.sangawar@gmail.com",
	}}
}

// MayViewTestChefs reports whether this email may see test-mode kitchens.
//
// An empty email (an anonymous caller) is always false, and an empty entry in
// the list can never match it — otherwise a stray blank line in the admin UI
// would silently expose every sandbox kitchen to every logged-out visitor.
func (p TestModePolicy) MayViewTestChefs(email string) bool {
	e := strings.TrimSpace(email)
	if e == "" {
		return false
	}
	for _, v := range p.ViewerEmails {
		if v := strings.TrimSpace(v); v != "" && strings.EqualFold(v, e) {
			return true
		}
	}
	return false
}

var (
	testModeCache     *TestModePolicy
	testModeFetchedAt time.Time
	testModeMu        sync.RWMutex
)

// GetTestModePolicy returns the current policy, cached for platformConfigTTL.
// A missing or malformed row falls back to DefaultTestModePolicy() so the
// policy is always a valid value.
func GetTestModePolicy() TestModePolicy {
	testModeMu.RLock()
	if testModeCache != nil && time.Since(testModeFetchedAt) < platformConfigTTL {
		defer testModeMu.RUnlock()
		return *testModeCache
	}
	testModeMu.RUnlock()

	testModeMu.Lock()
	defer testModeMu.Unlock()
	// Double-check after acquiring the write lock.
	if testModeCache != nil && time.Since(testModeFetchedAt) < platformConfigTTL {
		return *testModeCache
	}

	fresh := loadTestModePolicyFromDB()
	testModeCache = &fresh
	testModeFetchedAt = time.Now()
	return fresh
}

// InvalidateTestModePolicy drops the cache so the next read refetches from DB.
// Call after writing the policy via the admin API.
func InvalidateTestModePolicy() {
	testModeMu.Lock()
	defer testModeMu.Unlock()
	testModeCache = nil
}

// SaveTestModePolicy persists the policy as a single JSON blob in
// PlatformSettings under testModePolicyKey and invalidates the cache.
func SaveTestModePolicy(p TestModePolicy, updatedBy *uuid.UUID) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}

	var setting models.PlatformSettings
	err = database.DB.Where("key = ?", testModePolicyKey).First(&setting).Error
	if err != nil {
		setting = models.PlatformSettings{
			Key:       testModePolicyKey,
			Value:     string(raw),
			Type:      "json",
			UpdatedBy: updatedBy,
		}
		if err := database.DB.Create(&setting).Error; err != nil {
			return err
		}
	} else {
		setting.Value = string(raw)
		setting.Type = "json"
		setting.UpdatedBy = updatedBy
		if err := database.DB.Save(&setting).Error; err != nil {
			return err
		}
	}

	InvalidateTestModePolicy()
	return nil
}

func loadTestModePolicyFromDB() TestModePolicy {
	def := DefaultTestModePolicy()
	if database.DB == nil {
		return def
	}
	var setting models.PlatformSettings
	if err := database.DB.Where("key = ?", testModePolicyKey).First(&setting).Error; err != nil {
		return def
	}
	if setting.Value == "" {
		return def
	}
	var parsed TestModePolicy
	if err := json.Unmarshal([]byte(setting.Value), &parsed); err != nil {
		log.Printf("test_mode_policy: parse failed, using defaults: %v", err)
		return def
	}
	// A saved policy with an empty list is a legitimate choice — it means
	// "nobody but admins can see test kitchens" — so it is deliberately NOT
	// backfilled with defaults. Only an absent or unparseable row falls back.
	return parsed
}
