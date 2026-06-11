import { describe, it, expect, jest } from '@jest/globals';

// The functions under test are pure mappers; mock lib/api so importing the
// module doesn't pull axios + the auth store at test time.
jest.mock('../lib/api', () => ({ api: { get: jest.fn() } }));

import { mapChef, mapMenuItem } from './useChefs';

describe('mapChef', () => {
  it('translates the API shape to the customer UI shape', () => {
    const chef = mapChef({
      id: 'c1',
      businessName: 'Anita Kitchen',
      cuisines: ['North Indian', 'Chinese'],
      rating: 4.5,
      totalReviews: 12,
      isOnline: true,
      acceptingOrders: true,
      bannerImage: 'banner.jpg',
      prepTime: '30 min',
      minimumOrder: 150,
      deliveryFee: 29,
      latitude: 18.5,
      longitude: 73.8,
    });

    expect(chef).toMatchObject({
      id: 'c1',
      name: 'Anita Kitchen',
      cuisine: 'North Indian · Chinese',
      rating: 4.5,
      reviewCount: 12,
      isOpen: true,
      imageUrl: 'banner.jpg',
      deliveryTime: '30 min',
      minimumOrder: 150,
      deliveryFee: 29,
      latitude: 18.5,
      longitude: 73.8,
    });
  });

  it('is open only when online AND accepting orders', () => {
    expect(mapChef({ id: 'c', isOnline: true, acceptingOrders: false }).isOpen).toBe(false);
    expect(mapChef({ id: 'c', isOnline: false, acceptingOrders: true }).isOpen).toBe(false);
    expect(mapChef({ id: 'c', isOnline: true, acceptingOrders: true }).isOpen).toBe(true);
  });

  it('prefers banner > kitchen photo > avatar for the card image', () => {
    expect(
      mapChef({ id: 'c', bannerImage: 'b.jpg', kitchenPhotos: ['k.jpg'], profileImage: 'p.jpg' }).imageUrl,
    ).toBe('b.jpg');
    expect(
      mapChef({ id: 'c', kitchenPhotos: ['k.jpg'], profileImage: 'p.jpg' }).imageUrl,
    ).toBe('k.jpg');
    expect(mapChef({ id: 'c', profileImage: 'p.jpg' }).imageUrl).toBe('p.jpg');
    expect(mapChef({ id: 'c' }).imageUrl).toBeUndefined();
  });

  it('applies safe defaults for missing fields', () => {
    const chef = mapChef({ id: 'c' });
    expect(chef.name).toBe('');
    expect(chef.cuisine).toBe('');
    expect(chef.rating).toBe(0);
    expect(chef.reviewCount).toBe(0);
    expect(chef.isOpen).toBe(false);
  });
});

describe('mapMenuItem', () => {
  const categories = new Map([['cat-1', 'Starters']]);

  it('maps fields and resolves category name', () => {
    const mi = mapMenuItem(
      { id: 'm1', chefId: 'c1', categoryId: 'cat-1', name: 'Samosa', price: 40 },
      categories,
    );
    expect(mi).toMatchObject({
      id: 'm1',
      chefId: 'c1',
      name: 'Samosa',
      price: 40,
      category: 'Starters',
      isAvailable: true, // defaults true when omitted
    });
  });

  it('prefers imageUrl, then images[0].url', () => {
    expect(
      mapMenuItem({ id: 'm', chefId: 'c', name: 'x', price: 1, imageUrl: 'a.jpg', images: [{ url: 'b.jpg' }] }, categories).imageUrl,
    ).toBe('a.jpg');
    expect(
      mapMenuItem({ id: 'm', chefId: 'c', name: 'x', price: 1, images: [{ url: 'b.jpg' }] }, categories).imageUrl,
    ).toBe('b.jpg');
  });

  it('honors an explicit isAvailable:false', () => {
    expect(
      mapMenuItem({ id: 'm', chefId: 'c', name: 'x', price: 1, isAvailable: false }, categories).isAvailable,
    ).toBe(false);
  });
});
