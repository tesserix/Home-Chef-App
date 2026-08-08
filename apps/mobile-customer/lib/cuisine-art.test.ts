import { describe, it, expect } from '@jest/globals';

import { collectCuisineArt } from './cuisine-art';

const CUISINES = ['All', 'North Indian', 'South Indian', 'Chinese'];

function chef(name: string, cuisine: string, imageUrl?: string) {
  return { name, cuisine, imageUrl };
}

describe('collectCuisineArt', () => {
  it('gives each category the first real photo listed under it', () => {
    const art = collectCuisineArt(
      [
        chef('Anita', 'North Indian · Chinese', 'anita.jpg'),
        chef('Ravi', 'South Indian', 'ravi.jpg'),
      ],
      {},
      CUISINES,
    );

    expect(art['North Indian']).toBe('anita.jpg');
    expect(art['Chinese']).toBe('anita.jpg');
    expect(art['South Indian']).toBe('ravi.jpg');
  });

  it('fronts the whole rail with the first photo on the page', () => {
    const art = collectCuisineArt([chef('Anita', 'Chinese', 'anita.jpg')], {}, CUISINES);
    expect(art.All).toBe('anita.jpg');
  });

  it('never drops a photo it already had', () => {
    // Selecting a category refetches a narrowed list; without this the other
    // tiles would blank out every time the customer filters.
    const art = collectCuisineArt(
      [chef('Ravi', 'South Indian', 'ravi.jpg')],
      { Chinese: 'anita.jpg' },
      CUISINES,
    );
    expect(art.Chinese).toBe('anita.jpg');
    expect(art['South Indian']).toBe('ravi.jpg');
  });

  it('keeps the first photo rather than churning on every refetch', () => {
    const art = collectCuisineArt(
      [chef('Meera', 'Chinese', 'meera.jpg')],
      { Chinese: 'anita.jpg' },
      CUISINES,
    );
    expect(art.Chinese).toBe('anita.jpg');
  });

  it('returns the same object when nothing new arrived', () => {
    // Referential stability — this feeds a render, and a fresh object every
    // fetch would re-render the rail for no reason.
    const seen = { Chinese: 'anita.jpg', All: 'anita.jpg' };
    expect(collectCuisineArt([chef('Ravi', 'Chinese')], seen, CUISINES)).toBe(seen);
    expect(collectCuisineArt([], seen, CUISINES)).toBe(seen);
  });

  it('ignores a chef with no photo', () => {
    expect(collectCuisineArt([chef('Ravi', 'Chinese')], {}, CUISINES)).toEqual({});
  });

  it('matches the category name case-insensitively, not as a substring', () => {
    // "Indian" must not soak up "North Indian"; "chinese" must still match.
    const art = collectCuisineArt(
      [chef('Anita', 'north indian', 'anita.jpg')],
      {},
      CUISINES,
    );
    expect(art['North Indian']).toBe('anita.jpg');
    expect(art['South Indian']).toBeUndefined();
  });
});
