# Codebase Concerns

> Generated: 2026-04-05
> Focus: Technical debt, known issues, security, performance, fragile areas

---

## 1. Security Concerns

### CRITICAL: No JWT Signature Verification on Fallback Path
- **Location:** `apps/api/middleware/auth.go:123-220`
- **Issue:** When using `x-jwt-claim-sub` header fallback, the middleware trusts the header value without verifying the source. The comment says "BFF is trusted" but there's no verification that the request actually came from the BFF vs. a direct external call.
- **Impact:** If the API is accessible directly (not behind Istio/BFF), any attacker can set `x-jwt-claim-sub` to impersonate any user.
- **Remediation:** Add source validation (e.g., check for Istio-injected headers, verify request origin, or enforce network-level isolation).

### CRITICAL: Keycloak JWT Decoded Without Signature Verification
- **Location:** `apps/api/middleware/auth.go:18-72` (`extractKeycloakRoles`)
- **Issue:** JWT payload is base64-decoded and roles extracted without cryptographic verification. Comment says "safe because BFF already validated" but this allows role escalation if the header is spoofed.
- **Impact:** An attacker who can reach the API directly can craft a JWT with `super_admin` role.

### HIGH: Super Admin Email Auto-Escalation
- **Location:** `apps/api/middleware/auth.go:213-215`
- **Issue:** `IsSuperAdminEmail()` auto-escalates certain email addresses to admin role regardless of actual permissions. If the hardcoded email list is compromised or if user registration allows those emails, it's a privilege escalation vector.

### HIGH: No Rate Limiting on Most Endpoints
- **Issue:** Rate limiting only exists on chat messages (`apps/api/handlers/chat.go:20-27`). All other endpoints (auth, orders, payments) have no rate limiting.
- **Impact:** Brute force attacks, enumeration, and abuse are possible.
- **Additional:** The chat rate limiter uses an in-memory map without cleanup, which will leak memory over time.

### MEDIUM: CORS Allows Credentials with Multiple Origins
- **Location:** `apps/api/routes/routes.go:26-39`
- **Issue:** `AllowCredentials: true` with a list of specific origins. While not wildcard, localhost origins are included in production config.

### MEDIUM: No CSRF Protection on Backend
- **Issue:** Frontend portals send CSRF tokens (`apps/delivery-portal/`, `apps/admin-portal/`) but the Go API has no CSRF validation middleware. The tokens are sent but never checked.

---

## 2. Technical Debt

### No Test Coverage
- **Issue:** Zero test files found in the entire Go codebase. No `*_test.go` files exist.
- **Impact:** No regression safety net. Any change can break existing functionality silently.
- **Remediation:** Add unit tests for handlers and services, integration tests for database operations.

### TODOs in Production Code
- `apps/api/models/chef.go:169` — `TODO: populate when delivery fee model is added`
- `apps/api/handlers/staff.go:323` — `TODO: Send email via SendGrid when integrated`
- `apps/vendor-portal/src/features/dashboard/pages/DashboardPage.tsx:509-514` — Accept/reject order mutations not implemented (hardcoded TODOs)

### Keycloak References in GIP-Migrated Codebase
- **Location:** `apps/api/middleware/auth.go` — Function named `extractKeycloakRoles`, comments reference "Keycloak"
- **Issue:** The project migrated to Google Identity Platform (GIP) but auth middleware still references Keycloak naming and patterns. This creates confusion about the actual auth provider.

### Type Assertions Without Safety Checks
- **Location:** `apps/api/middleware/auth.go:313,318,328`
- **Issue:** `userID.(uuid.UUID)`, `role.(models.UserRole)`, `user.(*models.User)` — type assertions without the `ok` check. Will panic if the context value is wrong type.
- **Impact:** Runtime panics on unexpected context state.

---

## 3. Performance Concerns

### Database Query in Auth Middleware
- **Location:** `apps/api/middleware/auth.go:102,135`
- **Issue:** Every authenticated request hits the database to load the full user record (`database.DB.First(&user, ...)`). No caching layer.
- **Impact:** High DB load under traffic. Every API call = 1 extra DB query minimum.
- **Remediation:** Add Redis-based user cache with short TTL, or use JWT claims without DB lookup for non-critical paths.

