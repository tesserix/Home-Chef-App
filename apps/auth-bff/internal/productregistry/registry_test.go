package productregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_ResolveByExactHost(t *testing.T) {
	r, err := Load("../../homechef-products.yaml")
	require.NoError(t, err)

	app, err := r.ResolveByHost("admin.fe3dr.com")
	require.NoError(t, err)
	assert.Equal(t, "admin-portal", app.Name)
	assert.Equal(t, "internal", app.AuthContext)
	assert.Equal(t, "admin", app.DefaultRole)
}

// The customer and vendor web hosts must resolve. They were dropped in #22
// when the portals were sunset, which took production logins down with them:
// GIP sign-in succeeded and then /auth/exchange answered unknown_host. These
// assertions fail if an entry is removed again.
func TestRegistry_ResolveWebPortalHosts(t *testing.T) {
	r, err := Load("../../homechef-products.yaml")
	require.NoError(t, err)

	for _, tc := range []struct {
		host        string
		wantApp     string
		wantContext string
		wantRole    string
	}{
		{"fe3dr.com", "web", "customer", "customer"},
		{"www.fe3dr.com", "web", "customer", "customer"},
		{"localhost:5173", "web", "customer", "customer"},
		// role is "chef", not "vendor" — the API's UserRole enum has no vendor
		// member and RequireChef() gates every /chef/* route.
		{"vendors.fe3dr.com", "vendor-portal", "business", "chef"},
		{"localhost:5174", "vendor-portal", "business", "chef"},
	} {
		t.Run(tc.host, func(t *testing.T) {
			app, err := r.ResolveByHost(tc.host)
			require.NoError(t, err)
			assert.Equal(t, tc.wantApp, app.Name)
			assert.Equal(t, tc.wantContext, app.AuthContext)
			assert.Equal(t, tc.wantRole, app.DefaultRole)
		})
	}
}

func TestRegistry_ResolveByLocalhost(t *testing.T) {
	r, _ := Load("../../homechef-products.yaml")
	app, err := r.ResolveByHost("localhost:5176")
	require.NoError(t, err)
	assert.Equal(t, "admin-portal", app.Name)
	assert.Equal(t, "admin", app.DefaultRole)
}

func TestRegistry_UnknownHost_Errors(t *testing.T) {
	r, _ := Load("../../homechef-products.yaml")
	_, err := r.ResolveByHost("attacker.example.com")
	assert.ErrorIs(t, err, ErrUnknownHost)
}

// SessionCookieForHost is the seam session.Handler and apiproxy.Deps use to
// isolate each portal's session cookie on the shared .fe3dr.com cookie
// domain (see homechef-products.yaml's sessionCookie field). This is a
// direct regression test for the cross-app cookie collision: web, vendor,
// and admin must each resolve to their own distinct cookie name.
func TestRegistry_SessionCookieForHost(t *testing.T) {
	r, err := Load("../../homechef-products.yaml")
	require.NoError(t, err)

	for _, tc := range []struct {
		host       string
		wantCookie string
	}{
		{"fe3dr.com", "hc_session"},
		{"www.fe3dr.com", "hc_session"},
		{"vendors.fe3dr.com", "hc_vendor_session"},
		{"admin.fe3dr.com", "hc_admin_session"},
	} {
		t.Run(tc.host, func(t *testing.T) {
			assert.Equal(t, tc.wantCookie, r.SessionCookieForHost(tc.host))
		})
	}
}

// An unmatched host returns "" (not an error, not a made-up name) so callers
// can apply their own default rather than the registry hardcoding one.
func TestRegistry_SessionCookieForHost_UnknownHost_ReturnsEmpty(t *testing.T) {
	r, _ := Load("../../homechef-products.yaml")
	assert.Equal(t, "", r.SessionCookieForHost("attacker.example.com"))
}

func TestRegistry_MobileTenantAllowlist(t *testing.T) {
	r, _ := Load("../../homechef-products.yaml")
	assert.True(t, r.IsMobileTenantAllowed("HomeChef-Customer-rqg8a"))
	assert.True(t, r.IsMobileTenantAllowed("HomeChef-Business-8s8ql"))
	// Internal/admin pool is mobile-allowed too — it powers apps/mobile-admin
	// (the internal pool resolves to role=admin in autologin).
	assert.True(t, r.IsMobileTenantAllowed("HomeChef-Internal-gyofe"))
	// A tenant not in the allowlist is still rejected.
	assert.False(t, r.IsMobileTenantAllowed("HomeChef-Unknown-zzzzz"))
}

// Every app's defaultRole must be a member of the Go API's UserRole enum
// (models.UserRole: customer|chef|delivery|admin|fleet_manager). A role
// outside it mints a session that authenticates but then 403s
// "Insufficient permissions" on every RBAC-gated route — which is exactly
// what `defaultRole: vendor` did to the chef portal.
func TestRegistry_DefaultRolesAreValidAPIRoles(t *testing.T) {
	r, err := Load("../../homechef-products.yaml")
	require.NoError(t, err)

	valid := map[string]bool{
		"customer": true, "chef": true, "delivery": true,
		"admin": true, "fleet_manager": true,
	}

	for _, host := range []string{"fe3dr.com", "vendors.fe3dr.com", "admin.fe3dr.com"} {
		app, err := r.ResolveByHost(host)
		require.NoError(t, err, host)
		assert.True(t, valid[app.DefaultRole],
			"%s: defaultRole %q is not a models.UserRole", host, app.DefaultRole)
	}
}
