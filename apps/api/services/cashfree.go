package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services/money"
)

// cashfree.go — the Cashfree Payment Gateway adapter.
//
// Deliberately shaped as a near-mirror of razorpay.go: same per-mode credential
// slots, same cache + invalidate semantics, same "nil means not configured and
// callers already handle nil" contract, same injectable baseURL test seam. The
// symmetry is the point — the money paths switch on models.PaymentProvider and
// otherwise read identically, so a reviewer comparing the two files sees the
// gateway differences and nothing else.
//
// Four things genuinely differ from Razorpay, and each one is a place a careless
// port would lose money:
//
//  1. AMOUNTS ARE RUPEES ON THE WIRE. Razorpay speaks integer paise; Cashfree
//     speaks a decimal number of rupees ("order_amount": 140.25). Every amount
//     crossing this boundary goes through cashfreeAmount, which stores paise
//     internally and renders the decimal by integer arithmetic — never by
//     formatting a float64, which is how 140.25 becomes 140.25000000000001.
//
//  2. ENVIRONMENT IS THE HOSTNAME. Razorpay serves live and test from one host
//     and tells them apart by key prefix; Cashfree has sandbox.cashfree.com vs
//     api.cashfree.com. The host is therefore resolved from the CREDENTIALS
//     (see cashfreeBaseURLFor) rather than from the platform's live/test
//     partition, so a slot holding sandbox keys reaches the sandbox and works —
//     exactly as the equivalent Razorpay slot already does. The test slot is
//     pinned to sandbox regardless, so a sandbox order can never move real
//     money.
//
//  3. THERE IS NO CLIENT-SIDE PAYMENT SIGNATURE. Razorpay Checkout hands the
//     client an HMAC of order_id|payment_id that the server re-computes. The
//     Cashfree SDK returns no such thing; the authority is a server-side
//     GET /orders/{id} + GET /orders/{id}/payments. That is a stronger gate, not
//     a weaker one — it never trusts a client-supplied value at all — but it
//     means verification MUST fetch, and must not be written to "tolerate a
//     missing signature" the way the Razorpay path does.
//
//  4. REFUND IDEMPOTENCY IS FIRST-CLASS. Cashfree takes a merchant-supplied
//     refund_id (3–40 alphanumeric) and treats a repeat as the same refund,
//     instead of Razorpay's per-endpoint X-Refund-Idempotency header. We feed it
//     the same normalizeIdempotencyKey digest the Razorpay path uses, so one
//     logical refund has one identity on either gateway.

const (
	// Environment is selected by HOST, so these are not interchangeable with a
	// key swap — see note 2 above.
	cashfreeLiveBaseURL = "https://api.cashfree.com/pg"
	cashfreeTestBaseURL = "https://sandbox.cashfree.com/pg"

	// cashfreeAPIVersion pins the response schema. Cashfree versions its API by
	// date header and keeps old versions working, so this is a deliberate pin
	// rather than "latest": the payment/refund/webhook shapes this file parses
	// are the 2023-08-01 ones. Bumping it is a reviewed change, not a drift.
	cashfreeAPIVersion = "2023-08-01"

	// cashfreeCacheTTL mirrors razorpayCacheTTL. Admin writes call
	// InvalidateCashfreeFor so a key change is picked up immediately; the TTL
	// only covers rotations made outside the app (e.g. via gcloud).
	cashfreeCacheTTL = 5 * time.Minute

	cashfreeRequestTimeout = 30 * time.Second
)

// Cashfree credentials live in GCP Secret Manager, product-scoped ("homechef-")
// so they cannot collide with other products sharing tesseracthub-480811.
// Exported for the same reason the Razorpay ones are: the admin WRITE path and
// the client READ path must share one source of truth. A past drift wrote
// Razorpay keys to one name and read them from another, so admin-entered keys
// were silently never used — that bug is not worth reproducing here.
const (
	SecretCashfreeAppID         = "prod-homechef-cashfree-app-id"
	SecretCashfreeSecretKey     = "prod-homechef-cashfree-secret-key"
	SecretCashfreeWebhookSecret = "prod-homechef-cashfree-webhook-secret"

	// Test-mode slot, used by kitchens an admin has marked as test. Separate
	// secrets rather than a suffix on the values above, so clearing or rotating
	// one slot cannot disturb the other.
	SecretCashfreeTestAppID         = "prod-homechef-cashfree-test-app-id"
	SecretCashfreeTestSecretKey     = "prod-homechef-cashfree-test-secret-key"
	SecretCashfreeTestWebhookSecret = "prod-homechef-cashfree-test-webhook-secret"
)

// CashfreeClient handles all Cashfree PG interactions. Every sensitive field is
// unexported — secrets cannot be read from outside this package.
type CashfreeClient struct {
	appID         string
	secretKey     string
	webhookSecret string
	// mode is the credential slot this client was built for. It pins the test
	// slot to the sandbox host; for the live slot the host follows the
	// credentials themselves (cashfreeBaseURLFor).
	mode string
	// baseURL overrides the resolved host. Empty in every production path;
	// tests point it at an httptest.Server so the order/payment/refund seams can
	// be driven end-to-end without a live gateway.
	baseURL   string
	fetchedAt time.Time
}

