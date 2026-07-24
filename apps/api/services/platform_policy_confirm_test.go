package services

import (
	"testing"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
)

// The auto-confirm flow ships ON: the policy default must be true so the flow
// runs without any admin action.
func TestDefaultPlatformPolicy_ConfirmReceiptFlowDefaultsTrue(t *testing.T) {
	if !DefaultPlatformPolicy().ConfirmReceiptFlowEnabled {
		t.Fatal("expected ConfirmReceiptFlowEnabled to default true")
	}
}

// An admin turning the flow OFF from the console must stick — a pointer partial
// distinguishes an explicit false from an absent key, so the default-on value
// is genuinely overridable to off (a real kill switch).
func TestLoadPlatformPolicy_ConfirmReceiptFlow_HonorsAdminFalse(t *testing.T) {
	db := setupHoldDB(t)
	old := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = old })

	if err := db.Exec(
		`INSERT INTO platform_settings (id, key, value, type) VALUES (?,?,?,?)`,
		uuid.New().String(), platformPolicyKey, `{"confirmReceiptFlowEnabled": false}`, "json",
	).Error; err != nil {
		t.Fatal(err)
	}

	p := loadPlatformPolicyFromDB()
	if p.ConfirmReceiptFlowEnabled {
		t.Fatal("expected admin-set false to override the default-on value")
	}
}

// Absent key → keeps the default-on value.
func TestLoadPlatformPolicy_ConfirmReceiptFlow_AbsentKeepsDefaultTrue(t *testing.T) {
	db := setupHoldDB(t)
	old := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = old })

	// A policy row that sets other fields but omits confirmReceiptFlowEnabled.
	if err := db.Exec(
		`INSERT INTO platform_settings (id, key, value, type) VALUES (?,?,?,?)`,
		uuid.New().String(), platformPolicyKey, `{"taxPercent": 5}`, "json",
	).Error; err != nil {
		t.Fatal(err)
	}

	p := loadPlatformPolicyFromDB()
	if !p.ConfirmReceiptFlowEnabled {
		t.Fatal("expected the default-on value to survive when the key is absent")
	}
}
