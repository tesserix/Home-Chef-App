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
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

const stripeBaseURL = "https://api.stripe.com/v1"

// stripeCacheTTL bounds a cached client — admin key updates call
// InvalidateStripe() so the next read picks up new keys immediately.
const stripeCacheTTL = 5 * time.Minute

// StripeSecretNames is shared by runtime readers and admin writers.
func StripeSecretNames(mode string) (secretKey, publishableKey, webhookSecret, keyID string) {
	prefix := "prod-homechef-stripe-"
	if models.IsTestMode(mode) {
		prefix += "test-"
	}
	return prefix + "secret-key", prefix + "publishable-key", prefix + "webhook-secret", prefix + "key-id"
}

// StripeClient handles all Stripe REST API interactions. Same shape as the
// Cashfree client so the admin UI and payment handlers can treat them
// symmetrically.
type StripeClient struct {
	secretKey      string
	keyID          string
	publishableKey string
	webhookSecret  string
	fetchedAt      time.Time
	httpClient     *http.Client
}

var (
	stripeClients = map[string]*StripeClient{}
	stripeMu      sync.Mutex
)

func fetchStripeFor(ctx context.Context, mode string) (*StripeClient, error) {
	skName, pkName, whName, idName := StripeSecretNames(mode)
	sk, skErr := GetPlatformSecret(ctx, skName)
	pk, _ := GetPlatformSecret(ctx, pkName)
	wh, _ := GetPlatformSecret(ctx, whName)
	id, _ := GetPlatformSecret(ctx, idName)
	if skErr != nil || isPlaceholderValue(sk) {
		cfg := config.AppConfig
		if cfg == nil {
			return nil, fmt.Errorf("Stripe slot is not configured")
		}
		if models.IsTestMode(mode) {
			sk, pk, wh, id = cfg.StripeTestSecretKey, cfg.StripeTestPublishableKey, cfg.StripeTestWebhookSecret, cfg.StripeTestKeyID
		} else {
			sk, pk, wh, id = cfg.StripeSecretKey, cfg.StripePublishableKey, cfg.StripeWebhookSecret, cfg.StripeKeyID
		}
	}
	client := &StripeClient{secretKey: sk, publishableKey: pk, webhookSecret: wh, keyID: id, fetchedAt: time.Now()}
	if !client.IsTestMode() && !strings.HasPrefix(sk, "sk_live_") && !strings.HasPrefix(sk, "rk_live_") {
		return nil, fmt.Errorf("Stripe slot has no valid API credential")
	}
	if models.IsTestMode(mode) && !client.IsTestMode() {
		return nil, fmt.Errorf("Stripe test slot cannot use live credentials")
	}
	if pk != "" && ((client.IsTestMode() && !strings.HasPrefix(pk, "pk_test_")) || (!client.IsTestMode() && !strings.HasPrefix(pk, "pk_live_"))) {
		return nil, fmt.Errorf("Stripe publishable key environment mismatch")
	}
	return client, nil
}

func GetStripe() *StripeClient { return GetStripeFor(models.ChefModeLive) }

func GetStripeFor(mode string) *StripeClient {
	mode = models.NormalizeMode(mode)
	stripeMu.Lock()
	defer stripeMu.Unlock()
	cached := stripeClients[mode]
	if cached != nil && time.Since(cached.fetchedAt) < stripeCacheTTL {
		return cached
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fresh, err := fetchStripeFor(ctx, mode)
	if err != nil {
		log.Printf("Stripe %s credential refresh unavailable", mode)
		return nil
	}
	stripeClients[mode] = fresh
	return fresh
}

func InvalidateStripeFor(mode string) {
	stripeMu.Lock()
	defer stripeMu.Unlock()
	delete(stripeClients, models.NormalizeMode(mode))
}

func InvalidateStripe() {
	stripeMu.Lock()
	defer stripeMu.Unlock()
	stripeClients = map[string]*StripeClient{}
}

func InitStripe() {
	for _, mode := range []string{models.ChefModeLive, models.ChefModeTest} {
		if GetStripeFor(mode) != nil {
			log.Printf("Stripe %s slot initialized", mode)
		}
	}
}

// --- Connect Accounts (Stripe Connect) ---

// StripeConnectAccountRequest creates a Connect account for a chef/driver.
// Uses "express" accounts by default — Stripe hosts the onboarding flow
// (KYC, bank details) so we don't have to build country-specific forms.
type StripeConnectAccountRequest struct {
	Type         string // express, standard, custom
	Country      string // ISO-3166 alpha-2 (e.g. "US", "GB")
	Email        string
	BusinessType string // individual, company
}

type StripeConnectAccountResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Country string `json:"country"`
	Email   string `json:"email"`
	Type    string `json:"type"`
	Charges bool   `json:"charges_enabled"`
	Payouts bool   `json:"payouts_enabled"`
	Details bool   `json:"details_submitted"`
}

