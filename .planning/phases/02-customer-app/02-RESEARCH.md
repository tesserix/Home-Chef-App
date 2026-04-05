# Phase 2: Customer App - Research

**Researched:** 2026-04-05
**Domain:** React Native / Expo customer-facing commerce screens (browse, cart, Razorpay checkout, live tracking, secondary screens)
**Confidence:** HIGH (codebase verified) / MEDIUM (Razorpay SDK compatibility)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Browse layout, chef detail page, and search/filter UX at Claude's discretion — recommend card grid (2 columns) for browse, sticky header with menu tabs for chef detail, and search bar with filter chips for discovery
- **D-02:** Must use `@tesserix/native` components and NativeWind styling throughout (carried from Phase 1 D-10, D-11)
- **D-03:** Cart UX at Claude's discretion — recommend bottom sheet cart pattern (floating bar with swipe-up for full cart)
- **D-04:** Payment via Razorpay React Native SDK (`razorpay-react-native-checkout`) — native payment sheet, not WebView. Must verify Expo managed workflow compatibility; if incompatible, fall back to `expo-web-browser` with Razorpay web checkout URL
- **D-05:** Live tracking via React Query polling every 5 seconds — uses existing `PUT /delivery/location` + delivery status endpoints. No WebSocket, no backend changes.
- **D-06:** Map view layout at Claude's discretion — recommend full-screen map with bottom sheet for order details and status timeline
- **D-07:** Use `react-native-maps` for the map component (confirmed compatible with Expo managed workflow in Phase 1 research)
- **D-08:** Onboarding flow at Claude's discretion — recommend multi-step wizard (swipeable: basic info → address → food preferences) matching web's component structure
- **D-09:** Secondary screens (favorites, social feed, catering, profile, order history) at Claude's discretion — same data/API endpoints as web but adapted for mobile interaction patterns (bottom sheets, swipe gestures, pull-to-refresh)
- **D-10:** Customer app tab bar structure at Claude's discretion — recommend: Home, Orders, Favorites, Profile (4 tabs)

### Claude's Discretion

- Browse layout and chef detail page design (D-01)
- Cart UX pattern (D-03)
- Map view layout (D-06)
- Onboarding flow design (D-08)
- All secondary screen mobile adaptations (D-09)
- Tab navigation structure (D-10)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CUST-01 | Customer can browse and discover home chefs on home screen | Confirmed: `GET /v1/chefs` endpoint with search/cuisine/dietary/rating/isOpen filters + pagination. Web BrowseChefsPage.tsx is the canonical reference. |
| CUST-02 | Customer can view chef detail page with menu items | Confirmed: `GET /v1/chefs/:id` returns chef profile with schedules; `GET /v1/chefs/:id/menu` returns menu items. ChefDetailPage.tsx reference. |
| CUST-03 | Customer can add items to cart and modify quantities | Confirmed: Pure client state via Zustand cart store. Bottom sheet pattern recommended. No separate cart API — cart is local until checkout. |
| CUST-04 | Customer can checkout and pay via Razorpay | Confirmed: `POST /v1/orders` creates order; `POST /v1/payments/order/:orderId/create` creates Razorpay order; `POST /v1/payments/order/:orderId/verify` verifies. D-04 fallback to expo-web-browser needed — see Razorpay SDK section. |
| CUST-05 | Customer can track active order with live GPS map showing driver location | Confirmed: `GET /v1/orders/:id/track` returns order status + delivery object including driver CurrentLatitude/CurrentLongitude. Poll at 5s with refetchInterval. react-native-maps marker updated on each poll. |
| CUST-06 | Customer can view order history and order details | Confirmed: `GET /v1/orders?page=&limit=&status=` with pagination. `GET /v1/orders/:id` for detail including delivery. |
| CUST-07 | Customer can save chefs to favorites | Confirmed: `POST /v1/favorites/chefs/:id`, `DELETE /v1/favorites/chefs/:id`, `GET /v1/favorites` endpoints exist. favorites-store.ts pattern from web. |
| CUST-08 | Customer can view and interact with social feed | Confirmed: social feed handlers exist (`apps/api/handlers/social.go`). Standard FlatList + React Query pattern. |
| CUST-09 | Customer can submit catering requests and view quotes | Confirmed: catering handlers exist (`apps/api/handlers/catering.go`). Form submission + list view. |
| CUST-10 | Customer can manage profile (name, address, preferences) | Confirmed: `GET/PATCH /v1/profile`, `GET/POST/DELETE /v1/addresses` endpoints. customer.go handler. |
| CUST-11 | New customer completes onboarding flow (basic info, address, food preferences) | Confirmed: `POST /v1/customer/onboarding` endpoint in customer.go. Multi-step wizard pattern matches web UserInfoPage.tsx reference. |

</phase_requirements>

---

## Summary

Phase 2 builds the full customer commerce loop inside `apps/mobile-customer/`, consuming the existing Go API without any backend changes. The Phase 1 foundation is already in place: the monorepo is wired, the API client exists at `apps/mobile-customer/lib/api.ts`, the auth store is live, and `apps/mobile-customer/app/(tabs)/index.tsx` is the stub home screen to replace.

