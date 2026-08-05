// Hosted Cashfree checkout.
//
// ONE payment surface for every client that is not the customer app. The chef
// app opens this in a browser session rather than embedding a WebView, because
// react-native-webview is a NATIVE module: adopting it would force a new store
// build before any chef could pay, where this ships with the page. The same URL
// serves any web surface, so there is no second implementation to keep in step.
//
// The client is never trusted for the outcome. Cashfree hands back no signed
// success payload, so this page only reports "the sheet closed" — the server
// re-fetches the captured payment from Cashfree before anything is marked paid.

'use client';

import { Suspense, useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';

const SDK_SRC = 'https://sdk.cashfree.com/js/v3/cashfree.js';

// Only our own app schemes and our own origin may be returned to. Without this
// the page would bounce a visitor anywhere a crafted `ret` pointed — a
// redirector wearing our domain, on a page people arrive at mid-payment.
// The trailing slash is load-bearing: without it `https://fe3dr.com.evil.com`
// would pass the prefix test.
const ALLOWED_RETURN_PREFIXES = [
  'homechef-vendor://',
  'homechef-customer://',
  'homechef-delivery://',
  'https://fe3dr.com/',
  'https://vendors.fe3dr.com/',
];

function safeReturnUrl(raw: string | null): string | null {
  if (!raw) return null;
  return ALLOWED_RETURN_PREFIXES.some((p) => raw.startsWith(p)) ? raw : null;
}

function CheckoutRunner() {
  const params = useSearchParams();
  const [message, setMessage] = useState('Opening secure checkout…');

  const session = params.get('session');
  const mode = params.get('mode') === 'sandbox' ? 'sandbox' : 'production';
  const ret = safeReturnUrl(params.get('ret'));

  useEffect(() => {
    if (!session) {
      setMessage('This payment link is missing its session. Go back and try again.');
      return;
    }

    // Hand control back to whoever sent us here. `status` is a hint for the UI
    // only — the server decides whether the money actually arrived.
    const leave = (status: string) => {
      if (!ret) {
        setMessage(
          status === 'error'
            ? 'Payment could not be completed. You can close this window.'
            : 'Payment finished. You can close this window and return to the app.',
        );
        return;
      }
      const sep = ret.includes('?') ? '&' : '?';
      window.location.href = `${ret}${sep}status=${encodeURIComponent(status)}`;
    };

    // Loaded unpinned and without SRI, as Cashfree documents: they ship fixes to
    // this SDK continuously and a pinned hash would hard-fail checkout the moment
    // they publish one. Safe because the SDK is never trusted for the outcome.
    const script = document.createElement('script');
    script.src = SDK_SRC;
    script.async = true;
    script.onerror = () => setMessage('Could not reach the payment provider. Check your connection.');
    script.onload = () => {
      try {
        // @ts-expect-error — injected by the Cashfree SDK at runtime.
        const cashfree = Cashfree({ mode });
        cashfree
          .checkout({ paymentSessionId: session, redirectTarget: '_modal' })
          .then((result: { error?: { message?: string } } | undefined) => {
            leave(result?.error ? 'error' : 'settled');
          })
          // A throw is not proof of failure: the chef may have paid and the sheet
          // errored afterwards. Settle and let the server adjudicate.
          .catch(() => leave('settled'));
      } catch {
        setMessage('Could not open checkout. Go back and try again.');
      }
    };
    document.body.appendChild(script);
    return () => {
      script.remove();
    };
  }, [session, mode, ret]);

  return <p className="text-charcoal-soft">{message}</p>;
}

export default function PayPage() {
  return (
    <main
      id="main"
      className="mx-auto flex min-h-[60vh] max-w-md flex-col items-center justify-center px-5 text-center"
    >
      <h1 className="font-display text-xl font-semibold tracking-tight text-charcoal">
        Secure payment
      </h1>
      <div className="mt-3">
        {/* useSearchParams needs a boundary for the static export to prerender. */}
        <Suspense fallback={<p className="text-charcoal-soft">Loading…</p>}>
          <CheckoutRunner />
        </Suspense>
      </div>
      <p className="mt-8 text-sm text-charcoal-soft">
        Payments are processed by Cashfree. Fe3dr never sees your card details.
      </p>
    </main>
  );
}
