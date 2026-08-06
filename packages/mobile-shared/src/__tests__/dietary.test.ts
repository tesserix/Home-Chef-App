import { describe, expect, it } from 'vitest';
import { BAKERY_DIET_OPTIONS, DIET_OPTIONS, findItemConflicts } from '../dietary';

// Parity with apps/api/services/dietary_test.go — a diet implies the allergens
// it rules out (#1065), so an eggless customer is warned off an egg cake even
// when they never listed eggs as an allergy.
describe('findItemConflicts — diet-implied allergens', () => {
  it('warns an eggless customer about a cake containing eggs', () => {
    const got = findItemConflicts({ dietaryPreferences: ['eggless'] }, { allergens: ['eggs'] });
    expect(got).toHaveLength(1);
    expect(got[0]).toMatchObject({ type: 'allergen', label: 'Eggs' });
  });

  it('leaves an eggless customer alone on an eggless cake', () => {
    expect(findItemConflicts({ dietaryPreferences: ['eggless'] }, { allergens: ['dairy'] })).toHaveLength(0);
  });

  it('warns a vegan about dairy frosting', () => {
    const got = findItemConflicts({ dietaryPreferences: ['vegan'] }, { allergens: ['dairy'] });
    expect(got).toHaveLength(1);
    expect(got[0].label).toBe('Dairy (milk)');
  });

  it('warns a gluten-free customer about a wheat sponge', () => {
    expect(findItemConflicts({ dietaryPreferences: ['gluten-free'] }, { allergens: ['gluten'] })).toHaveLength(1);
  });

  it('does not double-warn when the customer both avoids and diets away an allergen', () => {
    const got = findItemConflicts(
      { dietaryPreferences: ['eggless'], foodAllergies: ['eggs'] },
      { allergens: ['eggs'] },
    );
    expect(got).toHaveLength(1);
  });

  it('says nothing when no preference is set', () => {
    expect(findItemConflicts({}, { allergens: ['eggs'] })).toHaveLength(0);
  });
});

describe('bakery taxonomy', () => {
  it('offers eggless and sugar-free as first-class diets', () => {
    const values = DIET_OPTIONS.map((o) => o.value);
    expect(values).toContain('eggless');
    expect(values).toContain('sugar-free');
  });

  it('keeps the bakery filter list a subset of the full taxonomy', () => {
    const all = new Set(DIET_OPTIONS.map((o) => o.value));
    for (const o of BAKERY_DIET_OPTIONS) expect(all.has(o.value)).toBe(true);
  });
});
