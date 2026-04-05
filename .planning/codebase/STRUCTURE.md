# Codebase Structure

**Analysis Date:** 2026-04-05

## Directory Layout

```
Home-Chef-App/
├── .github/                    # CI/CD workflows
├── .planning/                  # Planning artifacts and analysis docs
├── docs/                       # Documentation
├── apps/                       # Monorepo applications
│   ├── api/                    # Go HTTP backend (Gin)
│   ├── web/                    # Customer storefront (React SPA)
│   ├── vendor-portal/          # Vendor dashboard (React SPA)
│   ├── admin-portal/           # Admin dashboard (React SPA)
│   └── delivery-portal/        # Delivery driver app (React SPA)
├── package.json                # Root workspace config
├── pnpm-workspace.yaml         # pnpm monorepo definition
├── docker-compose.yml          # Local dev services (DB, Redis, NATS)
└── pnpm-lock.yaml              # Dependency lockfile
```

## Directory Purposes

**`apps/api/`:**
- Purpose: Go HTTP API backend, business logic, database operations
- Contains: Handlers, services, models, middleware, database, routes, config
- Key files: `main.go` (entry), `routes/routes.go` (endpoint definitions)
- Language: Go 1.26

**`apps/web/`:**
- Purpose: Customer-facing storefront (browse menu, place orders, track delivery)
- Contains: React components, pages, features, shared UI library
- Key structure: `src/` with app/, features/, shared/, styles/
- Language: TypeScript + React 19

**`apps/vendor-portal/`:**
- Purpose: Home chef management portal (menu, orders, earnings, kitchen setup)
- Contains: React components, pages, features, shared UI library
- Key structure: `src/` with app/, features/, shared/, styles/
- Language: TypeScript + React 19

**`apps/admin-portal/`:**
- Purpose: Platform admin dashboard (user management, approvals, analytics)
- Contains: React components, pages, features, shared UI library
- Key structure: `src/` with app/, features/, shared/, styles/
- Language: TypeScript + React 19

**`apps/delivery-portal/`:**
- Purpose: Delivery driver mobile app (live orders, navigation, earnings)
- Contains: React components, pages, features, shared UI library
- Key structure: `src/` with app/, features/, shared/, styles/
- Language: TypeScript + React 19

## Key File Locations

**Backend Entry Points:**
- `apps/api/main.go` - Application startup, service initialization, graceful shutdown
- `apps/api/routes/routes.go` - Gin router setup, endpoint definitions, middleware chain

**Frontend Entry Points:**
- `apps/web/src/main.tsx` - React root mount, StrictMode, service worker registration
- `apps/vendor-portal/src/main.tsx` - React root mount
- `apps/admin-portal/src/main.tsx` - React root mount
- `apps/delivery-portal/src/main.tsx` - React root mount

**Backend Configuration:**
- `apps/api/config/config.go` - Load environment variables, database URL construction
- `apps/api/.env.example` - Template for required environment variables

**Backend Middleware:**
- `apps/api/middleware/auth.go` - JWT token extraction, role decoding
- `apps/api/middleware/rbac.go` - Role-based access control
- `apps/api/middleware/metrics.go` - Prometheus metrics collection

**Backend Handlers (API Endpoints):**
- `apps/api/handlers/health.go` - /health, /metrics endpoints
- `apps/api/handlers/auth.go` - Login, register, session management
- `apps/api/handlers/chefs.go` - Chef profile, menu management
- `apps/api/handlers/orders.go` - Order CRUD, order tracking
- `apps/api/handlers/payment.go` - Payment processing, Razorpay webhooks
- `apps/api/handlers/menu.go` - Menu items, categories, preferences
- `apps/api/handlers/customer.go` - Customer profiles, addresses, favorites
- Plus 22 more handler files for notifications, reviews, delivery, admin features, etc.

**Backend Services (Business Logic):**
- `apps/api/services/storage.go` - GCS file upload/download
- `apps/api/services/razorpay.go` - Razorpay payment integration
- `apps/api/services/notifications.go` - Email/push notification service
- `apps/api/services/redis.go` - Redis client initialization
- `apps/api/services/nats.go` - NATS messaging client
- `apps/api/services/email.go` - SendGrid email sending
- `apps/api/services/push.go` - Firebase push notifications
- `apps/api/services/billing.go` - Billing logic
- `apps/api/services/forex.go` - Currency conversion

**Backend Models (Domain Types):**
- `apps/api/models/user.go` - User, Auth structs
- `apps/api/models/chef.go` - Chef profile, kitchen details
- `apps/api/models/order.go` - Order, OrderItem structs
- `apps/api/models/menu.go` - Menu, MenuItem, Category structs
- `apps/api/models/payment.go` - Payment, Invoice structs
- `apps/api/models/customer.go` - Customer profile struct
- Plus 20+ more model files for all domain entities

**Frontend App Structure (all React apps follow this pattern):**
- `src/main.tsx` - Entry point, React mounting
- `src/app/App.tsx` - Root component, provider setup
- `src/app/routes/index.tsx` - React Router routes, lazy-loaded pages
- `src/app/providers/AuthProvider.tsx` - Authentication context/hook
- `src/app/store/auth-store.ts` - Zustand auth state
- `src/app/store/onboarding-store.ts` - Zustand onboarding form state

**Frontend Features (example vendor-portal, others similar):**
- `src/features/auth/` - Login, register pages
- `src/features/dashboard/` - Dashboard page
- `src/features/menu/` - Menu management, item form/view pages
- `src/features/orders/` - Live orders, order history pages
- `src/features/earnings/` - Earnings, payouts pages
- `src/features/profile/` - Profile, kitchen setup pages
- `src/features/onboarding/` - Vendor onboarding steps (multi-step form)
- `src/features/reviews/` - Customer reviews page
- `src/features/analytics/` - Analytics dashboard
- `src/features/settings/` - Settings page

