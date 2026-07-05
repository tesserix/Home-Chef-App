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
	resp     *apiclient.UpsertUserResponse
	err      error
	lastReq  apiclient.UpsertUserRequest
	captured bool
}

func (f *fakeAPI) UpsertUser(ctx context.Context, req apiclient.UpsertUserRequest) (*apiclient.UpsertUserResponse, error) {
	f.lastReq = req
	f.captured = true
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
		&fakeGIP{tok: &gip.VerifiedToken{UID: "g1", Email: "x@y.com", TenantID: "HomeChef-Customer-rqg8a", Provider: "google.com", Claims: map[string]any{}}},
		&fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}},
		&fakeSessions{encoded: "sess-abc"},
	)
	r := gin.New()
	NewHandler(deps).Register(r)

	body := `{"id_token":"valid.test.token","expected_tenant_id":"HomeChef-Customer-rqg8a"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"session_token":"sess-abc"`)
	assert.Contains(t, w.Body.String(), `"id":"u1"`)
}

func TestAutoLogin_ForwardsNameAvatarEmailVerified(t *testing.T) {
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	deps := newDeps(t,
		&fakeGIP{tok: &gip.VerifiedToken{
			UID: "g1", Email: "x@y.com", TenantID: "HomeChef-Customer-rqg8a", Provider: "google.com",
			Name: "Ada Lovelace", Picture: "https://example.com/ada.png", EmailVerified: true,
			Claims: map[string]any{},
		}},
		api,
		&fakeSessions{encoded: "sess-abc"},
	)
	r := gin.New()
	NewHandler(deps).Register(r)

	body := `{"id_token":"valid.test.token","expected_tenant_id":"HomeChef-Customer-rqg8a"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	require.True(t, api.captured)
	assert.Equal(t, "Ada Lovelace", api.lastReq.Name)
	assert.Equal(t, "https://example.com/ada.png", api.lastReq.Avatar)
	assert.True(t, api.lastReq.EmailVerified)
}

func TestAutoLogin_TenantNotAllowedForMobile_403(t *testing.T) {
	deps := newDeps(t,
		&fakeGIP{tok: &gip.VerifiedToken{}},
		&fakeAPI{},
		&fakeSessions{},
	)
	r := gin.New()
	NewHandler(deps).Register(r)
	body := `{"id_token":"t","expected_tenant_id":"HomeChef-Unknown-zzzzz"}`
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
	body := `{"id_token":"bad","expected_tenant_id":"HomeChef-Customer-rqg8a"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_token")
}

func TestAutoLogin_RoleClaim_Overrides(t *testing.T) {
	// The Go API's UserRole enum has no "driver" — drivers map to "delivery".
	// Assert the claim override flows through with a value the API accepts.
	deps := newDeps(t,
		&fakeGIP{tok: &gip.VerifiedToken{
			UID: "g1", Email: "x@y.com", TenantID: "HomeChef-Business-8s8ql", Provider: "google.com",
			Claims: map[string]any{"role": "delivery"},
		}},
		&fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}},
		&fakeSessions{encoded: "sess"},
	)
	r := gin.New()
	NewHandler(deps).Register(r)
	body := `{"id_token":"t","expected_tenant_id":"HomeChef-Business-8s8ql"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), `"role":"delivery"`)
}

func TestAutoLogin_AdminEmailNotInAllowlist_403(t *testing.T) {
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "allowed@fe3dr.com")
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	deps := newDeps(t,
		&fakeGIP{tok: &gip.VerifiedToken{UID: "g1", Email: "intruder@evil.com", TenantID: "HomeChef-Internal-gyofe", Provider: "password", Claims: map[string]any{}}},
		api,
		&fakeSessions{encoded: "sess"},
	)
	r := gin.New()
	NewHandler(deps).Register(r)
	body := `{"id_token":"t","expected_tenant_id":"HomeChef-Internal-gyofe"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 403, w.Code)
	assert.Contains(t, w.Body.String(), "email_not_allowed")
	// A rejected admin must never be upserted.
	require.False(t, api.captured)
}

func TestAutoLogin_AdminAllowlistUnset_Denied(t *testing.T) {
	// Fail-closed: an unconfigured allowlist must DENY admin login, not allow it.
	// With the k8s secret mounted optional and the Istio mesh not stripping
	// X-User-* headers, a missing allowlist would otherwise grant admin to any
	// verified email.
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "")
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	deps := newDeps(t,
		&fakeGIP{tok: &gip.VerifiedToken{UID: "g1", Email: "admin@fe3dr.com", TenantID: "HomeChef-Internal-gyofe", Provider: "password", Claims: map[string]any{}}},
		api,
		&fakeSessions{encoded: "sess"},
	)
	r := gin.New()
	NewHandler(deps).Register(r)
	body := `{"id_token":"t","expected_tenant_id":"HomeChef-Internal-gyofe"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "email_not_allowed")
	require.False(t, api.captured, "a denied admin must never be upserted")
}

// A non-admin (customer/business/delivery) pool must be unaffected by the admin
// allowlist — the fail-closed rule applies only to internal/admin logins.
func TestAutoLogin_NonAdminUnaffectedByAllowlist(t *testing.T) {
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "")
	deps := newDeps(t,
		&fakeGIP{tok: &gip.VerifiedToken{UID: "g1", Email: "cust@fe3dr.com", TenantID: "HomeChef-Customer-rqg8a", Provider: "password", Claims: map[string]any{}}},
		&fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}},
		&fakeSessions{encoded: "sess"},
	)
	r := gin.New()
	NewHandler(deps).Register(r)
	body := `{"id_token":"t","expected_tenant_id":"HomeChef-Customer-rqg8a"}`
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code, w.Body.String())
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
