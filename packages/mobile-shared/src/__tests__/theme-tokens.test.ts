// theme-tokens.test.ts — RED phase: tests for @tesserix/native token bridge

import { describe, it, expect } from 'vitest';

describe('theme tokens bridge', () => {
  it('nativeTokens export is an object — not undefined — when imported', async () => {
    const { nativeTokens } = await import('../theme/tokens');
    expect(nativeTokens).toBeDefined();
    expect(typeof nativeTokens).toBe('object');
    expect(nativeTokens).not.toBeNull();
  });

  it('colors export is an object', async () => {
    const { colors } = await import('../theme/tokens');
    expect(colors).toBeDefined();
    expect(typeof colors).toBe('object');
  });

  it('spacing export is an object', async () => {
    const { spacing } = await import('../theme/tokens');
    expect(spacing).toBeDefined();
    expect(typeof spacing).toBe('object');
  });

  it('typography export is an object', async () => {
    const { typography } = await import('../theme/tokens');
    expect(typography).toBeDefined();
    expect(typeof typography).toBe('object');
  });
});
