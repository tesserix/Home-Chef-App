# Architecture Patterns: Expo Mobile Apps in Existing Monorepo

**Domain:** Food delivery mobile apps (Expo/React Native)
**Researched:** 2026-04-05
**Confidence note:** Web search and WebFetch tools were unavailable during research. All findings are based on training knowledge (cutoff August 2025) of the Expo, expo-router, and pnpm monorepo ecosystems, cross-referenced with this project's existing codebase structure. Confidence levels reflect this.

---

## Recommended Architecture

### High-Level Overview

Three independent Expo apps live alongside the existing web apps in `apps/`. They share typed API client code and domain types via a workspace package (`packages/mobile-shared/`). Each app is fully standalone for building and deployment (EAS Build), while sharing common logic through the workspace package — not through cross-app imports.

```
Home-Chef-App/
├── apps/
│   ├── api/                        # Go API backend (unchanged)
│   ├── web/                        # Customer web (unchanged)
│   ├── vendor-portal/              # Vendor web (unchanged)
│   ├── admin-portal/               # Admin web (unchanged)
│   ├── delivery-portal/            # Delivery web (unchanged)
│   ├── mobile-customer/            # NEW: Expo customer app
│   ├── mobile-vendor/              # NEW: Expo vendor/chef app
│   └── mobile-delivery/            # NEW: Expo driver app
└── packages/
    └── mobile-shared/              # NEW: Shared API client, types, hooks
```

The `packages/` directory already exists in `pnpm-workspace.yaml` (`packages/*`) but is currently unused. This is the right home for shared mobile code.

---

## Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| `packages/mobile-shared/` | API client factory, domain types, shared hooks, auth token management, push notification registration | Consumed by all 3 mobile apps |
| `apps/mobile-customer/` | Customer UX: browse chefs, cart, checkout, order tracking with GPS map, social feed, catering | `mobile-shared`, `@tesserix/native`, Go API |
| `apps/mobile-vendor/` | Vendor UX: menu management, live orders, earnings, analytics, kitchen settings | `mobile-shared`, `@tesserix/native`, Go API |
| `apps/mobile-delivery/` | Driver UX: available deliveries, active delivery GPS navigation, fleet, earnings | `mobile-shared`, `@tesserix/native`, Go API |
| `@tesserix/native` | Brand-consistent React Native UI components, design tokens | Consumed by all 3 mobile apps |
| `apps/api/` | Go REST API (no changes required) | Consumed by all 3 mobile apps via HTTP |

**Strict rule:** Mobile apps do NOT import from each other. All cross-app shared code lives in `packages/mobile-shared/` only.

---

## Monorepo Structure: Each Expo App

Each mobile app follows this internal layout, driven by expo-router's file-system conventions:

```
apps/mobile-customer/
├── app/                            # expo-router: all routes live here
│   ├── _layout.tsx                 # Root layout: providers, auth guard
│   ├── (auth)/                     # Auth group (no tab bar)
│   │   ├── _layout.tsx
│   │   ├── login.tsx
│   │   ├── register.tsx
│   │   └── forgot-password.tsx
│   ├── (onboarding)/               # Onboarding group (no tab bar)
│   │   ├── _layout.tsx
│   │   └── [step].tsx              # Dynamic step: address, preferences
│   └── (tabs)/                     # Tab group (persistent tab bar)
│       ├── _layout.tsx             # Tab bar definition
│       ├── index.tsx               # Home / chef discovery
│       ├── orders/
│       │   ├── index.tsx           # Order history
│       │   └── [id]/
│       │       ├── index.tsx       # Order detail
│       │       └── tracking.tsx    # Live GPS tracking
│       ├── favorites.tsx
│       ├── social.tsx
│       └── profile/
│           ├── index.tsx
│           └── settings.tsx
├── components/                     # App-specific components
│   ├── chef/
│   ├── cart/
│   ├── checkout/
│   └── shared/                     # Small reusable pieces not in design system
├── hooks/                          # App-specific hooks
├── store/                          # Zustand stores (auth, cart, etc.)
├── lib/                            # App-specific utilities
├── constants/                      # Colors, sizes, route names
├── app.json                        # Expo app config
├── eas.json                        # EAS Build config
├── metro.config.js                 # Metro bundler (REQUIRED for monorepo)
├── babel.config.js
├── tsconfig.json
└── package.json
```

