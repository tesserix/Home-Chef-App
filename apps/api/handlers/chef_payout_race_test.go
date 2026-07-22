package handlers

// chef_payout_race_test.go — concurrency proof for review finding 1
// (payout-release-governor fix wave): SavePayoutDetails read the chef row,
// called services.RegisterSettlementAccount, then persisted the result with
// no transaction and no row lock. Two near-simultaneous requests (a
// double-click, or a client retry — now far more likely because this call is
// synchronous and makes up to 4 external HTTP calls) both read
// RazorpayAccountID=="", so both call POST /v2/accounts — each minting a
// FULL linked account (stakeholder + live bank settlement) at Razorpay. The
// last persist wins; the other account is orphaned at Razorpay with live
// settlement on it and no way to merge its money back.
//
// This needs real Postgres. sqlite in-memory serialises writes and treats
// SELECT ... FOR UPDATE as a no-op, so it cannot demonstrate the race in
// either direction — a green run there would be meaningless. Same convention
// as handlers/menu_image_race_test.go:
//
//   HOMECHEF_TEST_PG='host=127.0.0.1 port=55432 user=postgres dbname=racetest sslmode=disable'
//
// NOTE on how "real" this proof is: Postgres's row lock genuinely serialises
// the racing transactions — a concurrent SELECT ... FOR UPDATE (or even a
// plain UPDATE) on the same chef row blocks until the winner's transaction
// commits, so this is not a timing-window test that merely usually passes.
// The one honest caveat is direction, not mechanism: this test can prove the
// fix (transaction + row lock) drives the account-creation hit count to
// exactly 1. Run against the PRE-fix handler it is expected to be flaky-red
// (>1 hits most runs, occasionally 1 if the scheduler happens to serialise
// the racers anyway) rather than deterministic-red — the absence of any
// synchronization in the old code means the race is real but not guaranteed
// to fire on every single run. The fixed code's row lock is what makes the
// assertion deterministic.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

func payoutRacePG(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("HOMECHEF_TEST_PG")
	if dsn == "" {
		t.Skip("HOMECHEF_TEST_PG not set — concurrency proof needs real Postgres")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	return db
}

// payoutRaceUsersDDL / payoutRaceChefProfilesDDL list every column GORM maps
// for models.User / models.ChefProfile that SavePayoutDetails' Preload("User")
// read and its Save/Updates calls touch — the Postgres-real-types twin of the
// sqlite chefProfilesGuardDDL (chef_fulfillment_guard_test.go) this package
// already relies on for the same handler's non-concurrent tests.
const payoutRaceUsersDDL = `CREATE TABLE IF NOT EXISTS users (
	id uuid PRIMARY KEY,
	email text NOT NULL DEFAULT '', first_name text NOT NULL DEFAULT '', last_name text NOT NULL DEFAULT '',
	phone text NOT NULL DEFAULT '',
	email_enc text DEFAULT '', email_bidx text DEFAULT '',
	first_name_enc text DEFAULT '', last_name_enc text DEFAULT '',
	phone_enc text DEFAULT '', phone_bidx text DEFAULT '',
	avatar text NOT NULL DEFAULT '', role varchar(20) NOT NULL DEFAULT 'customer',
	gip_uid text, gip_tenant_id text, gip_provider text, auth_pool varchar(16),
	is_active boolean NOT NULL DEFAULT true, phone_verified boolean NOT NULL DEFAULT false,
	fcm_token text NOT NULL DEFAULT '',
	marketing_consent boolean NOT NULL DEFAULT false, marketing_consent_at timestamptz,
	last_login_at timestamptz, created_at timestamptz, updated_at timestamptz, deleted_at timestamptz
)`

const payoutRaceChefProfilesDDL = `CREATE TABLE IF NOT EXISTS chef_profiles (
	id uuid PRIMARY KEY, user_id uuid, business_name text DEFAULT '',
	address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '',
	slug text DEFAULT '', description text DEFAULT '',
	profile_image text DEFAULT '', banner_image text DEFAULT '',
	cuisines text DEFAULT '{}', specialties text DEFAULT '{}',
	prep_time text DEFAULT '', minimum_order double precision DEFAULT 0,
	delivery_radius double precision DEFAULT 10, service_radius double precision DEFAULT 10,
	offers_pickup boolean DEFAULT false, offers_self_delivery boolean DEFAULT false,
	self_delivery_base_fee double precision DEFAULT 0, self_delivery_free_radius_km double precision DEFAULT 0,
	self_delivery_per_km double precision DEFAULT 0, self_delivery_max_fee double precision DEFAULT 0,
	self_delivery_max_distance_km double precision DEFAULT 0,
	rating double precision DEFAULT 0, total_reviews integer DEFAULT 0, total_orders integer DEFAULT 0,
	issue_count integer DEFAULT 0,
	is_verified boolean DEFAULT false, verified_at timestamptz,
	is_active boolean DEFAULT true, accepting_orders boolean DEFAULT true, paused_until timestamptz,
	auto_schedule_enabled boolean DEFAULT false,
	kitchen_photos text DEFAULT '{}', kitchen_type varchar(20) DEFAULT 'home_kitchen',
	address_line1 text DEFAULT '', address_line2 text DEFAULT '',
	city text DEFAULT '', state text DEFAULT '', postal_code text DEFAULT '',
	latitude double precision DEFAULT 0, longitude double precision DEFAULT 0,
	is_featured boolean DEFAULT false, featured_until timestamptz,
	stripe_account_id text DEFAULT '', razorpay_account_id text DEFAULT '',
	payment_provider varchar(20) DEFAULT 'razorpay', payout_country text DEFAULT 'IN',
	stripe_charges_enabled boolean DEFAULT false, stripe_payouts_enabled boolean DEFAULT false,
	razorpay_product_id text DEFAULT '', razorpay_settlement_status text DEFAULT '',
	razorpay_settlement_requirements text DEFAULT '', razorpay_stakeholder_created boolean DEFAULT false,
	payout_auto_release varchar(8) DEFAULT '',
	payout_method text DEFAULT '', bank_account_number text DEFAULT '',
	bank_ifsc text DEFAULT '', bank_account_name text DEFAULT '', upi_id text DEFAULT '',
	pan_number varchar(10) DEFAULT '', fssai_license_number varchar(14) DEFAULT '', gstin varchar(15) DEFAULT '',
	fssai_override_until timestamptz, fssai_override_reason text DEFAULT '', fssai_override_by uuid,
	created_at timestamptz, updated_at timestamptz
)`

// payoutRaceAuditDDL mirrors chefGuardAuditDDL (chef_fulfillment_guard_test.go)
// — SavePayoutDetails' services.LogAudit call needs this table to exist or it
// just logs a warning per call, which is harmless but noisy under 8 racers.
const payoutRaceAuditDDL = `CREATE TABLE IF NOT EXISTS audit_logs (
	id uuid DEFAULT gen_random_uuid(),
	user_id uuid, action text, entity_type text, entity_id text,
	old_value text, new_value text, ip_address text, user_agent text,
	correlation_id text, created_at timestamptz
)`

// seedPayoutRaceChef creates the tables (idempotent — IF NOT EXISTS) and one
// fresh user/chef pair with no Razorpay account yet, the state every racer
// contends over.
func seedPayoutRaceChef(t *testing.T, db *gorm.DB) (userID, chefID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(payoutRaceUsersDDL).Error)
	require.NoError(t, db.Exec(payoutRaceChefProfilesDDL).Error)
	require.NoError(t, db.Exec(payoutRaceAuditDDL).Error)

	userID, chefID = uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, first_name, last_name, phone, role) VALUES ($1,$2,$3,$4,$5,'chef')`,
		userID, "chef-race-"+userID.String()[:8]+"@example.com", "Anita", "Rao", "9876543210").Error)
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name) VALUES ($1,$2,$3)`,
		chefID, userID, "Anita's Kitchen "+chefID.String()[:8]).Error)

	t.Cleanup(func() {
		db.Exec(`DELETE FROM chef_profiles WHERE id = $1`, chefID)
		db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	})
	return userID, chefID
}

