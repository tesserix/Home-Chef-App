# First-10-Chefs Concierge Script

For the first 10 kitchens, onboard them **by hand** and watch the system live. The goal isn't scale — it's catching every rough edge before it compounds, and making the first chefs feel personally taken care of.

## Before the call (you, 10 min)

- Have the chef's phone number + the city they cook in.
- Open three things side by side:
  - Sentry (`homechef-api`) — watch for errors as they tap.
  - Cloud Logging filtered to their requests — grep the `X-Request-ID` / `correlation_id` from a request, or filter by their user id once created.
  - The admin view of their approval queue.

## On the call (45–60 min, screen-share or in person)

1. **Install** — TestFlight (iOS) / internal track (Android). Confirm they're on the latest build.
2. **Sign up + onboarding wizard** — watch them go through it. Note every spot they hesitate. Common snags: FSSAI number format, GSTIN optional-but-confusing, address autofill.
3. **Documents** — FSSAI license upload. Confirm it lands in the approval queue; approve it live so they feel the loop close.
4. **First menu item** — add one dish with a photo + price + HSN (leave default if unsure). Confirm it renders.
5. **Payout details** — bank or UPI. Confirm the masked values save (sensitive fields go to Secret Manager, not the DB).
6. **Go "Open"** — flip the kitchen open. Show them the **Pause** options (Back in 15/30/60) for when they step away.
7. **Place a test order** (you, from a customer account) → walk them through accept → prep → ready. Then **deliver** it and show the auto-emailed GST invoice + that it appears in **Earnings**.
8. **Earnings tour** — breakdown (commission/GST/TDS/net), weekly **Statements**, **TDS certificate**, **Refunds** section. This is where trust is won — they see exactly what they'll be paid.

## Watch for (live, while they tap)

- Any Sentry error tied to their `correlation_id` → triage immediately, tell them "one sec, fixing that."
- 426 upgrade wall → they're on an old build; push them the latest.
- Push permission denied → re-prompt; deep-link tap-through won't work without it.
- Slow first request → confirm min-scale 1 kept a pod warm (should be instant).

## After (you, 15 min)

- Send a thank-you + your direct number ("text me anything, anytime, for the first month").
- Log the 3 worst friction points. After 10 chefs, the repeated ones become the next sprint.
- Verify their weekly statement generates that Sunday→Monday (check `weekly-statement: scan complete` in logs).

## The bar

A first chef should finish the call with: a live kitchen, an approved license, a menu item, a completed test order, and the feeling that a real person has their back. If any of those didn't happen, the call isn't done.
