# Wave 1 — Infrastructure Track Runbook

**Date:** 2026-06-05
**Wave window:** Weeks 1–2 of the 8-week production-readiness push (ship target ~2026-07-31)
**Owner:** Mahesh (solo dev, executing by hand, optionally delegating prep to subagents)
**Scope:** 3 infra items from `PROD-READINESS.md` Wave 1 → Infra

---

## 1. Purpose & scope

Covers the three infra items in `PROD-READINESS.md` Wave 1:

1. Cloud SQL automated backups + restore drill
2. Always-warm pod floor on `homechef-auth-bff` (kills the cold-start 503 race)
3. Cloudflare WAF rules on `*.fe3dr.com` (managed ruleset + custom rate-limit on `/auth/auto-login`)

Not application code. Spans **three venues**:

| Venue | Used for |
|---|---|
| `Home-Chef-App/` (this repo) | Docs only — this runbook + captured snapshots. No live manifests here. |
| `tesserix-k8s/` sibling | Helm charts + ArgoCD apps for `homechef-*` (auth-bff Deployment + KEDA ScaledObject) |
| `tesserix-infra/` sibling | Terraform for Cloud SQL (`terraform/02-core/cloud-sql.tf`) |
| GCP console + `gcloud` | Cloud SQL retention edits + restore drill |
| Cloudflare dashboard | WAF managed ruleset + custom rate-limit rule |

Reader executes by hand. Subagents prep YAML diffs and command lists; they cannot push, `gcloud`, or click — credentials live with the human.

**Architectural correction discovered while drafting:** parent CLAUDE.md describes the Tesserix platform as Knative-everywhere, but `homechef-auth-bff` is actually a vanilla `apps/v1 Deployment` with a KEDA `ScaledObject` (see `tesserix-k8s/charts/apps/homechef-auth-bff/templates/{deployment,scaledobject}.yaml`). PROD-READINESS's "`min_scale: 1` (Knative annotation)" wording is wrong for Home Chef. The equivalent lever is `keda.minReplicaCount` in the chart's `values.yaml`. Runbook honours the intent (always-warm pod), not the literal annotation.

---

## 2. Prerequisites

- [ ] `gcloud config get-value project` → `tesseracthub-480811`
- [ ] `kubectl config current-context` → `gke_tesseracthub-480811_asia-south1_tesseract-prod-in-gke` (per NEXT-SESSION.md)
- [ ] Both sibling repos cloned: `/Users/Mahesh.Sangawar/personal/tesserix-new/{tesserix-k8s,tesserix-infra}` (both confirmed present at draft time)
- [ ] Push access on `github.com/tesserix/tesserix-k8s` and `tesserix-infra`
- [ ] ArgoCD UI access at `argocd.tesserix.app` (needed for Sync + Rollback; bypasses Istio per parent CLAUDE.md)
- [ ] Cloudflare dashboard access for `fe3dr.com` zone, `Zone Admin` role
- [ ] Cloudflare plan tier visible in Overview — **Managed WAF requires Pro ($20/mo); upgrade is a precondition for Task 3**

---

## 3. Inventory snapshot to capture before any change

Capture into `docs/superpowers/specs/2026-06-05-wave1-infra-snapshot.md` (create as you go). Without it, you cannot prove what changed or roll back cleanly.

```bash
# Cloud SQL current backup config
gcloud sql instances describe tesserix-main --project tesseracthub-480811 \
  --format='value(settings.backupConfiguration.enabled,
                  settings.backupConfiguration.pointInTimeRecoveryEnabled,
                  settings.backupConfiguration.transactionLogRetentionDays,
                  settings.backupConfiguration.backupRetentionSettings.retainedBackups,
                  settings.backupConfiguration.startTime)'
gcloud sql backups list --instance tesserix-main --project tesseracthub-480811 --limit 20

# Auth-bff state
kubectl -n homechef get deployment homechef-auth-bff -o yaml > /tmp/authbff-deploy-before.yaml
kubectl -n homechef get scaledobject homechef-auth-bff -o yaml > /tmp/authbff-keda-before.yaml
kubectl -n homechef get pods -l app.kubernetes.io/name=homechef-auth-bff -o wide

# ArgoCD revision (for rollback target)
# UI: argocd.tesserix.app → homechef-auth-bff → History and Rollback → screenshot

# Cloudflare current state — dashboard, fe3dr.com zone:
#   Overview          → plan tier (Free / Pro / Business)
#   Security → WAF → Managed Rules   → enabled rulesets
#   Security → WAF → Custom rules    → list
#   Security → WAF → Rate limiting   → list
```

