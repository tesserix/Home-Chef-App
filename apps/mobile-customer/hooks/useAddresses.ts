// Hooks for fetching and creating delivery addresses.

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Address } from '../types/customer';

export function useAddresses() {
  return useQuery<{ data: Address[] }>({
    queryKey: ['addresses'],
    queryFn: () => api.get('/v1/addresses').then((r) => r.data),
    staleTime: 1000 * 60 * 5, // 5 minutes
  });
}

export function useCreateAddress() {
  const queryClient = useQueryClient();
  return useMutation<{ data: Address }, Error, Omit<Address, 'id'>>({
    mutationFn: (payload) => api.post('/v1/addresses', payload).then((r) => r.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['addresses'] });
    },
  });
}
