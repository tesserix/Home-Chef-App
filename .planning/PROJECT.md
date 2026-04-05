# Home Chef Mobile Apps

## What This Is

Three native mobile apps (Customer, Vendor/Chef, Delivery Driver) for the Home Chef platform, built with Expo (React Native) and the existing `@tesserix/native` design system. The apps replicate full feature parity with the existing web portals (`apps/web/`, `apps/vendor-portal/`, `apps/delivery-portal/`) while adding mobile-native capabilities like GPS/maps for live delivery tracking and navigation. All apps consume the existing Go API backend.

## Core Value

Customers can browse home chefs, order food, and track delivery on their phones — while chefs manage orders and drivers navigate deliveries natively.

## Requirements

### Validated

- ✓ Go API backend with all endpoints (auth, orders, menu, delivery, payments, etc.) — existing
- ✓ `@tesserix/native` design system package with shared tokens/theme — existing
- ✓ Customer web portal with browse, order, cart, checkout, favorites, social, catering — existing
- ✓ Vendor web portal with dashboard, menu management, orders, earnings, analytics — existing
- ✓ Delivery web portal with available/active deliveries, fleet, earnings, onboarding — existing

### Active

#### Customer App
- [ ] Auth (login, register) with existing API
- [ ] Onboarding flow (basic info, address, preferences)
- [ ] Home screen with chef discovery/browse
- [ ] Chef detail page with menu
- [ ] Cart and checkout with payment
- [ ] Order tracking with live GPS map
- [ ] Order history and order detail
- [ ] Favorites list
- [ ] Customer profile management
- [ ] Social feed
- [ ] Catering requests and quotes

#### Vendor/Chef App
- [ ] Auth (login, register, forgot password)
- [ ] Chef onboarding (personal info, kitchen details, operations, documents, policies)
- [ ] Dashboard with live overview
- [ ] Menu management (list, create/edit, view items)
- [ ] Live orders with accept/reject
- [ ] Order history
- [ ] Earnings and payouts
- [ ] Analytics
- [ ] Customer reviews
- [ ] Push notifications for new orders
- [ ] Profile and kitchen setup
- [ ] Settings

#### Delivery Driver App
- [ ] Auth (login, register)
- [ ] Driver onboarding (personal info, vehicle, payout, documents, subscription)
- [ ] Dashboard
- [ ] Available deliveries with accept
- [ ] Active delivery with GPS navigation
- [ ] Delivery history
- [ ] Fleet management (overview, partners)
- [ ] Staff management
- [ ] Earnings
- [ ] Profile and settings

#### Shared / Infrastructure
- [ ] Expo monorepo setup within existing `apps/` directory
- [ ] `@tesserix/native` design system integration
- [ ] Shared API client layer across all three apps
- [ ] Push notification infrastructure (Expo Notifications)
- [ ] GPS/Maps integration (react-native-maps + location services)
- [ ] iOS and Android build configuration (EAS Build)
- [ ] Deep linking support

### Out of Scope

- Admin mobile app — admin stays web-only, complex dashboards not suited for mobile
- Offline mode — requires significant sync infrastructure, defer to future
- In-app payments (Apple Pay / Google Pay) — use existing Razorpay web checkout for v1
- Video calling between customer and chef — not in web version either
- AR menu preview — future enhancement
- Tablet-optimized layouts — phone-first for v1

## Context

- The Go API at `apps/api/` is the single backend — no API changes needed for mobile, just consume existing endpoints
- Four web frontends exist: `apps/web/` (customer), `apps/vendor-portal/` (vendor), `apps/delivery-portal/` (driver), `apps/admin-portal/` (admin)
- All web apps use Vite + React 19 + React Router v7 + Tailwind v4 + Radix UI + `@tesserix/web` design system
- Design system monorepo already has `@tesserix/native` package for React Native components
- Auth uses JWT tokens via `Authorization: Bearer` header — mobile apps can use the same mechanism
- Current web delivery portal already has some mobile-optimized views
- Existing API handlers at `apps/api/handlers/` cover: auth, orders, menu, delivery, payment, chat, social, catering, favorites, reviews, notifications, promo codes, support tickets

## Constraints

- **Stack**: Expo (React Native) with TypeScript — matches existing frontend skill set
- **Design System**: Must use `@tesserix/native` — brand consistency across platforms
- **Repo**: Apps live inside existing monorepo at `apps/mobile-customer/`, `apps/mobile-vendor/`, `apps/mobile-delivery/`
- **Platforms**: iOS and Android (both required)
- **API**: No backend changes — consume existing Go API as-is
- **Budget**: Use EAS Build for CI/CD (Expo's managed build service)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Expo (React Native) over Flutter | Team already uses React/TypeScript, reuse knowledge and design system | — Pending |
| Three separate apps over one multi-role app | Each user type has distinct workflows, separate store listings, cleaner UX | — Pending |
| In monorepo alongside web apps | Shared types, API client code, easier development | — Pending |
| `@tesserix/native` design system | Already exists, enforces brand consistency | — Pending |
| GPS/Maps as mobile-native addition | Key differentiator over web — live delivery tracking and driver navigation | — Pending |
| Full feature parity with web | Users expect the same capabilities on mobile | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? -> Move to Out of Scope with reason
2. Requirements validated? -> Move to Validated with phase reference
3. New requirements emerged? -> Add to Active
4. Decisions to log? -> Add to Key Decisions
5. "What This Is" still accurate? -> Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-05 after initialization*
