/**
 * Shared payout-method logic (#739).
 *
 * The chef supplies payout details in two places — the Settings screen
 * (app/payout.tsx) and the onboarding wizard step — so the parts that must not
 * diverge live here: the API contract, the validation rules, and the payload
 * shape. The two screens differ only in chrome.
 *
 * UPI is not an accepted payout method (#767): Razorpay Route settles by
 * NEFT/IMPS to a bank account and has no VPA destination, so a chef who
 * nominated UPI could never be paid. Bank transfer is the only option.
 *
 * Sensitive values (account number) are POSTed straight to /chef/payout, which
 * stores them in GCP Secret Manager. They are deliberately never written to the
 * onboarding draft in AsyncStorage — that would put a chef's bank account
 * number in device plaintext and undo the whole reason the backend blanks those
 * columns.
 */

export type PayoutMethod = 'bank_transfer';

/** GET /chef/payout — sensitive fields arrive already masked. */
export interface PayoutDetailsResponse {
  payoutMethod: string;
  bankAccountName: string;
  bankAccountNumber: string;
  bankIFSC: string;
  upiId: string;
  razorpayConnected: boolean;
  stripeConnected: boolean;
  paymentProvider: string;
  payoutCountry: string;
}

/** POST /chef/payout body. */
export interface SavePayoutPayload {
  payoutMethod: PayoutMethod;
  bankAccountNumber?: string;
  bankIFSC?: string;
  bankAccountName?: string;
}

/** GET /chef/payout/readiness — the payout-setup gate (#739). */
export interface PayoutReadiness {
  level: 'off' | 'method_on_file' | 'verified';
  methodOnFile: boolean;
  verified: boolean;
  /** Whether the chef satisfies the gate. */
  ready: boolean;
  /** Whether failing the gate actually blocks. False during the grace window. */
  enforced: boolean;
  reasonCode?: 'payout_method_missing' | 'payout_method_unverified';
  graceActive: boolean;
}

/** Raw form state, before trimming. */
export interface PayoutFormValues {
  bankAccountName: string;
  bankAccountNumber: string;
  bankIFSC: string;
}

export const emptyPayoutForm: PayoutFormValues = {
  bankAccountName: '',
  bankAccountNumber: '',
  bankIFSC: '',
};

/**
 * IFSC is 4 letters, then 0, then 6 alphanumerics — the RBI format. Validating
 * it client-side matters more here than in most forms: a typo does not surface
 * as a form error but as a payout that silently fails days later, at which
 * point the chef has already cooked and delivered the food.
 */
const IFSC_PATTERN = /^[A-Z]{4}0[A-Z0-9]{6}$/;

/** Indian bank account numbers run 9–18 digits across banks. */
const ACCOUNT_PATTERN = /^\d{9,18}$/;

export interface PayoutValidationError {
  field: keyof PayoutFormValues;
  message: string;
}

/**
 * Validates the bank-transfer fields.
 *
 * Returns every problem rather than only the first, so the chef fixes the
 * whole form in one pass instead of discovering errors one submit at a time.
 */
export function validatePayoutInput(values: PayoutFormValues): PayoutValidationError[] {
  const errors: PayoutValidationError[] = [];

  if (!values.bankAccountName.trim()) {
    errors.push({ field: 'bankAccountName', message: 'Enter the account holder name' });
  }

  const account = values.bankAccountNumber.replace(/\s/g, '');
  if (!account) {
    errors.push({ field: 'bankAccountNumber', message: 'Enter your account number' });
  } else if (!ACCOUNT_PATTERN.test(account)) {
    errors.push({
      field: 'bankAccountNumber',
      message: 'Account numbers are 9 to 18 digits',
    });
  }

  const ifsc = values.bankIFSC.trim().toUpperCase();
  if (!ifsc) {
    errors.push({ field: 'bankIFSC', message: 'Enter your IFSC code' });
  } else if (!IFSC_PATTERN.test(ifsc)) {
    errors.push({ field: 'bankIFSC', message: 'That does not look like a valid IFSC' });
  }
  return errors;
}

/** Builds the POST body for the bank-transfer payout. */
export function buildPayoutPayload(values: PayoutFormValues): SavePayoutPayload {
  return {
    payoutMethod: 'bank_transfer',
    bankAccountName: values.bankAccountName.trim(),
    bankAccountNumber: values.bankAccountNumber.replace(/\s/g, ''),
    bankIFSC: values.bankIFSC.trim().toUpperCase(),
  };
}

/**
 * A non-sensitive summary safe to keep in the onboarding draft and show on the
 * Review step. Never contains a full account number.
 */
export function summarisePayout(values: PayoutFormValues): string {
  const account = values.bankAccountNumber.replace(/\s/g, '');
  return `Bank ••••${account.slice(-4)}`;
}
