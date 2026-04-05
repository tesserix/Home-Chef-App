# Requirements: Home Chef Mobile Apps

**Defined:** 2026-04-05
**Core Value:** Customers can browse home chefs, order food, and track delivery on their phones — while chefs manage orders and drivers navigate deliveries natively.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Foundation

- [ ] **FOUND-01**: Expo monorepo scaffold with Metro config resolving pnpm workspace packages
- [ ] **FOUND-02**: Shared `packages/mobile-shared/` package with typed API client, hooks, and utils
- [ ] **FOUND-03**: `@tesserix/native` design system integrated with shared color tokens and theme matching web
- [ ] **FOUND-04**: EAS Build configuration for iOS and Android (all 3 apps)
- [ ] **FOUND-05**: Deep linking with distinct URI schemes per app (homechef-customer://, homechef-vendor://, homechef-driver://)
- [ ] **FOUND-06**: Secure token storage via expo-secure-store (not AsyncStorage)

### Authentication

- [ ] **AUTH-01**: User can log in with email and password (all 3 apps)
- [ ] **AUTH-02**: User can register a new account (all 3 apps)
- [ ] **AUTH-03**: User can reset password via email (vendor app)
- [ ] **AUTH-04**: User can sign in with Google (all 3 apps)
- [ ] **AUTH-05**: User can sign in with Apple (all 3 apps — App Store requirement)
- [ ] **AUTH-06**: User can authenticate via biometrics (Face ID / fingerprint) after first login
- [ ] **AUTH-07**: JWT tokens auto-refresh in background without session interruption

### Customer App — Browse & Order

- [ ] **CUST-01**: Customer can browse and discover home chefs on home screen
- [ ] **CUST-02**: Customer can view chef detail page with menu items
- [ ] **CUST-03**: Customer can add items to cart and modify quantities
- [ ] **CUST-04**: Customer can checkout and pay via Razorpay
- [ ] **CUST-05**: Customer can track active order with live GPS map showing driver location
- [ ] **CUST-06**: Customer can view order history and order details
- [ ] **CUST-07**: Customer can save chefs to favorites
- [ ] **CUST-08**: Customer can view and interact with social feed
- [ ] **CUST-09**: Customer can submit catering requests and view quotes
- [ ] **CUST-10**: Customer can manage profile (name, address, preferences)

### Customer App — Onboarding

- [ ] **CUST-11**: New customer completes onboarding flow (basic info, address, food preferences)

### Vendor/Chef App — Menu & Orders

- [ ] **VEND-01**: Chef can view dashboard with live overview (today's orders, earnings, rating)
- [ ] **VEND-02**: Chef can manage menu items (list, create, edit, delete)
- [ ] **VEND-03**: Chef can take food photos with device camera when adding menu items
- [ ] **VEND-04**: Chef can view and accept/reject live incoming orders
- [ ] **VEND-05**: Chef can view order history
- [ ] **VEND-06**: Chef can view earnings and payout details
- [ ] **VEND-07**: Chef can view analytics (sales trends, popular items)
- [ ] **VEND-08**: Chef can view and respond to customer reviews
- [ ] **VEND-09**: Chef can manage profile and kitchen setup
- [ ] **VEND-10**: Chef can manage settings and notification preferences

### Vendor/Chef App — Onboarding

- [ ] **VEND-11**: New chef completes onboarding flow (personal info, kitchen details, operations, documents, policies, review)

### Delivery Driver App — Deliveries

- [ ] **DRIV-01**: Driver can view dashboard with stats (today/week/month deliveries, earnings)
- [ ] **DRIV-02**: Driver can browse and accept available deliveries
- [ ] **DRIV-03**: Driver can view active delivery with pickup/dropoff details
- [ ] **DRIV-04**: Driver can navigate to pickup/dropoff via native maps app (Google Maps/Apple Maps/Waze)
- [ ] **DRIV-05**: Driver can update delivery status (picked up, in transit, delivered)
- [ ] **DRIV-06**: Driver can view delivery history
- [ ] **DRIV-07**: Driver can view earnings
- [ ] **DRIV-08**: Driver can manage fleet (overview, partners)
- [ ] **DRIV-09**: Driver can manage staff
- [ ] **DRIV-10**: Driver can manage profile and settings

### Delivery Driver App — Onboarding

- [ ] **DRIV-11**: New driver completes onboarding flow (personal info, vehicle details, payout, documents, subscription plan, review)

### Delivery Driver App — GPS

- [ ] **DRIV-12**: Driver app sends background GPS location updates while on active delivery
- [ ] **DRIV-13**: Background location permission requested only when driver goes online (not on first launch)

### Push Notifications

- [ ] **PUSH-01**: Customer receives push notifications for order status updates (accepted, ready, picked up, delivered)
- [ ] **PUSH-02**: Vendor receives push notifications for new incoming orders
- [ ] **PUSH-03**: Vendor can accept/reject orders from notification actions (lock screen)
- [ ] **PUSH-04**: Driver receives push notifications for new available deliveries
- [ ] **PUSH-05**: App icon badge shows unread notification count
- [ ] **PUSH-06**: FCM device tokens registered via existing `PUT /profile/device-token` endpoint (raw FCM tokens, not Expo Push Tokens)

### UX Polish

- [ ] **UX-01**: Pull-to-refresh on all list views
- [ ] **UX-02**: Skeleton loading screens on key views (home, orders, menu)
- [ ] **UX-03**: Graceful offline error state (not crash or blank screen)
- [ ] **UX-04**: Haptic feedback on key actions (order placed, accept/reject, delivery complete)

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Differentiators

- **DIFF-01**: Live animated driver pin on customer order tracking map (smooth movement via WebSocket)
- **DIFF-02**: Shake to report issue (driver app)
- **DIFF-03**: App Store rating prompt after successful delivery
- **DIFF-04**: Scheduled order reminders (push 30 min before delivery)
- **DIFF-05**: Photo confirmation on delivery (proof-of-delivery)
- **DIFF-06**: One-tap reorder from order history
- **DIFF-07**: Smart ETA countdown display
- **DIFF-08**: Notification preference management (choose which push types)
- **DIFF-09**: Chef availability toggle home screen widget (iOS/Android)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Full offline mode with sync | Complex conflict resolution infrastructure, risk outweighs benefit for v1 |
| In-app native payment (Apple Pay / Google Pay) | High effort, PCI scope expansion, Razorpay web checkout works for v1 |
| Custom video calling | Not in web version, high infrastructure cost (WebRTC) |
| AR menu preview | Device-dependent, complex, low ROI for v1 |
| Tablet-optimized layouts | Phone-first for v1, doubles design effort |
| Background location tracking for customers | Privacy-invasive, battery draining, App Store scrutiny |
| Admin mobile app | Admin stays web-only — complex dashboards not suited for mobile |
| Live chat audio/video | Out of scope, text-only chat already exists |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| FOUND-01 | Phase 1 | Pending |
| FOUND-02 | Phase 1 | Pending |
| FOUND-03 | Phase 1 | Pending |
| FOUND-04 | Phase 1 | Pending |
| FOUND-05 | Phase 1 | Pending |
| FOUND-06 | Phase 1 | Pending |
| AUTH-01 | Phase 1 | Pending |
| AUTH-02 | Phase 1 | Pending |
| AUTH-03 | Phase 1 | Pending |
| AUTH-04 | Phase 1 | Pending |
| AUTH-05 | Phase 1 | Pending |
| AUTH-06 | Phase 1 | Pending |
| AUTH-07 | Phase 1 | Pending |
| CUST-01 | Phase 2 | Pending |
| CUST-02 | Phase 2 | Pending |
| CUST-03 | Phase 2 | Pending |
| CUST-04 | Phase 2 | Pending |
| CUST-05 | Phase 2 | Pending |
| CUST-06 | Phase 2 | Pending |
| CUST-07 | Phase 2 | Pending |
| CUST-08 | Phase 2 | Pending |
| CUST-09 | Phase 2 | Pending |
| CUST-10 | Phase 2 | Pending |
| CUST-11 | Phase 2 | Pending |
| VEND-01 | Phase 3 | Pending |
| VEND-02 | Phase 3 | Pending |
| VEND-03 | Phase 3 | Pending |
| VEND-04 | Phase 3 | Pending |
| VEND-05 | Phase 3 | Pending |
| VEND-06 | Phase 3 | Pending |
| VEND-07 | Phase 3 | Pending |
| VEND-08 | Phase 3 | Pending |
| VEND-09 | Phase 3 | Pending |
| VEND-10 | Phase 3 | Pending |
| VEND-11 | Phase 3 | Pending |
| DRIV-01 | Phase 3 | Pending |
| DRIV-02 | Phase 3 | Pending |
| DRIV-03 | Phase 3 | Pending |
| DRIV-04 | Phase 3 | Pending |
| DRIV-05 | Phase 3 | Pending |
| DRIV-06 | Phase 3 | Pending |
| DRIV-07 | Phase 3 | Pending |
| DRIV-08 | Phase 3 | Pending |
| DRIV-09 | Phase 3 | Pending |
| DRIV-10 | Phase 3 | Pending |
| DRIV-11 | Phase 3 | Pending |
| DRIV-12 | Phase 4 | Pending |
| DRIV-13 | Phase 4 | Pending |
| PUSH-01 | Phase 4 | Pending |
| PUSH-02 | Phase 4 | Pending |
| PUSH-03 | Phase 4 | Pending |
| PUSH-04 | Phase 4 | Pending |
| PUSH-05 | Phase 4 | Pending |
| PUSH-06 | Phase 4 | Pending |
| UX-01 | Phase 4 | Pending |
| UX-02 | Phase 4 | Pending |
| UX-03 | Phase 4 | Pending |
| UX-04 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 58 total
- Mapped to phases: 58
- Unmapped: 0

---
*Requirements defined: 2026-04-05*
*Last updated: 2026-04-05 after roadmap creation — all 58 requirements mapped*
