# Roadmap: Home Chef Mobile Apps

## Overview

Three Expo (React Native) mobile apps — Customer, Vendor/Chef, and Delivery Driver — built inside the existing monorepo and consuming the Go API backend. The journey moves from verified infrastructure through a working commerce loop (customer orders, vendor accepts), then native GPS and push notification capabilities, closing with UX polish. Each phase leaves all three apps in a testable, deployable state.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Foundation + Auth** - Monorepo scaffold, shared infrastructure, and working auth across all three apps
- [ ] **Phase 2: Customer App** - Complete customer commerce loop from chef discovery through live order tracking
- [ ] **Phase 3: Vendor App + Driver Core** - Full vendor workflow and driver core screens (without GPS/push)
- [ ] **Phase 4: GPS, Push + Polish** - Background GPS, push notifications, and UX polish to ship-ready state

## Phase Details

### Phase 1: Foundation + Auth
**Goal**: All three apps launch, resolve monorepo packages, store tokens securely, and users can sign in and create accounts
**Depends on**: Nothing (first phase)
**Requirements**: FOUND-01, FOUND-02, FOUND-03, FOUND-04, FOUND-05, FOUND-06, AUTH-01, AUTH-02, AUTH-03, AUTH-04, AUTH-05, AUTH-06, AUTH-07
**Success Criteria** (what must be TRUE):
  1. All three apps launch in Expo Go without Metro resolution errors and `@tesserix/native` components render correctly
  2. A user can log in with email/password, and their JWT token survives an app restart (stored in expo-secure-store, not AsyncStorage)
  3. A user can register a new account and a vendor can reset their password via email link
  4. Google Sign-In and Apple Sign-In complete successfully on device; biometric login works after first email login
  5. EAS Build produces a valid iOS and Android binary for each of the three apps (verified via `eas build --local`)
**Plans**: 4 plans
Plans:
- [x] 01-01-PLAN.md — Monorepo scaffold: 3 Expo app skeletons with Metro monorepo config and EAS build profiles
- [x] 01-02-PLAN.md — Shared package: packages/mobile-shared with API client, types, auth hooks, and secure storage
- [x] 01-03-PLAN.md — Email/password auth: shared screens wired into all 3 apps with JWT persistence and auth guard
- [x] 01-04-PLAN.md — Social login and biometrics: Google, Apple, and Face ID/fingerprint across all 3 apps
**UI hint**: yes

### Phase 2: Customer App
**Goal**: A customer can browse chefs, place an order, pay, and watch their delivery on a live map
**Depends on**: Phase 1
**Requirements**: CUST-01, CUST-02, CUST-03, CUST-04, CUST-05, CUST-06, CUST-07, CUST-08, CUST-09, CUST-10, CUST-11
**Success Criteria** (what must be TRUE):
  1. New customer completes onboarding (basic info, address, food preferences) before reaching the home screen
  2. Customer can browse the chef grid, open a chef detail page, add menu items to cart, and complete Razorpay checkout
  3. Customer can see a live map showing the driver's current location while an order is active
  4. Customer can view past orders, manage their profile, save chefs to favorites, browse the social feed, and submit a catering request
**Plans**: 5 plans
Plans:
- [x] 02-01-PLAN.md — Tab nav shell (4 tabs), customer onboarding wizard (3 steps), cart store + type definitions
- [x] 02-02-PLAN.md — Chef browse home screen, chef detail with menu tabs, cart bottom sheet (CUST-01, 02, 03)
- [x] 02-03-PLAN.md — Checkout screen + Razorpay web-browser payment flow + order hooks (CUST-04)
- [x] 02-04-PLAN.md — Live order tracking map with 5s polling and status timeline (CUST-05)
- [x] 02-05-PLAN.md — Secondary screens: order history, favorites, social feed, catering, profile (CUST-06–10)
**UI hint**: yes