### In-Memory Rate Limiter Won't Scale
- **Location:** `apps/api/handlers/chat.go:22`
- **Issue:** `rateLimit map[uuid.UUID]time.Time` is per-process, not shared across instances. No mutex protection visible — potential data race under concurrent access.
- **Impact:** Rate limits don't work with multiple replicas. Memory leak from never-cleaned map entries.

### No Connection Pooling Configuration Visible
- **Issue:** Database connection pool settings not found in the codebase. Using GORM defaults which may not be optimal for production.

---

## 4. Fragile Areas

### Auth Middleware Fallthrough Logic
- **Location:** `apps/api/middleware/auth.go:86-220`
- **Issue:** Complex fallthrough from JWT auth to `x-jwt-claim-sub` header auth. If JWT parsing fails (expired, malformed), it silently falls through to header-based auth. This makes auth behavior unpredictable.
- **Risk:** A legitimate user with an expired JWT could be authenticated via a stale header, or vice versa.

### Auto-Provisioning Users from Headers
- **Location:** `apps/api/middleware/auth.go:136-182`
- **Issue:** Users are auto-created from `x-jwt-claim-*` headers if not found in DB. If these headers can be spoofed, arbitrary users can be created.
- **Risk:** User table pollution, potential privilege escalation.

### Payment Service — Razorpay Secret Handling
- **Location:** `apps/api/services/razorpay.go`
- **Issue:** Recent commit (935d467) hardened Razorpay secret handling by clearing from config after init, but the pattern of loading secrets into config structs before clearing is fragile.

---

## 5. Missing Features / Gaps

### No Structured Logging
- **Issue:** Uses `log.Fatalf` / `log.Println` (standard library) instead of structured logging (logrus, slog, zerolog). No request IDs, no correlation IDs in logs.
- **Impact:** Difficult to trace requests across services in production.

### No Health Check Beyond Basic Ping
- **Location:** `apps/api/handlers/health.go`
- **Issue:** Health endpoint likely only returns 200 OK without checking DB connectivity, Redis availability, or downstream service health.

### No Input Validation Framework
- **Issue:** Relies on Gin's binding tags for basic validation. No schema-based validation, no request size limits visible, no sanitization middleware.

### No Graceful Degradation
- **Issue:** If Redis or external services are unavailable, behavior is undefined. No circuit breaker patterns (except the Go API itself doesn't seem to use `gobreaker`).

---

## 6. Scaling Risks

### Single Monolithic API
- **Issue:** All functionality (auth, orders, payments, chat, delivery, social, catering, etc.) is in a single `apps/api/` Go binary with ~30 handlers.
- **Impact:** Cannot scale individual features independently. A spike in chat traffic affects order processing.
- **Note:** This is the Home-Chef-App specifically — the broader Tesserix platform uses microservices, but this app is monolithic.

### No Database Migration Tooling Visible
- **Issue:** Uses `database.DB.AutoMigrate()` (GORM auto-migration) which is not safe for production schema changes. No migration files, no versioned migrations.
- **Impact:** Schema changes in production are unpredictable. Cannot roll back migrations. Risk of data loss.

---

## 7. Dependency Risks

### Direct GCP SDK Usage Without Abstraction
- **Location:** `apps/api/services/secrets.go`, `apps/api/services/storage.go`
- **Issue:** GCP Secret Manager and Storage are used directly without interfaces. Makes testing impossible without real GCP credentials.

### No Dependency Vulnerability Scanning
- **Issue:** No `govulncheck`, no Dependabot config, no Snyk integration visible. Dependencies may have known CVEs.

---

## Summary Priority Matrix

| Priority | Issue | Effort |
|----------|-------|--------|
| P0 | JWT/header auth bypass risk | Medium |
| P0 | Zero test coverage | High |
| P1 | No rate limiting (most endpoints) | Medium |
| P1 | CSRF tokens sent but never validated | Low |
| P1 | Auto-migrate in production | Medium |
| P2 | DB query per auth request | Medium |
| P2 | Keycloak naming confusion | Low |
| P2 | In-memory rate limiter races | Low |
| P3 | No structured logging | Medium |
| P3 | Missing health check depth | Low |
