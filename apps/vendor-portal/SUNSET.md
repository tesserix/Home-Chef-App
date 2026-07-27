# apps/vendor-portal — SUNSET

This React (Vite) vendor/chef **web portal** is **decommissioned**. Home Chef is app-only
(owner decision 2026-06-10): chefs manage their kitchen on the mobile app, not the web.

**Replaced by:** `apps/mobile-vendor` (Expo). The marketing landing at `fe3dr.com` is the only web surface.

**De-wired (2026-06-12), partially reversed since:**
- `.github/workflows/homechef-vendor-portal-build.yml` was **not** removed, contrary to what
  this file said. It still exists with its `push` trigger commented out but `pull_request`
  live, so the app is linted/built/tested on every PR that touches it — just not deployed.
- (It was never a `docker-compose.yml` dev service.)

**Kept** in the repo for history/reference — and still being maintained: the portal is
receiving fixes again as of 2026-07-27 (see `.planning/quick/260727-web-mobile-consistency/`
and `260727-web-password-and-social-fixes/`). Treat "decommissioned" above as the intent,
not the current state; verify from `.github/workflows/` before assuming this app is dead.

**Deferred to the production cutover** (owner-controlled, in `tesserix-infra` + Cloudflare — NOT done here):
retire the `homechef-vendor-portal` ArgoCD app and 301 the web route → landing. Until that flip,
the last-built image keeps serving the web UI.

**Also deferred:** removing the `vendor-portal` entry from `apps/auth-bff/homechef-products.yaml`.
Mirrors how `apps/web` was sunset (its registry entry was left in place) and the 5A note that
auth-bff cleanup happens *once the portals are actually gone*.

> ⚠️ Critical: `vendors.fe3dr.com` is the **mobile vendor app's API + auth-bff host** — it MUST keep
> serving. Only the web UI dies; the Go API (`homechef-api`) and auth-bff on that host are unaffected
> (their CI — `homechef-api-build.yml`, `homechef-auth-bff-build.yml` — is untouched).