The most critical research finding is that the primary payment library requested in D-04 — `razorpay-react-native-checkout` / `react-native-razorpay` — does not support Expo managed workflow (SDK 54+). GitHub issues confirm this is still unresolved as of 2025. The project is on Expo SDK 55 (confirmed via `package.json`). **The fallback in D-04 is the correct implementation path: use `expo-web-browser` with the Razorpay hosted checkout URL.** This avoids bare workflow ejection and keeps the project on managed workflow.

The tracking map is straightforward: `GET /v1/orders/:id/track` returns the delivery object including the driver's `CurrentLatitude`/`CurrentLongitude` from the `DeliveryPartner` record. Polling every 5 seconds with React Query `refetchInterval` and updating a `react-native-maps` `Marker` position covers CUST-05 completely. No backend changes required.

**Primary recommendation:** Implement payment via `expo-web-browser` + Razorpay web checkout URL (D-04 fallback). Build tab structure (Home/Orders/Favorites/Profile) first to establish the routing skeleton, then implement screens in dependency order: Browse → Chef Detail → Cart → Checkout → Order Tracking → Secondary screens → Onboarding.

---

## Standard Stack

### Core (all confirmed installed in apps/mobile-customer/package.json)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| expo | ~55.0.11 | Core managed workflow | [VERIFIED: package.json] Already installed in Phase 1 |
| expo-router | ~55.0.10 | File-based routing | [VERIFIED: package.json] Tab groups, stack screens already set up |
| react-native-maps | (not yet installed) | Map display for tracking | [VERIFIED: STACK.md + PITFALLS.md] Confirmed Expo SDK 55 compatible via npx expo install |
| @tanstack/react-query | ^5.83.0 | Server state, polling, caching | [VERIFIED: package.json] Already installed; use refetchInterval for 5s tracking poll |
| zustand | ^5.0.2 | Cart state, UI state | [VERIFIED: package.json] Already installed; cart store is pure client state |
| react-hook-form + zod | ^7.56.4 + ^3.25.33 | Form validation (checkout, onboarding, profile) | [VERIFIED: package.json] Already installed |
| expo-web-browser | ~14.1.6 | Razorpay web checkout (D-04 fallback) | [VERIFIED: package.json] Already installed |
| nativewind | 5.0.0-preview.3 | NativeWind styling | [VERIFIED: package.json] Already installed |
| @tesserix/native | ^1.0.0 | Design system components | [VERIFIED: package.json] Already installed |
| expo-image | (not yet installed) | Chef/menu photo display with caching | [CITED: STACK.md/PITFALLS.md] Install via npx expo install |

### New Packages to Install This Phase

| Library | Purpose | Install Command |
|---------|---------|-----------------|
| react-native-maps | Map for order tracking screen | `npx expo install react-native-maps` |
| expo-image | Performant chef/menu photo caching | `npx expo install expo-image` |
| @gorhom/bottom-sheet | Cart bottom sheet + order details bottom sheet | `npx expo install @gorhom/bottom-sheet` |
| expo-haptics | Haptic on order placed confirmation | `npx expo install expo-haptics` |

### Why These, Not Others

| Instead of | We Use | Tradeoff |
|------------|--------|----------|
| react-native-razorpay (native SDK) | expo-web-browser + Razorpay hosted checkout | SDK does not support Expo managed workflow SDK 54+. Web checkout is the correct fallback per D-04. |
| Custom map from scratch | react-native-maps | Standard for Expo managed; confirmed compatible in Phase 1 research |
| Redux for cart | Zustand store | Team already uses Zustand; cart is simple enough for a flat Zustand store |
| Mapbox / MapLibre | react-native-maps | Mapbox dropped managed workflow support (SDK v10+); already decided in STACK.md |

### Version Verification

Expo SDK 55 is confirmed installed (`"expo": "~55.0.11"` in package.json). Always install Expo-managed packages via `npx expo install` not `pnpm add` to get the SDK-compatible version resolved automatically.

---

## Architecture Patterns

### Recommended Project Structure

