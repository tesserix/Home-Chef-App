# In-app FSSAI registration request

**Owner decisions, 5 Aug 2026.** A chef asks us, from the vendor app, to obtain
their FSSAI registration. They pay us, upload their documents, and we file the
application on FoSCoS on their behalf. The request appears in the tesserix-home
admin **and** in the `chef-onboarding@fe3dr.com` inbox, and the chef tracks its
status in the app.

Public-facing counterpart already live: `fe3dr.com/fssai/` (PR #1004).

## The money — all-in, we pay FSSAI

The chef pays **one** amount. Unlike the website's self-serve path, they do not
pay FSSAI separately; we do, out of what they paid us.

| | 1 year |
|---|---:|
| Charged to the chef (₹150 + 18% GST) | **₹177.00** |
| GST collected, owed to government | −₹27.00 |
| Net receipt | ₹150.00 |
| FSSAI fee (₹100 + 18% GST) | −₹118.00 |
| Cashfree MDR (≈2% + GST on ₹177) | ≈ −₹4.18 |
| **Owner's margin** | **≈ ₹27.82** |

Longer terms multiply the government fee only (₹100/year), never the service fee
— it is one form regardless of term.

### GST — verified, and it changed the model

FSSAI licensing/registration **was** exempt under Entry 47 of Notification
12/2017 (government services for a registration required by law). **That
exemption was withdrawn with effect from 18 July 2022**, so FSSAI's fee now
carries 18% GST. The 18% rate applies to both legs: FSSAI's fee and our service
fee.

**Zivana is GST-registered** (confirmed by the owner, 5 Aug 2026), so the ₹18
GST on the FSSAI fee is claimable as **input credit** — the working margin is
therefore **≈ ₹45.82**, not ₹27.82. The ₹27.82 figure in the table above is the
cash position before that credit is claimed.

⚠ **Items for the CA:**
- **Pure agent (CGST Rule 33)** may allow the government fee to be excluded from
  the value of our supply, which would change what we charge GST on. Not
  assumed here: we currently charge 18% on the full ₹150.
- **Unverified:** whether FoSCoS collects ₹100 or ₹118 at its payment screen.
  The published fee table says ₹100. One screenshot at checkout settles it.

### Nothing is hardcoded

All four figures are **PlatformSettings**, admin-editable at runtime per the
repo's existing pricing convention — the service fee, the GST rate, the
government fee per year, and whether the flow is enabled at all. No deploy is
needed to correct any of them once the CA rules.

## Documents come with the payment

**Owner's call (revised 5 Aug 2026): the chef pays first *along with* the
documents.** The chef assembles the whole request while it is unpaid, and paying
is the last act. This removes the paid-but-incomplete hole the earlier
pay-then-upload ordering created:

- The row is created as an **unpaid draft** (`awaiting_payment`) so documents
  have something to attach to. Nothing is charged and nothing is owed.
- `POST /checkout` **refuses to mint a Cashfree order** while
  `NeedsDocuments()` — so is `MarkFssaiPaid`. The guard is on the money path,
  not only in the UI.
- Capture moves the request straight to `submitted` and fires the onboarding
  email. There is no state in which we hold a chef's money without the documents
  we need to file.

This is what makes the fee safe to declare non-refundable: every paid request is
one we can actually act on.

## Non-refundable, and cancellable only before that

The fee is non-refundable from the moment it is taken, and the chef is told so
by the server (`nonRefundable` / `nonRefundableNotice` on the quote) *before*
they pay. Until payment they may edit, replace documents, or discard the draft
outright. After it, `DELETE /chef/fssai/requests/:id` answers 409.

`refunded` survives as an **admin-only** terminal state — a chargeback or a
refund we are obliged to make still has to be recordable — but no chef action
reaches it.

## Lifecycle

```
awaiting_payment → submitted → in_progress → filed → issued
                                              └→ rejected → refunded
```

The chef's tracker shows four steps (Submitted · In progress · Filed · Issued,
plus rejection); `awaiting_payment` renders as the form, not a status. Admin
moves every state after `submitted` from tesserix-home, and the admin queue
excludes unpaid drafts — they are a chef mid-form, not work. `filed` requires an
`ApplicationRef` — the FoSCoS reference is the single most useful thing we hand
back, because it lets the chef track their own application independently of us.

