# Technology Stack

**Analysis Date:** 2026-04-05

## Languages

**Primary:**
- Go 1.26.1 - Backend API (`apps/api/go.mod`), all service logic
- TypeScript 5.7.2 - All frontend applications and React components
- YAML - Docker Compose, Kubernetes manifests

**Secondary:**
- JavaScript (Node.js 22.x) - Runtime for all Next.js/Vite applications
- Bash - Development scripts, Docker setup

## Runtime

**Environment:**
- Go 1.26.1 (backend services)
- Node.js 22.x (all frontend applications - Vite, React)
- Docker (multi-stage containerization for all services)
- Alpine Linux 3.19 (production container images)

**Package Manager:**
- npm (frontend apps: web, admin-portal, vendor-portal, delivery-portal)
  - `package-lock.json` present in all Node projects
  - Version: pnpm 9.15.1 (monorepo manager at workspace root)
- Go modules (backend)
  - `go.mod` / `go.sum` in `apps/api/`

## Frameworks

**Core Backend:**
- Gin v1.10.0 (`apps/api/go.mod`) - HTTP framework for all Go services
- GORM v1.25.12 + PostgreSQL driver - ORM for database operations
- PostgreSQL 16 - Primary data store (via docker-compose)

**Frontend:**
- Vite 6.0.3 - Build tool and dev server (web, admin-portal, vendor-portal, delivery-portal)
- React 19.0.0 - UI rendering for all frontend applications
- React Router v7.1.0 - Client-side routing

**UI Components & Styling:**
- Radix UI (comprehensive primitives: avatar, checkbox, dialog, dropdown, label, select, separator, slot, switch, tabs, toast, tooltip)
  - All via `@radix-ui/*` packages
- Tailwind CSS v4.x - Utility-first CSS framework
- shadcn/ui (via `@tesserix/web` v1.2.0) - Pre-built Radix UI + Tailwind components
- Framer Motion v11.15.0 - Animations
- Lucide React v0.468.0 - Icon library
- `class-variance-authority` v0.7.1 - Component variant management
- `clsx` v2.1.1 - Conditional class names

**State Management:**
- Zustand v5.0.2 - Client state (web, admin-portal, vendor-portal, delivery-portal)
- TanStack React Query v5.62.8 - Server state and caching (web, admin-portal, vendor-portal, delivery-portal)
- React Context - Built-in context API usage

**Forms & Validation:**
- React Hook Form v7.54.1 - Form handling across all frontends
- Zod v3.24.1 - TypeScript-first schema validation
- `@hookform/resolvers` v3.9.1 - Integration with Zod

**Data & Utilities:**
- date-fns v4.1.0 - Date manipulation
- axios v1.13.0 - HTTP client for API requests
- qrcode.react v4.2.0 - QR code generation
- sonner v1.7.1 - Toast notifications
- tailwind-merge v2.6.0 - Tailwind class merging

## Key Dependencies

**Backend (Go):**
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

**Frontend (npm):**
- `@tanstack/react-query-devtools` v5.62.8 - React Query debugging tools
- `@tesserix/web` v1.2.0 - Shared design system component library
- ESLint v9.17.0 - Linting
- Vitest v2.1.8 - Unit testing framework
- `@vitest/coverage-v8` v2.1.8 - Code coverage reporting
- TypeScript ESLint v8.18.0 - TypeScript linting
- Prettier v3.4.2 - Code formatting

## Configuration

