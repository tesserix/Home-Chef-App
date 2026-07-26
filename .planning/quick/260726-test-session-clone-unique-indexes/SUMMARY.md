---
slug: test-session-clone-unique-indexes
status: complete
completed: 2026-07-26
branch: fix/test-session-clone-unique-indexes
commit: 5c95417c
---

# Summary

Opening a test session for any chef holding a weekly menu failed with
`SQLSTATE 23505` on `idx_weekly_menus_chef_id`. Root cause: the live→test clone
`INSERT … SELECT`s rows back into the same table changing only the partition
columns, so every unique index that ignores `mode` rejects the copy.

## What shipped

| Area | Change |
|---|---|
| `database/database.go` | `postMigrate` drops the three mode-blind indexes and creates split partial pairs — `WHERE mode='live'` keyed on the natural key, `WHERE mode='test'` keyed on natural key + `test_session_id` — for `weekly_menus`, `weekly_menu_items`, `daily_menus` |
| `models/weekly_menu.go`, `models/daily_menu.go` | Mode-blind `uniqueIndex` tags demoted to plain lookup indexes under new `_lookup` names so AutoMigrate stops recreating the dropped ones |
| `services/test_session_clone.go` | New `sqlExpr` override type; `order_number` now **derived** per session rather than copied verbatim; `cloneRows` gained a table alias + optional join |
| `services/test_session_clone.go` | `weekly_menu_items` and `daily_menu_items` added to `cfg`, with `daily_menu_id` remapped onto the cloned parent via `cloned_from_id` |
| tests | `setupSessionDB` now creates the production unique indexes; two new regression tests |

## Why split partial indexes rather than adding `mode` to the key

Purge is a separate step from close, so a closed session's test rows survive.
A single `UNIQUE(chef_id, mode)` passes the first open and fails the second
against session 1's leftovers. `TestCloneSurvivesUniqueConstraintsAcrossSessions`
opens two consecutive sessions specifically to hold that line.

## Why it wasn't caught

`setupSessionDB` built sqlite fixtures with **no unique indexes at all**, so the
clone never met the constraint it violates in prod. Adding them turned five
existing tests red on `orders.order_number` — the latent second bug — before any
fix was written.

## Verification

- `go build ./...` clean; `go vet` clean on touched packages.
- Full `go test ./...` on `apps/api`: all pass.
- New tests confirmed failing before the fix, passing after.
- Prod checked for pre-existing duplicates that would make the **fatal**
  `postMigrate` loop crash-loop the API on boot: zero across all six new
  partial indexes.

## Prod data repair

Chef `aa3fb37f` (Amma Ka Kitchen) was stranded — `mode='live'` with
`active_test_session_id` set and session `bbfc912e` still `open`, a state
`CloseTestSession` cannot produce since it clears both together. Applied the
close semantics directly on the primary. `closed_by_id` left NULL: no admin
actually performed the close, and attributing it to one would be false.

How the chef got there is still unexplained — something set `mode` back to
`live` without closing the session. Worth a follow-up issue.

## Follow-ups

- Find the code path that flips `mode` without going through `CloseTestSession`.
- `partitionedTables` and the clone's `cfg` are still two hand-maintained lists;
  the comment claims adding to one can't be forgotten by the other, but
  `weekly_menu_items` / `daily_menu_items` proved otherwise.
