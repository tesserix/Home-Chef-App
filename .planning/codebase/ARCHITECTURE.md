# Architecture

**Analysis Date:** 2026-04-05

## Pattern Overview

**Overall:** Monorepo with three independent React SPA frontends and a single Go HTTP API backend. Multi-user application with separate portals for vendors (home chefs), customers (web storefront), admins (platform management), and delivery drivers.

**Key Characteristics:**
- Microarchitecture: 5 applications in single monorepo (`apps/` directory)
- Backend-for-frontend pattern: Shared Go API (`apps/api`) with consistent REST endpoints
- Client-side state management: React Query for server state, Zustand for local state
- Server-side session handling: Encrypted JWT cookies (auth BFF pattern)
- Multi-tenancy: Tenant ID extracted from JWT claims and propagated via headers

## Layers

**API Backend:**
- Purpose: RESTful HTTP API with business logic, database operations, external integrations
- Location: `apps/api/`
- Contains: Go handlers, services, database models, middleware
- Depends on: PostgreSQL, Redis (optional), NATS (optional), GCS, Razorpay, SendGrid, etc.
- Used by: All four frontends via HTTP requests

**Frontend Applications:**
- Purpose: React SPA with client-side routing, forms, state management, UI rendering
- Locations: 
  - `apps/web/` - Customer-facing storefront
  - `apps/vendor-portal/` - Home chef dashboard
  - `apps/admin-portal/` - Platform admin dashboard
  - `apps/delivery-portal/` - Delivery driver mobile app (Vite + React)
- Depends on: HTTP API, local storage, Zustand stores
- Used by: End users via browsers

## Data Flow

**Authentication Flow:**
1. User submits credentials on frontend (email/password)
2. Frontend calls `POST /api/v1/auth/login` on Go API
3. API validates credentials against database
4. API returns JWT token + sets encrypted cookie
5. Frontend stores JWT in memory + receives secure cookie
6. Subsequent requests include JWT in Authorization header or rely on cookie
7. Middleware (`middleware/auth.go`) extracts JWT and decodes claims
8. Tenant ID extracted from claims or X-Request-ID header
9. Request context enriched with user and tenant data

**API Request Flow:**
1. Frontend initiates HTTP request with JWT bearer token
2. Cloudflare (edge) routes to backend API
3. Gin router matches endpoint pattern (e.g., `/api/v1/products`)
4. Middleware stack processes request:
   - CORS validation
   - Auth middleware: JWT decoding, role extraction
   - Metrics/logging middleware
5. Handler layer parses request, calls validation, invokes service
6. Service layer executes business logic, calls repository
7. Repository executes GORM queries (tenant-scoped)
8. Response marshaled to JSON, returned to client

**Real-Time Features (NATS):**
1. Service publishes events to NATS when business logic changes (orders, notifications)
2. NotificationService subscribes to topics, sends emails/push notifications
3. Topic structure: `{feature}.{action}` (e.g., `order.created`, `payment.success`)

**State Management:**
- Server state: PostgreSQL (single shared database)
- Session state: Encrypted JWT cookie (no Redis sessions)
- Client state: Zustand stores + React Query cache
- Tenant context: X-Tenant-ID header propagated through all requests

## Key Abstractions

**Handler Pattern:**
- Purpose: HTTP request parsing, validation, response formatting
- Examples: `handlers/auth.go`, `handlers/orders.go`, `handlers/chefs.go`
- Pattern: Factory singleton (`NewAuthHandler() *AuthHandler`), method receiver pattern
- Responsibility: Only HTTP concerns, no business logic

**Service Pattern:**
- Purpose: Business logic, orchestration, external integrations
- Examples: `services/storage.go`, `services/notifications.go`, `services/razorpay.go`
- Pattern: Module-level functions (no struct), package-level state (singleton client)
- Responsibility: Complex logic, NATS publishing, email/payment APIs

