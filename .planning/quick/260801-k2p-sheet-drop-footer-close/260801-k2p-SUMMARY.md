---
gsd_summary_version: 1.0
quick_id: 260801-k2p
slug: sheet-drop-footer-close
status: complete
date: 2026-08-01
issue: 877
branch: fix/sheet-drop-footer-close
commits:
  - 310464da
  - 8594af4a
---

# Quick Task 260801-k2p — Summary

Closes tesserix/Home-Chef-App#877.

## What changed

Two commits, split so the shared-package work is reviewable on its own:

**`310464da` — `feat(sheets)`: the grabber becomes real**

- New `packages/mobile-shared/src/ui/useSheetDrag.ts`. Downward-only pan;
  release past 96px **or** 0.5px/ms velocity dismisses, otherwise settles back.
- `SheetBase` wraps the grabber pill in a padded drag strip carrying the pan
  handlers. The pill is 40×4 — unusably small as a touch target — so the
  wrapper supplies ~24pt of height across the panel width.
- Gated on `showHandle`: no pill drawn, no gesture. An invisible drag region is
  a gesture users can't discover.

**`8594af4a` — `fix(customer)`: the buttons go**

- `Sheet.tsx`: the ghost cancel button is now opt-in. `cancelLabel` lost its
  `= 'Cancel'` default and only renders when passed.
- `MealPlanSheet`: dropped `cancelLabel="Close"`.
- `MenuCategorySheet`: footer Close removed; grabber + drag added; scroll
  content now carries `insets.bottom + 12` padding, which the deleted footer
  used to supply.

## Two things worth flagging to a reviewer

**The native-driver trap.** `translateY` is already animated by
`useNativeDriver: true`. Attaching a JS-driven `Animated.event` to that same
node throws at runtime ("Attempting to run JS driven animation on animated node
that has been moved to 'native'"). The hook therefore tracks with
`translateY.setValue(dy)` — legal on a native-driven node — and the settle-back
is itself a native-driven timing. This is documented in the file header so the
next person doesn't "simplify" it into a crash.

**A layout regression caught before commit.** `MenuCategorySheet`'s panel
carried `maxHeight: '80%'`. Wrapping it in an `Animated.View` for the drag
transform would have broken that silently — a percentage height resolves
against the parent, and the new wrapper has auto height, so the cap would stop
resolving and the sheet could grow past 80%. `maxHeight` moved up onto the
wrapper, whose parent (the flex-1 backdrop) still has a resolvable height.

## Blast radius (verified, not assumed)

`grep -l SheetBase` is misleading — `Dialog.tsx`, `Toast.tsx` and
`UndoSnackbar.tsx` only mention it in comments. Actual render sites: `FilterSheet`,
`AddressSwitcherSheet`, `Sheet.tsx`, `ReportSheet` (×2).

Making the cancel button opt-in breaks no caller: `<Sheet>` has exactly two
call sites and **both already pass `cancelLabel`** — `order/[id]/index.tsx:1031`
("Not yet", a real choice, kept) and `MealPlanSheet` ("Close", removed). No
consumer relied on the default.

## Verification

- **tsc, all three apps, against a stashed baseline:** customer 1→1,
  vendor 2→2, delivery 19→19. **Zero new type errors.** (Customer's one is the
  known pre-existing `lib/payment.ts(114,7)` expo-router path typing.)
- **Visual, iOS sim** (temp `app/dev-sheet-check.tsx` route, since simctl can't
  tap; route deleted after): Close button gone, grabber renders, active-category
  coral accent intact, clear bottom padding below the last row, 80% cap still
  holding.

## Not verified / known gaps

- **The drag gesture itself was not exercised.** simctl cannot inject pan
  gestures. Thresholds (96px / 0.5 velocity) are reasoned, not tuned on-device.
  Someone should drag these sheets by hand before merge.
- **`MenuCategorySheet` doesn't honor reduce-motion** for the settle-back. It
  already ignored it for the Modal's `animationType="slide"`, so this is not a
  new violation, but `.impeccable.md` wants it honored everywhere.
  `SheetBase` *does* pass `reduceMotion` through correctly.
- **No lint run** — the mobile apps have no `eslint.config.js` (the configs in
  CLAUDE.md belong to the web apps).
- **No tests run** — `jest-expo` fails to load its preset (missing
  `@react-native/jest-preset` peer dep). Fails at require-time before any test
  file is read, so it is environmental, not caused by this change. The script is
  `jest --passWithNoTests` regardless.
- **Prettier is a trap in this repo.** There is no prettier config anywhere, so
  `npx prettier --write` applies double-quote defaults against a single-quote
  codebase and turned a ~90-line diff into ~680 lines of churn. Reverted and
  re-applied by hand. Do not run prettier here without a config.

## Follow-ups

- Tune drag thresholds on-device.
- Plumb reduce-motion into `MenuCategorySheet`, or migrate it onto `SheetBase`
  (blocked on reconciling its controlled `visible` API with SheetBase's
  imperative ref).
- Add a prettier config so formatting is reproducible.
