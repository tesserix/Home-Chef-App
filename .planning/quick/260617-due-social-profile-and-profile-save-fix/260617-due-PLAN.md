---
phase: quick-260617-due
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - apps/auth-bff/internal/gip/verifier.go
  - apps/auth-bff/internal/autologin/service.go
  - apps/auth-bff/internal/apiclient/users.go
  - apps/api/handlers/internal_users.go
  - apps/api/handlers/internal_users_test.go
  - apps/auth-bff/internal/gip/verifier_test.go
  - apps/auth-bff/internal/autologin/service_test.go
  - apps/auth-bff/internal/apiclient/users_test.go
  - packages/mobile-shared/src/auth/sign-in.ts
  - apps/mobile-customer/app/(auth)/login.tsx
  - apps/mobile-vendor/app/(auth)/login.tsx
  - apps/mobile-delivery/app/(auth)/login.tsx
  - apps/mobile-customer/hooks/useProfile.ts
  - apps/mobile-customer/app/(tabs)/profile.tsx
autonomous: true
requirements: [SOCIAL-PROFILE, ACCOUNT-LINK, APPLE-NAME, PROFILE-PERSIST]

must_haves:
  truths:
    - "Social login (Google/Apple) populates user name and avatar on the user row"
    - "An unverified password signup cannot re-bind (hijack) a verified social account's existing row"
    - "Apple's first-authorization full name flows into the Firebase displayName and subsequent ID tokens"
    - "Customer profile edits persist visibly: revisiting the profile screen shows saved values"
    - "Existing non-empty avatar/name is never overwritten on repeat login (backfill-only)"
  artifacts:
    - path: apps/auth-bff/internal/gip/verifier.go
      provides: "Name/Picture/EmailVerified extraction from GIP token claims"
      contains: "Picture"
    - path: apps/api/handlers/internal_users.go
      provides: "Avatar + email_verified upsert with backfill-only and verified-gated re-bind"
      contains: "EmailVerified"
    - path: packages/mobile-shared/src/auth/sign-in.ts
      provides: "Apple full-name capture into Firebase displayName"
    - path: apps/mobile-customer/hooks/useProfile.ts
      provides: "Flat CustomerProfile read/write (no {data} envelope)"
  key_links:
    - from: apps/auth-bff/internal/autologin/service.go
      to: apps/auth-bff/internal/apiclient/users.go
      via: "UpsertUserRequest now carries Name/Avatar/EmailVerified"
      pattern: "Avatar:\\s*tok\\.Picture"
    - from: apps/mobile-customer/app/(tabs)/profile.tsx
      to: apps/mobile-customer/hooks/useProfile.ts
      via: "profile screen reads flat object (data, not data.data)"
      pattern: "const profile = data"
---

<objective>
Populate social-login profiles (name + avatar), harden same-email account linking with an email_verified gate, capture Apple's first-authorization name into the Firebase displayName, and fix the customer "profile edits don't persist in UI" bug.

Purpose: Social sign-ins currently produce nameless, avatar-less user rows; an unverified password signup could re-bind a verified social account's row; Apple's one-time name is lost; and customer profile edits silently appear unsaved due to a read-side envelope mismatch.

Output: Backend upsert path forwards name/avatar/email_verified with backfill-only writes and a verified-gated re-bind; mobile Apple flow sets displayName; customer profile read path consumes the flat API body. No DB migration, no EAS builds.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@./CLAUDE.md

<architecture>
Flow: mobile (Expo) → Firebase/GIP sign-in → id_token → auth-bff `/auth/auto-login`
(verifies token, extracts claims) → Go API `/internal/users/upsert` (HMAC-signed) → user row.

Mobile data calls go through axios (`packages/mobile-shared/src/api/client.ts`, plain).
`r.data` is the raw flat HTTP body. The Go API returns FLAT objects — there is NO `{data}` envelope.

