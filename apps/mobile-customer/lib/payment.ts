// Shared payment launch flow. Used by the cart checkout (first attempt) and by
// "Retry payment" / "Pay now" on an unpaid order. Centralising it keeps the
// create-order → native-sheet hand-off identical everywhere.
//
// Uses the react-native-razorpay NATIVE checkout sheet (not a WebView) so the
// customer never sees a web page load — just our screens and the native sheet.

import RazorpayCheckout from 'react-native-razorpay';
import { router } from 'expo-router';
import { api } from './api';
import { useCartStore } from '../store/cart-store';

export interface RazorpayPaymentData {
  // "wallet" + paid:true when store credit covers the full total — no gateway sheet.
  provider?: string;
  paid?: boolean;
  walletApplied?: number;
  razorpayOrderId: string;
  razorpayKeyId: string;
  amount: number;
  currency: string;
  orderNumber?: string;
  prefill?: {
    name?: string;
    email?: string;
    phone?: string;
  };
}

// react-native-razorpay ships no types — model the bits we use.
interface RazorpaySuccess {
  razorpay_payment_id: string;
  razorpay_order_id: string;
  razorpay_signature: string;
}
interface RazorpayError {
  code?: number; // 2 = PAYMENT_CANCELLED (user dismissed the sheet)
  description?: string;
}

/**
 * Create (or re-create) the Razorpay payment for an existing order, open the
 * NATIVE checkout sheet, and route to the result screen. Safe to call on a
 * pending order to retry — the server rejects already-paid orders with 400.
 *
 * The result screen is authoritative: it polls the order's real paymentStatus
 * (set by the verify below OR the payment.captured webhook), so a failed
 * client-side verify never shows a false failure.
 *
 * @param orderId  internal order id
 * @param opts.walletAmount  store credit to apply (#141)
 */
export async function startOrderPayment(
  orderId: string,
  opts: { walletAmount?: number } = {},
): Promise<void> {
  const resp = await api.post<{ data: RazorpayPaymentData }>(
    `/v1/payments/order/${orderId}/create`,
    opts.walletAmount && opts.walletAmount > 0 ? { walletAmount: opts.walletAmount } : {},
  );
  const data = resp.data.data ?? (resp.data as unknown as RazorpayPaymentData);

  // Full-wallet order: store credit covered the total, so the server already
  // marked it paid — no gateway sheet. Go straight to the result poller.
  if (data.provider === 'wallet' || data.paid) {
    useCartStore.getState().clearCart();
    router.replace(`/payment/result?order_id=${orderId}`);
    return;
  }

  const options = {
    key: data.razorpayKeyId,
    order_id: data.razorpayOrderId,
    amount: data.amount,
    currency: data.currency ?? 'INR',
    name: 'Fe3dr',
    description: 'Order payment',
    prefill: {
      name: data.prefill?.name ?? '',
      email: data.prefill?.email ?? '',
      contact: data.prefill?.phone ?? '',
    },
    theme: { color: '#FF385C' },
  };

  try {
    const result: RazorpaySuccess = await RazorpayCheckout.open(options);
    // Fast-path verify. The result screen polls the server status as a backstop
    // (webhook), so we swallow a verify failure here rather than surfacing it.
    try {
      await api.post(`/v1/payments/order/${orderId}/verify`, {
        razorpayPaymentId: result.razorpay_payment_id,
        razorpayOrderId: result.razorpay_order_id,
        razorpaySignature: result.razorpay_signature,
      });
      useCartStore.getState().clearCart();
    } catch {
      // ignore — the result screen confirms via polling
    }
    router.replace(`/payment/result?order_id=${orderId}`);
  } catch (err) {
    const e = err as RazorpayError;
    // User dismissed the sheet — return to where they were, instantly.
    if (e?.code === 2 || /cancel/i.test(e?.description ?? '')) {
      router.back();
      return;
    }
    // Genuine failure — the result screen shows status + a Retry option.
    router.replace(`/payment/result?order_id=${orderId}`);
  }
}
