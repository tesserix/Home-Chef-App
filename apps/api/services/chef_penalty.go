package services

// chef_penalty.go — the chef-cancellation levy (#834 item 6, docs/refund-policy-v3-spec.md).
//
// Policy v3: a chef who cancels a confirmed order close to service still refunds the customer
// 100% (that already happens — see handlers/chef_order_cancel.go), and on top of that is
// charged a percentage of the order value, netted off their next weekly settlement and shown
// as a line on it.
//
// NO penalty mechanism of ANY kind existed before this. services/payout_recovery.go recovers
// FAILED payouts; it does not levy. So this file is the whole thing: raise, guard, waive,
// deduct.
//
// TWO GUARDS, both deliberate and both configurable:
//   1. GRACE — the first N cancellations inside a rolling window are exempt. A kitchen fire or
//      a family emergency is not fraud, and auto-fining it with no recourse loses chefs faster
//      than the levy recovers.
//   2. WAIVER — an admin can cancel any raised levy, with the reason recorded.
//
// Everything is idempotent on SourceKey ("chefcancel:<orderID>"), so a retried cancel, a
// concurrent double-submit, or a re-driven webhook levies exactly once.

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ErrPenaltyNotPending — a waiver targeted a levy that is already waived or deducted.
var ErrPenaltyNotPending = errors.New("penalty is not pending")

// ChefCancelPenaltySourceKey is the natural key of an order's cancellation levy.
func ChefCancelPenaltySourceKey(orderID uuid.UUID) string { return "chefcancel:" + orderID.String() }

// ChefCancelPenaltyConfig is the resolved live policy for the levy.
type ChefCancelPenaltyConfig struct {
	Enabled     bool
	Percent     float64
	LeadHours   float64
	GraceCount  int
	GraceWindow time.Duration
}

// ChefCancelPenaltyPolicy resolves the levy config from platform policy, so ops retunes the
// rate, the lead threshold and both guards at runtime rather than through a deploy.
func ChefCancelPenaltyPolicy() ChefCancelPenaltyConfig {
	p := GetPlatformPolicy()
	days := p.ChefCancelPenaltyGraceDays
	if days < 0 {
		days = 0
	}
	return ChefCancelPenaltyConfig{
		Enabled:     p.ChefCancelPenaltyEnabled && p.ChefCancelPenaltyPercent > 0,
		Percent:     p.ChefCancelPenaltyPercent,
		LeadHours:   p.ChefCancelPenaltyLeadHours,
		GraceCount:  p.ChefCancelPenaltyGraceCount,
		GraceWindow: time.Duration(days) * 24 * time.Hour,
	}
}

