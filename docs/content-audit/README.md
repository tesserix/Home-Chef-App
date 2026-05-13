# Home Chef Content Audit

Comprehensive audit of all user-facing static content across the 7 Home Chef apps + Go API. Produces a sequenced execution backlog for downstream rewrite phases (CW-01..CW-10).

Design spec: [../superpowers/specs/2026-05-13-content-audit-design.md](../superpowers/specs/2026-05-13-content-audit-design.md)
Execution plan: [../superpowers/plans/2026-05-13-content-audit-execution.md](../superpowers/plans/2026-05-13-content-audit-execution.md)

Generated: 2026-05-13

## Artifacts

| File | Status | Purpose |
|---|---|---|
| [INVENTORY-SCHEMA.md](./INVENTORY-SCHEMA.md) | DONE | Inventory row schema reference |
| [CONTENT-INVENTORY.md](./CONTENT-INVENTORY.md) | DONE | Every user-facing string, organized by category × app (1,720 rows) |
| [STYLE-GUIDE.md](./STYLE-GUIDE.md) | DONE | Voice, persona tone, vocabulary (60+ pairs), microcopy formulas |
| [lens-briefs/](./lens-briefs/) | DONE | Reusable prompts for the four audit lenses |
| [AUDIT-FINDINGS.md](./AUDIT-FINDINGS.md) | DONE | Rolled-up index of all 2,417 findings (sortable) |
| [findings/](./findings/) | DONE | Per-category finding detail (9 files) |
| [REWRITE-BACKLOG.md](./REWRITE-BACKLOG.md) | DONE | 10-phase prioritized execution sequence (CW-01..CW-10) |

## Reading order

1. **`STYLE-GUIDE.md`** — the voice the audit measures against
2. **`REWRITE-BACKLOG.md`** — the sequenced execution phases (CW-01..CW-10)
3. **`AUDIT-FINDINGS.md`** — drill into specific findings by lens / severity / category
4. **`findings/<category>.md`** — full detail for any specific finding
5. **`CONTENT-INVENTORY.md`** — coverage reference for a specific surface

## Headline numbers

| Metric | Value |
|---|---|
| Apps covered | 7 (web, vendor-portal, delivery-portal, admin-portal, mobile-customer, mobile-vendor, mobile-delivery) + Go API |
| Surfaces inventoried | 1,720 |
| Findings produced | 2,417 |
| P0 (launch-blocker) findings | 292 |
| P1 (launch-recommended) findings | 545 |
| P2 (polish) findings | 827 |
| P3 (nice-to-have) findings | 753 |
| Legal-lens findings requiring lawyer review | 617 |
| Execution phases | 10 (+ 1 empty overflow sentinel) |
| Launch-blocker phases total | CW-01..CW-04 (854 findings) |

## Top P0 cross-category clusters

From `AUDIT-FINDINGS.md`:

1. **AI-slop / fake metrics** (26 P0) — hardcoded "500+ Home Chefs", fabricated "Sarah M." testimonial, "Join thousands" patterns
2. **Brand identity drift** (35 P0) — Fe3dr / HomeChef / "Fe3dr by HomeChef" / "HomeChef Delivery" all coexist
3. **DPDP Act consent infrastructure missing** (41 P0) — no granular consent at signup, no grievance officer disclosure, no cookie banner, no children's age gate
4. **FSSAI gaps** (19 P0) — Optional/Required contradiction, license number missing on every transactional surface, allergen disclosure absent
5. **Routing bugs** (14 P0) — `DeliveryAssigned` email written for driver but sent to customer, duplicate-push from two NATS queue groups
6. **Dead conversion paths** (6 P0) — `/become-chef` route doesn't exist, `delivery-portal/RegisterPage` is a silent redirect to /login
7. **"Saved locally" + "Coming soon" features collecting real data** (15 P0) — bank details collected for nonexistent payout integration
8. **Order status taxonomy fragmentation** (33 P0) — 3 independent status-label maps in the API produce different copy for the same event
9. **Aadhaar/PAN collection without UIDAI offline-eKYC** (13 P0) — raw upload exposes platform to Aadhaar Act §29 + PMLA
10. **Missing legal disclosures** (36 P0) — grievance officer, refund timeline, GST invoice line all absent from required surfaces

## Launch gate

**Must ship pre-launch:**
- CW-01 — Legal launch-blockers (711 findings) — lawyer review required
- CW-02 — Customer signup + core UX P0/P1 (53 findings)
- CW-03 — Chef onboarding + vendor verbs P0/P1 (38 findings)
- CW-04 — Driver onboarding + delivery glanceable P0/P1 (52 findings)

**Strongly recommended pre-launch:**
- CW-05 — Marketing surfaces voice unification (71 findings)
- CW-06 — Transactional content sweep (287 findings)

**Iterative post-launch OK:**
- CW-07 — Errors + empty states unification (313 findings)
- CW-08 — Microcopy + persona polish (734 findings)
- CW-09 — Help + SEO/meta (92 findings)
- CW-10 — Admin-portal copy (66 findings)

## Success criteria (from design spec)

| Criterion | Met? | Evidence |
|---|---|---|
| 1. CONTENT-INVENTORY.md covers every visible static string across 7 apps + API | ✓ | 1,720 rows across all 7 apps + API |
| 2. STYLE-GUIDE.md concrete enough for copywriter/LLM rewrites | ✓ | 7 sections, 60+ vocab pairs, 5 microcopy formulas, persona tone matrix |
| 3. AUDIT-FINDINGS.md has ≥1 finding per high-traffic surface; P0 legal flags lawyer review | ✓ | 2,417 findings; 617 legal findings all flag `depends_on: needs lawyer review` |
| 4. REWRITE-BACKLOG.md sequenced so `/gsd-plan-phase CW-01` is immediately invocable | ✓ | CW-01 finding_ids enumerated by sub-bundle (CW-01a..CW-01f) |
| 5. Four artifacts readable in order without reading code | ✓ | Reading-order section above |

## Next step

Run `/gsd-plan-phase CW-01` to start the first execution phase (legal launch-blockers).

For CW-01, recommend starting with sub-bundle CW-01a (Missing legal pages — T&C, Privacy, Refund, Cookie) so subsequent sub-bundles have target pages to populate.
