---
slug: vendor-store-submission-prep
date: 2026-08-09
status: in-progress
---

# Vendor app — App Store / Play submission prep

Get `apps/mobile-vendor` (Fe3dr Vendor, `com.tesserix.homechef.vendor`) ready to
submit, and capture fresh 6.9" screenshots from a production-backed simulator
build.

Vendor goes first because it shares ~80% of the review surface with Customer
(privacy labels, account deletion, Apple revocation, Data Safety) but carries
none of Customer's risk: no UGC (skips Apple 1.2), no consumer purchase flow,
and no test-mode listing problem.

## Tasks

- [x] Verify the production demo accounts actually reach a reviewable state
      (GIP sign-in → BFF auto-login → authenticated API reads)
- [x] Fill the App Review block in `apps/mobile-vendor/store.config.json` with
      the verified credentials
- [x] Fix `prod-sim` in `apps/mobile-vendor/eas.json` — it resolved to the
      `preview` EAS environment, which has no `NODE_AUTH_TOKEN`, so the build
      died in `eas-build-pre-install`
- [x] Record the verified account state and the Customer test-mode findings in
      `docs/store-release/README.md`
- [ ] Capture 6.9" screenshots (iPhone 17 Pro Max, 1320 × 2868)
- [ ] App Review contact phone number — pending from owner

## Out of scope / handed back

- Sign in with Apple **revocation** env in prod (`APPLE_TEAM_ID`,
  `APPLE_KEY_ID`, `APPLE_SIGNIN_PRIVATE_KEY_B64`,
  `APPLE_SERVICES_CLIENT_ID_VENDOR`) — not provisioned in
  `tesserix-k8s/charts/apps/homechef-api/`. Blocks Apple 5.1.1(v) for both apps.
- Production chef data — take *Saffron Home Kitchen* live and suspend
  *My Kitchen*. Both need an internal-pool admin; the seeded accounts cannot
  perform `/admin/*` calls.
