// Catering hooks — endpoints confirmed from apps/api/handlers/catering.go
// POST /v1/catering/requests      → create request (body: CateringRequestInput)
// GET  /v1/catering/requests      → list my requests
// GET  /v1/catering/requests/:id  → single request with quotes
// POST /v1/catering/quotes/:id/accept → accept a quote

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export interface CateringRequest {
  id: string;
  status: 'open' | 'quoted' | 'accepted' | 'completed' | 'cancelled';
  eventType: string;
  eventDate: string;
  eventTime?: string;
  guestCount: number;
  budget?: number;
  cuisineTypes?: string[];
  dietaryNeeds?: string[];
  menuStyle?: string;
  description?: string;
  venueName?: string;
  addressLine1?: string;
  city: string;
  state: string;
  postalCode?: string;
  contactName?: string;
  contactPhone?: string;
  contactEmail?: string;
  quoteDeadline?: string;
  createdAt: string;
}

export interface CateringListResponse {
  data: CateringRequest[];
  total: number;
  page: number;
  limit: number;
}

export interface CreateCateringInput {
  eventType: string;
  eventDate: string; // YYYY-MM-DD
  eventTime?: string;
  guestCount: number;
  budget?: number;
  cuisineTypes?: string[];
  dietaryNeeds?: string[];
  menuStyle?: string;
  description?: string;
  venueName?: string;
  addressLine1?: string;
  addressLine2?: string;
  city: string;
  state: string;
  postalCode?: string;
  latitude?: number;
  longitude?: number;
  contactName?: string;
  contactPhone?: string;
  contactEmail?: string;
}

export function useCateringRequests(params: { page?: number; status?: string } = {}) {
  return useQuery<CateringListResponse>({
    queryKey: ['catering-requests', params],
    queryFn: () =>
      api
        .get('/v1/catering/requests', { params })
        .then((r) => r.data as CateringListResponse),
    staleTime: 1000 * 60, // 1 minute
  });
}

export function useCreateCateringRequest() {
  const queryClient = useQueryClient();

  return useMutation<{ data: CateringRequest }, Error, CreateCateringInput>({
    mutationFn: (payload: CreateCateringInput) =>
      api
        .post('/v1/catering/requests', payload)
        .then((r) => r.data as { data: CateringRequest }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['catering-requests'] });
    },
  });
}
