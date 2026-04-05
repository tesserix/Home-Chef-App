# Phase 1: Foundation + Auth - Context

**Gathered:** 2026-04-05
**Status:** Ready for planning

<domain>
## Phase Boundary

All three Expo mobile apps launch, resolve monorepo packages, store tokens securely, and users can sign in and create accounts. This phase delivers the infrastructure foundation and working authentication — no feature screens beyond auth.

</domain>

<decisions>
## Implementation Decisions

### Monorepo Structure
- **D-01:** App directories named by Claude's discretion (e.g., `apps/mobile-customer/`, `apps/mobile-vendor/`, `apps/mobile-driver/` or similar clear convention)
- **D-02:** Shared mobile code lives in `packages/mobile-shared/` — pnpm-workspace.yaml already covers `packages/*`
- **D-03:** Shared package includes everything reusable: typed API client, response types matching Go API, auth hooks, storage utils, AND shared screens (auth, onboarding) and navigation utilities
- **D-04:** Metro config in each app must include `watchFolders` pointing to monorepo root and `nodeModulesPaths` for pnpm symlink resolution — this is non-negotiable for `@tesserix/native` and `packages/mobile-shared/` imports

### Auth Strategy
- **D-05:** Direct JWT auth via `POST /api/v1/auth/login` — store JWT in expo-secure-store, send as `Bearer` header on all requests. Same pattern as web apps.
- **D-06:** JWT auto-refresh via axios interceptor — intercept 401, call refresh endpoint, retry original request
- **D-07:** Social login (Google/Apple) approach at Claude's discretion — recommend native SDKs (expo-apple-authentication + Google Sign-In) sending ID token to Go API
- **D-08:** Biometric auth trigger at Claude's discretion — recommend optional setting with unlock on app resume from background
- **D-09:** FCM device tokens registered using `getDevicePushTokenAsync()` (raw FCM tokens), NOT `getExpoPushTokenAsync()` — existing `push.go` uses raw FCM tokens via `sendToToken()`

### Design System
- **D-10:** Import design tokens (colors, spacing, typography) directly from `@tesserix/native` package — same tokens as web
- **D-11:** Use NativeWind (Tailwind CSS for React Native) for styling — team already knows Tailwind v4 from web apps, keeps class names consistent across platforms

### EAS Build
- **D-12:** Bundle identifiers: `com.homechef.customer`, `com.homechef.vendor`, `com.homechef.driver`
- **D-13:** Build approach at Claude's discretion — recommend EAS managed builds with `projectRoot` per app, dev client build for driver app (background location blocks Expo Go)
- **D-14:** Deep link URI schemes: distinct per app (e.g., `homechef-customer://`, `homechef-vendor://`, `homechef-driver://`) — baked into binary, cannot be changed via OTA

### Claude's Discretion
- App directory naming convention (D-01)
- Social login SDK approach (D-07)
- Biometric auth UX pattern (D-08)
- EAS build strategy details (D-13)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Context
- `.planning/PROJECT.md` — Project vision, constraints, key decisions
- `.planning/REQUIREMENTS.md` — FOUND-01 through FOUND-06, AUTH-01 through AUTH-07

### Research Findings
- `.planning/research/STACK.md` — Recommended Expo SDK version, libraries with versions, Metro config pattern
- `.planning/research/ARCHITECTURE.md` — Monorepo structure, shared code strategy, navigation architecture
- `.planning/research/PITFALLS.md` — Phase 1 blockers: Metro config, EAS monorepo root, FCM token type, deep link schemes
- `.planning/research/FEATURES.md` — Table stakes features list, dependency chain

### Existing Codebase
- `apps/api/middleware/auth.go` — JWT validation, x-jwt-claim-sub fallback, auto-provisioning, Keycloak role extraction
- `apps/api/handlers/auth.go` — Login, register, password reset endpoints
- `apps/api/services/push.go` — FCM push notification service using raw FCM tokens via `sendToToken()`
- `apps/api/routes/routes.go` — All API routes including `PUT /profile/device-token` for FCM token registration
- `.planning/codebase/STACK.md` — Current tech stack (Vite, React 19, Tailwind v4, Radix UI)
- `.planning/codebase/CONCERNS.md` — Security concerns including JWT/header auth bypass risk

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `apps/api/middleware/auth.go` — Auth middleware handles both JWT Bearer and x-jwt-claim-sub header auth. Mobile apps will use the JWT Bearer path directly.
- `apps/api/services/push.go` — FCM push service already built with `SendPushNotification()` and `SendPushToMultiple()`. Uses raw FCM tokens stored in `users.fcm_token` column.
- `apps/web/src/shared/` — Web app shared utilities (API client patterns, auth store, hooks) can be studied for mobile equivalents
- `apps/vendor-portal/src/shared/services/api-client.ts` — Typed API client with CSRF support, auth token injection, error handling. Pattern to replicate for mobile.

### Established Patterns
- All web apps use Zustand v5 for client state + React Query for server state — mobile should match
- API responses follow consistent JSON envelope: `{"error": "code", "message": "human-readable"}` for errors
- Auth tokens: JWT with userId, email, role claims. Refresh tokens stored in DB.
- CORS config allows specific origins — may need to add mobile deep link origins or use wildcard for native apps

### Integration Points
- `POST /api/v1/auth/login` — Login endpoint returns JWT + refresh token
- `POST /api/v1/auth/register` — Registration endpoint
- `POST /api/v1/auth/refresh` — Token refresh
- `PUT /api/v1/profile/device-token` — FCM token registration (body: `{"token": "..."}`)
- `POST /api/v1/auth/google` / `POST /api/v1/auth/apple` — Social login endpoints (if they exist, need verification)

</code_context>

<specifics>
## Specific Ideas

- Same color tokens and theme as web — user explicitly requested brand consistency across platforms
- NativeWind chosen specifically because team already knows Tailwind from web apps — minimize learning curve
- `packages/mobile-shared/` should include shared auth screens and onboarding screens, not just utilities — maximizes code reuse across the 3 apps

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-foundation-auth*
*Context gathered: 2026-04-05*
