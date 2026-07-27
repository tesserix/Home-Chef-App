---
slug: web-password-and-social-fixes
created: 2026-07-27
mode: quick
branch: fix/web-mobile-consistency-2
follows: 260727-web-mobile-consistency
---

# Fix the broken web features the consistency audit surfaced

Follow-up to `260727-web-mobile-consistency` (PR #802, merged `19182f56`). That pass
removed surfaces that should not exist. This one fixes surfaces that exist but are
**broken** — found by diffing every `apiClient` call in both web apps against the 433
routes the Go API actually registers.

Two endpoints are called that the API has never had, and one route is ungated.

## Findings

### 1. Change password 404s in BOTH web apps

`apps/web/src/features/customer/pages/ProfilePage.tsx:1584` and
`apps/vendor-portal/src/features/settings/pages/SettingsPage.tsx:434` both
`PUT /profile/password`. **There is no such route** — the API registers no
change-password endpoint at all. The user fills in current + new password, submits,
and gets a generic "Failed to update password".

The mobile apps are the reference and they don't have one either: mobile-customer's
"Change password" row (`(tabs)/profile.tsx`) pushes to `(auth)/forgot-password`, i.e.
email a reset link — gated on `canChangePassword` so SSO accounts never see it.

### 2. Neither web app offers password recovery at all

No forgot-password page or route in either app. `apps/web`'s login link to
`/forgot-password` was removed in PR #802 precisely because it 404'd.

Both apps already ship a written-but-**never-called** `sendPasswordReset(email)`:

- `apps/vendor-portal/.../auth-service.ts:183` — correct: posts to
  `/api/v1/auth/password-reset/request` with `app: 'vendor'`, and carries a long
  comment on why Firebase's own mailer is deliberately avoided (unauthenticated
  `firebaseapp.com` sender, Gmail spam-files it, reads as phishing).
- `apps/web/.../auth-service.ts:191` — **does exactly what that comment warns
  against**: calls Firebase `sendPasswordResetEmail` directly.

The API side is complete: `POST /auth/password-reset/request` (generic response,
anti-enumeration, rate-limited) and `GET /auth/password-reset/consume`.
`services.TenantForApp` maps `vendor|chef|business` → business tenant, everything
else → customer tenant — passing the wrong `app` silently sends nothing.

### 3. Social feed calls endpoints that don't exist, on an ungated route

`apps/web/src/features/social/pages/SocialFeedPage.tsx`:
- `GET /social/posts` — API has `/social/feed`
- `POST /social/posts/:id/save` — exists nowhere in the API
- `POST /social/posts/:id/like` — correct

`mobile-customer/hooks/useSocial.ts` uses `/v1/social/feed` + `/v1/social/posts/:id/like`
and has no save. Meanwhile `nav-items.ts` and the footer both hide the feed behind
`SOCIAL_ENABLED`, but `<Route path="feed">` is **not** gated — so the page stays
reachable by URL and fails on load.

## Tasks

1. **`apps/web` — route `sendPasswordReset` through our API.** Replace the Firebase
   call with the same `fetch` vendor-portal uses, `app: 'customer'`. Keep the
   anti-enumeration behaviour (resolve on every normal outcome).

2. **Add a ForgotPassword page + route to both web apps**, and restore the
   "Forgot password?" link on both login screens. Mirror
   `mobile-shared/screens/ForgotPasswordScreen`: email field → success state that does
   not confirm whether the address exists.

3. **Repoint both change-password forms.** Replace the broken current/new/confirm form
   with the reset-email flow, matching mobile. Keep the existing SSO gate
   (`isSocialLogin` in web; check vendor's equivalent) so password-less accounts don't
   see it.

4. **Fix the social feed.** `/social/posts` → `/social/feed`; drop the save mutation and
   its UI; gate `<Route path="feed">` on `SOCIAL_ENABLED` so it matches the nav.

5. **Reconcile the stale sunset docs.** `apps/web/SUNSET.md` says "paused,
   manual-dispatch only" but `homechef-web-build.yml` has a live `push: main` trigger
   and publishes `homechef-web-app`; `apps/vendor-portal/SUNSET.md` says its workflow
   was deleted when it still exists with `pull_request` live.

## Verification

- `tsc --noEmit`, `vite build`, `eslint src`, `vitest run` on both apps.
- Re-run the apiClient-vs-Go-routes diff: the only remaining miss should be the
  `/some/other/endpoint` fixture in `api-client.test.tsx`.
- Re-run the link-vs-route cross-check: still empty for both apps.
