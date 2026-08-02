package services

// customer_risk_test.go — the customer refund-abuse engine (#937).
//
// Every assertion here protects a real person from the engine, not the platform from a
// customer: a rate that is not a rate below MinOrders, a victim of one bad kitchen who
// must never be banded up, and a shadow rollout that is genuinely inert. The abuse
// detection itself is the easy half.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func setupRiskDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	// Raw DDL rather than AutoMigrate: the models carry Postgres `gen_random_uuid()`
	// column defaults sqlite cannot parse. Same pattern as the other services tests.
	for _, s := range []string{
		`CREATE TABLE customer_risk_events (id TEXT PRIMARY KEY, customer_id TEXT, kind TEXT,
			source_key TEXT UNIQUE, order_id TEXT, chef_id TEXT, amount REAL, weight REAL,
			reason TEXT, occurred_at DATETIME, created_at DATETIME)`,
		`CREATE TABLE customer_risk_profiles (id TEXT PRIMARY KEY, user_id TEXT UNIQUE,
			window_days INTEGER, orders_in_window INTEGER, issues_reported INTEGER,
			issues_auto_refunded INTEGER, issues_rejected INTEGER, cancellations_late INTEGER,
			meal_plan_refunds INTEGER, delivery_failure_claims INTEGER, no_evidence_claims INTEGER,
			distinct_chefs_claimed INTEGER, distinct_chefs_ordered INTEGER, refunded_amount REAL,
			spent_amount REAL, refunded_share REAL, claim_rate REAL, score REAL, band TEXT,
			status TEXT, status_reason TEXT, flagged_at DATETIME, reviewed_at DATETIME,
			reviewed_by TEXT, last_event_at DATETIME, recomputed_at DATETIME,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE customer_risk_actions (id TEXT PRIMARY KEY, user_id TEXT, actor_id TEXT,
			from_status TEXT, to_status TEXT, note TEXT, score_snapshot REAL, band_snapshot TEXT,
			created_at DATETIME)`,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT UNIQUE, value TEXT,
			type TEXT, updated_by TEXT, updated_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT,
			aggregate_type TEXT, aggregate_id TEXT, payload TEXT, status TEXT, attempts INT,
			last_error TEXT, next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME,
			published_at DATETIME)`,
		// Only the columns the confounder guard reads.
		`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, total_orders INTEGER, issue_count INTEGER)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func riskSetting(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Create(&models.PlatformSettings{Key: key, Value: value}).Error)
}

// seedOrders records n paid orders spread across the chefs given, round-robin.
func seedOrders(t *testing.T, db *gorm.DB, customerID uuid.UUID, n int, chefs ...uuid.UUID) {
	t.Helper()
	for i := range n {
		chef := chefs[i%len(chefs)]
		_, err := RecordRiskEvent(db, RecordRiskEventInput{
			CustomerID: customerID,
			Kind:       models.RiskOrderPlaced,
			SourceKey:  "order:" + uuid.New().String(),
			ChefID:     &chef,
			Amount:     500,
		})
		require.NoError(t, err)
	}
}

// seedClaims records n reported issues against the chefs given, round-robin.
func seedClaims(t *testing.T, db *gorm.DB, customerID uuid.UUID, n int, chefs ...uuid.UUID) {
	t.Helper()
	for i := range n {
		chef := chefs[i%len(chefs)]
		_, err := RecordRiskEvent(db, RecordRiskEventInput{
			CustomerID: customerID,
			Kind:       models.RiskIssueReported,
			SourceKey:  "issue:" + uuid.New().String(),
			ChefID:     &chef,
			Amount:     250,
		})
		require.NoError(t, err)
	}
}

// A retried or concurrent write of the same act counts once. Without this the counter
// inflates on every refund-path retry and bands honest customers.
func TestRecordRiskEvent_IdempotentOnSourceKey(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	in := RecordRiskEventInput{
		CustomerID: customer,
		Kind:       models.RiskIssueReported,
		SourceKey:  "issue:fixed-key",
		Amount:     120,
	}

	first, err := RecordRiskEvent(db, in)
	require.NoError(t, err)
	require.True(t, first)

	second, err := RecordRiskEvent(db, in)
	require.NoError(t, err)
	require.False(t, second, "a replayed event must not insert a second row")

	var n int64
	db.Model(&models.CustomerRiskEvent{}).Where("customer_id = ?", customer).Count(&n)
	require.EqualValues(t, 1, n)
}

