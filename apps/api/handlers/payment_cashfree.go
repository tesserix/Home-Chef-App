package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// payment_cashfree.go — the Cashfree half of the order payment flow: create,
// verify, and webhook. Split out of payment.go (already ~2k lines) because the
// three legs are one cohesive flow and because the Cashfree differences are
// easier to review as a block than interleaved with the Razorpay ones.
//
// The money semantics are IDENTICAL to the Razorpay path — same credit quote,
// same frozen commission rate, same completeOrderPaymentTx transition guard,
// same wallet settlement seam, same refund coordinator. The only differences are
// the ones the gateway forces:
//
//   - No gateway split. Cashfree captures the whole payable amount to the
//     platform merchant account; the chef and rider are paid through the
//     statement/payout path. See models.ProviderSupportsGatewaySplit.
//   - No client-side signature to verify. The authority is a server-side fetch
//     of the order's payments.
//   - Both ids land in the razorpay_order_id / razorpay_payment_id columns. That
//     reuse is deliberate and documented on models.GatewayOrderIDColumn: it keeps
//     the partial unique indexes, the reconcile cron, the meal-plan advance
//     lookup and the tip settle path on ONE query each.

// createCashfreePayment creates (or re-uses) a Cashfree order and hands the
// client the payment_session_id its SDK opens checkout with.
//
// The credit path is the Razorpay path, deliberately: BuildCreditQuote is the
// same call /quote makes, so the figure the customer was shown and the figure
// charged here come from one computation. PlanWalletFunding is called with NO
// settlements — that yields the identical CapturePaise arithmetic while producing
// no transfers and no top-ups, which is exactly right for a gateway that does not
// split.
func (h *PaymentHandler) createCashfreePayment(c *gin.Context, order *models.Order, userID uuid.UUID, creditReq services.CreditRequest) {
	cf := services.GetCashfreeFor(order.Mode)
	if cf == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	totalPaise := services.ToPaise(order.Total)

	// FSSAI hard lockout (#32/#93): record the regulatory trail exactly as the
	// Razorpay path does. There is no transfer to suppress here — nothing is split
	// at the gateway — but the chef's payout still has to be withheld downstream,
	// and the audit entry is what the payout path and the regulator both read.
	if services.IsChefFSSAIExpired(&order.Chef) {
		chefAmount := chefNetPayout(order)
		middleware.RecordFSSAILockout("payout_withheld")
		log.Printf("fssai-lockout: withholding chef payout order=%s chef=%s amount=%.2f (cashfree)",
			order.OrderNumber, order.Chef.ID, chefAmount)
		services.LogSystemAudit(c, "chef.payout.fssai_withheld", "chef", order.Chef.ID.String(), nil, map[string]any{
			"orderNumber":    order.OrderNumber,
			"withheldAmount": chefAmount,
			"reason":         "fssai_licence_expired",
			"provider":       models.PaymentProviderCashfree,
		})
	}

	quote, err := services.BuildCreditQuote(database.DB, order, userID, creditReq, creditFlags())
	if err != nil {
		log.Printf("credit-quote failed order=%s: %v", order.OrderNumber, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not price this order"})
		return
	}
	walletApplied := services.FromPaise(quote.WalletAppliedPaise)
	loyaltyApplied := services.FromPaise(quote.PointsAppliedPaise)

	creditPaise := quote.WalletAppliedPaise + quote.PointsAppliedPaise
	// nil settlements: same capture arithmetic, no gateway split. Passing the real
	// settlements here would build Route TransferSpecs that no Cashfree call
	// consumes — they'd be silently dropped, and a reader would reasonably assume
	// the chef was being paid at the gateway when they aren't.
	plan := services.PlanWalletFunding(totalPaise, creditPaise, creditPaise, nil)

	// Fully-credit-covered order: no gateway leg at all. Shared with the Razorpay
	// path — it stamps provider=wallet, so nothing downstream looks for a Cashfree
	// payment that was never made.
	if plan.FullWallet {
		h.settleFullWalletOrder(c, order, plan, walletApplied, loyaltyApplied, quote.PointsAppliedPoints)
		return
	}

	cfOrderID := nextCashfreeOrderID(order.ID, order.RazorpayOrderID)

	// Re-use an existing ACTIVE Cashfree order rather than minting a second one for
	// the same purchase. This matters more than it looks: a payment_session_id is
	// short-lived, so "Pay now" on an unpaid order comes back here routinely. If a
	// second order id were minted while the first was still payable, a customer who
	// completed the FIRST session would generate a webhook for an order id we no
	// longer store — money captured against an order nothing recognises. Reuse
	// keeps exactly one live session per order.
	if order.RazorpayOrderID != "" && order.RazorpayOrderID == cfOrderID {
		if existing, ferr := cf.FetchOrder(order.RazorpayOrderID); ferr == nil {
			if existing.OrderStatus == services.CashfreeOrderPaid {
				// Already paid at the gateway but our row says otherwise — the verify
				// call and the webhook were both lost. Don't open a new checkout;
				// settle it from the gateway's own record.
				h.finishCashfreeFromGateway(c, order, existing)
				return
			}
			if existing.IsPayable() && existing.AmountPaise.Paise() == plan.CapturePaise {
				h.respondCashfreeSession(c, order, cf, existing, plan.CapturePaise, walletApplied, loyaltyApplied, quote)
				return
			}
			// Payable but for a DIFFERENT amount (credit applied or removed since):
			// that session would charge the wrong figure, so fall through and mint a
			// fresh order id below.
			if existing.IsPayable() {
				cfOrderID = bumpCashfreeOrderID(cfOrderID)
				log.Printf("cashfree: order %s session amount changed (%d → %d paise) — minting %s",
					order.OrderNumber, existing.AmountPaise.Paise(), plan.CapturePaise, cfOrderID)
			}
		} else {
			log.Printf("cashfree: could not read existing order %s for %s (%v) — creating a new one",
				order.RazorpayOrderID, order.OrderNumber, ferr)
		}
	}

	cfOrder, err := cf.CreateOrder(&services.CashfreeOrderRequest{
		OrderID:     cfOrderID,
		AmountPaise: services.CashfreeAmountFromPaise(plan.CapturePaise),
		Currency:    "INR",
		Customer: services.CashfreeCustomerDetails{
			// customer_id must be stable per customer (Cashfree keys saved
			// instruments off it) and alphanumeric — a UUID with the hyphens
			// stripped satisfies both.
			CustomerID:    strings.ReplaceAll(userID.String(), "-", ""),
			CustomerPhone: order.Customer.Phone,
			CustomerName:  strings.TrimSpace(order.Customer.FirstName + " " + order.Customer.LastName),
			CustomerEmail: order.Customer.Email,
		},
		// order_meta is omitted entirely: no return_url and no notify_url.
		//
		// return_url is unnecessary — the SDK/modal flows resolve in-page, and the
		// authority for "was this paid" is the server-side fetch in
		// verifyCashfreePayment plus the webhook, never a browser redirect we could
		// be walked into. notify_url is omitted so webhooks keep coming to the ONE
		// dashboard-configured endpoint whose secret we verify against; a per-order
		// URL would route deliveries somewhere the signature check doesn't cover.
		Tags: map[string]string{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"customer_id":  userID.String(),
		},
		OrderNote: fmt.Sprintf("HomeChef order %s", order.OrderNumber),
		// Per (order, capture amount): a retry re-derives the same key so Cashfree
		// dedups it, while a genuinely re-priced order gets a distinct one.
		IdempotencyKey: fmt.Sprintf("cf-order:%s:%d", order.ID, plan.CapturePaise),
	})
	if err != nil {
		// LAST-RESORT GATEWAY FALLBACK.
		//
		// services.SelectCheckoutGateway already skips Cashfree when the slot has no
		// credentials, but "has credentials" is not "credentials work". Cashfree
		// separates sandbox from production by HOSTNAME, so a live slot holding test
		// credentials resolves to a perfectly valid client that then 401s against
		// api.cashfree.com — the normal state while a merchant account is still in
		// review. Without this branch, making Cashfree the preferred provider would
		// turn that into a hard failure on every real checkout.
		//
		// Nothing has been stamped on the order yet (the gateway call comes before
		// the payment-column write), so handing off to Razorpay is clean: it stamps
		// payment_provider=razorpay itself, and the order ends up wholly on one
		// gateway. Deliberately NOT retried on Cashfree — a credential problem does
		// not fix itself within one request.
		log.Printf("cashfree: order create failed for %s (%v) — falling back to razorpay for this checkout",
			order.OrderNumber, err)
		if services.GetRazorpayFor(order.Mode) != nil {
			h.createRazorpayPayment(c, order, userID, creditReq)
			return
		}
		log.Printf("cashfree: no razorpay fallback available for %s either", order.OrderNumber)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
		return
	}
	if !cfOrder.IsPayable() {
		// A freshly created order that isn't payable means Cashfree accepted it but
		// won't take money for it (expired/terminated). Surfacing 500 rather than
		// handing the client a dead session is the honest answer.
		log.Printf("cashfree: freshly created order %s is not payable (status=%s)", cfOrder.OrderID, cfOrder.OrderStatus)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
		return
	}

	// Omit(clause.Associations) for the same reason the Razorpay path does: `order`
	// carries preloaded Customer/Chef/Delivery, and without it GORM cascades an
	// upsert into those rows on every payment stamp — and fails the whole update if
	// any association column mismatches.
	if res := database.DB.Model(order).Omit(clause.Associations).Updates(map[string]interface{}{
		models.GatewayOrderIDColumn: cfOrder.OrderID,
		"payment_provider":          models.PaymentProviderCashfree,
		"wallet_applied":            walletApplied,
		"loyalty_applied":           loyaltyApplied,
		"loyalty_points_spent":      quote.PointsAppliedPoints,
	}); res.Error != nil {
		// Losing this write would strand the customer's applied credit AND leave us
		// unable to recognise the webhook for the order we just created.
		log.Printf("Failed to stamp cashfree payment columns order=%s: %v", order.OrderNumber, res.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
		return
	}
	order.RazorpayOrderID = cfOrder.OrderID
	order.PaymentProvider = models.PaymentProviderCashfree

	h.respondCashfreeSession(c, order, cf, cfOrder, plan.CapturePaise, walletApplied, loyaltyApplied, quote)
}

// respondCashfreeSession writes the checkout hand-off payload.
//
// Shape mirrors the Razorpay response field-for-field where the concept exists
// (provider, amount, payable, walletApplied, prefill) so the clients share one
// branch of credit/summary handling and differ only in which SDK they open.
// `mode` is included because the Cashfree SDKs need to be told SANDBOX vs
// PRODUCTION explicitly — unlike Razorpay, where the key prefix carries it.
func (h *PaymentHandler) respondCashfreeSession(
	c *gin.Context, order *models.Order, cf *services.CashfreeClient,
	cfOrder *services.CashfreeOrderResponse, capturePaise int,
	walletApplied, loyaltyApplied float64, quote services.CreditQuote,
) {
	env := "PRODUCTION"
	if cf.IsSandbox() {
		env = "SANDBOX"
	}
	c.JSON(http.StatusOK, gin.H{
		"provider":                 models.PaymentProviderCashfree,
		"cashfreeOrderId":          cfOrder.OrderID,
		"cashfreePaymentSessionId": cfOrder.PaymentSessionID,
		"cashfreeAppId":            cf.GetAppID(),
		"cashfreeEnv":              env,
		"amount":                   capturePaise,
		"walletApplied":            walletApplied,
		"loyaltyApplied":           loyaltyApplied,
		"pointsApplied":            quote.PointsAppliedPoints,
		"payable":                  services.FromPaise(quote.PayablePaise),
		"currency":                 "INR",
		"orderNumber":              order.OrderNumber,
		"prefill": gin.H{
			"name":  strings.TrimSpace(order.Customer.FirstName + " " + order.Customer.LastName),
			"email": order.Customer.Email,
			"phone": order.Customer.Phone,
		},
	})
}

// finishCashfreeFromGateway settles an order the gateway already shows as PAID
// while our row still says unpaid — both the client verify and the webhook were
// lost. Rather than opening a second checkout (which is how a customer gets
// charged twice), read the captured payment and complete the order from it.
func (h *PaymentHandler) finishCashfreeFromGateway(c *gin.Context, order *models.Order, cfOrder *services.CashfreeOrderResponse) {
	log.Printf("cashfree: order %s already PAID at gateway (%s) — settling from gateway state",
		order.OrderNumber, cfOrder.OrderID)
	if ok, msg := h.settleCashfreeOrder(order, cfOrder.OrderID); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"provider":    models.PaymentProviderCashfree,
		"paid":        true,
		"orderNumber": order.OrderNumber,
		"message":     "Payment already received",
	})
}

