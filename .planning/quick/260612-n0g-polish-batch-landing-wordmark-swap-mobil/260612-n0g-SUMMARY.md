---
phase: quick
plan: 260612-n0g
subsystem: branding
tags: [svg, wordmark, adaptive-icon, splash-screen, expo, docs]
requires: []
provides:
  - "Vector source of truth for the Home Chef mark (apps/web-landing/public/homechef-logo.svg)"
  - "Transparent RGBA adaptive-icon foregrounds + branded splash screens for both mobile apps"
  - "expo-splash-screen plugin configured per Expo SDK 55 conventions"
affects: [web-landing, mobile-customer, mobile-vendor, docs]
tech-stack:
  added: ["expo-splash-screen ~55.0.21 (both mobile apps)"]
  patterns: ["single vector source of truth — all brand PNGs derived from homechef-logo.svg"]
key-files:
  created:
    - apps/web-landing/public/homechef-logo.svg
  modified:
    - apps/web-landing/components/wordmark.tsx
    - apps/mobile-customer/assets/adaptive-icon.png
    - apps/mobile-customer/assets/splash-icon.png
    - apps/mobile-customer/app.json
    - apps/mobile-customer/package.json
    - apps/mobile-vendor/assets/adaptive-icon.png
    - apps/mobile-vendor/assets/splash-icon.png
    - apps/mobile-vendor/app.json
    - apps/mobile-vendor/package.json
    - pnpm-lock.yaml
    - docs/ops/CUTOVER-RUNBOOK.md
    - PROD-READINESS.md
decisions:
  - "SVG rasterization via sharp 0.34.5 (libvips) — the claimed ImageMagick rsvg delegate is NOT installed; magick's internal MSVG renderer rendered the logo black"
  - "expo-splash-screen pinned to ~55.0.21 (the version expo's bundledNativeModules.json maps for SDK 55)"
  - "Adaptive-icon glyph scaled to ~59% of the 1024 canvas (scale 13) — comfortably inside the 66% safe-zone circle, verified visually"
metrics:
  duration: ~15 min
  completed: 2026-06-12
---

# Quick Task 260612-n0g: Polish Batch (Wordmark Swap, Mobile Assets, Doc Drift) Summary

Hand-authored vector recreation of the coral bowl+house+steam mark now drives the web-landing wordmark and both mobile apps' Android adaptive icons + new splash screens; runbook/readiness docs synced to shipped reality.

## What Was Done

### Task 1 — homechef-logo.svg + Wordmark swap (`c7812fb`)
- Authored `apps/web-landing/public/homechef-logo.svg` (viewBox 0 0 64 64): rounded rect rx=14 with vertical gradient `#F34055` → `#FA6F5C` → `#FAA15D`, glyph as white stroked paths (3 wavy steam strokes, wide rim bar, two bowl-body arcs, pitched-roof house with chimney stub docked below the rim). Iterated once: initial roof peak crossed the rim bar; lowered the house so the peak sits below the rim with a clear gap, matching the source PNG composition.
- Rewrote the SVG inside `wordmark.tsx` with the identical geometry (gradient id `hc-wm-grad` to avoid DOM collisions). Component API (`{ inverted }`), 28px size, logotype span all unchanged. JSDoc updated.
- `npx tsc --noEmit` in apps/web-landing: clean (exit 0).

### Task 2 — Mobile adaptive icons + splash screens (`39ebfdd`)
- Derived a glyph-only SVG (white strokes, transparent bg, `translate(96 126.5) scale(13)` → glyph ≈59% of the 1024 canvas) from the Task 1 paths.
- Generated `adaptive-icon.png` (1024x1024 RGBA, transparent corners) and `splash-icon.png` (512x512 RGBA) and copied into both apps' `assets/`. Verified `srgba` channels + fully transparent corner pixel via `magick identify`, and visually confirmed the glyph inside the 66% adaptive-icon safe-zone circle.
- Both app.json: `android.adaptiveIcon.backgroundColor` `#ffffff` → `#FA6F5C`; added the `expo-splash-screen` config-plugin block (`image: ./assets/splash-icon.png`, `imageWidth: 200`, `resizeMode: contain`, `backgroundColor: #FA6F5C`).
- Added `expo-splash-screen: ~55.0.21` to both app package.jsons + lockfile (+19 lines).
- `npx expo config --type public` exits 0 in both app dirs (only the pre-existing Sentry org/project warning in vendor).

