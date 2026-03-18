# HomeChef Platform - System Architecture Document

**Version:** 1.0
**Last Updated:** December 2024

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [High-Level System Design](#2-high-level-system-design)
3. [Frontend Architecture](#3-frontend-architecture)
4. [Backend Architecture](#4-backend-architecture)
5. [Database Design](#5-database-design)
6. [API Design](#6-api-design)
7. [Authentication & Authorization](#7-authentication--authorization)
8. [Real-time Communication](#8-real-time-communication)
9. [Infrastructure & Deployment](#9-infrastructure--deployment)
10. [Mock System Architecture](#10-mock-system-architecture)
11. [Security Architecture](#11-security-architecture)
12. [Monitoring & Observability](#12-monitoring--observability)

---

## 1. Architecture Overview

### 1.1 Design Principles

| Principle | Description |
|-----------|-------------|
| **Microservices** | Independent, deployable services for each domain |
| **API-First** | All functionality exposed via well-documented APIs |
| **Event-Driven** | Async communication for loose coupling |
| **Cloud-Native** | Containerized, orchestrated, auto-scaling |
| **Mock-Ready** | Feature flags for seamless mock/real switching |

### 1.2 Technology Decisions

```
┌─────────────────────────────────────────────────────────────────────┐
│                         TECHNOLOGY STACK                            │
├─────────────────────────────────────────────────────────────────────┤
│  FRONTEND          │  BACKEND           │  DATABASE                 │
│  ─────────         │  ───────           │  ────────                 │
│  React 18+         │  Go 1.21+          │  PostgreSQL 15            │
│  TypeScript 5+     │  Gin Framework     │  Redis 7                  │
│  Vite              │  GORM ORM          │  Elasticsearch 8          │
│  TanStack Query    │  JWT Auth          │  S3 (Files)               │
│  Zustand           │  Swagger/OpenAPI   │                           │
│  Tailwind CSS      │                    │                           │
│  shadcn/ui         │                    │                           │
├─────────────────────────────────────────────────────────────────────┤
│  MOBILE            │  MESSAGING         │  INFRASTRUCTURE           │
│  ─────────         │  ─────────         │  ──────────────           │
│  React Native      │  RabbitMQ          │  Docker                   │
│  Expo              │  WebSocket         │  Kubernetes               │
│                    │  Firebase FCM      │  AWS/GCP                  │
│                    │                    │  Terraform                │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. High-Level System Design

### 2.1 System Context Diagram

```
                                    ┌─────────────────┐
                                    │   EXTERNAL      │
                                    │   SERVICES      │
                                    ├─────────────────┤
                                    │ • Google OAuth  │
                                    │ • Facebook Auth │
                                    │ • Apple Sign In │
                                    │ • Stripe        │
                                    │ • Google Maps   │
                                    │ • Twilio        │
                                    │ • SendGrid      │
                                    │ • Firebase FCM  │
                                    └────────┬────────┘
                                             │
┌──────────────┐  ┌──────────────┐  ┌────────▼────────┐  ┌──────────────┐
│   CUSTOMER   │  │    CHEF      │  │                 │  │   DELIVERY   │
│     APP      │  │     APP      │  │    HOMECHEF     │  │   PARTNER    │
│  (Web/Mobile)│  │  (Web/Mobile)│  │    PLATFORM     │  │     APP      │
└──────┬───────┘  └──────┬───────┘  │                 │  └──────┬───────┘
       │                 │          └────────┬────────┘         │
       │                 │                   │                  │
       └─────────────────┴─────────┬─────────┴──────────────────┘
                                   │
                          ┌────────▼────────┐
                          │   API GATEWAY   │
                          │   (Kong/Nginx)  │
                          └────────┬────────┘
                                   │
       ┌───────────────────────────┼───────────────────────────┐
       │                           │                           │
┌──────▼──────┐  ┌────────────────▼────────────────┐  ┌───────▼──────┐
│    ADMIN    │  │         MICROSERVICES          │  │    FLEET     │
│    PANEL    │  │  ┌─────┐ ┌─────┐ ┌─────┐      │  │   MANAGER    │
│    (Web)    │  │  │User │ │Order│ │Chef │      │  │    (Web)     │
│             │  │  │Svc  │ │Svc  │ │Svc  │      │  │              │
└─────────────┘  │  └─────┘ └─────┘ └─────┘      │  └──────────────┘
                 │  ┌─────┐ ┌─────┐ ┌─────┐      │
                 │  │Deliv│ │Pay  │ │Notif│      │
                 │  │Svc  │ │Svc  │ │Svc  │      │
                 │  └─────┘ └─────┘ └─────┘      │
                 └─────────────────────────────────┘
```

### 2.2 Component Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT LAYER                                    │
├─────────────────┬─────────────────┬─────────────────┬─────────────────────────┤
│  Customer Web   │   Customer App  │    Chef Web     │   Chef App    │
│  (React SPA)    │ (React Native)  │   (React SPA)   │(React Native) │
├─────────────────┴─────────────────┴─────────────────┴─────────────────────────┤
│                                                                              │
│  Admin Panel (React)  │  Fleet Manager (React)  │  Delivery App (RN)        │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              API GATEWAY                                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │Rate Limiting│  │   Auth      │  │   Routing   │  │   Logging   │        │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SERVICE LAYER (Go)                                 │
├─────────────────┬─────────────────┬─────────────────┬─────────────────────────┤
│                 │                 │                 │                         │
│  ┌───────────┐  │  ┌───────────┐  │  ┌───────────┐  │  ┌───────────┐         │
│  │   USER    │  │  │   CHEF    │  │  │   ORDER   │  │  │ DELIVERY  │         │
│  │  SERVICE  │  │  │  SERVICE  │  │  │  SERVICE  │  │  │  SERVICE  │         │
│  │           │  │  │           │  │  │           │  │  │           │         │
│  │• Register │  │  │• Onboard  │  │  │• Create   │  │  │• Assign   │         │
│  │• Login    │  │  │• Menu     │  │  │• Track    │  │  │• Track    │         │
│  │• Profile  │  │  │• Schedule │  │  │• History  │  │  │• Complete │         │
│  │• Auth     │  │  │• Verify   │  │  │• Ratings  │  │  │• Route    │         │
│  └───────────┘  │  └───────────┘  │  └───────────┘  │  └───────────┘         │
│                 │                 │                 │                         │
├─────────────────┼─────────────────┼─────────────────┼─────────────────────────┤
│                 │                 │                 │                         │
│  ┌───────────┐  │  ┌───────────┐  │  ┌───────────┐  │  ┌───────────┐         │
│  │  PAYMENT  │  │  │  SEARCH   │  │  │NOTIFICATION│ │  │ ANALYTICS │         │
│  │  SERVICE  │  │  │  SERVICE  │  │  │  SERVICE  │  │  │  SERVICE  │         │
│  │           │  │  │           │  │  │           │  │  │           │         │
│  │• Process  │  │  │• Chefs    │  │  │• Push     │  │  │• Reports  │         │
│  │• Refund   │  │  │• Menu     │  │  │• SMS      │  │  │• Metrics  │         │
│  │• Payout   │  │  │• Filter   │  │  │• Email    │  │  │• Insights │         │
│  │• History  │  │  │• Geo      │  │  │• In-App   │  │  │• Export   │         │
│  └───────────┘  │  └───────────┘  │  └───────────┘  │  └───────────┘         │
│                 │                 │                 │                         │
└─────────────────┴─────────────────┴─────────────────┴─────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            MESSAGE BROKER                                    │
│                         (RabbitMQ / Kafka)                                   │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐              │
│  │  Order Events   │  │ Delivery Events │  │  Payment Events │              │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘              │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                             DATA LAYER                                       │
├─────────────────┬─────────────────┬─────────────────┬─────────────────────────┤
│   PostgreSQL    │     Redis       │  Elasticsearch  │        S3              │
│   (Primary DB)  │    (Cache)      │    (Search)     │     (Files)            │
│                 │                 │                 │                         │
│  • Users        │  • Sessions     │  • Chef Index   │  • Images              │
│  • Orders       │  • Cache        │  • Menu Index   │  • Documents           │
│  • Payments     │  • Rate Limit   │  • Geo Search   │  • Reports             │
│  • Chefs        │  • Real-time    │                 │                         │
└─────────────────┴─────────────────┴─────────────────┴─────────────────────────┘
```

---

## 3. Frontend Architecture

### 3.1 Project Structure

```
homechef-web/
├── src/
│   ├── app/                    # App configuration
│   │   ├── providers/          # Context providers
│   │   ├── routes/             # Route definitions
│   │   └── store/              # Global state (Zustand)
│   │
│   ├── features/               # Feature-based modules
│   │   ├── auth/
│   │   │   ├── components/
│   │   │   ├── hooks/
│   │   │   ├── services/
│   │   │   ├── types/
│   │   │   └── index.ts
│   │   ├── customer/
│   │   │   ├── browse/
│   │   │   ├── cart/
│   │   │   ├── orders/
│   │   │   └── profile/
│   │   ├── chef/
│   │   │   ├── dashboard/
│   │   │   ├── menu/
│   │   │   ├── orders/
│   │   │   └── earnings/
│   │   ├── delivery/
│   │   ├── admin/
│   │   └── fleet/
│   │
│   ├── shared/                 # Shared utilities
│   │   ├── components/         # Reusable UI components
│   │   │   ├── ui/            # Base components (shadcn)
│   │   │   ├── forms/         # Form components
│   │   │   ├── layout/        # Layout components
│   │   │   └── feedback/      # Alerts, toasts, modals
│   │   ├── hooks/              # Custom hooks
│   │   ├── services/           # API services
│   │   ├── utils/              # Helper functions
│   │   ├── types/              # TypeScript types
│   │   └── constants/          # App constants
│   │
│   ├── mock/                   # Mock data system
│   │   ├── data/               # Mock data files
│   │   ├── handlers/           # MSW handlers
│   │   └── server.ts           # Mock server setup
│   │
│   └── styles/                 # Global styles
│       └── globals.css
│
├── public/
├── tests/
├── .env.example
├── package.json
├── tsconfig.json
├── vite.config.ts
└── tailwind.config.js
```

### 3.2 State Management

```typescript
// Using Zustand for global state
// src/app/store/index.ts

import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';

// Auth Store
interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (credentials: LoginCredentials) => Promise<void>;
  logout: () => void;
  setUser: (user: User) => void;
}

export const useAuthStore = create<AuthState>()(
  devtools(
    persist(
      (set, get) => ({
        user: null,
        token: null,
        isAuthenticated: false,
        login: async (credentials) => {
          const response = await authService.login(credentials);
          set({
            user: response.user,
            token: response.token,
            isAuthenticated: true
          });
        },
        logout: () => {
          set({ user: null, token: null, isAuthenticated: false });
        },
        setUser: (user) => set({ user }),
      }),
      { name: 'auth-storage' }
    )
  )
);

// Cart Store
interface CartState {
  items: CartItem[];
  chefId: string | null;
  addItem: (item: MenuItem, quantity: number) => void;
  removeItem: (itemId: string) => void;
  updateQuantity: (itemId: string, quantity: number) => void;
  clearCart: () => void;
  total: () => number;
}

export const useCartStore = create<CartState>()(
  persist(
    (set, get) => ({
      items: [],
      chefId: null,
      addItem: (item, quantity) => {
        const currentItems = get().items;
        const existingIndex = currentItems.findIndex(i => i.id === item.id);

        if (existingIndex > -1) {
          const updated = [...currentItems];
          updated[existingIndex].quantity += quantity;
          set({ items: updated });
        } else {
          set({
            items: [...currentItems, { ...item, quantity }],
            chefId: item.chefId
          });
        }
      },
      removeItem: (itemId) => {
        set({ items: get().items.filter(i => i.id !== itemId) });
      },
      updateQuantity: (itemId, quantity) => {
        const updated = get().items.map(i =>
          i.id === itemId ? { ...i, quantity } : i
        );
        set({ items: updated });
      },
      clearCart: () => set({ items: [], chefId: null }),
      total: () => get().items.reduce((sum, i) => sum + (i.price * i.quantity), 0),
    }),
    { name: 'cart-storage' }
  )
);
```

### 3.3 API Layer with Mock Support

```typescript
// src/shared/services/api-client.ts

import axios, { AxiosInstance } from 'axios';
import { env } from '@/app/config/env';

const MOCK_MODE = env.MOCK_MODE === 'true';

class ApiClient {
  private client: AxiosInstance;
  private mockHandlers: Map<string, Function>;

  constructor() {
    this.client = axios.create({
      baseURL: env.API_BASE_URL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.mockHandlers = new Map();
    this.setupInterceptors();
  }

  private setupInterceptors() {
    // Request interceptor - add auth token
    this.client.interceptors.request.use((config) => {
      const token = useAuthStore.getState().token;
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    });

    // Response interceptor - handle errors
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          useAuthStore.getState().logout();
        }
        return Promise.reject(error);
      }
    );
  }

  async get<T>(endpoint: string, params?: object): Promise<T> {
    if (MOCK_MODE) {
      return this.getMockData<T>(endpoint, params);
    }
    const response = await this.client.get<T>(endpoint, { params });
    return response.data;
  }

  async post<T>(endpoint: string, data?: object): Promise<T> {
    if (MOCK_MODE) {
      return this.postMockData<T>(endpoint, data);
    }
    const response = await this.client.post<T>(endpoint, data);
    return response.data;
  }

  async put<T>(endpoint: string, data?: object): Promise<T> {
    if (MOCK_MODE) {
      return this.putMockData<T>(endpoint, data);
    }
    const response = await this.client.put<T>(endpoint, data);
    return response.data;
  }

  async delete<T>(endpoint: string): Promise<T> {
    if (MOCK_MODE) {
      return this.deleteMockData<T>(endpoint);
    }
    const response = await this.client.delete<T>(endpoint);
    return response.data;
  }

  // Mock data methods - delegates to mock service
  private async getMockData<T>(endpoint: string, params?: object): Promise<T> {
    const { mockService } = await import('@/mock/mock-service');
    return mockService.get<T>(endpoint, params);
  }

  private async postMockData<T>(endpoint: string, data?: object): Promise<T> {
    const { mockService } = await import('@/mock/mock-service');
    return mockService.post<T>(endpoint, data);
  }

  private async putMockData<T>(endpoint: string, data?: object): Promise<T> {
    const { mockService } = await import('@/mock/mock-service');
    return mockService.put<T>(endpoint, data);
  }

  private async deleteMockData<T>(endpoint: string): Promise<T> {
    const { mockService } = await import('@/mock/mock-service');
    return mockService.delete<T>(endpoint);
  }
}

export const apiClient = new ApiClient();
```

### 3.4 React Query Integration

```typescript
// src/features/customer/browse/hooks/useChefs.ts

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { chefService } from '../services/chef-service';
import { ChefFilters, Chef } from '../types';

export const chefKeys = {
  all: ['chefs'] as const,
  lists: () => [...chefKeys.all, 'list'] as const,
  list: (filters: ChefFilters) => [...chefKeys.lists(), filters] as const,
  details: () => [...chefKeys.all, 'detail'] as const,
  detail: (id: string) => [...chefKeys.details(), id] as const,
};

export function useChefs(filters: ChefFilters) {
  return useQuery({
    queryKey: chefKeys.list(filters),
    queryFn: () => chefService.getChefs(filters),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

export function useChef(id: string) {
  return useQuery({
    queryKey: chefKeys.detail(id),
    queryFn: () => chefService.getChefById(id),
    enabled: !!id,
  });
}

export function useChefMenu(chefId: string) {
  return useQuery({
    queryKey: ['chef', chefId, 'menu'],
    queryFn: () => chefService.getChefMenu(chefId),
    enabled: !!chefId,
  });
}

// Mutation example
export function useToggleFavoriteChef() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (chefId: string) => chefService.toggleFavorite(chefId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['favorites'] });
    },
  });
}
```

### 3.5 Component Architecture

```typescript
// src/shared/components/ui/Button.tsx
// Using shadcn/ui pattern

import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/shared/utils/cn';

const buttonVariants = cva(
  'inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90',
        destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
        outline: 'border border-input hover:bg-accent hover:text-accent-foreground',
        secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
        ghost: 'hover:bg-accent hover:text-accent-foreground',
        link: 'underline-offset-4 hover:underline text-primary',
      },
      size: {
        default: 'h-10 py-2 px-4',
        sm: 'h-9 px-3 rounded-md',
        lg: 'h-11 px-8 rounded-md',
        icon: 'h-10 w-10',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  loading?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, loading, children, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button';
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        disabled={loading || props.disabled}
        {...props}
      >
        {loading ? (
          <>
            <span className="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
            Loading...
          </>
        ) : (
          children
        )}
      </Comp>
    );
  }
);
Button.displayName = 'Button';

export { Button, buttonVariants };
```

---

## 4. Backend Architecture

### 4.1 Project Structure (Go)

```
homechef-api/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
│
├── internal/
│   ├── config/                  # Configuration
│   │   ├── config.go
│   │   └── env.go
│   │
│   ├── domain/                  # Domain models (entities)
│   │   ├── user.go
│   │   ├── chef.go
│   │   ├── order.go
│   │   ├── menu_item.go
│   │   ├── delivery.go
│   │   └── payment.go
│   │
│   ├── repository/              # Data access layer
│   │   ├── interfaces.go
│   │   ├── user_repository.go
│   │   ├── chef_repository.go
│   │   ├── order_repository.go
│   │   └── ...
│   │
│   ├── service/                 # Business logic layer
│   │   ├── user_service.go
│   │   ├── chef_service.go
│   │   ├── order_service.go
│   │   ├── delivery_service.go
│   │   ├── payment_service.go
│   │   └── notification_service.go
│   │
│   ├── handler/                 # HTTP handlers (controllers)
│   │   ├── user_handler.go
│   │   ├── chef_handler.go
│   │   ├── order_handler.go
│   │   ├── delivery_handler.go
│   │   └── admin_handler.go
│   │
│   ├── middleware/              # HTTP middleware
│   │   ├── auth.go
│   │   ├── cors.go
│   │   ├── logging.go
│   │   ├── ratelimit.go
│   │   └── recovery.go
│   │
│   ├── router/                  # Route definitions
│   │   └── router.go
│   │
│   └── dto/                     # Data transfer objects
│       ├── requests/
│       └── responses/
│
├── pkg/                         # Public packages
│   ├── auth/                    # JWT, OAuth
│   │   ├── jwt.go
│   │   └── oauth.go
│   ├── database/                # DB connection
│   │   ├── postgres.go
│   │   └── redis.go
│   ├── messaging/               # Message queue
│   │   └── rabbitmq.go
│   ├── storage/                 # File storage
│   │   └── s3.go
│   ├── email/                   # Email service
│   │   └── sendgrid.go
│   ├── sms/                     # SMS service
│   │   └── twilio.go
│   ├── payment/                 # Payment gateway
│   │   └── stripe.go
│   ├── maps/                    # Maps service
│   │   └── google.go
│   └── validator/               # Input validation
│       └── validator.go
│
├── migrations/                  # Database migrations
├── docs/                        # API documentation
│   └── swagger.yaml
├── scripts/                     # Build/deploy scripts
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── Makefile
```

### 4.2 Domain Models

```go
// internal/domain/user.go

package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleCustomer UserRole = "customer"
	RoleChef     UserRole = "chef"
	RoleDelivery UserRole = "delivery"
	RoleAdmin    UserRole = "admin"
)

type User struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email         string     `json:"email" gorm:"uniqueIndex;not null"`
	Phone         string     `json:"phone" gorm:"uniqueIndex"`
	PasswordHash  string     `json:"-" gorm:"not null"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	Avatar        string     `json:"avatar"`
	Role          UserRole   `json:"role" gorm:"type:varchar(20);default:'customer'"`
	EmailVerified bool       `json:"email_verified" gorm:"default:false"`
	PhoneVerified bool       `json:"phone_verified" gorm:"default:false"`
	IsActive      bool       `json:"is_active" gorm:"default:true"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	CreatedAt     time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt     *time.Time `json:"deleted_at" gorm:"index"`

	// Relations
	Addresses []Address `json:"addresses,omitempty" gorm:"foreignKey:UserID"`
	ChefProfile *ChefProfile `json:"chef_profile,omitempty" gorm:"foreignKey:UserID"`
}

type Address struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID     uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	Label      string    `json:"label"` // Home, Work, etc.
	Line1      string    `json:"line1" gorm:"not null"`
	Line2      string    `json:"line2"`
	City       string    `json:"city" gorm:"not null"`
	State      string    `json:"state" gorm:"not null"`
	PostalCode string    `json:"postal_code" gorm:"not null"`
	Country    string    `json:"country" gorm:"not null;default:'US'"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	IsDefault  bool      `json:"is_default" gorm:"default:false"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
```

```go
// internal/domain/chef.go

package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ChefStatus string

const (
	ChefStatusPending  ChefStatus = "pending"
	ChefStatusApproved ChefStatus = "approved"
	ChefStatusRejected ChefStatus = "rejected"
	ChefStatusSuspended ChefStatus = "suspended"
)

type ChefProfile struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID          uuid.UUID      `json:"user_id" gorm:"type:uuid;uniqueIndex;not null"`
	BusinessName    string         `json:"business_name" gorm:"not null"`
	Description     string         `json:"description"`
	Cuisines        pq.StringArray `json:"cuisines" gorm:"type:text[]"`
	Specialties     pq.StringArray `json:"specialties" gorm:"type:text[]"`
	ProfileImage    string         `json:"profile_image"`
	BannerImage     string         `json:"banner_image"`
	KitchenImages   pq.StringArray `json:"kitchen_images" gorm:"type:text[]"`
	Status          ChefStatus     `json:"status" gorm:"type:varchar(20);default:'pending'"`
	Rating          float64        `json:"rating" gorm:"default:0"`
	TotalReviews    int            `json:"total_reviews" gorm:"default:0"`
	TotalOrders     int            `json:"total_orders" gorm:"default:0"`
	IsOnline        bool           `json:"is_online" gorm:"default:false"`
	AcceptingOrders bool           `json:"accepting_orders" gorm:"default:false"`

	// Location
	AddressID       uuid.UUID `json:"address_id" gorm:"type:uuid"`
	ServiceRadius   float64   `json:"service_radius" gorm:"default:10"` // km
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`

	// Operating Hours (stored as JSON)
	OperatingHours  string `json:"operating_hours" gorm:"type:jsonb"`

	// Verification
	IDVerified      bool       `json:"id_verified" gorm:"default:false"`
	KitchenVerified bool       `json:"kitchen_verified" gorm:"default:false"`
	FoodLicense     string     `json:"food_license"`
	LicenseVerified bool       `json:"license_verified" gorm:"default:false"`
	VerifiedAt      *time.Time `json:"verified_at"`

	// Bank details for payouts
	BankAccountID   string `json:"bank_account_id"`

	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`

	// Relations
	User      User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	MenuItems []MenuItem `json:"menu_items,omitempty" gorm:"foreignKey:ChefID"`
}
```

```go
// internal/domain/order.go

package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusAccepted   OrderStatus = "accepted"
	OrderStatusPreparing  OrderStatus = "preparing"
	OrderStatusReady      OrderStatus = "ready"
	OrderStatusPickedUp   OrderStatus = "picked_up"
	OrderStatusDelivering OrderStatus = "delivering"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusRefunded   OrderStatus = "refunded"
)

type Order struct {
	ID              uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrderNumber     string      `json:"order_number" gorm:"uniqueIndex;not null"`
	CustomerID      uuid.UUID   `json:"customer_id" gorm:"type:uuid;not null"`
	ChefID          uuid.UUID   `json:"chef_id" gorm:"type:uuid;not null"`
	DeliveryID      *uuid.UUID  `json:"delivery_id" gorm:"type:uuid"`
	Status          OrderStatus `json:"status" gorm:"type:varchar(20);default:'pending'"`

	// Delivery Address
	DeliveryAddressID uuid.UUID `json:"delivery_address_id" gorm:"type:uuid;not null"`
	DeliveryLat       float64   `json:"delivery_lat"`
	DeliveryLng       float64   `json:"delivery_lng"`
	DeliveryAddress   string    `json:"delivery_address"` // Denormalized for history

	// Pricing
	Subtotal       float64 `json:"subtotal" gorm:"type:decimal(10,2);not null"`
	DeliveryFee    float64 `json:"delivery_fee" gorm:"type:decimal(10,2);default:0"`
	ServiceFee     float64 `json:"service_fee" gorm:"type:decimal(10,2);default:0"`
	Tax            float64 `json:"tax" gorm:"type:decimal(10,2);default:0"`
	Discount       float64 `json:"discount" gorm:"type:decimal(10,2);default:0"`
	Tip            float64 `json:"tip" gorm:"type:decimal(10,2);default:0"`
	Total          float64 `json:"total" gorm:"type:decimal(10,2);not null"`

	// Promo
	PromoCode      string `json:"promo_code"`

	// Special Instructions
	Instructions   string `json:"instructions"`

	// Timing
	ScheduledFor   *time.Time `json:"scheduled_for"`
	EstimatedReady *time.Time `json:"estimated_ready"`
	EstimatedDelivery *time.Time `json:"estimated_delivery"`
	AcceptedAt     *time.Time `json:"accepted_at"`
	PreparedAt     *time.Time `json:"prepared_at"`
	PickedUpAt     *time.Time `json:"picked_up_at"`
	DeliveredAt    *time.Time `json:"delivered_at"`
	CancelledAt    *time.Time `json:"cancelled_at"`
	CancelReason   string     `json:"cancel_reason"`

	// Payment
	PaymentID      string `json:"payment_id"`
	PaymentStatus  string `json:"payment_status"`
	PaymentMethod  string `json:"payment_method"`

	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`

	// Relations
	Customer      User          `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	Chef          ChefProfile   `json:"chef,omitempty" gorm:"foreignKey:ChefID"`
	Items         []OrderItem   `json:"items,omitempty" gorm:"foreignKey:OrderID"`
	Delivery      *Delivery     `json:"delivery,omitempty" gorm:"foreignKey:DeliveryID"`
}

type OrderItem struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrderID     uuid.UUID `json:"order_id" gorm:"type:uuid;not null"`
	MenuItemID  uuid.UUID `json:"menu_item_id" gorm:"type:uuid;not null"`
	Name        string    `json:"name" gorm:"not null"` // Denormalized
	Price       float64   `json:"price" gorm:"type:decimal(10,2);not null"`
	Quantity    int       `json:"quantity" gorm:"not null;default:1"`
	Subtotal    float64   `json:"subtotal" gorm:"type:decimal(10,2);not null"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}
```

### 4.3 Service Layer

```go
// internal/service/order_service.go

package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"homechef/internal/domain"
	"homechef/internal/repository"
	"homechef/pkg/messaging"
)

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidOrderState = errors.New("invalid order state transition")
	ErrChefUnavailable   = errors.New("chef is not accepting orders")
)

type OrderService struct {
	orderRepo    repository.OrderRepository
	chefRepo     repository.ChefRepository
	menuRepo     repository.MenuItemRepository
	userRepo     repository.UserRepository
	paymentSvc   *PaymentService
	notifySvc    *NotificationService
	messageBus   messaging.MessageBus
}

func NewOrderService(
	orderRepo repository.OrderRepository,
	chefRepo repository.ChefRepository,
	menuRepo repository.MenuItemRepository,
	userRepo repository.UserRepository,
	paymentSvc *PaymentService,
	notifySvc *NotificationService,
	messageBus messaging.MessageBus,
) *OrderService {
	return &OrderService{
		orderRepo:  orderRepo,
		chefRepo:   chefRepo,
		menuRepo:   menuRepo,
		userRepo:   userRepo,
		paymentSvc: paymentSvc,
		notifySvc:  notifySvc,
		messageBus: messageBus,
	}
}

type CreateOrderRequest struct {
	CustomerID        uuid.UUID
	ChefID            uuid.UUID
	Items             []OrderItemRequest
	DeliveryAddressID uuid.UUID
	PromoCode         string
	Instructions      string
	ScheduledFor      *time.Time
	PaymentMethod     string
}

type OrderItemRequest struct {
	MenuItemID uuid.UUID
	Quantity   int
	Notes      string
}

func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*domain.Order, error) {
	// 1. Verify chef is accepting orders
	chef, err := s.chefRepo.GetByID(ctx, req.ChefID)
	if err != nil {
		return nil, err
	}
	if !chef.AcceptingOrders || !chef.IsOnline {
		return nil, ErrChefUnavailable
	}

	// 2. Validate and fetch menu items
	var orderItems []domain.OrderItem
	var subtotal float64

	for _, item := range req.Items {
		menuItem, err := s.menuRepo.GetByID(ctx, item.MenuItemID)
		if err != nil {
			return nil, fmt.Errorf("menu item %s not found", item.MenuItemID)
		}
		if !menuItem.IsAvailable {
			return nil, fmt.Errorf("menu item %s is not available", menuItem.Name)
		}

		itemSubtotal := menuItem.Price * float64(item.Quantity)
		orderItems = append(orderItems, domain.OrderItem{
			MenuItemID: item.MenuItemID,
			Name:       menuItem.Name,
			Price:      menuItem.Price,
			Quantity:   item.Quantity,
			Subtotal:   itemSubtotal,
			Notes:      item.Notes,
		})
		subtotal += itemSubtotal
	}

	// 3. Calculate fees
	deliveryFee := s.calculateDeliveryFee(chef, req.DeliveryAddressID)
	serviceFee := subtotal * 0.05 // 5% service fee
	tax := subtotal * 0.08 // 8% tax (simplified)

	// 4. Apply promo code if provided
	discount := 0.0
	if req.PromoCode != "" {
		discount, err = s.applyPromoCode(ctx, req.PromoCode, subtotal)
		if err != nil {
			return nil, err
		}
	}

	total := subtotal + deliveryFee + serviceFee + tax - discount

	// 5. Create order
	order := &domain.Order{
		OrderNumber:       s.generateOrderNumber(),
		CustomerID:        req.CustomerID,
		ChefID:            req.ChefID,
		Status:            domain.OrderStatusPending,
		DeliveryAddressID: req.DeliveryAddressID,
		Subtotal:          subtotal,
		DeliveryFee:       deliveryFee,
		ServiceFee:        serviceFee,
		Tax:               tax,
		Discount:          discount,
		Total:             total,
		PromoCode:         req.PromoCode,
		Instructions:      req.Instructions,
		ScheduledFor:      req.ScheduledFor,
		PaymentMethod:     req.PaymentMethod,
		Items:             orderItems,
	}

	// 6. Process payment
	paymentIntent, err := s.paymentSvc.CreatePaymentIntent(ctx, order)
	if err != nil {
		return nil, fmt.Errorf("payment failed: %w", err)
	}
	order.PaymentID = paymentIntent.ID
	order.PaymentStatus = "pending"

	// 7. Save order
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// 8. Publish order created event
	s.messageBus.Publish("order.created", map[string]interface{}{
		"order_id":    order.ID,
		"customer_id": order.CustomerID,
		"chef_id":     order.ChefID,
	})

	// 9. Notify chef
	s.notifySvc.SendPushNotification(ctx, chef.UserID, "New Order!",
		fmt.Sprintf("You have a new order #%s", order.OrderNumber))

	return order, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, newStatus domain.OrderStatus, actorID uuid.UUID) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	// Validate state transition
	if !s.isValidTransition(order.Status, newStatus) {
		return nil, ErrInvalidOrderState
	}

	// Update timestamps based on status
	now := time.Now()
	switch newStatus {
	case domain.OrderStatusAccepted:
		order.AcceptedAt = &now
		order.EstimatedReady = s.calculateEstimatedReady(order)
	case domain.OrderStatusPreparing:
		// No additional timestamp
	case domain.OrderStatusReady:
		order.PreparedAt = &now
	case domain.OrderStatusPickedUp:
		order.PickedUpAt = &now
	case domain.OrderStatusDelivered:
		order.DeliveredAt = &now
	case domain.OrderStatusCancelled:
		order.CancelledAt = &now
	}

	order.Status = newStatus

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, err
	}

	// Publish status update event
	s.messageBus.Publish("order.status_updated", map[string]interface{}{
		"order_id":   order.ID,
		"old_status": order.Status,
		"new_status": newStatus,
	})

	// Notify customer
	s.notifyCustomerStatusUpdate(ctx, order)

	return order, nil
}

func (s *OrderService) isValidTransition(current, new domain.OrderStatus) bool {
	validTransitions := map[domain.OrderStatus][]domain.OrderStatus{
		domain.OrderStatusPending:    {domain.OrderStatusAccepted, domain.OrderStatusCancelled},
		domain.OrderStatusAccepted:   {domain.OrderStatusPreparing, domain.OrderStatusCancelled},
		domain.OrderStatusPreparing:  {domain.OrderStatusReady, domain.OrderStatusCancelled},
		domain.OrderStatusReady:      {domain.OrderStatusPickedUp},
		domain.OrderStatusPickedUp:   {domain.OrderStatusDelivering},
		domain.OrderStatusDelivering: {domain.OrderStatusDelivered},
	}

	allowed, exists := validTransitions[current]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == new {
			return true
		}
	}
	return false
}

func (s *OrderService) generateOrderNumber() string {
	return fmt.Sprintf("HC%d", time.Now().UnixNano())
}

func (s *OrderService) calculateDeliveryFee(chef *domain.ChefProfile, addressID uuid.UUID) float64 {
	// Simplified - would calculate based on distance
	return 3.99
}

func (s *OrderService) calculateEstimatedReady(order *domain.Order) *time.Time {
	// Simplified - would calculate based on items and chef's avg prep time
	estimated := time.Now().Add(30 * time.Minute)
	return &estimated
}

func (s *OrderService) applyPromoCode(ctx context.Context, code string, subtotal float64) (float64, error) {
	// Simplified promo code logic
	return 0, nil
}

func (s *OrderService) notifyCustomerStatusUpdate(ctx context.Context, order *domain.Order) {
	messages := map[domain.OrderStatus]string{
		domain.OrderStatusAccepted:   "Your order has been accepted! Chef is preparing your food.",
		domain.OrderStatusPreparing:  "Your food is being prepared.",
		domain.OrderStatusReady:      "Your order is ready for pickup!",
		domain.OrderStatusPickedUp:   "Your order has been picked up and is on the way!",
		domain.OrderStatusDelivered:  "Your order has been delivered. Enjoy!",
		domain.OrderStatusCancelled:  "Your order has been cancelled.",
	}

	if msg, ok := messages[order.Status]; ok {
		s.notifySvc.SendPushNotification(ctx, order.CustomerID, "Order Update", msg)
	}
}
```

### 4.4 HTTP Handler Layer

```go
// internal/handler/order_handler.go

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"homechef/internal/dto/requests"
	"homechef/internal/dto/responses"
	"homechef/internal/service"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// CreateOrder godoc
// @Summary Create a new order
// @Description Create a new food order for a customer
// @Tags Orders
// @Accept json
// @Produce json
// @Param order body requests.CreateOrderRequest true "Order details"
// @Success 201 {object} responses.OrderResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Router /orders [post]
// @Security BearerAuth
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req requests.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.Error(err.Error()))
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, _ := c.Get("userID")
	customerID := userID.(uuid.UUID)

	order, err := h.orderService.CreateOrder(c.Request.Context(), service.CreateOrderRequest{
		CustomerID:        customerID,
		ChefID:            req.ChefID,
		Items:             mapOrderItems(req.Items),
		DeliveryAddressID: req.DeliveryAddressID,
		PromoCode:         req.PromoCode,
		Instructions:      req.Instructions,
		ScheduledFor:      req.ScheduledFor,
		PaymentMethod:     req.PaymentMethod,
	})
	if err != nil {
		switch err {
		case service.ErrChefUnavailable:
			c.JSON(http.StatusBadRequest, responses.Error("Chef is not accepting orders"))
		default:
			c.JSON(http.StatusInternalServerError, responses.Error(err.Error()))
		}
		return
	}

	c.JSON(http.StatusCreated, responses.Success(mapOrderToResponse(order)))
}

// GetOrder godoc
// @Summary Get order by ID
// @Description Get details of a specific order
// @Tags Orders
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} responses.OrderResponse
// @Failure 404 {object} responses.ErrorResponse
// @Router /orders/{id} [get]
// @Security BearerAuth
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.Error("Invalid order ID"))
		return
	}

	order, err := h.orderService.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.Error("Order not found"))
		return
	}

	c.JSON(http.StatusOK, responses.Success(mapOrderToResponse(order)))
}

