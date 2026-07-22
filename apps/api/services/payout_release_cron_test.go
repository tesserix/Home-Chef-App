package services

// payout_release_cron_test.go — #741 — mapping an order onto the release
// decision. The sweep itself is a loop; the interesting behaviour is what it
// feeds the governor (BuildReleaseInput) and the noise guard that keeps a
// still-maturing order from flooding the outbox (recordPayoutBlock).
//
// newPayoutReleaseTestDB builds on setupPlatformSettingsDB (premium_pricing_test.go)
// for the platform_settings table, rather than duplicating it, and adds the
// orders / chef_profiles / payout_ledger_entries / outbox_events tables this
// package's own tests need. The orders DDL carries every column models.Order
// maps (mirroring handlers/chef_delivered_gate_test.go's ordersDDL) as a
// defensive measure: BuildReleaseInput itself only reads the in-memory
// *models.Order it is handed (no re-fetch), so no test here needs a full
// gorm .Save(&order) — deliberately avoided, since a Save on a struct with a
// preloaded Chef association cascades into an update of chef_profiles too,
// which would need that table's full column set as well. Kept comprehensive
// anyway so a future test that DOES persist a full order doesn't silently hit
// the "missing column" class of breakage this task was warned about.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

// ordersDDL mirrors handlers/chef_delivered_gate_test.go's constant of the same
// intent: every column GORM's db.Save(&order) writes, so a full-model save
// never fails on a missing column.
const payoutReleaseOrdersDDL = `CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '',
	id text PRIMARY KEY,
	order_number text, customer_id text, chef_id text, delivery_id text,
	status text, payment_status text, payment_method text, fulfillment_type text,
	subtotal real DEFAULT 0, delivery_fee real DEFAULT 0, service_fee real DEFAULT 0,
	tax real DEFAULT 0, tax_rate real DEFAULT 0, tax_name text, commission_rate real DEFAULT 0,
	tip real DEFAULT 0,
	chef_tip real DEFAULT 0, driver_tip real DEFAULT 0, discount real DEFAULT 0,
	chef_funded_discount real DEFAULT 0, total real DEFAULT 0, wallet_applied real DEFAULT 0,
	promo_code text, currency text,
	delivery_address_line1 text, delivery_address_line2 text, delivery_address_city text,
	delivery_address_state text, delivery_address_postal_code text, delivery_address_country text,
	delivery_latitude real DEFAULT 0, delivery_longitude real DEFAULT 0, delivery_instructions text,
	estimated_prep_time integer DEFAULT 0, estimated_delivery_time integer DEFAULT 0,
	scheduled_for datetime, delivery_slot text,
	requested_fulfillment_at datetime, confirmed_fulfillment_at datetime, fulfillment_time_status text,
	accept_reminder_count integer DEFAULT 0, last_accept_reminder_at datetime,
	accepted_at datetime, prepared_at datetime, picked_up_at datetime, delivered_at datetime,
	cancelled_at datetime, cancel_reason text, special_instructions text,
	ready_photo_url text, handover_photo_url text,
	payment_provider text, stripe_payment_intent_id text, razorpay_order_id text,
	razorpay_payment_id text, refund_id text, refunded_at datetime, refund_amount real DEFAULT 0,
	refund_reason text, refund_initiated_by text,
	payout_hold_status text DEFAULT '', customer_confirmed_at datetime,
	payout_settled_at datetime, payout_settle_attempts integer DEFAULT 0,
	created_at datetime, updated_at datetime, deleted_at datetime)`

const payoutReleaseChefProfilesDDL = `CREATE TABLE chef_profiles (
	address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '',
	id text PRIMARY KEY, user_id text, business_name text, state text DEFAULT '',
	razorpay_account_id text DEFAULT '', razorpay_settlement_status text DEFAULT '',
	payout_auto_release text DEFAULT '')`

const payoutReleaseLedgerDDL = `CREATE TABLE payout_ledger_entries (
	id text PRIMARY KEY, tenant_id text NOT NULL, payee_type text NOT NULL, payee_id text NOT NULL,
	kind text NOT NULL, amount_minor integer NOT NULL, currency text NOT NULL,
	source_type text NOT NULL, source_id text NOT NULL,
	matures_at datetime, batch_id text, actor_id text, reason text, created_at datetime)`

const payoutReleaseOutboxDDL = `CREATE TABLE outbox_events (
	id text PRIMARY KEY, subject text, msg_id text, aggregate_type text, aggregate_id text,
	payload text, status text, attempts integer DEFAULT 0, last_error text, next_retry_at datetime,
	created_at datetime, updated_at datetime, published_at datetime)`

// newPayoutReleaseTestDB extends setupPlatformSettingsDB (platform_settings)
// with the tables BuildReleaseInput / recordPayoutBlock touch.
func newPayoutReleaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupPlatformSettingsDB(t)
	for _, s := range []string{
		payoutReleaseOrdersDDL,
		payoutReleaseChefProfilesDDL,
		payoutReleaseLedgerDDL,
		payoutReleaseOutboxDDL,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

// ptrTime is a small &t helper for building *time.Time literals inline.
func ptrTime(t time.Time) *time.Time { return &t }

// seedDeliveredOrder inserts a chef profile (optionally mutated by mutateChef)
// and a delivered order referencing it, returning the order with Chef preloaded
// — the same shape runPayoutReleaseSweep loads.
func seedDeliveredOrder(t *testing.T, db *gorm.DB, mutateChef func(*models.ChefProfile)) *models.Order {
	t.Helper()

	chef := &models.ChefProfile{
		ID:                       uuid.New(),
		RazorpaySettlementStatus: "activated",
	}
	if mutateChef != nil {
		mutateChef(chef)
	}
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, razorpay_account_id, razorpay_settlement_status, payout_auto_release, state)
		 VALUES (?,?,?,?,?)`,
		chef.ID.String(), chef.RazorpayAccountID, chef.RazorpaySettlementStatus, chef.PayoutAutoRelease, chef.State,
	).Error)

	orderID := uuid.New()
	delivered := time.Now().Add(-3 * time.Hour) // safely matured against the 2h default
	require.NoError(t, db.Exec(
		`INSERT INTO orders
			(id, order_number, chef_id, status, subtotal, tax, chef_tip, delivery_fee, chef_funded_discount,
			 commission_rate, total, delivery_address_state, delivered_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		orderID.String(), "ORD-"+orderID.String()[:8], chef.ID.String(), string(models.OrderStatusDelivered),
		500.0, 25.0, 0.0, 40.0, 0.0, 0.0, 565.0, "", delivered,
	).Error)

	var order models.Order
	require.NoError(t, db.Preload("Chef").First(&order, "id = ?", orderID.String()).Error)
	return &order
}

