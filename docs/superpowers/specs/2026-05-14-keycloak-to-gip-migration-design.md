# Keycloak → GIP Migration Design

**Date:** 2026-05-14
**Status:** Approved (brainstorming phase complete; awaiting user review of this spec)
**Repos touched:** `Home-Chef-App`, `tesserix-k8s`
**Reference implementation:** `mark8ly/services/auth-bff/`

## Goal

Replace Keycloak (currently fronting all auth for Home-Chef-App via an external BFF) with Google Identity Platform (GIP), matching the architecture mark8ly already runs. Remove every Keycloak touchpoint from code, infra, and secrets after a 14-day soak.

## Non-Goals

- TOTP MFA (deferred to a follow-on phase; GIP-native SMS available for admin pool if needed)
- OpenFGA (deferred; claim-based RBAC is sufficient for current roles)
- Keycloak user data migration (fresh-start cutover — users re-register)
- Account-linking UX for same-email-across-pools (deferred; admin tooling for now)
- Single-sign-on across Tesserix products (out of scope; pools are isolated by design)

## Decisions Locked During Brainstorming

| Decision | Choice |
|---|---|
| Ambition level | Full mark8ly architecture adoption (minus FGA + MFA) |
| User migration | Fresh start — no export/import, users re-register |
| Tenant pools | 3 — `HomeChef-Customer`, `HomeChef-Business` (vendor+driver), `HomeChef-Internal` (admin) |
| Mobile auth | `@react-native-firebase/auth` + BFF `/auth/auto-login`; drop HS256 |
| BFF placement | `apps/auth-bff/` in this monorepo (mirrors `mark8ly/services/auth-bff/`) |
| OpenFGA | Skipped — roles via GIP custom claims |
| MFA | Skipped — GIP-native SMS available later |
| Sign-in methods | Email+password, Google, Apple, Facebook, Phone OTP |
| Cutover | Hard cutover in a single ~45-min maintenance window |

## §1 — Architecture Overview

A new `homechef-auth-bff` Go service (in `apps/auth-bff/`) sits between every client and the Go API. It owns the OIDC dance with GIP, issues encrypted session cookies (web) or bearer sessions (mobile), and forwards verified identity to downstream services via HMAC-signed headers. The Go API stops minting HS256 tokens; it only verifies what the BFF sends.

```
┌── Web (web / vendor / delivery / admin) ────────┐
│   Firebase JS SDK → BFF /auth/callback          │
│   ← Set-Cookie: hc_session (AES-GCM)            │
└─────────────────────────────────────────────────┘
                       │
┌── Mobile (customer / vendor / delivery) ────────┐
│   @react-native-firebase/auth                   │
│   → BFF /auth/auto-login {id_token, tenant_id}  │
│   ← { session_token, expires_at }               │
└─────────────────────────────────────────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │   homechef-auth-bff (Go)     │
        │   apps/auth-bff/             │
        │   - OIDC callback            │
        │   - GIP token verifier       │
        │   - Session encrypt/decrypt  │
        │   - Role-claim → header map  │
        │   - /auth/{session,refresh,  │
        │      logout,switch-context}  │
        └──────────────────────────────┘
                       │
                       │  X-User-Id, X-User-Email, X-User-Role,
                       │  X-Auth-Pool (customer|business|internal),
                       │  X-Internal-Auth (HMAC) — BFF-signed
                       ▼
        ┌──────────────────────────────┐
        │   apps/api (Go API)          │
        │   - Verifies BFF HMAC header │
        │   - Trusts X-* identity      │
        │   - Authorizes via role +    │
        │     resource ownership       │
        │   - NO HS256 minting         │
        └──────────────────────────────┘
                       │
                       ▼
                  Postgres
                  (no password hashes,
                   no refresh_tokens table)
```

### Invariants

- **One source of truth for identity** = GIP tenant pools. The Go DB no longer stores passwords, refresh tokens, or password-reset tokens.
- **One source of truth for trust** = the BFF. Every Go API request carries BFF-signed headers; the API rejects unsigned requests.
- **Pool isolation enforced at the BFF** via `firebase.tenant` claim validation, not just per-app config.
- **Roles live in GIP custom claims** (`role: customer|vendor|driver|admin`). Set at signup; admin-claims sync CronJob keeps internal-pool admins authoritative.
- **Email/password lives in GIP** (`signInWithPassword`). Go API's `Register/Login/ForgotPassword` endpoints are deleted.

## §2 — Components

### 2.1 `apps/auth-bff/` (new Go service)

Mirrors `mark8ly/services/auth-bff/` layout. Strip out OpenFGA + MFA; keep everything else.

