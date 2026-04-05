# Testing Patterns

**Analysis Date:** 2026-04-05

## Test Framework

**Runner:**
- Vitest v2.1.8 (configured for all TypeScript apps)
- Config: Implicit Vite-based configuration (no separate vitest.config.ts found)
- Go testing: Standard `testing` package (import "testing")

**Assertion Library:**
- Vitest built-in assertions (no explicit chai/assert imports detected)
- Go: No external assertion library; standard `testing.T` with manual assertions

**Run Commands:**
```bash
# TypeScript/React apps
pnpm test                 # Run all tests in watch mode
pnpm test:coverage        # Generate coverage report (--coverage flag)

# All workspaces
pnpm -r test              # Run tests across all packages
pnpm -r test:coverage     # Coverage across all packages

# Go API
go test ./...             # (Standard Go test runner)
go test -v ./...          # Verbose output
go test -race ./...       # Race condition detection
go test -coverprofile=coverage.out ./...  # Coverage report
```

## Test File Organization

**Location:**
- TypeScript tests co-located with source files (same directory)
- Naming convention: Not yet implemented (no `.test.ts` or `.spec.ts` files found in codebase)
- Go tests: `{package}_test.go` in same directory as source

**Structure:**
- Feature-based directory organization (`src/features/{feature}/`)
- Each feature has `pages/`, `services/`, `hooks/` subdirectories
- Tests would be placed alongside implementation files

**Current State:**
- Testing infrastructure configured but **NO TESTS CURRENTLY WRITTEN**
- Vitest is installed and configured as a dev dependency
- No test coverage metrics established yet

## Test Structure

**TypeScript Pattern (Not Yet Implemented):**
Based on Vitest 2.1.8 setup, tests would follow this structure:

```typescript
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';

describe('useAuth hook', () => {
  beforeEach(() => {
    // Setup
  });

  afterEach(() => {
    // Cleanup
  });

  it('should return null for unauthenticated users', () => {
    // Arrange
    // Act
    // Assert
  });
});
```

**Go Pattern (Standard testing):**
```go
package models_test

import (
  "testing"
  "github.com/homechef/api/models"
)

func TestUserValidation(t *testing.T) {
  user := &models.User{
    Email: "test@example.com",
  }
  
  if user.Email != "test@example.com" {
    t.Errorf("Expected email to be test@example.com, got %s", user.Email)
  }
}
```

## Mocking

**Framework:**
- Vitest supports mocking via `vi.mock()` (built-in)
- Go: No mocking library detected; manual mocking via interfaces

**Patterns (Not Yet Implemented):**
TypeScript would use Vitest mocking:
```typescript
import { describe, it, expect, vi } from 'vitest';

vi.mock('@/shared/services/api-client');

describe('authService', () => {
  it('should call apiClient.get', async () => {
    // Arrange
    const mockGet = vi.fn().mockResolvedValue({ authenticated: true });
    
    // Act
    // Assert
    expect(mockGet).toHaveBeenCalledWith(...);
  });
});
```

**What to Mock:**
- External API calls (fetch, apiClient methods)
- Zustand store initializations
- React Router hooks (useNavigate, useLocation)
- Async service operations

**What NOT to Mock:**
- Core utility functions (cn, getBreakpoint)
- Local state management within component
- Component rendering logic (test via React Testing Library if tests exist)

## Fixtures and Factories

**Test Data:**
- No fixtures or factories currently implemented
- Would be placed in `src/mock/` directory (exists but contains mock-service.ts for development)
- Example location: `src/__fixtures__/user.fixture.ts` or `src/mock/factories/`

**Mock Service:**
- File: `src/mock/mock-service.ts` (exists in vendor-portal and web apps)
- Purpose: Development mocking, not for tests
- Contains sample data for features during UI development

## Coverage

**Requirements:**
- Not yet enforced (no configuration in package.json)
- Recommended minimum: 80%+ for services and hooks
- Lower priority for UI component rendering (focus on business logic)

**View Coverage:**
```bash
pnpm test:coverage
# Generates coverage in {app}/coverage/ directory
# Coverage report types: LCOV, JSON summary
```

**Current State:**
- Coverage tool installed (`@vitest/coverage-v8` v2.1.8)
- No CI integration or minimum thresholds configured

## Test Types

