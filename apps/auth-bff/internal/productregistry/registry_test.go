package productregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_ResolveByExactHost(t *testing.T) {
	r, err := Load("../../homechef-products.yaml")
	require.NoError(t, err)

	app, err := r.ResolveByHost("fe3dr.com")
	require.NoError(t, err)
	assert.Equal(t, "web", app.Name)
	assert.Equal(t, "customer", app.AuthContext)
	assert.Equal(t, "customer", app.DefaultRole)
}

func TestRegistry_ResolveByLocalhost(t *testing.T) {
	r, _ := Load("../../homechef-products.yaml")
	app, err := r.ResolveByHost("localhost:5174")
	require.NoError(t, err)
	assert.Equal(t, "vendor-portal", app.Name)
	assert.Equal(t, "vendor", app.DefaultRole)
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
