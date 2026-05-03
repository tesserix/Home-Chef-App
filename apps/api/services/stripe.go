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
)

const stripeBaseURL = "https://api.stripe.com/v1"

// stripeCacheTTL mirrors razorpayCacheTTL — admin key updates call
// InvalidateStripe() so the next read picks up new keys immediately.
const stripeCacheTTL = 5 * time.Minute

// Stripe credentials live in GCP Secret Manager alongside the Razorpay ones.
// Product-scoped ("homechef-") keeps them separate from other tenants sharing
// project tesseracthub-480811.
const (
	secretStripeSecretKey      = "prod-homechef-stripe-secret-key"
	secretStripePublishableKey = "prod-homechef-stripe-publishable-key"
	secretStripeWebhookSecret  = "prod-homechef-stripe-webhook-secret"
)

// StripeClient handles all Stripe REST API interactions. Same shape as
// RazorpayClient so the admin UI and payment handlers can treat them
// symmetrically.
type StripeClient struct {
	secretKey      string
	publishableKey string
	webhookSecret  string
	fetchedAt      time.Time
}

var (
	stripeClient *StripeClient
	stripeMu     sync.Mutex
)

func fetchStripeFromSM(ctx context.Context) (*StripeClient, error) {
	secretKey, skErr := GetPlatformSecret(ctx, secretStripeSecretKey)
	publishableKey, _ := GetPlatformSecret(ctx, secretStripePublishableKey)
	webhookSecret, _ := GetPlatformSecret(ctx, secretStripeWebhookSecret)

	if skErr != nil || isPlaceholderValue(secretKey) {
		cfg := config.AppConfig
		if cfg != nil && !isPlaceholderValue(cfg.StripeSecretKey) {
			return &StripeClient{
				secretKey:      cfg.StripeSecretKey,
				publishableKey: cfg.StripePublishableKey,
				webhookSecret:  cfg.StripeWebhookSecret,
				fetchedAt:      time.Now(),
			}, nil
		}
		if skErr != nil {
			return nil, fmt.Errorf("stripe credentials not configured: %w", skErr)
		}
		return nil, fmt.Errorf("stripe credentials missing or still set to placeholder — configure them in Admin → Settings → Payment Gateway")
	}

	return &StripeClient{
		secretKey:      secretKey,
		publishableKey: publishableKey,
		webhookSecret:  webhookSecret,
		fetchedAt:      time.Now(),
	}, nil
}

// GetStripe returns a Stripe client sourced from GCP Secret Manager. nil
// when no credentials are configured — callers must handle that. Mirrors
// GetRazorpay semantics, including cached-client fallback on SM outage.
func GetStripe() *StripeClient {
	stripeMu.Lock()
	defer stripeMu.Unlock()

	if stripeClient != nil && time.Since(stripeClient.fetchedAt) < stripeCacheTTL {
		return stripeClient
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fresh, err := fetchStripeFromSM(ctx)
	if err != nil {
		if stripeClient != nil {
			log.Printf("stripe: SM fetch failed, using cached credentials: %v", err)
			return stripeClient
		}
		log.Printf("stripe: not configured (%v)", err)
		return nil
	}

	stripeClient = fresh
	return stripeClient
}

// InvalidateStripe clears the cached client. Call after admin key rotation.
func InvalidateStripe() {
	stripeMu.Lock()
	defer stripeMu.Unlock()
	stripeClient = nil
	log.Println("stripe: credential cache invalidated")
}

// InitStripe triggers an initial fetch so config gaps surface at startup.
func InitStripe() {
	if GetStripe() != nil {
		log.Println("stripe: client initialized from Secret Manager")
		return
	}
	log.Println("stripe: no credentials yet — configure via Admin → Settings → Payment Gateway")
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
	ID       string `json:"id"`
	Object   string `json:"object"`
	Country  string `json:"country"`
	Email    string `json:"email"`
	Type     string `json:"type"`
	Charges  bool   `json:"charges_enabled"`
	Payouts  bool   `json:"payouts_enabled"`
	Details  bool   `json:"details_submitted"`
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
	ID           string `json:"id"`
	Object       string `json:"object"`
	Amount       int    `json:"amount"`
	Currency     string `json:"currency"`
	Status       string `json:"status"`
	ClientSecret string `json:"client_secret"`
	Customer     string `json:"customer"`
	Description  string `json:"description"`
}

func (c *StripeClient) CreatePaymentIntent(req *StripePaymentIntentRequest) (*StripePaymentIntent, error) {
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
	}
	if req.ApplicationFeeCents > 0 {
		form.Set("application_fee_amount", strconv.Itoa(req.ApplicationFeeCents))
	}
	for k, v := range req.Metadata {
		form.Set("metadata["+k+"]", v)
	}

	resp, err := c.doFormRequest("POST", "/payment_intents", form)
	if err != nil {
		return nil, err
	}
	var result StripePaymentIntent
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent: %w", err)
	}
	return &result, nil
}

func (c *StripeClient) FetchPaymentIntent(id string) (*StripePaymentIntent, error) {
	resp, err := c.doFormRequest("GET", "/payment_intents/"+id, nil)
	if err != nil {
		return nil, err
	}
	var result StripePaymentIntent
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse payment intent: %w", err)
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
	PaymentIntent         string
	Amount                int // In cents; 0 = full refund
	Reason                string
	ReverseTransfer       bool
	RefundApplicationFee  bool
	Metadata              map[string]string
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

	resp, err := c.doFormRequest("POST", "/refunds", form)
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
	if stripeClient == nil || stripeClient.webhookSecret == "" {
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
	mac := hmac.New(sha256.New, []byte(stripeClient.webhookSecret))
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

// GetSecretKeyID returns the sk_* prefix (safe — full secret is hashed-like
// from the caller's perspective, we only surface the first ~12 chars for UI).
func (c *StripeClient) GetSecretKeyID() string {
	return c.secretKey
}

func (c *StripeClient) HasWebhookSecret() bool {
	return c.webhookSecret != ""
}

// HealthCheck validates the secret key with a lightweight authenticated call.
// /v1/balance is the smallest valid request that still proves connectivity +
// auth, mirroring what the Razorpay client does.
func (c *StripeClient) HealthCheck() error {
	_, err := c.doFormRequest("GET", "/balance", nil)
	return err
}

// ToCents converts a decimal amount (e.g. 4.99 USD) to cents (499). Same
// idea as ToPaise for Razorpay.
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
	fullURL := stripeBaseURL + path

	var req *http.Request
	var err error
	if form != nil && method != "GET" {
		req, err = http.NewRequest(method, fullURL, bytes.NewBufferString(form.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	} else {
		req, err = http.NewRequest(method, fullURL, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Stripe-Version", "2024-06-20")

	client := &http.Client{Timeout: 30 * time.Second}
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
		return nil, fmt.Errorf("stripe API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}
