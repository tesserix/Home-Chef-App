package services

// chef_payout.go — THE one writer of order_chef_payouts (D-21).
//
//	net = food + delivery + tip − penalty
//
// No commission, no GST, no TDS. The platform's revenue is the customer-paid
// platform fee, and the platform accounts for all GST under CGST s.9(5) — the
// e-commerce operator is liable for the tax on restaurant service supplied
// through it, not the kitchen, whether or not the kitchen is registered.
//
// Deliberately NOT built on ComputeOrderEarnings: that computes the settlement
// view (commission, commission GST, TDS, and food GST credited to the chef) and
// still feeds the weekly statement, the FY statement and the TDS certificate.
// Changing it would restate figures chefs have already been reconciled against —
// the hazard ChefTaxOf was written to avoid. This is a separate, forward-only
// view; the statement path is untouched.
//
// See .planning/CHEF-PAYOUT-VIEW-DESIGN.md.

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// ChefPayoutBreakdown is the computed payout, before it is persisted. Pure, so
// the arithmetic is testable without a database.
type ChefPayoutBreakdown struct {
	FoodAmount  float64
	DeliveryFee float64
	ChefTip     float64
	Penalty     float64
	NetPayout   float64
}

// ComputeChefPayout builds an order's chef payout.
//
// The delivery fee counts ONLY when the chef carried the leg themselves — under
// any other fulfilment the platform or a 3PL carried it and the fee is not the
// kitchen's. A free-zone or pickup order contributes 0, and the caller renders
// no delivery line at all rather than a ₹0 one.
//
// Never negative: a penalty larger than the order cannot make the chef owe money
// on it. The remainder stays outstanding in the penalty ledger.
func ComputeChefPayout(order *models.Order, penalty float64) ChefPayoutBreakdown {
	b := ChefPayoutBreakdown{
		FoodAmount: Round2(order.Subtotal),
		ChefTip:    Round2(order.ChefTip),
		Penalty:    Round2(penalty),
	}
	if order.FulfillmentType == models.FulfillmentChefDelivery {
		// The fee the chef actually settled on: they may have lowered it at accept
		// (#703), and the difference was refunded to the customer.
		fee := order.DeliveryFee
		if order.DeliveryFeeFinal != nil {
			fee = *order.DeliveryFeeFinal
		}
		if fee > 0 {
			b.DeliveryFee = Round2(fee)
		}
	}
	if b.Penalty < 0 {
		b.Penalty = 0
	}
	// NetPayout is the EXACT sum of the lines above, to the paise — never an
	// independently rounded figure. Each component is snapped to paise first and
	// the total is then the sum of those snapped values, so the breakdown a chef
	// reads always adds up to the amount they are paid. (Round2 on the sum removes
	// IEEE-754 float noise like 320.00000000000006; it never moves a real value.)
	// The inverse — rounding the total on its own — is how a receipt comes to
	// out-sum its own lines by a paise.
	net := b.FoodAmount + b.DeliveryFee + b.ChefTip - b.Penalty
	if net < 0 {
		// A penalty larger than the order cannot make the chef owe money on it;
		// the remainder stays outstanding in the penalty ledger.
		net = 0
	}
	b.NetPayout = Round2(net)
	return b
}

// RecordChefPayout persists an order's payout exactly once.
//
// Idempotent on order_id: a retried delivery, a re-driven webhook or two
// concurrent writers settle a single row. On conflict the figures are refreshed
// (a penalty can be raised after delivery) but a row that has already been
// released or reversed is left alone — money that moved is not restated.
func RecordChefPayout(db *gorm.DB, order *models.Order, penalty float64) (*models.OrderChefPayout, error) {
	b := ComputeChefPayout(order, penalty)
	row := models.OrderChefPayout{
		OrderID:     order.ID,
		ChefID:      order.ChefID,
		Currency:    order.Currency,
		FoodAmount:  b.FoodAmount,
		DeliveryFee: b.DeliveryFee,
		ChefTip:     b.ChefTip,
		Penalty:     b.Penalty,
		NetPayout:   b.NetPayout,
		Status:      models.ChefPayoutPending,
		ComputedAt:  time.Now().UTC(),
	}
	if row.Currency == "" {
		row.Currency = "INR"
	}

	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "order_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"food_amount":  row.FoodAmount,
			"delivery_fee": row.DeliveryFee,
			"chef_tip":     row.ChefTip,
			"penalty":      row.Penalty,
			"net_payout":   row.NetPayout,
			"computed_at":  row.ComputedAt,
			"updated_at":   row.ComputedAt,
		}),
		// Only a row still pending may be refreshed. Released/reversed is settled.
		Where: clause.Where{Exprs: []clause.Expression{
			clause.Eq{Column: clause.Column{Table: "order_chef_payouts", Name: "status"}, Value: models.ChefPayoutPending},
		}},
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ChefOrderPenalty is the levy attributed to one order, for display on its
// payout. Reads the penalty ledger rather than copying it: the ledger stays the
// authority for whether and when the money is actually taken (it is netted off a
// weekly settlement, not this order).
//
// Waived levies count 0 — an admin cancelled them, so they were never owed.
// Best-effort: a read failure yields 0 rather than blocking a delivery.
func ChefOrderPenalty(db *gorm.DB, orderID uuid.UUID) float64 {
	var p models.ChefPenalty
	err := db.Where("source_key = ? AND status <> ?",
		ChefCancelPenaltySourceKey(orderID), models.ChefPenaltyWaived).First(&p).Error
	if err != nil {
		return 0
	}
	return p.Amount
}

// GetChefPayout returns an order's payout row, or nil when none exists yet
// (the order has not reached delivery).
func GetChefPayout(db *gorm.DB, orderID uuid.UUID) (*models.OrderChefPayout, error) {
	var row models.OrderChefPayout
	err := db.Where("order_id = ?", orderID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
