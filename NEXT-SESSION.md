# Next session — Home Chef vendor app continuation

## Where we are (2026-06-04, end of dedicated-gateway + location-service + onboarding-redesign marathon)

This session was a long one. We took prod login from broken → working through three architectural fixes, then ported mark8ly's location service to homechef with India seeding + Photon-backed autocomplete, then did a full redesign pass on all 6 onboarding screens. Current build installed on the iPhone 17 Pro sim is `build-1780547633895.tar.gz` from commit `5bf2d3a`. Sim is pinned to Koramangala (`12.9352, 77.6245`).

## What shipped (this session, latest first)

| Commit | Repo | Scope |
|---|---|---|
| `5bf2d3a` | Home-Chef-App | feat(mobile-vendor): full onboarding redesign — section labels, hairline groups, upload tiles, edit links |
| `e177f6a` | Home-Chef-App | fix(mobile-vendor): documents.tsx — picker handlers now try/catch + useToast on success/error + canonical FormData filename |
| `a2166c1` | Home-Chef-App | polish(mobile-vendor): minor onboarding tweaks (preceding the full redesign — left alone, fine) |
| `d2f07d7` | Home-Chef-App | polish(mobile-vendor): declutter Step 2 address — left-aligned section title, GPS+search as paired primary actions |
| `b63a024` | Home-Chef-App | feat(location): Photon-backed `/locations/autocomplete` proxy + mobile street-level autocomplete (mirrors mark8ly), India-filtered |
| `2cfa1f0` | Home-Chef-App | feat(api): India-only location reference data — ISO-keyed Country/State/City/Postcode models lifted from mark8ly + idempotent seed of 1 country + 36 states + 49 cities + ~110 PIN codes |
| earlier | Home-Chef-App | mobile-vendor: useLocations hook, kitchen-details state/city chip strips + PIN autocomplete + GPS auto-fill (Apple/Google Maps reverse geocode + canonicalize via /postcodes/search) + locked India badge (later removed) |
| `bd06525` | Home-Chef-App | feat: GPS auto-fill on Step 2 + locked India badge (badge later removed) |
| `1c5b225` | Home-Chef-App | fix: onboarding routing guard now allows any step path while status=not_started; Input disabled state visually distinct |
| `5b16cc0` | Home-Chef-App | revert(auth): drop X-Session-Token interim — gateway swap means plain `Authorization: Bearer` works again |
| `5844f8c1` | tesserix-k8s | feat(homechef): AuthorizationPolicy allowing fe3dr.com on the dedicated gateway |
| `2ffb25e6` | tesserix-k8s | chore(homechef): AppProject permits istio-ingress namespace + security.istio.io |
| `c8ce66aa` | tesserix-k8s | feat(homechef): dedicated istio ingress gateway — isolates fe3dr.com from legacy Keycloak RequestAuthentication |
| earlier in session | tesserix-k8s | wiring of `homechef-ingress-gateway` ArgoCD app + Gateway resource + VirtualService rebind to homechef-gateway |
| `c614e74d` | tesserix-k8s | fix(homechef-api): set BFF_SESSION_URL so api middleware can verify mobile Bearer tokens via auth-bff /auth/session |
| `36adf3a` | Home-Chef-App | fix(api): post-AutoMigrate DDL drops legacy idx_users_email so per-pool email uniqueness takes effect |
| `021c11d` | Home-Chef-App | chore(mobile-vendor): drop unused LabeledRow, widen tab labels (Me…) , toast on review reply |

## Production state — verify first