// LevyChefCancelPenalty raises the cancellation levy for an order the chef has just cancelled.
//
// leadHours is how much notice the customer got (scheduled service time − now); a cancellation
// with no scheduled time is treated as immediate (lead 0), because an on-demand order the chef
// already accepted is being cooked now. basis is the order value the rate applies to.
//
// Returns the raised penalty, or nil when none was due — levy disabled, enough notice given,
// inside the grace allowance, or already levied for this order. A nil penalty is NOT an error:
// every skip reason is a normal outcome.
//
// Best-effort by contract: the caller must not fail a cancellation because the levy could not
// be recorded. The customer refund is the money that matters; a missed levy is recoverable.
func LevyChefCancelPenalty(db *gorm.DB, chefID, userID, orderID uuid.UUID, reference string, basis, leadHours float64) (*models.ChefPenalty, error) {
	cfg := ChefCancelPenaltyPolicy()
	if !cfg.Enabled || basis <= 0 {
		return nil, nil
	}
	if leadHours >= cfg.LeadHours {
		return nil, nil // enough notice — no levy
	}
	amount := Round2(basis * cfg.Percent / 100)
	if amount <= 0 {
		return nil, nil
	}

	now := time.Now().UTC()
	var raised *models.ChefPenalty
	err := db.Transaction(func(tx *gorm.DB) error {
		// Idempotency FIRST: an already-levied order short-circuits before the grace count,
		// so a retry can't consume a second grace slot or raise a second levy.
		var existing models.ChefPenalty
		switch err := tx.Where("source_key = ?", ChefCancelPenaltySourceKey(orderID)).First(&existing).Error; {
		case err == nil:
			return nil // already levied
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}
		// GRACE: count the levies ALREADY RAISED in the window. Waived ones still count —
		// the allowance is for cancellations, not for penalties that happened to stick;
		// otherwise a chef granted a waiver silently earns a second free cancellation.
		if cfg.GraceCount > 0 {
			var used int64
			q := tx.Model(&models.ChefPenalty{}).
				Where("chef_id = ? AND kind = ?", chefID, models.ChefPenaltyCancelLate)
			if cfg.GraceWindow > 0 {
				q = q.Where("occurred_at >= ?", now.Add(-cfg.GraceWindow))
			}
			if err := q.Count(&used).Error; err != nil {
				return err
			}
			if used < int64(cfg.GraceCount) {
				// Record the exempt cancellation as a WAIVED (zero) row so it consumes the
				// allowance and shows on the chef's history — a grace that leaves no trace
				// would be granted again forever.
				grace := models.ChefPenalty{
					ChefID: chefID, UserID: userID, Kind: models.ChefPenaltyCancelLate,
					Status: models.ChefPenaltyWaived, SourceKey: ChefCancelPenaltySourceKey(orderID),
					OrderID: &orderID, Reference: reference, Currency: EarningsCurrency,
					BasisAmount: Round2(basis), RatePercent: cfg.Percent, Amount: 0,
					LeadHours: Round2(leadHours), OccurredAt: now, WaivedAt: &now,
					Reason: fmt.Sprintf("Cancelled %.1fh before service", leadHours),
					WaiveReason: fmt.Sprintf("Within the allowance of %d cancellation(s) per %d days",
						cfg.GraceCount, int(cfg.GraceWindow.Hours()/24)),
				}
				if err := tx.Create(&grace).Error; err != nil {
					return err
				}
				return nil
			}
		}
		p := models.ChefPenalty{
			ChefID: chefID, UserID: userID, Kind: models.ChefPenaltyCancelLate,
			Status: models.ChefPenaltyPending, SourceKey: ChefCancelPenaltySourceKey(orderID),
			OrderID: &orderID, Reference: reference, Currency: EarningsCurrency,
			BasisAmount: Round2(basis), RatePercent: cfg.Percent, Amount: amount,
			LeadHours: Round2(leadHours), OccurredAt: now,
			Reason: fmt.Sprintf("Cancelled %.1fh before service (less than the %.0fh notice window)",
				leadHours, cfg.LeadHours),
		}
		if err := tx.Create(&p).Error; err != nil {
			if isDuplicateKeyErr(err) {
				return nil // concurrent writer levied first
			}
			return err
		}
		raised = &p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return raised, nil
}

// GatewayFeeSourceKey is the natural key of a gateway-fee levy. It reuses the SAME identity
// already guaranteed by gateway_idempotency.go's key builders (identical across a retry of the
// same refund, distinct across different refunds) instead of inventing a new one, so idempotency
// here is exactly as strong as the underlying gateway call's idempotency.
func GatewayFeeSourceKey(logicalRefundKey string) string { return "gatewayfee:" + logicalRefundKey }

// GatewayFeeLevyConfig is the resolved live policy for the gateway-fee levy (#885).
type GatewayFeeLevyConfig struct {
	Enabled         bool
	Percent         float64
	GraceEnabled    bool
	GraceCount      int
	GraceWindow     time.Duration
	StackWithCancel bool
}

// GatewayFeeLevyPolicy resolves the gateway-fee levy config from platform policy, mirroring
// ChefCancelPenaltyPolicy()'s shape.
func GatewayFeeLevyPolicy() GatewayFeeLevyConfig {
	p := GetPlatformPolicy()
	days := p.GatewayFeeLevyGraceDays
	if days < 0 {
		days = 0
	}
	return GatewayFeeLevyConfig{
		Enabled:         p.GatewayFeeLevyEnabled && p.GatewayFeeLevyPercent > 0,
		Percent:         p.GatewayFeeLevyPercent,
		GraceEnabled:    p.GatewayFeeLevyGraceEnabled,
		GraceCount:      p.GatewayFeeLevyGraceCount,
		GraceWindow:     time.Duration(days) * 24 * time.Hour,
		StackWithCancel: p.GatewayFeeLevyStackWithCancelLevy,
	}
}

// LevyGatewayFeePenalty raises the gateway-fee levy (#885) for a Cashfree refund the chef is at
// fault for.
//
// Best-effort by contract (same as LevyChefCancelPenalty — callers must never fail or roll back
// a refund because this could not be recorded). Called from handlers/chef_order_cancel.go
// (CancelOrder, CancelOrderItem, RefundOrder) and handlers/payment.go (InitiateRefund's Cashfree
// branch, chef-initiated only). NOT called for admin-initiated InitiateRefund, the #475
// CancellationRequest arbitration flow, or non-Cashfree providers — decision 3: fault gating is
// chef-fault-only; admin-initiated is ambiguous and defaults to NOT levying because a wrong levy
// takes real money from a chef.
//
// provider is the order's payment provider (models.NormalizeProvider output); only Cashfree
// levies. refundedAmount is the amount actually sent to the gateway for THIS refund — never the
// order's original total. logicalRefundKey is the SAME key used for the underlying gateway
// idempotency (RefundFullIdempotencyKey / RefundLineIdempotencyKey / RefundPartialIdempotencyKey),
// so a retry of the same refund levies at most once.
//
// Returns the raised penalty, or nil when none was due — levy disabled, zero/negative refunded
// amount, non-Cashfree provider, already levied for this logical refund, an order that already
// carries a cancel_late levy (unless StackWithCancel is on), or inside the grace allowance. A nil
// penalty is NOT an error: every skip reason is a normal outcome.
func LevyGatewayFeePenalty(db *gorm.DB, chefID, userID, orderID uuid.UUID, reference, provider string, refundedAmount float64, logicalRefundKey string) (*models.ChefPenalty, error) {
	cfg := GatewayFeeLevyPolicy()
	if !cfg.Enabled || refundedAmount <= 0 {
		return nil, nil
	}
	if provider != models.PaymentProviderCashfree {
		// Cashfree-only per #885 scope. Razorpay/Stripe fee recovery is a noted follow-up,
		// not this change.
		return nil, nil
	}
	amount := Round2(refundedAmount * cfg.Percent / 100)
	if amount <= 0 {
		return nil, nil
	}
	sourceKey := GatewayFeeSourceKey(logicalRefundKey)

	now := time.Now().UTC()
	var raised *models.ChefPenalty
	err := db.Transaction(func(tx *gorm.DB) error {
		// Idempotency FIRST: a retry of the same logical refund short-circuits before the
		// grace/stacking checks, so it can't consume a second grace slot or raise a second
		// levy — mirrors LevyChefCancelPenalty's ordering.
		var existing models.ChefPenalty
		switch err := tx.Where("source_key = ?", sourceKey).First(&existing).Error; {
		case err == nil:
			return nil // already levied
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		// STACKING GUARD: one cancellation is one penalty event (decision 6) — an order that
		// already raised a cancel_late levy (any status; a waived grace row still counts)
		// does not also raise a gateway_fee levy, unless explicitly configured to stack.
		if !cfg.StackWithCancel {
			var cancelLevy models.ChefPenalty
			switch err := tx.Where("order_id = ? AND kind = ?", orderID, models.ChefPenaltyCancelLate).
				First(&cancelLevy).Error; {
			case err == nil:
				return nil // already penalized via cancel_late — skip
			case !errors.Is(err, gorm.ErrRecordNotFound):
				return err
			}
		}

		// GRACE: off by default (decision 4) — a gateway fee is a pass-through cost actually
		// incurred, not an accountability penalty that needs an emergency exemption. Only
		// consulted when explicitly enabled.
		if cfg.GraceEnabled && cfg.GraceCount > 0 {
			var used int64
			q := tx.Model(&models.ChefPenalty{}).
				Where("chef_id = ? AND kind = ?", chefID, models.ChefPenaltyGatewayFee)
			if cfg.GraceWindow > 0 {
				q = q.Where("occurred_at >= ?", now.Add(-cfg.GraceWindow))
			}
			if err := q.Count(&used).Error; err != nil {
				return err
			}
			if used < int64(cfg.GraceCount) {
				grace := models.ChefPenalty{
					ChefID: chefID, UserID: userID, Kind: models.ChefPenaltyGatewayFee,
					Status: models.ChefPenaltyWaived, SourceKey: sourceKey,
					OrderID: &orderID, Reference: reference, Currency: EarningsCurrency,
					BasisAmount: Round2(refundedAmount), RatePercent: cfg.Percent, Amount: 0,
					OccurredAt: now, WaivedAt: &now,
					Reason: "Payment gateway fee on a chef-fault refund",
					WaiveReason: fmt.Sprintf("Within the allowance of %d gateway-fee event(s) per %d days",
						cfg.GraceCount, int(cfg.GraceWindow.Hours()/24)),
				}
				if err := tx.Create(&grace).Error; err != nil {
					return err
				}
				return nil
			}
		}

		p := models.ChefPenalty{
			ChefID: chefID, UserID: userID, Kind: models.ChefPenaltyGatewayFee,
			Status: models.ChefPenaltyPending, SourceKey: sourceKey,
			OrderID: &orderID, Reference: reference, Currency: EarningsCurrency,
			BasisAmount: Round2(refundedAmount), RatePercent: cfg.Percent, Amount: amount,
			OccurredAt: now,
			Reason:     "Payment gateway fee on a chef-fault refund",
		}
		if err := tx.Create(&p).Error; err != nil {
			if isDuplicateKeyErr(err) {
				return nil // concurrent writer levied first
			}
			return err
		}
		raised = &p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return raised, nil
}

// PendingChefPenaltyTotal sums a chef's outstanding levies — what the next settlement will
// deduct.
func PendingChefPenaltyTotal(db *gorm.DB, chefID uuid.UUID) (float64, error) {
	var total float64
	err := db.Model(&models.ChefPenalty{}).
		Where("chef_id = ? AND status = ?", chefID, models.ChefPenaltyPending).
		Select("COALESCE(SUM(amount), 0)").Scan(&total).Error
	return Round2(total), err
}

// WaiveChefPenalty cancels a pending levy. Guarded on `pending`, so a levy already netted off a
// settlement can't be waived after the fact (that would need a credit, not a waiver).
func WaiveChefPenalty(db *gorm.DB, penaltyID, adminID uuid.UUID, reason string) error {
	now := time.Now().UTC()
	res := db.Model(&models.ChefPenalty{}).
		Where("id = ? AND status = ?", penaltyID, models.ChefPenaltyPending).
		Updates(map[string]any{
			"status": models.ChefPenaltyWaived, "waived_by": adminID,
			"waived_at": now, "waive_reason": reason, "amount": 0,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPenaltyNotPending
	}
	return nil
}

// ApplyChefPenaltiesToStatement nets a chef's pending levies off a freshly-issued weekly
// settlement: it claims them onto the statement and reduces its NetPayout by the total.
//
// The claim is the guard against double-deduction — each levy is stamped with the statement
// that consumed it in the SAME transaction that reduces the payout, so a re-run of statement
// generation finds nothing pending and deducts nothing twice. Returns the amount deducted.
//
// A levy that would take the payout below zero is still claimed in full and the payout floored
// at zero: the alternative (partially consuming a levy) needs a running balance the settlement
// model does not have. In practice the weekly gross dwarfs a single 6% levy.
func ApplyChefPenaltiesToStatement(db *gorm.DB, stmt *models.WeeklyStatement) (float64, error) {
	var deducted float64
	err := db.Transaction(func(tx *gorm.DB) error {
		var pending []models.ChefPenalty
		if err := tx.Where("chef_id = ? AND status = ?", stmt.ChefID, models.ChefPenaltyPending).
			Find(&pending).Error; err != nil {
			return err
		}
		if len(pending) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, 0, len(pending))
		var total float64
		for i := range pending {
			ids = append(ids, pending[i].ID)
			total += pending[i].Amount
		}
		total = Round2(total)
		now := time.Now().UTC()
		// Guarded on `pending` so a concurrent statement run claims each levy at most once;
		// RowsAffected tells us how many we actually won.
		res := tx.Model(&models.ChefPenalty{}).
			Where("id IN ? AND status = ?", ids, models.ChefPenaltyPending).
			Updates(map[string]any{
				"status": models.ChefPenaltyDeducted, "deducted_statement_id": stmt.ID, "deducted_at": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // a concurrent run took them all
		}
		if res.RowsAffected != int64(len(ids)) {
			// Partially lost the race — re-read what WE actually claimed so the payout is
			// reduced by exactly the levies stamped with this statement, never by a sibling's.
			var claimed []models.ChefPenalty
			if err := tx.Where("deducted_statement_id = ?", stmt.ID).Find(&claimed).Error; err != nil {
				return err
			}
			total = 0
			for i := range claimed {
				total += claimed[i].Amount
			}
			total = Round2(total)
		}
		net := Round2(stmt.NetPayout - total)
		if net < 0 {
			net = 0
		}
		if err := tx.Model(&models.WeeklyStatement{}).Where("id = ?", stmt.ID).
			Updates(map[string]any{"penalty_deductions": total, "net_payout": net}).Error; err != nil {
			return err
		}
		stmt.PenaltyDeductions = total
		stmt.NetPayout = net
		deducted = total
		return nil
	})
	return deducted, err
}

// NotifyCustomerOfChefCancelPenalty tells the customer their chef cancelled, what they get back,
// and that the chef was held accountable. Best-effort: a failed push must never fail a cancel.
func NotifyCustomerOfChefCancelPenalty(customerID uuid.UUID, orderRef string, refund float64, levied bool) {
	body := fmt.Sprintf("Your chef cancelled order %s. ₹%.0f — the full amount including taxes and fees — is on its way back to you.", orderRef, refund)
	if levied {
		body += " We've also charged the kitchen a cancellation fee."
	}
	if err := SendPushNotification(customerID, "Your order was cancelled", body,
		map[string]string{"type": "chef_cancelled", "order_ref": orderRef}); err != nil {
		log.Printf("chef-cancel penalty: customer notification failed for %s: %v", orderRef, err)
	}
}
