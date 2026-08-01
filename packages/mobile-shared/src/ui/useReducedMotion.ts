// useReducedMotion — the OS "Reduce Motion" accessibility read+subscribe
// pattern, extracted into a single shared hook.
//
// This exact block (one-shot read via `isReduceMotionEnabled()` + live
// updates via the `reduceMotionChanged` listener) used to be copy-pasted
// verbatim across five files in this directory (SheetBase, UndoSnackbar,
// Toast, Skeleton, OnboardingScaffold). GitHub #881 AC4 asks for a single
// shared hook instead — this is that hook.

import { useEffect, useState } from 'react';
import { AccessibilityInfo } from 'react-native';

/**
 * watchReducedMotion — the plain (non-React) core of the hook below.
 *
 * Reads the OS setting once, then keeps emitting on every change until the
 * returned unsubscribe function is called. Split out from the hook so it is
 * directly testable without a React renderer — `packages/mobile-shared` has
 * no renderer dependency and this change deliberately does not add one.
 *
 * @param onChange - called with the current value; never called synchronously
 * @returns unsubscribe — stops the listener and suppresses any late resolve
 */
export function watchReducedMotion(onChange: (enabled: boolean) => void): () => void {
  let alive = true;

  AccessibilityInfo.isReduceMotionEnabled()
    .then((enabled) => {
      if (alive) onChange(enabled);
    })
    .catch(() => {
      /* default: animate */
    });

  const sub = AccessibilityInfo.addEventListener('reduceMotionChanged', (enabled) => {
    if (alive) onChange(enabled);
  });

  return () => {
    alive = false;
    sub.remove();
  };
}

/**
 * useReducedMotion — reads the OS "Reduce Motion" accessibility setting.
 *
 * Returns `false` on first render (before the async check resolves), then
 * updates once `AccessibilityInfo.isReduceMotionEnabled()` resolves and
 * again whenever the OS setting changes mid-session.
 */
export function useReducedMotion(): boolean {
  const [reduceMotion, setReduceMotion] = useState(false);

  useEffect(() => watchReducedMotion(setReduceMotion), []);

  return reduceMotion;
}