**Backend Configuration (`apps/api/`):**
- `.env.example` - Template for all required environment variables
- `config/config.go` - Centralized configuration loading via environment variables
- `Dockerfile` - Multi-stage build for containerization
- Key env vars required:
  - Database: `DATABASE_URL`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`
  - JWT: `JWT_SECRET`, `JWT_EXPIRATION_HOURS`, `REFRESH_TOKEN_DAYS`
  - OAuth: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `FACEBOOK_APP_ID`, `FACEBOOK_APP_SECRET`, `APPLE_CLIENT_ID`, `APPLE_TEAM_ID`, `APPLE_KEY_ID`
  - Storage: `GCS_PROJECT_ID`, `GCS_PUBLIC_BUCKET`, `GCS_PRIVATE_BUCKET`
  - Payments: `RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET`, `RAZORPAY_WEBHOOK_SECRET`
  - Email: `SENDGRID_API_KEY`, `FROM_EMAIL`, `FROM_NAME`
  - SMS: `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_PHONE_NUMBER`
  - Messaging: `REDIS_URL`, `NATS_URL`

**Frontend Configuration:**
- `vite.config.ts` - Vite build configuration (present in each app)
- `tsconfig.json` - TypeScript configuration for each app
- `docker-compose.yml` (root) - Local development orchestration
  - Defines services: postgres, redis, nats, api, web, adminer, mailhog

**Docker Services (docker-compose.yml):**
```
Services:
- postgres:16-alpine (database)
- redis:7-alpine (caching)
- nats:2.10-alpine (messaging with JetStream)
- Go API backend
- Vite dev servers for frontends
- Adminer (optional DB UI)
- Mailhog (optional email testing)
```

## Storage & Persistence

**Primary Database:**
- PostgreSQL 16 (via `postgres:16-alpine` in docker-compose)
- Connection pool: 5 max open, 2 idle
- GORM ORM handles migrations and queries

**Caching:**
- Redis 7 (via `redis:7-alpine` in docker-compose)
- Client: `github.com/redis/go-redis/v9`
- Used for: Session caching, general app caching

**File Storage:**
- Google Cloud Storage (GCS) for production
  - Public bucket: `homechef-prod-assets-in`
  - Private bucket: `homechef-prod-docs-in`
  - Client: `cloud.google.com/go/storage`
  - Local file uploads directory: `/app/uploads` (via docker volume)

**Secret Management:**
- GCP Secret Manager for production secrets
  - Used for vendor/driver payment fields
  - Client: `cloud.google.com/go/secretmanager`
  - Secret ID format: `<env>-<role>-payment-<entityId>-<field>`

## Messaging & Event Systems

**Message Broker:**
- NATS 2.10 with JetStream enabled (via `nats:2.10-alpine`)
- Client: `github.com/nats-io/nats.go`
- Usage: Event publishing/subscription for async workflows
- Subjects defined: `orders.*`, `chef.*`, `delivery.*`, `payments.*`, `users.registered`, `notifications.*`, `approvals.*`, `driver.*`, `subscription.*`, etc.

## Authentication

**OAuth Providers Supported (configured but implementation-ready):**
- Google OAuth2 - `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`
- Facebook OAuth - `FACEBOOK_APP_ID`, `FACEBOOK_APP_SECRET`
- Apple Sign-In - `APPLE_CLIENT_ID`, `APPLE_TEAM_ID`, `APPLE_KEY_ID`

**Token Management:**
- JWT (JSON Web Tokens) via `github.com/golang-jwt/jwt/v5`
- Secret: `JWT_SECRET` (configured in environment)
- Expiration: `JWT_EXPIRATION_HOURS` (default: 24h)
- Refresh tokens: `REFRESH_TOKEN_DAYS` (default: 30 days)

## Platform Requirements

**Development:**
- Go 1.26.1
- Node.js >= 22.0.0
- pnpm >= 9.0.0 (for monorepo management)
- Docker & Docker Compose (for local services)
- PostgreSQL 16 client tools (psql)
- Git

**Production:**
- Docker container runtime
- PostgreSQL 16 database
- Redis 7 server
- NATS 2.10 server with JetStream
- GCP credentials for:
  - Cloud Storage (file uploads)
  - Secret Manager (payment secrets)
- Environment variables injected at runtime

## Build & Development

**Development Server:**
```bash
pnpm dev              # Runs Vite dev server for main web app
pnpm dev:vendor       # Vendor portal
pnpm dev:admin        # Admin portal
pnpm dev:delivery     # Delivery portal
cd apps/api && go run cmd/server/main.go  # Go backend
```

**Build Process:**
```bash
pnpm build            # TypeScript + Vite build for web app
pnpm build:vendor     # Vendor portal
pnpm build:admin      # Admin portal
pnpm build:delivery   # Delivery portal
pnpm build:all        # Build all apps
```

**Linting & Formatting:**
```bash
pnpm lint             # ESLint for all apps
pnpm lint:fix         # Fix linting issues
pnpm format           # Prettier format all files
```

**Testing:**
```bash
pnpm test             # Run all tests
pnpm test:coverage    # Generate coverage reports
```

**Type Checking:**
```bash
pnpm typecheck        # Full TypeScript check across all apps
```

---

*Stack analysis: 2026-04-05*
