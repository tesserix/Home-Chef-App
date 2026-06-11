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

### 3. 🔧 Native rebuild / iOS signing fix — unblocks two things
`expo run:ios` fails "No code signing certificates" (even sim); `xcodebuild` Debug-sim fails on
`PrecompileModule RNFBApp` (Firebase explicit-modules). Fixing this unblocks **(a)** the 16:9
cover **cropper** (code-complete, `components/ImageCropper.tsx`, just needs expo-image-manipulator
linked) and **(b)** the **Hindi UI verification** (expo-localization). Likely needs an Apple Dev
signing cert in Xcode and/or a `CLANG/SWIFT_ENABLE_EXPLICIT_MODULES=NO` build-setting workaround.

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
  `homechef-vendor://expo-development-client/?url=http%3A%2F%2Flocalhost%3A8082`. Native rebuild broken (see #3).
- **Test token:** re-capture if `/tmp/seed-token.txt` stale — temp `console.log` of
  `useAuthStore.getState().accessToken` in `app/_layout.tsx`, read from Metro log, revert.
- ~100 pre-existing lucide TS errors in mobile-vendor — filter on touched files.
- `gh pr create` fails on tesserix org (EMU); `git push` + `gh run watch` work.

## Suggested order
~~Delivery zones~~ (vendor tickets DONE) → signing fix (unblocks cropper + Hindi) → web sunset +
landing → platform/tesserix-home integration. Delivery-zone enforcement is deferred feature work
(ordering already works) and is mostly customer-app — pick it up only when delivery-area gating is
actually wanted.
