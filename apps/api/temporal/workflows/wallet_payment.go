package workflows

// wallet_payment.go — the durable mixed wallet + external (UPI/card) payment flow
// (docs/wallet-ledger-plan.md, Phase 5). It reserves the wallet portion as a hold, waits for the
// external gateway leg to resolve, then CAPTURES the hold on success or RELEASES it on
// failure/timeout — so a dropped or failed external payment never strands the customer's wallet
// money, and a worker crash resumes with no duplicate movement. Wallet-only orders (no external
// leg) capture immediately. Built on the Phase 3 ledger holds
// (services.{Place,Capture,Release}WalletHold), gated OFF until the ledger is live.

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"

	apitemporal "github.com/homechef/api/temporal"
)

// Signals the flow listens for — forwarded by the payment verify handler / webhook once the
// external gateway leg resolves.
const (
	SignalWalletPaymentCaptured = "wallet_payment.captured"
	SignalWalletPaymentFailed   = "wallet_payment.failed"
)

// WalletPaymentInput starts the flow for one order's payment. The hold is keyed by
// (RefType="order", RefID=OrderID) — the same per-order idempotency the rest of checkout uses.
type WalletPaymentInput struct {
	OrderID                uuid.UUID
	UserID                 uuid.UUID
	WalletAmountMinor      int64 // wallet portion to reserve (0 = no wallet used)
	ExternalAmountMinor    int64 // gateway portion (0 = wallet-only → capture immediately)
	ExternalTimeoutSeconds int   // how long to wait for the gateway leg before releasing
}

// WalletHoldActivityInput / WalletHoldRefInput are the activity arguments.
type WalletHoldActivityInput struct {
	UserID      uuid.UUID
	AmountMinor int64
	RefType     string
	RefID       string
}
type WalletHoldRefInput struct {
	RefType string
	RefID   string
}

// Transport seams — the worker wires these to services.* (avoids a workflows→services import
// cycle). Nil in unit tests unless the test overrides them.
var (
	PlaceWalletHoldFunc   func(ctx context.Context, in WalletHoldActivityInput) error
	CaptureWalletHoldFunc func(ctx context.Context, in WalletHoldRefInput) error
	ReleaseWalletHoldFunc func(ctx context.Context, in WalletHoldRefInput) error
)

// PlaceWalletHoldActivity reserves the wallet portion of the payment.
func PlaceWalletHoldActivity(ctx context.Context, in WalletHoldActivityInput) error {
	if PlaceWalletHoldFunc == nil {
		return nil
	}
	return PlaceWalletHoldFunc(ctx, in)
}

// CaptureWalletHoldActivity spends the reserved hold once the external leg succeeded.
func CaptureWalletHoldActivity(ctx context.Context, in WalletHoldRefInput) error {
	if CaptureWalletHoldFunc == nil {
		return nil
	}
	return CaptureWalletHoldFunc(ctx, in)
}

// ReleaseWalletHoldActivity returns the reserved hold to the customer (compensation) when the
// external leg failed or timed out.
func ReleaseWalletHoldActivity(ctx context.Context, in WalletHoldRefInput) error {
	if ReleaseWalletHoldFunc == nil {
		return nil
	}
	return ReleaseWalletHoldFunc(ctx, in)
}

// WalletPaymentWorkflow reserves the wallet portion, waits for the external leg, then captures
// (success) or releases (failure/timeout) the hold. Idempotent activities + Temporal durability
// mean a worker crash resumes with no duplicate money movement.
func WalletPaymentWorkflow(ctx workflow.Context, in WalletPaymentInput) error {
	ref := WalletHoldRefInput{RefType: "order", RefID: in.OrderID.String()}
	actx := apitemporal.Activities(ctx, 30*time.Second)

	// 1. Reserve the wallet portion. If this fails, the payment can't use wallet — surface it.
	if in.WalletAmountMinor > 0 {
		if err := workflow.ExecuteActivity(actx, PlaceWalletHoldActivity, WalletHoldActivityInput{
			UserID: in.UserID, AmountMinor: in.WalletAmountMinor, RefType: ref.RefType, RefID: ref.RefID,
		}).Get(ctx, nil); err != nil {
			return err
		}
	}

	// 2. Resolve the external leg. Wallet-only (no external) → captured immediately.
	captured := true
	if in.ExternalAmountMinor > 0 {
		capturedCh := workflow.GetSignalChannel(ctx, SignalWalletPaymentCaptured)
		failedCh := workflow.GetSignalChannel(ctx, SignalWalletPaymentFailed)
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(capturedCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); captured = true })
		sel.AddReceive(failedCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); captured = false })
		sel.AddFuture(workflow.NewTimer(ctx, time.Duration(in.ExternalTimeoutSeconds)*time.Second), func(workflow.Future) { captured = false })
		sel.Select(ctx)
	}

	// 3. Nothing reserved → nothing to settle.
	if in.WalletAmountMinor <= 0 {
		return nil
	}
	// Capture on success; release (compensation) on failure/timeout so the customer's wallet
	// money is never stranded by a failed external leg.
	if captured {
		return workflow.ExecuteActivity(actx, CaptureWalletHoldActivity, ref).Get(ctx, nil)
	}
	return workflow.ExecuteActivity(actx, ReleaseWalletHoldActivity, ref).Get(ctx, nil)
}
