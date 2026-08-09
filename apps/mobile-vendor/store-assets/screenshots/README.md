# Fe3dr Vendor — App Store screenshots

Captured 2026-08-09 from EAS build `34fd54b4` (`prod-sim` profile, release mode
against production), signed in as `vendor@fe3dr.com` → *Saffron Home Kitchen*.

`ios-6.9/` is 1320 × 2868 (iPhone 17 Pro Max). With `supportsTablet: false`,
6.9" is the only size App Store Connect requires.

| File | Screen | Notes |
|---|---|---|
| — | Dashboard | **Missing — blocked.** See below. |
| `02-orders-history.png` | Orders → History | Every order is from the same customer ("Priya"), and rejected/cancelled rows are visible. Usable, not flattering. |
| `03-menu.png` | Menu | **No dish photos** — every item renders the placeholder glyph. Weak as a hero shot for a food app. |
| `04-earnings.png` | Earnings | Strongest shot. Commission / GST / TDS breakdown reads credibly for India. |
| `05-more.png` | More | Clean; shows the app's breadth. |

## The dashboard shot is blocked by test mode

*Saffron Home Kitchen* is `testMode: true`, so the Dashboard renders a large
amber banner:

> **TEST MODE** — The Fe3dr team is running tests on your kitchen to investigate
> an issue. **Orders and earnings shown here are not real, and customers cannot
> order from you right now.** … Reference: test session 1

Two consequences:

1. The Dashboard — the single most important vendor screenshot — cannot be
   captured until the kitchen is live.
2. More seriously, **an App Store reviewer signing into the demo account sees
   this banner**, which reads as a non-final build (guideline 2.1). This makes
   taking the kitchen live a blocker for the Vendor submission itself, not just
   for screenshots.

Fix, then re-capture the Dashboard:

```
PATCH /admin/chefs/e150c72a-42e2-4beb-8cb1-389666dd813c/mode
{"mode": "live", "reason": "App Store review"}
```

Requires an internal-pool admin (`RequirePool(PoolInternal)` + `RequireAdmin`).
Note this makes the kitchen take **real** payments.

## Also worth fixing before capture

Menu items on the demo chef have no images. The chef's own profile, banner and
kitchen photos are set, but every dish falls back to a placeholder. For a
food-marketplace listing that undercuts the Menu screenshot — add photos to a
handful of dishes on `vendor@fe3dr.com` first.

## Recapturing

See `docs/store-release/README.md` §9 for the build and capture commands.
Typing via `idb ui text` truncates at `@` and races on longer strings — type
the email one character at a time with a short delay.
