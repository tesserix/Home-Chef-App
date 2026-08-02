package handlers

// order_issue_risk_test.go — the customer-scoped guard on the report endpoint (#937).
//
// Pins the two things that must be true of the ladder: it is inert until enforcement is
// switched on, and when it does bite it takes away the AUTOMATIC refund, never the claim.

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// addRiskTables extends the order-issue fixture with the risk ledger and the wallet the
// auto-refund credits, so a suppressed refund is distinguishable from a failed one.
func addRiskTables(t *testing.T, db *gorm.DB) {
	t.Helper()
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
		`CREATE TABLE wallets (id TEXT PRIMARY KEY, user_id TEXT UNIQUE, balance REAL DEFAULT 0,
			currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE wallet_txns (id TEXT PRIMARY KEY, wallet_id TEXT, user_id TEXT, type TEXT,
			source TEXT, amount REAL, balance_after REAL, currency TEXT, order_id TEXT, reason TEXT,
			created_by TEXT, idempotency_key TEXT UNIQUE, created_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
}

// seedRiskOrder writes a delivered, paid, refundable order with one ₹200 line.
func seedRiskOrder(t *testing.T, db *gorm.DB, customerID, chefID uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()
	orderID, itemID := uuid.New(), uuid.New()
	now := time.Now()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, chef_id, payment_status, status,
		subtotal, tax, total, refund_amount, delivered_at, created_at, updated_at)
		VALUES (?, ?, ?, 'completed', 'delivered', 200, 0, 200, 0, ?, ?, ?)`,
		orderID.String(), customerID.String(), chefID.String(), now, now, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, name, price, quantity, subtotal)
		VALUES (?, ?, 'Dal', 200, 1, 200)`, itemID.String(), orderID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, issue_count) VALUES (?, ?, 0)`,
		chefID.String(), uuid.New().String()).Error)
	return orderID, itemID
}

func setRiskProfile(t *testing.T, db *gorm.DB, userID uuid.UUID, band models.RiskBand, status models.RiskStatus) {
	t.Helper()
	require.NoError(t, db.Create(&models.CustomerRiskProfile{
		UserID: userID, Band: band, Status: status, WindowDays: 90,
	}).Error)
}

// seedClaimHistory writes a real ledger of paid orders and prior claims across several
// kitchens. The band has to be EARNED from these events — the handler rescores before it
// decides, so a band planted straight onto the profile would be recomputed away.
func seedClaimHistory(t *testing.T, db *gorm.DB, customerID uuid.UUID, orders, claims int) {
	t.Helper()
	chefs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	write := func(kind models.CustomerRiskEventKind, n int, amount float64) {
		for i := range n {
			chef := chefs[i%len(chefs)]
			require.NoError(t, db.Create(&models.CustomerRiskEvent{
				CustomerID: customerID,
				Kind:       kind,
				SourceKey:  string(kind) + ":" + uuid.New().String(),
				ChefID:     &chef,
				Amount:     amount,
				Weight:     services.RiskEventWeights[kind],
				OccurredAt: time.Now().Add(-24 * time.Hour),
			}).Error)
		}
	}
	write(models.RiskOrderPlaced, orders, 500)
	write(models.RiskIssueReported, claims, 250)
}

func enableRiskEnforcement(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Create(&models.PlatformSettings{
		Key: "customer_risk.enforcement_enabled", Value: "true",
	}).Error)
}

// A restricted account cannot open a new claim — and nothing is written, so the
// restriction cannot be used to quietly inflate anyone's counters.
func TestReportIssue_RestrictedAccountIsRefused(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	addRiskTables(t, db)

	customerID, chefID := uuid.New(), uuid.New()
	orderID, itemID := seedRiskOrder(t, db, customerID, chefID)
	enableRiskEnforcement(t, db)
	setRiskProfile(t, db, customerID, models.BandSevere, models.RiskStatusRestricted)

	w := reportIssueReq(t, customerID, orderID, "reason=missing_item&affectedItemIds="+itemID.String())
	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "contact support")

	var issues int64
	db.Model(&models.OrderIssue{}).Where("order_id = ?", orderID).Count(&issues)
	require.Zero(t, issues, "a refused claim must not be filed")
}

// An elevated customer still files the claim and is still owed the money — the refund
// simply stops being automatic and goes to assisted review.
func TestReportIssue_ElevatedLosesTheAutoRefundNotTheClaim(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	addRiskTables(t, db)

	customerID, chefID := uuid.New(), uuid.New()
	orderID, itemID := seedRiskOrder(t, db, customerID, chefID)
	enableRiskEnforcement(t, db)
	seedClaimHistory(t, db, customerID, 5, 5)

	w := reportIssueReq(t, customerID, orderID, "reason=missing_item&affectedItemIds="+itemID.String())
	require.Equal(t, http.StatusCreated, w.Code)

	profile := services.GetCustomerRiskProfile(db, customerID)
	require.NotNil(t, profile)
	require.True(t, profile.Band.AtLeast(models.BandElevated),
		"fixture must earn an elevated band, got %s (score %.1f)", profile.Band, profile.Score)

	var body struct {
		Status       string  `json:"status"`
		RefundAmount float64 `json:"refundAmount"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, string(models.IssuePending), body.Status, "must await assisted review, not auto-refund")
	require.Zero(t, body.RefundAmount)

	var credited int64
	db.Table("wallet_txns").Where("user_id = ?", customerID.String()).Count(&credited)
	require.Zero(t, credited, "no money may move automatically for an elevated customer")
}

// The same customer, with enforcement off, is refunded instantly exactly as before. This
// is the assertion that makes the shadow rollout safe.
func TestReportIssue_ShadowModeStillAutoRefunds(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	addRiskTables(t, db)

	customerID, chefID := uuid.New(), uuid.New()
	orderID, itemID := seedRiskOrder(t, db, customerID, chefID)
	seedClaimHistory(t, db, customerID, 5, 5)

	w := reportIssueReq(t, customerID, orderID, "reason=missing_item&affectedItemIds="+itemID.String())
	require.Equal(t, http.StatusCreated, w.Code)

	var body struct {
		Status       string  `json:"status"`
		RefundAmount float64 `json:"refundAmount"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, string(models.IssueAutoRefunded), body.Status)
	require.EqualValues(t, 200, body.RefundAmount)
}

// Filing a claim appends to the ledger — including the photo-free-complaint signal, which
// only exists at report time.
func TestReportIssue_AppendsRiskLedgerEntries(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	addRiskTables(t, db)

	customerID, chefID := uuid.New(), uuid.New()
	orderID, itemID := seedRiskOrder(t, db, customerID, chefID)

	w := reportIssueReq(t, customerID, orderID, "reason=quality_issue&affectedItemIds="+itemID.String())
	require.Equal(t, http.StatusCreated, w.Code)

	var kinds []string
	require.NoError(t, db.Model(&models.CustomerRiskEvent{}).
		Where("customer_id = ?", customerID).Pluck("kind", &kinds).Error)
	require.Contains(t, kinds, string(models.RiskIssueReported))
	require.Contains(t, kinds, string(models.RiskNoEvidenceClaim),
		"a photo-free quality complaint is a report-time signal and must be recorded")
}
