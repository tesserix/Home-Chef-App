package productregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_ResolveByExactHost(t *testing.T) {
	r, err := Load("../../homechef-products.yaml")
	require.NoError(t, err)

	// Web portals (web/vendor-portal/delivery-portal) were sunset (#22);
	// admin-portal is the remaining OIDC browser app.
	app, err := r.ResolveByHost("admin.fe3dr.com")
	require.NoError(t, err)
	assert.Equal(t, "admin-portal", app.Name)
	assert.Equal(t, "internal", app.AuthContext)
	assert.Equal(t, "admin", app.DefaultRole)
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
	assert.False(t, r.IsMobileTenantAllowed("HomeChef-Internal-gyofe"))
}
