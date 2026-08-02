---
name: fe3dr
description: Working knowledge of the fe3dr / HomeChef platform — the customer app, the chef (vendor) app, the Go API, and the Otto support stack. Use when navigating or changing either mobile app, answering "where does X live", wiring support/Otto behaviour, driving the iOS simulators for an end-to-end check, or shipping any of it to prod.
---

# fe3dr / HomeChef

A home-chef food-delivery marketplace: hungry customer → home cook → driver.
Production domain `fe3dr.com`. Everything below is verified against the repo,
not inferred from names.

**The live products are the three Expo apps.** `apps/web`, `apps/vendor-portal`
and `apps/delivery-portal` are sunset (see `SUNSET.md` in each). There is no
admin app in this repo — operator surfaces live in `../tesserix-home` under
`apps/web/app/admin/apps/homechef/`, talking to this repo's `/admin/*` API.

## Repo layout

| Path | What it is |
|---|---|
| `apps/api` | Go 1.26 + Gin + GORM, PostgreSQL. The only backend. |
| `apps/mobile-customer` | Expo customer app. `Fe3dr`, `com.tesserix.homechef.customer` |
| `apps/mobile-vendor` | Expo chef app. `Fe3dr Vendor`, `com.tesserix.homechef.vendor` |
| `apps/mobile-delivery` | Expo driver app |
| `apps/mobile-admin` | Separate mobile admin client |
| `apps/auth-bff` | Auth backend-for-frontend |
| `apps/web-landing` | Marketing site |

Sibling repos: `../tesserix-k8s` (all Helm/ArgoCD/SQL), `../tesserix-home`
(admin UI), `../slm-support-platform` (Otto).

## Customer app — `apps/mobile-customer`

56 routes. Tabs: **Home · Orders · Plans · Saved · Profile**.

Account lives under **Profile**:

- Profile → Wallet (`/wallet`), Rewards (`/loyalty`), Referral (`/referral`)
- Profile → Food preferences (`/profile/preferences`), ChefBook (`/chefbook`)
- Profile → Catering (`/catering`), My meal plans (`/meal-plans`),
  My subscriptions (`/subscriptions`)
- Profile → Download my data (`/data-privacy`), Blocked accounts,
  **Help & support** (`/support-chat`), Legal
- Profile → Change password, Pause or delete account (`/data-privacy`)

Key flows:

- **Order** — browse chefs → `chef/[id]` → `cart` → `checkout` →
  `payment/checkout` → `payment/cashfree` → `payment/result`
- **Post-order** — `order/[id]/` with `track`, `receipt`, `messages`,
  `report-issue`, `review`, `tip`
- **Meal plans** — `book-meal-plan`, `meal-plans/[id]`,
  `meal-plans/refund-choices`, `meal-subscription/[chefId]`
- **Group** — `group/[code]`, `group-order/[id]`
- Also: `chefs-map`, `search-dishes`, `refund`, `security`, `notifications`

Onboarding: `user-info` → `address` → `preferences`.

## Chef app — `apps/mobile-vendor`

57 routes. Tabs: **Dashboard · Orders · Menu · More**. Almost everything
else hangs off **More**, grouped:

- **Kitchen** — Meal plans (`/meal-plans`), Capacity (`/capacity`),
  Catering (`/catering`), Reviews (`/reviews`), ChefBook (`/chefbook`)
- **Money** — Earnings (`/earnings`), Payout (`/payout`),
  Rewards (`/rewards`), Expenses (`/expenses`), Analytics (`/analytics`)
- **Requests** — Cancellations (`/cancel-requests`),
  Admin requests (`/admin-requests`), Documents (`/documents/renew`)
- **Account** — Profile, Notifications (`/notification-preferences`),
  Language, Settings, Help & support (`/support`), Legal

**The two screens everyone confuses.** Get this right or you will send a chef
to a screen that cannot do what you told them:

- **More → Payout** — sets the *bank account* earnings are paid into
  (account number, IFSC).
- **More → Earnings** — shows payouts and transactions that *already
  happened*.

There is no screen called "Payouts". Saying so is a real bug — it is what
made Otto's answers wrong before the vendor KB namespace was split out.

