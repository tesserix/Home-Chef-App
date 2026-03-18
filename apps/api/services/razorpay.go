package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/homechef/api/config"
)

const razorpayBaseURL = "https://api.razorpay.com/v1"

// RazorpayClient handles all Razorpay API interactions
type RazorpayClient struct {
	keyID     string
	keySecret string
}

var razorpayClient *RazorpayClient

// InitRazorpay initializes the Razorpay client
func InitRazorpay() {
	cfg := config.AppConfig
	if cfg.RazorpayKeyID == "" || cfg.RazorpayKeySecret == "" {
		log.Println("Warning: Razorpay credentials not configured")
		return
	}
	razorpayClient = &RazorpayClient{
		keyID:     cfg.RazorpayKeyID,
		keySecret: cfg.RazorpayKeySecret,
	}
	log.Println("Razorpay client initialized")
}

// GetRazorpay returns the initialized Razorpay client
func GetRazorpay() *RazorpayClient {
	return razorpayClient
}

// --- Linked Accounts (Razorpay Route) ---

// LinkedAccountRequest represents the request to create a Razorpay Route linked account
type LinkedAccountRequest struct {
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	LegalName     string `json:"legal_business_name"`
	BusinessType  string `json:"business_type"` // individual, partnership, etc.
	ContactName   string `json:"contact_name"`
	LegalInfo     *LinkedAccountLegalInfo `json:"legal_info,omitempty"`
	Profile       *LinkedAccountProfile   `json:"profile,omitempty"`
}

type LinkedAccountLegalInfo struct {
	Pan string `json:"pan,omitempty"`
	Gst string `json:"gst,omitempty"`
}

type LinkedAccountProfile struct {
	Category    string                   `json:"category,omitempty"`
	SubCategory string                   `json:"subcategory,omitempty"`
	Addresses   *LinkedAccountAddresses  `json:"addresses,omitempty"`
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
	Account  string  `json:"account"`  // Razorpay linked account ID
	Amount   int     `json:"amount"`   // Amount in paise (INR smallest unit)
	Currency string  `json:"currency"`
	Notes    map[string]string `json:"notes,omitempty"`
	OnHold   bool    `json:"on_hold,omitempty"` // Hold transfer until delivery confirmed
}

// OrderRequest creates a Razorpay order with optional Route transfers
type OrderRequest struct {
	Amount    int               `json:"amount"`    // Total in paise
	Currency  string            `json:"currency"`
	Receipt   string            `json:"receipt"`   // Order number
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

// CreateOrder creates a Razorpay order with Route transfer splits
func (c *RazorpayClient) CreateOrder(req *OrderRequest) (*OrderResponse, error) {
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

// --- Refunds ---

// RefundRequest creates a refund on a payment
type RefundRequest struct {
	Amount int               `json:"amount"` // In paise; 0 = full refund
	Speed  string            `json:"speed"`  // "normal" or "optimum"
	Notes  map[string]string `json:"notes,omitempty"`
	Receipt string           `json:"receipt,omitempty"`
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

	resp, err := c.doRequest("POST", fmt.Sprintf("/payments/%s/refund", paymentID), body)
	if err != nil {
		return nil, err
	}

	var result RefundResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// --- Payment Fetch ---

// PaymentResponse from Razorpay
type PaymentResponse struct {
	ID            string `json:"id"`
	Entity        string `json:"entity"`
	Amount        int    `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	OrderID       string `json:"order_id"`
	Method        string `json:"method"` // card, upi, netbanking, wallet
	Email         string `json:"email"`
	Contact       string `json:"contact"`
	Captured      bool   `json:"captured"`
	AmountRefunded int   `json:"amount_refunded"`
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

// --- Webhook Verification ---

// VerifyWebhookSignature validates that a webhook payload came from Razorpay
func VerifyWebhookSignature(payload []byte, signature string) bool {
	cfg := config.AppConfig
	if cfg.RazorpayWebhookSecret == "" {
		log.Println("Warning: Razorpay webhook secret not configured")
		return false
	}

	mac := hmac.New(sha256.New, []byte(cfg.RazorpayWebhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// GetKeyID returns the Razorpay publishable key ID (for frontend checkout)
func (c *RazorpayClient) GetKeyID() string {
	return c.keyID
}

// HealthCheck validates Razorpay credentials by making a lightweight API call
func (c *RazorpayClient) HealthCheck() error {
	_, err := c.doRequest("GET", "/payments?count=0", nil)
	return err
}

// --- Helpers ---

// ToPaise converts a float amount (e.g. 499.00 INR) to paise (49900)
func ToPaise(amount float64) int {
	return int(amount * 100)
}

// FromPaise converts paise to float amount
func FromPaise(paise int) float64 {
	return float64(paise) / 100.0
}

// doRequest executes an authenticated HTTP request to the Razorpay API
func (c *RazorpayClient) doRequest(method, path string, body []byte) ([]byte, error) {
	url := razorpayBaseURL + path

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.keyID, c.keySecret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("razorpay request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("razorpay API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
