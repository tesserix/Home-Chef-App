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

	provider := models.NormalizeProvider(order.PaymentProvider)

	// Chef is read by exactly one thing — Stripe's currency fallback when the order has no
	// currency of its own — so it is loaded only then, rather than joined onto every refund.
	// Best-effort: CurrencyForCountry("") already has a default, which is the same position
	// the legacy path is in when its caller didn't preload.
	if provider == models.PaymentProviderStripe && order.Currency == "" {
		if err := database.DB.Preload("Chef").First(&order, "id = ?", req.OrderID).Error; err != nil {
			log.Printf("orderrefund-gateway: load chef for stripe currency on order %s: %v", req.OrderID, err)
		}
	}

	gatewayShare := req.Amount

	// THE FUNDING-RAIL SPLIT. Every rail that funded the order gets back its exact
	// proportion of whatever is being refunded — wallet and loyalty to store credit,
	// the remainder to the provider.
	//
	// Pro-rata rather than gateway-first: the on-demand refundable base is the WHOLE
	// total and the cancellation policy then applies a percentage, so a partial refund
	// is routine. Refunding the provider first would hand a credit-funded order pure
	// cash and return the customer's own credit only once the cash ran out.
	//
	// Both credit slices land in the WALLET. The loyalty slice returns as rupees, not
	// as restored points — an owner decision. Its exposure is bounded because the
	// monthly redemption cap is NOT released on refund (see MonthlyRedeemedPaise), so
	// a redeem-then-cancel loop can convert at most MonthlyRedeemCap per customer per
	// 30 days, no more than they could have spent outright.
	if provider != models.PaymentProviderWallet && (order.WalletApplied > 0 || order.LoyaltyApplied > 0) {
		split := SplitRefundByFunding(&order, ToPaise(req.Amount))

		// Keyed on the coordinator's per-scope operation id, NOT on the order: two
		// successive partial refunds are distinct operations and must each credit,
		// while a re-drive of the SAME operation must not.
		credit := func(paise int, source models.WalletTxnSource, label string) float64 {
			if paise <= 0 {
				return 0
			}
			amount := FromPaise(paise)
			if _, err := CreditWallet(database.DB, order.CustomerID, amount,
				source, &order.ID,
				fmt.Sprintf("%s refund for order %s: %s", label, order.OrderNumber, req.Reason),
				label+":"+req.IdempotencyKey, nil); err != nil {
				// Best-effort, matching the legacy path: the provider slice is the more
				// urgent half and the reconcile cron's refund_mismatch check backstops the
				// store credit. Failing here would strand the captured money too.
				log.Printf("orderrefund-gateway: %s slice re-credit failed order=%s: %v", label, order.OrderNumber, err)
				CaptureBackgroundError(err)
				return 0
			}
			return amount
		}
		walletBack := credit(split.WalletPaise, models.WalletSourceRefund, "refund-wallet")
		loyaltyBack := credit(split.LoyaltyPaise, models.WalletSourceLoyalty, "refund-loyalty")

		// Record what each rail has now been returned so a later partial refund
		// computes its share against the REMAINING funded amount rather than the
		// original — otherwise two 60% refunds would each take 60% of the original
		// slice and together return 120% of it.
		if walletBack > 0 || loyaltyBack > 0 {
			if err := database.DB.Model(&models.Order{}).Where("id = ?", order.ID).
				Updates(map[string]any{
					"wallet_refunded":  order.WalletRefunded + walletBack,
					"loyalty_refunded": order.LoyaltyRefunded + loyaltyBack,
				}).Error; err != nil {
				log.Printf("orderrefund-gateway: stamp refunded rails failed order=%s: %v", order.OrderNumber, err)
				CaptureBackgroundError(err)
			}
		}

		// The provider can never refund more than it captured. Pro-rata already keeps
		// the card slice within the captured amount, but the credit caps above can push
		// their remainder onto the card; clamp and return any excess as credit so the
		// customer is still made whole.
		gatewayShare = FromPaise(split.CardPaise)
		if capture := order.Total - order.WalletApplied - order.LoyaltyApplied; gatewayShare > capture {
			if over := ToPaise(gatewayShare) - ToPaise(capture); over > 0 {
				credit(over, models.WalletSourceRefund, "refund-overflow")
			}
			gatewayShare = capture
		}
	}

	if gatewayShare <= 0 {
		// Fully credit-funded refund — there is no provider leg to run.
		return "", nil
	}
	return runCancellationGatewayRefund(&order, provider, gatewayShare, req.Actor, req.Reason, req.IdempotencyKey)
}