Known from reading the repos (skip re-discovery):

- `tesserix-infra/terraform/02-core/cloud-sql.tf`: backups enabled, PITR on, `transaction_log_retention_days = 7`, `retained_backups = 14`, `start_time = "03:00"`. PROD-READINESS warned about the 7-day default; current is already 14 but below the 30 it targets. Decision in Task 1.
- `tesserix-k8s/charts/apps/homechef-auth-bff/values.yaml`: `keda.minReplicaCount: 2` already set, `pdb.minAvailable: 1` already set. The literal "0 → 1" change is already in place. What is missing is **proof** — a load test showing cold-start 503s are gone. Task 2 reframes around evidence, not configuration.

---

## 4. Task-by-task runbook

### Task 1 — Cloud SQL automated backups + restore drill

**Goal:** prove the backup → restore loop end-to-end; capture real RTO and RPO numbers.

**Venue:** Terraform (`tesserix-infra/terraform/02-core/cloud-sql.tf`, resource `google_sql_database_instance.main`, block `backup_configuration` ~line 31) + `gcloud` for the drill. GCP console for browsing: GCP → SQL → `tesserix-main` → Backups.

**Pre-change check** (also in §3): `gcloud sql instances describe tesserix-main ...` → expected today `14   7`.

**Decision — retention:** current is 14 daily + 7 PITR. Bump to 30 daily + 14 PITR. 230 MB DB × 30 incremental ≈ < $1/mo extra; pre-approved by cost-discipline guardrail.

**Change procedure:**

1. Edit `tesserix-infra/terraform/02-core/cloud-sql.tf`:
   ```hcl
       backup_configuration {
         enabled                        = true
         point_in_time_recovery_enabled = true
         start_time                     = "03:00"
   -    transaction_log_retention_days = 7
   +    transaction_log_retention_days = 14
         backup_retention_settings {
   -      retained_backups = 14
   +      retained_backups = 30
         }
       }
   ```

2. From `tesserix-infra/terraform/02-core/`:
   ```bash
   terraform plan -out wave1-backup-bump.tfplan
   terraform apply wave1-backup-bump.tfplan
   ```
   If the plan shows any "destroy" or "replace" on the instance, **STOP** — `deletion_protection = true` should block this, but treat any destroy line as a hard stop and re-read before applying.

3. On-demand backup (so the drill runs against fresh, known data):
   ```bash
   gcloud sql backups create --instance tesserix-main \
     --project tesseracthub-480811 \
     --description "wave1-pre-restore-drill-$(date -u +%Y%m%dT%H%M%SZ)"
   ```

4. Restore drill — **clone to temp instance**, never restore-in-place:
   ```bash
   BACKUP_ID=$(gcloud sql backups list --instance tesserix-main \
       --project tesseracthub-480811 --limit 1 --format='value(id)')
   gcloud sql instances clone tesserix-main tesserix-restore-drill \
       --project tesseracthub-480811 --backup-id "$BACKUP_ID"
   # 5–15 min for 230 MB.
   ```

5. Validate the clone:
   ```bash
   gcloud sql instances describe tesserix-restore-drill \
     --project tesseracthub-480811 --format='value(state)'  # wait until RUNNABLE
   ```
   Connect via Cloud SQL Auth Proxy or a workload pod, run `SELECT schemaname, relname, n_live_tup FROM pg_stat_user_tables ORDER BY n_live_tup DESC LIMIT 20;` and spot-check `chefs`, `orders`, `order_items` row counts vs prod.

6. Record in snapshot doc:
   - **RTO** = wall-clock from clone start to schema-verified connection. Expect 8–15 min on db-f1-micro.
   - **RPO** = backup-only worst case ≈ 24h; PITR with transaction logs ≈ 5 min. Document both.