// CreateConnectAccount creates a Stripe Connect account. Returns the account
// ID that must be stored on the chef/driver profile and used as the
// transfer destination when capturing payments.
func (c *StripeClient) CreateConnectAccount(req *StripeConnectAccountRequest) (*StripeConnectAccountResponse, error) {
	accType := req.Type
	if accType == "" {
		accType = "express"
	}
	businessType := req.BusinessType
	if businessType == "" {
		businessType = "individual"
	}

	form := url.Values{}
	form.Set("type", accType)
	if req.Country != "" {
		form.Set("country", req.Country)
	}
	if req.Email != "" {
		form.Set("email", req.Email)
	}
	form.Set("business_type", businessType)
	form.Set("capabilities[transfers][requested]", "true")
	form.Set("capabilities[card_payments][requested]", "true")

	resp, err := c.doFormRequest("POST", "/accounts", form)
	if err != nil {
		return nil, err
	}

	var result StripeConnectAccountResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse connect account response: %w", err)
	}
	return &result, nil
}

// StripeAccountLink returns a one-time onboarding URL the chef/driver opens
// to complete KYC and bank details. Links expire in a few minutes, so
// create fresh each time the vendor portal opens the flow.
type StripeAccountLink struct {
	URL       string `json:"url"`
	ExpiresAt int64  `json:"expires_at"`
	Created   int64  `json:"created"`
}

func (c *StripeClient) CreateAccountLink(accountID, refreshURL, returnURL string) (*StripeAccountLink, error) {
	form := url.Values{}
	form.Set("account", accountID)
	form.Set("refresh_url", refreshURL)
	form.Set("return_url", returnURL)
	form.Set("type", "account_onboarding")

	resp, err := c.doFormRequest("POST", "/account_links", form)
	if err != nil {
		return nil, err
	}

	var result StripeAccountLink
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse account link response: %w", err)
	}
	return &result, nil
}

// FetchConnectAccount returns current capability status — useful to show
// "onboarding complete" vs "action required" in the vendor portal.
func (c *StripeClient) FetchConnectAccount(accountID string) (*StripeConnectAccountResponse, error) {
	resp, err := c.doFormRequest("GET", "/accounts/"+accountID, nil)
	if err != nil {
		return nil, err
	}
	var result StripeConnectAccountResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse connect account response: %w", err)
	}
	return &result, nil
}

// --- Payment Intents ---

// StripePaymentIntentRequest mirrors what CreateOrderPayment needs: amount
// in the smallest currency unit (cents for USD, etc.), currency, the chef's
// connected account ID (so funds settle directly), and an application fee
// for the platform cut.
type StripePaymentIntentRequest struct {
	IdempotencyKey      string
	Amount              int
	Currency            string
	ReceiptEmail        string
	Customer            string
	Metadata            map[string]string
	DestinationAccount  string // acct_... — funds transfer to this account
	ApplicationFeeCents int    // platform's cut in cents
	Description         string
}