```
apps/auth-bff/
├── cmd/server/main.go              wiring: config, gip verifier, session, audit
├── internal/
│   ├── gip/verifier.go             RS256 JWT + JWKS cache + firebase.tenant validation
│   ├── session/cookie.go           AES-GCM encrypt/decrypt, HttpOnly Secure SameSite=Lax
│   ├── session/handler.go          GET /auth/session, POST /auth/logout, POST /auth/switch-context
│   ├── autologin/                  POST /auth/auto-login (mobile entry)
│   ├── oidc/                       /auth/login + /auth/callback (web OIDC redirect flow)
│   ├── productregistry/            loads homechef-products.yaml (3 pools, 4 web apps, 3 mobile apps)
│   ├── headerproxy/                signs X-* headers with HMAC for downstream services
│   └── ratelimit/                  per-IP + per-email throttling on login endpoints
├── migrations/                     1 table only: auth_audit_events (best-effort log)
├── Dockerfile
├── homechef-products.yaml          per-app config (hosts, tenant pool, OAuth client, cookie name)
└── go.mod                          module github.com/homechef/auth-bff
```

### 2.2 BFF endpoints

| Method | Path | Purpose | Used by |
|---|---|---|---|
| GET  | `/auth/login`           | Start OIDC, redirect to GIP | Web only |
| GET  | `/auth/callback`        | Receive OIDC code, mint cookie | Web only |
| POST | `/auth/exchange`        | Trade GIP id_token (from Firebase JS) → cookie | Web (email/password flow) |
| POST | `/auth/auto-login`      | Trade GIP id_token → session token | Mobile only |
| GET  | `/auth/session`         | Return current user | Web + Mobile |
| POST | `/auth/logout`          | Clear cookie/session, audit event | Both |
| POST | `/auth/switch-context`  | Re-mint session with different role | Admin tooling |
| POST | `/auth/refresh`         | Refresh session if id_token still valid | Both |
| GET  | `/auth/csrf`            | Return CSRF token (double-submit cookie) | Web only |
| GET  | `/healthz`              | K8s probe | Infra |

### 2.3 `apps/api/` (Go API) surface changes

**Delete:**
- `handlers/auth.go` → `Register`, `Login`, `OAuthLogin`, `ForgotPassword`, `ResetPassword`, `verifyGoogleToken`, `verifyFacebookToken`, `verifyAppleToken` (~750 LOC)
- `middleware/auth.go` → `GenerateTokens`, `GenerateTokensWithContext`, `extractKeycloakRoles`, the 5-method auth fallback
- `models/user.go` → `Password` column, `RefreshToken`, `PasswordResetToken`, `EmailVerificationToken` tables
- `config/config.go` → `JWT_SECRET`, `JWT_EXPIRATION_HOURS`, `REFRESH_TOKEN_DAYS`, all client-secret env vars

**Replace:**
- New `middleware/bff_auth.go` — verifies `X-Internal-Auth` HMAC, reads `X-User-*` headers into Gin context. ~80 LOC.

**Keep:**
- `User` model (drop auth columns, keep id/email/name/role/profile fields)
- All non-auth handlers — read user from context as before

### 2.4 Frontends (4 web apps)

Per app, ~3 files change:

```
src/features/auth/services/auth-service.ts   replace getLoginUrl() → Firebase JS + POST /auth/exchange
src/app/store/auth-store.ts                  drop direct API auth path; session = /auth/session response
src/lib/firebase.ts                          NEW: initialize firebase/auth pinned to pool tenant ID
```

Delete: `kc_idp_hint` query params, `kc_action=UPDATE_PASSWORD` redirects, all Keycloak realm references in code/comments.

### 2.5 Mobile apps (3 Expo apps)

```
packages/mobile-shared/auth/                 NEW directory (mirrors mark8ly's pattern)
  ├── firebase.ts                            initialize @react-native-firebase/auth + tenantId
  ├── sign-in.ts                             signIn/signOut/getIdToken/onAuthStateChanged
  ├── bff-session.ts                         POST /auth/auto-login, store bearer in SecureStore
  └── provider.tsx                           AuthProvider context

apps/mobile-{customer,vendor,delivery}/
  ├── store/auth-store.ts                    re-export from mobile-shared
  └── app/(auth)/login.tsx                   Firebase signInWithCredential instead of GoogleSignin alone
```

Delete: direct `/api/v1/auth/oauth` calls, HS256 token storage.

### 2.6 Infra (`tesserix-k8s`) summary

See §6 for full detail.

```
NEW:
charts/apps/homechef-auth-bff/
argocd/prod/apps/homechef/auth-bff.yaml
external-secrets/prod/homechef/externalsecret.yaml (extended)
terraform-new/stacks/11-identity-platform/terraform.tfvars (+3 tenants)
manifests/homechef-istio/virtualservice-auth.yaml

DELETE (after 14-day soak):
argocd/prod/infrastructure/identity-customer.yaml
argocd/prod/infrastructure/identity-internal.yaml
argocd/pilot/infrastructure/keycloak.yaml
charts/thirdparty/keycloak (if present)
sealedsecret/prod/identity/keycloak-*.yaml  (12 IdP + admin/db/redis secrets)
external-secrets/prod/identity-internal/
```