// verifyCashfreePayment confirms a payment after the client's checkout closes.
//
// Unlike the Razorpay leg there is NO client-supplied signature to check, and no
// client-supplied amount or payment id is trusted at all: the only inputs are the
// order id (which must match what we stamped) and the gateway's own record. That
// makes this strictly harder to spoof than a signature check — but it also means
// the fetch is mandatory, so a gateway outage surfaces as a 502 rather than being
// waved through.
func (h *PaymentHandler) verifyCashfreePayment(c *gin.Context, order *models.Order, cfOrderID string) {
	if cfOrderID == "" {
		// Fall back to the stamped id: a client that closed checkout without
		// echoing the order id back still deserves a verify, and the stamped value
		// is the trustworthy one anyway.
		cfOrderID = order.RazorpayOrderID
	}
	if cfOrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cashfreeOrderId is required"})
		return
	}
	if order.RazorpayOrderID != cfOrderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID mismatch"})
		return
	}

	ok, msg := h.settleCashfreeOrder(order, cfOrderID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Payment verified", "status": "completed"})
}

// settleCashfreeOrder is the shared "the gateway says this is paid, so make the
// order paid" core, used by the verify leg AND by the already-PAID recovery in
// the create leg. Returns (false, reason) on any gate failure.
//
// Every hard gate the Razorpay verify applies is applied here, through the same
// services.ValidateCapturedPayment: the payment must be captured, belong to THIS
// gateway order, and cover the expected amount. Without that binding any
// successful payment on the merchant account could be replayed to settle a
// different order for free.
func (h *PaymentHandler) settleCashfreeOrder(order *models.Order, cfOrderID string) (bool, string) {
	cf := services.GetCashfreeFor(order.Mode)
	if cf == nil {
		return false, "Payment gateway not configured"
	}

	payment, err := cf.SuccessfulPayment(cfOrderID)
	if err != nil {
		log.Printf("Failed to fetch Cashfree payments for order %s: %v", cfOrderID, err)
		return false, "Failed to verify payment"
	}
	if payment == nil {
		return false, "Payment not completed"
	}

	// Expected capture = Total − wallet − loyalty, identical to the Razorpay leg.
	// Both credit rails shrink the capture at checkout, so omitting either term
	// rejects every credit-funded order with a false "amount does not match".
	expectedPaise := services.ToPaise(order.Total) - services.ToPaise(order.WalletApplied) - services.ToPaise(order.LoyaltyApplied)
	if expectedPaise < 0 {
		expectedPaise = 0
	}

	// ValidateCapturedPayment speaks Razorpay's "captured"; Cashfree's captured
	// state is "SUCCESS". Normalising here — rather than loosening the shared gate
	// or writing a second one — keeps ONE implementation of the binding checks
	// that every verify leg in the codebase is required to apply.
	status := payment.PaymentStatus
	if payment.IsCaptured() {
		status = "captured"
	}
	if valid, reason := services.ValidateCapturedPayment(
		status, payment.OrderID, order.RazorpayOrderID,
		payment.AmountPaise.Paise(), expectedPaise,
	); !valid {
		log.Printf("cashfree verify rejected order=%s: %s (paymentOrder=%s expected=%s amount=%d expectedPaise=%d)",
			order.OrderNumber, reason, payment.OrderID, order.RazorpayOrderID,
			payment.AmountPaise.Paise(), expectedPaise)
		return false, reason
	}

	cfPaymentID := payment.CFPaymentID.String()

	// Mark paid + stage the chef push and order.paid event atomically. The payment
	// is already captured at the gateway, so a DB hiccup must not fail the client —
	// it is logged, sent to Sentry, and the reconcile cron catches the drift.
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		_, err := completeCashfreeOrderTx(tx, order, payment.MethodLabel(), cfPaymentID, payment.AmountPaise.Paise())
		return err
	}); err != nil {
		log.Printf("Failed to persist cashfree payment completion for order %s: %v", order.ID, err)
		services.CaptureBackgroundError(err)
	} else {
		services.MaybeGrantReward(database.DB, order.ID)
		services.StartOrderSaga(order.ID)
	}

	// Debit the applied store credit and burn the loyalty points now that the
	// capture is confirmed. Idempotent, and a no-op without credit. There are no
	// chef/driver top-ups to issue — Cashfree captures the whole amount to the
	// platform, so settleOrderWallet's transfer leg is skipped for this provider
	// (see its provider guard).
	settleOrderWallet(order)
	return true, ""
}

