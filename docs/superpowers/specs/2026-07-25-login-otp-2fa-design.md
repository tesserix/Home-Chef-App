# Login OTP (2FA) with trusted devices

**Date:** 2026-07-25
**Status:** Approved for implementation

## Problem

Login is single-factor today. Anyone holding a user's password — or a
Google/Apple account — is that user. For chefs and admins that means access to
payouts, customer addresses and refund controls.

Add a second factor at login, delivered to the registered email or the
registered phone, with an option to remember a device so the challenge does not
repeat on every sign-in.

## Decisions

| Question | Decision |
|---|---|
| Who gets it | Opt-in per account. Nobody is forced; no existing user is locked out. |
| When it fires | Every fresh login from a device that is not remembered. |
| Channels | Email (existing SendGrid→Resend path) and phone (Firebase phone verification). |
| Remember device | Persists until explicitly revoked. No time-based expiry. |
| Recovery | Either enrolled channel, plus 8 one-time backup codes, plus an audited admin reset. |
| Admin reset | 24-hour durable hold with a user cancel signal; two-admin approval to apply immediately. |
| Enforcement point | The Go API, at the resource boundary. |

### Why the API owns the gate

Authentication is GIP (Firebase). Mobile signs in with the Firebase SDK and
calls the API with a Bearer ID token; web portals use `apps/auth-bff`, which
holds an HttpOnly cookie session and forwards HMAC-signed identity headers.

Putting the gate in the API means enforcement sits at the resource boundary,
where it cannot be bypassed, and it reuses machinery that is already
production-hardened: `services/email_otp.go` (Redis codes, resend cooldown,
attempt caps, hourly send windows), the PII encryption scheme, the audit
service and RBAC.

The alternative — gating in auth-bff — was rejected because mobile authenticates
directly to the API and only consults the BFF for identity resolution, and
because the BFF has no OTP service, no email sender, no user table and no device
store. All four would have to be rebuilt there.

GIP-native MFA was rejected outright: it supports SMS and TOTP only, so it
cannot serve the email channel at all, and it has no trusted-device primitive.

## Architecture

Feature flag `MFA_ENABLED` in `config.go`, default off, matching the repo's
env-flag convention.

| Unit | Responsibility |
|---|---|
| `services/otp_challenge.go` | Purpose-namespaced OTP core. Generalises the existing email OTP key/cooldown/attempt logic rather than forking it. |
| `services/mfa.go` | Enrollment state and challenge orchestration. |
| `services/trusted_device.go` | Issue, verify and revoke device tokens. |
| `services/backup_codes.go` | Generate and redeem one-time codes. |
| `middleware/mfa_gate.go` | Enforcement. |
| `handlers/mfa.go` | HTTP surface. |
| `temporal/workflows/admin_mfa_reset.go` | Durable hold on admin-initiated resets. |

### Purpose namespacing (security-critical)

`IsEmailOTPVerified` writes a persistent marker at `email_otp:ok:<uid>:<email>`.
If login MFA reused those keys, every user who verified their email during
onboarding would silently auto-pass the login challenge.

The OTP core therefore keys every entry by purpose:

```
otp:<purpose>:code:<uid>:<subject>
otp:<purpose>:att:<uid>:<subject>
otp:<purpose>:cd:<uid>:<subject>
otp:<purpose>:snd:<uid>:<subject>
```

Purposes are `onboarding_email`, `mfa_enroll` and `mfa_login`. A marker from one
purpose can never satisfy another. The existing onboarding call sites keep their
current key shape so live verifications are not invalidated on deploy.

### Data model

New tables live in
`tesserix-k8s/charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql`
— no SQL in this repo.

**`user_mfa_settings`** — `user_id` PK, `enabled`, `email_enrolled`,
`phone_enrolled`, `phone_e164_enc`, `phone_bidx`, `enrolled_at`, `disabled_at`,
`updated_at`.

The MFA phone is stored **separately from `users.phone`**, encrypted with the
existing PII scheme. Sharing the column would mean a profile edit silently
relocates the second factor, which is an account-takeover path.

**`mfa_backup_codes`** — `id`, `user_id`, `code_hmac`, `used_at`, `created_at`.
Codes are stored as HMAC-SHA256, never plaintext.

**`trusted_devices`** — `id`, `user_id`, `app`, `token_hash`, `label`,
`platform`, `created_at`, `last_seen_at`, `revoked_at`.

Device tokens are 32 random bytes, SHA-256 at rest, returned to the client
exactly once, and scoped to one user *and* one app — a stolen vendor token
cannot trust the admin app.

