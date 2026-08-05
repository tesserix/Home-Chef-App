package services

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

// cashfree_charge.go — Cashfree for the charges that are not à la carte orders:
// catering deposits, group-order shares, promo purchases (and, via its own
// escrow-aware path, the meal-plan advance).
//
// Each of these used to mint Razorpay directly, bypassing SelectCheckoutGateway
// entirely, so Cashfree being the platform default only ever applied to orders.
// The two helpers here are the whole seam: mint, then verify from the gateway.
//
// The Cashfree order id is ALWAYS the paying row's own UUID. That is what makes
// this work without new id columns — the capture is re-findable from the row
// alone, so nothing has to be stored beyond which gateway took it.

// ChargeGatewayFor resolves the gateway for a non-order charge. Same seam as
// checkout, so a kitchen cannot end up taking its orders on one rail and its
// deposits on another.
func ChargeGatewayFor(configuredProvider, mode string) string {
	return SelectCheckoutGateway(configuredProvider, mode)
}

// CashfreeCustomerFor builds the payer block Cashfree requires. The id must be
// alphanumeric, so the UUID goes in without its hyphens — matching the order path.
func CashfreeCustomerFor(u *models.User) CashfreeCustomerDetails {
	if u == nil {
		return CashfreeCustomerDetails{}
	}
	return CashfreeCustomerDetails{
		CustomerID:    strings.ReplaceAll(u.ID.String(), "-", ""),
		CustomerPhone: u.Phone,
		CustomerName:  strings.TrimSpace(u.FirstName + " " + u.LastName),
		CustomerEmail: u.Email,
	}
}

// CreateCashfreeCharge mints the Cashfree order for one charge row.
//
// The wire request is deliberately IDENTICAL in shape to the à la carte order
// path (handlers/payment_cashfree.go): same fields, same omissions. That path is
// the one proven against the live merchant account every day, and a charge that
// sent a slightly different body was the difference between orders minting
// happily and FSSAI filing failing with Cashfree's generic 500.
//
// Idempotency comes from the ORDER ID, which is the charge row's own UUID:
// creating the same order twice returns 409 and CreateOrder reads the existing
// order back. The separate x-idempotency-key header the order path never sends
// bought nothing on top of that, so it is not sent here either.
//
// note and idempotencyScope are accepted and ignored — kept so the four call
// sites read the same and nobody has to relearn the signature.
func CreateCashfreeCharge(
	mode string,
	rowID uuid.UUID,
	amount float64,
	currency string,
	cust CashfreeCustomerDetails,
	tags map[string]string,
	_ string,
	_ string,
) (*CashfreeOrderResponse, error) {
	cf := GetCashfreeFor(mode)
	if cf == nil {
		return nil, fmt.Errorf("cashfree not configured")
	}
	if currency == "" {
		currency = "INR"
	}
	return cf.CreateOrder(&CashfreeOrderRequest{
		OrderID:     rowID.String(),
		AmountPaise: cashfreeAmount(ToPaise(amount)),
		Currency:    currency,
		Customer:    cust,
		Tags:        tags,
	})
}

// VerifyCashfreeCharge returns the captured payment for a charge, or an error when
// the gateway has nothing captured for at least minAmount.
//
// There is no client signature to check — Cashfree hands the client nothing it could
// sign — so this fetch IS the verification. The binding is the order id: it is the
// charge row's own UUID, so a capture found under it cannot belong to another row,
// which is the guarantee the Razorpay paths get from (order id + amount).
func VerifyCashfreeCharge(mode, cfOrderID string, minAmount float64) (*CashfreePayment, error) {
	cf := GetCashfreeFor(mode)
	if cf == nil {
		return nil, fmt.Errorf("cashfree not configured")
	}
	if cfOrderID == "" {
		return nil, fmt.Errorf("no gateway order on this charge")
	}
	pay, err := cf.SuccessfulPayment(cfOrderID)
	if err != nil {
		return nil, fmt.Errorf("fetch cashfree payment: %w", err)
	}
	if pay == nil || !pay.IsCaptured() {
		return nil, fmt.Errorf("payment not captured")
	}
	if pay.AmountPaise.Paise() < ToPaise(minAmount) {
		return nil, fmt.Errorf("payment amount does not match the charge")
	}
	return pay, nil
}

// CashfreeEnvLabel reports SANDBOX/PRODUCTION for a mode. The client cannot infer
// it — sandbox and production are different hosts and the session id looks alike.
func CashfreeEnvLabel(mode string) string {
	if cf := GetCashfreeFor(mode); cf != nil && cf.IsSandbox() {
		return "SANDBOX"
	}
	return "PRODUCTION"
}