type StripePaymentIntent struct {
	OnBehalfOf     string `json:"on_behalf_of"`
	Livemode       bool   `json:"livemode"`
	ID             string `json:"id"`
	Object         string `json:"object"`
	Amount         int    `json:"amount"`
	AmountReceived int    `json:"amount_received"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	ClientSecret   string `json:"client_secret"`
	Customer       string `json:"customer"`
	Description    string `json:"description"`
}

func (c *StripeClient) CreatePaymentIntent(ctx context.Context, req *StripePaymentIntentRequest) (*StripePaymentIntent, error) {
	form := url.Values{}
	form.Set("amount", strconv.Itoa(req.Amount))
	form.Set("currency", strings.ToLower(req.Currency))
	form.Set("automatic_payment_methods[enabled]", "true")
	if req.ReceiptEmail != "" {
		form.Set("receipt_email", req.ReceiptEmail)
	}
	if req.Customer != "" {
		form.Set("customer", req.Customer)
	}
	if req.Description != "" {
		form.Set("description", req.Description)
	}
	if req.DestinationAccount != "" {
		form.Set("transfer_data[destination]", req.DestinationAccount)
		form.Set("on_behalf_of", req.DestinationAccount)
	}
	if req.ApplicationFeeCents > 0 {
		form.Set("application_fee_amount", strconv.Itoa(req.ApplicationFeeCents))
	}
	for k, v := range req.Metadata {
		form.Set("metadata["+k+"]", v)
	}

	resp, err := c.doFormRequestContext(ctx, "POST", "/payment_intents", form, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	var result StripePaymentIntent
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent: %w", err)
	}
	return &result, nil
}

func (c *StripeClient) FetchPaymentIntent(ctx context.Context, id string) (*StripePaymentIntent, error) {
	resp, err := c.doFormRequestContext(ctx, "GET", "/payment_intents/"+url.PathEscape(id), nil, "")
	if err != nil {
		return nil, err
	}
	var result StripePaymentIntent
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent: %w", err)
	}
	return &result, nil
}

// SetPaymentIntentSettlement repairs an unconfirmed destination charge in place.
func (c *StripeClient) SetPaymentIntentSettlement(ctx context.Context, id, account string) (*StripePaymentIntent, error) {
	if id == "" || account == "" {
		return nil, fmt.Errorf("payment intent and settlement account are required")
	}
	form := url.Values{"on_behalf_of": {account}}
	resp, err := c.doFormRequestContext(ctx, "POST", "/payment_intents/"+url.PathEscape(id), form, "fe3dr-settlement-"+id+"-"+account)
	if err != nil {
		return nil, err
	}
	var result StripePaymentIntent
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse payment intent: %w", err)
	}
	return &result, nil
}

// --- Transfers (for driver payouts once delivery is confirmed) ---

// StripeTransferRequest moves money from the platform's Stripe balance
// into a connected account. Used for driver payouts because we can't
// split a single PaymentIntent across two destinations — the chef gets
// the PaymentIntent's transfer_data target, and we settle the driver's
// portion with a follow-up Transfer when delivery is confirmed.
type StripeTransferRequest struct {
	Amount        int
	Currency      string
	Destination   string // driver's acct_... Connect ID
	TransferGroup string // order number or id so related txns group in dashboard
	Description   string
	Metadata      map[string]string
}

type StripeTransfer struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Amount      int    `json:"amount"`
	Currency    string `json:"currency"`
	Destination string `json:"destination"`
	Created     int64  `json:"created"`
}

func (c *StripeClient) CreateTransfer(req *StripeTransferRequest) (*StripeTransfer, error) {
	form := url.Values{}
	form.Set("amount", strconv.Itoa(req.Amount))
	form.Set("currency", strings.ToLower(req.Currency))
	form.Set("destination", req.Destination)
	if req.TransferGroup != "" {
		form.Set("transfer_group", req.TransferGroup)
	}
	if req.Description != "" {
		form.Set("description", req.Description)
	}
	for k, v := range req.Metadata {
		form.Set("metadata["+k+"]", v)
	}

	resp, err := c.doFormRequest("POST", "/transfers", form)
	if err != nil {
		return nil, err
	}
	var result StripeTransfer
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse transfer: %w", err)
	}
	return &result, nil
}

// --- Refunds ---

type StripeRefundRequest struct {
	IdempotencyKey       string
	PaymentIntent        string
	Amount               int // In cents; 0 = full refund
	Reason               string
	ReverseTransfer      bool
	RefundApplicationFee bool
	Metadata             map[string]string
}

type StripeRefund struct {
	ID            string `json:"id"`
	Object        string `json:"object"`
	Amount        int    `json:"amount"`
	Currency      string `json:"currency"`
	PaymentIntent string `json:"payment_intent"`
	Status        string `json:"status"`
	Reason        string `json:"reason"`
}

func (c *StripeClient) CreateRefund(req *StripeRefundRequest) (*StripeRefund, error) {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, fmt.Errorf("stripe refund idempotency key is required")
	}
	form := url.Values{}
	form.Set("payment_intent", req.PaymentIntent)
	if req.Amount > 0 {
		form.Set("amount", strconv.Itoa(req.Amount))
	}
	if req.Reason != "" {
		form.Set("reason", req.Reason)
	}
	if req.ReverseTransfer {
		form.Set("reverse_transfer", "true")
	}
	if req.RefundApplicationFee {
		form.Set("refund_application_fee", "true")
	}
	for k, v := range req.Metadata {
		form.Set("metadata["+k+"]", v)
	}

	resp, err := c.doFormRequestContext(context.Background(), "POST", "/refunds", form, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	var result StripeRefund
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse refund: %w", err)
	}
	return &result, nil
}

// --- Webhook verification ---

// stripeWebhookTolerance bounds how far a webhook's timestamp may drift from
// our wall clock. Stripe recommends 5 minutes; matches their official SDK.
const stripeWebhookTolerance = 5 * time.Minute

// VerifyStripeWebhookSignature validates the Stripe-Signature header against
// the configured webhook secret. Stripe's format is
// `t=<timestamp>,v1=<hex-sha256>` — we HMAC `<timestamp>.<payload>` with
// the secret and compare in constant time. Also rejects events older than
// stripeWebhookTolerance to prevent replay.
func VerifyStripeWebhookSignature(payload []byte, sigHeader string) bool {
	return VerifyStripeWebhookSignatureFor(models.ChefModeLive, payload, sigHeader)
}

func VerifyStripeWebhookSignatureFor(mode string, payload []byte, sigHeader string) bool {
	client := GetStripeFor(mode)
	if client == nil {
		return false
	}
	return client.verifyWebhookSignature(payload, sigHeader)
}

func (c *StripeClient) verifyWebhookSignature(payload []byte, sigHeader string) bool {
	if c.webhookSecret == "" {
		log.Println("Warning: Stripe webhook secret not configured")
		return false
	}

	var ts string
	var signatures []string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			ts = kv[1]
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}
	if ts == "" || len(signatures) == 0 {
		return false
	}

	// Replay protection — reject events whose timestamp is too far from now.
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}
	eventTime := time.Unix(tsInt, 0)
	if drift := time.Since(eventTime); drift > stripeWebhookTolerance || drift < -stripeWebhookTolerance {
		log.Printf("Stripe webhook rejected: timestamp drift %v exceeds tolerance", drift)
		return false
	}

	signedPayload := ts + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, s := range signatures {
		if hmac.Equal([]byte(expected), []byte(s)) {
			return true
		}
	}
	return false
}

// --- Misc helpers ---

// GetPublishableKey returns the pk_* key safe to expose to the frontend.
func (c *StripeClient) GetPublishableKey() string {
	return c.publishableKey
}

// IsTestMode identifies sandbox credentials without exposing key material.
func (c *StripeClient) IsTestMode() bool {
	return strings.HasPrefix(c.secretKey, "sk_test_") || strings.HasPrefix(c.secretKey, "rk_test_")
}

// GetSecretKeyID returns Stripe's non-secret management identifier.
func (c *StripeClient) GetSecretKeyID() string { return c.keyID }

func (c *StripeClient) HasWebhookSecret() bool {
	return c.webhookSecret != ""
}

// HealthCheck validates the secret key with a lightweight authenticated call.
// /v1/balance is the smallest valid request that still proves connectivity +
// auth, mirroring what the Cashfree client does.
func (c *StripeClient) HealthCheck() error {
	_, err := c.doFormRequest("GET", "/balance", nil)
	return err
}

// ToCents converts a decimal amount (e.g. 4.99 USD) to cents (499). Same
// idea as ToPaise for INR.
func ToCents(amount float64) int {
	return int(amount * 100)
}

// FromCents converts cents to the decimal amount.
func FromCents(cents int) float64 {
	return float64(cents) / 100.0
}

// doFormRequest executes an authenticated HTTP request to the Stripe API
// using form-encoded bodies (Stripe's standard). GET requests pass nil.
func (c *StripeClient) doFormRequest(method, path string, form url.Values) ([]byte, error) {
	return c.doFormRequestContext(context.Background(), method, path, form, "")
}

func (c *StripeClient) doFormRequestContext(ctx context.Context, method, path string, form url.Values, idempotencyKey string) ([]byte, error) {
	fullURL := stripeBaseURL + path

	var req *http.Request
	var err error
	if form != nil && method != "GET" {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, bytes.NewBufferString(form.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	} else {
		req, err = http.NewRequestWithContext(ctx, method, fullURL, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Stripe-Version", "2024-06-20")

	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	client := c.httpClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stripe request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe API error (HTTP %d)", resp.StatusCode)
	}
	return respBody, nil
}
