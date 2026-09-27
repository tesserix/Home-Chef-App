package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestPausedSecretMutationsFailBeforeBackendAccess(t *testing.T) {
	t.Setenv("APP_SECRET_WRITES_PAUSED", "true")
	ctx := context.Background()
	for name, mutate := range map[string]func() error{
		"vendor":          func() error { return StoreVendorSecret(ctx, "vendor", "bank-ifsc", "test") },
		"driver":          func() error { return StoreDriverSecret(ctx, "driver", "bank-ifsc", "test") },
		"gateway":         func() error { return StorePlatformSecret(ctx, "prod-homechef-cashfree-app-id", "test") },
		"delete":          func() error { return DeleteVendorSecret(ctx, "vendor", "bank-ifsc") },
		"account erasure": func() error { return (chefCascade{}).Purge(nil, uuid.New()) },
	} {
		t.Run(name, func(t *testing.T) {
			if err := mutate(); !errors.Is(err, ErrSecretWritesPaused) {
				t.Fatalf("expected pause error, got %v", err)
			}
		})
	}
}

func TestSecretWritePauseFailsClosedForInvalidSetting(t *testing.T) {
	for setting, want := range map[string]bool{"": false, "false": false, "true": true, "invalid": true} {
		t.Run(setting, func(t *testing.T) {
			t.Setenv("APP_SECRET_WRITES_PAUSED", setting)
			if got := SecretWritesPaused(); got != want {
				t.Fatalf("paused=%v, want %v", got, want)
			}
		})
	}
}
