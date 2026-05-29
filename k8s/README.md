# Local kind deploy — HomeChef

This directory holds the ONE local Secret used to run HomeChef on a local
**kind** cluster. It exists only for local/dev. Production credentials come
from GCP Secret Manager via ExternalSecrets and never live here.

## The one Secret

`secrets.example.yaml` defines `homechef-local-secrets` in the `homechef`
namespace. Every HomeChef chart's `values-local.yaml` (in the `tesserix-k8s`
repo) reads from this single Secret.

```bash
cp k8s/secrets.example.yaml k8s/secrets.yaml      # secrets.yaml is gitignored
# edit k8s/secrets.yaml — at minimum set POSTGRES_PASSWORD == DB_PASSWORD
kubectl create namespace homechef --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f k8s/secrets.yaml
```

Key rules:
- `DB_PASSWORD` / `REDIS_PASSWORD` **must equal** the shared local-infra
  password (`local-sandbox-dev`) — HomeChef now connects to the shared
  `local-infra` CNPG Postgres (role `homechef`, db `homechef_db`) and Redis
  (logical DB 1), not a bundled per-product Postgres. The `local-infra` chart
  reflects the same password into this namespace as Secret `local-infra-creds`.
- `BFF_INTERNAL_HMAC_KEY` **must match** between the API and the auth-bff pods.

## Bring up the stack (from the tesserix-k8s repo)

```bash
# 1) Shared local infra (CNPG Postgres + Redis + NATS + Mongo for all products).
#    Deploy ONCE; it provisions the homechef slice and reflects local-infra-creds.
helm upgrade --install local-infra charts/apps/local-infra -n local-infra --create-namespace

# 2) API + frontends (build & kind-load each image first, tagged :local)
helm upgrade --install homechef-api charts/apps/homechef-api \
  -n homechef -f charts/apps/homechef-api/values-local.yaml
helm upgrade --install homechef-web charts/apps/homechef-web \
  -n homechef -f charts/apps/homechef-web/values-local.yaml
helm upgrade --install homechef-admin-portal charts/apps/homechef-admin-portal \
  -n homechef -f charts/apps/homechef-admin-portal/values-local.yaml
helm upgrade --install homechef-vendor-portal charts/apps/homechef-vendor-portal \
  -n homechef -f charts/apps/homechef-vendor-portal/values-local.yaml
helm upgrade --install homechef-delivery-portal charts/apps/homechef-delivery-portal \
  -n homechef -f charts/apps/homechef-delivery-portal/values-local.yaml
helm upgrade --install homechef-auth-bff charts/apps/homechef-auth-bff \
  -n homechef -f charts/apps/homechef-auth-bff/values-local.yaml
```

Build & load an image into kind before installing its chart, e.g.:

```bash
docker build -t homechef-api:local -f apps/api/Dockerfile .
kind load docker-image homechef-api:local
```

## Reach the services

kind has no Kong/Istio ingress by default — use port-forward:

```bash
kubectl -n homechef port-forward svc/homechef-api 8080:8080
kubectl -n homechef port-forward svc/homechef-web 5173:80
kubectl -n homechef port-forward svc/homechef-auth-bff 8090:8080
```

## Wipe / reset local data

HomeChef's data lives in the shared `local-infra` datastores. Reset only
HomeChef's slice (drops+recreates `homechef_db`, flushes Redis logical DB 1);
other products are untouched:

```bash
../tesserix-k8s/charts/apps/local-infra/clean-product.sh homechef
```

## Prod source — GCP Secret Manager mapping

In prod, `tesserix-k8s/external-secrets/prod/homechef/externalsecret.yaml` syncs
these from GCP Secret Manager (project `tesseracthub-480811`):

| Local key | GCP SM secret |
|-----------|---------------|
| GIP_WEB_API_KEY | prod-homechef-gip-web-api-key |
| HOMECHEF_CUSTOMER_CLIENT_SECRET | prod-homechef-customer-client-secret |
| HOMECHEF_BUSINESS_CLIENT_SECRET | prod-homechef-business-client-secret |
| HOMECHEF_INTERNAL_CLIENT_SECRET | prod-homechef-internal-client-secret |
| SESSION_ENCRYPT_KEY | prod-homechef-session-encrypt-key |
| BFF_INTERNAL_HMAC_KEY | prod-homechef-bff-internal-hmac-key |
| HOMECHEF_ADMIN_ALLOWED_EMAILS | prod-homechef-admin-allowed-emails |
| INTERNAL_AUTH_SECRET (otto, if run) | prod-support-platform-otto-internal-auth |

Note: JWT_SECRET / JWT_REFRESH_SECRET / DB_PASSWORD / REDIS_PASSWORD are not
ExternalSecret-backed in prod (DB/Redis come from CNPG/Redis app secrets); set
local throwaway values. Read a prod value:
`gcloud secrets versions access latest --secret=<name> --project=tesseracthub-480811`
