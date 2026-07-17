# Local setup — HomeChef on `sandboxctl`

Bring HomeChef up on a local Kubernetes sandbox with
[`sandboxctl`](https://github.com/tesserix/sandboxctl) (kind + Argo CD + Istio +
in-cluster registry/Gitea, on `https://*.sandbox.app:8443`).

> HomeChef is a multi-service monorepo, and **`sandboxctl` deploys one chart =
> one app = one URL**. Deploy the **shared local infra** once (CNPG Postgres +
> Redis + NATS + Mongo for all products), then each HomeChef service. `deploy`
> auto-applies `k8s/secrets.yaml` and auto-picks each chart's `values-local.yaml`.

## 0. Prerequisites (one time)

```sh
command -v sandboxctl >/dev/null || brew install tesserix/tap/sandboxctl
```

Charts live in `../tesserix-k8s/charts/apps` (sibling checkout).

### Deploy the shared local infra FIRST

All products share one set of datastores in the `local-infra` namespace. Bring
the sandbox up with the CNPG operator, then deploy the `local-infra` chart once:

```sh
sandboxctl up --with-cnpg --podman-disk 80 --podman-memory 12g
sandboxctl deploy --chart ../tesserix-k8s/charts/apps/local-infra \
  --name local-infra --no-build
```

This provisions:

- **Postgres** (CNPG) `local-pg-rw.local-infra.svc.cluster.local:5432` — HomeChef
  gets role `homechef` + db `homechef_db`.
- **Redis** `redis.local-infra.svc.cluster.local:6379` — HomeChef owns logical DB `1`.
- **NATS** `nats.local-infra.svc.cluster.local:4222` and **Mongo**
  `mongodb.local-infra.svc.cluster.local:27017` (HomeChef uses neither today).

Products **auto-connect** via stable DNS plus the reflected `local-infra-creds`
Secret (the `local-infra` chart copies it into the `homechef` namespace with the
same throwaway password, `local-sandbox-dev`). No per-product Postgres to deploy.

## 1. Fill in local secrets

```sh
cp k8s/secrets.example.yaml k8s/secrets.yaml      # gitignored
$EDITOR k8s/secrets.yaml
```

`homechef-local-secrets` covers POSTGRES_PASSWORD, REDIS_PASSWORD, JWT secrets,
the auth-bff GIP/session keys, etc. See the GCP Secret Manager mapping in
[`k8s/README.md`](k8s/README.md). Most can stay as throwaway values for non-login
local testing.

## 2. Bring the platform up (first run ≈ 10 min)

Already covered in step 0 (`sandboxctl up --with-cnpg ...` + the `local-infra`
deploy). First `up` ≈ 10 min (image pulls). Later runs are fast.

## 3. Credentials & status

```sh
sandboxctl creds
sandboxctl status
```

## 4. Deploy HomeChef

From the **HomeChef repo root** (`cd /path/to/Home-Chef-App`):

> No local Postgres step — HomeChef connects to the shared `local-infra`
> datastores deployed in step 0 (DB `homechef_db`, Redis logical DB 1) via stable
> DNS and the reflected `local-infra-creds` Secret.

```sh
# 4a. API (Go) — builds from apps/api ; uses the local.enabled gated deployment
sandboxctl deploy --repo apps/api --chart ../tesserix-k8s/charts/apps/homechef-api      --name homechef-api      --purge-old-tags
sandboxctl deploy --repo .        --chart ../tesserix-k8s/charts/apps/homechef-auth-bff  --name homechef-auth-bff  --purge-old-tags

# 4c. Frontends (Vite SPAs, build from monorepo root)
sandboxctl deploy --repo . --chart ../tesserix-k8s/charts/apps/homechef-web             --name homechef-web             --purge-old-tags
sandboxctl deploy --repo . --chart ../tesserix-k8s/charts/apps/homechef-vendor-portal   --name homechef-vendor-portal   --purge-old-tags
sandboxctl deploy --repo . --chart ../tesserix-k8s/charts/apps/homechef-delivery-portal --name homechef-delivery-portal --purge-old-tags
```

URLs: `https://homechef-web.sandbox.app:8443`,
… (after `Synced + Healthy`).

> Adjust each `--repo` to the actual build dir of that service in this monorepo
> (`apps/web`, `apps/vendor-portal`, `apps/delivery-portal`,
> `apps/api`). For a true one-command bring-up, add a root `sandboxctl.yaml`
> multi-image manifest — ask and I'll generate it.

## 5. Dependencies

- **Postgres / Redis / NATS / Mongo** all come from the shared `local-infra`
  deploy in step 0 — nothing extra to install. HomeChef's `values-local.yaml`
  already points at `local-pg-rw.local-infra` (db `homechef_db`) and
  `redis.local-infra` (logical DB 1).
- Dual Keycloak auth won't fully work locally — the overlays make services
  **boot**; full login needs an IdP.

## 6. Reset HomeChef's data (shared infra)

Wipe only HomeChef's slice of the shared datastores (drops+recreates
`homechef_db`, flushes Redis logical DB 1) — other products are untouched:

```sh
../tesserix-k8s/charts/apps/local-infra/clean-product.sh homechef
```

## 7. Keep images & disk clean

```sh
podman system prune -a -f --volumes && podman builder prune -af
```

## 8. Redeploy / tear down

```sh
sandboxctl deploy --repo apps/api --chart ../tesserix-k8s/charts/apps/homechef-api --name homechef-api --purge-old-tags
sandboxctl undeploy --name homechef-api
sandboxctl down
```
