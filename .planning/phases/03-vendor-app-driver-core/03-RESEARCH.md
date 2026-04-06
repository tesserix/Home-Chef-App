# Phase 3: Vendor App + Driver Core - Research

**Researched:** 2026-04-06
**Domain:** React Native (Expo) — Vendor Chef App + Delivery Driver App
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Vendor Live Orders**
- D-01: Order queue UX — card stack (full-width cards) with prominent Accept/Reject buttons. Cards show customer name, items, total, elapsed time.
- D-02: New order alert — polling every 10s via React Query with in-app sound + haptic feedback when new order count increases.
- D-03: Accept/Reject UX — instant action with 3-second undo snackbar (optimistic update).

**Vendor Menu Management**
- D-04: Menu edit UX — separate edit screen for full edits, inline availability toggle on menu list cards.
- D-05: Food photo capture — `expo-image-picker` with camera and library options; upload to GCS via `POST /chef/menu/items/:itemId/images`.
- D-06: Quick availability toggle — switch directly on menu list card.

**Vendor Onboarding**
- D-07: 6-step wizard with progress indicator (personal info → kitchen details → operations → documents → policies → review).
- D-08: Document upload — `expo-image-picker` (camera + library); PDF via `expo-document-picker`. Required: ID proof, FSSAI license.

**Driver Delivery Workflow**
- D-09: Available deliveries — list view with distance + payout info, one-tap accept.
- D-10: Active delivery screen — current step (Pickup/Dropoff) prominently displayed with address card and large Navigate button.
- D-11: Navigate button — `Linking.openURL` with platform-native default (iOS: `maps://`, Android: `comgooglemaps://` with geo: fallback).
- D-12: Status update UX — swipe-to-confirm (slide-to-confirm) for status transitions to prevent accidental taps while driving.

**Shared Patterns (Carried from Phase 1 & 2)**
- D-13: Both apps use `@tesserix/native` components + NativeWind styling.
- D-14: Both apps use `packages/mobile-shared/` for API client, auth, storage.
- D-15: Both apps use Zustand + React Query for state.
- D-16: Both apps use expo-router with route groups: `(auth)/`, `(onboarding)/`, `(tabs)/`.
- D-17: Tab navigation — Vendor: Dashboard/Orders/Menu/More; Driver: Dashboard/Available/Active/More.

### Claude's Discretion
- Almost everything — user deferred all specific UX decisions.
- Tab structures, screen layouts, interaction patterns all to be determined following Phase 1-2 patterns.