## §3 — Data Flow

### 3.1 Web sign-in (Google / Apple / Facebook social)

```
Browser              BFF                 GIP                 API
  │  GET /auth/login?app=vendor-portal&provider=google
  ├────────────────►│
  │                 │ resolve product from Host → HomeChef-Business pool
  │                 │ build OIDC URL with tenantId + state + nonce
  │  302 to GIP authorize URL
  │◄────────────────┤
  │  GIP consent + Google chooser, user picks account
  │
  │  GET /auth/callback?code=...&state=...
  ├────────────────►│
  │                 │  exchange code for id_token
  │                 ├──────────────────►│
  │                 │◄──────────────────┤  id_token (RS256, claims: sub, email, firebase.tenant)
  │                 │ verify signature via JWKS cache
  │                 │ assert firebase.tenant == HomeChef-Business
  │                 │ read role from custom claim (or default by pool)
  │                 │ upsert user row in API DB (idempotent)
  │                 ├──────────────────────────────────►│  POST /internal/users/upsert
  │                 │◄──────────────────────────────────┤  200 {user_id}
  │                 │ encrypt {uid, email, role, pool, iat, exp} → cookie
  │  Set-Cookie: hc_session=...; HttpOnly; Secure; SameSite=Lax; Domain=.fe3dr.com
  │  302 to postLoginUrl (/dashboard)
  │◄────────────────┤
```

### 3.2 Web sign-in (email + password)

```
Browser                    BFF                    GIP
  │  user enters email + password into Firebase JS form
  │  signInWithEmailAndPassword (pinned to tenant)
  ├──────────────────────────────────────────────►│
  │◄──────────────────────────────────────────────┤  id_token
  │  POST /auth/exchange  { id_token }
  ├──────────────────────────►│
  │                           │ verify + tenant check + upsert
  │  Set-Cookie: hc_session; 200 { user }
  │◄──────────────────────────┤
```

The browser does sign-in directly with GIP (Firebase JS). The BFF only exchanges the resulting id_token for a cookie. No password ever transits the BFF.

### 3.3 Mobile sign-in (any provider)

```
App                        BFF                    GIP
  │  user taps Sign in with Google
  │  @react-native-firebase/auth.signInWithCredential(googleCred)
  ├──────────────────────────────────────────────►│
  │◄──────────────────────────────────────────────┤  id_token + refresh_token (stored by Firebase SDK)
  │  POST /auth/auto-login { id_token, expected_tenant_id }
  ├──────────────────────────►│
  │                           │ verify + tenant check + upsert
  │  200 { session_token, expires_at, user }
  │◄──────────────────────────┤
  │  store session_token in SecureStore
  │  use as `Authorization: Bearer <session_token>` on API calls
```

`session_token` is the same AES-GCM blob as the web cookie; mobile carries it in a header instead of a cookie.

### 3.4 Authenticated API request (web or mobile, identical downstream)

```
Client            BFF                                  API
  │  GET /api/v1/orders
  │  Cookie: hc_session=...     (web)
  │  Authorization: Bearer ...  (mobile)
  ├──────────►│ decrypt session, validate exp
  │           │ if expired and refresh possible → refresh via GIP, re-mint
  │           │ build downstream headers:
  │           │   X-User-Id, X-User-Email, X-User-Role, X-Auth-Pool
  │           │   X-Internal-Auth: HMAC-SHA256(method+path+body+ts, shared_secret)
  │           │   X-Auth-Ts: <unix>
  │           ├────────────────────────────────────►│
  │           │                                     │ middleware/bff_auth.go:
  │           │                                     │   verify HMAC + recency (<60s)
  │           │                                     │   set ctx user_id, email, role, pool
  │           │                                     │ handler reads ctx, runs business logic
  │           │◄────────────────────────────────────┤  200 { orders }
  │◄──────────┤
```

### 3.5 Sign-out

```
Client            BFF
  │  POST /auth/logout
  ├──────────►│ clear server-side session state (LRU + audit)
  │           │ audit event → audit_service (async)
  │  200; Set-Cookie: hc_session=; Max-Age=0
  │◄──────────┤
```

The BFF holds only `viewer` roles on GIP, so it cannot forcibly revoke Firebase refresh tokens. Logout clears the session cookie and (for mobile) marks the bearer revoked in an in-memory + audit-store LRU. The Firebase refresh token expires naturally; if a revoked bearer is replayed the BFF rejects it. Forced revocation (e.g., account compromise) is handled by the `gip-admin-claims` operator job which holds the admin role and runs out-of-band.

### 3.6 Error cases

