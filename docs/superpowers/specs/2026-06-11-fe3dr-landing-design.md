# fe3dr.com landing page — design spec (2026-06-11)

## Goal
A beautiful, SEO-optimized public marketing landing for **Home Chef** (home-cooked food
delivery marketplace) at `fe3dr.com`. It replaces the sunset customer web app as the only web
surface. Primary audience: hungry customers downloading the app to order; secondary: home chefs
("Cook with us"). Owner decisions captured below.

## Decisions (locked)
- **Stack:** new **Next.js 16 + React 19** app at `apps/web-landing`, **static export**
  (`output: 'export'`) — pure SSG (best SEO/LCP, no Node runtime, deploys as static files behind
  nginx; matches budget + infra). Node 22. TypeScript.
- **Old `apps/web` (Vite React SPA) is de-wired** (removed from docker-compose / CI / deploy) but
  **kept in the repo**. Not deleted. Production DNS/routing cutover is a separate manual step the
  owner controls — NOT part of this build.
- **Brand: Airbnb-style coral consumer palette** (matches the customer app visitors will download).
  This is `.impeccable.md` → "Customer palette addendum":
  - Accent **coral** `#FF385C` (pressed `#E00B41`, tint `#FFE8EC`) — the single accent (CTAs, links, focus).
  - Text **charcoal** `#222222` / soft `#717171`. Canvas **white** `#FFFFFF`. Hairline `#EBEBEB`. Soft surface `#F7F7F7`. Success `#008A05`.
  - **Type:** Geist (display, 600/700) + Inter (body, 400/500/600), tabular figures for numerals.
  - Feel: white-first, photo-forward, generous space, confident/appetizing/quietly modern.
  - **Anti-references (hard):** Swiggy/Zomato red saturation, AI-slop gradients (cyan-on-dark, purple-blue), glassmorphism, gradient text on headings, hero-metric layouts, same-size card grids, bounce/elastic motion.
- **Hero CTA: live App Store + Play Store badges** (owner choice 2026-06-11, despite apps not yet
  published — badge links use placeholder store URLs with a clear TODO until the real listings exist).
- **Imagery:** tasteful Unsplash-style food/chef photos as clearly-swappable placeholders so the
  page looks appetizing now; swap for owned photography later.

## Sections (top → bottom)
1. **Nav** — Home Chef wordmark · How it works · For chefs · `Get the app` (coral). Sticky, hairline on scroll.
2. **Hero** — asymmetric (not centered). Bold Geist headline ("Real home-cooked food, from kitchens near you" — final copy at build), Inter subhead, **App Store + Play Store badges**, an appetizing food/chef hero image. No fake metrics.
3. **How it works** — 3 steps: Browse home chefs → Order in a few taps → Track to your door. Icon + short copy.
4. **Why Home Chef** — value props: real local cooks · FSSAI-verified kitchens · live delivery tracking · fair to chefs. Editorial, hairline-separated, photo-supported.
5. **Cook with us** — chef-recruitment band ("Turn your kitchen into a business") → chef CTA / vendor-app badge.
6. **Trust strip** — honest + pre-launch-appropriate (FSSAI-verified, secure payments, "Launching in [city]"). NO fabricated testimonials/metrics (brand is anti-FOMO).
7. **Footer** — Privacy · Terms/EULA · Refund · Contact + store badges + social. Legal links point to stub pages/anchors for now (real web legal content is a separate pending item).

## Cross-cutting
- **SEO:** Next `metadata` (title/description/canonical), OpenGraph + Twitter cards, `sitemap.xml`,
  `robots.txt`, JSON-LD (`Organization` + `MobileApplication`), `lang="en"`, semantic landmarks
  (`<header><nav><main><footer>`), descriptive alt text, real heading hierarchy (one `<h1>`).
- **A11y:** WCAG 2.1 AA. Skip link first focusable. Visible coral focus ring (2px + 2px offset),
  never `outline:none`. `prefers-reduced-motion` honored. AA contrast (coral-on-white for large/CTA).
- **Motion:** Framer Motion or CSS. Entrances ease-out-quart `cubic-bezier(0.22,1,0.36,1)`; animate
  `opacity`/`transform` only; no bounce/elastic; reduced-motion → instant.
- **Design system:** `@tesserix/web` primitives where useful (`transpilePackages`), Tailwind v4,
  coral tokens as CSS variables (mirror customer `customerColors`), never hardcode hex in components.
- **Deploy:** `Dockerfile` builds the Next static export (`next build` → `out/`) and serves via
  **nginx** (mirror old `apps/web/nginx.conf` security headers/SPA-less static serving). Must build
  green (`npm run build` produces `out/` with static HTML).

## Out of scope (this build)
- Production DNS/routing cutover (owner-controlled, deferred until store listings live).
- Real legal-page content, owned photography, real store URLs, email/waitlist (we chose live badges).
- Decommissioning vendor-portal/delivery-portal web UIs (separate sunset step).

## Definition of done
- `apps/web-landing` builds to static HTML (`out/`) with 0 errors; renders all 7 sections on
  desktop + mobile; coral brand + Geist/Inter; live store badges; Unsplash placeholders.
- SEO essentials present (metadata, OG, sitemap, robots, JSON-LD, semantic HTML, one h1).
- a11y: skip link, focus rings, reduced-motion, AA contrast.
- Old `apps/web` de-wired from docker-compose + CI, dir retained.
- Deployable via Dockerfile+nginx. (Prod cutover deferred.)
