package services

// fssai_badge_test.go — the hygiene/food-safety badge (#35) must agree with the
// FSSAI lockout: a chef shows the badge iff they hold a verified, non-expired
// FSSAI licence (latest verified expiry wins, so a renewal lights it back up).

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func TestChefsWithValidFSSAI(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE chef_documents (
		id text PRIMARY KEY, chef_id text, type text, status text, expiry_date datetime
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	now := time.Now()
	future := now.AddDate(0, 0, 90)
	past := now.AddDate(0, 0, -5)
	add := func(chef uuid.UUID, status models.DocumentStatus, exp time.Time) {
		t.Helper()
		if err := db.Exec(`INSERT INTO chef_documents (id, chef_id, type, status, expiry_date) VALUES (?,?,?,?,?)`,
			uuid.New().String(), chef.String(), string(models.DocFSSAILicense), string(status), exp).Error; err != nil {
			t.Fatal(err)
		}
	}

	valid := uuid.New()
	add(valid, models.DocStatusVerified, future)
	expired := uuid.New()
	add(expired, models.DocStatusVerified, past)
	pending := uuid.New()
	add(pending, models.DocStatusPending, future) // not verified
	renewed := uuid.New()
	add(renewed, models.DocStatusVerified, past)   // old lapsed
	add(renewed, models.DocStatusVerified, future) // verified renewal
	nodoc := uuid.New()

	got := ChefsWithValidFSSAI([]uuid.UUID{valid, expired, pending, renewed, nodoc})
	if !got[valid] {
		t.Error("verified non-expired FSSAI should earn the badge")
	}
	if got[expired] {
		t.Error("expired FSSAI must not earn the badge")
	}
	if got[pending] {
		t.Error("unverified (pending) FSSAI must not earn the badge")
	}
	if !got[renewed] {
		t.Error("a verified renewal (latest expiry in future) should earn the badge")
	}
	if got[nodoc] {
		t.Error("no FSSAI on file → no badge")
	}

	// Single-chef convenience form.
	if !ChefHasValidFSSAI(valid) {
		t.Error("ChefHasValidFSSAI(valid) should be true")
	}
	if ChefHasValidFSSAI(expired) {
		t.Error("ChefHasValidFSSAI(expired) should be false")
	}
}
