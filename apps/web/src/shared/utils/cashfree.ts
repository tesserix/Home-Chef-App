import { toast } from 'sonner';

// Cashfree web-checkout helper. Mirrors razorpay.ts so tips (#45), group
// split-pay (#46) and the cart checkout share one create → open → verify path
// regardless of which gateway the kitchen settles through.
//
// Three things differ from the Razorpay helper, and each is forced by the gateway:
//
//  1. THE SDK IS LOADED ON DEMAND. Razorpay's checkout.js is a blocking <script>
//     in index.html because every order used it. Cashfree is per-kitchen, so
//     loading it eagerly would cost every visitor a third-party script they will
//     probably never use. loadCashfreeSdk() injects it the first time a Cashfree
//     order actually reaches checkout and caches the promise.
//
//  2. THE ENVIRONMENT IS EXPLICIT. Razorpay's key id carries rzp_test_/rzp_live_,
//     so the SDK infers it. Cashfree has no such marker — sandbox and production
//     are different hosts — so the mode has to be passed in, and it comes from the
//     SERVER (cashfreeEnv), never inferred client-side. Guessing wrong here opens
//     a checkout against the wrong environment entirely.
//
//  3. THERE IS NO CLIENT SIGNATURE TO HAND BACK. Razorpay returns
//     (payment_id, order_id, signature) that the server re-computes. Cashfree
//     returns only a local result, so the ONLY authority for "was this paid" is
//     our server's verify call, which fetches the payment from Cashfree itself.
//     onSettled therefore fires for any non-error close and lets the server
//     decide — the client's own view of success is never trusted.

const CASHFREE_SDK_URL = 'https://sdk.cashfree.com/js/v3/cashfree.js';

export interface CashfreeChargeData {
  cashfreePaymentSessionId: string;
  cashfreeOrderId: string;
  /** "SANDBOX" | "PRODUCTION" — server-resolved, never inferred here. */
  cashfreeEnv?: string;
  amount: number; // paise, for display parity with the Razorpay payload
  currency: string;
}

// The v3 SDK's surface, narrowed to what we use — it ships no types.
type CashfreeCheckoutResult = {
  error?: { message?: string };
  redirect?: boolean;
  paymentDetails?: { paymentMessage?: string };
};
type CashfreeInstance = {
  checkout: (opts: {
    paymentSessionId: string;
    redirectTarget?: string;
  }) => Promise<CashfreeCheckoutResult>;
};
type CashfreeFactory = (opts: { mode: 'sandbox' | 'production' }) => CashfreeInstance;

declare global {
  interface Window {
    Cashfree?: CashfreeFactory;
  }
}

let sdkPromise: Promise<CashfreeFactory | null> | null = null;

/**
 * Injects the Cashfree v3 SDK once and resolves with its factory.
 *
 * The promise is cached including its failure: a blocked script (ad-blocker,
 * offline, CSP) must not have every subsequent attempt re-inject a <script> that
 * will fail the same way. Resolves null rather than rejecting so callers handle
 * one shape.
 */
function loadCashfreeSdk(): Promise<CashfreeFactory | null> {
  if (window.Cashfree) return Promise.resolve(window.Cashfree);
  if (sdkPromise) return sdkPromise;

  sdkPromise = new Promise((resolve) => {
    const existing = document.querySelector<HTMLScriptElement>(
      `script[src="${CASHFREE_SDK_URL}"]`
    );
    const script = existing ?? document.createElement('script');
    const done = () => resolve(window.Cashfree ?? null);

    script.addEventListener('load', done, { once: true });
    script.addEventListener('error', () => resolve(null), { once: true });

    if (!existing) {
      script.src = CASHFREE_SDK_URL;
      script.async = true;
      document.head.appendChild(script);
    }
  });
  return sdkPromise;
}

/**
 * Opens the Cashfree checkout modal for a prepared payment session.
 *
 * `onSettled` runs when the customer finishes or closes the sheet WITHOUT an
 * SDK-level error. It deliberately does not mean "paid" — the caller must verify
 * against our server, which reads the captured payment from Cashfree. Treating a
 * modal close as success is exactly how an unpaid order gets marked paid.
 *
 * `onDismiss` runs only when the SDK reports a genuine failure or the sheet could
 * not be opened at all.
 */
export async function openCashfreeCheckout(opts: {
  data: CashfreeChargeData;
  onSettled: () => Promise<void> | void;
  onDismiss?: () => void;
}): Promise<void> {
  const factory = await loadCashfreeSdk();
  if (!factory) {
    toast.error('Payment gateway is loading. Please try again.');
    opts.onDismiss?.();
    return;
  }

  // Default to production on an absent/unrecognised value. The asymmetry is
  // deliberate and matches the server's NormalizeMode: a wrong "sandbox" would
  // open a checkout that captures no real money and looks like it worked, while a
  // wrong "production" fails loudly against a sandbox session id.
  const mode = String(opts.data.cashfreeEnv).toUpperCase() === 'SANDBOX' ? 'sandbox' : 'production';

  try {
    const result = await factory({ mode }).checkout({
      paymentSessionId: opts.data.cashfreePaymentSessionId,
      // "_modal" keeps the customer on our page — no full-page redirect away and
      // back, which is why the server sets no return_url.
      redirectTarget: '_modal',
    });

    if (result?.error) {
      toast.error(result.error.message || 'Payment was not completed.');
      opts.onDismiss?.();
      return;
    }
    await opts.onSettled();
  } catch (e: unknown) {
    // An SDK throw is not proof the payment failed — the customer may have paid
    // and the sheet errored afterwards. Hand off to onSettled so the SERVER
    // decides, rather than telling the customer it failed and stranding a
    // captured payment.
    console.error('cashfree checkout error', e);
    await opts.onSettled();
  }
}
