# apps/web — PAUSED (temporarily disabled)

This React (Vite) customer **web ordering** app is **temporarily disabled / paused**, **not
decommissioned** — it is **planned to return** (owner direction 2026-06-18: "don't remove, just
disable it, we'll bring back our web"). For now Home Chef is app-first: customers order on the
mobile apps while the web is paused.

**Kept on purpose** — the app code stays in the repo so it can be brought back. It is just not
built or deployed at the moment.

**While paused:**
- `apps/web-landing` (Next.js marketing landing) serves `fe3dr.com`. Run it with `pnpm dev:landing`.
- The `web` service was removed from `docker-compose.yml`.
- `.github/workflows/homechef-web-build.yml` is **kept but disabled** — manual-dispatch only, never
  runs automatically (and its deploy job is gated to push-on-`main`, so even a manual run won't
  deploy). Re-enable the commented-out `push` / `pull_request` triggers to reactivate.
- `homechef-web-release.yml` (semver release builds) is not present right now.

**To bring the web back:**
1. Uncomment the `push` / `pull_request` triggers in `homechef-web-build.yml`.
2. Coordinate the `homechef-web` ksvc slot — it currently serves the `web-landing` image, so decide
   the routing/cutover in `tesserix-k8s` + Cloudflare before re-enabling deploys.
3. (Optional) restore a `homechef-web-release.yml` for semver release images.

> Note: `vendors.fe3dr.com` (mobile API / auth-bff host) is **unaffected** — only the customer web
> UI is paused.