- **`firebase.tenant` mismatch** (customer pool token sent to admin app) → 401, audit event, generic error message.
- **Session expired, no refresh path** → BFF returns 401; web app redirects to `/auth/login`; mobile clears SecureStore + signs user out of Firebase SDK.
- **GIP unreachable** → BFF returns 503 with `Retry-After`; clients show generic auth-temporarily-unavailable screen, never bypass.
- **Clock skew on `X-Auth-Ts`** → 60s window with explicit error code, fixed via NTP.
- **Replay attack on `X-Internal-Auth`** → HMAC includes timestamp + path + body hash; API rejects duplicate `(ts, path, body_hash)` within 60s via in-process LRU.

## §4 — Configuration & Secrets

### 4.1 `apps/auth-bff/homechef-products.yaml`

```yaml
platformDomain: fe3dr.com

products:
  - name: homechef
    domain: fe3dr.com
    apps:
      - name: web
        hosts: ["fe3dr.com", "www.fe3dr.com", "localhost:5173"]
        gipTenantId: HomeChef-Customer-xxxxx
        oauthClientId: "<from GCP Console>"
        clientSecretEnv: HOMECHEF_CUSTOMER_CLIENT_SECRET
        sessionCookie: hc_session
        callbackPath: /auth/callback
        callbackHost: fe3dr.com
        postLoginUrl: /
        postLogoutUrl: /
        authContext: customer
        defaultRole: customer
        signInMethods: [password, google.com, apple.com, facebook.com, phone]
        allowedOrigins: ["https://fe3dr.com", "https://www.fe3dr.com", "http://localhost:5173"]

      - name: vendor-portal
        hosts: ["vendors.fe3dr.com", "localhost:5174"]
        gipTenantId: HomeChef-Business-xxxxx
        oauthClientId: "<from GCP Console>"
        clientSecretEnv: HOMECHEF_BUSINESS_CLIENT_SECRET
        sessionCookie: hc_vendor_session
        callbackPath: /auth/callback
        postLoginUrl: /dashboard
        authContext: business
        defaultRole: vendor
        signInMethods: [password, google.com, apple.com, phone]
        allowedOrigins: ["https://vendors.fe3dr.com", "http://localhost:5174"]

      - name: delivery-portal
        hosts: ["delivery.fe3dr.com", "localhost:5175"]
        gipTenantId: HomeChef-Business-xxxxx        # same pool as vendor
        oauthClientId: "<same client as vendor-portal>"
        clientSecretEnv: HOMECHEF_BUSINESS_CLIENT_SECRET
        sessionCookie: hc_driver_session
        callbackPath: /auth/callback
        postLoginUrl: /jobs
        authContext: business
        defaultRole: driver
        signInMethods: [password, google.com, phone]
        allowedOrigins: ["https://delivery.fe3dr.com", "http://localhost:5175"]

      - name: admin-portal
        hosts: ["admin.fe3dr.com", "localhost:5176"]
        gipTenantId: HomeChef-Internal-xxxxx
        oauthClientId: "<from GCP Console>"
        clientSecretEnv: HOMECHEF_INTERNAL_CLIENT_SECRET
        sessionCookie: hc_admin_session
        callbackPath: /auth/callback
        postLoginUrl: /
        authContext: internal
        defaultRole: admin
        signInMethods: [password, google.com]
        allowedEmailsEnv: HOMECHEF_ADMIN_ALLOWED_EMAILS
        allowedOrigins: ["https://admin.fe3dr.com", "http://localhost:5176"]

mobileTenantAllowlist:
  - HomeChef-Customer-xxxxx
  - HomeChef-Business-xxxxx
  # HomeChef-Internal intentionally absent — no internal mobile app
```

Vendor and driver share the `HomeChef-Business` pool (one identity per businessperson), but get different default roles based on which app onboarded them.

### 4.2 GIP / GCP setup (Terraform)

Added to `tesserix-k8s/terraform-new/stacks/11-identity-platform/terraform.tfvars`:

```hcl
tenants = [
  # ... existing mark8ly tenants stay ...
  { name = "HomeChef-Customer", display_name = "HomeChef Customer", allow_password_signup = true,  enable_email_link_signin = false },
  { name = "HomeChef-Business", display_name = "HomeChef Business", allow_password_signup = true,  enable_email_link_signin = false },
  { name = "HomeChef-Internal", display_name = "HomeChef Internal Staff", allow_password_signup = false, enable_email_link_signin = true },
]
```

