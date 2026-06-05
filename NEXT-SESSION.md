# Next session — pick up where we stopped (2026-06-05 end of day)

## TL;DR

Vendor app is in a working demo state. 4 menu items + 4 orders + 2 reviews + 1 admin approval seeded for chef `mahesh.sangawar@gmail.com` against prod DB. Backend is at commit `6d9f659`, mobile build `build-1780638xxx` is the latest local artifact (uninstalled — you'll install first).

The next BIG thing is **executing PROD-READINESS.md Wave 1** — observability, security, distribution. Plan is committed at repo root; read it before kicking off.

## State as of session end

### Code on `main`
| Commit | Scope |
|---|---|
| `cff75c6` | feat(api): chef earnings breakdown + order detail + menu availability + is_veg + document expiry (W2 of session) |
| `9d18256` | feat(mobile-vendor): onboarding fixes + custom tab bar + 5 chef features |
| `6f21c9c` | chore: drop stray root eas.json, export DietIcon from @homechef/mobile-shared |
| `8f5ceb8` | fix(api): rename :id to :itemId in PUT /chef/menu/items/:id/availability to avoid Gin route param conflict |
| `6d9f659` | fix(api+mobile-vendor): dashboard recentOrders + todayEarnings, orders list envelope shape, isVeg honors backend flag, analytics popularItems shape, isPulling on orders queue/history |

### Uncommitted (you decide whether to keep)
- `apps/mobile-vendor/components/vendor/PendingOrderCard.tsx` — fixed Accept button visibility (was invisible white-on-white due to iOS Pressable array-style bug); switched to object-style with inner View
- `apps/mobile-vendor/app/(tabs)/index.tsx` — wired `onOpenDetail` on Dashboard's PendingOrderCard so tapping the customer/total area opens `/orders/:id`
- These were intended to ship; latest mobile build (`b0tzfsoaw` task) was built AFTER these edits but not installed yet. Install + verify, then commit if good.

### Backend prod state
- `homechef-api-00076` serving 100% traffic (deployed from `6d9f659`)
- `auth-bff` unchanged
- All new endpoints live and tested via curl:
  - `GET /chef/earnings/breakdown?period=week|month|cycle`
  - `GET /chef/orders/:orderId`
  - `GET /chef/documents/expiring?withinDays=30`
  - `PUT /chef/menu/items/:itemId/availability`
  - Modified: `GET /chef/dashboard` (adds `recentOrders`, `todayEarnings`), `GET /chef/orders` (envelope shape: `{orders, total, page, limit}`)

### Seeded prod data for the test chef
- **Chef**: `mahesh.sangawar@gmail.com`, user_id `dcba6316-e735-4fcd-b671-a0165e4c92b8`, chef_id `7fc77bb0-a2ca-400c-aab4-7ca9e9ea4481`, business_name `Test`, state `Maharashtra`, city `Mumbai`, is_verified `true`
- **FSSAI document**: expires 2026-06-17 (banner active at 11 days)
- **4 menu items** with Unsplash food images:
  - Paneer Butter Masala — ₹280, veg
  - Veg Hyderabadi Biryani — ₹220, veg
  - Chicken Biryani — ₹320, non-veg
  - Butter Chicken — ₹340, non-veg
- **1 customer**: Priya Sharma, `priya.testcustomer@example.com`, user_id `00000000-0000-0000-0000-000000000c11`
- **4 orders**:
  - `HC-DEMO-001` delivered 2 days ago — 2 × Paneer Butter Masala, ₹630
  - `HC-DEMO-002` delivered 1 day ago — 1 × Chicken + 1 × Veg Biryani, ₹610
  - `HC-DEMO-003` **pending** (~minutes old) — Chicken Biryani + Paneer Butter Masala, ₹605, special instructions present
  - `HC-DEMO-004` **preparing** (~minutes old) — Butter Chicken, ₹380
- **2 reviews**: 5★ on `HC-DEMO-001`, 4★ on `HC-DEMO-002` (with constructive packaging complaint)
- **1 admin approval**: `info_requested` type=kitchen_onboarding, 4h old, asking for clearer FSSAI photo

## What's working end-to-end on the sim build

- Sign-in via GIP (test chef + password works)
- Onboarding routing: `verified` chefs → `/(tabs)`, `pending_review` → `/pending`, `in_progress` → wizard at next step
- Tab bar: 4 columns evenly distributed, full labels (Dashboard, Orders, Menu, More), persimmon top-line on active tab, locked font scaling
- Dashboard: FSSAI banner, action queue card with Reject/Accept (Accept readability fix uncommitted — see above)
- Menu: 4 items with veg/non-veg icons (paneer + veg biryani = green per backend `is_veg`), in-stock Switch with optimistic toggle
- Earnings: breakdown math against 2 delivered orders (gross ₹1,240, commission ₹165, CGST+SGST ₹14.85+₹14.85, TDS ₹12.40, net ₹1,062.60)
- Order detail screen: customer call/sms, items with diet icons, delivery address → Maps, timing, pricing
- Analytics: no longer crashes (renders "X% of orders" instead of broken `revenue.toLocaleString`)

## Known issues that did NOT make this session

(all in PROD-READINESS.md Wave 2)
- **Per-line order cancellation** (chef discovers mid-prep that 1 of N items can't be made) — backend + mobile both need work. Was bumped to a proper feature in the Wave 2 plan today.
- **FSSAI expiry capture in upload form** — backend accepts `expiryDate`, mobile form has no date picker. Manual seed only for now.
- **Doc renewal screen for verified chefs** — no path to re-upload an expired doc in-app today
- **Admin-requests inbox UI** — endpoint exists (`GET /chef/admin-requests`), mobile doesn't consume it yet. Seeded approval is invisible to the chef.
- **Order cancellation by chef** after accept (any flavor)
- **Menu item image upload UI** — backend endpoint exists, mobile form has no picker
- **Reviews list / inbox**

## iOS Pressable bug — pay attention

We hit this 5+ times across the session. **`<Pressable style={({pressed}) => [styles.x, ...]}>` (function returning an ARRAY) silently drops layout props (flex, backgroundColor, padding, alignSelf) on iOS.** Use OBJECT-style return OR wrap content in an inner `<View>`. See `~/.claude/projects/-Users-Mahesh-Sangawar-personal-tesserix-new-Home-Chef-App/memory/feedback_ios_pressable_array_style.md`. The latest bite was the Accept button (the uncommitted PendingOrderCard fix).

## Operational gotchas (still true from prior session)

- Use `npx eas-cli@18.4.0` — 20.x silently fails simulator builds (exit 0, no artifact)
- Always `cd apps/mobile-vendor` before running eas-cli; if cwd is repo root, eas-cli creates a stray root `eas.json` missing the prod-sim profile
- Sim UUID: `AD109A46-2F99-43C3-8AAA-FEE68DC8499E`, bundle `com.homechef.vendor`, location pinned to Koramangala
- Kargo overrides `argocd/prod/apps/homechef/*.yaml` image tags — don't roll back via manifest edits, use ArgoCD UI Rollback or push a revert
- kube context for prod GKE: `gke_tesseracthub-480811_asia-south1_tesseract-prod-in-gke`
- Postgres pod: `homechef-postgres-10`, ns `homechef`. Password via `kubectl get secret homechef-postgres-app-credentials -o jsonpath='{.data.password}' | base64 -d`
- Backend `chef_documents` and `menu_items` got new columns via AutoMigrate — `expiry_date`, `is_veg`. Verified live.
- 82 Dependabot vulnerabilities, Trivy gate still warn-only per locked decision. Wave 1 will flip the gate.

## First three actions next session

1. **Install** the latest mobile build (`b0tzfsoaw` artifact at `apps/mobile-vendor/build-*.tar.gz` — newest) and verify the Accept button + dashboard tap-to-detail
2. **Commit** the two uncommitted mobile fixes (`PendingOrderCard.tsx` + `(tabs)/index.tsx`) — suggested message: `fix(mobile-vendor): Accept button visibility (iOS Pressable array bug) + dashboard pending cards now tap into order detail`
3. **Read PROD-READINESS.md** at repo root. We agreed: 8-week public launch, you + agents executing, 4 waves of 2 weeks. Wave 1 starts with TestFlight pipeline, Sentry, rate limiting, idempotency, Dependabot sweep. Dispatch the backend + mobile subagents per Wave 1's parallelization breakdown.

## Suggested next-session prompt

```
Continuing the Home Chef vendor app from 2026-06-05. Read NEXT-SESSION.md, PROD-READINESS.md, and the two memory files (feedback_no_editorial_style + feedback_ios_pressable_array_style) before any work.

State:
- Backend at homechef-api-00076 (commit 6d9f659), all session endpoints live
- Seeded chef data in prod for mahesh.sangawar@gmail.com (4 menu items, 4 orders, 2 reviews, 1 admin approval, FSSAI expiring in ~10 days)
- 2 uncommitted mobile fixes pending verify + commit (Accept button visibility, dashboard tap-to-detail)
- Latest local mobile build is the newest build-*.tar.gz in apps/mobile-vendor/, NOT yet installed on sim AD109A46

First three actions:
1. Install latest mobile build + smoke test Accept button + tap-to-detail on dashboard
2. Commit the two pending fixes
3. Kick off PROD-READINESS.md Wave 1 — TestFlight pipeline, Sentry on mobile+backend, rate limiting + idempotency, Dependabot sweep, Trivy gate flip. Dispatch backend + mobile subagents per wave's parallelization plan.

Operational gotchas — see NEXT-SESSION.md "Operational gotchas" section. Specifically: npx eas-cli@18.4.0, cd apps/mobile-vendor before eas, Kargo overrides manifest tags, GKE context gke_tesseracthub-480811_asia-south1_tesseract-prod-in-gke for kubectl.
```
