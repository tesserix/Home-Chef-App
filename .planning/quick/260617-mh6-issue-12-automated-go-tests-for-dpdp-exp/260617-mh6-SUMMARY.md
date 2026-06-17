---
quick_id: 260617-mh6
title: "Issue #12 — automated Go tests for DPDP export/delete handlers"
status: complete
date: 2026-06-17
branch: test/dpdp-export-delete-issue-12
commit: (see git log — test commit on branch)
---

# Summary — 260617-mh6

## What
Added `apps/api/handlers/chef_dpdp_test.go` — 11 tests (test-only, +403 lines)
covering the DPDP Act 2023 data-subject endpoints for Issue #12.

| Area | Tests |
|---|---|
| Export | attachment + dated filename + DPDP §11 notice + user block; sensitive-field omission (allow-list); chef profile + menu inclusion; 404 for unknown user |
| Delete — confirmation gate | missing `confirmEmail` → 400 (row preserved); wrong `confirmEmail` → 400 (row preserved) |
| Delete — retention | soft-delete user, hidden from default scope but row retained with `deleted_at`; `retainUntil` ≈ 30 days |
| Delete — cascade | chef profile hard-deleted; menu items `is_available=false` |
| Delete — audit | PII-safe `audit_logs` row: action `chef.account.delete`, entity `user`/id; payload has no email/phone, records `retainUntil` |
| Delete — retry | documents current 404-on-retry behaviour |
| Unit | `sanitizeUserForExport` allow-lists safe fields only |

## Test approach (matches #10)
In-memory SQLite (`gorm.io/driver/sqlite`); hand-rolled tables because models
carry the Postgres-only `default:gen_random_uuid()` (verified: AutoMigrate emits
invalid SQLite DDL). Reused the existing `setupDB` helper from
`internal_users_test.go` for the `users` table; added `chef_profiles`,
`menu_items`, `audit_logs`. The handler + `services.LogAudit` read the global
`database.DB`, so each test points it at the test DB and restores via
`t.Cleanup` (no `t.Parallel()`).

## Verification
- `go test ./handlers/ -run 'Export|Delete|Sanitize' -v` → 11/11 PASS.
- `go test ./... -race` → all packages green.
- `go build ./...` → clean.

## Findings surfaced (NOT fixed — test-only PR)
1. **`already_deleted` idempotency branch is unreachable.** `DeleteMyAccount`
   uses default-scoped `database.DB.First`, which excludes soft-deleted rows, so
   a retried delete returns **404** rather than the intended idempotent **200
   already_deleted**. Recommended one-line fix: `.Unscoped()` on that lookup.
   Test `..._RetryAfterDelete_DocumentsIdempotencyGap` pins current behaviour.
2. **`ChefProfile` has no `gorm.DeletedAt`** → chef profile is HARD-deleted on
   erasure (only `user` is soft-deleted + retained). Intentional (kitchen
   disappears immediately) but worth noting against the "30-day retention" framing.
3. Minor: handler's `log.Printf` writes the deleted user's email to stdout
   (server-side log PII). The audit row itself is PII-safe.

## Notes
- Test-only change. Merging rebuilds + redeploys `api` (harmless; binary unchanged
  by `_test.go`).
- Suggest moving issue #12 to `in review` (same as #10).
</content>
