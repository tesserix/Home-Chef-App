// Art for the Home category rail, sourced from the kitchens themselves.
//
// The rail used to be tinted circles with a lucide glyph, because illustrated
// category art was a deliverable we did not have and stock food photos read as
// filler. Real chef photography is neither: it is this city's actual food, it
// costs no new asset, and it makes the rail look like the product rather than a
// wireframe. The glyph stays as the fallback for a category with nothing under
// it yet.

/** The subset of a chef the rail needs — keeps this pure and trivially testable. */
interface ArtSource {
  cuisine?: string;
  imageUrl?: string;
}

export type CuisineArt = Record<string, string>;

/** The rail's leftmost tile, which stands for the whole list rather than one cuisine. */
const ALL = 'All';

function cuisineNames(chef: ArtSource): string[] {
  return (chef.cuisine ?? '')
    .split('·')
    .map((c) => c.trim().toLowerCase())
    .filter(Boolean);
}

/**
 * Folds the chefs on screen into the category → photo map, additively.
 *
 * `seen` is never overwritten: the first photo a category gets is the one it
 * keeps, so filtering to one cuisine cannot blank the rest of the rail, and a
 * refetch cannot shuffle it. Returns `seen` itself when nothing new arrived, so
 * a render can depend on the result.
 */
export function collectCuisineArt(
  chefs: ArtSource[],
  seen: CuisineArt,
  categories: readonly string[],
): CuisineArt {
  let next: CuisineArt | null = null;
  const claim = (category: string, url: string) => {
    if (seen[category] || next?.[category]) return;
    next = next ?? { ...seen };
    next[category] = url;
  };

  for (const chef of chefs) {
    const url = chef.imageUrl;
    if (!url) continue;
    claim(ALL, url);
    const names = cuisineNames(chef);
    for (const category of categories) {
      if (category !== ALL && names.includes(category.toLowerCase())) {
        claim(category, url);
      }
    }
  }

  return next ?? seen;
}
