# Phase 3: Vendor App + Driver Core - Context

**Gathered:** 2026-04-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Complete Vendor/Chef app (dashboard, menu management with camera photos, live order queue with accept/reject, order history, earnings, analytics, reviews, profile, kitchen setup, settings, 6-step onboarding) AND Driver app core (dashboard, available deliveries, active delivery with native maps navigation handoff, status updates, delivery history, fleet management, staff management, earnings, profile, settings, 6-step onboarding). Background GPS and push notifications are Phase 4 — this phase delivers the full UI workflow using polling only.

</domain>

<decisions>
## Implementation Decisions

### Vendor Live Orders
- **D-01:** Order queue UX at Claude's discretion — recommend card stack (full-width cards) with prominent Accept/Reject buttons. Cards show customer name, items, total, and elapsed time since order placed.
- **D-02:** New order alert at Claude's discretion — recommend polling every 10s via React Query with in-app sound + haptic feedback when new order count increases. Real push notifications come in Phase 4.
- **D-03:** Accept/Reject UX at Claude's discretion — recommend instant action with 3-second undo snackbar (optimistic update). Allows fast workflow with safety net.

### Vendor Menu Management
- **D-04:** Menu edit UX at Claude's discretion — recommend separate edit screen for full edits (matches web's MenuItemFormPage), with inline availability toggle on menu list cards for quick sold-out marking.
- **D-05:** Food photo capture at Claude's discretion — recommend `expo-image-picker` with both camera and library options. Upload to GCS via existing `POST /api/v1/upload` endpoint.
- **D-06:** Quick availability toggle at Claude's discretion — recommend switch directly on menu list card for one-tap sold-out marking during busy service.

### Vendor Onboarding
- **D-07:** 6-step onboarding flow at Claude's discretion — recommend step-by-step wizard with progress indicator (matches web: personal info → kitchen details → operations → documents → policies → review). One screen per step for focused input.
- **D-08:** Document upload at Claude's discretion — recommend `expo-image-picker` with camera + library, PDF support via `expo-document-picker`. Required docs: ID proof, FSSAI license.

### Driver Delivery Workflow
- **D-09:** Available deliveries UX at Claude's discretion — recommend list view with distance + payout info (one-tap accept). Map pin view is a Phase 4+ enhancement.
- **D-10:** Active delivery screen at Claude's discretion — recommend current step (Pickup/Dropoff) prominently displayed with address card and large Navigate button.
- **D-11:** Navigate button at Claude's discretion — recommend platform-native default (iOS opens Apple Maps via `maps://`, Android opens Google Maps via `comgooglemaps://` or geo: fallback). Use `Linking.openURL`.
- **D-12:** Status update UX at Claude's discretion — recommend swipe-to-confirm pattern (slide-to-confirm) for status transitions (picked up → in transit → delivered) to prevent accidental taps while driving.

### Shared Patterns (Carried from Phase 1 & 2)
- **D-13:** Both apps use `@tesserix/native` components + NativeWind styling (Phase 1 D-10, D-11)
- **D-14:** Both apps use `packages/mobile-shared/` for API client, auth, storage (Phase 1 D-02, D-03)
- **D-15:** Both apps use Zustand + React Query for state (Phase 2 pattern)
- **D-16:** Both apps use expo-router with route groups: (auth)/, (onboarding)/, (tabs)/ (Phase 1 pattern)
- **D-17:** Tab navigation structure at Claude's discretion per app (recommend: Vendor — Dashboard/Orders/Menu/More; Driver — Dashboard/Available/Active/More)

### Claude's Discretion
- Almost everything — user deferred all specific UX decisions
- Tab structures, screen layouts, interaction patterns all to be determined by Claude following Phase 1-2 established patterns

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project Context
- `.planning/PROJECT.md` — Project vision, constraints
- `.planning/REQUIREMENTS.md` — VEND-01 through VEND-11, DRIV-01 through DRIV-11

