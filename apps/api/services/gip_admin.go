package services

// gip_admin.go — the only place this API talks to Google Identity Platform as
// an administrator.
//
// Account deletion has to kill the credential, not just the row: leaving the
// GIP identity alive means the user's password or Google sign-in still works
// after they asked us to delete their account, which fails both the DPDP
// erasure duty and a store reviewer's smoke test.
//
// Identity ownership otherwise lives in auth-bff (a shared image serving other
// products), so this is a deliberately minimal REST client rather than a second
// full identity integration: one call, accounts:delete, tenant-scoped.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2/google"

	"github.com/homechef/api/config"
)

const gipIdentityToolkitBase = "https://identitytoolkit.googleapis.com/v1"

// ErrGIPAccountNotFound means Identity Platform holds no account for that email
// in that tenant. Callers on the password-reset path MUST swallow it and answer
// exactly as they would on success — telling a stranger which addresses are
// registered is an account-enumeration oracle.
var ErrGIPAccountNotFound = errors.New("gip: account not found")

// gipHTTPClient is overridable in tests.
var gipHTTPClient = &http.Client{Timeout: 15 * time.Second}

// DeleteGIPAccount removes the Identity Platform user behind uid, scoped to the
// tenant the account signed in through.
//
// Idempotent by design: a missing account reports success. Deletion is retried
// (post-commit here, and again by the purge sweeper), and by the second attempt
// the account is legitimately already gone — treating that as an error would
// wedge the retry loop forever.
//
// Returns nil without calling out when GIP is not configured, so local and test
// environments are unaffected.
func DeleteGIPAccount(ctx context.Context, tenantID, uid string) error {
	if uid == "" {
		return fmt.Errorf("gip: uid is required")
	}
	projectID := config.AppConfig.GCSProjectID
	if projectID == "" {
		return nil // not configured — nothing to tear down
	}

	url := fmt.Sprintf("%s/projects/%s", gipIdentityToolkitBase, projectID)
	if tenantID != "" {
		url += "/tenants/" + tenantID
	}
	url += "/accounts:delete"

	body, err := json.Marshal(map[string]string{"localId": uid})
	if err != nil {
		return fmt.Errorf("gip: encode delete request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gip: build delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Application Default Credentials — the same workload service account that
	// already reaches Secret Manager and GCS.
	ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("gip: obtain credentials: %w", err)
	}
	tok, err := ts.Token()
	if err != nil {
		return fmt.Errorf("gip: obtain access token: %w", err)
	}
	tok.SetAuthHeader(req)

	resp, err := gipHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("gip: delete account: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	// Already gone is success — see the idempotency note above.
	if strings.Contains(string(payload), "USER_NOT_FOUND") {
		return nil
	}
	return fmt.Errorf("gip: delete account: status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
}

// GenerateGIPPasswordResetLink asks Identity Platform for a password-reset link
// WITHOUT letting it send the email.
//
// Firebase's own delivery is unusable for this product: it sends from
// noreply@<project>.firebaseapp.com, an unauthenticated domain with no SPF/DKIM
// alignment to fe3dr.com, so Gmail files the mail as spam ("previous messages
// from firebaseapp.com were marked as spam"). The message it composes is also
// branded with the GCP project name ("Reset your password for Tesserix") and
// pastes a raw firebaseapp.com URL into the body, which reads as phishing.
//
// So we take only the part Firebase is authoritative for — minting a valid,
// expiring, single-use reset token — and deliver it ourselves through the
// platform mailer that already lands in the inbox.
//
// The returned link is a CREDENTIAL: anyone holding it can set the account's
// password. It must never be logged, echoed in an API response, or attached to
// an error.
func GenerateGIPPasswordResetLink(ctx context.Context, tenantID, email string) (string, error) {
	if email == "" {
		return "", fmt.Errorf("gip: email is required")
	}
	projectID := config.AppConfig.GCSProjectID
	if projectID == "" {
		return "", fmt.Errorf("gip: project not configured")
	}

	url := fmt.Sprintf("%s/projects/%s", gipIdentityToolkitBase, projectID)
	if tenantID != "" {
		url += "/tenants/" + tenantID
	}
	url += "/accounts:sendOobCode"

	// returnOobLink=true is what suppresses Firebase's own email and hands the
	// link back to us instead.
	body, err := json.Marshal(map[string]any{
		"requestType":   "PASSWORD_RESET",
		"email":         email,
		"returnOobLink": true,
	})
	if err != nil {
		return "", fmt.Errorf("gip: encode reset request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("gip: build reset request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	ts, err := google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("gip: obtain credentials: %w", err)
	}
	tok, err := ts.Token()
	if err != nil {
		return "", fmt.Errorf("gip: obtain access token: %w", err)
	}
	tok.SetAuthHeader(req)

	resp, err := gipHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gip: request reset link: %w", err)
	}
	defer resp.Body.Close()

	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if resp.StatusCode != http.StatusOK {
		// EMAIL_NOT_FOUND is not an error the caller should surface — the
		// endpoint above answers identically either way (anti-enumeration).
		if strings.Contains(string(payload), "EMAIL_NOT_FOUND") {
			return "", ErrGIPAccountNotFound
		}
		// The body IS included, deliberately. It was omitted at first out of
		// caution about the reset link leaking into logs — but oobLink is only
		// ever present on a 200, so a failure body cannot carry it. All that
		// caution achieved was hiding the reason for a production 400 behind a
		// bare status code. Identity Platform puts a machine-readable reason
		// here (INVALID_EMAIL, OPERATION_NOT_ALLOWED, …) and it is the only
		// thing that makes this failure diagnosable.
		return "", fmt.Errorf("gip: request reset link: status %d: %s",
			resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var out struct {
		OOBLink string `json:"oobLink"`
	}
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", fmt.Errorf("gip: decode reset link response: %w", err)
	}
	if out.OOBLink == "" {
		return "", fmt.Errorf("gip: reset link missing from response")
	}
	return out.OOBLink, nil
}
