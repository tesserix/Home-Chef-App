import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// Regression guard for the Saved tab (blank-image bug). The favorites API returns
// the chef as a raw ChefProfileResponse — businessName / bannerImage /
// profileImage — but the Saved screen renders ChefCard/SavedChefRow, which read
// the MAPPED Chef shape (name / imageUrl). useFavorites once passed the raw shape
// straight through, so imageUrl was undefined and every saved chef drew a blank
// grey box. useChefs' mapChef is documented as the single API→UI translation
// point; this test pins that the favorites hook actually runs it.

jest.mock('@tanstack/react-query', () => ({
  useQuery: (opts: { queryFn: () => unknown }) => ({ queryFn: opts.queryFn }),
  useMutation: (opts: { mutationFn: (v: unknown) => unknown }) => ({
    mutationFn: opts.mutationFn,
  }),
  useQueryClient: () => ({ invalidateQueries: jest.fn() }),
}));

jest.mock('../lib/api', () => ({ api: { get: jest.fn(), post: jest.fn(), delete: jest.fn() } }));

import { api } from '../lib/api';
import { useFavorites, type FavoritesResponse } from './useFavorites';

type MockFn = ReturnType<typeof jest.fn>;
const mockApi = api as unknown as { get: MockFn; post: MockFn; delete: MockFn };
const asQuery = (hook: unknown) =>
  hook as unknown as { queryFn: () => Promise<FavoritesResponse> };

// A favorites payload exactly as the backend sends it: the chef is a raw
// ChefProfileResponse, NOT the mapped customer shape.
const rawFavoritesPayload = {
  count: 1,
  max: 7,
  data: [
    {
      id: 'fav-1',
      chefId: 'chef-1',
      createdAt: '2026-07-18T00:00:00Z',
      chef: {
        id: 'chef-1',
        businessName: 'Anita Kitchen',
        bannerImage: 'https://cdn/banner.jpg',
        profileImage: 'https://cdn/avatar.jpg',
        cuisines: ['North Indian', 'Chinese'],
        rating: 4.5,
        totalReviews: 12,
        isOnline: true,
        acceptingOrders: true,
        minimumOrder: 150,
      },
    },
  ],
};

beforeEach(() => {
  mockApi.get.mockClear();
});

describe('useFavorites maps the chef into the UI shape', () => {
  it('reads the versioned favorites endpoint', async () => {
    mockApi.get.mockResolvedValueOnce({ data: rawFavoritesPayload });
    await asQuery(useFavorites()).queryFn();
    expect(mockApi.get).toHaveBeenCalledWith('/v1/favorites/chefs');
  });

  it('populates imageUrl from the chef banner — the blank-image bug', async () => {
    mockApi.get.mockResolvedValueOnce({ data: rawFavoritesPayload });

    const result = await asQuery(useFavorites()).queryFn();
    const chef = result.data[0].chef;

    // These fields exist ONLY on the mapped shape. Their presence proves mapChef ran.
    expect(chef.imageUrl).toBe('https://cdn/banner.jpg');
    expect(chef.name).toBe('Anita Kitchen');
    expect(chef.cuisine).toBe('North Indian · Chinese');
    expect(chef.rating).toBe(4.5);
    expect(chef.reviewCount).toBe(12);
    expect(chef.isOpen).toBe(true);

    // The raw fields must NOT leak through — a screen reading `name`/`imageUrl`
    // gets nothing from them, which is exactly how the bug rendered.
    const asRaw = chef as unknown as Record<string, unknown>;
    expect(asRaw.businessName).toBeUndefined();
    expect(asRaw.bannerImage).toBeUndefined();
  });

  it('preserves the entry envelope (id / chefId / counts) around the mapped chef', async () => {
    mockApi.get.mockResolvedValueOnce({ data: rawFavoritesPayload });

    const result = await asQuery(useFavorites()).queryFn();

    expect(result.count).toBe(1);
    expect(result.max).toBe(7);
    expect(result.data[0]).toMatchObject({ id: 'fav-1', chefId: 'chef-1' });
  });

  it('is safe when the API returns no favorites', async () => {
    mockApi.get.mockResolvedValueOnce({ data: { data: [], count: 0, max: 7 } });
    const result = await asQuery(useFavorites()).queryFn();
    expect(result.data).toEqual([]);
  });
});
