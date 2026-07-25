# Account Deactivation, Deletion & 6-Month Restore — Design

**Status:** approved
**Date:** 2026-07-25
**Drives:** iOS + Android store submission for `mobile-customer`, `mobile-vendor`, `mobile-delivery`

## Problem

Apple App Store guideline 5.1.1(v) requires any app with account creation to offer
account deletion **initiated inside the app**. Google Play requires the same plus a
web-accessible deletion URL reachable without installing the app. fe3dr satisfies
neither today for chef or driver, and has three latent defects in the customer path.

The product requirement adds a retention dimension: deletion must not destroy the
user's data outright. A returning user within six months should get their history
back rather than starting from nothing — but must re-earn approval, not silently
inherit their previous good standing.

## Current State

| | Customer | Chef | Driver |
|---|---|---|---|
| In-app delete UI | `mobile-customer/app/data-privacy.tsx` | "contact support" alert | "contact support" alert |
| Backend delete | `POST /me/delete` | `POST /chef/me/delete` | `POST /driver/me/delete` |
| Deactivate | none | none | none |
| Restore | none | none | none |

Backend erasure exists for all three roles (`handlers/dpdp_common.go`,
`customer_dpdp.go`, `chef_dpdp.go`, `driver_dpdp.go`): soft-delete via GORM's
`DeletedAt` plus a 30-day retention notice. The sweeper it references was never
built, so nothing is ever actually purged.

### Defects found during design

1. **Deleting an account permanently locks that email out.**
   `handlers/internal_users.go:112` looks up with GORM's default scope, so a
   soft-deleted user is invisible and the code falls through to `INSERT`. But
   `idx_users_email_per_pool` (`tesserix-k8s .../homechef_db.sql:4831`) is *not*
   filtered on `deleted_at`, so the ghost row still owns the `(lower(email),
   auth_pool)` slot. The insert dies on duplicate key and returns 502. Re-joining
   is impossible today.

2. **Deletion does not revoke access.** `middleware/bff_auth.go:248` aborts only
   when `IsActive` is false. A soft-deleted user is simply *not found*, and the
   request proceeds. Handlers reading `middleware.GetUserID(c)` (token-derived,
   not DB-derived) keep working normally for the life of the session.

3. **The GIP credential is never touched.** `apps/api` has no Identity Platform
   admin capability at all — `config.go:309` notes OAuth secrets are "owned by
   auth-bff/GIP". The user's password/Google identity stays valid after deletion.

A useful asset also exists: `users.is_active=false` already produces a 403
"Account is suspended" at `bff_auth.go:249`. That is the substrate for deactivation.

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Where data lives during retention | Stays in Postgres, soft-deleted | Restore is a flag flip at full fidelity; no lossy re-import, no FK rebuilding |
| Restore trigger | Re-signup detects the ghost row | Credential is genuinely dead, and it retires defect 1 as a side effect |
| Deactivation | Reversible self-serve pause on `is_active` | Reuses a working 403 path; no retention timer |
| Re-approval on restore | Reset to pending, identity docs re-uploaded | Deletion may have *been* the compliance problem; expired/revoked docs must not slip back |
| In-flight money | Block with 409 + machine-readable blockers | No silent forfeiture, no orphaned escrow |
| GIP deletion | New local Identity Toolkit admin client | auth-bff is a shared out-of-repo image serving other products |
| Total PII retention | Exactly 6 months | 180-day restore window, then real erasure; only PII-stripped financial records are archived |

## State Model

Four states on `users`, expressed as columns — no new state table.

| State | Columns | Reversible |
|---|---|---|
| `active` | — | — |
| `deactivated` | `is_active=false`, `deactivated_at` | Yes, self-serve, no timer |
| `pending_deletion` | `deleted_at`, `purge_after = deleted_at + 180d` | Yes, until `purge_after` |
| `purged` | row gone | No |

`purge_after` is stored rather than derived so the sweeper can index it and so
changing the window later does not retroactively reinterpret existing rows.

New columns — added in **tesserix-k8s**
(`charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`) per
the repo rule that no `.sql` lives in application repos:

```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS deactivated_at  timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS purge_after     timestamptz;
ALTER TABLE users ADD COLUMN IF NOT EXISTS deletion_reason text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS restored_at     timestamptz;
CREATE INDEX IF NOT EXISTS idx_users_purge_after
  ON users (purge_after) WHERE purge_after IS NOT NULL;
```