1. **Mobile build on sim**: `build-1780547633895.tar.gz` installed on `AD109A46-2F99-43C3-8AAA-FEE68DC8499E` (iPhone 17 Pro). Bundle contains "WHEN YOU COOK" string (Step 3 redesign confirmed shipped).
2. **API**: `vendors.fe3dr.com` is routed via the **dedicated homechef-ingressgateway** (Cloudflare tunnel `homechef-prod` points at `http://homechef-ingressgateway.istio-ingress.svc.cluster.local:80`). Legacy Keycloak `RequestAuthentication` policies (jwt-auth-customer / jwt-auth-internal) are no longer in the path for fe3dr.com hosts. Mark8ly + Fanzone still use them on the shared gateway.
3. **Backend Bearer fallback**: `BFF_SESSION_URL` env on homechef-api ksvc points at `http://homechef-auth-bff.homechef.svc.cluster.local:8080/auth/session`. The api's `middleware/bff_auth.go` falls back to bearer-via-auth-bff when HMAC is absent.
4. **Database schema**: Legacy `idx_users_email` is dropped in homechef-api's post-AutoMigrate DDL (see `apps/api/database/database.go`). Per-pool email uniqueness is active. Location tables are seeded on each boot via `services.SeedLocations()` in `apps/api/main.go`.

**Sanity probes:**
```bash
# Auth path through new gateway — should be 401 JSON, NOT Istio 'Jwt is not in the form'
curl -sS -i -H "Authorization: Bearer notajwt" https://vendors.fe3dr.com/api/v1/chef/dashboard | head -3

# Location data
curl -sS https://vendors.fe3dr.com/api/v1/locations/countries/IN/states | jq '.data | length'   # expect 36
curl -sS "https://vendors.fe3dr.com/api/v1/locations/postcodes/search?q=Koram" | jq           # expect 560034 Koramangala
curl -sS "https://vendors.fe3dr.com/api/v1/locations/autocomplete?q=Raheja+Exotica" | jq      # expect Photon hits, Mumbai
```

## What's likely next session

User was actively walking through the 6 redesigned onboarding screens at session end. Expected follow-ups:

1. **Visual polish iteration on the onboarding redesign** — the user has high taste and will likely flag specific screens that still feel off. Each step (`apps/mobile-vendor/app/(onboarding)/`):
   - `personal-info.tsx` — identity + contact cards, locked email
   - `kitchen-details.tsx` — already most-iterated; GPS + search + state/city pickers
   - `operations.tsx` — WHEN YOU COOK / PREP TIME / DELIVERY AREA sections with Lucide icons
   - `documents.tsx` — upload tiles (dashed empty → checkmark filled) + CheckCircle + Replace link. Picker handlers have try/catch + toasts now. **Image upload itself was structurally correct — the bug was silent error handling**.
   - `policies.tsx` — bullet T&C card, full-row checkbox
   - `review.tsx` — per-section Edit deep-links + document readiness strip
   - `pending.tsx` — post-submit waiting state
2. **Backend asks queue still pending** (from prior session, the user has signaled willingness):
   - `is_veg` boolean on MenuItem (kills the dietaryTags hack)
   - `customerName` + `customerPhone` on OrderResponse
   - `recentOrders` array on `/chef/dashboard`
   - `GET /chef/orders/:orderId`
   - Platform charges + GST split on Earnings
3. **Expanding seeded PIN dataset** — only ~110 PINs seeded. Photon covers everything else via `/locations/autocomplete`, but if the chef community lands in non-major-metro areas the chip-strip city pickers will be empty. Loading the full Indian dataset (~150k PINs) is a follow-on data-engineering task.
4. **mark8ly mobile parity** — mark8ly's mobile uses `Authorization: Bearer` too. Once they test mobile against prod they'll hit the same Istio JWT issue. Either give them a dedicated gateway (mirroring the homechef pattern) or let them adopt the X-Session-Token interim. The pattern from this session is reusable.

## Locked architectural decisions (do NOT regress)

