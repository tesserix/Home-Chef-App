// useFssai.ts — server state for the chef's FSSAI filing request.
//
// The request is the single source of truth for what the chef must do next
// (`fssaiChefAction`), so every mutation refreshes it rather than reasoning
// about the next state locally. A screen that guessed would offer "Pay" against
// documents the server never received.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { multipartConfig } from '@homechef/mobile-shared/api';
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

interface CheckoutResponse {
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

/** Refreshes the request after any mutation. One helper so no call site can
 *  forget and leave the screen showing a stale step. */
function useRefreshRequest() {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: REQUEST_KEY });
  };
}

/** Starts the chef's draft. Nothing is charged: documents are attached to the
 *  row before it can be paid for, so the row has to exist first. */
export function useCreateFssaiRequest() {
  const refresh = useRefreshRequest();
  return useMutation({
    mutationFn: (input: CreateFssaiRequestInput) =>
      api
        .post<{ request: FssaiRequest }>('/chef/fssai/requests', input)
        .then((r) => r.data.request),
    onSuccess: refresh,
  });
}

/** Uploads a document and attaches it in one call. The file never round-trips
 *  through the client as a reference — the server names the stored object, so a
 *  client cannot point a request at someone else's identity documents. */
export function useUploadFssaiDocument() {
  const refresh = useRefreshRequest();
  return useMutation({
    mutationFn: async (vars: {
      requestId: string;
      kind: FssaiDocumentKind;
      uri: string;
      mimeType?: string;
    }) => {
      const name = vars.uri.split('/').pop() ?? `${vars.kind}.jpg`;
      const body = new FormData();
      body.append('file', {
        uri: vars.uri,
        name,
        type: vars.mimeType ?? 'image/jpeg',
      } as unknown as Blob);
      body.append('kind', vars.kind);
      const res = await api.post<{ request: FssaiRequest }>(
        `/chef/fssai/requests/${vars.requestId}/upload`,
        body,
        multipartConfig(),
      );
      return res.data.request;
    },
    onSuccess: refresh,
  });
}

/** Removes an optional document the chef added by mistake. The required two
 *  are replaced by re-uploading, never removed. */
export function useRemoveFssaiDocument() {
  const refresh = useRefreshRequest();
  return useMutation({
    mutationFn: (vars: { requestId: string; kind: FssaiDocumentKind }) =>
      api
        .delete<{ request: FssaiRequest }>(
          `/chef/fssai/requests/${vars.requestId}/documents/${vars.kind}`,
        )
        .then((r) => r.data.request),
    onSuccess: refresh,
  });
}

/** Mints the Cashfree session for a completed draft. The server refuses if the
 *  documents are not in, so the money can never be taken against a request we
 *  cannot file. */
export function useStartFssaiPayment() {
  const refresh = useRefreshRequest();
  return useMutation({
    mutationFn: (requestId: string) =>
      api
        .post<CheckoutResponse>(`/chef/fssai/requests/${requestId}/checkout`)
        .then((r) => r.data),
    onSettled: refresh,
  });
}

/** Asks the server to verify the capture with Cashfree. The app never decides
 *  that a payment succeeded — it has nothing signed to prove it. */
export function useConfirmFssaiPayment() {
  const refresh = useRefreshRequest();
  return useMutation({
    mutationFn: (requestId: string) =>
      api
        .post<{ request: FssaiRequest }>(`/chef/fssai/requests/${requestId}/confirm`)
        .then((r) => r.data.request),
    onSettled: refresh,
  });
}

/** Discards a draft the chef has not paid for. The service is optional and they
 *  may change their mind — but only until the money moves. */
export function useCancelFssaiRequest() {
  const refresh = useRefreshRequest();
  return useMutation({
    mutationFn: (requestId: string) =>
      api.delete(`/chef/fssai/requests/${requestId}`).then((r) => r.data),
    onSuccess: refresh,
  });
}
