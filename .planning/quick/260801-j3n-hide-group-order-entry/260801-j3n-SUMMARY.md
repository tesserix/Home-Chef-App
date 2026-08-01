---
gsd_summary_version: 1.0
quick_id: 260801-j3n
slug: hide-group-order-entry
status: complete
date: 2026-08-01
issue: 875
branch: fix/hide-group-order-entry
commit: 60d09004
---

# Quick Task 260801-j3n — Summary

Closes tesserix/Home-Chef-App#875.

## What changed

`apps/mobile-customer/lib/features.ts` — `GROUP_ORDERS_ENABLED` flipped
`true` → `false`, with the doc comment rewritten to say why it's off and what
the flag does *not* gate.

One line of behaviour, ten lines of comment. That ratio is deliberate: the next
person to read this needs to know that flipping it back on requires the
split-pay flow to be live, and that joining an existing group was never gated.

## Task outcomes

**Task 1 — flip the flag.** Done.

**Task 2 — prove no other entry point is reachable.** Done, and it confirmed
the flag flip is sufficient. Full sweep of
`group-order|/group/|useGroupOrder|GroupOrder` across `app/`, `components/`,
and `hooks/` produced these classes:

- **Gated (a):** `chef/[id].tsx:126` `startGroupOrder` — the only initiator.
  Reached solely via `onStartGroupOrder` passed to `ChefMenuTab` (`[id].tsx:634`),
  which renders inside the `GROUP_ORDERS_ENABLED` check at `ChefMenuTab.tsx:141`.
  No second call site.
- **Must stay reachable (b):** `app/group/[code].tsx` (invite deep-link landing),
  `app/group-order/[id].tsx` (hub), `payment/checkout.tsx:199` and
  `payment/cashfree.tsx:144` (post-payment redirects into an existing group).
- **Ungated entry points (c):** none.

`hooks/useGroupOrder.ts` and `hooks/useConfirmReceived.ts` are data layer — no
UI entry point, left alone.

**Task 3 — typecheck.** `npx tsc --noEmit` in `apps/mobile-customer`: 1 error,
`lib/payment.ts(114,7) TS2345` (expo-router path typing). **Pre-existing** —
verified by stashing the change and re-running: 1 error before, 1 error after.
Not introduced here, not fixed here.

## Not done / deliberately out of scope

- No routes, hooks, or API calls deleted. This is a visibility gate.
- The pre-existing `payment.ts` route-typing error was left alone — unrelated
  to this issue and fixing it would widen a one-line PR.
- Not visually verified on the simulator yet. The change is a boolean read by
  an existing, already-wired conditional, so the risk is low, but the chef menu
  bottom spacing with the row gone has not been eyeballed.

## Follow-ups

- #877 (footer Close button on sheets) is the other open customer-app UI issue.
