package services

import (
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// customer_risk.go — the customer refund-abuse engine (#937). Records every money-out
// act a customer causes into an idempotent ledger, derives a RATE-based score over a
// rolling window, and answers one question for the issue-refund path: should this
// customer's next claim still be refunded automatically?
//
// Rate, never count: three claims in ninety orders is a normal customer; three in four
// is not. The engine flags — it never blocks. Blocking is an admin decision.

// CustomerRiskConfig is the runtime policy, read from PlatformSettings `customer_risk.*`
// so ops retunes thresholds without a deploy. Mirrors GetIssueConfig.
type CustomerRiskConfig struct {
	Enabled    bool `json:"enabled"`
	WindowDays int  `json:"windowDays"`
	MinOrders  int  `json:"minOrders"`

	WatchScore    float64 `json:"watchScore"`
	ElevatedScore float64 `json:"elevatedScore"`
	SevereScore   float64 `json:"severeScore"`

	// Normalisation points — the value at which a component contributes its full weight.
	ClaimRateFull     float64 `json:"claimRateFull"`
	RefundedShareFull float64 `json:"refundedShareFull"`
	SpreadFullChefs   float64 `json:"spreadFullChefs"`
	SignalFull        float64 `json:"signalFull"`

	WeightClaimRate float64 `json:"weightClaimRate"`
	WeightMoney     float64 `json:"weightMoney"`
	WeightSpread    float64 `json:"weightSpread"`
	WeightSignal    float64 `json:"weightSignal"`

	// SuppressAutoRefundBand is the band at which a claim stops being auto-refunded and
	// is routed to assisted review instead.
	SuppressAutoRefundBand models.RiskBand `json:"suppressAutoRefundBand"`
	// EnforcementEnabled gates every behaviour change. OFF on day one: score in shadow,
	// look at who it flags on real traffic, tune, then enforce.
	EnforcementEnabled bool `json:"enforcementEnabled"`

	// ChefIssueRateFloor is the platform-wide issue rate above which a kitchen is treated
	// as the likely cause of a customer's concentrated claims (the confounder guard).
	ChefIssueRateFloor float64 `json:"chefIssueRateFloor"`
	ChefMinOrders      int     `json:"chefMinOrders"`
}

// DefaultCustomerRiskConfig is the policy applied when no PlatformSettings rows exist.
func DefaultCustomerRiskConfig() CustomerRiskConfig {
	return CustomerRiskConfig{
		Enabled:                true,
		WindowDays:             90,
		MinOrders:              4,
		WatchScore:             40,
		ElevatedScore:          60,
		SevereScore:            80,
		ClaimRateFull:          0.5,
		RefundedShareFull:      0.4,
		SpreadFullChefs:        4,
		SignalFull:             1.0,
		WeightClaimRate:        0.40,
		WeightMoney:            0.25,
		WeightSpread:           0.15,
		WeightSignal:           0.20,
		SuppressAutoRefundBand: models.BandElevated,
		EnforcementEnabled:     false,
		ChefIssueRateFloor:     0.15,
		ChefMinOrders:          20,
	}
}

// GetCustomerRiskConfig reads the policy from PlatformSettings, falling back per-key to
// the defaults. A read failure yields the defaults rather than an error: risk scoring
// must never be the reason a refund path fails.
func GetCustomerRiskConfig(db *gorm.DB) CustomerRiskConfig {
	cfg := DefaultCustomerRiskConfig()
	var settings []models.PlatformSettings
	if err := db.Where("key LIKE ?", "customer_risk.%").Find(&settings).Error; err != nil {
		return cfg
	}
	for _, s := range settings {
		switch s.Key {
		case "customer_risk.enabled":
			cfg.Enabled = isTruthySetting(s.Value)
		case "customer_risk.enforcement_enabled":
			cfg.EnforcementEnabled = isTruthySetting(s.Value)
		case "customer_risk.window_days":
			setIntSetting(&cfg.WindowDays, s.Value)
		case "customer_risk.min_orders":
			setIntSetting(&cfg.MinOrders, s.Value)
		case "customer_risk.chef_min_orders":
			setIntSetting(&cfg.ChefMinOrders, s.Value)
		case "customer_risk.watch_score":
			setFloatSetting(&cfg.WatchScore, s.Value)
		case "customer_risk.elevated_score":
			setFloatSetting(&cfg.ElevatedScore, s.Value)
		case "customer_risk.severe_score":
			setFloatSetting(&cfg.SevereScore, s.Value)
		case "customer_risk.claim_rate_full":
			setFloatSetting(&cfg.ClaimRateFull, s.Value)
		case "customer_risk.refunded_share_full":
			setFloatSetting(&cfg.RefundedShareFull, s.Value)
		case "customer_risk.spread_full_chefs":
			setFloatSetting(&cfg.SpreadFullChefs, s.Value)
		case "customer_risk.signal_full":
			setFloatSetting(&cfg.SignalFull, s.Value)
		case "customer_risk.weight_claim_rate":
			setFloatSetting(&cfg.WeightClaimRate, s.Value)
		case "customer_risk.weight_money":
			setFloatSetting(&cfg.WeightMoney, s.Value)
		case "customer_risk.weight_spread":
			setFloatSetting(&cfg.WeightSpread, s.Value)
		case "customer_risk.weight_signal":
			setFloatSetting(&cfg.WeightSignal, s.Value)
		case "customer_risk.chef_issue_rate_floor":
			setFloatSetting(&cfg.ChefIssueRateFloor, s.Value)
		case "customer_risk.suppress_auto_refund_band":
			if b := models.RiskBand(s.Value); models.ValidRiskBand(b) {
				cfg.SuppressAutoRefundBand = b
			}
		}
	}
	return cfg
}

func isTruthySetting(v string) bool { return v == "true" || v == "1" }

func setIntSetting(dst *int, v string) {
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		*dst = n
	}
}