### Prior Phase Decisions
- `.planning/phases/01-foundation-auth/01-CONTEXT.md` — Monorepo structure, auth, design system (all apps)
- `.planning/phases/02-customer-app/02-CONTEXT.md` — Mobile UX patterns established in customer app (reuse)

### Shared Infrastructure (Phase 1)
- `packages/mobile-shared/src/api/client.ts` — API client factory (both apps reuse)
- `packages/mobile-shared/src/hooks/useAuth.ts` — Auth store (both apps)
- `packages/mobile-shared/src/theme/tokens.ts` — Design tokens
- `packages/mobile-shared/src/screens/LoginScreen.tsx` — Shared auth screens

### Existing Web Vendor Portal
- `apps/vendor-portal/src/features/dashboard/pages/DashboardPage.tsx` — Dashboard layout
- `apps/vendor-portal/src/features/menu/pages/MenuPage.tsx` — Menu list
- `apps/vendor-portal/src/features/menu/pages/MenuItemFormPage.tsx` — Menu item form
- `apps/vendor-portal/src/features/menu/pages/MenuItemViewPage.tsx` — Menu item detail
- `apps/vendor-portal/src/features/orders/pages/LiveOrdersPage.tsx` — Live orders queue
- `apps/vendor-portal/src/features/orders/pages/OrderHistoryPage.tsx` — Order history
- `apps/vendor-portal/src/features/earnings/pages/EarningsPage.tsx` — Earnings summary
- `apps/vendor-portal/src/features/earnings/pages/PayoutsPage.tsx` — Payouts list
- `apps/vendor-portal/src/features/analytics/pages/AnalyticsPage.tsx` — Analytics charts
- `apps/vendor-portal/src/features/reviews/pages/ReviewsPage.tsx` — Customer reviews
- `apps/vendor-portal/src/features/profile/pages/ProfilePage.tsx` — Profile
- `apps/vendor-portal/src/features/profile/pages/KitchenSetupPage.tsx` — Kitchen setup
- `apps/vendor-portal/src/features/settings/pages/SettingsPage.tsx` — Settings
- `apps/vendor-portal/src/features/onboarding/components/StepPersonalInfo.tsx` — Onboarding step 1
- `apps/vendor-portal/src/features/onboarding/components/StepKitchenDetails.tsx` — Step 2
- `apps/vendor-portal/src/features/onboarding/components/StepOperations.tsx` — Step 3
- `apps/vendor-portal/src/features/onboarding/components/StepDocuments.tsx` — Step 4
- `apps/vendor-portal/src/features/onboarding/components/StepPolicies.tsx` — Step 5
- `apps/vendor-portal/src/features/onboarding/components/StepReview.tsx` — Step 6

### Existing Web Delivery Portal
- `apps/delivery-portal/src/features/dashboard/pages/DashboardPage.tsx` — Driver dashboard
- `apps/delivery-portal/src/features/deliveries/pages/AvailableDeliveriesPage.tsx` — Available list
- `apps/delivery-portal/src/features/deliveries/pages/ActiveDeliveryPage.tsx` — Active delivery
- `apps/delivery-portal/src/features/deliveries/pages/DeliveryHistoryPage.tsx` — History
- `apps/delivery-portal/src/features/earnings/pages/EarningsPage.tsx` — Earnings
- `apps/delivery-portal/src/features/fleet/pages/FleetOverviewPage.tsx` — Fleet overview
- `apps/delivery-portal/src/features/fleet/pages/PartnersPage.tsx` — Partners
- `apps/delivery-portal/src/features/fleet/pages/PartnerDetailPage.tsx` — Partner detail
- `apps/delivery-portal/src/features/staff/pages/StaffPage.tsx` — Staff management
- `apps/delivery-portal/src/features/profile/pages/ProfilePage.tsx` — Profile
- `apps/delivery-portal/src/features/settings/pages/SettingsPage.tsx` — Settings
- `apps/delivery-portal/src/features/onboarding/components/StepPersonalInfo.tsx` — Driver onboarding step 1
- `apps/delivery-portal/src/features/onboarding/components/StepVehicleDetails.tsx` — Step 2
- `apps/delivery-portal/src/features/onboarding/components/StepPayoutDetails.tsx` — Step 3
- `apps/delivery-portal/src/features/onboarding/components/StepDocuments.tsx` — Step 4
- `apps/delivery-portal/src/features/onboarding/components/StepSubscriptionPlan.tsx` — Step 5
- `apps/delivery-portal/src/features/onboarding/components/StepReview.tsx` — Step 6

