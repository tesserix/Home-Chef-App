import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@/shared/services/api-client';

// FSSAI filing request — the web twin of apps/mobile-vendor/hooks/useFssai.ts.
//
// The rules live on the server and are read from the response, never restated
// here: pricing lines, `needsDocuments`, `canPay` and the non-refundable notice
// all arrive priced and decided. This file is transport plus types.

export interface FssaiQuote {
  termYears: number;
  /** FSSAI's own fee for the whole term, before tax. Paid to the government. */
  governmentFee: number;
  governmentTax: number;
  /** Ours, once per application whatever the term. */
  serviceFee: number;
  serviceTax: number;
  /** The exact sum of the four lines above. Never re-add them. */
  total: number;
  currency: string;
  gstPercent: number;
}

export type FssaiDocumentKind = 'photo' | 'identity' | 'address_proof';

export const FSSAI_DOCUMENT_KINDS: FssaiDocumentKind[] = [
  'photo',
  'identity',
  'address_proof',
];

export const FSSAI_DOCUMENT_LABELS: Record<FssaiDocumentKind, string> = {
  photo: 'Passport-style photo',
  identity: 'Government photo ID',
  address_proof: 'Kitchen address proof',
};

export const FSSAI_DOCUMENT_HINTS: Record<FssaiDocumentKind, string> = {
  photo: 'A clear photo of your face, like a passport photo.',
  identity: 'Aadhaar, PAN or Voter ID — all four corners readable.',
  address_proof: 'Only if your kitchen is not at the address on that ID.',
};

export interface FssaiDocument {
  kind: FssaiDocumentKind;
  fileName: string;
  /** Short-lived signed URL, minted per response. Never store or share it. */
  fileUrl?: string;
}

export type FssaiStatus =
  | 'awaiting_payment'
  | 'submitted'
  | 'in_progress'
  | 'filed'
  | 'issued'
  | 'rejected'
  | 'refunded';

export interface FssaiRequest {
  id: string;
  status: FssaiStatus;
  termYears: number;
  kitchenName: string;
  applicantName: string;
  feeAmount: number;
  feeTax: number;
  feeTotal: number;
  currency: string;
  paidAt?: string | null;
  /** The FoSCoS reference, once we have filed — how a chef tracks their own
   *  application with the state authority, independently of us. */
  applicationRef?: string;
  registrationNo?: string;
  rejectedReason?: string;
  submittedAt?: string | null;
  filedAt?: string | null;
  issuedAt?: string | null;
  documents: FssaiDocument[];
  /** Server-computed. The app must not re-derive FSSAI's document rules. */
  needsDocuments: boolean;
  /** Server-computed: unpaid and complete, so payment may be started. */
  canPay: boolean;
  createdAt: string;
}

export const FSSAI_TRACKER_STEPS = [
  { key: 'submitted', label: 'Submitted' },
  { key: 'in_progress', label: 'We’re preparing your form' },
  { key: 'filed', label: 'Filed with FSSAI' },
  { key: 'issued', label: 'Registration issued' },
] as const;

const STEP_ORDER: FssaiStatus[] = ['submitted', 'in_progress', 'filed', 'issued'];

/** How many tracker steps are complete. -1 while the request is an unpaid
 *  draft, which the page renders as the form rather than a tracker. */
export function fssaiStepIndex(status: FssaiStatus): number {
  return STEP_ORDER.indexOf(status);
}

export function isFssaiClosed(status: FssaiStatus): boolean {
  return status === 'issued' || status === 'rejected' || status === 'refunded';
}

/** True once money has been taken. Past this the chef cannot cancel and the
 *  fee is not refundable. */
export function isFssaiPaid(status: FssaiStatus): boolean {
  return status !== 'awaiting_payment';
}

export function fssaiStatusLabel(status: FssaiStatus): string {
  switch (status) {
    case 'awaiting_payment':
      return 'Not sent yet';
    case 'submitted':
      return 'With our team';
    case 'in_progress':
      return 'Being prepared';
    case 'filed':
      return 'Filed with FSSAI';
    case 'issued':
      return 'Registration issued';
    case 'rejected':
      return 'Could not proceed';
    case 'refunded':
      return 'Refunded';
    default:
      return status;
  }
}

interface QuoteResponse {
  enabled: boolean;
  quote: FssaiQuote;
  /** Every term 1–5, so the picker renders without a call per year. */
  terms: FssaiQuote[];
  nonRefundable: boolean;
  nonRefundableNotice: string;
}

/** What onboarding already told us, so the chef corrects a form rather than
 *  retyping one. A starting point, not an authority — FSSAI wants the address
 *  the food is actually cooked at, and only the chef knows if that still
 *  matches what they registered with us. */
export interface FssaiPrefill {
  kitchenName: string;
  applicantName: string;
  contactPhone: string;
  contactEmail: string;
  addressLine1: string;
  addressLine2: string;
  city: string;
  state: string;
  postalCode: string;
}

