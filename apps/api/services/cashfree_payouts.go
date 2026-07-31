package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
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
	"github.com/homechef/api/payouts"
)

// cashfree_payouts.go — the Cashfree PAYOUTS adapter: money OUT to chefs and
// riders.
//
// Cashfree Payouts is a SEPARATE PRODUCT from Cashfree PG with its own dashboard
// section, its own API keys and its own hosts. Sharing credentials between them
// is not possible, which is why this file carries its own secret names and its
// own client rather than extending CashfreeClient. The two do share the
// cashfreeAmount rupee-decimal type, because the wire format is identical and one
// money conversion is better than two.
//
// This closes the loop the PG adapter deliberately left open: because Cashfree PG
// captures the full order to the platform and splits nothing at the gateway (see
// models.ProviderSupportsGatewaySplit), chefs and riders are owed money the
// platform is holding. Until now that was disbursed by hand — models/statement.go
// records that automation was "gated on an Indian entity". Payouts is what
// removes that blocker.
//
// ── The one thing to get right ───────────────────────────────────────────────
//
// A payout call that times out may still have sent the money. Everything here is
// arranged so that case is RESOLVABLE rather than guessed at: the batch's
// idempotency key becomes Cashfree's transfer_id, and GetPayoutByReference reads
// it back. A caller that retries a transfer instead of reading it back can pay
// somebody twice, and unlike a double charge there is no refund endpoint to undo
// it — you are asking a stranger to send the money back.

const (
	// Payouts hosts. Same sandbox/production split as PG, different path prefix.
	cashfreePayoutLiveBaseURL = "https://api.cashfree.com/payout"
	cashfreePayoutTestBaseURL = "https://sandbox.cashfree.com/payout"

	// Pinned deliberately: the v2 beneficiary/transfer shapes parsed below are
	// the 2024-01-01 ones. Bumping it is a reviewed change.
	cashfreePayoutAPIVersion = "2024-01-01"

	cashfreePayoutCacheTTL       = 5 * time.Minute
	cashfreePayoutRequestTimeout = 30 * time.Second

	// RailName is what Batch.Provider records for this rail. Persisted, so it
	// must not be renamed without migrating historical rows.
	CashfreePayoutRailName = "cashfree_payouts"
)

// Payout credentials live in their own Secret Manager entries — they are a
// different product's keys, not the PG ones.
const (
	SecretCashfreePayoutClientID      = "prod-homechef-cashfree-payout-client-id"
	SecretCashfreePayoutClientSecret  = "prod-homechef-cashfree-payout-client-secret"
	SecretCashfreePayoutWebhookSecret = "prod-homechef-cashfree-payout-webhook-secret"

	SecretCashfreePayoutTestClientID      = "prod-homechef-cashfree-payout-test-client-id"
	SecretCashfreePayoutTestClientSecret  = "prod-homechef-cashfree-payout-test-client-secret"
	SecretCashfreePayoutTestWebhookSecret = "prod-homechef-cashfree-payout-test-webhook-secret"

	// The RSA PUBLIC KEY Cashfree issues on request (care@cashfree.com), used to
	// authenticate from a dynamic IP. See the signature note on signRequest —
	// this is what makes the integration survive a NAT IP change. Optional: an
	// empty slot falls back to IP whitelisting.
	SecretCashfreePayoutPublicKey     = "prod-homechef-cashfree-payout-public-key"
	SecretCashfreePayoutTestPublicKey = "prod-homechef-cashfree-payout-test-public-key"
)

// CashfreePayoutPublicKeySecretName returns the public-key slot for a mode.
func CashfreePayoutPublicKeySecretName(mode string) string {
	if models.IsTestMode(mode) {
		return SecretCashfreePayoutTestPublicKey
	}
	return SecretCashfreePayoutPublicKey
}

// CashfreePayoutClient talks to the Cashfree Payouts API for one credential slot.
type CashfreePayoutClient struct {
	clientID      string
	clientSecret  string
	webhookSecret string
	mode          string
	baseURL       string // test seam; empty in production
	// publicKey is Cashfree's RSA public key, parsed once at construction. nil
	// when no key is configured, in which case requests rely on IP whitelisting.
	publicKey *rsa.PublicKey
	fetchedAt time.Time
}

