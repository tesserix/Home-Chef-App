// Command backfill-provider-credentials encrypts legacy plaintext credentials
// in delivery_providers. It is dry-run by default and requires explicit project
// confirmation before writing.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/piicrypto"
)

func main() {
	apply := flag.Bool("apply", false, "write encrypted credentials; omitted means dry-run")
	confirmProject := flag.String("confirm-project", "", "exact GCP project ID required with --apply")
	batchSize := flag.Int("batch-size", 100, "rows per bounded transaction (1-1000)")
	flag.Parse()

	config.Load()
	projectID := config.AppConfig.GCSProjectID
	if err := validateBackfillExecution(*apply, *confirmProject, projectID); err != nil {
		log.Fatal(err)
	}
	if !config.AppConfig.PIIEncryptionEnabled {
		log.Fatal("provider credential backfill requires PII_ENCRYPTION_ENABLED=true")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := piicrypto.InitIfEnabled(ctx, true, projectID); err != nil {
		log.Fatalf("initialize provider credential encryption: %v", err)
	}
	if err := database.Connect(); err != nil {
		log.Fatalf("connect for provider credential backfill: %v", err)
	}
	sqlDB, err := database.DB.DB()
	if err != nil {
		log.Fatalf("open provider credential backfill database handle: %v", err)
	}
	defer sqlDB.Close()

	result, err := database.BackfillDeliveryProviderCredentials(ctx, database.DB, database.CredentialBackfillOptions{
		BatchSize: *batchSize,
		DryRun:    !*apply,
	})
	if err != nil {
		log.Fatalf("provider credential backfill: %v", err)
	}
	mode := "dry-run"
	if *apply {
		mode = "apply"
	}
	log.Printf(
		"provider credential backfill complete mode=%s project=%s scanned_rows=%d pending_rows=%d pending_fields=%d updated_rows=%d",
		mode,
		projectID,
		result.ScannedRows,
		result.PendingRows,
		result.PendingFields,
		result.UpdatedRows,
	)
}

func validateBackfillExecution(apply bool, confirmedProject, configuredProject string) error {
	if configuredProject == "" {
		return errors.New("provider credential backfill requires a configured GCP project")
	}
	if !apply {
		return nil
	}
	if confirmedProject == "" {
		return errors.New("--apply requires --confirm-project with the exact configured GCP project ID")
	}
	if confirmedProject != configuredProject {
		return fmt.Errorf("confirmed project %q does not match configured project %q", confirmedProject, configuredProject)
	}
	return nil
}
