// Routes a signed-out customer may sit on.
//
// The root layout sends a guest back to the tabs from anywhere they shouldn't
// be. Browsing is the front door (App Review 5.1.1(iv)), so the kitchens, their
// reviews, ChefBook, the map, dish search and the policies all have to stay
// reachable — everything else needs an account and bounces.

const GUEST_ROUTES = new Set([
  'chef',
  'chefbook',
  'chefs-map',
  'search-dishes',
  'terms',
  'privacy',
  'eula',
  'legal',
  'data-privacy',
]);

/** @param segments - expo-router's useSegments() for the current route. */
export function isGuestBrowsable(segments: string[]): boolean {
  const root = segments[0];
  return root !== undefined && GUEST_ROUTES.has(root);
}
