---
phase: quick-260617-due
plan: 01
subsystem: auth + mobile-profile
tags: [auth, gip, social-login, account-linking, apple-signin, profile, react-query]
requires:
  - "GIP token claims (name, picture, email_verified) from Firebase/Google/Apple"
  - "models.User Avatar/FirstName/LastName columns (already exist — no migration)"
provides:
  - "Name/Picture/EmailVerified extraction from GIP tokens (auth-bff)"
  - "Avatar + email_verified forwarding through autologin → apiclient → API upsert"
  - "Backfill-only profile writes (never clobber edited name/avatar)"
  - "email_verified-gated same-email re-bind (anti-hijack)"
  - "Apple first-auth name → Firebase displayName"
  - "Flat customer profile read (no {data} envelope)"
affects:
  - apps/auth-bff
  - apps/api
  - packages/mobile-shared
  - apps/mobile-customer
tech-stack:
  added: []
  patterns:
    - "Safe comma-ok claim extraction (zero-value defaults, never panic)"
    - "Backfill-only field writes (set only when stored value is empty)"
    - "Security gate on email_verified before account re-bind"
    - "Best-effort non-fatal displayName update in try/catch"
key-files:
  created:
    - .planning/quick/260617-due-social-profile-and-profile-save-fix/deferred-items.md
  modified:
    - apps/auth-bff/internal/gip/verifier.go
    - apps/auth-bff/internal/gip/verifier_test.go
    - apps/auth-bff/internal/autologin/service.go
    - apps/auth-bff/internal/autologin/service_test.go
    - apps/auth-bff/internal/apiclient/users.go
    - apps/auth-bff/internal/apiclient/users_test.go
    - apps/api/handlers/internal_users.go
    - apps/api/handlers/internal_users_test.go
    - packages/mobile-shared/src/auth/sign-in.ts
    - apps/mobile-customer/app/(auth)/login.tsx
    - apps/mobile-vendor/app/(auth)/login.tsx
    - apps/mobile-delivery/app/(auth)/login.tsx
    - apps/mobile-customer/hooks/useProfile.ts
    - apps/mobile-customer/app/(tabs)/profile.tsx
decisions:
  - "CustomerProfileResponse kept as a type alias of CustomerProfile (not deleted) to avoid breaking existing imports while removing the {data} envelope"
  - "email_verified is request-only (not persisted) — it gates re-bind; no DB column added (matches no-migration constraint)"
  - "Vendor/delivery profile screens already read flat — left unchanged (audit documented below)"
metrics:
  duration: ~25m
  completed: 2026-06-17
---

# Phase quick-260617-due Plan 01: Social Profile + Profile-Save Fix Summary

Social logins now populate name + avatar (backfill-only), same-email re-bind is gated on a verified email so unverified password signups cannot hijack a verified social account row, Apple's one-time first-authorization name flows into the Firebase displayName, and the customer profile screen reads the flat API body so edits persist visibly.

## What Was Built

**Workstream A — Social profile population (Tasks 1 & 2)**
- `gip.VerifiedToken` gained `Name`, `Picture`, `EmailVerified`, extracted from the `name` / `picture` / `email_verified` claims via safe comma-ok assertions (missing/wrong-typed claims default to zero values, never panic).
- `AutoLogin()` forwards `Name=tok.Name`, `Avatar=tok.Picture`, `EmailVerified=tok.EmailVerified` into the upsert request.
- `apiclient.UpsertUserRequest` and `handlers.UpsertUserRequest` both marshal/bind `avatar` + `email_verified`.
- API upsert: CREATE persists `Avatar`; repeat login (gip_uid HIT) and the email re-bind branch backfill avatar **only when empty** — never overwrite an edited avatar/name.

**Workstream B — Account-linking hardening (Task 2)**
- The same-email re-bind branch is now gated on `req.EmailVerified == true`. An unverified password signup using an email already owned by a verified social account falls through to a fresh INSERT (separate row) instead of hijacking the existing row. Google/Apple GIP tokens are always `email_verified=true`; only unverified password signups are false.

**Workstream C — Apple first-auth name (Task 3)**
- `signInWithAppleCredential(idToken, rawNonce, fullName?)` now accepts Apple's nullable `fullName`. After credential sign-in, if the Firebase user's `displayName` is empty and a name is present, it builds `"given family"` (trimmed, blank parts skipped) and calls `updateProfile({ displayName })` best-effort (wrapped in try/catch, non-fatal). The same userCredential is returned; inputs are not mutated.
- All three login screens (`customer`, `vendor`, `delivery`) pass `cred.fullName` through.

