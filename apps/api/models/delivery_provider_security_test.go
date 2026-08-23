package models

import "testing"

func TestDeliveryProviderCredentialsUseEncryptedColumns(t *testing.T) {
	provider := DeliveryProvider{
		APIKey:        EncryptedString("api-key"),
		APISecret:     EncryptedString("api-secret"),
		WebhookSecret: EncryptedString("webhook-secret"),
	}
	var _ EncryptedString = provider.APIKey
	var _ EncryptedString = provider.APISecret
	var _ EncryptedString = provider.WebhookSecret
}
