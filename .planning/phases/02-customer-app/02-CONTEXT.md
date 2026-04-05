# Phase 2: Customer App - Context

**Gathered:** 2026-04-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Complete customer commerce loop in the mobile app: onboarding, chef discovery, browsing menus, cart, Razorpay checkout, live order tracking with GPS map, order history, favorites, social feed, catering requests, and profile management. All screens consume the existing Go API endpoints.

</domain>

<decisions>
## Implementation Decisions

### Home & Chef Discovery
- **D-01:** Browse layout, chef detail page, and search/filter UX at Claude's discretion — recommend card grid (2 columns) for browse, sticky header with menu tabs for chef detail, and search bar with filter chips for discovery
- **D-02:** Must use `@tesserix/native` components and NativeWind styling throughout (carried from Phase 1 D-10, D-11)

### Cart & Checkout
- **D-03:** Cart UX at Claude's discretion — recommend bottom sheet cart pattern (floating bar with swipe-up for full cart)
- **D-04:** Payment via Razorpay React Native SDK (`razorpay-react-native-checkout`) — native payment sheet, not WebView. Must verify Expo managed workflow compatibility; if incompatible, fall back to `expo-web-browser` with Razorpay web checkout URL

### Order Tracking Map
- **D-05:** Live tracking via React Query polling every 5 seconds — uses existing `PUT /delivery/location` + delivery status endpoints. No WebSocket, no backend changes.
- **D-06:** Map view layout at Claude's discretion — recommend full-screen map with bottom sheet for order details and status timeline
- **D-07:** Use `react-native-maps` for the map component (confirmed compatible with Expo managed workflow in Phase 1 research)

### Secondary Screens
- **D-08:** Onboarding flow at Claude's discretion — recommend multi-step wizard (swipeable: basic info → address → food preferences) matching web's component structure
- **D-09:** Secondary screens (favorites, social feed, catering, profile, order history) at Claude's discretion — same data/API endpoints as web but adapted for mobile interaction patterns (bottom sheets, swipe gestures, pull-to-refresh)

### Tab Navigation
- **D-10:** Customer app tab bar structure at Claude's discretion — recommend: Home, Orders, Favorites, Profile (4 tabs)

### Claude's Discretion
- Browse layout and chef detail page design (D-01)
- Cart UX pattern (D-03)
- Map view layout (D-06)
- Onboarding flow design (D-08)
- All secondary screen mobile adaptations (D-09)
- Tab navigation structure (D-10)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Context
- `.planning/PROJECT.md` — Project vision, constraints, key decisions
- `.planning/REQUIREMENTS.md` — CUST-01 through CUST-11

### Phase 1 Foundation
- `.planning/phases/01-foundation-auth/01-CONTEXT.md` — Monorepo structure, auth strategy, design system decisions
- `packages/mobile-shared/src/api/client.ts` — API client factory with 401 interceptor (reuse for all customer API calls)
- `packages/mobile-shared/src/hooks/useAuth.ts` — Auth hooks (reuse for auth guard)
- `packages/mobile-shared/src/theme/tokens.ts` — Design token bridge from @tesserix/native

### Existing Web Customer App
- `apps/web/src/features/customer/pages/HomePage.tsx` — Web home page (reference for mobile)
- `apps/web/src/features/customer/pages/BrowseChefsPage.tsx` — Chef browse grid layout
- `apps/web/src/features/customer/pages/ChefDetailPage.tsx` — Chef detail with menu
- `apps/web/src/features/customer/pages/CartPage.tsx` — Cart implementation
- `apps/web/src/features/customer/pages/CheckoutPage.tsx` — Checkout with Razorpay
- `apps/web/src/features/customer/pages/OrderDetailPage.tsx` — Order tracking
- `apps/web/src/features/customer/pages/OrdersPage.tsx` — Order history
- `apps/web/src/features/customer/pages/FavoritesPage.tsx` — Favorites list
- `apps/web/src/features/customer/pages/ProfilePage.tsx` — Profile management
- `apps/web/src/features/social/pages/SocialFeedPage.tsx` — Social feed
- `apps/web/src/features/catering/pages/CateringRequestPage.tsx` — Catering request form
- `apps/web/src/features/catering/pages/CateringQuotesPage.tsx` — Catering quotes list
- `apps/web/src/features/onboarding/pages/UserInfoPage.tsx` — Onboarding wizard

### Go API Endpoints
- `apps/api/handlers/chefs.go` — Chef listing, detail, search endpoints
- `apps/api/handlers/menu.go` — Menu items for a chef
- `apps/api/handlers/orders.go` — Order creation, history, detail
- `apps/api/handlers/payment.go` — Razorpay payment creation and verification
- `apps/api/handlers/delivery.go` — Delivery tracking, driver location
- `apps/api/handlers/favorites.go` — Favorites CRUD
- `apps/api/handlers/social.go` — Social feed endpoints
- `apps/api/handlers/catering.go` — Catering request and quotes
- `apps/api/handlers/customer.go` — Customer profile, onboarding

### Research
- `.planning/research/STACK.md` — react-native-maps, Expo SDK version
- `.planning/research/PITFALLS.md` — Razorpay WebView behavior, location polling DB load

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `packages/mobile-shared/src/api/client.ts` — createApiClient with Bearer injection + 401 refresh. Customer app creates its own instance with `EXPO_PUBLIC_API_URL`
- `packages/mobile-shared/src/hooks/useAuth.ts` — useAuthStore Zustand store with hydrateFromStorage. Already wired in Phase 1
- `packages/mobile-shared/src/theme/tokens.ts` — @tesserix/native color/spacing/typography tokens
- `packages/mobile-shared/src/screens/` — Shared auth screens pattern to follow for shared customer screens if applicable
- `apps/web/src/features/customer/` — All web customer pages as reference implementations

### Established Patterns
- Zustand v5 for client state (auth store, cart state) + React Query for server state (chef lists, orders)
- NativeWind className props on all custom components
- expo-router file-based routing with route groups: (auth)/, (tabs)/, (onboarding)/
- Auth guard in root _layout.tsx redirects unauthenticated users

### Integration Points
- `apps/mobile-customer/app/(tabs)/` — Tab screens to be created here
- `apps/mobile-customer/app/(tabs)/index.tsx` — Stub home screen from Phase 1, replace with actual home
- `apps/mobile-customer/lib/api.ts` — Per-app API client instance from Phase 1
- `apps/mobile-customer/store/auth-store.ts` — Auth store re-export from Phase 1

</code_context>

<specifics>
## Specific Ideas

- Razorpay React Native SDK specifically requested over WebView approach
- Same color tokens and theme as web — brand consistency is non-negotiable
- Live tracking via polling (not WebSocket) for v1 — keep it simple

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-customer-app*
*Context gathered: 2026-04-05*