Onboarding, in order: `personal-info` → `kitchen-details` → `operations` →
`documents` → `payout` → `policies` → `review` → `pending`.
A kitchen cannot take orders until verified; expiring documents surface at
More → Documents and More → Admin requests.

Support lives at `app/support/` — `index`, `new`, `chat`, `[id]`.

## Otto — the support assistant

Lives in `../slm-support-platform`, not here.

- `services/otto` — conversations, messages, SSE, feedback (MongoDB)
- `services/slm-router` — the agent loop: retrieve → rerank → infer → post
- `services/slm-inference` — Qwen 2.5 1.5B on CPU. Small: it retrieves well
  and answers accurately when the KB covers the question, but multi-turn
  diagnostic reasoning is at the edge of what it does reliably.

**Tenants are separate.** `homechef` (customer) and `homechef-vendor` (chef)
each have their own `ragNamespace` and their own system prompt. They were
one aliased tenant until 2026-08-02; sharing meant chefs got customer-app
navigation. Do not re-merge them.

| Thing | Where |
|---|---|
| KB chunks | `tesserix-k8s/charts/apps/support-platform-kb-seed/values.yaml` |
| System prompts | `tesserix-k8s/.../support-platform-slm-router/templates/configmap-prompts.yaml` |
| Tenant routing | `tesserix-k8s/.../support-platform-slm-router/values.yaml` |
| Universal response rules | `slm-router/internal/orchestrator/prompt.go` |

Editing a KB chunk re-embeds it on the next kb-seed run — the Job is
keyed on the ConfigMap hash, so bump the `*-kb-meta` revision string to
force a re-run.

**Otto never talks over a person.** `processOne` reads conversation
ownership first and stays silent when there is an assignee or
`needs_human` is set (`orchestrator.go`, `ConversationState.HumanOwned()`).
A state-lookup failure answers anyway — a lost reply beats an overlap.

## Driving the simulators

Bundle IDs above. Simulator UDIDs are per-machine — look them up, never
reuse one from a previous session:

```bash
xcrun simctl list devices booted
```

Debug builds bake their Metro port in: **customer → 8082, vendor → 8081**.
A wrong port silently loads another project's bundle.

```bash
U=<udid>
xcrun simctl launch $U com.tesserix.homechef.vendor
idb ui tap  --udid $U <x> <y>          # points, not pixels
idb ui text --udid $U "some text"
xcrun simctl io $U screenshot out.png
```

Screenshots come back at 3× on a Pro Max: divide pixel coordinates by 3 to
get tap points. The floating Expo dev-tools gear sits top-right and will
swallow taps aimed near it.

**End-to-end support check:** vendor app → More → Help & support → Chat with
support → pick a topic (this reveals a Summary field) → fill Summary *and*
Message → Start chat. Otto replies in ~30–60s. Confirm the decision from the
router rather than the UI:

```bash
p=$(kubectl get pods -n support-platform -o name | grep slm-router | head -1)
kubectl logs $p -n support-platform --since=5m \
  | grep -E "assistant reply posted|human owns|staying silent"
```

## Shipping

Never run container builds, pushes or deploys — verify with language tooling
and hand the deploy command over.

```bash
cd apps/api && go build ./... && go vet ./... && go test ./...
helm template t charts/apps/<chart>        # for tesserix-k8s edits
```

- **CI** — every repo except `tesserix-k8s` is private with limited Actions
  minutes: make public → push → wait for green → make private again.
  `tesserix-k8s` is permanently public; just push.
- **Images** — Kargo watches the registry on a 5-minute poll and promotes the
  newest `main-<sha>`; ArgoCD syncs. No manual tag bump.
- **Schemas** — all SQL lives in `tesserix-k8s`, never in this repo. App repos
  hold ORM models only.
- **Never** `kubectl apply/patch/edit`. Change `tesserix-k8s`, push, let
  ArgoCD sync. Read-only `kubectl logs/get/describe` is fine.

## Gotchas

- Grepping this repo for an admin UI returns nothing. That is expected, not
  a missing feature — check `tesserix-home`. This has been mis-diagnosed at
  least twice, including issue #876.
- `apps/admin-portal` still exists on disk but holds only stale build
  artifacts; the source was deleted.
- Design tokens: the mobile Tailwind token is still named `herb` but renders
  persimmon `#C2410C`. Renaming it would touch 49 files. See `.impeccable.md`.