The vendor and delivery apps follow the same shape with their own feature-specific tab structures.

---

## Shared Code Strategy (`packages/mobile-shared/`)

This package is the single source of shared code across all 3 mobile apps. It does NOT contain UI components — those come from `@tesserix/native`.

```
packages/mobile-shared/
├── src/
│   ├── api/
│   │   ├── client.ts               # Axios instance factory: injects base URL + auth header
│   │   ├── auth.ts                 # login(), register(), refreshToken() calls
│   │   ├── orders.ts               # getOrders(), createOrder(), getOrderById()
│   │   ├── chefs.ts                # getChefs(), getChefById(), getChefMenu()
│   │   ├── delivery.ts             # getDeliveries(), updateDeliveryStatus()
│   │   ├── menu.ts                 # getMenuItems(), createMenuItem()
│   │   ├── customer.ts             # getProfile(), updateProfile(), getFavorites()
│   │   ├── payments.ts             # initializePayment(), getPaymentStatus()
│   │   ├── notifications.ts        # registerPushToken(), getNotifications()
│   │   └── index.ts                # Re-exports
│   ├── types/
│   │   ├── user.ts                 # User, AuthUser, JWTClaims
│   │   ├── chef.ts                 # Chef, KitchenDetails
│   │   ├── order.ts                # Order, OrderItem, OrderStatus
│   │   ├── menu.ts                 # MenuItem, MenuCategory
│   │   ├── delivery.ts             # Delivery, DeliveryStatus, Location
│   │   ├── payment.ts              # Payment, PaymentStatus
│   │   └── index.ts
│   ├── hooks/
│   │   ├── useAuth.ts              # JWT storage, login/logout, token refresh
│   │   ├── useLocation.ts          # expo-location wrapper: get/watch position
│   │   ├── usePushNotifications.ts # expo-notifications: register token, handle events
│   │   └── index.ts
│   └── utils/
│       ├── storage.ts              # expo-secure-store wrappers for token persistence
│       ├── format.ts               # Currency, date, distance formatting
│       └── index.ts
├── package.json
└── tsconfig.json
```

**Key decision: API client is a factory, not a singleton.** Each app calls `createApiClient({ baseURL, getToken })` passing its own token retrieval function. This keeps `mobile-shared` stateless and testable.

**Confidence: HIGH** — This pattern is standard for pnpm monorepos and matches how the existing web apps share `@tesserix/web`.

---

## Navigation Architecture (expo-router)

**Use expo-router v3+ file-based routing.** This matches the project's TypeScript/React conventions and gives deep linking for free.

### Route Group Structure Per App

**Customer App tabs:** Home (chef discovery), Orders, Favorites, Social, Profile

**Vendor App tabs:** Dashboard, Orders (live), Menu, Earnings, Profile

**Delivery App tabs:** Dashboard, Available, Active Delivery (map), History, Profile

### Layout Hierarchy

```
Root _layout.tsx
  └── Providers: QueryClientProvider, ThemeProvider (from @tesserix/native), AuthProvider
      └── Auth gate: redirect to (auth)/login if not authenticated
          └── Onboarding gate: redirect to (onboarding) if profile incomplete
              └── (tabs)/_layout.tsx: tab bar with badge counts
                  └── Feature screens
```

### Deep Linking

expo-router automatically generates deep link schemes. Configure in `app.json`:

```json
{
  "expo": {
    "scheme": "homechef-customer",
    "android": { "intentFilters": [...] },
    "ios": { "associatedDomains": [...] }
  }
}
```

Each app gets a distinct scheme: `homechef-customer://`, `homechef-vendor://`, `homechef-driver://`.

**Confidence: HIGH** — expo-router file-based routing is the current standard (Expo SDK 51+) and well-established as of August 2025.

---

## State Management Approach

Mirror what the web apps already use (Zustand + React Query) — the team already knows these patterns.

### Server State: TanStack React Query

All API calls go through React Query. The query client is configured in the root `_layout.tsx` per app. Query functions call the typed API functions from `mobile-shared`.

```
useQuery({ queryKey: ['orders', userId], queryFn: () => api.getOrders() })
useMutation({ mutationFn: api.createOrder, onSuccess: () => queryClient.invalidateQueries(...) })
```

### Local / Session State: Zustand

Each app has its own store directory. Common stores:

