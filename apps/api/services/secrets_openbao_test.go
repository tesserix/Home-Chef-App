package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/homechef/api/internal/appsecrets"
)

func TestPlatformSecretOperationsUseConfiguredOpenBaoBackend(t *testing.T) {
	value := "initial"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/auth/kubernetes/login" {
			_, _ = w.Write([]byte(`{"auth":{"client_token":"test-token","lease_duration":600}}`))
			return
		}
		if r.URL.Path != "/v1/kv/data/homechef/homechef-api/fe3dr-cashfree-app-id" {
			t.Errorf("wrong path: %s", r.URL.Path)
		}
		if r.Method == http.MethodPost {
			var body struct {
				Data map[string]string `json:"data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			value = body.Data["value"]
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"data": map[string]string{"value": value}}})
	}))
	defer server.Close()
	token := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(token, []byte("jwt"), 0600); err != nil {
		t.Fatal(err)
	}
	client, err := appsecrets.New(server.URL, "runtime-homechef-api", token)
	if err != nil {
		t.Fatal(err)
	}
	previous := baoClient
	baoClient = client
	t.Cleanup(func() { baoClient = previous })
	if err := StorePlatformSecret(context.Background(), "prod-homechef-cashfree-app-id", "updated"); err != nil {
		t.Fatal(err)
	}
	got, err := GetPlatformSecret(context.Background(), "prod-homechef-cashfree-app-id")
	if err != nil || got != "updated" {
		t.Fatalf("backend roundtrip failed: %v", err)
	}
}