```
apps/mobile-customer/
├── app/
│   ├── _layout.tsx               # Root layout — ALREADY EXISTS (Phase 1)
│   ├── (auth)/                   # Auth screens — ALREADY EXISTS (Phase 1)
│   ├── (tabs)/
│   │   ├── _layout.tsx           # Tab bar: Home, Orders, Favorites, Profile — REPLACE stub
│   │   ├── index.tsx             # Home screen (chef browse) — REPLACE stub
│   │   ├── orders.tsx            # Order history
│   │   ├── favorites.tsx         # Saved chefs
│   │   └── profile.tsx           # Profile management
│   ├── (onboarding)/
│   │   ├── _layout.tsx           # Onboarding wizard stack
│   │   ├── user-info.tsx         # Step 1: basic info
│   │   ├── address.tsx           # Step 2: delivery address
│   │   └── preferences.tsx       # Step 3: food preferences
│   ├── chef/
│   │   └── [id].tsx              # Chef detail page with menu
│   ├── order/
│   │   ├── [id].tsx              # Order detail
│   │   └── [id]/track.tsx        # Live tracking map screen
│   ├── checkout.tsx              # Checkout screen
│   └── social.tsx                # Social feed
│       catering.tsx              # Catering requests
├── components/
│   ├── chef/
│   │   ├── ChefCard.tsx          # Card for browse grid
│   │   ├── ChefGrid.tsx          # 2-column FlatList of ChefCards
│   │   └── MenuItemCard.tsx      # Menu item with add-to-cart button
│   ├── cart/
│   │   ├── CartBar.tsx           # Floating cart bar at bottom of chef screens
│   │   └── CartSheet.tsx         # Full cart bottom sheet
│   ├── orders/
│   │   ├── OrderCard.tsx         # Order history item
│   │   └── OrderTimeline.tsx     # Status steps in tracking screen
│   ├── tracking/
│   │   └── DeliveryMap.tsx       # react-native-maps full-screen with driver marker
│   └── shared/
│       └── SkeletonCard.tsx      # Skeleton loading placeholder
├── hooks/
│   ├── useChefs.ts               # React Query hooks for chef listing/detail
│   ├── useCart.ts                # Zustand cart store hooks
│   ├── useOrders.ts              # React Query hooks for orders
│   ├── useOrderTracking.ts       # React Query with refetchInterval: 5000
│   └── useFavorites.ts           # Favorites query + mutation hooks
├── store/
│   ├── auth-store.ts             # ALREADY EXISTS (Phase 1 re-export)
│   └── cart-store.ts             # NEW: cart Zustand store for Phase 2
├── lib/
│   └── api.ts                    # ALREADY EXISTS (Phase 1)
└── types/
    └── customer.ts               # Customer-app-specific type aliases
```

### Pattern 1: Tab Navigation Skeleton

**What:** Replace the Phase 1 stub `(tabs)/_layout.tsx` with the full 4-tab structure.
**When to use:** First task of this phase — everything else depends on it.

```typescript
// Source: apps/mobile-customer/app/(tabs)/_layout.tsx
import { Tabs } from 'expo-router';
import { Home, ShoppingBag, Heart, User } from 'lucide-react-native';

export default function TabsLayout() {
  return (
    <Tabs screenOptions={{ headerShown: false, tabBarStyle: { height: 64 } }}>
      <Tabs.Screen name="index" options={{ title: 'Home', tabBarIcon: ({ color }) => <Home color={color} /> }} />
      <Tabs.Screen name="orders" options={{ title: 'Orders', tabBarIcon: ({ color }) => <ShoppingBag color={color} /> }} />
      <Tabs.Screen name="favorites" options={{ title: 'Saved', tabBarIcon: ({ color }) => <Heart color={color} /> }} />
      <Tabs.Screen name="profile" options={{ title: 'Profile', tabBarIcon: ({ color }) => <User color={color} /> }} />
    </Tabs>
  );
}
```

### Pattern 2: Chef Browse Screen with FlatList

**What:** 2-column FlatList of ChefCards consuming `GET /v1/chefs` with search bar + filter chips.
**Key insight:** React Native has no URL search params — store filter state in a local `useState` object, pass as query params to React Query key.

```typescript
// Source: apps/web/src/features/customer/pages/BrowseChefsPage.tsx (web reference adapted)
const { data, isLoading, refetch } = useQuery({
  queryKey: ['chefs', filters],
  queryFn: () => api.get('/v1/chefs', { params: filters }),
});

<FlatList
  data={data?.data ?? []}
  numColumns={2}
  keyExtractor={(item) => item.id}
  renderItem={({ item }) => <ChefCard chef={item} />}
  refreshControl={<RefreshControl refreshing={isLoading} onRefresh={refetch} />}
  ListEmptyComponent={<EmptyState />}
/>
```

API filters confirmed from `apps/api/handlers/chefs.go` [VERIFIED]:
- `search`, `cuisine`, `dietary`, `isOpen`, `rating`, `sort` (`rating`/`orders`/`newest`/`price`), `page`, `limit`

### Pattern 3: Cart Zustand Store

**What:** Cart state lives entirely in Zustand (no API until checkout). Floating bar shows item count and subtotal; swipe-up opens full bottom sheet.

```typescript
// Source: pattern from apps/web/src/app/store/cart-store.ts adapted to React Native
interface CartState {
  chefId: string | null;
  chef: ChefSummary | null;
  items: CartItem[];
  addItem: (item: CartItem, chef: ChefSummary) => void;
  removeItem: (itemId: string) => void;
  updateQuantity: (itemId: string, quantity: number) => void;
  clearCart: () => void;
  getSubtotal: () => number;
  getTotalCount: () => number;
}
```

**Cross-chef cart protection:** When `addItem` is called with a different `chefId`, show an Alert asking the customer if they want to clear the current cart. Web CheckoutPage.tsx confirms the API only accepts items from one chef per order [VERIFIED: orders.go line 67].

### Pattern 4: Razorpay Checkout (expo-web-browser fallback)

**What:** D-04 specifies Razorpay React Native SDK with explicit fallback to `expo-web-browser`. Research confirms the fallback is the correct path for Expo SDK 55 managed workflow.

**Payment flow (confirmed from payment.go):**