**Repository Pattern (Database):**
- Purpose: Data access abstraction over GORM
- Location: `database/` directory
- Pattern: Not explicitly abstracted; handlers/services call GORM directly
- Responsibility: Query construction, tenant filtering

**Model Pattern:**
- Purpose: Domain types, request/response structs, constants
- Location: `models/` directory (26 model files)
- Examples: `models/chef.go`, `models/order.go`, `models/payment.go`
- Pattern: GORM-tagged structs for ORM mapping
- Responsibility: Type definitions only

**React Hooks Pattern (Frontend):**
- Purpose: Reusable stateful logic
- Pattern: Custom hooks using `useQuery` from React Query, `useState`, `useEffect`
- Examples: `AuthProvider` uses hooks for session management
- Responsibility: Data fetching, caching, local state

**Zustand Store Pattern (Frontend):**
- Purpose: Cross-component state management
- Examples: `auth-store.ts`, `onboarding-store.ts`
- Pattern: Create store with actions and selectors
- Responsibility: User session, form state, temporary UI state

## Entry Points

**API Entry Point:**
- Location: `apps/api/main.go`
- Triggers: Application startup
- Responsibilities:
  - Load config via `config.Load()`
  - Connect to database and run migrations
  - Initialize GCS storage, Razorpay, Redis, NATS clients
  - Setup Gin router via `routes.SetupRouter()`
  - Bind to port (default 8080)
  - Graceful shutdown (30s timeout for in-flight requests)

**Frontend Entry Points:**
- Location: `apps/{app}/src/main.tsx`
- Triggers: Browser load
- Responsibilities:
  - Mount React app to DOM element with id="root"
  - Setup strict mode for development checks
  - (web) Register service worker for PWA offline support

**Router Entry Point:**
- Location: `apps/{app}/src/app/routes/index.tsx`
- Pattern: Lazy-loaded route components with retry on chunk failures
- Responsibilities:
  - Define public/protected route structure
  - Handle auth state checks (redirect to login if not authenticated)
  - Lazy load components with Suspense boundary
  - Chunk retry: Auto-reload page once on 404 chunk failures

## Error Handling

**Strategy:** Two-layer error handling — graceful initialization, explicit error responses

**Backend Patterns:**
- Database connection: Retry loop at startup, fatals if still failing (see `main.go:24`)
- Optional services: Warn but continue if Redis/NATS unavailable (non-critical)
- Handler errors: Return HTTP error response with JSON envelope (see middleware)
- Middleware: Return 401/403 on auth failures, 400 on validation errors
- No panics in request handlers — only in `main.go` startup failures

**Frontend Patterns:**
- Auth errors: Redirect to login page with error query param
- Network errors: Toast notification via Sonner (bottom-right)
- Chunk load failures: Automatic page reload once per session (see `lazyWithRetry()`)
- Error boundary: Handled implicitly via Suspense fallback (LoadingScreen)

## Cross-Cutting Concerns

**Logging:**
- Backend: Standard Go `log` package to stdout (JSON-formatted via cloud logging)
- Frontend: `console.log` avoided in production (linting rule: warn if present)

**Validation:**
- Input: Gin binding tags (`binding:"required,email"`) + custom validators in handlers
- Tenant: Middleware validates tenant presence in JWT claims
- API contracts: TypeScript types in frontend enforce request/response shapes

**Authentication:**
- Provider: Custom JWT-based (no external provider like Keycloak in this version)
- Token format: JWT with claims: `sub` (user ID), `tenant_id`, `email`, `roles`
- Verification: JWT signature validation in middleware (see `middleware/auth.go`)
- Sessions: Encrypted cookie store (browser-level, no server-side session DB)
- MFA/2FA: Not detected in codebase

**Authorization:**
- Pattern: Role-based (extracted from JWT claims)
- RBAC locations: Not detected; middleware extracts roles but no explicit permission checks
- Vendor/Admin separation: Implicitly via different frontends and database rows
- Multi-tenancy: Application-level via `tenant_id` column + header propagation

---

*Architecture analysis: 2026-04-05*
