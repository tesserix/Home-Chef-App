# Fe3dr v1 — Owner-gated P0 launch runbook

The four remaining v1 blockers are operational and owner-gated (credentials / Apple / legal). Engineering is done; each item below is a checklist you can execute directly. Do them in any order — they're independent — but **#25 Razorpay** and **#14 APNs** should be verified before **#20 App Store submit**.

Verified-safe launch config (no change needed): delivery zones OFF, 3PL providers dark (`is_enabled=false`), money-flow flags default OFF, Razorpay keys secret-driven, force-upgrade env-driven.

---

## #25 — Razorpay live switch

**Goal:** move payments from test to live and confirm one real ₹ charge settles.

1. **Get live keys** from the Razorpay Dashboard → Settings → API Keys → *Regenerate Live Keys*. You get `rzp_live_...` (key id) + a secret (shown once).
2. **Update the secret** `homechef-api-secrets` (GCP Secret Manager) — set:
   - `RAZORPAY_KEY_ID` = `rzp_live_...`
   - `RAZORPAY_KEY_SECRET` = live secret
   - `RAZORPAY_WEBHOOK_SECRET` = (set in step 4)
   The app auto-detects live vs test from the `rzp_live_` / `rzp_test_` prefix (`handlers/admin.go:906`), so no code/flag flip is needed.
3. **Roll the API** so it picks up the new secret (ESO refresh → pod restart, or the admin settings endpoint's `InvalidateRazorpay`).
4. **Register the live webhook** in the Razorpay Dashboard → Settings → Webhooks:
   - URL: `https://api.fe3dr.com/webhooks/razorpay` (HMAC-verified, no auth — `handlers/payment.go:909`)
   - Events: `payment.captured`, `payment.failed`, `refund.processed` (at minimum)
   - Copy the signing secret into `RAZORPAY_WEBHOOK_SECRET` (step 2) and re-roll.
5. **Smoke a live txn:** place a real low-value order in the customer app on TestFlight, pay with a real card/UPI, confirm: order → `paid`, webhook received (check API logs), and a **refund** from the chef app settles. Refund the smoke charge afterward.

**Done when:** a live ₹ charge captures, the webhook marks the order paid, and a refund processes. (#6 test happy-path is already CLOSED → this is unblocked.)

---

## #14 — APNs production certificate (iOS push)

**Goal:** real push notifications on TestFlight/production iOS builds. Delivery is FCM → APNs, so the Apple key goes into Firebase.

1. **Apple Developer → Certificates, Identifiers & Profiles → Keys** → create an **APNs Auth Key** (`.p8`). Note the **Key ID** and your **Team ID** (`2CRHRRYBPL`, Tesserix Pty Ltd).
2. **Firebase console** (project `tesseracthub-480811`) → Project Settings → Cloud Messaging → *Apple app configuration* → upload the `.p8`, enter Key ID + Team ID + the iOS bundle IDs (customer + vendor).
3. **EAS credentials:** confirm the push key is registered for each app: `eas credentials -p ios` (per app dir) → Push Notifications. EAS can reuse the same APNs key.
4. **Verify on TestFlight:** install the current TestFlight build, grant notifications, trigger a push (place/advance an order so the API fires an FCM push) and confirm it arrives on the locked device.

**Done when:** a backend-triggered push lands on a physical device from a TestFlight build (both apps).

---

## #19 — Legal counsel sign-off

**Goal:** fill the legally-required specifics; policy bodies are already code-complete.

Entity is **Tesserix Pty Ltd** (ACN 694 070 865 / ABN 59 694 070 865, NSW; NSW governing law, DPDP-aligned) with an India GST contracting entity as residual.

1. **GSTIN** — insert the India GST number wherever the invoice/legal templates expect it (invoice PDF footer, Terms).
2. **Grievance Officer** — name + contact (DPDP/IT-Rules requirement for India) in the Privacy policy.
3. **DPO email** — `dpo@fe3dr.com` (provision the mailbox/alias).
4. **Counsel review** — have legal confirm Terms, Privacy, EULA, refund/cancellation, and the Grievance mechanism read correctly for the AU entity + India operations.
5. Rebuild the apps if any of these are baked into the bundle rather than fetched (legal screens are in-app: `app/terms.tsx`, `privacy.tsx`, `eula.tsx`).

**Done when:** counsel signs off and the three specifics (GSTIN, Grievance Officer, DPO email) are live in-app.

---

## #20 — App Store submission (both apps)

**Goal:** submit customer + vendor apps to App Store review (India listing). Pipeline is zero-touch.

1. **Build → TestFlight:** push a tag or dispatch the workflow:
   - Customer: `git tag customer-v1.0.0 && git push origin customer-v1.0.0` (glob `customer-v*`, `.eas/workflows/ios-testflight.yml`), or Actions → *ios-testflight* → Run workflow.
   - Vendor: `git tag vendor-v1.0.0 && git push origin vendor-v1.0.0` (glob `vendor-v*`), or dispatch.
2. **App Store Connect** (App IDs are owner-gated — confirm both apps exist):
   - Screenshots: required sizes (6.7" + 6.5" + 5.5", iPad if enabled) for each app.
   - IN listing copy: name, subtitle, description, keywords, support URL, privacy policy URL (`https://fe3dr.com/...`).
   - Data-safety / privacy questionnaire (matches the Privacy policy from #19).
   - Age rating, category (Food & Drink).
3. **Attach the TestFlight build**, submit for review. Provide a demo reviewer account (customer: `demo@test.com` / `Test@123`; create a vendor demo login) and reviewer notes explaining the chef-only, delivery-area-scoped flow.
4. Respond to any reviewer rejections; resubmit.

**Done when:** both apps are *Waiting for Review* / *In Review* with complete IN listings.

---

## Optional engineering (not owner-gated — can be driven separately)
- **#8** full E2E order-lifecycle test — money-math lifecycle test is automatable; gateway/push/PDF/email paths need the live infra above (see the on-device checklist that ships with #8).
- **#11** a11y + bundle audit — static audit done; quick wins landing via a focused PR.
- **#17** Cloudflare WAF — post-launch.

## Parallel
- **Borzo 3PL** — parked on the owner's test `X-DV-Auth-Token`. On paste: live-validate (calculate→create→status→cancel), store in Secret Manager, flip the DB provider `is_enabled=true`, PR, deploy. The delivery-area discovery gate (#372) automatically loosens once a 3PL is live.
