---
slug: web-password-and-social-fixes
created: 2026-07-27
completed: 2026-07-27
mode: quick
status: complete
branch: fix/web-mobile-consistency-2
follows: 260727-web-mobile-consistency
---

# Summary — fixed the broken web features the consistency audit surfaced

Found by diffing every `apiClient` call in both web apps against the **433 routes the Go
API actually registers**. Two endpoints were being called that the API has never had.

## What was broken

### Change password 404'd in both web apps

`PUT /profile/password` — **no such route exists**, and the API has no change-password
endpoint at all. Customers (ProfilePage) and chefs (SettingsPage) filled in current + new
password, submitted, and got "Failed to update password" / "Check your current password" —
blaming the user for a route that was never there.

**Fixed** by adopting the mobile pattern: mobile's "Change password" row opens the
forgot-password flow, because email-a-reset-link is the only mechanism the backend has.
Both web apps now do the same. The existing SSO gates were kept, so Google/Apple accounts
still see the "handled by your provider" notice instead.

### Neither web app offered password recovery

No forgot-password route in either. Both apps already shipped a written-but-never-called
`sendPasswordReset()`, and **vendor-portal already had a complete `ForgotPasswordPage.tsx`**
that nothing routed to — so a chef who forgot their password had no way back in.

**Fixed:** routed the vendor page, added the equivalent page for customers, restored the
"Forgot password?" link on both login screens.

Also corrected a real behavioural bug: `apps/web`'s `sendPasswordReset` called Firebase's
`sendPasswordResetEmail` directly — precisely what vendor-portal's copy carries a long
comment warning against (Firebase delivers from an unauthenticated
`noreply@<project>.firebaseapp.com` that Gmail files as spam, branded with the GCP project
name, containing a raw firebaseapp.com URL that reads as phishing). It now posts to our API
like the vendor and mobile apps do, with `app: 'customer'` to select the right Identity
Platform tenant — accounts are tenant-scoped, and the wrong tenant silently sends nothing.

### Social feed called endpoints that don't exist

- `GET /social/posts` → the API serves `/social/feed`. **The feed could never load.**
- `POST /social/posts/:id/save` → exists in neither the API nor mobile. Removed the
  bookmark control rather than keep a button that always failed.
- `POST /social/posts/:id/like` was already correct.

`mobile-customer/hooks/useSocial.ts` uses exactly feed + like, which is now what web uses.
Also gated `<Route path="feed">` on `SOCIAL_ENABLED` — nav and footer already gated it, so
the route had stayed reachable by URL independently of the flag.

### Stale sunset docs

Both `SUNSET.md` files asserted things that are no longer true and would mislead the next
person: `apps/web`'s claimed "manual-dispatch only, never runs automatically" when the
workflow has a live `push: main` trigger publishing `homechef-web-app`;
`apps/vendor-portal`'s claimed its workflow was deleted when it still exists with
`pull_request` live. Rewritten to describe current state, with the history kept.

## Verification

- `tsc --noEmit` clean on both apps.
- `vite build` succeeds for both.
- `eslint src` — **0 errors** on both (21 pre-existing `react-refresh` warnings each, in
  untouched design-system files).
- `vitest run` in `apps/web` — **24/24 passing**.
- **apiClient-vs-Go-routes diff re-run:** vendor-portal now **0** unmatched. web has 2, both
  verified benign — `/some/other/endpoint` is a fixture in `api-client.test.tsx`, and
  `/meal-subscriptions/:id/${action}` is type-constrained to `pause|resume|cancel`, all of
  which are real routes (an artefact of normalising `${…}` to a param).
- **link-vs-route cross-check re-run:** clean for both apps.

## Not verified

The reset emails were not sent end-to-end against a live backend — `POST
/auth/password-reset/request` needs Redis (it refuses without a single-use store) plus a
working mailer and GIP. The client contract matches vendor-portal's already-shipped call
and the API handler's expected body, but **the first real send is unproven**. Worth one
manual run per app before relying on it.

## Follow-ups

- Parity gaps from the previous pass still stand (see
  `260727-web-mobile-consistency/SUMMARY.md`) — notifications, dish search, chefs map, live
  tracking, address management, support, catering, payouts, i18n.
- `vendor-portal` still has no test files at all.
- Neither web app has a `/social` → `/feed` alias; mobile calls the same surface `/social`.
