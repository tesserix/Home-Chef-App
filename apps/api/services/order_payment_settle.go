package services

// order_payment_settle.go — the single implementation of "an order's payment is
// captured, so make the order paid" (#872 step 2). Moved out of `handlers`
// (#872 step 2, Task 1) so it can be called from three places instead of two:
// the Cashfree/Razorpay HTTP verify legs (handlers) AND the order-payment
// reconcile cron (order_payment_reconcile_cron.go, this package) — which needed
// this core and could not previously reach it, since `services` cannot import
// `handlers`.
//
// This file grows across the extraction: Task 1 moves the guarded completion
// transaction (CompletionBlockedStatuses, NotifyChefNewOrderTx,
// CompleteOrderPaymentTx, CompleteRazorpayOrderTx, CompleteCashfreeOrderTx).
// Task 2 adds the wallet-settlement cluster (OrderSettlements,
// ApplyChefRecoveryDeduction, DebitOrderWallet, SettleWalletTopUps,
// SettleOrderWallet). Task 3 adds the two gateway-settle cores
// (SettleCashfreeOrder, SettleRazorpayOrderFromPayment).

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
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
// core shared by the Razorpay / Stripe / wallet / cron verify+settle paths (#395 item 2,
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

// CompleteRazorpayOrderTx is the Razorpay-specific wrapper over CompleteOrderPaymentTx
// (#395). The caller's post-tx steps (referral reward, saga, wallet DEBIT) are
// idempotently keyed; the wallet TOP-UP transfer is NOT (see #395 follow-up) — but this
// helper does not invoke it, and on a raced (RowsAffected==0) verify the top-up
// recomputes the same deterministic split, so this does not add a double-transfer path.
func CompleteRazorpayOrderTx(tx *gorm.DB, order *models.Order, method, paymentID string, amountPaise int) (bool, error) {
	return CompleteOrderPaymentTx(tx, order,
		map[string]interface{}{"payment_method": method, "razorpay_payment_id": paymentID},
		map[string]interface{}{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"amount":       FromPaise(amountPaise),
			"method":       method,
			"provider":     "razorpay",
		})
}

// CompleteCashfreeOrderTx is the Cashfree wrapper over the provider-generic
// CompleteOrderPaymentTx, mirroring CompleteRazorpayOrderTx. The guarded UPDATE
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

// OrderSettlements derives the chef + driver payouts for an order, chef first so
// its (larger) food payout stays a single payment-linked transfer when the capture
// allows (#141). The chef slice is NET (ChefNetPayoutFor, reading the frozen
// order.CommissionRate), further reduced by any outstanding recovery balance the
// chef owes the platform (#741, ApplyChefRecoveryDeduction) before the transfer is
// created — Route transfers are per-payment, so a penalty cannot be netted across
// orders after the fact the way a daily batch would. The chef account is cleared
// when its FSSAI licence has lapsed, so that slice is withheld and never
// transferred. Requires Chef and Delivery.DeliveryPartner preloaded. Deterministic —
// because the rate is frozen on the row, create and verify produce the identical
// split for a given order (recovery is re-derived from the ledger identically both
// times, since nothing here mutates it).
//
// LANDMINE: this is one of TWO uncoordinated places that act on the same
// non-discharging recovery debt — see ApplyChefRecoveryDeduction's doc comment
// below, and services/payout_release_cron.go's BuildReleaseInput (the
// RecoveryBalance sweep block), for the full explanation before touching either.
func OrderSettlements(db *gorm.DB, order *models.Order) []Settlement {
	chefAccount := order.Chef.RazorpayAccountID
	if IsChefFSSAIExpired(&order.Chef) {
		chefAccount = ""
	}
	chefAmount := ToPaise(ChefNetPayoutFor(order))
	if chefAccount != "" {
		chefAmount = ApplyChefRecoveryDeduction(db, order, chefAmount)
	}

	driverAccount := ""
	if order.Delivery != nil {
		driverAccount = order.Delivery.DeliveryPartner.RazorpayAccountID
	}
	return []Settlement{
		{Account: chefAccount, Amount: chefAmount, Hold: true,
			Notes: map[string]string{"purpose": "food_payment", "order_number": order.OrderNumber}},
		{Account: driverAccount, Amount: ToPaise(order.DeliveryFee + order.DriverTip), Hold: true,
			Notes: map[string]string{"purpose": "delivery_payment", "order_number": order.OrderNumber}},
	}
}

