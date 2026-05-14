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
	Role        string `json:"role"`
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
