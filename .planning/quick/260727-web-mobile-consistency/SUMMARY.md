---
slug: web-mobile-consistency
created: 2026-07-27
completed: 2026-07-27
mode: quick
status: complete
branch: fix/web-mobile-consistency
---

# Summary — web apps realigned as a subset of the mobile apps

Remove-only cleanup. `apps/web` and `apps/vendor-portal` no longer advertise
surfaces the product does not ship, and every internal link in both apps now
resolves to a route that exists.

## What changed

### `apps/vendor-portal`

- **Removed the Premium tier.** Deleted `features/billing/` (`PremiumPage.tsx`),
  its route, and the `Premium` sidebar entry + `Sparkles` import. `mobile-vendor`
  has no premium concept anywhere. The backend (`PUT /v1/chef/subscription/tier`,
  `#44`) is **untouched** — if Premium is revived, restore the page from this
  commit's parent.
- **Surfaced `/prep`.** `PrepPage` (bulk subscription prep, `#50`, itself written
  for "web parity with the vendor mobile app") was routed but in no sidebar and
  linked from nowhere, so it was reachable only by typing the URL. Added one
  sidebar entry beside Weekly Menu.

### `apps/web`

- **Deleted three embedded role portals**: `/chef/*` (7 pages), `/admin/*` (6),
  `/delivery/*` (3), plus `Chef/Admin/DeliveryLayout.tsx` and the
  `features/chef|admin|delivery` trees — 19 files. They predate the portal split
  and duplicated `vendor-portal` / `mobile-vendor`, `mobile-admin`, and
  `delivery-portal` / `mobile-delivery`. `mobile-customer` is customer-only.
- **Simplified `ProtectedRoute`** — its `roles` guard had no callers left once
  those trees went, so a session is now the only gate.
- **Fixed 7 dead links** that fell through `<Route path="*">` and silently
  returned the user home rather than erroring:

  | Link | Was | Now |
  |---|---|---|
  | `/settings` | account drawer row, shown to **every signed-in customer** | removed; account settings are on `/profile`, already linked from the drawer header |
  | `/forgot-password` | LoginPage "Forgot password?" | removed — it stranded people mid-recovery |
  | `/become-chef` | footer + HomePage CTA | → `https://vendors.fe3dr.com`, the destination `PARTNER_NAV` already uses |
  | `/chef-resources` | footer + HomePage "Learn More" | removed — no such page exists |
  | `/about` | footer | removed |
  | `/help` | footer + RefundPolicyPage | removed |
  | `/how-it-works` | HeroSection secondary CTA | removed |

  `/settings` was the sharpest: the only `settings` route in the app belonged to
  the deleted admin tree. Also linked the existing-but-unlinked `/refund` page
  from the footer alongside Privacy and Terms.

## Verification

- `tsc --noEmit` clean on both apps.
- `vite build` succeeds for both (would have caught any dangling import).
- `eslint src`: **0 errors** on both; the 21 warnings each are pre-existing
  `react-refresh/only-export-components` in untouched design-system files.
- `vitest run` in `apps/web`: **24/24 passing**. `vendor-portal` has no test
  files — pre-existing, not introduced here.
- Re-ran the link-vs-route cross-check on both apps after the deletions: **no
  internal `to=`/`href=` points outside its route table.**

## Not done — deliberate

Scope was remove-only (owner call 2026-07-27). The web is now a clean *subset*
of mobile; these gaps are recorded, not built:

**`apps/web` lacks, vs `mobile-customer`:** password reset, notifications, dish
search, chefs map, live order tracking, order messages / receipt / report-issue,
address management, blocked accounts, support chat, change password.

**`apps/vendor-portal` lacks, vs `mobile-vendor`:** catering, payout bank
account, document renewal, notification preferences, language (EN/हिन्दी),
support tickets, legal pages.

Password reset is the most user-visible: web login no longer offers recovery at
all. Worth its own issue.

## Note for whoever reads the sunset docs

`apps/web/SUNSET.md` and `apps/vendor-portal/SUNSET.md` are **stale**. They
describe both apps as paused/decommissioned, but `homechef-web-build.yml` now
publishes `homechef-web-app` on every `main` commit (its `push` trigger is live,
and Kargo needs an image at every promoted tag). This work ships. Those two docs
should be reconciled with reality separately — not touched here to keep the diff
remove-only.
