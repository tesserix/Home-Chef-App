// fssai.ts — the chef's FSSAI filing request, client side.
//
// Every figure here comes from the SERVER (`/chef/fssai/quote`). Nothing is
// priced, taxed or re-added in the app: the same quote drives the screen, the
// Cashfree charge and the admin's verification, so the number a chef agrees to
// cannot differ from the one they are charged.
//
// See .planning/FSSAI-IN-APP-REQUEST-DESIGN.md.

/** Priced by the server for one term. Render these lines; never re-sum them. */
export interface FssaiQuote {
  termYears: number;
  /** FSSAI's own fee for the whole term, before tax. Paid to the government. */
  governmentFee: number;
  governmentTax: number;
  /** Ours, once per application whatever the term. */
  serviceFee: number;
  serviceTax: number;
  /** The exact sum of the four lines above. */
  total: number;
  currency: string;
  gstPercent: number;
}

/** What FSSAI requires. `address_proof` only when the kitchen address differs
 *  from the one on the submitted photo ID. */
export type FssaiDocumentKind = 'photo' | 'identity' | 'address_proof';

export const FSSAI_REQUIRED_DOCUMENTS: FssaiDocumentKind[] = ['photo', 'identity'];
export const FSSAI_DOCUMENT_KINDS: FssaiDocumentKind[] = [
  'photo',
  'identity',
  'address_proof',
];

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
  /** The FoSCoS reference, once we have filed. This is what lets a chef track
   *  their own application with the state authority, independently of us. */
  applicationRef?: string;
  registrationNo?: string;
  rejectedReason?: string;
  submittedAt?: string | null;
  filedAt?: string | null;
  issuedAt?: string | null;
  documents: FssaiDocument[];
  /** Server-computed: whether FSSAI's required documents are still missing. The
   *  app must not re-derive this — the rules are FSSAI's, not ours. */
  needsDocuments: boolean;
  /** Server-computed: unpaid and complete, so the payment may be started. */
  canPay: boolean;
  createdAt: string;
}

/** The four steps a chef sees once they have paid. `awaiting_payment` is the
 *  draft they are still building, so it is a form rather than a step. */
export const FSSAI_TRACKER_STEPS = [
  { key: 'submitted', label: 'Submitted' },
  { key: 'in_progress', label: 'We’re preparing your form' },
  { key: 'filed', label: 'Filed with FSSAI' },
  { key: 'issued', label: 'Registration issued' },
] as const;

const STEP_ORDER: FssaiStatus[] = ['submitted', 'in_progress', 'filed', 'issued'];

/** How many tracker steps are complete. -1 while the request is still an unpaid
 *  draft, which the screen renders as the form rather than a tracker. */
export function fssaiStepIndex(status: FssaiStatus): number {
  return STEP_ORDER.indexOf(status);
}

/** A request that is finished, one way or another. */
export function isFssaiClosed(status: FssaiStatus): boolean {
  return status === 'issued' || status === 'rejected' || status === 'refunded';
}

/** True once money has been taken. Past this the chef cannot cancel and the fee
 *  is not refundable. */
export function isFssaiPaid(status: FssaiStatus): boolean {
  return status !== 'awaiting_payment';
}

/** What the chef must do next, or null when the ball is with us. Drives the
 *  single call-to-action on the screen, so the app never shows two. */
export function fssaiChefAction(r: FssaiRequest | null): 'documents' | 'pay' | null {
  if (!r || r.status !== 'awaiting_payment') return null;
  return r.needsDocuments ? 'documents' : 'pay';
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

/** Hosted checkout on fe3dr.com, opened in a browser session.
 *
 *  Deliberately not an in-app WebView: react-native-webview is a native module,
 *  so adopting it would strand this behind a new store build. The page is shared
 *  with every other non-customer surface, so there is one checkout to maintain.
 */
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
