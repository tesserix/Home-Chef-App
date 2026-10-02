import { useEffect, useRef, useState } from 'react';
import { loadStripeJs, type StripeElements, type StripeInstance, type StripePaymentElement } from '@/shared/utils/load-stripe';

export type StripeCheckoutPayment = {
  clientSecret: string;
  publishableKey: string;
  stripePaymentIntentId: string;
};

type Props = {
  orderId: string;
  payment: StripeCheckoutPayment;
  onConfirmed: () => Promise<void> | void;
  onCancel: () => void;
};

export function StripeCheckout({ orderId, payment, onConfirmed, onCancel }: Props) {
  const mount = useRef<HTMLDivElement>(null);
  const checkout = useRef<{ stripe: StripeInstance; elements: StripeElements } | null>(null);
  const submitting = useRef(false);
  const [ready, setReady] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let disposed = false;
    let element: StripePaymentElement | undefined;
    void (async () => {
      try {
        const stripe = await loadStripeJs(payment.publishableKey);
        if (disposed) return;
        if (!stripe || !mount.current) throw new Error('Stripe could not load. Return to your order and retry.');
        const elements = stripe.elements({ clientSecret: payment.clientSecret });
        element = elements.create('payment');
        element.on('ready', () => { if (!disposed) setReady(true); });
        checkout.current = { stripe, elements };
        element.mount(mount.current);
      } catch {
        if (!disposed) setError('Stripe could not load. Return to your order and retry.');
      }
    })();
    return () => { disposed = true; checkout.current = null; element?.destroy(); };
  }, [payment.clientSecret, payment.publishableKey]);

  async function pay() {
    if (!ready || !checkout.current || submitting.current) return;
    submitting.current = true;
    setBusy(true);
    setError(null);
    try {
      const { stripe, elements } = checkout.current;
      const result = await stripe.confirmPayment({
        elements,
        confirmParams: { return_url: `${window.location.origin}/orders/${encodeURIComponent(orderId)}?stripe_pi=${encodeURIComponent(payment.stripePaymentIntentId)}` },
        redirect: 'if_required',
      });
      if (result.error) { setError(result.error.message || 'Payment could not be completed.'); return; }
      await onConfirmed();
    } catch {
      setError('Payment status is uncertain. Return to your order to check before retrying.');
    } finally {
      submitting.current = false;
      setBusy(false);
    }
  }

  return (
    <section className="mx-auto max-w-lg space-y-4 p-6" aria-label="Stripe payment">
      <h1 className="text-2xl font-semibold">Complete your payment</h1>
      <p>Your payment details are collected securely by Stripe.</p>
      <form onSubmit={(event) => { event.preventDefault(); void pay(); }} className="space-y-4">
        <div ref={mount} />
        {!ready && !error && <p role="status">Loading secure payment form…</p>}
        {error && <p role="alert">{error}</p>}
        <button type="submit" disabled={!ready || busy} className="rounded-lg bg-ink px-4 py-3 text-white disabled:opacity-50">{busy ? 'Confirming…' : 'Pay securely'}</button>
        <button type="button" disabled={busy} onClick={onCancel} className="ml-3 rounded-lg border px-4 py-3">Return to order</button>
      </form>
    </section>
  );
}