interface RequestResponse {
  request: FssaiRequest | null;
  enabled: boolean;
  prefill?: FssaiPrefill;
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

const REQUEST_KEY = ['chef', 'fssai', 'request'];
const QUOTE_KEY = ['chef', 'fssai', 'quote'];

/** Pricing and whether the service is open. The figures only change when an
 *  admin edits the policy, so it is cached for the session. */
export function useFssaiQuote() {
  return useQuery<QuoteResponse>({
    queryKey: QUOTE_KEY,
    queryFn: () => apiClient.get<QuoteResponse>('/chef/fssai/quote'),
    staleTime: 5 * 60_000,
  });
}

/** The chef's current request, or null. */
export function useFssaiRequest() {
  return useQuery<RequestResponse>({
    queryKey: REQUEST_KEY,
    queryFn: () => apiClient.get<RequestResponse>('/chef/fssai/request'),
    staleTime: 15_000,
  });
}

/** Refreshes the request after any mutation, so no call site can forget and
 *  leave the page showing a stale step. */
function useRefreshRequest() {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: REQUEST_KEY });
  };
}

/** Starts the chef's draft. Nothing is charged: the documents attach to the
 *  row before it can be paid for, so the row has to exist first. */
export function useCreateFssaiRequest() {
  const refresh = useRefreshRequest();
  return useMutation<FssaiRequest, unknown, CreateFssaiRequestInput>({
    mutationFn: (input) =>
      apiClient
        .post<{ request: FssaiRequest }>('/chef/fssai/requests', input)
        .then((r) => r.request),
    onSuccess: refresh,
  });
}

/** Uploads a document and attaches it in one call. Multipart, so it goes
 *  through postForm — the browser must set the boundary itself. */
export function useUploadFssaiDocument() {
  const refresh = useRefreshRequest();
  return useMutation<
    FssaiRequest,
    unknown,
    { requestId: string; kind: FssaiDocumentKind; file: File }
  >({
    mutationFn: ({ requestId, kind, file }) => {
      const form = new FormData();
      form.append('file', file);
      form.append('kind', kind);
      return apiClient
        .postForm<{ request: FssaiRequest }>(`/chef/fssai/requests/${requestId}/upload`, form)
        .then((r) => r.request);
    },
    onSuccess: refresh,
  });
}

/** Removes an optional document added by mistake. The required two are
 *  replaced by re-uploading, never removed. */
export function useRemoveFssaiDocument() {
  const refresh = useRefreshRequest();
  return useMutation<unknown, unknown, { requestId: string; kind: FssaiDocumentKind }>({
    mutationFn: ({ requestId, kind }) =>
      apiClient.delete(`/chef/fssai/requests/${requestId}/documents/${kind}`),
    onSuccess: refresh,
  });
}

/** Mints the Cashfree session for a completed draft. The server refuses if the
 *  documents are not in, so money is never taken against a request we cannot
 *  file. */
export function useStartFssaiPayment() {
  const refresh = useRefreshRequest();
  return useMutation<CheckoutResponse, unknown, string>({
    mutationFn: (requestId) =>
      apiClient.post<CheckoutResponse>(`/chef/fssai/requests/${requestId}/checkout`),
    onSettled: refresh,
  });
}

/** Asks the server to verify the capture with Cashfree. The browser never
 *  decides that a payment succeeded — it has nothing signed to prove it. */
export function useConfirmFssaiPayment() {
  const refresh = useRefreshRequest();
  return useMutation<FssaiRequest, unknown, string>({
    mutationFn: (requestId) =>
      apiClient
        .post<{ request: FssaiRequest }>(`/chef/fssai/requests/${requestId}/confirm`)
        .then((r) => r.request),
    onSettled: refresh,
  });
}

/** Discards a draft the chef has not paid for. */
export function useCancelFssaiRequest() {
  const refresh = useRefreshRequest();
  return useMutation<unknown, unknown, string>({
    mutationFn: (requestId) => apiClient.delete(`/chef/fssai/requests/${requestId}`),
    onSuccess: refresh,
  });
}

/** Hosted Cashfree checkout on fe3dr.com — the same page the mobile app opens,
 *  so there is one checkout to maintain rather than a web-only variant. */
export function buildCashfreeCheckoutUrl(opts: {
  paymentSessionId: string;
  /** "SANDBOX" | "PRODUCTION", resolved server-side — never guessed here. */
  env?: string;
  returnUrl: string;
}): string {
  const mode = (opts.env ?? '').toUpperCase() === 'SANDBOX' ? 'sandbox' : 'production';
  const q = new URLSearchParams({
    session: opts.paymentSessionId,
    mode,
    ret: opts.returnUrl,
  });
  return `https://fe3dr.com/pay/?${q.toString()}`;
}
