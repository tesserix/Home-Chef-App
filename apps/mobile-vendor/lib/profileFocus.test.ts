import { describe, it, expect } from '@jest/globals';

import { focusedProfileSection } from './profileFocus';

// A deep link that silently scrolls nowhere is indistinguishable from a missing
// feature — which is exactly how the bakery opt-in got reported as absent.
describe('focusedProfileSection', () => {
  it('resolves the bakery deep link', () => {
    expect(focusedProfileSection('bakery')).toBe('kitchen');
  });

  it('resolves the kitchen block by its own name', () => {
    expect(focusedProfileSection('kitchen')).toBe('kitchen');
  });

  it('ignores case and stray whitespace from a hand-typed link', () => {
    expect(focusedProfileSection(' Bakery ')).toBe('kitchen');
  });

  it('takes the first value when the router hands back an array', () => {
    expect(focusedProfileSection(['bakery', 'kitchen'])).toBe('kitchen');
  });

  it('returns null when no section was asked for', () => {
    expect(focusedProfileSection(undefined)).toBeNull();
    expect(focusedProfileSection('')).toBeNull();
    expect(focusedProfileSection([])).toBeNull();
  });

  it('returns null for a section that does not exist', () => {
    expect(focusedProfileSection('payout')).toBeNull();
  });
});