var (
	cashfreePayoutClients = map[string]*CashfreePayoutClient{}
	cashfreePayoutMu      sync.Mutex
)

// CashfreePayoutSecretNames is the exported form the admin write path uses, so
// the slot an admin saves into and the slot the app reads from cannot drift.
func CashfreePayoutSecretNames(mode string) (clientID, clientSecret, webhookSecret string) {
	if models.IsTestMode(mode) {
		return SecretCashfreePayoutTestClientID, SecretCashfreePayoutTestClientSecret, SecretCashfreePayoutTestWebhookSecret
	}
	return SecretCashfreePayoutClientID, SecretCashfreePayoutClientSecret, SecretCashfreePayoutWebhookSecret
}

// cashfreePayoutBaseURLFor resolves the Payouts host for a slot.
//
// Same rule as the PG adapter (see cashfreeBaseURLFor): the environment follows
// the CREDENTIALS, so a live slot holding sandbox keys reaches the sandbox and
// works rather than 401-ing against production. The asymmetry is the same and
// matters more here, because this rail sends money OUT: the test slot is pinned
// to sandbox whatever its credentials say, so a sandbox disbursement can never
// reach a real bank account.
func cashfreePayoutBaseURLFor(mode, clientID, clientSecret string) string {
	if models.IsTestMode(mode) {
		return cashfreePayoutTestBaseURL
	}
	if cashfreeCredentialsAreSandbox(clientID, clientSecret) {
		return cashfreePayoutTestBaseURL
	}
	return cashfreePayoutLiveBaseURL
}

func fetchCashfreePayoutFromSM(ctx context.Context, mode string) (*CashfreePayoutClient, error) {
	mode = models.NormalizeMode(mode)
	idName, secretName, webhookName := CashfreePayoutSecretNames(mode)

	clientID, idErr := GetPlatformSecret(ctx, idName)
	clientSecret, secErr := GetPlatformSecret(ctx, secretName)
	webhookSecret, _ := GetPlatformSecret(ctx, webhookName)
	// Optional: absent means authenticate by whitelisted IP instead.
	publicKeyPEM, _ := GetPlatformSecret(ctx, CashfreePayoutPublicKeySecretName(mode))
	publicKey, pkErr := parseCashfreePublicKey(publicKeyPEM)
	if pkErr != nil {
		// Refuse rather than silently dropping to IP auth — see the note in do().
		return nil, fmt.Errorf("cashfree-payouts[%s]: %w", mode, pkErr)
	}

	if idErr != nil || secErr != nil || isPlaceholderValue(clientID) || isPlaceholderValue(clientSecret) {
		// Dev fallback, LIVE-ONLY for the same reason every other slot is: one
		// set of env vars cannot describe two environments, and serving them to
		// the test slot would point sandbox payouts at live credentials — which
		// for a MONEY-OUT rail means real disbursements from a test run.
		if !models.IsTestMode(mode) {
			if cfg := config.AppConfig; cfg != nil &&
				!isPlaceholderValue(cfg.CashfreePayoutClientID) && !isPlaceholderValue(cfg.CashfreePayoutClientSecret) {
				return &CashfreePayoutClient{
					clientID:      cfg.CashfreePayoutClientID,
					clientSecret:  cfg.CashfreePayoutClientSecret,
					webhookSecret: cfg.CashfreePayoutWebhookSecret,
					mode:          mode,
					fetchedAt:     time.Now(),
				}, nil
			}
		}
		if idErr != nil {
			return nil, fmt.Errorf("cashfree-payouts[%s] credentials not configured: %w", mode, idErr)
		}
		return nil, fmt.Errorf("cashfree-payouts[%s] credentials missing or still set to placeholder — configure them in Admin → Settings → Payouts", mode)
	}

	return &CashfreePayoutClient{
		clientID:      clientID,
		clientSecret:  clientSecret,
		webhookSecret: webhookSecret,
		mode:          mode,
		publicKey:     publicKey,
		fetchedAt:     time.Now(),
	}, nil
}

