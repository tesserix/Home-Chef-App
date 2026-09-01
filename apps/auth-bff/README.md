# homechef-auth-bff

Auth Backend-For-Frontend (BFF) for Home-Chef-App. Implements authentication via
Zitadel (issuer `https://auth.tesserix.app`) for the customer web SPA and four
Expo mobile apps:

- Web SPA redirects to the hosted Zitadel login; the BFF runs the OIDC
  code+PKCE flow (`/auth/login` -> `/auth/callback`) and sets an encrypted
  session cookie.
- Mobile apps obtain a Zitadel id_token natively -> BFF `/auth/auto-login`
  with `{id_token, pool}` -> bearer session token.
- BFF forwards verified-identity headers to `homechef-api` via HMAC-signed internal calls.

Three auth pools route by audience:

- `customer` — storefront customers (web + mobile-customer)
- `business` — vendors and drivers (mobile-vendor / mobile-delivery)
- `internal` — admin staff; allowlist-gated via `HOMECHEF_ADMIN_ALLOWED_EMAILS`

Historical context: this service replaced a Keycloak deployment, then a Google
Identity Platform (GIP) integration. See
`docs/superpowers/specs/2026-05-14-keycloak-to-gip-migration-design.md` for the
earlier migration; GIP columns remain in the DB for rollback only.
