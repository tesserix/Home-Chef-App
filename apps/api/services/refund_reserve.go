package services

// refund_reserve.go — thin wrappers over services/refundreserve (#690).
//
// The reservation protocol MOVED to refundreserve because it is the cross-path mutex,
// and services/orderrefund's coordinator must take the same claim as these legacy paths
// while both are live during #690's migration — but orderrefund cannot import services
// (services imports orderrefund as call sites migrate), so the protocol had to live below
// both. See refundreserve/reserve.go for the full #609 rationale.
//
// These wrappers keep the existing call sites (cancellation_order_refund, payment.go,
// chef_order_cancel, temporal_order, order_issue) and their tests unchanged — the move is
// a pure refactor with no behaviour change.

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/services/refundreserve"
)

// ReserveFullRefund — see refundreserve.ReserveFullRefund.
func ReserveFullRefund(db *gorm.DB, orderID uuid.UUID) (amount float64, won bool, err error) {
	return refundreserve.ReserveFullRefund(db, orderID)
}

// ReserveRefund — see refundreserve.ReserveRefund. Kept on the original multi-return
// signature so the existing callers are untouched; refundreserve.Reservation is the
// richer shape new code (the coordinator) uses.
func ReserveRefund(db *gorm.DB, orderID uuid.UUID, requested float64) (amount, priorRefunded float64, fullRefund, won bool, err error) {
	r, err := refundreserve.ReserveRefund(db, orderID, requested)
	if err != nil {
		return 0, 0, false, false, err
	}
	return r.Amount, r.PriorRefunded, r.FullRefund, r.Won, nil
}

// ReleaseRefundReservation — see refundreserve.ReleaseRefundReservation.
func ReleaseRefundReservation(db *gorm.DB, orderID uuid.UUID, amount float64) {
	refundreserve.ReleaseRefundReservation(db, orderID, amount)
}

// ReleaseFullRefundReservation — see refundreserve.ReleaseFullRefundReservation.
func ReleaseFullRefundReservation(db *gorm.DB, orderID uuid.UUID, amount float64) {
	refundreserve.ReleaseFullRefundReservation(db, orderID, amount)
}
