import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// A guest arrives with device GPS and no delivery address. Land them in a city
// with no kitchens yet — the simulator's San Francisco, say — and the geo filter
// emptied the feed, so the one thing guest browsing exists for (see a real
// kitchen, tap it, want to sign up) never appeared. When the located feed comes
// back empty for a guest, drop the area filters rather than show a wall.

import type { ChefFilters } from './useChefs';

const mockUseChefs = jest.fn<(f: ChefFilters, o?: { enabled?: boolean }) => unknown>();
const mockUseIsGuest = jest.fn<() => boolean>();

// Wrapped, not passed by reference: the factory runs before the consts above are
// initialised, so a bare reference lands as undefined.
jest.mock('./useChefs', () => ({
  useChefs: (f: ChefFilters, o?: { enabled?: boolean }) => mockUseChefs(f, o),
}));
jest.mock('./useRequireAccount', () => ({ useIsGuest: () => mockUseIsGuest() }));

import { useDiscoveryChefs } from './useDiscoveryChefs';

const chef = { id: 'chef-1', name: 'Saffron Home Kitchen' };

function result(chefs: unknown[], isLoading = false) {
  return { data: { data: chefs }, isLoading, isFetching: false, refetch: jest.fn() };
}

const located: ChefFilters = { lat: 37.78, lng: -122.4, radius: 25, cuisine: 'North Indian' };

beforeEach(() => {
  mockUseChefs.mockReset();
  mockUseIsGuest.mockReset();
});

describe('useDiscoveryChefs', () => {
  it('shows the located kitchens when the area has some', () => {
    mockUseIsGuest.mockReturnValue(true);
    mockUseChefs.mockReturnValue(result([chef]));

    const r = useDiscoveryChefs(located);

    expect(r.chefs).toEqual([chef]);
    expect(r.showingBeyondArea).toBe(false);
    // The fallback query stays disabled while the area has kitchens.
    expect(mockUseChefs).toHaveBeenLastCalledWith(expect.anything(), { enabled: false });
  });

  it('falls back to kitchens beyond the area when a guest has none nearby', () => {
    mockUseIsGuest.mockReturnValue(true);
    mockUseChefs.mockReturnValueOnce(result([])).mockReturnValueOnce(result([chef]));

    const r = useDiscoveryChefs(located);

    expect(r.chefs).toEqual([chef]);
    expect(r.showingBeyondArea).toBe(true);
    // Same filters minus the geography — a guest's cuisine choice still holds.
    expect(mockUseChefs).toHaveBeenLastCalledWith(
      { cuisine: 'North Indian' },
      { enabled: true },
    );
  });

  it('leaves a signed-in customer with the honest empty area', () => {
    mockUseIsGuest.mockReturnValue(false);
    mockUseChefs.mockReturnValue(result([]));

    const r = useDiscoveryChefs(located);

    expect(r.chefs).toEqual([]);
    expect(r.showingBeyondArea).toBe(false);
    expect(mockUseChefs).toHaveBeenLastCalledWith(expect.anything(), { enabled: false });
  });

  it('waits for the located feed before deciding it is empty', () => {
    mockUseIsGuest.mockReturnValue(true);
    mockUseChefs.mockReturnValue(result([], true));

    const r = useDiscoveryChefs(located);

    expect(r.isLoading).toBe(true);
    expect(r.showingBeyondArea).toBe(false);
    expect(mockUseChefs).toHaveBeenLastCalledWith(expect.anything(), { enabled: false });
  });

  it('does not fall back when there was no area filter to begin with', () => {
    mockUseIsGuest.mockReturnValue(true);
    mockUseChefs.mockReturnValue(result([]));

    const r = useDiscoveryChefs({ cuisine: 'Chinese' });

    expect(r.showingBeyondArea).toBe(false);
    expect(mockUseChefs).toHaveBeenLastCalledWith(expect.anything(), { enabled: false });
  });
});
