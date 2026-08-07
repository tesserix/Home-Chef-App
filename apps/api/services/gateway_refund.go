package services

import (
	"fmt"
	"strings"

	"github.com/homechef/api/models"
)

// gateway_refund.go — THE single provider switch for "send ₹X back to the
// customer's original payment method".
//
// Before this file, six call sites each did their own version of it:
//
//	handlers/payment.go            InitiateRefund
//	handlers/chef_order_cancel.go  full cancel, per-line cancel, goodwill partial
//	services/cancellation_execute.go   arbitration refund
//	services/deferred_cancel_refund.go retry cron
//
// Four of the six reached straight for GetRazorpayFor + RazorpayPaymentID, and
// three of those were fronted by `if order.PaymentProvider != "razorpay" { 422 }`.
// That is a stable arrangement with exactly one INR gateway and a money-losing one
// with two: the guard reads "not refundable" for a perfectly refundable Cashfree
// order, and without the guard the call reaches for a payment id that gateway
// never issued.
//
// So the routing lives here, once. A call site's job is to decide the AMOUNT, the
// NOTES and the IDEMPOTENCY KEY — all things it genuinely knows — and nothing
// about which gateway is involved.
//
// The two invariants from gateway_idempotency.go still bind every caller:
// identical key across retries of the same logical refund, distinct key across
// different refunds. They matter more here, not less, because Cashfree turns the
// key into the refund_id itself.

// GatewayRefundResult is the provider-agnostic outcome of one refund.
type GatewayRefundResult struct {
	// RefundID is the gateway's reference, stored in Order.RefundID.
	RefundID string
	// Status is the PLATFORM vocabulary ("processed" / "pending" / "failed"), not
	// the gateway's own, so a caller persisting it doesn't have to know which
	// gateway produced it.
	Status string
}

// GatewayRefundAvailable reports whether this order's gateway client is
// configured and reachable enough to attempt a refund right now.
//
// Callers that defer-and-retry on an unavailable gateway (the chef cancel path,
// the retry cron) use this instead of a nil-check on a specific client, so a
// Cashfree order defers for the same reason and by the same route a Razorpay one
// does rather than falling through a razorpay-shaped nil check.
func GatewayRefundAvailable(order *models.Order) bool {
	if order == nil || !order.GatewayRefundable() {
		return false
	}
	switch models.NormalizeProvider(order.PaymentProvider) {
	case models.PaymentProviderCashfree:
		return GetCashfreeFor(order.Mode) != nil
	case models.PaymentProviderStripe:
		return GetStripe() != nil
	case models.PaymentProviderWallet:
		return false
	default:
		return GetRazorpayFor(order.Mode) != nil
	}
}

