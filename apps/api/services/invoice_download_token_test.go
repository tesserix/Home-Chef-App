package services

// invoice_download_token_test.go — the signed receipt-download link is a
// capability URL: whoever holds it can pull that one receipt. So it must be
// unforgeable, single-purpose, order+user scoped, and short-lived. These pin all
// of that.

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
)

func withInvoiceKey(t *testing.T, key string) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig = &config.Config{BFFInternalHMACKey: []byte(key)}
	t.Cleanup(func() { config.AppConfig = prev })
}

func TestInvoiceToken_RoundTrips(t *testing.T) {
	withInvoiceKey(t, "test-hmac-key")
	order, user := uuid.New(), uuid.New()

	tok, err := MintInvoiceDownloadToken(order, user)
	require.NoError(t, err)

	gotOrder, gotUser, err := VerifyInvoiceDownloadToken(tok)
	require.NoError(t, err)
	require.Equal(t, order, gotOrder)
	require.Equal(t, user, gotUser, "the token carries WHO may download, so the route can scope to them")
}

func TestInvoiceToken_TamperedSignatureRejected(t *testing.T) {
	withInvoiceKey(t, "test-hmac-key")
	tok, _ := MintInvoiceDownloadToken(uuid.New(), uuid.New())

	// Flip the last char of the signature.
	bad := tok[:len(tok)-1] + string(rune(tok[len(tok)-1])+1)
	_, _, err := VerifyInvoiceDownloadToken(bad)
	require.Error(t, err, "a tampered signature must never verify")
}

// A token signed with a different key must not verify — the signature is the
// only thing standing between a URL and someone else's receipt.
func TestInvoiceToken_WrongKeyRejected(t *testing.T) {
	order, user := uuid.New(), uuid.New()
	withInvoiceKey(t, "key-A")
	tok, _ := MintInvoiceDownloadToken(order, user)

	withInvoiceKey(t, "key-B")
	_, _, err := VerifyInvoiceDownloadToken(tok)
	require.Error(t, err)
}

func TestInvoiceToken_ExpiredRejected(t *testing.T) {
	withInvoiceKey(t, "test-hmac-key")
	order, user := uuid.New(), uuid.New()

	// Hand-build a token that expired a minute ago, signed correctly.
	payload := "invoice_download:" + order.String() + ":" + user.String() + ":" +
		itoa(time.Now().Add(-time.Minute).Unix())
	tok := encode(payload) + "." + signInvoicePayload(payload, []byte("test-hmac-key"))

	_, _, err := VerifyInvoiceDownloadToken(tok)
	require.ErrorContains(t, err, "expired", "a stale link must stop working")
}

// A validly-signed token for a DIFFERENT purpose must not be accepted as an
// invoice-download token, so a signature reused across features can't cross over.
func TestInvoiceToken_WrongPurposeRejected(t *testing.T) {
	withInvoiceKey(t, "test-hmac-key")
	order, user := uuid.New(), uuid.New()
	payload := "something_else:" + order.String() + ":" + user.String() + ":" +
		itoa(time.Now().Add(time.Hour).Unix())
	tok := encode(payload) + "." + signInvoicePayload(payload, []byte("test-hmac-key"))

	_, _, err := VerifyInvoiceDownloadToken(tok)
	require.ErrorContains(t, err, "purpose")
}

func TestInvoiceToken_MalformedRejected(t *testing.T) {
	withInvoiceKey(t, "test-hmac-key")
	for _, bad := range []string{"", "nodot", "a.b.c.d.e", "...."} {
		_, _, err := VerifyInvoiceDownloadToken(bad)
		require.Error(t, err, "malformed token %q must be refused, not panic", bad)
	}
}

func TestInvoiceToken_NoKeyConfigured_Errors(t *testing.T) {
	prev := config.AppConfig
	config.AppConfig = &config.Config{BFFInternalHMACKey: nil}
	t.Cleanup(func() { config.AppConfig = prev })

	_, err := MintInvoiceDownloadToken(uuid.New(), uuid.New())
	require.Error(t, err, "refuse to mint an unsigned token rather than mint a forgeable one")
}

// small local helpers so the test doesn't pull extra imports
func itoa(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func encode(s string) string {
	// mirror MintInvoiceDownloadToken's base64url of the payload
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
