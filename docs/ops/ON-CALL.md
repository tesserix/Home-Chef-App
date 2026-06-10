# Home Chef — On-Call (solo founder)

Even solo, define **what wakes you** vs. what waits until morning. The goal: protect sleep from noise, but never miss real money/trust-breaking failures.

## What triggers a page (P1 — act now, any hour)

A page = phone alarm, not just an email. Wire these via Sentry alert rules + an UptimeRobot/BetterStack monitor on the health endpoints.

| Condition | Why it's a page |
|---|---|
| `vendors.fe3dr.com/api/v1/mobile/min-version` down > 3 min | Whole API is unreachable — chefs can't operate |
| `auth.fe3dr.com/health` down > 3 min | No one can log in |
| Sentry: spike > 20 errors in 5 min on `homechef-api` | Something broke broadly |
| Razorpay capture/refund failure rate > 25% over 10 min | Money is failing |
| `reconciliation DRIFT` with a refund mismatch | Money moved without a record — investigate same day |

## What waits until morning (P2 — review with coffee)

- A single chef's push not delivered.
- FSSAI reminder cron skipped a day (multi-day windows tolerate it).
- Elevated p95 latency without errors.
- A flaky non-critical endpoint.
- Dependabot advisories.

## Page response checklist

1. **Ack** — silence the alarm so you can think.
2. **Scope** — one chef or everyone? Check the health URLs + Sentry.
3. **Identify the bucket** — match to a [RUNBOOK](./RUNBOOK.md) section.
4. **Mitigate, don't fix** — restart / rollback first; root-cause after service is restored.
5. **Communicate** — if customer-facing > 5 min, post to the [status page](./STATUS-PAGE.md).
6. **Write it down** — append a 3-line post-incident note (what, cause, fix) to this repo.

## Escalation (when you're the only human)

- Payments: Razorpay/Stripe support dashboards.
- Infra: GCP support (check the support tier on the billing account).
- Auth: Google Identity Platform status + GCP support.

## Quiet hours

With min-scale 1 and fail-open middleware (rate-limit, idempotency, FSSAI dedup all fail **open** on Redis loss), most degradations don't take the service down. Trust the P1 list — if it's not on it, it can wait.
