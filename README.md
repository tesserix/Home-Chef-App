# HomeChef

[![HomeChef API](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-api-build.yml/badge.svg?branch=main)](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-api-build.yml)
[![HomeChef Web](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-web-build.yml/badge.svg?branch=main)](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-web-build.yml)
[![HomeChef Admin Portal](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-admin-portal-build.yml/badge.svg?branch=main)](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-admin-portal-build.yml)
[![HomeChef Vendor Portal](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-vendor-portal-build.yml/badge.svg?branch=main)](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-vendor-portal-build.yml)
[![HomeChef Delivery Portal](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-delivery-portal-build.yml/badge.svg?branch=main)](https://github.com/tesserix/Home-Chef-App/actions/workflows/homechef-delivery-portal-build.yml)
[![Security alert](https://github.com/tesserix/Home-Chef-App/actions/workflows/security-alert.yml/badge.svg?branch=main)](https://github.com/tesserix/Home-Chef-App/actions/workflows/security-alert.yml)

Food-delivery platform at [fe3dr.com](https://fe3dr.com) — home chefs cook,
drivers deliver, customers order. Go / Gin backend, four React SPAs,
three Expo mobile apps, all deployed as Knative services on GKE behind
Istio.

---

## Stack at a glance

| Layer          | Tech                                                        |
|----------------|-------------------------------------------------------------|
| Backend API    | Go 1.26.1, Gin, GORM, PostgreSQL 16, Redis 7, NATS 2.10 JS  |
| Web apps       | React 19, Vite 6, Tailwind v4, Radix UI, TanStack Query     |
| Mobile         | Expo (React Native), `@tesserix/native` design system       |
| Auth           | Dual Keycloak (customer + internal realms) via auth-bff     |
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
│   ├── web/                 Customer SPA — fe3dr.com
│   ├── admin-portal/        Internal admin SPA — admin.fe3dr.com
│   ├── vendor-portal/       Chef / vendor SPA — vendors.fe3dr.com
│   ├── delivery-portal/     Driver SPA — delivery.fe3dr.com
│   ├── mobile-customer/     Expo (iOS + Android)
│   ├── mobile-vendor/       Expo (iOS + Android)
│   └── mobile-delivery/     Expo (iOS + Android)
├── packages/                Shared TS libs (if any)
├── docker-compose.yml       Local stack — Postgres, Redis, NATS, API, web
├── pnpm-workspace.yaml      Workspace definition (apps/* + packages/*)
└── .github/workflows/       7 CI workflows + base-image-refresh + security-alert
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
pnpm dev             # @homechef/web       → http://localhost:5173
pnpm dev:vendor      # @homechef/vendor-portal
pnpm dev:admin       # @homechef/admin-portal
pnpm dev:delivery    # @homechef/delivery-portal
pnpm dev:api         # Go backend (also runs in compose, this is for edits)
```

Mobile apps (each opens Expo Dev Tools):

```bash
pnpm dev:customer
pnpm dev:mobile-vendor
pnpm dev:mobile-delivery
```

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

Deploys are driven by the five CI workflows under `.github/workflows/`:

| Workflow                              | Image                                                   | Knative ksvc             |
|---------------------------------------|---------------------------------------------------------|--------------------------|
| `homechef-api-build.yml`              | `ghcr.io/tesserix/home-chef-app/homechef-api`           | `homechef-api`           |
| `homechef-web-build.yml`              | `ghcr.io/tesserix/home-chef-app/homechef-web`           | `homechef-web`           |
| `homechef-admin-portal-build.yml`     | `ghcr.io/tesserix/home-chef-app/homechef-admin-portal`  | `homechef-admin-portal`  |
| `homechef-vendor-portal-build.yml`    | `ghcr.io/tesserix/home-chef-app/homechef-vendor-portal` | `homechef-vendor-portal` |
| `homechef-delivery-portal-build.yml`  | `ghcr.io/tesserix/home-chef-app/homechef-delivery-portal` | `homechef-delivery-portal` |

Each workflow:

1. Builds a multi-stage Docker image against `ghcr.io/tesserix/base-*`.
2. Pushes to GHCR with `main`, `latest`, and `main-<shortsha>` tags.
3. Authenticates to GKE via Workload Identity Federation and
   `kubectl patch ksvc` to roll the Knative service.
4. **Trivy CRITICAL+HIGH gate** (`ignore-unfixed: true`) — a fresh CVE
   fails the run and short-circuits the deploy.
5. Uploads a SARIF report to the GitHub Security tab.

On any build failure, `security-alert.yml` fires via `workflow_run`
and emails the security distribution list (`SMTP_USERNAME` +
`SMTP_PASSWORD` repo secrets required).

The `base-image-refresh.yml` workflow listens for the weekly
`repository_dispatch: tesserix-base-images-updated` event from
[`tesserix/base-docker-images`](https://github.com/tesserix/base-docker-images)
and dispatches every build workflow so base-image CVE fixes propagate
without a per-image PR.

## Release images

`homechef-api-release.yml` and `homechef-web-release.yml` publish
semver-tagged immutable release images (same Trivy gate + SBOM +
attestation).

---

## Domains (Istio VirtualServices)

| Domain                            | Target                                     |
|-----------------------------------|--------------------------------------------|
| `fe3dr.com`, `www.fe3dr.com`      | `homechef-web`                             |
| `vendors.fe3dr.com`               | `homechef-vendor-portal`                   |
| `admin.fe3dr.com`                 | `homechef-admin-portal`                    |
| `delivery.fe3dr.com`              | `homechef-delivery-portal`                 |
| `api.fe3dr.com`                   | `homechef-api`                             |
| `identity.fe3dr.com`              | Keycloak customer realm (`homechef`)       |
| `internal-identity.fe3dr.com`     | Keycloak internal realm (`tesserix-internal`) |

Path prefixes on `fe3dr.com`:

- `/bff/` → customer auth BFF
- `/auth/` → Keycloak auth callback
- `/driver-bff/` → driver auth
- `/api/*` → API service
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
| `SMTP_USERNAME`    | `security-alert.yml` (Gmail)            |
| `SMTP_PASSWORD`    | `security-alert.yml` (Gmail app password) |

WIF (`workload_identity_provider`) and the CI service account are set
as repo variables, not secrets.
