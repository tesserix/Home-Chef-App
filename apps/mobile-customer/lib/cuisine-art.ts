// Art for the Home category rail — one curated photo per cuisine.
//
// The rail first drew tinted glyphs, then borrowed a chef's banner for every
// category that chef was tagged with, which put a bowl of chole under
// "Chinese". Curated photography is the only version that can't be wrong: each
// tile shows the food its label promises, in the same square crop.

export const CUISINE_CATEGORIES = [
  'All',
  'North Indian',
  'South Indian',
  'Chinese',
  'Continental',
  'Italian',
  'Healthy',
] as const;

export type CuisineCategory = (typeof CUISINE_CATEGORIES)[number];

// Unsplash ids, each opened and checked to show the dish named beside it.
const PHOTO_IDS: Record<CuisineCategory, string> = {
  All: 'photo-1680993032090-1ef7ea9b51e5', // steel thali — dal, sabzi, rice, puri
  'North Indian': 'photo-1631452180519-c014fe946bc7', // paneer butter masala with roti
  'South Indian': 'photo-1694849789325-914b71ab4075', // masala dosa with chutney
  Chinese: 'photo-1585032226651-759b368d7246', // hakka noodles, top-down
  Continental: 'photo-1598515214211-89d3c73ae83b', // grilled chicken with buttered veg
  Italian: 'photo-1565299624946-b28f40a0ae38', // wood-fired pizza, sliced
  Healthy: 'photo-1512621776951-a57141f2eefd', // salad bowl with avocado
};

/**
 * Square crop for the circular rail tile, at the pixel size actually drawn.
 *
 * The crop is what keeps the rail aligned — every tile is framed the same way,
 * whatever the source photo's aspect. Unknown category returns nothing, so the
 * caller keeps its glyph fallback.
 */
export function cuisineArtUri(category: string, sizePx: number): string | undefined {
  const id = PHOTO_IDS[category as CuisineCategory];
  if (!id) return undefined;
  return `https://images.unsplash.com/${id}?auto=format&fit=crop&w=${sizePx}&h=${sizePx}&q=70`;
}
