import { describe, it, expect } from '@jest/globals';

import {
  categoryIdForName,
  categoryNameForId,
  prepTimeLabel,
  prepTimeForLabel,
} from './menuSelectOptions';

const categories = [
  { id: 'c1', name: 'Starters & Snacks' },
  { id: 'c2', name: 'Main Course' },
];

describe('categoryNameForId', () => {
  it('shows the saved category by name', () => {
    expect(categoryNameForId(categories, 'c2')).toBe('Main Course');
  });

  it('shows nothing for an id the chef no longer has', () => {
    expect(categoryNameForId(categories, 'deleted')).toBe('');
  });

  it('shows nothing when no category is picked yet', () => {
    expect(categoryNameForId(categories, '')).toBe('');
  });
});

describe('categoryIdForName', () => {
  it('maps the picked name back to the id the API stores', () => {
    expect(categoryIdForName(categories, 'Starters & Snacks')).toBe('c1');
  });

  it('keeps the first of two categories sharing a name', () => {
    const duplicated = [...categories, { id: 'c3', name: 'Main Course' }];

    expect(categoryIdForName(duplicated, 'Main Course')).toBe('c2');
  });

  it('returns nothing for a name that matches no category', () => {
    expect(categoryIdForName(categories, 'Desserts')).toBe('');
  });
});

describe('prep time labels', () => {
  it('round-trips a value through its label', () => {
    expect(prepTimeForLabel(prepTimeLabel(30))).toBe(30);
  });

  it('reads as minutes rather than a bare number', () => {
    expect(prepTimeLabel(45)).toBe('45 min');
  });

  it('returns nothing for a label that is not one of the options', () => {
    expect(prepTimeForLabel('quickly')).toBeNull();
  });
});
