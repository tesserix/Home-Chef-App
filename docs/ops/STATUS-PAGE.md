# Home Chef — Status Page Plan

Public status at **`status.fe3dr.com`**. Goal: when something breaks, chefs see "we know, we're on it" instead of guessing.

## Option A — Hosted (recommended for launch)

- **BetterStack** or **Instatus** free tier. Hosted, handles incident posting + subscriber emails, sub-domain CNAME.
- Add `status.fe3dr.com` as a CNAME (Cloudflare) to the provider.
- Create monitors that ping:
  - `https://vendors.fe3dr.com/api/v1/mobile/min-version?platform=ios&app=vendor` (API)
  - `https://auth.fe3dr.com/health` (auth)
  - `https://vendors.fe3dr.com` (frontend)
- Auto-incident when a monitor is down > 3 min (matches the P1 page threshold).

## Option B — Roll-your-own (cheapest)

- Static page on Cloudflare Pages at `status.fe3dr.com`.
- A Cloudflare Worker cron (every 1 min) curls the health URLs, writes status JSON to KV; the page reads it.
- Manual incident banner via a KV flag you flip from the dashboard.

## Components to surface

| Component | Health source |
|---|---|
| Vendor app API | `/api/v1/mobile/min-version` |
| Login / Auth | `auth.fe3dr.com/health` |
| Payments | manual toggle (driven by Razorpay status) |
| Push notifications | manual toggle |

## Incident template (post within 5 min of a P1)

```
[Investigating] Some chefs may have trouble <symptom>. We're looking into it. — <time>
[Identified]   Cause found: <one line>. Fix in progress. — <time>
[Monitoring]   Fix deployed, watching recovery. — <time>
[Resolved]     Service restored. Sorry for the disruption. — <time>
```

Keep it human and specific. "Payments are failing for ~10 min, refunds are safe and will reconcile" beats "investigating an issue."
