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
 * Picks the customer's ACTIVE address — the one shown in the home-screen switcher
 * and used as the delivery target: the default address, else the first saved.
 *
 * This must honour the customer's explicit choice. It deliberately does NOT
 * filter on coordinates: when the customer switches to an address that has no
 * saved map location, the switcher still shows THAT address (with a "no location"
 * hint) rather than silently reverting to a different, coordinate-bearing one.
 * Coordinates for discovery are handled separately in useCustomerCoords, which
 * degrades to un-located discovery when the active address has no point — never
 * mis-locates to another address.
 */
export function pickActiveAddress(addresses: Address[]): Address | undefined {
  if (addresses.length === 0) return undefined;
  return addresses.find((a) => a.isDefault) ?? addresses[0];
}

/**
 * The customer's active location for discovery + delivery-reach checks.
 *
 * Sourced from the ACTIVE address (see pickActiveAddress) — the same address the
 * switcher shows. Returns its coordinates when it has a usable point; returns null
 * when it doesn't, so callers omit lat/lng and the API falls back to un-located
 * discovery (all nearby chefs, no delivery-area gate). It never borrows another
 * address's coordinates, so a customer who switched to a not-yet-located address
 * stays un-located rather than mis-located to a different city.
 */
export function useCustomerCoords(): Coords | null {
  const { data } = useAddresses();
  const addresses = data?.data ?? [];

  return useMemo(() => {
    const active = pickActiveAddress(addresses);
    if (!active || !hasUsableCoords(active)) return null;
    return { lat: active.latitude as number, lng: active.longitude as number };
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
