---
name: e2e-sim-testing
description: Use when running end-to-end tests of the Fe3dr customer and vendor apps on the iOS simulators with idb — driving taps and text, placing test-UPI orders, cross-checking the chef side, recording pass/fail in the test plan, and filing a GitHub issue per failure. Triggers on "e2e test the apps", "test on the simulator", "run the test plan", idb, simctl.
---

# E2E testing the Fe3dr apps on the simulators

Two apps, two simulators, one flow. Almost every customer action has a chef-side
consequence — a case is only passed when **both** sides were observed.

Product/route knowledge lives in the `fe3dr` skill. This skill is the mechanics.

## Devices

```bash
xcrun simctl list devices booted
```

| Sim | Bundle id | Metro port |
|---|---|---|
| Fe3dr Customer | `com.tesserix.homechef.customer` | 8082 |
| Fe3dr Vendor | `com.tesserix.homechef.vendor` | 8081 |

UDIDs are per-machine — resolve them once per session, never reuse them from a
transcript. A debug build bakes its Metro port in; a wrong port silently loads
another project's bundle rather than failing.

```bash
CUST=$(xcrun simctl list devices booted | grep 'Fe3dr Customer' | grep -o '[0-9A-F-]\{36\}')
VEND=$(xcrun simctl list devices booted | grep 'Fe3dr Vendor'   | grep -o '[0-9A-F-]\{36\}')
```

## Driving a device

```bash
xcrun simctl launch $CUST com.tesserix.homechef.customer
xcrun simctl io $CUST screenshot shot.png     # 3x on Pro Max
idb ui tap  --udid $CUST <x> <y>              # POINTS, not pixels
idb ui text --udid $CUST "some text"
idb ui swipe --udid $CUST <x1> <y1> <x2> <y2> --duration 0.3
```

**Coordinates.** Screenshots come back at 3× (1320×2868 on a Pro Max). Divide
every pixel coordinate you read off a screenshot by 3 before passing it to
`idb ui tap`. Getting this wrong taps the top-left eighth of the screen and
looks like "the button does nothing".

**Rules that save a run:**
- Screenshot after *every* tap. RN transitions are async; tapping blind
  compounds one missed navigation into ten bogus failures.
- The floating Expo dev-tools gear sits top-right and swallows taps aimed near
  it. Route around it, don't fight it.
- `idb ui text` types into whatever holds focus — tap the field first, confirm
  the caret in a screenshot, then type.
- Never trigger a native alert you can't see; screenshot before dismissing.
- Scrolling: swipe from ~70% height to ~30% height, same x.

## Payments — Cashfree sandbox, UPI only

The gateway is **Cashfree**, opened as the v3 web SDK inside a WebView
(`app/payment/cashfree.tsx`), not the native SDK — so it ships by OTA.

Cashfree splits sandbox from production **by host**, and there is no key prefix
to infer it from. The environment is resolved **server-side** from the chef's
`mode` (`live` / `test`) and handed to the app as `cashfreeEnv` →
the route's `env` param. Before the first payment of a run:

- `env=SANDBOX` and a WebView on `sandbox.cashfree.com` ⇒ safe to pay.
- `PRODUCTION` ⇒ the kitchen is live. **Stop, pay nothing, tell the user.**

Prefer UPI. The sandbox card row dead-ends on a saved-card vault OTP; if UPI is
unusable on the sim (no UPI app to receive an intent hand-off), fall back to
**Net Banking + OTP `111000`** and note the substitution in the plan.

**There is no client-verifiable success.** Cashfree returns no
(payment_id, signature) pair, so any non-error close routes to
`/payment/result`, which polls the server's `paymentStatus`. Treat the server as
the only authority — never record a case as paid off the WebView's own message.

## Test data hygiene

Orders placed in a run are real prod rows. Keep them identifiable and clean up:
- Order from a kitchen designated for testing, not a live chef, wherever the
  scenario allows it.
- Cancel or complete every order you open — never leave a chef staring at a
  pending order from a test.
- Record the order id in the plan row so it can be traced later.

## Recording results

The plan is a markdown table per feature area with a **Status** column:

`✅ Pass · ❌ Fail · ⚠️ Partial / blocked · ⏳ Not yet run · 🚫 N/A`

Fill the Status the moment a case finishes, not at the end of the run. For a
failure, the Notes cell must carry: what was tapped, what was expected, what
actually rendered, and the issue number. "Didn't work" is not a result.

## Filing issues for failures

One issue per failure, on `tesserix/Home-Chef-App`:

```bash
gh issue create --repo tesserix/Home-Chef-App \
  --title "fix(<area>): <observed behaviour>" \
  --label bug --label mobile --label testing \
  --body "..."
```

Body: **Steps** (numbered, from a cold app launch) · **Expected** · **Actual** ·
**Environment** (app, sim, API, build) · **Test case** (the plan row id). Attach
the screenshot when the failure is visual. Then write the issue number back into
the plan row — a failure with no issue link is an untracked bug.

Search first: `gh issue list --repo tesserix/Home-Chef-App --search "<symptom>"`.
Comment on the existing issue rather than opening a duplicate.

## Reporting

Report what was observed. A case not reached is `⏳`, not `✅`. If the run is
cut short, say which areas went unexercised — a plan with optimistic ticks is
worse than no plan.
