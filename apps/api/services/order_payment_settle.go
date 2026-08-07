package services

// order_payment_settle.go — the single implementation of "an order's payment is
// captured, so make the order paid" (#872 step 2). Moved out of `handlers`
// (#872 step 2, Task 1) so it can be called from three places instead of two:
// the Cashfree HTTP verify legs (handlers) AND the order-payment
// reconcile cron (order_payment_reconcile_cron.go, this package) — which needed
// this core and could not previously reach it, since `services` cannot import
// `handlers`.
//
// This file grows across the extraction: Task 1 moves the guarded completion
// transaction (CompletionBlockedStatuses, NotifyChefNewOrderTx,
// CompleteOrderPaymentTx, CompleteCashfreeOrderTx).
// Task 2 adds the wallet-settlement cluster (OrderSettlements,
// ApplyChefRecoveryDeduction, DebitOrderWallet, SettleOrderWallet). Task 3 adds
// the two gateway-settle cores
// (SettleCashfreeOrder).

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

// ErrPaymentGatewayUnavailable means the gateway client for this order's mode
// is not configured — a server-side misconfiguration, not something the
// client caused or can retry into working (#872 final item).
var ErrPaymentGatewayUnavailable = errors.New("payment gateway not configured")

// ErrPaymentGatewayFetchFailed means the mandatory server-side fetch of the
// gateway's own payment record failed — transport error, timeout, 5xx, or an
// unparseable response. This is distinct from the fetch succeeding and
// reporting no captured payment: that is a terminal "genuinely unpaid"
// outcome, this is a retryable upstream failure (#872 final item).
var ErrPaymentGatewayFetchFailed = errors.New("failed to fetch payment from gateway")

// NotifyChefNewOrderTx stages the actionable "new order" push to the chef
// within a payment-completion transaction. Orders are created pre-payment, so
// the chef is only notified once money is captured (previously this fired in
// CreateOrder, pushing unpaid/abandoned orders to the kitchen — see
// handlers/orders.go). Callers MUST guard the call on the pending→completed
// transition so a webhook arriving after the client verify (or vice-versa)
// doesn't double-notify; the OrderEvent mirrors the one CreateOrder used.
func NotifyChefNewOrderTx(tx *gorm.DB, order *models.Order) error {
	return EnqueueOrderEvent(tx, SubjectChefNewOrder, OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	})
}

// CompletionBlockedStatuses are the payment states a success/capture/verify must NOT
// re-complete from: already `completed` (idempotent — no duplicate side effects) and
// `refunded` (a refund is terminal; re-stamping it `completed` would silently re-enable
// the chef payout on money already returned to the customer — #563). `failed` is
// deliberately NOT here, so a retry-after-decline can still complete the order.
var CompletionBlockedStatuses = []models.PaymentStatus{models.PaymentCompleted, models.PaymentRefunded}

// CompleteOrderPaymentTx flips an order pending→completed exactly ONCE and, only on
// that single transition, notifies the chef and emits order.paid — the provider-generic
// core shared by the gateway / wallet / cron verify+settle paths (#395 item 2,
// #555, #872 step 2). Every one of them used to read `wasUnpaid` from the in-memory
// order and then update unconditionally + emit order.paid unconditionally, so a webhook
// or re-verify that completed the order underneath left a duplicate chef "new order"
// push AND a duplicate order.paid event. The guarded UPDATE (WHERE payment_status <>
// 'completed') makes exactly one racing path perform the transition; the chef notify +
// event fire only on RowsAffected>0. `updates` are the provider-specific columns to
// stamp alongside payment_status=completed; `event` is the order.paid payload. Returns
// whether THIS call performed the transition.
func CompleteOrderPaymentTx(tx *gorm.DB, order *models.Order, updates, event map[string]interface{}) (bool, error) {
	cols := map[string]interface{}{"payment_status": models.PaymentCompleted}
	for k, v := range updates {
		cols[k] = v
	}
	res := tx.Model(&models.Order{}).
		Where("id = ? AND payment_status NOT IN ?", order.ID, CompletionBlockedStatuses).
		Updates(cols)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil // a concurrent webhook/verify already completed it — no re-fire
	}
	// Keep the in-memory order consistent for any post-tx idempotent steps.
	order.PaymentStatus = models.PaymentCompleted
	if err := NotifyChefNewOrderTx(tx, order); err != nil {
		return false, err
	}
	if err := EnqueueEvent(tx, "orders.paid", "order.paid", order.CustomerID, event); err != nil {
		return false, err
	}
	// #937: the risk denominator. Every claim rate is meaningless without a count of what
	// the customer actually bought, and this guarded transition is the one place a paid
	// order is counted exactly once across all providers. Savepoint-isolated inside.
	chefID := order.ChefID
	TrackRiskEvent(tx, RecordRiskEventInput{
		CustomerID: order.CustomerID,
		Kind:       models.RiskOrderPlaced,
		SourceKey:  "order:" + order.ID.String(),
		OrderID:    &order.ID,
		ChefID:     &chefID,
		Amount:     order.Total,
	})
	return true, nil
}

