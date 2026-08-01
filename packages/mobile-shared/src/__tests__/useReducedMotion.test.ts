// useReducedMotion — coverage for the shared OS "Reduce Motion" hook
// (GitHub #881 AC4). Mocks AccessibilityInfo locally (the shared
// src/__mocks__/react-native.ts has no AccessibilityInfo today and is used
// by unrelated screen tests, so this mock stays scoped to this file).

import { createElement } from 'react';
import { act, create } from 'react-test-renderer';
import { describe, expect, it, vi } from 'vitest';

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

import { useReducedMotion } from '../ui/useReducedMotion';

function Harness({ onValue }: { onValue: (v: boolean) => void }) {
  onValue(useReducedMotion());
  return null;
}

describe('useReducedMotion', () => {
  it('returns false synchronously on first render, before the async check resolves', async () => {
    let resolveCheck: (v: boolean) => void = () => {};
    isReduceMotionEnabled.mockReturnValue(
      new Promise<boolean>((resolve) => {
        resolveCheck = resolve;
      }),
    );
    addEventListener.mockReturnValue({ remove: vi.fn() });

    const values: boolean[] = [];
    let renderer: ReturnType<typeof create>;
    act(() => {
      renderer = create(createElement(Harness, { onValue: (v) => values.push(v) }));
    });

    expect(values[0]).toBe(false);

    await act(async () => {
      resolveCheck(false);
      await Promise.resolve();
    });

    renderer!.unmount();
  });

  it('flips to true once isReduceMotionEnabled() resolves true', async () => {
    isReduceMotionEnabled.mockResolvedValue(true);
    addEventListener.mockReturnValue({ remove: vi.fn() });

    const values: boolean[] = [];
    let renderer: ReturnType<typeof create>;
    await act(async () => {
      renderer = create(createElement(Harness, { onValue: (v) => values.push(v) }));
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(values[values.length - 1]).toBe(true);

    renderer!.unmount();
  });

  it('updates live from false to true when reduceMotionChanged fires with true', async () => {
    isReduceMotionEnabled.mockResolvedValue(false);
    let changedCallback: (enabled: boolean) => void = () => {};
    addEventListener.mockImplementation((_event: string, cb: (enabled: boolean) => void) => {
      changedCallback = cb;
      return { remove: vi.fn() };
    });

    const values: boolean[] = [];
    let renderer: ReturnType<typeof create>;
    await act(async () => {
      renderer = create(createElement(Harness, { onValue: (v) => values.push(v) }));
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(values[values.length - 1]).toBe(false);

    await act(async () => {
      changedCallback(true);
    });

    expect(values[values.length - 1]).toBe(true);

    renderer!.unmount();
  });

  it('calls subscription.remove() when the consuming component unmounts', async () => {
    isReduceMotionEnabled.mockResolvedValue(false);
    const remove = vi.fn();
    addEventListener.mockReturnValue({ remove });

    let renderer: ReturnType<typeof create>;
    await act(async () => {
      renderer = create(createElement(Harness, { onValue: () => {} }));
      await Promise.resolve();
    });

    expect(remove).not.toHaveBeenCalled();

    act(() => {
      renderer!.unmount();
    });

    expect(remove).toHaveBeenCalledTimes(1);
  });
});
