package services

import (
	"context"
	"fmt"
	"log"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/homechef/api/config"
)

var secretClient *secretmanager.Client

// InitSecretManager initializes the GCP Secret Manager client.
// Uses default credentials (Workload Identity on GKE, ADC locally).
func InitSecretManager() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create secret manager client: %w", err)
	}
	secretClient = client
	log.Println("GCP Secret Manager client initialized")
	return nil
}

// CloseSecretManager closes the Secret Manager client
func CloseSecretManager() {
	if secretClient != nil {
		secretClient.Close()
	}
}

// vendorSecretID builds the secret ID for a vendor (chef) payment field.
// Format: prod-vendor-payment-<vendorId>-<field>
func vendorSecretID(vendorID, field string) string {
	return paymentSecretID("vendor", vendorID, field)
}

// DriverSecretID builds the secret ID for a driver payment field.
// Format: prod-driver-payment-<driverId>-<field>
func driverSecretID(driverID, field string) string {
	return paymentSecretID("driver", driverID, field)
}

// paymentSecretID builds the full secret ID.
// Format: <env>-<role>-payment-<entityId>-<field>
func paymentSecretID(role, entityID, field string) string {
	env := "prod"
	if config.IsDevelopment() {
		env = "dev"
	}
	return fmt.Sprintf("%s-%s-payment-%s-%s", env, role, entityID, field)
}

// StoreVendorSecret creates or updates a secret for a vendor's payment field.
func StoreVendorSecret(ctx context.Context, vendorID, field, value string) error {
	return storeSecret(ctx, vendorSecretID(vendorID, field), vendorID, field, value)
}

// GetVendorSecret retrieves the latest version of a vendor's payment secret.
func GetVendorSecret(ctx context.Context, vendorID, field string) (string, error) {
	return getSecret(ctx, vendorSecretID(vendorID, field))
}

// storeSecret creates or updates a secret in GCP Secret Manager.
func storeSecret(ctx context.Context, secretID, entityID, field, value string) error {
	if secretClient == nil {
		return fmt.Errorf("secret manager not initialized")
	}
	if value == "" {
		return nil
	}

	projectID := config.AppConfig.GCSProjectID
	parent := fmt.Sprintf("projects/%s", projectID)
	secretPath := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretID)

	// Create the secret (idempotent — if it exists, we add a new version)
	_, err := secretClient.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   parent,
		SecretId: secretID,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
			Labels: map[string]string{
				"app":       "homechef",
				"type":      "payment",
				"entity-id": entityID,
				"field":     field,
			},
		},
	})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("failed to create secret %s: %w", secretID, err)
	}

	// Add the secret version with the value
	_, err = secretClient.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretPath,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(value),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add secret version for %s: %w", secretID, err)
	}

	return nil
}

// getSecret retrieves the latest version of a secret.
func getSecret(ctx context.Context, secretID string) (string, error) {
	if secretClient == nil {
		return "", fmt.Errorf("secret manager not initialized")
	}

	projectID := config.AppConfig.GCSProjectID
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretID)

	result, err := secretClient.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to access secret %s: %w", secretID, err)
	}

	return string(result.Payload.Data), nil
}

// StoreDriverSecret creates or updates a secret for a driver's payment field.
func StoreDriverSecret(ctx context.Context, driverID, field, value string) error {
	return storeSecret(ctx, driverSecretID(driverID, field), driverID, field, value)
}

// GetDriverSecret retrieves the latest version of a driver's payment secret.
func GetDriverSecret(ctx context.Context, driverID, field string) (string, error) {
	return getSecret(ctx, driverSecretID(driverID, field))
}

// StorePlatformSecret stores a platform-level secret (e.g. Razorpay API keys).
// The secretName is the full GCP Secret Manager secret ID (e.g. "prod-razorpay-key-id").
func StorePlatformSecret(ctx context.Context, secretName, value string) error {
	if secretClient == nil {
		return fmt.Errorf("secret manager not initialized")
	}
	if value == "" {
		return nil
	}

	projectID := config.AppConfig.GCSProjectID
	parent := fmt.Sprintf("projects/%s", projectID)
	secretPath := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretName)

	// Create secret if it doesn't exist
	_, err := secretClient.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   parent,
		SecretId: secretName,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
			Labels: map[string]string{
				"app":  "homechef",
				"type": "platform",
			},
		},
	})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("failed to create secret %s: %w", secretName, err)
	}

	// Add new version
	_, err = secretClient.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretPath,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(value),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add version for %s: %w", secretName, err)
	}

	log.Printf("Platform secret %s updated", secretName)
	return nil
}

// DeleteVendorSecret destroys all versions of a vendor payment secret.
func DeleteVendorSecret(ctx context.Context, vendorID, field string) error {
	if secretClient == nil {
		return fmt.Errorf("secret manager not initialized")
	}

	projectID := config.AppConfig.GCSProjectID
	secretID := vendorSecretID(vendorID, field)
	name := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretID)

	err := secretClient.DeleteSecret(ctx, &secretmanagerpb.DeleteSecretRequest{
		Name: name,
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete secret %s: %w", secretID, err)
	}

	return nil
}

// isAlreadyExists checks if the error is an "already exists" gRPC error
func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "AlreadyExists") || contains(err.Error(), "already exists")
}

// isNotFound checks if the error is a "not found" gRPC error
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "NotFound") || contains(err.Error(), "not found")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