func setFloatSetting(dst *float64, v string) {
	if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
		*dst = f
	}
}

// RiskEventWeights is the per-kind scoring weight. A REJECTED claim is the strongest
// signal we have — an admin looked at the evidence and said no — so it outweighs the
// report itself. `order_placed` carries no weight: it is the denominator.
var RiskEventWeights = map[models.CustomerRiskEventKind]float64{
	models.RiskOrderPlaced:           0,
	models.RiskIssueReported:         1.0,
	models.RiskIssueAutoRefunded:     0.5,
	models.RiskIssueResolved:         0.3,
	models.RiskIssueRejected:         2.0,
	models.RiskNoEvidenceClaim:       0.5,
	models.RiskLateWindowClaim:       0.5,
	models.RiskCancelLate:            0.75,
	models.RiskMealPlanRefund:        0.5,
	models.RiskDeliveryFailureClaim:  1.0,
	models.RiskDeliveryFaultCustomer: 2.0,
	models.RiskChargeback:            3.0,
}

// RecordRiskEventInput is one act to append to a customer's ledger.
type RecordRiskEventInput struct {
	CustomerID uuid.UUID
	Kind       models.CustomerRiskEventKind
	SourceKey  string
	OrderID    *uuid.UUID
	ChefID     *uuid.UUID
	Amount     float64
	Reason     string
	OccurredAt time.Time
}

// RecordRiskEvent appends one event, exactly once. The unique SourceKey plus ON CONFLICT
// DO NOTHING makes a retried or concurrent write a no-op, so a refund path that runs
// twice cannot double-count its own customer.
//
// Returns whether a row was actually inserted. Callers should treat any error as
// non-fatal: risk accounting must never fail a refund, cancellation, or order.
func RecordRiskEvent(db *gorm.DB, in RecordRiskEventInput) (bool, error) {
	if in.CustomerID == uuid.Nil || in.SourceKey == "" {
		return false, nil
	}
	occurred := in.OccurredAt
	if occurred.IsZero() {
		occurred = time.Now()
	}
	ev := models.CustomerRiskEvent{
		CustomerID: in.CustomerID,
		Kind:       in.Kind,
		SourceKey:  in.SourceKey,
		OrderID:    in.OrderID,
		ChefID:     in.ChefID,
		Amount:     in.Amount,
		Weight:     RiskEventWeights[in.Kind],
		Reason:     in.Reason,
		OccurredAt: occurred,
	}
	res := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "source_key"}}, DoNothing: true}).Create(&ev)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// TrackRiskEvent records an event and refreshes the profile, swallowing every error into
// a log line. This is the entry point for the money paths — they call it for its effect
// on the ledger, never for a result they act on.
//
// The work runs inside its own transaction, which GORM turns into a SAVEPOINT when the
// caller is already in one. That is the whole reason this is safe to call from inside a
// payment or refund transaction: on Postgres a failed statement poisons the enclosing
// transaction, so swallowing the error alone would not be enough — the savepoint is what
// keeps a risk-accounting failure from rolling back a customer's refund.
func TrackRiskEvent(db *gorm.DB, in RecordRiskEventInput) {
	err := db.Transaction(func(tx *gorm.DB) error {
		inserted, err := RecordRiskEvent(tx, in)
		if err != nil || !inserted {
			return err // duplicate → the profile already reflects it
		}
		// order_placed fires on the payment-completion path, which is hot and already
		// holding an order lock — it only writes the row and skips the rescore. The
		// denominator is read straight from the ledger by every recompute that follows
		// (EvaluateIssueClaim on the next claim, or an admin opening the profile), so the
		// score is never computed against a stale order count.
		if in.Kind == models.RiskOrderPlaced {
			return nil
		}
		_, err = RecomputeCustomerRisk(tx, in.CustomerID)
		return err
	})
	if err != nil {
		log.Printf("customer risk: track %s for %s failed: %v", in.Kind, in.CustomerID, err)
	}
}

