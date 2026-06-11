# Next session — finish remaining Home Chef work (handoff 2026-06-11)

Continuing the Home Chef vendor app. Tree is clean, everything pushed to `main`.
Read this + `PROD-READINESS.md` + `VENDOR-TICKETS-SPEC.md`. Don't re-discover state.

## What's DONE + live this session
- **OCR** (Cloud Vision) — `POST /chef/documents/ocr` extracts FSSAI number + expiry; mobile
  pre-fills in renew + onboarding. Prod `homechef-api-00092`, verified.
- **Vendor profile** — name, description, avatar, **cover image** (16:9, `contentFit:cover`), 4
  kitchen photos; customer `ChefCard` hero now prefers the cover. **20-dish menu** across 6
  categories with photos, correct veg/non-veg.
- **Fixes:** doc-replace 500 (`file_url` column), re-upload nav trap (→ `/documents/renew`),
  ACTION-REQUIRED cards (neutral + severity-colored red/amber/grey left stripe), status badges
  (pending/preparing → grey, ready → green), `ENVIRONMENT=production`, Cloud Trace IAM
  (`cloudtrace.agent`), `isVeg` sent on menu create/update.
- **Uber-monochrome** vendor redesign (persimmon retired; ink actions + functional green/amber).

## PENDING — prioritized

### 1. 🟡 Delivery-zone enforcement — feature work, NOT an outage (premise corrected 2026-06-11)
**Ordering currently WORKS.** The zone gate at `orders.go:261` is opt-in (`if HasActiveZones()`);
with all 18 zones `is_active=false`, `HasActiveZones()` returns false → the coord/zone check is
**skipped entirely**. Deactivating the zones is what *unblocked* ordering, not what broke it. So the
earlier "🔴 blocks ALL ordering" framing was inverted. Re-enabling proper delivery-area enforcement
is feature-completion, deferred (owner decision 2026-06-11), and needs: (a) backend autocomplete to
forward Photon `geometry` (pure JS/backend — sidesteps the broken native rebuild; do NOT add
`expo-location`), (b) customer app to carry coords from the picked suggestion onto the saved address,
(c) real zone bounds + a coverage-policy decision, then flip zones active. See memory
`project_delivery_zones_disabled`. Mostly **customer-app** work.

### 2. ✅ Vendor tickets — DONE (2026-06-11)
Built per `VENDOR-TICKETS-SPEC.md`: `hooks/useSupport.ts` + 3 screens (`app/support/{index,new,[id]}.tsx`)
+ `components/vendor/TicketStatus{Chip,Stepper}.tsx` + More-tab entry (`LifeBuoy`) + `more.support`
i18n (en/hi). Full live API cycle (create→detail→reply→close) verified against prod
`/api/v1/support/tickets`. No backend changes. ⏳ remaining: visual tap-through on the sim (Metro 8082)
+ commit.

### 3. ✅ Native build fixed (2026-06-11) — cropper + Hindi unblocked
Root-caused the `PrecompileModule RNFBApp` failure to TWO independent issues under Xcode 26.5
explicit modules: **(1)** RNFBApp (framework module via `use_frameworks! static`) imported
non-modular React-Core headers → `-Werror=non-modular-include-in-framework-module`; **(2)** the
**prebuilt React-Core** Debug binary lacked `react_rendererdebug` symbols (`getDebugProps`) that
from-source `ExpoModulesCore` references → linker error. **Both** are fixed by one line —
`buildReactNativeFromSource: true` in `app.json` (expo-build-properties.ios): from-source RN makes
React-Core a real `React.framework` module (so RNFBApp's includes become modular) AND compiles the
debug renderer symbols in the same config. Verified: clean Debug **sim** build SUCCEEDED, app
installs + launches + stays alive on the sim with ImageManipulator (cropper) + Localization (Hindi)
+ Firebase all linked in. Tradeoff: from-source builds are slower (acceptable).
- **Signing** was a red herring for the sim path — sim builds need none (`CODE_SIGNING_ALLOWED=NO`);
  the "No code signing certificates" only blocks **device** builds, which still need the Apple Dev
  account (external blocker #6). Cropper + Hindi verify on the **sim**, so they're unblocked now.
- Local builds skip the Sentry source-map upload phase via `ios/.xcode.env.local`
  (`SENTRY_DISABLE_AUTO_UPLOAD=true`) — no Sentry org token locally; EAS still uploads with secrets.
- ⏳ remaining: visual tap-through of the cropper + Hindi screens on the sim (Metro 8082).

### 4. 🌐 Web sunset (Wave 5A, decided) + landing page
Build the `fe3dr.com` landing page (Uber-style, store badges, "Cook with us" chef CTA, legal
footer). Decommission `apps/vendor-portal` + `apps/delivery-portal` (301 → landing); clean
auth-bff registry + repo. See PROD-READINESS "Web sunset" items + memory `project_web_sunset_app_only`.

### 5. 🏗️ Platform integration (tesserix-home + infra — separate repos, you DO have access)
- `tesserix-home`: product config (`lib/products/configs.ts`), `app/admin/apps/homechef/page.tsx`,
  sidebar nav; the **platform-admin approval UI** for chef requests.
- Platform `auth-bff/products.yaml` homechef entry; `tesserix-infra/services.yaml` registration.
- (You have git access to `tesserix-k8s` — the "no access" assumption was wrong; clone works.)

### 6. 🍎 Blocked on Apple Developer enrollment (external)
TestFlight, APNs prod cert, App Store screenshots/listing/submission.

### 7. Operational / legal
Cloud SQL backup+restore drill · Cloudflare WAF · Trivy gate flip (6 vulns) · Privacy/EULA URLs ·
FSSAI/FoSCoS API (external).

## Gotchas
- **Prod GKE:** context `gke_tesseracthub-480811_asia-south1_tesseract-prod-in-gke`, ns `homechef`.
  Deploy = push to `main` (paths `apps/api/**` / `apps/auth-bff/**`) → CI → bump → ArgoCD (~3-4 min).
  Never `kubectl patch` prod image tags (ArgoCD self-heals).
- **Sim/Metro:** `npx expo start --dev-client --port 8082`; launch via
  `homechef-vendor://expo-development-client/?url=http%3A%2F%2Flocalhost%3A8082`. Native build now
  works (from-source RN, see #3); a fresh Debug sim build is installed on the iPhone 17 Pro sim.
  Local sim build = `xcodebuild -workspace ios/HomeChefVendor.xcworkspace -scheme HomeChefVendor
  -configuration Debug -destination 'platform=iOS Simulator,id=<booted>' CODE_SIGNING_ALLOWED=NO build`
  (or `expo run:ios`); first from-source build ~6–8 min.
- **Test token:** re-capture if `/tmp/seed-token.txt` stale — temp `console.log` of
  `useAuthStore.getState().accessToken` in `app/_layout.tsx`, read from Metro log, revert.
- ~100 pre-existing lucide TS errors in mobile-vendor — filter on touched files.
- `gh pr create` fails on tesserix org (EMU); `git push` + `gh run watch` work.

## Suggested order
~~Delivery zones~~ · ~~vendor tickets~~ · ~~native build fix~~ (all DONE) → web sunset + landing →
platform/tesserix-home integration. Delivery-zone enforcement is deferred feature work (ordering
already works) and is mostly customer-app — pick it up only when delivery-area gating is actually
wanted. Cropper + Hindi just need a visual tap-through on the now-working sim build.
