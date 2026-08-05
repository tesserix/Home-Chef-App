import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import { mapChef } from './useChefs';
import type { Chef } from '../types/customer';

export interface ChefAudience {
  chefId: string;
  liked: boolean;
  subscribed: boolean;
  likeCount: number;
  subscriberCount: number;
}

export interface ChefSubscriptionEntry {
  id: string;
  chefId: string;
  chef: Chef;
  notifyMenu: boolean;
  notifyPriceChange: boolean;
  notifyAvailability: boolean;
  notifyArticles: boolean;
  createdAt: string;
}

export const chefAudienceKey = (chefId: string) => ['chef-audience', chefId];

/** The viewer's own like/subscribe state plus the public totals. */
export function useChefAudience(chefId: string | undefined) {
  return useQuery<ChefAudience>({
    queryKey: chefAudienceKey(chefId ?? ''),
    enabled: Boolean(chefId),
    queryFn: async () =>
      (await api.get(`/v1/chefs/${chefId}/audience`)).data.audience as ChefAudience,
    staleTime: 1000 * 60,
  });
}

type AudienceAction = 'like' | 'subscribe';

/**
 * Toggles a like or a subscription.
 *
 * Every endpoint returns the resulting audience state, so the response is
 * written straight into the cache rather than triggering a refetch — the count
 * next to the button is the thing the customer just changed, and a round trip
 * to re-read it is what makes these buttons feel laggy.
 */
function useAudienceToggle(chefId: string | undefined, action: AudienceAction) {
  const queryClient = useQueryClient();

  return useMutation<ChefAudience, Error, boolean>({
    mutationFn: async (isOn: boolean) => {
      const path = `/v1/chefs/${chefId}/${action}`;
      const res = isOn ? await api.delete(path) : await api.post(path);
      return res.data.audience as ChefAudience;
    },
    onSuccess: (audience) => {
      queryClient.setQueryData(chefAudienceKey(chefId ?? ''), audience);
      // Ranking and the "kitchens you follow" list both move with this.
      void queryClient.invalidateQueries({ queryKey: ['chefs'] });
      if (action === 'subscribe') {
        void queryClient.invalidateQueries({ queryKey: ['chef-subscriptions'] });
      }
    },
  });
}

export const useToggleChefLike = (chefId: string | undefined) =>
  useAudienceToggle(chefId, 'like');

export const useToggleChefSubscription = (chefId: string | undefined) =>
  useAudienceToggle(chefId, 'subscribe');

/** The kitchens this customer subscribes to. */
export function useChefSubscriptions() {
  return useQuery<ChefSubscriptionEntry[]>({
    queryKey: ['chef-subscriptions'],
    queryFn: async () => {
      const raw = (await api.get('/v1/me/chef-subscriptions')).data as {
        subscriptions?: ChefSubscriptionEntry[];
      };
      // mapChef is the single API→UI translation point (see useChefs); without
      // it the chef card renders an undefined image.
      return (raw.subscriptions ?? []).map((entry) => ({
        ...entry,
        chef: mapChef(entry.chef as unknown as Parameters<typeof mapChef>[0]),
      }));
    },
    staleTime: 1000 * 60 * 2,
  });
}

export type ChefNotifyKind =
  | 'notifyMenu'
  | 'notifyPriceChange'
  | 'notifyAvailability'
  | 'notifyArticles';

/** Flips one notification kind on an existing subscription. */
export function useUpdateSubscriptionNotify(chefId: string | undefined) {
  const queryClient = useQueryClient();

  return useMutation<ChefAudience, Error, Partial<Record<ChefNotifyKind, boolean>>>({
    mutationFn: async (prefs) =>
      (await api.patch(`/v1/chefs/${chefId}/subscribe`, prefs)).data.audience as ChefAudience,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['chef-subscriptions'] });
    },
  });
}