// riskAggregate is the raw ledger roll-up for one customer over the window.
type riskAggregate struct {
	orders        int
	spent         float64
	refunded      float64
	weightedSum   float64
	counts        map[models.CustomerRiskEventKind]int
	chefsOrdered  map[uuid.UUID]bool
	chefsClaimed  map[uuid.UUID]int
	lastEventAt   *time.Time
	claimTopChef  uuid.UUID
	claimTopCount int
}

// claimKinds are the event kinds that count as a customer-initiated claim for the
// claim-rate numerator. Auto/resolved refunds are outcomes of a report already counted
// here, so including them would double-count one claim.
var claimKinds = map[models.CustomerRiskEventKind]bool{
	models.RiskIssueReported:        true,
	models.RiskDeliveryFailureClaim: true,
	models.RiskChargeback:           true,
}

// aggregateRisk rolls the ledger window into counters. Chef spread is tracked as a map
// so a customer who claims five times against one kitchen reads very differently from
// one who claims once against five.
func aggregateRisk(db *gorm.DB, customerID uuid.UUID, since time.Time) (riskAggregate, error) {
	agg := riskAggregate{
		counts:       map[models.CustomerRiskEventKind]int{},
		chefsOrdered: map[uuid.UUID]bool{},
		chefsClaimed: map[uuid.UUID]int{},
	}
	var events []models.CustomerRiskEvent
	if err := db.Where("customer_id = ? AND occurred_at >= ?", customerID, since).
		Order("occurred_at ASC").Find(&events).Error; err != nil {
		return agg, err
	}
	for i := range events {
		ev := events[i]
		agg.counts[ev.Kind]++
		agg.weightedSum += ev.Weight
		if agg.lastEventAt == nil || ev.OccurredAt.After(*agg.lastEventAt) {
			at := ev.OccurredAt
			agg.lastEventAt = &at
		}
		if ev.Kind == models.RiskOrderPlaced {
			agg.orders++
			agg.spent += ev.Amount
			if ev.ChefID != nil {
				agg.chefsOrdered[*ev.ChefID] = true
			}
			continue
		}
		agg.refunded += ev.Amount
		if claimKinds[ev.Kind] && ev.ChefID != nil {
			agg.chefsClaimed[*ev.ChefID]++
			if n := agg.chefsClaimed[*ev.ChefID]; n > agg.claimTopCount {
				agg.claimTopCount, agg.claimTopChef = n, *ev.ChefID
			}
		}
	}
	return agg, nil
}

// norm clamps x/full into 0..1. A non-positive `full` disables the component.
func norm(x, full float64) float64 {
	if full <= 0 {
		return 0
	}
	if v := x / full; v < 1 {
		if v < 0 {
			return 0
		}
		return v
	}
	return 1
}

// bandFor maps a score to a band using the configured thresholds.
func bandFor(cfg CustomerRiskConfig, score float64) models.RiskBand {
	switch {
	case score >= cfg.SevereScore:
		return models.BandSevere
	case score >= cfg.ElevatedScore:
		return models.BandElevated
	case score >= cfg.WatchScore:
		return models.BandWatch
	}
	return models.BandNormal
}

// chefIsHighIssueRate reports whether a kitchen's own platform-wide issue rate is high
// enough that a customer's claims against it are more likely the kitchen's fault than
// the customer's. Needs a meaningful order count — a chef with 2 orders and 1 issue is
// noise, not a signal.
func chefIsHighIssueRate(db *gorm.DB, cfg CustomerRiskConfig, chefID uuid.UUID) bool {
	if chefID == uuid.Nil || cfg.ChefIssueRateFloor <= 0 {
		return false
	}
	var chef models.ChefProfile
	if err := db.Select("id", "total_orders", "issue_count").First(&chef, "id = ?", chefID).Error; err != nil {
		return false
	}
	if chef.TotalOrders < cfg.ChefMinOrders {
		return false
	}
	return float64(chef.IssueCount)/float64(chef.TotalOrders) >= cfg.ChefIssueRateFloor
}

