package services

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type stripeTransport func(*http.Request) (*http.Response, error)

func (f stripeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestStripeIntentCreateUsesIdempotencyAndRequestContext(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "request")
	client := &StripeClient{secretKey: "sk_test_fixture", httpClient: &http.Client{Transport: stripeTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "request", r.Context().Value(contextKey{}))
		require.Equal(t, "order-123", r.Header.Get("Idempotency-Key"))
		require.Equal(t, "/v1/payment_intents", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "aud", r.Form.Get("currency"))
		require.Equal(t, "5000", r.Form.Get("amount"))
		require.Equal(t, "acct_vendor", r.Form.Get("transfer_data[destination]"))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"pi_test","amount":5000,"currency":"aud","status":"requires_payment_method","client_secret":"fixture"}`)), Header: http.Header{}}, nil
	})}}
	intent, err := client.CreatePaymentIntent(ctx, &StripePaymentIntentRequest{Amount: 5000, Currency: "AUD", DestinationAccount: "acct_vendor", IdempotencyKey: "order-123"})
	require.NoError(t, err)
	require.Equal(t, "pi_test", intent.ID)
}

func TestStripeErrorsDoNotExposeProviderResponseBody(t *testing.T) {
	client := &StripeClient{httpClient: &http.Client{Transport: stripeTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"private payment details","client_secret":"do_not_log"}}`)), Header: http.Header{}}, nil
	})}}
	_, err := client.FetchPaymentIntent(t.Context(), "pi_test")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "private payment details")
	require.NotContains(t, err.Error(), "do_not_log")
	require.Contains(t, err.Error(), "400")
}

func TestStripeKeyIDNeverReturnsSecretMaterial(t *testing.T) {
	client := &StripeClient{secretKey: "sk_test_do_not_expose"}
	require.Empty(t, client.GetSecretKeyID())
}
