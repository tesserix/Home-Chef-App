package apiclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/homechef/auth-bff/internal/headerproxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: read request body and put it back so subsequent readers (verifier) see it too
func readBody(t *testing.T, r *http.Request) []byte {
	t.Helper()
	if r.Body == nil {
		return nil
	}
	b, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	return b
}

func TestClient_UpsertUser_ForwardsIdentityWithHMAC(t *testing.T) {
	key := []byte("test-key-32-bytes-padding-padding!")
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: key, Window: time.Minute})

	var received UpsertUserRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := readBody(t, r)
		_, err := signer.Verify(r, body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &received))
		_ = json.NewEncoder(w).Encode(UpsertUserResponse{UserID: "u1"})
	}))
	defer srv.Close()

	c := New(srv.URL, signer)
	resp, err := c.UpsertUser(context.Background(), UpsertUserRequest{
		GIPUid: "gip-u", GIPTenantID: "HomeChef-Customer", GIPProvider: "google.com",
		AuthPool: "customer", Email: "a@b.com", Name: "A", Role: "customer",
	})
	require.NoError(t, err)
	assert.Equal(t, "u1", resp.UserID)
	assert.Equal(t, "gip-u", received.GIPUid)
	assert.Equal(t, "HomeChef-Customer", received.GIPTenantID)
	assert.Equal(t, "customer", received.AuthPool)
}

func TestClient_UpsertUser_NonOKResponse_Errors(t *testing.T) {
	key := []byte("test-key-32-bytes-padding-padding!")
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: key, Window: time.Minute})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	c := New(srv.URL, signer)
	_, err := c.UpsertUser(context.Background(), UpsertUserRequest{GIPUid: "x", Email: "x@x.com", Role: "customer", AuthPool: "customer"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
}

func TestClient_UpsertUser_NetworkError_Errors(t *testing.T) {
	key := []byte("test-key-32-bytes-padding-padding!")
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: key, Window: time.Minute})

	c := New("http://127.0.0.1:1", signer) // port 1 is unreachable
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := c.UpsertUser(ctx, UpsertUserRequest{GIPUid: "x", Email: "x@x.com", Role: "customer", AuthPool: "customer"})
	require.Error(t, err)
}
