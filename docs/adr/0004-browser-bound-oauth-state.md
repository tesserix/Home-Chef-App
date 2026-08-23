# ADR-0004: Browser-bound stateless OAuth state

## Status

Accepted — 2026-08-23

## Context

The auth BFF runs at least two replicas, but its authorization-code flow kept
OAuth state in one replica's memory. A callback routed to another replica was
rejected even when the login was valid. Sticky routing would reduce but not
remove this failure during rollouts or pod loss.

An OAuth state entry is below 1 KiB and lives for five minutes. At a planning
ceiling of 100 login starts per second, a shared store would hold at most about
30,000 short-lived entries. The auth path retains the service's existing
availability target and should add less than 1 ms p99 of in-process work; exact
production login rate and latency telemetry remain an operational measurement.

The state must bind the callback to the browser that began login, protect the
OIDC nonce, app name, and return path from tampering, tolerate callbacks on any
replica, and resist replay. A copied callback URL alone must be insufficient.

## Options considered

1. Keep the in-memory store and add sticky sessions. Rejected because replica
   loss and rolling replacement still discard valid state.
2. Store one-shot state in Valkey. This gives atomic consumption but adds a new
   critical dependency and auth-BFF deployment configuration for a five-minute
   browser round trip.
3. Seal state with AES-GCM and bind it to a per-login browser cookie. Chosen
   because every replica already has the session encryption key and can verify
   the state without network I/O.
4. Sign serialized state without browser binding. Rejected because anyone who
   obtains the callback URL could replay it from another browser.

## Decision

Derive a dedicated 256-bit OAuth-state key from the existing session encryption
key with HMAC-SHA-256 domain separation. Seal each state entry with AES-GCM and
a fresh nonce. Put a random binding verifier in a host-only, `HttpOnly`,
`SameSite=Lax`, callback-path cookie; include only its SHA-256 digest inside the
sealed envelope. Use a distinct cookie name per login so parallel tabs do not
overwrite one another.

The callback expires the binding cookie before completing validation. It
rejects missing or mismatched cookies, malformed or modified envelopes, future
issuance, expiry, and app or OIDC nonce mismatch. The identity provider's
authorization code remains single-use, closing the concurrent replay window
that a browser cookie alone cannot atomically consume.

Production cookies use the `__Secure-` prefix. Local HTTP development uses the
same host-only attributes without `Secure`, because browsers reject
`__Secure-` cookies over HTTP.

## Consequences

- Login and callback can land on different replicas with no shared state.
- A state token or callback URL copied without the binding cookie is useless.
- Valkey downtime does not become an authentication outage.
- Rotating the session key invalidates outstanding logins for at most five
  minutes; existing sessions are already subject to the same rotation.
- A callback failure consumes only that login's verifier cookie and fails
  closed. Other parallel login attempts remain usable.
- The provider's single-use code is part of replay protection; replacing the
  provider requires preserving that guarantee or adopting an atomic shared
  store.

## Migration and rollback

Deployment is compatible with existing clients because the public OAuth
parameters and callback endpoint do not change. During a mixed rollout, a new
login handled by an old pod or vice versa can fail once; a retry starts a fresh
five-minute flow. Roll back by restoring the prior store implementation. No
database or cache data needs migration or cleanup.

The runtime cost is local hashing and authenticated encryption plus one
short-lived browser cookie; there is no additional infrastructure charge.