// The whole premise: judge the RATE, not the count. Three claims across ninety orders is
// an ordinary customer and must stay `normal`.
func TestScore_ManyOrdersFewClaimsStaysNormal(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	chefs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	seedOrders(t, db, customer, 90, chefs...)
	seedClaims(t, db, customer, 3, chefs...)

	p, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)
	require.Equal(t, models.BandNormal, p.Band)
	require.Equal(t, 90, p.OrdersInWindow)
	require.Equal(t, 3, p.IssuesReported)
}

// The same three claims across four orders is not the same thing at all.
func TestScore_FewOrdersManyClaimsBandsUp(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	chefs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	seedOrders(t, db, customer, 4, chefs...)
	seedClaims(t, db, customer, 3, chefs...)

	p, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)
	require.True(t, p.Band.AtLeast(models.BandElevated),
		"3 claims in 4 orders should band at least elevated, got %s (score %.1f)", p.Band, p.Score)
}

// Below MinOrders nobody is scored. A rate over two orders is not a rate, and a new
// customer with one bad meal is not a suspect.
func TestScore_BelowMinOrdersIsNeverScored(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	chef := uuid.New()

	seedOrders(t, db, customer, 2, chef)
	seedClaims(t, db, customer, 2, chef)

	p, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)
	require.Equal(t, models.BandNormal, p.Band)
	require.Zero(t, p.Score)
}

