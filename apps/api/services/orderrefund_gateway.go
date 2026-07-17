package services

// orderrefund_gateway.go — the provider-routing half of the refund coordinator (#690).
//
// The coordinator (services/orderrefund) owns the SAGA: reserve, move, finalize. It knows
// nothing about Razorpay vs Stripe vs wallet, and it must not — that is what keeps it
// importable from anywhere without dragging the provider clients along, and what keeps a
// provider quirk from turning into a saga bug.
//
// This adapter owns DELIVERY: given "₹X is owed back on order Y", get ₹X to the customer.
// That is genuinely more than one call, which is why it is not a one-line wrapper:
//
//   - PROVIDER ROUTING — wallet / stripe / razorpay, each with its own id, its own minor-unit
//     rule, its own client. Forgetting this is #691, live today in order_issue.go.
//   - THE WALLET-CAPTURE SPLIT (#141) — a wallet-funded order only CAPTURED
//     (Total − WalletApplied) at the provider, so the provider physically cannot refund the
//     whole amount owed. The difference goes back as store credit. The coordinator reserves
//     the FULL amount either way; splitting it is a delivery detail and must not leak upward,
//     or the reservation would under-count and the next refund would over-refund.
//
// It reuses runCancellationGatewayRefund for the provider switch rather than restating it —
// that function is the switch InitiateRefund already mirrors, and a second copy is exactly
// the duplication #687 exists to remove.

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services/orderrefund"
)

// OrderRefundGateway routes coordinator refunds to the right provider.
type OrderRefundGateway struct{}

// compile-time proof this satisfies the coordinator's port.
var _ orderrefund.Gateway = (*OrderRefundGateway)(nil)

// NewOrderRefundCoordinator builds the coordinator the refund call sites use.
//
// Enabled is hardcoded true: there is no REFUND_GATEWAY_ENABLED flag today, and adding a
// switch that can silently stop refunds is not something to do as a side effect of a
// refactor. The parameter exists for tests and for a future kill switch that should be
// introduced deliberately, with a default of ON.
func NewOrderRefundCoordinator() *orderrefund.Coordinator {
	return orderrefund.NewCoordinator(database.DB, OrderRefundGateway{}, true)
}

// RefundPayment delivers req.Amount back to the customer, splitting across store credit and
// the provider when the order was part-funded by the wallet. Returns the provider's refund
// reference.
func (OrderRefundGateway) RefundPayment(_ context.Context, req orderrefund.GatewayRequest) (string, error) {
	// Load fresh rather than trusting a caller snapshot: Total/WalletApplied drive the
	// split. Reading after the reservation is safe — the claim is held, so no concurrent
	// refund can move them.
	var order models.Order
	if err := database.DB.First(&order, "id = ?", req.OrderID).Error; err != nil {
		return "", fmt.Errorf("orderrefund-gateway: load order %s: %w", req.OrderID, err)
	}

	provider := strings.ToLower(order.PaymentProvider)
	if provider == "" {
		provider = "razorpay"
	}

	// Chef is read by exactly one thing — Stripe's currency fallback when the order has no
	// currency of its own — so it is loaded only then, rather than joined onto every refund.
	// Best-effort: CurrencyForCountry("") already has a default, which is the same position
	// the legacy path is in when its caller didn't preload.
	if provider == "stripe" && order.Currency == "" {
		if err := database.DB.Preload("Chef").First(&order, "id = ?", req.OrderID).Error; err != nil {
			log.Printf("orderrefund-gateway: load chef for stripe currency on order %s: %v", req.OrderID, err)
		}
	}

	gatewayShare := req.Amount

	// Wallet-at-checkout (#141): only (Total − WalletApplied) was captured at the provider,
	// so it cannot refund more than that. Re-credit the wallet-funded slice as store credit
	// and cap the provider refund to the captured amount. Parity with InitiateRefund and
	// RefundOrderForCancellation.
	//
	// The wallet credit is idempotent on "refund-wallet:<orderID>", so a coordinator retry
	// for the same order re-credits nothing — which is what makes this safe to sit behind a
	// gateway call that the saga may re-drive.
	if order.WalletApplied > 0 && provider != "wallet" {
		capture := order.Total - order.WalletApplied
		if gatewayShare > capture {
			walletPortion := gatewayShare - capture
			if _, err := CreditWallet(database.DB, order.CustomerID, walletPortion,
				models.WalletSourceRefund, &order.ID,
				fmt.Sprintf("Wallet-portion refund for order %s: %s", order.OrderNumber, req.Reason),
				"refund-wallet:"+order.ID.String(), nil); err != nil {
				// Best-effort, matching the legacy path: the provider slice is the larger and
				// more urgent half, and the reconcile cron's refund_mismatch check backstops
				// the store credit. Failing the whole refund here would strand the captured
				// money too.
				log.Printf("orderrefund-gateway: wallet-portion re-credit failed order=%s: %v", order.OrderNumber, err)
				CaptureBackgroundError(err)
			}
			gatewayShare = capture
		}
	}

	return runCancellationGatewayRefund(&order, provider, gatewayShare, req.Actor, req.Reason, req.IdempotencyKey)
}
