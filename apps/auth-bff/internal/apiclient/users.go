package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/homechef/auth-bff/internal/headerproxy"
)

type UpsertUserRequest struct {
	GIPUid      string `json:"gip_uid"`
	GIPTenantID string `json:"gip_tenant_id"`
	GIPProvider string `json:"gip_provider"`
	AuthPool    string `json:"auth_pool"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	// Avatar is the profile picture URL derived from the GIP token's "picture"
	// claim. apps/api backfills it only when the stored avatar is empty.
	Avatar string `json:"avatar"`
	// EmailVerified forwards the GIP token's email_verified claim so apps/api
	// can gate same-email account re-bind on a verified identity.
	EmailVerified bool   `json:"email_verified"`
	Role          string `json:"role"`
	// MarketingConsent forwards the DPDP §6 opt-in collected at registration
	// to apps/api. Forwarded as a JSON boolean; omitting it (e.g., from the
	// social-login / OIDC callback path) defaults to false on the API side.
	MarketingConsent bool `json:"marketing_consent"`
	// Device metadata for the install this sign-in came from, so apps/api can
	// register it and warn the owner about one they haven't used before
	// (#1164). All optional — older clients send none of it.
	DeviceID    string `json:"device_id,omitempty"`
	Platform    string `json:"platform,omitempty"`
	DeviceLabel string `json:"device_label,omitempty"`
	AppVersion  string `json:"app_version,omitempty"`
	// IP is the caller's address as seen by the BFF, never a client-supplied value.
	IP string `json:"ip,omitempty"`
}

type UpsertUserResponse struct {
	UserID string `json:"user_id"`
}

type Client struct {
	base   string
	signer *headerproxy.Signer
	http   *http.Client
}

func New(baseURL string, signer *headerproxy.Signer) *Client {
	return &Client{
		base:   baseURL,
		signer: signer,
		http:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) UpsertUser(ctx context.Context, req UpsertUserRequest) (*UpsertUserResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/internal/users/upsert", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	if err := c.signer.Sign(r, body, headerproxy.Identity{
		UserID: req.GIPUid, Email: req.Email, Role: req.Role, Pool: req.AuthPool,
	}); err != nil {
		return nil, err
	}
	resp, err := c.http.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upsert: %d %s", resp.StatusCode, b)
	}
	var out UpsertUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