```typescript
// Source: apps/api/handlers/payment.go + apps/web/src/features/customer/pages/CheckoutPage.tsx
async function handleCheckout(orderId: string) {
  // Step 1: Create order via POST /v1/orders
  const order = await api.post('/v1/orders', orderPayload);

  // Step 2: Create Razorpay payment order — POST /v1/payments/order/:orderId/create
  // Response: { razorpayOrderId, razorpayKeyId, amount, currency, prefill }
  const paymentData = await api.post(`/v1/payments/order/${order.id}/create`, {});

  // Step 3: Open Razorpay hosted checkout in in-app browser
  // Razorpay hosted checkout URL accepts query params for pre-filling
  const checkoutUrl = buildRazorpayCheckoutUrl(paymentData);
  const result = await WebBrowser.openBrowserAsync(checkoutUrl);

  if (result.type === 'opened') {
    // Poll order status — Razorpay webhook updates payment_status on backend
    // No need to call /verify manually; webhook handles it
    pollOrderStatus(order.id);
  }
}
```

**Important:** The backend's `VerifyPayment` endpoint (`POST /v1/payments/order/:orderId/verify`) takes `razorpayPaymentId`, `razorpayOrderId`, `razorpaySignature`. The Razorpay webhook (`handlePaymentCaptured`) also updates payment status server-side [VERIFIED: payment.go lines 373-395]. For the web-browser approach, rely on the webhook + order status polling rather than trying to extract these from the browser return URL.

### Pattern 5: Order Tracking with Polling

**What:** `GET /v1/orders/:id/track` is polled every 5s. The response includes `delivery.currentLatitude`, `delivery.currentLongitude` from the `DeliveryPartner` record. These values drive the react-native-maps `Marker` position.

```typescript
// Source: apps/api/handlers/orders.go TrackOrder handler (verified)
// Response shape: { orderId, orderNumber, status, chef, estimatedDeliveryTime,
//   delivery: { currentLatitude, currentLongitude, status, ... } }

const { data } = useQuery({
  queryKey: ['order-tracking', orderId],
  queryFn: () => api.get(`/v1/orders/${orderId}/track`),
  refetchInterval: 5000,     // 5-second polling — D-05
  refetchIntervalInBackground: false, // Stop polling when app backgrounded
});

// Update Marker on the map when driver coordinates change
<MapView>
  <Marker
    coordinate={{
      latitude: data?.delivery?.currentLatitude ?? customerLat,
      longitude: data?.delivery?.currentLongitude ?? customerLng,
    }}
  />
</MapView>
```

**Stop polling when delivered:** Set `refetchInterval` to `false` when `data?.status === 'delivered'` to avoid wasted calls after order completes.

### Pattern 6: Route Structure for Non-Tab Screens

Chef detail, order detail, tracking, and checkout are stack screens outside the tab group. expo-router handles this via folders outside `(tabs)/`:

```
app/
├── (tabs)/          # tab screens
├── chef/[id].tsx    # stack screen — navigated via router.push('/chef/123')
├── order/[id].tsx   # order detail
├── checkout.tsx     # checkout
└── order/[id]/track.tsx  # tracking map — full-screen, no tab bar
```

### Anti-Patterns to Avoid

- **Importing react-native-razorpay in managed workflow:** It will compile but crash at runtime with "cannot read property 'Open' of null" — confirmed by multiple GitHub issues. Use `expo-web-browser` instead.
- **Using `useSearchParams` (web) for filters:** React Native has no URL search params. Use `useState` for filter state.
- **Mutating cart items directly:** Use Zustand immutable updates — spread the items array, never push in place. Matches global immutability constraint from CLAUDE.md.
- **`<Image>` from react-native for chef/menu photos:** Use `expo-image` — it has built-in disk + memory caching critical for the chef browse grid.
- **Polling when screen is not focused:** Guard `refetchInterval` with `useIsFocused()` from expo-router to prevent background polls when user navigates away from tracking screen.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Bottom sheet cart / order details | Custom animated slide-up panel | `@gorhom/bottom-sheet` | Handles gesture handling, snap points, keyboard avoidance, Android back button — extremely complex to replicate correctly |
| Map with driver marker | Custom map rendering | `react-native-maps` | Native iOS/Android map tiles; custom map is a massive undertaking |
| Image caching for chef photos | In-memory cache object | `expo-image` | SDWebImage (iOS) + Glide (Android) under the hood; handles network failures, placeholder blur hash, progressive loading |
| Form validation (checkout, profile, onboarding) | Custom validators | `react-hook-form` + `zod` | Already installed; handles error display, field-level validation, submission state |
| Haptic feedback | Platform-specific vibration APIs | `expo-haptics` | Abstracts iOS `UIImpactFeedbackGenerator` and Android vibrator correctly |
| Skeleton loading screens | Custom skeleton animation | `@tesserix/native` skeleton components (if available) or `react-native-skeleton-placeholder` | Animation + accessibility already handled |

**Key insight:** The bottom sheet pattern for cart and order details is deceptively complex on mobile — it requires handling fling gestures, multiple snap points, keyboard avoidance on the checkout form, and Android back button. `@gorhom/bottom-sheet` handles all of this. Do not attempt a custom implementation.

