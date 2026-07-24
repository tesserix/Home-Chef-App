package services

// temporal_confirm.go — start + signal forwarding for the confirm-receipt
// reminder flow (see docs plan 2026-07-24-auto-confirm-delivery.md, Task 4).
// Mirrors StartOrderSaga/signalOrderSaga in temporal_order.go: gated behind
// config.AppConfig.ConfirmReceiptFlowEnabled (default OFF) AND temporalRT
// being set, so prod behaviour is unchanged until the flow is validated. The
// existing 24h payout-auto-confirm cron remains the fallback either way.

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

func confirmFlowID(orderID uuid.UUID) string { return "homechef:confirm:" + orderID.String() }

// confirmFlowActive reports whether the confirm-receipt flow should be driven
// (Temporal up + flag on).
func confirmFlowActive() bool {
	return temporalRT != nil && config.AppConfig != nil && config.AppConfig.ConfirmReceiptFlowEnabled
}

// StartConfirmReceiptFlow durably starts the reminder+auto-confirm workflow
// for a delivered order. Idempotent on the order-keyed workflow ID, so a
// re-trigger of the delivered transition never starts a second flow. No-op
// when the flow is disabled or Temporal is down (the 24h cron remains the
// safety net).
func StartConfirmReceiptFlow(orderID uuid.UUID) {
	if !confirmFlowActive() {
		return
	}
	in := workflows.ConfirmReceiptInput{
		OrderID:                 orderID,
		ReminderIntervalSeconds: ConfirmReminderIntervalMinutes(database.DB) * 60,
		MaxReminders:            ConfirmReminderMaxCount(database.DB),
	}
	if _, err := temporalRT.Start(context.Background(), apitemporal.TaskQueueOrders, confirmFlowID(orderID), workflows.ConfirmReceiptWorkflow, in); err != nil {
		// An "already started" error is the expected idempotent case; anything
		// else is logged (the 24h cron remains the safety net).
		log.Printf("confirm-receipt flow: start failed for %s: %v", orderID, err)
	}
}

// signalConfirmFlow forwards a signal to a running confirm-receipt flow.
// Best-effort: if the flow isn't running (disabled, already finished, or
// Temporal unavailable) the signal is dropped with a log — the caller's own
// synchronous handling already did the authoritative work.
func signalConfirmFlow(orderID uuid.UUID, signal string) {
	if !confirmFlowActive() {
		return
	}
	if err := temporalRT.Signal(context.Background(), confirmFlowID(orderID), signal, nil); err != nil {
		log.Printf("confirm-receipt flow: signal %s for %s dropped: %v", signal, orderID, err)
	}
}

// SignalOrderConfirmedFlow tells a running flow the customer confirmed
// receipt, ending the reminder loop early.
func SignalOrderConfirmedFlow(orderID uuid.UUID) {
	signalConfirmFlow(orderID, workflows.SignalOrderConfirmed)
}

// SignalOrderDisputedFlow tells a running flow a dispute opened on the order,
// ending the reminder loop early.
func SignalOrderDisputedFlow(orderID uuid.UUID) {
	signalConfirmFlow(orderID, workflows.SignalOrderDisputed)
}
