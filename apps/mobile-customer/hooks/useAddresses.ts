// Hooks for fetching and creating delivery addresses.
//
// Wire-shape note (matches apps/api/handlers/address.go + models/user.go):
//   • GET /v1/addresses returns a BARE ARRAY (not {data}), each item shaped
//     { id, label, line1, line2, city, state, postalCode, isDefault, ... }.
//   • POST /v1/addresses REQUIRES a non-empty `label` (binding:"required")
//     and reads line1 / postalCode (not addressLine1 / pincode).
// The customer app's domain `Address` type uses addressLine1/addressLine2/
// pincode, so we map at this hook boundary — same pattern as the chef mappers
// in hooks/useChefs.ts. Without this map, addressData.data is undefined and the
// checkout "Delivery Address" list renders empty even when a row exists.

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Address } from '../types/customer';

// Raw address shape as returned by the Go API.
interface ApiAddress {
  id: string;
  label?: string;
  line1: string;
  line2?: string;
  city: string;
  state: string;
  postalCode: string;
  isDefault?: boolean;
}

function mapAddress(a: ApiAddress): Address {
  return {
    id: a.id,
    label: a.label || undefined,
    addressLine1: a.line1,
    addressLine2: a.line2 || undefined,
    city: a.city,
    state: a.state,
    pincode: a.postalCode,
    isDefault: a.isDefault,
  };
}

export function useAddresses() {
  return useQuery<{ data: Address[] }>({
    queryKey: ['addresses'],
    queryFn: () =>
      api.get<ApiAddress[]>('/v1/addresses').then((r) => ({
        data: (r.data ?? []).map(mapAddress),
      })),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

export function useCreateAddress() {
  const queryClient = useQueryClient();
  return useMutation<{ data: Address }, Error, Omit<Address, 'id'>>({
    mutationFn: (payload) =>
      api
        .post<ApiAddress>('/v1/addresses', {
          // Backend `label` is required; default to "Home" when the user didn't
          // pick one. line1 / postalCode are the wire field names.
          label: payload.label?.trim() || 'Home',
          line1: payload.addressLine1,
          line2: payload.addressLine2 ?? '',
          city: payload.city,
          state: payload.state,
          postalCode: payload.pincode,
          isDefault: payload.isDefault ?? false,
        })
        .then((r) => ({ data: mapAddress(r.data) })),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['addresses'] });
    },
  });
}