Manual GCP Console steps (Terraform doesn't manage these):
- Enable Google, Apple, Facebook, Phone providers per tenant
- Register OAuth 2.0 Web client per app (4 total) with redirect URI `https://<host>/auth/callback`
- Configure authorized domains: `fe3dr.com`, `www.fe3dr.com`, `vendors.fe3dr.com`, `delivery.fe3dr.com`, `admin.fe3dr.com`
- Add Apple Sign-In service ID + key (Apple Developer)
- Add Facebook app ID + secret
- Configure SMS region + monthly quota for Phone OTP

### 4.3 Secrets (GCP Secret Manager → ExternalSecret → K8s)

| Secret name | Contents | Consumed by |
|---|---|---|
| `prod-homechef-gip-web-api-key` | Identity Toolkit browser API key | Web apps (build env), BFF |
| `prod-homechef-customer-client-secret` | OAuth client secret for customer app | BFF |
| `prod-homechef-business-client-secret` | OAuth client secret for vendor+driver | BFF |
| `prod-homechef-internal-client-secret` | OAuth client secret for admin | BFF |
| `prod-homechef-session-encrypt-key` | 32-byte random for AES-GCM | BFF |
| `prod-homechef-bff-internal-hmac-key` | 32-byte random for X-Internal-Auth | BFF + apps/api |
| `prod-homechef-admin-allowed-emails` | Comma-separated allowlist | BFF + gip-admin-claims cron |

### 4.4 BFF env vars

```
HTTP_PORT=8080
ENV=prod
LOG_LEVEL=info
PRODUCTS_CONFIG_PATH=/etc/auth-bff/homechef-products.yaml
GIP_PROJECT_ID=tesseracthub-480811
GIP_PROJECT_NUMBER=849928263410
GIP_WEB_API_KEY=<from secret>
HOMECHEF_CUSTOMER_CLIENT_SECRET=<from secret>
HOMECHEF_BUSINESS_CLIENT_SECRET=<from secret>
HOMECHEF_INTERNAL_CLIENT_SECRET=<from secret>
SESSION_ENCRYPT_KEY=<from secret, 32 bytes base64>
SESSION_COOKIE_DOMAIN=.fe3dr.com
SESSION_MAX_AGE_HOURS=168
BFF_INTERNAL_HMAC_KEY=<from secret, 32 bytes base64>
API_BASE_URL=http://homechef-api.homechef.svc.cluster.local:8080
HOMECHEF_ADMIN_ALLOWED_EMAILS=<from secret>
AUDIT_ENDPOINT=http://audit-service.global.svc.cluster.local:8080/internal/events
```

### 4.5 `apps/api` env cleanup

Deleted: `JWT_SECRET`, `JWT_EXPIRATION_HOURS`, `REFRESH_TOKEN_DAYS`, `GOOGLE_CLIENT_SECRET`, `FACEBOOK_APP_SECRET`, `APPLE_CLIENT_ID`, `APPLE_TEAM_ID`, `APPLE_KEY_ID`.

Added: `BFF_INTERNAL_HMAC_KEY`, `BFF_AUTH_TS_WINDOW_SECONDS=60`.

### 4.6 Frontend env (per Vite app)

```
VITE_BFF_URL=https://fe3dr.com
VITE_GIP_PROJECT_ID=tesseracthub-480811
VITE_GIP_API_KEY=<public>
VITE_GIP_TENANT_ID=HomeChef-Customer-xxxxx
VITE_GIP_AUTH_DOMAIN=tesseracthub-480811.firebaseapp.com
```

Removed: `VITE_GOOGLE_CLIENT_ID`, `VITE_FACEBOOK_APP_ID`, `VITE_APPLE_CLIENT_ID`.

### 4.7 Mobile env (per Expo app)

```
EXPO_PUBLIC_BFF_URL=https://fe3dr.com
EXPO_PUBLIC_GIP_TENANT_ID=HomeChef-Customer-xxxxx
EXPO_PUBLIC_GIP_API_KEY=<public>
EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID=<existing>     # KEEP — Firebase SDK needs this for Google sign-in
```

Plus `google-services.json` (Android) and `GoogleService-Info.plist` (iOS) per app.

### 4.8 Local dev

Hit real GIP, no emulator (matches mark8ly). Dedicated `Dev-HomeChef-*` tenants in the prod GCP project with `localhost` redirect URIs. `scripts/load-secrets.sh` pulls `dev-homechef-*` secrets from GCP Secret Manager into a gitignored `.env.local`. `docker-compose.yml` adds an `auth-bff` service on port 8081.

## §5 — Database Changes

### 5.1 Migration: drop password + token tables

```sql
-- up
ALTER TABLE users DROP COLUMN IF EXISTS password;
ALTER TABLE users DROP COLUMN IF EXISTS totp_secret;
ALTER TABLE users DROP COLUMN IF EXISTS totp_enabled;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
ALTER TABLE users DROP COLUMN IF EXISTS auth_provider;
ALTER TABLE users DROP COLUMN IF EXISTS provider_id;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS email_verification_tokens;
```

```sql
-- down (rollback safety; rollback past T+30m loses newly-created users — accepted per fresh-start)
ALTER TABLE users ADD COLUMN password TEXT;
ALTER TABLE users ADD COLUMN totp_secret TEXT;
ALTER TABLE users ADD COLUMN totp_enabled BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN email_verified BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN auth_provider TEXT;
ALTER TABLE users ADD COLUMN provider_id TEXT;
```

### 5.2 Migration: add GIP identity columns

```sql
ALTER TABLE users
  ADD COLUMN gip_uid          TEXT UNIQUE NOT NULL,
  ADD COLUMN gip_tenant_id    TEXT NOT NULL,
  ADD COLUMN gip_provider     TEXT NOT NULL,           -- password | google.com | apple.com | facebook.com | phone
  ADD COLUMN auth_pool        TEXT NOT NULL,           -- customer | business | internal
  ADD COLUMN role             TEXT NOT NULL,           -- customer | vendor | driver | admin
  ADD COLUMN last_login_at    TIMESTAMPTZ;

CREATE INDEX idx_users_gip_uid ON users (gip_uid);
CREATE INDEX idx_users_email_pool ON users (email, auth_pool);
CREATE UNIQUE INDEX idx_users_email_per_pool ON users (lower(email), auth_pool);

-- email no longer globally unique; uniqueness is per-pool
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
```

### 5.3 Go `User` model (after)

```go
type User struct {
    ID           uuid.UUID
    Email        string                           // uniqueness via composite index
    Name         string
    GIPUid       string    `gorm:"uniqueIndex"`
    GIPTenantID  string
    GIPProvider  string                           // "password" | "google.com" | ...
    AuthPool     AuthPool                         // customer | business | internal
    Role         UserRole                         // customer | vendor | driver | admin
    LastLoginAt  *time.Time
    // existing non-auth fields untouched (avatar, phone, addresses, etc.)
}

type AuthPool string
const (
    PoolCustomer AuthPool = "customer"
    PoolBusiness AuthPool = "business"
    PoolInternal AuthPool = "internal"
)
```

### 5.4 `POST /internal/users/upsert`

Called by BFF on every sign-in, signed with `X-Internal-Auth` HMAC:

```go
type UpsertUserRequest struct {
    GIPUid      string
    GIPTenantID string
    GIPProvider string
    AuthPool    AuthPool
    Email       string
    Name        string
    Role        UserRole          // BFF's default for this pool, or claim override
}
```

Behavior:
1. Look up by `gip_uid`. If found: update `last_login_at`, `name` (if changed), return user.
2. Not found: insert with given fields. Role is set from BFF default or GIP custom claim.
3. Idempotent — safe to retry.

This endpoint is the **only** mutator of identity fields.

### 5.5 Cross-pool same-email rule

`chef@example.com` existing in both Customer and Business pools = two separate user rows with different `gip_uid` values. Independent profiles and orders. UI doesn't link them automatically — admin tools can offer "merge" later if needed.

## §6 — Infra Changes (`tesserix-k8s`)

### 6.1 New artifacts

```
charts/apps/homechef-auth-bff/
├── Chart.yaml
├── values.yaml                                       # 2 replicas prod, 1 dev
├── values-uat.yaml
└── templates/{deployment,service,configmap-products,authorization-policy,_helpers.tpl}

argocd/prod/apps/homechef/auth-bff.yaml
external-secrets/prod/homechef/externalsecret.yaml   # extended with 7 keys
terraform-new/stacks/11-identity-platform/terraform.tfvars
manifests/homechef-istio/virtualservice-auth.yaml
```

### 6.2 Routing

**Before:**
- `identity.fe3dr.com` → Keycloak customer realm
- `internal-identity.fe3dr.com` → Keycloak internal realm
- `/bff/`, `/driver-bff/`, `/admin-bff/` path prefixes → external homechef-auth-bff

**After:**
- Single `/auth/*` prefix per host → `homechef-auth-bff.homechef.svc`
- BFF resolves product/app/pool from `Host` header against `homechef-products.yaml`
- `identity.fe3dr.com` + `internal-identity.fe3dr.com` deleted

### 6.3 Istio VirtualService (new)

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: homechef-auth
  namespace: homechef
spec:
  hosts: [fe3dr.com, www.fe3dr.com, vendors.fe3dr.com, delivery.fe3dr.com, admin.fe3dr.com]
  gateways: [istio-ingress/tesseract-gateway]
  http:
    - match: [{ uri: { prefix: /auth/ } }]
      route:
        - destination: { host: homechef-auth-bff.homechef.svc.cluster.local, port: { number: 8080 } }
      timeout: 10s
      retries: { attempts: 2, retryOn: 5xx,reset,connect-failure }
```

Existing per-app VirtualServices have `/bff/`, `/driver-bff/`, `/admin-bff/` match rules removed.

Istio AuthorizationPolicy: allow ingress only from istio-ingress namespace and from `apps/api` workload SA; deny everything else (defense in depth — HMAC is primary trust).

### 6.4 Deployment

```yaml
image:
  repository: asia-south1-docker.pkg.dev/tesseracthub-480811/ghcr-remote/homechef/auth-bff
  tag: main-<sha>                  # bumped by Kargo
replicas: 2                        # prod; 1 dev
resources:
  requests: { cpu: 50m,  memory: 128Mi }
  limits:   { cpu: 500m, memory: 256Mi }
serviceAccount:
  create: true
  annotations:
    iam.gke.io/gcp-service-account: homechef-auth-bff@tesseracthub-480811.iam.gserviceaccount.com
```

Regular K8s Deployment (not Knative) — auth-bff needs low-latency cold-start avoidance.

### 6.5 IAM

New GSA `homechef-auth-bff@tesseracthub-480811.iam.gserviceaccount.com`:
- `roles/identitytoolkit.viewer` (read GIP user records for `/auth/me/providers`)
- `roles/firebaseauth.viewer` (decode tenant pool info, JWKS access)

Admin-claims sync continues on existing `app-secrets-marketplace-prod` GSA; one config entry added to read `prod-homechef-admin-allowed-emails`.

### 6.6 Kargo

`projects/homechef` in `tesserix/kargo-manifests` exists. Add the auth-bff image to its Warehouse. Two stages: `dev` (auto-promote `main-*`) and `prod` (manual approval).

### 6.7 CI

New `apps/auth-bff/.github/workflows/auth-bff-ci.yml` mirroring the existing `homechef-web` build: `go test -race -coverprofile`, build + push to GAR, trigger Kargo Warehouse refresh.

### 6.8 Decommission targets (after 14-day soak)

```
argocd/prod/infrastructure/identity-customer.yaml
argocd/prod/infrastructure/identity-internal.yaml
argocd/pilot/infrastructure/keycloak.yaml
argocd/prod/projects/identity.yaml
charts/apps/keycloak/                                    # if present
external-secrets/prod/identity-internal/
sealedsecret/prod/identity/keycloak-*.yaml               # 12 IdP + admin/db/redis
manifests/identity-customer/                             # any remaining
manifests/identity-internal/
```

After ArgoCD prunes namespaces:
- Drop `keycloak_customer` + `keycloak_internal` databases from `global-postgres` CNPG cluster
- Remove `prod-keycloak-*` family from GCP Secret Manager
- Delete DNS records for `identity.fe3dr.com` + `internal-identity.fe3dr.com`

Estimate: **~30 files added, ~40 files deleted** in the identity layer.

### 6.9 Non-prod

`devtest` cluster — replicate pattern with `Dev-HomeChef-*` tenants in the same prod GCP project, separate OAuth clients with `localhost` redirect URIs. Same decommission timing for `argocd/pilot/infrastructure/keycloak.yaml`.

## §7 — Cutover Plan

### 7.1 Preconditions (T-0 readiness checklist)

```
[ ] §1–§6 code merged on a feature branch
[ ] 3 GIP tenants provisioned via Terraform; sign-in methods enabled
[ ] 4 OAuth clients registered with prod redirect URIs
[ ] Apple Sign-In service ID + key uploaded; Facebook app reviewed; SMS quota approved
[ ] All 7 secrets present in GCP Secret Manager
[ ] homechef-auth-bff image built and present in GAR
[ ] ExternalSecret synced (kubectl get secret homechef-auth-bff-secrets)
[ ] Staging soak: full sign-in flow on devtest for all 3 pools × 5 methods × 7 apps
[ ] Mobile builds submitted; iOS approved, Android internal ready
[ ] BFF_INTERNAL_HMAC_KEY rotated into apps/api ExternalSecret
[ ] users table backed up to GCS
[ ] User comms drafted (email + in-app banner + status page)
[ ] Rollback runbook reviewed by on-call
[ ] Two operators on the call (one driver, one observer)
```

### 7.2 Cutover sequence (~45 min, Sunday 02:00 IST)

| T | Action |
|---|---|
| **T-7d** | Email all users about cutover + maintenance window |
| **T-24h** | In-app banner active across all 4 web apps |
| **T-1h** | App Store force-update flag flipped — old mobile builds show "please update" |
| **T-15m** | Maintenance page in Cloudflare Worker; existing sessions drained |
| **T-10m** | ArgoCD: pause auto-sync on homechef + identity-customer + identity-internal |
| **T-5m** | Apply DB migration: drop password/token tables + add GIP identity columns |
| **T-0** | Apply ArgoCD: homechef-auth-bff + new VirtualServices + apps/api with HMAC env |
| **T+2m** | Smoke: `/auth/login` on each of 4 hosts, sign-in with seed account per pool |
| **T+5m** | Smoke: `/api/v1/users/me` from each frontend with new session |
| **T+8m** | Smoke: mobile auto-login on TestFlight + Play internal |
| **T+10m** | Remove maintenance page |
| **T+15m** | Resume ArgoCD sync on homechef; leave identity-* paused |
| **T+30m** | Rollback decision deadline |
| **T+45m** | Declare cutover complete; post status update |

Keycloak namespaces stay running but un-routed for 14 days. After day 14, run §6.9 decommission PR.

### 7.3 Testing

**Layer 1 — Unit (BFF, 80% coverage target)**
- `internal/gip/verifier_test.go` — valid / expired / wrong tenant / wrong issuer / clock skew / malformed
- `internal/session/cookie_test.go` — round-trip, tampered ciphertext, MaxAge
- `internal/productregistry/test.go` — host resolution, wildcards, localhost, missing host → 400
- `internal/headerproxy/test.go` — HMAC sign/verify, replay window, body-hash
- `apps/api/middleware/bff_auth_test.go` — happy path, replay rejection, ts-skew, missing header

**Layer 2 — Integration**
- Both binaries in CI, real Postgres, fake GIP token from a test JWK
- `/auth/login` → `/auth/callback` → `/auth/session` → `/api/v1/users/me` end-to-end
- Verify users row created idempotently; second sign-in updates `last_login_at` only

**Layer 3 — E2E (Playwright web, Detox mobile)**
- 17 web flows (4 apps × 5 methods, minus 3 methods admin doesn't support)
- 18 mobile flows (3 apps × 2 platforms × 3 methods)
- Cross-pool isolation test: customer pool id_token rejected by admin app
- Cross-role test: drop `users.gip_uid` row mid-session → next API call 401, web redirects

**Layer 4 — Manual on staging**
- Admin allowlist enforcement
- Phone OTP delivery (one IN, one US number)
- Apple Sign-In on physical iOS device (not simulator)
- Account takeover via tenant mismatch

**Test data:** `cust1@hc.test`, `vendor1@hc.test`, `driver1@hc.test`, `admin1@hc.test` in GIP staging tenants; passwords in on-call vault.

### 7.4 Monitoring (T-0 to T+72h)

Dashboard: `grafana.internal/d/homechef-auth-cutover`

| Metric | Source | Alert |
|---|---|---|
| `/auth/callback` success rate | BFF Prometheus | < 95% → PagerDuty |
| `/auth/callback` p99 latency | BFF Prometheus | > 2s → warn |
| GIP `verifyIdToken` error rate | BFF custom counter | > 1% → PagerDuty |
| BFF↔API HMAC failures | apps/api Prometheus | > 0.1% → PagerDuty |
| `users.gip_uid` insert rate | DB exporter | < baseline → warn |
| 401 rate on protected paths | Istio access logs | > 5% delta → warn |
| Mobile `/auth/auto-login` 4xx rate | BFF | > 2% → page mobile lead |

### 7.5 Rollback

**Window:** decision must be made by T+30m. After that, rollback loses any GIP users created since cutover.

**Triggers (any one):**
- `/auth/callback` success rate < 80% for 5 min
- GIP outright down for > 3 min (confirmed via GCP status page)
- DB migration corrupted user data
- Client-secret misconfiguration causing all social sign-ins to fail

**Steps (~15 min):**
1. ArgoCD: resume sync on identity-customer + identity-internal
2. ArgoCD: revert homechef-istio VirtualServices to pre-cutover commit
3. Re-add identity.fe3dr.com + internal-identity.fe3dr.com VirtualServices
4. external-dns re-publishes DNS records
5. apps/api rollback to pre-cutover image (keeps HS256 minting alive)
6. Mobile users on the new build see auth errors — push hotfix with reinstall instructions
7. Reverse the maintenance page; post status update

### 7.6 Post-cutover follow-ups (out of scope, tracked separately)

```
[ ] Day 14: run §6.8 decommission PR
[ ] Day 30: drop keycloak_customer + keycloak_internal databases
[ ] Day 30: remove prod-keycloak-* from Secret Manager (with rotation log)
[ ] Day 30: delete scripts/identity/keycloak-to-gip-migrate.py (fresh-start chosen)
[ ] Backlog: TOTP MFA if security review demands
[ ] Backlog: OpenFGA if cross-role authorization needs grow
[ ] Backlog: Account linking ("you also have a customer account — merge?")
```

### 7.7 User communication

**T-7d email** (subject: "Sign in changes coming to Home Chef"):
> We're upgrading our sign-in system on Sunday between 02:00 and 03:00 IST. After the upgrade, you'll need to create a new account — we're moving to Google's identity platform for stronger security. Your past orders, addresses, and chef relationships stay linked once you sign in with the same email. Vendors and drivers: please re-verify your KYC during your first sign-in.

**In-app banner T-24h:**
> Sign in unchanged today. From Sunday 02:00 IST, you'll re-register on first visit. Same email keeps your data.

**Status page T-0:**
> Maintenance window in progress. Sign-in unavailable until ~03:00 IST.

## Open Questions

None blocking. Items deferred to backlog:
- TOTP MFA (post-migration phase, gated on security review)
- OpenFGA adoption (if cross-role authorization grows)
- Account-linking UI for same-email-across-pools
- Single-sign-on across Tesserix products (out of scope)

## Implementation Plan

Once this spec is approved, the next step is `writing-plans` skill to produce a phase-by-phase task breakdown.
