# HomeChef Platform - Development Progress Guide

**Project Status:** IN PROGRESS
**Target Launch Date:** February 14th, 2026
**Last Updated:** December 2024

---

## IMPORTANT: Read This First

This document serves as the **single source of truth** for the HomeChef platform development. Claude (or any AI assistant) should reference this document when continuing development to understand:
- Current progress and completed features
- Pending tasks and priorities
- Technical decisions made
- Code patterns and conventions to follow
- Security requirements

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Tech Stack & Versions](#2-tech-stack--versions)
3. [Project Structure](#3-project-structure)
4. [Development Progress](#4-development-progress)
5. [Feature Checklist](#5-feature-checklist)
6. [Code Conventions](#6-code-conventions)
7. [Security Requirements](#7-security-requirements)
8. [Mock Data System](#8-mock-data-system)
9. [Environment Setup](#9-environment-setup)
10. [Current Sprint](#10-current-sprint)
11. [Known Issues](#11-known-issues)
12. [Decision Log](#12-decision-log)

---

## 1. Project Overview

**HomeChef** is a production-ready food delivery platform connecting home chefs with customers seeking authentic homemade food. The platform includes:

### Core Interfaces
1. **Customer App** - Browse, order, track, review
2. **Chef App** - Manage menu, orders, earnings, social posts
3. **Admin Panel** - Platform management, verification, analytics
4. **Delivery Partner App** - Delivery management, navigation
5. **Fleet Manager Portal** - Fleet operations, compliance

### Key Features
- **Multi-role Authentication** with Social Login (Google, Facebook, Apple)
- **Hardened RBAC** - Strict role-based access control
- **Catering System** - Quote requests, chef bidding, booking
- **Chef Social Feed** - Post culinary content (no direct contact info)
- **Real-time Tracking** - WebSocket-based order and delivery tracking
- **Payment Processing** - Stripe integration with escrow
- **Review & Rating System** - With moderation

### Business Rules (CRITICAL)
- Customers CANNOT see chef's direct contact information
- All communication goes through platform messaging
- Chefs cannot share contact details in social posts (auto-moderation)
- Commission: 15-20% on orders, 10% on catering

---

## 2. Tech Stack & Versions

### Frontend
```
Node.js:        v25.2.1
React:          19.x (latest)
TypeScript:     5.7.x
Vite:           6.x
Tailwind CSS:   4.x (latest)
React Router:   7.x
TanStack Query: 5.x
Zustand:        5.x
Radix UI:       Latest
Lucide Icons:   Latest
```

### Backend
```
Go:             1.25.5
Gin:            1.10.x
GORM:           1.25.x
JWT-Go:         5.x
Validator:      10.x
Swagger:        Latest
```

### Database & Infrastructure
```
PostgreSQL:     16.x
Redis:          7.x
RabbitMQ:       3.13.x
Elasticsearch:  8.x
Docker:         Latest
```

### Package Manager
```
pnpm:           9.x (for frontend monorepo)
```

---

## 3. Project Structure

```
Home-Chef-App/
├── INPROGRESS.md              # THIS FILE - Development guide
├── README.md                  # Project documentation
├── docker-compose.yml         # Local development setup
├── docker-compose.prod.yml    # Production setup
├── Makefile                   # Build commands
│
├── docs/                      # Documentation
│   ├── REQUIREMENTS.md        # Full PRD
│   ├── ARCHITECTURE.md        # System architecture
│   ├── DATABASE_SCHEMA.md     # Database design
│   ├── API_SPECIFICATION.md   # API docs
│   └── SECURITY.md            # Security guidelines
│
├── apps/                      # Applications
│   ├── web/                   # React frontend (all interfaces)
│   │   ├── src/
│   │   │   ├── app/           # App setup, providers, routes
│   │   │   ├── features/      # Feature modules
│   │   │   │   ├── auth/      # Authentication
│   │   │   │   ├── customer/  # Customer interface
│   │   │   │   ├── chef/      # Chef interface
│   │   │   │   ├── admin/     # Admin panel
│   │   │   │   ├── delivery/  # Delivery partner
│   │   │   │   ├── fleet/     # Fleet manager
│   │   │   │   ├── catering/  # Catering system
│   │   │   │   └── social/    # Chef social feed
│   │   │   ├── shared/        # Shared components
│   │   │   ├── mock/          # Mock data & handlers
│   │   │   └── styles/        # Global styles
│   │   ├── public/
│   │   ├── package.json
│   │   ├── vite.config.ts
│   │   ├── tailwind.config.ts
│   │   └── tsconfig.json
│   │
│   └── api/                   # Go backend
│       ├── cmd/
│       │   └── server/
│       │       └── main.go
│       ├── internal/
│       │   ├── config/
│       │   ├── domain/
│       │   ├── repository/
│       │   ├── service/
│       │   ├── handler/
│       │   ├── middleware/
│       │   ├── router/
│       │   └── dto/
│       ├── pkg/
│       │   ├── auth/
│       │   ├── database/
│       │   ├── messaging/
│       │   ├── storage/
│       │   └── validator/
│       ├── migrations/
│       ├── go.mod
│       ├── go.sum
│       └── Dockerfile
│
├── packages/                  # Shared packages
│   ├── ui/                    # Shared UI components
│   ├── types/                 # Shared TypeScript types
│   └── config/                # Shared configurations
│
├── scripts/                   # Build & deployment scripts
├── .github/                   # GitHub Actions CI/CD
└── infrastructure/            # Terraform/K8s configs
```

---

## 4. Development Progress

### Phase 1: Foundation (Current)
| Task | Status | Notes |
|------|--------|-------|
| Project documentation | ✅ Complete | PRD, Architecture, DB Schema, API Spec |
| Monorepo setup | 🔄 In Progress | |
| Frontend scaffold | ⬜ Pending | |
| Backend scaffold | ⬜ Pending | |
| Database migrations | ⬜ Pending | |
| Mock data system | ⬜ Pending | |
| Authentication | ⬜ Pending | |

### Phase 2: Core Features
| Task | Status | Notes |
|------|--------|-------|
| Customer browse/search | ⬜ Pending | |
| Chef menu management | ⬜ Pending | |
| Cart & checkout | ⬜ Pending | |
| Order management | ⬜ Pending | |
| Real-time tracking | ⬜ Pending | |
| Payments | ⬜ Pending | |

### Phase 3: Advanced Features
| Task | Status | Notes |
|------|--------|-------|
| Admin panel | ⬜ Pending | |
| Delivery partner app | ⬜ Pending | |
| Fleet management | ⬜ Pending | |
| Catering system | ⬜ Pending | |
| Chef social feed | ⬜ Pending | |
| Reviews & ratings | ⬜ Pending | |

### Phase 4: Production Ready
| Task | Status | Notes |
|------|--------|-------|
| Security hardening | ⬜ Pending | |
| Performance optimization | ⬜ Pending | |
| Testing (unit, e2e) | ⬜ Pending | |
| CI/CD pipeline | ⬜ Pending | |
| Documentation | ⬜ Pending | |
| Launch preparation | ⬜ Pending | |

---

## 5. Feature Checklist

### 5.1 Authentication & Authorization
- [ ] Email/password registration & login
- [ ] Phone OTP verification
- [ ] Google OAuth integration
- [ ] Facebook OAuth integration
- [ ] Apple Sign-In integration
- [ ] JWT token management with refresh
- [ ] Role-based access control (RBAC)
- [ ] Permission-based route guards
- [ ] Session management
- [ ] Password reset flow
- [ ] Email verification flow
- [ ] Account deactivation

### 5.2 Customer Interface
- [ ] Location detection & address management
- [ ] Chef discovery with filters
- [ ] Search (chefs, dishes, cuisines)
- [ ] Chef profile view
- [ ] Menu browsing
- [ ] Cart management (add, update, remove)
- [ ] Checkout flow
- [ ] Payment processing
- [ ] Order tracking (real-time)
- [ ] Order history
- [ ] Review submission
- [ ] Favorites/saved chefs
- [ ] Notifications
- [ ] Profile management
- [ ] Catering request form

### 5.3 Chef Interface
- [ ] Onboarding wizard
- [ ] Profile management
- [ ] Kitchen photo upload
- [ ] Document upload (verification)
- [ ] Menu item CRUD
- [ ] Category management
- [ ] Availability toggle
- [ ] Operating hours setup
- [ ] New order alerts
- [ ] Order accept/reject
- [ ] Order status updates
- [ ] Earnings dashboard
- [ ] Payout history
- [ ] Analytics
- [ ] Social feed posting
- [ ] Catering quote responses

### 5.4 Delivery Partner Interface
- [ ] Registration & onboarding
- [ ] Document verification
- [ ] Online/offline toggle
- [ ] Order assignment notifications
- [ ] Accept/decline deliveries
- [ ] Navigation integration
- [ ] Status updates
- [ ] Proof of delivery
- [ ] Earnings tracking
- [ ] Performance metrics

### 5.5 Admin Panel
- [ ] Dashboard with KPIs
- [ ] User management
- [ ] Chef verification queue
- [ ] Chef management
- [ ] Delivery partner management
- [ ] Order management
- [ ] Dispute resolution
- [ ] Refund processing
- [ ] Promo code management
- [ ] Content moderation
- [ ] Analytics & reports
- [ ] Platform settings
- [ ] Notification broadcasting

### 5.6 Fleet Management
- [ ] Live map view
- [ ] Partner tracking
- [ ] Manual order assignment
- [ ] Performance monitoring
- [ ] Compliance tracking
- [ ] Communication tools

### 5.7 Catering System
- [ ] Catering request form
- [ ] Request matching to chefs
- [ ] Quote submission by chefs
- [ ] Quote comparison view
- [ ] Quote acceptance
- [ ] Booking confirmation
- [ ] Catering order management
- [ ] Catering payments
- [ ] Catering reviews

### 5.8 Chef Social Feed
- [ ] Post creation (text + images)
- [ ] Feed display
- [ ] Like/react to posts
- [ ] Save posts
- [ ] Content moderation
- [ ] Auto-filter contact info
- [ ] Hashtag support
- [ ] Chef discovery via posts

---

## 6. Code Conventions

### 6.1 TypeScript/React

```typescript
// File naming: kebab-case
// Component files: PascalCase.tsx
// Hook files: use-hook-name.ts
// Utility files: utility-name.ts

// Component structure
export function ComponentName({ prop1, prop2 }: ComponentProps) {
  // Hooks first
  const [state, setState] = useState<Type>(initial);
  const { data } = useQuery({ ... });

  // Handlers
  const handleClick = () => { ... };

  // Effects
  useEffect(() => { ... }, [deps]);

  // Render
  return (
    <div className="...">
      ...
    </div>
  );
}

// Always export named, not default
export { ComponentName };

// Types at top of file or in separate types.ts
interface ComponentProps {
  prop1: string;
  prop2: number;
}
```

### 6.2 API Service Pattern

```typescript
// src/features/[feature]/services/[feature]-service.ts
import { apiClient } from '@/shared/services/api-client';
import type { Chef, ChefFilters } from '../types';

export const chefService = {
  async getChefs(filters: ChefFilters) {
    return apiClient.get<PaginatedResponse<Chef>>('/chefs', filters);
  },

  async getChefById(id: string) {
    return apiClient.get<Chef>(`/chefs/${id}`);
  },

  // ... more methods
};
```

### 6.3 Go Backend

```go
// Package structure: internal/[layer]/[entity].go
// Naming: PascalCase for exports, camelCase for internal

// Handler pattern
func (h *ChefHandler) GetChefs(c *gin.Context) {
    var filters dto.ChefFilters
    if err := c.ShouldBindQuery(&filters); err != nil {
        c.JSON(http.StatusBadRequest, response.Error(err.Error()))
        return
    }

    chefs, total, err := h.service.GetChefs(c.Request.Context(), filters)
    if err != nil {
        c.JSON(http.StatusInternalServerError, response.Error(err.Error()))
        return
    }

    c.JSON(http.StatusOK, response.SuccessList(chefs, total, filters.Page, filters.Limit))
}

// Service pattern
func (s *ChefService) GetChefs(ctx context.Context, filters dto.ChefFilters) ([]domain.Chef, int64, error) {
    return s.repo.FindAll(ctx, filters)
}
```

### 6.4 Tailwind CSS Classes

```tsx
// Use consistent class ordering:
// 1. Layout (flex, grid, position)
// 2. Sizing (w, h, p, m)
// 3. Typography (font, text)
// 4. Colors (bg, text, border)
// 5. Effects (shadow, opacity)
// 6. States (hover, focus, active)
// 7. Responsive (sm:, md:, lg:)

<div className="flex items-center justify-between p-4 text-sm text-gray-600 bg-white rounded-lg shadow-sm hover:shadow-md transition-shadow md:p-6">
```

---

## 7. Security Requirements

### 7.1 RBAC Implementation

```typescript
// Roles hierarchy
enum Role {
  SUPER_ADMIN = 'super_admin',  // Full access
  ADMIN = 'admin',              // Platform management
  FLEET_MANAGER = 'fleet_manager', // Delivery operations
  CHEF = 'chef',                // Chef features only
  DELIVERY = 'delivery',        // Delivery features only
  CUSTOMER = 'customer',        // Customer features only
}

// Permissions
enum Permission {
  // User management
  USERS_READ = 'users:read',
  USERS_WRITE = 'users:write',
  USERS_DELETE = 'users:delete',

  // Chef management
  CHEFS_READ = 'chefs:read',
  CHEFS_WRITE = 'chefs:write',
  CHEFS_VERIFY = 'chefs:verify',

  // Order management
  ORDERS_READ_OWN = 'orders:read:own',
  ORDERS_READ_ALL = 'orders:read:all',
  ORDERS_WRITE = 'orders:write',
  ORDERS_CANCEL = 'orders:cancel',
  ORDERS_REFUND = 'orders:refund',

  // ... more permissions
}

// Role -> Permissions mapping
const ROLE_PERMISSIONS: Record<Role, Permission[]> = {
  [Role.CUSTOMER]: [
    Permission.ORDERS_READ_OWN,
    Permission.ORDERS_WRITE,
    // Customer can NOT see: chef phone, chef address details, chef email
  ],
  [Role.CHEF]: [
    Permission.CHEFS_WRITE, // own profile only
    Permission.ORDERS_READ_OWN,
    // Chef can NOT see: customer phone (masked), customer full address (only area)
  ],
  // ... more mappings
};
```

### 7.2 Data Protection Rules

| Data | Customer View | Chef View | Admin View |
|------|---------------|-----------|------------|
| Chef Phone | ❌ Never | N/A | ✅ Full |
| Chef Email | ❌ Never | N/A | ✅ Full |
| Chef Address | ❌ Area only | N/A | ✅ Full |
| Customer Phone | N/A | ❌ Masked | ✅ Full |
| Customer Email | N/A | ❌ Never | ✅ Full |
| Customer Address | N/A | ✅ For delivery | ✅ Full |

### 7.3 Content Moderation

```typescript
// Auto-filter patterns for chef social posts
const BLOCKED_PATTERNS = [
  /\b\d{10}\b/,                    // Phone numbers
  /\b\d{3}[-.]?\d{3}[-.]?\d{4}\b/, // Phone formats
  /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/, // Emails
  /whatsapp|telegram|signal/i,     // Messaging apps
  /call me|contact me|dm me/i,     // Contact requests
  /\b(instagram|facebook|twitter|tiktok)\.com\/\w+/i, // Social links
];

// Server-side validation before post save
function moderateContent(content: string): ModerationResult {
  const flagged = BLOCKED_PATTERNS.some(pattern => pattern.test(content));
  return {
    approved: !flagged,
    reason: flagged ? 'Contains contact information' : null
  };
}
```

### 7.4 API Security

- All endpoints require authentication (except public browse)
- Rate limiting: 100 req/min for users, 1000 req/min for service accounts
- Request signing for sensitive operations
- CORS restricted to allowed origins
- SQL injection prevention via parameterized queries
- XSS prevention via input sanitization
- CSRF tokens for state-changing operations

---

## 8. Mock Data System

### 8.1 Mock Mode Toggle

```typescript
// Environment variable controls mock mode
// .env.development
VITE_MOCK_MODE=true
VITE_API_URL=http://localhost:8080/api/v1

// .env.production
VITE_MOCK_MODE=false
VITE_API_URL=https://api.homechef.com/v1
```

### 8.2 Mock Service Implementation

```typescript
// src/mock/mock-service.ts
class MockService {
  private delay = 300; // Simulate network delay

  async get<T>(endpoint: string, params?: object): Promise<T> {
    await this.simulateDelay();
    return this.handleEndpoint('GET', endpoint, params);
  }

  // Handler routing
  private handleEndpoint(method: string, endpoint: string, data?: any) {
    const handlers: Record<string, () => any> = {
      'GET /chefs': () => this.getChefs(data),
      'GET /chefs/:id': () => this.getChefById(data),
      'POST /orders': () => this.createOrder(data),
      // ... all endpoints
    };
    // Match and execute
  }
}
```

### 8.3 Mock Data Files

```
src/mock/data/
├── users.ts          # Sample users (all roles)
├── chefs.ts          # Chef profiles
├── menu-items.ts     # Menu items
├── orders.ts         # Sample orders
├── reviews.ts        # Reviews
├── deliveries.ts     # Delivery data
├── catering.ts       # Catering requests
└── social-posts.ts   # Chef social posts
```

---

## 9. Environment Setup

### 9.1 Prerequisites

```bash
# Required versions
node --version    # v25.2.1
go version        # go1.25.5
pnpm --version    # 9.x

# Install pnpm if needed
npm install -g pnpm@latest
```

### 9.2 Development Setup

```bash
# Clone and install
git clone <repo>
cd Home-Chef-App

# Install frontend dependencies
pnpm install

# Start frontend (with mock data)
pnpm --filter web dev

# Start backend (separate terminal)
cd apps/api
go mod download
go run cmd/server/main.go

# Start with Docker (full stack)
docker-compose up -d
```

### 9.3 Environment Variables

```bash
# apps/web/.env.development
VITE_MOCK_MODE=true
VITE_API_URL=http://localhost:8080/api/v1
VITE_WS_URL=ws://localhost:8080/ws
VITE_GOOGLE_CLIENT_ID=xxx
VITE_FACEBOOK_APP_ID=xxx
VITE_STRIPE_PUBLIC_KEY=pk_test_xxx
VITE_GOOGLE_MAPS_KEY=xxx

# apps/api/.env
PORT=8080
DATABASE_URL=postgres://user:pass@localhost:5432/homechef
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key
JWT_EXPIRY=1h
REFRESH_TOKEN_EXPIRY=7d
```

---

## 10. Current Sprint

### Sprint Goal
Set up complete project foundation with authentication and mock data system.

### Tasks
1. [x] Create INPROGRESS.md
2. [ ] Initialize monorepo with pnpm
3. [ ] Set up React app with Vite + TypeScript
4. [ ] Configure Tailwind CSS 4.x
5. [ ] Set up routing structure
6. [ ] Implement mock service
7. [ ] Create seed mock data
8. [ ] Build authentication UI
9. [ ] Implement auth store (Zustand)
10. [ ] Set up Go backend scaffold
11. [ ] Create basic API routes

### Blockers
- None currently

### Notes
- Focus on getting a working demo with mock data first
- Authentication should support all social providers but can use mock initially
- Ensure RBAC is implemented from the start

---

## 11. Known Issues

| ID | Issue | Severity | Status | Notes |
|----|-------|----------|--------|-------|
| - | None yet | - | - | - |

---

## 12. Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| Dec 2024 | Use Vite over Next.js | Simpler, faster builds, no SSR needed initially | Team |
| Dec 2024 | Use Zustand over Redux | Lighter, simpler API, sufficient for our needs | Team |
| Dec 2024 | Use pnpm monorepo | Better disk usage, faster installs, native workspace support | Team |
| Dec 2024 | Tailwind CSS 4.x | Latest features, better performance | Team |
| Dec 2024 | Go over Node.js backend | Better performance, type safety, team familiarity | Team |
| Dec 2024 | PostgreSQL + PostGIS | Spatial queries for location-based features | Team |
| Dec 2024 | RBAC from day 1 | Security-first approach for production readiness | Team |

---

## Quick Commands

```bash
# Frontend
pnpm dev              # Start dev server
pnpm build            # Production build
pnpm test             # Run tests
pnpm lint             # Lint code

# Backend
make run              # Start server
make test             # Run tests
make migrate          # Run migrations
make swagger          # Generate API docs

# Docker
docker-compose up -d  # Start all services
docker-compose down   # Stop all services
docker-compose logs   # View logs
```

---

## Contact & Resources

- **Documentation:** `/docs/` folder
- **API Docs:** http://localhost:8080/swagger
- **Design System:** http://localhost:3000/storybook

---

**Remember:** Always update this file when making significant changes to keep the development guide accurate!