// GetCashfreePayoutFor returns the cached client for one mode, or nil when that
// slot is not configured. Callers must handle nil.
func GetCashfreePayoutFor(mode string) *CashfreePayoutClient {
	mode = models.NormalizeMode(mode)

	cashfreePayoutMu.Lock()
	defer cashfreePayoutMu.Unlock()

	if c := cashfreePayoutClients[mode]; c != nil && time.Since(c.fetchedAt) < cashfreePayoutCacheTTL {
		return c
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fresh, err := fetchCashfreePayoutFromSM(ctx, mode)
	if err != nil {
		if c := cashfreePayoutClients[mode]; c != nil {
			log.Printf("cashfree-payouts[%s]: SM fetch failed, using cached credentials: %v", mode, err)
			return c
		}
		log.Printf("cashfree-payouts[%s]: not configured (%v)", mode, err)
		return nil
	}
	cashfreePayoutClients[mode] = fresh
	return fresh
}

// GetCashfreePayout is the live slot.
func GetCashfreePayout() *CashfreePayoutClient { return GetCashfreePayoutFor(models.ChefModeLive) }

func snapshotCashfreePayoutClient(mode string) *CashfreePayoutClient {
	cashfreePayoutMu.Lock()
	defer cashfreePayoutMu.Unlock()
	return cashfreePayoutClients[models.NormalizeMode(mode)]
}

// InvalidateCashfreePayoutFor clears one slot's cached client.
func InvalidateCashfreePayoutFor(mode string) {
	mode = models.NormalizeMode(mode)
	cashfreePayoutMu.Lock()
	defer cashfreePayoutMu.Unlock()
	delete(cashfreePayoutClients, mode)
	log.Printf("cashfree-payouts[%s]: credential cache invalidated", mode)
}

// InvalidateCashfreePayout clears every slot.
func InvalidateCashfreePayout() {
	InvalidateCashfreePayoutFor(models.ChefModeLive)
	InvalidateCashfreePayoutFor(models.ChefModeTest)
}

// SetCashfreePayoutClientFor installs a client into one slot. Test seam only.
func SetCashfreePayoutClientFor(mode string, c *CashfreePayoutClient) {
	mode = models.NormalizeMode(mode)
	cashfreePayoutMu.Lock()
	defer cashfreePayoutMu.Unlock()
	if c != nil && c.fetchedAt.IsZero() {
		c.fetchedAt = time.Now()
	}
	cashfreePayoutClients[mode] = c
}

// NewCashfreePayoutTestClient builds a client pointed at an httptest.Server.
// Test-only.
func NewCashfreePayoutTestClient(baseURL, clientID, clientSecret, webhookSecret, mode string) *CashfreePayoutClient {
	return &CashfreePayoutClient{
		clientID:      clientID,
		clientSecret:  clientSecret,
		webhookSecret: webhookSecret,
		mode:          models.NormalizeMode(mode),
		baseURL:       baseURL,
		fetchedAt:     time.Now(),
	}
}

// InitCashfreePayouts probes both slots at startup.
func InitCashfreePayouts() {
	for _, mode := range []string{models.ChefModeLive, models.ChefModeTest} {
		if GetCashfreePayoutFor(mode) != nil {
			log.Printf("cashfree-payouts[%s]: client initialized from Secret Manager", mode)
			continue
		}
		log.Printf("cashfree-payouts[%s]: no credentials yet — configure via Admin → Settings → Payouts", mode)
	}
}

// GetClientID returns the public client identifier (for the admin surface).
func (c *CashfreePayoutClient) GetClientID() string { return c.clientID }

// HasWebhookSecret reports whether a webhook secret is set, without exposing it.
func (c *CashfreePayoutClient) HasWebhookSecret() bool { return c.webhookSecret != "" }

// Mode reports which credential slot this client is.
func (c *CashfreePayoutClient) Mode() string { return c.mode }

// IsSandbox derives the environment from the resolved host, so an injected test
// baseURL is reported honestly.
func (c *CashfreePayoutClient) IsSandbox() bool {
	return strings.Contains(c.resolvedBaseURL(), "sandbox.cashfree.com")
}

// --- Beneficiaries ---

type cfBeneficiaryInstrument struct {
	BankAccountNumber string `json:"bank_account_number,omitempty"`
	BankIFSC          string `json:"bank_ifsc,omitempty"`
	VPA               string `json:"vpa,omitempty"`
}

type cfBeneficiaryContact struct {
	Email       string `json:"beneficiary_email,omitempty"`
	Phone       string `json:"beneficiary_phone,omitempty"`
	CountryCode string `json:"beneficiary_country_code,omitempty"`
}

type cfBeneficiaryRequest struct {
	BeneficiaryID     string                   `json:"beneficiary_id"`
	BeneficiaryName   string                   `json:"beneficiary_name"`
	InstrumentDetails *cfBeneficiaryInstrument `json:"beneficiary_instrument_details,omitempty"`
	ContactDetails    *cfBeneficiaryContact    `json:"beneficiary_contact_details,omitempty"`
}

type cfBeneficiaryResponse struct {
	BeneficiaryID     string `json:"beneficiary_id"`
	BeneficiaryName   string `json:"beneficiary_name"`
	BeneficiaryStatus string `json:"beneficiary_status"`
	AddedOn           string `json:"added_on,omitempty"`
}

// cashfreeBeneficiaryStatus maps Cashfree's beneficiary_status onto the engine's
// MethodStatus.
//
// INITIATED is pending, NOT verified — money must not be sent to a destination
// the rail has not yet accepted. Everything terminal-bad collapses to invalid so
// the admin surface has one thing to act on.
func cashfreeBeneficiaryStatus(s string) payouts.MethodStatus {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "VERIFIED":
		return payouts.MethodVerified
	case "INITIATED":
		return payouts.MethodPending
	case "INVALID", "FAILED", "CANCELLED", "DELETED":
		return payouts.MethodInvalid
	default:
		// An unrecognised status must NOT be treated as payable. Failing closed
		// costs a delayed payout; failing open sends money somewhere unverified.
		return payouts.MethodPending
	}
}