### Auth / gateway
- HomeChef has its OWN `istio-ingressgateway` workload in `istio-ingress` namespace, label `istio: homechef-ingressgateway`. Selector specifically does NOT match the legacy Keycloak RequestAuthentication policies which still target `istio: ingressgateway` and `istio: custom-ingressgateway`.
- Cloudflare tunnel `homechef-prod` (id `03e498ae-dad7-4e6a-9744-6fa782956861`) public hostnames now point at `http://homechef-ingressgateway.istio-ingress.svc.cluster.local:80`. All 8 fe3dr.com hostnames (`fe3dr.com`, `www.fe3dr.com`, `vendors.fe3dr.com`, `admin.fe3dr.com`, `delivery.fe3dr.com`, `api.fe3dr.com`, `identity.fe3dr.com`, `internal-identity.fe3dr.com`) routed through the new gateway. **Tunnel public-hostname routing is NOT IaC** — it's the Cloudflare Zero Trust dashboard.
- API auth: `Authorization: Bearer <session_token>` from mobile. API's `middleware/bff_auth.go` HMAC-verifies BFF requests; falls back to `BFF_SESSION_URL` to validate bearer via `homechef-auth-bff.homechef.svc:8080/auth/session`. Auth-bff uses cookies for web, accepts the same bearer for mobile.

### Location data
- Models are ISO-keyed: `Country.Code` (char(2)), `State.ID` (varchar(10), shape "IN-MH"), `City.ID` (slug, shape "IN-KA-bengaluru"), `Postcode.Code` (varchar(10), India PIN). No UUIDs, no IsActive flags.
- Seeded inline in `apps/api/services/seed_locations.go`. Idempotent via `clause.OnConflict{DoNothing: true}`.
- Pre-AutoMigrate DDL in `apps/api/database/database.go` drops legacy UUID-keyed tables iff they still have `is_active` column. Safe on prod (had no data).
- Endpoints: `GET /api/v1/locations/{countries, countries/:code/states, states/:code/cities, cities/:cityName/postcodes, postcodes/search, autocomplete}`. The last one (`autocomplete`) proxies to `https://photon.komoot.io/api/`, India-filtered. No API key.

### Mobile patterns
- iOS Pressable inner-View pattern for row-shaped Pressables (function-style `style={({pressed}) => [...]}` drops flex/bg on iOS).
- `useToast()` for non-blocking feedback; `Alert.alert` for destructive confirmations only.
- Onboarding draft state in Zustand store `apps/mobile-vendor/store/onboarding-store.ts` with `persist` + `createJSONStorage(AsyncStorage)`. Routing guard in `_layout.tsx` allows any of the 6 onboarding steps while backend `status === 'not_started'`.
- `Input` component disabled state: bone background + muted ink text when `editable={false}`.

### Operational gotchas
- Use `npx eas-cli@18.4.0` for builds — 20.x silently loses simulator profile, exits 0 with no artifact.
- **Build MUST run from `apps/mobile-vendor/`** — if cwd is repo root, eas-cli auto-generates a stray root `eas.json` without `prod-sim` profile. Use a chained `cd ... && find -delete && eas build` command.
- Sim UUID: `AD109A46-2F99-43C3-8AAA-FEE68DC8499E` (iPhone 17 Pro). Bundle ID: `com.homechef.vendor`. Sim location pinned to Koramangala for address-autocomplete testing — set via `xcrun simctl location <UUID> set <lat,lon>`.
- Kargo overrides `argocd/prod/apps/homechef/*.yaml` image tags. Manifest is the floor, NOT the deployed tag. Don't try to roll back by editing manifests — use ArgoCD UI Rollback or push a code revert.
- Two ArgoCD AppProjects were touched: `homechef` (now permits `istio-ingress` namespace + `security.istio.io` group).
- Trivy gate is still `exit-code: '0'` (warn-only) across all 8 build workflows. Re-tighten once Dependabot backlog (~78 vulns at end of session) is addressed.

## Verified test chef
- Email: `mahesh.sangawar@gmail.com`
- Current user_id in prod: `dcba6316-e735-4fcd-b671-a0165e4c92b8` (auth_pool=business, gip_uid=uvWmEC2MxnUmb3UyujlOTWPzjVf2)
- No chef_profile yet — the user was actively walking through onboarding when the session ended

