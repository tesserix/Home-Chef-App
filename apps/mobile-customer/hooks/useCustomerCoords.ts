import { useMemo } from 'react';
import { useAddresses } from './useAddresses';

export interface Coords {
  lat: number;
  lng: number;
}

/**
 * The customer's active location for discovery + delivery-reach checks.
 *
 * Sourced from the saved delivery addresses: the default address's coordinates,
 * else the first address that has real coordinates. Returns null when no address
 * has usable coordinates — callers then omit lat/lng and the API falls back to
 * un-located discovery (all nearby chefs, no delivery-area gate). Coordinates are
 * captured when the customer picks an address from the geocoder, so a customer
 * who only typed a raw address stays un-located rather than mis-located.
 */
export function useCustomerCoords(): Coords | null {
  const { data } = useAddresses();
  const addresses = data?.data ?? [];

  return useMemo(() => {
    const withCoords = addresses.filter(
      (a) =>
        typeof a.latitude === 'number' &&
        typeof a.longitude === 'number' &&
        (a.latitude !== 0 || a.longitude !== 0),
    );
    if (withCoords.length === 0) return null;

    const preferred = withCoords.find((a) => a.isDefault) ?? withCoords[0];
    return { lat: preferred.latitude as number, lng: preferred.longitude as number };
  }, [addresses]);
}
