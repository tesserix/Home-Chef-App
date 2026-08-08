// Routes a signed-out customer may sit on.
//
// The root layout sends a guest back to the tabs from anywhere they shouldn't
// be. Browsing is the front door (App Review 5.1.1(iv)), so the kitchens, their
// reviews, ChefBook, dish search and the policies all have to stay reachable —
// everything else needs an account and bounces.
//
// The chefs map is currently OUT of that surface: `CHEFS_MAP_ENABLED` is false
// while no Maps SDK key exists, and a guest must not be sent to a screen that
// crashes on Android. It rejoins the browse surface automatically when the flag
// flips, so the 5.1.1(iv) claim stays accurate either way.

import { CHEFS_MAP_ENABLED } from './features';

const GUEST_ROUTES = new Set([
  'chef',
  'chefbook',
  ...(CHEFS_MAP_ENABLED ? ['chefs-map'] : []),
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
