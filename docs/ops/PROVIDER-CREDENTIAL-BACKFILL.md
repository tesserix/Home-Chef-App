# Delivery-provider credential backfill

`delivery_providers.api_key`, `api_secret`, and `webhook_secret` are encrypted
by `models.EncryptedString` on new writes. Rows written before that type change
must be rewritten after PII encryption is initialized.

The command is dry-run by default, processes UUID-ordered batches, validates
existing ciphertext, and logs counts only. It never prints credential values.
Each write uses the values read by the scan as a concurrency guard; an operator
edit made during the run causes the batch to roll back instead of being
overwritten. A failed or interrupted run is safe to resume because ciphertext
is not encrypted twice.

## Preconditions

- Obtain explicit approval naming the database environment and GCP project.
- Confirm `PII_ENCRYPTION_ENABLED=true` on every API and worker process that can
  read these rows. A partial fleet will fail to read encrypted values.
- Confirm the KMS key and wrapped-DEK secrets are reachable by the command's
  workload identity.
- Capture a database backup or point-in-time recovery marker according to the
  production runbook.
- Run during a low-write window and pause delivery-provider credential edits.

## Dry run

From `apps/api`, with the intended database and GCP environment configured:

```sh
go run ./cmd/backfill-provider-credentials --batch-size 100
```

Record `project`, `scanned_rows`, `pending_rows`, and `pending_fields`. A dry run
still initializes encryption so it can validate ciphertext already in the
table.

## Apply

Only after approval and dry-run review, replace the placeholder with the exact
configured project ID:

```sh
go run ./cmd/backfill-provider-credentials \
  --apply \
  --confirm-project YOUR_EXACT_GCP_PROJECT_ID \
  --batch-size 100
```

The command rejects an empty or mismatched project confirmation. Re-run the dry
run afterward; `pending_rows` and `pending_fields` must both be zero.

## Rollback

The application can read both plaintext and ciphertext while encryption is
active, so an interrupted backfill needs no rollback—fix the error and resume.
Do not turn encryption off after any row is encrypted. Restoring plaintext
requires an explicitly approved database restore or a separately reviewed
decrypt migration; this command intentionally has no decrypt mode.
