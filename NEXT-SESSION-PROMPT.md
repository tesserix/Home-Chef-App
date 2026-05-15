# Resume prompt — Home-Chef-App deploy automation parity with mark8ly

## Context from prior session (2026-05-13)

We just completed a normalization/a11y sweep across all 4 home-chef-app frontends, all landed at 20/20 audit scores, and pushed to `origin/main`. While verifying the theme fix on production (fe3dr.com), we discovered home-chef-app's deploy pipeline is **manual** — the image is built by GitHub Actions but the tesserix-k8s ArgoCD manifest needs manual editing to bump the tag.

**Today's manual deploy** (one-off — do not repeat the manual update next time):
- tesserix-k8s `6a224714` bumped `argocd/prod/apps/homechef/homechef-web.yaml` from `main-4eb62b7` → `main-e6c8d3d`
- Theme fix + 415 aria-hidden + full apps/web a11y sweep is now propagating to fe3dr.com

**Two gaps vs mark8ly we need to close:**

1. **No auto-bump CI job** — home-chef-app's 5 build workflows push images but don't update tesserix-k8s. Mark8ly's `mark8ly/.github/workflows/ci.yml` has a `bump-k8s` job that does this. Reference path: `/Users/Mahesh.Sangawar/personal/tesserix-new/mark8ly/.github/workflows/ci.yml` — search for `bump-k8s:` job.
2. **No Kargo project** — `argocd/prod/apps/homechef/homechef-web.yaml` has the annotation `kargo.akuity.io/authorized-stage: kargo-homechef:prod` but no actual Kargo Project/Warehouse/Stage resources exist for `kargo-homechef`. Mark8ly has `kargo-mark8ly` fully wired with promotion flow.

## Goal

Get home-chef-app on the same auto-deploy footing as mark8ly:
- Push to home-chef-app `main` → CI builds image → CI auto-bumps tesserix-k8s → ArgoCD syncs → prod updates (possibly gated through Kargo for prod promotion review)

## Surface area

### Part A — Auto-bump (lower-risk, do this first)

Add a `bump-k8s` job to **5 home-chef-app workflows**:
- `.github/workflows/homechef-web-build.yml`
- `.github/workflows/homechef-admin-portal-build.yml`
- `.github/workflows/homechef-vendor-portal-build.yml`
- `.github/workflows/homechef-delivery-portal-build.yml`
- `.github/workflows/homechef-api-build.yml`

Template to copy from: `mark8ly/.github/workflows/ci.yml` job `bump-k8s` (~30 lines). Key adjustments:

