# Home Chef — Incident Runbook

Top-5 incident types for the launch window. Each section: **detect → diagnose → mitigate → verify**.

## Environment quick-reference

| Thing | Value |
|---|---|
| Prod GKE context | `gke_tesseracthub-480811_asia-south1_tesseract-prod-in-gke` |
| Namespace | `homechef` |
| API (Knative) | `homechef-api` ksvc — min-scale 1, max-scale 10 |
| Auth BFF (Deployment) | `homechef-auth-bff` — 2 replicas |
| DB pooler | `homechef-postgres-pooler-rw` / `-ro` |
| Public host | `vendors.fe3dr.com` |
| DB | Cloud SQL `db-f1-micro` (shared, per-service DBs) |
| Image bump | CI `bump-k8s` → `tesserix-k8s` `argocd/prod/apps/homechef/homechef-api.yaml` |
| Tracing | Cloud Trace (filter by `X-Request-ID` / `trace_id` in logs) |
| Errors | Sentry project `homechef-api` |

**Never** `kubectl edit`/`patch` prod image tags — ArgoCD self-heals. Roll back via ArgoCD UI **Rollback** or revert the bump commit in `tesserix-k8s`.

---

## 1. auth-bff down (logins failing)

**Detect:** Login spins / 502 on `auth.fe3dr.com`; Sentry auth errors; `kubectl get deploy homechef-auth-bff` shows < 2 ready.

**Diagnose:**
```
kubectl --context <prod> -n homechef get deploy homechef-auth-bff
kubectl --context <prod> -n homechef logs deploy/homechef-auth-bff --tail=200 | grep -iE "error|panic|gip|secret"
```
Common causes: GIP credentials/secret rotation, BFF HMAC key mismatch with the API, OOM/crashloop.

**Mitigate:**
- Crashloop → `kubectl rollout restart deploy/homechef-auth-bff -n homechef`.
- Bad deploy → ArgoCD UI **Rollback** to the previous healthy revision.
- Secret issue → confirm the ExternalSecret synced; re-sync ESO.

**Verify:** `curl -sI https://auth.fe3dr.com/health` → 200; perform a real login.

---

## 2. DB connection pool exhausted

**Detect:** API 500s with "too many connections" / context-deadline; latency spike across all endpoints; Sentry DB errors.

**Diagnose:** Each API pod caps at `DB_MAX_OPEN_CONNS` (default **20**). Cloud SQL `db-f1-micro` total cap ≈ 100 across **all** clients (API pods + auth-bff + crons + migrations).
```
kubectl --context <prod> -n homechef get pods -l serving.knative.dev/service=homechef-api   # how many API pods are up
# In Cloud SQL → Monitoring → active connections.
```
Cause: pod count × pool > Postgres cap (e.g. a scale-up burst), or a leaked/long transaction.

**Mitigate:**
- Immediate: lower `DB_MAX_OPEN_CONNS` (e.g. 12) via the API ExternalSecret/env and restart, or cap `max-scale` lower temporarily.
- Kill stuck sessions in Cloud SQL if a query is hung.
- Persistent load → upgrade Cloud SQL tier (budget decision).

**Verify:** active connections back under cap; p95 latency normal.

---

## 3. Razorpay (payments) outage

**Detect:** Checkout/refund failures; `502 from gateway`; reconciliation cron logs `DriftGatewayUnreachable`.

**Diagnose:**
```
kubectl --context <prod> -n homechef logs <api-pod> | grep -iE "razorpay|refund|gateway"
```
Confirm scope at the Razorpay status page — is it them or us (creds/secret)?

**Mitigate:**
- Their outage → orders that don't need instant capture can proceed; surface a "payments temporarily unavailable" state. Do **not** mark orders paid without a gateway confirmation.
- Refunds that failed mid-flight: the order keeps `refund_amount`; the **reconciliation cron** flags drift next run. Reconcile manually from the Razorpay dashboard once service returns.
- Creds issue → re-sync the payment secret; `InvalidateRazorpay` happens on next secret fetch.

**Verify:** a test capture + refund succeed; reconciliation clean next morning.

---

## 4. GIP (Google Identity Platform) outage

**Detect:** All new logins fail across apps (auth-bff healthy but token verification errors); Google status page incident.

**Diagnose:** auth-bff logs show GIP token-exchange / JWKS errors. This is upstream — nothing to restart.

**Mitigate:**
- Existing sessions (encrypted JWT cookies) keep working — **don't** force logouts.
- Post a status-page notice (logins temporarily affected).
- Do not rotate GIP config during the incident.

**Verify:** new login works once Google recovers.

---

## 5. Knative scale-from-zero stuck / cold-start

**Detect:** First request after idle hangs or 503s. **Note:** `homechef-api` runs **min-scale 1**, so this should be rare — investigate if it happens.

**Diagnose:**
```
kubectl --context <prod> -n homechef get ksvc homechef-api
kubectl --context <prod> -n homechef get revisions -l serving.knative.dev/service=homechef-api
kubectl --context <prod> -n homechef get pods -l serving.knative.dev/service=homechef-api
```
Look for a revision stuck `Unschedulable` (Autopilot capacity), image pull error, or failing readiness probe.

**Mitigate:**
- Image pull → confirm the GAR-mirrored tag exists; re-trigger the bump if the tag is missing.
- Bad revision not going Ready → ArgoCD **Rollback**.
- Confirm `autoscaling.knative.dev/min-scale: "1"` is still present on the template (a warm pool prevents cold starts **and** keeps the cron jobs running).

**Verify:** `curl -s https://vendors.fe3dr.com/api/v1/mobile/min-version?platform=ios&app=vendor` → 200.

---

## Cron health (settlement, FSSAI, reconciliation, auto-resume)

All crons run in-process on `homechef-api` (reliable under min-scale 1; each also fires on startup).
```
kubectl --context <prod> -n homechef logs <api-pod> | grep -iE "weekly-statement|fssai-reminder|reconciliation|availability-resume"
```
- `reconciliation DRIFT` lines → a payment/refund mismatch needs finance review (also raised to Sentry).
- No `weekly-statement: scan complete` after a Monday → check the pod was up over the weekend.
