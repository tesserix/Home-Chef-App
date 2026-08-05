import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

// Chef tiffin-subscription offer (#284). Wired to GET/PUT
// /chef/subscription-config — the same pair the vendor portal uses
// (apps/vendor-portal/src/features/subscriptions/pages/SubscriptionSetupPage.tsx).
// The customer offer and the price engine read this config, so what the chef
// sets here is what a subscriber is quoted.

/** Meal slots a subscription can cover. Must match the API's accepted values. */
export const SUBSCRIPTION_SLOTS: { value: string; label: string }[] = [
  { value: 'breakfast', label: 'Breakfast' },
  { value: 'lunch', label: 'Lunch' },
  { value: 'dinner', label: 'Dinner' },
];

/** Billing cadences a customer can pick. */
export const SUBSCRIPTION_CADENCES: { value: string; label: string }[] = [
  { value: 'weekly', label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' },
];

export interface SubscriptionConfig {
  enabled: boolean;
  slots: string[];
  cadences: string[];
  perMealPrice: number;
  deliveryFee: number;
  dailyCapacity: number;
  cutoffTime: string; // "HH:MM" IST
  trialEnabled: boolean;
  trialDurationDays: number;
  trialPrice: number;
}

export interface SubscriptionConfigResponse {
  config: SubscriptionConfig;
  /**
   * Enabling the offer is refused server-side without a published weekly menu
   * (#1) — surfaced so the screen can say why the switch is blocked instead of
   * letting the chef discover it through a failed save.
   */
  hasPublishedMenu: boolean;
}

/** A never-configured chef gets zeroes, matching the API's fallback record. */
export const EMPTY_SUBSCRIPTION_CONFIG: SubscriptionConfig = {
  enabled: false,
  slots: ['lunch'],
  cadences: ['weekly', 'monthly'],
  perMealPrice: 0,
  deliveryFee: 0,
  dailyCapacity: 0,
  cutoffTime: '21:00',
  trialEnabled: false,
  trialDurationDays: 3,
  trialPrice: 0,
};

export function useSubscriptionConfig() {
  return useQuery<SubscriptionConfigResponse>({
    queryKey: ['chef', 'subscription-config'],
    queryFn: () =>
      api.get<SubscriptionConfigResponse>('/chef/subscription-config').then((r) => r.data),
    staleTime: 60_000,
  });
}

export function useUpdateSubscriptionConfig() {
  const queryClient = useQueryClient();
  return useMutation<SubscriptionConfig, Error, SubscriptionConfig>({
    mutationFn: (body) =>
      api
        .put<{ config: SubscriptionConfig }>('/chef/subscription-config', body)
        .then((r) => r.data.config),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['chef', 'subscription-config'] }),
  });
}
