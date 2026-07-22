package services

import (
	"encoding/json"
	"fmt"
)

// route_onboarding.go — registering a chef's settlement bank account with
// Razorpay (#740).
//
// Route pays a chef by settling their linked account's balance to a bank
// account registered ON THAT LINKED ACCOUNT; we never initiate a bank transfer.
// The legacy v1 CreateLinkedAccount sends identity only — no bank details — so
// an account created that way can receive transfers but can never pay them out.
// These are the v2 onboarding endpoints that close that gap:
//
//	POST  /v2/accounts                              → acc_XXX
//	POST  /v2/accounts/:id/stakeholders             → the person behind the account
//	POST  /v2/accounts/:id/products                 → request the "route" product
//	PATCH /v2/accounts/:id/products/:pid            → attach the settlement account
//
// Razorpay reviews the bank details asynchronously, so the last call returns an
// activation_status that may be needs_clarification with a requirements array.
// That is a normal outcome, not an error — the admin surface acts on it.

// razorpayBaseV2 hosts the onboarding APIs. razorpayBaseURL is pinned to /v1,
// which has no settlement configuration.
const razorpayBaseV2 = "https://api.razorpay.com/v2"

// V2AccountRequest creates a linked account.
type V2AccountRequest struct {
	Email        string                  `json:"email"`
	Phone        string                  `json:"phone"`
	LegalName    string                  `json:"legal_business_name"`
	BusinessType string                  `json:"business_type"`
	ContactName  string                  `json:"contact_name"`
	Type         string                  `json:"type,omitempty"`
	LegalInfo    *LinkedAccountLegalInfo `json:"legal_info,omitempty"`
	Profile      *LinkedAccountProfile   `json:"profile,omitempty"`
}

// V2AccountResponse is the created linked account.
type V2AccountResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
}

// StakeholderRequest identifies the individual behind the account, which
// Razorpay requires before a product configuration can be activated.
type StakeholderRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone *struct {
		Primary string `json:"primary,omitempty"`
	} `json:"phone,omitempty"`
}

// StakeholderResponse is the created stakeholder.
type StakeholderResponse struct {
	ID string `json:"id"`
}

// SettlementAccount is the chef's bank destination. Route settles by NEFT/IMPS
// to a bank account — there is no UPI VPA destination for a linked account, so
// a chef who nominates UPI cannot be paid on this rail.
type SettlementAccount struct {
	BeneficiaryName string
	AccountNumber   string
	IFSC            string
}

// RouteRequirement is one reason an activation is blocked, carrying the field
// at fault and the endpoint that resolves it.
type RouteRequirement struct {
	FieldReference string `json:"field_reference"`
	ReasonCode     string `json:"reason_code"`
	ResolutionURL  string `json:"resolution_url"`
}

// RouteProductResponse is a linked account's route product configuration.
//
// ActivationStatus is the field that decides whether a chef can actually be
// paid: only an activated configuration has a working settlement destination.
type RouteProductResponse struct {
	ID               string             `json:"id"`
	ActivationStatus string             `json:"activation_status"`
	Requirements     []RouteRequirement `json:"requirements,omitempty"`
}

// v2URL resolves a v2 path. A test client's baseURL carries no version, so the
// prefix is added explicitly and a test server sees the same path shape as
// production.
func (c *RazorpayClient) v2URL(path string) string {
	if c.baseURL != "" {
		return c.baseURL + "/v2" + path
	}
	return razorpayBaseV2 + path
}

// doV2 issues an authenticated request against the v2 API.
func (c *RazorpayClient) doV2(method, path string, payload any) ([]byte, error) {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}
	return c.doURL(method, c.v2URL(path), body, nil)
}

// CreateLinkedAccountV2 creates a linked account that can later be configured
// with a settlement destination.
func (c *RazorpayClient) CreateLinkedAccountV2(req *V2AccountRequest) (*V2AccountResponse, error) {
	if req.Type == "" {
		req.Type = "route"
	}
	raw, err := c.doV2("POST", "/accounts", req)
	if err != nil {
		return nil, err
	}
	var out V2AccountResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to parse account response: %w", err)
	}
	return &out, nil
}

// CreateStakeholder attaches the individual behind a linked account.
func (c *RazorpayClient) CreateStakeholder(accountID string, req *StakeholderRequest) (*StakeholderResponse, error) {
	raw, err := c.doV2("POST", "/accounts/"+accountID+"/stakeholders", req)
	if err != nil {
		return nil, err
	}
	var out StakeholderResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to parse stakeholder response: %w", err)
	}
	return &out, nil
}

// RequestRouteProduct asks Razorpay to enable the route product on an account,
// returning the configuration the settlement account is then attached to.
func (c *RazorpayClient) RequestRouteProduct(accountID string) (*RouteProductResponse, error) {
	raw, err := c.doV2("POST", "/accounts/"+accountID+"/products", map[string]string{
		"product_name": "route",
		"tnc_accepted": "true",
	})
	if err != nil {
		return nil, err
	}
	var out RouteProductResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to parse product response: %w", err)
	}
	return &out, nil
}

// ConfigureRouteSettlement attaches the chef's bank account to their route
// product configuration — the step that gives a released transfer somewhere to
// go.
//
// A needs_clarification response is returned as a normal result, not an error:
// Razorpay reviews the details asynchronously and the requirements have to
// reach the admin queue rather than being swallowed as a failure.
func (c *RazorpayClient) ConfigureRouteSettlement(accountID, productID string, acct *SettlementAccount) (*RouteProductResponse, error) {
	payload := map[string]any{
		"settlements": map[string]string{
			"account_number":   acct.AccountNumber,
			"ifsc_code":        acct.IFSC,
			"beneficiary_name": acct.BeneficiaryName,
		},
	}
	raw, err := c.doV2("PATCH", "/accounts/"+accountID+"/products/"+productID, payload)
	if err != nil {
		return nil, err
	}
	var out RouteProductResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to parse settlement response: %w", err)
	}
	return &out, nil
}