// completeCashfreeOrderTx is the Cashfree wrapper over the provider-generic
// completeOrderPaymentTx, mirroring completeRazorpayOrderTx. The guarded UPDATE
// inside completeOrderPaymentTx is what makes exactly one of a racing
// verify/webhook pair perform the pending→completed transition, so the chef push
// and order.paid event fire once.
func completeCashfreeOrderTx(tx *gorm.DB, order *models.Order, method, cfPaymentID string, amountPaise int) (bool, error) {
	return completeOrderPaymentTx(tx, order,
		map[string]interface{}{
			"payment_method":              method,
			models.GatewayPaymentIDColumn: cfPaymentID,
		},
		map[string]interface{}{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"amount":       services.FromPaise(amountPaise),
			"method":       method,
			"provider":     models.PaymentProviderCashfree,
		})
}

// --- Cashfree order id derivation ---

// nextCashfreeOrderID derives the Cashfree order_id for an order.
//
// The base is our own order UUID: deterministic, globally unique, 36 characters
// of alphanumerics and hyphens — inside Cashfree's 3–45 alphanumeric/-/_ window.
// Deterministic matters because it means a retry naturally lands on the SAME
// Cashfree order and can reuse its live session instead of creating a parallel
// one.
//
// `previous` is whatever is already stamped on the order. When it is a suffixed
// retry of this same order (…-r2) that value is kept, so a second retry can tell
// how many have happened without a counter column. When it belongs to a different
// gateway entirely (a Razorpay id, because the chef's provider changed) it is
// ignored and the base is returned.
func nextCashfreeOrderID(orderID uuid.UUID, previous string) string {
	base := orderID.String()
	if previous == base || strings.HasPrefix(previous, base+"-r") {
		return previous
	}
	return base
}

