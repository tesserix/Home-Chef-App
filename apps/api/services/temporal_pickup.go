package services

// temporal_pickup.go — start + signal forwarding for the ready-to-collect flow
// (temporal/workflows/pickup.go). Mirrors temporal_confirm.go exactly: gated on
// Temporal being up AND a deploy-time env switch AND a runtime admin toggle, so
// the flow can be killed from the console without a redeploy.
//
// Every function here is best-effort by design. The authoritative work — the
// order actually being marked ready, collected or cancelled — has already been
// committed by the caller. If Temporal is down, the customer still gets the
// ordinary status push; they just don't get the follow-up reminders.

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

func pickupFlowID(orderID uuid.UUID) string { return "homechef:pickup:" + orderID.String() }

// pickupFlowActive reports whether the ready-to-collect flow should be driven.
//
//   - Temporal must be up (temporalRT set).
//   - PICKUP_READY_FLOW_ENABLED (env, default true): a deploy-time master switch
//     — explicit false hard-disables regardless of the admin toggle.
//   - platform_policy.pickupReadyFlowEnabled (default true): the runtime kill
//     switch admins flip from the console (tesserix-home) without a redeploy.
func pickupFlowActive() bool {
	if temporalRT == nil {
		return false
	}
	if config.AppConfig != nil && !config.AppConfig.PickupReadyFlowEnabled {
		return false
	}
	return GetPlatformPolicy().PickupReadyFlowEnabled
}

// StartPickupReadyFlow durably starts the ready notice + reminder flow for a
// pickup order. Idempotent on the order-keyed workflow ID, so a chef who taps
// "Mark ready" twice never starts a second flow (and never double-notifies).
//
// No-op when the flow is disabled or Temporal is down — the customer still gets
// the ordinary `ready` status notification from the orders.updated path.
func StartPickupReadyFlow(orderID uuid.UUID) {
	if !pickupFlowActive() {
		return
	}
	in := workflows.PickupReadyInput{
		OrderID:                 orderID,
		ReminderIntervalSeconds: PickupReminderIntervalMinutes(database.DB) * 60,
		MaxReminders:            PickupReminderMaxCount(database.DB),
	}
	if _, err := temporalRT.Start(context.Background(), apitemporal.TaskQueueOrders, pickupFlowID(orderID), workflows.PickupReadyWorkflow, in); err != nil {
		// An "already started" error is the expected idempotent case; anything
		// else is logged and the flow is simply skipped for this order.
		log.Printf("pickup-ready flow: start failed for %s: %v", orderID, err)
	}
}

// signalPickupFlow forwards a signal to a running pickup flow. Best-effort: if
// the flow isn't running (disabled, already finished, Temporal unavailable) the
// signal is dropped with a log — the caller's synchronous handling already did
// the authoritative work.
func signalPickupFlow(orderID uuid.UUID, signal string) {
	if !pickupFlowActive() {
		return
	}
	if err := temporalRT.Signal(context.Background(), pickupFlowID(orderID), signal, nil); err != nil {
		log.Printf("pickup-ready flow: signal %s for %s dropped: %v", signal, orderID, err)
	}
}

// SignalOrderCollectedFlow tells a running flow the customer collected their
// order, ending the reminder loop early.
func SignalOrderCollectedFlow(orderID uuid.UUID) {
	signalPickupFlow(orderID, workflows.SignalOrderCollected)
}

// SignalPickupCancelledFlow tells a running flow the order was cancelled or
// refunded while waiting to be collected, ending the reminder loop early.
func SignalPickupCancelledFlow(orderID uuid.UUID) {
	signalPickupFlow(orderID, workflows.SignalPickupCancelled)
}