// CreateBeneficiary registers a destination.
//
// A duplicate beneficiary_id returns 409 — that is idempotent success, not a
// failure: a previous attempt already created it. The existing registration is
// fetched and returned, which is exactly why BeneficiaryIDFor is deterministic.
func (c *CashfreePayoutClient) CreateBeneficiary(ctx context.Context, req payouts.BeneficiaryRequest) (payouts.BeneficiaryResult, error) {
	if !req.Instrument.Valid() {
		return payouts.BeneficiaryResult{}, fmt.Errorf("cashfree-payouts: incomplete %s instrument", req.Instrument.Kind)
	}

	body := cfBeneficiaryRequest{
		BeneficiaryID:   req.BeneficiaryID,
		BeneficiaryName: sanitizeBeneficiaryName(req.Name),
		InstrumentDetails: &cfBeneficiaryInstrument{
			BankAccountNumber: req.Instrument.AccountNumber,
			BankIFSC:          req.Instrument.IFSC,
			VPA:               req.Instrument.VPA,
		},
	}
	if req.Email != "" || req.Phone != "" {
		body.ContactDetails = &cfBeneficiaryContact{Email: req.Email, Phone: req.Phone}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return payouts.BeneficiaryResult{}, fmt.Errorf("cashfree-payouts: marshal beneficiary: %w", err)
	}

	resp, status, err := c.do(ctx, http.MethodPost, "/beneficiary", raw)
	if err != nil {
		return payouts.BeneficiaryResult{}, err
	}
	if status == http.StatusConflict {
		log.Printf("cashfree-payouts[%s]: beneficiary %s already exists — reading it back", c.mode, req.BeneficiaryID)
		return c.FetchBeneficiary(ctx, req.BeneficiaryID)
	}
	if status >= 400 {
		// A 4xx on a beneficiary is the rail refusing the DESTINATION (bad IFSC,
		// name mismatch). Retrying cannot help, so it is surfaced as terminal.
		if status < 500 {
			return payouts.BeneficiaryResult{
				Status: payouts.MethodInvalid,
				Detail: cashfreePayoutErrorDetail(resp),
			}, fmt.Errorf("%w: %s", payouts.ErrBeneficiaryRejected, cashfreePayoutErrorDetail(resp))
		}
		return payouts.BeneficiaryResult{}, cashfreePayoutError(status, resp)
	}

	var out cfBeneficiaryResponse
	if err := json.Unmarshal(resp, &out); err != nil {
		return payouts.BeneficiaryResult{}, fmt.Errorf("cashfree-payouts: parse beneficiary: %w", err)
	}
	return payouts.BeneficiaryResult{
		BeneficiaryID: out.BeneficiaryID,
		Status:        cashfreeBeneficiaryStatus(out.BeneficiaryStatus),
	}, nil
}

