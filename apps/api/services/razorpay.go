package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services/money"
)

const razorpayBaseURL = "https://api.razorpay.com/v1"

// razorpayCacheTTL controls how often GetRazorpay refetches credentials from
// GCP Secret Manager. Admin writes call InvalidateRazorpay() so the next read
// picks up new keys immediately — the TTL only matters for external rotations
// (e.g. someone bumping the secret version via gcloud).
const razorpayCacheTTL = 5 * time.Minute

// Razorpay credentials live in three GCP Secret Manager secrets. Names are
// product-scoped ("homechef-") so they don't collide with other products
// sharing project tesseracthub-480811. These are the source of truth for
// both reads (GetRazorpay) and writes (admin UpdatePaymentGatewayKeys).
// Exported so the admin write path (UpdatePaymentGatewayKeys) and the read
// path (GetRazorpay) share one source of truth — a previous drift wrote keys
// to "prod-razorpay-*" while reads used "prod-homechef-razorpay-*", so
// admin-entered keys were silently never picked up.
const (
	SecretRazorpayKeyID         = "prod-homechef-razorpay-key-id"
	SecretRazorpayKeySecret     = "prod-homechef-razorpay-key-secret"
	SecretRazorpayWebhookSecret = "prod-homechef-razorpay-webhook-secret"

	// Test-mode credential slot, used by kitchens an admin has marked as test.
	// Deliberately separate secrets rather than a suffix on the values above, so
	// rotating or clearing one slot cannot disturb the other — and so the live
	// names stay exactly as they are, keeping existing production config valid.
	SecretRazorpayTestKeyID         = "prod-homechef-razorpay-test-key-id"
	SecretRazorpayTestKeySecret     = "prod-homechef-razorpay-test-key-secret"
	SecretRazorpayTestWebhookSecret = "prod-homechef-razorpay-test-webhook-secret"
)

// RazorpayClient handles all Razorpay API interactions.
// All sensitive fields are unexported — secrets cannot be read from outside this package.
type RazorpayClient struct {
	keyID         string
	keySecret     string
	webhookSecret string
	// baseURL overrides the Razorpay API host. Empty in every production path
	// (→ razorpayBaseURL); tests point it at an httptest.Server so the transfer
	// seams (hold/release/reverse) can be driven end-to-end without a live gateway.
	baseURL   string
	fetchedAt time.Time
}

// razorpayClients caches one client per mode ("live", "test"). Keyed rather
// than two variables so adding a slot is a data change, not a code change.
var (
	razorpayClients = map[string]*RazorpayClient{}
	razorpayMu      sync.Mutex
)

// isPlaceholderValue treats blank strings and the literal "placeholder"
// (used as a seed value by Helm bootstrap) as "not configured".
func isPlaceholderValue(v string) bool {
	return v == "" || v == "placeholder"
}

// fetchRazorpayFromSM pulls the three Razorpay credentials from GCP Secret
// Manager. The webhook secret is optional; the key ID and key secret are
// required. If SM is unreachable, we fall back to env-provided credentials
// — but only if they look real (not blank, not "placeholder"). This keeps
// local dev working without GCP access.
func fetchRazorpayFromSM(ctx context.Context, mode string) (*RazorpayClient, error) {
	mode = models.NormalizeMode(mode)
	idName, secretName, webhookName := razorpaySecretNames(mode)

	keyID, idErr := GetPlatformSecret(ctx, idName)
	keySecret, secErr := GetPlatformSecret(ctx, secretName)
	// Webhook secret is optional — webhooks simply won't verify if it's missing.
	webhookSecret, _ := GetPlatformSecret(ctx, webhookName)

	if idErr != nil || secErr != nil || isPlaceholderValue(keyID) || isPlaceholderValue(keySecret) {
		// Dev fallback: env-provided credentials (never used in prod since prod
		// relies on Secret Manager).
		//
		// Deliberately LIVE-ONLY. The RAZORPAY_KEY_* env vars predate this
		// feature and describe a single gateway, so applying them to the test
		// slot would serve live credentials to a sandbox order — the precise
		// mix-up this whole feature exists to make impossible. An unconfigured
		// test slot returns nil instead, which callers already handle.
		if !models.IsTestMode(mode) {
			cfg := config.AppConfig
			if !isPlaceholderValue(cfg.RazorpayKeyID) && !isPlaceholderValue(cfg.RazorpayKeySecret) {
				return &RazorpayClient{
					keyID:         cfg.RazorpayKeyID,
					keySecret:     cfg.RazorpayKeySecret,
					webhookSecret: cfg.RazorpayWebhookSecret,
					fetchedAt:     time.Now(),
				}, nil
			}
		}
		if idErr != nil {
			return nil, fmt.Errorf("razorpay[%s] credentials not configured: %w", mode, idErr)
		}
		return nil, fmt.Errorf("razorpay[%s] credentials missing or still set to placeholder — configure them in Admin → Settings → Payment Gateway", mode)
	}

	return &RazorpayClient{
		keyID:         keyID,
		keySecret:     keySecret,
		webhookSecret: webhookSecret,
		fetchedAt:     time.Now(),
	}, nil
}

