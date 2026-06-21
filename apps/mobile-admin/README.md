# Fe3dr Admin (mobile-admin)

Native admin/owner console for the Home Chef platform — iOS + Android, built
with Expo (React Native), `@tesserix/native`, and `@homechef/mobile-shared`. It
mirrors the web admin portal's capabilities and talks to the same Go API via the
auth-bff (`/auth/auto-login` for the session, `/api/v1/admin/*` for data).

## Features

- **Dashboard** — platform stats (revenue, orders, chefs, users), pending
  approvals callout, recent activity.
- **Chefs / Kitchens** — search + filter, detail, verify / reject / suspend.
- **Orders** — search + filter, full order detail (items, totals, customer, chef).
- **Approvals** — kitchen/document review queue with approve / reject /
  request-info, plus the home-chef review aids (non-home-kitchen +
  FSSAI-looks-commercial warnings).
- **Users** — search + filter, suspend / activate, jump to wallet.
- **FSSAI Lockouts** — locked & overridden lists, grant / clear temporary overrides.
- **Reviews** — moderate (hide with reason / unhide).
- **Wallets** — look up a customer wallet, view the ledger, credit / debit adjust.
- **Meal Plans** — subscription oversight (read-only).
- **Delivery** — fleet stats, partners, deliveries.
- **Staff** — team list, invite, deactivate / reactivate.
- **Analytics** — overview + orders-by-status.
- **Settings** — platform settings (read) + account.

## Auth

Firebase (GIP) sign-in against the **internal** tenant
`HomeChef-Internal-gyofe` → `/auth/auto-login` mints a BFF session whose role is
`admin` (the internal pool maps to `defaultRole: admin`). The session token is
sent as a Bearer header to `/api/v1/admin/*`, which is gated by
`bffAuth + RequireAdmin`.

> The tenant must be present in the auth-bff `mobileTenantAllowlist`
> (`apps/auth-bff/homechef-products.yaml`). This is added — redeploy auth-bff
> for it to take effect (the file is baked into the image).

## One-time setup (before building)

1. **Firebase config** — drop `GoogleService-Info.plist` (iOS) and
   `google-services.json` (Android) for the `com.tesserix.homechef.admin` app
   into this folder. (Both are gitignored.)
2. **EAS project** — `eas init` (fills `extra.eas.projectId` in `app.json`).
3. **Google OAuth (optional)** — create an iOS OAuth client for
   `com.tesserix.homechef.admin`; set its id in `eas.json`
   (`EXPO_PUBLIC_GOOGLE_IOS_CLIENT_ID`) and the reversed-client `iosUrlScheme`
   in `app.json`. Email/password sign-in works without this (the only path that
   works on an unsigned simulator build).
4. **Install** — from the repo root:
   `NODE_AUTH_TOKEN=$(gcloud secrets versions access latest --secret=prod-ghcr-token --project=tesseracthub-480811) pnpm install`

## Run (iOS simulator, against local API + BFF)

```bash
# build profile env points at localhost:8090 (BFF) / :8090/api/v1 (API)
eas build --profile local-sim --platform ios --local   # produces a .app
# install the .app on a booted simulator + launch Metro
pnpm start
```

For prod APIs use the `prod-sim` profile (`https://admin.fe3dr.com`).

## Notes

- Design follows `.impeccable.md`: ink-centric (one accent), hairlines, Geist +
  Inter, tabular numerals, functional colour only.
- No backend changes beyond the allowlist entry — all data comes from existing
  `/admin/*` endpoints.