// FetchBeneficiary reads a registered destination back.
func (c *CashfreePayoutClient) FetchBeneficiary(ctx context.Context, beneficiaryID string) (payouts.BeneficiaryResult, error) {
	path := "/beneficiary?beneficiary_id=" + url.QueryEscape(beneficiaryID)
	resp, status, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return payouts.BeneficiaryResult{}, err
	}
	if status == http.StatusNotFound {
		return payouts.BeneficiaryResult{}, payouts.ErrRailNotFound
	}
	if status >= 400 {
		return payouts.BeneficiaryResult{}, cashfreePayoutError(status, resp)
	}
	var out cfBeneficiaryResponse
	if err := json.Unmarshal(resp, &out); err != nil {
		return payouts.BeneficiaryResult{}, fmt.Errorf("cashfree-payouts: parse beneficiary: %w", err)
	}
	return payouts.BeneficiaryResult{
		BeneficiaryID: out.BeneficiaryID,
		Status:        cashfreeBeneficiaryStatus(out.BeneficiaryStatus),
	}, nil
}

// sanitizeBeneficiaryName strips everything Cashfree rejects.
//
// Cashfree allows only alphabets and whitespace in beneficiary_name, and rejects
// the whole registration otherwise. Bank account names routinely carry dots and
// ampersands ("M/s. A & B Foods"), so stripping here turns a hard rejection into
// a successful registration. The name is still matched against the account by the
// bank, so this cannot be used to register a mismatched name.
func sanitizeBeneficiaryName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune(r)
		default:
			// Replace with a space rather than deleting, so "A&B" becomes "A B"
			// instead of "AB" — closer to what the bank holds.
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// --- Transfers ---

type cfTransferBeneficiary struct {
	BeneficiaryID string `json:"beneficiary_id"`
}

type cfTransferRequest struct {
	TransferID       string                `json:"transfer_id"`
	TransferAmount   cashfreeAmount        `json:"transfer_amount"`
	TransferCurrency string                `json:"transfer_currency,omitempty"`
	TransferMode     string                `json:"transfer_mode,omitempty"`
	Beneficiary      cfTransferBeneficiary `json:"beneficiary_details"`
	Remarks          string                `json:"remarks,omitempty"`
}

type cfTransferResponse struct {
	CFTransferID      json.Number    `json:"cf_transfer_id"`
	TransferID        string         `json:"transfer_id"`
	Status            string         `json:"status"`
	StatusCode        string         `json:"status_code,omitempty"`
	StatusDescription string         `json:"status_description,omitempty"`
	TransferMode      string         `json:"transfer_mode,omitempty"`
	TransferUTR       string         `json:"transfer_utr,omitempty"`
	TransferAmount    cashfreeAmount `json:"transfer_amount,omitempty"`
	UpdatedOn         string         `json:"updated_on,omitempty"`
}

// cashfreeTransferStatus maps Cashfree's transfer status onto the rail's
// normalised vocabulary.
//
// The non-terminal set is larger than it looks and getting it wrong is the most
// expensive mistake available here. RECEIVED, QUEUED, PENDING, APPROVAL_PENDING
// and VALIDATION_PENDING all mean "still in flight": treating any of them as
// failed would mark the batch failed, credit the payee back into the ledger, and
// then the money would ALSO land — paying twice.
//
// REJECTED and MANUALLY_REJECTED are terminal-failed (never left Cashfree).
// REVERSED is its own state: money left and came back, which the ledger has to
// undo differently from a payment that never happened.
func cashfreeTransferStatus(s string) (payouts.RailStatus, bool) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "SUCCESS":
		return payouts.RailSuccess, true
	case "FAILED", "REJECTED", "MANUALLY_REJECTED":
		return payouts.RailFailed, true
	case "REVERSED":
		return payouts.RailReversed, true
	case "RECEIVED", "QUEUED", "PENDING", "APPROVAL_PENDING", "VALIDATION_PENDING":
		return payouts.RailPending, true
	default:
		// Unknown status: NOT terminal. An unrecognised state must leave the
		// batch executing so it is resolved by a later status read, never
		// guessed into a terminal state that moves the ledger.
		return payouts.RailPending, false
	}
}

