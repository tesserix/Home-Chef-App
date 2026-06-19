// Chef-side catering hooks (#55) — mirror apps/api/handlers/catering.go chef
// endpoints, matching the web chef CateringPage flow (open requests → submit
// quote → booked events → complete).
//
// GET  /chef/catering/requests            → open requests the chef can quote on
// POST /chef/catering/requests/:id/quote  → submit a quote
// GET  /chef/catering/quotes              → the chef's submitted quotes
// GET  /chef/catering/bookings            → confirmed/completed bookings (won)
// POST /chef/catering/requests/:id/complete → mark a booking completed

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export type CateringRequestStatus =
  | 'open'
  | 'quoted'
  | 'accepted'
  | 'confirmed'
  | 'completed'
  | 'cancelled';

export interface CateringRequest {
  id: string;
  status: CateringRequestStatus;
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
  quotesCount?: number;
  quoteDeadline?: string;
  depositAmount?: number;
  depositStatus?: string;
  createdAt: string;
}

export interface CateringQuote {
  id: string;
  requestId: string;
  chefId: string;
  status: 'pending' | 'accepted' | 'rejected' | 'expired';
  proposedMenu: string;
  menuItems: string[];
  pricePerPerson: number;
  totalPrice: number;
  depositAmount: number;
  notes?: string;
  includesSetup: boolean;
  includesServing: boolean;
  includesCleanup: boolean;
  includesEquipment: boolean;
  validUntil?: string;
  acceptedAt?: string;
  createdAt: string;
}

export interface CateringBooking {
  request: CateringRequest;
  quote: CateringQuote;
}

export interface SubmitQuoteInput {
  proposedMenu: string;
  menuItems?: string[];
  pricePerPerson: number;
  totalPrice: number;
  depositAmount?: number;
  notes?: string;
  includesSetup?: boolean;
  includesServing?: boolean;
  includesCleanup?: boolean;
  includesEquipment?: boolean;
  validDays?: number;
}

interface ListResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
}

export function useAvailableCateringRequests(filters: Record<string, string> = {}) {
  return useQuery<ListResponse<CateringRequest>>({
    queryKey: ['chef-catering-requests', filters],
    queryFn: () =>
      api
        .get('/chef/catering/requests', { params: filters })
        .then((r) => r.data as ListResponse<CateringRequest>),
    staleTime: 1000 * 30,
  });
}

export function useMyCateringQuotes(status?: string) {
  return useQuery<ListResponse<CateringQuote>>({
    queryKey: ['chef-catering-quotes', status ?? 'all'],
    queryFn: () =>
      api
        .get('/chef/catering/quotes', { params: status ? { status } : {} })
        .then((r) => r.data as ListResponse<CateringQuote>),
    staleTime: 1000 * 30,
  });
}

export function useCateringBookings() {
  return useQuery<{ data: CateringBooking[] }>({
    queryKey: ['chef-catering-bookings'],
    queryFn: () =>
      api.get('/chef/catering/bookings').then((r) => r.data as { data: CateringBooking[] }),
    staleTime: 1000 * 30,
  });
}

export function useSubmitCateringQuote() {
  const qc = useQueryClient();
  return useMutation<unknown, Error, { requestId: string; input: SubmitQuoteInput }>({
    mutationFn: ({ requestId, input }) =>
      api.post(`/chef/catering/requests/${requestId}/quote`, input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['chef-catering-requests'] });
      void qc.invalidateQueries({ queryKey: ['chef-catering-quotes'] });
    },
  });
}

export function useCompleteCateringBooking() {
  const qc = useQueryClient();
  return useMutation<unknown, Error, string>({
    mutationFn: (requestId: string) =>
      api.post(`/chef/catering/requests/${requestId}/complete`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chef-catering-bookings'] }),
  });
}
