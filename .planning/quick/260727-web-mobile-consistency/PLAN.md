---
slug: web-mobile-consistency
created: 2026-07-27
mode: quick
branch: fix/web-mobile-consistency
---

# Make the customer + vendor web apps a consistent subset of the mobile apps

## Problem

`apps/web` (customer) and `apps/vendor-portal` (chef) drifted from their mobile
counterparts. They advertise surfaces that either do not exist in the product
any more or never existed in the apps at all, so the web reads as a different —
and partly broken — product.

Audited `apps/web` vs `apps/mobile-customer` and `apps/vendor-portal` vs
`apps/mobile-vendor`. Three classes of drift:

### 1. `apps/vendor-portal` — a Premium tier the mobile app has never had

`/premium` (`features/billing/pages/PremiumPage.tsx`, plus a `Sparkles` sidebar
entry) offers a chef Premium upgrade. `apps/mobile-vendor` has **zero**
references to `premium` or `subscription/tier` — the only hit in the whole app
is prose in `lib/pricing-guidance.ts` about a "premium biryani".

The backend does implement it (`PUT /v1/chef/subscription/tier`, tagged `#44`,
`apps/api/routes/routes.go`), so the page is not broken — it is a feature the
product never shipped to chefs. Owner call 2026-07-27: remove it. The API
endpoints stay untouched, so the page is restorable from git if Premium returns.

### 2. `apps/web` — three whole role portals embedded in the customer app

`app/routes/index.tsx` mounts `/chef/*` (7 pages), `/admin/*` (6) and
`/delivery/*` (3) behind role guards, with their own `ChefLayout`, `AdminLayout`
and `DeliveryLayout`. These predate the portal split and now duplicate
`apps/vendor-portal` / `apps/mobile-vendor`, `apps/mobile-admin` and
`apps/delivery-portal` / `apps/mobile-delivery`.

`apps/mobile-customer` is customer-only. Verified self-contained: nothing outside
`app/routes/index.tsx` imports `features/chef|admin|delivery`, every internal
link in those trees stays inside its own tree, and `LoginPage` always navigates
to `/` — there is no role-based redirect to preserve.

### 3. `apps/web` — seven links that 404 into the catch-all

`<Route path="*">` redirects to `/`, so each of these silently bounces the user
home instead of erroring:

| Link | Where |
|---|---|
| `/about` | `MainLayout` footer |
| `/help` | `MainLayout` footer, `RefundPolicyPage` |
| `/chef-resources` | `MainLayout` footer, `HomePage` CTA |
| `/become-chef` | `MainLayout` footer, `HomePage` CTA |
| `/how-it-works` | `HeroSection` CTA |
| `/forgot-password` | `LoginPage` |
| `/settings` | `nav-items.ts` `ACCOUNT_SECONDARY_NAV` → account drawer |

`shared/components/layout/nav-items.ts` already documents four of these as
"which 404 today" and routes around them locally — but the footer, hero, login
screen and the drawer's own Settings row still ship them.

`/settings` is the sharpest: it renders in the account drawer for every
signed-in customer. The only `settings` route in the app belongs to the admin
tree (`/admin/settings`), which step 2 deletes.

## Scope

Remove-only. No new features, no parity build-out. The web becomes a clean
subset of mobile; the gaps where mobile has screens web lacks are recorded in
SUMMARY.md as follow-up, not built here.

## Tasks

1. **vendor-portal — drop Premium.** Delete `features/billing/pages/PremiumPage.tsx`
   (and `features/billing/` if it empties), its `lazyWithRetry` import and
   `<Route path="premium">` in `app/routes/index.tsx`, and the `Premium` entry +
   now-unused `Sparkles` import in `VendorLayout.tsx`.

2. **vendor-portal — surface the orphaned `/prep` route.** `PrepPage` is routed
   but appears in no sidebar and is linked from nowhere, so it is unreachable.
   Mobile groups it under Meal plans. Add one sidebar entry next to Weekly Menu.

3. **apps/web — delete the embedded role portals.** Remove the `/chef`, `/admin`
   and `/delivery` route blocks and their 16 lazy imports from
   `app/routes/index.tsx`; delete `features/chef/`, `features/admin/`,
   `features/delivery/` and `shared/components/layout/{Chef,Admin,Delivery}Layout.tsx`.
   Keep `ProtectedRoute`'s `roles` parameter only if still used.

4. **apps/web — fix the seven dead links.** Delete the `/about`, `/help`,
   `/chef-resources`, `/how-it-works` and `/forgot-password` links and the
   `/settings` drawer row. Point `/become-chef` at `https://vendors.fe3dr.com`
   as an external link, matching the destination `PARTNER_NAV` already uses for
   "Add your kitchen".

## Verification

- `pnpm --filter @homechef/web build` and `--filter ...vendor-portal build` pass
  (catches dangling imports from the deletions).
- Typecheck/lint clean — `noUnusedLocals` will catch orphaned icon imports.
- Re-run the link-vs-route cross-check after the route deletions: no internal
  `to=`/`href=` in either app may point outside its route table.
