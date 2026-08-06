// The chef screen's account-gated actions, in one place.
//
// Browsing a kitchen — menu, reviews, likes and subscriber counts — is open to
// anyone, so the sign-up prompt has to sit on the individual actions that need an
// identity rather than on the screen. Without it a guest's tap reached an
// auth-only endpoint, 401'd, and left the button exactly as it was.

import { useToggleChefLike, useToggleChefSubscription } from './useChefAudience';
import { useToggleFavorite, type ToggleFavoriteParams } from './useFavorites';
import { useRequireAccount } from './useRequireAccount';

export interface ChefGuestActions {
  /** @param liked - the chef's current like state, which the tap toggles away from. */
  toggleLike: (liked: boolean) => void;
  toggleSubscribe: (subscribed: boolean) => void;
  toggleFavorite: (params: ToggleFavoriteParams) => void;
  /** True while a like or subscribe is in flight, so a double tap can't race. */
  busy: boolean;
}

export function useChefGuestActions(chefId: string | undefined): ChefGuestActions {
  const requireAccount = useRequireAccount();
  const like = useToggleChefLike(chefId);
  const subscribe = useToggleChefSubscription(chefId);
  const favorite = useToggleFavorite();

  return {
    toggleLike: (liked) => {
      if (!requireAccount('like a kitchen')) return;
      like.mutate(liked);
    },
    toggleSubscribe: (subscribed) => {
      if (!requireAccount('subscribe to a kitchen')) return;
      subscribe.mutate(subscribed);
    },
    toggleFavorite: (params) => {
      if (!requireAccount('save a chef')) return;
      favorite.mutate(params);
    },
    busy: like.isPending || subscribe.isPending,
  };
}
