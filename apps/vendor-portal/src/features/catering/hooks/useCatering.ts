import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// Chef catering — the web twin of apps/mobile-vendor/app/catering.tsx.
//
// Customers post an event brief; chefs quote for it and, if accepted, it becomes
// a booking. The portal had no catering surface, so a chef on the web never saw
// open briefs and could not quote — the work simply went to whoever had the app.
//
// Backed by /chef/catering/{requests,quotes,bookings}.

export interface CateringRequest {
  id: string;
  status: string;
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
  quoteDeadline?: string;
}

export interface CateringQuote {
  id: string;
  requestId: string;
  status: string;
  proposedMenu: string;
  menuItems?: string[];
  pricePerPerson: number;
  totalPrice: number;
  depositAmount?: number;
  notes?: string;
  createdAt?: string;
}

/** Unwrap either a bare array or a {data} envelope. */
function list<T>(r: T[] | { data?: T[] } | null | undefined): T[] {
  if (Array.isArray(r)) return r;
  return r?.data ?? [];
}

/** Open briefs this chef could quote for. */
export function useAvailableCateringRequests() {
  return useQuery<CateringRequest[]>({
    queryKey: ['chef', 'catering', 'requests'],
    queryFn: () =>
      apiClient
        .get<CateringRequest[] | { data: CateringRequest[] }>('/chef/catering/requests')
        .then(list),
    staleTime: 60_000,
  });
}

/** Quotes this chef has already submitted. */
export function useMyCateringQuotes() {
  return useQuery<CateringQuote[]>({
    queryKey: ['chef', 'catering', 'quotes'],
    queryFn: () =>
      apiClient.get<CateringQuote[] | { data: CateringQuote[] }>('/chef/catering/quotes').then(list),
    staleTime: 60_000,
  });
}

/** Quotes a customer accepted — the jobs this chef is actually cooking. */
export function useCateringBookings() {
  return useQuery<CateringQuote[]>({
    queryKey: ['chef', 'catering', 'bookings'],
    queryFn: () =>
      apiClient
        .get<CateringQuote[] | { data: CateringQuote[] }>('/chef/catering/bookings')
        .then(list),
    staleTime: 60_000,
  });
}

export interface SubmitQuoteInput {
  requestId: string;
  proposedMenu: string;
  pricePerPerson: number;
  totalPrice: number;
  notes?: string;
  includesSetup?: boolean;
  includesServing?: boolean;
  includesCleanup?: boolean;
  includesEquipment?: boolean;
}

export function useSubmitCateringQuote() {
  const qc = useQueryClient();
  return useMutation<unknown, { httpStatus?: number }, SubmitQuoteInput>({
    mutationFn: ({ requestId, ...body }) =>
      apiClient.post(`/chef/catering/requests/${requestId}/quote`, body),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['chef', 'catering'] });
    },
  });
}

export function useCompleteCateringBooking() {
  const qc = useQueryClient();
  return useMutation<unknown, unknown, string>({
    mutationFn: (requestId: string) =>
      apiClient.post(`/chef/catering/requests/${requestId}/complete`),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['chef', 'catering'] });
    },
  });
}