**Frontend Shared:**
- `src/shared/components/ui/` - Button, Input, Dialog, Card, Badge, Rating, etc.
- `src/shared/components/layout/` - VendorLayout, CustomerLayout components
- `src/shared/types/` - TypeScript types (User, Chef, Order, etc.)
- `src/shared/constants/` - Brand, image constants
- `src/shared/utils/` - Utility functions (cn for classNames, animations)
- `src/shared/hooks/` - Custom React hooks (if any)

**Frontend Styles:**
- `src/styles/globals.css` - Tailwind CSS + global styles
- Tailwind configured in `tailwind.config.ts` (if present)

## Naming Conventions

**Backend Files:**
- Handlers: `{feature}.go` (e.g., `auth.go`, `orders.go`)
- Services: `{feature}.go` (e.g., `storage.go`, `notifications.go`)
- Models: `{entity}.go` (e.g., `user.go`, `order.go`)
- Package names: lowercase, single word (e.g., `handlers`, `services`, `models`)
- Function names: PascalCase for exported, camelCase for private

**Backend Structs:**
- Handlers: `{Feature}Handler` (e.g., `AuthHandler`, `OrderHandler`)
- Models (ORM): `{Entity}` (e.g., `Order`, `Chef`, `User`)
- Request/Response: `{Entity}{Action}Request/Response` (e.g., `CreateOrderRequest`)
- Database models: GORM-tagged with `gorm:` tags

**Frontend Files:**
- Components: PascalCase (e.g., `LoginPage.tsx`, `MenuItemForm.tsx`)
- Pages: PascalCase with "Page" suffix (e.g., `DashboardPage.tsx`)
- Services: camelCase with "service" suffix (e.g., `auth-service.ts`)
- Stores: camelCase with "store" suffix (e.g., `auth-store.ts`)
- Types: PascalCase (e.g., `User.ts`, `Order.ts`)
- Utilities: camelCase with descriptive name (e.g., `cn.ts`, `animations.ts`)

**Frontend Directories:**
- Features: kebab-case (e.g., `auth/`, `menu/`, `live-orders/`)
- Shared components: `components/` (nested by category: `ui/`, `layout/`)
- Shared utilities: `utils/`, `hooks/`, `types/`, `constants/`

**API Endpoints:**
- Pattern: RESTful `/api/v1/{resource}/{id}/{action}`
- Examples:
  - `GET /api/v1/chefs/{id}` - Get chef profile
  - `POST /api/v1/orders` - Create order
  - `PUT /api/v1/orders/{id}` - Update order
  - `GET /api/v1/menu/items` - List menu items
  - `POST /api/v1/auth/login` - Login

## Where to Add New Code

**New Backend Endpoint:**
1. Add handler method in `apps/api/handlers/{feature}.go`
   - If file doesn't exist, create it
   - Follow pattern: method receiver on `*{Feature}Handler`
   - Return JSON via `c.JSON(http.StatusOK, response)`
2. Add route in `apps/api/routes/routes.go`
   - Register handler via `r.POST("/api/v1/path", handler.Method)`
3. Add model struct in `apps/api/models/{entity}.go` if needed
4. Add service logic in `apps/api/services/{feature}.go` if needed

**New Frontend Page:**
1. Create page component in `apps/{app}/src/features/{feature}/pages/{Name}Page.tsx`
2. Add route in `apps/{app}/src/app/routes/index.tsx`
3. Add lazy-load wrapper: `lazyWithRetry(() => import(...))`
4. Define route: `<Route path="path" element={<ProtectedRoute><Page /></ProtectedRoute>} />`

**New React Component:**
1. UI component: `apps/{app}/src/shared/components/ui/{Name}.tsx`
2. Feature component: `apps/{app}/src/features/{feature}/components/{Name}.tsx`
3. Layout component: `apps/{app}/src/shared/components/layout/{Name}.tsx`

**New Shared Service (API calls):**
1. Create file: `apps/{app}/src/features/{feature}/services/{service-name}.ts`
2. Export functions that call `fetch()` or axios
3. Use `useQuery()` from React Query in components to call service

**New Store (State Management):**
1. Create file: `apps/{app}/src/app/store/{feature}-store.ts`
2. Define interface with state and actions
3. Use `zustand.create()` to initialize
4. Hook in components: `const { state, action } = use{Feature}Store()`

**New Model (Type Definition):**
1. Backend: Add struct in `apps/api/models/{entity}.go` with GORM tags
2. Frontend: Add type in `apps/{app}/src/shared/types/{entity}.ts`

## Special Directories

**`apps/api/database/`:**
- Purpose: Database connection, migration logic
- Generated: No
- Committed: Yes
- Contains: Connection initialization, migration files (if using migrations)

**`apps/api/config/`:**
- Purpose: Configuration loading from environment variables
- Generated: No
- Committed: Yes
- Contains: Config struct definition, Load() function

**`apps/{app}/src/mock/`:**
- Purpose: Mock data for development/testing (no backend API needed)
- Generated: No
- Committed: Yes
- Contains: Fake API responses, mock services (vendor-portal, web have this)

**`apps/{app}/dist/`:**
- Purpose: Built frontend output (minified JS, CSS, HTML)
- Generated: Yes (via `vite build`)
- Committed: No (in .gitignore)

**`apps/api/go.mod` and `go.sum`:**
- Purpose: Go dependency management
- Generated: No (maintained manually via `go get`)
- Committed: Yes

**Root `pnpm-lock.yaml`:**
- Purpose: Lock file for all npm dependencies across monorepo
- Generated: Automatically by pnpm (commit this)
- Committed: Yes

---

*Structure analysis: 2026-04-05*
