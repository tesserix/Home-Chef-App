import { useEffect, useLayoutEffect } from 'react';
import { AppState } from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { useAuthStore } from '../store/auth-store';
import { useCartStore } from '../store/cart-store';
import { useActiveAddress } from './useCustomerCoords';

export function SavedCartSync() {
  const { address } = useActiveAddress();
  const userId = useAuthStore(s => s.user?.id);
  const authenticated = useAuthStore(s => s.isAuthenticated);
  const hydrated = useCartStore(s => s.hasHydrated);
  const baskets = useCartStore(s => s.baskets);
  const ids = Object.keys(baskets).sort();
  const key = JSON.stringify([userId ?? 'guest', address?.id, address?.country, address?.latitude, address?.longitude]);
  useLayoutEffect(() => {
    if (!hydrated) return;
    const store = useCartStore.getState();
    store.setOwner(userId ?? 'guest');
    if (authenticated) {
      store.beginLocation(key);
      if (address?.id && Object.keys(store.baskets).length === 0) store.applyAvailability(key, []);
    }
  }, [hydrated, userId, authenticated, key, address?.id]);

  const availability = useQuery({
    queryKey: ['cart-availability', key, ids],
    enabled: hydrated && authenticated && !!address?.id && ids.length > 0,
    queryFn: async () => {
      const eligible: string[] = [];
      for (let start = 0; start < ids.length; start += 100) {
        const response = await api.post<{ chefIds: string[] }>(`/v1/addresses/${address!.id}/cart-availability`, { chefIds: ids.slice(start, start + 100) });
        eligible.push(...response.data.chefIds);
      }
      return eligible;
    },
    staleTime: 30_000,
    retry: 1,
  });
  useLayoutEffect(() => {
    if (availability.data) useCartStore.getState().applyAvailability(key, availability.data);
    else if (availability.isError) useCartStore.getState().applyAvailability(key, null);
  }, [key, availability.data, availability.isError]);
  useEffect(() => {
    const subscription = AppState.addEventListener('change', state => {
      if (state === 'active') {
        useCartStore.getState().pruneExpired();
        if (authenticated && address?.id && ids.length) void availability.refetch();
      }
    });
    return () => subscription.remove();
  }, [authenticated, address?.id, ids.join(','), availability.refetch]);
  return null;
}