// GetCustomerOrders godoc
// @Summary Get customer orders
// @Description Get all orders for the authenticated customer
// @Tags Orders
// @Produce json
// @Param status query string false "Filter by status"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} responses.OrderListResponse
// @Router /orders [get]
// @Security BearerAuth
func (h *OrderHandler) GetCustomerOrders(c *gin.Context) {
	userID, _ := c.Get("userID")
	customerID := userID.(uuid.UUID)

	filters := service.OrderFilters{
		CustomerID: &customerID,
		Status:     c.Query("status"),
		Page:       parseIntQuery(c, "page", 1),
		Limit:      parseIntQuery(c, "limit", 20),
	}

	orders, total, err := h.orderService.GetOrders(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessList(mapOrdersToResponse(orders), total, filters.Page, filters.Limit))
}

// UpdateOrderStatus godoc
// @Summary Update order status
// @Description Update the status of an order (for chef/delivery)
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param status body requests.UpdateStatusRequest true "New status"
// @Success 200 {object} responses.OrderResponse
// @Failure 400 {object} responses.ErrorResponse
// @Router /orders/{id}/status [put]
// @Security BearerAuth
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.Error("Invalid order ID"))
		return
	}

	var req requests.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.Error(err.Error()))
		return
	}

	userID, _ := c.Get("userID")
	actorID := userID.(uuid.UUID)

	order, err := h.orderService.UpdateOrderStatus(c.Request.Context(), orderID, req.Status, actorID)
	if err != nil {
		switch err {
		case service.ErrOrderNotFound:
			c.JSON(http.StatusNotFound, responses.Error("Order not found"))
		case service.ErrInvalidOrderState:
			c.JSON(http.StatusBadRequest, responses.Error("Invalid status transition"))
		default:
			c.JSON(http.StatusInternalServerError, responses.Error(err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, responses.Success(mapOrderToResponse(order)))
}

func mapOrderItems(items []requests.OrderItemRequest) []service.OrderItemRequest {
	result := make([]service.OrderItemRequest, len(items))
	for i, item := range items {
		result[i] = service.OrderItemRequest{
			MenuItemID: item.MenuItemID,
			Quantity:   item.Quantity,
			Notes:      item.Notes,
		}
	}
	return result
}

func mapOrderToResponse(order *domain.Order) responses.OrderResponse {
	// Map domain order to API response
	return responses.OrderResponse{
		ID:          order.ID,
		OrderNumber: order.OrderNumber,
		Status:      string(order.Status),
		Total:       order.Total,
		// ... other fields
	}
}

func parseIntQuery(c *gin.Context, key string, defaultVal int) int {
	// Parse int from query string with default
	return defaultVal
}
```

---

## 5. Database Design

### 5.1 Entity Relationship Diagram

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│     USERS       │     │  CHEF_PROFILES  │     │   MENU_ITEMS    │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id (PK)         │────<│ user_id (FK)    │────<│ chef_id (FK)    │
│ email           │     │ id (PK)         │     │ id (PK)         │
│ phone           │     │ business_name   │     │ name            │
│ password_hash   │     │ description     │     │ description     │
│ first_name      │     │ cuisines[]      │     │ price           │
│ last_name       │     │ rating          │     │ image_url       │
│ avatar          │     │ is_online       │     │ category        │
│ role            │     │ latitude        │     │ dietary_tags[]  │
│ created_at      │     │ longitude       │     │ is_available    │
└────────┬────────┘     │ verified_at     │     │ prep_time       │
         │              └─────────────────┘     └─────────────────┘
         │
         │
┌────────▼────────┐     ┌─────────────────┐     ┌─────────────────┐
│   ADDRESSES     │     │     ORDERS      │     │   ORDER_ITEMS   │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id (PK)         │     │ id (PK)         │────<│ order_id (FK)   │
│ user_id (FK)    │<────│ customer_id(FK) │     │ id (PK)         │
│ label           │     │ chef_id (FK)    │     │ menu_item_id    │
│ line1           │     │ delivery_id(FK) │     │ name            │
│ city            │     │ status          │     │ price           │
│ postal_code     │     │ total           │     │ quantity        │
│ latitude        │     │ payment_id      │     │ notes           │
│ longitude       │     │ created_at      │     └─────────────────┘
│ is_default      │     └────────┬────────┘
└─────────────────┘              │
                                 │
┌─────────────────┐     ┌────────▼────────┐     ┌─────────────────┐
│DELIVERY_PARTNERS│     │   DELIVERIES    │     │    REVIEWS      │
├─────────────────┤     ├─────────────────┤     ├─────────────────┤
│ id (PK)         │────<│ partner_id (FK) │     │ id (PK)         │
│ user_id (FK)    │     │ id (PK)         │     │ order_id (FK)   │
│ vehicle_type    │     │ order_id (FK)   │     │ customer_id(FK) │
│ license_number  │     │ status          │     │ chef_id (FK)    │
│ is_online       │     │ picked_up_at    │     │ rating          │
│ current_lat     │     │ delivered_at    │     │ comment         │
│ current_lng     │     │ delivery_proof  │     │ created_at      │
│ rating          │     └─────────────────┘     └─────────────────┘
└─────────────────┘
                        ┌─────────────────┐     ┌─────────────────┐
                        │    PAYMENTS     │     │   PROMO_CODES   │
                        ├─────────────────┤     ├─────────────────┤
                        │ id (PK)         │     │ id (PK)         │
                        │ order_id (FK)   │     │ code            │
                        │ amount          │     │ discount_type   │
                        │ status          │     │ discount_value  │
                        │ method          │     │ min_order       │
                        │ provider_ref    │     │ max_uses        │
                        │ created_at      │     │ expires_at      │
                        └─────────────────┘     └─────────────────┘
```

### 5.2 PostgreSQL Schema (see DATABASE_SCHEMA.md for complete schema)

---

## 6. API Design

### 6.1 API Endpoints Overview

```
BASE URL: /api/v1

Authentication:
  POST   /auth/register          - Register new user
  POST   /auth/login             - Login with email/password
  POST   /auth/login/social      - Social login (Google, Facebook, Apple)
  POST   /auth/verify-otp        - Verify phone OTP
  POST   /auth/refresh           - Refresh access token
  POST   /auth/logout            - Logout user
  POST   /auth/forgot-password   - Request password reset
  POST   /auth/reset-password    - Reset password

Users:
  GET    /users/me               - Get current user profile
  PUT    /users/me               - Update profile
  GET    /users/me/addresses     - Get user addresses
  POST   /users/me/addresses     - Add address
  PUT    /users/me/addresses/:id - Update address
  DELETE /users/me/addresses/:id - Delete address

Chefs:
  GET    /chefs                  - List chefs (with filters)
  GET    /chefs/:id              - Get chef details
  GET    /chefs/:id/menu         - Get chef menu
  GET    /chefs/:id/reviews      - Get chef reviews
  GET    /chefs/nearby           - Get nearby chefs

Chef Dashboard (requires chef role):
  GET    /chef/profile           - Get chef profile
  PUT    /chef/profile           - Update chef profile
  POST   /chef/menu              - Add menu item
  PUT    /chef/menu/:id          - Update menu item
  DELETE /chef/menu/:id          - Delete menu item
  PUT    /chef/availability      - Update availability
  GET    /chef/orders            - Get chef orders
  PUT    /chef/orders/:id/status - Update order status
  GET    /chef/earnings          - Get earnings summary
  GET    /chef/analytics         - Get analytics

Orders:
  POST   /orders                 - Create order
  GET    /orders                 - Get customer orders
  GET    /orders/:id             - Get order details
  GET    /orders/:id/track       - Track order (real-time)
  PUT    /orders/:id/cancel      - Cancel order

Cart:
  GET    /cart                   - Get cart
  POST   /cart/items             - Add item to cart
  PUT    /cart/items/:id         - Update cart item
  DELETE /cart/items/:id         - Remove cart item
  DELETE /cart                   - Clear cart

Payments:
  POST   /payments/intent        - Create payment intent
  POST   /payments/confirm       - Confirm payment
  GET    /payments/:id           - Get payment details

Reviews:
  POST   /orders/:id/review      - Submit review
  GET    /reviews                - Get user's reviews

Delivery Partner:
  GET    /delivery/profile       - Get delivery profile
  PUT    /delivery/availability  - Toggle availability
  GET    /delivery/orders        - Get assigned orders
  PUT    /delivery/orders/:id    - Update delivery status
  GET    /delivery/earnings      - Get earnings

Admin:
  GET    /admin/dashboard        - Dashboard stats
  GET    /admin/users            - List users
  GET    /admin/chefs            - List chefs
  PUT    /admin/chefs/:id/verify - Verify chef
  GET    /admin/orders           - List orders
  GET    /admin/analytics        - Analytics data
```

---

## 7. Authentication & Authorization

### 7.1 OAuth 2.0 Flow

```
┌──────────┐     ┌──────────┐     ┌──────────────┐     ┌──────────┐
│  Client  │     │ HomeChef │     │  OAuth       │     │  HomeChef│
│   App    │     │  Backend │     │  Provider    │     │  Database│
└────┬─────┘     └────┬─────┘     └──────┬───────┘     └────┬─────┘
     │                │                   │                  │
     │  1. Login      │                   │                  │
     │  Request       │                   │                  │
     │───────────────>│                   │                  │
     │                │                   │                  │
     │  2. Redirect   │                   │                  │
     │  to Provider   │                   │                  │
     │<───────────────│                   │                  │
     │                │                   │                  │
     │  3. User       │                   │                  │
     │  Authenticates │                   │                  │
     │───────────────────────────────────>│                  │
     │                │                   │                  │
     │  4. Auth Code  │                   │                  │
     │<───────────────────────────────────│                  │
     │                │                   │                  │
     │  5. Exchange   │                   │                  │
     │  Code          │                   │                  │
     │───────────────>│                   │                  │
     │                │  6. Exchange for  │                  │
     │                │  Access Token     │                  │
     │                │──────────────────>│                  │
     │                │                   │                  │
     │                │  7. User Info     │                  │
     │                │<──────────────────│                  │
     │                │                   │                  │
     │                │  8. Create/Update │                  │
     │                │  User             │                  │
     │                │─────────────────────────────────────>│
     │                │                   │                  │
     │  9. JWT Token  │                   │                  │
     │<───────────────│                   │                  │
     │                │                   │                  │
```

### 7.2 JWT Token Structure

```json
{
  "header": {
    "alg": "RS256",
    "typ": "JWT"
  },
  "payload": {
    "sub": "user-uuid",
    "email": "user@example.com",
    "role": "customer",
    "permissions": ["read:orders", "write:orders"],
    "iat": 1700000000,
    "exp": 1700003600,
    "iss": "homechef-api"
  }
}
```

### 7.3 Role-Based Access Control

```
┌─────────────────────────────────────────────────────────────────┐
│                        PERMISSIONS MATRIX                       │
├─────────────────┬──────────┬──────────┬──────────┬─────────────┤
│    Resource     │ Customer │   Chef   │ Delivery │    Admin    │
├─────────────────┼──────────┼──────────┼──────────┼─────────────┤
│ Browse Chefs    │    R     │    R     │    -     │     R       │
│ Own Profile     │   RW     │   RW     │   RW     │    RW       │
│ Own Orders      │   RW     │   RW     │   RW     │    RW       │
│ Menu Items      │    R     │  CRUD    │    -     │   CRUD      │
│ Chef Profile    │    R     │   RW     │    -     │    RW       │
│ Delivery Tasks  │    -     │    R     │   RW     │    RW       │
│ All Users       │    -     │    -     │    -     │   CRUD      │
│ All Chefs       │    -     │    -     │    -     │   CRUD      │
│ Platform Config │    -     │    -     │    -     │   CRUD      │
│ Analytics       │    -     │ Limited  │ Limited  │    Full     │
└─────────────────┴──────────┴──────────┴──────────┴─────────────┘
```

---

## 8. Real-time Communication

### 8.1 WebSocket Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                    WEBSOCKET GATEWAY                             │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                   Connection Manager                       │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐       │  │
│  │  │Customer │  │Customer │  │  Chef   │  │Delivery │       │  │
│  │  │   1     │  │   2     │  │   1     │  │   1     │       │  │
│  │  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘       │  │
│  │       │            │            │            │             │  │
│  │       └────────────┴────────────┴────────────┘             │  │
│  │                         │                                  │  │
│  │              ┌──────────▼──────────┐                       │  │
│  │              │   Event Dispatcher  │                       │  │
│  │              └──────────┬──────────┘                       │  │
│  └─────────────────────────┼──────────────────────────────────┘  │
│                            │                                     │
│  ┌─────────────────────────▼──────────────────────────────────┐  │
│  │                    Event Types                             │  │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐          │  │
│  │  │Order Status │ │  Location   │ │    Chat     │          │  │
│  │  │   Update    │ │   Update    │ │   Message   │          │  │
│  │  └─────────────┘ └─────────────┘ └─────────────┘          │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 8.2 Event Messages

```typescript
// Order Status Update
{
  "type": "ORDER_STATUS_UPDATE",
  "payload": {
    "orderId": "uuid",
    "status": "preparing",
    "timestamp": "2024-01-01T12:00:00Z",
    "estimatedTime": "2024-01-01T12:30:00Z"
  }
}

// Delivery Location Update
{
  "type": "DELIVERY_LOCATION",
  "payload": {
    "orderId": "uuid",
    "deliveryId": "uuid",
    "location": {
      "lat": 12.9716,
      "lng": 77.5946
    },
    "eta": "10 mins"
  }
}

// New Order (for chef)
{
  "type": "NEW_ORDER",
  "payload": {
    "orderId": "uuid",
    "orderNumber": "HC123456",
    "items": [...],
    "total": 25.99
  }
}
```

---

## 9. Infrastructure & Deployment

### 9.1 Cloud Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              AWS / GCP                                   │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                         CloudFront / CDN                          │  │
│  └───────────────────────────────┬───────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────▼───────────────────────────────────┐  │
│  │                     Application Load Balancer                     │  │
│  └───────────────────────────────┬───────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────▼───────────────────────────────────┐  │
│  │                        Kubernetes Cluster                         │  │
│  │  ┌─────────────────────────────────────────────────────────────┐  │  │
│  │  │                         Ingress                             │  │  │
│  │  └─────────────────────────────────────────────────────────────┘  │  │
│  │                              │                                    │  │
│  │  ┌─────────────┬─────────────┼─────────────┬─────────────────┐   │  │
│  │  │             │             │             │                 │   │  │
│  │  ▼             ▼             ▼             ▼                 ▼   │  │
│  │ ┌───┐        ┌───┐        ┌───┐        ┌───┐            ┌───┐   │  │
│  │ │API│        │API│        │API│        │WS │            │Web│   │  │
│  │ │ 1 │        │ 2 │        │ 3 │        │GW │            │App│   │  │
│  │ └───┘        └───┘        └───┘        └───┘            └───┘   │  │
│  │                                                                  │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                  │                                       │
│  ┌───────────────────────────────┼───────────────────────────────────┐  │
│  │                     Data Services                                 │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐         │  │
│  │  │PostgreSQL│  │  Redis   │  │ RabbitMQ │  │Elasticsearch│       │  │
│  │  │ (RDS)    │  │(Elastic) │  │          │  │            │       │  │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘         │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                     Storage & Media                               │  │
│  │  ┌─────────────────┐  ┌─────────────────┐                        │  │
│  │  │    S3 Bucket    │  │   CloudFront    │                        │  │
│  │  │   (Images)      │  │   (Media CDN)   │                        │  │
│  │  └─────────────────┘  └─────────────────┘                        │  │
│  └───────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

### 9.2 Docker Compose (Development)

```yaml
# docker-compose.yml
version: '3.8'

services:
  # Frontend
  web:
    build:
      context: ./apps/web
      dockerfile: Dockerfile.dev
    ports:
      - "3000:3000"
    volumes:
      - ./apps/web:/app
      - /app/node_modules
    environment:
      - VITE_API_URL=http://localhost:8080
      - VITE_MOCK_MODE=false
    depends_on:
      - api

  # Backend API
  api:
    build:
      context: ./apps/api
      dockerfile: Dockerfile.dev
    ports:
      - "8080:8080"
    volumes:
      - ./apps/api:/app
    environment:
      - DATABASE_URL=postgres://homechef:homechef@postgres:5432/homechef?sslmode=disable
      - REDIS_URL=redis://redis:6379
      - JWT_SECRET=your-dev-jwt-secret
      - ENVIRONMENT=development
    depends_on:
      - postgres
      - redis

  # PostgreSQL
  postgres:
    image: postgres:15-alpine
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_USER=homechef
      - POSTGRES_PASSWORD=homechef
      - POSTGRES_DB=homechef
    volumes:
      - postgres_data:/var/lib/postgresql/data

  # Redis
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  # RabbitMQ
  rabbitmq:
    image: rabbitmq:3-management-alpine
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      - RABBITMQ_DEFAULT_USER=homechef
      - RABBITMQ_DEFAULT_PASS=homechef

  # Elasticsearch (for search)
  elasticsearch:
    image: elasticsearch:8.11.0
    ports:
      - "9200:9200"
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
    volumes:
      - es_data:/usr/share/elasticsearch/data

volumes:
  postgres_data:
  redis_data:
  es_data:
```

### 9.3 CI/CD Pipeline

```yaml
# .github/workflows/deploy.yml
name: Deploy HomeChef

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'pnpm'

      - name: Install dependencies
        run: pnpm install

      - name: Run frontend tests
        run: pnpm --filter web test

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run backend tests
        run: cd apps/api && go test ./...

  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4

      - name: Build and push Docker images
        run: |
          docker build -t homechef/web ./apps/web
          docker build -t homechef/api ./apps/api
          docker push homechef/web
          docker push homechef/api

  deploy:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - name: Deploy to Kubernetes
        run: |
          kubectl apply -f k8s/
          kubectl rollout status deployment/web
          kubectl rollout status deployment/api
```

---

## 10. Mock System Architecture

### 10.1 Mock Data Service

```typescript
// src/mock/mock-service.ts

import { delay } from '@/shared/utils/delay';
import { mockChefs } from './data/chefs';
import { mockMenuItems } from './data/menu-items';
import { mockOrders } from './data/orders';
import { mockUsers } from './data/users';

// Simulated network delay
const MOCK_DELAY = 300;

class MockService {
  private data: Map<string, any[]>;

  constructor() {
    this.data = new Map([
      ['users', [...mockUsers]],
      ['chefs', [...mockChefs]],
      ['menuItems', [...mockMenuItems]],
      ['orders', [...mockOrders]],
      ['cart', []],
    ]);
  }

  async get<T>(endpoint: string, params?: object): Promise<T> {
    await delay(MOCK_DELAY);

    const handlers: Record<string, () => any> = {
      '/chefs': () => this.getChefs(params),
      '/chefs/:id': () => this.getChefById(params),
      '/chefs/:id/menu': () => this.getChefMenu(params),
      '/orders': () => this.getOrders(params),
      '/orders/:id': () => this.getOrderById(params),
      '/users/me': () => this.getCurrentUser(),
      '/cart': () => this.getCart(),
    };

    const handler = this.matchEndpoint(endpoint, handlers);
    if (handler) {
      return handler() as T;
    }

    throw new Error(`Mock handler not found for: ${endpoint}`);
  }

  async post<T>(endpoint: string, data?: object): Promise<T> {
    await delay(MOCK_DELAY);

    const handlers: Record<string, () => any> = {
      '/auth/login': () => this.login(data),
      '/auth/register': () => this.register(data),
      '/orders': () => this.createOrder(data),
      '/cart/items': () => this.addToCart(data),
    };

    const handler = this.matchEndpoint(endpoint, handlers);
    if (handler) {
      return handler() as T;
    }

    throw new Error(`Mock handler not found for: ${endpoint}`);
  }

  // Chef methods
  private getChefs(params?: any) {
    let chefs = this.data.get('chefs') || [];

    // Apply filters
    if (params?.cuisine) {
      chefs = chefs.filter(c => c.cuisines.includes(params.cuisine));
    }
    if (params?.search) {
      const search = params.search.toLowerCase();
      chefs = chefs.filter(c =>
        c.businessName.toLowerCase().includes(search) ||
        c.description.toLowerCase().includes(search)
      );
    }
    if (params?.rating) {
      chefs = chefs.filter(c => c.rating >= params.rating);
    }

    // Sort
    if (params?.sort === 'rating') {
      chefs.sort((a, b) => b.rating - a.rating);
    } else if (params?.sort === 'orders') {
      chefs.sort((a, b) => b.totalOrders - a.totalOrders);
    }

    // Pagination
    const page = params?.page || 1;
    const limit = params?.limit || 20;
    const start = (page - 1) * limit;
    const paginatedChefs = chefs.slice(start, start + limit);

    return {
      data: paginatedChefs,
      total: chefs.length,
      page,
      limit,
    };
  }

  private getChefById(params?: any) {
    const chefs = this.data.get('chefs') || [];
    const chef = chefs.find(c => c.id === params?.id);
    if (!chef) {
      throw { status: 404, message: 'Chef not found' };
    }
    return chef;
  }

  private getChefMenu(params?: any) {
    const menuItems = this.data.get('menuItems') || [];
    return menuItems.filter(item => item.chefId === params?.id);
  }

  // Order methods
  private createOrder(data?: any) {
    const orders = this.data.get('orders') || [];
    const newOrder = {
      id: `order-${Date.now()}`,
      orderNumber: `HC${Date.now()}`,
      status: 'pending',
      createdAt: new Date().toISOString(),
      ...data,
    };
    orders.push(newOrder);
    this.data.set('orders', orders);

    // Clear cart after order
    this.data.set('cart', []);

    return newOrder;
  }

  // Auth methods
  private login(data?: any) {
    const users = this.data.get('users') || [];
    const user = users.find(u => u.email === data?.email);

    if (!user || data?.password !== 'password123') {
      throw { status: 401, message: 'Invalid credentials' };
    }

    return {
      user,
      token: 'mock-jwt-token-' + user.id,
      refreshToken: 'mock-refresh-token-' + user.id,
    };
  }

  private matchEndpoint(endpoint: string, handlers: Record<string, () => any>): (() => any) | null {
    // Direct match
    if (handlers[endpoint]) {
      return handlers[endpoint];
    }

    // Pattern matching for parameterized routes
    for (const [pattern, handler] of Object.entries(handlers)) {
      const regex = pattern.replace(/:(\w+)/g, '([^/]+)');
      if (new RegExp(`^${regex}$`).test(endpoint)) {
        return handler;
      }
    }

    return null;
  }
}

export const mockService = new MockService();
```

### 10.2 Mock Data Files

```typescript
// src/mock/data/chefs.ts

import { Chef } from '@/shared/types';

export const mockChefs: Chef[] = [
  {
    id: 'chef-1',
    userId: 'user-chef-1',
    businessName: "Meena's Kitchen",
    description: 'Authentic South Indian home cooking with a modern twist. Specializing in dosas, idlis, and traditional curries.',
    cuisines: ['South Indian', 'Tamil'],
    specialties: ['Dosa', 'Biryani', 'Sambar'],
    profileImage: '/mock-images/chef-1.jpg',
    bannerImage: '/mock-images/chef-1-banner.jpg',
    rating: 4.8,
    totalReviews: 156,
    totalOrders: 423,
    isOnline: true,
    acceptingOrders: true,
    serviceRadius: 10,
    latitude: 12.9716,
    longitude: 77.5946,
    prepTime: '30-45 mins',
    priceRange: '$$',
    deliveryFee: 2.99,
    minimumOrder: 15,
    operatingHours: {
      monday: { open: '08:00', close: '21:00' },
      tuesday: { open: '08:00', close: '21:00' },
      wednesday: { open: '08:00', close: '21:00' },
      thursday: { open: '08:00', close: '21:00' },
      friday: { open: '08:00', close: '22:00' },
      saturday: { open: '09:00', close: '22:00' },
      sunday: { open: '09:00', close: '20:00' },
    },
    verified: true,
    verifiedAt: '2024-01-15T10:00:00Z',
    createdAt: '2023-06-01T08:00:00Z',
  },
  {
    id: 'chef-2',
    userId: 'user-chef-2',
    businessName: 'Nonna Rosa Italian',
    description: 'Traditional Italian recipes passed down through generations. Handmade pasta, wood-fired pizza flavors, and authentic sauces.',
    cuisines: ['Italian', 'Mediterranean'],
    specialties: ['Pasta', 'Pizza', 'Risotto'],
    profileImage: '/mock-images/chef-2.jpg',
    bannerImage: '/mock-images/chef-2-banner.jpg',
    rating: 4.9,
    totalReviews: 234,
    totalOrders: 567,
    isOnline: true,
    acceptingOrders: true,
    serviceRadius: 8,
    latitude: 12.9816,
    longitude: 77.5846,
    prepTime: '40-55 mins',
    priceRange: '$$$',
    deliveryFee: 3.99,
    minimumOrder: 20,
    operatingHours: {
      monday: { open: '11:00', close: '22:00' },
      tuesday: { open: '11:00', close: '22:00' },
      wednesday: { open: '11:00', close: '22:00' },
      thursday: { open: '11:00', close: '22:00' },
      friday: { open: '11:00', close: '23:00' },
      saturday: { open: '11:00', close: '23:00' },
      sunday: { open: '12:00', close: '21:00' },
    },
    verified: true,
    verifiedAt: '2024-02-10T14:00:00Z',
    createdAt: '2023-08-15T10:00:00Z',
  },
  // Add more mock chefs...
];
```

---

## 11. Security Architecture

### 11.1 Security Layers

```
┌─────────────────────────────────────────────────────────────────┐
│                      SECURITY LAYERS                            │
├─────────────────────────────────────────────────────────────────┤
│  Layer 1: Edge Security                                         │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  • DDoS Protection (CloudFlare/AWS Shield)              │   │
│  │  • WAF Rules                                            │   │
│  │  • Rate Limiting                                        │   │
│  │  • TLS 1.3 Encryption                                   │   │
│  └─────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│  Layer 2: Application Security                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  • JWT Token Validation                                 │   │
│  │  • RBAC Authorization                                   │   │
│  │  • Input Validation/Sanitization                        │   │
│  │  • CSRF Protection                                      │   │
│  │  • XSS Prevention                                       │   │
│  │  • SQL Injection Prevention                             │   │
│  └─────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│  Layer 3: Data Security                                         │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  • Encryption at Rest (AES-256)                         │   │
│  │  • Encryption in Transit (TLS)                          │   │
│  │  • PII Data Masking                                     │   │
│  │  • Secure Key Management (AWS KMS)                      │   │
│  │  • Database Access Controls                             │   │
│  └─────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│  Layer 4: Infrastructure Security                               │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  • VPC Isolation                                        │   │
│  │  • Security Groups                                      │   │
│  │  • Secrets Management                                   │   │
│  │  • Container Security                                   │   │
│  │  • Regular Security Audits                              │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## 12. Monitoring & Observability

### 12.1 Observability Stack

```
┌─────────────────────────────────────────────────────────────────┐
│                   OBSERVABILITY STACK                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │   METRICS   │  │   LOGGING   │  │   TRACING   │            │
│  │             │  │             │  │             │            │
│  │ Prometheus  │  │   ELK/Loki  │  │   Jaeger    │            │
│  │ + Grafana   │  │             │  │             │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    ALERTING                             │   │
│  │  ┌───────────┐  ┌───────────┐  ┌───────────┐           │   │
│  │  │PagerDuty  │  │   Slack   │  │   Email   │           │   │
│  │  └───────────┘  └───────────┘  └───────────┘           │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  ERROR TRACKING                         │   │
│  │              Sentry / Bugsnag                           │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 12.2 Key Metrics

| Category | Metric | Alert Threshold |
|----------|--------|-----------------|
| **API** | Response Time (p95) | > 500ms |
| **API** | Error Rate | > 1% |
| **API** | Request Rate | < 10% of baseline |
| **Database** | Connection Pool | > 80% utilized |
| **Database** | Query Time (p95) | > 100ms |
| **Cache** | Hit Rate | < 80% |
| **Orders** | Failed Orders | > 5% |
| **Payments** | Failed Transactions | > 2% |
| **Delivery** | Average Time | > 60 mins |

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | Dec 2024 | HomeChef Team | Initial architecture document |

---

*This document is confidential and intended for internal development use only.*
