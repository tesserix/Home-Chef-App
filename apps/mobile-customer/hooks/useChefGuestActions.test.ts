import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// A guest may browse a kitchen, its menu and its reviews — that is the whole point
// of guest mode. But Like, Subscribe and Save all POST to auth-only endpoints, so
// tapping one signed out fired a request that 401'd and changed nothing on screen:
// no prompt, no error, no sign-up. The gate belongs in front of the mutation.

const mockRequireAccount = jest.fn<(action: string) => boolean>();
const mockLikeMutate = jest.fn();
const mockSubscribeMutate = jest.fn();
const mockFavoriteMutate = jest.fn();

jest.mock('./useRequireAccount', () => ({ useRequireAccount: () => mockRequireAccount }));
jest.mock('./useChefAudience', () => ({
  useToggleChefLike: () => ({ mutate: mockLikeMutate, isPending: false }),
  useToggleChefSubscription: () => ({ mutate: mockSubscribeMutate, isPending: false }),
}));
jest.mock('./useFavorites', () => ({
  useToggleFavorite: () => ({ mutate: mockFavoriteMutate, isPending: false }),
}));

import { useChefGuestActions } from './useChefGuestActions';

const signedIn = () => mockRequireAccount.mockReturnValue(true);
const guest = () => mockRequireAccount.mockReturnValue(false);

beforeEach(() => {
  mockRequireAccount.mockReset();
  mockLikeMutate.mockReset();
  mockSubscribeMutate.mockReset();
  mockFavoriteMutate.mockReset();
});

describe('useChefGuestActions gates the actions a guest cannot perform', () => {
  it('sends no like for a guest, and asks them to sign in instead', () => {
    guest();

    useChefGuestActions('chef-1').toggleLike(false);

    expect(mockLikeMutate).not.toHaveBeenCalled();
    expect(mockRequireAccount).toHaveBeenCalledWith('like a kitchen');
  });

  it('sends no subscribe for a guest', () => {
    guest();

    useChefGuestActions('chef-1').toggleSubscribe(false);

    expect(mockSubscribeMutate).not.toHaveBeenCalled();
    expect(mockRequireAccount).toHaveBeenCalledWith('subscribe to a kitchen');
  });

  it('sends no favorite for a guest', () => {
    guest();

    useChefGuestActions('chef-1').toggleFavorite({ chefId: 'chef-1', isFavorited: false });

    expect(mockFavoriteMutate).not.toHaveBeenCalled();
    expect(mockRequireAccount).toHaveBeenCalledWith('save a chef');
  });

  it('passes the toggle straight through once there is an account', () => {
    signedIn();
    const actions = useChefGuestActions('chef-1');

    actions.toggleLike(true);
    actions.toggleSubscribe(true);
    actions.toggleFavorite({ chefId: 'chef-1', isFavorited: true });

    // The current state is what each mutation toggles away from.
    expect(mockLikeMutate).toHaveBeenCalledWith(true);
    expect(mockSubscribeMutate).toHaveBeenCalledWith(true);
    expect(mockFavoriteMutate).toHaveBeenCalledWith({ chefId: 'chef-1', isFavorited: true });
  });
});
