package autologin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/gip"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
)

// GIPVerifier is the surface of *gip.Verifier we need. Defined as interface for testability.
type GIPVerifier interface {
	Verify(ctx context.Context, raw string, expectedTenantID string) (*gip.VerifiedToken, error)
}

// APIClient is the surface of *apiclient.Client we need.
type APIClient interface {
	UpsertUser(ctx context.Context, req apiclient.UpsertUserRequest) (*apiclient.UpsertUserResponse, error)
}

// SessionManager is the surface of *session.Manager we need.
type SessionManager interface {
	Encode(*session.Payload) (string, error)
	MaxAge() time.Duration
}

type Deps struct {
	GIP      GIPVerifier
	Sessions SessionManager
	Registry *productregistry.Registry
	API      APIClient
}

type Request struct {
	IDToken          string `json:"id_token" binding:"required"`
	ExpectedTenantID string `json:"expected_tenant_id" binding:"required"`
}

type Response struct {
	SessionToken string `json:"session_token"`
	ExpiresAt    int64  `json:"expires_at"`
	User         User   `json:"user"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
	Pool  string `json:"pool"`
}

var (
	ErrTenantNotAllowed = errors.New("tenant not allowed for mobile")
	ErrTokenInvalid     = errors.New("id_token invalid")
)

func (d *Deps) AutoLogin(ctx context.Context, req Request) (*Response, error) {
	if !d.Registry.IsMobileTenantAllowed(req.ExpectedTenantID) {
		return nil, ErrTenantNotAllowed
	}
	tok, err := d.GIP.Verify(ctx, req.IDToken, req.ExpectedTenantID)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	pool := poolFromTenant(req.ExpectedTenantID)
	role := defaultRoleForPool(pool)
	if r, ok := tok.Claims["role"].(string); ok && r != "" {
		role = r
	}
	upsert, err := d.API.UpsertUser(ctx, apiclient.UpsertUserRequest{
		GIPUid:      tok.UID,
		GIPTenantID: tok.TenantID,
		GIPProvider: tok.Provider,
		AuthPool:    pool,
		Email:       tok.Email,
		Role:        role,
	})
	if err != nil {
		return nil, err
	}
	now := time.Now()
	exp := now.Add(d.Sessions.MaxAge())
	payload := &session.Payload{
		UID:       upsert.UserID,
		Email:     tok.Email,
		Pool:      pool,
		Role:      role,
		IssuedAt:  now.Unix(),
		ExpiresAt: exp.Unix(),
	}
	enc, err := d.Sessions.Encode(payload)
	if err != nil {
		return nil, err
	}
	return &Response{
		SessionToken: enc,
		ExpiresAt:    exp.Unix(),
		User: User{
			ID:    upsert.UserID,
			Email: tok.Email,
			Role:  role,
			Pool:  pool,
		},
	}, nil
}

// poolFromTenant extracts the pool name from tenant IDs of the form
// "HomeChef-{Pool}-{suffix}" and lowercases it.
func poolFromTenant(tenantID string) string {
	parts := strings.SplitN(tenantID, "-", 3)
	if len(parts) < 2 {
		return ""
	}
	return strings.ToLower(parts[1])
}

func defaultRoleForPool(pool string) string {
	switch pool {
	case "customer":
		return "customer"
	case "business":
		// Business pool default. The Go API's UserRole enum uses "chef"
		// (the catering operator) rather than "vendor"; a request can
		// still override via the role claim (e.g., "delivery").
		return "chef"
	case "internal":
		return "admin"
	default:
		return ""
	}
}