**Note:** splash/adaptive-icon changes require a native rebuild (prebuild/EAS) to appear on device — no rebuild was done in this task.

### Task 3 — Doc drift sync (`111084f`)
- CUTOVER-RUNBOOK §1: ExternalSecret, flip-the-switch, and apps-registry boxes flipped `[x]` with **DONE 2026-06-12** annotations; the release-tag box rewritten — the "push-to-main alone does not deploy it" claim was wrong (ci.yml deploys on main; release.yml only tags GHCR).
- PROD-READINESS: 5B infra-prereq annotated DONE; `[~]` Delivery (3PL) admin flipped `[x]`; 7E "Manage delivery providers" flipped `[x]` with the `c10d9aa` annotation.
- No files outside this repo touched.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] ImageMagick rsvg delegate not actually installed**
- **Found during:** Task 1 (first rasterization produced an all-black rounded rect)
- **Issue:** Plan stated "rsvg delegate confirmed available", but `rsvg-convert` is not on the system and magick fell back to its internal MSVG renderer, which can't handle the gradient/stroked groups.
- **Fix:** Used `sharp` 0.34.5 (already in root node_modules; libvips bundles librsvg) for all SVG→PNG rasterization. `magick identify` still used for PNG verification per plan.
- **Files modified:** none (tooling substitution only)

**2. [Rule 3 - Blocking] expo-splash-screen absent from the entire workspace**
- **Found during:** Task 2 step 5
- **Issue:** Not in either app's package.json, not hoisted, zero matches in pnpm-lock.yaml — the config plugin would not resolve.
- **Fix:** Added `expo-splash-screen@~55.0.21` (version from expo SDK 55 `bundledNativeModules.json`) to both app package.jsons and ran `pnpm install` (lockfile delta: +19 lines, expo-splash-screen only).
- **Files modified:** apps/mobile-customer/package.json, apps/mobile-vendor/package.json, pnpm-lock.yaml
- **Commit:** 39ebfdd

**3. Worktree base correction (startup protocol)**
- Worktree was based on `1690859`; hard-reset to the prescribed base `22e2d76` before any work (per dispatch instructions).

## Verification Results

- SVG rasterizes cleanly (sharp), composition matches source: gradient rounded rect + outline bowl/rim/house/chimney + 3 steam waves ✓
- `grep -c 'stroke="#fff"' wordmark.tsx` = 1; old filled-bowl path `M14 34h36` gone ✓
- All 4 PNGs `srgba`; adaptive icons 1024x1024 with transparent corners; splash 512x512 ✓
- Both app.json contain expo-splash-screen plugin + `#FA6F5C`; `expo config --type public` exit 0 both apps ✓
- Doc greps pass; no stray `[ ]`/`[~]` on the targeted lines ✓
- tsc: web-landing clean (customer-app baseline untouched — no mobile TS files changed) ✓

## Commits

| Task | Commit | Message |
|------|--------|---------|
| 1 | c7812fb | feat(web-landing): new bowl+house+steam wordmark + vector SVG logo source |
| 2 | 39ebfdd | feat(mobile): transparent adaptive-icon foregrounds + branded splash screens for customer + vendor |
| 3 | 111084f | docs: sync CUTOVER-RUNBOOK §1 + PROD-READINESS 5B/7E checkboxes to shipped reality |

## Known Stubs

None — all assets are final art derived from the authored vector. (Pre-existing: `tesserix-home public/homechef-icon.png` placeholder is out of scope, noted in PROD-READINESS.)

## Threat Flags

None — static brand assets, app config, and internal docs only; no new trust boundaries.

## Self-Check: PASSED

All 6 key files present on disk; all 3 task commits (c7812fb, 39ebfdd, 111084f) found in git log.
