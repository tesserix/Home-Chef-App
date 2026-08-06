import { describe, expect, it } from 'vitest';
import {
  bakerySummary,
  priceBakeryLine,
  weightChoices,
  type BakeryOption,
  type BakerySpec,
} from '../bakery';

// Parity with apps/api/services/bakery_test.go (#1065). The configurator must
// show the exact price the server will charge, so these cases mirror the Go
// ones number for number.

const opt = (o: Partial<BakeryOption> & Pick<BakeryOption, 'id' | 'kind' | 'name'>): BakeryOption => ({
  priceDelta: 0,
  priceMode: 'flat',
  dietaryTags: [],
  allergens: [],
  isAvailable: true,
  isDefault: false,
  sortOrder: 0,
  ...o,
});

const cake: BakerySpec = {
  id: 'spec-1',
  menuItemId: 'item-1',
  productType: 'cake',
  pricePerKg: 800,
  minWeightKg: 0.5,
  maxWeightKg: 5,
  weightStepKg: 0.5,
  servesPerKg: 8,
  allowMessage: true,
  maxMessageChars: 40,
  allowReferencePhoto: true,
  leadTimeHours: 24,
  occasions: ['birthday'],
  options: [
    opt({ id: 'shape-round', kind: 'shape', name: 'Round' }),
    opt({ id: 'shape-heart', kind: 'shape', name: 'Heart', priceDelta: 150 }),
    opt({ id: 'fl-choc', kind: 'flavour', name: 'Belgian chocolate', priceDelta: 200, priceMode: 'per_kg' }),
    opt({ id: 'egg-less', kind: 'egg', name: 'Eggless', priceDelta: 50, dietaryTags: ['eggless'] }),
    opt({ id: 'egg-with', kind: 'egg', name: 'With egg', allergens: ['eggs'] }),
  ],
};

describe('priceBakeryLine', () => {
  it('prices per kg and adds flat and per-kg deltas', () => {
    const { price, snapshot } = priceBakeryLine(cake, 0, {
      weightKg: 1.5,
      optionIds: ['shape-heart', 'fl-choc', 'egg-less'],
      messageOnCake: 'Happy Birthday Aarav',
      occasion: 'birthday',
    });

    // 800×1.5 + 150 + 200×1.5 + 50
    expect(price).toBe(1700);
    expect(snapshot.basePrice).toBe(1200);
    expect(snapshot.serves).toBe(12);
    expect(snapshot.dietaryTags).toEqual(['eggless']);
    expect(snapshot.allergens).toEqual([]);
    expect(snapshot.selections.map((s) => s.name)).toEqual(['Heart', 'Belgian chocolate', 'Eggless']);
  });

  it('carries the allergen that rides on the chosen option', () => {
    const { snapshot } = priceBakeryLine(cake, 0, {
      weightKg: 1,
      optionIds: ['shape-round', 'fl-choc', 'egg-with'],
    });
    expect(snapshot.allergens).toEqual(['eggs']);
    expect(snapshot.dietaryTags).toEqual([]);
  });

  it('uses the flat menu price when the product is not sold by weight', () => {
    const loaf: BakerySpec = { ...cake, pricePerKg: 0, options: [] };
    const { price, snapshot } = priceBakeryLine(loaf, 180, {});
    expect(price).toBe(180);
    expect(snapshot.weightKg).toBe(0);
  });

  const rejections: Array<[string, Parameters<typeof priceBakeryLine>[2], string]> = [
    ['no size on a per-kg cake', { optionIds: ['shape-round', 'fl-choc', 'egg-less'] }, 'choose a size'],
    ['a size above the maximum', { weightKg: 9, optionIds: ['shape-round', 'fl-choc', 'egg-less'] }, 'between'],
    ['a size off the step ladder', { weightKg: 1.2, optionIds: ['shape-round', 'fl-choc', 'egg-less'] }, 'steps'],
    ['no flavour chosen', { weightKg: 1, optionIds: ['shape-round', 'egg-less'] }, 'Flavour'],
    ['two shapes chosen', { weightKg: 1, optionIds: ['shape-round', 'shape-heart', 'fl-choc', 'egg-less'] }, 'just one'],
    ['an option from another cake', { weightKg: 1, optionIds: ['shape-round', 'fl-choc', 'egg-less', 'nope'] }, 'invalid'],
  ];

  for (const [name, input, want] of rejections) {
    it(`rejects ${name}`, () => {
      expect(() => priceBakeryLine(cake, 0, input)).toThrow(new RegExp(want, 'i'));
    });
  }

  it('rejects a message longer than the baker allows', () => {
    expect(() =>
      priceBakeryLine(cake, 0, {
        weightKg: 1,
        optionIds: ['shape-round', 'fl-choc', 'egg-less'],
        messageOnCake: 'x'.repeat(41),
      }),
    ).toThrow(/40 characters/);
  });

  it('rejects an unavailable option', () => {
    const spec: BakerySpec = {
      ...cake,
      options: cake.options.map((o) => (o.id === 'shape-heart' ? { ...o, isAvailable: false } : o)),
    };
    expect(() => priceBakeryLine(spec, 0, { weightKg: 1, optionIds: ['shape-heart', 'fl-choc', 'egg-less'] })).toThrow(
      /no longer available/,
    );
  });
});

describe('weightChoices', () => {
  it('expands min/max/step into the ladder the configurator shows', () => {
    expect(weightChoices(cake)).toEqual([0.5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5]);
  });

  it('is empty for a flat-priced product', () => {
    expect(weightChoices({ ...cake, pricePerKg: 0 })).toEqual([]);
  });
});

describe('bakerySummary', () => {
  it('renders one line for cart, docket, invoice and receipt', () => {
    const { snapshot } = priceBakeryLine(cake, 0, {
      weightKg: 1.5,
      optionIds: ['shape-heart', 'fl-choc', 'egg-less'],
      messageOnCake: 'Happy Birthday Aarav',
    });
    expect(bakerySummary(snapshot)).toBe('1.5 kg · Heart · Belgian chocolate · Eggless · “Happy Birthday Aarav”');
  });
});