**Unit Tests:**
- Scope: Individual functions, hooks, utilities
- Approach: Pure function testing with mock inputs
- Examples to implement:
  - `useIsMobile()` - responsive hook with breakpoint detection
  - `authService.getLoginUrl()` - URL building with parameters
  - `getBreakpoint()` - width-to-breakpoint mapping

**Integration Tests:**
- Scope: API endpoints, services with database interaction
- Approach: Test handler + database layer together
- Examples to implement:
  - `CreateOrder()` handler - request → validation → DB save → response
  - `GetChefs()` handler - query parsing → database query → pagination
  - Authentication flow - login → session → redirect

**E2E Tests:**
- Framework: Not configured (Playwright not in package.json)
- Critical flows: Auth flow, order creation, chef onboarding (would need implementation)
- Approach: Browser automation testing user journeys

## Test Execution Flow

**CI/CD Integration:**
- No test step found in GitHub Actions workflows (if they exist)
- Testing not yet part of build pipeline
- Linting (`eslint`) and type checking (`tsc`) are pre-commit hooks

**Development Workflow:**
```bash
# Watch mode during development
pnpm test

# Before committing
pnpm lint              # ESLint
pnpm typecheck         # tsc
pnpm test:coverage     # Verify coverage (when tests exist)
```

## Common Testing Patterns (To Be Implemented)

**Async Testing:**
```typescript
import { describe, it, expect } from 'vitest';

describe('async operations', () => {
  it('should handle promise resolution', async () => {
    const result = await authService.getSession();
    expect(result).toBeDefined();
  });

  it('should handle promise rejection', async () => {
    expect(async () => {
      await riskyOperation();
    }).rejects.toThrow();
  });
});
```

**Hook Testing:**
```typescript
import { renderHook, act } from '@testing-library/react';
import { useIsMobile } from '@/shared/hooks/useMobile';

describe('useIsMobile', () => {
  it('should detect mobile when window < 768px', () => {
    // Mock window.innerWidth
    global.innerWidth = 500;
    
    const { result } = renderHook(() => useIsMobile('md'));
    expect(result.current).toBe(true);
  });
});
```

**Zustand Store Testing:**
```typescript
import { describe, it, expect } from 'vitest';
import { useAuthStore } from '@/app/store/auth-store';

describe('useAuthStore', () => {
  it('should initialize with null user', () => {
    const { user, isAuthenticated } = useAuthStore.getState();
    expect(user).toBeNull();
    expect(isAuthenticated).toBe(false);
  });

  it('should set session on login', () => {
    const mockUser = { id: '123', email: 'test@example.com' };
    useAuthStore.getState().setSession(mockUser);
    
    const { user, isAuthenticated } = useAuthStore.getState();
    expect(user).toEqual(mockUser);
    expect(isAuthenticated).toBe(true);
  });
});
```

## Testing TODOs

**High Priority (Critical Paths):**
- [ ] Authentication flow tests (login, session refresh, logout)
- [ ] Order creation and validation
- [ ] Chef onboarding workflow
- [ ] API response handling and error cases

**Medium Priority:**
- [ ] Responsive hook tests (useIsMobile, useBreakpoint, etc.)
- [ ] Form validation and submission
- [ ] State management tests (Zustand stores)
- [ ] Service method tests (authService, apiClient)

**Low Priority:**
- [ ] UI component rendering tests
- [ ] Animation utility tests
- [ ] Utility function tests (cn, getBreakpoint)

## Go Testing Guidelines

**Setup:**
```go
func TestCreateOrder(t *testing.T) {
  // Setup test database
  // Setup test fixtures
  
  // Test logic
  
  // Teardown
}
```

**Handler Testing:**
- Create test HTTP requests with Gin context
- Mock database with test fixtures
- Assert response status and JSON payload

**Model Testing:**
- Test GORM hooks (BeforeSave, AfterCreate)
- Test validation logic
- Test relationships and foreign keys

## Notes

- **Current Status:** Testing framework is configured but no tests have been written yet
- **First Tests:** Should focus on auth service and handlers (critical user flows)
- **Integration:** Testing will need to be added to CI/CD pipeline once tests are implemented
- **Coverage Target:** Aim for 80%+ coverage on services and business logic, lower for UI rendering

---

*Testing analysis: 2026-04-05*
