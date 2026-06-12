# apps/mobile-delivery — SUNSET

This Expo (React Native) **delivery-driver app** is **decommissioned**. Home Chef no longer runs its
own delivery fleet — deliveries are fulfilled by **3PL providers (Shadowfax-class)**
(owner decision 2026-06-11, PROD-READINESS Wave 7). Managing delivery agents is overkill at this scale.

**Replaced by:** nothing — own-fleet is retired. 3PL providers supply riders + live tracking
(provider webhooks → `DeliveryResponse` → customer live map, Wave 7C).

**Status:**
- No GitHub Actions CI in this repo to remove (mobile apps build via EAS, not GH workflows).
- The own-driver **API routes were retired server-side** in Wave 7A (`apps/api/routes/routes.go`):
  driver onboarding, the driver-app endpoints (`/delivery/*`), driver Stripe Connect, driver
  subscription/referral, and `ManualAssignDelivery`. So this app's backend is already gone — it
  would 404 against prod.

**Kept** in the repo for history/reference — not deleted, not built, not deployed. DB tables
(`delivery_partners`, etc.) are retained until the legacy data is irrelevant.

**Deferred (owner-controlled):** removing the EAS project / pulling any published build from the
stores, if one was ever submitted.
