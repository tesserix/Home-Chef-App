# Security review — 2026-09-29

Reviewed from `fd604eb7`: API authentication and WebSocket authorization,
selected upload ownership/content validation paths, API/auth-BFF Go dependencies,
and the JavaScript workspace lockfile. This is a scoped review, not a claim that
every application path has been penetration tested.

## Fixed findings

1. **Account checks failed open on database errors.** A signed request for a
   suspended account reached the handler with HTTP 200 when the users query
   failed. Authentication now returns a generic HTTP 503 without executing the
   handler. This preserves sessions during an outage without bypassing account
   status enforcement; missing accounts still return 401.
2. **WebSocket tickets bypassed account checks.** A valid ticket continued to
   authorize suspended, deleted, or purged users. Ticket authentication now
   enters the same account enforcement path as signed HTTP requests. Regression
   tests cover active, suspended, deleted, purged, and database-unavailable cases.
3. **Empty-key signature verification accepted forged identities.** HMAC
   verification now rejects an empty configured key. This is defense in depth:
   normal API startup already requires a decoded key of at least 16 bytes.

Each authentication regression was observed failing before its fix. No endpoint
or role grant was added. Existing account and object authorization remain in
force, and ticket-authenticated requests now receive the account checks too.

## Dependency remediation

Updated Next.js, Vitest/coverage, xmldom, js-yaml, sharp, undici, browserslist,
baseline-browser-mapping, and decode-uri-component, including the lockfile.
Adapted the web test configuration and explicit Node type inclusion for Vitest 4.

`pnpm audit --json` moved from 24 advisories (2 critical, 15 high, 7 moderate)
to 2 high advisories, both for the version of `image-size`. Those two findings
are already mitigated by the existing `image-size@1.2.1` pnpm patch. All three
malicious ICNS/JXL/HEIF parser regression tests pass. The patch is retained to
preserve the older image-size API required by consumers; findings are not hidden
with audit exclusions. The landing site exports static files and disables the
Next.js image optimizer, limiting exposure to the reported Next.js server flaws.

`govulncheck ./...` in both `apps/api` and `apps/auth-bff` reports no reachable
vulnerabilities. It still reports three advisory-bearing required modules in
each module, with no vulnerable call paths; the BFF has one such imported
package. These are not represented as an entirely clean module inventory.

## Verification

- API: `go test -race ./...`, `go vet ./...`, `go build ./...`, and `gofmt -l`
  on changed Go files pass.
- Workspace: `pnpm test:security-dependencies` passes all 7 tests.
- Vitest: customer web, vendor portal, and mobile-shared pass 333 tests total;
  delivery portal has no Vitest tests (`--passWithNoTests`).
- Landing: `pnpm --filter @homechef/web-landing test` passes 3 tests.
- Web, vendor, delivery, and landing typecheck, lint, and production builds pass.
  Lint retains existing warnings (no errors); builds retain CSS/chunk warnings.
- Mobile-shared has no typecheck script; its 219 Vitest tests were run. Native
  mobile binaries and authenticated manual user journeys were not exercised.

## Delivery

Use the existing main-branch image builds, deploy-branch promotion, Kargo, and
Argo CD. No database migration, credential rotation, live configuration mutation,
or additional infrastructure is required. The deployment adds no resources.
Live rollout status is verified separately after merge.