func TestBuildReleaseInput_ReadsSettlementActivation(t *testing.T) {
	db := newPayoutReleaseTestDB(t)
	order := seedDeliveredOrder(t, db, func(c *models.ChefProfile) {
		c.RazorpaySettlementStatus = "activated"
	})

	in, err := BuildReleaseInput(db, order, time.Now())
	if err != nil {
		t.Fatalf("BuildReleaseInput: %v", err)
	}
	if !in.SettlementActivated {
		t.Fatal("an activated chef must be reported as activated")
	}
}

func TestBuildReleaseInput_TreatsNeedsClarificationAsNotActivated(t *testing.T) {
	// The single most dangerous mis-mapping: needs_clarification means
	// Razorpay has NOT accepted the bank account, so a release strands money.
	db := newPayoutReleaseTestDB(t)
	order := seedDeliveredOrder(t, db, func(c *models.ChefProfile) {
		c.RazorpaySettlementStatus = "needs_clarification"
	})

	in, err := BuildReleaseInput(db, order, time.Now())
	if err != nil {
		t.Fatalf("BuildReleaseInput: %v", err)
	}
	if in.SettlementActivated {
		t.Fatal("needs_clarification must not count as activated")
	}
}

func TestBuildReleaseInput_UsesConfiguredMaturation(t *testing.T) {
	db := newPayoutReleaseTestDB(t)
	setSetting(t, db, "payout.maturation_minutes", "120")
	order := seedDeliveredOrder(t, db, nil)

	in, err := BuildReleaseInput(db, order, time.Now())
	if err != nil {
		t.Fatalf("BuildReleaseInput: %v", err)
	}
	if in.Maturation != 2*time.Hour {
		t.Fatalf("maturation = %v, want 2h", in.Maturation)
	}
}

func TestBuildReleaseInput_FallsBackToTwoHours(t *testing.T) {
	// An unset or garbage setting must not mean "release immediately".
	db := newPayoutReleaseTestDB(t)
	setSetting(t, db, "payout.maturation_minutes", "not-a-number")
	order := seedDeliveredOrder(t, db, nil)

	in, err := BuildReleaseInput(db, order, time.Now())
	if err != nil {
		t.Fatalf("BuildReleaseInput: %v", err)
	}
	if in.Maturation != 2*time.Hour {
		t.Fatalf("maturation = %v, want the 2h fallback", in.Maturation)
	}
}

func TestBuildReleaseInput_FlagsAnOpenRefund(t *testing.T) {
	db := newPayoutReleaseTestDB(t)
	order := seedDeliveredOrder(t, db, nil)
	// BuildReleaseInput reads the in-memory order it is handed rather than
	// re-fetching, so mutating the field is enough — no round-trip needed.
	order.RefundedAt = ptrTime(time.Now())

	in, err := BuildReleaseInput(db, order, time.Now())
	if err != nil {
		t.Fatalf("BuildReleaseInput: %v", err)
	}
	if !in.RefundOpen {
		t.Fatal("a refunded order must block release")
	}
}

func TestRecordPayoutBlock_SkipsAStillMaturingOrder(t *testing.T) {
	db := newPayoutReleaseTestDB(t)
	order := seedDeliveredOrder(t, db, nil)
	recordPayoutBlock(db, order, payouts.ReleaseDecision{
		Reasons: []payouts.BlockReason{payouts.BlockNotMatured},
	})
	if countOutbox(t, db, SubjectPayoutBlocked) != 0 {
		t.Fatal("a still-maturing order is the normal case and must not be published")
	}
}

func TestRecordPayoutBlock_PublishesARealBlock(t *testing.T) {
	db := newPayoutReleaseTestDB(t)
	order := seedDeliveredOrder(t, db, nil)
	recordPayoutBlock(db, order, payouts.ReleaseDecision{
		Reasons: []payouts.BlockReason{payouts.BlockSettlementNotActivated},
	})
	if countOutbox(t, db, SubjectPayoutBlocked) != 1 {
		t.Fatal("a genuine block must reach the admin surface")
	}
}

func TestRecordPayoutBlock_StillPublishesWhenNotMaturedJoinsARealBlock(t *testing.T) {
	// The noise guard only silences the SOLE-reason "still maturing" case; when
	// a real block accompanies it, the admin must still see it.
	db := newPayoutReleaseTestDB(t)
	order := seedDeliveredOrder(t, db, nil)
	recordPayoutBlock(db, order, payouts.ReleaseDecision{
		Reasons: []payouts.BlockReason{payouts.BlockNotMatured, payouts.BlockSettlementNotActivated},
	})
	if countOutbox(t, db, SubjectPayoutBlocked) != 1 {
		t.Fatal("a real block alongside not_matured must still be published")
	}
}
