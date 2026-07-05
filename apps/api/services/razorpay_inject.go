package services

import "time"

// razorpay_inject.go — a narrow, exported seam for injecting a Razorpay client so
// tests in OTHER packages (handlers) can drive the money path (FetchPayment,
// signature verification) against an httptest.Server without a live gateway or GCP
// Secret Manager. Production never calls these — it goes through GetRazorpay's
// Secret-Manager path. The #559 injectable baseURL made the SERVICES-package seams
// testable; this makes them reachable from handler-package tests too (reused by
// #395·4 tip/group verification and #218 sandbox harnessing).

// SetRazorpayClient installs c as the process-wide Razorpay client (nil to clear).
// Test-only: production populates the client via GetRazorpay/Secret Manager. Guarded
// by the same mutex as the cache so it can't race a concurrent GetRazorpay.
func SetRazorpayClient(c *RazorpayClient) {
	razorpayMu.Lock()
	defer razorpayMu.Unlock()
	razorpayClient = c
}

// NewRazorpayTestClient builds a RazorpayClient whose API host is baseURL (an
// httptest.Server) and whose secrets are caller-controlled — so a handler test can
// make FetchPayment return a canned payment and VerifyPaymentSignature validate a
// test-computed HMAC. fetchedAt is stamped now so GetRazorpay serves it within the
// cache TTL rather than re-reading Secret Manager. Test-only.
func NewRazorpayTestClient(baseURL, keyID, keySecret, webhookSecret string) *RazorpayClient {
	return &RazorpayClient{
		keyID:         keyID,
		keySecret:     keySecret,
		webhookSecret: webhookSecret,
		baseURL:       baseURL,
		fetchedAt:     time.Now(),
	}
}