`idx_users_email_per_pool` is deliberately left non-partial. The ghost row holding
the email slot is what forces a returning user through the restore handshake
instead of quietly creating a duplicate account.

## Components

### `services/account_lifecycle.go` — state machine

Role-agnostic transitions: `Deactivate`, `Reactivate`, `RequestDeletion`,
`Restore`, `StartFresh`. Each takes a `models.User` and a role-specific
`RoleCascade` interface so the core logic is written and tested once.

```go
type RoleCascade interface {
    OnDeactivate(tx *gorm.DB, userID uuid.UUID) error
    OnDelete(tx *gorm.DB, userID uuid.UUID) error
    OnRestore(tx *gorm.DB, userID uuid.UUID) error
    Purge(tx *gorm.DB, userID uuid.UUID) error
}
```

Implementations: `chefCascade`, `customerCascade`, `driverCascade`.

### `services/account_blockers.go` — preconditions

Returns `[]Blocker{Code, Label, Count, Amount}`. Checked by both
`GET /me/deletion-eligibility` (pre-warning) and `POST /me/delete` (enforcement),
so the UI can never present a delete button that will 409 unexpectedly.

| Code | Role | Condition |
|---|---|---|
| `active_orders` | all | orders not in `delivered`/`cancelled` |
| `mealplan_escrow` | customer, chef | meal plan with unserved days holding escrow |
| `wallet_balance` | customer | `wallets.balance > 0` |
| `pending_payout` | chef | unreleased payout balance |
| `active_delivery` | driver | delivery assigned and not completed |

### `services/gipadmin.go` — credential teardown

Identity Toolkit REST `accounts:delete`, tenant-scoped (HomeChef runs a customer
pool and an internal pool). Idempotent: `USER_NOT_FOUND` is success, because
deletion is retried. Modelled on mark8ly's `internal/gipadmin/delete.go`.

### `services/account_archive.go` — financial archive

At purge time, builds a PII-stripped financial record (order totals, invoice ids,
payout references, tax lines — no name, email, phone, address or geo) and writes it
to the private GCS bucket via the existing `services.UploadPrivateFile`. Personal
data is erased, not archived. Indian tax law requires the financial trail; DPDP does
not permit keeping the PII alongside it.

### `services/account_purge_cron.go` — sweeper

Registered in `cronJobs()` (`services/cron_temporal.go`) as `account-purge`,
following the existing pair-of-functions pattern (`runAccountPurgeScan` +
`StartAccountPurgeCron`) so it works under both Temporal Schedules and the
in-process ticker fallback. Daily. Selects `purge_after <= now()` unscoped,
archives, then hard-deletes in FK order. Idempotent — a re-run over an
already-purged user is a no-op.

### `handlers/account_lifecycle.go` — HTTP

Reuses `dpdp_common.go` scaffolding. `chef_dpdp.go` currently duplicates that
scaffolding verbatim rather than calling it; it is folded onto the shared helpers
as part of this work, since this change touches that exact path.

## API

Per role, alongside the existing `/me/export` and `/me/delete`:

```
POST /me/deactivate       {reason?}       -> 200 {status:"deactivated", deactivatedAt}
POST /me/reactivate                       -> 200 {status:"active"}
GET  /me/deletion-eligibility             -> 200 {deletable:bool, blockers:[...]}
POST /me/delete           {confirmEmail}  -> 200 {status:"pending_deletion", deletedAt, purgeAfter}
                                          -> 409 {error, blockers:[...]}
POST /account/restore     {restoreToken}  -> 200 {status:"restored", reapprovalRequired:true}
POST /account/start-fresh {restoreToken}  -> 200 {status:"purged"}
```

`/account/*` sit outside the authenticated groups — see the handshake below.

## Restore Handshake

After deletion the GIP identity is gone, so a returning user has **no session** with
which to authenticate a restore call. Rather than teaching the auth middleware to
admit soft-deleted users, `UpsertUser` mints a short-lived restore token.

```
re-signup
  -> UpsertUser looks up Unscoped by (email, auth_pool), requires email_verified
  -> ghost row found, purge_after in the future
  -> 200 {status:"restorable", deletedAt, purgeAfter, restoreToken}
  -> app offers [Restore my account] [Start fresh]
```

