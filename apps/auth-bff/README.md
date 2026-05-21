# homechef-auth-bff

Auth Backend-For-Frontend (BFF) for Home-Chef-App. Implements authentication via
Google Identity Platform (GIP) for all four web SPAs and three Expo mobile apps:

- Web SPAs use Google Identity Services (GSI) → BFF `/auth/exchange` → encrypted session cookie.
- Mobile apps use `@react-native-firebase/auth` → BFF `/auth/auto-login` → bearer session token.
- BFF forwards verified-identity headers to `homechef-api` via HMAC-signed internal calls.

Three GIP tenant pools route by audience:

- `HomeChef-Customer-rqg8a` — storefront customers (web + mobile-customer)
- `HomeChef-Business-8s8ql` — vendors and drivers (vendor/delivery portals + mobile-vendor/delivery)
- `HomeChef-Internal-gyofe` — admin staff (admin portal); allowlist-gated

Historical context: this service replaced a Keycloak deployment. See
`docs/superpowers/specs/2026-05-14-keycloak-to-gip-migration-design.md` and
`docs/superpowers/plans/2026-05-14-keycloak-to-gip-migration.md` for the
migration design and execution plan.