### Deferred Ideas (OUT OF SCOPE)
- Background GPS and push notifications — Phase 4.
- Map pin view for available deliveries — Phase 4+ enhancement.
- Chef availability home screen widget — v2 (DIFF-09).
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| VEND-01 | Chef can view dashboard with live overview (today's orders, earnings, rating) | `GET /api/v1/chef/dashboard` confirmed — returns todayOrders, todayEarnings, rating, acceptingOrders |
| VEND-02 | Chef can manage menu items (list, create, edit, delete) | Full CRUD at `/api/v1/chef/menu/items` with categories; requires `RequireChef` middleware |
| VEND-03 | Chef can take food photos with device camera when adding menu items | `expo-image-picker` (already in Phase 2 customer app); upload to `POST /chef/menu/items/:itemId/images` |
| VEND-04 | Chef can view and accept/reject live incoming orders | `GET /chef/orders?status=pending`, `PUT /chef/orders/:orderId/status` with accept/reject body |
| VEND-05 | Chef can view order history | `GET /chef/orders` with pagination and status filter |
| VEND-06 | Chef can view earnings and payout details | `GET /chef/payout`, `GET /chef/subscription/earnings` |
| VEND-07 | Chef can view analytics (sales trends, popular items) | `GET /chef/analytics` with period query param |
| VEND-08 | Chef can view and respond to customer reviews | `GET /chef/reviews`, `POST /chef/reviews/:reviewId/reply` |
| VEND-09 | Chef can manage profile and kitchen setup | `GET /chef/profile`, `PUT /chef/profile` with kitchen photos |
| VEND-10 | Chef can manage settings and notification preferences | `GET /chef/settings`, `PUT /chef/settings` |
| VEND-11 | New chef completes onboarding flow | `GET /chef/onboarding/status`, `POST /chef/onboarding`, `POST /chef/documents` (6-step wizard) |
| DRIV-01 | Driver can view dashboard with stats (today/week/month deliveries, earnings) | `GET /api/v1/delivery/stats` — today/week/month deliveries + earnings confirmed |
| DRIV-02 | Driver can browse and accept available deliveries | `GET /delivery/available`, `POST /delivery/:id/accept` — requires isOnline + isVerified |
| DRIV-03 | Driver can view active delivery with pickup/dropoff details | `GET /delivery/current` — returns active delivery with order, chef, coordinates |
| DRIV-04 | Driver can navigate to pickup/dropoff via native maps app | `Linking.openURL` with platform-native URL scheme; no API call needed |
| DRIV-05 | Driver can update delivery status (picked up, in transit, delivered) | `PUT /delivery/:id/status` with valid status transition enum |
| DRIV-06 | Driver can view delivery history | `GET /delivery/orders` with pagination |
| DRIV-07 | Driver can view earnings | `GET /delivery/earnings` (confirmed endpoint in routes) |
| DRIV-08 | Driver can manage fleet (overview, partners) | `GET /delivery/staff/fleet/overview`, `GET /delivery/staff/fleet/partners` — requires staff permissions |
| DRIV-09 | Driver can manage staff | `GET /delivery/staff`, staff invite/manage endpoints — requires `SPViewStaff` / `SPInviteStaff` permissions |
| DRIV-10 | Driver can manage profile and settings | `GET /delivery/profile`, `PUT /delivery/profile`, `PUT /delivery/online` |
| DRIV-11 | New driver completes onboarding flow | 6-step: `/driver/onboarding/personal` → `/vehicle` → documents → `/payout` → subscription plan → `/submit` |
</phase_requirements>

---

## Summary

Phase 3 delivers two complete mobile apps — the Vendor (Chef) app and the Delivery Driver app — built entirely on patterns established in Phases 1 and 2. No new technology is introduced: both apps use the existing Expo SDK 55 scaffold, `@tesserix/native` + NativeWind styling, expo-router file routing, Zustand + React Query state management, and the `packages/mobile-shared/` API client.

The Go API already has all required endpoints for both apps. The chef endpoints require the `RequireChef` middleware (user must have role `chef`), which is granted during onboarding. Driver endpoints require `RequireDelivery`. Both onboarding flows use auth-only routes that allow role promotion during the wizard.

The largest complexity in this phase is the **delivery status machine**: the Go API enforces strict status transitions (Assigned → AtPickup → PickedUp → InTransit → AtDropoff → Delivered). The driver app's swipe-to-confirm UX must only present valid next states. A secondary complexity is the **fleet/staff management** screens for drivers — these require staff role permissions (`SPViewFleet`, `SPViewDeliveryPartners`, `SPViewStaff`, `SPInviteStaff`) which not all delivery partners will have. The screens must gracefully handle the 403 case.

**Primary recommendation:** Split work into 4 parallel plans — Vendor Core (onboarding + dashboard + orders), Vendor Menu + Extras (menu management, analytics, reviews, earnings, settings, profile), Driver Core (onboarding + dashboard + deliveries + status workflow), Driver Fleet + Extras (fleet, staff, earnings, history). Vendor and driver work streams share no code beyond `packages/mobile-shared/` and can execute concurrently.

---

## Standard Stack

### Core (both apps — already installed in scaffold)

| Library | Version | Purpose | Source |
|---------|---------|---------|--------|
| expo | ~55.0.11 | SDK base | [VERIFIED: package.json] |
| expo-router | ~55.0.10 | File-based routing | [VERIFIED: package.json] |
| react-native | 0.83.4 | RN runtime | [VERIFIED: package.json] |
| nativewind | 5.0.0-preview.3 | Tailwind for RN | [VERIFIED: package.json] |
| @tanstack/react-query | ^5.83.0 | Server state + polling | [VERIFIED: package.json] |
| zustand | ^5.0.2 | Client state | [VERIFIED: package.json] |
| react-hook-form + zod | ^7.56.4 + ^3.25.33 | Form validation | [VERIFIED: package.json] |
| axios | ^1.13.0 | HTTP client | [VERIFIED: package.json] |
| @homechef/mobile-shared | workspace:* | API client, auth hooks, screens | [VERIFIED: package.json] |
| @tesserix/native | ^1.0.0 | Design system | [VERIFIED: package.json] |
| react-native-reanimated | ~3.17.4 | Animations (slide-to-confirm) | [VERIFIED: package.json] |
| react-native-gesture-handler | ~2.24.0 | Gesture recognition | [VERIFIED: package.json] |

### Must-Add Dependencies (not yet in vendor/delivery apps)

These are in the customer app but missing from `apps/mobile-vendor/package.json` and `apps/mobile-delivery/package.json`:

| Library | Version | Purpose | Why Needed |
|---------|---------|---------|------------|
| @gorhom/bottom-sheet | ^5.2.8 | Bottom sheets for order detail expansion, status updates | D-03 undo snackbar, D-12 status panel |
| expo-haptics | ~55.0.11 | Haptic feedback on accept/reject, delivery complete | D-02, D-12 |
| expo-image | ~55.0.8 | Optimized image rendering for menu photos | VEND-02, VEND-03 |
| lucide-react-native | ^0.475.0 | Tab bar icons and UI icons | D-17 tab icons |
| expo-image-picker | (already in expo SDK) | Camera capture for menu photos and documents | VEND-03, VEND-11, DRIV-11 |
| expo-document-picker | (expo SDK) | PDF document upload for onboarding | VEND-11, DRIV-11 |

**Note on `react-native-maps`:** NOT needed for Phase 3. Maps are deferred to Phase 4 (background GPS). Driver navigation uses `Linking.openURL` to hand off to native maps app — no in-app map component.

**Installation (add to both vendor and delivery apps):**
```bash
npx expo install @gorhom/bottom-sheet expo-haptics expo-image lucide-react-native expo-document-picker
```
`expo-image-picker` is already part of the expo SDK and available without explicit install in managed workflow.

---

## Architecture Patterns

### Recommended Project Structure

#### Vendor App (`apps/mobile-vendor/`)
```
app/
├── _layout.tsx                    # Root: auth gate + onboarding gate
├── (auth)/                        # Login, Register, Forgot Password (Phase 1)
├── (onboarding)/                  # 6-step vendor onboarding wizard
│   ├── _layout.tsx                # Stack with gestureEnabled: false on step 1
│   ├── personal-info.tsx          # Step 1: Full name, phone, email
│   ├── kitchen-details.tsx        # Step 2: Business name, cuisine, description
│   ├── operations.tsx             # Step 3: Hours, prep time, service radius
│   ├── documents.tsx              # Step 4: ID proof, FSSAI upload
│   ├── policies.tsx               # Step 5: Terms, cancellation policy
│   └── review.tsx                 # Step 6: Summary + submit
├── (tabs)/
│   ├── _layout.tsx                # Tab bar: Dashboard | Orders | Menu | More
│   ├── index.tsx                  # Dashboard tab (VEND-01)
│   ├── orders.tsx                 # Orders tab — live queue + history toggle (VEND-04, VEND-05)
│   ├── menu.tsx                   # Menu tab — item list with availability toggle (VEND-02)
│   └── more.tsx                   # More tab: Profile, Earnings, Analytics, Reviews, Settings
├── menu/
│   ├── new.tsx                    # Create menu item with camera (VEND-03)
│   └── [itemId]/edit.tsx          # Edit menu item
└── review/
    └── [reviewId].tsx             # Review detail with reply (VEND-08)
```

#### Delivery App (`apps/mobile-delivery/`)
```
app/
├── _layout.tsx                    # Root: auth gate + onboarding gate
├── (auth)/                        # Login, Register (Phase 1)
├── (onboarding)/                  # 6-step driver onboarding wizard
│   ├── _layout.tsx
│   ├── personal.tsx               # Step 1: City, emergency contact, vehicle type
│   ├── vehicle.tsx                # Step 2: Vehicle details, license number
│   ├── documents.tsx              # Step 3: ID, license photos
│   ├── payout.tsx                 # Step 4: Bank/UPI details
│   ├── subscription.tsx           # Step 5: Subscription plan selection
│   └── review.tsx                 # Step 6: Summary + terms + submit
├── (tabs)/
│   ├── _layout.tsx                # Tab bar: Dashboard | Available | Active | More
│   ├── index.tsx                  # Dashboard (DRIV-01)
│   ├── available.tsx              # Available deliveries list (DRIV-02)
│   ├── active.tsx                 # Active delivery screen (DRIV-03, DRIV-04, DRIV-05)
│   └── more.tsx                   # More: History, Earnings, Fleet, Staff, Profile, Settings
├── delivery/
│   └── [id].tsx                   # Delivery history detail (DRIV-06)
└── fleet/
    ├── index.tsx                  # Fleet overview (DRIV-08)
    └── partner/[id].tsx           # Partner detail (DRIV-08)
```

### Pattern 1: Onboarding Gate in Root Layout (Phase 1/2 established)

Both apps must gate on onboarding completion just as the customer app does. The vendor app checks `GET /chef/onboarding/status` (returns `completed: bool`), the driver app checks `GET /driver/onboarding/status` (returns `onboardingComplete: bool`).

```typescript
// Source: apps/mobile-customer/app/_layout.tsx (established pattern)
export default function RootLayout() {
  const { isAuthenticated, isLoading, onboardingComplete, hydrateFromStorage } = useAuthStore();

  useEffect(() => {
    if (!isLoading) {
      if (!isAuthenticated) {
        router.replace('/(auth)/login');
      } else if (!onboardingComplete) {
        router.replace('/(onboarding)/personal-info'); // or step 1
      } else {
        router.replace('/(tabs)');
      }
    }
  }, [isAuthenticated, isLoading, onboardingComplete]);
  // ...
}
```

**Key difference from customer app:** `onboardingComplete` for vendor/driver is fetched from API on first authenticated load (not stored locally), since admin approval affects status post-submission. After submission, `onboardingComplete: true` routes to a "pending review" holding screen rather than directly to tabs.

### Pattern 2: React Query Polling for Live Orders (VEND-04)

10-second polling for vendor order queue. Use `refetchInterval` in React Query. Track order count in ref to detect new orders and trigger haptic/sound.

```typescript
// Vendor order queue polling pattern
const previousCountRef = useRef(0);

const { data } = useQuery({
  queryKey: ['chef', 'orders', 'pending'],
  queryFn: () => api.get('/chef/orders?status=pending'),
  refetchInterval: 10_000,        // 10s polling per D-02
  refetchIntervalInBackground: false, // stop polling when app is backgrounded (Phase 3)
});

useEffect(() => {
  const count = data?.data?.length ?? 0;
  if (count > previousCountRef.current) {
    Haptics.notificationAsync(Haptics.NotificationFeedbackType.Warning);
    // play sound if available
  }
  previousCountRef.current = count;
}, [data]);
```

### Pattern 3: Optimistic Accept/Reject with Undo (VEND-04, D-03)

React Query mutation with `onMutate` optimistic update + cancel timer for undo.

```typescript
// Source: React Query optimistic updates pattern [ASSUMED — standard React Query pattern]
const mutation = useMutation({
  mutationFn: ({ orderId, action }: { orderId: string; action: 'accepted' | 'rejected' }) =>
    api.put(`/chef/orders/${orderId}/status`, { status: action }),
  onMutate: async ({ orderId, action }) => {
    await queryClient.cancelQueries({ queryKey: ['chef', 'orders', 'pending'] });
    const previous = queryClient.getQueryData(['chef', 'orders', 'pending']);
    queryClient.setQueryData(['chef', 'orders', 'pending'], (old: Order[]) =>
      old.filter((o) => o.id !== orderId)
    );
    return { previous };
  },
  onError: (_, __, context) => {
    queryClient.setQueryData(['chef', 'orders', 'pending'], context?.previous);
  },
  onSettled: () => queryClient.invalidateQueries({ queryKey: ['chef', 'orders'] }),
});

// 3-second undo timer
const [pendingUndo, setPendingUndo] = useState<string | null>(null);
const undoTimer = useRef<NodeJS.Timeout | null>(null);

function handleAccept(orderId: string) {
  setPendingUndo(orderId);
  undoTimer.current = setTimeout(() => {
    mutation.mutate({ orderId, action: 'accepted' });
    setPendingUndo(null);
  }, 3000);
}

function handleUndo() {
  clearTimeout(undoTimer.current!);
  queryClient.invalidateQueries({ queryKey: ['chef', 'orders', 'pending'] });
  setPendingUndo(null);
}
```

### Pattern 4: Slide-to-Confirm Status Updates (DRIV-05, D-12)

Use `react-native-reanimated` + `react-native-gesture-handler` (both already installed) for swipe-to-confirm component. This prevents accidental taps while driving.

```typescript
// Source: react-native-reanimated PanGestureHandler pattern [ASSUMED — standard RN pattern]
// SlideToConfirm component: thumb slides rightward, triggers callback at threshold (>80% of track width)
// On release below threshold: spring-returns to start
// On reach threshold: callback fires, thumb locks, haptic feedback
```

### Pattern 5: Native Maps Navigation Handoff (DRIV-04, D-11)

```typescript
// Source: Expo Linking docs [CITED: docs.expo.dev/versions/latest/sdk/linking/]
import { Linking, Platform } from 'react-native';

function openNavigation(lat: number, lng: number, label: string) {
  const encoded = encodeURIComponent(label);
  const url = Platform.select({
    ios: `maps://?q=${encoded}&ll=${lat},${lng}`,
    android: `geo:${lat},${lng}?q=${lat},${lng}(${encoded})`,
  })!;

  Linking.canOpenURL(url).then((supported) => {
    if (supported) {
      Linking.openURL(url);
    } else {
      // Fallback: Google Maps web URL works universally
      Linking.openURL(`https://maps.google.com/?q=${lat},${lng}`);
    }
  });
}
```

### Pattern 6: Multi-Step Onboarding State (VEND-11, DRIV-11)

Wizard state managed with Zustand store (one store per app). The API saves each step individually (`POST /chef/onboarding`, `POST /driver/onboarding/personal`, etc.), so partial progress persists across app restarts.

```typescript
// Vendor onboarding Zustand store pattern
interface VendorOnboardingState {
  currentStep: 1 | 2 | 3 | 4 | 5 | 6;
  personalInfo: { fullName: string; phone: string; email: string };
  kitchenDetails: { businessName: string; cuisines: string[]; description: string };
  operations: { operatingHours: Record<string, DayHours>; prepTime: string; serviceRadius: number };
  documents: { idProofUrl: string | null; fssaiUrl: string | null };
  policies: { acceptedTerms: boolean };
  // actions
  setStep: (step: number) => void;
  updatePersonalInfo: (data: Partial<VendorOnboardingState['personalInfo']>) => void;
  // ...
}
```

### Pattern 7: File Upload with expo-image-picker (VEND-03, VEND-11, DRIV-11)

```typescript
// Source: customer app Phase 2 established pattern [VERIFIED: apps/mobile-customer uses expo-image-picker]
import * as ImagePicker from 'expo-image-picker';