// cashfreeClients caches one client per mode ("live", "test"), keyed rather than
// held in two variables so adding a slot stays a data change.
var (
	cashfreeClients = map[string]*CashfreeClient{}
	cashfreeMu      sync.Mutex
)

// CashfreeSecretNames is the exported form used by the admin write path, so the
// slot an admin saves into and the slot the app reads from cannot drift apart.
func CashfreeSecretNames(mode string) (appID, secretKey, webhookSecret string) {
	return cashfreeSecretNames(mode)
}

func cashfreeSecretNames(mode string) (appID, secretKey, webhookSecret string) {
	if models.IsTestMode(mode) {
		return SecretCashfreeTestAppID, SecretCashfreeTestSecretKey, SecretCashfreeTestWebhookSecret
	}
	return SecretCashfreeAppID, SecretCashfreeSecretKey, SecretCashfreeWebhookSecret
}

// cashfreeCredentialsAreSandbox reports whether a credential pair belongs to
// Cashfree's sandbox.
//
// Cashfree marks both halves: sandbox App IDs are prefixed "TEST", and sandbox
// secret keys carry "_test_" (cfsk_ma_test_…). Either signal is enough, and
// checking both means a rotation that changes one format does not silently flip
// an environment.
func cashfreeCredentialsAreSandbox(appID, secretKey string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(appID)), "TEST") ||
		strings.Contains(strings.ToLower(secretKey), "_test_")
}

// cashfreeBaseURLFor resolves the API host for a slot.
//
// The environment follows the CREDENTIALS, not the platform's live/test
// partition — which is how Razorpay already behaves, since its key prefix
// decides and its live slot therefore runs test keys perfectly happily.
// Binding the host to the slot instead made Cashfree the odd one out: sandbox
// credentials in the live slot produced a valid client that 401'd against
// api.cashfree.com, so every live checkout fell back to Razorpay and Cashfree
// could not be exercised at all before real live keys existed.
//
// The two directions are deliberately NOT symmetric:
//
//   - LIVE slot with sandbox credentials → sandbox. Harmless: no real money can
//     move, and it is the normal state while a merchant account is in review.
//   - TEST slot → sandbox, ALWAYS, whatever the credentials say. Honouring a
//     production credential there would let a sandbox order move real money,
//     which is the one outcome worth hard-coding against.
//
// So the failure direction is always "test money does not move", never "real
// money moves unexpectedly".
func cashfreeBaseURLFor(mode, appID, secretKey string) string {
	if models.IsTestMode(mode) {
		return cashfreeTestBaseURL
	}
	if cashfreeCredentialsAreSandbox(appID, secretKey) {
		return cashfreeTestBaseURL
	}
	return cashfreeLiveBaseURL
}

// fetchCashfreeFromSM pulls one mode's credentials from Secret Manager. The
// webhook secret is optional (webhooks simply won't verify without it); the app
// id and secret key are required.
func fetchCashfreeFromSM(ctx context.Context, mode string) (*CashfreeClient, error) {
	mode = models.NormalizeMode(mode)
	appIDName, secretName, webhookName := cashfreeSecretNames(mode)

	appID, idErr := GetPlatformSecret(ctx, appIDName)
	secretKey, secErr := GetPlatformSecret(ctx, secretName)
	webhookSecret, _ := GetPlatformSecret(ctx, webhookName)

	if idErr != nil || secErr != nil || isPlaceholderValue(appID) || isPlaceholderValue(secretKey) {
		// Dev fallback: env-provided credentials, so local work needs no GCP
		// access. Deliberately LIVE-ONLY, exactly as the Razorpay fallback is:
		// the CASHFREE_* env vars describe a single gateway, so honouring them
		// for the test slot would serve one environment's credentials to the
		// other — the precise mix-up per-mode slots exist to prevent. An
		// unconfigured test slot returns nil, which callers already handle.
		if !models.IsTestMode(mode) {
			if cfg := config.AppConfig; cfg != nil &&
				!isPlaceholderValue(cfg.CashfreeAppID) && !isPlaceholderValue(cfg.CashfreeSecretKey) {
				return &CashfreeClient{
					appID:         cfg.CashfreeAppID,
					secretKey:     cfg.CashfreeSecretKey,
					webhookSecret: cfg.CashfreeWebhookSecret,
					mode:          mode,
					fetchedAt:     time.Now(),
				}, nil
			}
		}
		if idErr != nil {
			return nil, fmt.Errorf("cashfree[%s] credentials not configured: %w", mode, idErr)
		}
		return nil, fmt.Errorf("cashfree[%s] credentials missing or still set to placeholder — configure them in Admin → Settings → Payment Gateway", mode)
	}

	return &CashfreeClient{
		appID:         appID,
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		mode:          mode,
		fetchedAt:     time.Now(),
	}, nil
}

