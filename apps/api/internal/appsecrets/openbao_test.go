package appsecrets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestReadAndWriteUseProductIdentifierAndCacheLogin(t *testing.T) {
	logins := 0
	value := "old"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/auth/kubernetes/login" {
			logins++
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body["role"] != "runtime-homechef-api" || body["jwt"] != "jwt" {
				t.Errorf("unexpected login")
			}
			_, _ = w.Write([]byte(`{"auth":{"client_token":"test-token","lease_duration":600}}`))
			return
		}
		if r.Header.Get("X-Vault-Token") != "test-token" {
			t.Error("missing token")
		}
		if r.URL.Path != "/v1/kv/data/homechef/homechef-api/fe3dr-cashfree-app-id" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Method == http.MethodPost {
			var body struct {
				Data map[string]string `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			value = body.Data["value"]
			_, _ = w.Write([]byte(`{"data":{"version":2}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"data": map[string]string{"value": value}}})
	}))
	defer server.Close()
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("jwt"), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := New(server.URL, "runtime-homechef-api", tokenFile)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Read(context.Background(), "prod-homechef-cashfree-app-id")
	if err != nil || got != "old" {
		t.Fatalf("read failed: %v", err)
	}
	if err := client.Write(context.Background(), "prod-homechef-cashfree-app-id", "new\n"); err != nil {
		t.Fatal(err)
	}
	got, err = client.Read(context.Background(), "prod-homechef-cashfree-app-id")
	if err != nil || got != "new\n" || logins != 1 {
		t.Fatalf("round-trip/login cache failed: %v", err)
	}
}

func TestRejectsOtherProductsAndTraversal(t *testing.T) {
	for _, name := range []string{"prod-mark8ly-secret", "prod-homechef-../other", "dev-homechef-cashfree-app-id", "prod-homechef-cashfree-app-id?x"} {
		if _, err := secretPath(name); err == nil {
			t.Errorf("accepted invalid name %s", name)
		}
	}
}

func TestFailuresDoNotLeakValuesOrRetryWrites(t *testing.T) {
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/auth/kubernetes/login" {
			_, _ = w.Write([]byte(`{"auth":{"client_token":"test-token","lease_duration":600}}`))
			return
		}
		writes++
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":["sensitive-response"]}`))
	}))
	defer server.Close()
	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte("jwt"), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := New(server.URL, "runtime-homechef-api", file)
	if err != nil {
		t.Fatal(err)
	}
	err = client.Write(context.Background(), "prod-homechef-cashfree-app-id", "sensitive-value")
	if err == nil || err.Error() != "OpenBao request status 403" || writes != 1 {
		t.Fatalf("unsafe write failure: %v", err)
	}
}

func TestDeleteIsRestrictedAndUsesKVMetadataSemantics(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/auth/kubernetes/login" {
			_, _ = w.Write([]byte(`{"auth":{"client_token":"test-token","lease_duration":600}}`))
			return
		}
		calls++
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/kv/metadata/homechef/homechef-api/fe3dr-vendor-payment-test-bank-ifsc" {
			t.Error("incorrect delete request")
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte("jwt"), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := New(server.URL, "runtime-homechef-api", file)
	if err != nil {
		t.Fatal(err)
	}
	if client.Delete(context.Background(), "prod-homechef-cashfree-app-id") == nil {
		t.Fatal("allowed deleting gateway configuration")
	}
	if err := client.Delete(context.Background(), "prod-homechef-vendor-payment-test-bank-ifsc"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("incorrect request count")
	}
}

func TestUnknownBackendFailsClosed(t *testing.T) {
	t.Setenv("APP_SECRET_STORE", "unknown")
	if _, err := FromEnvironment(); err == nil {
		t.Fatal("unknown backend accepted")
	}
}

func TestPaymentSelectionDoesNotSwitchPII(t *testing.T) {
	t.Setenv("APP_SECRET_STORE", "openbao")
	t.Setenv("OPENBAO_ADDR", "http://openbao.openbao.svc.cluster.local:8200")
	t.Setenv("OPENBAO_ROLE", "runtime-homechef-api")
	t.Setenv("PII_SECRET_STORE", "")
	payment, err := FromEnvironment()
	if err != nil || payment == nil {
		t.Fatalf("payment client not configured: %v", err)
	}
	pii, err := PIIFromEnvironment()
	if err != nil || pii != nil {
		t.Fatalf("payment switch changed PII: %v", err)
	}
}
