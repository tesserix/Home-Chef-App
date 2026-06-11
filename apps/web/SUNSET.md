# apps/web — SUNSET

This React (Vite) customer **web ordering** app is **decommissioned**. Home Chef is app-only
(owner decision 2026-06-10): customers order on the mobile apps, not the web.

**Replaced by:** `apps/web-landing` — the Next.js marketing landing at `fe3dr.com` (the only web
surface). Run it locally with `pnpm dev:landing`.

**De-wired (2026-06-11):**
- Removed the `web` service from `docker-compose.yml`.
- Removed `.github/workflows/homechef-web-build.yml` + `homechef-web-release.yml` (no more image
  builds/deploys for this app).

**Kept** in the repo for history/reference — not deleted, not built, not deployed.

**Production cutover** (separate, owner-controlled, in `tesserix-k8s` + Cloudflare): point
`fe3dr.com` at the `homechef-web-landing` deployment and retire the `homechef-web` ArgoCD app.
Until that flip, `fe3dr.com` continues to serve the last-built `homechef-web` image.

> Critical: `vendors.fe3dr.com` (mobile API / auth-bff host) is **unaffected** — only the web UI dies.