// GetCashfreeFor returns the cached client for one mode, fetching that mode's
// credentials on a cache miss. Slots are cached and invalidated independently,
// so a test-mode kitchen and a live one can transact concurrently.
//
// Returns nil when the slot is not configured — callers must handle nil, which
// is the same contract GetRazorpayFor has.
func GetCashfreeFor(mode string) *CashfreeClient {
	mode = models.NormalizeMode(mode)

	cashfreeMu.Lock()
	defer cashfreeMu.Unlock()

	if c := cashfreeClients[mode]; c != nil && time.Since(c.fetchedAt) < cashfreeCacheTTL {
		return c
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fresh, err := fetchCashfreeFromSM(ctx, mode)
	if err != nil {
		// Keep serving a stale client through a transient SM outage rather than
		// going dark mid-checkout; only fetchedAt is out of date.
		if c := cashfreeClients[mode]; c != nil {
			log.Printf("cashfree[%s]: SM fetch failed, using cached credentials: %v", mode, err)
			return c
		}
		log.Printf("cashfree[%s]: not configured (%v)", mode, err)
		return nil
	}

	cashfreeClients[mode] = fresh
	return fresh
}

// GetCashfree is the live credential slot, for the non-chef-scoped call sites
// (admin gateway status, reconciliation).
func GetCashfree() *CashfreeClient { return GetCashfreeFor(models.ChefModeLive) }

// snapshotCashfreeClient returns one slot's client pointer read under cashfreeMu.
// A CashfreeClient's fields are immutable after construction (every path assigns
// a whole new client rather than mutating), so the caller may read the returned
// client's fields without further locking. Used by the webhook verifier so a
// verify cannot race a credential refresh or invalidation.
func snapshotCashfreeClient(mode string) *CashfreeClient {
	cashfreeMu.Lock()
	defer cashfreeMu.Unlock()
	return cashfreeClients[models.NormalizeMode(mode)]
}

// InvalidateCashfreeFor clears one slot so the next read re-fetches. Scoped per
// mode so saving test keys cannot knock a healthy live gateway offline.
func InvalidateCashfreeFor(mode string) {
	mode = models.NormalizeMode(mode)
	cashfreeMu.Lock()
	defer cashfreeMu.Unlock()
	delete(cashfreeClients, mode)
	log.Printf("cashfree[%s]: credential cache invalidated", mode)
}

// InvalidateCashfree clears every slot.
func InvalidateCashfree() {
	InvalidateCashfreeFor(models.ChefModeLive)
	InvalidateCashfreeFor(models.ChefModeTest)
}

// SetCashfreeClientFor installs a client into one slot directly. Test seam only.
func SetCashfreeClientFor(mode string, c *CashfreeClient) {
	mode = models.NormalizeMode(mode)
	cashfreeMu.Lock()
	defer cashfreeMu.Unlock()
	if c != nil && c.fetchedAt.IsZero() {
		c.fetchedAt = time.Now()
	}
	cashfreeClients[mode] = c
}

// InitCashfree probes both slots at startup to surface configuration gaps early.
// Safe to call before secrets exist — each slot is populated on first use anyway.
func InitCashfree() {
	for _, mode := range []string{models.ChefModeLive, models.ChefModeTest} {
		if GetCashfreeFor(mode) != nil {
			log.Printf("cashfree[%s]: client initialized from Secret Manager", mode)
			continue
		}
		log.Printf("cashfree[%s]: no credentials yet — configure via Admin → Settings → Payment Gateway", mode)
	}
}

// --- Amounts ---

// cashfreeAmount is an amount in PAISE that marshals to (and unmarshals from)
// Cashfree's rupees-with-two-decimals wire format.
//
// The whole reason this type exists rather than a plain float64 field: rendering
// money by formatting a float64 is how a ₹140.25 order becomes 140.25000000000001
// on the wire, and how repeated ÷100 ×100 round-trips shave a paise off a
// settlement. Paise stays the internal unit everywhere in this codebase
// (money.Paise, ToPaise/FromPaise), so the conversion happens exactly once, here,
// at the boundary — by integer arithmetic on the way out and a single rounding
// step on the way in.
type cashfreeAmount int

// MarshalJSON renders paise as a rupee decimal with exactly two places, using
// integer division so no float ever touches the value.
func (a cashfreeAmount) MarshalJSON() ([]byte, error) {
	paise := int(a)
	sign := ""
	if paise < 0 {
		// Never expected — a negative charge or refund is a caller bug — but
		// rendering "-1.-5" would be worse than rendering "-1.05", and silently
		// clamping to zero would hide the bug.
		sign, paise = "-", -paise
	}
	return []byte(fmt.Sprintf("%s%d.%02d", sign, paise/100, paise%100)), nil
}

// UnmarshalJSON accepts Cashfree's number (or a stringified number, which some
// fields use) and rounds to paise through money.FromRupees — the same rounding
// ToPaise uses, so a value read back from Cashfree compares equal to the value
// we sent.
func (a *cashfreeAmount) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		*a = 0
		return nil
	}
	rupees, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("cashfree: parse amount %q: %w", s, err)
	}
	*a = cashfreeAmount(money.FromRupees(rupees))
	return nil
}

// Paise returns the amount in paise, the unit every caller works in.
func (a cashfreeAmount) Paise() int { return int(a) }

// --- Orders ---