One open request per chef at a time (`FssaiRequestOpen`); an abandoned unpaid
draft is discarded when they start again rather than locking them out.

## Documents

Exactly what FSSAI asks for, quoted from *Documents required for Registration
Certificate*:

1. `photo` — passport-style photo of the applicant. **Required.**
2. `identity` — government photo ID (Aadhaar, PAN, Voter ID). **Required.**
3. `address_proof` — only where the kitchen address differs from the address on
   that ID. **Conditional**, and only the chef can say — never demanded.

Files go to GCS through the existing upload path; the row holds the reference.

⚠ **PII.** These are exactly the identity documents `#710` column encryption
exists to protect. Two consequences: the onboarding email should carry links,
not attachments, and the documents must be purged on account deletion with the
rest of the chef's PII.

## Surfaces

1. **Vendor app** — a request screen (form → pay → upload) and a status tracker.
2. **Admin (tesserix-home)** — the queue, the document links, and the state
   transitions. Reached through the existing `/admin/*` HMAC proxy, so no new
   proxy code.
3. **Email** — `chef-onboarding@fe3dr.com` at `submitted`, carrying the kitchen
   details, the applicant's details, the term, and links to the documents.

## API

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/chef/fssai/requests` | Create the unpaid draft |
| `POST` | `/chef/fssai/requests/:id/upload` | Upload + attach a document (multipart) |
| `DELETE` | `/chef/fssai/requests/:id/documents/:kind` | Drop an optional document |
| `POST` | `/chef/fssai/requests/:id/checkout` | Mint the Cashfree order — refused while documents are missing |
| `POST` | `/chef/fssai/requests/:id/confirm` | Verify capture → `submitted` + email |
| `DELETE` | `/chef/fssai/requests/:id` | Discard an unpaid draft |
| `GET` | `/chef/fssai/request` | The chef's current request, for the tracker |
| `GET` | `/admin/fssai/requests` | Admin queue (paid + open by default) |
| `PATCH` | `/admin/fssai/requests/:id` | Status, ApplicationRef, RegistrationNo, notes |

There is deliberately **no endpoint that accepts a file reference from the
client.** The stored value is an object path that later gets signed, so a client
able to choose it could read another chef's identity documents. Upload and
attach are one call.

## Where a chef finds it

| Surface | Entry point |
|---|---|
| Vendor app | More → Requests → FSSAI registration |
| Vendor app | Profile → Licensing (existing chefs) |
| Vendor app | Onboarding → the pending screen, while under review |
| Vendor app | Onboarding → documents step, as a note (no chef profile exists yet) |
| Vendor web | Sidebar → FSSAI Registration |
| Vendor web | Profile → Documents section, when no licence is on file |
| Vendor web | Onboarding → documents step, as a note |

The offer card is one self-hiding component per platform: it renders nothing
when the service is switched off, and becomes a tracker link once a request
exists.

## Build order

1. Model + migration. ✅ `models/fssai_request.go`
2. PlatformSettings pricing keys + the quote function. ✅
3. Service: create/upload/checkout/confirm, with the money tests. ✅
4. Chef handlers + routes. ✅
5. Admin handlers + routes. ✅
6. Onboarding email template. ✅
7. Vendor app: request flow + tracker + entry points. ✅
8. Vendor web (`vendors.fe3dr.com`): same flow, same entry points. ✅
9. **tesserix-home admin queue — not done, separate repo.** The API is ready
   (`/admin/fssai/requests`); the operator UI has to be built there.
10. **Reconcile `fe3dr.com/fssai/` copy — not done.** The live page describes
    the chef paying FSSAI directly with us charging ₹50 + GST on top. The in-app
    flow is all-in at ₹177. Two different offers for one product: the page must
    present both, or the self-serve path must be restated.
