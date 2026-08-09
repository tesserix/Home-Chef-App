# Fe3dr Vendor — App Store screenshots

Captured 2026-08-09 from EAS build `34fd54b4` (`prod-sim` profile, release mode
against production), signed in as `vendor@fe3dr.com` → *Saffron Home Kitchen*.

`ios-6.9/` is 1320 × 2868 (iPhone 17 Pro Max). With `supportsTablet: false`,
6.9" is the only size App Store Connect requires.

| File | Screen | Notes |
|---|---|---|
| `01-dashboard.png` | Dashboard | Live kitchen, real activity, two meal-plan orders awaiting acceptance. |
| `02-orders-history.png` | Orders → History | Every order is from the same customer ("Priya"), and rejected/cancelled rows are visible. Usable, not flattering. |
| `03-menu.png` | Menu | **No dish photos** — every item renders the placeholder glyph. Weak as a hero shot for a food app. |
| `04-earnings.png` | Earnings | Strongest shot. Commission / GST / TDS breakdown reads credibly for India. |
| `05-more.png` | More | Clean; shows the app's breadth. |

## Test mode — resolved 2026-08-09

*Saffron Home Kitchen* was `testMode: true`, which made the Dashboard render a
large amber banner reading *"Orders and earnings shown here are not real, and
customers cannot order from you right now."* That blocked the Dashboard
screenshot **and** was a live Vendor submission risk in its own right — an App
Store reviewer signing into the demo account would have read it as a non-final
build (guideline 2.1).

The kitchen was returned to Live via the admin console (HomeChef → Chefs →
*Return to Live*); verified `mode: live`, `acceptingOrders: true`. The Dashboard
was then recaptured. **Saffron now takes real payments** — if it is ever moved
back to test mode, this screenshot goes stale and the banner returns.

The FSSAI "registration is ready" card was dismissed before capture: its
registration number (`1213232312`) is placeholder-looking seed data and does not
belong in a public store listing.

## Also worth fixing before capture

Menu items on the demo chef have no images. The chef's own profile, banner and
kitchen photos are set, but every dish falls back to a placeholder. For a
food-marketplace listing that undercuts the Menu screenshot — add photos to a
handful of dishes on `vendor@fe3dr.com` first.

## Recapturing

See `docs/store-release/README.md` §9 for the build and capture commands.
Typing via `idb ui text` truncates at `@` and races on longer strings — type
the email one character at a time with a short delay.