// RecomputeCustomerRisk rebuilds a customer's profile from the event ledger and upserts
// it. The ledger is the only source of truth; this is a cache the admin queue can sort.
//
// The profile row is created on first sight so an admin can act on a customer who has
// never tripped a threshold. Score is 0 and band `normal` until MinOrders is met.
func RecomputeCustomerRisk(db *gorm.DB, customerID uuid.UUID) (*models.CustomerRiskProfile, error) {
	cfg := GetCustomerRiskConfig(db)
	since := time.Now().AddDate(0, 0, -cfg.WindowDays)

	agg, err := aggregateRisk(db, customerID, since)
	if err != nil {
		return nil, err
	}

	profile := models.CustomerRiskProfile{UserID: customerID}
	if err := db.Where("user_id = ?", customerID).First(&profile).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		profile = models.CustomerRiskProfile{UserID: customerID, Status: models.RiskStatusOK, Band: models.BandNormal}
	}

	profile.WindowDays = cfg.WindowDays
	profile.OrdersInWindow = agg.orders
	profile.IssuesReported = agg.counts[models.RiskIssueReported]
	profile.IssuesAutoRefunded = agg.counts[models.RiskIssueAutoRefunded]
	profile.IssuesRejected = agg.counts[models.RiskIssueRejected]
	profile.CancellationsLate = agg.counts[models.RiskCancelLate]
	profile.MealPlanRefunds = agg.counts[models.RiskMealPlanRefund]
	profile.DeliveryFailureClaims = agg.counts[models.RiskDeliveryFailureClaim]
	profile.NoEvidenceClaims = agg.counts[models.RiskNoEvidenceClaim]
	profile.DistinctChefsOrdered = len(agg.chefsOrdered)
	profile.DistinctChefsClaimed = len(agg.chefsClaimed)
	profile.RefundedAmount = models.RoundAmount(agg.refunded)
	profile.SpentAmount = models.RoundAmount(agg.spent)
	profile.LastEventAt = agg.lastEventAt
	profile.RecomputedAt = time.Now()

	claims := agg.counts[models.RiskIssueReported] + agg.counts[models.RiskDeliveryFailureClaim] + agg.counts[models.RiskChargeback]
	if agg.orders > 0 {
		profile.ClaimRate = float64(claims) / float64(agg.orders)
	} else {
		profile.ClaimRate = 0
	}
	if agg.spent > 0 {
		profile.RefundedShare = agg.refunded / agg.spent
	} else {
		profile.RefundedShare = 0
	}

	profile.Score, profile.Band = scoreProfile(db, cfg, agg, profile)

	// A profile that reaches `severe` while nobody is looking gets flagged for review.
	// Only from a clean status: an admin's own decision is never overwritten by the engine.
	justFlagged := profile.Band == models.BandSevere && profile.Status == models.RiskStatusOK
	if justFlagged {
		now := time.Now()
		profile.Status = models.RiskStatusFlagged
		profile.FlaggedAt = &now
		profile.StatusReason = "auto-flagged: score reached severe"
	}

	// First sight upserts on user_id: two concurrent recomputes for the same customer both
	// miss the row above, and a plain insert would make the loser fail on the unique index.
	if profile.ID == uuid.Nil {
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			UpdateAll: true,
		}).Create(&profile).Error; err != nil {
			return nil, err
		}
	} else if err := db.Save(&profile).Error; err != nil {
		return nil, err
	}
	// Staged in the caller's transaction (transactional outbox) so the alert is never lost
	// if the process dies right after the flag commits.
	if justFlagged {
		if err := EnqueueEvent(db, SubjectRiskCustomerFlagged, "risk.customer_flagged", customerID, map[string]any{
			"userId":         customerID.String(),
			"score":          profile.Score,
			"band":           string(profile.Band),
			"issuesReported": profile.IssuesReported,
			"ordersInWindow": profile.OrdersInWindow,
			"refundedAmount": profile.RefundedAmount,
		}); err != nil {
			log.Printf("customer risk: enqueue flag alert for %s failed: %v", customerID, err)
		}
	}
	return &profile, nil
}