// Claims concentrated on ONE kitchen that is itself a known problem are the KITCHEN's
// fault. Banding the customer up here would punish the victims of a bad chef — the single
// most damaging false positive this engine can produce.
func TestScore_ConcentratedClaimsOnBadChefAreDiscounted(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	badChef := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, total_orders, issue_count) VALUES (?, ?, ?)`,
		badChef.String(), 100, 40).Error)

	seedOrders(t, db, customer, 6, badChef)
	seedClaims(t, db, customer, 5, badChef)

	p, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)
	require.False(t, p.Band.AtLeast(models.BandElevated),
		"claims against a high-issue-rate kitchen must not band the customer up, got %s (score %.1f)", p.Band, p.Score)
}

// The same claim pattern spread across several kitchens is the serial-claimer signature —
// it cannot be that five different kitchens all failed one person.
func TestScore_SpreadClaimsAcrossChefsBandsUp(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	chefs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}

	seedOrders(t, db, customer, 6, chefs...)
	seedClaims(t, db, customer, 5, chefs...)

	p, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)
	require.True(t, p.Band.AtLeast(models.BandElevated),
		"5 claims across 5 kitchens should band up, got %s (score %.1f)", p.Band, p.Score)
	require.Equal(t, 5, p.DistinctChefsClaimed)
}

// Events outside the rolling window do not count — a customer is judged on recent
// behaviour, not forever.
func TestScore_EventsOutsideWindowAreExcluded(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	chef := uuid.New()

	seedOrders(t, db, customer, 10, chef)
	old := time.Now().AddDate(0, 0, -200)
	for range 5 {
		_, err := RecordRiskEvent(db, RecordRiskEventInput{
			CustomerID: customer,
			Kind:       models.RiskIssueReported,
			SourceKey:  "issue:" + uuid.New().String(),
			ChefID:     &chef,
			OccurredAt: old,
		})
		require.NoError(t, err)
	}

	p, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)
	require.Zero(t, p.IssuesReported, "claims older than the window must not count")
	require.Equal(t, models.BandNormal, p.Band)
}

// Reaching `severe` raises a flag for review and stages the admin alert. Nothing happens
// to the customer — a human still has to look.
func TestRecompute_SevereAutoFlagsAndAlerts(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	chefs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}

	seedOrders(t, db, customer, 5, chefs...)
	seedClaims(t, db, customer, 5, chefs...)
	for range 3 {
		_, err := RecordRiskEvent(db, RecordRiskEventInput{
			CustomerID: customer,
			Kind:       models.RiskIssueRejected,
			SourceKey:  "issue-rejected:" + uuid.New().String(),
			ChefID:     &chefs[0],
		})
		require.NoError(t, err)
	}

	p, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)
	require.Equal(t, models.BandSevere, p.Band, "score %.1f", p.Score)
	require.Equal(t, models.RiskStatusFlagged, p.Status)

	var staged int64
	db.Model(&models.OutboxEvent{}).Where("subject = ?", SubjectRiskCustomerFlagged).Count(&staged)
	require.EqualValues(t, 1, staged, "the admin alert must be staged in the outbox")
}

// An admin decision is never overwritten by the engine on the next recompute.
func TestRecompute_DoesNotOverwriteAnAdminDecision(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	chefs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}

	seedOrders(t, db, customer, 5, chefs...)
	seedClaims(t, db, customer, 5, chefs...)
	p, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)

	p.Status = models.RiskStatusCleared
	require.NoError(t, db.Save(p).Error)

	seedClaims(t, db, customer, 1, chefs[0])
	again, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)
	require.Equal(t, models.RiskStatusCleared, again.Status)
}

// Shadow mode must be genuinely inert: score, flag, alert — but change nothing a customer
// can feel. This is what makes it safe to turn the engine on before it is calibrated.
func TestEvaluateIssueClaim_ShadowModeChangesNothing(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	chefs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}

	seedOrders(t, db, customer, 5, chefs...)
	seedClaims(t, db, customer, 5, chefs...)
	p, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)
	require.True(t, p.Band.AtLeast(models.BandElevated))

	d := EvaluateIssueClaim(db, customer)
	require.True(t, d.Allowed)
	require.True(t, d.AutoRefundAllowed, "enforcement is off — the auto-refund must still run")
	require.Equal(t, p.Band, d.Band, "the band is still reported so shadow mode is observable")
}

// With enforcement on, an elevated customer keeps the refund but loses the INSTANT one.
func TestEvaluateIssueClaim_ElevatedSuppressesAutoRefundOnly(t *testing.T) {
	db := setupRiskDB(t)
	riskSetting(t, db, "customer_risk.enforcement_enabled", "true")
	customer := uuid.New()
	chefs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}

	seedOrders(t, db, customer, 5, chefs...)
	seedClaims(t, db, customer, 5, chefs...)
	_, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)

	d := EvaluateIssueClaim(db, customer)
	require.True(t, d.Allowed, "the claim must still be accepted — only the automation stops")
	require.False(t, d.AutoRefundAllowed)
}

// Only an admin-set status refuses a claim outright.
func TestEvaluateIssueClaim_RestrictedRefusesTheClaim(t *testing.T) {
	db := setupRiskDB(t)
	riskSetting(t, db, "customer_risk.enforcement_enabled", "true")
	customer := uuid.New()

	require.NoError(t, db.Create(&models.CustomerRiskProfile{
		UserID: customer, Band: models.BandNormal, Status: models.RiskStatusRestricted,
	}).Error)

	d := EvaluateIssueClaim(db, customer)
	require.False(t, d.Allowed)
	require.False(t, d.AutoRefundAllowed)
}

// A customer nobody has ever scored is always allowed everything.
func TestEvaluateIssueClaim_UnknownCustomerIsUnaffected(t *testing.T) {
	db := setupRiskDB(t)
	riskSetting(t, db, "customer_risk.enforcement_enabled", "true")

	d := EvaluateIssueClaim(db, uuid.New())
	require.True(t, d.Allowed)
	require.True(t, d.AutoRefundAllowed)
	require.Equal(t, models.BandNormal, d.Band)
}

// The chef badge carries a band word and advice — never a score, a count, or a history.
func TestGetChefDisputeSignal_CarriesNoNumbers(t *testing.T) {
	db := setupRiskDB(t)
	riskSetting(t, db, "customer_risk.enforcement_enabled", "true")
	customer := uuid.New()
	require.NoError(t, db.Create(&models.CustomerRiskProfile{
		UserID: customer, Band: models.BandSevere, Status: models.RiskStatusFlagged, Score: 91,
		IssuesReported: 9, OrdersInWindow: 10,
	}).Error)

	sig := GetChefDisputeSignal(db, customer)
	require.NotNil(t, sig)
	require.Equal(t, models.BandSevere, sig.Band)
	require.NotEmpty(t, sig.Advice)
}

// In shadow, and for an ordinary customer, the chef sees nothing at all.
func TestGetChefDisputeSignal_SilentInShadowAndForNormalCustomers(t *testing.T) {
	db := setupRiskDB(t)
	customer := uuid.New()
	require.NoError(t, db.Create(&models.CustomerRiskProfile{
		UserID: customer, Band: models.BandSevere, Status: models.RiskStatusFlagged, Score: 91,
	}).Error)
	require.Nil(t, GetChefDisputeSignal(db, customer), "enforcement is off — chefs must see nothing")

	riskSetting(t, db, "customer_risk.enforcement_enabled", "true")
	ordinary := uuid.New()
	require.NoError(t, db.Create(&models.CustomerRiskProfile{
		UserID: ordinary, Band: models.BandWatch, Status: models.RiskStatusOK,
	}).Error)
	require.Nil(t, GetChefDisputeSignal(db, ordinary))
}

// Config comes from PlatformSettings so ops can retune without a deploy.
func TestGetCustomerRiskConfig_ReadsPlatformSettings(t *testing.T) {
	db := setupRiskDB(t)
	riskSetting(t, db, "customer_risk.window_days", "30")
	riskSetting(t, db, "customer_risk.min_orders", "10")
	riskSetting(t, db, "customer_risk.elevated_score", "55")
	riskSetting(t, db, "customer_risk.suppress_auto_refund_band", "severe")
	riskSetting(t, db, "customer_risk.enforcement_enabled", "1")

	cfg := GetCustomerRiskConfig(db)
	require.Equal(t, 30, cfg.WindowDays)
	require.Equal(t, 10, cfg.MinOrders)
	require.EqualValues(t, 55, cfg.ElevatedScore)
	require.Equal(t, models.BandSevere, cfg.SuppressAutoRefundBand)
	require.True(t, cfg.EnforcementEnabled)
}

// A disabled engine scores nobody, whatever the ledger says.
func TestScore_DisabledEngineScoresNobody(t *testing.T) {
	db := setupRiskDB(t)
	riskSetting(t, db, "customer_risk.enabled", "false")
	customer := uuid.New()
	chefs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	seedOrders(t, db, customer, 5, chefs...)
	seedClaims(t, db, customer, 5, chefs...)

	p, err := RecomputeCustomerRisk(db, customer)
	require.NoError(t, err)
	require.Zero(t, p.Score)
	require.Equal(t, models.BandNormal, p.Band)
}

// TrackRiskEvent is called from inside live refund transactions. A failure in it must
// never take the enclosing transaction down with it.
func TestTrackRiskEvent_FailureDoesNotBreakTheCallersTransaction(t *testing.T) {
	db := setupRiskDB(t)
	require.NoError(t, db.Exec(`DROP TABLE customer_risk_events`).Error)

	customer := uuid.New()
	committed := false
	err := db.Transaction(func(tx *gorm.DB) error {
		TrackRiskEvent(tx, RecordRiskEventInput{
			CustomerID: customer,
			Kind:       models.RiskIssueReported,
			SourceKey:  "issue:" + uuid.New().String(),
		})
		// The caller's own work must still commit.
		if err := tx.Create(&models.PlatformSettings{Key: "probe", Value: "ok"}).Error; err != nil {
			return err
		}
		committed = true
		return nil
	})
	require.NoError(t, err)
	require.True(t, committed)

	var n int64
	db.Model(&models.PlatformSettings{}).Where("key = ?", "probe").Count(&n)
	require.EqualValues(t, 1, n, "the caller's write must survive a risk-accounting failure")
}
