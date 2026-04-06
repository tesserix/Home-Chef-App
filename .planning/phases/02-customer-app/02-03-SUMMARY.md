---
phase: 02-customer-app
plan: 03
subsystem: mobile-customer
tags: [checkout, payment, razorpay, expo-web-browser, polling, addresses]
dependency_graph:
  requires:
    - 02-02 (cart store, CartItem/Order/Address types, api client)
  provides:
    - checkout screen with address selection and order summary
    - Razorpay hosted checkout via expo-web-browser
    - 60s payment status polling
    - cart cleared on confirmed payment
  affects:
    - apps/mobile-customer/app/checkout.tsx
    - apps/mobile-customer/app/payment/result.tsx
    - apps/mobile-customer/hooks/useOrderCheckout.ts
    - apps/mobile-customer/hooks/useAddresses.ts
tech_stack:
  added: []
  patterns:
    - expo-web-browser for Razorpay hosted checkout (D-04 fallback)
    - TanStack Query useMutation + polling useQuery for order/payment lifecycle
    - react-hook-form + zod for inline address form
    - 60s hard timeout + status-based early-stop for polling
key_files:
  created:
    - apps/mobile-customer/hooks/useOrderCheckout.ts
    - apps/mobile-customer/hooks/useAddresses.ts
    - apps/mobile-customer/app/checkout.tsx
    - apps/mobile-customer/app/payment/result.tsx
  modified: []
decisions:
  - "D-04 confirmed: expo-web-browser + Razorpay hosted checkout URL used — react-native-razorpay incompatible with Expo managed SDK 55"
  - "Payment confirmation via server-side webhook + client-side polling (not URL params from callback_url)"
  - "polling interval 3s, hard stop after 60s, early stop on any non-pending status"
metrics:
  duration_minutes: 22
  completed_date: "2026-04-06T02:34:20Z"
  tasks_completed: 2
  tasks_total: 2
  files_created: 4
  files_modified: 0
requirements:
  - CUST-04
---

# Phase 02 Plan 03: Checkout and Razorpay Payment Flow Summary

**One-liner:** Checkout screen using expo-web-browser + Razorpay hosted checkout with 3s polling and 60s timeout for payment confirmation.

## What Was Built

### Task 1: Order checkout and address hooks (commit a93f863)

- `hooks/useOrderCheckout.ts` — exports `useCreateOrder` (mutation: POST /v1/orders) and `useOrderStatus` (polling query: GET /v1/orders/:id, 3s interval, stops on non-pending status)
- `hooks/useAddresses.ts` — exports `useAddresses` (query: GET /v1/addresses, 5min stale) and `useCreateAddress` (mutation: POST /v1/addresses)

Both hook files deliberately do not overlap with `hooks/useOrderHistory.ts` (Plan 05 scope).

### Task 2: Checkout screen + Razorpay payment flow (commit a5006be)

- `app/checkout.tsx` — full checkout screen:
  - Delivery address selector (FlatList of saved addresses with radio selection, default pre-selected)
  - Inline "Add New Address" form (react-hook-form + zod, same schema as onboarding)
  - Read-only order summary (items, qty, price, subtotal, delivery fee, total)
  - Optional note-to-chef input
  - handlePlaceOrder: POST /v1/orders → POST /v1/payments/order/:id/create → build Razorpay hosted URL → WebBrowser.openBrowserAsync
  - Polling: useOrderStatus every 3s, clears cart and navigates to /order/:id on confirmed/preparing/ready/picked_up/delivered, shows error on cancelled
  - 60s hard timeout on polling
  - KeyboardAvoidingView for Android keyboard handling
  - Place Order button disabled when cart empty or no address selected

- `app/payment/result.tsx` — deep link handler for `homechef-customer://payment/result`, immediately navigates back to checkout (which is already polling)

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| expo-web-browser (not react-native-razorpay) | react-native-razorpay requires native modules incompatible with Expo managed workflow SDK 55 — confirmed D-04 from planning |
| Webhook + polling for payment confirmation | Razorpay callback_url params are not cryptographically verified client-side; server webhook is authoritative (T-02-03-02 accepted) |
| 60s polling timeout | T-02-03-05 mitigation — prevents infinite polling loop; user directed to order history on timeout |
| Free delivery fee (₹0) for v1 | Per project requirements: in-app payments use Razorpay web checkout for v1, no delivery fee calculation needed yet |

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all data is wired to live API endpoints. The payment data shape handles both `response.data.data` (standard envelope) and direct `response.data` (if API returns flat) to be resilient to the actual response shape from `payment.go`.

## Threat Surface Scan

No new trust boundaries introduced beyond what the plan's threat model covers. The `callback_url` deep link acceptance criteria and polling design match T-02-03-02 (accepted) and T-02-03-05 (mitigated with 60s timeout).

## Self-Check: PASSED

| Item | Status |
|------|--------|
| apps/mobile-customer/hooks/useOrderCheckout.ts | FOUND |
| apps/mobile-customer/hooks/useAddresses.ts | FOUND |
| apps/mobile-customer/app/checkout.tsx | FOUND |
| apps/mobile-customer/app/payment/result.tsx | FOUND |
| commit a93f863 (Task 1 hooks) | FOUND |
| commit a5006be (Task 2 screens) | FOUND |