7. Tear down (clone inherits `deletion_protection`):
   ```bash
   gcloud sql instances patch tesserix-restore-drill \
     --project tesseracthub-480811 --no-deletion-protection
   gcloud sql instances delete tesserix-restore-drill \
     --project tesseracthub-480811 --quiet
   ```

**Validation:**

```bash
# Backup retention now 30
gcloud sql instances describe tesserix-main --project tesseracthub-480811 \
  --format='value(settings.backupConfiguration.backupRetentionSettings.retainedBackups)'
# Expected: 30

# Most-recent backup is the on-demand one you just created
gcloud sql backups list --instance tesserix-main --project tesseracthub-480811 --limit 1
# Expected description: wave1-pre-restore-drill-<timestamp>

# Restore drill clone deleted
gcloud sql instances list --project tesseracthub-480811 --filter="name~tesserix-restore-drill"
# Expected: empty
```

**Rollback:** retention bump is non-destructive. If you must revert, `git revert` the Terraform diff and re-apply; existing extra backups prune naturally. On-demand backup self-prunes. Clone destroyed in step 7 — if you forgot, repeat step 7 (billing accrues until deleted).

**Estimated effort:** 2.5–3.5 hours.

**Risks + mitigations:**
- Terraform plan tries to replace the instance → `deletion_protection` blocks it; STOP at step 2 if you see a destroy line.
- Clone fails on private-network → check `gcloud sql operations list` for the actual error.
- Forgetting to delete the clone → ~$10/mo on db-f1-micro. Calendar reminder for 24h post-drill.

---

### Task 2 — Auth-bff "always one warm pod" guarantee

**Goal:** guarantee ≥ 1 ready `homechef-auth-bff` pod at all times; prove cold-start 503 race is gone with a reproducible load test.

**Venue:** `tesserix-k8s/charts/apps/homechef-auth-bff/values.yaml` (block `keda`, lines 14–19). Rollout via ArgoCD UI on `argocd.tesserix.app`. Argo app file `argocd/prod/apps/homechef/homechef-auth-bff.yaml` — **do NOT edit `image.tag` here, Kargo manages it** (NEXT-SESSION.md §gotchas).

**Pre-change check:**
```bash
kubectl -n homechef get scaledobject homechef-auth-bff \
  -o jsonpath='{.spec.minReplicaCount}{"  "}{.spec.maxReplicaCount}{"\n"}'   # expected: 2  10
kubectl -n homechef get pods -l app.kubernetes.io/name=homechef-auth-bff      # expected: 2 Running 1/1
```

**Discovery:** `keda.minReplicaCount: 2` is already in `values.yaml`. Floor exists. **Remaining gap is operational proof, not configuration.**

**Change procedure:**

1. **Reconcile drift.** If live `ScaledObject` shows `< 2`, force ArgoCD UI → `homechef-auth-bff` → **Sync**. Re-verify with the pre-change kubectl command.

2. **(Optional, recommended)** harden values.yaml to make the floor explicit + commented for future readers:
   ```yaml
   keda:
     enabled: true
     pollingInterval: 15
     cooldownPeriod: 300
     minReplicaCount: 2   # Wave 1 floor — never cold-start the login path
     maxReplicaCount: 10
   ```
   Commit + push to `tesserix-k8s`:
   ```bash
   cd /Users/Mahesh.Sangawar/personal/tesserix-new/tesserix-k8s
   git checkout -b wave1/authbff-warm-floor
   git add charts/apps/homechef-auth-bff/values.yaml
   git commit -m "chore(homechef-auth-bff): document warm-pod floor for Wave 1 infra"
   git push -u origin wave1/authbff-warm-floor
   ```
   Open PR, merge. ArgoCD auto-syncs. Kargo only manages `image.tag` overrides in `argocd/prod/apps/homechef/homechef-auth-bff.yaml`, NOT chart values — safe to merge directly. The Kargo gotcha (NEXT-SESSION.md) only bites on rollback via direct manifest edits, see below.

