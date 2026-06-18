// Catering hooks — endpoints from apps/api/handlers/catering.go (#55)
// POST /v1/catering/requests              → create request
// GET  /v1/catering/requests              → list my requests
// GET  /v1/catering/requests/:id          → single request + quotes
// POST /v1/catering/quotes/:id/accept     → accept a quote
// POST /v1/catering/quotes/:id/decline    → decline a quote
// POST /v1/catering/requests/:id/cancel   → cancel a request
// POST /v1/catering/requests/:id/deposit  → start the deposit charge (Razorpay)

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export type CateringStatus =
  | 'open'
  | 'quoted'
  | 'accepted'
  | 'confirmed'
  | 'completed'
  | 'cancelled';

export type DepositStatus = 'none' | 'pending' | 'paid';

export interface CateringRequest {
  id: string;
  status: CateringStatus;
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
  addressLine2?: string;
  city: string;
  state: string;
  postalCode?: string;
  contactName?: string;
  contactPhone?: string;
  contactEmail?: string;
  quoteDeadline?: string;
  acceptedQuoteId?: string;
  depositAmount?: number;
  depositStatus?: DepositStatus;
  depositPaidAt?: string;
  completedAt?: string;
  createdAt: string;
}

export interface CateringQuote {
  id: string;
  requestId: string;
  chefId: string;
  chef: {
    id: string;
    businessName: string;
    profileImage?: string;
    rating?: number;
    totalReviews?: number;
    verified?: boolean;
  };
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

export interface CateringRequestDetail {
  data: CateringRequest;
  quotes: CateringQuote[];
}

export interface DepositCharge {
  razorpayOrderId: string;
  razorpayKeyId: string;
  amount: number; // paise
  currency: string;
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

// Single request + its quotes — backs the detail screen.
export function useCateringRequest(id?: string) {
  return useQuery<CateringRequestDetail>({
    queryKey: ['catering-request', id],
    queryFn: () =>
      api.get(`/v1/catering/requests/${id}`).then((r) => r.data as CateringRequestDetail),
    enabled: !!id,
  });
}

export function useAcceptCateringQuote(requestId?: string) {
  const qc = useQueryClient();
  return useMutation<unknown, Error, string>({
    mutationFn: (quoteId: string) => api.post(`/v1/catering/quotes/${quoteId}/accept`, {}),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['catering-request', requestId] });
      void qc.invalidateQueries({ queryKey: ['catering-requests'] });
    },
  });
}

export function useDeclineCateringQuote(requestId?: string) {
  const qc = useQueryClient();
  return useMutation<unknown, Error, string>({
    mutationFn: (quoteId: string) => api.post(`/v1/catering/quotes/${quoteId}/decline`, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['catering-request', requestId] }),
  });
}

export function useCancelCateringRequest() {
  const qc = useQueryClient();
  return useMutation<unknown, Error, string>({
    mutationFn: (requestId: string) => api.post(`/v1/catering/requests/${requestId}/cancel`, {}),
    onSuccess: (_d, requestId) => {
      void qc.invalidateQueries({ queryKey: ['catering-request', requestId] });
      void qc.invalidateQueries({ queryKey: ['catering-requests'] });
    },
  });
}

// Starts the deposit charge — returns the Razorpay order to hand to the payment
// sheet. Verification happens in app/payment/checkout.tsx (kind=catering).
export function useCreateCateringDeposit() {
  return useMutation<DepositCharge, Error, string>({
    mutationFn: (requestId: string) =>
      api.post(`/v1/catering/requests/${requestId}/deposit`, {}).then((r) => r.data as DepositCharge),
  });
}