// cashfreeTransferMode maps a method kind onto Cashfree's transfer_mode.
func cashfreeTransferMode(k payouts.MethodKind) string {
	if k == payouts.MethodUPI {
		return "upi"
	}
	// banktransfer lets Cashfree pick IMPS/NEFT/RTGS by amount and hours, which
	// is better than pinning one: IMPS has a per-transfer ceiling and NEFT does
	// not settle out of hours.
	return "banktransfer"
}

// CreateTransfer initiates a disbursement.
//
// Returns payouts.ErrRailAmbiguous when the outcome is unknown (timeout, 5xx,
// transport failure). The caller MUST NOT retry on that — it must call
// GetTransfer with the same transfer_id.
func (c *CashfreePayoutClient) CreateTransfer(ctx context.Context, req payouts.DisburseRequest) (payouts.DisburseResult, error) {
	if req.IdempotencyKey == "" {
		return payouts.DisburseResult{}, errors.New("cashfree-payouts: transfer needs an idempotency key")
	}
	if req.BeneficiaryID == "" {
		return payouts.DisburseResult{}, errors.New("cashfree-payouts: transfer needs a beneficiary")
	}
	if !req.Amount.IsPositive() {
		return payouts.DisburseResult{}, fmt.Errorf("cashfree-payouts: transfer amount must be positive, got %s", req.Amount)
	}

	body, err := json.Marshal(cfTransferRequest{
		// The batch's idempotency key IS the transfer id. That equivalence is
		// what makes GetPayoutByReference able to answer "did this batch pay?".
		TransferID:       cashfreeTransferID(req.IdempotencyKey),
		TransferAmount:   cashfreeAmount(req.Amount.Minor),
		TransferCurrency: string(req.Amount.Currency),
		TransferMode:     cashfreeTransferMode(req.Kind),
		Beneficiary:      cfTransferBeneficiary{BeneficiaryID: req.BeneficiaryID},
		Remarks:          sanitizeRemarks(req.Remarks),
	})
	if err != nil {
		return payouts.DisburseResult{}, fmt.Errorf("cashfree-payouts: marshal transfer: %w", err)
	}

	resp, status, err := c.do(ctx, http.MethodPost, "/transfers", body)
	if err != nil {
		// Transport failure: the request may have been executed. Ambiguous.
		return payouts.DisburseResult{}, fmt.Errorf("%w: %v", payouts.ErrRailAmbiguous, err)
	}
	if status == http.StatusConflict {
		// Duplicate transfer_id — this exact payout was already accepted. Read
		// it back rather than reporting an error the caller might retry.
		log.Printf("cashfree-payouts[%s]: transfer %s already exists — reading it back", c.mode, req.IdempotencyKey)
		return c.GetTransfer(ctx, req.IdempotencyKey)
	}
	if status >= 500 {
		// Server-side failure gives no information about whether the transfer
		// was queued. Ambiguous by definition.
		return payouts.DisburseResult{}, fmt.Errorf("%w: HTTP %d", payouts.ErrRailAmbiguous, status)
	}
	if status >= 400 {
		// A 4xx is a rejected instruction — validation, limits, insufficient
		// balance. Money did not move, so this is safely terminal.
		return payouts.DisburseResult{
			Status:        payouts.RailFailed,
			FailureCode:   fmt.Sprintf("http_%d", status),
			FailureDetail: cashfreePayoutErrorDetail(resp),
		}, nil
	}

	return parseTransferResponse(resp)
}

