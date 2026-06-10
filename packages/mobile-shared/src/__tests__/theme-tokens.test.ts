// Tests that the first-class Home Chef design tokens are exported and have
// the shapes consumers depend on. If any of these break, the tailwind
// config in apps/mobile-{vendor,customer,delivery}/tailwind.config.js may
// also have drifted — keep the two in sync per .impeccable.md.

import { describe, it, expect } from 'vitest';

describe('theme tokens', () => {
  it('exports colors with the persimmon brand accent (`herb`)', async () => {
    const { colors } = await import('../theme/tokens');
    expect(colors.herb.DEFAULT).toBe('#C2410C');
    expect(colors.herb.soft).toBe('#9A3412');
    expect(colors.herb.tint).toBe('#FFEDD5');
  });

  it('exports the functional `success` green (vendor positive status)', async () => {
    const { colors } = await import('../theme/tokens');
    expect(colors.success.DEFAULT).toBe('#008A05');
    expect(colors.success.soft).toBe('#046A06');
    expect(colors.success.tint).toBe('#E6F4E6');
  });

  it('exports the Paper · Ink surface tokens', async () => {
    const { colors } = await import('../theme/tokens');
    expect(colors.paper).toBe('#FFFFFF');
    expect(colors.ink.DEFAULT).toBe('#0E0E0C');
    expect(colors.bone).toBe('#F5F5F4');
  });

  it('exports spacing as an integer ramp keyed 0..24', async () => {
    const { spacing } = await import('../theme/tokens');
    expect(spacing[4]).toBe(16);
    expect(spacing[8]).toBe(32);
    expect(spacing[24]).toBe(96);
  });

  it('exports the 4-step radius scale', async () => {
    const { radius } = await import('../theme/tokens');
    expect(radius.DEFAULT).toBe(8);
    expect(radius.lg).toBe(16);
    expect(radius.full).toBe(9999);
  });

  it('exports typography with Geist display + Inter body', async () => {
    const { typography } = await import('../theme/tokens');
    expect(typography.family.display).toBe('Geist');
    expect(typography.family.body).toBe('Inter');
    expect(typography.size.display.size).toBe(32);
  });

  it('exports the 3-step shadow scale', async () => {
    const { shadow } = await import('../theme/tokens');
    expect(shadow[1].shadowOpacity).toBeLessThan(shadow[2].shadowOpacity);
    expect(shadow[3].shadowRadius).toBeGreaterThan(shadow[1].shadowRadius);
  });

  it('exports motion with ease-out-quart entrance + 250ms default', async () => {
    const { motion } = await import('../theme/tokens');
    expect(motion.easing.entrance).toEqual([0.22, 1, 0.36, 1]);
    expect(motion.duration.default).toBe(250);
  });

  it('aggregates everything into a single `theme` object', async () => {
    const { theme } = await import('../theme/tokens');
    expect(theme.colors).toBeDefined();
    expect(theme.spacing).toBeDefined();
    expect(theme.radius).toBeDefined();
    expect(theme.typography).toBeDefined();
    expect(theme.shadow).toBeDefined();
    expect(theme.motion).toBeDefined();
  });
});