// scoreProfile computes the 0-100 score and its band.
//
// Two guards keep this honest, and both matter more than the arithmetic:
//   - below MinOrders nobody is scored, because a rate over 2 orders is not a rate;
//   - claims concentrated on ONE kitchen that itself has a high platform-wide issue
//     rate are the KITCHEN's problem. Those are discounted and capped below the
//     enforcing bands, so we never punish the customers of a bad chef.
func scoreProfile(db *gorm.DB, cfg CustomerRiskConfig, agg riskAggregate, p models.CustomerRiskProfile) (float64, models.RiskBand) {
	if !cfg.Enabled || agg.orders < cfg.MinOrders {
		return 0, models.BandNormal
	}

	perOrderSignal := agg.weightedSum / float64(agg.orders)
	score := 100 * (cfg.WeightClaimRate*norm(p.ClaimRate, cfg.ClaimRateFull) +
		cfg.WeightMoney*norm(p.RefundedShare, cfg.RefundedShareFull) +
		cfg.WeightSpread*norm(float64(p.DistinctChefsClaimed), cfg.SpreadFullChefs) +
		cfg.WeightSignal*norm(perOrderSignal, cfg.SignalFull))

	// Confounder guard: every claim landed on a single kitchen, and that kitchen is a
	// known problem. Halve the score and cap the band at `watch` (which enforces
	// nothing) so the profile stays visible to admins without ever gating the customer.
	if p.DistinctChefsClaimed == 1 && chefIsHighIssueRate(db, cfg, agg.claimTopChef) {
		score /= 2
		band := bandFor(cfg, score)
		if band.AtLeast(models.BandElevated) {
			band = models.BandWatch
		}
		return models.RoundAmount(score), band
	}

	return models.RoundAmount(score), bandFor(cfg, score)
}

// GetCustomerRiskProfile reads a profile without creating one. Returns nil when the
// customer has never been scored — the normal case for almost everybody.
func GetCustomerRiskProfile(db *gorm.DB, userID uuid.UUID) *models.CustomerRiskProfile {
	var p models.CustomerRiskProfile
	if err := db.Where("user_id = ?", userID).First(&p).Error; err != nil {
		return nil
	}
	return &p
}

// ClaimDecision is what the risk engine permits for one customer's next order issue.
// Band is always populated (so shadow mode is observable in logs); the two booleans only
// ever go false once enforcement is switched on.
type ClaimDecision struct {
	Allowed           bool
	AutoRefundAllowed bool
	Band              models.RiskBand
	Status            models.RiskStatus
	Reason            string
}

// EvaluateIssueClaim decides whether a customer may file an order issue, and whether it
// may still be settled automatically.
//
// The ladder is deliberately gentle at the top: an elevated customer keeps every right to
// a refund, they simply lose the INSTANT one and go to assisted review. That costs an
// honest customer a day and costs a serial claimer their entire method, and it can never
// wrongly deny anyone money. Only an admin-set status refuses a claim outright.
func EvaluateIssueClaim(db *gorm.DB, customerID uuid.UUID) ClaimDecision {
	d := ClaimDecision{Allowed: true, AutoRefundAllowed: true, Band: models.BandNormal, Status: models.RiskStatusOK}
	cfg := GetCustomerRiskConfig(db)
	if !cfg.Enabled {
		return d
	}
	// Rescore before deciding. Claims are rare, so this is affordable — and it is what
	// stops a customer who was banded up months and fifty clean orders ago from being
	// judged on a cache nobody refreshed. Falls back to the cached row if the rescore
	// fails: a stale band is better than silently treating everyone as normal.
	profile, err := RecomputeCustomerRisk(db, customerID)
	if err != nil || profile == nil {
		log.Printf("customer risk: rescore before claim for %s failed: %v", customerID, err)
		profile = GetCustomerRiskProfile(db, customerID)
	}
	if profile == nil {
		return d
	}
	d.Band, d.Status = profile.Band, profile.Status
	if !cfg.EnforcementEnabled {
		return d // shadow: score and observe, change nothing
	}
	if profile.RestrictsClaims() {
		d.Allowed, d.AutoRefundAllowed = false, false
		d.Reason = "account restricted from filing new claims"
		return d
	}
	if profile.Band.AtLeast(cfg.SuppressAutoRefundBand) {
		d.AutoRefundAllowed = false
		d.Reason = "elevated dispute history — routed to assisted review"
	}
	return d
}

// GetChefDisputeSignal returns the badge for a chef's view of an order, or nil when there
// is nothing worth saying. Gated on enforcement so the whole feature has ONE switch: a
// miscalibrated shadow score must not reach a chef and colour how they treat a customer.
func GetChefDisputeSignal(db *gorm.DB, customerID uuid.UUID) *models.DisputeSignal {
	cfg := GetCustomerRiskConfig(db)
	if !cfg.Enabled || !cfg.EnforcementEnabled {
		return nil
	}
	profile := GetCustomerRiskProfile(db, customerID)
	if profile == nil || !profile.Band.AtLeast(models.BandElevated) {
		return nil
	}
	return &models.DisputeSignal{
		Band:   profile.Band,
		Advice: "Photograph the packed order before handover.",
	}
}