// CashfreeCustomerDetails identifies the payer. customer_id and customer_phone
// are mandatory at Cashfree; phone must be present and is what UPI intent keys
// off, so a blank one fails order creation rather than degrading.
type CashfreeCustomerDetails struct {
	CustomerID    string `json:"customer_id"`
	CustomerPhone string `json:"customer_phone"`
	CustomerName  string `json:"customer_name,omitempty"`
	CustomerEmail string `json:"customer_email,omitempty"`
}

// CashfreeOrderMeta carries the redirect + notify hooks. ReturnURL is where the
// hosted/WebView checkout lands the customer; NotifyURL is left empty because
// webhooks are configured once in the Cashfree dashboard rather than per order
// (a per-order URL would silently bypass the signature secret we verify with).
type CashfreeOrderMeta struct {
	ReturnURL string `json:"return_url,omitempty"`
	NotifyURL string `json:"notify_url,omitempty"`
	// PaymentMethods restricts the offered instruments (e.g. "upi,cc,dc,nb").
	// Empty means "everything enabled on the account", which is what we want.
	PaymentMethods string `json:"payment_methods,omitempty"`
}

// CashfreeOrderRequest creates an order. AmountPaise is the platform's unit; it
// renders as Cashfree's rupee decimal.
//
// There are deliberately NO order_splits here. Cashfree captures the full amount
// to the platform merchant account and chef/rider money settles through the
// statement/payout path — see models.ProviderSupportsGatewaySplit for the full
// reasoning. If Easy Split is adopted later, the split array is added here and
// that predicate flips; nothing else in the money paths should need to change.
type CashfreeOrderRequest struct {
	OrderID     string                  `json:"order_id,omitempty"`
	AmountPaise cashfreeAmount          `json:"order_amount"`
	Currency    string                  `json:"order_currency"`
	Customer    CashfreeCustomerDetails `json:"customer_details"`
	Meta        *CashfreeOrderMeta      `json:"order_meta,omitempty"`
	Tags        map[string]string       `json:"order_tags,omitempty"`
	ExpiryTime  string                  `json:"order_expiry_time,omitempty"`
	OrderNote   string                  `json:"order_note,omitempty"`
	// Splits is the Easy Split allocation at capture: each vendor's share is
	// settled by Cashfree directly; order_amount minus the splits stays with
	// the platform merchant account. Empty means no split — full capture.
	Splits         []CashfreeOrderSplit `json:"order_splits,omitempty"`
	IdempotencyKey string               `json:"-"` // → x-idempotency-key header
}

// CashfreeOrderSplit is one vendor's share of an order, in paise (marshalled
// as Cashfree's rupee-decimal wire format like every other amount).
type CashfreeOrderSplit struct {
	VendorID    string         `json:"vendor_id"`
	AmountPaise cashfreeAmount `json:"amount"`
}

// CashfreeOrderResponse is the created (or fetched) order. PaymentSessionID is
// the token the client SDK opens checkout with — it is the Cashfree analogue of
// Razorpay's (key_id + order_id) pair, and it is short-lived, which is why the
// create path re-uses an ACTIVE order rather than minting a second one.
type CashfreeOrderResponse struct {
	CFOrderID        json.Number       `json:"cf_order_id"`
	OrderID          string            `json:"order_id"`
	PaymentSessionID string            `json:"payment_session_id"`
	OrderStatus      string            `json:"order_status"`
	AmountPaise      cashfreeAmount    `json:"order_amount"`
	Currency         string            `json:"order_currency"`
	Tags             map[string]string `json:"order_tags,omitempty"`
	CreatedAt        string            `json:"created_at,omitempty"`
	ExpiryTime       string            `json:"order_expiry_time,omitempty"`
}

// Cashfree order_status values.
const (
	CashfreeOrderActive = "ACTIVE"
	CashfreeOrderPaid   = "PAID"
	// EXPIRED / TERMINATED / TERMINATION_REQUESTED are all dead: the session
	// cannot be paid, so the create path must mint a fresh order rather than
	// hand the client a token that will bounce.
	CashfreeOrderExpired    = "EXPIRED"
	CashfreeOrderTerminated = "TERMINATED"
)

// IsPayable reports whether checkout can still be opened against this order.
func (o *CashfreeOrderResponse) IsPayable() bool {
	return o != nil && o.OrderStatus == CashfreeOrderActive && o.PaymentSessionID != ""
}

// CreateOrder creates a Cashfree order.
//
// A duplicate order_id returns 409 from Cashfree. That is NOT an error here: it
// means a previous attempt already created this order (a timeout-after-success,
// or the customer tapping "Pay" twice), so the existing order is fetched and
// returned. Treating the 409 as a failure would strand a perfectly payable order
// and — worse — tempt the caller into minting a second order id for the same
// purchase, which is how a payment arrives for an order id nothing recognises.
func (c *CashfreeClient) CreateOrder(req *CashfreeOrderRequest) (*CashfreeOrderResponse, error) {
	if req.Currency == "" {
		req.Currency = "INR"
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("cashfree: marshal order request: %w", err)
	}

	var headers map[string]string
	if req.IdempotencyKey != "" {
		headers = map[string]string{cashfreeHeaderIdempotency: normalizeIdempotencyKey(req.IdempotencyKey)}
	}

	resp, status, err := c.do("POST", "/orders", body, headers)
	if err != nil {
		return nil, err
	}
	if status == http.StatusConflict && req.OrderID != "" {
		log.Printf("cashfree[%s]: order %s already exists — reusing it", c.mode, req.OrderID)
		return c.FetchOrder(req.OrderID)
	}
	if status >= 400 {
		return nil, cashfreeError(status, resp)
	}

	var result CashfreeOrderResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("cashfree: parse order response: %w", err)
	}
	return &result, nil
}