### Go API Endpoints
- `apps/api/handlers/menu.go` — Menu CRUD
- `apps/api/handlers/orders.go` — Order listing, accept/reject, history
- `apps/api/handlers/chefs.go` — Chef profile, dashboard stats
- `apps/api/handlers/delivery.go` — Driver endpoints: available, accept, status updates, history
- `apps/api/handlers/staff.go` — Staff management
- `apps/api/handlers/upload.go` — File upload for photos and documents
- `apps/api/handlers/reviews.go` — Review listing
- `apps/api/handlers/driver_onboarding.go` — Driver onboarding API
- `apps/api/routes/routes.go` — All routes

### Scaffolded Apps (Phase 1)
- `apps/mobile-vendor/` — Expo app ready for feature screens
- `apps/mobile-delivery/` — Expo app ready for feature screens

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- All Phase 1 shared infrastructure (API client, auth, storage, theme tokens) already wired in `apps/mobile-vendor/` and `apps/mobile-delivery/`
- Phase 2 customer app patterns can be referenced for bottom sheets, pull-to-refresh, form validation with Zod
- `packages/mobile-shared/src/screens/LoginScreen.tsx` — Already wired in both vendor and driver apps
- `@gorhom/bottom-sheet` (installed in Phase 2) — Reuse for vendor order cards expansion and driver status update sheets
- `expo-image-picker` pattern — Customer app may have used it; reuse for menu photos and documents

### Established Patterns
- Zustand for client state (potentially: vendor order queue state, driver active delivery state)
- React Query for server state with polling (10s for vendor orders, 5s for driver location updates in Phase 4)
- NativeWind className everywhere
- Onboarding gate in root `_layout.tsx` (Phase 1 pattern) — extend for vendor and driver onboarding checks

### Integration Points
- `apps/mobile-vendor/app/(tabs)/` — Vendor tab screens to be created
- `apps/mobile-vendor/app/(onboarding)/` — Vendor 6-step onboarding
- `apps/mobile-delivery/app/(tabs)/` — Driver tab screens to be created
- `apps/mobile-delivery/app/(onboarding)/` — Driver 6-step onboarding
- `apps/mobile-vendor/lib/api.ts` — Already has API client instance from Phase 1
- `apps/mobile-delivery/lib/api.ts` — Already has API client instance from Phase 1

### Phase Size Warning
This phase covers 22 requirements across TWO apps. At coarse granularity, expect ~4-6 plans split cleanly between vendor and driver work streams. Plans can run in parallel since vendor and driver apps don't share code beyond `packages/mobile-shared/`.

</code_context>

<specifics>
## Specific Ideas

- Both apps follow the same Phase 1-2 patterns — no new technology introductions
- Phase size is large (22 requirements, 2 apps) — plans should split vendor and driver for parallelization
- All UX decisions at Claude's discretion — follow mobile best practices and Phase 2 conventions

</specifics>

<deferred>
## Deferred Ideas

- Background GPS and push notifications — Phase 4
- Map pin view for available deliveries — Phase 4+ enhancement
- Chef availability home screen widget — v2 (DIFF-09)

</deferred>

---

*Phase: 03-vendor-app-driver-core*
*Context gathered: 2026-04-06*
