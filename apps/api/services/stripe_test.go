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

func TestStripeRefundRejectsMissingIdempotencyBeforeSending(t *testing.T) {
	calls := 0
	client := &StripeClient{httpClient: &http.Client{Transport: stripeTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"re_test","status":"succeeded"}`)), Header: http.Header{}}, nil
	})}}
	_, err := client.CreateRefund(&StripeRefundRequest{PaymentIntent: "pi_test", Amount: 500})
	require.Error(t, err)
	require.Zero(t, calls)
}

func TestStripeRefundSendsStableKeyAndReversesBothShares(t *testing.T) {
	for _, currency := range []string{"aud", "nzd"} {
		t.Run(currency, func(t *testing.T) {
			calls := 0
			client := &StripeClient{httpClient: &http.Client{Transport: stripeTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, "test-refund", r.Header.Get("Idempotency-Key"))
				require.Equal(t, "/v1/refunds", r.URL.Path)
				require.NoError(t, r.ParseForm())
				require.Equal(t, "500", r.Form.Get("amount"))
				require.Equal(t, "pi_test", r.Form.Get("payment_intent"))
				require.Equal(t, "true", r.Form.Get("reverse_transfer"))
				require.Equal(t, "true", r.Form.Get("refund_application_fee"))
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"re_test","status":"succeeded","amount":500,"currency":"` + currency + `"}`)), Header: http.Header{}}, nil
			})}}
			req := &StripeRefundRequest{PaymentIntent: "pi_test", Amount: 500, IdempotencyKey: "test-refund", ReverseTransfer: true, RefundApplicationFee: true}
			for i := 0; i < 2; i++ {
				result, err := client.CreateRefund(req)
				require.NoError(t, err)
				require.Equal(t, 500, result.Amount)
				require.Equal(t, currency, result.Currency)
			}
			require.Equal(t, 2, calls)
		})
	}
}

func TestStripeDestinationChargeUsesVendorAsSettlementMerchant(t *testing.T) {
	for _, destination := range []string{"acct_nz_vendor", "acct_au_vendor", ""} {
		t.Run(destination, func(t *testing.T) {
			client := &StripeClient{secretKey: "sk_test_fixture", httpClient: &http.Client{Transport: stripeTransport(func(r *http.Request) (*http.Response, error) {
				require.NoError(t, r.ParseForm())
				require.Equal(t, destination, r.Form.Get("transfer_data[destination]"))
				require.Equal(t, destination, r.Form.Get("on_behalf_of"))
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"pi_test","status":"requires_payment_method"}`)), Header: http.Header{}}, nil
			})}}
			_, err := client.CreatePaymentIntent(t.Context(), &StripePaymentIntentRequest{Amount: 2062, Currency: "NZD", DestinationAccount: destination, IdempotencyKey: "order-test"})
			require.NoError(t, err)
		})
	}
}

func TestStripeSettlementRepairUsesSameIntentAndStableIdempotency(t *testing.T) {
	client := &StripeClient{secretKey: "sk_test_fixture", httpClient: &http.Client{Transport: stripeTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/payment_intents/pi_saved", r.URL.Path)
		require.Equal(t, "fe3dr-settlement-pi_saved-acct_nz_vendor", r.Header.Get("Idempotency-Key"))
		require.NoError(t, r.ParseForm())
		require.Equal(t, "acct_nz_vendor", r.Form.Get("on_behalf_of"))
		require.Len(t, r.Form, 1)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"pi_saved","on_behalf_of":"acct_nz_vendor","status":"requires_payment_method"}`)), Header: http.Header{}}, nil
	})}}
	for i := 0; i < 2; i++ {
		pi, err := client.SetPaymentIntentSettlement(t.Context(), "pi_saved", "acct_nz_vendor")
		require.NoError(t, err)
		require.Equal(t, "pi_saved", pi.ID)
		require.Equal(t, "acct_nz_vendor", pi.OnBehalfOf)
	}
}
