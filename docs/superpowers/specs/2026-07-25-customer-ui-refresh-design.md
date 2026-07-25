# Customer App UI Refresh — Wallet, Profile, Chef Menu

**Date:** 2026-07-25
**Status:** Approved, implementing
**Reference:** Uber Eats wallet, profile, and menu-jump sheet (owner-supplied screenshots)

Three independent screens, shipped in order **wallet → chef menu → profile** so the
cheap wins land before the large profile refactor. All three are client-only: no
API change, no schema, no new endpoints.

---

## 1. Wallet — one balance, two sources

### Problem

`app/wallet.tsx` shows a small card with the wallet balance and a transaction
list. Loyalty points live on an entirely separate `app/loyalty.tsx`. A customer
holding both has no single place that answers "how much can I spend?".

### Design

One headline figure — wallet credit plus the rupee value of points — with the two
sources itemised beneath, mirroring Uber's Cash / Gift Cards split:

```
Fe3dr balance
₹212.40                        ›

  ⓘ  ₹150.40 — Wallet credit
  ⓘ  ₹62.00 — 1,240 loyalty points

Usable on food & delivery.
Fees and taxes are paid separately.
```

Both figures are already client-side: `useWallet().balance` and
`useLoyalty()` → `balance × config.redeemRate`.

The two `ⓘ` affordances carry more weight here than in the reference, because our
two balances genuinely behave differently — wallet credit never expires and has no
per-order cap; points expire after a year and are capped at 10% of food subtotal
and ₹300/month. Each opens a short explainer sheet stating exactly that.

The footer reuses the **same wording as the checkout credits card** — "Fees and
taxes are paid separately" — so the one rule a customer must understand about
credit is phrased identically wherever it appears.

**Nothing is deleted.** `app/loyalty.tsx` remains the points ledger and is reached
by tapping the points row; the transaction list stays below the card.

No CTA in the button slot. There is no customer top-up endpoint — the `walletTopUp`
code in the API funds chef/driver transfers from the platform balance and is
unrelated — so an "Add funds" button would have nothing behind it.

---

## 2. Profile — a hub, not a form

### Problem

`app/(tabs)/profile.tsx` is 799 lines and leads with an inline react-hook-form
block (`Controller` fields for first name, last name, phone). The navigational
quick-links sit *below* that form. This is why it reads cluttered next to the
reference: Uber's profile is a **hub**; ours is a **form that also has links**.

### Design

```
Samyak Rout                    (SR)

 ♡ Favourites   ▤ Wallet   ▥ Orders

 ┌─ Refer & earn ────────────┐
 │ Give ₹100, get ₹100        │
 └────────────────────────────┘

 ⚇  Meal plans
 ⊟  Catering
 ⊕  Social feed
 ⚙  Settings
```

- **Header** — name + monogram avatar, tappable, routing to the new edit screen.
  The monogram stays charcoal, not coral: identity is not a call to action, and
  coral remains reserved for actions and selection.
- **Three tiles** — Favourites / Wallet / Orders, matching the reference.
- **Nudge cards** — referral and similar promotional entries.
- **List rows** — everything else.

The Personal-Info form **moves wholesale** to a new `app/profile/edit.tsx`. This is
a move, not a rewrite: the schema, `Controller` fields, and submit handler transfer
intact.

**Every existing feature gate is preserved.** The current quick-links grid filters
its entries on flags (`CATERING_ENABLED`, `SOCIAL_ENABLED`, `TIFFIN_ENABLED`,
subscription visibility). Rows carry the identical conditions — the refresh must
not surface a destination the flags currently hide.

---

## 3. Chef menu — sections with a jump sheet

### Problem

`app/chef/[id].tsx:150-177` derives categories and *filters*: selecting a category
replaces the list with only that category's items. The reference renders every
category as a section in one continuous scroll and uses the sheet to **jump**.
Filtering hides the menu; sectioning reveals it. That difference is the point.

### Design

`components/chef/ChefMenuTab.tsx` renders a `SectionList` with one section per
category, plus:

- **Sticky tab bar** — the existing chip row, now tracking scroll position via
  `viewabilityConfig` so the chip for the section in view is highlighted, and
  scrolling to a section when tapped.
- **Floating `≡ Menu` pill** — bottom-centre, above the cart bar.
- **Category sheet** — each category with its item count; tapping calls
  `scrollToLocation`.

```
── sticky ──────────────────────
 Starters  Mains  Breads  Desserts
────────────────────────────────
 Starters
  • Samosa            ₹60
  • Paneer Tikka     ₹220
 Mains
  • Lamb Rogan Josh  ₹340

         ╭──────────╮
         │ ≡  Menu  │   ← floating
         ╰──────────╯
```

### Known risk

`scrollToLocation` on a `SectionList` with variable-height rows lands imprecisely
without `getItemLayout`. Our rows genuinely vary — photo vs no photo, one vs two
description lines — so `getItemLayout` cannot be computed honestly. The mitigation
is `onScrollToIndexFailed` with a retry, which is the standard fix and reliable in
practice, but it is the one part of this work that needs **device testing rather
than a typecheck** to confirm.

---

## Styling

All three follow `.impeccable.md`: coral accent only on actions and selection,
hairline separators rather than bordered cards, tabular numerals on every money
figure, 44px minimum touch targets, `cubic-bezier(0.22, 1, 0.36, 1)` easing with no
bounce, `prefers-reduced-motion` honoured.

## Verification

`tsc --noEmit` for each screen. The menu jump additionally needs an emulator pass —
scroll-to-section accuracy is not something a typecheck can prove.
