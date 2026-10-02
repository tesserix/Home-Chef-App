// Minimal Stripe.js loader. Loading the SDK on demand (rather than a build
// dependency on @stripe/stripe-js) keeps the bundle small for the India-only
// Cashfree customer journey — the Stripe path only runs when a chef is
// configured for international payouts.
//
// Uses the official loader script from js.stripe.com which is required by
// Stripe's PCI compliance rules (their SDK must be served fresh, not
// bundled).

type StripeError = { message?: string };

export type StripePaymentElement = {
  mount: (target: HTMLElement) => void;
  destroy: () => void;
  on: (event: 'ready', listener: () => void) => void;
};

export type StripeElements = {
  create: (type: 'payment') => StripePaymentElement;
};

export type StripeInstance = {
  elements: (options: { clientSecret: string }) => StripeElements;
  confirmPayment: (args: {
    elements: StripeElements;
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
let loadPromise: Promise<boolean> | null = null;

export async function loadStripeJs(publishableKey: string): Promise<StripeInstance | null> {
  if (!publishableKey) return null;
  if (typeof window.Stripe === 'function') return window.Stripe(publishableKey);

  if (!loadPromise) {
    loadPromise = new Promise<boolean>((resolve) => {
      const existing = document.querySelector<HTMLScriptElement>(`script[src="${SCRIPT_SRC}"]`);
      const script = existing ?? document.createElement('script');
      const finish = (loaded: boolean) => {
        clearTimeout(timeout);
        script.removeEventListener('load', onLoad);
        script.removeEventListener('error', onError);
        if (!loaded) script.remove();
        resolve(loaded);
      };
      const onLoad = () => finish(typeof window.Stripe === 'function');
      const onError = () => finish(false);
      const timeout = setTimeout(onError, 15_000);
      script.addEventListener('load', onLoad, { once: true });
      script.addEventListener('error', onError, { once: true });
      if (!existing) {
        script.src = SCRIPT_SRC;
        script.async = true;
        document.head.appendChild(script);
      }
    });
  }
  const pending = loadPromise;
  const loaded = await pending;
  if (!loaded) {
    if (loadPromise === pending) loadPromise = null;
    return null;
  }
  const constructor = window.Stripe as StripeConstructor | undefined;
  return constructor?.(publishableKey) ?? null;
}