3. **Run the load test** from a workstation outside the cluster:
   ```bash
   hey -z 60s -q 5 -c 5   https://auth.fe3dr.com/healthz > /tmp/authbff-baseline.txt
   hey -z 30s -q 20 -c 20 https://auth.fe3dr.com/healthz > /tmp/authbff-burst.txt
   ```
   Expect: 0 × 503 in baseline; ≤ 1% × 503 in burst. Capture both files into the snapshot doc.

**Validation:**
```bash
kubectl -n homechef get scaledobject homechef-auth-bff -o jsonpath='{.spec.minReplicaCount}'   # 2
kubectl -n homechef get deployment   homechef-auth-bff -o jsonpath='{.status.readyReplicas}'   # >=2
kubectl -n homechef get pdb          homechef-auth-bff -o jsonpath='{.spec.minAvailable}'      # 1
# Plus: zero 503s in load-test baseline window — paste into snapshot doc
```

**Rollback (priority order):**
1. **Preferred — ArgoCD UI Rollback.** App → History and Rollback → previous revision. Required path per NEXT-SESSION.md gotcha (Kargo fights direct manifest edits on `image.tag`).
2. If the rollback is values-only (no image change), `git revert` on `tesserix-k8s/main` and push. ArgoCD picks it up in ≤ 3 min via selfHeal.
3. **Do NOT edit `argocd/prod/apps/homechef/homechef-auth-bff.yaml` directly** — Kargo silently re-overwrites it next cycle.

**Estimated effort:** 1.5–2.5 hours.

**Risks + mitigations:**
- Drift between `values.yaml` and live ScaledObject → caught by §3 pre-check; ArgoCD selfHeal corrects within 3 min.
- Load test trips Cloudflare rate-limit → run Task 2 BEFORE Task 3 (sequencing §5 enforces this).
- KEDA controller down → PDB `minAvailable: 1` keeps ≥ 1 pod during rolling deploys.

---

### Task 3 — Cloudflare WAF on `*.fe3dr.com`

**Goal:** enable Cloudflare Managed + OWASP rulesets on `fe3dr.com` plus one custom rate-limit on `POST /auth/auto-login` to brake brute-force before it reaches `auth-bff`.

**Venue:** Cloudflare dashboard, `fe3dr.com` zone. No repo edits — Cloudflare config is not in Terraform today (see Open Questions §8).

**Pre-change check:** captured in §3 inventory (plan tier, current WAF state, custom rules, rate-limit rules). **If plan is Free, STOP — Managed WAF requires Pro ($20/mo, pre-approved).**

**Change procedure (all dashboard clicks on `fe3dr.com` zone):**

1. **Upgrade plan if needed.** Plans → Pro → confirm (per-zone billing).

2. **Managed Ruleset.** Security → WAF → Managed Rules → **Cloudflare Managed Ruleset** → Enable. Actions: `Managed Challenge` for medium/high, `Block` for critical (defaults). Save.

3. **OWASP Core.** Same screen → **OWASP Core Ruleset** → Enable. Paranoia **PL1**, Score threshold **25**, Action **Block**. Save.

4. **Custom rate-limit rule.** Security → WAF → Rate limiting rules → **Create rule**.
   - Name: `wave1-authbff-auto-login-bruteforce`
   - Expression: `(http.request.method eq "POST" and http.request.uri.path eq "/auth/auto-login")`
   - Rate: `10 requests per 1 minute` per `IP address`
   - Action: `Block` for `1 hour` (default 429 response)
   - Save → Deploy.

5. **Smoke test:**
   ```bash
   for i in $(seq 1 12); do
     curl -s -o /dev/null -w "%{http_code}\n" \
       -X POST https://auth.fe3dr.com/auth/auto-login -d '{"test":true}'
   done
   ```
   Expect: first 10 return whatever the app returns (likely `401`), requests 11+ return `429` for an hour. Clear early via rule's "Disable temporarily".

**Validation:**
- Dashboard → Security → Events, filter Action `block` + `managed_challenge` → events flow within 5 min.
- Curl smoke test returns `429` on requests 11+.
- `homechef-auth-bff` logs show NO additional `/auth/auto-login` POSTs after the 10th in that minute (proves edge blocked, not app).
- `https://auth.fe3dr.com/healthz` at human pace returns `200`, no false-positive challenge.

