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
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2/google"

	"github.com/homechef/api/config"
)

const gipIdentityToolkitBase = "https://identitytoolkit.googleapis.com/v1"

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
