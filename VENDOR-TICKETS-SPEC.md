# Vendor Tickets — build spec (mobile-only)

Build a vendor support-ticket feature in `apps/mobile-vendor`: chefs raise platform
issues / feature requests, see status, and reply in a thread. **No backend changes** —
the Home Chef API already has the full ticket API; mirror Mark8ly's proven UI pattern.

## Backend already exists (do NOT rebuild)
`apps/api/models/support_ticket.go`, `handlers/support.go`, routes in `routes.go`:

- `POST /api/v1/support/tickets` — body `{ category (req), subject (req), description (req), priority?, orderId? }` → created ticket
- `GET  /api/v1/support/tickets` — → `{ "data": Ticket[], "pagination": {...} }` (raw SupportTicket JSON)
- `GET  /api/v1/support/tickets/:id` — → `SupportTicketResponse` (bare object, includes `messages[]`)
- `POST /api/v1/support/tickets/:id/messages` — body `{ content (req) }`
- `PUT  /api/v1/support/tickets/:id/close`

Routes are under the authenticated `v1` group (NOT `/chef`); reporterRole is derived from the
user. The api client (`lib/api.ts`) already adds `Authorization: Bearer` + `X-Platform`.

**Ticket fields** (list = raw struct): `id, ticketNumber, reporterId, reporterRole, category,
priority, status, subject, description, orderId?, assignedToId?, createdAt, updatedAt,
resolvedAt?, closedAt?`.
**Detail adds** `reporterName` + `messages: [{ id, ticketId, senderId, senderRole, senderName,
content, isInternal, createdAt }]`.

- **category**: `order_issue | payment_issue | account_issue | chef_complaint | delivery_complaint | technical | other`
- **priority**: `low | medium | high | urgent` (default medium)
- **status**: `open | in_progress | waiting_on_customer | waiting_on_chef | resolved | closed`

Verify first: `curl -H "Authorization: Bearer $TOKEN" -H "X-Platform: ios" https://vendors.fe3dr.com/api/v1/support/tickets` → `{"data":[],...}`.

## Mark8ly pattern to mirror
Source (read for reference): `/Users/Mahesh.Sangawar/personal/tesserix-new/mark8ly/apps/storefront/app/account/tickets/`
- **List** — rows: subject + toned **StatusChip** + ticketNumber + date.
- **Detail** — signature element is a **4-stage `TicketStatusStepper`**: *Opened → In progress →
  Resolved → Closed* (read-only; `waiting_*` collapses into "In progress"). Then a **transcript**
  (message thread, sender role + name), a **ReplyForm** (textarea → Send reply), and a **close** action.
- **Create** — subject + priority (descriptive labels) + description.

## Screens to build (pure JS — works via Metro 8082, no native module)
1. `app/support/index.tsx` — **My tickets** list + "New ticket" button. Row: subject, StatusChip,
   category label, ticketNumber, relative date. Empty state ("No tickets yet — tap New ticket…").
2. `app/support/new.tsx` — **Create**: category chips curated for vendors →
   App problem→`technical`, Payments & payouts→`payment_issue`, Account & verification→`account_issue`,
   An order→`order_issue`, **Feature request**→`other`, Something else→`other`. + subject input +
   description textarea + priority (optional, default medium). Submit → POST → replace to `/support/[id]`.
3. `app/support/[id].tsx` — **Detail**: subject + 4-stage stepper + description + transcript
   (your messages = chef/right, support = staff/left, show senderName) + reply textarea + Send +
   Close button (hidden when status is `closed`/`resolved`).
4. `hooks/useSupport.ts` — `useSupportTickets()` (list), `useTicket(id)`, `useCreateTicket()`,
   `useAddMessage()`, `useCloseTicket()`. Query keys `['chef','support', ...]`. Invalidate on mutate.
5. `app/(tabs)/more.tsx` — add entry under a Support section:
   `{ labelKey: 'support', caption: 'Report an issue or request a feature', route: '/support', Icon: LifeBuoy }`
   (lucide `LifeBuoy` or `MessageSquare`). Check how `labelKey` → label (likely `t('more.'+labelKey)`)
   and add the i18n key.

## Styling — Uber monochrome (see [[project_vendor_app_uber_monochrome]])
- **Ink** for actions (Send, New ticket, close), tabs, links.
- **StatusChip tones**: `open` → neutral grey (mist bg + ink.soft text); `in_progress`/`waiting_*`
  → amber (amber.tint + ink); `resolved` → **success green** (success.tint + success.soft);
  `closed` → muted grey. Priority `urgent`/`high` → destructive accent dot.
- Stepper: completed steps = ink fill, current = ink, upcoming = mist; resolved stage = success green.
- **iOS Pressable bug** (critical, see [[feedback_ios_pressable_array_style]]): never put layout on a
  Pressable's function-style array `style`; put layout on an inner View / children-as-function.
- i18n: app uses `t()` with `locales/en.json` + `hi.json` (keep parity). Either add keys for the
  support strings in both, or ship English now and note Hindi as a follow-up.

## Gotchas
- Test live on the sim: Metro on **8082** (`npx expo start --dev-client --port 8082`), launch via
  `homechef-vendor://expo-development-client/?url=http%3A%2F%2Flocalhost%3A8082`, then `homechef-vendor://support`.
- Token for curl tests: re-capture if `/tmp/seed-token.txt` is stale (temp `console.log` of
  `useAuthStore.getState().accessToken` in `app/_layout.tsx`, read from Metro log, then revert).
- ~100 pre-existing lucide TS errors in mobile-vendor — filter on touched files only.
- Deploy is N/A here (mobile-only; no API change).

## Definition of done
Create a ticket → see it in the list with the right status chip → open it → see the stepper +
your description → reply → reply appears in the transcript → close it → stepper shows Closed.
Verified on the sim.