`restoreToken` is an HMAC over `userID | newGIPUid | exp`, 15-minute TTL, signed
with the existing BFF key from `middleware/bff_auth.go`. It authorises exactly one
of the two follow-up calls and nothing else.

`email_verified` is required before the ghost is even revealed. Without it, an
unverified password signup on a known address could probe for, or seize, a deleted
account — the same hijack `internal_users.go:105` already guards against on the
live re-bind path.

`start-fresh` purges the ghost synchronously and then creates the new user, which
is what permanently retires the duplicate-key lockout.

## Restore Semantics

Restore returns data but **not** standing.

Common: `deleted_at` and `purge_after` cleared, `is_active=true`, `restored_at`
stamped, new GIP identity bound to the existing row.

**Chef** — `is_approved=false`, `accepting_orders=false`, every
`menu_items.is_approved=false`. Identity documents (`pan_card`, `aadhaar_card`,
`fssai_license`, `food_safety_cert`, `cancelled_cheque`) deleted, rows and GCS
objects both, and must be re-uploaded. Kitchen photos, profile image, menu photos
and descriptions survive. An `ApprovalRequest{Type: kitchen_onboarding, Status:
pending}` is created carrying a `restored_account` marker plus the prior deletion
date and reason, so the admin reviews with that context rather than blind.

**Driver** — `is_verified=false`, held out of dispatch, identity documents deleted,
`ApprovalRequest{Type: driver_onboarding}` raised.

**Customer** — nothing to re-approve. Addresses, orders, wallet and meal plans
return as they were.

Net effect: a restored chef is invisible to customers until an admin re-approves,
exactly like a first-time applicant.

## Deactivation Semantics

`is_active=false` + `deactivated_at`, reusing the working 403 path.

- **Chef** — `accepting_orders=false`. **`auto_schedule_enabled` must also be
  cleared**: the `kitchen-schedule` cron (`cron_temporal.go:33`) flips
  `accepting_orders` back on from operating hours, so without this guard a
  deactivated kitchen silently reopens. The cron additionally gains a
  `is_active = true AND deleted_at IS NULL` filter as defence in depth.
- **Customer** — push tokens deregistered, marketing suppressed.
- **Driver** — `is_online=false`, removed from dispatch.

Reactivation restores access and leaves `accepting_orders` off until the chef opts
back in. Approval is **not** reset: a pause is not a deletion.

## Error Handling

- All transitions are idempotent. Re-deleting returns the existing
  `pending_deletion` payload; re-deactivating is a no-op 200.
- Cascades run inside the same transaction as the state change; a failed cascade
  rolls the whole transition back.
- GIP deletion runs **post-commit, best-effort, logged at WARN** so an Identity
  Platform hiccup cannot abort a committed erasure. The purge sweeper retries it.
- The sweeper isolates per user: one user's purge failing does not stop the batch.

## Testing

Go, tests first per task:

- blocker computation per role, including the zero-blocker path
- state-machine transitions and their idempotency
- restore resets approval, deletes identity docs, keeps menu photos
- `UpsertUser` returns `restorable` for a ghost, `INSERT`s when past `purge_after`
- restore-token HMAC: valid, expired, wrong-user, replayed
- sweeper idempotency — a second run over a purged user is a no-op
- `kitchen-schedule` cron skips deactivated and soft-deleted chefs
- `bff_auth` aborts on a soft-deleted user

Mobile (jest): confirm-gating on exact email match, blocker screen rendering,
restore/start-fresh choice.

## Compliance Artifacts

- Apple 5.1.1(v) — in-app initiation in all three apps.
- Google Play — plus `/account-deletion` in `apps/web-landing`, reachable without
  installing, stating what is deleted, what is retained and for how long.
- Privacy policy (`apps/web-landing/app/privacy`, `apps/web/.../PrivacyPolicyPage`)
  updated from the stale 30-day figure to: 180-day restore window, then erasure;
  PII-stripped financial records retained for the statutory period.
- The `retainUntil` copy in `dpdp_common.go:137` currently promises 30 days and
  must move to 180 to match.

## Out of Scope

- Admin-initiated restore after `purge_after` (the data is genuinely gone).
- Restoring a purged account from the financial archive — it holds no PII by design.
- `apps/web` customer SPA, which is disabled.
