# Summary — 260610-iy5: Customer app v2 (Airbnb redesign + ordering flow fixes)

**Status:** Complete · **Date:** 2026-06-10 · **Final commit:** `55258a5`

## What shipped

### Wave 0/A/B/C — full Airbnb redesign (earlier in session, commit `34fbf83`)
Home, chef detail, checkout, favorites, orders, order detail, live tracking, payment
result, profile, social, catering, onboarding, shared auth screens. Palette: white
canvas / charcoal / rausch coral `#FF385C`. Plus the critical data-layer fixes that
made the app work end-to-end (API base `/api`, onboarding endpoint + flat address
payload, chef mappers, `GestureHandlerRootView`, friendly errors). See
`NEXT-SESSION-PROMPT.md` for the full list.

### Final two functional items (this follow-up, commits `688ad1f`, `55258a5`)

1. **Checkout delivery address was empty even after onboarding** (`688ad1f`).
   Root cause was NOT auth — `bffAuth` has a Bearer-token fallback (`verifyBearer`)
   so the mobile GIP token is accepted, same as orders/favorites. The real bug was a
   shape/field mismatch: `GET /v1/addresses` returns a **bare array** with
   `line1/line2/postalCode`, but the hook expected `{data: Address[]}` with
   `addressLine1/pincode`. Fixed by mapping at the `useAddresses.ts` boundary (chef
   pattern). Also fixed `useCreateAddress` to send the backend-required `label` and the
   `line1/postalCode` wire names. Verified live on sim — the saved Mumbai address now
   renders, pre-selected, with the "Default" badge.

2. **Per-item special instructions** (`55258a5`). `CartItem.instructions` +
   `setInstructions` immutable store action + per-line "Add a note" input in the cart
   sheet + each item's note sent as `notes` (the `CreateOrderItem` wire field) on
   CreateOrder. Order-level `note` was already wired.

## Verification
- tsc: 77 errors = baseline, **0 net new** (confirmed by stash/recount).
- Grep gates: no `herb` in customer; coral confined to customer.
- Live sim QA confirmed item 1 against prod backend. Item 2 is tsc-clean; the cart
  sheet can't be screenshot-tested without tap input (simctl limitation) — no item in
  cart on a fresh launch.

### Address labels (follow-up, commit `7b3dff2`)
Home/Work/Other selector (shared `components/address/AddressLabelSelect.tsx`) on
both the checkout "Add new address" form and the onboarding address step; label
mapped through `useAddresses` and sent on create + onboarding submit; label shown
as a chip in the checkout address list. Verified live — existing address renders a
"Home" chip beside "Default".

## Open follow-up
- **Order detail can't show a label chip:** the order's `deliveryAddress` is a flat
  point-in-time snapshot on the orders table (`AddressResponse` = line1/line2/city/
  state/postalCode) with **no label column**. Adding a chip there needs a backend
  change (add `DeliveryAddressLabel`) — out of scope (no backend changes).
- **Order detail address fields are mismapped** (pre-existing, separate from this
  task): `app/order/[id]` reads `deliveryAddress.addressLine1/pincode` but the order
  API returns `line1/postalCode`, so those render undefined. Same bug class as the
  checkout fix; `hooks/useOrderHistory.ts` casts the response with no mapper. Worth a
  small follow-up (map AddressResponse → Address in `useOrder`/`useOrders`).
- "Other" label is a fixed value; no custom free-text label input yet.
