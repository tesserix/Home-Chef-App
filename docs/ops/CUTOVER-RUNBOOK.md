# Home Chef — Platform Integration & Cutover Runbook

Owner-controlled steps that finish Waves **5A** (web sunset), **5B** (tesserix-home
integration), and **7A** (own-fleet retirement). These run against GCP / Cloudflare /
`tesserix-k8s` — they are **not** app code in this repo. The codeable prerequisites
are already merged (see each section).

**Concrete Home Chef facts** (verified from `tesserix-k8s/charts/apps/homechef-api/values-prod.yaml`):

| Thing | Value |
|---|---|
| Namespace | `homechef` |
| CNPG cluster | `homechef-postgres` (rw svc `homechef-postgres-rw.homechef.svc.cluster.local`) |
| Database | `homechef_db` (single DB) |
| App role (owner) | `homechef` |
| GCP project | `tesseracthub-480811` |
| Hosts | `fe3dr.com` (landing), `vendors.fe3dr.com` (vendor web + **mobile API/auth**), `delivery.fe3dr.com` (driver web), `admin.fe3dr.com` (admin) |

---

## 1. tesserix-home DB access — `homechef_platform_admin` role (5B)

tesserix-home reads/writes Home Chef data **directly** as a least-privilege CRUD role
(no DDL) — the mark8ly pattern. Full reference + rationale:
`tesserix-k8s/docs/cross-db-admin.md` → "Adding a new product".

**Already done (this repo):** the payout DDL (`WeeklyStatement.status/paidAt/payoutRef`,
commit `63753bd`) — the admin role can't `ALTER`, so the owning service added the columns.

**To provision (one-time, against prod):**

```bash
PROJECT=tesseracthub-480811
PRODUCT=homechef
DB=homechef_db
APP_ROLE=homechef
SECRET_KEY=prod-homechef-platform-admin-password

# 1. Generate + store the role password (32 ASCII chars, UTF-8-safe).
PWD=$(openssl rand -base64 24 | tr -d '\n')
echo -n "$PWD" | gcloud secrets create $SECRET_KEY \
  --project=$PROJECT --data-file=- --replication-policy=automatic

# 2. Create the role on the homechef-postgres primary.
PG_POD=$(kubectl -n $PRODUCT get pods -l cnpg.io/cluster=${PRODUCT}-postgres,role=primary -o name | head -1)
kubectl -n $PRODUCT exec $PG_POD -c postgres -- psql -U postgres -d postgres -v ON_ERROR_STOP=1 -c "
  DO \$\$
  BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${PRODUCT}_platform_admin') THEN
      CREATE ROLE ${PRODUCT}_platform_admin LOGIN PASSWORD '$PWD'
        NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
    ELSE
      ALTER ROLE ${PRODUCT}_platform_admin WITH PASSWORD '$PWD';
    END IF;
  END
  \$\$;
"

# 3. Grants on homechef_db. Note the ALTER DEFAULT PRIVILEGES FOR ROLE homechef —
#    homechef's tables are created/owned by the `homechef` app role (GORM
#    AutoMigrate), so future tables must auto-inherit from THAT owner too.
kubectl -n $PRODUCT exec $PG_POD -c postgres -- psql -U postgres -d $DB -v ON_ERROR_STOP=1 -c "
  GRANT CONNECT ON DATABASE $DB TO ${PRODUCT}_platform_admin;
  GRANT USAGE ON SCHEMA public TO ${PRODUCT}_platform_admin;
  GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ${PRODUCT}_platform_admin;
  GRANT USAGE, UPDATE ON ALL SEQUENCES IN SCHEMA public TO ${PRODUCT}_platform_admin;
  ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ${PRODUCT}_platform_admin;
  ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, UPDATE ON SEQUENCES TO ${PRODUCT}_platform_admin;
  ALTER DEFAULT PRIVILEGES FOR ROLE ${APP_ROLE} IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ${PRODUCT}_platform_admin;
  ALTER DEFAULT PRIVILEGES FOR ROLE ${APP_ROLE} IN SCHEMA public
    GRANT USAGE, UPDATE ON SEQUENCES TO ${PRODUCT}_platform_admin;
"
unset PWD
```