**Rollback:**
- Single rule: Rate limiting rules → toggle off OR Delete.
- Managed Ruleset: Managed Rules → toggle off (~30s globally).
- Plan downgrade: Plans → Free. Disable rules first to stop enforcement sooner.

**Estimated effort:** 1.5–2 hours.

**Risks + mitigations:**
- False positives on chef logins → PL1 is lowest paranoia; if events appear, drop offending rule action to `Managed Challenge` or add an exception expression.
- Forgotten load test trips rate-limit → temp "Skip" rule for the test IP. Sequencing §5 also handles this.
- Pro plan billing $20/mo — pre-approved.
- No Terraform — capture all rules in snapshot doc for rebuild if zone is lost.

---

## 5. Sequencing

Strict order:

1. **Task 1 (Cloud SQL)** — most reversible; highest info value (you learn real RTO/RPO); independent of auth path.
2. **Task 2 (auth-bff warm floor + load test)** — must come **before** Task 3 because the load test would otherwise trip the Task 3 rate-limit rule.
3. **Task 3 (Cloudflare WAF)** — dashboard-only, cost decision, highest false-positive risk on live chef traffic. Doing it last means tasks 1 and 2 are already proven before any user-facing edge change.

Buffer ≥ 30 min between tasks to read kubectl pod state + Cloudflare Security Events and confirm no regression.

---

## 6. Acceptance criteria mapped to PROD-READINESS Wave 1 DoD

| DoD item | Verifying command | Pass condition |
|---|---|---|
| Cloud SQL backups verified via restore drill | Task 1 steps 4–7 | Drill completed; RTO + RPO captured in snapshot doc |
| Cold-start 503 on auth-bff reproducibly gone (load test) | `hey -z 60s -q 5 -c 5 https://auth.fe3dr.com/healthz` (Task 2 step 3) | 0 × 503 in baseline; ≤ 1% in burst |
| Cloudflare WAF on `*.fe3dr.com` (implicit) | Task 3 smoke-test curl loop | Request 11+ returns `429`, confirmed in Security Events |

Sentry-panic-30s DoD and backend rate-limit-429 test are owned by the backend track, not this runbook.

---

## 7. Subagent prep checklist

| Task | CAN | CANNOT |
|---|---|---|
| Task 1 | Draft `cloud-sql.tf` retention diff; draft gcloud command sequence; draft restore-drill validation SQL | `terraform apply`, `gcloud sql backups create`, clone, delete clone |
| Task 2 | Draft `values.yaml` diff; draft git branch/commit/push commands; draft `hey`/`vegeta` invocation; pre-write snapshot section | Push to `tesserix-k8s/main`, click ArgoCD Sync/Rollback, run the load test |
| Task 3 | Draft custom-rule expression; draft curl smoke test; pre-write snapshot section with placeholders | Click Cloudflare dashboard, enable rules, upgrade plan |

Useful delegation: ask a subagent up front to scaffold `2026-06-05-wave1-infra-snapshot.md` with empty per-task sections + §3 commands pre-pasted, so you only fill in outputs.

---

## 8. Open questions / blockers

1. **Cloudflare plan tier on `fe3dr.com` — Free or Pro?** Blocker for Task 3. Confirm in dashboard Overview before allocating time. Pro upgrade is $20/mo, pre-approved.
2. **Cloudflare config in Terraform anywhere?** Quick grep in both sibling repos for `cloudflare_zone` / `cloudflare_ruleset`. If found, Task 3 must flow through that Terraform path, not dashboard-only.
3. **Non-prod restore target.** Runbook clones to a fresh instance `tesserix-restore-drill` in the same project. If a separate dev GCP project exists with its own Cloud SQL, decide whether to target that for cleaner blast radius. Default = in-place clone.
4. **`homechef-auth-bff` is Deployment + KEDA, not Knative.** Confirm intent (always-warm pod) is satisfied by `keda.minReplicaCount: 2` + `pdb.minAvailable: 1`. Update PROD-READINESS wording to "always-warm floor on auth-bff (KEDA `minReplicaCount`)" so future readers don't search for a non-existent Knative annotation.
5. **ArgoCD revision history depth.** App spec does not override default (10 revisions); ≥ 3 needed for safe rollback. Flagging only — should be fine.
