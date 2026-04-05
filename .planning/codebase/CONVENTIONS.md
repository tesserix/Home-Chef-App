# Coding Conventions

**Analysis Date:** 2026-04-05

## Naming Patterns

**Files:**
- React components: PascalCase with `.tsx` extension (e.g., `LoginPage.tsx`, `AuthProvider.tsx`)
- Services: camelCase with `-service.ts` suffix (e.g., `auth-service.ts`, `api-client.ts`, `upload-service.ts`)
- Hooks: `use` prefix in camelCase (e.g., `useMobile.ts`, `useDraftForm.ts`, `useAuth()` hook function)
- Utilities: camelCase with descriptive names (e.g., `cn.ts`, `animations.ts`)
- Types/Models: `.ts` files in `shared/types/` directory (e.g., `auth.ts`, `index.ts`)
- Go handlers: `{resource}.go` in `handlers/` (e.g., `orders.go`, `chefs.go`)
- Go models: PascalCase structs in `models/{resource}.go` (e.g., `user.go`, `chef.go`)

**Functions:**
- React components: PascalCase (e.g., `export function LoginPage()`, `export function AuthProvider()`)
- Async service methods: camelCase with action verb (e.g., `getSession()`, `refreshSession()`, `logout()`)
- Handler methods in Go: PascalCase (e.g., `CreateOrder()`, `GetChefs()`)
- Custom hooks: `use` prefix in camelCase (e.g., `useIsMobile()`, `useBreakpoint()`, `useOnlineStatus()`)
- Utility functions: camelCase, declarative names (e.g., `getBreakpoint()`, `buildUrl()`)

**Variables:**
- React component props: camelCase (e.g., `className`, `children`, `isLoading`)
- State variables: camelCase (e.g., `isAuthenticated`, `needsOnboarding`, `onboardingStatus`)
- Constants in Go: PascalCase with prefixes for type (e.g., `RoleCustomer`, `ProviderGoogle`, `RoleAdmin`)
- Loop variables: short camelCase or underscore (e.g., `i`, `_`)
- Destructured values: camelCase matching property names (e.g., `const { user, isAuthenticated } = ...`)

**Types & Interfaces:**
- TypeScript interfaces: PascalCase with `Props` suffix for component props (e.g., `AuthContextValue`, `CreateOrderRequest`, `DashboardStats`)
- Go structs: PascalCase (e.g., `User`, `ChefProfile`, `Order`)
- Go constants for enum-like values: PascalCase (e.g., `RoleChef`, `ProviderEmail`, `StatusApproved`)
- Type unions in TypeScript: camelCase with meaningful descriptors (e.g., `OnboardingStatus = 'not_started' | 'in_progress' | 'submitted'`)

## Code Style

**Formatting:**
- TypeScript/JavaScript: Prettier 3.4.2 with `prettier-plugin-tailwindcss` enabled
- Go: Standard Go formatting via `gofmt`
- Line length: Prettier default (80 characters for most content)
- Indentation: 2 spaces (TypeScript), implicit via gofmt (Go)

**Linting:**
- TypeScript/JavaScript: ESLint 9.17.0 with typescript-eslint
- Config location: `eslint.config.js` per app (e.g., `apps/vendor-portal/eslint.config.js`)
- Key rules: React Hooks linting, React Refresh warnings for component exports, unused variables flagged if not prefixed with `_`
- No explicit Prettier config file; uses Prettier defaults with Tailwind CSS class sorting

**Tailwind CSS:**
- Automatic class sorting via `prettier-plugin-tailwindcss`
- Classes are automatically reordered on save/format
- Utility-first approach across all frontends
- Custom CSS in `styles/` directory for animations and global overrides

## Import Organization

**Order:**
1. External packages (React, Zustand, React Router, UI libraries)
2. Relative imports from `@/` alias (features, shared services, components)
3. Type imports separated with `type` keyword

**Example:**
```typescript
import { useState, useEffect, useCallback, type ReactNode } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useAuthStore } from '../store/auth-store';
import { apiClient } from '@/shared/services/api-client';
import type { SessionUser, SocialProvider } from '@/shared/types/auth';
```

**Path Aliases:**
- `@/*` maps to `src/` (configured in `tsconfig.json`)
- All relative imports use `@/` prefix for cross-feature imports
- Local imports use relative paths (e.g., `../store/auth-store`)

**Go imports:**
- Standard library imports first (e.g., `"context"`, `"log"`, `"time"`)
- Third-party packages next (e.g., `"github.com/gin-gonic/gin"`)
- Internal imports last (e.g., `"github.com/homechef/api/models"`)

## Error Handling

**TypeScript/JavaScript Patterns:**
- Async operations use try-catch blocks with proper error narrowing
- Error type is `unknown`, narrowed via `instanceof Error` check
- API errors wrapped in `ApiError` interface with `success: false`, `error: { code, message }`
- Silent catches for non-critical operations (e.g., logout attempt, CSRF token fetch)
- Session errors (401) clear auth state via Zustand store without hard redirect

**Go Patterns:**
- Functions return `error` as last return value
- Errors checked immediately with early return pattern: `if err != nil { c.JSON(...); return }`
- Validation errors return 400 with `gin.H{"error": "message"}`
- Database errors return 400 with descriptive messages (e.g., "Chef not found or not available")
- No panics in handlers; only in `main()` for startup failures

**Examples:**
```typescript
// TypeScript error handling
try {
  const res = await fetch(url);
  if (!res.ok) return null;
  return await res.json();
} catch {
  return null;
}

// API response error handling
if (response.status === 401) {
  useAuthStore.getState().clearAuth();
  throw { success: false, error: { code: 'SESSION_EXPIRED' } };
}
```