// ApplyChefRecoveryDeduction reduces a chef's gross transfer (in paise) by
// whatever recovery balance they still owe the platform (#741) — a penalty
// raised against them (e.g. an order-issue clawback) that could not be netted
// against a Route transfer already sent, so it comes off the next one instead.
//
// Fails OPEN on a ledger read error: pays the unadjusted gross. A comparison
// we cannot make must not silently confiscate a chef's whole payout for this
// order — that would turn a transient DB error into a permanent, unrecorded
// loss with no reconcile path revisiting it. The debt is not lost by paying
// gross here: ApplyRecoveryDeduction only reads the ledger and never
// discharges it, so the same outstanding balance is re-derived and correctly
// deducted from this chef's next order regardless of whether this read
// succeeded.
//
// LANDMINE (final money-safety review): this checkout-time reduction and the
// sweep's release-time block (BuildReleaseInput's RecoveryBalance in
// services/payout_release_cron.go) are two UNCOORDINATED places handling the
// SAME debt — one reduces the transfer here, the other blocks release there,
// and neither knows the other exists. Recovery is non-discharging (see
// services/payout_recovery.go): nothing anywhere writes a resolving ledger
// entry, so the same full debt is re-derived and can be re-applied by BOTH
// sites against the SAME outstanding balance. No penalty/ledger writer may
// ship until exactly one of these two mechanisms actually collects-and-
// discharges the debt and the other is changed to defer to it — do not add a
// third site, and do not wire a discharging writer to only one of the two
// without also fixing the other.
//
// On the success path, a deduction that actually reduces the transfer
// (deducted > 0) writes a system audit row — order, chef, gross, and deducted
// paise only, nothing else — so a reduced payout is never silent.
func ApplyChefRecoveryDeduction(db *gorm.DB, order *models.Order, grossPaise int) int {
	gross := payouts.Money{Minor: int64(grossPaise), Currency: payouts.CurrencyINR}
	net, deducted, err := ApplyRecoveryDeduction(db, order.ChefID, gross, time.Now())
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

// SettleWalletTopUps funds the chef/driver portion that the gateway capture could
// not cover, via direct transfers from the platform balance (#141). Failures are
// logged but not fatal — the money is already captured and the reconciliation job
// retries; failing here would wrongly tell the client the order is unpaid.
func SettleWalletTopUps(order *models.Order, topUps []TransferSpec) {
	rz := GetRazorpayFor(order.Mode)
	if rz == nil {
		return
	}
	settleWalletTopUpsWith(order.ID, order.OrderNumber, topUps, func(leg int, t TransferSpec) error {
		_, err := rz.CreateTransfer(&DirectTransferRequest{
			Account: t.Account, Amount: t.Amount, Currency: t.Currency, OnHold: t.OnHold, Notes: t.Notes,
			// Per (order, leg-index, account) — the same identity as the processed_events claim
			// (#554/#558); a retried settlement re-derives the same key so Razorpay dedups each
			// chef/driver top-up, and two legs sharing one account stay independently keyed. #574.
			IdempotencyKey: TopupIdempotencyKey(order.ID, leg, t.Account),
		})
		return err
	})
}

// settleWalletTopUpsWith issues each platform-funded top-up transfer AT MOST ONCE per
// (order, account), idempotently (#554). A retried VerifyPayment used to re-issue the
// same real money transfer because CreateTransfer had no dedup. Now each (order,
// account) is claimed in the processed_events ledger before the transfer; a repeat
// settlement finds the claim and skips. On a transfer failure the claim is released so
// the NEXT settlement re-attempts it — so a gateway blip retries without ever
// double-paying. doTransfer is the gateway seam (real in prod, a fake in tests). A
// crash between claim and a successful transfer strands that one top-up (recoverable
// by the settlement reconcile — #398/#3), which is the safe side of the trade-off:
// never a double transfer.
func settleWalletTopUpsWith(orderID uuid.UUID, orderNumber string, topUps []TransferSpec, doTransfer func(leg int, t TransferSpec) error) {
	// #558: key each leg by its index in the deterministic DirectTopUps list (stable across
	// retries), so two legs sharing one Razorpay payout account stay independently idempotent.
	for leg, t := range topUps {
		firstTime, err := ClaimWalletTopUp(database.DB, orderID, leg, t.Account)
		if err != nil {
			log.Printf("wallet-topup: claim failed order=%s leg=%d account=%s: %v", orderNumber, leg, t.Account, err)
			continue
		}
		if !firstTime {
			continue // already transferred for this (order, leg, account) — no double
		}
		if err := doTransfer(leg, t); err != nil {
			ReleaseWalletTopUp(database.DB, orderID, leg, t.Account) // let a retry re-attempt
			log.Printf("wallet-topup: direct transfer failed order=%s leg=%d account=%s amount=%d: %v",
				orderNumber, leg, t.Account, t.Amount, err)
		}
	}
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
	// The chef/driver top-ups only exist to make up what a GATEWAY SPLIT could not
	// cover: Route funds each settlement from the capture as far as it reaches, and
	// the platform balance pays the rest. A provider that doesn't split at the
	// gateway has no shortfall to top up — the whole amount was captured to the
	// platform, and the chef/rider are paid through the statement/payout path. Running
	// the top-ups anyway would pay them a second time out of the platform balance.
	//
	// The debit above still had to happen: the customer's credit was applied and
	// spent regardless of which gateway took the remainder.
	if !models.ProviderSupportsGatewaySplit(order.PaymentProvider) {
		return
	}
	// The top-up plan must reconstruct the FULL credit applied — wallet plus the
	// loyalty slice — or the chef/driver would be short-paid by the points portion.
	appliedPaise := ToPaise(order.WalletApplied) + ToPaise(order.LoyaltyApplied)
	plan := PlanWalletFunding(ToPaise(order.Total), appliedPaise, appliedPaise, OrderSettlements(database.DB, order))
	SettleWalletTopUps(order, plan.DirectTopUps)
}

// SettleCashfreeOrder is the shared "the gateway says this is paid, so make the
// order paid" core (#872 step 2, Task 3), used by the Cashfree HTTP verify leg,
// the already-PAID recovery in the create leg, AND the order-payment reconcile
// cron — all three ask Cashfree the identical question against
// order.RazorpayOrderID (the redundant caller-supplied order id parameter this
// had in `handlers` is gone: both existing callers already guaranteed it equalled
// order.RazorpayOrderID by the time they called it). Returns (false, reason) on
// any gate failure.
//
// Every hard gate the Razorpay verify applies is applied here, through the same
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

	payment, err := cf.SuccessfulPayment(order.RazorpayOrderID)
	if err != nil {
		log.Printf("Failed to fetch Cashfree payments for order %s: %v", order.RazorpayOrderID, err)
		return false, "Could not verify payment with the gateway — please try again in a moment",
			fmt.Errorf("%w: %v", ErrPaymentGatewayFetchFailed, err)
	}
	if payment == nil {
		return false, "Payment not completed", nil
	}

	// Expected capture = Total − wallet − loyalty, identical to the Razorpay leg.
	// Both credit rails shrink the capture at checkout, so omitting either term
	// rejects every credit-funded order with a false "amount does not match".
	expectedPaise := ToPaise(order.Total) - ToPaise(order.WalletApplied) - ToPaise(order.LoyaltyApplied)
	if expectedPaise < 0 {
		expectedPaise = 0
	}

	// ValidateCapturedPayment speaks Razorpay's "captured"; Cashfree's captured
	// state is "SUCCESS". Normalising here — rather than loosening the shared gate
	// or writing a second one — keeps ONE implementation of the binding checks
	// that every settle path in the codebase is required to apply.
	status := payment.PaymentStatus
	if payment.IsCaptured() {
		status = "captured"
	}
	if valid, reason := ValidateCapturedPayment(
		status, payment.OrderID, order.RazorpayOrderID,
		payment.AmountPaise.Paise(), expectedPaise,
	); !valid {
		log.Printf("cashfree settle rejected order=%s: %s (paymentOrder=%s expected=%s amount=%d expectedPaise=%d)",
			order.OrderNumber, reason, payment.OrderID, order.RazorpayOrderID,
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
	}

	// Debit the applied store credit and burn the loyalty points now that the
	// capture is confirmed. Idempotent, and a no-op without credit. There are no
	// chef/driver top-ups to issue — Cashfree captures the whole amount to the
	// platform, so SettleOrderWallet's transfer leg is skipped for this provider
	// (see its provider guard).
	SettleOrderWallet(order)
	return true, "", nil
}

// SettleRazorpayOrderFromPayment is Razorpay's counterpart to SettleCashfreeOrder
// (#872 step 2, Task 3): the same binding-gate-then-complete-then-settle shape,
// but it takes an ALREADY-FETCHED PaymentResponse rather than fetching it itself,
// because HOW it is fetched differs by caller — the HTTP verify leg fetches by the
// client-supplied payment id (rz.FetchPayment, unchanged, same Razorpay endpoint
// hit as before this extraction); the reconcile cron has no client-supplied id, so
// it discovers the captured payment via rz.FetchOrderPayments(order.RazorpayOrderID)
// and passes in whichever entry is captured and bound to this order. Unifying on
// FetchOrderPayments for both callers was considered and rejected: it would change
// which Razorpay endpoint the HTTP verify leg calls and could match a DIFFERENT
// captured payment than the one the client is claiming when an order has multiple
// attempts — a real behavior change on the money-critical path.
//
// Message-text note: this routes the underpayment/binding-mismatch rejections
// through ValidateCapturedPayment, so they now return the SAME generic strings
// Cashfree already does ("Payment not captured", "Payment does not belong to this
// order", "Payment amount does not match the expected amount") instead of the
// bespoke ones the old inline handler check built. No test asserts on the old
// exact text, only on HTTP status code and resulting payment_status — this
// unifies the codebase onto ONE binding-gate implementation, and additionally
// means a Razorpay rejection now gets ValidateCapturedPayment's log line for
// free (a strict improvement for the cron's loud-underpayment requirement, not a
// regression).
func SettleRazorpayOrderFromPayment(order *models.Order, payment *PaymentResponse) (bool, string) {
	// The gateway only captured (Total − WalletApplied − LoyaltyApplied): both
	// store credit AND loyalty points are applied at checkout and shrink the
	// capture identically (see CreateOrderPayment: creditPaise = wallet + points →
	// plan.CapturePaise).
	expectedPaise := ToPaise(order.Total) - ToPaise(order.WalletApplied) - ToPaise(order.LoyaltyApplied)
	if expectedPaise < 0 {
		expectedPaise = 0
	}
	if valid, reason := ValidateCapturedPayment(
		payment.Status, payment.OrderID, order.RazorpayOrderID,
		payment.Amount, expectedPaise,
	); !valid {
		log.Printf("razorpay settle rejected order=%s: %s (paymentOrder=%s expected=%s amount=%d expectedPaise=%d)",
			order.OrderNumber, reason, payment.OrderID, order.RazorpayOrderID, payment.Amount, expectedPaise)
		return false, reason
	}

	// Mark the order paid and stage the chef push + order.paid event atomically
	// (transactional outbox). Payment already captured at the gateway, so a DB
	// hiccup must not fail the caller — it's logged + sent to Sentry and the
	// reconciliation cron catches any drift.
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		_, err := CompleteRazorpayOrderTx(tx, order, payment.Method, payment.ID, payment.Amount)
		return err
	}); err != nil {
		log.Printf("Failed to persist payment completion + event for order %s: %v", order.ID, err)
		CaptureBackgroundError(err)
	} else {
		// Referral reward (#38) on the referee's first paid order — idempotent,
		// so a later webhook/cron for the same order won't double-pay.
		MaybeGrantReward(database.DB, order.ID)
		// Start the durable order saga (#122) — gated, idempotent, no-op when off.
		StartOrderSaga(order.ID)
		NotifyPaymentSucceeded(database.DB, order.ID)
	}

	// Wallet-at-checkout settlement (#141): now that the gateway capture is
	// confirmed, debit the applied store credit and top up the chef/driver portion
	// the capture couldn't cover, from the platform balance. Idempotent; a no-op
	// without credit.
	SettleOrderWallet(order)
	return true, ""
}
