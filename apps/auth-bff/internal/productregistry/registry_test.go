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
		{"vendors.fe3dr.com", "vendor-portal", "business", "vendor"},
		{"localhost:5174", "vendor-portal", "business", "vendor"},
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
