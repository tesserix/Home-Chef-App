# fe3dr runtime secret backend (preparation)

Tracks https://github.com/tesserix/tesserix-k8s/issues/1159.
This code is not enabled in production. It prepares the direct runtime consumers;
copying values and changing ESO cannot migrate these calls by themselves.

## Selection and scope

`APP_SECRET_STORE=openbao` selects OpenBao for payment gateway settings and
vendor/driver payment fields. `PII_SECRET_STORE=openbao` is a separate opt-in so
PII can move last. Both default to GCP. Unknown selectors fail initialization;
OpenBao request failures do not fall back to GCP or create a second writer.

Set `OPENBAO_ADDR` to the in-cluster service, `OPENBAO_ROLE` to an approved runtime
role and `OPENBAO_PII_ROLE` to the read-only API role. The client reads the mounted
Kubernetes service-account token for login and re-authenticates before lease
expiry. It uses bounded requests, refuses redirects and does not include remote
response bodies or values in errors. The exact runtime write policy is a
separate GitOps prerequisite; no grant is created by this application code.

A GCP identifier `prod-homechef-<name>` maps to
`kv/data/homechef/homechef-api/fe3dr-<name>`, field `value`. The backend accepts
only production fe3dr identifiers. Credentials preserve bytes/newlines. KV v2
updates create versions; payment-owner deletion destroys metadata/all versions
to preserve the existing account-erasure semantics. Gateway configuration cannot
be deleted through that method. The server policy must independently enforce
read/create/update scope and payment-owner-only metadata deletion.

PII base64 decoding and KMS unwrap are unchanged. The existing wrapped DEK and
blind-index key must be copied exactly; no re-keying or new encryption scheme is
part of this migration. KMS, storage and recovery identities remain in GCP.

## Gates before enabling either switch

1. The tested `APP_SECRET_WRITES_PAUSED=true` guard rejects all seven
   payment-detail/gateway mutation endpoints with HTTP 503 and Retry-After,
   blocks secret-service writes/deletes, and defers background vendor-account
   erasure before its side effects. Empty/`false` leaves writes enabled; unknown
   settings fail closed. Reads are unaffected. Obtain scoped production pause
   approval before enabling it; it has not been deployed or enabled.
2. Fix onboarding's fire-and-forget secret-write success reporting. It currently
   logs failures after committing other state. A new backend must not introduce
   silent missing or partial payment details. Define retry/idempotency handling.
3. Deploy pause-capable code while still on GCP. Wait until all old pods and
   outstanding writers are drained; then reconcile source changes since staging.
4. Verify exact runtime capabilities and test sandbox read/write/delete, failure,
   lease refresh and authorization cases. Coordinate the Cashfree sandbox pair
   also consumed by Dwellm8. Do not make real payment transactions as a test.
5. Switch payment clients using a single write authority, verify readiness and
   functionality, then unpause. Rollback after new writes requires reverse
   reconciliation; switching an environment variable back alone is insufficient.
6. Move PII separately after existing-data decryption/blind-index and restore
   evidence. Keep source versions until all readers and writers are verified.
7. Delete only fully migrated GCP sources with independently verified recoverable
   captures. Keep the migration issue open for every unresolved gate.

The draft adapter has HTTP-boundary and service-routing tests, plus the existing
PII/service suites. Those are preparation evidence, not proof of safe live
cutover. Startup health alone does not prove payment-provider functionality or
existing encrypted-data compatibility.