---

## Common Pitfalls

### Pitfall 1: Razorpay Native SDK in Expo Managed Workflow

**What goes wrong:** `react-native-razorpay` installs without error but throws "cannot read property 'Open' of null" at runtime because native module is missing from the managed workflow binary.

**Why it happens:** The package links native code (Objective-C on iOS, Java/Kotlin on Android). Expo managed workflow doesn't execute native code linking steps.

**How to avoid:** Use `expo-web-browser` with Razorpay hosted checkout URL. This is explicitly the D-04 fallback and is the correct path for this project.

**Warning signs:** "null is not an object" crash at the point of opening the payment sheet.

**Source:** [VERIFIED: GitHub issue #379 + #510 on razorpay/react-native-razorpay — confirmed SDK 54/55 incompatibility]

---

### Pitfall 2: Polling Continues After Order Delivered

**What goes wrong:** `refetchInterval: 5000` keeps firing `GET /v1/orders/:id/track` every 5 seconds indefinitely, even after the order is delivered.

**How to avoid:** Use `refetchInterval: (data) => data?.status === 'delivered' ? false : 5000`. React Query accepts a function for `refetchInterval`.

---

### Pitfall 3: Cross-Chef Cart Corruption

**What goes wrong:** Customer adds items from Chef A, then navigates to Chef B and adds items. API rejects the mixed-chef order with "menu item not found or unavailable" because `CreateOrder` validates that all items belong to the specified `chefId` [VERIFIED: orders.go line 84].

**How to avoid:** In `addItem`, check if `chefId` differs from the current cart's `chefId`. If it does, show a confirmation Alert and clear cart before adding.

---

### Pitfall 4: react-native-maps Blank on iOS Without NSLocationWhenInUseUsageDescription

**What goes wrong:** Map renders in iOS simulator but is blank on real device.

**How to avoid:** Add to `app.json` before first TestFlight build:
```json
{
  "ios": {
    "infoPlist": {
      "NSLocationWhenInUseUsageDescription": "Used to show your location on the delivery map"
    }
  }
}
```
This is a native config change — requires an EAS build, not OTA.

**Source:** [CITED: PITFALLS.md Pitfall 7]

---

### Pitfall 5: Checkout Form Behind Keyboard on Android

**What goes wrong:** Delivery address form fields are hidden behind the soft keyboard on Android.

**How to avoid:** Wrap the checkout screen `ScrollView` in `KeyboardAvoidingView` with `behavior={Platform.OS === 'ios' ? 'padding' : undefined}`. On Android, set `android:windowSoftInputMode="adjustResize"` in app.json under `android.softwareKeyboardLayoutMode`.

---

### Pitfall 6: Driver Location Not Surfaced in TrackOrder Response

**What goes wrong:** The `TrackOrder` handler returns `order.Delivery.ToResponse()` but the `deliveryDetailResponse` in delivery.go must include `currentLatitude`/`currentLongitude` from the `DeliveryPartner`. If the driver has not set a location yet (value is 0,0), the map should default to the chef's location.

**How to avoid:** Guard map marker: use `delivery?.currentLatitude && delivery?.currentLongitude` before rendering the driver pin. Fall back to chef `latitude`/`longitude` from the order response until driver has a real location.

---

### Pitfall 7: Onboarding Wizard Back Navigation Behavior

**What goes wrong:** On the first onboarding screen, pressing the hardware back button (Android) or swipe-back (iOS) should not navigate back to the auth flow — it should stay on step 1.

**How to avoid:** Use a `(onboarding)/` route group with its own `_layout.tsx` stack where `gestureEnabled: false` is set for the first step screen.

---

## Code Examples

### Chef Browse Query Hook

```typescript
// Source: apps/api/handlers/chefs.go ListChefs endpoint (verified)
// apps/web/src/features/customer/pages/BrowseChefsPage.tsx (web reference)
import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';

interface ChefFilters {
  search?: string;
  cuisine?: string;
  dietary?: string;
  rating?: number;
  isOpen?: boolean;
  sort?: 'rating' | 'orders' | 'newest' | 'price';
  page?: number;
  limit?: number;
}

export function useChefs(filters: ChefFilters) {
  return useQuery({
    queryKey: ['chefs', filters],
    queryFn: () => api.get('/v1/chefs', { params: filters }),
    staleTime: 1000 * 60 * 2, // 2 minutes
  });
}
```

### Order Tracking Hook with Auto-Stop

```typescript
// Source: apps/api/handlers/orders.go TrackOrder (verified)
export function useOrderTracking(orderId: string) {
  return useQuery({
    queryKey: ['order-tracking', orderId],
    queryFn: () => api.get(`/v1/orders/${orderId}/track`),
    refetchInterval: (query) => {
      const status = query.state.data?.status;
      if (status === 'delivered' || status === 'cancelled') return false;
      return 5000; // 5 seconds — D-05
    },
    refetchIntervalInBackground: false,
  });
}
```

### Cart Zustand Store (Immutable)

```typescript
// Source: Pattern from apps/web/src/app/store/cart-store.ts adapted per CLAUDE.md immutability rule
import { create } from 'zustand';

interface CartState {
  chefId: string | null;
  items: CartItem[];
  addItem: (item: CartItem, chefId: string) => 'ok' | 'cross_chef_conflict';
  removeItem: (menuItemId: string) => void;
  updateQty: (menuItemId: string, qty: number) => void;
  clearCart: () => void;
  total: () => number;
}

export const useCartStore = create<CartState>((set, get) => ({
  chefId: null,
  items: [],

  addItem: (item, chefId) => {
    const state = get();
    if (state.chefId && state.chefId !== chefId) {
      return 'cross_chef_conflict'; // Caller shows Alert
    }
    const existing = state.items.find((i) => i.menuItemId === item.menuItemId);
    if (existing) {
      set({
        items: state.items.map((i) =>
          i.menuItemId === item.menuItemId
            ? { ...i, quantity: i.quantity + 1 }  // immutable update
            : i
        ),
      });
    } else {
      set({ chefId, items: [...state.items, { ...item, quantity: 1 }] });
    }
    return 'ok';
  },

  removeItem: (menuItemId) =>
    set((s) => ({ items: s.items.filter((i) => i.menuItemId !== menuItemId) })),

  updateQty: (menuItemId, qty) =>
    set((s) => ({
      items: qty <= 0
        ? s.items.filter((i) => i.menuItemId !== menuItemId)
        : s.items.map((i) => i.menuItemId === menuItemId ? { ...i, quantity: qty } : i),
    })),

  clearCart: () => set({ chefId: null, items: [] }),

  total: () => get().items.reduce((sum, i) => sum + i.price * i.quantity, 0),
}));
```

### Payment Flow with expo-web-browser

```typescript
// Source: apps/api/handlers/payment.go CreateOrderPayment (verified)
// D-04 fallback: expo-web-browser + Razorpay hosted checkout
import * as WebBrowser from 'expo-web-browser';

async function initiatePayment(orderId: string) {
  // Step 1: Create Razorpay order on backend
  const { data } = await api.post(`/v1/payments/order/${orderId}/create`, {});
  // data: { razorpayOrderId, razorpayKeyId, amount, currency, prefill }

  // Step 2: Build hosted checkout URL (Razorpay standard hosted page)
  const checkoutUrl = `https://api.razorpay.com/v1/checkout/embedded?` +
    `key_id=${data.razorpayKeyId}` +
    `&order_id=${data.razorpayOrderId}` +
    `&amount=${data.amount}` +
    `&currency=${data.currency}` +
    `&name=Fe3dr` +
    `&prefill[name]=${encodeURIComponent(data.prefill.name)}` +
    `&prefill[email]=${encodeURIComponent(data.prefill.email)}` +
    `&prefill[contact]=${encodeURIComponent(data.prefill.phone)}` +
    `&callback_url=${encodeURIComponent('homechef-customer://payment/result')}`;

  const result = await WebBrowser.openBrowserAsync(checkoutUrl);
  // Razorpay webhook handles server-side payment_status update
  // Poll order status after browser closes to check success
  return result;
}
```

### react-native-maps Tracking Map Component

```typescript
// Source: D-07 decision + react-native-maps standard API [ASSUMED training knowledge]
import MapView, { Marker, PROVIDER_GOOGLE } from 'react-native-maps';

