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
 * useReducedMotion — reads the OS "Reduce Motion" accessibility setting.
 *
 * Returns `false` on first render (before the async check resolves), then
 * updates once `AccessibilityInfo.isReduceMotionEnabled()` resolves and
 * again whenever the OS setting changes mid-session.
 */
export function useReducedMotion(): boolean {
  const [reduceMotion, setReduceMotion] = useState(false);

  useEffect(() => {
    let alive = true;
    AccessibilityInfo.isReduceMotionEnabled()
      .then((enabled) => {
        if (alive) setReduceMotion(enabled);
      })
      .catch(() => {});
    const sub = AccessibilityInfo.addEventListener('reduceMotionChanged', (enabled) => {
      setReduceMotion(enabled);
    });
    return () => {
      alive = false;
      sub.remove();
    };
  }, []);

  return reduceMotion;
}
