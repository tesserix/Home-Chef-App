package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// TestIsChefFSSAIExpired exercises the food-safety lockout decision matrix from
// issue #32. We build a throwaway in-memory chef_documents table by hand rather
// than AutoMigrate, because the GORM models carry Postgres-specific tags (uuid +
// gen_random_uuid()) and a ChefProfile association sqlite can't replicate.
// IsChefFSSAIExpired only reads chef_documents, so that is all the schema we need.
func TestIsChefFSSAIExpired(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE chef_documents (
		id text PRIMARY KEY, chef_id text, type text, file_name text, file_path text,
		bucket text, content_type text, file_size integer, status text,
		rejection_reason text, expiry_date datetime, created_at datetime, updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	now := time.Now()
	past := now.AddDate(0, 0, -2)
	future := now.AddDate(0, 0, 30)

	insert := func(chefID uuid.UUID, status models.DocumentStatus, expiry *time.Time) {
		t.Helper()
		var exp interface{}
		if expiry != nil {
			exp = *expiry
		}
		if err := db.Exec(
			`INSERT INTO chef_documents (id, chef_id, type, status, expiry_date, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			uuid.New().String(), chefID.String(), string(models.DocFSSAILicense), string(status), exp, now,
		).Error; err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	inChef := func(id uuid.UUID) *models.ChefProfile {
		return &models.ChefProfile{ID: id, PayoutCountry: "IN"}
	}

	t.Run("nil chef is not locked", func(t *testing.T) {
		if IsChefFSSAIExpired(nil) {
			t.Fatal("nil chef must not be locked")
		}
	})

	t.Run("non-India chef is never FSSAI-locked", func(t *testing.T) {
		id := uuid.New()
		insert(id, models.DocStatusVerified, &past) // even an expired doc must not lock a non-IN chef
		if IsChefFSSAIExpired(&models.ChefProfile{ID: id, PayoutCountry: "US"}) {
			t.Fatal("non-IN chef must never be FSSAI-locked")
		}
	})

	t.Run("no FSSAI doc is not locked", func(t *testing.T) {
		if IsChefFSSAIExpired(inChef(uuid.New())) {
			t.Fatal("chef with no FSSAI doc must not be locked (onboarding/backfill concern)")
		}
	})

	t.Run("verified future expiry is valid", func(t *testing.T) {
		id := uuid.New()
		insert(id, models.DocStatusVerified, &future)
		if IsChefFSSAIExpired(inChef(id)) {
			t.Fatal("verified future expiry must be valid")
		}
	})

	t.Run("verified past expiry is locked", func(t *testing.T) {
		id := uuid.New()
		insert(id, models.DocStatusVerified, &past)
		if !IsChefFSSAIExpired(inChef(id)) {
			t.Fatal("verified past expiry must be locked")
		}
	})

	t.Run("pending-only expired doc is not treated as a lapsed verified licence", func(t *testing.T) {
		id := uuid.New()
		insert(id, models.DocStatusPending, &past)
		if IsChefFSSAIExpired(inChef(id)) {
			t.Fatal("a pending (never-verified) doc must not count as a lapsed licence")
		}
	})

	t.Run("verified-expired stays locked despite an unverified renewal (anti-gaming)", func(t *testing.T) {
		id := uuid.New()
		insert(id, models.DocStatusVerified, &past)  // lapsed verified licence
		insert(id, models.DocStatusPending, &future) // unverified renewal
		if !IsChefFSSAIExpired(inChef(id)) {
			t.Fatal("an unverified renewal must not lift the lock")
		}
	})

	t.Run("verified renewal lifts the lock", func(t *testing.T) {
		id := uuid.New()
		insert(id, models.DocStatusVerified, &past)   // old lapsed licence
		insert(id, models.DocStatusVerified, &future) // verified renewal — latest expiry wins
		if IsChefFSSAIExpired(inChef(id)) {
			t.Fatal("a verified renewal with future expiry must lift the lock")
		}
	})

	t.Run("verified doc with NULL expiry is not locked (legacy/backfill)", func(t *testing.T) {
		id := uuid.New()
		insert(id, models.DocStatusVerified, nil)
		if IsChefFSSAIExpired(inChef(id)) {
			t.Fatal("verified doc with NULL expiry must not lock (handled by backfill)")
		}
	})

	t.Run("active admin override suspends the lockout (#93)", func(t *testing.T) {
		id := uuid.New()
		insert(id, models.DocStatusVerified, &past) // lapsed verified licence
		until := now.AddDate(0, 0, 5)
		chef := &models.ChefProfile{ID: id, PayoutCountry: "IN", FSSAIOverrideUntil: &until}
		if IsChefFSSAIExpired(chef) {
			t.Fatal("an active admin override must suspend the lockout")
		}
	})

	t.Run("lapsed admin override lets the lockout re-apply (#93)", func(t *testing.T) {
		id := uuid.New()
		insert(id, models.DocStatusVerified, &past) // lapsed verified licence
		expiredOverride := now.AddDate(0, 0, -1)
		chef := &models.ChefProfile{ID: id, PayoutCountry: "IN", FSSAIOverrideUntil: &expiredOverride}
		if !IsChefFSSAIExpired(chef) {
			t.Fatal("once an override lapses the expiry lockout must re-apply")
		}
	})
}

// TestExcludeFSSAILocked asserts the set-based listing filter (#91) mirrors
// IsChefFSSAIExpired — in particular that a chef who RENEWED (old expired doc +
// verified future doc) stays visible, and non-IN / no-doc / null-expiry chefs
// are never hidden.
func TestExcludeFSSAILocked(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE chef_profiles (id text PRIMARY KEY, payout_country text, fssai_override_until datetime)`).Error; err != nil {
		t.Fatalf("create chef_profiles: %v", err)
	}
	if err := db.Exec(`CREATE TABLE chef_documents (
		id text PRIMARY KEY, chef_id text, type text, status text, expiry_date datetime
	)`).Error; err != nil {
		t.Fatalf("create chef_documents: %v", err)
	}

	now := time.Now()
	past := now.AddDate(0, 0, -5)
	future := now.AddDate(0, 0, 30)

	addChef := func(country string) uuid.UUID {
		t.Helper()
		id := uuid.New()
		if err := db.Exec(`INSERT INTO chef_profiles (id, payout_country) VALUES (?, ?)`, id.String(), country).Error; err != nil {
			t.Fatalf("insert chef: %v", err)
		}
		return id
	}
	addChefOverride := func(country string, overrideUntil time.Time) uuid.UUID {
		t.Helper()
		id := uuid.New()
		if err := db.Exec(`INSERT INTO chef_profiles (id, payout_country, fssai_override_until) VALUES (?, ?, ?)`,
			id.String(), country, overrideUntil).Error; err != nil {
			t.Fatalf("insert chef: %v", err)
		}
		return id
	}
	addDoc := func(chefID uuid.UUID, status models.DocumentStatus, expiry *time.Time) {
		t.Helper()
		var exp interface{}
		if expiry != nil {
			exp = *expiry
		}
		if err := db.Exec(`INSERT INTO chef_documents (id, chef_id, type, status, expiry_date) VALUES (?, ?, ?, ?, ?)`,
			uuid.New().String(), chefID.String(), string(models.DocFSSAILicense), string(status), exp).Error; err != nil {
			t.Fatalf("insert doc: %v", err)
		}
	}

	valid := addChef("IN")
	addDoc(valid, models.DocStatusVerified, &future)
	expired := addChef("IN")
	addDoc(expired, models.DocStatusVerified, &past)
	renewed := addChef("IN") // old expired doc + verified renewal -> must stay visible
	addDoc(renewed, models.DocStatusVerified, &past)
	addDoc(renewed, models.DocStatusVerified, &future)
	intl := addChef("US")
	addDoc(intl, models.DocStatusVerified, &past)
	nodoc := addChef("IN")
	nullexp := addChef("IN")
	addDoc(nullexp, models.DocStatusVerified, nil)
	overridden := addChefOverride("IN", future) // expired doc but active override -> visible
	addDoc(overridden, models.DocStatusVerified, &past)
	overrideLapsed := addChefOverride("IN", past) // expired doc + lapsed override -> hidden
	addDoc(overrideLapsed, models.DocStatusVerified, &past)

	var ids []string
	if err := db.Table("chef_profiles").Scopes(ExcludeFSSAILocked).Pluck("id", &ids).Error; err != nil {
		t.Fatalf("query: %v", err)
	}
	got := map[string]bool{}
	for _, id := range ids {
		got[id] = true
	}

	for name, id := range map[string]uuid.UUID{"valid": valid, "renewed": renewed, "non-IN": intl, "no-doc": nodoc, "null-expiry": nullexp, "overridden": overridden} {
		if !got[id.String()] {
			t.Errorf("%s chef should be visible in listings but was hidden", name)
		}
	}
	for name, id := range map[string]uuid.UUID{"expired": expired, "override-lapsed": overrideLapsed} {
		if got[id.String()] {
			t.Errorf("%s chef must be hidden from listings", name)
		}
	}
}