Columns Avatar, FirstName, LastName ALREADY EXIST on models.User — NO migration.

CONFIRMED (do NOT re-investigate): the write path persists correctly.
`.Model(&loadedStruct).Updates(map)` auto-scopes by primary key — the earlier
"needs explicit .Where()" theory is WRONG. Workstream D's bug is purely the READ path
(envelope mismatch), not the write.
</architecture>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Backend — forward name/avatar/email_verified through verify → autologin → apiclient</name>
  <files>
    apps/auth-bff/internal/gip/verifier.go,
    apps/auth-bff/internal/gip/verifier_test.go,
    apps/auth-bff/internal/autologin/service.go,
    apps/auth-bff/internal/autologin/service_test.go,
    apps/auth-bff/internal/apiclient/users.go,
    apps/auth-bff/internal/apiclient/users_test.go
  </files>
  <behavior>
    - Verify(): given claims with name/picture/email_verified, VerifiedToken.Name/Picture/EmailVerified are populated.
    - Verify(): missing claims default to "" / "" / false without error.
    - AutoLogin(): the UpsertUserRequest sent to the API carries Name=tok.Name, Avatar=tok.Picture, EmailVerified=tok.EmailVerified.
    - apiclient: UpsertUserRequest marshals `avatar` and `email_verified` JSON keys in the request body.
  </behavior>
  <action>
    1. `verifier.go`: ADD fields `Name string`, `Picture string`, `EmailVerified bool` to the `VerifiedToken` struct. In `Verify()`, extract from claims: `name` (string), `picture` (string), `email_verified` (bool) and set them on the returned VerifiedToken. Use safe type assertions (comma-ok) so missing/wrong-typed claims default to zero values — never panic.
    2. `apiclient/users.go`: ADD `Avatar string `json:"avatar"`` and `EmailVerified bool `json:"email_verified"`` to `UpsertUserRequest` (which already has GIPUid, GIPTenantID, GIPProvider, AuthPool, Email, Name, Role, MarketingConsent).
    3. `autologin/service.go`: in `AutoLogin()` where it builds the `apiclient.UpsertUserRequest`, ADD `Name: tok.Name`, `Avatar: tok.Picture`, `EmailVerified: tok.EmailVerified` (it currently passes GIPUid/GIPTenantID/GIPProvider/AuthPool/Email/Role and does NOT pass Name).
    4. Extend tests: `verifier_test.go` covers name/picture/email_verified extraction + missing-claim defaults; `service_test.go` asserts the forwarded request contains Name/Avatar/EmailVerified; `users_test.go` asserts the JSON body marshals `avatar` and `email_verified`.
    Follow Go conventions (comment exported fields). Single-line commit.
  </action>
  <verify>
    <automated>cd apps/auth-bff && go test ./internal/gip/... ./internal/autologin/... ./internal/apiclient/...</automated>
  </verify>
  <done>auth-bff tests pass; VerifiedToken exposes Name/Picture/EmailVerified; AutoLogin forwards all three into UpsertUserRequest; apiclient request marshals `avatar` + `email_verified`.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: API — upsert avatar (backfill-only) + email_verified-gated re-bind</name>
  <files>
    apps/api/handlers/internal_users.go,
    apps/api/handlers/internal_users_test.go
  </files>
  <behavior>
    - CREATE: a brand-new user is inserted with Name and Avatar from the request.
    - Backfill-only: on repeat login (gip_uid HIT), an existing non-empty Avatar/FirstName/LastName is NOT overwritten; a blank one is filled.
    - Re-bind gated: gip_uid MISS + existing same-email+auth_pool row + email_verified=FALSE → do NOT re-bind; a NEW user row is inserted instead.
    - Re-bind allowed: same conditions but email_verified=TRUE → existing row IS re-bound to the new gip_uid (current behavior preserved).
  </behavior>
  <action>
    1. `UpsertUserRequest` struct: ADD `Avatar string `json:"avatar"`` and `EmailVerified bool `json:"email_verified"``.
    2. CREATE path (the `models.User{...}` literal): set `Avatar: req.Avatar` (Name/FirstName/LastName handling already exists — leave it, just add avatar).
    3. gip_uid-HIT default branch AND the email re-bind branch: BACKFILL-ONLY for avatar — set `u.Avatar` ONLY if currently `""`, mirroring the existing name-backfill (FirstName/LastName set only when both blank). Never overwrite a non-empty Avatar or name.
    4. Email re-bind branch (currently ~lines 97-119: gip_uid miss → lookup existing row by email+auth_pool → re-bind to new gip_uid): GATE the re-bind on `req.EmailVerified == true`. If the incoming token's email is NOT verified, do NOT re-bind — fall through to the new-user INSERT path. Add a clear comment explaining the security rationale: an unverified password signup must not hijack a verified social account's row. Note context: Google/Apple GIP tokens are always email_verified=true; only unverified password signups are false.
    5. `internal_users_test.go`: add the four cases from <behavior> — (1) name+avatar on CREATE; (2) backfill-only on repeat (non-empty NOT overwritten); (3) email_verified=false does NOT re-bind (new row created); (4) email_verified=true DOES re-bind.
    Use immutable/explicit field assignment per project style. Single-line commit.
  </action>
  <verify>
    <automated>cd apps/api && go test ./handlers/ -run TestUpsert</automated>
  </verify>
  <done>All four upsert cases pass; CREATE persists avatar; repeat login is backfill-only; unverified email does not re-bind; verified email re-binds.</done>