**Workstream D — Profile-save read fix (Task 4)**
- Root cause confirmed: API returns the profile FLAT (`{ id, firstName, ... }`), but the customer hook typed it as `{ data: CustomerProfile }` and `profile.tsx` did `const profile = data?.data` → `undefined` → form never repopulated → looked unsaved. Write path was already fine.
- `useProfile()` / `useUpdateProfile()` now return the flat `CustomerProfile` for GET + PUT; `CustomerProfileResponse` is kept as a type alias of `CustomerProfile`. `onSuccess` invalidation preserved.
- `profile.tsx` reads `const profile = data` (flat). All field references (firstName, lastName, phone, email, cuisinePreferences) resolve against the flat object.

## Vendor / Delivery Profile Audit (Task 4 requirement)

| App | File | Finding | Action |
|-----|------|---------|--------|
| mobile-vendor | `app/profile.tsx` | `useChefProfile` uses `useQuery<ChefProfile>` with `api.get<ChefProfile>('/chef/profile').then((r) => r.data)`; screen consumes `data?.businessName` etc. directly — already flat, no `data?.data`. | Left unchanged (already correct) |
| mobile-delivery | `app/driver-profile.tsx` | `useDriverProfile` uses `useQuery<DriverProfile>` with `.then((r) => r.data as DriverProfile)`; screen consumes `const { data: profile }` directly — already flat. | Left unchanged (already correct) |

The `data?.data` occurrences elsewhere in the vendor app (`reviews.tsx`, `(onboarding)/pending.tsx`, `hooks/useLocations.ts`, `hooks/useAdminRequests.ts`, `(onboarding)/kitchen-details.tsx`) are non-profile screens whose endpoints genuinely return an envelope; they are out of this task's scope and working as designed.

## Tests

- **auth-bff** (TDD RED→GREEN): `verifier_test.go` (+ name/picture/email_verified extraction, + missing-claim zero-value defaults), `service_test.go` (+ forwards Name/Avatar/EmailVerified), `users_test.go` (+ marshals `avatar` + `email_verified`). `go test ./...` green.
- **api** (TDD RED→GREEN): `internal_users_test.go` (+ name+avatar on CREATE, + backfill-only on repeat, + unverified email does NOT re-bind / new row, + verified email DOES re-bind). `go test ./...` green.
- **mobile**: `packages/mobile-shared` `tsc --noEmit` and `apps/mobile-customer` `tsc --noEmit` show zero new errors from touched files (pre-existing unrelated errors documented in deferred-items.md). No EAS builds, no prebuild.

## Threat Model Compliance

| Threat ID | Disposition | Status |
|-----------|-------------|--------|
| T-due-01 (spoofing: re-bind by email) | mitigate | DONE — re-bind gated on `email_verified == true` (Task 2), covered by `TestUpsert_UnverifiedEmail_DoesNotRebind_CreatesNewRow` + `TestUpsert_VerifiedEmail_DoesRebind` |
| T-due-02 (tampering: overwrite profile fields) | mitigate | DONE — backfill-only avatar/name on repeat login, covered by `TestUpsert_RepeatLogin_DoesNotOverwriteAvatarOrName` |
| T-due-03 (forged email_verified) | accept | HMAC-signed internal call; no external caller can set it |
| T-due-04 (Apple displayName update) | accept | best-effort, scoped to the authenticated user |

## Deviations from Plan

None — plan executed exactly as written. (One operational note: a worktree-base correction was applied at startup — the plan-check abbreviation `a1d70c6c` was a typo for the real base `a1d70c69…`; the worktree was hard-reset to the correct plan-base commit before any work.)

## Deferred Issues

Pre-existing, out-of-scope typecheck errors (NOT introduced by this plan, confirmed by stash-and-recount) are logged in `deferred-items.md`:
- `packages/mobile-shared/src/components/OfflineBanner.tsx` (2) — NativeWind `className` JSX typing.
- `apps/mobile-customer/app/_layout.tsx` (1) — expo-notifications `NotificationBehavior` type drift.
- `apps/mobile-customer/components/cart/CartSheet.tsx` (2) — stale `@ts-expect-error` directives.

## Commits

- `a7a299f` feat(quick-260617-due): forward name/avatar/email_verified through verify→autologin→apiclient
- `6783a39` feat(quick-260617-due): upsert avatar backfill-only + email_verified-gated re-bind
- `c75bae3` feat(quick-260617-due): capture Apple first-auth name into Firebase displayName
- `d878ac3` fix(quick-260617-due): read customer profile flat (drop {data} envelope mismatch)

## Self-Check: PASSED

All 12 created/modified files verified present on disk; all 4 task commits verified in git history.
