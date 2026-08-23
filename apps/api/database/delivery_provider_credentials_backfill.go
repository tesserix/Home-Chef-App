package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/homechef/api/piicrypto"
	"gorm.io/gorm"
)

const maxCredentialBackfillBatchSize = 1000

// CredentialBackfillOptions controls the bounded provider-credential sweep.
// DryRun scans and reports without writing.
type CredentialBackfillOptions struct {
	BatchSize int
	DryRun    bool
}

// CredentialBackfillResult contains counts only; credential values must never
// enter command output or logs.
type CredentialBackfillResult struct {
	ScannedRows   int64
	PendingRows   int64
	PendingFields int64
	UpdatedRows   int64
}

type deliveryProviderCredentialRow struct {
	ID            string
	APIKey        string
	APISecret     string
	WebhookSecret string
}

type credentialClassifier func(string) (bool, error)
type credentialEncryptor func(string) (string, error)

// BackfillDeliveryProviderCredentials encrypts legacy plaintext provider
// credentials. Encryption must already be active; otherwise it fails closed.
func BackfillDeliveryProviderCredentials(
	ctx context.Context,
	db *gorm.DB,
	options CredentialBackfillOptions,
) (CredentialBackfillResult, error) {
	if !piicrypto.Active() {
		return CredentialBackfillResult{}, errors.New("provider credential backfill requires active PII encryption")
	}
	classify := func(value string) (bool, error) {
		if !piicrypto.IsCiphertext(value) {
			return false, nil
		}
		if _, err := piicrypto.DecryptPII(value); err != nil {
			return true, fmt.Errorf("validate existing provider credential ciphertext: %w", err)
		}
		return true, nil
	}
	return backfillDeliveryProviderCredentials(ctx, db, options, classify, piicrypto.EncryptPII)
}

func backfillDeliveryProviderCredentials(
	ctx context.Context,
	db *gorm.DB,
	options CredentialBackfillOptions,
	classify credentialClassifier,
	encrypt credentialEncryptor,
) (CredentialBackfillResult, error) {
	var result CredentialBackfillResult
	if db == nil {
		return result, errors.New("provider credential backfill requires a database")
	}
	if options.BatchSize <= 0 || options.BatchSize > maxCredentialBackfillBatchSize {
		return result, fmt.Errorf("provider credential backfill batch size must be between 1 and %d", maxCredentialBackfillBatchSize)
	}
	if classify == nil || encrypt == nil {
		return result, errors.New("provider credential backfill requires encryption functions")
	}

	lastID := ""
	for {
		var rows []deliveryProviderCredentialRow
		query := db.WithContext(ctx).
			Table("delivery_providers").
			Select(`id,
				COALESCE(api_key, '') AS api_key,
				COALESCE(api_secret, '') AS api_secret,
				COALESCE(webhook_secret, '') AS webhook_secret`).
			Order("id ASC").
			Limit(options.BatchSize)
		if lastID != "" {
			query = query.Where("id > ?", lastID)
		}
		if err := query.Scan(&rows).Error; err != nil {
			return result, fmt.Errorf("scan delivery provider credentials: %w", err)
		}
		if len(rows) == 0 {
			return result, nil
		}
		result.ScannedRows += int64(len(rows))

		pending := make([]deliveryProviderCredentialRow, 0, len(rows))
		for _, row := range rows {
			fields, err := pendingCredentialFields(row, classify)
			if err != nil {
				return result, fmt.Errorf("classify delivery provider %s credentials: %w", row.ID, err)
			}
			if fields == 0 {
				continue
			}
			result.PendingRows++
			result.PendingFields += int64(fields)
			pending = append(pending, row)
		}

		if !options.DryRun && len(pending) > 0 {
			err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				if tx.Dialector.Name() == "postgres" {
					if err := tx.Exec("SET LOCAL lock_timeout = '2s'").Error; err != nil {
						return fmt.Errorf("set credential backfill lock timeout: %w", err)
					}
					if err := tx.Exec("SET LOCAL statement_timeout = '30s'").Error; err != nil {
						return fmt.Errorf("set credential backfill statement timeout: %w", err)
					}
				}
				for _, row := range pending {
					updates, err := encryptedCredentialUpdates(row, classify, encrypt)
					if err != nil {
						return fmt.Errorf("encrypt delivery provider credentials: %w", err)
					}
					updated := tx.Table("delivery_providers").
						Where(`id = ?
							AND COALESCE(api_key, '') = ?
							AND COALESCE(api_secret, '') = ?
							AND COALESCE(webhook_secret, '') = ?`,
							row.ID, row.APIKey, row.APISecret, row.WebhookSecret).
						Updates(updates)
					if updated.Error != nil {
						return fmt.Errorf("update delivery provider %s credentials: %w", row.ID, updated.Error)
					}
					if updated.RowsAffected != 1 {
						return fmt.Errorf("update delivery provider %s credentials: row changed concurrently", row.ID)
					}
				}
				return nil
			})
			if err != nil {
				return result, err
			}
			result.UpdatedRows += int64(len(pending))
		}
		lastID = rows[len(rows)-1].ID
	}
}

func pendingCredentialFields(row deliveryProviderCredentialRow, classify credentialClassifier) (int, error) {
	pending := 0
	for _, value := range []string{row.APIKey, row.APISecret, row.WebhookSecret} {
		if value == "" {
			continue
		}
		sealed, err := classify(value)
		if err != nil {
			return 0, err
		}
		if !sealed {
			pending++
		}
	}
	return pending, nil
}

func encryptedCredentialUpdates(
	row deliveryProviderCredentialRow,
	classify credentialClassifier,
	encrypt credentialEncryptor,
) (map[string]any, error) {
	updates := make(map[string]any, 3)
	values := []struct {
		column string
		value  string
	}{
		{column: "api_key", value: row.APIKey},
		{column: "api_secret", value: row.APISecret},
		{column: "webhook_secret", value: row.WebhookSecret},
	}
	for _, field := range values {
		if field.value == "" {
			continue
		}
		sealed, err := classify(field.value)
		if err != nil {
			return nil, err
		}
		if sealed {
			continue
		}
		ciphertext, err := encrypt(field.value)
		if err != nil {
			return nil, err
		}
		updates[field.column] = ciphertext
	}
	return updates, nil
}