```go
// Go error handling
func (h *OrderHandler) CreateOrder(c *gin.Context) {
  var req CreateOrderRequest
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
    return
  }
  
  if !chef.AcceptingOrders {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Chef is not accepting orders"})
    return
  }
}
```

## Logging

**Framework:**
- TypeScript: `console` object for development logging (no logger library imported in samples)
- Go: Standard `log` package for startup/shutdown events
- All logs to stdout for container collection

**Patterns:**
- Development: Text format via `log.Printf()`
- Console logs avoided in production code per linting rules (would be flagged by post-tool hooks)
- Structured logging in async operations: log error context on try-catch failure

**Example:**
```go
if err := database.Connect(); err != nil {
  log.Fatalf("Failed to connect to database: %v", err)
}
```

## Comments

**When to Comment:**
- JSDoc comments on exported functions and services (2-3 line descriptions)
- TypeScript methods in classes: `//` comments explaining complex logic
- Go code: Comments on exported types/functions per Go convention
- Inline comments for non-obvious algorithms or workarounds

**JSDoc/TSDoc Format:**
- Used extensively in service files (e.g., `auth-service.ts`, `api-client.ts`)
- Format: `/** @param ... @returns ... */` for functions with complex signatures
- Example: `/** Build the OIDC login URL. Redirects to Keycloak via BFF. @param provider - Optional social provider @param returnTo - URL to redirect back to after login */`

**Go Comments:**
- Exported types/functions must have comment starting with function/type name
- Package-level comments in main handler files
- Example: `// CreateOrder creates a new order`

## Function Design

**Size:**
- Target: < 50 lines per function
- Complex handlers split into smaller private helper methods
- Example: `CreateOrder()` is ~150 lines but includes full validation + item calculation inline

**Parameters:**
- TypeScript: Use interfaces for multiple related parameters (e.g., `CreateOrderRequest` with 10 fields)
- Go: Struct binding with Gin (e.g., `c.ShouldBindJSON(&req)`)
- Callbacks: Typed explicitly (e.g., `onSelect: (id: string) => void`)

**Return Values:**
- Async functions return `Promise<T | null>` when nullability is intentional
- Go functions return `(Result, error)` tuple; error is last
- React components return JSX with proper typing

**Examples:**
```typescript
// TypeScript with interface parameter
async getSession(): Promise<SessionResponse | null> {
  try {
    const res = await fetch(`${BFF_URL}/auth/session`, {
      credentials: 'include',
    });
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}

// Hook with typed return
export function useIsMobile(breakpoint: Breakpoint = 'md'): boolean {
  const [isMobile, setIsMobile] = useState(...);
  // ...
  return isMobile;
}
```

## Module Design

**Exports:**
- Named exports for utilities (e.g., `export function useIsMobile()`, `export const apiClient = ...`)
- Re-exports for barrel files: `export { cn } from '@tesserix/web'`
- Zustand stores exported as `export const useAuthStore = create<AuthState>(...)`

**Barrel Files:**
- `shared/types/index.ts` re-exports all types via `export * from './auth'`
- Service aggregation: `shared/services/` has individual service exports

**Service Singleton Pattern:**
```typescript
export const apiClient = new ApiClient(API_URL);
export const authService = { getLoginUrl, getSession, logout, ... };
```

**Zustand Store Pattern:**
```typescript
export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  setSession: (user) => set({ user, isAuthenticated: true }),
  initialize: async () => { ... }
}));
```

## TypeScript Configuration

**Strict Mode:** Enabled (`strict: true` in `tsconfig.json`)
- `noUnusedLocals`: true — unused variables cause compile error (prefix with `_` to ignore)
- `noUnusedParameters`: true — unused parameters cause compile error
- `noFallthroughCasesInSwitch`: true — switch statements must have break/return
- `noUncheckedIndexedAccess`: true — array/object access must be checked for undefined

**Type Inference:**
- Let TypeScript infer obvious local variable types
- Explicit types required for exported functions and component props
- Function return types explicitly declared

## React Patterns

**Component Structure:**
- Functional components with hooks (no class components)
- Props typed via named interface with `Props` suffix or as destructured parameters
- Custom hooks extracted for reusable logic (e.g., `useMobile`, `useOnlineStatus`)
- Context Providers with custom hooks (e.g., `AuthProvider` with `useAuth()` hook)

**State Management:**
- Zustand for global state (auth, onboarding form data)
- React Query for server state (list queries, mutations)
- Local useState for UI state (modals, form input)

**Example:**
```typescript
interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const { user, isAuthenticated } = useAuthStore();
  const { login, logout } = useAuth();
  
  return <AuthContext.Provider value={...}>{children}</AuthContext.Provider>;
}
```

## Go Patterns

**Handler Methods:**
- Receive `*gin.Context` parameter
- Parse request via `c.ShouldBindJSON(&request)`
- Validate immediately after binding
- Return JSON response via `c.JSON(statusCode, gin.H{...})`
- Early return on error (no error handling chains)

**Service Methods:**
- Encapsulate business logic separate from handlers
- Async patterns via goroutines with channels (if needed)
- Dependency injection via constructor (e.g., `NewOrderHandler()`)

**Model Conventions:**
- GORM struct tags for database mapping: `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
- JSON struct tags for API responses: `json:"id"` with `,omitempty` for optional fields
- Relationships via pointers and `gorm:"foreignKey:..."` tags

---

*Convention analysis: 2026-04-05*
