package services

// invoice_download_token.go — short-lived signed links for the receipt PDF
// (#receipt mobile parity).
//
// The mobile app can't add expo-file-system/expo-sharing (the workspace lockfile
// can't be regenerated locally), so it can't save an authenticated PDF blob the
// way the web app does. Instead it opens the PDF in the in-app browser — but the
// invoice endpoint needs auth the browser can't send (a Bearer header). This
// mints a token the app fetches (authenticated) and puts in the URL, so the
// browser-opened link authenticates itself.
//
// The token is deliberately narrow: one order, one user, one purpose, minutes of
// life, HMAC-signed with the same server secret as BFF auth. It is NOT a general
// session — it can only fetch that one receipt, and only briefly.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
)

// invoiceTokenTTL — how long a download link is valid. Long enough to tap through
// from the app to the browser, short enough that a leaked URL is near-useless.
const invoiceTokenTTL = 10 * time.Minute

// invoiceTokenPurpose namespaces the signature so a token minted here can never
// be replayed against some other HMAC-signed feature.
const invoiceTokenPurpose = "invoice_download"

func invoiceTokenKey() []byte {
	if config.AppConfig != nil {
		return config.AppConfig.BFFInternalHMACKey
	}
	return nil
}

// MintInvoiceDownloadToken returns a signed token authorising `userID` to
// download `orderID`'s receipt for the next invoiceTokenTTL.
func MintInvoiceDownloadToken(orderID, userID uuid.UUID) (string, error) {
	key := invoiceTokenKey()
	if len(key) == 0 {
		return "", fmt.Errorf("invoice token: signing key not configured")
	}
	exp := time.Now().Add(invoiceTokenTTL).Unix()
	payload := fmt.Sprintf("%s:%s:%s:%d", invoiceTokenPurpose, orderID, userID, exp)
	sig := signInvoicePayload(payload, key)
	// payload is base64url'd so ':' and the raw bytes survive a URL path segment.
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig, nil
}

// VerifyInvoiceDownloadToken checks a token and returns the (orderID, userID) it
// authorises. A tampered, expired, wrong-purpose, or malformed token is refused.
func VerifyInvoiceDownloadToken(token string) (orderID, userID uuid.UUID, err error) {
	key := invoiceTokenKey()
	if len(key) == 0 {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invoice token: signing key not configured")
	}
	dot := strings.LastIndexByte(token, '.')
	if dot <= 0 {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invoice token: malformed")
	}
	encoded, sig := token[:dot], token[dot+1:]
	raw, decErr := base64.RawURLEncoding.DecodeString(encoded)
	if decErr != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invoice token: malformed payload")
	}
	payload := string(raw)

	// Constant-time signature check BEFORE trusting any field in the payload.
	if !hmac.Equal([]byte(sig), []byte(signInvoicePayload(payload, key))) {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invoice token: bad signature")
	}

	parts := strings.Split(payload, ":")
	if len(parts) != 4 || parts[0] != invoiceTokenPurpose {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invoice token: wrong purpose")
	}
	oid, oErr := uuid.Parse(parts[1])
	uid, uErr := uuid.Parse(parts[2])
	exp, eErr := strconv.ParseInt(parts[3], 10, 64)
	if oErr != nil || uErr != nil || eErr != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invoice token: unparseable")
	}
	if time.Now().Unix() > exp {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invoice token: expired")
	}
	return oid, uid, nil
}

func signInvoicePayload(payload string, key []byte) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