// bumpCashfreeOrderID advances a Cashfree order id to its next retry suffix,
// used only when the existing session cannot be reused (dead, or priced for a
// different amount). Stays within the 45-character limit: 36 + "-r" + digits.
func bumpCashfreeOrderID(current string) string {
	base, n := current, 1
	if i := strings.LastIndex(current, "-r"); i > 0 {
		if parsed, err := strconv.Atoi(current[i+2:]); err == nil {
			base, n = current[:i], parsed
		}
	}
	return fmt.Sprintf("%s-r%d", base, n+1)
}

// --- Webhooks ---

// cashfreeConsumer names this endpoint in the processed_events dedup ledger.
// Distinct from the Razorpay consumer so the two gateways' event-id spaces cannot
// collide — a shared consumer would let one gateway's event id suppress the
// other's.
const cashfreeConsumer = "webhook:cashfree"

// cashfreeWebhookEnvelope is the common shape of every Cashfree PG webhook.
type cashfreeWebhookEnvelope struct {
	Type      string          `json:"type"`
	EventTime string          `json:"event_time"`
	Data      json.RawMessage `json:"data"`
}

// Cashfree webhook types we act on.
const (
	cfWebhookPaymentSuccess = "PAYMENT_SUCCESS_WEBHOOK"
	cfWebhookPaymentFailed  = "PAYMENT_FAILED_WEBHOOK"
	cfWebhookUserDropped    = "PAYMENT_USER_DROPPED_WEBHOOK"
	cfWebhookRefundStatus   = "REFUND_STATUS_WEBHOOK"
)

