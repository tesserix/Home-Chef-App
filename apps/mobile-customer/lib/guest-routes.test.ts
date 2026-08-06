import { describe, it, expect } from '@jest/globals';

// The root layout bounced a signed-out customer back to the tabs from anything
// outside (auth)/(tabs). Tapping a chef card from the guest feed therefore
// returned them to the home screen instead of opening the kitchen — the feed
// they were given to browse led nowhere.

import { isGuestBrowsable } from './guest-routes';

describe('isGuestBrowsable', () => {
  it('lets a guest open a kitchen and its reviews', () => {
    expect(isGuestBrowsable(['chef', '[id]'])).toBe(true);
    expect(isGuestBrowsable(['chef', '[id]', 'reviews'])).toBe(true);
  });

  it('lets a guest read ChefBook', () => {
    expect(isGuestBrowsable(['chefbook'])).toBe(true);
    expect(isGuestBrowsable(['chefbook', '[slug]'])).toBe(true);
  });

  it('lets a guest browse the map and dish search', () => {
    expect(isGuestBrowsable(['chefs-map'])).toBe(true);
    expect(isGuestBrowsable(['search-dishes'])).toBe(true);
  });

  // App Review expects the policies to be readable before anyone signs up.
  it('lets a guest read the legal pages', () => {
    expect(isGuestBrowsable(['terms'])).toBe(true);
    expect(isGuestBrowsable(['privacy'])).toBe(true);
    expect(isGuestBrowsable(['eula'])).toBe(true);
    expect(isGuestBrowsable(['legal'])).toBe(true);
    expect(isGuestBrowsable(['data-privacy'])).toBe(true);
  });

  it('still sends a guest home from anything that needs an account', () => {
    expect(isGuestBrowsable(['checkout'])).toBe(false);
    expect(isGuestBrowsable(['cart'])).toBe(false);
    expect(isGuestBrowsable(['wallet'])).toBe(false);
    expect(isGuestBrowsable(['order', '[id]'])).toBe(false);
    expect(isGuestBrowsable(['profile', 'edit'])).toBe(false);
  });

  it('treats an empty route as not browsable, so the guard still runs', () => {
    expect(isGuestBrowsable([])).toBe(false);
  });
});