Ephemeral challenge state (code hash, attempts, cooldown) stays in Redis.

## Flows

### Enrollment

Settings → Security → "Turn on two-factor".

- **Email:** OTP with purpose `mfa_enroll` to the registered address.
- **Phone:** the client runs Firebase phone verification; Google sends the SMS.
  The client posts the resulting phone credential to the API, which verifies it
  with the Firebase Admin SDK and stores the E.164 number encrypted. Possession
  is proven by Google, not by us.

`enabled` flips true only once at least one channel is enrolled **and** the user
acknowledges the 8 backup codes, which are shown exactly once.

### Login challenge

1. Client authenticates with GIP, calls the API with `X-Device-Token` if held.
2. `MFAGate` returns `403 mfa_required` with masked channel hints.
3. `POST /auth/mfa/challenge {channel}` delivers the code.
4. `POST /auth/mfa/verify {challenge_id, code | firebase_credential | backup_code}`
   elevates the session (Redis marker, TTL = session lifetime). With
   `remember_device: true` a device token is minted and returned once.

The gate allowlists `/auth/mfa/*`, health and version. Everything else is
refused until elevated.

### Trusted devices

Stored in SecureStore on mobile and localStorage in the web portals, sent as
`X-Device-Token`. Verified by hash; `last_seen_at` updated on use.

Revoked individually or all at once from settings, and automatically on password
change, on 2FA disable, and on admin reset. There is no time-based expiry, by
decision.

### Recovery

Either enrolled channel satisfies the challenge. Backup codes are single-use,
with a low-balance warning at two remaining. Admin reset runs the Temporal
workflow below.

## NATS

Fan-out only, never in the critical path. Subjects follow the existing
`notifications.*` pattern:

`auth.mfa.enabled`, `auth.mfa.disabled`, `auth.device.trusted`,
`auth.device.revoked`, `auth.mfa.backup_codes_low`, `auth.mfa.failed_burst`.

These drive the security emails that make 2FA protective — "a new device signed
in", "two-factor was turned off" — plus audit records.

## Temporal

`AdminMFAResetWorkflow`: notify the user, wait a durable 24 hours, then apply
the reset unless a `cancel` signal arrives. An `expedite` signal carrying a
second admin's ID applies it immediately.

A compromised admin account therefore cannot instantly take over an account, and
the legitimate owner gets a window to stop it.

## What deliberately stays synchronous

The OTP send. Publishing a login code to NATS and returning `200 {"sent": true}`
would tell the user their code is on the way when the consumer might be down —
exactly the failure that left push notifications silently broken for a day when
the worker's long-polls were being reset. Email delivery is a direct call with
the SendGrid→Resend fallback, and the endpoint returns a real error if it fails.

No Temporal in the challenge path either; it must be sub-second.

## Error handling

| Condition | Response |
|---|---|
| 2FA on, device untrusted, not elevated | `403 mfa_required` + masked channels |
| Wrong code | `400 invalid_code`, attempt counter incremented |
| Attempts exhausted | `429 too_many_attempts`, challenge destroyed, re-challenge required |
| Resend inside cooldown | `429 resend_cooldown` with `retry_after` |
| Hourly send cap hit | `429 send_limit` |
| Email delivery failed | `502 delivery_failed` — never a false success |
| Backup code reused | `400 invalid_code`, indistinguishable from a wrong code |
| Device token unknown or revoked | Treated as untrusted; challenge proceeds |

Responses never reveal whether an address exists, and never echo the full email
or phone.

## Testing

- **OTP core:** purpose isolation (an `onboarding_email` marker must not satisfy
  `mfa_login`), cooldown, attempt ceiling, send window, code expiry.
- **Trusted devices:** hash-at-rest, app scoping, revocation cascades.
- **Backup codes:** single use, HMAC comparison, low-balance event.
- **Gate:** allowlist correctness, flag off = no-op, enabled-but-untrusted =
  403, elevated = pass.
- **Workflow:** cancel before the timer, expedite with two admins, apply after
  the hold.

Go tests use the existing `miniredis` and sqlite harness. Note the recorded
gotcha: sqlite bare-`Scan` of a UUID needs an explicit cast.

## Rollout

1. Schema to `tesserix-k8s`, applied by the bootstrap CronJob.
2. API behind `MFA_ENABLED=false`.
3. Enable the flag; the feature is opt-in, so nothing changes until a user
   turns it on.
4. Client UI ships per app; mobile requires a TestFlight/Play build.

## Out of scope

TOTP authenticator apps, WebAuthn/passkeys, and risk-based challenges. The
channel abstraction leaves room for TOTP later.