// FetchOrder reads an order, including a still-valid payment_session_id for an
// ACTIVE one. This is what makes a retry of an unpaid order reuse its existing
// Cashfree order instead of creating a parallel one.
func (c *CashfreeClient) FetchOrder(orderID string) (*CashfreeOrderResponse, error) {
	resp, status, err := c.do("GET", "/orders/"+orderID, nil, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, cashfreeError(status, resp)
	}
	var result CashfreeOrderResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("cashfree: parse fetch-order response: %w", err)
	}
	return &result, nil
}

// --- Payments ---

// CashfreePayment is one payment attempt against an order.
//
// PaymentMethod is deliberately json.RawMessage: Cashfree returns an OBJECT
// there ({"upi":{…}} / {"card":{…}}), not a string. PaymentGroup is the flat
// label ("upi", "credit_card", "net_banking", …) and is what MethodLabel maps
// into the platform's own short vocabulary — decoding the object into a typed
// struct would couple us to every instrument Cashfree adds.
type CashfreePayment struct {
	CFPaymentID    json.Number     `json:"cf_payment_id"`
	OrderID        string          `json:"order_id"`
	PaymentStatus  string          `json:"payment_status"`
	AmountPaise    cashfreeAmount  `json:"payment_amount"`
	Currency       string          `json:"payment_currency"`
	PaymentGroup   string          `json:"payment_group"`
	PaymentMethod  json.RawMessage `json:"payment_method,omitempty"`
	PaymentTime    string          `json:"payment_time,omitempty"`
	BankReference  string          `json:"bank_reference,omitempty"`
	PaymentMessage string          `json:"payment_message,omitempty"`
}

// Cashfree payment_status values.
const (
	CashfreePaymentSuccess     = "SUCCESS"
	CashfreePaymentFailed      = "FAILED"
	CashfreePaymentPending     = "PENDING"
	CashfreePaymentUserDropped = "USER_DROPPED"
)

// IsCaptured reports whether this payment actually took the money. Named to
// match the question the verify paths ask of Razorpay's `captured` — Cashfree
// has no separate authorize/capture step for the methods we accept, so SUCCESS
// is the captured state.
func (p *CashfreePayment) IsCaptured() bool {
	return p != nil && p.PaymentStatus == CashfreePaymentSuccess
}

// IsInFlight reports whether this attempt could still go on to take the money.
// PENDING is the state a card sits in while the bank's OTP/3DS page is open and
// the state a UPI collect sits in while the payer decides — it is NOT "no
// payment", and treating it as such is how an order gets cancelled out from
// under a charge that then succeeds.
//
// Only the three states that can never move money again are terminal. Anything
// Cashfree adds later reads as in-flight, because the cost of the two mistakes
// is not symmetric: waiting one more tick on a genuinely dead attempt is free,
// while cancelling a live one strands a real charge on a cancelled order.
func (p *CashfreePayment) IsInFlight() bool {
	if p == nil {
		return false
	}
	switch p.PaymentStatus {
	case CashfreePaymentSuccess, CashfreePaymentFailed, CashfreePaymentUserDropped:
		return false
	default:
		return true
	}
}

// MethodLabel maps Cashfree's payment_group onto the short method vocabulary the
// platform already stores in Order.PaymentMethod ("upi", "card", "netbanking",
// "wallet"), so a receipt or an admin screen reads identically no matter which
// gateway took the payment. An unrecognised group passes through as-is rather
// than being flattened to "other" — a new instrument should show up in the data,
// not disappear into a bucket.
func (p *CashfreePayment) MethodLabel() string {
	if p == nil {
		return ""
	}
	switch strings.ToLower(p.PaymentGroup) {
	case "upi":
		return "upi"
	case "credit_card", "debit_card", "card", "credit_card_emi", "debit_card_emi", "cardless_emi":
		return "card"
	case "net_banking", "netbanking":
		return "netbanking"
	case "wallet", "app":
		return "wallet"
	case "pay_later", "paylater":
		return "paylater"
	default:
		return strings.ToLower(p.PaymentGroup)
	}
}

// FetchOrderPayments returns every payment attempt on an order. This is the
// authority the verify path uses — there is no client signature to check, so the
// gateway's own record of what was paid is the only trustworthy input.
func (c *CashfreeClient) FetchOrderPayments(orderID string) ([]CashfreePayment, error) {
	resp, status, err := c.do("GET", "/orders/"+orderID+"/payments", nil, nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		// No payment attempts yet — an empty list, not a failure. The verify
		// path reads this as "not paid", which is exactly right.
		return nil, nil
	}
	if status >= 400 {
		return nil, cashfreeError(status, resp)
	}
	var result []CashfreePayment
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("cashfree: parse order payments: %w", err)
	}
	return result, nil
}

