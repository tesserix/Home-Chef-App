import { describe, it, expect, jest } from '@jest/globals';

jest.mock('../lib/api', () => ({ api: { get: jest.fn() } }));

import { dishSearchParams } from './useSearchDishes';

describe('dishSearchParams', () => {
  it('scopes the search to the customer location when it is known', () => {
    expect(dishSearchParams('dal', { lat: -37.81, lng: 144.96 })).toEqual({
      q: 'dal',
      page: 1,
      limit: 30,
      lat: -37.81,
      lng: 144.96,
    });
  });

  it('searches unscoped when the location is unknown', () => {
    expect(dishSearchParams('dal', null)).toEqual({ q: 'dal', page: 1, limit: 30 });
  });
});