// TestSavePayoutDetailsRace_ExactlyOneLinkedAccountCreated fires several
// simultaneous SavePayoutDetails calls for the SAME chef and asserts the
// Razorpay stub sees exactly one POST /v2/accounts — proving the fix (a
// transaction that locks the chef row) prevents two racers from both minting
// a linked account.
func TestSavePayoutDetailsRace_ExactlyOneLinkedAccountCreated(t *testing.T) {
	db := payoutRacePG(t)
	userID, _ := seedPayoutRaceChef(t, db)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	// SavePayoutDetails' secret-storage goroutine calls config.IsDevelopment(),
	// which nil-derefs unless AppConfig is set. Same non-restoring pattern as
	// chef_payout_settlement_test.go's setupChefPayoutSettlementDB — never
	// written back to nil, since that goroutine can still be running after
	// this test function returns.
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{Environment: "test"}
	}

	// A thread-safe stub: httptest.Server dispatches each request on its own
	// goroutine, and this test intentionally sends genuinely concurrent
	// requests, so an unguarded counter here would itself be a data race
	// (unlike chef_payout_settlement_test.go's settlementRouteServer, which
	// is only ever driven by ONE request at a time).
	var accountCreateHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/accounts":
			accountCreateHits.Add(1)
			_, _ = w.Write([]byte(`{"id":"acc_123","status":"created"}`))
		case strings.HasSuffix(r.URL.Path, "/stakeholders"):
			_, _ = w.Write([]byte(`{"id":"sth_1"}`))
		case strings.Contains(r.URL.Path, "/products/"):
			_, _ = w.Write([]byte(`{"id":"acc_prd_1","activation_status":"activated"}`))
		case strings.HasSuffix(r.URL.Path, "/products"):
			_, _ = w.Write([]byte(`{"id":"acc_prd_1","activation_status":"created"}`))
		default:
			_, _ = w.Write([]byte(`{"id":"acc_123","status":"created"}`))
		}
	}))
	t.Cleanup(srv.Close)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "k", "s", ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })

	router := payoutSettlementRouter(userID)
	payload, err := json.Marshal(bankTransferPayload())
	require.NoError(t, err)

	const racers = 8
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // release together to maximise overlap
			req := httptest.NewRequest(http.MethodPost, "/chef/payout", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}()
	}
	close(start)
	wg.Wait()

	if got := accountCreateHits.Load(); got != 1 {
		t.Fatalf("POST /v2/accounts hits = %d, want exactly 1 — a race must never mint two linked accounts for one chef", got)
	}
}