// GetTransfer resolves what actually happened to a transfer.
//
// This is the answer to an ambiguous CreateTransfer, and the ONLY safe way to
// decide whether a retry is permitted. ErrRailNotFound means Cashfree has no
// record — the only proof money did not move.
func (c *CashfreePayoutClient) GetTransfer(ctx context.Context, idempotencyKey string) (payouts.DisburseResult, error) {
	path := "/transfers?transfer_id=" + url.QueryEscape(cashfreeTransferID(idempotencyKey))
	resp, status, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return payouts.DisburseResult{}, err
	}
	if status == http.StatusNotFound {
		return payouts.DisburseResult{}, payouts.ErrRailNotFound
	}
	if status >= 400 {
		return payouts.DisburseResult{}, cashfreePayoutError(status, resp)
	}
	return parseTransferResponse(resp)
}

func parseTransferResponse(raw []byte) (payouts.DisburseResult, error) {
	var out cfTransferResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return payouts.DisburseResult{}, fmt.Errorf("cashfree-payouts: parse transfer: %w", err)
	}
	st, known := cashfreeTransferStatus(out.Status)
	if !known {
		log.Printf("cashfree-payouts: unrecognised transfer status %q for %s — holding as pending",
			out.Status, out.TransferID)
	}
	res := payouts.DisburseResult{
		Status:    st,
		Reference: out.CFTransferID.String(),
		UTR:       out.TransferUTR,
	}
	if st == payouts.RailFailed || st == payouts.RailReversed {
		res.FailureCode = out.StatusCode
		res.FailureDetail = out.StatusDescription
	}
	if st == payouts.RailSuccess && out.UpdatedOn != "" {
		if t, err := time.Parse(time.RFC3339, out.UpdatedOn); err == nil {
			res.SettledAt = &t
		}
	}
	return res, nil
}

// cashfreeTransferID renders a batch idempotency key as a Cashfree transfer_id.
//
// Cashfree allows alphabets, numbers, underscore and hyphen. The engine's keys
// (payouts.IdempotencyKeyFor) contain colons, so they are normalised — the same
// digest approach the PG refund path uses, keeping one logical operation to one
// identity across both products.
func cashfreeTransferID(logical string) string {
	return "hcp_" + normalizeIdempotencyKey(logical)
}

// sanitizeRemarks trims to what Cashfree accepts on a bank narration:
// alphabets, numbers and spaces only, and short enough to survive truncation.
func sanitizeRemarks(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == ' ':
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if len(out) > 60 {
		out = out[:60]
	}
	if out == "" {
		out = "HomeChef payout"
	}
	return out
}

// --- Request signing (the alternative to IP whitelisting) ---

// cashfreeSignatureTTL bounds how long one signature stays usable. Cashfree
// accepts a short window around the encrypted timestamp; regenerating per request
// is cheap (one RSA op) and avoids any question of drift, so no caching.
const cashfreeSignatureTTL = 5 * time.Minute

// parseCashfreePublicKey decodes the PEM Cashfree issues.
//
// Accepts both PKIX ("BEGIN PUBLIC KEY") and PKCS#1 ("BEGIN RSA PUBLIC KEY"),
// because which one arrives depends on how the key was exported and getting the
// wrong parser produces a baffling "structure error" rather than a clear
// message.
func parseCashfreePublicKey(pemData string) (*rsa.PublicKey, error) {
	pemData = strings.TrimSpace(pemData)
	if pemData == "" {
		return nil, nil
	}
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("cashfree-payouts: public key is not valid PEM")
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("cashfree-payouts: public key is not RSA")
		}
		return rsaPub, nil
	}
	rsaPub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("cashfree-payouts: parse public key: %w", err)
	}
	return rsaPub, nil
}

