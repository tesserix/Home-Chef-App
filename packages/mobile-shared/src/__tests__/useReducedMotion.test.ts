// watchReducedMotion — coverage for the shared OS "Reduce Motion" subscription
// (GitHub #881 AC4). Mocks AccessibilityInfo locally (the shared
// src/__mocks__/react-native.ts has no AccessibilityInfo today and is used
// by unrelated screen tests, so this mock stays scoped to this file).
//
// These exercise `watchReducedMotion`, the plain core of `useReducedMotion`,
// rather than the hook itself: `packages/mobile-shared` has no React renderer
// dependency and this change deliberately avoids adding one. The hook is a
// four-line `useState` + `useEffect` wrapper over exactly this function.

import { beforeEach, describe, expect, it, vi } from 'vitest';

const { isReduceMotionEnabled, addEventListener } = vi.hoisted(() => ({
  isReduceMotionEnabled: vi.fn(),
  addEventListener: vi.fn(),
}));

vi.mock('react-native', () => ({
  AccessibilityInfo: {
    isReduceMotionEnabled,
    addEventListener,
  },
}));

import { watchReducedMotion } from '../ui/useReducedMotion';

describe('watchReducedMotion', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('does not emit synchronously — the hook renders false until the check resolves', () => {
    isReduceMotionEnabled.mockReturnValue(new Promise<boolean>(() => {}));
    addEventListener.mockReturnValue({ remove: vi.fn() });

    const onChange = vi.fn();
    watchReducedMotion(onChange);

    expect(onChange).not.toHaveBeenCalled();
  });

  it('emits true once isReduceMotionEnabled() resolves true', async () => {
    isReduceMotionEnabled.mockResolvedValue(true);
    addEventListener.mockReturnValue({ remove: vi.fn() });

    const onChange = vi.fn();
    watchReducedMotion(onChange);
    await vi.waitFor(() => expect(onChange).toHaveBeenCalledWith(true));
  });

  it('emits again when reduceMotionChanged fires', async () => {
    isReduceMotionEnabled.mockResolvedValue(false);
    let changedCallback: (enabled: boolean) => void = () => {};
    addEventListener.mockImplementation((_event: string, cb: (enabled: boolean) => void) => {
      changedCallback = cb;
      return { remove: vi.fn() };
    });

    const onChange = vi.fn();
    watchReducedMotion(onChange);
    await vi.waitFor(() => expect(onChange).toHaveBeenCalledWith(false));

    changedCallback(true);

    expect(onChange).toHaveBeenLastCalledWith(true);
  });

  it('removes the listener and stops emitting after unsubscribe', async () => {
    isReduceMotionEnabled.mockResolvedValue(false);
    const remove = vi.fn();
    let changedCallback: (enabled: boolean) => void = () => {};
    addEventListener.mockImplementation((_event: string, cb: (enabled: boolean) => void) => {
      changedCallback = cb;
      return { remove };
    });

    const onChange = vi.fn();
    const unsubscribe = watchReducedMotion(onChange);
    await vi.waitFor(() => expect(onChange).toHaveBeenCalledWith(false));

    unsubscribe();
    expect(remove).toHaveBeenCalledTimes(1);

    // A listener that fires after unsubscribe must not reach the consumer —
    // in the hook that would be a setState on an unmounted component.
    changedCallback(true);
    expect(onChange).not.toHaveBeenCalledWith(true);
  });

  it('falls back to animating when the OS check rejects', async () => {
    isReduceMotionEnabled.mockRejectedValue(new Error('unavailable'));
    addEventListener.mockReturnValue({ remove: vi.fn() });

    const onChange = vi.fn();
    watchReducedMotion(onChange);
    await vi.waitFor(() => expect(isReduceMotionEnabled).toHaveBeenCalled());

    expect(onChange).not.toHaveBeenCalled();
  });
});