// CompleteCashfreeOrderTx is the Cashfree wrapper over the provider-generic
// CompleteOrderPaymentTx. The guarded UPDATE
// inside CompleteOrderPaymentTx is what makes exactly one of a racing
// verify/webhook/cron trio perform the pending→completed transition, so the chef push
// and order.paid event fire once.
func CompleteCashfreeOrderTx(tx *gorm.DB, order *models.Order, method, cfPaymentID string, amountPaise int) (bool, error) {
	return CompleteOrderPaymentTx(tx, order,
		map[string]interface{}{
			"payment_method":              method,
			models.GatewayPaymentIDColumn: cfPaymentID,
		},
		map[string]interface{}{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"amount":       FromPaise(amountPaise),
			"method":       method,
			"provider":     models.PaymentProviderCashfree,
		})
}

// ApplyChefRecoveryDeduction reduces a chef's gross transfer (in paise) by
// whatever recovery balance they still owe the platform (#741) — a penalty
// raised against them (e.g. an order-issue clawback) that could not be netted
// against a Route transfer already sent, so it comes off the next one instead.
//
// This is the ONE site that collects (#1079). It is where money is actually
// withheld, so it is where the resolving ledger entry is written; the sweep's
// release-time check (BuildReleaseInput in services/payout_release_cron.go)
// only reads, and defers to the discharge this makes. Collection is idempotent
// on the order, so a retried settle re-derives the same net without collecting
// twice.
//
// Fails OPEN on a ledger read error: pays the unadjusted gross. A comparison we
// cannot make must not silently confiscate a chef's whole payout for this order
// — that would turn a transient DB error into a permanent, unrecorded loss with
// no reconcile path revisiting it. Nothing is discharged on that path either,
// so the debt is still owed and is collected from the next order.
//
// On the success path, a deduction that actually reduces the transfer
// (deducted > 0) writes a system audit row — order, chef, gross, and deducted
// paise only, nothing else — so a reduced payout is never silent.
func ApplyChefRecoveryDeduction(db *gorm.DB, order *models.Order, grossPaise int) int {
	gross := payouts.Money{Minor: int64(grossPaise), Currency: payouts.CurrencyINR}
	net, deducted, err := CollectRecoveryDeduction(db, order.ChefID, gross, "order", order.ID.String(), time.Now())
	if err != nil {
		log.Printf("recovery-deduction: ledger read failed, paying gross order=%s chef=%s gross_paise=%d: %v",
			order.OrderNumber, order.ChefID, grossPaise, err)
		CaptureBackgroundError(fmt.Errorf(
			"recovery-deduction: order=%s chef=%s: %w", order.OrderNumber, order.ChefID, err))
		return grossPaise
	}
	if deducted.Minor > 0 {
		LogSystemAudit(nil, "chef.payout.recovery_deducted", "chef", order.ChefID.String(), nil, map[string]any{
			"orderId":       order.ID.String(),
			"orderNumber":   order.OrderNumber,
			"chefId":        order.ChefID.String(),
			"grossPaise":    grossPaise,
			"deductedPaise": deducted.Minor,
			"netPaise":      net.Minor,
		})
	}
	return int(net.Minor)
}

// DebitOrderWallet debits the customer's store credit for the wallet applied to an
// order, idempotent on the order so a retry never double-debits (#141).
func DebitOrderWallet(order *models.Order) error {
	if order.WalletApplied <= 0 {
		return nil
	}
	_, err := DebitWallet(database.DB, order.CustomerID, order.WalletApplied,
		models.WalletSourceOrderPayment, &order.ID, "checkout", "wallet-debit:"+order.ID.String(), nil)
	return err
}

// SettleOrderWallet settles the wallet-at-checkout slice once a gateway capture is
// confirmed: debit the applied store credit and issue the platform-funded chef/driver
// top-ups the capture couldn't cover (#141). Both are idempotent (DebitWallet keyed
// per order; each top-up claimed per (order, account) — #554), so it runs safely from
// BOTH the client verify path AND the payment.captured webhook (#395·3): whichever
// confirms the capture first settles, a duplicate is a no-op, and a retried webhook
// re-attempts a settlement whose first delivery crashed mid-way. No-op when there is no
// applied credit. REQUIRES order.Chef + order.Delivery.DeliveryPartner preloaded — the
// top-up split reads their Route accounts (an un-preloaded order would top up "").
func SettleOrderWallet(order *models.Order) {
	// Loyalty points are burned on the SAME seam as the wallet debit — after the
	// capture is confirmed, keyed to the order — so an abandoned or failed checkout
	// never costs the customer their points. Idempotent per order.
	if order.LoyaltyPointsSpent > 0 {
		if err := RedeemLoyaltyToOrder(database.DB, order.CustomerID, order.ID, order.LoyaltyPointsSpent); err != nil {
			log.Printf("loyalty-debit failed order=%s: %v", order.OrderNumber, err)
			CaptureBackgroundError(err)
		}
	}
	if order.WalletApplied <= 0 && order.LoyaltyApplied <= 0 {
		return
	}
	if err := DebitOrderWallet(order); err != nil {
		// A GENUINE debit failure (e.g. ErrInsufficientWalletBalance if the balance
		// was drained between checkout and this now-delayed settlement) must NOT go on
		// to fund the chef/driver top-up — that would pay them the wallet-covered slice
		// off store credit the platform never collected. Leave the slice unsettled for
		// the reconcile/ops path. DebitOrderWallet returns nil on the idempotent
		// already-debited case, so this only bites a true first-time failure and never
		// blocks a legitimate re-settlement.
		log.Printf("wallet-debit failed order=%s: %v", order.OrderNumber, err)
		CaptureBackgroundError(err)
		return
	}
}