// signRequest builds the x-cf-signature header, or "" when no key is configured.
//
// The scheme is Cashfree's: RSA-encrypt "<clientId>.<unixSeconds>" with THEIR
// public key and base64 the ciphertext. Note this is encryption, not a digital
// signature — only Cashfree holds the private key, so only Cashfree can recover
// the timestamp and check it is fresh. That is what proves the caller holds the
// key material rather than merely knowing the client id.
//
// This exists because Cashfree Payouts otherwise authenticates by source IP, and
// this platform's egress is a Cloud NAT address allocated AUTO_ONLY — it can
// change without warning. An IP whitelist would work until GCP reallocated it and
// then stop paying anyone, silently. Signing removes that failure mode entirely.
func (c *CashfreePayoutClient) signRequest() (string, error) {
	if c.publicKey == nil {
		return "", nil
	}
	payload := c.clientID + "." + strconv.FormatInt(time.Now().Unix(), 10)
	// RSA-OAEP with SHA-1 (MGF1-SHA1). Verified against the live sandbox, not
	// taken from the documentation — which describes only "RSA encryption" and is
	// widely paraphrased as PKCS#1 v1.5. It is not: v1.5 and OAEP-SHA256 both
	// return `401 Signature mismatch`, and only OAEP-SHA1 authenticates.
	//
	// SHA-1 here is OAEP's mask/label hash, not a signature digest, so this is
	// not a collision-resistance dependency — and the choice is Cashfree's
	// server-side decryption anyway, not ours.
	cipher, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, c.publicKey, []byte(payload), nil)
	if err != nil {
		return "", fmt.Errorf("cashfree-payouts: sign request: %w", err)
	}
	return base64.StdEncoding.EncodeToString(cipher), nil
}

// SignatureConfigured reports whether this client can authenticate without IP
// whitelisting. Surfaced on the admin screen, because "which auth mode am I in"
// is the first question when a payout starts returning 403.
func (c *CashfreePayoutClient) SignatureConfigured() bool { return c.publicKey != nil }

// --- HTTP ---

func (c *CashfreePayoutClient) resolvedBaseURL() string {
	if c.baseURL != "" {
		return c.baseURL
	}
	return cashfreePayoutBaseURLFor(c.mode, c.clientID, c.clientSecret)
}

func (c *CashfreePayoutClient) do(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	reqURL := c.resolvedBaseURL() + path

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, reqURL, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, reqURL, nil)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("cashfree-payouts: build request: %w", err)
	}

	req.Header.Set("x-client-id", c.clientID)
	req.Header.Set("x-client-secret", c.clientSecret)
	req.Header.Set("x-api-version", cashfreePayoutAPIVersion)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Sign when a public key is configured. Without it the call authenticates by
	// source IP alone, which Cashfree Payouts requires to be whitelisted.
	if sig, err := c.signRequest(); err != nil {
		// A configured-but-unusable key is a hard failure, not a silent fallback
		// to IP auth: falling back would work in whichever environment happens
		// to be whitelisted and fail in the other, which is the least debuggable
		// outcome available.
		return nil, 0, err
	} else if sig != "" {
		req.Header.Set("x-cf-signature", sig)
	}

	client := &http.Client{Timeout: cashfreePayoutRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("cashfree-payouts request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("cashfree-payouts: read response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}

// cashfreePayoutErrorDetail extracts only Cashfree's structured message.
//
// Never the raw body: payout validation errors echo back the submitted account
// number and IFSC, and those must not reach a log line, a Sentry event or an
// admin error banner.
func cashfreePayoutErrorDetail(body []byte) string {
	var parsed struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && (parsed.Code != "" || parsed.Message != "") {
		if parsed.Code != "" && parsed.Message != "" {
			return parsed.Code + ": " + parsed.Message
		}
		return parsed.Code + parsed.Message
	}
	return "gateway rejected the request"
}

func cashfreePayoutError(status int, body []byte) error {
	return fmt.Errorf("cashfree-payouts API error (HTTP %d): %s", status, cashfreePayoutErrorDetail(body))
}

// HealthCheck validates credentials with one cheap authenticated call.
//
// Fetches a beneficiary id that will never exist: a 404 proves the request was
// authenticated and routed. Only 401/403 means the credentials are wrong — which
// is precisely what an admin needs to know, and the same probe shape the PG
// client uses.
func (c *CashfreePayoutClient) HealthCheck(ctx context.Context) error {
	resp, status, err := c.do(ctx, http.MethodGet, "/beneficiary?beneficiary_id=hc_healthcheck_absent", nil)
	if err != nil {
		return err
	}
	switch {
	case status == http.StatusNotFound, status == http.StatusOK:
		return nil
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return fmt.Errorf("cashfree payout credentials rejected: %s", cashfreePayoutErrorDetail(resp))
	default:
		return cashfreePayoutError(status, resp)
	}
}
