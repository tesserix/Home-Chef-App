---
slug: web-parity-to-mobile
created: 2026-07-28
mode: quick
branch: feat/web-parity-customer-orders
follows: 260727-web-password-and-social-fixes
---

# Bring the web apps up to mobile (mobile is the source of truth)

Owner direction 2026-07-28: **mobile is the source of truth; web lags and catches up.**
That inverts the two earlier passes (#802 removed web-only surfaces, #803 repaired broken
ones). This one *ports* mobile features into web. Customer first, then vendor.

## How the backlog was derived

Not by reading screens — by diffing call sites: every `/v1/...` endpoint referenced in
`apps/mobile-customer` (107 distinct) against every `apiClient.*` path in `apps/web/src`
(85). **28 are mobile-only.**

**Use the script below, not a shell grep.** Three naive versions each overstated the gap
and would have had us rebuild features web already has:

1. Matching only `get|post|put|patch|delete` missed `apiClient.upload` and
   `apiClient.getBlob` — hid that web already implements `report-issue` and `/reviews`.
2. Single-line matching missed calls whose path sits on the next line — hid
   `POST /payments/order/:id/tip`, which web's TipPage does call.
3. `<[^>]*>` for generics broke on nesting like `<PaginatedResponse<SocialPost>>` — hid
   `/social/feed` and `/chefs`.

Always run the sanity probes; if a known-present endpoint reports missing, the regex is
wrong, not the code.

```python
import re, os
web_re = re.compile(r'apiClient\.\w+[^(\n]*\(\s*[`\'"]([^`\'"]+)', re.S)  # [^(]* skips generics
mob_re = re.compile(r'[`\'"](/v1/[^`\'"]+)')
def scan(root, rx):
    out = set()
    for d, _, fs in os.walk(root):
        if 'node_modules' in d: continue
        for f in fs:
            if f.endswith(('.ts', '.tsx')):
                out.update(rx.findall(open(os.path.join(d, f), encoding='utf8').read()))
    return out
def norm(p):
    p = re.sub(r'\$\{[^}]*\}', ':p', p); p = re.sub(r'^/v1', '', p)
    return p.split('?')[0].rstrip('/') or '/'
web = {norm(p) for p in scan('apps/web/src', web_re)}
mob = {norm(p) for p in scan('apps/mobile-customer', mob_re)}
skip = re.compile(r'chef-1|plan-9|/p1/|^/\.\.\.|^/$|…|\$\{')       # test fixtures
for probe in ('/social/feed', '/chefs', '/payments/order/:p/tip', '/orders/:p/report-issue'):
    assert probe in web, f'regex broken: {probe} is present in web'
print(sorted(m for m in mob if m and not skip.search(m) and m not in web))
```

Re-run it to check progress — it is the definition of "done" here.

## Already parity — do not "fix"

- `report-issue`, `/reviews` — web has both (via `apiClient.upload`).
- **Invoice.** Mobile opens a signed URL (`GET /orders/:id/invoice-link`); web downloads
  the file (`GET /orders/:id/invoice.pdf`, `apiClient.getBlob`). Both routes exist in the
  API. This is correct platform divergence, not lag — leave it.
- Tip — web calls `POST /payments/order/:id/tip` then `/payments/tip/:tipId/verify`.

## Do NOT port (dark on mobile too)

Checked before scoping; porting these would build features nobody can use:

- **Messaging** — `/customer/orders/:id/messages`. `MESSAGING_ENABLED = false` in BOTH
  `apps/web/src/shared/config/features.ts` and `apps/mobile-customer/lib/features.ts`.
  `order/[id]/index.tsx:761` gates the entry point, and the comment says the API 503s
  until infra is ready. Port it gated, or not at all — do not ship it live.
- **Catering** — `/catering/requests/*`. `CATERING_ENABLED = false` on both sides. Web
  already has `catering` + `catering/quotes` routes behind the same flag.

All 8 feature flags are otherwise **exactly aligned** web↔mobile (verified by diff), so
gating decisions carry across unchanged.

## The 22 portable gaps

28 mobile-only, minus 5 skipped (messaging ×1, catering ×4) and 1 non-gap (invoice-link).

### Chunk 1 — order lifecycle (this branch)

| Endpoint | Note |
|---|---|
| `POST /orders/:id/confirm-received` | **Ties into payout escrow.** Customer confirmation releases the chef's hold. Web customers cannot confirm at all, so every web order waits out the 24h auto-confirm cron (`payout_auto_confirm_cron.go`). Not broken — slower, and the customer has no way to speed it up or see why. Highest value in the whole backlog. |
| `POST /group-orders/:id/confirm-received` | Same, host-side, for group orders. |
| `GET /orders/:id/track` | Live tracking — see chunk 2, split out deliberately. |

### Chunk 2 — live tracking

`GET /orders/:id/track` plus the WS feed (`useOrderTrackingWS`). Its own chunk because
mobile renders a map + bottom sheet and **web has no map dependency today**. Decide the web
map, and whether web tracking is a map or a status timeline, before starting.

### Chunk 3 — discovery & address

`/search/dishes`, `/chefs/:id/daily-menu`, `/chefs/:id/fulfillment-times`,
`/locations/postcodes/search`. `#809`/`#813` already touched web address search — check
what landed before rebuilding.

### Chunk 4 — meal plans / tiffin (largest)

`/meal-plans`, `/meal-plans/:id`, `/cancel`, `/verify-payment`,
`/meal-plans/:id/days/:d/{confirm-received,refund-medium,skip}`,
`/customer/meal-plan-refund-choices`, `/meal-subscriptions/:id/fulfillments`,
`/tiffin/confirm-today`. `TIFFIN_ENABLED = true` both sides — live and lagging.

### Chunk 5 — account & safety

`/blocks` (+`:id`) blocked accounts, `/support/chat`, `/reports`,
`/customer/onboarding/status`, plus the security / profile-edit / food-preferences /
account-paused / EULA screens.

## Then: vendor

`vendor-portal` vs `mobile-vendor` — catering (`/chef/catering/{requests,quotes,bookings}`,
web has **zero** catering routes), payout bank details, document renewal, notification
preferences, language (EN/हिन्दी), support tickets, legal, chef agreement, daily menu,
refund decisions, review detail, admin-request detail.

## Explicitly NOT touched: vendor `/subscriptions`

The one place web is **ahead** of mobile. `SubscriptionSetupPage` → `PUT
/chef/subscription-config` is the only UI that writes `enabled` and `perMealPrice`, and
`ValidateMealSelection` gates the customer offer on it (`meal_subscription.go:53`,
`ErrMealSubNotConfigured`). `apps/mobile-vendor` never calls it — verified exhaustively.

Deleting it "for consistency" would leave nobody able to switch tiffin on while mobile
customers already see a Plans tab. Left in place; **mobile-vendor should gain this screen**
— that is a mobile task, not a web one.
