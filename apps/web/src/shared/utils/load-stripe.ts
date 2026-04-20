// Minimal Stripe.js loader. Loading the SDK on demand (rather than a build
// dependency on @stripe/stripe-js) keeps the bundle small for the India-only
// Razorpay customer journey — the Stripe path only runs when a chef is
// configured for international payouts.
//
// Uses the official loader script from js.stripe.com which is required by
// Stripe's PCI compliance rules (their SDK must be served fresh, not
// bundled).

type StripeError = { message?: string };

export type StripeInstance = {
  confirmPayment: (args: {
    clientSecret: string;
    confirmParams: { return_url: string };
    redirect?: 'always' | 'if_required';
  }) => Promise<{ error?: StripeError }>;
  retrievePaymentIntent: (clientSecret: string) => Promise<{
    paymentIntent?: { id: string; status: string };
    error?: StripeError;
  }>;
};

type StripeConstructor = (publishableKey: string) => StripeInstance;

declare global {
  interface Window {
    Stripe?: StripeConstructor;
  }
}

const SCRIPT_SRC = 'https://js.stripe.com/v3/';
let loadPromise: Promise<StripeInstance | null> | null = null;

export function loadStripeJs(publishableKey: string): Promise<StripeInstance | null> {
  if (!publishableKey) return Promise.resolve(null);

  if (typeof window.Stripe === 'function') {
    return Promise.resolve(window.Stripe(publishableKey));
  }

  if (loadPromise) return loadPromise;

  loadPromise = new Promise((resolve) => {
    const existing = document.querySelector<HTMLScriptElement>(`script[src="${SCRIPT_SRC}"]`);
    const onReady = () => {
      if (typeof window.Stripe === 'function') {
        resolve(window.Stripe(publishableKey));
      } else {
        resolve(null);
      }
    };
    if (existing) {
      existing.addEventListener('load', onReady, { once: true });
      return;
    }
    const script = document.createElement('script');
    script.src = SCRIPT_SRC;
    script.async = true;
    script.onload = onReady;
    script.onerror = () => resolve(null);
    document.head.appendChild(script);
  });

  return loadPromise;
}
