# apps/delivery-portal — SUNSET

This React (Vite) delivery-driver **web portal** is **decommissioned**. Home Chef no longer runs
its own delivery fleet — deliveries are fulfilled by **3PL providers (Shadowfax-class)**
(owner decision 2026-06-11, PROD-READINESS Wave 7). There are no own-drivers to serve.

**Replaced by:** nothing — own-fleet is retired. 3PL riders are managed by the provider; tracking
flows in via provider webhooks → the customer app's live map.

**De-wired (2026-06-12):**
- Removed `.github/workflows/homechef-delivery-portal-build.yml` (no more image builds/deploys).
- (It was never a `docker-compose.yml` dev service.)

**Related (Wave 7A):** the own-driver API routes (`/delivery/*`, `/driver/*` onboarding, driver
Stripe/subscription, `ManualAssignDelivery`) were retired server-side; admin/staff partner views
are read-only. The companion mobile app `apps/mobile-delivery` is also sunset (see its SUNSET.md).

**Kept** in the repo for history/reference — not deleted, not built, not deployed.

**Deferred to the production cutover** (owner-controlled, in `tesserix-infra` + Cloudflare — NOT done
here): retire the `homechef-delivery-portal` ArgoCD app and 301 `delivery.fe3dr.com` → landing.
Confirm nothing API-critical rides `delivery.fe3dr.com` before teardown. The `delivery-portal`
entry in `apps/auth-bff/homechef-products.yaml` is left in place until then (mirrors `apps/web`).
