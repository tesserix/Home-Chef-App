/**
 * Shared payout-method logic (#739).
 *
 * The chef now supplies payout details in two places — the Settings screen
 * (app/payout.tsx) and the onboarding wizard step — so the parts that must not
 * diverge live here: the API contract, the validation rules, and the payload
 * shape. The two screens differ only in chrome.
 *
 * Sensitive values (account number, UPI ID) are POSTed straight to
 * /chef/payout, which stores them in GCP Secret Manager. They are deliberately
 * never written to the onboarding draft in AsyncStorage — that would put a
 * chef's bank account number in device plaintext and undo the whole reason the
 * backend blanks those columns.
 */

export type PayoutMethod = 'bank_transfer' | 'upi';

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
  upiId?: string;
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
  upiId: string;
}

export const emptyPayoutForm: PayoutFormValues = {
  bankAccountName: '',
  bankAccountNumber: '',
  bankIFSC: '',
  upiId: '',
};

/**
 * IFSC is 4 letters, then 0, then 6 alphanumerics — the RBI format. Validating
 * it client-side matters more here than in most forms: a typo does not surface
 * as a form error but as a payout that silently fails days later, at which
 * point the chef has already cooked and delivered the food.
 */
const IFSC_PATTERN = /^[A-Z]{4}0[A-Z0-9]{6}$/;

/** UPI VPA: handle@psp. Kept deliberately permissive — PSP suffixes change. */
const VPA_PATTERN = /^[a-zA-Z0-9.\-_]{2,256}@[a-zA-Z]{2,64}$/;

/** Indian bank account numbers run 9–18 digits across banks. */
const ACCOUNT_PATTERN = /^\d{9,18}$/;

export interface PayoutValidationError {
  field: keyof PayoutFormValues;
  message: string;
}

/**
 * Validates the fields the chosen method actually requires.
 *
 * Returns every problem rather than only the first, so the chef fixes the
 * whole form in one pass instead of discovering errors one submit at a time.
 */
export function validatePayoutInput(
  method: PayoutMethod,
  values: PayoutFormValues,
): PayoutValidationError[] {
  const errors: PayoutValidationError[] = [];

  if (method === 'bank_transfer') {
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

  const vpa = values.upiId.trim();
  if (!vpa) {
    errors.push({ field: 'upiId', message: 'Enter your UPI ID' });
  } else if (!VPA_PATTERN.test(vpa)) {
    errors.push({ field: 'upiId', message: 'UPI IDs look like name@bank' });
  }
  return errors;
}

/**
 * Builds the POST body, sending only the fields the chosen method needs.
 *
 * Omitting the other method's fields is deliberate: the backend writes every
 * non-empty field to Secret Manager, so posting a stale UPI ID alongside new
 * bank details would leave a second, unused payout destination on file.
 */
export function buildPayoutPayload(
  method: PayoutMethod,
  values: PayoutFormValues,
): SavePayoutPayload {
  if (method === 'bank_transfer') {
    return {
      payoutMethod: 'bank_transfer',
      bankAccountName: values.bankAccountName.trim(),
      bankAccountNumber: values.bankAccountNumber.replace(/\s/g, ''),
      bankIFSC: values.bankIFSC.trim().toUpperCase(),
    };
  }
  return { payoutMethod: 'upi', upiId: values.upiId.trim() };
}

/**
 * A non-sensitive summary safe to keep in the onboarding draft and show on the
 * Review step. Never contains a full account number or VPA.
 */
export function summarisePayout(method: PayoutMethod, values: PayoutFormValues): string {
  if (method === 'bank_transfer') {
    const account = values.bankAccountNumber.replace(/\s/g, '');
    return `Bank ••••${account.slice(-4)}`;
  }
  const vpa = values.upiId.trim();
  const at = vpa.indexOf('@');
  // Keep the PSP visible (it tells the chef which app will receive the money)
  // and mask the handle, which is the identifying part.
  return at > 0 ? `UPI ••••${vpa.slice(Math.max(0, at - 2))}` : 'UPI';
}
