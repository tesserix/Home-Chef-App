// useFssai.ts — server state for the chef's FSSAI filing request.
//
// The request is the single source of truth for what the chef must do next
// (`fssaiChefAction`), so every mutation refreshes it rather than reasoning
// about the next state locally. A screen that guessed would show "upload your
// documents" against a payment the server never saw.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { FssaiDocumentKind, FssaiQuote, FssaiRequest } from '../lib/fssai';

const REQUEST_KEY = ['chef', 'fssai', 'request'];
const QUOTE_KEY = ['chef', 'fssai', 'quote'];

interface QuoteResponse {
  enabled: boolean;
  quote: FssaiQuote;
  /** Every term 1–5, so the picker renders without a call per year. */
  terms: FssaiQuote[];
  nonRefundable: boolean;
  /** Stated by the server so the disclosure and the rule cannot drift apart. */
  nonRefundableNotice: string;
}

interface RequestResponse {
  request: FssaiRequest | null;
  enabled: boolean;
}

interface CreateResponse {
  request: FssaiRequest;
  cashfreePaymentSessionId: string;
  cashfreeOrderId: string;
  cashfreeEnv: string;
}

export interface CreateFssaiRequestInput {
  kitchenName: string;
  applicantName: string;
  contactPhone: string;
  contactEmail: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  state: string;
  postalCode: string;
  termYears: number;
}

/** Pricing and whether the service is open. Cheap and stable, so it is cached
 *  for the session — the figures only change when an admin edits the policy. */
export function useFssaiQuote() {
  return useQuery<QuoteResponse>({
    queryKey: QUOTE_KEY,
    queryFn: () => api.get<QuoteResponse>('/chef/fssai/quote').then((r) => r.data),
    staleTime: 5 * 60_000,
  });
}

/** The chef's current request, or null. */
export function useFssaiRequest() {
  return useQuery<RequestResponse>({
    queryKey: REQUEST_KEY,
    queryFn: () => api.get<RequestResponse>('/chef/fssai/request').then((r) => r.data),
    staleTime: 15_000,
  });
}

/** Creates the request and returns the Cashfree session that pays for it. The
 *  row exists before any money is asked for, so an abandoned payment is a
 *  visible row rather than a silent gap. */
export function useCreateFssaiRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateFssaiRequestInput) =>
      api.post<CreateResponse>('/chef/fssai/requests', input).then((r) => r.data),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: REQUEST_KEY });
    },
  });
}

/** Asks the server to verify the capture with Cashfree. The app never decides
 *  that a payment succeeded — it has nothing signed to prove it. */
export function useConfirmFssaiPayment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (requestId: string) =>
      api.post(`/chef/fssai/requests/${requestId}/confirm`).then((r) => r.data),
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: REQUEST_KEY });
    },
  });
}

/** Attaches an uploaded document. The server decides when the request is
 *  complete and sends it to onboarding — the app does not. */
export function useAttachFssaiDocument() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: {
      requestId: string;
      kind: FssaiDocumentKind;
      fileUrl: string;
      fileName: string;
    }) =>
      api
        .post(`/chef/fssai/requests/${vars.requestId}/documents`, {
          kind: vars.kind,
          fileUrl: vars.fileUrl,
          fileName: vars.fileName,
        })
        .then((r) => r.data),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: REQUEST_KEY });
    },
  });
}

/** Backs out of a request the chef has not paid for. The service is optional
 *  and they may change their mind — but only until the money moves. */
export function useCancelFssaiRequest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (requestId: string) =>
      api.delete(`/chef/fssai/requests/${requestId}`).then((r) => r.data),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: REQUEST_KEY });
    },
  });
}