// SuccessfulPayment returns the captured payment on an order, or nil when there
// isn't one. Centralised because "which of these attempts is the real one" is a
// question every caller would otherwise answer slightly differently — and one
// that must never be answered by picking the first element.
func (c *CashfreeClient) SuccessfulPayment(orderID string) (*CashfreePayment, error) {
	payments, err := c.FetchOrderPayments(orderID)
	if err != nil {
		return nil, err
	}
	for i := range payments {
		if payments[i].IsCaptured() {
			return &payments[i], nil
		}
	}
	return nil, nil
}

// FetchPayment reads one payment by its Cashfree id. Cashfree scopes payments
// under their order, so unlike Razorpay's FetchPayment this needs both ids.
func (c *CashfreeClient) FetchPayment(orderID, cfPaymentID string) (*CashfreePayment, error) {
	resp, status, err := c.do("GET", "/orders/"+orderID+"/payments/"+cfPaymentID, nil, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, cashfreeError(status, resp)
	}
	var result CashfreePayment
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("cashfree: parse payment response: %w", err)
	}
	return &result, nil
}

// --- Refunds ---

// CashfreeRefundRequest issues a refund against an order.
//
// RefundID is REQUIRED by Cashfree and is the idempotency key: a repeat of the
// same refund_id is the same refund, not a second one. Callers pass the same
// LOGICAL operation id used for Razorpay (see gateway_idempotency.go builders)
// and CreateRefund normalizes it into Cashfree's 3–40 alphanumeric window.
type CashfreeRefundRequest struct {
	AmountPaise cashfreeAmount `json:"refund_amount"`
	RefundID    string         `json:"refund_id"`
	Note        string         `json:"refund_note,omitempty"`
	// Speed is STANDARD or INSTANT; empty means STANDARD. The platform uses
	// STANDARD to match Razorpay's "normal".
	Speed string `json:"refund_speed,omitempty"`
	// IdempotencyKey is the LOGICAL operation id. json:"-" — it becomes both the
	// refund_id and the x-idempotency-key header, never a body field of its own.
	IdempotencyKey string `json:"-"`
}

// CashfreeRefund is the created (or fetched) refund.
type CashfreeRefund struct {
	CFRefundID   json.Number    `json:"cf_refund_id"`
	CFPaymentID  json.Number    `json:"cf_payment_id"`
	RefundID     string         `json:"refund_id"`
	OrderID      string         `json:"order_id"`
	RefundStatus string         `json:"refund_status"`
	AmountPaise  cashfreeAmount `json:"refund_amount"`
	// RefundChargePaise is Cashfree's own reported processing charge for this refund, in
	// paise via the same cashfreeAmount rupee-decimal wire encoding as every other money
	// field on this struct — observability only per #885 decision 2 (the levy itself stays
	// a configured flat rate of the refunded amount, never this actual-fee field); useful
	// later for comparing the configured rate against real Cashfree charges. Reuses
	// cashfreeAmount's existing UnmarshalJSON, which already tolerates a missing/empty
	// field as zero, so no special-casing is needed for orders where Cashfree reports no
	// charge.
	RefundChargePaise cashfreeAmount `json:"refund_charge"`
	Currency          string         `json:"refund_currency,omitempty"`
	RefundType        string         `json:"refund_type,omitempty"`
	RefundNote        string         `json:"refund_note,omitempty"`
	ProcessedAt       string         `json:"processed_at,omitempty"`
}

// Cashfree refund_status values.
const (
	CashfreeRefundSuccess   = "SUCCESS"
	CashfreeRefundPending   = "PENDING"
	CashfreeRefundOnHold    = "ONHOLD"
	CashfreeRefundCancelled = "CANCELLED"
	CashfreeRefundFailed    = "FAILED"
)

// PlatformRefundStatus maps Cashfree's refund_status onto the status vocabulary
// the platform already persists for Razorpay refunds ("processed" / "pending" /
// "failed"), so order.refund status reads the same regardless of gateway.
//
// ONHOLD maps to pending, not failed: the money is still coming, it is just held
// for review at Cashfree. Calling it failed would make the reconcile cron
// re-issue a refund that is already in flight.
func PlatformRefundStatus(cashfreeStatus string) string {
	switch strings.ToUpper(cashfreeStatus) {
	case CashfreeRefundSuccess:
		return "processed"
	case CashfreeRefundPending, CashfreeRefundOnHold:
		return "pending"
	case CashfreeRefundFailed, CashfreeRefundCancelled:
		return "failed"
	default:
		return strings.ToLower(cashfreeStatus)
	}
}

// cashfreeRefundID normalizes a logical operation id into Cashfree's refund_id
// window: 3–40 characters, alphanumeric. normalizeIdempotencyKey already yields
// 32 lowercase hex characters, which satisfies both bounds and the charset — and
// crucially is the SAME digest the Razorpay path sends in its
// X-Refund-Idempotency header, so one logical refund carries one identity across
// both gateways.
func cashfreeRefundID(logical string) string {
	return normalizeIdempotencyKey(logical)
}

