# apps/vendor-portal — SUNSET

This React (Vite) vendor/chef **web portal** is **decommissioned**. Home Chef is app-only
(owner decision 2026-06-10): chefs manage their kitchen on the mobile app, not the web.

**Replaced by:** `apps/mobile-vendor` (Expo). The marketing landing at `fe3dr.com` is the only web surface.

**De-wired (2026-06-12):**
- Removed `.github/workflows/homechef-vendor-portal-build.yml` (no more image builds/deploys for this app).
- (It was never a `docker-compose.yml` dev service.)

**Kept** in the repo for history/reference — not deleted, not built, not deployed.

**Deferred to the production cutover** (owner-controlled, in `tesserix-infra` + Cloudflare — NOT done here):
retire the `homechef-vendor-portal` ArgoCD app and 301 the web route → landing. Until that flip,
the last-built image keeps serving the web UI.

**Also deferred:** removing the `vendor-portal` entry from `apps/auth-bff/homechef-products.yaml`.
Mirrors how `apps/web` was sunset (its registry entry was left in place) and the 5A note that
auth-bff cleanup happens *once the portals are actually gone*.

> ⚠️ Critical: `vendors.fe3dr.com` is the **mobile vendor app's API + auth-bff host** — it MUST keep
> serving. Only the web UI dies; the Go API (`homechef-api`) and auth-bff on that host are unaffected
> (their CI — `homechef-api-build.yml`, `homechef-auth-bff-build.yml` — is untouched).