| Mark8ly | Home-Chef-App |
|---|---|
| Updates `tesserix-k8s/charts/apps/mark8ly-{svc}/values.yaml` via yq | Updates `tesserix-k8s/argocd/prod/apps/homechef/homechef-{svc}.yaml` parameters block |
| Tag format: `sha-${SHORT_SHA}` (12 chars) | Tag format: `main-${SHORT_SHA}` (7 chars, current homechef convention) |
| Bumps ALL mark8ly services in one job (`for svc in admin storefront ...`) | Each home-chef workflow bumps its own service only (so individual app changes don't cascade) |
| Secret: `TESSERIX_K8S_BOT` (already exists) | Same secret — verify access to `argocd/prod/apps/homechef/` paths |

The yq command for home-chef-app YAMLs will look like:
```bash
yq -i '(.spec.source.helm.parameters[] | select(.name == "image.tag")).value = "main-'"$SHORT_SHA"'"' \
  "tesserix-k8s/argocd/prod/apps/homechef/homechef-${svc}.yaml"
```

Also bump the `podAnnotations.client\.knative\.dev/updateTimestamp` to force a Knative revision (we did this manually today — value uses ISO timestamp).

### Part B — Kargo for homechef (higher-risk, do second)

Create `kargo-homechef` project mirroring mark8ly's setup. Investigate where mark8ly's Kargo resources live in tesserix-k8s (start with `find tesserix-k8s -name "*kargo*"` from the repo root and look for Project/Warehouse/Stage/PromotionTemplate manifests under `kargo/` or `manifests/` directories — `kargo/devtest/` is the Kargo *instance*, not the project resources).

The five home-chef services that need to be subscribed to the Kargo project (based on ArgoCD apps):
- `homechef-web` (fe3dr.com)
- `homechef-admin-portal` 
- `homechef-vendor-portal`
- `homechef-delivery-portal`
- `homechef-api` (auth-bff + main API)

Plus auxiliary services if present (check `argocd/prod/apps/homechef/`):
```
find tesserix-k8s/argocd/prod/apps/homechef -name "*.yaml" -exec grep -l "kargo.akuity.io" {} +
```

## Constraints

- **Don't break mark8ly's pipeline** — bump-k8s changes should not touch `charts/apps/mark8ly-*/values.yaml` paths.
- **Don't auto-deploy to prod without review** — if Kargo gates prod, that's the right pattern. If we don't have Kargo yet, the bump-k8s job will push directly to ArgoCD which auto-syncs (current behavior since `syncPolicy.automated.prune: false` is on homechef-web). Decide: do we want Kargo gating before adding the auto-bump, or auto-bump first and add Kargo gating later?
- **Verify secret access**: `TESSERIX_K8S_BOT` PAT needs `repo` scope on tesserix/tesserix-k8s. Already configured for mark8ly — should work for homechef too. Test with one workflow before rolling to all 5.

## Suggested execution order

1. **Read mark8ly's `bump-k8s` job** in full (`mark8ly/.github/workflows/ci.yml`) — understand structure, error handling, conditional logic (only on push-to-main).
2. **Read one home-chef-app build workflow** (`homechef-web-build.yml`) — understand current structure, where to insert the new job, what outputs are available (SHORT_SHA from `steps.meta`).
3. **Add bump-k8s job to ONE workflow** (start with `homechef-web-build.yml`). Test by pushing a no-op commit to apps/web/, verify:
   - GH Actions builds image with new tag
   - bump-k8s job runs after image build succeeds
   - tesserix-k8s gets a new commit bumping homechef-web.yaml
   - ArgoCD syncs and fe3dr.com updates
4. **Replicate to remaining 4 workflows** if step 3 works cleanly.
5. **Add Kargo for homechef** — clone mark8ly's Kargo resources, rename `mark8ly` → `homechef`, point Warehouses at the 5 home-chef-app image repos in GAR.

## Files to read first (next session)

```
mark8ly/.github/workflows/ci.yml                              # bump-k8s job template
Home-Chef-App/.github/workflows/homechef-web-build.yml        # current state
tesserix-k8s/argocd/prod/apps/homechef/homechef-web.yaml      # target of bumps
tesserix-k8s/argocd/prod/apps/mark8ly/storefront.yaml         # mark8ly's equivalent for comparison
# Find Kargo resources:
find tesserix-k8s -name "*.yaml" | xargs grep -l "kind: Project\|kind: Warehouse\|kind: Stage\|kind: PromotionTemplate" 2>/dev/null | grep -i kargo
```

## Today's manual deploy commits (reference)

- **Home-Chef-App main**: `0824445` → `e6c8d3d` (8 commits, all apps/web fixes)
- **tesserix-k8s main**: `6a224714` (one-off manual bump from `main-4eb62b7` → `main-e6c8d3d`)

## Stop conditions

- If Kargo turns out to be substantially more complex than mark8ly's setup (e.g., needs new RBAC, GAR pull-secret), pause and ask user before proceeding.
- If `TESSERIX_K8S_BOT` doesn't have access to homechef paths, ask user to grant before continuing.
- If adding bump-k8s to one workflow accidentally pushes to mark8ly paths, revert immediately and verify path filters.

## After completion — verify

1. Push a no-op change to `apps/web/` (e.g., a comment in HomePage.tsx)
2. Watch GH Actions: image build → bump-k8s job
3. Confirm tesserix-k8s gets a commit with the new SHA
4. Confirm ArgoCD syncs (look at `https://argocd.tesserix.app` or `kubectl -n argocd get application homechef-web`)
5. Hard-refresh fe3dr.com, confirm the new code shipped

## Don't forget

- The theme toggle fix needs hard-refresh on user's browser even after deploy (browser cache holds the old JS bundle). Mention to user when shipping.
- The 4 LCP hero images we marked with `fetchPriority="high"` should show measurable LCP improvement in PageSpeed / WebPageTest after deploy. Worth a screenshot-before/after for the team.
