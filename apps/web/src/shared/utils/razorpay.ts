import { toast } from 'sonner';

// Razorpay web-checkout helper (#220). Reuses the checkout.js SDK loaded in
// index.html. Mirrors CheckoutPage.confirmRazorpayPayment so tips (#45) and
// group split-pay (#46) share one create → open → verify path.

export interface RazorpayChargeData {
  razorpayOrderId: string;
  razorpayKeyId: string;
  amount: number; // paise
  currency: string;
}

export function openRazorpayCheckout(opts: {
  data: RazorpayChargeData;
  description: string;
  prefill?: { name?: string; email?: string };
  onVerified: (resp: RazorpayPaymentResponse) => Promise<void> | void;
  onDismiss?: () => void;
}): void {
  if (!window.Razorpay) {
    toast.error('Payment gateway is loading. Please try again.');
    return;
  }
  const options: RazorpayOptions = {
    key: opts.data.razorpayKeyId,
    amount: opts.data.amount,
    currency: opts.data.currency,
    name: 'Fe3dr',
    description: opts.description,
    order_id: opts.data.razorpayOrderId,
    prefill: opts.prefill ?? {},
    theme: { color: '#3e6b3c' },
    handler: async (resp) => {
      await opts.onVerified(resp);
    },
    modal: { ondismiss: () => opts.onDismiss?.() },
  };
  new window.Razorpay(options).open();
}