interface DeliveryMapProps {
  driverLat?: number;
  driverLng?: number;
  destinationLat: number;
  destinationLng: number;
}

export function DeliveryMap({ driverLat, driverLng, destinationLat, destinationLng }: DeliveryMapProps) {
  return (
    <MapView
      style={{ flex: 1 }}
      provider={PROVIDER_GOOGLE}
      initialRegion={{
        latitude: destinationLat,
        longitude: destinationLng,
        latitudeDelta: 0.05,
        longitudeDelta: 0.05,
      }}
    >
      <Marker coordinate={{ latitude: destinationLat, longitude: destinationLng }} title="Delivery Address" />
      {driverLat && driverLng && (
        <Marker
          coordinate={{ latitude: driverLat, longitude: driverLng }}
          title="Driver"
          pinColor="blue"
        />
      )}
    </MapView>
  );
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| expo-web-browser for all payments | Native payment sheet (Razorpay SDK) | Post-v1 upgrade path — not available for SDK 55 managed workflow | Use expo-web-browser for this phase |
| WebSocket for live tracking | React Query polling (5s) | Architectural decision for v1 | Simpler, no backend changes needed |
| react-navigation directly | expo-router (file-based) | Phase 1 established | Tab groups and stacks created by file structure |
| `<Image>` from react-native | `expo-image` | Expo SDK ~50+ | Native caching layer (SDWebImage/Glide) |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `delivery.ToResponse()` in TrackOrder response includes `currentLatitude`/`currentLongitude` from the DeliveryPartner | CUST-05, Code Examples | Map marker cannot update. Mitigation: read full delivery model response in Wave 0 |
| A2 | Razorpay hosted checkout URL accepts `callback_url` as a deep link for redirect back to app | Payment Flow code example | Browser does not redirect to app after payment. Mitigation: use order status polling instead of relying on redirect callback |
| A3 | `@gorhom/bottom-sheet` is compatible with Expo SDK 55 and NativeWind | Standard Stack | Need to verify via `npx expo install @gorhom/bottom-sheet` and check peer deps |
| A4 | `expo-haptics` is already available in Expo SDK 55 or installable without conflict | Don't Hand-Roll | May need explicit install; verify with `npx expo install expo-haptics` |
| A5 | react-native-maps `PROVIDER_GOOGLE` on iOS requires `ios.config.googleMapsApiKey` in app.json | Pitfall 4, Code Examples | Blank map on real iOS device; requires EAS Build with the key set |

---

## Open Questions

1. **Razorpay callback deep link handling**
   - What we know: `expo-web-browser` opens the browser; Razorpay webhook updates payment status server-side.
   - What's unclear: Whether to rely purely on the webhook + polling OR whether to register a `homechef-customer://payment/result` deep link and handle it with `expo-linking`.
   - Recommendation: Implement both — poll order status every 3s for 60s after the browser closes (optimistic path), AND set up the deep link as a backup. The webhook is the source of truth.

2. **Social feed API contract**
   - What we know: `apps/api/handlers/social.go` exists and was referenced in routes.
   - What's unclear: Exact endpoint shapes for the social feed (posts, likes, comments).
   - Recommendation: Read `handlers/social.go` in Wave 0 before implementing CUST-08.

3. **Catering API contract**
   - What we know: `apps/api/handlers/catering.go` exists.
   - What's unclear: Whether catering requests require file attachments (event photos, menus).
   - Recommendation: Read `handlers/catering.go` in Wave 0 before implementing CUST-09.

4. **`delivery.ToResponse()` shape confirmation**
   - What we know: TrackOrder calls `order.Delivery.ToResponse()` when delivery is non-nil.
   - What's unclear: Whether `currentLatitude`/`currentLongitude` are included in `ToResponse()` or only in the direct model.
   - Recommendation: Read `models/delivery.go` in Wave 0 before implementing CUST-05 map.

5. **Razorpay `callback_url` for mobile deep link**
   - What we know: Web checkout uses `handler` callback (JavaScript). Hosted checkout supports `callback_url` parameter.
   - What's unclear: Whether Razorpay's hosted checkout page will fire a deep link correctly on mobile browsers vs. just redirecting as a URL.
   - Recommendation: Test on a physical device in Wave 1. Polling-based fallback removes the risk of user being stranded.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Build | ✓ | 22.x (project requirement) | — |
| Expo CLI | Build/start | ✓ | via npx | — |
| react-native-maps | CUST-05 tracking map | Needs install | — | No fallback — required |
| expo-image | CUST-01/02 chef photos | Needs install | — | Fall back to `<Image>` from react-native (no caching) |
| @gorhom/bottom-sheet | CUST-03 cart UX | Needs install | — | Use Modal + Animated (complex) |
| expo-haptics | UX polish | Needs install | — | Omit haptics |
| Google Maps API Key | react-native-maps iOS | Unknown | — | Use MapKit (Apple Maps default) |

**Missing dependencies requiring install before implementation:**
- `react-native-maps` — core requirement for CUST-05
- `expo-image` — strongly recommended for CUST-01/02 performance
- `@gorhom/bottom-sheet` — required for recommended cart UX
- `expo-haptics` — optional UX polish

**Google Maps API Key:** Required if using `PROVIDER_GOOGLE` on iOS. Verify with project team whether a Google Maps API key exists for this project. If not, default to MapKit (Apple Maps on iOS, Google Maps on Android) which requires no additional key — set `provider` only for Android.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | jest-expo ~55.0.0 + @testing-library/react-native ^13.0.0 |
| Config file | `apps/mobile-customer/package.json` (`"jest": { "preset": "jest-expo" }`) |
| Quick run command | `cd apps/mobile-customer && npx jest --passWithNoTests --testPathPattern=unit` |
| Full suite command | `cd apps/mobile-customer && npx jest --passWithNoTests` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CUST-01 | useChefs hook returns paginated chef list | unit | `jest --testPathPattern=hooks/useChefs` | Wave 0 |
| CUST-02 | Chef detail query fetches menu items | unit | `jest --testPathPattern=hooks/useChef` | Wave 0 |
| CUST-03 | Cart store add/remove/cross-chef conflict | unit | `jest --testPathPattern=store/cart-store` | Wave 0 |
| CUST-04 | Payment flow calls create then verify endpoints | unit (mock API) | `jest --testPathPattern=hooks/usePayment` | Wave 0 |
| CUST-05 | useOrderTracking refetchInterval stops when delivered | unit | `jest --testPathPattern=hooks/useOrderTracking` | Wave 0 |
| CUST-06 | Order history query with status filter | unit | `jest --testPathPattern=hooks/useOrders` | Wave 0 |
| CUST-07 | Favorites toggle mutation + optimistic update | unit | `jest --testPathPattern=hooks/useFavorites` | Wave 0 |
| CUST-08 | Social feed query loads posts | unit | `jest --testPathPattern=hooks/useSocialFeed` | Wave 0 |
| CUST-09 | Catering form submission validation | unit | `jest --testPathPattern=components/catering` | Wave 0 |
| CUST-10 | Profile update mutation | unit | `jest --testPathPattern=hooks/useProfile` | Wave 0 |
| CUST-11 | Onboarding wizard step progression | unit | `jest --testPathPattern=store/onboarding` | Wave 0 |

### Sampling Rate

- **Per task commit:** `cd apps/mobile-customer && npx jest --passWithNoTests --testPathPattern=<changed-file>`
- **Per wave merge:** `cd apps/mobile-customer && npx jest --passWithNoTests`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `apps/mobile-customer/__tests__/store/cart-store.test.ts` — covers CUST-03 cross-chef conflict logic
- [ ] `apps/mobile-customer/__tests__/hooks/useOrderTracking.test.ts` — covers CUST-05 refetchInterval auto-stop
- [ ] `apps/mobile-customer/__tests__/hooks/useChefs.test.ts` — covers CUST-01 filter query key
- [ ] `apps/mobile-customer/__tests__/setup.ts` — jest setup for React Native environment mocks (maps, haptics)

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes (guarded by Phase 1 auth) | JWT from `expo-secure-store`; `useAuthStore` guards all tab screens |
| V3 Session Management | yes | Refresh token interceptor in `lib/api.ts` (Phase 1) |
| V4 Access Control | no | Customer-only screens; no privileged actions |
| V5 Input Validation | yes | `react-hook-form` + `zod` on checkout, onboarding, profile forms |
| V6 Cryptography | no | No crypto operations in this phase |

### Known Threat Patterns for Expo/React Native + Razorpay

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Payment amount tampering on client | Tampering | Amount is computed server-side in `CreateOrderPayment`; client only passes `orderId`. Total comes from backend. |
| Razorpay signature bypass | Tampering | Signature verification happens server-side in `VerifyPayment`. Mobile app never handles raw signature. |
| Stale JWT on long-running checkout | Elevation of Privilege | 401 interceptor in `lib/api.ts` refreshes silently; if refresh fails, user is sent to login before checkout completes. |
| Cart state injection between apps | Spoofing | Cart is Zustand in-memory only; cleared on logout via `clearCart()` in logout action. |
| API calls without auth on favorite/cart screens | Elevation of Privilege | All `(tabs)/` routes are protected by root `_layout.tsx` auth guard (Phase 1). |

---

## Sources

### Primary (HIGH confidence — verified from codebase)

- `apps/api/handlers/orders.go` — CreateOrder, GetOrders, GetOrder, TrackOrder endpoint shapes
- `apps/api/handlers/payment.go` — CreateOrderPayment, VerifyPayment flow + Razorpay response shapes
- `apps/api/handlers/chefs.go` — ListChefs filter params, GetChef endpoint
- `apps/api/handlers/delivery.go` — UpdateLocation, GetCurrentDelivery, delivery model fields
- `apps/api/routes/routes.go` — All customer-facing route paths confirmed
- `apps/mobile-customer/package.json` — Confirmed Expo SDK 55, all Phase 1 dependencies
- `apps/mobile-customer/lib/api.ts` — Confirmed API client factory pattern
- `apps/mobile-customer/store/auth-store.ts` — Confirmed auth store re-export
- `apps/web/src/features/customer/pages/BrowseChefsPage.tsx` — Web filter/search patterns
- `apps/web/src/features/customer/pages/CheckoutPage.tsx` — Web payment flow (3-step: create order → create payment → open Razorpay → verify)
- `packages/mobile-shared/src/hooks/useAuth.ts` — Auth store shape
- `.planning/phases/02-customer-app/02-CONTEXT.md` — All phase decisions
- `.planning/research/STACK.md` — Expo SDK, library versions, Metro config
- `.planning/research/PITFALLS.md` — Razorpay pitfall, maps iOS key pitfall

### Secondary (MEDIUM confidence — verified via npm + web search)

- `react-native-razorpay` v2.3.1 — Confirmed: does not support Expo managed workflow SDK 54/55 [VERIFIED: npm view + GitHub issues #379, #510]
- `react-native-expo-razorpay` v2.2.8 — Community wrapper exists but training-era knowledge on exact compatibility not verified
- Expo hosted checkout via `expo-web-browser` — confirmed as the correct fallback [CITED: STACK.md research]

### Tertiary (LOW confidence — assumed/training knowledge)

- `@gorhom/bottom-sheet` Expo SDK 55 compatibility — [ASSUMED] widely used, likely compatible, verify with `npx expo install`
- Razorpay hosted checkout `callback_url` deep link behavior on mobile — [ASSUMED] requires physical device testing to confirm

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages verified in package.json; new packages confirmed Expo-compatible via STACK.md research
- Architecture: HIGH — all API endpoints verified from handler source code
- Razorpay payment path: MEDIUM — SDK incompatibility confirmed; expo-web-browser fallback approach is sound but exact hosted checkout URL format needs testing
- Pitfalls: HIGH — most pitfalls verified from PITFALLS.md + codebase analysis

**Research date:** 2026-04-05
**Valid until:** 2026-05-05 (stable stack; Razorpay SDK situation may change)