// CreateRefund issues a refund on a paid order.
//
// A 409 (duplicate refund_id) means this exact refund was already accepted — a
// timeout-after-success retry. The existing refund is fetched and returned, so
// the caller sees success and does not re-reserve or double-credit. This is the
// behaviour Razorpay gets from its idempotency header; Cashfree expresses it as
// a conflict, and mishandling it is how a customer receives two refunds.
func (c *CashfreeClient) CreateRefund(orderID string, req *CashfreeRefundRequest) (*CashfreeRefund, error) {
	if orderID == "" {
		return nil, errors.New("cashfree: refund needs an order id")
	}
	if req.RefundID == "" {
		if req.IdempotencyKey == "" {
			// Refusing here is deliberate. Letting Cashfree mint the id would
			// remove the dedup that stops a retry becoming a second real refund.
			return nil, errors.New("cashfree: refund needs an idempotency key to derive refund_id from")
		}
		req.RefundID = cashfreeRefundID(req.IdempotencyKey)
	}
	if req.Speed == "" {
		req.Speed = "STANDARD"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("cashfree: marshal refund request: %w", err)
	}

	var headers map[string]string
	if req.IdempotencyKey != "" {
		headers = map[string]string{cashfreeHeaderIdempotency: normalizeIdempotencyKey(req.IdempotencyKey)}
	}

	resp, status, err := c.do("POST", "/orders/"+orderID+"/refunds", body, headers)
	if err != nil {
		return nil, err
	}
	if status == http.StatusConflict || status == http.StatusUnprocessableEntity {
		// 409 = duplicate refund_id; 422 = idempotency replay. Either way the
		// refund exists — read it back rather than reporting a failure that
		// would send the caller round again.
		if existing, ferr := c.FetchRefund(orderID, req.RefundID); ferr == nil {
			log.Printf("cashfree[%s]: refund %s on order %s already existed — reusing it", c.mode, req.RefundID, orderID)
			return existing, nil
		}
		return nil, cashfreeError(status, resp)
	}
	if status >= 400 {
		return nil, cashfreeError(status, resp)
	}

	var result CashfreeRefund
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("cashfree: parse refund response: %w", err)
	}
	return &result, nil
}

// FetchOrderRefunds lists every refund on an order.
func (c *CashfreeClient) FetchOrderRefunds(orderID string) ([]CashfreeRefund, error) {
	resp, status, err := c.do("GET", "/orders/"+orderID+"/refunds", nil, nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil // no refunds yet — an empty list, not a failure
	}
	if status >= 400 {
		return nil, cashfreeError(status, resp)
	}
	var result []CashfreeRefund
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("cashfree: parse order refunds: %w", err)
	}
	return result, nil
}

// OrderRefundedPaise totals the SETTLED refunds on an order — the Cashfree
// equivalent of Razorpay's payment.amount_refunded, which the reconciliation
// cron compares against the platform's own cumulative refunded figure.
//
// Only SUCCESS counts. Including PENDING or ONHOLD would overstate what has
// actually gone back to the customer and make the reconciler flag a full refund
// that has not landed yet; including FAILED or CANCELLED would count money that
// never moved at all. This is the same "settled only" basis amount_refunded uses.
func (c *CashfreeClient) OrderRefundedPaise(orderID string) (int, error) {
	refunds, err := c.FetchOrderRefunds(orderID)
	if err != nil {
		return 0, err
	}
	total := 0
	for i := range refunds {
		if strings.ToUpper(refunds[i].RefundStatus) == CashfreeRefundSuccess {
			total += refunds[i].AmountPaise.Paise()
		}
	}
	return total, nil
}

// FetchRefund reads one refund by the merchant-supplied refund_id.
func (c *CashfreeClient) FetchRefund(orderID, refundID string) (*CashfreeRefund, error) {
	resp, status, err := c.do("GET", "/orders/"+orderID+"/refunds/"+refundID, nil, nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, cashfreeError(status, resp)
	}
	var result CashfreeRefund
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("cashfree: parse fetch-refund response: %w", err)
	}
	return &result, nil
}

// --- Webhook verification ---

// Cashfree webhook headers.
const (
	CashfreeWebhookSignatureHeader = "x-webhook-signature"
	CashfreeWebhookTimestampHeader = "x-webhook-timestamp"
)

// cashfreeWebhookMaxSkew bounds how old a webhook's timestamp may be.
//
// The signature covers timestamp+body, so an attacker who captures one delivery
// can replay it verbatim forever and it will keep verifying. Razorpay's path
// relies on the processed_events claim for that; this adds a time bound on top,
// because the timestamp is right there in the signed payload and not checking it
// throws away the one anti-replay signal the scheme hands us. Generous enough to
// absorb Cashfree's own retry schedule and any clock drift.
const cashfreeWebhookMaxSkew = 30 * time.Minute

