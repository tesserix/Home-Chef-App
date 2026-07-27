---
slug: vendor-portal-push-deploy
created: 2026-07-28
completed: 2026-07-28
mode: quick
status: complete
branch: fix/vendor-portal-push-deploy
follows: 260727-web-password-and-social-fixes
---

# Summary — vendor-portal builds and deploys on main, like every other service

Follow-up to deploying #802/#803, where shipping the vendor portal took a manual
dispatch plus a hand-purge of a cached GAR tag.

## The failure this removes

`homechef-vendor-portal-build.yml` had its `push:` trigger commented out, so:

1. `ensure-image-tags` treated vendor-portal as never-building and always
   **carried the previous image forward** under the new `main-<sha>` tag. The tag
   advanced every commit while the running binary did not — a green ArgoCD showing
   the latest sha proved nothing.
2. Shipping meant a `deploy=true` dispatch, which re-pointed a tag the GAR
   pull-through had **already cached** against the carried-forward blob. The
   mirror kept serving the stale digest until the cached tag was deleted by hand
   (`gcloud artifacts docker tags delete`, then a rollout restart).

## Changes

**`homechef-vendor-portal-build.yml`**
- Enabled `push:` on `main` only — matching `homechef-web-build.yml`. The
  commented-out block also listed `feat/**`, `feature/**`, `bugfix/**`,
  `hotfix/**`; those were dropped, since web does not publish per-branch images.
- `push: ${{ inputs.deploy == true }}` → `${{ github.event_name == 'push' || inputs.deploy == true }}`.
  **Enabling the trigger alone would have silently done nothing** — on a push
  event `inputs` is empty, so the gate was false and the job would build and
  never publish, leaving carry-forward in charge.
- Widened both Trivy gates the same way, so main-published images are still
  scanned rather than shipping unscanned.
- `workflow_dispatch` kept, for shipping a commit that misses these paths.

**`homechef-ensure-image-tags.yml`**
- `POLL[homechef-vendor-portal]` `"false"` → real change detection, so the job
  **waits** for the build instead of racing it. Leaving this false would have
  automated the exact stale-image bug on every commit.
- Added `changed_any()` and fixed two pre-existing under-detections of the same
  class: `homechef-web-app` builds on `packages/**` + `pnpm-lock.yaml` and
  `homechef-web` on `pnpm-lock.yaml`, but both were detected by app directory
  only — so a lockfile-only commit rebuilt them while this job called them
  unchanged and carried a stale image forward.
- Documented that each prefix list must mirror its build workflow's
  `push.paths`, and that under-detecting is the dangerous direction
  (over-detecting only times out without advancing deploy).

`delivery-portal` stays `"false"` — it genuinely has no push-triggered build.

## Verification

- Both workflows parse (`yaml.safe_load`).
- Every `run:` block passes `bash -n`.
- `changed_any` truth-tested: vendor-portal edit → true; lockfile-only → true
  (its build does run); api-only → false (correctly carries forward).
- `api` and `auth-bff` deliberately still use single-prefix `changed` — their
  builds trigger on their app dir only, so widening them would make this job wait
  for a build that never runs.

## Not verified

The first real push-triggered vendor-portal deploy has not happened yet — this PR
touches `.github/workflows/homechef-vendor-portal-build.yml`, which is itself in
the path filter, so merging it should trigger exactly that. Worth watching the
first run: confirm ensure-image-tags **waits** for the build rather than carrying
forward, and confirm the running pod digest matches the new GHCR digest (check
the digest, not the tag).
