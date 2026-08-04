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

⚠ **Two items for the CA, both of which move the margin:**
- The ₹18 GST on the FSSAI fee should be claimable as **input credit** if Zivana
  is GST-registered → margin ≈ ₹45.82 rather than ₹27.82.
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

## Payment first — and the hole it opens

**Owner's call: pay first, then upload.** The consequence is a chef who pays and
never uploads, so it is designed for rather than discovered later:

- The row is created **at payment initiation** (`awaiting_payment`), so money is
  never taken against a row that does not exist.
- After capture the request sits in `awaiting_documents` — a first-class state
  with its own admin queue and a refund path, not an edge case.
- The onboarding email fires only at `submitted` (paid **and** documented),
  because there is nothing to send before that.

## Lifecycle

```
awaiting_payment → awaiting_documents → submitted → in_progress → filed → issued
                                                                     └→ rejected → refunded
```

The chef's tracker shows five steps (Submitted · In progress · Filed · Issued,
plus rejection); `awaiting_documents` renders as an "upload your documents"
prompt rather than a status. Admin moves every state after `submitted` from
tesserix-home. `filed` requires an `ApplicationRef` — the FoSCoS reference is
the single most useful thing we hand back, because it lets the chef track their
own application independently of us.

One open request per chef at a time (`FssaiRequestOpen`).

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
| `POST` | `/chef/fssai/requests` | Create + mint the Cashfree order |
| `POST` | `/chef/fssai/requests/:id/confirm` | Verify capture → `awaiting_documents` |
| `POST` | `/chef/fssai/requests/:id/documents` | Attach a document; completes → `submitted` + email |
| `GET` | `/chef/fssai/request` | The chef's current request, for the tracker |
| `GET` | `/admin/fssai/requests` | Admin queue |
| `PATCH` | `/admin/fssai/requests/:id` | Status, ApplicationRef, RegistrationNo, notes |

## Build order

1. Model + migration. ✅ `models/fssai_request.go`
2. PlatformSettings pricing keys + the quote function.
3. Service: create/confirm/attach/submit, with the money tests.
4. Chef handlers + routes.
5. Admin handlers + routes.
6. Onboarding email template.
7. Vendor app: request flow + tracker.
8. tesserix-home admin queue (separate repo).
9. Reconcile `fe3dr.com/fssai/` copy: the page currently describes the chef
   paying FSSAI directly and us ₹50 + GST on top. The in-app flow is all-in at
   ₹177. Two different offers for one product — the page must present both, or
   the self-serve path must be restated.