async function pickAndUploadMenuPhoto(itemId: string) {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ['images'],
    quality: 0.8,
    allowsEditing: true,
    aspect: [4, 3],
  });

  if (!result.canceled && result.assets[0]) {
    const asset = result.assets[0];
    const formData = new FormData();
    formData.append('file', {
      uri: asset.uri,
      name: `menu-photo.jpg`,
      type: 'image/jpeg',
    } as unknown as Blob);

    await api.post(`/chef/menu/items/${itemId}/images`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  }
}

async function pickDocument() {
  const result = await DocumentPicker.getDocumentAsync({
    type: ['image/*', 'application/pdf'],
    copyToCacheDirectory: true,
  });
  // upload as above with POST /chef/documents?type=id_proof etc.
}
```

### Anti-Patterns to Avoid

- **Polling in background:** Do NOT set `refetchIntervalInBackground: true` in Phase 3 — background location/push are Phase 4. Keep polling foreground-only.
- **Status transitions client-side only:** NEVER skip the API call for delivery status updates and show only local state. The API enforces valid transitions and updates order status atomically.
- **Direct image upload from URL:** The API's `POST /chef/menu/items/:itemId/images` expects `multipart/form-data` with a `file` field, not a JSON body with a URL. Always upload the binary.
- **Onboarding gate based solely on JWT role:** The `role = chef` check is insufficient — a rejected chef still has role=chef but `completed: false`. Always check `/chef/onboarding/status` endpoint.
- **Assuming driver can always see fleet/staff:** These endpoints require specific staff permissions. A typical delivery partner (not fleet manager) will get 403. Screens must show "not authorized" state gracefully, not crash.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Swipe-to-confirm gesture | Custom PanResponder from scratch | `react-native-reanimated` + `react-native-gesture-handler` | Both already installed; PanResponder lacks velocity tracking needed for smooth UX |
| Bottom sheet for order detail | Custom Modal with animation | `@gorhom/bottom-sheet` v5 (Phase 2 pattern) | Handles keyboard avoidance, snap points, portal rendering — complex to replicate |
| Form validation | Manual field-by-field checks | React Hook Form + Zod (already installed) | Established in Phase 1-2; all forms use this pattern |
| Haptic feedback | Native module bridge | `expo-haptics` | Already in customer app; one-line API |
| Document upload encoding | Manual base64 + upload | `FormData` with `expo-image-picker` / `expo-document-picker` | Established in Phase 2 patterns |
| Distance calculation for available deliveries | Haversine from scratch | API already returns `distance` field | `GetAvailableDeliveries` handler computes Haversine server-side |
| Tab icon set | Custom SVGs | `lucide-react-native` (Phase 2 pattern) | Same icons as customer app; already in use |

**Key insight:** This phase is pure screen implementation — the API, design system, state management, and shared infrastructure are all in place. Every "new" problem has an established solution in the Phase 2 customer app.

---

## API Endpoint Inventory

### Vendor (Chef) App Endpoints

All under `/api/v1/` with `Authorization: Bearer <jwt>` header. Routes requiring chef role use `middleware.RequireChef()`.

| Endpoint | Method | Auth | Handler | Purpose |
|----------|--------|------|---------|---------|
| `/chef/onboarding/status` | GET | Auth | `UploadHandler.GetOnboardingStatus` | Returns `{status, completed, step, chefId, profile}` |
| `/chef/onboarding` | POST | Auth | `UploadHandler.Onboarding` | Submit/update onboarding form (all steps in one JSON) |
| `/chef/documents` | POST | Auth | `UploadHandler.UploadDocument` | Upload ID/FSSAI document (`multipart/form-data`, field `type`) |
| `/chef/documents` | GET | Auth | `UploadHandler.GetDocuments` | List uploaded documents |
| `/chef/profile-image` | POST | Auth | `UploadHandler.UploadProfileImage` | Upload chef profile photo |
| `/chef/banner-image` | POST | Auth | `UploadHandler.UploadBannerImage` | Upload banner image |
| `/chef/kitchen-photos` | POST | Auth | `UploadHandler.UploadKitchenPhoto` | Add kitchen photo (max 5) |
| `/chef/dashboard` | GET | Chef | `ChefHandler.GetChefDashboard` | Today stats, recent orders, rating |
| `/chef/profile` | GET | Chef | `ChefHandler.GetChefProfile` | Full profile |
| `/chef/profile` | PUT | Chef | `ChefHandler.UpdateChefProfile` | Update profile |
| `/chef/orders` | GET | Chef | `ChefHandler.GetChefOrders` | Orders list with `status` + `page` params |
| `/chef/orders/:orderId/status` | PUT | Chef | `ChefHandler.UpdateOrderStatus` | Accept/reject: `{status: "accepted"|"rejected", reason?}` |
| `/chef/reviews` | GET | Chef | `ChefHandler.GetChefReviewsForDashboard` | All reviews |
| `/chef/reviews/:reviewId/reply` | POST | Chef | `ChefHandler.ReplyToReview` | `{reply: "..."}` |
| `/chef/settings` | GET | Chef | `ChefHandler.GetChefSettings` | Notification prefs, availability |
| `/chef/settings` | PUT | Chef | `ChefHandler.UpdateChefSettings` | Update settings |
| `/chef/analytics` | GET | Chef | `ChefHandler.GetChefAnalytics` | Sales trends; `period` query param |
| `/chef/payout` | GET | Chef | `ChefHandler.GetPayoutDetails` | Bank details (masked) + payout summary |
| `/chef/menu` | GET | Auth | `MenuHandler.GetChefMenuItems` | Menu items list with images |
| `/chef/menu/categories` | GET/POST | Auth | `MenuHandler.GetCategories / CreateCategory` | Category management |
| `/chef/menu/items` | POST | Auth | `MenuHandler.CreateMenuItem` | Create item (triggers approval request) |
| `/chef/menu/items/:itemId` | GET/PUT/DELETE | Auth | `MenuHandler.*` | Item CRUD; price changes trigger approval |
| `/chef/menu/items/:itemId/images` | POST | Auth | `MenuHandler.UploadMenuItemImage` | Upload food photo (`multipart`, max 5) |

**Important:** Menu routes (`/chef/menu/*`) use `AuthMiddleware` only (not `RequireChef`) — accessible during onboarding before approval. Chef dashboard routes (`/chef/dashboard`, `/chef/orders`, etc.) use `RequireChef`.

### Driver App Endpoints

| Endpoint | Method | Auth | Purpose |
|----------|--------|------|---------|
| `/driver/onboarding/status` | GET | Auth | `{step, status, onboardingComplete, verificationStatus, profile}` |
| `/driver/onboarding/personal` | POST | Auth | Step 1: `{city, emergencyContact, emergencyPhone, vehicleType, dateOfBirth?}` |
| `/driver/onboarding/vehicle` | POST | Auth | Step 2: `{vehicleType, vehicleMake, vehicleModel, vehicleYear, vehicleColor, vehicleNumber, licenseNumber}` |
| `/driver/onboarding/documents` | POST | Auth | Step 3: Document upload via existing delivery upload endpoint |
| `/driver/onboarding/payout` | POST | Auth | Step 4: `{payoutMethod, bankAccountNumber?, bankIFSC?, upiId?}` |
| `/driver/subscription/plans` | GET | Auth | Step 5: Available subscription plans |
| `/driver/subscription/plan` | POST | Auth | Step 5: Select plan |
| `/driver/onboarding/submit` | POST | Auth | Step 6: `{termsAccepted: true}` — sets `onboardingComplete: true` |
| `/delivery/stats` | GET | Delivery | Dashboard stats (today/week/month) |
| `/delivery/profile` | GET/PUT | Delivery | Driver profile |
| `/delivery/online` | PUT | Delivery | `{isOnline: bool}` — toggle online status |
| `/delivery/available` | GET | Delivery | Available orders (requires `isOnline: true`, `isVerified: true`) |
| `/delivery/:id/accept` | POST | Delivery | Accept an available delivery |
| `/delivery/current` | GET | Delivery | Current active delivery with order + coordinates |
| `/delivery/:id/status` | PUT | Delivery | Update status — see transition table below |
| `/delivery/orders` | GET | Delivery | Delivery history with pagination |
| `/delivery/earnings` | GET | Delivery | Earnings summary |
| `/delivery/staff/fleet/overview` | GET | Delivery + `SPViewFleet` | Fleet overview |
| `/delivery/staff/fleet/partners` | GET | Delivery + `SPViewDeliveryPartners` | Partner list |
| `/delivery/staff` | GET | Delivery + `SPViewStaff` | Staff list |
| `/delivery/staff/invitations` | POST | Delivery + `SPInviteStaff` | Send staff invite |

### Delivery Status Transition Machine

The API enforces this state machine in `UpdateDeliveryStatus`:

```
assigned → at_pickup → picked_up → in_transit → at_dropoff → delivered
                ↘                       ↘               ↘
             cancelled              cancelled          cancelled
```

The mobile swipe-to-confirm UI should only present valid next states based on current `delivery.status`. Map current status to allowed actions:

| Current Status | User-facing action(s) |
|----------------|----------------------|
| `assigned` | "Arrived at Kitchen" (→ `at_pickup`) |
| `at_pickup` | "Picked Up" (→ `picked_up`) |
| `picked_up` | "In Transit" (→ `in_transit`) |
| `in_transit` | "At Dropoff" (→ `at_dropoff`) |
| `at_dropoff` | "Delivered" (→ `delivered`) |

Each transition fires `PUT /delivery/:id/status` with `{status: "<new_status>"}`.

---

## Common Pitfalls

### Pitfall 1: Chef Role Required for Dashboard but Not Menu
**What goes wrong:** API returns 403 on `/chef/dashboard` and `/chef/orders` even after onboarding form is submitted — because chef role requires admin approval (`isVerified: true`).
**Why it happens:** `/chef/menu/*` uses `AuthMiddleware` only (no role check) to allow menu creation during onboarding. `/chef/dashboard` uses `RequireChef` which checks `user.role === "chef"` AND `chef.is_verified`.
**How to avoid:** The onboarding gate screens for vendor must check `onboardingStatus` from the API and route to a "pending review" holding screen (not tabs) for `status: "pending_review"` or `status: "submitted"`. Only `status: "verified"` routes to full tabs.
**Warning signs:** 403 on dashboard API call despite user having completed onboarding form.

### Pitfall 2: Delivery Available Endpoint Returns Empty When Offline
**What goes wrong:** `GET /delivery/available` returns `{data: [], message: "Go online to see available deliveries"}` when `isOnline: false` — this is a 200 response, not an error.
**Why it happens:** Driver must explicitly toggle online via `PUT /delivery/online` before deliveries appear.
**How to avoid:** The Available tab must show an "You are offline" banner with a toggle-online button when `isOnline: false`. Check partner profile's `isOnline` field on tab mount.
**Warning signs:** Empty list with no loading error.

### Pitfall 3: Document Upload Type Field is Required
**What goes wrong:** `POST /chef/documents` returns 400 "Document type is required" even when file is attached.
**Why it happens:** The endpoint reads `type` from `c.PostForm("type")` — it must be a multipart form field, not a query param. The client must append `type` as a form field alongside the file.
**How to avoid:** `formData.append('type', 'id_proof')` before appending the file field.
**Warning signs:** 400 Bad Request on document upload despite correct file.

### Pitfall 4: Price Changes Trigger Admin Approval, Not Immediate Update
**What goes wrong:** Chef updates menu item price; the price appears to save (200 OK) but doesn't reflect immediately on customer-facing menu.
**Why it happens:** `UpdateMenuItem` creates an `ApprovalRequest` of type `pricing_change` when price changes. The item is updated in DB immediately but admin review is triggered.
**How to avoid:** Show an informational banner after price save: "Price change submitted for review." This is existing platform behavior — do not try to bypass it. The mobile app should communicate this clearly.

### Pitfall 5: Driver Onboarding Step 3 Uses Delivery Upload Endpoint, Not Driver Endpoint
**What goes wrong:** Driver onboarding document upload sends to wrong endpoint.
**Why it happens:** Route is `POST /driver/onboarding/documents` which maps to `deliveryHandler.UploadPartnerDocument` (NOT `driverHandler`). The form field is `file` and the document type determines storage.
**How to avoid:** Use `/api/v1/driver/onboarding/documents` during onboarding (auth-only, no delivery role needed). After onboarding, use `/api/v1/delivery/documents` (delivery role required).

### Pitfall 6: @gorhom/bottom-sheet and expo-router Portal Conflict
**What goes wrong:** Bottom sheet renders beneath tab bar or navigation elements.
**Why it happens:** `@gorhom/bottom-sheet` v5 uses React portals; expo-router's root layout must wrap with `GestureHandlerRootView` and `BottomSheetModalProvider` at the very top level.
**How to avoid:** Wrap the root `QueryClientProvider` with these providers in `_layout.tsx`:
```tsx
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { BottomSheetModalProvider } from '@gorhom/bottom-sheet';
// Wrap around QueryClientProvider
```
**Warning signs:** Bottom sheet appears clipped or behind navigation bar.

### Pitfall 7: NativeWind v5 Preview — className Requires Babel Plugin Active
**What goes wrong:** `className` prop on React Native components has no visible effect.
**Why it happens:** NativeWind 5.0.0-preview.3 requires `babel.config.js` to include the NativeWind preset AND the `tailwindcss/preset` in `tailwind.config.js`. Already configured in Phase 1 — verify it's not accidentally removed.
**Warning signs:** All styles using `className` silently ignored; default RN styles applied instead.

---

## Code Examples

### Vendor Tab Bar with Icons (D-17)
```typescript
// Source: apps/mobile-customer/app/(tabs)/_layout.tsx — Phase 2 established pattern [VERIFIED]
import { Tabs } from 'expo-router';
import { LayoutDashboard, ClipboardList, UtensilsCrossed, MoreHorizontal } from 'lucide-react-native';

export default function VendorTabsLayout() {
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarStyle: { height: 64, paddingBottom: 8 },
        tabBarActiveTintColor: '#FF6B35',    // brand orange from @tesserix/native tokens
        tabBarInactiveTintColor: '#9CA3AF',
      }}
    >
      <Tabs.Screen name="index" options={{ title: 'Dashboard', tabBarIcon: ({ color }) => <LayoutDashboard size={22} color={color} /> }} />
      <Tabs.Screen name="orders" options={{ title: 'Orders', tabBarIcon: ({ color }) => <ClipboardList size={22} color={color} /> }} />
      <Tabs.Screen name="menu" options={{ title: 'Menu', tabBarIcon: ({ color }) => <UtensilsCrossed size={22} color={color} /> }} />
      <Tabs.Screen name="more" options={{ title: 'More', tabBarIcon: ({ color }) => <MoreHorizontal size={22} color={color} /> }} />
    </Tabs>
  );
}
```

### Driver Tab Bar (D-17)
```typescript
// Source: Phase 2 pattern [VERIFIED]
import { Tabs } from 'expo-router';
import { LayoutDashboard, MapPin, Navigation, MoreHorizontal } from 'lucide-react-native';

export default function DriverTabsLayout() {
  return (
    <Tabs screenOptions={{ headerShown: false, tabBarActiveTintColor: '#FF6B35', tabBarInactiveTintColor: '#9CA3AF', tabBarStyle: { height: 64, paddingBottom: 8 } }}>
      <Tabs.Screen name="index" options={{ title: 'Dashboard', tabBarIcon: ({ color }) => <LayoutDashboard size={22} color={color} /> }} />
      <Tabs.Screen name="available" options={{ title: 'Available', tabBarIcon: ({ color }) => <MapPin size={22} color={color} /> }} />
      <Tabs.Screen name="active" options={{ title: 'Active', tabBarIcon: ({ color }) => <Navigation size={22} color={color} /> }} />
      <Tabs.Screen name="more" options={{ title: 'More', tabBarIcon: ({ color }) => <MoreHorizontal size={22} color={color} /> }} />
    </Tabs>
  );
}
```

### Chef Order Accept with Undo Snackbar (D-03)
```typescript
// Derived from React Query optimistic update pattern [ASSUMED — standard React Query pattern]
// Key: mutation fires after 3s unless undone; optimistic update removes card immediately
const UNDO_DELAY_MS = 3000;

function useOrderAction() {
  const queryClient = useQueryClient();
  const [pendingUndo, setPendingUndo] = useState<{ orderId: string; action: string } | null>(null);
  const timerRef = useRef<NodeJS.Timeout | null>(null);

  const mutation = useMutation({
    mutationFn: ({ orderId, action }: { orderId: string; action: 'accepted' | 'rejected' }) =>
      vendorApi.put(`/chef/orders/${orderId}/status`, { status: action }),
    onSettled: () => queryClient.invalidateQueries({ queryKey: ['chef', 'orders'] }),
  });

  function trigger(orderId: string, action: 'accepted' | 'rejected') {
    // Optimistically remove from pending queue
    queryClient.setQueryData<Order[]>(['chef', 'orders', 'pending'], (old) =>
      (old ?? []).filter((o) => o.id !== orderId)
    );
    setPendingUndo({ orderId, action });
    timerRef.current = setTimeout(() => {
      mutation.mutate({ orderId, action });
      setPendingUndo(null);
    }, UNDO_DELAY_MS);
  }

  function undo() {
    if (timerRef.current) clearTimeout(timerRef.current);
    queryClient.invalidateQueries({ queryKey: ['chef', 'orders', 'pending'] });
    setPendingUndo(null);
  }

  return { trigger, undo, pendingUndo };
}
```

### Vendor Onboarding Gate Check
```typescript
// Vendor-specific onboarding gate in _layout.tsx
// Status values from GET /chef/onboarding/status:
//   "not_started" → route to /(onboarding)/personal-info
//   "in_progress" / "submitted" / "pending_review" → route to /(onboarding)/pending-review (holding screen)
//   "rejected" / "info_requested" → route to /(onboarding)/review (re-submit)
//   "verified" → route to /(tabs)

const ONBOARDING_COMPLETED_STATUSES = ['verified'] as const;
const ONBOARDING_REVIEW_STATUSES = ['submitted', 'pending_review'] as const;
const ONBOARDING_RESTART_STATUSES = ['rejected', 'info_requested'] as const;
```

### Driver Navigation Handoff (D-11)
```typescript
// Source: Expo Linking [CITED: docs.expo.dev/versions/latest/sdk/linking/]
import { Linking, Platform } from 'react-native';

export function openNativeNavigation(lat: number, lng: number, label: string): void {
  const encoded = encodeURIComponent(label);
  const url = Platform.select({
    ios: `maps://?q=${encoded}&ll=${lat},${lng}&dirflg=d`,
    android: `geo:${lat},${lng}?q=${lat},${lng}(${encoded})`,
  })!;

  void Linking.canOpenURL(url).then((supported) => {
    const target = supported ? url : `https://maps.google.com/maps?daddr=${lat},${lng}`;
    void Linking.openURL(target);
  });
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-----------------|--------------|--------|
| WebSocket for live order updates | React Query polling (10s) | Phase 3 decision | Simpler, no WS infrastructure needed for v1 |
| Push notifications for new orders | Polling + in-app haptic | Phase 3 (push deferred to Phase 4) | Polling is Phase 3 interim; push is Phase 4 |
| In-app maps for navigation | Native maps handoff via `Linking` | Phase 3 decision | Avoids `react-native-maps` dependency until Phase 4 GPS work |

**Deprecated/outdated:**
- `getExpoPushTokenAsync()` — Do NOT use; existing backend uses raw FCM tokens via `getDevicePushTokenAsync()` (Phase 1 locked decision)

---

## Open Questions

1. **Vendor Pending Review Screen**
   - What we know: After onboarding submission (`status: "pending_review"`), `onboardingComplete: true` routes app away from onboarding wizard.
   - What's unclear: Should the holding screen be part of `(onboarding)/` or `(tabs)/` route group? If part of `(tabs)`, does the tab bar show?
   - Recommendation: Create `app/(pending)/index.tsx` as a third route group — no tab bar, shows status banner + estimated review time. Same pattern as multi-step forms that need a post-submit state.

2. **Fleet/Staff Screen Authorization for Average Drivers**
   - What we know: `GET /delivery/staff/fleet/*` requires `SPViewFleet` permission. A typical delivery partner does NOT have this staff role.
   - What's unclear: Whether the "Fleet" and "Staff" tabs/screens should be hidden entirely for non-staff drivers or shown as locked.
   - Recommendation: Conditionally render fleet/staff entries in More screen based on `GET /delivery/staff/me` response. If 404 (not a staff member), hide fleet/staff menu items entirely.

3. **Chef Dashboard "Today's Orders" Count vs Live Queue**
   - What we know: `/chef/dashboard` returns today's summary stats. `/chef/orders?status=pending` returns live queue.
   - What's unclear: Does the dashboard page also poll, or does it show static data refreshed on pull-to-refresh?
   - Recommendation: Dashboard shows static data (no polling). Only the Orders tab polls every 10s. This matches the web portal pattern.

---

## Environment Availability

Step 2.6: SKIPPED — Phase 3 is purely mobile screen implementation. No new external services, CLIs, or runtimes are introduced beyond what was established in Phases 1 and 2 (Node.js 22, pnpm, Expo CLI, existing Go API).

---

## Project Constraints (from CLAUDE.md)

Directives extracted from `./CLAUDE.md` that affect Phase 3 implementation:

| Constraint | Source | Impact on Phase 3 |
|-----------|--------|-------------------|
| No backend changes | Constraints section | All screens must work with existing Go API endpoints as-is |
| `@tesserix/native` required for brand consistency | Constraints + D-13 | All UI components use `@tesserix/native` + NativeWind className |
| TypeScript throughout | Stack | All `.tsx` files; no `.jsx`; explicit types on exports |
| Immutable state updates only | CLAUDE.md coding-style | Zustand stores use spread; no direct mutation |
| No `console.log` in production code | TypeScript hooks | Use structured logging or remove before commit |
| Functions < 50 lines | coding-style | Complex screens split into smaller components and hooks |
| Files < 800 lines | coding-style | Each screen file stays focused; extract components |
| React functional components + hooks only | React Patterns | No class components |
| React Query for server state, Zustand for client state | conventions | Confirmed by Phase 2 patterns |
| NativeWind className everywhere | D-11, D-13 | No StyleSheet.create — use className prop |
| expo-router file-based routing | D-16 | Route groups: `(auth)/`, `(onboarding)/`, `(tabs)/` |
| Single-line commit messages | global CLAUDE.md | Conventional commits: `feat: add vendor order queue screen` |

---

## Sources

### Primary (HIGH confidence)
- `apps/mobile-vendor/package.json` — Exact versions of all installed dependencies [VERIFIED: file read]
- `apps/mobile-delivery/package.json` — Delivery app dependencies [VERIFIED: file read]
- `apps/mobile-customer/package.json` — Phase 2 additions (@gorhom/bottom-sheet, expo-haptics, expo-image, lucide-react-native, react-native-maps) [VERIFIED: file read]
- `apps/api/handlers/menu.go` — Full menu CRUD API with multipart upload endpoints [VERIFIED: file read]
- `apps/api/handlers/delivery.go` — Driver stats, available, accept, status update, current delivery [VERIFIED: file read]
- `apps/api/handlers/driver_onboarding.go` — Full 6-step driver onboarding flow [VERIFIED: file read]
- `apps/api/handlers/upload.go` — Chef onboarding, document upload, kitchen photos [VERIFIED: file read]
- `apps/api/routes/routes.go` — All route registrations, auth middleware requirements [VERIFIED: file read]
- `apps/mobile-customer/app/_layout.tsx` — Onboarding gate pattern [VERIFIED: file read]
- `apps/mobile-customer/app/(tabs)/_layout.tsx` — Tab bar pattern with lucide icons [VERIFIED: file read]
- `apps/mobile-vendor/app/_layout.tsx` — Current vendor app root layout [VERIFIED: file read]
- `.planning/phases/03-vendor-app-driver-core/03-CONTEXT.md` — All locked decisions [VERIFIED: file read]

### Secondary (MEDIUM confidence)
- expo-router Stack/Tabs patterns — derived from Phase 1-2 implementations [VERIFIED in codebase]
- @gorhom/bottom-sheet v5 provider setup pattern — [ASSUMED — standard pattern, verify with docs if issues arise]
- React Query optimistic update pattern — [ASSUMED — standard React Query v5 pattern]

### Tertiary (LOW confidence — flag for validation)
- Slide-to-confirm implementation using reanimated + gesture-handler — [ASSUMED — no existing example in codebase; use community SlideToConfirm component or implement from scratch]

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Optimistic update + undo timer pattern is the correct approach for D-03 | Code Examples | Low — if React Query cache invalidation behaves differently, undo may cause flicker |
| A2 | Slide-to-confirm should use reanimated + gesture-handler (not a library) | Don't Hand-Roll | Low — a community library like `rn-slide-to-confirm` may be simpler; planner should verify if team prefers it |
| A3 | @gorhom/bottom-sheet needs `GestureHandlerRootView` + `BottomSheetModalProvider` in root layout | Pitfalls | Medium — v5 docs should be checked; wrong root setup causes rendering bugs |
| A4 | Fleet/staff screens should be hidden (not locked) for non-staff drivers | Open Questions | Low — UX choice, either approach works technically |
| A5 | `expo-document-picker` is needed for PDF uploads (not in vendor/delivery package.json) | Standard Stack | Medium — if only image documents required for this version, `expo-image-picker` alone may suffice |

**If this table is empty:** All claims in this research were verified or cited.

---

## Metadata

**Confidence breakdown:**
- API endpoints: HIGH — verified by reading all handler and route files
- Dependency versions: HIGH — read directly from package.json files
- Phase 2 patterns: HIGH — verified by reading customer app implementation
- New interaction patterns (slide-to-confirm, bottom sheet): MEDIUM — reanimated/gesture handler confirmed installed; exact implementation patterns ASSUMED
- Onboarding status flow: HIGH — read from upload.go handler logic directly

**Research date:** 2026-04-06
**Valid until:** 2026-05-06 (stable dependencies; API is internal)
