import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Address } from '../types/customer';

export interface CustomerProfile {
  id: string;
  firstName: string;
  lastName: string;
  email: string;
  phone?: string;
  avatar?: string;
  cuisinePreferences?: string[];
  dietaryPreferences?: string[];
  foodAllergies?: string[];
  spiceTolerance?: string;
  householdSize?: string;
}

// The Go API returns the profile FLAT (top-level fields), not wrapped in a
// { data } envelope. Kept as an alias so existing imports keep compiling.
export type CustomerProfileResponse = CustomerProfile;

export interface AddressListResponse {
  data: Address[];
  count: number;
}

export type UpdateProfilePayload = Partial<
  Pick<
    CustomerProfile,
    | 'firstName'
    | 'lastName'
    | 'phone'
    | 'cuisinePreferences'
    | 'dietaryPreferences'
    | 'foodAllergies'
    | 'spiceTolerance'
    | 'householdSize'
  >
>;

export function useProfile() {
  return useQuery<CustomerProfile>({
    queryKey: ['profile'],
    queryFn: () =>
      api.get('/v1/customer/profile').then((r) => r.data as CustomerProfile),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

export function useUpdateProfile() {
  const queryClient = useQueryClient();

  return useMutation<CustomerProfile, Error, UpdateProfilePayload>({
    mutationFn: (payload: UpdateProfilePayload) =>
      api
        .put('/v1/customer/profile', payload)
        .then((r) => r.data as CustomerProfile),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['profile'] });
    },
  });
}

export function useAddresses() {
  return useQuery<AddressListResponse>({
    queryKey: ['addresses'],
    queryFn: () =>
      api.get('/v1/addresses').then((r) => r.data as AddressListResponse),
    staleTime: 1000 * 60 * 5,
  });
}
