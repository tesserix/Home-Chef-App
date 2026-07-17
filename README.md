# HomeChef

[![HomeChef API](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-api-build.yml/badge.svg?branch=main)](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-api-build.yml)
[![HomeChef Auth BFF](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-auth-bff-build.yml/badge.svg?branch=main)](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-auth-bff-build.yml)
[![HomeChef Web Landing](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-web-landing-build.yml/badge.svg?branch=main)](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-web-landing-build.yml)

Food-delivery platform at [fe3dr.com](https://fe3dr.com) — home chefs cook,
drivers deliver, customers order. Go / Gin backend, a Next.js marketing site
(`web-landing`), two React ops SPAs (vendor / delivery), and three
Expo mobile apps — the customer storefront is mobile-first for now (the
`apps/web` customer SPA is temporarily **disabled** / paused, planned to return;
see `apps/web/SUNSET.md`). All deployed as Knative services on GKE behind Istio.

---

## Stack at a glance

| Layer          | Tech                                                        |
|----------------|-------------------------------------------------------------|
| Backend API    | Go 1.26.1, Gin, GORM, PostgreSQL 16, Redis 7, NATS 2.10 JS  |
| Web apps       | React 19, Vite 8, Tailwind v4, Radix UI, TanStack Query (ops SPAs); Next.js for `web-landing` |
| Mobile         | Expo (React Native), `@tesserix/native` design system       |
| Auth           | Google Identity Platform (GIP) via `apps/auth-bff` (3 tenant pools) |
| Payments       | Razorpay Route (split payments)                             |
| Messaging      | NATS JetStream — `orders.*`, `chef.*`, `delivery.*`, etc.   |
| Storage        | GCS (images), GCP Secret Manager (secrets)                  |
| Container base | `ghcr.io/tesserix/base-*` (Trivy-gated weekly rebuilds)     |
| Deploy         | GHCR → GKE (`homechef` namespace, Knative + Istio + ArgoCD) |

---

## Repo layout

```
Home-Chef-App/
├── apps/
│   ├── api/                 Go / Gin backend (port 8080)
│   ├── web/                 Customer SPA — DISABLED / paused (planned to return; see apps/web/SUNSET.md)
│   ├── web-landing/         Next.js marketing site — fe3dr.com
│   ├── vendor-portal/       Chef / vendor SPA — vendors.fe3dr.com
│   ├── delivery-portal/     Driver SPA — delivery.fe3dr.com
│   ├── mobile-customer/     Expo (iOS + Android) — customer storefront
│   ├── mobile-vendor/       Expo (iOS + Android)
│   └── mobile-delivery/     Expo (iOS + Android)
├── packages/                Shared TS libs (mobile-shared, etc.)
├── docker-compose.yml       Local stack — Postgres, Redis, NATS, API
├── pnpm-workspace.yaml      Workspace definition (apps/* + packages/*)
└── .github/workflows/       7 workflows (5 build/release + dependabot-auto-merge + base-image-refresh)
```

Each app has its own `Dockerfile` — all built from the monorepo root
with `pnpm --filter` (frontends) or their own `go.mod` (api).

---

## Prerequisites

- Node.js ≥ 22
- pnpm ≥ 9
- Go 1.26.1
- Docker + Docker Compose
- `gcloud` CLI (for pulling `prod-ghcr-token` from GCP Secret Manager)

## Install

The frontends depend on `@tesserix/web` which is published to GitHub
Packages, so install needs a read-scoped token:

```bash
NODE_AUTH_TOKEN=$(gcloud secrets versions access latest \
    --secret=prod-ghcr-token --project=tesseracthub-480811) \
  pnpm install
```

## Local dev

One-shot full-stack via Docker Compose (Postgres, Redis, NATS, API):

```bash
docker compose up -d
```

Then run the frontends directly so Vite HMR works:

```bash
pnpm dev:landing     # @homechef/web-landing  → http://localhost:5173 (fe3dr.com marketing)
pnpm dev:vendor      # @homechef/vendor-portal
pnpm dev:delivery    # @homechef/delivery-portal
pnpm dev:api         # Go backend (also runs in compose, this is for edits)
```

The customer storefront is the `mobile-customer` Expo app (see below) for now —
the `pnpm dev` / `apps/web` customer SPA is temporarily disabled (paused, planned
to return), so it is not built or deployed at the moment.

Mobile apps (each opens Expo Dev Tools):

```bash
pnpm dev:customer
pnpm dev:mobile-vendor
pnpm dev:mobile-delivery
```

### Mobile auth (GIP / Firebase) setup — one-time

Each mobile app authenticates against Google Identity Platform via
`@react-native-firebase/auth` and `@react-native-google-signin/google-signin`.
Before you can run any mobile app you need:

1. **Per-app env vars.** Copy `apps/mobile-{customer,vendor,delivery}/.env.example`
   to `.env.local` in the same directory and fill in:
   - `EXPO_PUBLIC_BFF_URL` — the auth-bff base URL (local: `http://localhost:8081`)
   - `EXPO_PUBLIC_GIP_TENANT_ID` — already set per-app in the example file
   - `EXPO_PUBLIC_GIP_API_KEY` — `gcloud secrets versions access latest --secret=prod-homechef-gip-web-api-key`
   - `EXPO_PUBLIC_GOOGLE_WEB_CLIENT_ID` and `EXPO_PUBLIC_GOOGLE_IOS_CLIENT_ID` —
     from GCP Console → APIs & Services → Credentials → `home-chief-web`.

2. **Firebase config files** (one per app per platform). These are NOT in the
   repo. From the Firebase Console (project `tesseracthub-480811`):
   - Download `google-services.json` and place at
     `apps/mobile-{customer,vendor,delivery}/google-services.json` (Android).
   - Download `GoogleService-Info.plist` and place at
     `apps/mobile-{customer,vendor,delivery}/GoogleService-Info.plist` (iOS).
   - Reference both in each app's `app.config.ts` under the `android` /
     `ios` blocks.

3. **EAS builds.** Expo Application Services does not read `.env*` files at
   build time. Mirror every `EXPO_PUBLIC_*` var into the build profile's
   `env` block in each `eas.json`. Firebase config files can be committed
   (they contain no real secrets) or injected via `eas secrets`.

## Lint / typecheck / test

```bash
pnpm lint              # ESLint across every workspace
pnpm typecheck         # tsc --noEmit across every TS workspace
pnpm test              # vitest across every workspace (frontends)
cd apps/api && go test ./...   # backend tests
```

Formatting is Prettier + `prettier-plugin-tailwindcss`; run
`pnpm format` before committing.

---

## Deployment

Deploys are driven by the build workflows under `.github/workflows/`:

| Workflow                              | Image                                                   | Knative ksvc             |
|---------------------------------------|---------------------------------------------------------|--------------------------|
| `homechef-api-build.yml`              | `ghcr.io/tesserix/home-chef-app/homechef-api`           | `homechef-api`           |
| `homechef-auth-bff-build.yml`         | `ghcr.io/tesserix/home-chef-app/homechef-auth-bff`      | `homechef-auth-bff`      |
| `homechef-web-landing-build.yml`      | `ghcr.io/tesserix/home-chef-app/homechef-web-landing`   | `homechef-web` (cutover slot) |

> `apps/web` (the customer SPA) is temporarily **disabled** / paused — its CI
> build is currently turned off (the `homechef-web-build.yml` /
> `homechef-web-release.yml` workflows aren't active right now) and `fe3dr.com`
> is served by `web-landing` for now. The app code is kept and it's planned to
> return. The `vendor-portal` and `delivery-portal` SPAs do not yet have
> dedicated build workflows in this repo.

Each workflow:

1. Builds a multi-stage Docker image against `ghcr.io/tesserix/base-*`.
2. Pushes to GHCR tagged `<branch>` and `main-<shortsha>` (no `:latest` —
   Kargo selects the newest matching tag from the mirror).
3. On push to `main`, the `bump-k8s` job commits the new image tag (+
   `updateTimestamp`) into `argocd/prod/apps/homechef/homechef-<svc>.yaml`
   in [`tesserix-k8s`](https://github.com/tesserix/tesserix-k8s); Argo CD
   (via the `kargo-homechef` prod Stage) rolls the Knative service. No
   manual `kubectl apply`.
4. **Trivy CRITICAL+HIGH gate** (`ignore-unfixed: true`) — a fresh CVE
   fails the run and short-circuits the deploy.
5. Uploads a SARIF report to the GitHub Security tab.

The `base-image-refresh.yml` workflow listens for the weekly
`repository_dispatch: tesserix-base-images-updated` event from
[`tesserix/base-docker-images`](https://github.com/tesserix/base-docker-images)
and dispatches every build workflow so base-image CVE fixes propagate
without a per-image PR.

## Release images

`homechef-api-release.yml` publishes semver-tagged immutable release images
(same Trivy gate + SBOM + attestation).

---

## Domains (Istio VirtualServices)

| Domain                            | Target                                     |
|-----------------------------------|--------------------------------------------|
| `fe3dr.com`, `www.fe3dr.com`      | `homechef-web`                             |
| `vendors.fe3dr.com`               | `homechef-vendor-portal`                   |
| `admin.fe3dr.com`                 | `homechef-admin-portal` (retired — app moved to the Tesserix admin; deployment still serves its last image) |
| `delivery.fe3dr.com`              | `homechef-delivery-portal`                 |
| `api.fe3dr.com`                   | `homechef-api`                             |

Path prefixes on `fe3dr.com` (and the other portal domains):

- `/bff/` → `homechef-auth-bff` (session bootstrap, used by SPA fetch proxy)
- `/auth/*` → `homechef-auth-bff` (GIP exchange, auto-login, session, logout, refresh, csrf)
- `/api/*` → `homechef-api`
- `/ws/*` → WebSocket (3600s timeout)

---

## Infra

- **PostgreSQL 16** — `postgresql.postgresql-homechef.svc:5432`, db
  `homechef_db`, 60Gi, 300 max connections.
- **Redis** — `redis.redis-homechef.svc:6379`, 4Gi, auth enabled
  (session store).
- **Cloudflare Tunnel** — token from GCP Secret Manager
  (`prod-homechef-cloudflare-tunnel-token`).
- **External Secrets** — `homechef-api-secrets`, `homechef-auth-bff-secrets`
  synced from GCP Secret Manager.
- **DB schema bootstrap** — CronJob in `tesserix-k8s`
  (`charts/apps/db-schema-bootstrap/schemas/homechef/`). Runs every
  30 min, idempotent.
- **GCP Service Account** —
  `app-secrets-homechef-prod@tesseracthub-480811.iam.gserviceaccount.com`.

All Helm charts and ArgoCD apps live in
[`tesserix-k8s`](https://github.com/tesserix/tesserix-k8s) under
`charts/apps/homechef-*` and `argocd/prod/apps/homechef/`.

---

## Required repo secrets

| Secret             | Used by                                 |
|--------------------|-----------------------------------------|
| `PKG_READ_TOKEN`   | Installing `@tesserix/web` from GHCR    |
| `NPM_TOKEN`        | BuildKit secret mount in every frontend Dockerfile |

WIF (`workload_identity_provider`) and the CI service account are set
as repo variables, not secrets.
