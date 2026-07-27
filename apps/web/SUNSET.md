# apps/web — ACTIVE again (was paused)

> **This file described a pause that is over.** It is kept because the history explains the
> odd shape of the CI setup, but the "paused / not built / manual-dispatch only" statements
> below no longer hold. Verify from `.github/workflows/`, not from this file.

This React (Vite) customer **web ordering** app is **built and deployed on every `main`
commit** as of 2026-07-27. It was paused 2026-06-11 and always intended to return (owner
direction 2026-06-18: "don't remove, just disable it, we'll bring back our web").

**Current state:**
- `.github/workflows/homechef-web-build.yml` has a **live `push: main` trigger** and
  publishes its own image, `homechef-web-app`.
- The separate image name is what un-blocked it. While the SPA and the marketing landing
  both published to `homechef-web`, the landing always won the tag and the SPA could never
  ship — which is why the workflow had been disabled rather than deleted.
- Kargo **requires** it to build on every promoted commit: the prod Stage derives
  `image.tag = main-<sha7>`, so an image has to exist at that tag for every commit that
  changes the app.
- `apps/web-landing` (Next.js marketing landing) still serves `fe3dr.com`. Run it with
  `pnpm dev:landing`.
- The `web` service is still absent from `docker-compose.yml`.
- `homechef-web-release.yml` (semver release builds) is still not present.

**Still outstanding:** the `homechef-web` ksvc slot serves the landing image, so the
routing/cutover for the SPA in `tesserix-k8s` + Cloudflare has not been decided.

> Note: `vendors.fe3dr.com` (mobile API / auth-bff host) is **unaffected** — only the customer web
> UI is paused.