// SettleCashfreeOrder is the shared "the gateway says this is paid, so make the
// order paid" core (#872 step 2, Task 3), used by the Cashfree HTTP verify leg,
// the already-PAID recovery in the create leg, AND the order-payment reconcile
// cron — all three ask Cashfree the identical question against
// order.GatewayOrderID (the redundant caller-supplied order id parameter this
// had in `handlers` is gone: both existing callers already guaranteed it equalled
// order.GatewayOrderID by the time they called it). Returns (false, reason) on
// any gate failure.
//
// Every hard gate the HTTP verify applies is applied here, through the same
// ValidateCapturedPayment: the payment must be captured, belong to THIS gateway
// order, and cover the expected amount. Without that binding any successful
// payment on the merchant account could be replayed to settle a different order
// for free.
//
// The third return value carries the retryable/terminal distinction (#872
// final item): nil on every terminal branch (payment==nil, a
// ValidateCapturedPayment rejection, and the success path), and one of
// ErrPaymentGatewayUnavailable / ErrPaymentGatewayFetchFailed (wrapped) on the
// two branches a caller should treat as retryable rather than "not paid".
func SettleCashfreeOrder(order *models.Order) (bool, string, error) {
	cf := GetCashfreeFor(order.Mode)
	if cf == nil {
		return false, "Payment gateway not configured", ErrPaymentGatewayUnavailable
	}

	payment, err := cf.SuccessfulPayment(order.GatewayOrderID)
	if err != nil {
		log.Printf("Failed to fetch Cashfree payments for order %s: %v", order.GatewayOrderID, err)
		return false, "Could not verify payment with the gateway — please try again in a moment",
			fmt.Errorf("%w: %v", ErrPaymentGatewayFetchFailed, err)
	}
	if payment == nil {
		return false, "Payment not completed", nil
	}

	// Expected capture = Total − wallet − loyalty, identical to the HTTP verify leg.
	// Both credit rails shrink the capture at checkout, so omitting either term
	// rejects every credit-funded order with a false "amount does not match".
	expectedPaise := ToPaise(order.Total) - ToPaise(order.WalletApplied) - ToPaise(order.LoyaltyApplied)
	if expectedPaise < 0 {
		expectedPaise = 0
	}

	// ValidateCapturedPayment speaks a gateway's "captured"; Cashfree's captured
	// state is "SUCCESS". Normalising here — rather than loosening the shared gate
	// or writing a second one — keeps ONE implementation of the binding checks
	// that every settle path in the codebase is required to apply.
	status := payment.PaymentStatus
	if payment.IsCaptured() {
		status = "captured"
	}
	if valid, reason := ValidateCapturedPayment(
		status, payment.OrderID, order.GatewayOrderID,
		payment.AmountPaise.Paise(), expectedPaise,
	); !valid {
		log.Printf("cashfree settle rejected order=%s: %s (paymentOrder=%s expected=%s amount=%d expectedPaise=%d)",
			order.OrderNumber, reason, payment.OrderID, order.GatewayOrderID,
			payment.AmountPaise.Paise(), expectedPaise)
		return false, reason, nil
	}

	cfPaymentID := payment.CFPaymentID.String()

	// Mark paid + stage the chef push and order.paid event atomically. The payment
	// is already captured at the gateway, so a DB hiccup must not fail the caller —
	// it is logged, sent to Sentry, and the reconcile cron catches the drift.
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		_, err := CompleteCashfreeOrderTx(tx, order, payment.MethodLabel(), cfPaymentID, payment.AmountPaise.Paise())
		return err
	}); err != nil {
		log.Printf("Failed to persist cashfree payment completion for order %s: %v", order.ID, err)
		CaptureBackgroundError(err)
	} else {
		MaybeGrantReward(database.DB, order.ID)
		StartOrderSaga(order.ID)
		NotifyPaymentSucceeded(database.DB, order.ID)
		// End the durable payment poll now rather than on its next tick.
		// Best-effort: the poll reaches the same answer by itself.
		SignalPaymentResolved(order.ID)
	}

	// Debit the applied store credit and burn the loyalty points now that the
	// capture is confirmed. Idempotent, and a no-op without credit. There are no
	// chef/driver top-ups to issue — Cashfree captures the whole amount to the
	// platform, so SettleOrderWallet's transfer leg is skipped for this provider
	// (see its provider guard).
	SettleOrderWallet(order)
	return true, "", nil
}
