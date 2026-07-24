package services

// temporal_wallet.go — start + signal forwarding for the durable wallet-payment flow
// (docs/wallet-ledger-plan.md, Phase 5). Mirrors StartConfirmReceiptFlow. Gated behind
// config.AppConfig.WalletPaymentFlowEnabled (default OFF) AND temporalRT being set AND the ledger
// being live (LedgerShadowActive) — the flow reserves/settles LEDGER holds, so it does nothing
// useful until the ledger is authoritative. Inert (a no-op) until all three line up.

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

func walletPayFlowID(orderID uuid.UUID) string { return "homechef:walletpay:" + orderID.String() }

// walletPaymentFlowActive reports whether the wallet-payment flow should be driven: Temporal up,
// the env flag on (default off), and the ledger live (holds are meaningless otherwise).
func walletPaymentFlowActive() bool {
	if temporalRT == nil {
		return false
	}
	if config.AppConfig == nil || !config.AppConfig.WalletPaymentFlowEnabled {
		return false
	}
	return LedgerShadowActive()
}

// StartWalletPayment durably starts the mixed wallet + external payment flow for an order.
// Idempotent on the order-keyed workflow ID, so a re-trigger never starts a second flow. No-op
// when disabled / Temporal down / ledger off.
func StartWalletPayment(orderID, userID uuid.UUID, walletMinor, externalMinor int64, externalTimeoutSeconds int) {
	if !walletPaymentFlowActive() {
		return
	}
	in := workflows.WalletPaymentInput{
		OrderID: orderID, UserID: userID,
		WalletAmountMinor: walletMinor, ExternalAmountMinor: externalMinor,
		ExternalTimeoutSeconds: externalTimeoutSeconds,
	}
	if _, err := temporalRT.Start(context.Background(), apitemporal.TaskQueuePayments, walletPayFlowID(orderID), workflows.WalletPaymentWorkflow, in); err != nil {
		// "Already started" is the expected idempotent case; anything else is logged.
		log.Printf("wallet-payment flow: start failed for %s: %v", orderID, err)
	}
}

func signalWalletPayment(orderID uuid.UUID, signal string) {
	if !walletPaymentFlowActive() {
		return
	}
	if err := temporalRT.Signal(context.Background(), walletPayFlowID(orderID), signal, nil); err != nil {
		log.Printf("wallet-payment flow: signal %s for %s dropped: %v", signal, orderID, err)
	}
}

// SignalWalletPaymentCaptured tells a running flow the external gateway leg succeeded → capture.
func SignalWalletPaymentCaptured(orderID uuid.UUID) {
	signalWalletPayment(orderID, workflows.SignalWalletPaymentCaptured)
}

// SignalWalletPaymentFailed tells a running flow the external gateway leg failed → release.
func SignalWalletPaymentFailed(orderID uuid.UUID) {
	signalWalletPayment(orderID, workflows.SignalWalletPaymentFailed)
}