To wipe the test user and re-run onboarding fresh:
```bash
PGPASS=$(kubectl -n homechef get secret homechef-postgres-app-credentials -o jsonpath='{.data.password}' | base64 -d)
kubectl -n homechef exec homechef-postgres-10 -c postgres -- env PGPASSWORD="$PGPASS" psql -h localhost -U homechef -d homechef_db -c "DELETE FROM users WHERE email = 'mahesh.sangawar@gmail.com';"
```
Then uninstall + reinstall the app on the sim to wipe SecureStore + AsyncStorage.

## Build + install workflow

```bash
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App/apps/mobile-vendor && \
  rm -f /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App/eas.json && \
  find . -maxdepth 1 -name 'build-*.tar.gz' -delete && \
  EXPO_NO_TELEMETRY=1 npx eas-cli@18.4.0 build --local --platform ios --profile prod-sim --non-interactive

# When the artifact lands:
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App/apps/mobile-vendor && \
  BUILD=$(ls -t build-*.tar.gz | head -1) && \
  rm -rf /tmp/hcv-app && mkdir -p /tmp/hcv-app && \
  tar -xzf "$BUILD" -C /tmp/hcv-app && \
  xcrun simctl terminate AD109A46-2F99-43C3-8AAA-FEE68DC8499E com.homechef.vendor 2>/dev/null ; \
  xcrun simctl install AD109A46-2F99-43C3-8AAA-FEE68DC8499E /tmp/hcv-app/HomeChefVendor.app && \
  xcrun simctl launch AD109A46-2F99-43C3-8AAA-FEE68DC8499E com.homechef.vendor
```

## Suggested next-session prompt

```
Continue Home Chef vendor app onboarding polish + verification. Last session did three big things end-to-end:

1. Dedicated istio ingress gateway for homechef (fe3dr.com no longer behind legacy Keycloak RequestAuthentication). Login works on the prod sim.
2. Ported mark8ly's location service — India-only Country/State/City/Postcode seeded on boot, /locations/* endpoints live, Photon proxy for street-level autocomplete via /locations/autocomplete.
3. Full UX redesign of all 6 onboarding screens (commit 5bf2d3a). Build 1780547633895 installed on iPhone 17 Pro sim AD109A46-2F99-43C3-8AAA-FEE68DC8499E.

Read NEXT-SESSION.md and ~/.claude/projects/-Users-Mahesh-Sangawar-personal-tesserix-new-Home-Chef-App/memory/feedback_no_editorial_style.md before any work.

State to verify first:
- Probe vendors.fe3dr.com auth path: `curl -sS -i -H "Authorization: Bearer notajwt" https://vendors.fe3dr.com/api/v1/chef/dashboard | head -3` — expect 401 JSON (NOT Istio "Jwt is not in the form").
- Photon autocomplete: `curl -sS 'https://vendors.fe3dr.com/api/v1/locations/autocomplete?q=Raheja+Exotica' | jq` — expect Photon hits.
- App should already be installed on sim AD109A46. Sim location pinned to Koramangala (12.9352, 77.6245).

User was actively walking the redesigned onboarding when the session ended. Most likely follow-ups:
- Visual polish iterations per step
- Wire up image-upload visual states (the bug was silent failures — fixed via try/catch + toasts in e177f6a; the redesign added upload-tile visual states)
- Backend asks queue (is_veg, customerName on OrderResponse, recentOrders on /chef/dashboard, GET /chef/orders/:orderId, platform charges + GST split)

Operational gotchas — re-read NEXT-SESSION.md. Specifically: use `npx eas-cli@18.4.0` (20.x silently fails); always cd into apps/mobile-vendor before running eas-cli; Kargo overrides argocd manifests so don't edit them to roll back.

Verified test chef: mahesh.sangawar@gmail.com, currently user_id dcba6316-e735-4fcd-b671-a0165e4c92b8 in business pool with no chef_profile (walking through onboarding).
```