### Phase 3: Vendor App + Driver Core
**Goal**: A chef can manage their kitchen and live order queue; a driver can complete the full delivery workflow without GPS push
**Depends on**: Phase 2
**Requirements**: VEND-01, VEND-02, VEND-03, VEND-04, VEND-05, VEND-06, VEND-07, VEND-08, VEND-09, VEND-10, VEND-11, DRIV-01, DRIV-02, DRIV-03, DRIV-04, DRIV-05, DRIV-06, DRIV-07, DRIV-08, DRIV-09, DRIV-10, DRIV-11
**Success Criteria** (what must be TRUE):
  1. New chef completes full onboarding (personal info, kitchen details, operations, documents, policies, review) and reaches the vendor dashboard
  2. Chef can create and edit menu items including taking a food photo with the device camera
  3. Chef can view incoming live orders and accept or reject them from the order queue screen
  4. New driver completes onboarding (personal info, vehicle, payout, documents, subscription) and reaches the driver dashboard
  5. Driver can view available deliveries, accept one, view pickup/dropoff details, update status (picked up / in transit / delivered), and tap Navigate to open Google Maps or Apple Maps with the destination pre-loaded
**Plans**: 6 plans
Plans:
- [x] 03-01-PLAN.md — Vendor onboarding wizard (6 steps), pending/rejected holding screen, 4-tab shell (VEND-11)
- [x] 03-02-PLAN.md — Vendor dashboard and live order queue with 10s polling, optimistic accept/reject + 3s undo (VEND-01, 04, 05)
- [x] 03-03-PLAN.md — Vendor menu CRUD with camera photo upload, earnings, analytics, reviews, profile, settings (VEND-02, 03, 06, 07, 08, 09, 10)
- [x] 03-04-PLAN.md — Driver onboarding wizard (6 steps), pending/rejected holding screen, 4-tab shell (DRIV-11)
- [x] 03-05-PLAN.md — Driver dashboard, available deliveries, active delivery with native maps + swipe-to-confirm status (DRIV-01, 02, 03, 04, 05)
- [x] 03-06-PLAN.md — Driver history, earnings, fleet + staff (403 graceful), profile, settings (DRIV-06, 07, 08, 09, 10)
**UI hint**: yes

### Phase 4: GPS, Push + Polish
**Goal**: Driver app streams background GPS, all three apps receive and act on push notifications, and the UX is polished and ship-ready
**Depends on**: Phase 3
**Requirements**: DRIV-12, DRIV-13, PUSH-01, PUSH-02, PUSH-03, PUSH-04, PUSH-05, PUSH-06, UX-01, UX-02, UX-03, UX-04
**Success Criteria** (what must be TRUE):
  1. While on an active delivery the driver app sends background GPS location updates; the customer's order tracking map updates with the driver's position
  2. Background location permission is requested only when a driver goes online for the first time, not on app launch
  3. Customer receives push notifications for all order lifecycle events (accepted, ready, picked up, delivered); vendor receives push for new orders; driver receives push for available deliveries
  4. Vendor can accept or reject an order directly from a notification on the lock screen without opening the app
  5. All key list views support pull-to-refresh; skeleton screens appear on home, orders, and menu; losing connectivity shows a graceful error state rather than a blank screen or crash; haptic feedback fires on order placed, accept/reject, and delivery complete
**Plans**: 4 plans
Plans:
- [ ] 04-01-PLAN.md — Go API backend: DeliveryResponse serialization fix, rate limiter on PUT /delivery/location, NATS location publish, WebSocket tracking endpoint, actionable push payload structs
- [ ] 04-02-PLAN.md — Driver background GPS: expo-location + expo-task-manager setup, rationale modal, background task with SecureStore JWT, GPS start/stop wired to delivery accept/complete; customer WebSocket tracking hook
- [ ] 04-03-PLAN.md — Push notifications: FCM token wiring + Android channels + iOS vendor categories across all 3 apps; backend NATS-to-push bridge for vendor/driver/customer events
- [ ] 04-04-PLAN.md — UX polish: pull-to-refresh on all list views, skeleton screens, OfflineBanner, haptic feedback on key actions
**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation + Auth | 0/4 | Not started | - |
| 2. Customer App | 0/5 | Not started | - |
| 3. Vendor App + Driver Core | 0/6 | Not started | - |
| 4. GPS, Push + Polish | 0/4 | Not started | - |
