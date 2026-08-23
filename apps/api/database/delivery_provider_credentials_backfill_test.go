package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDeliveryProviderCredentialBackfill_DryRunApplyAndIdempotency(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE delivery_providers (
		id text PRIMARY KEY,
		api_key text,
		api_secret text,
		webhook_secret text
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO delivery_providers
		(id, api_key, api_secret, webhook_secret) VALUES
		('a', 'plain-key-a', 'sealed:secret-a', ''),
		('b', '', 'plain-secret-b', 'plain-webhook-b'),
		('c', 'sealed:key-c', 'sealed:secret-c', 'sealed:webhook-c')`).Error)

	classify := func(value string) (bool, error) { return len(value) >= 7 && value[:7] == "sealed:", nil }
	encrypt := func(value string) (string, error) { return "sealed:" + value, nil }

	dryRun, err := backfillDeliveryProviderCredentials(
		context.Background(), db,
		CredentialBackfillOptions{BatchSize: 1, DryRun: true},
		classify, encrypt,
	)
	require.NoError(t, err)
	assert.Equal(t, CredentialBackfillResult{
		ScannedRows:   3,
		PendingRows:   2,
		PendingFields: 3,
	}, dryRun)
	assert.Equal(t, "plain-key-a", readProviderCredential(t, db, "a", "api_key"))

	applied, err := backfillDeliveryProviderCredentials(
		context.Background(), db,
		CredentialBackfillOptions{BatchSize: 1},
		classify, encrypt,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(3), applied.ScannedRows)
	assert.Equal(t, int64(2), applied.PendingRows)
	assert.Equal(t, int64(3), applied.PendingFields)
	assert.Equal(t, int64(2), applied.UpdatedRows)
	assert.Equal(t, "sealed:plain-key-a", readProviderCredential(t, db, "a", "api_key"))
	assert.Equal(t, "sealed:secret-a", readProviderCredential(t, db, "a", "api_secret"))
	assert.Equal(t, "", readProviderCredential(t, db, "a", "webhook_secret"))
	assert.Equal(t, "sealed:plain-secret-b", readProviderCredential(t, db, "b", "api_secret"))
	assert.Equal(t, "sealed:plain-webhook-b", readProviderCredential(t, db, "b", "webhook_secret"))

	second, err := backfillDeliveryProviderCredentials(
		context.Background(), db,
		CredentialBackfillOptions{BatchSize: 2},
		classify, encrypt,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(3), second.ScannedRows)
	assert.Zero(t, second.PendingRows)
	assert.Zero(t, second.PendingFields)
	assert.Zero(t, second.UpdatedRows)
}

func TestDeliveryProviderCredentialBackfill_RollsBackBatchOnEncryptionFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE delivery_providers (
		id text PRIMARY KEY,
		api_key text,
		api_secret text,
		webhook_secret text
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO delivery_providers VALUES
		('a', 'first-key', '', ''),
		('b', 'second-key', '', '')`).Error)

	encrypt := func(value string) (string, error) {
		if value == "second-key" {
			return "", fmt.Errorf("injected encryption failure")
		}
		return "sealed:" + value, nil
	}
	_, err = backfillDeliveryProviderCredentials(
		context.Background(), db,
		CredentialBackfillOptions{BatchSize: 2},
		func(value string) (bool, error) { return len(value) >= 7 && value[:7] == "sealed:", nil },
		encrypt,
	)
	require.ErrorContains(t, err, "encrypt delivery provider credentials")
	assert.Equal(t, "first-key", readProviderCredential(t, db, "a", "api_key"),
		"the first row in a failed batch must roll back")
}

func TestDeliveryProviderCredentialBackfill_RejectsUnsafeConfiguration(t *testing.T) {
	_, err := backfillDeliveryProviderCredentials(
		context.Background(), nil,
		CredentialBackfillOptions{BatchSize: 0},
		func(string) (bool, error) { return false, nil },
		func(value string) (string, error) { return value, nil },
	)
	require.Error(t, err)
}

func readProviderCredential(t *testing.T, db *gorm.DB, id, column string) string {
	t.Helper()
	require.Contains(t, []string{"api_key", "api_secret", "webhook_secret"}, column)
	var value string
	require.NoError(t, db.Raw(
		"SELECT "+column+" FROM delivery_providers WHERE id = ?", id,
	).Scan(&value).Error)
	return value
}