// VerifyCashfreeWebhookMode validates a webhook against BOTH credential slots
// and reports which one signed it.
//
// Cashfree signs base64(HMAC-SHA256(timestamp + rawBody, secretKey)) — note that
// the RAW body must be used, not a re-marshalled parse, or the digest will not
// match. Live is tried first.
//
// Like the Razorpay equivalent, the returned mode is a HINT, not an authority:
// if both slots ever hold the same key every event resolves to live. The caller
// MUST additionally compare it against the target record's own mode and drop a
// mismatch — that comparison, not this function, is what keeps the live and test
// worlds apart. Both attempts use constant-time comparison.
func VerifyCashfreeWebhookMode(timestamp string, payload []byte, signature string) (bool, string) {
	if signature == "" || timestamp == "" {
		return false, ""
	}
	if !cashfreeTimestampFresh(timestamp) {
		log.Printf("cashfree webhook: rejecting stale timestamp %q", timestamp)
		return false, ""
	}

	configured := false
	for _, mode := range []string{models.ChefModeLive, models.ChefModeTest} {
		c := snapshotCashfreeClient(mode)
		if c == nil || c.webhookSecret == "" {
			continue
		}
		configured = true
		mac := hmac.New(sha256.New, []byte(c.webhookSecret))
		mac.Write([]byte(timestamp))
		mac.Write(payload)
		expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		if hmac.Equal([]byte(expected), []byte(signature)) {
			return true, mode
		}
	}
	if !configured {
		log.Println("Warning: Cashfree webhook secret not configured for any mode")
	}
	return false, ""
}

// cashfreeTimestampFresh bounds the signed timestamp. Cashfree sends epoch
// seconds; an unparseable value fails closed rather than being waved through,
// since a caller cannot distinguish "no replay protection" from "verified".
func cashfreeTimestampFresh(ts string) bool {
	secs, err := strconv.ParseInt(strings.TrimSpace(ts), 10, 64)
	if err != nil {
		return false
	}
	age := time.Since(time.Unix(secs, 0))
	if age < 0 {
		age = -age // tolerate a slightly fast sender clock
	}
	return age <= cashfreeWebhookMaxSkew
}

// --- Introspection (admin surface) ---

// GetAppID returns the Cashfree app id (the public client identifier, the rough
// analogue of a Razorpay key id). Safe to expose to a frontend.
func (c *CashfreeClient) GetAppID() string { return c.appID }

// Mode reports which credential slot this client is.
func (c *CashfreeClient) Mode() string { return c.mode }

// IsSandbox reports whether this client talks to the Cashfree sandbox. Derived
// from the resolved host rather than the mode string, so an injected test
// baseURL is reported honestly.
func (c *CashfreeClient) IsSandbox() bool {
	return strings.Contains(c.resolvedBaseURL(), "sandbox.cashfree.com")
}

// HasWebhookSecret reports whether a webhook secret is configured, without
// exposing it.
func (c *CashfreeClient) HasWebhookSecret() bool { return c.webhookSecret != "" }

// HealthCheck validates credentials with one cheap authenticated call.
//
// Cashfree has no "list payments" endpoint to probe, so this fetches a sentinel
// order id that will never exist. A 404 is the SUCCESS case: the request was
// authenticated and routed, there simply is no such order. Only an auth failure
// (401/403) or a transport error means the credentials are wrong — which is
// precisely what the admin needs to know.
func (c *CashfreeClient) HealthCheck() error {
	const sentinel = "homechef-healthcheck-nonexistent"
	resp, status, err := c.do("GET", "/orders/"+sentinel, nil, nil)
	if err != nil {
		return err
	}
	switch {
	case status == http.StatusNotFound, status == http.StatusOK:
		return nil
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return fmt.Errorf("cashfree credentials rejected: %w", cashfreeError(status, resp))
	default:
		return cashfreeError(status, resp)
	}
}

// --- HTTP ---

const cashfreeHeaderIdempotency = "x-idempotency-key"

func (c *CashfreeClient) resolvedBaseURL() string {
	if c.baseURL != "" {
		return c.baseURL
	}
	return cashfreeBaseURLFor(c.mode, c.appID, c.secretKey)
}

// do performs one authenticated round-trip and returns the body and status.
// Status is returned rather than folded into an error because several callers
// treat specific non-2xx codes as success (409 on create, 404 on health check
// and on an order with no payments yet) — collapsing them into an error string
// and re-parsing it would be fragile.
func (c *CashfreeClient) do(method, path string, body []byte, extraHeaders map[string]string) ([]byte, int, error) {
	url := c.resolvedBaseURL() + path

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("cashfree: build request: %w", err)
	}

	// Cashfree authenticates with a header pair, not HTTP Basic.
	req.Header.Set("x-client-id", c.appID)
	req.Header.Set("x-client-secret", c.secretKey)
	req.Header.Set("x-api-version", cashfreeAPIVersion)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range extraHeaders {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: cashfreeRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("cashfree request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("cashfree: read response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}

// cashfreeError builds an error from Cashfree's structured error envelope
// ({"message":…,"code":…,"type":…}), falling back to the status alone.
//
// Only the structured fields are surfaced, never the raw body — the same
// discipline doURLSanitized applies on the Razorpay side. Cashfree echoes
// submitted values back in some validation errors, and a rejected customer phone
// or bank detail must not ride an error message into a log line or a Sentry
// event.
func cashfreeError(status int, body []byte) error {
	var parsed struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && (parsed.Code != "" || parsed.Message != "") {
		return fmt.Errorf("cashfree API error (HTTP %d): %s: %s", status, parsed.Code, parsed.Message)
	}
	return fmt.Errorf("cashfree API error (HTTP %d)", status)
}