// RazorpaySecretNames is the exported form used by the admin write path, so the
// slot the admin saves into and the slot the app reads from can never drift
// apart — a drift of exactly that kind once made admin-entered keys silently
// invisible to the app.
func RazorpaySecretNames(mode string) (keyID, keySecret, webhookSecret string) {
	return razorpaySecretNames(mode)
}

// razorpaySecretNames returns the three Secret Manager keys backing a mode's
// credential slot. Separate secrets per slot — rather than one value with a
// mode suffix — mean rotating or clearing one slot cannot disturb the other,
// and the live names are unchanged so everything already configured in
// production keeps resolving.
func razorpaySecretNames(mode string) (keyID, keySecret, webhookSecret string) {
	if models.IsTestMode(mode) {
		return SecretRazorpayTestKeyID, SecretRazorpayTestKeySecret, SecretRazorpayTestWebhookSecret
	}
	return SecretRazorpayKeyID, SecretRazorpayKeySecret, SecretRazorpayWebhookSecret
}

// GetRazorpayFor returns the cached client for one mode, fetching that mode's
// credentials from Secret Manager on a cache miss. The two slots are cached and
// invalidated independently, so a chef in test mode and a chef in live mode can
// transact concurrently against different gateways.
//
// An unrecognised mode resolves to live (see models.NormalizeMode): a typo in a
// caller must never fall through to sandbox credentials, which would appear to
// succeed while capturing no real money.
//
// Returns nil when that slot is not configured. Callers must handle nil, as
// they already did.
func GetRazorpayFor(mode string) *RazorpayClient {
	mode = models.NormalizeMode(mode)

	razorpayMu.Lock()
	defer razorpayMu.Unlock()

	if c := razorpayClients[mode]; c != nil && time.Since(c.fetchedAt) < razorpayCacheTTL {
		return c
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fresh, err := fetchRazorpayFromSM(ctx, mode)
	if err != nil {
		// If the fetch fails but we have a cached client, keep serving it
		// (better than going dark during a transient SM outage). Only the
		// fetchedAt timestamp goes stale.
		if c := razorpayClients[mode]; c != nil {
			log.Printf("razorpay[%s]: SM fetch failed, using cached credentials: %v", mode, err)
			return c
		}
		log.Printf("razorpay[%s]: not configured (%v)", mode, err)
		return nil
	}

	razorpayClients[mode] = fresh
	return fresh
}

// GetRazorpay is the live credential slot. Retained under its original name so
// the non-chef-scoped call sites — admin gateway status, reconciliation, wallet
// top-ups, platform subscriptions — keep compiling and keep meaning exactly
// what they meant before this feature existed.
func GetRazorpay() *RazorpayClient { return GetRazorpayFor(models.ChefModeLive) }

// snapshotRazorpayClient returns one slot's client pointer read under razorpayMu.
// A RazorpayClient's fields are immutable after construction (GetRazorpayFor/
// SetRazorpayClientFor assign a whole new client, never mutate in place), so callers
// may read the returned client's fields without further locking. Used by the
// signature verifiers so a payment/webhook verify can't race a credential refresh
// (GetRazorpayFor post-TTL) or invalidation (#395·5).
func snapshotRazorpayClient(mode string) *RazorpayClient {
	razorpayMu.Lock()
	defer razorpayMu.Unlock()
	return razorpayClients[models.NormalizeMode(mode)]
}

// InvalidateRazorpayFor clears one slot's cached client so the next read
// re-fetches from Secret Manager. Scoped per mode so saving test keys in the
// admin UI cannot knock a healthy live gateway offline.
func InvalidateRazorpayFor(mode string) {
	mode = models.NormalizeMode(mode)
	razorpayMu.Lock()
	defer razorpayMu.Unlock()
	delete(razorpayClients, mode)
	log.Printf("razorpay[%s]: credential cache invalidated", mode)
}

// InvalidateRazorpay clears every slot.
func InvalidateRazorpay() {
	InvalidateRazorpayFor(models.ChefModeLive)
	InvalidateRazorpayFor(models.ChefModeTest)
}

// SetRazorpayClientFor installs a client into one slot directly. Test seam only.
func SetRazorpayClientFor(mode string, c *RazorpayClient) {
	mode = models.NormalizeMode(mode)
	razorpayMu.Lock()
	defer razorpayMu.Unlock()
	if c != nil && c.fetchedAt.IsZero() {
		c.fetchedAt = time.Now()
	}
	razorpayClients[mode] = c
}

// InitRazorpay probes both slots at startup to surface configuration problems
// early. Safe to call when secrets aren't set yet — it logs per slot and
// returns. Each slot is populated on first use regardless.
func InitRazorpay() {
	for _, mode := range []string{models.ChefModeLive, models.ChefModeTest} {
		if GetRazorpayFor(mode) != nil {
			log.Printf("razorpay[%s]: client initialized from Secret Manager", mode)
			continue
		}
		log.Printf("razorpay[%s]: no credentials yet — configure via Admin → Settings → Payment Gateway", mode)
	}
}

// --- Linked Accounts (Razorpay Route) ---

// LinkedAccountRequest represents the request to create a Razorpay Route linked account
type LinkedAccountRequest struct {
	Email        string                  `json:"email"`
	Phone        string                  `json:"phone"`
	LegalName    string                  `json:"legal_business_name"`
	BusinessType string                  `json:"business_type"` // individual, partnership, etc.
	ContactName  string                  `json:"contact_name"`
	LegalInfo    *LinkedAccountLegalInfo `json:"legal_info,omitempty"`
	Profile      *LinkedAccountProfile   `json:"profile,omitempty"`
}

type LinkedAccountLegalInfo struct {
	Pan string `json:"pan,omitempty"`
	Gst string `json:"gst,omitempty"`
}

type LinkedAccountProfile struct {
	Category    string                  `json:"category,omitempty"`
	SubCategory string                  `json:"subcategory,omitempty"`
	Addresses   *LinkedAccountAddresses `json:"addresses,omitempty"`
}

type LinkedAccountAddresses struct {
	Registered *LinkedAccountAddress `json:"registered,omitempty"`
}

type LinkedAccountAddress struct {
	Street1    string `json:"street1"`
	Street2    string `json:"street2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode int    `json:"postal_code"`
	Country    string `json:"country"`
}

// LinkedAccountResponse from Razorpay
type LinkedAccountResponse struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
}

// CreateLinkedAccount creates a Razorpay Route linked account for a chef or driver
func (c *RazorpayClient) CreateLinkedAccount(req *LinkedAccountRequest) (*LinkedAccountResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/accounts", body)
	if err != nil {
		return nil, err
	}

	var result LinkedAccountResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// --- Orders with Route Transfers ---

// TransferSpec defines a payment split for Razorpay Route
type TransferSpec struct {
	Account  string            `json:"account"` // Razorpay linked account ID
	Amount   int               `json:"amount"`  // Amount in paise (INR smallest unit)
	Currency string            `json:"currency"`
	Notes    map[string]string `json:"notes,omitempty"`
	OnHold   bool              `json:"on_hold,omitempty"` // Hold transfer until delivery confirmed
}

// OrderRequest creates a Razorpay order with optional Route transfers
type OrderRequest struct {
	Amount    int               `json:"amount"` // Total in paise
	Currency  string            `json:"currency"`
	Receipt   string            `json:"receipt"` // Order number
	Notes     map[string]string `json:"notes,omitempty"`
	Transfers []TransferSpec    `json:"transfers,omitempty"`
}

// OrderResponse from Razorpay
type OrderResponse struct {
	ID       string `json:"id"`
	Entity   string `json:"entity"`
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
	Receipt  string `json:"receipt"`
	Status   string `json:"status"`
}

// razorpayReceiptMax is Razorpay's hard limit on the `receipt` field. Exceeding
// it fails order creation outright — i.e. the customer cannot pay.
const razorpayReceiptMax = 40

// clampReceipt keeps a receipt inside Razorpay's limit.
//
// Reference numbers are kitchen-prefixed (ChefRef), and several call sites add
// their own prefix on top ("refund-", "TIP-"), so a long kitchen name can push a
// receipt past 40. This is the single place that guarantees it never reaches
// Razorpay — which is why ChefRefPrefix is free to favour readable prefixes over
// a cap tight enough to make the arithmetic work unaided.
//
// Truncation keeps the TAIL, because the unique part of every reference number
// lives at the end — trimming the front costs readability, trimming the back
// would let two receipts collide and break reconciliation.
func clampReceipt(s string) string {
	if len(s) <= razorpayReceiptMax {
		return s
	}
	return s[len(s)-razorpayReceiptMax:]
}

// CreateOrder creates a Razorpay order with Route transfer splits
func (c *RazorpayClient) CreateOrder(req *OrderRequest) (*OrderResponse, error) {
	req.Receipt = clampReceipt(req.Receipt)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/orders", body)
	if err != nil {
		return nil, err
	}

	var result OrderResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// --- Recurring Subscriptions (UPI Autopay e-mandate) — customer meal subs (#281) ---

// SubscriptionRequest creates a Razorpay recurring Subscription. PlanID references
// a pre-created Razorpay Plan (owner-configured per cadence/amount); the customer
// approves a UPI-Autopay e-mandate via the returned ShortURL.
type SubscriptionRequest struct {
	PlanID         string            `json:"plan_id"`
	TotalCount     int               `json:"total_count"` // number of billing cycles
	CustomerNotify int               `json:"customer_notify,omitempty"`
	Notes          map[string]string `json:"notes,omitempty"`
}

// SubscriptionResponse is the created subscription. ShortURL is the mandate
// approval link the customer is sent to.
type SubscriptionResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	ShortURL string `json:"short_url"`
	PlanID   string `json:"plan_id"`
}

// CreateSubscription creates a recurring Razorpay Subscription. The live UPI-Autopay
// mandate approval is owner-tested against the gateway.
func (c *RazorpayClient) CreateSubscription(req *SubscriptionRequest) (*SubscriptionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal subscription request: %w", err)
	}
	resp, err := c.doRequest("POST", "/subscriptions", body)
	if err != nil {
		return nil, err
	}
	var result SubscriptionResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse subscription response: %w", err)
	}
	return &result, nil
}

// CancelSubscription cancels a Razorpay Subscription (immediately or at cycle end).
func (c *RazorpayClient) CancelSubscription(subID string, atCycleEnd bool) error {
	cancelAt := 0
	if atCycleEnd {
		cancelAt = 1
	}
	body, _ := json.Marshal(map[string]any{"cancel_at_cycle_end": cancelAt})
	_, err := c.doRequest("POST", fmt.Sprintf("/subscriptions/%s/cancel", subID), body)
	return err
}

// --- Refunds ---

// RefundRequest creates a refund on a payment
type RefundRequest struct {
	Amount  int               `json:"amount"` // In paise; 0 = full refund
	Speed   string            `json:"speed"`  // "normal" or "optimum"
	Notes   map[string]string `json:"notes,omitempty"`
	Receipt string            `json:"receipt,omitempty"`
	// IdempotencyKey is a LOGICAL operation id (see gateway_idempotency.go builders).
	// When set, CreateRefund normalizes it into the X-Refund-Idempotency header so a
	// timeout-after-success retry is deduped by Razorpay (#574). json:"-" — never part
	// of the request body (the body must be byte-identical across retries for dedup).
	IdempotencyKey string `json:"-"`
}

// RefundResponse from Razorpay
type RefundResponse struct {
	ID        string `json:"id"`
	Entity    string `json:"entity"`
	Amount    int    `json:"amount"`
	Currency  string `json:"currency"`
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"` // processed, pending, failed
	Speed     string `json:"speed_processed"`
}

// CreateRefund issues a refund on a captured payment
// The refund goes back to the customer. Chef/driver transfers that already settled
// are reversed automatically by Razorpay Route.
func (c *RazorpayClient) CreateRefund(paymentID string, req *RefundRequest) (*RefundResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var headers map[string]string
	if req.IdempotencyKey != "" {
		headers = map[string]string{headerRefundIdempotency: normalizeIdempotencyKey(req.IdempotencyKey)}
	}
	resp, err := c.doRequestWithHeaders("POST", fmt.Sprintf("/payments/%s/refund", paymentID), body, headers)
	if err != nil {
		return nil, err
	}

	var result RefundResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// --- Direct transfers (platform balance → linked account) ---

// DirectTransferRequest moves money from the PLATFORM's Razorpay balance directly
// to a linked account, independent of any single payment. Unlike a payment-linked
// Route transfer (capped by the captured payment), this draws from the pooled
// platform balance — which is how a wallet-covered order tops up the chef/driver
// shortfall when the gateway capture alone can't fund the full split (#141).
type DirectTransferRequest struct {
	Account  string            `json:"account"`           // destination linked account (acc_...)
	Amount   int               `json:"amount"`            // paise
	Currency string            `json:"currency"`          // "INR"
	OnHold   bool              `json:"on_hold,omitempty"` // hold until delivery confirmation
	Notes    map[string]string `json:"notes,omitempty"`
	// IdempotencyKey is a LOGICAL operation id (see gateway_idempotency.go builders).
	// When set, CreateTransfer normalizes it into the X-Transfer-Idempotency header so a
	// timeout-after-success retry is deduped by Razorpay (#574). json:"-" — never part of
	// the request body (the body must be byte-identical across retries for dedup).
	IdempotencyKey string `json:"-"`
}

// TransferResponse from Razorpay's /transfers endpoint.
type TransferResponse struct {
	ID            string `json:"id"`
	Entity        string `json:"entity"`
	RecipientType string `json:"recipient_settlement_id"`
	Account       string `json:"recipient"`
	Amount        int    `json:"amount"`
	// AmountReversed is how much of this transfer Razorpay has already reversed —
	// (Amount − AmountReversed) is what remains un-reversed, so a repeated partial
	// claw-back (#549) never tries to reverse more than is left on the transfer.
	AmountReversed int    `json:"amount_reversed"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	OnHold         bool   `json:"on_hold"`
}

// CreateTransfer issues a direct transfer from the platform balance to a linked
// account. Used to settle the wallet-funded portion of an order's chef/driver
// split that the gateway capture could not cover (#141).
func (c *RazorpayClient) CreateTransfer(req *DirectTransferRequest) (*TransferResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var headers map[string]string
	if req.IdempotencyKey != "" {
		headers = map[string]string{headerTransferIdempotency: normalizeIdempotencyKey(req.IdempotencyKey)}
	}
	resp, err := c.doRequestWithHeaders("POST", "/transfers", body, headers)
	if err != nil {
		return nil, err
	}

	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// ReleaseTransfer clears the on-hold flag on a held Route transfer so Razorpay
// settles it to the linked account. This is the escrow "release on delivery"
// primitive the tiffin meal-plan flow needs (#194): chef payouts are created
// OnHold at confirmation and released per delivered day. Releasing an already
// released transfer is a no-op on Razorpay's side; callers should also DB-guard
// to avoid the round-trip.
func (c *RazorpayClient) ReleaseTransfer(transferID string) (*TransferResponse, error) {
	body, _ := json.Marshal(map[string]any{"on_hold": false})
	resp, err := c.doRequest("PATCH", fmt.Sprintf("/transfers/%s", transferID), body)
	if err != nil {
		return nil, err
	}
	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// FetchOrderTransfers lists the Route transfers attached to a Razorpay order —
// used to find an order's held chef/rider transfers to release on delivery (#217).
func (c *RazorpayClient) FetchOrderTransfers(orderID string) ([]TransferResponse, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/orders/%s/transfers", orderID), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []TransferResponse `json:"items"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return result.Items, nil
}

// FetchTransfer retrieves a single Route transfer by id — used by the escrow
// conservation reconcile (#398) to read the gateway state (amount, amount_reversed,
// on_hold, status) of a DIRECT transfer (meal-plan-day / group hold) that isn't
// attached to a Razorpay order and so can't be found via FetchOrderTransfers.
func (c *RazorpayClient) FetchTransfer(transferID string) (*TransferResponse, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/transfers/%s", transferID), nil)
	if err != nil {
		return nil, err
	}
	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// ReverseTransfer reverses a (settled or held) Route transfer, returning the
// funds to the platform balance — used to claw back a chef payout when a
// confirmed-but-not-yet-delivered day is later refunded (escrow refund path,
// #194). amountPaise of 0 reverses the full transfer.
func (c *RazorpayClient) ReverseTransfer(transferID string, amountPaise int) (*TransferResponse, error) {
	payload := map[string]any{}
	if amountPaise > 0 {
		payload["amount"] = amountPaise
	}
	body, _ := json.Marshal(payload)
	resp, err := c.doRequest("POST", fmt.Sprintf("/transfers/%s/reversals", transferID), body)
	if err != nil {
		return nil, err
	}
	var result TransferResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// --- Payment Fetch ---

// PaymentResponse from Razorpay
type PaymentResponse struct {
	ID             string `json:"id"`
	Entity         string `json:"entity"`
	Amount         int    `json:"amount"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	OrderID        string `json:"order_id"`
	Method         string `json:"method"` // card, upi, netbanking, wallet
	Email          string `json:"email"`
	Contact        string `json:"contact"`
	Captured       bool   `json:"captured"`
	AmountRefunded int    `json:"amount_refunded"`
}

// FetchPayment retrieves payment details
func (c *RazorpayClient) FetchPayment(paymentID string) (*PaymentResponse, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/payments/%s", paymentID), nil)
	if err != nil {
		return nil, err
	}

	var result PaymentResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// FetchOrderPayments returns all payments made against a Razorpay order (GET
// /orders/:id/payments). Used by the meal-plan advance reconcile to discover whether a
// plan whose gateway order id we hold was actually paid, when both the client verify
// call and the payment.captured webhook were lost.
func (c *RazorpayClient) FetchOrderPayments(orderID string) ([]PaymentResponse, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/orders/%s/payments", orderID), nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []PaymentResponse `json:"items"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse order payments: %w", err)
	}
	return result.Items, nil
}

// VerifyPaymentSignatureFor validates a Razorpay Checkout payment signature
// against ONE mode's key secret:
// HMAC-SHA256(order_id + "|" + payment_id, keySecret) must equal the signature
// the client received from Razorpay. This proves Razorpay authorized THIS
// (order, payment) pair for our merchant, so a captured payment from a
// different order can't be reused to settle this one. Constant-time compare.
//
// Always call this with the ORDER's mode, never the chef's current mode — the
// chef may have been flipped since the payment was taken.
func VerifyPaymentSignatureFor(mode, razorpayOrderID, razorpayPaymentID, signature string) bool {
	c := snapshotRazorpayClient(mode) // #395·5: read the client under the cache mutex
	if c == nil || c.keySecret == "" {
		log.Printf("Warning: Razorpay key secret not configured for mode %s", models.NormalizeMode(mode))
		return false
	}
	mac := hmac.New(sha256.New, []byte(c.keySecret))
	mac.Write([]byte(razorpayOrderID + "|" + razorpayPaymentID))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// VerifyPaymentSignature verifies against the live slot. Retained for the call
// sites that have no order in hand.
func VerifyPaymentSignature(razorpayOrderID, razorpayPaymentID, signature string) bool {
	return VerifyPaymentSignatureFor(models.ChefModeLive, razorpayOrderID, razorpayPaymentID, signature)
}

// GetKeyID returns the Razorpay publishable key ID (for frontend checkout).
// This is safe to expose — it's the equivalent of a Stripe publishable key.
func (c *RazorpayClient) GetKeyID() string {
	return c.keyID
}

// HasWebhookSecret returns whether a webhook secret is configured (without exposing it).
func (c *RazorpayClient) HasWebhookSecret() bool {
	return c.webhookSecret != ""
}

// HealthCheck validates Razorpay credentials by making a lightweight API call.
// We fetch one payment (count=1) — Razorpay rejects count=0 with a 400, so
// this is the smallest valid request that still proves auth + connectivity.
func (c *RazorpayClient) HealthCheck() error {
	_, err := c.doRequest("GET", "/payments?count=1", nil)
	return err
}

// --- Helpers ---

// ToPaise converts a float amount (e.g. 499.00 INR) to paise (49900). It ROUNDS
// (not truncates): int(amount*100) alone loses a paise on IEEE-754 near-integer
// products (0.29*100 == 28.9999… → 28), which under-charges/under-settles real
// Route money now that Round2-ed NET payouts feed straight in (#518). Rounds
// identically to ToMinor(amount, "INR") so both gateway paths mint the same
// minor-unit value (#524/#396).
//
// #396 Phase 2: delegates to money.FromRupees — the single paise-conversion the
// integer-paise migration standardises on (behaviour-identical to the prior
// math.Round). New code should prefer the money package's Paise type directly.
func ToPaise(amount float64) int {
	return int(money.FromRupees(amount))
}

// FromPaise converts paise to float amount.
func FromPaise(paise int) float64 {
	return money.Paise(paise).Rupees()
}

// ValidateCapturedPayment is the single hard gate every payment "verify" leg
// (order, catering deposit, featured-ad, tip, group share) MUST apply: a
// gateway-fetched payment has to be captured, belong to the EXPECTED Razorpay
// order, and cover the EXPECTED amount (paise). payment.OrderID / Amount /
// Status come from the gateway (FetchPayment), so the client can't forge them —
// without this binding, any captured payment on the merchant account (e.g. a ₹1
// charge, or a payment from an unrelated order) could be replayed to settle a
// different object for free. Returns (true, "") when valid, else (false, reason).
func ValidateCapturedPayment(paymentStatus, paymentOrderID, expectedOrderID string, paymentAmountPaise, expectedPaise int) (bool, string) {
	if paymentStatus != "captured" {
		return false, "Payment not captured"
	}
	if expectedOrderID == "" {
		// The verify leg was called without first creating the gateway order.
		return false, "Start the payment first"
	}
	if paymentOrderID != expectedOrderID {
		return false, "Payment does not belong to this order"
	}
	if paymentAmountPaise < expectedPaise {
		return false, "Payment amount does not match the expected amount"
	}
	return true, ""
}

// Razorpay's endpoint-specific idempotency header names (#574). Razorpay does NOT
// expose a single Idempotency-Key header — each money-moving endpoint has its own.
const (
	headerRefundIdempotency   = "X-Refund-Idempotency"
	headerTransferIdempotency = "X-Transfer-Idempotency"
)

// doRequest executes an authenticated HTTP request to the Razorpay API.
func (c *RazorpayClient) doRequest(method, path string, body []byte) ([]byte, error) {
	return c.doRequestWithHeaders(method, path, body, nil)
}

// doRequestWithHeaders is doRequest plus caller-supplied headers (an empty value is
// skipped) — used to attach the per-endpoint idempotency header (#574).
func (c *RazorpayClient) doRequestWithHeaders(method, path string, body []byte, extraHeaders map[string]string) ([]byte, error) {
	base := c.baseURL
	if base == "" {
		base = razorpayBaseURL
	}
	return c.doURL(method, base+path, body, extraHeaders)
}

// doURL is doRequestWithHeaders against an already-resolved absolute URL. Split
// out so the v2 onboarding endpoints (#740) — which sit on a different API
// version than razorpayBaseURL — reuse one auth and error-handling path.
func (c *RazorpayClient) doURL(method, url string, body []byte, extraHeaders map[string]string) ([]byte, error) {
	respBody, status, err := c.rawDo(method, url, body, extraHeaders)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("razorpay API error (HTTP %d): %s", status, string(respBody))
	}
	return respBody, nil
}

// rawDo performs the authenticated HTTP round-trip and returns the raw
// response body and status code, leaving error-message formatting to doURL.
func (c *RazorpayClient) rawDo(method, url string, body []byte, extraHeaders map[string]string) ([]byte, int, error) {
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.keyID, c.keySecret)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range extraHeaders {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("razorpay request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
