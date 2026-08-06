import { describe, it, expect } from '@jest/globals';

import { filterOptions } from './optionSearch';

const STATES = [
  'Andaman and Nicobar Islands',
  'Andhra Pradesh',
  'Karnataka',
  'Tamil Nadu',
  'West Bengal',
];

// A 36-entry state list is only usable if typing narrows it, and a chef types
// what they say, not what the list is sorted by.
describe('filterOptions', () => {
  it('returns everything for an empty query', () => {
    expect(filterOptions(STATES, '')).toEqual(STATES);
    expect(filterOptions(STATES, '   ')).toEqual(STATES);
  });

  it('matches on a prefix', () => {
    expect(filterOptions(STATES, 'kar')).toEqual(['Karnataka']);
  });

  it('matches mid-word, so "bengal" finds West Bengal', () => {
    expect(filterOptions(STATES, 'bengal')).toEqual(['West Bengal']);
  });

  it('ignores case and surrounding whitespace', () => {
    expect(filterOptions(STATES, '  TAMIL ')).toEqual(['Tamil Nadu']);
  });

  it('keeps list order when several match', () => {
    expect(filterOptions(STATES, 'and')).toEqual([
      'Andaman and Nicobar Islands',
      'Andhra Pradesh',
    ]);
  });

  it('returns nothing when there is no match', () => {
    expect(filterOptions(STATES, 'zzz')).toEqual([]);
  });
});