</task>

<task type="auto">
  <name>Task 3: Mobile — capture Apple first-auth name into Firebase displayName</name>
  <files>
    packages/mobile-shared/src/auth/sign-in.ts,
    apps/mobile-customer/app/(auth)/login.tsx,
    apps/mobile-vendor/app/(auth)/login.tsx,
    apps/mobile-delivery/app/(auth)/login.tsx
  </files>
  <action>
    1. `sign-in.ts` `signInWithAppleCredential(idToken, rawNonce)`: change the signature to accept an optional `fullName` (Apple's AppleAuthentication FullName object — fields `givenName`/`familyName`, both nullable). After `auth().signInWithCredential(cred)` resolves, if the resulting firebase user's `displayName` is empty AND `fullName` has at least one of givenName/familyName, build `"given family"` (trim, single space, skip blank parts) and call the firebase user's updateProfile/updateDisplayName to set displayName. This makes the name flow into subsequent ID tokens (Apple returns the name ONLY on the FIRST authorization). Keep best-effort: wrap the displayName update in try/catch (non-fatal). Return the SAME userCredential. Do NOT mutate inputs — build the name string into a new local.
    2. Each app `(auth)/login.tsx` `handleAppleSignIn`: pass `cred.fullName` to `signInWithAppleCredential` (the `AppleAuthentication.signInAsync` result already includes `fullName` since FULL_NAME scope is requested). Keep the existing `rawNonce` '' arg. Order: `signInWithAppleCredential(idToken, '', cred.fullName)` (or named per current arg order).
  </action>
  <verify>
    <automated>cd packages/mobile-shared && (npm run typecheck 2>/dev/null || npx tsc --noEmit -p tsconfig.json)</automated>
  </verify>
  <done>signInWithAppleCredential accepts fullName and sets Firebase displayName best-effort when empty; all three login screens pass cred.fullName; typecheck clean.</done>
</task>

<task type="auto">
  <name>Task 4: Mobile — fix customer profile read-side envelope mismatch (+ audit vendor/delivery)</name>
  <files>
    apps/mobile-customer/hooks/useProfile.ts,
    apps/mobile-customer/app/(tabs)/profile.tsx
  </files>
  <action>
    Root cause (CONFIRMED): backend returns the profile FLAT via `c.JSON(200, profile.ToResponse(&user))` (body = { id, firstName, lastName, ... }), but the customer app types it as `CustomerProfileResponse = { data: CustomerProfile }` and `profile.tsx:110` does `const profile = data?.data` → undefined → form never repopulates → looks unsaved. Write path is FINE.
    1. `useProfile.ts`: GET and PUT responses are the FLAT `CustomerProfile` (top-level: id, firstName, lastName, email, phone, avatar, cuisinePreferences, dietaryPreferences, foodAllergies, spiceTolerance, householdSize). Change `useProfile()`'s queryFn to `return r.data as CustomerProfile`, and `useUpdateProfile()`'s mutationFn likewise. Drop/replace the misleading `CustomerProfileResponse` `{data}` wrapper (or make it `type CustomerProfileResponse = CustomerProfile`). KEEP the `onSuccess` `invalidateQueries(['profile'])`.
    2. `profile.tsx`: change `const profile = data?.data` to `const profile = data` (flat). Verify every reference (firstName, lastName, phone, cuisinePreferences, etc.) still resolves against the flat object.
    3. AUDIT `apps/mobile-vendor` and `apps/mobile-delivery` profile/account screens + hooks for the SAME `data?.data` envelope mismatch. If present, apply the identical flat-read fix. If their screens use a different (working) shape, leave them unchanged. DOCUMENT what was found (which files, fixed-or-left) in the SUMMARY.
  </action>
  <verify>
    <automated>cd apps/mobile-customer && (npm run typecheck 2>/dev/null || npx tsc --noEmit)</automated>
  </verify>
  <done>Customer useProfile returns flat CustomerProfile for GET+PUT; profile.tsx reads `data` (not `data.data`); all field refs resolve; vendor/delivery audited and either fixed or documented as already-correct; typecheck clean.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| mobile → auth-bff | Untrusted client presents a Firebase/GIP id_token; auth-bff verifies signature + extracts claims |
| auth-bff → API /internal/users/upsert | HMAC-signed internal call; carries identity claims including email_verified |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-due-01 | Spoofing | account re-bind by email | mitigate | Gate same-email re-bind on `email_verified == true` (Task 2). Unverified password signup cannot hijack a verified social account row. |
| T-due-02 | Tampering | upsert overwriting profile fields | mitigate | Backfill-only writes — never overwrite a non-empty Avatar/FirstName/LastName on repeat login (Task 2). |
| T-due-03 | Elevation | forged email_verified in request body | accept | `/internal/users/upsert` is HMAC-signed and only callable by auth-bff, which derives email_verified from a signature-verified GIP token. No external caller can set it. |
| T-due-04 | Information disclosure | Apple displayName update | accept | Name is the user's own; updateProfile is best-effort and scoped to the authenticated firebase user. |
</threat_model>

<verification>
- `cd apps/auth-bff && go test ./...` passes.
- `cd apps/api && go test ./...` passes.
- Touched mobile apps typecheck (no EAS builds).
- Manual trace (no build required): a Google/Apple sign-in for a new user yields a row with name + avatar; a repeat sign-in does not clobber an edited name/avatar; an unverified password signup with an email already owned by a verified social account creates a separate row.
</verification>

<success_criteria>
- Social login populates name + avatar on the user row (Workstream A).
- Same-email re-bind is gated on email_verified; unverified cannot hijack (Workstream B).
- Apple first-auth name is written to Firebase displayName and flows into later tokens (Workstream C).
- Customer profile edits persist visibly; read path consumes the flat API body; vendor/delivery audited (Workstream D).
- All required Go tests added and passing; mobile typechecks clean.
- No DB migration; no EAS builds; atomic single-line commits per task; no ROADMAP.md change.
</success_criteria>

<output>
After completion, create `.planning/quick/260617-due-social-profile-and-profile-save-fix/260617-due-SUMMARY.md`
</output>
