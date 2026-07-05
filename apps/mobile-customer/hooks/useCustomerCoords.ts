import { useMemo } from 'react';
import { useAddresses } from './useAddresses';
import type { Address } from '../types/customer';

export interface Coords {
  lat: number;
  lng: number;
}

// Exported so callers that need to flag "this address won't affect nearby
// chefs" (e.g. AddressSwitcherSheet) use the exact same test as
// pickActiveAddress, rather than a second, possibly-diverging copy.
export function hasUsableCoords(a: Address): boolean {
  return (
    typeof a.latitude === 'number' &&
    typeof a.longitude === 'number' &&
    (a.latitude !== 0 || a.longitude !== 0)
  );
}

/**
 * Picks the address that drives chef discovery + delivery-reach checks: the
 * default address among those with usable coordinates, else the first address
 * that has coordinates. Returns undefined when no address has usable coords.
 *
 * Exported (not just used internally by useCustomerCoords) so the home-screen
 * address switcher (components/address/AddressSwitcher.tsx) can display the
 * SAME address that is actually driving discovery — the two must never
 * disagree on "which address is active."
 */
export function pickActiveAddress(addresses: Address[]): Address | undefined {
  const withCoords = addresses.filter(hasUsableCoords);
  if (withCoords.length === 0) return undefined;
  return withCoords.find((a) => a.isDefault) ?? withCoords[0];
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
    const preferred = pickActiveAddress(addresses);
    if (!preferred) return null;
    return { lat: preferred.latitude as number, lng: preferred.longitude as number };
  }, [addresses]);
}

/**
 * The full Address record backing useCustomerCoords's selection, plus the raw
 * list and load state — for UI that needs to SHOW and let the customer CHANGE
 * the active address (the home-screen switcher), not just consume its coords.
 */
export function useActiveAddress(): {
  address: Address | undefined;
  addresses: Address[];
  isLoading: boolean;
} {
  const { data, isLoading } = useAddresses();
  const addresses = data?.data ?? [];
  const address = useMemo(() => pickActiveAddress(addresses), [addresses]);
  return { address, addresses, isLoading };
}
