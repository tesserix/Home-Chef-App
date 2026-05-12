<!-- GSD:project-start source:PROJECT.md -->
## Project

**Home Chef Mobile Apps**

Three native mobile apps (Customer, Vendor/Chef, Delivery Driver) for the Home Chef platform, built with Expo (React Native) and the existing `@tesserix/native` design system. The apps replicate full feature parity with the existing web portals (`apps/web/`, `apps/vendor-portal/`, `apps/delivery-portal/`) while adding mobile-native capabilities like GPS/maps for live delivery tracking and navigation. All apps consume the existing Go API backend.

**Core Value:** Customers can browse home chefs, order food, and track delivery on their phones — while chefs manage orders and drivers navigate deliveries natively.

### Constraints

- **Stack**: Expo (React Native) with TypeScript — matches existing frontend skill set
- **Design System**: Must use `@tesserix/native` — brand consistency across platforms
- **Repo**: Apps live inside existing monorepo at `apps/mobile-customer/`, `apps/mobile-vendor/`, `apps/mobile-delivery/`
- **Platforms**: iOS and Android (both required)
- **API**: No backend changes — consume existing Go API as-is
- **Budget**: Use EAS Build for CI/CD (Expo's managed build service)
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Languages
- Go 1.26.1 - Backend API (`apps/api/go.mod`), all service logic
- TypeScript 5.7.2 - All frontend applications and React components
- YAML - Docker Compose, Kubernetes manifests
- JavaScript (Node.js 22.x) - Runtime for all Next.js/Vite applications
- Bash - Development scripts, Docker setup
## Runtime
- Go 1.26.1 (backend services)
- Node.js 22.x (all frontend applications - Vite, React)
- Docker (multi-stage containerization for all services)
- Alpine Linux 3.19 (production container images)
- npm (frontend apps: web, admin-portal, vendor-portal, delivery-portal)
- Go modules (backend)
## Frameworks
- Gin v1.10.0 (`apps/api/go.mod`) - HTTP framework for all Go services
- GORM v1.25.12 + PostgreSQL driver - ORM for database operations
- PostgreSQL 16 - Primary data store (via docker-compose)
- Vite 6.0.3 - Build tool and dev server (web, admin-portal, vendor-portal, delivery-portal)
- React 19.0.0 - UI rendering for all frontend applications
- React Router v7.1.0 - Client-side routing
- Radix UI (comprehensive primitives: avatar, checkbox, dialog, dropdown, label, select, separator, slot, switch, tabs, toast, tooltip)
- Tailwind CSS v4.x - Utility-first CSS framework
- shadcn/ui (via `@tesserix/web` v1.2.0) - Pre-built Radix UI + Tailwind components
- Framer Motion v11.15.0 - Animations
- Lucide React v0.468.0 - Icon library
- `class-variance-authority` v0.7.1 - Component variant management
- `clsx` v2.1.1 - Conditional class names
- Zustand v5.0.2 - Client state (web, admin-portal, vendor-portal, delivery-portal)
- TanStack React Query v5.62.8 - Server state and caching (web, admin-portal, vendor-portal, delivery-portal)
- React Context - Built-in context API usage
- React Hook Form v7.54.1 - Form handling across all frontends
- Zod v3.24.1 - TypeScript-first schema validation
- `@hookform/resolvers` v3.9.1 - Integration with Zod
- date-fns v4.1.0 - Date manipulation
- axios v1.13.0 - HTTP client for API requests
- qrcode.react v4.2.0 - QR code generation
- sonner v1.7.1 - Toast notifications
- tailwind-merge v2.6.0 - Tailwind class merging
## Key Dependencies
- `cloud.google.com/go/secretmanager` v1.16.0 - GCP Secret Manager for vendor payment secrets
- `cloud.google.com/go/storage` v1.61.2 - Google Cloud Storage (GCS) for file uploads
- `github.com/golang-jwt/jwt/v5` v5.2.1 - JWT token handling
- `github.com/google/uuid` v1.6.0 - UUID generation
- `github.com/joho/godotenv` v1.5.1 - Local environment file loading
- `github.com/lib/pq` v1.10.9 - PostgreSQL driver
- `github.com/redis/go-redis/v9` v9.18.0 - Redis client for caching
- `github.com/nats-io/nats.go` v1.47.0 - NATS messaging (event publishing/subscription)
- `github.com/prometheus/client_golang` v1.23.2 - Prometheus metrics
- `golang.org/x/crypto` v0.48.0 - Cryptographic utilities
- `github.com/gin-contrib/cors` v1.7.2 - CORS middleware for Gin
- `@tanstack/react-query-devtools` v5.62.8 - React Query debugging tools
- `@tesserix/web` v1.2.0 - Shared design system component library
- ESLint v9.17.0 - Linting
- Vitest v2.1.8 - Unit testing framework
- `@vitest/coverage-v8` v2.1.8 - Code coverage reporting
- TypeScript ESLint v8.18.0 - TypeScript linting
- Prettier v3.4.2 - Code formatting
## Configuration
- `.env.example` - Template for all required environment variables
- `config/config.go` - Centralized configuration loading via environment variables
- `Dockerfile` - Multi-stage build for containerization
- Key env vars required:
- `vite.config.ts` - Vite build configuration (present in each app)
- `tsconfig.json` - TypeScript configuration for each app
- `docker-compose.yml` (root) - Local development orchestration
- postgres:16-alpine (database)
- redis:7-alpine (caching)
- nats:2.10-alpine (messaging with JetStream)
- Go API backend
- Vite dev servers for frontends
- Adminer (optional DB UI)
- Mailhog (optional email testing)
## Storage & Persistence
- PostgreSQL 16 (via `postgres:16-alpine` in docker-compose)
- Connection pool: 5 max open, 2 idle
- GORM ORM handles migrations and queries
- Redis 7 (via `redis:7-alpine` in docker-compose)
- Client: `github.com/redis/go-redis/v9`
- Used for: Session caching, general app caching
- Google Cloud Storage (GCS) for production
- GCP Secret Manager for production secrets
## Messaging & Event Systems
- NATS 2.10 with JetStream enabled (via `nats:2.10-alpine`)
- Client: `github.com/nats-io/nats.go`
- Usage: Event publishing/subscription for async workflows
- Subjects defined: `orders.*`, `chef.*`, `delivery.*`, `payments.*`, `users.registered`, `notifications.*`, `approvals.*`, `driver.*`, `subscription.*`, etc.
## Authentication
- Google OAuth2 - `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`
- Facebook OAuth - `FACEBOOK_APP_ID`, `FACEBOOK_APP_SECRET`
- Apple Sign-In - `APPLE_CLIENT_ID`, `APPLE_TEAM_ID`, `APPLE_KEY_ID`
- JWT (JSON Web Tokens) via `github.com/golang-jwt/jwt/v5`
- Secret: `JWT_SECRET` (configured in environment)
- Expiration: `JWT_EXPIRATION_HOURS` (default: 24h)
- Refresh tokens: `REFRESH_TOKEN_DAYS` (default: 30 days)
## Platform Requirements
- Go 1.26.1
- Node.js >= 22.0.0
- pnpm >= 9.0.0 (for monorepo management)
- Docker & Docker Compose (for local services)
- PostgreSQL 16 client tools (psql)
- Git
- Docker container runtime
- PostgreSQL 16 database
- Redis 7 server
- NATS 2.10 server with JetStream
- GCP credentials for:
- Environment variables injected at runtime
## Build & Development
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Naming Patterns
- React components: PascalCase with `.tsx` extension (e.g., `LoginPage.tsx`, `AuthProvider.tsx`)
- Services: camelCase with `-service.ts` suffix (e.g., `auth-service.ts`, `api-client.ts`, `upload-service.ts`)
- Hooks: `use` prefix in camelCase (e.g., `useMobile.ts`, `useDraftForm.ts`, `useAuth()` hook function)
- Utilities: camelCase with descriptive names (e.g., `cn.ts`, `animations.ts`)
- Types/Models: `.ts` files in `shared/types/` directory (e.g., `auth.ts`, `index.ts`)
- Go handlers: `{resource}.go` in `handlers/` (e.g., `orders.go`, `chefs.go`)
- Go models: PascalCase structs in `models/{resource}.go` (e.g., `user.go`, `chef.go`)
- React components: PascalCase (e.g., `export function LoginPage()`, `export function AuthProvider()`)
- Async service methods: camelCase with action verb (e.g., `getSession()`, `refreshSession()`, `logout()`)
- Handler methods in Go: PascalCase (e.g., `CreateOrder()`, `GetChefs()`)
- Custom hooks: `use` prefix in camelCase (e.g., `useIsMobile()`, `useBreakpoint()`, `useOnlineStatus()`)
- Utility functions: camelCase, declarative names (e.g., `getBreakpoint()`, `buildUrl()`)
- React component props: camelCase (e.g., `className`, `children`, `isLoading`)
- State variables: camelCase (e.g., `isAuthenticated`, `needsOnboarding`, `onboardingStatus`)
- Constants in Go: PascalCase with prefixes for type (e.g., `RoleCustomer`, `ProviderGoogle`, `RoleAdmin`)
- Loop variables: short camelCase or underscore (e.g., `i`, `_`)
- Destructured values: camelCase matching property names (e.g., `const { user, isAuthenticated } = ...`)
- TypeScript interfaces: PascalCase with `Props` suffix for component props (e.g., `AuthContextValue`, `CreateOrderRequest`, `DashboardStats`)
- Go structs: PascalCase (e.g., `User`, `ChefProfile`, `Order`)
- Go constants for enum-like values: PascalCase (e.g., `RoleChef`, `ProviderEmail`, `StatusApproved`)
- Type unions in TypeScript: camelCase with meaningful descriptors (e.g., `OnboardingStatus = 'not_started' | 'in_progress' | 'submitted'`)
## Code Style
- TypeScript/JavaScript: Prettier 3.4.2 with `prettier-plugin-tailwindcss` enabled
- Go: Standard Go formatting via `gofmt`
- Line length: Prettier default (80 characters for most content)
- Indentation: 2 spaces (TypeScript), implicit via gofmt (Go)
- TypeScript/JavaScript: ESLint 9.17.0 with typescript-eslint
- Config location: `eslint.config.js` per app (e.g., `apps/vendor-portal/eslint.config.js`)
- Key rules: React Hooks linting, React Refresh warnings for component exports, unused variables flagged if not prefixed with `_`
- No explicit Prettier config file; uses Prettier defaults with Tailwind CSS class sorting
- Automatic class sorting via `prettier-plugin-tailwindcss`
- Classes are automatically reordered on save/format
- Utility-first approach across all frontends
- Custom CSS in `styles/` directory for animations and global overrides
## Import Organization
- `@/*` maps to `src/` (configured in `tsconfig.json`)
- All relative imports use `@/` prefix for cross-feature imports
- Local imports use relative paths (e.g., `../store/auth-store`)
- Standard library imports first (e.g., `"context"`, `"log"`, `"time"`)
- Third-party packages next (e.g., `"github.com/gin-gonic/gin"`)
- Internal imports last (e.g., `"github.com/homechef/api/models"`)
## Error Handling
- Async operations use try-catch blocks with proper error narrowing
- Error type is `unknown`, narrowed via `instanceof Error` check
- API errors wrapped in `ApiError` interface with `success: false`, `error: { code, message }`
- Silent catches for non-critical operations (e.g., logout attempt, CSRF token fetch)
- Session errors (401) clear auth state via Zustand store without hard redirect
- Functions return `error` as last return value
- Errors checked immediately with early return pattern: `if err != nil { c.JSON(...); return }`
- Validation errors return 400 with `gin.H{"error": "message"}`
- Database errors return 400 with descriptive messages (e.g., "Chef not found or not available")
- No panics in handlers; only in `main()` for startup failures
## Logging
- TypeScript: `console` object for development logging (no logger library imported in samples)
- Go: Standard `log` package for startup/shutdown events
- All logs to stdout for container collection
- Development: Text format via `log.Printf()`
- Console logs avoided in production code per linting rules (would be flagged by post-tool hooks)
- Structured logging in async operations: log error context on try-catch failure
## Comments
- JSDoc comments on exported functions and services (2-3 line descriptions)
- TypeScript methods in classes: `//` comments explaining complex logic
- Go code: Comments on exported types/functions per Go convention
- Inline comments for non-obvious algorithms or workarounds
- Used extensively in service files (e.g., `auth-service.ts`, `api-client.ts`)
- Format: `/** @param ... @returns ... */` for functions with complex signatures
- Example: `/** Build the OIDC login URL. Redirects to Keycloak via BFF. @param provider - Optional social provider @param returnTo - URL to redirect back to after login */`
- Exported types/functions must have comment starting with function/type name
- Package-level comments in main handler files
- Example: `// CreateOrder creates a new order`
## Function Design
- Target: < 50 lines per function
- Complex handlers split into smaller private helper methods
- Example: `CreateOrder()` is ~150 lines but includes full validation + item calculation inline
- TypeScript: Use interfaces for multiple related parameters (e.g., `CreateOrderRequest` with 10 fields)
- Go: Struct binding with Gin (e.g., `c.ShouldBindJSON(&req)`)
- Callbacks: Typed explicitly (e.g., `onSelect: (id: string) => void`)
- Async functions return `Promise<T | null>` when nullability is intentional
- Go functions return `(Result, error)` tuple; error is last
- React components return JSX with proper typing
## Module Design
- Named exports for utilities (e.g., `export function useIsMobile()`, `export const apiClient = ...`)
- Re-exports for barrel files: `export { cn } from '@tesserix/web'`
- Zustand stores exported as `export const useAuthStore = create<AuthState>(...)`
- `shared/types/index.ts` re-exports all types via `export * from './auth'`
- Service aggregation: `shared/services/` has individual service exports
## TypeScript Configuration
- `noUnusedLocals`: true — unused variables cause compile error (prefix with `_` to ignore)
- `noUnusedParameters`: true — unused parameters cause compile error
- `noFallthroughCasesInSwitch`: true — switch statements must have break/return
- `noUncheckedIndexedAccess`: true — array/object access must be checked for undefined
- Let TypeScript infer obvious local variable types
- Explicit types required for exported functions and component props
- Function return types explicitly declared
## React Patterns
- Functional components with hooks (no class components)
- Props typed via named interface with `Props` suffix or as destructured parameters
- Custom hooks extracted for reusable logic (e.g., `useMobile`, `useOnlineStatus`)
- Context Providers with custom hooks (e.g., `AuthProvider` with `useAuth()` hook)
- Zustand for global state (auth, onboarding form data)
- React Query for server state (list queries, mutations)
- Local useState for UI state (modals, form input)
## Go Patterns
- Receive `*gin.Context` parameter
- Parse request via `c.ShouldBindJSON(&request)`
- Validate immediately after binding
- Return JSON response via `c.JSON(statusCode, gin.H{...})`
- Early return on error (no error handling chains)
- Encapsulate business logic separate from handlers
- Async patterns via goroutines with channels (if needed)
- Dependency injection via constructor (e.g., `NewOrderHandler()`)
- GORM struct tags for database mapping: `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
- JSON struct tags for API responses: `json:"id"` with `,omitempty` for optional fields
- Relationships via pointers and `gorm:"foreignKey:..."` tags
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## Pattern Overview
- Microarchitecture: 5 applications in single monorepo (`apps/` directory)
- Backend-for-frontend pattern: Shared Go API (`apps/api`) with consistent REST endpoints
- Client-side state management: React Query for server state, Zustand for local state
- Server-side session handling: Encrypted JWT cookies (auth BFF pattern)
- Multi-tenancy: Tenant ID extracted from JWT claims and propagated via headers
## Layers
- Purpose: RESTful HTTP API with business logic, database operations, external integrations
- Location: `apps/api/`
- Contains: Go handlers, services, database models, middleware
- Depends on: PostgreSQL, Redis (optional), NATS (optional), GCS, Razorpay, SendGrid, etc.
- Used by: All four frontends via HTTP requests
- Purpose: React SPA with client-side routing, forms, state management, UI rendering
- Locations: 
- Depends on: HTTP API, local storage, Zustand stores
- Used by: End users via browsers
## Data Flow
- Server state: PostgreSQL (single shared database)
- Session state: Encrypted JWT cookie (no Redis sessions)
- Client state: Zustand stores + React Query cache
- Tenant context: X-Tenant-ID header propagated through all requests
## Key Abstractions
- Purpose: HTTP request parsing, validation, response formatting
- Examples: `handlers/auth.go`, `handlers/orders.go`, `handlers/chefs.go`
- Pattern: Factory singleton (`NewAuthHandler() *AuthHandler`), method receiver pattern
- Responsibility: Only HTTP concerns, no business logic
- Purpose: Business logic, orchestration, external integrations
- Examples: `services/storage.go`, `services/notifications.go`, `services/razorpay.go`
- Pattern: Module-level functions (no struct), package-level state (singleton client)
- Responsibility: Complex logic, NATS publishing, email/payment APIs
- Purpose: Data access abstraction over GORM
- Location: `database/` directory
- Pattern: Not explicitly abstracted; handlers/services call GORM directly
- Responsibility: Query construction, tenant filtering
- Purpose: Domain types, request/response structs, constants
- Location: `models/` directory (26 model files)
- Examples: `models/chef.go`, `models/order.go`, `models/payment.go`
- Pattern: GORM-tagged structs for ORM mapping
- Responsibility: Type definitions only
- Purpose: Reusable stateful logic
- Pattern: Custom hooks using `useQuery` from React Query, `useState`, `useEffect`
- Examples: `AuthProvider` uses hooks for session management
- Responsibility: Data fetching, caching, local state
- Purpose: Cross-component state management
- Examples: `auth-store.ts`, `onboarding-store.ts`
- Pattern: Create store with actions and selectors
- Responsibility: User session, form state, temporary UI state
## Entry Points
- Location: `apps/api/main.go`
- Triggers: Application startup
- Responsibilities:
- Location: `apps/{app}/src/main.tsx`
- Triggers: Browser load
- Responsibilities:
- Location: `apps/{app}/src/app/routes/index.tsx`
- Pattern: Lazy-loaded route components with retry on chunk failures
- Responsibilities:
## Error Handling
- Database connection: Retry loop at startup, fatals if still failing (see `main.go:24`)
- Optional services: Warn but continue if Redis/NATS unavailable (non-critical)
- Handler errors: Return HTTP error response with JSON envelope (see middleware)
- Middleware: Return 401/403 on auth failures, 400 on validation errors
- No panics in request handlers — only in `main.go` startup failures
- Auth errors: Redirect to login page with error query param
- Network errors: Toast notification via Sonner (bottom-right)
- Chunk load failures: Automatic page reload once per session (see `lazyWithRetry()`)
- Error boundary: Handled implicitly via Suspense fallback (LoadingScreen)
## Cross-Cutting Concerns
- Backend: Standard Go `log` package to stdout (JSON-formatted via cloud logging)
- Frontend: `console.log` avoided in production (linting rule: warn if present)
- Input: Gin binding tags (`binding:"required,email"`) + custom validators in handlers
- Tenant: Middleware validates tenant presence in JWT claims
- API contracts: TypeScript types in frontend enforce request/response shapes
- Provider: Custom JWT-based (no external provider like Keycloak in this version)
- Token format: JWT with claims: `sub` (user ID), `tenant_id`, `email`, `roles`
- Verification: JWT signature validation in middleware (see `middleware/auth.go`)
- Sessions: Encrypted cookie store (browser-level, no server-side session DB)
- MFA/2FA: Not detected in codebase
- Pattern: Role-based (extracted from JWT claims)
- RBAC locations: Not detected; middleware extracts roles but no explicit permission checks
- Vendor/Admin separation: Implicitly via different frontends and database rows
- Multi-tenancy: Application-level via `tenant_id` column + header propagation
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, or `.github/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->

## Design Context

> Full source of truth: `.impeccable.md` at repo root. This summary is loaded automatically; read the full file before any design work.

### Users
Three audiences on their own apps, one brand: **customers** (hungry, distracted, want appetite + speed), **home chefs / vendors** (kitchen-hands, time-pressured, want efficiency), **drivers** (in motion, glanceable only). Plus internal **admins** (desktop ops). Job: connect hungry person → home cook → driver, with all three trusting each other through the app.

### Brand Personality
**Confident · Appetizing · Quietly modern.** Confident not loud (no urgency tricks). Appetizing not artisanal (photography and space do the work, not terracotta-and-cream kitsch). Quietly modern not trendy (restrained type, one accent, ages well). Emotional goals: hunger, trust, calm. Explicitly NOT chasing FOMO, gamification, or aggressive consumer hype.

### Aesthetic Direction — Refined consumer (Stripe Atlas / Cash App / Apple Food)
- **Light-first, dark supported.** Both modes must work; driver/vendor often dark in low light.
- **Palette: Paper · Ink · Herb.**
  - **Paper** `oklch(0.985 0.003 80)` — warm off-white page background (never `#fff`)
  - **Ink** `oklch(0.18 0.01 80)` — near-black warm-tinted (never `#000`)
  - **Herb** `oklch(0.48 0.13 145)` — deep green, **the single brand accent.** Primary CTA, focus, success, selected. Reads "fresh food" without the red-saturated competitor look.
  - **Bone / Mist** — elevated surfaces and hairlines.
  - **Functional only:** `Paprika` (destructive), `Amber` (warning). Never decorative.
- **Anti-references — explicitly avoid:** the legacy terracotta/cream/amber/Playfair artisanal system, red-saturated delivery-app look (Swiggy/Zomato/DoorDash), AI slop palette (cyan-on-dark, purple-blue gradients), glassmorphism, 2018 indie ecommerce, hero metric layouts, generic SaaS dashboards.

### Typography
- **Display: Geist Sans** (geometric, variable, 600/700 weight only) — headlines and marketing moments.
- **Body / UI: Inter Variable** — body 400, emphasis 500, headings 600.
- **Numerals:** Tabular figures for every price, ID, ETA, or aligned stat.
- **Scale:** Fluid `clamp()`, 1.2 ratio mobile / 1.25 desktop. Largest type ~2.5rem mobile / ~3rem desktop. **No display-2xl 4.5rem.**
- **Forbidden:** Playfair Display, Caveat, gradient text, ALL CAPS body.

### Spatial & Motion
- **Radius:** 8px default, 16px sheets/modals. No 24px+ super-rounded pills.
- **Hairlines, not bordered cards.** Three-step shadow scale, elevation only.
- **Spacing:** 4 / 8 / 12 / 16 / 24 / 32 / 48 / 64 / 96. Fluid `clamp()` between viewports.
- **Touch targets:** 44px customer/vendor, 48px driver.
- **Motion easing:** `cubic-bezier(0.22, 1, 0.36, 1)` for entrances. **No bounce, no elastic, no overshoot** — the existing `bounce-in` keyframe must go.
- **Animate:** `opacity` and `transform` only. Never `width/height/top/left/padding`.
- **Durations:** 150ms / 250ms / 400ms; driver app halves all.
- `prefers-reduced-motion` honored everywhere.

### Accessibility
WCAG 2.1 AA baseline. AAA for body text where practical. Skip link first on web. Semantic landmarks always. Visible 2px herb focus ring with 2px offset (never `outline: none`). Driver app: 7:1 contrast floor, all type ≥16px. Form errors via `aria-describedby` / `aria-invalid`.

### Design Principles
1. **Photo-forward, chrome-light.** Food and faces carry the brand. UI chrome shrinks.
2. **One accent, used sparingly.** Herb is for primary action, focus, success — nothing decorative. More than one herb element per screen competing for the eye = one is wrong.
3. **Confidence through restraint.** No bounce, no glass, no gradient text, no decorative shadows, no hero metrics. If it feels loud, it's wrong.
4. **Per-role density, shared language.** Customer breathes, vendor packs, driver strips down. Tokens shared, layouts adapt.
5. **Trust the system.** Use the tokens, `@tesserix/web` primitives, Geist + Inter, the easing curve. Don't invent per-feature colors/radii/motion.

The legacy terracotta-cream-amber system in `apps/web/src/styles/globals.css` is being migrated. New screens follow this file; old screens are work-in-progress.
