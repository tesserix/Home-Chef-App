# ADR 0004: Retire durable Temporal schedules compatibly

## Context

Temporal Schedules outlive application deployments. Removing a cron job from
the worker registry does not remove its Schedule or its already-started
workflows. The retired `meal-plan-hold-reconcile` Schedule continued firing
every 10 minutes (144 starts/day), while one workflow retried the now-unknown
activity more than 18,000 times. This work is outside the request path, but the
operational target is zero permanently failing workflows and no executions for
retired business logic.

The escrow day-transfer layer was intentionally removed in #1106. Restoring its
money-moving handler would contradict the current settlement model. Deleting
the Schedule imperatively would be unaudited and would not protect an in-flight
workflow or another environment that still has the durable Schedule.

## Decision

Retired cron names remain as compatibility tombstones in the worker. A workflow
carrying a retired name completes successfully without executing business
logic. On API startup, the schedule manager pauses each retired Schedule and
records the retirement reason in its note. A missing Schedule is accepted so
new environments remain clean. Other pause failures are logged, but do not
start the in-process ticker fallback; the tombstone still contains the failure
until the next startup retries the pause.

Future cron removals must add the stable job name and reason to the retirement
registry in the same release that removes the handler. Tombstones stay until
workflow retention guarantees that no old execution can call them.

## Consequences

- Existing retries drain safely after the worker rollout.
- The durable Schedule is preserved for audit and rollback, but stops creating
  new workflows.
- A failed pause may create up to 144 no-op workflows/day until the next retry;
  it cannot move money or duplicate active cron execution.
- Rollback is explicit: restore the handler and unpause the Schedule. Reverting
  only this code does not silently reactivate retired money movement.

## Alternatives rejected

- Restore the deleted escrow handler: violates the current payout design.
- Delete the Schedule: loses audit/rollback state and does not handle in-flight
  workflows.
- Ignore the retries: leaves permanent workflow failures and alert noise.
