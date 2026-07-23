package services

import (
	"testing"

	"github.com/homechef/api/models"
)

// #741 — resolving the master switch, the per-chef tri-state and the default.
// The precedence is the safety property: a kill switch that individual records
// can opt out of is not a kill switch.
//
// setupPlatformSettingsDB / setSetting are the existing sqlite-backed
// platform_settings helpers from premium_pricing_test.go; reused verbatim
// rather than duplicated.

func TestPayoutAutomation_MasterSwitchOffBeatsEverything(t *testing.T) {
	db := setupPlatformSettingsDB(t)
	setSetting(t, db, "payout.sweep_enabled", "false")
	setSetting(t, db, "payout.auto_release_default", "on")

	chef := &models.ChefProfile{PayoutAutoRelease: PayoutAutoOn}
	if PayoutAutomationEnabled(db, chef) {
		t.Fatal("a per-chef 'on' must not override the master kill switch")
	}
}

func TestPayoutAutomation_ChefOnWithMasterOn(t *testing.T) {
	db := setupPlatformSettingsDB(t)
	setSetting(t, db, "payout.sweep_enabled", "true")
	setSetting(t, db, "payout.auto_release_default", "off")

	chef := &models.ChefProfile{PayoutAutoRelease: PayoutAutoOn}
	if !PayoutAutomationEnabled(db, chef) {
		t.Fatal("an explicitly enabled chef releases even when the default is off")
	}
}

func TestPayoutAutomation_ChefOffBeatsDefaultOn(t *testing.T) {
	db := setupPlatformSettingsDB(t)
	setSetting(t, db, "payout.sweep_enabled", "true")
	setSetting(t, db, "payout.auto_release_default", "on")

	chef := &models.ChefProfile{PayoutAutoRelease: PayoutAutoOff}
	if PayoutAutomationEnabled(db, chef) {
		t.Fatal("a suspended chef must not auto-release")
	}
}

func TestPayoutAutomation_UnsetFollowsTheDefault(t *testing.T) {
	// setSetting is INSERT-only and platform_settings.key is uniquely indexed
	// (mirroring the production schema), so flipping the same key twice needs
	// two fresh DBs rather than one mutated in place — the assertions are
	// otherwise identical to the brief.
	chef := &models.ChefProfile{PayoutAutoRelease: ""}

	dbOff := setupPlatformSettingsDB(t)
	setSetting(t, dbOff, "payout.sweep_enabled", "true")
	setSetting(t, dbOff, "payout.auto_release_default", "off")
	if PayoutAutomationEnabled(dbOff, chef) {
		t.Fatal("unset must follow the default (off)")
	}

	dbOn := setupPlatformSettingsDB(t)
	setSetting(t, dbOn, "payout.sweep_enabled", "true")
	setSetting(t, dbOn, "payout.auto_release_default", "on")
	if !PayoutAutomationEnabled(dbOn, chef) {
		t.Fatal("unset must follow the default (on)")
	}
}

func TestPayoutAutomation_DefaultsClosedWhenUnconfigured(t *testing.T) {
	// An empty settings table must not start moving money.
	db := setupPlatformSettingsDB(t)
	chef := &models.ChefProfile{}
	if PayoutAutomationEnabled(db, chef) {
		t.Fatal("missing settings must default to no automation")
	}
}

func TestPayoutAutomation_GarbageValuesFailClosed(t *testing.T) {
	db := setupPlatformSettingsDB(t)
	setSetting(t, db, "payout.sweep_enabled", "yes-please")
	chef := &models.ChefProfile{PayoutAutoRelease: PayoutAutoOn}
	if PayoutAutomationEnabled(db, chef) {
		t.Fatal("an unparseable master switch must fail closed")
	}
}

func TestPayoutAutomation_MissingSweepSettingWithChefOn(t *testing.T) {
	// The kill switch absent is not the kill switch permitting. An explicitly
	// enabled chef must still not release when payout.sweep_enabled has never
	// been written — otherwise a fresh environment starts moving money.
	db := setupPlatformSettingsDB(t)
	setSetting(t, db, "payout.auto_release_default", "on")
	// deliberately never set payout.sweep_enabled

	chef := &models.ChefProfile{PayoutAutoRelease: PayoutAutoOn}
	if PayoutAutomationEnabled(db, chef) {
		t.Fatal("an absent master switch must not permit automation")
	}
}
