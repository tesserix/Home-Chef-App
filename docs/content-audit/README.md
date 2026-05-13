# Home Chef Content Audit

Comprehensive audit of all user-facing static content across the 7 Home Chef apps and the Go API.

Design spec: [../superpowers/specs/2026-05-13-content-audit-design.md](../superpowers/specs/2026-05-13-content-audit-design.md)
Execution plan: [../superpowers/plans/2026-05-13-content-audit-execution.md](../superpowers/plans/2026-05-13-content-audit-execution.md)

## Artifacts

| File | Status | Purpose |
|---|---|---|
| `INVENTORY-SCHEMA.md` | TODO | Schema reference for inventory rows |
| `CONTENT-INVENTORY.md` | TODO | Every user-facing string, by category × app |
| `STYLE-GUIDE.md` | TODO | Voice, vocabulary, microcopy formulas |
| `lens-briefs/` | TODO | Reusable prompts for audit lenses |
| `AUDIT-FINDINGS.md` | TODO | Rolled-up index of all findings |
| `findings/*.md` | TODO | Per-category finding detail |
| `REWRITE-BACKLOG.md` | TODO | 10-phase prioritized execution sequence |

## How to read this audit

1. Skim `STYLE-GUIDE.md` for the voice the audit measures against
2. Read `REWRITE-BACKLOG.md` for the sequenced execution phases (CW-01..CW-10)
3. Drill into specific findings via `AUDIT-FINDINGS.md` → `findings/<category>.md`
4. Use `CONTENT-INVENTORY.md` to confirm coverage of a specific surface