**Then in `tesserix-k8s` (PRs, ArgoCD-synced):**
- [x] `charts/thirdparty/istio-config/values.yaml` — `homechef` added under `crossProductNamespaceAccess.tesserix` (egress tesserix → homechef:5432). **DONE** (`643ee706`).
- [x] `charts/apps/company/` — `HOMECHEF_DB_*` env staged behind `database.homechef.enabled` (default **false**). **DONE** (`39d7f807`). Renders nothing until flipped, so it's safe on `main`.
- [x] `external-secrets/prod/tesserix/externalsecret.yaml` — add a `homechef-platform-admin` ExternalSecret pointing at `prod-homechef-platform-admin-password` (mirrors the `mark8ly-platform-admin` block: `type: kubernetes.io/basic-auth`, `username: "homechef_platform_admin"`, `password` from the GCP key). **DONE 2026-06-12** — added in tesserix-k8s `external-secrets/prod/tesserix/externalsecret.yaml`.
- [x] **Flip the switch:** set `database.homechef.enabled: true` in `charts/apps/company/values.yaml` — **in the SAME change as the ExternalSecret above** (so the pod's `secretKeyRef` resolves). This is what actually wires the tesserix-home pod to homechef_db. **DONE 2026-06-12** — `database.homechef.enabled: true` in `charts/apps/company/values.yaml`.
- [x] Insert a row into `tesserix-postgres.tesserix_admin.apps` (product slug `homechef`, DB host `homechef-postgres-rw.homechef.svc.cluster.local`, db `homechef_db`, role-secret name, admin URL) — drives the Apps grid tile. **DONE 2026-06-12** — row inserted into `tesserix_admin.apps`.
- [x] ~~Cut a tesserix-home release tag~~ **Not needed — corrected 2026-06-12:** tesserix-home deploys via `ci.yml` on push to `main` (build + set image SHA); `release.yml` only tags GHCR on `v*.*.*` tags. The homechef admin pages shipped with the normal main-branch deploy.

**Then in tesserix-home (the 5B session):** add `lib/db/homechef.ts` (a `pg.Pool` reading
`HOMECHEF_DB_*`, mirroring `lib/db/mark8ly.ts`) and the `homechef` `ProductConfig`. Payout
list / mark-paid / commission-config / CSV are **direct DB** through that pool — never the
homechef-api. Every mutating call follows the cross-db-admin write conventions (txn + audit row).

---

## 2. Secret management from tesserix-home (5B)

tesserix-home talks to **GCP Secret Manager directly** via `@google-cloud/secret-manager`
(already in its stack) — GCP SM is platform infra, not product-owned, so **no homechef-api
endpoint** (an earlier endpoint attempt was reverted). Implement in the 5B session:

- Curated allowlist of GCP secret IDs (do NOT allow arbitrary creation):
  Razorpay test/live key-id + key-secret + webhook-secret, SendGrid API key, Firebase
  service account, JWT secret. **Confirm each ID against the real GCP SM names + the
  homechef ESO `remoteRefs`** before enabling.
- Write path: `addSecretVersion` (set/rotate). Status: `getSecretVersion` on `latest`
  (set-vs-unset + `createTime` for "last rotated") — **never `accessSecretVersion`** for
  status (don't fetch the value).
- Hard requirements: platform-super-admin only; audited (**never log the value**); UI
  write-only / masked.
- **Propagation:** new GCP SM version → ESO sync → k8s secret → env var → **pod rollout
  required** for env-injected secrets to take effect. Surface this in the UI. (This is the
  mechanism for the Wave-6 Razorpay `test→live` switch + the Shadowfax keys.)

---

## 3. Web + own-fleet production cutover (5A / 7A)

**Already done (this repo):** `SUNSET.md` on `vendor-portal` / `delivery-portal` /
`mobile-delivery`; web-portal CI removed; own-driver API routes retired (`303e543`, Wave 7A).

**Sequencing:** flip DNS/routing **only after both store listings are live** — a landing with
dead store links is worse than the current apps. Then:

- [ ] **Retire ArgoCD apps:** `homechef-vendor-portal`, `homechef-delivery-portal` (and confirm `homechef-web` → landing is fully cut over). Prune the `charts/apps/homechef-{vendor,delivery}-portal` apps.
- [ ] **Cloudflare 301s:** `vendors.fe3dr.com/*` (web paths) and `delivery.fe3dr.com/*` → `fe3dr.com`.
  > ⚠️ **Guardrail:** `vendors.fe3dr.com` is the **mobile vendor app's API + auth-bff host**.
  > 301 only the **web UI paths** — the `/api/*` and auth-bff routes on that host MUST keep
  > serving. Verify with a device login + dashboard fetch after the flip (smoke test).
  > Confirm nothing API-critical rides `delivery.fe3dr.com` before 301'ing it.
- [ ] **auth-bff registry cleanup** (do this LAST, after the above): remove the `web`,
  `vendor-portal`, `delivery-portal` blocks from `apps/auth-bff/homechef-products.yaml`;
  keep `admin-portal` + the `mobileTenantAllowlist`. Left in place until now on purpose
  (deployments were live; mobile vendor auth on `vendors.fe3dr.com`).
- [ ] **mobile-delivery:** unpublish any EAS build / store listing if one exists.

---

## 4. Infra hardening checklist (Waves 1/3/4 operational tail)

- [ ] **Cloud SQL automated backups + restore drill** — verify retention on the
  `homechef-postgres` CNPG cluster (or Cloud SQL if applicable), run a non-prod restore,
  document RTO/RPO. (CNPG: confirm `backup` + scheduled `Backup`/`ScheduledBackup` objects.)
- [ ] **Cloudflare WAF** — managed ruleset on `*.fe3dr.com` + a custom rate rule for
  `/auth/auto-login` brute force. Needs Cloudflare **Pro** plan.
- [ ] **Trivy gate flip** — flip the CI scan from warn-only to fail-on-critical+high once
  the remaining critical/high deps are cleared (GitHub Dependabot currently reports 37;
  triage before flipping or the gate blocks all builds).

---

## Revert (DB role)

```sql
REVOKE ALL PRIVILEGES ON DATABASE homechef_db FROM homechef_platform_admin;
DROP OWNED BY homechef_platform_admin CASCADE;
DROP ROLE homechef_platform_admin;
```
Then delete the GCP SM key, remove the ExternalSecret, and drop `homechef` from
`crossProductNamespaceAccess`. See `tesserix-k8s/docs/cross-db-admin.md`.
