import type { Address } from '../types/customer';

type AddressParts = Pick<Address, 'addressLine1' | 'addressLine2' | 'city' | 'state' | 'pincode'>;

interface Place {
  streetNumber: string | null;
  street: string | null;
  name: string | null;
  city: string | null;
  region: string | null;
  postalCode: string | null;
}

const joinParts = (parts: (string | null | undefined)[]): string =>
  parts.map((p) => p?.trim()).filter(Boolean).join(', ');

/** "12 Queen St, Level 4, Auckland, Auckland 1010". */
export function fullAddress(a: AddressParts): string {
  const regionLine = [a.state, a.pincode].map((p) => p?.trim()).filter(Boolean).join(' ');
  return joinParts([a.addressLine1, a.addressLine2, a.city, regionLine]);
}

/** The customer's own name for the address ("Office"), else the full address. */
export function savedAddressLabel(a: Address): string {
  return a.label?.trim() || fullAddress(a);
}

/** Full street address from an OS reverse-geocode result. */
export function placeLabel(place: Place): string {
  const street = [place.streetNumber, place.street].filter(Boolean).join(' ') || place.name || '';
  return fullAddress({
    addressLine1: street === place.city ? '' : street,
    city: place.city ?? '',
    state: place.region ?? '',
    pincode: place.postalCode ?? '',
  });
}