// IssueOrderGatewayRefund refunds amountPaise on the order's original payment
// method.
//
// notes are attached to the refund at the gateway for reconciliation — they are
// informational and never load-bearing. idempotencyKey is the LOGICAL operation
// id (see the gateway_idempotency.go builders) and IS load-bearing: on Razorpay
// it becomes the X-Refund-Idempotency header, on Cashfree the refund_id itself.
// Passing a key that collides with a different refund makes the gateway silently
// dedup the second one — the customer is simply never paid, with no error to
// notice.
//
// Wallet is NOT handled here: a wallet refund is a ledger credit, not a gateway
// call, and it needs a *gorm.DB the caller owns. runCancellationGatewayRefund is
// the layer that routes wallet-vs-gateway.
func IssueOrderGatewayRefund(order *models.Order, amountPaise int, notes map[string]string, idempotencyKey string) (*GatewayRefundResult, error) {
	if order == nil {
		return nil, fmt.Errorf("gateway-refund: nil order")
	}
	if amountPaise <= 0 {
		return nil, fmt.Errorf("gateway-refund: amount must be positive, got %d paise", amountPaise)
	}
	if idempotencyKey == "" {
		// Refusing is deliberate: without a key a retry-after-timeout becomes a
		// SECOND real refund on Razorpay, and on Cashfree there is no refund_id to
		// send at all. An accidental omission must fail loudly here rather than
		// quietly at the gateway.
		return nil, fmt.Errorf("gateway-refund: idempotency key is required")
	}

	provider := models.NormalizeProvider(order.PaymentProvider)
	reference := order.GatewayRefundReference()
	if reference == "" {
		return nil, fmt.Errorf("gateway-refund: order %s has no %s payment to refund", order.ID, provider)
	}

	switch provider {
	case models.PaymentProviderCashfree:
		cf := GetCashfreeFor(order.Mode)
		if cf == nil {
			return nil, fmt.Errorf("cashfree gateway not configured")
		}
		// reference is the ORDER id here — Cashfree refunds are order-scoped.
		r, err := cf.CreateRefund(reference, &CashfreeRefundRequest{
			AmountPaise:    cashfreeAmount(amountPaise),
			Note:           cashfreeRefundNote(order, notes),
			IdempotencyKey: idempotencyKey,
			// Stated explicitly so the vendor's share is reversed the same way
			// whether or not Cashfree's proportional debiting is on. See
			// BuildRefundSplits.
			Splits: BuildRefundSplits(order, ToPaise(order.RefundAmount), amountPaise),
		})
		if err != nil {
			return nil, err
		}
		return &GatewayRefundResult{
			RefundID: r.RefundID,
			Status:   PlatformRefundStatus(r.RefundStatus),
		}, nil

	case models.PaymentProviderStripe:
		st := GetStripe()
		if st == nil {
			return nil, fmt.Errorf("stripe gateway not configured")
		}
		currency := strings.ToLower(order.Currency)
		if currency == "" {
			// Chef may not be preloaded; CurrencyForCountry("") has a sane default,
			// which is the same position the legacy call sites were in.
			currency = CurrencyForCountry(order.Chef.PayoutCountry)
		}
		// amountPaise is INR minor units. For a Stripe order the order's own
		// currency governs, and its minor unit may not be 1/100 — so convert back
		// through rupees and re-scale rather than passing paise straight through.
		r, err := st.CreateRefund(&StripeRefundRequest{
			PaymentIntent:        reference,
			Amount:               ToMinor(FromPaise(amountPaise), currency),
			Reason:               "requested_by_customer",
			ReverseTransfer:      true,
			RefundApplicationFee: true,
			Metadata:             notes,
		})
		if err != nil {
			return nil, err
		}
		return &GatewayRefundResult{RefundID: r.ID, Status: r.Status}, nil

	default: // razorpay
		rz := GetRazorpayFor(order.Mode)
		if rz == nil {
			return nil, fmt.Errorf("razorpay gateway not configured")
		}
		r, err := rz.CreateRefund(reference, &RefundRequest{
			Amount:         amountPaise,
			Speed:          "normal",
			Notes:          notes,
			Receipt:        fmt.Sprintf("refund-%s", order.OrderNumber),
			IdempotencyKey: idempotencyKey,
		})
		if err != nil {
			return nil, err
		}
		return &GatewayRefundResult{RefundID: r.ID, Status: r.Status}, nil
	}
}

// cashfreeRefundNote flattens the notes map into Cashfree's single free-text
// refund_note, which is capped at 100 characters.
//
// Cashfree has no structured notes field, so the reconciliation-relevant bits are
// composed by hand rather than by ranging over the map — map iteration order is
// random in Go, so a range would produce a different note on every retry, and a
// truncated-differently note is a needless reconciliation puzzle. Order number
// and reason are what a human reading a Cashfree dashboard row actually needs;
// the order UUID is already on the order_tags set at creation.
func cashfreeRefundNote(order *models.Order, notes map[string]string) string {
	note := order.OrderNumber
	if reason := notes["reason"]; reason != "" {
		note += ": " + reason
	}
	if len(note) > 100 {
		note = note[:100]
	}
	return note
}
