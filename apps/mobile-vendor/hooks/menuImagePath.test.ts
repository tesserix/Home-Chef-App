import { describe, it, expect } from '@jest/globals';

import { menuItemImagesPath } from './useVendorMenu';

// The bug this guards: app/menu/new.tsx constructed useUploadMenuPhoto(itemId)
// while itemId was still '' — the id only exists after the create call
// resolves, and setState does not update the already-built mutation inside the
// same handler. Every photo for a newly created item therefore POSTed to
// `/chef/menu/items//images`, 404'd, and was swallowed by a bare catch, so
// items saved with no photos at all and no error shown.
//
// Confirmed in production: "Lamb Biryani" created through the app landed with
// 0 rows in menu_item_images and an empty image_url.

describe('menuItemImagesPath', () => {
  it('builds the upload path for a real item id', () => {
    expect(menuItemImagesPath('abc-123')).toBe('/chef/menu/items/abc-123/images');
  });

  it('refuses an empty id instead of posting to a collapsed path', () => {
    // `/chef/menu/items//images` is a different route entirely — it 404s, and
    // the failure is invisible to the chef.
    expect(() => menuItemImagesPath('')).toThrow(/item id/i);
  });

  it('refuses a whitespace-only id', () => {
    expect(() => menuItemImagesPath('   ')).toThrow(/item id/i);
  });

  it('refuses an undefined id', () => {
    expect(() => menuItemImagesPath(undefined as unknown as string)).toThrow(/item id/i);
  });
});
