package autologin

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/gip"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeGIP struct {
	tok *gip.VerifiedToken
	err error
}

func (f *fakeGIP) Verify(ctx context.Context, raw, tenant string) (*gip.VerifiedToken, error) {
	return f.tok, f.err
}

type fakeAPI struct {
	resp *apiclient.UpsertUserResponse
	err  error
}

func (f *fakeAPI) UpsertUser(ctx context.Context, req apiclient.UpsertUserRequest) (*apiclient.UpsertUserResponse, error) {
	return f.resp, f.err
}

type fakeSessions struct {
	encoded string
	err     error
}

func (f *fakeSessions) Encode(p *session.Payload) (string, error) { return f.encoded, f.err }
func (f *fakeSessions) MaxAge() time.Duration                     { return time.Hour }

func newDeps(t *testing.T, gipResult *fakeGIP, apiResult *fakeAPI, sessResult *fakeSessions) *Deps {
	t.Helper()
	reg, err := productregistry.Load("../../homechef-products.yaml")
	require.NoError(t, err)
	return &Deps{
		GIP:      gipResult,
		API:      apiResult,
		Sessions: sessResult,
		Registry: reg,
	}
}

func TestAutoLogin_Happy(t *testing.T) {
	deps := newDeps(t,
		&fakeGIP{tok: &gip.VerifiedToken{UID: "g1", Email: "x@y.com", TenantID: "HomeChef-Customer-xxxxx", Provider: "google.com", Claims: map[string]any{}}},
		&fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}},
		&fakeSessions{encoded: "sess-abc"},
	)
	r := gin.New()
	NewHandler(deps).Register(r)

	body := `{"id_token":"valid.test.token","expected_tenant_id":"HomeChef-Customer-xxxxx"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"session_token":"sess-abc"`)
	assert.Contains(t, w.Body.String(), `"id":"u1"`)
}

func TestAutoLogin_TenantNotAllowedForMobile_403(t *testing.T) {
	deps := newDeps(t,
		&fakeGIP{tok: &gip.VerifiedToken{}},
		&fakeAPI{},
		&fakeSessions{},
	)
	r := gin.New()
	NewHandler(deps).Register(r)
	body := `{"id_token":"t","expected_tenant_id":"HomeChef-Internal-xxxxx"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 403, w.Code)
	assert.Contains(t, w.Body.String(), "tenant_not_allowed")
}

func TestAutoLogin_TokenInvalid_401(t *testing.T) {
	deps := newDeps(t,
		&fakeGIP{err: errors.New("bad token")},
		&fakeAPI{},
		&fakeSessions{},
	)
	r := gin.New()
	NewHandler(deps).Register(r)
	body := `{"id_token":"bad","expected_tenant_id":"HomeChef-Customer-xxxxx"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_token")
}

func TestAutoLogin_RoleClaim_Overrides(t *testing.T) {
	deps := newDeps(t,
		&fakeGIP{tok: &gip.VerifiedToken{
			UID: "g1", Email: "x@y.com", TenantID: "HomeChef-Business-xxxxx", Provider: "google.com",
			Claims: map[string]any{"role": "driver"},
		}},
		&fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}},
		&fakeSessions{encoded: "sess"},
	)
	r := gin.New()
	NewHandler(deps).Register(r)
	body := `{"id_token":"t","expected_tenant_id":"HomeChef-Business-xxxxx"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"role":"driver"`)
}

func TestAutoLogin_InvalidBody_400(t *testing.T) {
	deps := newDeps(t, &fakeGIP{}, &fakeAPI{}, &fakeSessions{})
	r := gin.New()
	NewHandler(deps).Register(r)
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 400, w.Code)
}
