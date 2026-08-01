import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// The one reusable source for platform-level pricing: GET /platform/config,
// the same policy CreateOrder charges against. Nothing money-shaped should
// hardcode a rate — read it from here so an admin change propagates everywhere.

export interface PlatformConfig {
  /** Percent of subtotal charged as the platform fee (includes gateway costs). */
  platformFeePercent: number;
  taxPercent: number;
  baseDeliveryFee: number;
  timezone: string;
  openingTime: string;
  closingTime: string;
  operatingDays: number[] | null;
  isOpen: boolean;
  closedMessage?: string;
}

/** Matches DefaultPlatformPolicy on the server — used until the config loads. */
export const DEFAULT_PLATFORM_FEE_PERCENT = 4.99;

export function usePlatformConfig() {
  return useQuery<PlatformConfig>({
    queryKey: ['platform-config'],
    queryFn: () => apiClient.get<PlatformConfig>('/platform/config'),
    staleTime: 1000 * 60 * 5,
  });
}

/** The live platform-fee percent, falling back to the server default while loading. */
export function usePlatformFeePercent(): number {
  const { data } = usePlatformConfig();
  return data?.platformFeePercent ?? DEFAULT_PLATFORM_FEE_PERCENT;
}