// CashfreeWebhook handles Cashfree PG webhook events.
// POST /webhooks/cashfree (no auth — verified via HMAC signature)
//
// Structurally identical to RazorpayWebhook: verify, dedup, dispatch, and release
// the dedup claim on a transient handler error so a redelivery can re-run. The
// signature scheme differs (base64 HMAC over timestamp+rawBody, see
// services.VerifyCashfreeWebhookMode) and the raw body must be used — a
// re-marshalled parse will not match.
func (h *PaymentHandler) CashfreeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
		return
	}

	signature := c.GetHeader(services.CashfreeWebhookSignatureHeader)
	timestamp := c.GetHeader(services.CashfreeWebhookTimestampHeader)
	// signedMode says WHICH credential slot signed this event, and it scopes every
	// database write below, so a live-signed webhook can never mutate a
	// test-partition record or vice versa.
	authentic, signedMode := services.VerifyCashfreeWebhookMode(timestamp, body, signature)
	if !authentic {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	var event cashfreeWebhookEnvelope
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	log.Printf("Cashfree webhook received: %s", event.Type)

	// Event-level replay dedup (#462). The signature proves authenticity, not
	// freshness; without this a provider retry re-fires the side effects (chef
	// push, referral grant) that the per-effect conditional UPDATEs don't cover.
	// Claimed AFTER the signature check so a forged request can't poison the
	// ledger. Cashfree sends no event-id header, so the id is a body hash — which
	// is exactly right here, since a genuine retry re-sends a byte-identical body.
	eventID := services.WebhookEventID("", body)
	firstTime, err := services.ClaimWebhookEvent(database.DB, cashfreeConsumer, eventID, event.Type)
	if err != nil {
		log.Printf("cashfree webhook: claim failed for %s: %v", eventID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dedup unavailable"})
		return
	}
	if !firstTime {
		log.Printf("cashfree webhook: duplicate event %s (%s) — skipping", eventID, event.Type)
		c.JSON(http.StatusOK, gin.H{"status": "duplicate"})
		return
	}

	var derr error
	switch event.Type {
	case cfWebhookPaymentSuccess:
		derr = h.handleCashfreePaymentSuccess(event.Data, signedMode)
	case cfWebhookPaymentFailed, cfWebhookUserDropped:
		derr = h.handleCashfreePaymentUnsuccessful(event.Data, signedMode, event.Type)
	case cfWebhookRefundStatus:
		derr = h.handleCashfreeRefundStatus(event.Data, signedMode)
	default:
		log.Printf("Unhandled Cashfree webhook event: %s", event.Type)
	}
	if derr != nil {
		// Transient failure: release the claim so a later redelivery re-runs,
		// otherwise the dedup would strand the event. Still ACK 200, matching the
		// Razorpay endpoint's contract.
		services.ReleaseWebhookEvent(database.DB, cashfreeConsumer, eventID)
		log.Printf("cashfree webhook: handler error for %s (%s): %v", eventID, event.Type, derr)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// cashfreePaymentEvent is the data block of a payment webhook.
type cashfreePaymentEvent struct {
	Order struct {
		OrderID string `json:"order_id"`
	} `json:"order"`
	Payment services.CashfreePayment `json:"payment"`
}

// handleCashfreePaymentSuccess mirrors handlePaymentCaptured: flip the order to
// completed exactly once, then run the fallbacks for the two order shapes whose
// gateway order id does NOT live on `orders` (a meal-plan advance, a tip charge).
//
// Returns a non-nil error only for a TRANSIENT failure, so the webhook layer
// releases the dedup claim and a redelivery re-runs. A parse failure is permanent
// (nil → keep the claim; a retry won't parse either).
func (h *PaymentHandler) handleCashfreePaymentSuccess(payload json.RawMessage, signedMode string) error {
	var data cashfreePaymentEvent
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse Cashfree payment success payload: %v", err)
		return nil
	}

	cfOrderID := data.Order.OrderID
	if cfOrderID == "" {
		cfOrderID = data.Payment.OrderID
	}
	cfPaymentID := data.Payment.CFPaymentID.String()
	if cfOrderID == "" || cfPaymentID == "" {
		log.Printf("cashfree webhook: payment success with no order/payment id — ignoring")
		return nil
	}
	log.Printf("Cashfree payment success: %s (order: %s, amount: %d paise)",
		cfPaymentID, cfOrderID, data.Payment.AmountPaise.Paise())

	mode := models.NormalizeMode(signedMode)

	// Idempotent update, guarded on payment_status so a retry can't re-fire the
	// downstream effects and a late duplicate can't re-stamp a refunded order back
	// to completed (#563). Scoped by provider as well as mode: the gateway id
	// columns are shared, so the provider filter guarantees a Cashfree event can
	// only ever settle a Cashfree order.
	res := database.DB.Model(&models.Order{}).
		Where(models.GatewayOrderIDColumn+" = ? AND mode = ? AND payment_provider = ? AND payment_status NOT IN ?",
			cfOrderID, mode, models.PaymentProviderCashfree, completionBlockedStatuses).
		Updates(map[string]interface{}{
			"payment_status":              models.PaymentCompleted,
			"payment_method":              data.Payment.MethodLabel(),
			models.GatewayPaymentIDColumn: cfPaymentID,
		})
	if res.Error != nil {
		log.Printf("Failed to apply Cashfree payment success for order %s: %v", cfOrderID, res.Error)
		return res.Error
	}

	if res.RowsAffected == 0 {
		// No regular order matched — this may be a meal-plan ADVANCE, whose gateway
		// order id lives on meal_plans. Confirming it here is what stops a dropped
		// client verify from stranding a captured advance: money taken, plan left
		// awaiting_customer, chef payout never held.
		confirmed, mpErr := h.confirmMealPlanAdvanceFromWebhook(cfOrderID, cfPaymentID)
		if mpErr != nil {
			log.Printf("cashfree payment success: meal-plan advance confirm failed for order %s: %v", cfOrderID, mpErr)
			return mpErr // transient → release the claim so a redelivery re-runs
		}
		if confirmed {
			log.Printf("cashfree payment success: confirmed meal-plan advance for order %s (payment %s)", cfOrderID, cfPaymentID)
		} else {
			log.Printf("cashfree payment success already processed for order %s (payment %s) — skipping", cfOrderID, cfPaymentID)
		}
	} else {
		var ord models.Order
		if err := database.DB.Where(models.GatewayOrderIDColumn+" = ? AND mode = ?", cfOrderID, mode).
			First(&ord).Error; err == nil {
			services.MaybeGrantReward(database.DB, ord.ID)
			services.StartOrderSaga(ord.ID)
			// The guarded update above makes this the single pending→completed
			// transition, so the client verify path won't also push. Best-effort.
			if err := notifyChefNewOrderTx(database.DB, &ord); err != nil {
				log.Printf("Failed to enqueue chef new-order push for order %s: %v", ord.ID, err)
				services.CaptureBackgroundError(err)
			}
		}
	}

	// Settle the wallet-at-checkout slice for a WEBHOOK-driven completion (the
	// client dropped before calling verify). Idempotent, so it is a no-op when
	// verify already settled, and it re-attempts a settlement a crashed earlier
	// delivery left partial — which is why it runs on every delivery, not only the
	// winning one.
	var walletOrd models.Order
	if err := database.DB.Preload("Chef").Preload("Delivery.DeliveryPartner").
		Where(models.GatewayOrderIDColumn+" = ? AND mode = ? AND payment_status = ? AND (wallet_applied > 0 OR loyalty_applied > 0)",
			cfOrderID, mode, models.PaymentCompleted).First(&walletOrd).Error; err == nil {
		settleOrderWallet(&walletOrd)
	}

	// A post-delivery tip is its own gateway order (#45); confirm it here too.
	// Idempotent, and a harmless no-op when this order isn't a tip charge.
	markTipPaidByRazorpayOrder(cfOrderID, cfPaymentID)
	return nil
}

// handleCashfreePaymentUnsuccessful covers PAYMENT_FAILED_WEBHOOK and
// PAYMENT_USER_DROPPED_WEBHOOK.
//
// Both mark the order failed, and both are guarded against overwriting a
// terminal state so an out-of-order delivery can't undo a completed or refunded
// payment. A user-dropped payment is deliberately treated as failed rather than
// left pending: the order is retryable from `failed` (see
// completionBlockedStatuses, which excludes it), so this loses nothing and stops
// abandoned checkouts sitting in `pending` forever.
func (h *PaymentHandler) handleCashfreePaymentUnsuccessful(payload json.RawMessage, signedMode, eventType string) error {
	var data cashfreePaymentEvent
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse Cashfree %s payload: %v", eventType, err)
		return nil
	}
	cfOrderID := data.Order.OrderID
	if cfOrderID == "" {
		cfOrderID = data.Payment.OrderID
	}
	if cfOrderID == "" {
		return nil
	}
	log.Printf("Cashfree %s for order %s (%s)", eventType, cfOrderID, data.Payment.PaymentMessage)

	return database.DB.Model(&models.Order{}).
		Where(models.GatewayOrderIDColumn+" = ? AND mode = ? AND payment_provider = ? AND payment_status NOT IN ?",
			cfOrderID, models.NormalizeMode(signedMode), models.PaymentProviderCashfree,
			[]models.PaymentStatus{models.PaymentCompleted, models.PaymentFailed, models.PaymentRefunded}).
		Update("payment_status", models.PaymentFailed).Error
}

// cashfreeRefundEvent is the data block of REFUND_STATUS_WEBHOOK.
type cashfreeRefundEvent struct {
	Refund services.CashfreeRefund `json:"refund"`
}

// handleCashfreeRefundStatus mirrors handleRefundProcessed, including the #635
// rule that only a FULL refund may stamp refunded_at.
//
// That rule is load-bearing: every per-line cancel and goodwill partial fires a
// refund webhook against the same order, but refunded_at is the WHOLE-order
// marker — the release-side payout guards block the entire chef payout on
// `refunded_at IS NOT NULL`, and claimOrderItemForCancel reads it as "the whole
// order was refunded". Stamping it on a partial would silently freeze a chef's
// payout for an order that was only partly refunded.
//
// Only SUCCESS is acted on. PENDING/ONHOLD carry no settled amount to compare,
// and FAILED/CANCELLED must not stamp a refund that did not happen.
func (h *PaymentHandler) handleCashfreeRefundStatus(payload json.RawMessage, signedMode string) error {
	var data cashfreeRefundEvent
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse Cashfree refund payload: %v", err)
		return nil
	}
	refund := data.Refund
	if refund.OrderID == "" {
		return nil
	}
	if strings.ToUpper(refund.RefundStatus) != services.CashfreeRefundSuccess {
		log.Printf("Cashfree refund %s on order %s is %s — not stamping",
			refund.RefundID, refund.OrderID, refund.RefundStatus)
		return nil
	}
	log.Printf("Cashfree refund succeeded: %s (order: %s, amount: %d paise)",
		refund.RefundID, refund.OrderID, refund.AmountPaise.Paise())

	// One query for total, wallet, loyalty, existing refund id and the per-line
	// refund sum, for the same reason handleRefundProcessed does it in one:
	// reserveOrderItemForCancel reduces order.Total in the SAME tx that records a
	// line refund, so comparing a later webhook's amount against the already-reduced
	// Total would double-count it and mis-classify a partial as full (cancelling the
	// last line drives Total to 0, making ANY refund look full). Separate reads race
	// a concurrent per-line cancel, and because the idempotency short-circuit locks
	// in on refund_id, a missed stamp would be permanent.
	var row struct {
		ID             string
		Total          float64
		WalletApplied  float64
		LoyaltyApplied float64
		RefundID       string
		PerLine        float64
	}
	q := database.DB.Raw(`
		SELECT o.id AS id, o.total AS total, o.wallet_applied AS wallet_applied,
		       o.loyalty_applied AS loyalty_applied, o.refund_id AS refund_id,
		       COALESCE((SELECT SUM(oi.refund_amount) FROM order_items oi
		                 WHERE oi.order_id = o.id AND oi.is_cancelled = ?), 0) AS per_line
		FROM orders o
		WHERE o.`+models.GatewayOrderIDColumn+` = ? AND o.mode = ? AND o.payment_provider = ?
		LIMIT 1`,
		true, refund.OrderID, models.NormalizeMode(signedMode), models.PaymentProviderCashfree).Scan(&row)
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return nil // no order for this gateway order (e.g. a tip refund) — nothing to do
	}
	// Idempotent: Cashfree retries; skip if we already recorded THIS refund id.
	if row.RefundID == refund.RefundID {
		return nil
	}

	// The captured amount is Total (+ per-line refunds added back) − the credit
	// rails, since only the gateway-charged portion can come back through Cashfree.
	capturedPaise := services.ToPaise(row.Total + row.PerLine - row.WalletApplied - row.LoyaltyApplied)
	updates := map[string]interface{}{"refund_id": refund.RefundID}
	if refund.AmountPaise.Paise() >= capturedPaise {
		now := time.Now()
		updates["refunded_at"] = &now
	}
	return database.DB.Model(&models.Order{}).
		Where("id = ? AND (refund_id IS NULL OR refund_id <> ?)", row.ID, refund.RefundID).
		Updates(updates).Error
}
