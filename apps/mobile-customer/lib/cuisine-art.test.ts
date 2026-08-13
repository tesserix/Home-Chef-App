import { describe, it, expect } from '@jest/globals';

import { CUISINE_CATEGORIES, cuisineArtUri } from './cuisine-art';

describe('cuisineArtUri', () => {
  it('gives every rail category a photo', () => {
    for (const category of CUISINE_CATEGORIES) {
      expect(cuisineArtUri(category, 168)).toMatch(/^https:\/\/images\.unsplash\.com\/photo-/);
    }
  });

  it('gives every category a photo of its own, so no two tiles look alike', () => {
    const uris = CUISINE_CATEGORIES.map((c) => cuisineArtUri(c, 168));
    expect(new Set(uris).size).toBe(CUISINE_CATEGORIES.length);
  });

  it('crops square at the requested size, so every tile is framed alike', () => {
    expect(cuisineArtUri('Chinese', 168)).toContain('fit=crop&w=168&h=168');
  });

  it('has no photo for a category it does not know, leaving the glyph fallback', () => {
    expect(cuisineArtUri('Mughlai', 168)).toBeUndefined();
  });
});
