---
slug: test-session-clone-unique-indexes
created: 2026-07-26
mode: quick
branch: fix/test-session-clone-unique-indexes
---

# Fix live→test clone unique-index collisions

## Problem

Opening a test session for chef `aa3fb37f` (Amma Ka Kitchen) fails:

```
test-session: clone chef aa3fb37f-…: clone: copy weekly_menus:
ERROR: duplicate key value violates unique constraint "idx_weekly_menus_chef_id" (SQLSTATE 23505)
```

## Root cause

`CloneChefIntoSession` copies a kitchen into the test partition with
`INSERT … SELECT` back into the *same* table, changing only `mode`,
`test_session_id` and `cloned_from_id`. Every other column — including the
natural key — is copied verbatim.

Three unique indexes on partitioned tables do not include `mode`, so the test
replica collides with its live original. Verified against the prod primary
(`homechef-postgres-20`, `homechef_db`):

| Table | Index | State |
|---|---|---|
| `weekly_menus` | `UNIQUE (chef_id)` | failing now |
| `daily_menus` | `UNIQUE (chef_id, date)` | latent — 0 daily menus today |
| `orders` | `UNIQUE (order_number)` | latent — `order_number` copied verbatim |

`_pkey` on `id` is safe (the clone generates a fresh UUID). The Razorpay/Stripe
unique indexes are safe — they are partial (`WHERE col <> ''`) and the clone
already blanks those columns.

Widening to `UNIQUE(chef_id, mode)` is **not** sufficient: purge is a separate
step from close, so a closed session's test rows survive and session 2 would
collide with session 1's leftovers.

Why tests missed it: `setupSessionDB` builds sqlite fixtures with **no unique
indexes at all**, so the clone never met the constraint it violates in prod.

Secondary: `weekly_menu_items` and `daily_menu_items` are listed in
`partitionedTables` (so purge covers them) but absent from
`CloneChefIntoSession`'s `cfg`, so a cloned menu arrives with no dishes.

## Tasks

1. **RED** — extend `setupSessionDB` with `weekly_menus`, `weekly_menu_items`,
   `daily_menus`, `daily_menu_items` carrying the *production* unique indexes,
   plus `UNIQUE(order_number)` on `orders`. Add a regression test that opens
   **two consecutive** sessions for a chef holding a weekly menu, a daily menu
   with items, and windowed orders. Must fail with 23505.
2. **GREEN — schema.** `postMigrate` in `apps/api/database/database.go`: drop the
   mode-blind indexes, create split partial ones following the existing
   `idx_addresses_one_default_per_user` idiom:
   - `UNIQUE (chef_id) WHERE mode = 'live'`
   - `UNIQUE (chef_id, test_session_id) WHERE mode = 'test'`
   for `weekly_menus`, `daily_menus`, and `weekly_menu_items`' `idx_weekly_cell`.
3. **GREEN — models.** Drop the now-misleading mode-blind `uniqueIndex` tags in
   `models/weekly_menu.go` and `models/daily_menu.go` so AutoMigrate stops
   recreating them. `Order.OrderNumber` keeps its global unique — see task 4.
4. **GREEN — clone.** Add an `sqlExpr` override type so `order_number` is
   *derived* per session rather than copied (order numbers must stay globally
   unique for invoicing). Generalise `cloneRows` with a table alias and optional
   join so `daily_menu_items` can remap `daily_menu_id` onto the cloned parent
   via `cloned_from_id`.
5. **GREEN — coverage.** Add `weekly_menu_items` and `daily_menu_items` to `cfg`.
6. Full `go test ./...` on the api module.

## Out of scope

- Chef `aa3fb37f` is stranded: `mode='live'` with `active_test_session_id` set
  and session `bbfc912e` still `open`. `CloseTestSession` clears both together,
  so it never ran. Repaired separately after this ships.

## Verification

- New regression test fails before the fix, passes after.
- Existing `test_session_test.go` / `test_session_clone_test.go` still pass.
- Prod index definitions match the postMigrate block after deploy.