| Store | Contents | Scope |
|-------|----------|-------|
| `auth-store.ts` | JWT token, user profile, isAuthenticated | All 3 apps |
| `cart-store.ts` | Cart items, total, chef context | Customer only |
| `order-tracking-store.ts` | Live order location, status polling | Customer only |
| `active-delivery-store.ts` | Current delivery, driver location, navigation state | Delivery only |
| `live-orders-store.ts` | Incoming orders queue, accept/reject state | Vendor only |

**Confidence: HIGH** — TanStack Query v5 + Zustand v5 work with React Native/Expo exactly as they do in the web apps. No special mobile considerations.

### Token Persistence

**Do NOT use localStorage (not available in React Native).** Use `expo-secure-store` from `mobile-shared/utils/storage.ts`. The auth store hydrates from secure store on app launch.

```typescript
// Conceptual pattern — NOT localStorage
await SecureStore.setItemAsync('auth_token', token)
const token = await SecureStore.getItemAsync('auth_token')
```

**Confidence: HIGH** — expo-secure-store is the standard for token persistence in Expo apps.

---

## Metro Bundler Configuration (Critical Monorepo Requirement)

**This is the single most important configuration step.** Metro (React Native's bundler) does not understand pnpm workspaces by default. Without this config, Metro will fail to resolve packages outside each app's `node_modules`.

Each Expo app needs `metro.config.js`:

```javascript
// apps/mobile-customer/metro.config.js
const { getDefaultConfig } = require('expo/metro-config');
const { FileStore } = require('metro-cache');
const path = require('path');

const projectRoot = __dirname;
const monorepoRoot = path.resolve(projectRoot, '../..');

const config = getDefaultConfig(projectRoot);

// Watch monorepo root so Metro can resolve workspace packages
config.watchFolders = [monorepoRoot];

// Add monorepo node_modules to resolver paths
config.resolver.nodeModulesPaths = [
  path.resolve(projectRoot, 'node_modules'),
  path.resolve(monorepoRoot, 'node_modules'),
];

// Required for pnpm: resolve symlinked packages
config.resolver.disableHierarchicalLookup = true;

config.cacheStores = [
  new FileStore({ root: path.join(projectRoot, '.metro-cache') }),
];

module.exports = config;
```

All three apps need this file with the same content (only `projectRoot` changes automatically via `__dirname`).

**Confidence: HIGH** — This is the documented Expo monorepo setup requirement, unchanged since Expo SDK 49.

---

## Data Flow Direction

```
Go API (apps/api/)
    ^
    | HTTP (Bearer JWT)
    |
packages/mobile-shared/api/*        <-- typed API functions, axios instance
    ^
    | imported by
    |
apps/mobile-{customer,vendor,delivery}/
    hooks/ (React Query useQuery/useMutation wrapping mobile-shared API)
    store/ (Zustand: session state, local UI state)
    app/   (expo-router screens consuming hooks and stores)
```

**GPS/Maps data flow (delivery app):**

```
expo-location (background location)
    |
active-delivery-store (Zustand)
    |
PATCH /api/v1/delivery/{id}/location  (polling or WebSocket — check API capability)
    |
Customer tracking screen (via React Query polling GET /api/v1/orders/{id}/tracking)
```

**Push notification data flow:**

```
Expo Push Service (managed by EAS)
    ^
    | push token registered at login
    |
apps/api/ (stores token, sends via Firebase FCM on order events)
    |
mobile-shared/hooks/usePushNotifications.ts (registers token, handles foreground events)
    |
App-specific notification handlers (navigate to order, show alert)
```

---

## Build Order (What Depends on What)

This determines the phase sequencing for implementation.

```
Phase 1: packages/mobile-shared/
  - No dependencies on apps, only on expo-secure-store and axios
  - Must exist before any app can be built
  - Build: tsc --noEmit (type check only, no bundling needed for workspace package)

Phase 2: apps/mobile-{customer,vendor,delivery}/ (can be built in parallel)
  - Depends on: packages/mobile-shared/, @tesserix/native
  - Build: expo export / eas build
  - Metro config must reference monorepoRoot correctly

EAS Build (CI):
  - Each app has its own eas.json
  - Build profiles: development (simulator), preview (internal testing), production
  - GitHub Actions: trigger per-app on path changes (apps/mobile-customer/**)
```

**Dependency graph:**

```
@tesserix/native         (external, pre-existing)
packages/mobile-shared/  (internal, build first)
    |-- apps/mobile-customer/
    |-- apps/mobile-vendor/
    └-- apps/mobile-delivery/
```

There are no circular dependencies. The three mobile apps are independent of each other at build time.

---

## Patterns to Follow

### Pattern 1: Query-First Screen Architecture

Each screen is thin — it calls a hook, renders loading/error states, then renders data. No direct API calls in screen files.

```typescript
// apps/mobile-customer/app/(tabs)/index.tsx
export default function HomeScreen() {
  const { data: chefs, isLoading, error } = useChefs({ latitude, longitude });

  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorView message={error.message} />;

  return <ChefList chefs={chefs} />;
}
```

### Pattern 2: Auth Guard in Root Layout

Centralize auth checking in `app/_layout.tsx` using `useAuth()` from `mobile-shared`. Never check auth state inside individual screens.

```typescript
// apps/mobile-customer/app/_layout.tsx
export default function RootLayout() {
  const { isAuthenticated, isLoading } = useAuth();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace('/(auth)/login');
    }
  }, [isAuthenticated, isLoading]);

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <Stack />
      </ThemeProvider>
    </QueryClientProvider>
  );
}
```

### Pattern 3: Typed API Client per App

Each app creates its API client instance at startup, injecting the token getter from its auth store.

```typescript
// apps/mobile-customer/lib/api.ts
import { createApiClient } from '@homechef/mobile-shared/api';
import { useAuthStore } from '../store/auth-store';

export const api = createApiClient({
  baseURL: process.env.EXPO_PUBLIC_API_URL,
  getToken: () => useAuthStore.getState().token,
});
```

### Pattern 4: GPS Location Service (Delivery App Only)

Active delivery screen uses expo-location in background task mode. The driver's location is stored in Zustand and periodically synced to the API.

```typescript
// apps/mobile-delivery/hooks/useActiveDelivery.ts
// - Starts background location task on delivery accept
// - Updates active-delivery-store every N seconds
// - Calls PATCH /api/v1/delivery/{id}/location
// - Stops on delivery complete
```

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: Cross-App Imports

**What:** Importing directly from another app, e.g., `import { foo } from '../../mobile-customer/...'`
**Why bad:** Creates coupling between independent apps, breaks EAS Build isolation
**Instead:** Extract to `packages/mobile-shared/`

### Anti-Pattern 2: Putting API Calls Directly in Screen Files

**What:** `fetch('/api/v1/chefs')` inside a screen component
**Why bad:** Untestable, no caching, duplicated error handling
**Instead:** API call in `mobile-shared/api/`, wrapped in React Query hook in `hooks/`, consumed in screen

### Anti-Pattern 3: Shared Metro Cache Across Apps

**What:** Pointing all apps at the same `.metro-cache` directory
**Why bad:** Metro cache is app-specific (different entry points, different resolvers)
**Instead:** Each app has `.metro-cache/` in its own directory (add to `.gitignore`)

### Anti-Pattern 4: Using localStorage / AsyncStorage for JWT

**What:** Storing auth token in `AsyncStorage` (unencrypted)
**Why bad:** AsyncStorage is unencrypted, readable on rooted/jailbroken devices
**Instead:** `expo-secure-store` (uses iOS Keychain / Android Keystore)

### Anti-Pattern 5: Separate QueryClient Instances in Child Components

**What:** Creating `new QueryClient()` inside a feature component
**Why bad:** Breaks shared cache, causes redundant API calls
**Instead:** Single `QueryClient` created once in `app/_layout.tsx`, passed via `QueryClientProvider`

---

## pnpm Workspace Integration

### Root `pnpm-workspace.yaml` (already correct, no change needed)

```yaml
packages:
  - "apps/*"
  - "packages/*"
```

### Mobile Shared Package `package.json`

```json
{
  "name": "@homechef/mobile-shared",
  "version": "1.0.0",
  "private": true,
  "main": "src/index.ts",
  "exports": {
    "./api": "./src/api/index.ts",
    "./types": "./src/types/index.ts",
    "./hooks": "./src/hooks/index.ts",
    "./utils": "./src/utils/index.ts"
  }
}
```

No build step required — Metro resolves TypeScript source directly. This avoids a tsup/tsc compile step for the shared package.

### Mobile App `package.json` (example)

```json
{
  "name": "@homechef/mobile-customer",
  "dependencies": {
    "@homechef/mobile-shared": "workspace:*",
    "@tesserix/native": "^1.0.0",
    "expo": "~52.0.0",
    "expo-router": "~4.0.0",
    "react": "18.3.1",
    "react-native": "0.76.x",
    "@tanstack/react-query": "^5.x",
    "zustand": "^5.x",
    "expo-secure-store": "~14.0.0",
    "expo-location": "~18.0.0",
    "expo-notifications": "~0.29.0",
    "react-native-maps": "~1.18.0"
  }
}
```

**Note on React version:** Expo SDK 52 ships with React 18.3.1 and React Native 0.76. The web apps use React 19. pnpm handles this correctly since each app has isolated `node_modules` — the mobile apps use React 18 and web apps use React 19 without conflict.

**Confidence: MEDIUM** — Expo SDK 52 + React 18.3.1 is confirmed accurate as of August 2025. Specific patch versions may have advanced. Verify exact version ranges when scaffolding.

---

## Scalability Considerations

| Concern | Now (3 apps, small team) | Later (if needed) |
|---------|--------------------------|-------------------|
| Shared code growth | Single `packages/mobile-shared/` is fine | Split into `mobile-api`, `mobile-hooks` if >2000 lines |
| Build times | EAS Build cloud, parallel per app | Enable EAS Build caching by commit hash |
| Design system | `@tesserix/native` (external) | No change needed |
| Real-time | REST polling for order tracking | Add WebSocket support to `mobile-shared` transport layer |
| GPS background | expo-location background task | No change — already the right API |

---

## Phase-Specific Notes for Roadmap

**Phase 1 — Foundation (must come first):**
- Create `packages/mobile-shared/` with API client, types, and auth hook
- Configure Metro in all 3 apps with monorepo watchFolders
- Set up expo-router root layout with QueryClientProvider + AuthProvider
- Verify `@tesserix/native` resolves correctly in Metro
- This phase unblocks all subsequent feature work across all 3 apps

**Phase 2 — Auth + Onboarding (one app to validate, then replicate):**
- Build customer app auth first (simplest onboarding flow)
- Validate the full auth → onboarding → main tabs navigation stack works
- Replicate pattern to vendor and delivery apps (same structure, different steps)
- Token storage via expo-secure-store confirmed working

**Phase 3 — Core Features (can parallelise per app after Phase 2):**
- Customer: chef discovery + menu + cart + checkout
- Vendor: live orders (push notifications critical here) + menu management
- Delivery: available deliveries + active delivery GPS map

**Phase 4 — GPS / Real-Time (delivery app, then customer tracking):**
- expo-location background task (requires app permissions, EAS Build to test on device)
- react-native-maps for driver navigation view and customer tracking view
- This phase requires physical device testing — simulators cannot test background GPS

**Phase 5 — Notifications + EAS Build CI:**
- expo-notifications push token registration
- EAS Build GitHub Actions integration
- TestFlight / Internal Testing setup

---

## Sources

All findings are from training knowledge (Expo SDK documentation, expo-router documentation, pnpm monorepo guides as of August 2025 training cutoff).

- Expo monorepo guide: https://docs.expo.dev/guides/monorepos/ (not fetched — permission denied)
- expo-router docs: https://docs.expo.dev/router/introduction/ (not fetched — permission denied)
- TanStack Query React Native: https://tanstack.com/query/latest/docs/framework/react/react-native (not fetched)
- expo-secure-store: https://docs.expo.dev/versions/latest/sdk/securestore/ (not fetched)

**Confidence summary:**

| Area | Confidence | Basis |
|------|------------|-------|
| Metro monorepo config | HIGH | Core Expo requirement, unchanged since SDK 49 |
| expo-router file-based routing | HIGH | SDK 51+ default, well-established |
| pnpm workspace integration | HIGH | Standard pnpm behavior, matches existing workspace.yaml |
| Zustand + React Query for state | HIGH | Same libraries as web apps, React Native compatible |
| expo-secure-store for tokens | HIGH | Standard recommendation, no alternative |
| Expo SDK version (52) | MEDIUM | Verify current SDK version at scaffold time |
| `@tesserix/native` package details | LOW | Package confirmed to exist per PROJECT.md, internals unknown |
| Real-time/WebSocket support in API | LOW | Not confirmed — current API may be REST-only |
