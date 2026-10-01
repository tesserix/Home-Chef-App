package services

// INR keeps the existing food + delivery + tip - penalty display policy.
// AU/NZ food proceeds match the Stripe allocation, including food GST and
// commission. Persisted rows remain immutable once released or reversed.

import (
	"strings"
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
// The delivery fee counts ONLY when the leg is the chef's (ChefEarnsDeliveryFee)
// — under any other fulfilment the platform or a 3PL carried it and the fee is
// not the kitchen's. A free-zone or pickup order contributes 0, and the caller
// renders no delivery line at all rather than a ₹0 one.
//
// Never negative: a penalty larger than the order cannot make the chef owe money
// on it. The remainder stays outstanding in the penalty ledger.
func ComputeChefPayout(order *models.Order, penalty float64) ChefPayoutBreakdown {
	b := ChefPayoutBreakdown{
		FoodAmount: Round2(order.Subtotal),
		ChefTip:    Round2(order.ChefTip),
		Penalty:    Round2(penalty),
	}
	if order.ChefEarnsDeliveryFee() {
		// EffectiveDeliveryFee, not the raw charge: an order from before delivery
		// pricing was fixed at checkout may carry a reduced final fee.
		if fee := order.EffectiveDeliveryFee(); fee > 0 {
			b.DeliveryFee = Round2(fee)
		}
	}
	if currency := strings.ToUpper(strings.TrimSpace(order.Currency)); currency == "AUD" || currency == "NZD" {
		if order.ChefEarnsDeliveryFee() {
			b.DeliveryFee = Round2(b.DeliveryFee + order.DriverTip)
		}
		// International food proceeds use the same tax/commission split as Stripe.
		b.FoodAmount = Round2(ChefNetPayoutFor(order) - b.DeliveryFee - b.ChefTip)
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

// estimateFrom projects a payout for an order with no row yet — the same
// ComputeChefPayout, marked `estimated` so a caller can tell a projection from a
// settled figure.
func estimateFrom(order *models.Order, penalty float64) *models.ChefPayoutResponse {
	b := ComputeChefPayout(order, penalty)
	currency := order.Currency
	if currency == "" {
		currency = "INR"
	}
	if order.Status == models.OrderStatusRejected || order.Status == models.OrderStatusRefunded {
		return &models.ChefPayoutResponse{
			Currency: currency, Status: models.ChefPayoutReversed, ComputedAt: time.Now().UTC(),
		}
	}
	return &models.ChefPayoutResponse{
		FoodAmount: b.FoodAmount, DeliveryFee: b.DeliveryFee, ChefTip: b.ChefTip,
		Penalty: b.Penalty, NetPayout: b.NetPayout,
		Currency: currency, Status: models.ChefPayoutEstimated, ComputedAt: time.Now().UTC(),
	}
}

// ChefPayoutFor is what every chef- and admin-facing surface renders: the
// persisted row when the order has been delivered, otherwise the same formula
// computed from the live order and marked `estimated`.
//
// A chef decides whether to take an order when it ARRIVES, so gating the figure
// on the delivery-time write left every pending order showing the customer's
// total — the number this whole feature exists to stop showing a kitchen. One
// function, one formula: the row written at delivery just freezes what the chef
// was already looking at.
//
// Never nil, so a surface always has a figure to render.
func ChefPayoutFor(db *gorm.DB, order *models.Order) *models.ChefPayoutResponse {
	if order.Status == models.OrderStatusRefunded {
		return estimateFrom(order, 0)
	}
	if row, err := GetChefPayout(db, order.ID); err == nil && row != nil {
		return row.ToResponse()
	}
	return estimateFrom(order, ChefOrderPenalty(db, order.ID))
}

// ChefPayoutsFor is ChefPayoutFor for a page of orders, in two queries rather
// than two per order — the chef order list and the dashboard queue both render
// every row's payout.
func ChefPayoutsFor(db *gorm.DB, orders []models.Order) map[uuid.UUID]*models.ChefPayoutResponse {
	out := make(map[uuid.UUID]*models.ChefPayoutResponse, len(orders))
	if len(orders) == 0 {
		return out
	}
	ids := make([]uuid.UUID, len(orders))
	for i := range orders {
		ids[i] = orders[i].ID
	}

	var rows []models.OrderChefPayout
	db.Where("order_id IN ?", ids).Find(&rows)
	settled := make(map[uuid.UUID]*models.OrderChefPayout, len(rows))
	for i := range rows {
		settled[rows[i].OrderID] = &rows[i]
	}

	penalties := ChefOrderPenalties(db, ids)
	for i := range orders {
		id := orders[i].ID
		if orders[i].Status == models.OrderStatusRefunded {
			out[id] = estimateFrom(&orders[i], 0)
			continue
		}
		if row, ok := settled[id]; ok {
			out[id] = row.ToResponse()
			continue
		}
		out[id] = estimateFrom(&orders[i], penalties[id])
	}
	return out
}

// ChefOrderPenalties is the batched ChefOrderPenalty — the levy attributed to
// each of the given orders, absent meaning none. Waived levies are excluded (an
// admin cancelled them, so they were never owed).
func ChefOrderPenalties(db *gorm.DB, orderIDs []uuid.UUID) map[uuid.UUID]float64 {
	out := make(map[uuid.UUID]float64, len(orderIDs))
	if len(orderIDs) == 0 {
		return out
	}
	keys := make([]string, len(orderIDs))
	byKey := make(map[string]uuid.UUID, len(orderIDs))
	for i, id := range orderIDs {
		keys[i] = ChefCancelPenaltySourceKey(id)
		byKey[keys[i]] = id
	}
	var levies []models.ChefPenalty
	if err := db.Where("source_key IN ? AND status <> ?", keys, models.ChefPenaltyWaived).
		Find(&levies).Error; err != nil {
		return out
	}
	for _, p := range levies {
		if id, ok := byKey[p.SourceKey]; ok {
			out[id] = p.Amount
		}
	}
	return out
}
